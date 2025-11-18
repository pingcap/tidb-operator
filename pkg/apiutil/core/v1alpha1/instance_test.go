// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package coreutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
)

func TestPVCs(t *testing.T) {
	cases := []struct {
		desc string
		c    *v1alpha1.Cluster
		obj  *v1alpha1.PD
		ps   []PVCPatch

		isPanic bool
		pvcs    []*corev1.PersistentVolumeClaim
	}{
		{
			desc: "no volumes",
			c:    fake.FakeObj[v1alpha1.Cluster]("aaa"),
			obj:  fake.FakeObj[v1alpha1.PD]("aaa-hash"),
		},
		{
			desc: "one volume",
			c:    fake.FakeObj[v1alpha1.Cluster]("aaa"),
			obj: fake.FakeObj("aaa-hash", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Spec.Cluster.Name = "aaa"
				obj.Spec.Volumes = []v1alpha1.Volume{
					{
						Name:                      "data",
						Storage:                   resource.MustParse("10Gi"),
						StorageClassName:          ptr.To("sc"),
						VolumeAttributesClassName: ptr.To("vac"),
					},
				}
				return obj
			}),
			ps: []PVCPatch{WithLegacyK8sAppLabels(), EnableVAC(true)},

			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "data-aaa-pd-hash",
						Labels: map[string]string{
							v1alpha1.LabelKeyManagedBy:     v1alpha1.LabelValManagedByOperator,
							v1alpha1.LabelKeyComponent:     "pd",
							v1alpha1.LabelKeyCluster:       "aaa",
							v1alpha1.LabelKeyInstance:      "aaa-hash",
							v1alpha1.LabelKeyVolumeName:    "data",
							"app.kubernetes.io/component":  "pd",
							"app.kubernetes.io/instance":   "aaa",
							"app.kubernetes.io/managed-by": "tidb-operator",
							"app.kubernetes.io/name":       "tidb-cluster",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "core.pingcap.com/v1alpha1",
								Kind:               "PD",
								Name:               "aaa-hash",
								UID:                "",
								BlockOwnerDeletion: ptr.To(true),
								Controller:         ptr.To(true),
							},
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("10Gi"),
							},
						},
						StorageClassName:          ptr.To("sc"),
						VolumeAttributesClassName: ptr.To("vac"),
					},
				},
			},
		},
		{
			desc: "one volume -- no vac",
			c:    fake.FakeObj[v1alpha1.Cluster]("aaa"),
			obj: fake.FakeObj("aaa-hash", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Spec.Cluster.Name = "aaa"
				obj.Spec.Volumes = []v1alpha1.Volume{
					{
						Name:                      "data",
						Storage:                   resource.MustParse("10Gi"),
						StorageClassName:          ptr.To("sc"),
						VolumeAttributesClassName: ptr.To("vac"),
					},
				}
				return obj
			}),
			ps: []PVCPatch{WithLegacyK8sAppLabels()},

			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "data-aaa-pd-hash",
						Labels: map[string]string{
							v1alpha1.LabelKeyManagedBy:     v1alpha1.LabelValManagedByOperator,
							v1alpha1.LabelKeyComponent:     "pd",
							v1alpha1.LabelKeyCluster:       "aaa",
							v1alpha1.LabelKeyInstance:      "aaa-hash",
							v1alpha1.LabelKeyVolumeName:    "data",
							"app.kubernetes.io/component":  "pd",
							"app.kubernetes.io/instance":   "aaa",
							"app.kubernetes.io/managed-by": "tidb-operator",
							"app.kubernetes.io/name":       "tidb-cluster",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "core.pingcap.com/v1alpha1",
								Kind:               "PD",
								Name:               "aaa-hash",
								UID:                "",
								BlockOwnerDeletion: ptr.To(true),
								Controller:         ptr.To(true),
							},
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("10Gi"),
							},
						},
						StorageClassName: ptr.To("sc"),
					},
				},
			},
		},
		{
			desc: "two volume",
			c:    fake.FakeObj[v1alpha1.Cluster]("aaa"),
			obj: fake.FakeObj("aaa-hash", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Spec.Cluster.Name = "aaa"
				obj.Spec.Volumes = []v1alpha1.Volume{
					{
						Name:                      "data",
						Storage:                   resource.MustParse("10Gi"),
						StorageClassName:          ptr.To("sc"),
						VolumeAttributesClassName: ptr.To("vac"),
					},
					{
						Name:                      "tmp",
						Storage:                   resource.MustParse("20Gi"),
						StorageClassName:          ptr.To("sc2"),
						VolumeAttributesClassName: ptr.To("vac2"),
					},
				}
				return obj
			}),

			ps: []PVCPatch{WithLegacyK8sAppLabels(), EnableVAC(true)},

			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "data-aaa-pd-hash",
						Labels: map[string]string{
							v1alpha1.LabelKeyManagedBy:     v1alpha1.LabelValManagedByOperator,
							v1alpha1.LabelKeyComponent:     "pd",
							v1alpha1.LabelKeyCluster:       "aaa",
							v1alpha1.LabelKeyInstance:      "aaa-hash",
							v1alpha1.LabelKeyVolumeName:    "data",
							"app.kubernetes.io/component":  "pd",
							"app.kubernetes.io/instance":   "aaa",
							"app.kubernetes.io/managed-by": "tidb-operator",
							"app.kubernetes.io/name":       "tidb-cluster",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "core.pingcap.com/v1alpha1",
								Kind:               "PD",
								Name:               "aaa-hash",
								UID:                "",
								BlockOwnerDeletion: ptr.To(true),
								Controller:         ptr.To(true),
							},
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("10Gi"),
							},
						},
						StorageClassName:          ptr.To("sc"),
						VolumeAttributesClassName: ptr.To("vac"),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "tmp-aaa-pd-hash",
						Labels: map[string]string{
							v1alpha1.LabelKeyManagedBy:     v1alpha1.LabelValManagedByOperator,
							v1alpha1.LabelKeyComponent:     "pd",
							v1alpha1.LabelKeyCluster:       "aaa",
							v1alpha1.LabelKeyInstance:      "aaa-hash",
							v1alpha1.LabelKeyVolumeName:    "tmp",
							"app.kubernetes.io/component":  "pd",
							"app.kubernetes.io/instance":   "aaa",
							"app.kubernetes.io/managed-by": "tidb-operator",
							"app.kubernetes.io/name":       "tidb-cluster",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "core.pingcap.com/v1alpha1",
								Kind:               "PD",
								Name:               "aaa-hash",
								UID:                "",
								BlockOwnerDeletion: ptr.To(true),
								Controller:         ptr.To(true),
							},
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("20Gi"),
							},
						},
						StorageClassName:          ptr.To("sc2"),
						VolumeAttributesClassName: ptr.To("vac2"),
					},
				},
			},
		},
		{
			desc: "two volume -- overlay annotations",
			c:    fake.FakeObj[v1alpha1.Cluster]("aaa"),
			obj: fake.FakeObj("aaa-hash", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Spec.Cluster.Name = "aaa"
				obj.Spec.Volumes = []v1alpha1.Volume{
					{
						Name:                      "data",
						Storage:                   resource.MustParse("10Gi"),
						StorageClassName:          ptr.To("sc"),
						VolumeAttributesClassName: ptr.To("vac"),
					},
					{
						Name:                      "tmp",
						Storage:                   resource.MustParse("20Gi"),
						StorageClassName:          ptr.To("sc2"),
						VolumeAttributesClassName: ptr.To("vac2"),
					},
				}
				obj.Spec.Overlay = &v1alpha1.Overlay{
					PersistentVolumeClaims: []v1alpha1.NamedPersistentVolumeClaimOverlay{
						{
							Name: "tmp",
							PersistentVolumeClaim: v1alpha1.PersistentVolumeClaimOverlay{
								ObjectMeta: v1alpha1.ObjectMeta{
									Annotations: map[string]string{
										"test": "test",
									},
								},
							},
						},
					},
				}
				return obj
			}),

			ps: []PVCPatch{WithLegacyK8sAppLabels(), EnableVAC(true)},

			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "data-aaa-pd-hash",
						Labels: map[string]string{
							v1alpha1.LabelKeyManagedBy:     v1alpha1.LabelValManagedByOperator,
							v1alpha1.LabelKeyComponent:     "pd",
							v1alpha1.LabelKeyCluster:       "aaa",
							v1alpha1.LabelKeyInstance:      "aaa-hash",
							v1alpha1.LabelKeyVolumeName:    "data",
							"app.kubernetes.io/component":  "pd",
							"app.kubernetes.io/instance":   "aaa",
							"app.kubernetes.io/managed-by": "tidb-operator",
							"app.kubernetes.io/name":       "tidb-cluster",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "core.pingcap.com/v1alpha1",
								Kind:               "PD",
								Name:               "aaa-hash",
								UID:                "",
								BlockOwnerDeletion: ptr.To(true),
								Controller:         ptr.To(true),
							},
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("10Gi"),
							},
						},
						StorageClassName:          ptr.To("sc"),
						VolumeAttributesClassName: ptr.To("vac"),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "tmp-aaa-pd-hash",
						Labels: map[string]string{
							v1alpha1.LabelKeyManagedBy:     v1alpha1.LabelValManagedByOperator,
							v1alpha1.LabelKeyComponent:     "pd",
							v1alpha1.LabelKeyCluster:       "aaa",
							v1alpha1.LabelKeyInstance:      "aaa-hash",
							v1alpha1.LabelKeyVolumeName:    "tmp",
							"app.kubernetes.io/component":  "pd",
							"app.kubernetes.io/instance":   "aaa",
							"app.kubernetes.io/managed-by": "tidb-operator",
							"app.kubernetes.io/name":       "tidb-cluster",
						},
						Annotations: map[string]string{
							"test": "test",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "core.pingcap.com/v1alpha1",
								Kind:               "PD",
								Name:               "aaa-hash",
								UID:                "",
								BlockOwnerDeletion: ptr.To(true),
								Controller:         ptr.To(true),
							},
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("20Gi"),
							},
						},
						StorageClassName:          ptr.To("sc2"),
						VolumeAttributesClassName: ptr.To("vac2"),
					},
				},
			},
		},
		{
			desc: "panic overlay",
			c:    fake.FakeObj[v1alpha1.Cluster]("aaa"),
			obj: fake.FakeObj("aaa-hash", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Spec.Cluster.Name = "aaa"
				obj.Spec.Volumes = []v1alpha1.Volume{
					{
						Name:                      "data",
						Storage:                   resource.MustParse("10Gi"),
						StorageClassName:          ptr.To("sc"),
						VolumeAttributesClassName: ptr.To("vac"),
					},
				}
				obj.Spec.Overlay = &v1alpha1.Overlay{
					PersistentVolumeClaims: []v1alpha1.NamedPersistentVolumeClaimOverlay{
						{
							Name: "tmp",
							PersistentVolumeClaim: v1alpha1.PersistentVolumeClaimOverlay{
								ObjectMeta: v1alpha1.ObjectMeta{
									Annotations: map[string]string{
										"test": "test",
									},
								},
							},
						},
					},
				}
				return obj
			}),

			isPanic: true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()
			if c.isPanic {
				assert.Panics(tt, func() {
					PVCs[scope.PD](c.c, c.obj, c.ps...)
				})
			} else {
				pvcs := PVCs[scope.PD](c.c, c.obj, c.ps...)
				assert.Equal(tt, c.pvcs, pvcs)
			}
		})
	}
}
