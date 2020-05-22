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
	"fmt"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/scheduler/predicates"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	eventv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	schedulerapi "k8s.io/kubernetes/pkg/scheduler/api"
	schedulerapiv1 "k8s.io/kubernetes/pkg/scheduler/api/v1"
)

// Scheduler is an interface for external processes to influence scheduling
// decisions made by kubernetes. This is typically needed for resources not directly
// managed by kubernetes.
type Scheduler interface {
	// Filter based on extender-implemented predicate functions. The filtered list is
	// expected to be a subset of the supplied list.
	Filter(*schedulerapiv1.ExtenderArgs) (*schedulerapiv1.ExtenderFilterResult, error)

	// Preempt implements scheduler extender preempt verb.
	Preempt(args *schedulerapi.ExtenderPreemptionArgs) (*schedulerapi.ExtenderPreemptionResult, error)

	// Prioritize based on extender-implemented priority functions. The returned scores & weight
	// are used to compute the weighted score for an extender. The weighted scores are added to
	// the scores computed  by kubernetes scheduler. The total scores are used to do the host selection.
	Priority(*schedulerapiv1.ExtenderArgs) (schedulerapiv1.HostPriorityList, error)
}

type scheduler struct {
	// component => predicates
	predicates map[string][]predicates.Predicate

	kubeCli  kubernetes.Interface
	recorder record.EventRecorder
}

// NewScheduler returns a Scheduler
func NewScheduler(kubeCli kubernetes.Interface, cli versioned.Interface) Scheduler {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&eventv1.EventSinkImpl{
		Interface: eventv1.New(kubeCli.CoreV1().RESTClient()).Events("")})
	recorder := eventBroadcaster.NewRecorder(kubescheme.Scheme, apiv1.EventSource{Component: "tidb-scheduler"})
	predicatesByComponent := map[string][]predicates.Predicate{
		label.PDLabelVal: {
			predicates.NewHA(kubeCli, cli),
		},
		label.TiKVLabelVal: {
			predicates.NewHA(kubeCli, cli),
		},
	}
	if features.DefaultFeatureGate.Enabled(features.StableScheduling) {
		predicatesByComponent[label.TiDBLabelVal] = []predicates.Predicate{
			predicates.NewStableScheduling(kubeCli, cli),
		}
	}
	return &scheduler{
		predicates: predicatesByComponent,
		kubeCli:    kubeCli,
		recorder:   recorder,
	}
}

// Filter selects a set of nodes from *schedulerapiv1.ExtenderArgs.Nodes when this is a pd or tikv pod
// otherwise, returns the original nodes.
func (s *scheduler) Filter(args *schedulerapiv1.ExtenderArgs) (*schedulerapiv1.ExtenderFilterResult, error) {
	pod := args.Pod
	ns := pod.GetNamespace()
	podName := pod.GetName()
	kubeNodes := args.Nodes.Items

	if pod.Annotations != nil {
		if _, ok := pod.Annotations[label.AnnFailTiDBScheduler]; ok {
			return nil, FailureError{PodName: pod.Name}
		}
	}

	var instanceName string
	var exist bool
	if instanceName, exist = pod.Labels[label.InstanceLabelKey]; !exist {
		klog.Warningf("can't find instanceName in pod labels: %s/%s", ns, podName)
		return &schedulerapiv1.ExtenderFilterResult{
			Nodes: args.Nodes,
		}, nil
	}

	component, ok := pod.Labels[label.ComponentLabelKey]
	if !ok {
		klog.Warningf("can't find component label in pod labels: %s/%s", ns, podName)
		return &schedulerapiv1.ExtenderFilterResult{
			Nodes: args.Nodes,
		}, nil
	}

	predicatesByComponent, ok := s.predicates[component]
	if !ok {
		klog.Warningf("no predicate for component %q, ignored", component)
		return &schedulerapiv1.ExtenderFilterResult{
			Nodes: args.Nodes,
		}, nil
	}

	klog.Infof("scheduling pod: %s/%s", ns, podName)
	var err error
	for _, predicate := range predicatesByComponent {
		klog.Infof("entering predicate: %s, nodes: %v", predicate.Name(), predicates.GetNodeNames(kubeNodes))
		kubeNodes, err = predicate.Filter(instanceName, pod, kubeNodes)
		klog.Infof("leaving predicate: %s, nodes: %v", predicate.Name(), predicates.GetNodeNames(kubeNodes))
		if err != nil {
			s.recorder.Event(pod, apiv1.EventTypeWarning, predicate.Name(), err.Error())
			if len(kubeNodes) == 0 {
				break
			}
		}
	}

	return &schedulerapiv1.ExtenderFilterResult{
		Nodes: &apiv1.NodeList{Items: kubeNodes},
	}, nil
}

