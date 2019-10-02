// Copyright 2018 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package discovery

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
)

// PDName is a stable identifier of a PD process
type PDName string

// ClusterID is a stable identifier of a TIDB cluster
// A PDName is unique when scoped to a ClusterID
type ClusterID string

// TiDBDiscovery helps new PD member to discover all other members in cluster bootstrap phase.
type TiDBDiscovery interface {
	Discover(PDName, ClusterID, url.URL) (string, error)
	GetAddresses(ClusterID) ([]string, error)
	DeleteAddress(ClusterID, PDName) error
}

// Cluster is the information that discovery service needs from an implementation
type Cluster struct {
	Replicas        int32
	ResourceVersion string
	Scheme          string
}

// ClusterRefresh is an interface that has a GetCluster method
type ClusterRefresh interface {
	GetCluster(clusterID string) (Cluster, error)
}

// ClusterRefreshMembers has a GetMembers and a GetCluster method
type ClusterRefreshMembers interface {
	ClusterRefresh
	GetMembers(clusterID string) (*pdapi.MembersInfo, error)
}

type tidbDiscovery struct {
	lock     sync.Mutex
	clusters map[string]*clusterInfo
}

type tidbDiscoveryWaitMembers struct {
	tidbDiscovery
	refresh ClusterRefreshMembers
}

type tidbDiscoveryImmediate struct {
	tidbDiscovery
	refresh ClusterRefresh
}

func (td tidbDiscoveryImmediate) getDiscovery() tidbDiscovery   { return td.tidbDiscovery }
func (td tidbDiscoveryWaitMembers) getDiscovery() tidbDiscovery { return td.tidbDiscovery }

type hasDiscovery interface {
	getDiscovery() tidbDiscovery
}

type clusterInfo struct {
	resourceVersion string
	peers           map[PDName]url.URL
}

// NewTiDBDiscoveryWaitMembers returns a TiDBDiscovery
func NewTiDBDiscoveryWaitMembers(refresher ClusterRefreshMembers) TiDBDiscovery {
	return &tidbDiscoveryWaitMembers{
		tidbDiscovery: tidbDiscovery{clusters: map[string]*clusterInfo{}},
		refresh:       refresher,
	}
}

// NewTiDBDiscoveryImmediate returns a TiDBDiscovery
func NewTiDBDiscoveryImmediate(refresher ClusterRefresh) TiDBDiscovery {
	return &tidbDiscoveryImmediate{
		tidbDiscovery: tidbDiscovery{clusters: map[string]*clusterInfo{}},
		refresh:       refresher,
	}
}

func (td *tidbDiscovery) DeleteAddress(clusterID ClusterID, pdName PDName) error {
	currentCluster := td.clusters[string(clusterID)]
	if currentCluster == nil {
		return nil
	}
	delete(currentCluster.peers, pdName)
	return nil
}

func (td *tidbDiscoveryImmediate) DeleteAddress(clusterID ClusterID, pdName PDName) error {
	return td.tidbDiscovery.DeleteAddress(clusterID, pdName)
}

func (td *tidbDiscoveryWaitMembers) DeleteAddress(clusterID ClusterID, pdName PDName) error {
	return td.tidbDiscovery.DeleteAddress(clusterID, pdName)
}

func (td *tidbDiscoveryImmediate) GetAddresses(clusterID ClusterID) ([]string, error) {
	cluster, gerr := td.refresh.GetCluster(string(clusterID))
	if gerr != nil {
		return []string{}, gerr
	}

	currentCluster := td.clusters[string(clusterID)]
	if currentCluster == nil {
		return nil, nil
	}

	peersArr := make([]string, 0)
	for _, peerURL := range currentCluster.peers {
		if peerURL.Scheme == "" && cluster.Scheme != "" {
			peerURL.Scheme = cluster.Scheme
		}
		peersArr = append(peersArr, strings.ReplaceAll(peerURL.String(), ":2380", ":2379"))
	}

	return peersArr, nil
}

func (td *tidbDiscoveryWaitMembers) GetAddresses(clusterID ClusterID) ([]string, error) {
	// We rely on this returning an error before the first initial-cluster
	// The caller continues to call this API
	membersInfo, err := td.refresh.GetMembers(string(clusterID))
	if err != nil {
		return nil, err
	}

	membersArr := make([]string, 0)
	for _, member := range membersInfo.Members {
		memberURL := strings.ReplaceAll(member.PeerUrls[0], ":2380", ":2379")
		membersArr = append(membersArr, memberURL)
	}
	return membersArr, nil
}

