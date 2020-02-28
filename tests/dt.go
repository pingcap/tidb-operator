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
	"fmt"
	"time"

	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/tests/slack"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
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
		err := wait.PollImmediate(3*time.Second, time.Minute, func() (bool, error) {
			n, err := oa.kubeCli.CoreV1().Nodes().Get(node.Name, metav1.GetOptions{})
			if err != nil {
				klog.Errorf("get node:[%s] failed! error: %v", node.Name, err)
				return false, nil
			}
			index := i % RackNum
			n.Labels[RackLabel] = fmt.Sprintf("rack%d", index)
			_, err = oa.kubeCli.CoreV1().Nodes().Update(n)
			if err != nil {
				klog.Errorf("label node:[%s] failed! error: %v", node.Name, err)
				return false, nil
			}
			return true, nil
		})

		if err != nil {
			return fmt.Errorf("label nodes failed, error: %v", err)
		}
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
	pds, err := oa.kubeCli.CoreV1().Pods(cluster.Namespace).List(
		metav1.ListOptions{LabelSelector: labels.SelectorFromSet(
			label.New().Instance(cluster.ClusterName).PD().Labels(),
		).String()})
	if err != nil {
		return err
	}
	err = oa.checkPodsDisasterTolerance(pds.Items)
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
	err = oa.checkPodsDisasterTolerance(tikvs.Items)
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
	return oa.checkPodsDisasterTolerance(tidbs.Items)
}

func (oa *operatorActions) checkPodsDisasterTolerance(allPods []corev1.Pod) error {
	for _, pod := range allPods {
		if pod.Spec.Affinity == nil {
			return fmt.Errorf("the pod:[%s/%s] has not Affinity", pod.Namespace, pod.Name)
		}
		if pod.Spec.Affinity.PodAntiAffinity == nil {
			return fmt.Errorf("the pod:[%s/%s] has not Affinity.PodAntiAffinity", pod.Namespace, pod.Name)
		}
		if len(pod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution) == 0 {
			return fmt.Errorf("the pod:[%s/%s] has not PreferredDuringSchedulingIgnoredDuringExecution", pod.Namespace, pod.Name)
		}
		for _, prefer := range pod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
			if prefer.PodAffinityTerm.TopologyKey != RackLabel {
				return fmt.Errorf("the pod:[%s/%s] topology key is not %s", pod.Namespace, pod.Name, RackLabel)
			}
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
