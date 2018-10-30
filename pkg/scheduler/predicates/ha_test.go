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

package predicates

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMapNil(t *testing.T) {
	g := NewGomegaWithT(t)

	m := make(map[string][]string)
	arr := []string{"a", "b"}
	for _, item := range arr {
		m[item] = make([]string, 0)
	}

	g.Expect(m["c"] == nil).To(Equal(true))
	g.Expect(m["a"] == nil).To(Equal(false))
}

func TestHAFilter(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name      string
		ordinal   int32
		podFn     func(string, int32) *apiv1.Pod
		nodesFn   func() []apiv1.Node
		podListFn func(string, string, string) (*apiv1.PodList, error)
		expectFn  func([]apiv1.Node, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		clusterName := "demo"

		pod := test.podFn(clusterName, test.ordinal)
		nodes := test.nodesFn()

		ha := ha{podListFn: test.podListFn}
		test.expectFn(ha.Filter(clusterName, pod, nodes))
	}

	tests := []testcase{
		{
			name:    "component key is empty",
			ordinal: 0,
			podFn: func(clusterName string, ordinal int32) *apiv1.Pod {
				pod := newHAPDPod(clusterName, ordinal)
				pod.Labels = nil
				return pod
			},
			nodesFn:   fakeThreeNodes,
			podListFn: podListFn(map[string][]int32{}),
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(strings.Contains(err.Error(), "can't find component in pod labels")).To(Equal(true))
			},
		},
		{
			name:      "pod list return error",
			ordinal:   0,
			podFn:     newHAPDPod,
			nodesFn:   fakeThreeNodes,
			podListFn: podListErr(),
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(strings.Contains(err.Error(), "pod list error")).To(Equal(true))
			},
		},
		{
			name:    "get ordinal from podName error",
			ordinal: 0,
			podFn: func(clusterName string, ordinal int32) *apiv1.Pod {
				pod := newHAPDPod(clusterName, ordinal)
				pod.Name = "xxxx"
				return pod
			},
			nodesFn:   fakeThreeNodes,
			podListFn: podListFn(map[string][]int32{}),
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(strings.Contains(err.Error(), "strconv.Atoi: parsing")).To(Equal(true))
			},
		},
		{
			name:      "one pod, podName is wrong",
			ordinal:   0,
			podFn:     newHAPDPod,
			nodesFn:   fakeThreeNodes,
			podListFn: podNameWrongListFn(),
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(strings.Contains(err.Error(), "strconv.Atoi: parsing")).To(Equal(true))
			},
		},
		{
			name:      "the lower oridnal is not scheduled",
			ordinal:   1,
			podFn:     newHAPDPod,
			nodesFn:   fakeThreeNodes,
			podListFn: podListFn(map[string][]int32{"": []int32{0}}),
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(strings.Contains(err.Error(), "waiting for pod: default/demo-pd-0")).To(Equal(true))
			},
		},
		{
			name:      "no scheduled pods, three nodes, ordinal 0 should be scheduled to all nodes",
			ordinal:   0,
			podFn:     newHAPDPod,
			nodesFn:   fakeThreeNodes,
			podListFn: podListFn(map[string][]int32{}),
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(3))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-2", "kube-node-3"}))
			},
		},
		{
			name:      "ordinal 0 is scheduled to kube-node-1, ordinal 1 should be scheduled to kube-node-2 or kube-node-3",
			ordinal:   1,
			podFn:     newHAPDPod,
			nodesFn:   fakeThreeNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-1": []int32{0}}),
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(2))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-2", "kube-node-3"}))
			},
		},
		{
			name:      "ordinal 0 is scheduled to kube-node-3, get node-3, ordinal 1 should be scheduled to none",
			ordinal:   1,
			podFn:     newHAPDPod,
			nodesFn:   fakeOneNode,
			podListFn: podListFn(map[string][]int32{"kube-node-3": []int32{0}}),
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "the first 3 pods can't be scheduled to the same node")).To(Equal(true))
			},
		},
		{
			name:      "ordinal 0 is scheduled to kube-node-3, get 0 node, ordinal 1 should be scheduled to none",
			ordinal:   1,
			podFn:     newHAPDPod,
			nodesFn:   fakeZeroNode,
			podListFn: podListFn(map[string][]int32{"kube-node-3": []int32{0}}),
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "no suitable node for pod: default/demo-pd-1")).To(Equal(true))
			},
		},
		{
			name:      "ordinal 0 is scheduled to kube-node-2, ordinal 1 is kube-node-3, ordinal 2 should be scheduled to kube-node-1",
			ordinal:   2,
			podFn:     newHAPDPod,
			nodesFn:   fakeThreeNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-2": []int32{0}, "kube-node-3": []int32{1}}),
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(1))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1"}))
			},
		},
		{
			name:      "ordinal 0 is scheduled to kube-node-1, ordinal 1 is kube-node-3, ordinal 2 should be scheduled to kube-node-1",
			ordinal:   2,
			podFn:     newHAPDPod,
			nodesFn:   fakeTwoNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-1": []int32{0}, "kube-node-3": []int32{1}}),
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "the first 3 pods can't be scheduled to the same node")).To(Equal(true))
			},
		},
		{
			name:      "ordinal 0 is scheduled to kube-node-1, ordinal 1 is kube-node-3, get 0 node, ordinal 2 should be scheduled to none",
			ordinal:   2,
			podFn:     newHAPDPod,
			nodesFn:   fakeZeroNode,
			podListFn: podListFn(map[string][]int32{"kube-node-1": []int32{0}, "kube-node-3": []int32{1}}),
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "no suitable node for pod: default/demo-pd-2")).To(Equal(true))
			},
		},
		{
			name:      "the first three oridnals get to 3 nodes, the ordinal 3 should scheduled to 1,2,3",
			ordinal:   3,
			podFn:     newHAPDPod,
			nodesFn:   fakeThreeNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-2": []int32{0}, "kube-node-3": []int32{1}, "kube-node-1": []int32{2}}),
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(3))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-2", "kube-node-3"}))
			},
		},
		{
			name:      "the first four oridnals get to 3 nodes, the ordinal 4 should scheduled to 2,3",
			ordinal:   4,
			podFn:     newHAPDPod,
			nodesFn:   fakeThreeNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-2": []int32{0}, "kube-node-3": []int32{1}, "kube-node-1": []int32{2, 3}}),
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(2))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-2", "kube-node-3"}))
			},
		},
		{
			name:      "the first five oridnals get to 3 nodes, the ordinal 5 should scheduled to 2",
			ordinal:   5,
			podFn:     newHAPDPod,
			nodesFn:   fakeThreeNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-2": []int32{0}, "kube-node-3": []int32{1, 4}, "kube-node-1": []int32{2, 3}}),
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(1))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-2"}))
			},
		},
		{
			name:      "the first five oridnals get to 3 nodes, got 2 nodes(no node-2), the ordinal 5 should scheduled to 1,3",
			ordinal:   5,
			podFn:     newHAPDPod,
			nodesFn:   fakeTwoNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-2": []int32{0}, "kube-node-3": []int32{1, 4}, "kube-node-1": []int32{2, 3}}),
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(2))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-3"}))
			},
		},
		{
			name:      "the first five oridnals get to 3 nodes, got 1 nodes(no node-2 node-1), the ordinal 5 should scheduled to 3",
			ordinal:   5,
			podFn:     newHAPDPod,
			nodesFn:   fakeOneNode,
			podListFn: podListFn(map[string][]int32{"kube-node-2": []int32{0}, "kube-node-3": []int32{1, 4}, "kube-node-1": []int32{2, 3}}),
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(1))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-3"}))
			},
		},
		{
			name:      "the first five oridnals get to 3 nodes, got 0 nodes, the ordinal 5 should scheduled to none",
			ordinal:   5,
			podFn:     newHAPDPod,
			nodesFn:   fakeZeroNode,
			podListFn: podListFn(map[string][]int32{"kube-node-2": []int32{0}, "kube-node-3": []int32{1, 4}, "kube-node-1": []int32{2, 3}}),
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "no suitable node for pod: default/demo-pd-5")).To(Equal(true))
			},
		},
		{
			name:      "the first six oridnals get to 3 nodes, the ordinal 6 should scheduled to 1,2,3",
			ordinal:   6,
			podFn:     newHAPDPod,
			nodesFn:   fakeThreeNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-2": []int32{0, 5}, "kube-node-3": []int32{1, 4}, "kube-node-1": []int32{2, 3}}),
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(3))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-2", "kube-node-3"}))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newHAPDPod(clusterName string, ordinal int32) *apiv1.Pod {
	return &apiv1.Pod{
		TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", controller.PDMemberName(clusterName), ordinal),
			Namespace: corev1.NamespaceDefault,
			Labels:    label.New().PD().Labels(),
		},
	}
}

