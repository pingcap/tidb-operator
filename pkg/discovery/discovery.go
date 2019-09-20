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
	"github.com/pingcap/tidb-operator/pkg/pdapi"
)

// TiDBDiscovery helps new PD member to discover all other members in cluster bootstrap phase.
type TiDBDiscovery interface {
	Discover(string) (string, error)
}

type Cluster struct {
	PDClient        pdapi.PDClient
	Replicas        int32
	ResourceVersion string
	Scheme          string
}

type GetCluster = func(ns, tcName string) (Cluster, error)
type MakeGetCluster = func(pdControl pdapi.PDControlInterface) GetCluster

type tidbDiscovery struct {
	lock       sync.Mutex
	clusters   map[string]*clusterInfo
	pdControl  pdapi.PDControlInterface
	getCluster GetCluster
}

type clusterInfo struct {
	resourceVersion string
	peers           map[string]struct{}
}

// NewTiDBDiscovery returns a TiDBDiscovery
func NewTiDBDiscovery(getCluster MakeGetCluster) TiDBDiscovery {
	td := &tidbDiscovery{
		pdControl: pdapi.NewDefaultPDControl(),
		clusters:  map[string]*clusterInfo{},
	}
	td.getCluster = getCluster(td.pdControl)
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
	keyName := fmt.Sprintf("%s/%s", ns, tcName)
	cluster, err := td.getCluster(ns, tcName)
	if err != nil {
		return "", err
	}

	currentCluster := td.clusters[keyName]
	if currentCluster == nil || currentCluster.resourceVersion != cluster.ResourceVersion {
		td.clusters[keyName] = &clusterInfo{
			resourceVersion: cluster.ResourceVersion,
			peers:           map[string]struct{}{},
		}
	}
	currentCluster = td.clusters[keyName]
	currentCluster.peers[podName] = struct{}{}

	// TODO: the replicas should be the total replicas of pd sets.
	if len(currentCluster.peers) == int(cluster.Replicas) {
		delete(currentCluster.peers, podName)
		return fmt.Sprintf("--initial-cluster=%s=%s://%s", podName, cluster.Scheme, advertisePeerUrl), nil
	}

	membersInfo, err := cluster.PDClient.GetMembers()
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
