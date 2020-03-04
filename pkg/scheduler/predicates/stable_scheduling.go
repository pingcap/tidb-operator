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

package predicates

import (
	"errors"
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/label"
	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

const (
	// UnableToRunOnPreviousNodeReason represents the reason of event that we
	// cannot schedule the new pod of component member to its previous node.
	UnableToRunOnPreviousNodeReason = "UnableToRunOnPreviousNode"
)

var (
	// supportedComponents holds the supported components
	supportedComponents = sets.NewString(label.TiDBLabelVal)
)

type stableScheduling struct {
	kubeCli kubernetes.Interface
	cli     versioned.Interface
}

// NewStableScheduling returns a Predicate
func NewStableScheduling(kubeCli kubernetes.Interface, cli versioned.Interface) Predicate {
	p := &stableScheduling{
		kubeCli: kubeCli,
		cli:     cli,
	}
	return p
}

func (p *stableScheduling) Name() string {
	return "StableScheduling"
}

func (p *stableScheduling) findPreviousNodeInTC(tc *v1alpha1.TidbCluster, pod *apiv1.Pod) string {
	members := tc.Status.TiDB.Members
	if members == nil {
		return ""
	}
	tidbMember, ok := tc.Status.TiDB.Members[pod.Name]
	if !ok {
		return ""
	}
	return tidbMember.NodeName
}

func (p *stableScheduling) Filter(instanceName string, pod *apiv1.Pod, nodes []apiv1.Node) ([]apiv1.Node, error) {
	ns := pod.GetNamespace()
	podName := pod.GetName()
	component := pod.Labels[label.ComponentLabelKey]
	tcName := getTCNameFromPod(pod, component)

	if !supportedComponents.Has(component) {
		return nodes, nil
	}

	tc, err := p.cli.PingcapV1alpha1().TidbClusters(ns).Get(tcName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// However tidb-operator will delete pods when tidb cluster does
			// not exist anymore, it does no harm to fail the pod. But it's
			// best to not make any assumptions here.
			return nodes, nil
		}
		return nil, err
	}

	nodeName := p.findPreviousNodeInTC(tc, pod)

	if nodeName != "" {
		klog.V(2).Infof("found previous node %q for pod %q in TiDB cluster %q", nodeName, podName, tcName)
		for _, node := range nodes {
			if node.Name == nodeName {
				klog.V(2).Infof("previous node %q for pod %q in TiDB cluster %q exists in candicates, filter out other nodes", nodeName, podName, tcName)
				return []apiv1.Node{node}, nil
			}
		}
		return nodes, fmt.Errorf("cannot run %s/%s on its previous node %q", ns, podName, nodeName)
	}

	msg := fmt.Sprintf("no previous node exists for pod %q in TiDB cluster %s/%s", podName, ns, tcName)
	klog.Warning(msg)

	return nodes, errors.New(msg)
}
