// Copyright 2020 PingCAP, Inc.
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
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func newMockPVC(name, storageClass, storageRequest, capacity string) *v1.PersistentVolumeClaim {
	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: v1.NamespaceDefault,
			Name:      name,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: resource.MustParse(storageRequest),
				},
			},
			StorageClassName: pointer.StringPtr(storageClass),
		},
		Status: v1.PersistentVolumeClaimStatus{
			Phase: v1.ClaimBound,
			Capacity: v1.ResourceList{
				v1.ResourceStorage: resource.MustParse(capacity),
			},
		},
	}
}

func newStorageClass(name string, volumeExpansion bool) *storagev1.StorageClass {
	return &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		AllowVolumeExpansion: pointer.BoolPtr(volumeExpansion),
	}
}

func TestResizeVolumes(t *testing.T) {
	scName := "sc"

	testcases := map[string]struct {
		setup  func(ctx *componentVolumeContext)
		expect func(g *GomegaWithT, resizer *pvcResizer, ctx *componentVolumeContext, err error)
	}{
		"all volumes are resized": {
			setup: func(ctx *componentVolumeContext) {
				tc := ctx.cluster.(*v1alpha1.TidbCluster)
				ctx.status = &tc.Status.PD

				tc.Status.PD.Conditions = nil
				ctx.desiredVolumeQuantity = map[v1alpha1.StorageVolumeName]resource.Quantity{
					"volume-1": resource.MustParse("2Gi"),
				}
				ctx.actualPodVolumes = []*podVolumeContext{
					{
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cluster-pd-0"},
						},
						volumes: []*volume{
							{name: "volume-1", pvc: newMockPVC("volume-1-pvc-1", scName, "2Gi", "2Gi")},
						},
					},
					{
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cluster-pd-2"},
						},
						volumes: []*volume{
							{name: "volume-1", pvc: newMockPVC("volume-1-pvc-2", scName, "2Gi", "2Gi")},
						},
					},
					{
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cluster-pd-3"},
						},
						volumes: []*volume{
							{name: "volume-1", pvc: newMockPVC("volume-1-pvc-3", scName, "2Gi", "2Gi")},
						},
					},
				}
			},
			expect: func(g *WithT, resizer *pvcResizer, ctx *componentVolumeContext, err error) {
				g.Expect(err).Should(Succeed())
			},
		},
		"end resizing": {
			setup: func(ctx *componentVolumeContext) {
				tc := ctx.cluster.(*v1alpha1.TidbCluster)
				ctx.status = &tc.Status.PD

				tc.Status.PD.Conditions = []metav1.Condition{
					{
						Type:    v1alpha1.ComponentVolumeResizing,
						Status:  metav1.ConditionTrue,
						Reason:  "BeginResizing",
						Message: "Set resizing condition to begin resizing",
					},
				}
				ctx.desiredVolumeQuantity = map[v1alpha1.StorageVolumeName]resource.Quantity{
					"volume-1": resource.MustParse("2Gi"),
				}
				ctx.actualPodVolumes = []*podVolumeContext{
					{
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cluster-pd-0"},
						},
						volumes: []*volume{
							{name: "volume-1", pvc: newMockPVC("volume-1-pvc-1", scName, "2Gi", "2Gi")},
						},
					},
					{
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cluster-pd-2"},
						},
						volumes: []*volume{
							{name: "volume-1", pvc: newMockPVC("volume-1-pvc-2", scName, "2Gi", "2Gi")},
						},
					},
					{
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cluster-pd-3"},
						},
						volumes: []*volume{
							{name: "volume-1", pvc: newMockPVC("volume-1-pvc-3", scName, "2Gi", "2Gi")},
						},
					},
				}
			},
			expect: func(g *WithT, resizer *pvcResizer, ctx *componentVolumeContext, err error) {
				g.Expect(err).Should(Succeed())
				g.Expect(ctx.cluster.(*v1alpha1.TidbCluster).IsComponentVolumeResizing(ctx.status.GetMemberType())).Should(BeFalse())
			},
		},
		"begin resizing": {
			setup: func(ctx *componentVolumeContext) {
				tc := ctx.cluster.(*v1alpha1.TidbCluster)
				ctx.status = &tc.Status.PD

				tc.Status.PD.Conditions = nil
				ctx.desiredVolumeQuantity = map[v1alpha1.StorageVolumeName]resource.Quantity{
					"volume-1": resource.MustParse("2Gi"),
				}
				ctx.actualPodVolumes = []*podVolumeContext{
					{
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cluster-pd-0"},
						},
						volumes: []*volume{
							{name: "volume-1", pvc: newMockPVC("volume-1-pvc-1", scName, "2Gi", "1Gi")},
						},
					},
					{
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cluster-pd-2"},
						},
						volumes: []*volume{
							{name: "volume-1", pvc: newMockPVC("volume-1-pvc-2", scName, "2Gi", "2Gi")},
						},
					},
					{
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cluster-pd-3"},
						},
						volumes: []*volume{
							{name: "volume-1", pvc: newMockPVC("volume-1-pvc-3", scName, "2Gi", "2Gi")},
						},
					},
				}
			},
			expect: func(g *WithT, resizer *pvcResizer, ctx *componentVolumeContext, err error) {
				g.Expect(err).Should(HaveOccurred())
				g.Expect(err.Error()).Should(ContainSubstring("set condition before resizing volumes"))
				g.Expect(ctx.cluster.(*v1alpha1.TidbCluster).IsComponentVolumeResizing(ctx.status.GetMemberType())).Should(BeTrue())
			},
		},
		"need to resize some volumes": {
			setup: func(ctx *componentVolumeContext) {
				tc := ctx.cluster.(*v1alpha1.TidbCluster)
				ctx.status = &tc.Status.PD

				tc.Status.PD.Conditions = []metav1.Condition{
					{
						Type:    v1alpha1.ComponentVolumeResizing,
						Status:  metav1.ConditionTrue,
						Reason:  "BeginResizing",
						Message: "Set resizing condition to begin resizing",
					},
				}
				ctx.desiredVolumeQuantity = map[v1alpha1.StorageVolumeName]resource.Quantity{
					"volume-1": resource.MustParse("2Gi"),
				}
				ctx.actualPodVolumes = []*podVolumeContext{
					{
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cluster-pd-0"},
						},
						volumes: []*volume{
							{name: "volume-1", pvc: newMockPVC("volume-1-pvc-1", scName, "2Gi", "2Gi")},
						},
					},
					{
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cluster-pd-2"},
						},
						volumes: []*volume{
							{name: "volume-1", pvc: newMockPVC("volume-1-pvc-2", scName, "1Gi", "1Gi")},
						},
					},
					{
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cluster-pd-3"},
						},
						volumes: []*volume{
							{name: "volume-1", pvc: newMockPVC("volume-1-pvc-3", scName, "1Gi", "1Gi")},
						},
					},
				}
			},
			expect: func(g *WithT, resizer *pvcResizer, ctx *componentVolumeContext, err error) {
				g.Expect(err).Should(Succeed())
				g.Expect(ctx.cluster.(*v1alpha1.TidbCluster).IsComponentVolumeResizing(ctx.status.GetMemberType())).Should(BeTrue())

				cli := resizer.deps.KubeClientset.CoreV1().PersistentVolumeClaims(ctx.cluster.GetNamespace())

				pvc, err := cli.Get(context.TODO(), "volume-1-pvc-1", metav1.GetOptions{})
				g.Expect(err).To(Succeed())
				g.Expect(cmp.Diff(pvc, newMockPVC("volume-1-pvc-1", scName, "2Gi", "2Gi"))).To(BeEmpty(), "-want, +got")

				pvc, err = cli.Get(context.TODO(), "volume-1-pvc-2", metav1.GetOptions{})
				g.Expect(err).To(Succeed())
				g.Expect(cmp.Diff(pvc, newMockPVC("volume-1-pvc-2", scName, "2Gi", "1Gi"))).To(BeEmpty(), "-want, +got")

				pvc, err = cli.Get(context.TODO(), "volume-1-pvc-3", metav1.GetOptions{})
				g.Expect(err).To(Succeed())
				g.Expect(cmp.Diff(pvc, newMockPVC("volume-1-pvc-3", scName, "1Gi", "1Gi"))).To(BeEmpty(), "-want, +got")
			},
		},
		"some volumes are resizing": {
			setup: func(ctx *componentVolumeContext) {
				tc := ctx.cluster.(*v1alpha1.TidbCluster)
				ctx.status = &tc.Status.PD

				tc.Status.PD.Conditions = []metav1.Condition{
					{
						Type:    v1alpha1.ComponentVolumeResizing,
						Status:  metav1.ConditionTrue,
						Reason:  "BeginResizing",
						Message: "Set resizing condition to begin resizing",
					},
				}
				ctx.desiredVolumeQuantity = map[v1alpha1.StorageVolumeName]resource.Quantity{
					"volume-1": resource.MustParse("2Gi"),
				}
				ctx.actualPodVolumes = []*podVolumeContext{
					{
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cluster-pd-0"},
						},
						volumes: []*volume{
							{name: "volume-1", pvc: newMockPVC("volume-1-pvc-1", scName, "2Gi", "2Gi")},
						},
					},
					{
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cluster-pd-2"},
						},
						volumes: []*volume{
							{name: "volume-1", pvc: newMockPVC("volume-1-pvc-2", scName, "2Gi", "1Gi")},
						},
					},
					{
						pod: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cluster-pd-3"},
						},
						volumes: []*volume{
							{name: "volume-1", pvc: newMockPVC("volume-1-pvc-3", scName, "1Gi", "1Gi")},
						},
					},
				}
			},
			expect: func(g *WithT, resizer *pvcResizer, ctx *componentVolumeContext, err error) {
				g.Expect(err).Should(Succeed())
				g.Expect(ctx.cluster.(*v1alpha1.TidbCluster).IsComponentVolumeResizing(ctx.status.GetMemberType())).Should(BeTrue())

				cli := resizer.deps.KubeClientset.CoreV1().PersistentVolumeClaims(ctx.cluster.GetNamespace())

				pvc, err := cli.Get(context.TODO(), "volume-1-pvc-1", metav1.GetOptions{})
				g.Expect(err).To(Succeed())
				g.Expect(cmp.Diff(pvc, newMockPVC("volume-1-pvc-1", scName, "2Gi", "2Gi"))).To(BeEmpty(), "-want, +got")

				pvc, err = cli.Get(context.TODO(), "volume-1-pvc-2", metav1.GetOptions{})
				g.Expect(err).To(Succeed())
				g.Expect(cmp.Diff(pvc, newMockPVC("volume-1-pvc-2", scName, "2Gi", "1Gi"))).To(BeEmpty(), "-want, +got")

				pvc, err = cli.Get(context.TODO(), "volume-1-pvc-3", metav1.GetOptions{})
				g.Expect(err).To(Succeed())
				g.Expect(cmp.Diff(pvc, newMockPVC("volume-1-pvc-3", scName, "1Gi", "1Gi"))).To(BeEmpty(), "-want, +got")
			},
		},
	}

	for name, tt := range testcases {
		t.Run(name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fakeDeps := controller.NewFakeDependencies()
			informerFactory := fakeDeps.KubeInformerFactory

			resizer := &pvcResizer{
				deps: fakeDeps,
			}
			tc := &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{Namespace: v1.NamespaceDefault, Name: "test-cluster"},
				Spec: v1alpha1.TidbClusterSpec{
					PD: &v1alpha1.PDSpec{
						Replicas: 3,
					},
				},
			}
			vctx := &componentVolumeContext{
				cluster: tc,
			}
			tt.setup(vctx)

			for _, podVol := range vctx.actualPodVolumes {
				if podVol.pod != nil {
					fakeDeps.KubeClientset.CoreV1().Pods(tc.Namespace).Create(context.TODO(), podVol.pod, metav1.CreateOptions{})
				}
				for _, vol := range podVol.volumes {
					if vol.pvc != nil {
						fakeDeps.KubeClientset.CoreV1().PersistentVolumeClaims(tc.Namespace).Create(context.TODO(), vol.pvc, metav1.CreateOptions{})
					}
				}
			}
			fakeDeps.KubeClientset.StorageV1().StorageClasses().Create(context.TODO(), newStorageClass(scName, true), metav1.CreateOptions{})
			informerFactory.Start(ctx.Done())
			informerFactory.WaitForCacheSync(ctx.Done())

			err := resizer.resizeVolumes(vctx)
			tt.expect(g, resizer, vctx, err)
		})
	}
}

