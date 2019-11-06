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

	"github.com/pingcap/tidb-operator/pkg/pdapi"
	glog "k8s.io/klog"
)

// PDName is a stable identifier of a PD process
type PDName string

// ClusterName is a stable identifier of a TIDB cluster
// A PDName is unique when scoped to a ClusterName
type ClusterName string

// An Address is normally a PD node.
type Address struct {
	Name    string
	Address string
}

// TiDBDiscovery helps new PD member to discover all other members in cluster bootstrap phase.
type TiDBDiscovery interface {
	Discover(PDName, ClusterName, url.URL) (string, error)
	GetClientAddresses(ClusterName) ([]string, error)
	DeleteAddress(ClusterName, PDName) error
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

type tidbDiscoveryMembers struct {
	tidbDiscovery
	refresh ClusterRefreshMembers
}

type tidbDiscoveryNoMembers struct {
	tidbDiscovery
	refresh ClusterRefresh
}

func (td tidbDiscoveryNoMembers) getDiscovery() tidbDiscovery { return td.tidbDiscovery }
func (td tidbDiscoveryMembers) getDiscovery() tidbDiscovery   { return td.tidbDiscovery }

type hasDiscovery interface {
	getDiscovery() tidbDiscovery
}

type clusterInfo struct {
	resourceVersion string
	peers           map[PDName]url.URL
}

// NewTiDBDiscoveryWaitMembers returns a TiDBDiscovery
func NewTiDBDiscoveryWaitMembers(refresher ClusterRefreshMembers) TiDBDiscovery {
	return &tidbDiscoveryMembers{
		tidbDiscovery: tidbDiscovery{clusters: map[string]*clusterInfo{}},
		refresh:       refresher,
	}
}

// NewTiDBDiscoveryImmediate returns a TiDBDiscovery
func NewTiDBDiscoveryImmediate(refresher ClusterRefresh) TiDBDiscovery {
	return &tidbDiscoveryNoMembers{
		tidbDiscovery: tidbDiscovery{clusters: map[string]*clusterInfo{}},
		refresh:       refresher,
	}
}

func (td *tidbDiscovery) DeleteAddress(clusterID ClusterName, pdName PDName) error {
	currentCluster := td.clusters[string(clusterID)]
	if currentCluster == nil {
		return nil
	}
	delete(currentCluster.peers, pdName)
	return nil
}

func (td *tidbDiscoveryNoMembers) DeleteAddress(clusterID ClusterName, pdName PDName) error {
	return td.tidbDiscovery.DeleteAddress(clusterID, pdName)
}

func (td *tidbDiscoveryMembers) DeleteAddress(clusterID ClusterName, pdName PDName) error {
	return td.tidbDiscovery.DeleteAddress(clusterID, pdName)
}

func (td *tidbDiscoveryNoMembers) GetClientAddresses(clusterID ClusterName) ([]string, error) {
	addresses, err := td.getNamedAddresses(clusterID)
	if err != nil {
		return nil, err
	}
	return clientAddresses(addresses), nil
}

func clientAddresses(addresses []Address) []string {
	addressesNoName := make([]string, len(addresses))
	for i, address := range addresses {
		addressesNoName[i] = strings.ReplaceAll(address.Address, ":2380", ":2379")
	}
	return addressesNoName
}

func (td *tidbDiscoveryMembers) GetClientAddresses(clusterID ClusterName) ([]string, error) {
	addresses, err := td.getNamedAddresses(clusterID)
	if err != nil {
		return nil, err
	}
	return clientAddresses(addresses), nil
}

func (td *tidbDiscoveryNoMembers) getNamedAddresses(clusterID ClusterName) ([]Address, error) {
	cluster, gerr := td.refresh.GetCluster(string(clusterID))
	if gerr != nil {
		return nil, gerr
	}

	currentCluster := td.clusters[string(clusterID)]
	if currentCluster == nil {
		return nil, nil
	}

	peersArr := make([]Address, 0)
	for name, peerURL := range currentCluster.peers {
		if peerURL.Scheme == "" && cluster.Scheme != "" {
			peerURL.Scheme = cluster.Scheme
		}
		peersArr = append(peersArr, Address{
			Name:    string(name),
			Address: peerURL.String(),
		})
	}

	return peersArr, nil
}

func (td *tidbDiscoveryMembers) getNamedAddresses(clusterID ClusterName) ([]Address, error) {
	// We rely on this returning an error before the first initial-cluster
	// The caller continues to call this API
	membersInfo, err := td.refresh.GetMembers(string(clusterID))
	if err != nil {
		return nil, err
	}

	addresses := make([]Address, len(membersInfo.Members))
	for i, member := range membersInfo.Members {
		addresses[i] = Address{
			Name:    member.Name,
			Address: member.PeerUrls[0],
		}
	}
	return addresses, nil
}

// Discover starts the first PD immediately, join the rest
func (td *tidbDiscoveryNoMembers) Discover(pdName PDName, clusterID ClusterName, pdURL url.URL) (string, error) {
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

	if len(currentCluster.peers) < int(cluster.Replicas) {
		return "", fmt.Errorf("Waiting for peers to join")
	}

	if len(currentCluster.peers) > int(cluster.Replicas) {
		addressesNoName, err := td.GetClientAddresses(clusterID)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("--join=%s", strings.Join(addressesNoName, ",")), nil
	}

	addresses, err := td.getNamedAddresses(clusterID)
	if err != nil {
		return "", err
	}
	initialClusterArgs := make([]string, len(addresses))
	for i, address := range addresses {
		initialClusterArgs[i] = fmt.Sprintf("%s=%s", address.Name, address.Address)
	}

	return "--initial-cluster=" + strings.Join(initialClusterArgs, ","), nil
}


func validateEmpty(pdName PDName, clusterID ClusterName, pdURL url.URL) error {
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
func (td *tidbDiscoveryMembers) Discover(pdName PDName, clusterID ClusterName, pdURL url.URL) (string, error) {
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

	addresses, err := td.GetClientAddresses(clusterID)
	if err != nil {
		return "", err
	}
	delete(currentCluster.peers, pdName)
	return fmt.Sprintf("--join=%s", strings.Join(addresses, ",")), nil
}

// ParseAddress calls url.Parse but parses according to user expectation when there is no protocol or leading "//" or host brackets "[]"
// So it can accept HOST:IP as input.
// ParseAddress maintains the same url.String() representation
func ParseAddress(inputURL string) (url.URL, error) {
	parsedURL, err := url.Parse(inputURL)
	if err != nil {
		colonSplit := strings.Split(inputURL, ":")
		saved := false
		// just an ip address with a colon
		if !strings.Contains(inputURL, "//") && len(colonSplit) == 2 && strings.Count(colonSplit[0], ".") >= 3 {
			var newErr error
			newURL := "//" + inputURL
			parsedURL, newErr = url.Parse(newURL)
			if newErr == nil {
				saved = true
			}
		}
		if !saved {
			return url.URL{}, err
		}
	}

	// Golang expects a protocol or "//"" before the host if there is a colon port
	// This is expected behavior according to some RFC
	if parsedURL.Hostname() == "" && parsedURL.Scheme != "" && parsedURL.Opaque != "" && strings.Contains(inputURL, ":"+parsedURL.Opaque) {
		parsedURL.Host = parsedURL.Scheme + ":" + parsedURL.Opaque
		parsedURL.Scheme = ""
		parsedURL.Opaque = ""
	}
	return *parsedURL, nil
}

// ParseK8sAddress parses the url that we use in tidb-operator on K8s
func ParseK8sAddress(advertisePeerURL string) (PDName, ClusterName, url.URL, error) {
	parsedURL, err := ParseAddress(advertisePeerURL)
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

	return PDName(podName), ClusterName(fmt.Sprintf("%s/%s", ns, tcName)), parsedURL, nil
}
