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
	"os"
	"strings"
	"sync"

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/dmapi"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// TiDBDiscovery helps new PD and dm-master member to discover all other members in cluster bootstrap phase.
type TiDBDiscovery interface {
	Discover(string) (string, error)
	DiscoverDM(string) (string, error)
}

type tidbDiscovery struct {
	cli           versioned.Interface
	lock          sync.Mutex
	clusters      map[string]*clusterInfo
	dmClusters    map[string]*clusterInfo
	pdControl     pdapi.PDControlInterface
	masterControl dmapi.MasterControlInterface
}

type clusterInfo struct {
	resourceVersion string
	peers           map[string]struct{}
}

// NewTiDBDiscovery returns a TiDBDiscovery
func NewTiDBDiscovery(pdControl pdapi.PDControlInterface, masterControl dmapi.MasterControlInterface, cli versioned.Interface, kubeCli kubernetes.Interface) TiDBDiscovery {
	return &tidbDiscovery{
		cli:           cli,
		pdControl:     pdControl,
		masterControl: masterControl,
		clusters:      map[string]*clusterInfo{},
		dmClusters:    map[string]*clusterInfo{},
	}
}

func (td *tidbDiscovery) Discover(advertisePeerUrl string) (string, error) {
	td.lock.Lock()
	defer td.lock.Unlock()

	if advertisePeerUrl == "" {
		return "", fmt.Errorf("advertisePeerUrl is empty")
	}
	klog.Infof("advertisePeerUrl is: %s", advertisePeerUrl)
	strArr := strings.Split(advertisePeerUrl, ".")
	if len(strArr) != 4 {
		return "", fmt.Errorf("advertisePeerUrl format is wrong: %s", advertisePeerUrl)
	}

	podName, peerServiceName, ns := strArr[0], strArr[1], strArr[2]
	tcName := strings.TrimSuffix(peerServiceName, "-pd-peer")
	podNamespace := os.Getenv("MY_POD_NAMESPACE")
	if ns != podNamespace {
		return "", fmt.Errorf("the peer's namespace: %s is not equal to discovery namespace: %s", ns, podNamespace)
	}
	tc, err := td.cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	keyName := fmt.Sprintf("%s/%s", ns, tcName)
	// TODO: the replicas should be the total replicas of pd sets.
	replicas := tc.Spec.PD.Replicas

	currentCluster := td.clusters[keyName]
	if currentCluster == nil || currentCluster.resourceVersion != tc.ResourceVersion {
		td.clusters[keyName] = &clusterInfo{
			resourceVersion: tc.ResourceVersion,
			peers:           map[string]struct{}{},
		}
	}
	currentCluster = td.clusters[keyName]
	currentCluster.peers[podName] = struct{}{}

	if len(currentCluster.peers) == int(replicas) {
		delete(currentCluster.peers, podName)
		return fmt.Sprintf("--initial-cluster=%s=%s://%s", podName, tc.Scheme(), advertisePeerUrl), nil
	}

	var pdClient pdapi.PDClient
	if tc.IsHeterogeneous() {
		pdClient = td.pdControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.Spec.Cluster.Name, tc.IsTLSClusterEnabled())
	} else {
		pdClient = td.pdControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.IsTLSClusterEnabled())
	}

	membersInfo, err := pdClient.GetMembers()
	if err != nil {
		return "", err
	}

	membersArr := make([]string, 0)
	for _, member := range membersInfo.Members {
		memberURL := strings.ReplaceAll(member.PeerUrls[0], ":2380", ":2379")
		membersArr = append(membersArr, memberURL)
	}
	delete(currentCluster.peers, podName)
	return fmt.Sprintf("--join=%s", strings.Join(membersArr, ",")), nil
}

func (td *tidbDiscovery) DiscoverDM(advertisePeerUrl string) (string, error) {
	td.lock.Lock()
	defer td.lock.Unlock()

	if advertisePeerUrl == "" {
		return "", fmt.Errorf("dm advertisePeerUrl is empty")
	}
	klog.Infof("dm advertisePeerUrl is: %s", advertisePeerUrl)
	strArr := strings.Split(advertisePeerUrl, ".")
	if len(strArr) != 4 {
		return "", fmt.Errorf("dm advertisePeerUrl format is wrong: %s", advertisePeerUrl)
	}

	podName, peerServiceName, ns := strArr[0], strArr[1], strArr[2]
	dcName := strings.TrimSuffix(peerServiceName, "-dm-master-peer")
	podNamespace := os.Getenv("MY_POD_NAMESPACE")
	if ns != podNamespace {
		return "", fmt.Errorf("dm the peer's namespace: %s is not equal to discovery namespace: %s", ns, podNamespace)
	}
	dc, err := td.cli.PingcapV1alpha1().DMClusters(ns).Get(dcName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	keyName := fmt.Sprintf("%s/%s", ns, dcName)
	// TODO: the replicas should be the total replicas of dm master sets.
	replicas := dc.Spec.Master.Replicas

	currentCluster := td.dmClusters[keyName]
	if currentCluster == nil || currentCluster.resourceVersion != dc.ResourceVersion {
		td.dmClusters[keyName] = &clusterInfo{
			resourceVersion: dc.ResourceVersion,
			peers:           map[string]struct{}{},
		}
	}
	currentCluster = td.dmClusters[keyName]
	currentCluster.peers[podName] = struct{}{}

	if len(currentCluster.peers) == int(replicas) {
		delete(currentCluster.peers, podName)
		return fmt.Sprintf("--initial-cluster=%s=%s://%s", podName, dc.Scheme(), advertisePeerUrl), nil
	}

	masterClient := td.masterControl.GetMasterClient(dmapi.Namespace(dc.GetNamespace()), dc.GetName(), dc.IsTLSClusterEnabled())
	mastersInfos, err := masterClient.GetMasters()
	if err != nil {
		return "", err
	}

	mastersArr := make([]string, 0)
	for _, master := range mastersInfos {
		mastersArr = append(mastersArr, master.ClientURLs[0])
	}
	delete(currentCluster.peers, podName)
	return fmt.Sprintf("--join=%s", strings.Join(mastersArr, ",")), nil
}
