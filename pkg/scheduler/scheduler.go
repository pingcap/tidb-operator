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

package scheduler

import (
	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/scheduler/predicates"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	schedulerapiv1 "k8s.io/kubernetes/pkg/scheduler/api/v1"
)

// Scheduler is an interface for external processes to influence scheduling
// decisions made by kubernetes. This is typically needed for resources not directly
// managed by kubernetes.
type Scheduler interface {
	// Filter based on extender-implemented predicate functions. The filtered list is
	// expected to be a subset of the supplied list.
	Filter(*schedulerapiv1.ExtenderArgs) (*schedulerapiv1.ExtenderFilterResult, error)

	// Prioritize based on extender-implemented priority functions. The returned scores & weight
	// are used to compute the weighted score for an extender. The weighted scores are added to
	// the scores computed  by kubernetes scheduler. The total scores are used to do the host selection.
	Priority(*schedulerapiv1.ExtenderArgs) (schedulerapiv1.HostPriorityList, error)
}

type scheduler struct {
	kubeCli    kubernetes.Interface
	predicates []predicates.Predicate
}

// NewScheduler returns a Scheduler
func NewScheduler(kubeCli kubernetes.Interface) Scheduler {
	return &scheduler{
		kubeCli: kubeCli,
		predicates: []predicates.Predicate{
			predicates.NewHA(kubeCli),
		},
	}
}

// Filter selects a set of nodes from *schedulerapiv1.ExtenderArgs.Nodes when this is a pd or tikv pod
// otherwise, returns the original nodes.
func (s *scheduler) Filter(args *schedulerapiv1.ExtenderArgs) (*schedulerapiv1.ExtenderFilterResult, error) {
	pod := &args.Pod
	ns := pod.GetNamespace()
	podName := pod.GetName()
	kubeNodes := args.Nodes.Items

	var instanceName string
	var exist bool
	if instanceName, exist = pod.Labels[label.InstanceLabelKey]; !exist {
		glog.Warningf("can't find instanceName in pod labels: %s/%s", ns, podName)
		return &schedulerapiv1.ExtenderFilterResult{
			Nodes: args.Nodes,
		}, nil
	}
	if component := pod.Labels[label.ComponentLabelKey]; component != label.PDLabelVal && component != label.TiKVLabelVal {
		return &schedulerapiv1.ExtenderFilterResult{
			Nodes: args.Nodes,
		}, nil
	}

	glog.Infof("scheduling pod: %s/%s", ns, podName)
	var err error
	for _, predicate := range s.predicates {
		glog.V(4).Infof("entering predicate: %s, nodes: %v", predicate.Name(), getNodeNames(kubeNodes))
		kubeNodes, err = predicate.Filter(instanceName, pod, kubeNodes)
		if err != nil {
			return nil, err
		}
		glog.V(4).Infof("leaving predicate: %s, nodes: %v", predicate.Name(), getNodeNames(kubeNodes))
	}

	return &schedulerapiv1.ExtenderFilterResult{
		Nodes: &apiv1.NodeList{Items: kubeNodes},
	}, nil
}

// We don't pass `prioritizeVerb` to kubernetes scheduler extender's config file, this method will not be called.
func (s *scheduler) Priority(args *schedulerapiv1.ExtenderArgs) (schedulerapiv1.HostPriorityList, error) {
	result := schedulerapiv1.HostPriorityList{}

	// avoid index out of range panic
	if len(args.Nodes.Items) > 0 {
		result = append(result, schedulerapiv1.HostPriority{
			Host:  args.Nodes.Items[0].Name,
			Score: 0,
		})
	}

	return result, nil
}

var _ Scheduler = &scheduler{}

func getNodeNames(nodes []apiv1.Node) []string {
	nodeNames := make([]string, 0)
	for _, node := range nodes {
		nodeNames = append(nodeNames, node.GetName())
	}
	return nodeNames
}
