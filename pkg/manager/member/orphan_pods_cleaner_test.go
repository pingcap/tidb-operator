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

package member

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOrphanPodsCleanerClean(t *testing.T) {
	g := NewGomegaWithT(t)

	tc := newTidbClusterForPD()
	dc := newDMClusterForMaster()

	tests := []struct {
		name            string
		pods            []*corev1.Pod
		apiPods         []*corev1.Pod
		pvcs            []*corev1.PersistentVolumeClaim
		deletePodFailed bool
		testOnDM        bool
		expectFn        func(*GomegaWithT, map[string]string, *orphanPodsCleaner, error)
	}{
		{
			name: "no pods",
			pods: []*corev1.Pod{},
			pvcs: nil,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *orphanPodsCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(0))
			},
		},
		{
			name: "not pd or tikv pods",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).TiDB().Labels(),
					},
				},
			},
			pvcs: nil,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *orphanPodsCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pod-1"]).To(Equal(skipReasonOrphanPodsCleanerIsNotTarget))
			},
		},
		{
			name: "has no spec.volumes",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
			},
			pvcs: nil,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *orphanPodsCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pod-1"]).To(Equal(skipReasonOrphanPodsCleanerPVCNameIsEmpty))
			},
		},
		{
			name: "has no spec.volumes for dm-master",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.NewDM().Instance(dc.GetInstanceName()).DMMaster().Labels(),
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
			},
			pvcs:     nil,
			testOnDM: true,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *orphanPodsCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pod-1"]).To(Equal(skipReasonOrphanPodsCleanerPVCNameIsEmpty))
			},
		},
		{
			name: "has no spec.volumes for dm-worker",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.NewDM().Instance(dc.GetInstanceName()).DMWorker().Labels(),
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
			},
			pvcs:     nil,
			testOnDM: true,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *orphanPodsCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pod-1"]).To(Equal(skipReasonOrphanPodsCleanerPVCNameIsEmpty))
			},
		},
		{
			name: "claimName is empty",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "pd",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "",
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
			},
			pvcs: nil,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *orphanPodsCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pod-1"]).To(Equal(skipReasonOrphanPodsCleanerPVCNameIsEmpty))
			},
		},
		{
			name: "pvc is found",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "pd",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-1",
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
			},
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pvc-1",
						Namespace: metav1.NamespaceDefault,
					},
				},
			},
			expectFn: func(g *GomegaWithT, skipReason map[string]string, _ *orphanPodsCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pod-1"]).To(Equal(skipReasonOrphanPodsCleanerPVCIsFound))
			},
		},
		{
			name: "pvc is not found",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "pd",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-1",
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
			},
			pvcs: []*corev1.PersistentVolumeClaim{},
			expectFn: func(g *GomegaWithT, skipReason map[string]string, opc *orphanPodsCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(0))
				_, err = opc.deps.PodLister.Pods("default").Get("pod-1")
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "not found")).To(BeTrue())
			},
		},
		{
			name: "one of two pvcs is not found",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "pd1",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-1",
									},
								},
							},
							{
								Name: "pd0",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-0",
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
			},
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pvc-1",
						Namespace: metav1.NamespaceDefault,
					},
				},
			},
			expectFn: func(g *GomegaWithT, skipReason map[string]string, opc *orphanPodsCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(0))
				_, err = opc.deps.PodLister.Pods("default").Get("pod-1")
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "not found")).To(BeTrue())
			},
		},
		{
			// in theory, this is is possible because we can't check the PVC
			// and pod in an atomic operation.
			name: "pvc is not found but pod has been scheduled",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "pd",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-1",
									},
								},
							},
						},
						NodeName: "foobar",
					},
				},
			},
			pvcs: []*corev1.PersistentVolumeClaim{},
			expectFn: func(g *GomegaWithT, skipReason map[string]string, opc *orphanPodsCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pod-1"]).To(Equal(skipReasonOrphanPodsCleanerPodHasBeenScheduled))
			},
		},
		{
			name: "pvc is not found but pod is recreated in apiserver",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						UID:       "pod-1-uid",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "pd",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-1",
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
			},
			apiPods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						UID:       "pod-1-uid2",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "pd",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-1",
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
					},
				},
			},
			pvcs: []*corev1.PersistentVolumeClaim{},
			expectFn: func(g *GomegaWithT, skipReason map[string]string, opc *orphanPodsCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pod-1"]).To(Equal(skipReasonOrphanPodsCleanerPodRecreated))
			},
		},
		{
			name: "pvc is not found but pod is scheduled",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						UID:       "pod-1-uid",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "pd",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-1",
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
			},
			apiPods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						UID:       "pod-1-uid",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "pd",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-1",
									},
								},
							},
						},
						NodeName: "foo",
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
					},
				},
			},
			pvcs: []*corev1.PersistentVolumeClaim{},
			expectFn: func(g *GomegaWithT, skipReason map[string]string, opc *orphanPodsCleaner, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(skipReason)).To(Equal(1))
				g.Expect(skipReason["pod-1"]).To(Equal(skipReasonOrphanPodsCleanerPodHasBeenScheduled))
			},
		},
		{
			name: "pod delete failed",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "pd",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-1",
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
			},
			pvcs:            []*corev1.PersistentVolumeClaim{},
			deletePodFailed: true,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, opc *orphanPodsCleaner, err error) {
				g.Expect(len(skipReason)).To(Equal(0))
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "delete pod failed")).To(BeTrue())
			},
		},
		{
			name: "multiple pods",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-1",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).PD().Labels(),
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "pd",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-1",
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-2",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).TiKV().Labels(),
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "pd",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "",
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-3",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).TiKV().Labels(),
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "pd",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-3",
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-4",
						Namespace: metav1.NamespaceDefault,
						Labels:    label.New().Instance(tc.GetInstanceName()).TiDB().Labels(),
					},
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: "pd",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: "pvc-4",
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
			},
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pvc-2",
						Namespace: metav1.NamespaceDefault,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pvc-3",
						Namespace: metav1.NamespaceDefault,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pvc-4",
						Namespace: metav1.NamespaceDefault,
					},
				},
			},
			deletePodFailed: false,
			expectFn: func(g *GomegaWithT, skipReason map[string]string, opc *orphanPodsCleaner, err error) {
				g.Expect(len(skipReason)).To(Equal(3))
				g.Expect(skipReason["pod-2"]).To(Equal(skipReasonOrphanPodsCleanerPVCNameIsEmpty))
				g.Expect(skipReason["pod-3"]).To(Equal(skipReasonOrphanPodsCleanerPVCIsFound))
				g.Expect(skipReason["pod-4"]).To(Equal(skipReasonOrphanPodsCleanerIsNotTarget))
				g.Expect(err).NotTo(HaveOccurred())
				_, err = opc.deps.PodLister.Pods("default").Get("pod-1")
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "not found")).To(BeTrue())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeDeps := controller.NewFakeDependencies()
			opc := &orphanPodsCleaner{deps: fakeDeps}
			podIndexer := fakeDeps.KubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
			pvcIndexer := fakeDeps.KubeInformerFactory.Core().V1().PersistentVolumeClaims().Informer().GetIndexer()
			client := fakeDeps.KubeClientset
			podControl := fakeDeps.PodControl.(*controller.FakePodControl)
			if tt.pods != nil {
				for _, pod := range tt.pods {
					client.CoreV1().Pods(pod.Namespace).Create(pod)
					podIndexer.Add(pod)
				}
			}
			if tt.apiPods != nil {
				for _, pod := range tt.apiPods {
					client.CoreV1().Pods(pod.Namespace).Update(pod)
				}
			}
			if tt.pvcs != nil {
				for _, pvc := range tt.pvcs {
					client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(pvc)
					pvcIndexer.Add(pvc)
				}
			}
			if tt.deletePodFailed {
				podControl.SetDeletePodError(fmt.Errorf("delete pod failed"), 0)
			}

			var skipReason map[string]string
			var err error

			if tt.testOnDM {
				skipReason, err = opc.Clean(dc)
			} else {
				skipReason, err = opc.Clean(tc)
			}
			tt.expectFn(g, skipReason, opc, err)
		})
	}
}