// convertToNodeNameToMetaVictims converts from struct type to meta types.
func convertToNodeNameToMetaVictims(
	nodeToVictims map[string]*schedulerapi.Victims,
) map[string]*schedulerapi.MetaVictims {
	nodeNameToVictims := map[string]*schedulerapi.MetaVictims{}
	for nodeName, victims := range nodeToVictims {
		metaVictims := &schedulerapi.MetaVictims{
			Pods: []*schedulerapi.MetaPod{},
		}
		for _, pod := range victims.Pods {
			metaPod := &schedulerapi.MetaPod{
				UID: string(pod.UID),
			}
			metaVictims.Pods = append(metaVictims.Pods, metaPod)
		}
		nodeNameToVictims[nodeName] = metaVictims
	}
	return nodeNameToVictims
}

// There is a bug in Kubernetes 1.16.0 and before that the JSON tag in
// v1.ExtenderPreemptionArg is wrong. We must use Kubernetes internal types.
// https://github.com/kubernetes/kubernetes/blob/v1.16.0/pkg/scheduler/api/v1/types.go#L270
// TODO use `k8s.io/kubernetes/pkg/scheduler/apis/extender/v1` in 1.17
// TODO use `k8s.io/kube-scheduler/extender/v1` since 1.18
func (s *scheduler) Preempt(args *schedulerapi.ExtenderPreemptionArgs) (*schedulerapi.ExtenderPreemptionResult, error) {
	pod := args.Pod
	ns := pod.GetNamespace()
	podName := pod.GetName()

	var instanceName string
	var exist bool
	if instanceName, exist = pod.Labels[label.InstanceLabelKey]; !exist {
		klog.Warningf("can't find instanceName in pod labels: %s/%s", ns, podName)
		return &schedulerapi.ExtenderPreemptionResult{
			NodeNameToMetaVictims: convertToNodeNameToMetaVictims(args.NodeNameToVictims),
		}, nil
	}

	component, ok := pod.Labels[label.ComponentLabelKey]
	if !ok {
		klog.Warningf("can't find component label in pod labels: %s/%s", ns, podName)
		return &schedulerapi.ExtenderPreemptionResult{
			NodeNameToMetaVictims: convertToNodeNameToMetaVictims(args.NodeNameToVictims),
		}, nil
	}

	predicatesByComponent, ok := s.predicates[component]
	if !ok {
		klog.Warningf("no predicate for component %q, ignored", component)
		return &schedulerapi.ExtenderPreemptionResult{
			NodeNameToMetaVictims: convertToNodeNameToMetaVictims(args.NodeNameToVictims),
		}, nil
	}

	allNodeNames := []string{}
	for nodeName := range args.NodeNameToVictims {
		allNodeNames = append(allNodeNames, nodeName)
	}

	klog.Infof("preempting for pod %s/%s, potential nodes: %s", ns, podName, strings.Join(allNodeNames, ","))

	// extender Filter can't report the failed nodes are unresolvable or not,
	// see https://github.com/kubernetes/kubernetes/issues/91281
	// we need to filter out nodes in Preempt phase
	kubeNodes := make([]apiv1.Node, 0, len(args.NodeNameToVictims))
	for nodeName := range args.NodeNameToVictims {
		// optimize this when we have performance issue
		node, err := s.kubeCli.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		kubeNodes = append(kubeNodes, *node)
	}
	var err error
	for _, predicate := range predicatesByComponent {
		klog.Infof("entering preempt/predicate: %s, nodes: %v", predicate.Name(), predicates.GetNodeNames(kubeNodes))
		kubeNodes, err = predicate.Filter(instanceName, pod, kubeNodes)
		klog.Infof("leaving preempt/predicate: %s, nodes: %v", predicate.Name(), predicates.GetNodeNames(kubeNodes))
		if err != nil {
			return nil, err
		}
	}

	feasibleNodeNameToVictims := map[string]*schedulerapi.Victims{}
	for _, node := range kubeNodes {
		if victims, ok := args.NodeNameToVictims[node.Name]; ok {
			feasibleNodeNameToVictims[node.Name] = victims
		} else {
			return nil, fmt.Errorf("internal error: node %s does not found in args.NodeNameToVictims", node.Name)
		}
	}

	return &schedulerapi.ExtenderPreemptionResult{
		NodeNameToMetaVictims: convertToNodeNameToMetaVictims(feasibleNodeNameToVictims),
	}, nil
}

// FailureError is returned when the FailTiDBSchedulerLabelKey is seen
type FailureError struct {
	PodName string
}

func (ferr FailureError) Error() string {
	return fmt.Sprintf("pod %s had an intentional failure injected", ferr.PodName)
}

// We don't pass `prioritizeVerb` to kubernetes scheduler extender's config file, this method will not be called.
func (s *scheduler) Priority(args *schedulerapiv1.ExtenderArgs) (schedulerapiv1.HostPriorityList, error) {
	result := schedulerapiv1.HostPriorityList{}

	if args.Nodes != nil {
		for _, node := range args.Nodes.Items {
			result = append(result, schedulerapiv1.HostPriority{
				Host:  node.Name,
				Score: 0,
			})
		}
	}

	return result, nil
}

var _ Scheduler = &scheduler{}