func TestUpdateVolumeStatus(t *testing.T) {
	scName := "sc"

	tests := []struct {
		name   string
		setup  func(ctx *componentVolumeContext)
		expect map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus
	}{
		{
			name: "volumes start to resize",
			setup: func(ctx *componentVolumeContext) {
				ctx.desiredVolumeQuantity = map[v1alpha1.StorageVolumeName]resource.Quantity{
					"volume-1": resource.MustParse("2Gi"),
					"volume-2": resource.MustParse("2Gi"),
				}
				ctx.actualPodVolumes = []*podVolumeContext{
					{
						volumes: []*volume{
							{
								name: "volume-1",
								pvc:  newMockPVC("volume-1-pvc-1", scName, "2Gi", "1Gi"),
							},
							{
								name: "volume-2",
								pvc:  newMockPVC("volume-2-pvc-1", scName, "2Gi", "1Gi"),
							},
						},
					},
					{
						volumes: []*volume{
							{
								name: "volume-1",
								pvc:  newMockPVC("volume-1-pvc-2", scName, "2Gi", "1Gi"),
							},
							{
								name: "volume-2",
								pvc:  newMockPVC("volume-2-pvc-2", scName, "2Gi", "1Gi"),
							},
						},
					},
					{
						volumes: []*volume{
							{
								name: "volume-1",
								pvc:  newMockPVC("volume-1-pvc-3", scName, "2Gi", "1Gi"),
							},
							{
								name: "volume-2",
								pvc:  newMockPVC("volume-2-pvc-3", scName, "2Gi", "1Gi"),
							},
						},
					},
				}
			},
			expect: map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{
				"volume-1": {
					Name: "volume-1",
					ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
						BoundCount:      3,
						CurrentCount:    3,
						ResizedCount:    0,
						CurrentCapacity: resource.MustParse("1Gi"),
						ResizedCapacity: resource.MustParse("2Gi"),
					},
				},
				"volume-2": {
					Name: "volume-2",
					ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
						BoundCount:      3,
						CurrentCount:    3,
						ResizedCount:    0,
						CurrentCapacity: resource.MustParse("1Gi"),
						ResizedCapacity: resource.MustParse("2Gi"),
					},
				},
			},
		},
		{
			name: "some volumes is resizing",
			setup: func(ctx *componentVolumeContext) {
				ctx.desiredVolumeQuantity = map[v1alpha1.StorageVolumeName]resource.Quantity{
					"volume-1": resource.MustParse("2Gi"),
					"volume-2": resource.MustParse("2Gi"),
				}
				ctx.actualPodVolumes = []*podVolumeContext{
					{
						volumes: []*volume{
							{
								name: "volume-1",
								pvc:  newMockPVC("volume-1-pvc-1", scName, "2Gi", "2Gi"), // resized
							},
							{
								name: "volume-2",
								pvc:  newMockPVC("volume-2-pvc-1", scName, "2Gi", "1Gi"),
							},
						},
					},
					{
						volumes: []*volume{
							{
								name: "volume-1",
								pvc:  newMockPVC("volume-1-pvc-2", scName, "2Gi", "1Gi"),
							},
							{
								name: "volume-2",
								pvc:  newMockPVC("volume-2-pvc-2", scName, "2Gi", "2Gi"), // resized
							},
						},
					},
					{
						volumes: []*volume{
							{
								name: "volume-1",
								pvc:  newMockPVC("volume-1-pvc-3", scName, "2Gi", "1Gi"),
							},
							{
								name: "volume-2",
								pvc:  newMockPVC("volume-2-pvc-3", scName, "2Gi", "2Gi"), // resized
							},
						},
					},
				}
			},
			expect: map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{
				"volume-1": {
					Name: "volume-1",
					ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
						BoundCount:      3,
						CurrentCount:    2,
						ResizedCount:    1,
						CurrentCapacity: resource.MustParse("1Gi"),
						ResizedCapacity: resource.MustParse("2Gi"),
					},
				},
				"volume-2": {
					Name: "volume-2",
					ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
						BoundCount:      3,
						CurrentCount:    1,
						ResizedCount:    2,
						CurrentCapacity: resource.MustParse("1Gi"),
						ResizedCapacity: resource.MustParse("2Gi"),
					},
				},
			},
		},
		{
			name: "all volumes is resized",
			setup: func(ctx *componentVolumeContext) {
				ctx.desiredVolumeQuantity = map[v1alpha1.StorageVolumeName]resource.Quantity{
					"volume-1": resource.MustParse("2Gi"),
					"volume-2": resource.MustParse("2Gi"),
				}
				ctx.actualPodVolumes = []*podVolumeContext{
					{
						volumes: []*volume{
							{
								name: "volume-1",
								pvc:  newMockPVC("volume-1-pvc-1", scName, "2Gi", "2Gi"),
							},
							{
								name: "volume-2",
								pvc:  newMockPVC("volume-2-pvc-1", scName, "2Gi", "2Gi"),
							},
						},
					},
					{
						volumes: []*volume{
							{
								name: "volume-1",
								pvc:  newMockPVC("volume-1-pvc-2", scName, "2Gi", "2Gi"),
							},
							{
								name: "volume-2",
								pvc:  newMockPVC("volume-2-pvc-2", scName, "2Gi", "2Gi"),
							},
						},
					},
					{
						volumes: []*volume{
							{
								name: "volume-1",
								pvc:  newMockPVC("volume-1-pvc-3", scName, "2Gi", "2Gi"),
							},
							{
								name: "volume-2",
								pvc:  newMockPVC("volume-2-pvc-3", scName, "2Gi", "2Gi"),
							},
						},
					},
				}
			},
			expect: map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{
				"volume-1": {
					Name: "volume-1",
					ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
						BoundCount:      3,
						CurrentCount:    3,
						ResizedCount:    3,
						CurrentCapacity: resource.MustParse("2Gi"),
						ResizedCapacity: resource.MustParse("2Gi"),
					},
				},
				"volume-2": {
					Name: "volume-2",
					ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
						BoundCount:      3,
						CurrentCount:    3,
						ResizedCount:    3,
						CurrentCapacity: resource.MustParse("2Gi"),
						ResizedCapacity: resource.MustParse("2Gi"),
					},
				},
			},
		},
		{
			name: "remove lost volume status",
			setup: func(ctx *componentVolumeContext) {
				ctx.status = &v1alpha1.PDStatus{
					Volumes: map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{
						"volume-1": {Name: "volume-1"},
						"volume-2": {Name: "volume-2"},
					},
				}
				ctx.desiredVolumeQuantity = map[v1alpha1.StorageVolumeName]resource.Quantity{
					"volume-1": resource.MustParse("2Gi"),
				}
				ctx.actualPodVolumes = []*podVolumeContext{
					{
						volumes: []*volume{
							{
								name: "volume-1",
								pvc:  newMockPVC("volume-1-pvc-1", scName, "2Gi", "2Gi"),
							},
						},
					},
					{
						volumes: []*volume{
							{
								name: "volume-1",
								pvc:  newMockPVC("volume-1-pvc-2", scName, "2Gi", "2Gi"),
							},
						},
					},
					{
						volumes: []*volume{
							{
								name: "volume-1",
								pvc:  newMockPVC("volume-1-pvc-3", scName, "2Gi", "2Gi"),
							},
						},
					},
				}
			},
			expect: map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{
				"volume-1": {
					Name: "volume-1",
					ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
						BoundCount:      3,
						CurrentCount:    3,
						ResizedCount:    3,
						CurrentCapacity: resource.MustParse("2Gi"),
						ResizedCapacity: resource.MustParse("2Gi"),
					},
				},
			},
		},
		{
			name: "update existing volume status",
			setup: func(ctx *componentVolumeContext) {
				ctx.status = &v1alpha1.PDStatus{
					Volumes: map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{
						"volume-1": {Name: "volume-1"},
					},
				}
				ctx.desiredVolumeQuantity = map[v1alpha1.StorageVolumeName]resource.Quantity{
					"volume-1": resource.MustParse("2Gi"),
				}
				ctx.actualPodVolumes = []*podVolumeContext{
					{
						volumes: []*volume{
							{
								name: "volume-1",
								pvc:  newMockPVC("volume-1-pvc-1", scName, "2Gi", "2Gi"),
							},
						},
					},
					{
						volumes: []*volume{
							{
								name: "volume-1",
								pvc:  newMockPVC("volume-1-pvc-2", scName, "2Gi", "2Gi"),
							},
						},
					},
					{
						volumes: []*volume{
							{
								name: "volume-1",
								pvc:  newMockPVC("volume-1-pvc-3", scName, "2Gi", "2Gi"),
							},
						},
					},
				}
			},
			expect: map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{
				"volume-1": {
					Name: "volume-1",
					ObservedStorageVolumeStatus: v1alpha1.ObservedStorageVolumeStatus{
						BoundCount:      3,
						CurrentCount:    3,
						ResizedCount:    3,
						CurrentCapacity: resource.MustParse("2Gi"),
						ResizedCapacity: resource.MustParse("2Gi"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			resizer := &pvcResizer{
				deps: controller.NewFakeDependencies(),
			}

			ctx := &componentVolumeContext{}
			ctx.status = &v1alpha1.PDStatus{
				Volumes: map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{},
			}
			if tt.setup != nil {
				tt.setup(ctx)
			}

			resizer.updateVolumeStatus(ctx)
			diff := cmp.Diff(ctx.status.GetVolumes(), tt.expect)
			g.Expect(diff).Should(BeEmpty(), "unexpected (-want, +got)")
		})
	}
}

func TestClassifyVolumes(t *testing.T) {
	scName := "sc-1"

	diffVolumes := func(g *GomegaWithT, vols1 map[volumePhase][]*volume, vols2 map[volumePhase][]*volume) {
		phases := []volumePhase{needResize, resizing, resized}
		for _, phase := range phases {
			g.Expect(len(vols1[phase])).Should(Equal(len(vols2[phase])))
			for i := range vols1[phase] {
				g.Expect(diffVolume(vols1[phase][i], vols2[phase][i])).Should(BeEmpty(), "unexpected (-want, +got)")
			}
		}
	}

	tests := map[string]struct {
		setup func(*componentVolumeContext) []*volume
		sc    *storagev1.StorageClass

		expect func(*GomegaWithT, map[volumePhase][]*volume, error)
	}{
		"normal": {
			setup: func(ctx *componentVolumeContext) []*volume {
				ctx.desiredVolumeQuantity = map[v1alpha1.StorageVolumeName]resource.Quantity{
					"volume-1": resource.MustParse("2Gi"),
					"volume-2": resource.MustParse("2Gi"),
					"volume-3": resource.MustParse("2Gi"),
				}
				volumes := []*volume{
					newVolume("volume-1", newMockPVC("volume-1-pvc-1", scName, "2Gi", "2Gi")), // resized
					newVolume("volume-2", newMockPVC("volume-2-pvc-1", scName, "2Gi", "1Gi")), // resizing
					newVolume("volume-3", newMockPVC("volume-3-pvc-1", scName, "1Gi", "1Gi")), // need resize
				}

				return volumes
			},
			sc: newStorageClass(scName, true),
			expect: func(g *GomegaWithT, volumes map[volumePhase][]*volume, err error) {
				expectVolumes := map[volumePhase][]*volume{
					needResize: {
						newVolume("volume-3", newMockPVC("volume-3-pvc-1", scName, "1Gi", "1Gi")), // need resize
					},
					resized: {
						newVolume("volume-1", newMockPVC("volume-1-pvc-1", scName, "2Gi", "2Gi")), // resized
					},
					resizing: {
						newVolume("volume-2", newMockPVC("volume-2-pvc-1", scName, "2Gi", "1Gi")), // resizing
					},
				}

				g.Expect(err).To(Succeed())
				diffVolumes(g, volumes, expectVolumes)
			},
		},
		"shink storage": {
			setup: func(ctx *componentVolumeContext) []*volume {
				ctx.desiredVolumeQuantity = map[v1alpha1.StorageVolumeName]resource.Quantity{
					"volume-1": resource.MustParse("2Gi"),
				}
				volumes := []*volume{
					newVolume("volume-1", newMockPVC("volume-1-pvc-1", scName, "3Gi", "3Gi")), // need shink
				}
				return volumes
			},
			sc: newStorageClass(scName, true),
			expect: func(g *GomegaWithT, volumes map[volumePhase][]*volume, err error) {
				expectVolumes := map[volumePhase][]*volume{
					needResize: {},
					resized:    {},
					resizing:   {},
				}

				g.Expect(err).To(Succeed())
				diffVolumes(g, volumes, expectVolumes)
			},
		},
		"default storage class": {
			setup: func(ctx *componentVolumeContext) []*volume {
				ctx.desiredVolumeQuantity = map[v1alpha1.StorageVolumeName]resource.Quantity{
					"volume-1": resource.MustParse("2Gi"),
				}
				vol := newVolume("volume-1", newMockPVC("volume-1-pvc-1", scName, "1Gi", "1Gi"))
				vol.pvc.Spec.StorageClassName = nil
				volumes := []*volume{
					vol,
				}

				return volumes
			},
			sc: newStorageClass(scName, true),
			expect: func(g *GomegaWithT, volumes map[volumePhase][]*volume, err error) {
				expectVolumes := map[volumePhase][]*volume{
					needResize: {},
					resized:    {},
					resizing:   {},
				}

				g.Expect(err).To(Succeed())
				diffVolumes(g, volumes, expectVolumes)
			},
		},
		"unsupported storage class": {
			setup: func(ctx *componentVolumeContext) []*volume {
				ctx.desiredVolumeQuantity = map[v1alpha1.StorageVolumeName]resource.Quantity{
					"volume-1": resource.MustParse("2Gi"),
				}
				volumes := []*volume{
					newVolume("volume-1", newMockPVC("volume-1-pvc-1", scName, "1Gi", "1Gi")),
				}

				return volumes
			},
			sc: newStorageClass(scName, false),
			expect: func(g *GomegaWithT, volumes map[volumePhase][]*volume, err error) {
				expectVolumes := map[volumePhase][]*volume{
					needResize: {},
					resized:    {},
					resizing:   {},
				}

				g.Expect(err).To(Succeed())
				diffVolumes(g, volumes, expectVolumes)
			},
		},
		"not exist in desired volumes": {
			setup: func(ctx *componentVolumeContext) []*volume {
				ctx.desiredVolumeQuantity = map[v1alpha1.StorageVolumeName]resource.Quantity{}
				volumes := []*volume{
					newVolume("volume-1", newMockPVC("volume-1-pvc-1", scName, "1Gi", "1Gi")),
				}

				return volumes
			},
			sc: newStorageClass(scName, false),
			expect: func(g *GomegaWithT, volumes map[volumePhase][]*volume, err error) {
				expectVolumes := map[volumePhase][]*volume{
					needResize: {},
					resized:    {},
					resizing:   {},
				}

				g.Expect(err).To(Succeed())
				diffVolumes(g, volumes, expectVolumes)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			resizer := &pvcResizer{
				deps: controller.NewFakeDependencies(),
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.sc != nil {
				resizer.deps.KubeClientset.StorageV1().StorageClasses().Create(context.TODO(), tt.sc, metav1.CreateOptions{})
			}
			informerFactory := resizer.deps.KubeInformerFactory
			informerFactory.Start(ctx.Done())
			informerFactory.WaitForCacheSync(ctx.Done())

			vctx := &componentVolumeContext{}
			vctx.cluster = &v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cluster"},
			}
			vctx.status = &v1alpha1.PDStatus{}
			vctx.desiredVolumeQuantity = map[v1alpha1.StorageVolumeName]resource.Quantity{}
			volumes := tt.setup(vctx)
			classifiedVolumes, err := resizer.classifyVolumes(vctx, volumes)
			tt.expect(g, classifiedVolumes, err)
		})
	}
}

func TestResizeHook(t *testing.T) {
	t.Run("beginResize", func(t *testing.T) {
		g := NewGomegaWithT(t)
		resizer := &pvcResizer{}
		tc := &v1alpha1.TidbCluster{
			ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cluster"},
			Spec: v1alpha1.TidbClusterSpec{
				TiKV: &v1alpha1.TiKVSpec{},
			},
		}
		ctx := &componentVolumeContext{
			cluster: tc,
			status:  &tc.Status.TiKV,
		}

		err := resizer.beginResize(ctx)
		g.Expect(err).To(MatchError(controller.RequeueErrorf("set condition before resizing volumes for test-ns/test-cluster:tikv")))
		g.Expect(len(tc.Status.TiKV.Conditions)).To(Equal(1))
		g.Expect(tc.Status.TiKV.Conditions[0].Reason).To(Equal("BeginResizing"))
		g.Expect(tc.Status.TiKV.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
		g.Expect(tc.IsComponentVolumeResizing(v1alpha1.TiKVMemberType)).To(BeTrue())
	})

	t.Run("endResize", func(t *testing.T) {
		testcases := map[string]struct {
			setup  func(ctx *componentVolumeContext)
			expect func(g *GomegaWithT, p *pvcResizer, ctx *componentVolumeContext, err error)
		}{
			"set condition": {
				setup: func(ctx *componentVolumeContext) {
					ctx.status = &v1alpha1.PDStatus{
						Conditions: []metav1.Condition{
							{
								Type:    v1alpha1.ComponentVolumeResizing,
								Status:  metav1.ConditionTrue,
								Reason:  "BeginResizing",
								Message: "Set resizing condition to begin resizing",
							},
						},
					}
				},
				expect: func(g *GomegaWithT, p *pvcResizer, ctx *componentVolumeContext, err error) {
					conds := ctx.status.GetConditions()
					g.Expect(len(conds)).To(Equal(1))
					g.Expect(conds[0].Reason).To(Equal("EndResizing"))
					g.Expect(conds[0].Status).To(Equal(metav1.ConditionFalse))
					g.Expect(ctx.cluster.(*v1alpha1.TidbCluster).IsComponentVolumeResizing(v1alpha1.TiKVMemberType)).To(BeFalse())
				},
			},
			"remove eviction annotation for tikv": {
				setup: func(ctx *componentVolumeContext) {
					ctx.status = &v1alpha1.TiKVStatus{}
					ctx.actualPodVolumes = []*podVolumeContext{
						{
							pod: &corev1.Pod{
								ObjectMeta: metav1.ObjectMeta{
									Namespace: "test-ns", Name: "test-cluster-tikv-0",
									Annotations: map[string]string{v1alpha1.EvictLeaderAnnKeyForResize: "1"},
								},
							},
						},
						{
							pod: &corev1.Pod{
								ObjectMeta: metav1.ObjectMeta{
									Namespace: "test-ns", Name: "test-cluster-tikv-1",
								}},
						},
						{
							pod: &corev1.Pod{
								ObjectMeta: metav1.ObjectMeta{
									Namespace: "test-ns", Name: "test-cluster-tikv-2",
									Annotations: map[string]string{v1alpha1.EvictLeaderAnnKeyForResize: "123"},
								},
							},
						},
					}
				},
				expect: func(g *GomegaWithT, p *pvcResizer, ctx *componentVolumeContext, err error) {
					g.Expect(err).To(Succeed())

					cli := p.deps.KubeClientset.CoreV1().Pods(ctx.cluster.GetNamespace())
					pod, err := cli.Get(context.TODO(), "test-cluster-tikv-0", metav1.GetOptions{})
					g.Expect(err).To(Succeed())
					g.Expect(pod.Annotations).NotTo(HaveKey(v1alpha1.EvictLeaderAnnKeyForResize))
					pod, err = cli.Get(context.TODO(), "test-cluster-tikv-1", metav1.GetOptions{})
					g.Expect(err).To(Succeed())
					g.Expect(pod.Annotations).NotTo(HaveKey(v1alpha1.EvictLeaderAnnKeyForResize))
					pod, err = cli.Get(context.TODO(), "test-cluster-tikv-2", metav1.GetOptions{})
					g.Expect(err).To(Succeed())
					g.Expect(pod.Annotations).NotTo(HaveKey(v1alpha1.EvictLeaderAnnKeyForResize))
				},
			},
		}

		for name, tt := range testcases {
			t.Run(name, func(t *testing.T) {
				g := NewGomegaWithT(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				fakeDeps := controller.NewFakeDependencies()
				informerFactory := fakeDeps.KubeInformerFactory

				resizer := &pvcResizer{
					deps: fakeDeps,
				}
				tc := &v1alpha1.TidbCluster{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cluster"},
				}
				vctx := &componentVolumeContext{
					cluster: tc,
				}
				tt.setup(vctx)

				for _, podVol := range vctx.actualPodVolumes {
					if podVol.pod != nil {
						fakeDeps.KubeClientset.CoreV1().Pods(tc.Namespace).Create(context.TODO(), podVol.pod, metav1.CreateOptions{})
					}
				}
				informerFactory.Start(ctx.Done())
				informerFactory.WaitForCacheSync(ctx.Done())

				err := resizer.endResize(vctx)
				tt.expect(g, resizer, vctx, err)
			})
		}
	})

	t.Run("beforeResizeForPod", func(t *testing.T) {
		testcases := map[string]struct {
			setup  func(ctx *componentVolumeContext) *corev1.Pod
			expect func(g *GomegaWithT, p *pvcResizer, ctx *componentVolumeContext, err error)
		}{
			"succeed": {
				setup: func(ctx *componentVolumeContext) *corev1.Pod {
					tc := ctx.cluster.(*v1alpha1.TidbCluster)
					ctx.status = &tc.Status.TiKV

					tc.Status.TiKV = v1alpha1.TiKVStatus{
						Stores: map[string]v1alpha1.TiKVStore{
							"0": {
								ID:          "0",
								PodName:     "test-cluster-tikv-0",
								State:       v1alpha1.TiKVStateUp,
								LeaderCount: 10,
							},
							"1": {
								ID:          "1",
								PodName:     "test-cluster-tikv-1",
								State:       v1alpha1.TiKVStateUp,
								LeaderCount: 0,
							},
							"2": {
								ID:          "2",
								PodName:     "test-cluster-tikv-2",
								State:       v1alpha1.TiKVStateUp,
								LeaderCount: 10,
							},
						},
					}
					ctx.actualPodVolumes = []*podVolumeContext{
						{
							pod: &corev1.Pod{
								ObjectMeta: metav1.ObjectMeta{
									Namespace: "test-ns", Name: "test-cluster-tikv-0",
								},
							},
						},
						{
							pod: &corev1.Pod{
								ObjectMeta: metav1.ObjectMeta{
									Namespace: "test-ns", Name: "test-cluster-tikv-1",
									Annotations: map[string]string{v1alpha1.EvictLeaderAnnKeyForResize: "none"},
								}},
						},
						{
							pod: &corev1.Pod{
								ObjectMeta: metav1.ObjectMeta{
									Namespace: "test-ns", Name: "test-cluster-tikv-2",
								},
							},
						},
					}

					return ctx.actualPodVolumes[1].pod
				},
				expect: func(g *GomegaWithT, p *pvcResizer, ctx *componentVolumeContext, err error) {
					g.Expect(err).To(Succeed())
				},
			},
			"any store is not Up": {
				setup: func(ctx *componentVolumeContext) *corev1.Pod {
					tc := ctx.cluster.(*v1alpha1.TidbCluster)
					ctx.status = &tc.Status.TiKV

					tc.Status.TiKV = v1alpha1.TiKVStatus{
						Stores: map[string]v1alpha1.TiKVStore{
							"0": {
								ID:      "0",
								PodName: "test-cluster-tikv-0",
								State:   v1alpha1.TiKVStateUp,
							},
							"1": {
								ID:      "1",
								PodName: "test-cluster-tikv-1",
								State:   v1alpha1.TiKVStateDown,
							},
							"2": {
								ID:      "2",
								PodName: "test-cluster-tikv-2",
								State:   v1alpha1.TiKVStateUp,
							},
						},
					}
					ctx.actualPodVolumes = []*podVolumeContext{
						{
							pod: &corev1.Pod{
								ObjectMeta: metav1.ObjectMeta{
									Namespace: "test-ns", Name: "test-cluster-tikv-0",
								},
							},
						},
						{
							pod: &corev1.Pod{
								ObjectMeta: metav1.ObjectMeta{
									Namespace: "test-ns", Name: "test-cluster-tikv-1",
								}},
						},
						{
							pod: &corev1.Pod{
								ObjectMeta: metav1.ObjectMeta{
									Namespace: "test-ns", Name: "test-cluster-tikv-2",
								},
							},
						},
					}

					return ctx.actualPodVolumes[2].pod
				},
				expect: func(g *GomegaWithT, p *pvcResizer, ctx *componentVolumeContext, err error) {
					g.Expect(err).To(HaveOccurred())
					g.Expect(err.Error()).To(ContainSubstring("store 1 of pod test-cluster-tikv-1 is not ready"))
				},
			},
			"sync leader eviction annotation": {
				setup: func(ctx *componentVolumeContext) *corev1.Pod {
					tc := ctx.cluster.(*v1alpha1.TidbCluster)
					ctx.status = &tc.Status.TiKV

					tc.Status.TiKV = v1alpha1.TiKVStatus{
						Stores: map[string]v1alpha1.TiKVStore{
							"0": {
								ID:      "0",
								PodName: "test-cluster-tikv-0",
								State:   v1alpha1.TiKVStateUp,
							},
							"1": {
								ID:      "1",
								PodName: "test-cluster-tikv-1",
								State:   v1alpha1.TiKVStateUp,
							},
							"2": {
								ID:      "2",
								PodName: "test-cluster-tikv-2",
								State:   v1alpha1.TiKVStateUp,
							},
						},
					}
					ctx.actualPodVolumes = []*podVolumeContext{
						{
							pod: &corev1.Pod{
								ObjectMeta: metav1.ObjectMeta{
									Namespace: "test-ns", Name: "test-cluster-tikv-0",
									Annotations: map[string]string{v1alpha1.EvictLeaderAnnKeyForResize: "none"},
								},
							},
						},
						{
							pod: &corev1.Pod{
								ObjectMeta: metav1.ObjectMeta{
									Namespace: "test-ns", Name: "test-cluster-tikv-1",
								}},
						},
						{
							pod: &corev1.Pod{
								ObjectMeta: metav1.ObjectMeta{
									Namespace: "test-ns", Name: "test-cluster-tikv-2",
									Annotations: map[string]string{v1alpha1.EvictLeaderAnnKeyForResize: "none"},
								},
							},
						},
					}

					return ctx.actualPodVolumes[1].pod
				},
				expect: func(g *GomegaWithT, p *pvcResizer, ctx *componentVolumeContext, err error) {
					g.Expect(err).To(Succeed())

					cli := p.deps.KubeClientset.CoreV1().Pods(ctx.cluster.GetNamespace())
					pod, err := cli.Get(context.TODO(), "test-cluster-tikv-0", metav1.GetOptions{})
					g.Expect(err).To(Succeed())
					g.Expect(pod.Annotations).NotTo(HaveKey(v1alpha1.EvictLeaderAnnKeyForResize))
					pod, err = cli.Get(context.TODO(), "test-cluster-tikv-1", metav1.GetOptions{})
					g.Expect(err).To(Succeed())
					g.Expect(pod.Annotations).To(HaveKey(v1alpha1.EvictLeaderAnnKeyForResize))
					pod, err = cli.Get(context.TODO(), "test-cluster-tikv-2", metav1.GetOptions{})
					g.Expect(err).To(Succeed())
					g.Expect(pod.Annotations).To(HaveKey(v1alpha1.EvictLeaderAnnKeyForResize))
				},
			},
			"wait store to be 0": {
				setup: func(ctx *componentVolumeContext) *corev1.Pod {
					tc := ctx.cluster.(*v1alpha1.TidbCluster)
					ctx.status = &tc.Status.TiKV

					tc.Status.TiKV = v1alpha1.TiKVStatus{
						Stores: map[string]v1alpha1.TiKVStore{
							"0": {
								ID:          "0",
								PodName:     "test-cluster-tikv-0",
								State:       v1alpha1.TiKVStateUp,
								LeaderCount: 10,
							},
							"1": {
								ID:          "1",
								PodName:     "test-cluster-tikv-1",
								State:       v1alpha1.TiKVStateUp,
								LeaderCount: 10,
							},
							"2": {
								ID:          "2",
								PodName:     "test-cluster-tikv-2",
								State:       v1alpha1.TiKVStateUp,
								LeaderCount: 10,
							},
						},
					}
					ctx.actualPodVolumes = []*podVolumeContext{
						{
							pod: &corev1.Pod{
								ObjectMeta: metav1.ObjectMeta{
									Namespace: "test-ns", Name: "test-cluster-tikv-0",
								},
							},
						},
						{
							pod: &corev1.Pod{
								ObjectMeta: metav1.ObjectMeta{
									Namespace: "test-ns", Name: "test-cluster-tikv-1",
									Annotations: map[string]string{v1alpha1.EvictLeaderAnnKeyForResize: "none"},
								}},
						},
						{
							pod: &corev1.Pod{
								ObjectMeta: metav1.ObjectMeta{
									Namespace: "test-ns", Name: "test-cluster-tikv-2",
								},
							},
						},
					}

					return ctx.actualPodVolumes[1].pod
				},
				expect: func(g *GomegaWithT, p *pvcResizer, ctx *componentVolumeContext, err error) {
					g.Expect(err).To(HaveOccurred())
					g.Expect(err.Error()).To(ContainSubstring("wait for leader count of store 1 to be 0"))
				},
			},
		}

		for name, tt := range testcases {
			t.Run(name, func(t *testing.T) {
				g := NewGomegaWithT(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				fakeDeps := controller.NewFakeDependencies()
				informerFactory := fakeDeps.KubeInformerFactory
				informerFactory.Start(ctx.Done())
				informerFactory.WaitForCacheSync(ctx.Done())

				resizer := &pvcResizer{
					deps: fakeDeps,
				}
				tc := &v1alpha1.TidbCluster{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-cluster"},
					Spec: v1alpha1.TidbClusterSpec{
						TiKV: &v1alpha1.TiKVSpec{
							Replicas: 3,
						},
					},
				}
				vctx := &componentVolumeContext{
					cluster: tc,
				}
				resizePod := tt.setup(vctx)
				volumes := []*volume{}
				for _, podVol := range vctx.actualPodVolumes {
					if podVol.pod != nil {
						fakeDeps.KubeClientset.CoreV1().Pods(tc.Namespace).Create(context.TODO(), podVol.pod, metav1.CreateOptions{})
					}
					if podVol.pod == resizePod {
						volumes = podVol.volumes
					}
				}

				err := resizer.beforeResizeForPod(vctx, resizePod, volumes)
				tt.expect(g, resizer, vctx, err)
			})
		}
	})
}

func newVolume(name v1alpha1.StorageVolumeName, pvc *corev1.PersistentVolumeClaim) *volume {
	return &volume{name: name, pvc: pvc}
}

func diffVolume(v1 *volume, v2 *volume) string {
	if diff := cmp.Diff(v1.name, v2.name); diff != "" {
		return diff
	}
	if diff := cmp.Diff(v1.pvc, v2.pvc); diff != "" {
		return diff
	}
	return ""
}
