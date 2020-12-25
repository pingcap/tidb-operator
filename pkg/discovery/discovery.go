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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
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
	VerifyPDEndpoint(string) (string, error)
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

type pdEndpointURL struct {
	scheme       string
	pdMemberName string
	pdMemberPort string
	tcName       string
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
		pdAddresses := tc.Spec.PDAddresses
		// Join an existing PD cluster if tc.Spec.PDAddresses is set
		if len(pdAddresses) != 0 {
			return fmt.Sprintf("--join=%s", strings.Join(pdAddresses, ",")), nil
		}
		// Initialize the PD cluster with the FQDN format service record if tc.Spec.ClusterDomain is set.
		if len(tc.Spec.ClusterDomain) > 0 {
			return fmt.Sprintf("--initial-cluster=%s=%s://%s", strArr[0], tc.Scheme(), advertisePeerUrl), nil
		}
		// Initialize the PD cluster in the normal format service record.
		return fmt.Sprintf("--initial-cluster=%s=%s://%s", podName, tc.Scheme(), advertisePeerUrl), nil
	}

	var pdClients []pdapi.PDClient

	if tc.Spec.PD != nil {
		pdClients = append(pdClients, d.pdControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.IsTLSClusterEnabled()))
	}

	if tc.Spec.Cluster != nil && len(tc.Spec.Cluster.Name) > 0 {
		namespace := tc.Spec.Cluster.Namespace
		if len(namespace) == 0 {
			namespace = tc.GetNamespace()
		}
		pdClients = append(pdClients, d.pdControl.GetClusterRefPDClient(pdapi.Namespace(namespace), tc.Spec.Cluster.Name, tc.Spec.Cluster.ClusterDomain, tc.IsTLSClusterEnabled()))
	}

	for _, pdMember := range tc.Status.PD.PeerMembers {
		pdClients = append(pdClients, d.pdControl.GetPeerPDClient(pdapi.Namespace(ns), tc.Name, tc.IsTLSClusterEnabled(), pdMember.ClientURL, pdMember.Name))
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
		// Corresponds to https://github.com/tikv/pd/blob/43baea981b406df26cd49e8b99cc42354f0a6696/server/join/join.go#L88.
		// When multi-cluster enabled, the PD member name is not pod name(cluster1-pd-0) but the FQDN (cluster1-pd-0.cluster1-pd-peer.pingcap.svc.cluster.local).
		// For example,
		// advertisePeerURL without cluster domain: strArr = ["cluster1-pd-0.cluster1-pd-peer.pingcap.svc","2380"], member.Name = cluster1-pd-0, podName = cluster1-pd-0
		// advertisePeerURL with cluster domain: strArr = ["cluster1-pd-0.cluster1-pd-peer.pingcap.svc.cluster.local","2380"], member.Name = cluster1-pd-0.cluster1-pd-peer.pingcap.svc.cluster.local, podName = cluster1-pd-0
		// So we use podName when advertisePeerURL without cluster domain and use strArr[0] when advertisePeerURL with cluster domain
		//
		// In some failure situations, for example, delete the pd's data directory, pd will try to restart
		// and get join info from discovery service. But pd embed etcd may still have the registered member info,
		// which will return the argument to join pd itself, which is not suggested in pd.
		if member.Name == podName || member.Name == strArr[0] {
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

func (d *tidbDiscovery) VerifyPDEndpoint(pdURL string) (string, error) {
	// save a copy of pdURL for failure
	pdURL = strings.Trim(pdURL, "\n")
	copyAdvertisePeerURL := pdURL
	pdEndpoint := parsePDURL(pdURL)

	ns := os.Getenv("MY_POD_NAMESPACE")
	tc, err := d.cli.PingcapV1alpha1().TidbClusters(ns).Get(pdEndpoint.tcName, metav1.GetOptions{})
	if err != nil {
		return pdURL, err
	}

	noScheme := false
	if len(pdEndpoint.scheme) == 0 {
		noScheme = true
		if tc.IsTLSClusterEnabled() {
			pdEndpoint.scheme = "https"
		} else {
			pdEndpoint.scheme = "http"
		}
		pdURL = fmt.Sprintf("%s://%s", pdEndpoint.scheme, pdURL)
	}

	if d.pdEndpointHealthCheck(tc, pdURL, pdEndpoint.pdMemberName) {
		if noScheme {
			return fmt.Sprintf("%s:%s", pdEndpoint.pdMemberName, pdEndpoint.pdMemberPort), nil
		}
		return pdURL, nil
	}

	for _, pdMember := range tc.Status.PD.PeerMembers {
		if d.pdEndpointHealthCheck(tc, pdMember.ClientURL, pdMember.Name) {
			if noScheme {
				return fmt.Sprintf("%s:%s", pdMember.Name, pdEndpoint.pdMemberPort), nil
			}
			return pdMember.ClientURL, nil
		}
	}

	// if failed, we should return the default value here
	return copyAdvertisePeerURL, nil
}

// pdEndpointHealthCheck checks if PD PeerEndpoint is working
func (d *tidbDiscovery) pdEndpointHealthCheck(tc *v1alpha1.TidbCluster, advertisePeerURL string, peerName string) bool {
	pdClient := d.pdControl.GetPeerPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.IsTLSClusterEnabled(), advertisePeerURL, peerName)
	_, err := pdClient.GetHealth()
	return err == nil
}

// parsePDURL parses pdURL to PDEndpoint related information
func parsePDURL(pdURL string) *pdEndpointURL {
	// Deal with scheme
	pdEndpoint := &pdEndpointURL{
		scheme:       "",
		pdMemberName: "",
		pdMemberPort: "2379",
		tcName:       "",
	}

	scheme := strings.Split(pdURL, "://")
	if len(scheme) == 1 {
		pdEndpoint.scheme = ""
		pdEndpoint.pdMemberName = scheme[0]
	} else {
		pdEndpoint.scheme = scheme[0]
		pdEndpoint.pdMemberName = scheme[1]
	}

	// Deal with port
	hostURLArr := strings.Split(pdEndpoint.pdMemberName, ":")
	if len(hostURLArr) > 1 {
		pdEndpoint.pdMemberName = hostURLArr[0]
		pdEndpoint.pdMemberPort = hostURLArr[1]
	}

	// Deal with tcName
	hostArrs := strings.Split(pdEndpoint.pdMemberName, "-pd")
	pdEndpoint.tcName = hostArrs[0]

	return pdEndpoint
}
