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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMapAndIntNil(t *testing.T) {
	g := NewGomegaWithT(t)

	m := make(map[string][]string)
	arr := []string{"a", "b"}
	for _, item := range arr {
		m[item] = make([]string, 0)
	}

	g.Expect(m["c"] == nil).To(Equal(true))
	g.Expect(m["a"] == nil).To(Equal(false))

	var i *int
	g.Expect(i).To(BeNil())
}

func TestSortEmptyArr(t *testing.T) {
	g := NewGomegaWithT(t)

	arr := make([]int, 0)
	sort.Ints(arr)

	g.Expect(len(arr)).To(BeZero())
}

func TestHAFilter(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name      string
		podFn     func(string, string, int32) *apiv1.Pod
		nodesFn   func() []apiv1.Node
		podListFn func(string, string, string) (*apiv1.PodList, error)
		pvcGetFn  func(string, string) (*apiv1.PersistentVolumeClaim, error)
		tcGetFn   func(string, string) (*v1alpha1.TidbCluster, error)
		expectFn  func([]apiv1.Node, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		instanceName := "demo"
		clusterName := "cluster-1"

		pod := test.podFn(instanceName, clusterName, 0)
		nodes := test.nodesFn()

		ha := ha{podListFn: test.podListFn, pvcGetFn: test.pvcGetFn, tcGetFn: test.tcGetFn}
		test.expectFn(ha.Filter(instanceName, pod, nodes))
	}

	tests := []testcase{
		{
			name:    "one node, one scheduled pod recreated and its pvc is bound, return the node",
			podFn:   newHAPDPod,
			nodesFn: fakeOneNode,
			pvcGetFn: func(ns string, pvcName string) (*corev1.PersistentVolumeClaim, error) {
				return &corev1.PersistentVolumeClaim{
					TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "pd-cluster-1-pd-0"},
					Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
				}, nil
			},
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(1))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1"}))
			},
		},
		{
			name:    "already scheduled pod recreated, get pvc failed",
			podFn:   newHAPDPod,
			nodesFn: fakeOneNode,
			pvcGetFn: func(ns string, pvcName string) (*corev1.PersistentVolumeClaim, error) {
				return nil, fmt.Errorf("get pvc failed")
			},
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "get pvc failed")).To(BeTrue())
			},
		},
		{
			name:      "list pod failed",
			podFn:     newHAPDPod,
			nodesFn:   fakeThreeNodes,
			podListFn: podListErr(),
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "list pods failed")).To(BeTrue())
			},
		},
		{
			name:      "get tidbcluster failed",
			podFn:     newHAPDPod,
			nodesFn:   fakeThreeNodes,
			podListFn: podListFn(map[string][]int32{}),
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				return nil, fmt.Errorf("get tidbcluster failed")
			},
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "get tidbcluster failed")).To(BeTrue())
			},
		},
		{
			name:      "zero node, return zero node",
			podFn:     newHAPDPod,
			nodesFn:   fakeZeroNode,
			podListFn: podListFn(map[string][]int32{}),
			tcGetFn:   tcGetFn,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "kube nodes is empty")).To(BeTrue())
			},
		},
		{
			name:    "one node, no pod scheduled, return the node",
			podFn:   newHAPDPod,
			nodesFn: fakeOneNode,
			pvcGetFn: func(ns string, pvcName string) (*corev1.PersistentVolumeClaim, error) {
				return &corev1.PersistentVolumeClaim{
					TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "pd-cluster-1-pd-0"},
					Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending},
				}, nil
			},
			podListFn: podListFn(map[string][]int32{}),
			tcGetFn:   tcGetFn,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(1))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1"}))
			},
		},
		{
			name:    "one node, one pod scheduled, return zero node",
			podFn:   newHAPDPod,
			nodesFn: fakeOneNode,
			pvcGetFn: func(ns string, pvcName string) (*corev1.PersistentVolumeClaim, error) {
				return &corev1.PersistentVolumeClaim{
					TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "pd-cluster-1-pd-0"},
					Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending},
				}, nil
			},
			podListFn: podListFn(map[string][]int32{"kube-node-1": {1}}),
			tcGetFn:   tcGetFn,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "can't find a node from: ")).To(BeTrue())
			},
		},
		{
			name:      "two nodes, one pod scheduled on the node, return one node",
			podFn:     newHAPDPod,
			nodesFn:   fakeTwoNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-1": {0}}),
			tcGetFn:   tcGetFn,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(1))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-2"}))
			},
		},
		{
			name:      "two nodes, two pods scheduled on these two nodes, return zero node",
			podFn:     newHAPDPod,
			nodesFn:   fakeTwoNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}}),
			tcGetFn:   tcGetFn,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "can't find a node from: ")).To(BeTrue())
				g.Expect(len(nodes)).To(Equal(0))
			},
		},
		{
			name:      "three nodes, zero pod scheduled, return all the three nodes",
			podFn:     newHAPDPod,
			nodesFn:   fakeThreeNodes,
			podListFn: podListFn(map[string][]int32{}),
			tcGetFn:   tcGetFn,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(3))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-2", "kube-node-3"}))
			},
		},
		{
			name:      "three nodes, one pod scheduled, return two nodes",
			podFn:     newHAPDPod,
			nodesFn:   fakeThreeNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-1": {0}}),
			tcGetFn:   tcGetFn,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(2))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-2", "kube-node-3"}))
			},
		},
		{
			name:      "three nodes, two pods scheduled, return one node",
			podFn:     newHAPDPod,
			nodesFn:   fakeThreeNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}}),
			tcGetFn:   tcGetFn,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(1))
				g.Expect(nodes[0].Name).To(Equal("kube-node-3"))
			},
		},
		{
			name:      "three nodes, one pod not scheduled on these three nodes, return all the three nodes",
			podFn:     newHAPDPod,
			nodesFn:   fakeThreeNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-4": {4}}),
			tcGetFn:   tcGetFn,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(3))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-2", "kube-node-3"}))
			},
		},
		{
			name:      "three nodes, three pods scheduled on these three nodes, replicas is 4, can't scheduled",
			podFn:     newHAPDPod,
			nodesFn:   fakeThreeNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}, "kube-node-3": {2}}),
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.PD.Replicas = 4
				return tc, nil
			},
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "can't find a node from: ")).To(BeTrue())
				g.Expect(len(nodes)).To(Equal(0))
			},
		},
		{
			name:      "three nodes, three pods scheduled on these three nodes, replicas is 5, return three nodes",
			podFn:     newHAPDPod,
			nodesFn:   fakeThreeNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}, "kube-node-3": {2}}),
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.PD.Replicas = 5
				return tc, nil
			},
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(3))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-2", "kube-node-3"}))
			},
		},
		{
			name:      "three nodes, four pods scheduled on these three nodes, replicas is 5, return two nodes",
			podFn:     newHAPDPod,
			nodesFn:   fakeThreeNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-1": {0, 3}, "kube-node-2": {1}, "kube-node-3": {2}}),
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.PD.Replicas = 5
				return tc, nil
			},
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(2))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-2", "kube-node-3"}))
			},
		},
		{
			name:      "four nodes, three pods scheduled on these three nodes, replicas is 4, return the fourth node",
			podFn:     newHAPDPod,
			nodesFn:   fakeFourNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}, "kube-node-3": {2}}),
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.PD.Replicas = 4
				return tc, nil
			},
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(1))
				g.Expect(nodes[0].Name).To(Equal("kube-node-4"))
			},
		},
		{
			name:      "four nodes, four pods scheduled on these four nodes, replicas is 5, return these four nodes",
			podFn:     newHAPDPod,
			nodesFn:   fakeFourNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}, "kube-node-3": {2}, "kube-node-4": {3}}),
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.PD.Replicas = 5
				return tc, nil
			},
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(4))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-2", "kube-node-3", "kube-node-4"}))
			},
		},
		{
			name:      "four nodes, four pods scheduled on these four nodes, replicas is 6, return these four nodes",
			podFn:     newHAPDPod,
			nodesFn:   fakeFourNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}, "kube-node-3": {2}, "kube-node-4": {3}}),
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.PD.Replicas = 6
				return tc, nil
			},
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(4))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-2", "kube-node-3", "kube-node-4"}))
			},
		},
		{
			name:      "four nodes, five pods scheduled on these four nodes, replicas is 6, return these three nodes",
			podFn:     newHAPDPod,
			nodesFn:   fakeFourNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-1": {0, 4}, "kube-node-2": {1}, "kube-node-3": {2}, "kube-node-4": {3}}),
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.PD.Replicas = 6
				return tc, nil
			},
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(3))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-2", "kube-node-3", "kube-node-4"}))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newHAPDPod(instanceName, clusterName string, ordinal int32) *apiv1.Pod {
	return &apiv1.Pod{
		TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", controller.PDMemberName(clusterName), ordinal),
			Namespace: corev1.NamespaceDefault,
			Labels:    label.New().Instance(instanceName).PD().Labels(),
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

func podListErr() func(string, string, string) (*apiv1.PodList, error) {
	return func(ns, clusterName, component string) (*apiv1.PodList, error) {
		return nil, errors.New("list pods failed")
	}
}

func tcGetFn(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{Kind: "TidbCluster", APIVersion: "v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      tcName,
			Namespace: ns,
		},
		Spec: v1alpha1.TidbClusterSpec{
			PD: v1alpha1.PDSpec{Replicas: 3},
		},
	}, nil
}

func getSortedNodeNames(nodes []apiv1.Node) []string {
	arr := make([]string, 0)
	for _, node := range nodes {
		arr = append(arr, node.GetName())
	}
	sort.Strings(arr)
	return arr
}
