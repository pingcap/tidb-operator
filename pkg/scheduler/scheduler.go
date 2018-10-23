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
	kubeCli kubernetes.Interface
	// predicates []predicates.Predicate
}

// NewScheduler returns a Scheduler
func NewScheduler(kubeCli kubernetes.Interface) Scheduler {
	return &scheduler{
		kubeCli: kubeCli,
		// predicates: []predicates.Predicate{
		//	highavailability.NewHighAvailability(10, kubeCli),
		//},
	}
}

// Filter select a node from *schedulerapiv1.ExtenderArgs.Nodes when this is a pd or tikv pod
// else return the original nodes.
func (s *scheduler) Filter(args *schedulerapiv1.ExtenderArgs) (*schedulerapiv1.ExtenderFilterResult, error) {
	pod := &args.Pod
	ns := pod.GetNamespace()
	podName := pod.GetName()
	kubeNodes := args.Nodes.Items

	return nil, nil
}

// We didn't pass `prioritizeVerb` to kubernetes scheduler extender's config file, this method will not be called.
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
