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
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/scheduler/predicates"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/record"
	schedulerapi "k8s.io/kubernetes/pkg/scheduler/api"
	schedulerapiv1 "k8s.io/kubernetes/pkg/scheduler/api/v1"
)

func TestSchedulerFilter(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name      string
		args      *schedulerapiv1.ExtenderArgs
		predicate predicates.Predicate
		expectFn  func(*GomegaWithT, *schedulerapiv1.ExtenderFilterResult, error)
	}

	recorder := record.NewFakeRecorder(10)

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		s := &scheduler{
			predicates: map[string][]predicates.Predicate{
				label.PDLabelVal: {
					test.predicate,
				},
				label.TiKVLabelVal: {
					test.predicate,
				},
				label.TiDBLabelVal: {
					test.predicate,
				},
			},

			recorder: recorder,
		}
		re, err := s.Filter(test.args)
		test.expectFn(g, re, err)
	}

	tests := []testcase{
		{
			name: "pod instance label is empty",
			args: &schedulerapiv1.ExtenderArgs{
				Pod: &apiv1.Pod{
					TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: corev1.NamespaceDefault,
					},
				},
				Nodes: &apiv1.NodeList{
					TypeMeta: metav1.TypeMeta{Kind: "NodeList", APIVersion: "v1"},
					ListMeta: metav1.ListMeta{ResourceVersion: "9999"},
					Items:    []apiv1.Node{},
				},
			},
			predicate: &predicates.FakePredicate{},
			expectFn: func(g *GomegaWithT, result *schedulerapiv1.ExtenderFilterResult, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(result.Nodes.ResourceVersion).To(Equal("9999"))
				g.Expect(len(result.Nodes.Items)).To(Equal(0))
			},
		},
		{
			name: "pod is not pd or tikv or tidb",
			args: &schedulerapiv1.ExtenderArgs{
				Pod: &apiv1.Pod{
					TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: corev1.NamespaceDefault,
						Labels: map[string]string{
							label.InstanceLabelKey:  "tc-1",
							label.ComponentLabelKey: "other",
						},
					},
				},
				Nodes: &apiv1.NodeList{
					TypeMeta: metav1.TypeMeta{Kind: "NodeList", APIVersion: "v1"},
					ListMeta: metav1.ListMeta{ResourceVersion: "9999"},
					Items:    []apiv1.Node{},
				},
			},
			predicate: &predicates.FakePredicate{},
			expectFn: func(g *GomegaWithT, result *schedulerapiv1.ExtenderFilterResult, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(result.Nodes.ResourceVersion).To(Equal("9999"))
				g.Expect(len(result.Nodes.Items)).To(Equal(0))
			},
		},
		{
			name: "predicate returns error",
			args: &schedulerapiv1.ExtenderArgs{
				Pod: &apiv1.Pod{
					TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: corev1.NamespaceDefault,
						Labels: map[string]string{
							label.InstanceLabelKey:  "tc-1",
							label.ComponentLabelKey: "pd",
						},
					},
				},
				Nodes: &apiv1.NodeList{
					TypeMeta: metav1.TypeMeta{Kind: "NodeList", APIVersion: "v1"},
					ListMeta: metav1.ListMeta{ResourceVersion: "9999"},
					Items:    []apiv1.Node{},
				},
			},
			predicate: &predicates.FakePredicate{Err: fmt.Errorf("predicate error")},
			expectFn: func(g *GomegaWithT, result *schedulerapiv1.ExtenderFilterResult, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				events := predicates.CollectEvents(recorder.Events)
				g.Expect(events).To(HaveLen(1))
				g.Expect(events[0]).To(ContainSubstring("predicate error"))
				g.Expect(result.Nodes.Items).To(BeNil())
			},
		},
		{
			name: "predicate success",
			args: &schedulerapiv1.ExtenderArgs{
				Pod: &apiv1.Pod{
					TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: corev1.NamespaceDefault,
						Labels: map[string]string{
							label.InstanceLabelKey:  "tc-1",
							label.ComponentLabelKey: "pd",
						},
					},
				},
				Nodes: &apiv1.NodeList{
					TypeMeta: metav1.TypeMeta{Kind: "NodeList", APIVersion: "v1"},
					ListMeta: metav1.ListMeta{ResourceVersion: "9999"},
					Items: []apiv1.Node{
						{
							TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Name: "node-1",
							},
						},
					},
				},
			},
			predicate: &predicates.FakePredicate{},
			expectFn: func(g *GomegaWithT, result *schedulerapiv1.ExtenderFilterResult, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(result.Nodes.Items[0].Name).To(Equal("node-1"))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestSchedulerPriority(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name     string
		args     *schedulerapiv1.ExtenderArgs
		expectFn func(*GomegaWithT, schedulerapiv1.HostPriorityList, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		s := scheduler{}
		re, err := s.Priority(test.args)
		test.expectFn(g, re, err)
	}

	tests := []testcase{
		{
			name: "nodes is nil",
			args: &schedulerapiv1.ExtenderArgs{
				Nodes: nil,
			},
			expectFn: func(g *GomegaWithT, result schedulerapiv1.HostPriorityList, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(result)).To(Equal(0))
			},
		},
		{
			name: "have 1 node",
			args: &schedulerapiv1.ExtenderArgs{
				Nodes: &apiv1.NodeList{
					TypeMeta: metav1.TypeMeta{Kind: "NodeList", APIVersion: "v1"},
					ListMeta: metav1.ListMeta{},
					Items: []apiv1.Node{
						{
							TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Name: "node-1",
							},
						},
					},
				},
			},
			expectFn: func(g *GomegaWithT, result schedulerapiv1.HostPriorityList, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(result)).To(Equal(1))
				g.Expect(result[0].Host).To(Equal("node-1"))
				g.Expect(result[0].Score).To(Equal(0))
			},
		},
		{
			name: "have 2 nodes",
			args: &schedulerapiv1.ExtenderArgs{
				Nodes: &apiv1.NodeList{
					TypeMeta: metav1.TypeMeta{Kind: "NodeList", APIVersion: "v1"},
					ListMeta: metav1.ListMeta{},
					Items: []apiv1.Node{
						{
							TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Name: "node-1",
							},
						},
						{
							TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Name: "node-2",
							},
						},
					},
				},
			},
			expectFn: func(g *GomegaWithT, result schedulerapiv1.HostPriorityList, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(result)).To(Equal(2))
				g.Expect(result[0].Host).To(Equal("node-1"))
				g.Expect(result[0].Score).To(Equal(0))
				g.Expect(result[1].Host).To(Equal("node-2"))
				g.Expect(result[1].Score).To(Equal(0))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestSchedulerPreempt(t *testing.T) {
	victims := &schedulerapi.Victims{
		Pods: []*apiv1.Pod{},
	}
	metaVictims := &schedulerapi.MetaVictims{
		Pods: []*schedulerapi.MetaPod{},
	}
	nodeA := &apiv1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-a",
		},
	}
	nodeB := &apiv1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-b",
		},
	}
	nodeC := &apiv1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-c",
		},
	}
	unrelatedPod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: apiv1.NamespaceDefault,
			Name:      "test",
		},
	}
	pdPod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: apiv1.NamespaceDefault,
			Name:      "test",
			Labels: map[string]string{
				label.InstanceLabelKey:  "test",
				label.ComponentLabelKey: "pd",
			},
		},
	}
	tests := []struct {
		name       string
		nodes      []*apiv1.Node
		args       *schedulerapi.ExtenderPreemptionArgs
		predicates map[string][]predicates.Predicate
		wantResult *schedulerapi.ExtenderPreemptionResult
		wantErr    string
	}{
		{
			name:  "unrelated pod",
			nodes: []*apiv1.Node{nodeA, nodeB, nodeC},
			args: &schedulerapi.ExtenderPreemptionArgs{
				Pod: unrelatedPod,
				NodeNameToVictims: map[string]*schedulerapi.Victims{
					"node-a": victims,
					"node-b": victims,
					"node-c": victims,
				},
			},
			wantResult: &schedulerapi.ExtenderPreemptionResult{
				NodeNameToMetaVictims: map[string]*schedulerapi.MetaVictims{
					"node-a": metaVictims,
					"node-b": metaVictims,
					"node-c": metaVictims,
				},
			},
		},
		{
			name:  "node does not exist anymore",
			nodes: []*apiv1.Node{nodeA, nodeB},
			predicates: map[string][]predicates.Predicate{
				label.PDLabelVal: {
					&predicates.FakePredicate{},
				},
			},
			args: &schedulerapi.ExtenderPreemptionArgs{
				Pod: pdPod,
				NodeNameToVictims: map[string]*schedulerapi.Victims{
					"node-a": victims,
					"node-b": victims,
					"node-c": victims,
				},
			},
			wantResult: nil,
			wantErr:    `nodes "node-c" not found`,
		},
		{
			name:  "one of nominated nodes is feasible",
			nodes: []*apiv1.Node{nodeA, nodeB, nodeC},
			predicates: map[string][]predicates.Predicate{
				label.PDLabelVal: {
					&predicates.FakePredicate{
						Nodes: []apiv1.Node{
							*nodeA,
						},
					},
				},
			},
			args: &schedulerapi.ExtenderPreemptionArgs{
				Pod: pdPod,
				NodeNameToVictims: map[string]*schedulerapi.Victims{
					"node-a": victims,
					"node-b": victims,
					"node-c": victims,
				},
			},
			wantResult: &schedulerapi.ExtenderPreemptionResult{
				NodeNameToMetaVictims: map[string]*schedulerapi.MetaVictims{
					"node-a": metaVictims,
				},
			},
		},
		{
			name:  "none of nominated nodes is feasible",
			nodes: []*apiv1.Node{nodeA, nodeB, nodeC},
			predicates: map[string][]predicates.Predicate{
				label.PDLabelVal: {
					&predicates.FakePredicate{
						Nodes: []apiv1.Node{},
					},
				},
			},
			args: &schedulerapi.ExtenderPreemptionArgs{
				Pod: pdPod,
				NodeNameToVictims: map[string]*schedulerapi.Victims{
					"node-a": victims,
					"node-b": victims,
					"node-c": victims,
				},
			},
			wantResult: &schedulerapi.ExtenderPreemptionResult{
				NodeNameToMetaVictims: map[string]*schedulerapi.MetaVictims{},
			},
		},
		{
			name:  "all nominated nodes are feasible",
			nodes: []*apiv1.Node{nodeA, nodeB, nodeC},
			predicates: map[string][]predicates.Predicate{
				label.PDLabelVal: {
					&predicates.FakePredicate{
						Nodes: []apiv1.Node{
							*nodeA,
							*nodeB,
							*nodeC,
						},
					},
				},
			},
			args: &schedulerapi.ExtenderPreemptionArgs{
				Pod: pdPod,
				NodeNameToVictims: map[string]*schedulerapi.Victims{
					"node-a": victims,
					"node-b": victims,
					"node-c": victims,
				},
			},
			wantResult: &schedulerapi.ExtenderPreemptionResult{
				NodeNameToMetaVictims: map[string]*schedulerapi.MetaVictims{
					"node-a": metaVictims,
					"node-b": metaVictims,
					"node-c": metaVictims,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeCli := fake.NewSimpleClientset()
			s := &scheduler{
				kubeCli:  kubeCli,
				recorder: record.NewFakeRecorder(10),
			}
			if tt.predicates != nil {
				s.predicates = tt.predicates
			}
			if tt.nodes != nil {
				for _, node := range tt.nodes {
					kubeCli.CoreV1().Nodes().Create(node)
				}
			}
			kubeCli.CoreV1().Pods(apiv1.NamespaceDefault).Create(tt.args.Pod)
			result, err := s.Preempt(tt.args)
			if diff := cmp.Diff(tt.wantResult, result); diff != "" {
				t.Errorf("unexpected (-want, +got): %s", diff)
			}
			if tt.wantErr == "" && err != nil {
				t.Errorf("expects error is nil, got %v", err)
			}
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expects error is %v, got %v", tt.wantErr, err)
				}
			}
		})
	}
}