// Discover starts the first PD immediately, join the rest
func (td *tidbDiscoveryImmediate) Discover(pdName PDName, clusterID ClusterID, pdURL url.URL) (string, error) {
	if err := validateEmpty(pdName, clusterID, pdURL); err != nil {
		return "", err
	}
	glog.Infof("url is: %s", pdURL.String())

	cluster, gerr := td.refresh.GetCluster(string(clusterID))
	if gerr != nil {
		return "", gerr
	}

	td.lock.Lock()
	defer td.lock.Unlock()

	currentCluster := td.clusters[string(clusterID)]
	if currentCluster == nil || currentCluster.resourceVersion != cluster.ResourceVersion {
		currentCluster = &clusterInfo{
			resourceVersion: cluster.ResourceVersion,
			peers:           make(map[PDName]url.URL),
		}
		td.clusters[string(clusterID)] = currentCluster
	}
	currentCluster.peers[pdName] = pdURL

	if len(currentCluster.peers) == 1 {
		if pdURL.Scheme == "" && cluster.Scheme != "" {
			pdURL.Scheme = cluster.Scheme
		}

		return fmt.Sprintf("--initial-cluster=%s=%s", pdName, pdURL.String()), nil
	}

	addresses, err := td.GetAddresses(clusterID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("--join=%s", strings.Join(addresses, ",")), nil
}

func validateEmpty(pdName PDName, clusterID ClusterID, pdURL url.URL) error {
	if pdURL.String() == "" {
		return fmt.Errorf("url is empty")
	}
	if pdName == "" {
		return fmt.Errorf("pd name is empty")
	}
	if clusterID == "" {
		return fmt.Errorf("cluster id is empty")
	}
	return nil
}

// Discover waits for all PD to join before starting the cluster
// this approach was probably more useful before the PD isinitialized status API was available
func (td *tidbDiscoveryWaitMembers) Discover(pdName PDName, clusterID ClusterID, pdURL url.URL) (string, error) {
	if err := validateEmpty(pdName, clusterID, pdURL); err != nil {
		return "", err
	}
	glog.Infof("url is: %s", pdURL.String())

	cluster, gerr := td.refresh.GetCluster(string(clusterID))
	if gerr != nil {
		return "", gerr
	}

	td.lock.Lock()
	defer td.lock.Unlock()

	currentCluster := td.clusters[string(clusterID)]
	if currentCluster == nil || currentCluster.resourceVersion != cluster.ResourceVersion {
		currentCluster = &clusterInfo{
			resourceVersion: cluster.ResourceVersion,
			peers:           make(map[PDName]url.URL),
		}
		td.clusters[string(clusterID)] = currentCluster
	}
	currentCluster.peers[pdName] = pdURL

	if len(currentCluster.peers) == int(cluster.Replicas) {
		delete(currentCluster.peers, pdName)

		if pdURL.Scheme == "" && cluster.Scheme != "" {
			pdURL.Scheme = cluster.Scheme
		}

		return fmt.Sprintf("--initial-cluster=%s=%s", pdName, pdURL.String()), nil
	}

	addresses, err := td.GetAddresses(clusterID)
	if err != nil {
		return "", err
	}
	delete(currentCluster.peers, pdName)
	return fmt.Sprintf("--join=%s", strings.Join(addresses, ",")), nil
}

// ParseURL calls url.Parse and corrects a bug in the implementation
func ParseURL(inputURL string) (url.URL, error) {
	parsedURL, err := url.Parse(inputURL)
	if err != nil {
		return url.URL{}, err
	}

	// port parsing is broken when there is no protocol
	if parsedURL.Hostname() == "" && parsedURL.Scheme != "" && parsedURL.Opaque != "" && strings.Contains(inputURL, ":"+parsedURL.Opaque) {
		parsedURL.Host = parsedURL.Scheme + ":" + parsedURL.Opaque
		parsedURL.Scheme = ""
		parsedURL.Opaque = ""
	}
	return *parsedURL, nil
}

// ParseK8sURL parses the url that we use in tidb-operator on K8s
func ParseK8sURL(advertisePeerURL string) (PDName, ClusterID, url.URL, error) {
	parsedURL, err := ParseURL(advertisePeerURL)
	if err != nil {
		return "", "", url.URL{}, err
	}

	strArr := strings.Split(advertisePeerURL, ".")
	if len(strArr) != 4 {
		return "", "", parsedURL, fmt.Errorf("advertisePeerURL format is wrong: %s", advertisePeerURL)
	}

	podName, peerServiceName, ns := strArr[0], strArr[1], strArr[2]
	tcName := strings.TrimSuffix(peerServiceName, "-pd-peer")
	podNamespace := os.Getenv("MY_POD_NAMESPACE")
	if ns != podNamespace {
		return "", "", parsedURL, fmt.Errorf("the peer's namespace: %s is not equal to discovery namespace: %s", ns, podNamespace)
	}

	return PDName(podName), ClusterID(fmt.Sprintf("%s/%s", ns, tcName)), parsedURL, nil
}