func podListFn(nodePodMap map[string][]int32) func(string, string, string) (*apiv1.PodList, error) {
	return func(ns, clusterName, component string) (*apiv1.PodList, error) {
		podList := &apiv1.PodList{
			TypeMeta: metav1.TypeMeta{Kind: "PodList", APIVersion: "v1"},
			Items:    []apiv1.Pod{},
		}
		for nodeName, podsOrdinalArr := range nodePodMap {
			for _, podOrdinal := range podsOrdinalArr {
				podList.Items = append(podList.Items, apiv1.Pod{
					TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%d", controller.PDMemberName(clusterName), podOrdinal),
						Namespace: corev1.NamespaceDefault,
						Labels:    label.New().PD().Labels(),
					},
					Spec: apiv1.PodSpec{
						NodeName: nodeName,
					},
				})
			}
		}
		return podList, nil
	}
}

func podNameWrongListFn() func(string, string, string) (*apiv1.PodList, error) {
	return func(ns, clusterName, component string) (*apiv1.PodList, error) {
		podList := &apiv1.PodList{
			TypeMeta: metav1.TypeMeta{Kind: "PodList", APIVersion: "v1"},
			Items: []apiv1.Pod{
				{
					TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "xxx",
						Namespace: corev1.NamespaceDefault,
						Labels:    label.New().PD().Labels(),
					},
					Spec: apiv1.PodSpec{
						NodeName: "kube-node-1",
					},
				},
			},
		}
		return podList, nil
	}
}

func podListErr() func(string, string, string) (*apiv1.PodList, error) {
	return func(ns, clusterName, component string) (*apiv1.PodList, error) {
		return nil, errors.New("pod list error")
	}
}

func getSortedNodeNames(nodes []apiv1.Node) []string {
	arr := make([]string, 0)
	for _, node := range nodes {
		arr = append(arr, node.GetName())
	}
	sort.Strings(arr)
	return arr
}
