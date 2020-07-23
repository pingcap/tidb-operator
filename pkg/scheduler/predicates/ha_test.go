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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
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
}

func TestHARealAcquireLockFn(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name        string
		podFn       func(string, string, int32) *apiv1.Pod
		podGetFn    func(string, string) (*apiv1.Pod, error)
		pvcListFn   func(ns, instanceName, component string) (*apiv1.PersistentVolumeClaimList, error)
		updatePVCFn func(*apiv1.PersistentVolumeClaim) error
		expectFn    func(*apiv1.PersistentVolumeClaim, *apiv1.PersistentVolumeClaim, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		instanceName := "demo"
		clusterName := "cluster-1"

		ha := ha{
			pvcListFn:   test.pvcListFn,
			updatePVCFn: test.updatePVCFn,
			podGetFn:    podGetScheduled(),
		}
		if test.podGetFn != nil {
			ha.podGetFn = test.podGetFn
		}
		pod := test.podFn(instanceName, clusterName, 0)

		test.expectFn(ha.realAcquireLock(pod))
	}

	tests := []testcase{
		{
			name:  "pvcListFn failed",
			podFn: newHAPDPod,
			pvcListFn: func(ns, instanceName, component string) (*corev1.PersistentVolumeClaimList, error) {
				return nil, fmt.Errorf("failed to list pvc")
			},
			expectFn: func(schedulingPVC, currentPVC *apiv1.PersistentVolumeClaim, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "failed to list pvc")).To(BeTrue())
			},
		},
		{
			name:  "can't find current pvc",
			podFn: newHAPDPod,
			pvcListFn: func(ns, instanceName, component string) (*corev1.PersistentVolumeClaimList, error) {
				return &corev1.PersistentVolumeClaimList{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaimList", APIVersion: "v1"},
					Items: []corev1.PersistentVolumeClaim{
						{
							TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Namespace: metav1.NamespaceDefault,
								Name:      "pd-cluster-1-pd-1",
							},
						},
						{
							TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Namespace: metav1.NamespaceDefault,
								Name:      "pd-cluster-1-pd-2",
							},
						},
					},
				}, nil
			},
			expectFn: func(schedulingPVC, currentPVC *apiv1.PersistentVolumeClaim, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "can't find current Pod")).To(BeTrue())
			},
		},
		{
			name:  "no scheduling pod, setCurrentPodScheduling success",
			podFn: newHAPDPod,
			pvcListFn: func(ns, instanceName, component string) (*corev1.PersistentVolumeClaimList, error) {
				return &corev1.PersistentVolumeClaimList{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaimList", APIVersion: "v1"},
					Items: []corev1.PersistentVolumeClaim{
						{
							TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Namespace: metav1.NamespaceDefault,
								Name:      "pd-cluster-1-pd-0",
							},
						},
						{
							TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Namespace: metav1.NamespaceDefault,
								Name:      "pd-cluster-1-pd-1",
							},
						},
					},
				}, nil
			},
			updatePVCFn: func(claim *corev1.PersistentVolumeClaim) error {
				return nil
			},
			expectFn: func(schedulingPVC, currentPVC *apiv1.PersistentVolumeClaim, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(schedulingPVC).To(BeNil())
				g.Expect(currentPVC.Annotations[label.AnnPVCPodScheduling]).NotTo(BeEmpty())
			},
		},
		{
			name:  "no scheduling pod, setCurrentPodScheduling failed",
			podFn: newHAPDPod,
			pvcListFn: func(ns, instanceName, component string) (*corev1.PersistentVolumeClaimList, error) {
				return &corev1.PersistentVolumeClaimList{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaimList", APIVersion: "v1"},
					Items: []corev1.PersistentVolumeClaim{
						{
							TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Namespace: metav1.NamespaceDefault,
								Name:      "pd-cluster-1-pd-0",
							},
						},
						{
							TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Namespace: metav1.NamespaceDefault,
								Name:      "pd-cluster-1-pd-1",
							},
						},
					},
				}, nil
			},
			updatePVCFn: func(claim *corev1.PersistentVolumeClaim) error {
				return fmt.Errorf("setCurrentPodScheduling failed")
			},
			expectFn: func(schedulingPVC, currentPVC *apiv1.PersistentVolumeClaim, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(schedulingPVC).To(BeNil())
				g.Expect(strings.Contains(err.Error(), "setCurrentPodScheduling failed")).To(BeTrue())
			},
		},
		{
			name:  "current pvc is scheduling",
			podFn: newHAPDPod,
			pvcListFn: func(ns, instanceName, component string) (*corev1.PersistentVolumeClaimList, error) {
				return &corev1.PersistentVolumeClaimList{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaimList", APIVersion: "v1"},
					Items: []corev1.PersistentVolumeClaim{
						{
							TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Namespace:   metav1.NamespaceDefault,
								Name:        "pd-cluster-1-pd-0",
								Annotations: map[string]string{label.AnnPVCPodScheduling: "true"},
							},
						},
						{
							TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Namespace: metav1.NamespaceDefault,
								Name:      "pd-cluster-1-pd-1",
							},
						},
					},
				}, nil
			},
			expectFn: func(schedulingPVC, currentPVC *apiv1.PersistentVolumeClaim, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(schedulingPVC).To(Equal(currentPVC))
			},
		},
		{
			name:  "get scheduling pvc's pod error",
			podFn: newHAPDPod,
			pvcListFn: func(ns, instanceName, component string) (*corev1.PersistentVolumeClaimList, error) {
				return &corev1.PersistentVolumeClaimList{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaimList", APIVersion: "v1"},
					Items: []corev1.PersistentVolumeClaim{
						{
							TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Namespace: metav1.NamespaceDefault,
								Name:      "pd-cluster-1-pd-0",
							},
						},
						{
							TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Namespace:   metav1.NamespaceDefault,
								Name:        "pd-cluster-1-pd-1",
								Annotations: map[string]string{label.AnnPVCPodScheduling: "true"},
							},
							Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
						},
					},
				}, nil
			},
			podGetFn: podGetErr(),
			expectFn: func(schedulingPVC, currentPVC *apiv1.PersistentVolumeClaim, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(schedulingPVC.Annotations[label.AnnPVCPodScheduling]).NotTo(BeEmpty())
				g.Expect(strings.Contains(err.Error(), "get pod failed")).To(BeTrue())
			},
		},
		{
			name:  "scheduling pvc is not bound",
			podFn: newHAPDPod,
			pvcListFn: func(ns, instanceName, component string) (*corev1.PersistentVolumeClaimList, error) {
				return &corev1.PersistentVolumeClaimList{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaimList", APIVersion: "v1"},
					Items: []corev1.PersistentVolumeClaim{
						{
							TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Namespace: metav1.NamespaceDefault,
								Name:      "pd-cluster-1-pd-0",
							},
						},
						{
							TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Namespace:   metav1.NamespaceDefault,
								Name:        "pd-cluster-1-pd-1",
								Annotations: map[string]string{label.AnnPVCPodScheduling: "true"},
							},
							Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending},
						},
					},
				}, nil
			},
			expectFn: func(schedulingPVC, currentPVC *apiv1.PersistentVolumeClaim, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "waiting for Pod ")).To(BeTrue())
			},
		},
		{
			name:  "scheduling pvc is bound, but pod not scheduled",
			podFn: newHAPDPod,
			pvcListFn: func(ns, instanceName, component string) (*corev1.PersistentVolumeClaimList, error) {
				return &corev1.PersistentVolumeClaimList{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaimList", APIVersion: "v1"},
					Items: []corev1.PersistentVolumeClaim{
						{
							TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Namespace: metav1.NamespaceDefault,
								Name:      "pd-cluster-1-pd-0",
							},
						},
						{
							TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Namespace:   metav1.NamespaceDefault,
								Name:        "pd-cluster-1-pd-1",
								Annotations: map[string]string{label.AnnPVCPodScheduling: "true"},
							},
							Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
						},
					},
				}, nil
			},
			podGetFn: podGetNotScheduled(),
			updatePVCFn: func(claim *corev1.PersistentVolumeClaim) error {
				return nil
			},
			expectFn: func(schedulingPVC, currentPVC *apiv1.PersistentVolumeClaim, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(schedulingPVC.Annotations[label.AnnPVCPodScheduling]).NotTo(BeEmpty())
				g.Expect(strings.Contains(err.Error(), "waiting for Pod ")).To(BeTrue())
			},
		},
		{
			name:  "scheduling pvc is bound, update pvc failed",
			podFn: newHAPDPod,
			pvcListFn: func(ns, instanceName, component string) (*corev1.PersistentVolumeClaimList, error) {
				return &corev1.PersistentVolumeClaimList{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaimList", APIVersion: "v1"},
					Items: []corev1.PersistentVolumeClaim{
						{
							TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Namespace: metav1.NamespaceDefault,
								Name:      "pd-cluster-1-pd-0",
							},
						},
						{
							TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Namespace:   metav1.NamespaceDefault,
								Name:        "pd-cluster-1-pd-1",
								Annotations: map[string]string{label.AnnPVCPodScheduling: "true"},
							},
							Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
						},
					},
				}, nil
			},
			updatePVCFn: func(claim *corev1.PersistentVolumeClaim) error {
				return fmt.Errorf("failed to update pvc")
			},
			expectFn: func(schedulingPVC, currentPVC *apiv1.PersistentVolumeClaim, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(schedulingPVC.Annotations[label.AnnPVCPodScheduling]).To(BeEmpty())
				g.Expect(strings.Contains(err.Error(), "failed to update pvc")).To(BeTrue())
			},
		},
		{
			name:  "scheduling pvc is bound, update success",
			podFn: newHAPDPod,
			pvcListFn: func(ns, instanceName, component string) (*corev1.PersistentVolumeClaimList, error) {
				return &corev1.PersistentVolumeClaimList{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaimList", APIVersion: "v1"},
					Items: []corev1.PersistentVolumeClaim{
						{
							TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Namespace: metav1.NamespaceDefault,
								Name:      "pd-cluster-1-pd-0",
							},
						},
						{
							TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Namespace:   metav1.NamespaceDefault,
								Name:        "pd-cluster-1-pd-1",
								Annotations: map[string]string{label.AnnPVCPodScheduling: "true"},
							},
							Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
						},
					},
				}, nil
			},
			updatePVCFn: func(claim *corev1.PersistentVolumeClaim) error {
				return nil
			},
			expectFn: func(schedulingPVC, currentPVC *apiv1.PersistentVolumeClaim, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(schedulingPVC.Annotations[label.AnnPVCPodScheduling]).To(BeEmpty())
				g.Expect(currentPVC.Annotations[label.AnnPVCPodScheduling]).NotTo(BeEmpty())
			},
		},
		{
			name:  "scheduling pvc is defer deleting, current pvc acquire lock",
			podFn: newHAPDPod,
			pvcListFn: func(ns, instanceName, component string) (*corev1.PersistentVolumeClaimList, error) {
				return &corev1.PersistentVolumeClaimList{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaimList", APIVersion: "v1"},
					Items: []corev1.PersistentVolumeClaim{
						{
							TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Namespace: metav1.NamespaceDefault,
								Name:      "pd-cluster-1-pd-0",
							},
						},
						{
							TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
							ObjectMeta: metav1.ObjectMeta{
								Namespace: metav1.NamespaceDefault,
								Name:      "pd-cluster-1-pd-1",
								Annotations: map[string]string{
									label.AnnPVCPodScheduling: "true",
									label.AnnPVCDeferDeleting: "true",
								},
							},
							Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
						},
					},
				}, nil
			},
			podGetFn: podGetErr(),
			updatePVCFn: func(claim *corev1.PersistentVolumeClaim) error {
				return nil
			},
			expectFn: func(schedulingPVC, currentPVC *apiv1.PersistentVolumeClaim, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(schedulingPVC.Annotations[label.AnnPVCPodScheduling]).To(BeEmpty())
				g.Expect(currentPVC.Annotations[label.AnnPVCPodScheduling]).NotTo(BeEmpty())
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestHAFilter(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name               string
		podFn              func(string, string, int32) *apiv1.Pod
		nodesFn            func() []apiv1.Node
		podListFn          func(string, string, string) (*apiv1.PodList, error)
		podGetFn           func(string, string) (*apiv1.Pod, error)
		pvcGetFn           func(string, string) (*apiv1.PersistentVolumeClaim, error)
		tcGetFn            func(string, string) (*v1alpha1.TidbCluster, error)
		scheduledNodeGetFn func(string) (*apiv1.Node, error)
		acquireLockFn      func(*apiv1.Pod) (*apiv1.PersistentVolumeClaim, *apiv1.PersistentVolumeClaim, error)
		expectFn           func([]apiv1.Node, error)
	}

	topologyKey := "zone"
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		instanceName := "demo"
		clusterName := "cluster-1"

		pod := test.podFn(instanceName, clusterName, 0)
		nodes := test.nodesFn()

		ha := ha{
			podListFn:          test.podListFn,
			pvcGetFn:           test.pvcGetFn,
			tcGetFn:            test.tcGetFn,
			scheduledNodeGetFn: test.scheduledNodeGetFn,
			acquireLockFn:      test.acquireLockFn,
		}
		n, err := ha.Filter(instanceName, pod, nodes)
		test.expectFn(n, err)
	}

	tests := []testcase{
		{
			name:    "one topology, one scheduled pod recreated and its pvc is bound, return the topologys",
			podFn:   newHAPDPod,
			nodesFn: fakeOneNode,
			pvcGetFn: func(ns string, pvcName string) (*corev1.PersistentVolumeClaim, error) {
				return &corev1.PersistentVolumeClaim{
					TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "pd-cluster-1-pd-0"},
					Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
				}, nil
			},
			acquireLockFn: acquireSuccess,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(1))
				g.Expect(nodes[0].Labels[topologyKey]).To(Equal("zone1"))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1"}))
			},
		},
		{
			name:    "acquired lock failed",
			podFn:   newHAPDPod,
			nodesFn: fakeOneNode,
			pvcGetFn: func(ns string, pvcName string) (*corev1.PersistentVolumeClaim, error) {
				return &corev1.PersistentVolumeClaim{
					TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "pd-cluster-1-pd-0"},
					Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
				}, nil
			},
			acquireLockFn: func(pod *corev1.Pod) (*apiv1.PersistentVolumeClaim, *apiv1.PersistentVolumeClaim, error) {
				return nil, nil, fmt.Errorf("failed to acquire the lock")
			},
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "failed to acquire the lock")).To(BeTrue())
			},
		},
		{
			name:    "already scheduled pod recreated, get pvc failed",
			podFn:   newHAPDPod,
			nodesFn: fakeOneNode,
			pvcGetFn: func(ns string, pvcName string) (*corev1.PersistentVolumeClaim, error) {
				return nil, fmt.Errorf("get pvc failed")
			},
			acquireLockFn: acquireSuccess,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "get pvc failed")).To(BeTrue())
			},
		},
		{
			name:          "list pod failed",
			podFn:         newHAPDPod,
			nodesFn:       fakeThreeNodes,
			podListFn:     podListErr(),
			acquireLockFn: acquireSuccess,
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
			acquireLockFn: acquireSuccess,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "get tidbcluster failed")).To(BeTrue())
			},
		},
		{
			name:          "zero node, return zero node",
			podFn:         newHAPDPod,
			nodesFn:       fakeZeroNode,
			podListFn:     podListFn(map[string][]int32{}),
			tcGetFn:       tcGetFn,
			acquireLockFn: acquireSuccess,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "no nodes available to schedule pods")).To(BeTrue())
			},
		},
		{
			name:    "one topology, one replicas, return one topology",
			podFn:   newHAPDPod,
			nodesFn: fakeOneNode,
			pvcGetFn: func(ns string, pvcName string) (*corev1.PersistentVolumeClaim, error) {
				return &corev1.PersistentVolumeClaim{
					TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "pd-cluster-1-pd-0"},
					Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending},
				}, nil
			},
			podListFn:     podListFn(map[string][]int32{}),
			tcGetFn:       tcGetOneReplicasFn,
			acquireLockFn: acquireSuccess,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(1))
				g.Expect(nodes[0].Labels[topologyKey]).To(Equal("zone1"))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1"}))
			},
		},
		{
			name:    "two topology, one replicas, return two topology",
			podFn:   newHAPDPod,
			nodesFn: fakeTwoNodes,
			pvcGetFn: func(ns string, pvcName string) (*corev1.PersistentVolumeClaim, error) {
				return &corev1.PersistentVolumeClaim{
					TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "pd-cluster-1-pd-0"},
					Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending},
				}, nil
			},
			podListFn:     podListFn(map[string][]int32{}),
			tcGetFn:       tcGetOneReplicasFn,
			acquireLockFn: acquireSuccess,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(2))
				g.Expect(getSortedTopologies(nodes, topologyKey)).To(Equal([]string{"zone1", "zone2"}))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-2"}))
			},
		},
		{
			name:    "one topology, two replicas, return one topology",
			podFn:   newHAPDPod,
			nodesFn: fakeOneNode,
			pvcGetFn: func(ns string, pvcName string) (*corev1.PersistentVolumeClaim, error) {
				return &corev1.PersistentVolumeClaim{
					TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "pd-cluster-1-pd-0"},
					Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending},
				}, nil
			},
			podListFn:     podListFn(map[string][]int32{}),
			tcGetFn:       tcGetTwoReplicasFn,
			acquireLockFn: acquireSuccess,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(1))
				g.Expect(nodes[0].Labels[topologyKey]).To(Equal("zone1"))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1"}))
			},
		},
		{
			name:    "two topologies, two replicas, return two topologies",
			podFn:   newHAPDPod,
			nodesFn: fakeTwoNodes,
			pvcGetFn: func(ns string, pvcName string) (*corev1.PersistentVolumeClaim, error) {
				return &corev1.PersistentVolumeClaim{
					TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "pd-cluster-1-pd-0"},
					Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending},
				}, nil
			},
			podListFn:     podListFn(map[string][]int32{}),
			tcGetFn:       tcGetTwoReplicasFn,
			acquireLockFn: acquireSuccess,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(2))
				g.Expect(getSortedTopologies(nodes, topologyKey)).To(Equal([]string{"zone1", "zone2"}))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-2"}))
			},
		},
		{
			name:    "one topology, no pod scheduled, return the topology",
			podFn:   newHAPDPod,
			nodesFn: fakeOneNode,
			pvcGetFn: func(ns string, pvcName string) (*corev1.PersistentVolumeClaim, error) {
				return &corev1.PersistentVolumeClaim{
					TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "pd-cluster-1-pd-0"},
					Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending},
				}, nil
			},
			acquireLockFn: acquireSuccess,
			podListFn:     podListFn(map[string][]int32{}),
			tcGetFn:       tcGetFn,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(1))
				g.Expect(nodes[0].Labels[topologyKey]).To(Equal("zone1"))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1"}))
			},
		},
		{
			name:    "one topology, one pod scheduled, return zero topology",
			podFn:   newHAPDPod,
			nodesFn: fakeOneNode,
			pvcGetFn: func(ns string, pvcName string) (*corev1.PersistentVolumeClaim, error) {
				return &corev1.PersistentVolumeClaim{
					TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Name: "pd-cluster-1-pd-0"},
					Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending},
				}, nil
			},
			podListFn:          podListFn(map[string][]int32{"kube-node-1": {1}}),
			acquireLockFn:      acquireSuccess,
			tcGetFn:            tcGetFn,
			scheduledNodeGetFn: fakeZeroScheduledNode,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).To(HaveOccurred())
				// g.Expect(err.Error()).To(ContainSubstring("unable to schedule to nodes: kube-node-1 (1 pd pods), max pods per node: 1"))
				g.Expect(len(nodes)).To(Equal(0))
			},
		},
		{
			name:               "two topologies, one pod scheduled on the topology, return one topology",
			podFn:              newHAPDPod,
			nodesFn:            fakeTwoNodes,
			podListFn:          podListFn(map[string][]int32{"kube-node-1": {0}}),
			acquireLockFn:      acquireSuccess,
			tcGetFn:            tcGetFn,
			scheduledNodeGetFn: fakeZeroScheduledNode,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(1))
				g.Expect(nodes[0].Labels[topologyKey]).To(Equal("zone2"))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-2"}))
			},
		},
		{
			name:               "two topologies, two pods scheduled on these two topologies, return zero topology",
			podFn:              newHAPDPod,
			nodesFn:            fakeTwoNodes,
			podListFn:          podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}}),
			acquireLockFn:      acquireSuccess,
			tcGetFn:            tcGetFn,
			scheduledNodeGetFn: fakeZeroScheduledNode,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).To(HaveOccurred())
				// g.Expect(err.Error()).To(ContainSubstring("unable to schedule to nodes: kube-node-1 (1 pd pods), kube-node-2 (1 pd pods), max pods per node: 1"))
				g.Expect(len(nodes)).To(Equal(0))
			},
		},
		{
			name:               "three topologies, zero pod scheduled, return all the three topologies",
			podFn:              newHAPDPod,
			nodesFn:            fakeThreeNodes,
			podListFn:          podListFn(map[string][]int32{}),
			acquireLockFn:      acquireSuccess,
			tcGetFn:            tcGetFn,
			scheduledNodeGetFn: fakeZeroScheduledNode,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(3))
				g.Expect(getSortedTopologies(nodes, topologyKey)).To(Equal([]string{"zone1", "zone2", "zone3"}))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-2", "kube-node-3"}))
			},
		},
		{
			name:               "three topologies, one pod scheduled, return two topologies",
			podFn:              newHAPDPod,
			nodesFn:            fakeThreeNodes,
			podListFn:          podListFn(map[string][]int32{"kube-node-1": {0}}),
			acquireLockFn:      acquireSuccess,
			tcGetFn:            tcGetFn,
			scheduledNodeGetFn: fakeZeroScheduledNode,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(2))
				g.Expect(getSortedTopologies(nodes, topologyKey)).To(Equal([]string{"zone2", "zone3"}))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-2", "kube-node-3"}))
			},
		},
		{
			name:               "three topologies, two pods scheduled, return one topology",
			podFn:              newHAPDPod,
			nodesFn:            fakeThreeNodes,
			podListFn:          podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}}),
			acquireLockFn:      acquireSuccess,
			tcGetFn:            tcGetFn,
			scheduledNodeGetFn: fakeZeroScheduledNode,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(1))
				g.Expect(nodes[0].Labels[topologyKey]).To(Equal("zone3"))
				g.Expect(nodes[0].Name).To(Equal("kube-node-3"))
			},
		},
		{
			name:               "three topologies, one pod not scheduled on these three topologies, return all the three nodes",
			podFn:              newHAPDPod,
			nodesFn:            fakeThreeNodes,
			podListFn:          podListFn(map[string][]int32{"kube-node-4": {4}}),
			acquireLockFn:      acquireSuccess,
			tcGetFn:            tcGetFn,
			scheduledNodeGetFn: fakeZeroScheduledNode,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(3))
				g.Expect(getSortedTopologies(nodes, topologyKey)).To(Equal([]string{"zone1", "zone2", "zone3"}))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-2", "kube-node-3"}))
			},
		},
		{
			name:      "three topologies, three pods scheduled on these two topologies, replicas is 5, return one topology",
			podFn:     newHAPDPod,
			nodesFn:   fakeSkipNodes(map[string]string{"kube-node-1": "zone1", "kube-node-5": "zone2"}),
			podListFn: podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1, 2}}),
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.PD.Replicas = 5
				return tc, nil
			},
			scheduledNodeGetFn: fakeScheduledNode("kube-node-2", "zone2"),
			acquireLockFn:      acquireSuccess,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(1))
				g.Expect(getSortedTopologies(nodes, topologyKey)).To(Equal([]string{"zone1"}))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1"}))
			},
		},
		{
			name:      "three topologies, two pods scheduled on these two topologies, replicas is 5, return two topologies",
			podFn:     newHAPDPod,
			nodesFn:   fakeSkipNodes(map[string]string{"kube-node-1": "zone1", "kube-node-5": "zone2"}),
			podListFn: podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}}),
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.PD.Replicas = 5
				return tc, nil
			},
			scheduledNodeGetFn: fakeScheduledNode("kube-node-2", "zone2"),
			acquireLockFn:      acquireSuccess,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(2))
				g.Expect(getSortedTopologies(nodes, topologyKey)).To(Equal([]string{"zone1", "zone2"}))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-5"}))
			},
		},
		{
			name:      "three topologies, three pods scheduled on these three topologies, replicas is 5, return three topologies",
			podFn:     newHAPDPod,
			nodesFn:   fakeThreeNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}, "kube-node-3": {2}}),
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.PD.Replicas = 5
				return tc, nil
			},
			scheduledNodeGetFn: fakeZeroScheduledNode,
			acquireLockFn:      acquireSuccess,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(3))
				g.Expect(getSortedTopologies(nodes, topologyKey)).To(Equal([]string{"zone1", "zone2", "zone3"}))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-2", "kube-node-3"}))
			},
		},
		{
			name:          "three topologies, four pods scheduled on these three topologies, replicas is 5, return two topologies",
			podFn:         newHAPDPod,
			nodesFn:       fakeThreeNodes,
			podListFn:     podListFn(map[string][]int32{"kube-node-1": {0, 3}, "kube-node-2": {1}, "kube-node-3": {2}}),
			acquireLockFn: acquireSuccess,
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.PD.Replicas = 5
				return tc, nil
			},
			scheduledNodeGetFn: fakeZeroScheduledNode,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(2))
				g.Expect(getSortedTopologies(nodes, topologyKey)).To(Equal([]string{"zone2", "zone3"}))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-2", "kube-node-3"}))
			},
		},
		{
			name:      "three topologies, four nodes, three pods scheduled on these three topologies, replicas is 4, return zero topology",
			podFn:     newHAPDPod,
			nodesFn:   fakeFourNodesWithThreeTopologies,
			podListFn: podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}, "kube-node-3": {2}}),
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.PD.Replicas = 4
				return tc, nil
			},
			scheduledNodeGetFn: fakeZeroScheduledNode,
			acquireLockFn:      acquireSuccess,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(0))
			},
		},
		{
			name:      "three topologies, four nodes, four pods scheduled on these three topologies, replicas is 5, return these two topologies",
			podFn:     newHAPDPod,
			nodesFn:   fakeFourNodesWithThreeTopologies,
			podListFn: podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}, "kube-node-3": {2}, "kube-node-4": {3}}),
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.PD.Replicas = 6
				return tc, nil
			},
			scheduledNodeGetFn: fakeZeroScheduledNode,
			acquireLockFn:      acquireSuccess,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(2))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-2"}))
			},
		},
		{
			name:      "four topologies, four nodes, four pods scheduled on these four topologies, replicas is 5, return four topologies",
			podFn:     newHAPDPod,
			nodesFn:   fakeFourNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}, "kube-node-3": {2}, "kube-node-4": {3}}),
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.PD.Replicas = 6
				return tc, nil
			},
			scheduledNodeGetFn: fakeZeroScheduledNode,
			acquireLockFn:      acquireSuccess,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(4))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-2", "kube-node-3", "kube-node-4"}))
			},
		},
		{
			name:      "four topologies, four nodes, four pods scheduled on these four topologies, replicas is 5, return one topology",
			podFn:     newHAPDPod,
			nodesFn:   fakeFourNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}, "kube-node-3": {}, "kube-node-4": {2, 3}}),
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.PD.Replicas = 6
				return tc, nil
			},
			scheduledNodeGetFn: fakeZeroScheduledNode,
			acquireLockFn:      acquireSuccess,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(1))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-3"}))
			},
		},
		{
			name:      "four topologies, four nodes, four pods scheduled on these four topologies, replicas is 5, return two topologies",
			podFn:     newHAPDPod,
			nodesFn:   fakeFourNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-1": {0, 1}, "kube-node-2": {}, "kube-node-3": {}, "kube-node-4": {2, 3}}),
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.PD.Replicas = 6
				return tc, nil
			},
			scheduledNodeGetFn: fakeZeroScheduledNode,
			acquireLockFn:      acquireSuccess,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(2))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-2", "kube-node-3"}))
			},
		},
		{
			name:      "two topologies, 2,2 pods scheduled on these two topologies, replicas is 5, can't schedule",
			podFn:     newHATiKVPod,
			nodesFn:   fakeTwoNodes,
			podListFn: podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}}),
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.TiKV.Replicas = 5
				return tc, nil
			},
			scheduledNodeGetFn: fakeZeroScheduledNode,
			acquireLockFn:      acquireSuccess,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).To(HaveOccurred())
				//g.Expect(err.Error()).To(ContainSubstring("unable to schedule to nodes: kube-node-1 (1 tikv pods), kube-node-2 (1 tikv pods), max pods per node: 1"))
				g.Expect(len(nodes)).To(Equal(0))
			},
		},
		{
			name:          "three topologies, three pods scheduled on these three topologies, replicas is 4, return all the three topologies",
			podFn:         newHATiKVPod,
			nodesFn:       fakeThreeNodes,
			podListFn:     podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}, "kube-node-3": {2}}),
			acquireLockFn: acquireSuccess,
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.TiKV.Replicas = 4
				return tc, nil
			},
			scheduledNodeGetFn: fakeZeroScheduledNode,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(3))
				g.Expect(getSortedTopologies(nodes, topologyKey)).To(Equal([]string{"zone1", "zone2", "zone3"}))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-2", "kube-node-3"}))
			},
		},
		{
			name:          "three topologies, four pods scheduled on these topologies, replicas is 5, return two topologies",
			podFn:         newHATiKVPod,
			nodesFn:       fakeThreeNodes,
			podListFn:     podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}, "kube-node-3": {2, 3}}),
			acquireLockFn: acquireSuccess,
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.TiKV.Replicas = 4
				return tc, nil
			},
			scheduledNodeGetFn: fakeZeroScheduledNode,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(2))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-2"}))
			},
		},
		{
			name:          "three topologies, four nodes, four pods scheduled on these topologies, replicas is 5, return one topology",
			podFn:         newHATiKVPod,
			nodesFn:       fakeFourNodesWithThreeTopologies,
			podListFn:     podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}, "kube-node-3": {2}, "kube-node-4": {3}}),
			acquireLockFn: acquireSuccess,
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.TiKV.Replicas = 4
				return tc, nil
			},
			scheduledNodeGetFn: fakeZeroScheduledNode,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(2))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-2"}))
			},
		},
		{
			name:          "four topologies, four nodes, four pods scheduled on these topologies, replicas is 5, return four topologies",
			podFn:         newHATiKVPod,
			nodesFn:       fakeFourNodes,
			podListFn:     podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}, "kube-node-3": {2}, "kube-node-4": {3}}),
			acquireLockFn: acquireSuccess,
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.TiKV.Replicas = 4
				return tc, nil
			},
			scheduledNodeGetFn: fakeZeroScheduledNode,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(4))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-1", "kube-node-2", "kube-node-3", "kube-node-4"}))
			},
		},
		{
			name:          "four topologies, four nodes, four pods scheduled on these topologies, replicas is 5, return one topology",
			podFn:         newHATiKVPod,
			nodesFn:       fakeFourNodes,
			podListFn:     podListFn(map[string][]int32{"kube-node-1": {0}, "kube-node-2": {1}, "kube-node-3": {}, "kube-node-4": {2, 3}}),
			acquireLockFn: acquireSuccess,
			tcGetFn: func(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
				tc, _ := tcGetFn(ns, tcName)
				tc.Spec.TiKV.Replicas = 4
				return tc, nil
			},
			scheduledNodeGetFn: fakeZeroScheduledNode,
			expectFn: func(nodes []apiv1.Node, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(nodes)).To(Equal(1))
				g.Expect(getSortedNodeNames(nodes)).To(Equal([]string{"kube-node-3"}))
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

func newHATiKVPod(instanceName, clusterName string, ordinal int32) *apiv1.Pod {
	return &apiv1.Pod{
		TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", controller.TiKVMemberName(clusterName), ordinal),
			Namespace: corev1.NamespaceDefault,
			Labels:    label.New().Instance(instanceName).TiKV().Labels(),
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

func podGetErr() func(string, string) (*apiv1.Pod, error) {
	return func(ns, podName string) (*apiv1.Pod, error) {
		return nil, errors.New("get pod failed")
	}
}

func podGetScheduled() func(string, string) (*apiv1.Pod, error) {
	return func(ns, podName string) (*apiv1.Pod, error) {
		return &apiv1.Pod{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			Spec: apiv1.PodSpec{
				NodeName: "node-1",
			},
		}, nil
	}
}

func podGetNotScheduled() func(string, string) (*apiv1.Pod, error) {
	return func(ns, podName string) (*apiv1.Pod, error) {
		return &apiv1.Pod{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			Spec: apiv1.PodSpec{
				NodeName: "",
			},
		}, nil
	}
}

func tcGetFn(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{Kind: "TidbCluster", APIVersion: "v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:        tcName,
			Namespace:   ns,
			Annotations: map[string]string{"pingcap.com/ha-topology-key": "zone"},
		},
		Spec: v1alpha1.TidbClusterSpec{
			PD:        &v1alpha1.PDSpec{Replicas: 3},
			TiKV:      &v1alpha1.TiKVSpec{},
			TiDB:      &v1alpha1.TiDBSpec{},
			Discovery: &v1alpha1.DiscoverySpec{},
		},
	}, nil
}

func tcGetOneReplicasFn(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{Kind: "TidbCluster", APIVersion: "v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      tcName,
			Namespace: ns,
		},
		Spec: v1alpha1.TidbClusterSpec{
			PD: &v1alpha1.PDSpec{Replicas: 1},
		},
	}, nil
}

func tcGetTwoReplicasFn(ns string, tcName string) (*v1alpha1.TidbCluster, error) {
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{Kind: "TidbCluster", APIVersion: "v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      tcName,
			Namespace: ns,
		},
		Spec: v1alpha1.TidbClusterSpec{
			PD: &v1alpha1.PDSpec{Replicas: 2},
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

func getSortedTopologies(nodes []apiv1.Node, topologyKey string) []string {
	arr := make([]string, 0)
	for _, node := range nodes {
		arr = append(arr, node.Labels[topologyKey])
	}
	sort.Strings(arr)
	return arr
}

func acquireSuccess(*apiv1.Pod) (*apiv1.PersistentVolumeClaim, *apiv1.PersistentVolumeClaim, error) {
	return nil, nil, nil
}
