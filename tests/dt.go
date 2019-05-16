// Copyright 2019 PingCAP, Inc.
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

package tests

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/tests/slack"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	RackLabel = "rack"
	RackNum   = 3
)

// RegionInfo records detail region info for api usage.
type RegionInfo struct {
	ID          uint64              `json:"id"`
	StartKey    string              `json:"start_key"`
	EndKey      string              `json:"end_key"`
	RegionEpoch *metapb.RegionEpoch `json:"epoch,omitempty"`
	Peers       []*metapb.Peer      `json:"peers,omitempty"`

	Leader          *metapb.Peer      `json:"leader,omitempty"`
	DownPeers       []*pdpb.PeerStats `json:"down_peers,omitempty"`
	PendingPeers    []*metapb.Peer    `json:"pending_peers,omitempty"`
	WrittenBytes    uint64            `json:"written_bytes,omitempty"`
	ReadBytes       uint64            `json:"read_bytes,omitempty"`
	ApproximateSize int64             `json:"approximate_size,omitempty"`
	ApproximateKeys int64             `json:"approximate_keys,omitempty"`
}

// RegionsInfo contains some regions with the detailed region info.
type RegionsInfo struct {
	Count   int           `json:"count"`
	Regions []*RegionInfo `json:"regions"`
}

func (oa *operatorActions) LabelNodes() error {
	nodes, err := oa.kubeCli.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for i, node := range nodes.Items {
		index := i % RackNum
		node.Labels[RackLabel] = fmt.Sprintf("rack%d", index)
		oa.kubeCli.CoreV1().Nodes().Update(&node)
	}
	return nil
}

func (oa *operatorActions) LabelNodesOrDie() {
	err := oa.LabelNodes()
	if err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) CheckDisasterTolerance(cluster *TidbClusterConfig) error {
	nodeMap := map[string]corev1.Node{}
	nodes, err := oa.kubeCli.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, node := range nodes.Items {
		nodeMap[node.Name] = node
	}

	pds, err := oa.kubeCli.CoreV1().Pods(cluster.Namespace).List(
		metav1.ListOptions{LabelSelector: labels.SelectorFromSet(
			label.New().Instance(cluster.ClusterName).PD().Labels(),
		).String()})
	if err != nil {
		return err
	}
	err = oa.checkDisasterTolerance(pds.Items, nodeMap)
	if err != nil {
		return err
	}

	tikvs, err := oa.kubeCli.CoreV1().Pods(cluster.Namespace).List(
		metav1.ListOptions{LabelSelector: labels.SelectorFromSet(
			label.New().Instance(cluster.ClusterName).TiKV().Labels(),
		).String()})
	if err != nil {
		return err
	}
	err = oa.checkDisasterTolerance(tikvs.Items, nodeMap)
	if err != nil {
		return err
	}

	tidbs, err := oa.kubeCli.CoreV1().Pods(cluster.Namespace).List(
		metav1.ListOptions{LabelSelector: labels.SelectorFromSet(
			label.New().Instance(cluster.ClusterName).TiDB().Labels(),
		).String()})
	if err != nil {
		return err
	}
	return oa.checkDisasterTolerance(tidbs.Items, nodeMap)
}

func (oa *operatorActions) checkDisasterTolerance(allPods []corev1.Pod, nodeMap map[string]corev1.Node) error {
	rackPods := map[string][]corev1.Pod{}
	for _, pod := range allPods {
		if node, exist := nodeMap[pod.Spec.NodeName]; exist {
			pods, exist := rackPods[node.Labels[RackLabel]]
			if !exist {
				pods = []corev1.Pod{}
			}
			pods = append(pods, pod)
			rackPods[node.Labels[RackLabel]] = pods
		}
	}

	podNum := len(allPods)
	maxPodsOneRack := podNum / RackNum
	mod := podNum % RackNum
	if mod > 0 {
		maxPodsOneRack = maxPodsOneRack + 1
	}

	for rack, pods := range rackPods {
		if len(pods) > maxPodsOneRack {
			return fmt.Errorf("the rack:[%s] have pods more than %d", rack, maxPodsOneRack)
		}
	}
	return nil
}

func (oa *operatorActions) CheckDisasterToleranceOrDie(cluster *TidbClusterConfig) {
	err := oa.CheckDisasterTolerance(cluster)
	if err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) CheckDataRegionDisasterToleranceOrDie(cluster *TidbClusterConfig) {
	err := oa.CheckDataRegionDisasterTolerance(cluster)
	if err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) CheckDataRegionDisasterTolerance(cluster *TidbClusterConfig) error {
	pdClient := http.Client{
		Timeout: 10 * time.Second,
	}
	url := fmt.Sprintf("http://%s-pd.%s:2379/pd/api/v1/regions", cluster.ClusterName, cluster.Namespace)
	resp, err := pdClient.Get(url)
	if err != nil {
		return err
	}
	buf, _ := ioutil.ReadAll(resp.Body)
	regions := &RegionsInfo{}
	err = json.Unmarshal(buf, &regions)
	if err != nil {
		return err
	}

	rackNodeMap, err := oa.getNodeRackMap()
	if err != nil {
		return err
	}

	for _, region := range regions.Regions {
		regionRacks := map[string]uint64{}
		for _, peer := range region.Peers {
			storeID := strconv.FormatUint(peer.StoreId, 10)
			nodeName, err := oa.getNodeByStoreId(storeID, cluster)
			if err != nil {
				return err
			}
			rackName := rackNodeMap[nodeName]
			if otherID, exist := regionRacks[rackName]; exist {
				return fmt.Errorf("region[%d]'s peer: [%d]and[%d] are in same rack:[%s]", region.ID, otherID, peer.Id, rackName)
			}
			regionRacks[rackName] = peer.Id
		}
	}
	return nil
}

func (oa *operatorActions) getNodeByStoreId(storeID string, cluster *TidbClusterConfig) (string, error) {
	tc, err := oa.cli.PingcapV1alpha1().TidbClusters(cluster.Namespace).Get(cluster.ClusterName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	if store, exist := tc.Status.TiKV.Stores[storeID]; exist {
		pod, err := oa.kubeCli.CoreV1().Pods(cluster.Namespace).Get(store.PodName, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		return pod.Spec.NodeName, nil
	}

	return "", fmt.Errorf("the storeID:[%s] is not exist in tidbCluster:[%s] Status", storeID, cluster.FullName())
}

func (oa *operatorActions) getNodeRackMap() (map[string]string, error) {
	rackNodeMap := map[string]string{}
	nodes, err := oa.kubeCli.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return rackNodeMap, err
	}
	for _, node := range nodes.Items {
		rackNodeMap[node.Name] = node.Labels[RackLabel]
	}

	return rackNodeMap, nil
}
