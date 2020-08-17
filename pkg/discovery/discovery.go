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
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// TiDBDiscovery helps new PD member to discover all other members in cluster bootstrap phase.
type TiDBDiscovery interface {
	Discover(string) (string, error)
}

type tidbDiscovery struct {
	cli       versioned.Interface
	lock      sync.Mutex
	clusters  map[string]*clusterInfo
	pdControl pdapi.PDControlInterface
}

type clusterInfo struct {
	resourceVersion string
	peers           map[string]struct{}
}

// NewTiDBDiscovery returns a TiDBDiscovery
func NewTiDBDiscovery(pdControl pdapi.PDControlInterface, cli versioned.Interface, kubeCli kubernetes.Interface) TiDBDiscovery {
	return &tidbDiscovery{
		cli:       cli,
		pdControl: pdControl,
		clusters:  map[string]*clusterInfo{},
	}
}

func (td *tidbDiscovery) Discover(advertisePeerUrl string) (string, error) {
	td.lock.Lock()
	defer td.lock.Unlock()

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
	clusterDomain := strings.Join(hostArr[4:], ".")
	tcName := strings.TrimSuffix(peerServiceName, "-pd-peer")
	podNamespace := os.Getenv("MY_POD_NAMESPACE")
	podTcName := os.Getenv("TC_NAME")

	tc, err := td.cli.PingcapV1alpha1().TidbClusters(podNamespace).Get(podTcName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	keyName := fmt.Sprintf("%s/%s", ns, tcName)
	if clusterDomain != "" {
		keyName = fmt.Sprintf("%s/%s", keyName, clusterDomain)
	}
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

	if len(currentCluster.peers) == int(replicas) && tc.Spec.Cluster == nil {
		delete(currentCluster.peers, podName)
		return fmt.Sprintf("--initial-cluster=%s=%s://%s", podName, tc.Scheme(), advertisePeerUrl), nil
	}

	var pdClient pdapi.PDClient
	if tc.IsHeterogeneous() {
		pdClient = td.pdControl.GetPDClient(pdapi.Namespace(tc.Spec.Cluster.Namespace), tc.Spec.Cluster.Name, tc.Spec.Cluster.Domain, tc.IsTLSClusterEnabled())
	} else {
		pdClient = td.pdControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName(), tc.Spec.ClusterDomain, tc.IsTLSClusterEnabled())
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
