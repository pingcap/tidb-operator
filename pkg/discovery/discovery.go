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

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pdapi"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TiDBDiscovery helps new PD member to discover all other members in cluster bootstrap phase.
type TiDBDiscovery interface {
	Discover(string) (string, error)
}

type tidbDiscovery struct {
	cli       versioned.Interface
	lock      sync.Mutex
	clusters  map[string]*clusterInfo
	tcGetFn   func(ns, tcName string) (*v1alpha1.TidbCluster, error)
	pdControl pdapi.PDControlInterface
}

type clusterInfo struct {
	resourceVersion string
	peers           map[string]struct{}
}

// NewTiDBDiscovery returns a TiDBDiscovery
func NewTiDBDiscovery(cli versioned.Interface) TiDBDiscovery {
	td := &tidbDiscovery{
		cli:       cli,
		pdControl: pdapi.NewDefaultPDControl(),
		clusters:  map[string]*clusterInfo{},
	}
	td.tcGetFn = td.realTCGetFn
	return td
}

func (td *tidbDiscovery) Discover(advertisePeerUrl string) (string, error) {
	td.lock.Lock()
	defer td.lock.Unlock()

	if advertisePeerUrl == "" {
		return "", fmt.Errorf("advertisePeerUrl is empty")
	}
	glog.Infof("advertisePeerUrl is: %s", advertisePeerUrl)
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
	tc, err := td.tcGetFn(ns, tcName)
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
		return fmt.Sprintf("--initial-cluster=%s=http://%s", podName, advertisePeerUrl), nil
	}

	pdClient := td.pdControl.GetPDClient(pdapi.Namespace(tc.GetNamespace()), tc.GetName())
	membersInfo, err := pdClient.GetMembers()
	if err != nil {
		return "", err
	}

	membersArr := make([]string, 0)
	for _, member := range membersInfo.Members {
		membersArr = append(membersArr, member.PeerUrls[0])
	}
	delete(currentCluster.peers, podName)
	return fmt.Sprintf("--join=%s", strings.Join(membersArr, ",")), nil
}

func (td *tidbDiscovery) realTCGetFn(ns, tcName string) (*v1alpha1.TidbCluster, error) {
	return td.cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
}
