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

func (d *tidbDiscovery) Discover(advertisePeerUrl string) (string, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	if advertisePeerUrl == "" {
		return "", fmt.Errorf("advertisePeerUrl is empty")
	}
	klog.Infof("advertisePeerUrl is: %s", advertisePeerUrl)
	strArr := strings.Split(advertisePeerUrl, ":")
	hostArr := strings.Split(strArr[0], ".")

	if len(hostArr) < 4 || hostArr[3] != "svc" {
		return "", fmt.Errorf("advertisePeerUrl format is wrong: %s", advertisePeerUrl)
	}

	podName, peerServiceName, ns := hostArr[0], hostArr[1], hostArr[2]
	tcName := strings.TrimSuffix(peerServiceName, "-pd-peer")
	podNamespace := os.Getenv("MY_POD_NAMESPACE")

	if ns != podNamespace {
		return "", fmt.Errorf("the peer's namespace: %s is not equal to discovery namespace: %s", ns, podNamespace)
	}
	tc, err := d.cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	keyName := fmt.Sprintf("%s/%s", ns, tcName)
	pdAddresses := tc.Spec.PDAddresses

	currentCluster := d.clusters[keyName]
	if currentCluster == nil || currentCluster.resourceVersion != tc.ResourceVersion {
		d.clusters[keyName] = &clusterInfo{
			resourceVersion: tc.ResourceVersion,
			peers:           map[string]struct{}{},
		}
	}
	currentCluster = d.clusters[keyName]
	currentCluster.peers[podName] = struct{}{}

	// Should take failover replicas into consideration
	if len(currentCluster.peers) == int(tc.PDStsDesiredReplicas()) && tc.Spec.Cluster == nil {
		delete(currentCluster.peers, podName)
		if len(pdAddresses) != 0 {
			return fmt.Sprintf("--join=%s", strings.Join(pdAddresses, ",")), nil
		}
		if len(tc.Spec.ClusterDomain) > 0 {
			return fmt.Sprintf("--initial-cluster=%s=%s://%s", strArr[0], tc.Scheme(), advertisePeerUrl), nil
		}
		return fmt.Sprintf("--initial-cluster=%s=%s://%s", podName, tc.Scheme(), advertisePeerUrl), nil
	}

	var pdClients []pdapi.PDClient
	if tc.Spec.Cluster != nil && len(tc.Spec.Cluster.Name) > 0 {
		namespace := tc.Spec.Cluster.Namespace
		if len(namespace) == 0 {
			namespace = tc.GetNamespace()
		}
		pdClients = append(pdClients, d.pdControl.GetClusterRefPDClient(pdapi.Namespace(namespace), tc.Spec.Cluster.Name, tc.Spec.Cluster.ClusterDomain, tc.IsTLSClusterEnabled()))
	}
	if tc.Spec.PD != nil {
		pdClients = append(pdClients, d.pdControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.IsTLSClusterEnabled()))
	}

	var membersInfo *pdapi.MembersInfo
	for _, client := range pdClients {
		membersInfo, err = client.GetMembers()
		if err == nil {
			break
		}
	}
	if err != nil {
		return "", err
	}

	membersArr := make([]string, 0)
	for _, member := range membersInfo.Members {
		// In some failure situations, for example, delete the pd's data directory, pd will try to restart
		// and get join info from discovery service. But pd embed etcd may still have the registered member info,
		// which will return the argument to join pd itself, which is not suggested in pd.
		if member.Name == podName {
			continue
		}
		memberURL := strings.ReplaceAll(member.PeerUrls[0], ":2380", ":2379")
		membersArr = append(membersArr, memberURL)
	}
	delete(currentCluster.peers, podName)
	return fmt.Sprintf("--join=%s", strings.Join(membersArr, ",")), nil
}

func (d *tidbDiscovery) DiscoverDM(advertisePeerUrl string) (string, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	if advertisePeerUrl == "" {
		return "", fmt.Errorf("dm advertisePeerUrl is empty")
	}
	klog.Infof("dm advertisePeerUrl is: %s", advertisePeerUrl)
	strArr := strings.Split(advertisePeerUrl, ".")
	if len(strArr) != 2 {
		return "", fmt.Errorf("dm advertisePeerUrl format is wrong: %s", advertisePeerUrl)
	}

	podName, peerServiceNameWithPort := strArr[0], strArr[1]
	strArr = strings.Split(peerServiceNameWithPort, ":")
	if len(strArr) != 2 {
		return "", fmt.Errorf("dm advertisePeerUrl format is wrong: %s", advertisePeerUrl)
	}
	peerServiceName := strArr[0]
	dcName := strings.TrimSuffix(peerServiceName, "-dm-master-peer")
	ns := os.Getenv("MY_POD_NAMESPACE")

	dc, err := d.cli.PingcapV1alpha1().DMClusters(ns).Get(dcName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	keyName := fmt.Sprintf("%s/%s", ns, dcName)

	currentCluster := d.dmClusters[keyName]
	if currentCluster == nil || currentCluster.resourceVersion != dc.ResourceVersion {
		d.dmClusters[keyName] = &clusterInfo{
			resourceVersion: dc.ResourceVersion,
			peers:           map[string]struct{}{},
		}
	}
	currentCluster = d.dmClusters[keyName]
	currentCluster.peers[podName] = struct{}{}

	if len(currentCluster.peers) == int(dc.MasterStsDesiredReplicas()) {
		delete(currentCluster.peers, podName)
		return fmt.Sprintf("--initial-cluster=%s=%s://%s", podName, dc.Scheme(), advertisePeerUrl), nil
	}

	masterClient := d.masterControl.GetMasterClient(dc.GetNamespace(), dc.GetName(), dc.IsTLSClusterEnabled())
	mastersInfos, err := masterClient.GetMasters()
	if err != nil {
		return "", err
	}

	mastersArr := make([]string, 0)
	for _, master := range mastersInfos {
		// In some failure situations, for example, delete the dm-master's data directory, dm-master will try to restart
		// and get join info from discovery service. But dm-master embed etcd may still have the registered member info,
		// which will return the argument to join dm-master itself, which is not allowed in dm-master.
		if master.Name == podName {
			continue
		}
		memberURL := strings.ReplaceAll(master.PeerURLs[0], ":8291", ":8261")
		mastersArr = append(mastersArr, memberURL)
	}
	delete(currentCluster.peers, podName)
	return fmt.Sprintf("--join=%s", strings.Join(mastersArr, ",")), nil
}
