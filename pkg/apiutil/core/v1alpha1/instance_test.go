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
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
)

func TestRetryIfInstancesReadyButNotAvailableReturnsRemainingTime(t *testing.T) {
	now := time.Now()
	pd := fake.FakeObj("pd-0", func(obj *v1alpha1.PD) *v1alpha1.PD {
		obj.Status.Conditions = []metav1.Condition{
			{
				Type:               v1alpha1.CondReady,
				Status:             metav1.ConditionTrue,
				ObservedGeneration: obj.Generation,
				LastTransitionTime: metav1.NewTime(now.Add(-56 * time.Second)),
			},
		}
		return obj
	})

	retryAfter := RetryIfInstancesReadyButNotAvailable[scope.PD]([]*v1alpha1.PD{pd}, 60)
	assert.Greater(t, retryAfter, time.Duration(0))
	assert.LessOrEqual(t, retryAfter, 4*time.Second)

	maxRemainPD := fake.FakeObj("pd-1", func(obj *v1alpha1.PD) *v1alpha1.PD {
		obj.Status.Conditions = []metav1.Condition{
			{
				Type:               v1alpha1.CondReady,
				Status:             metav1.ConditionTrue,
				ObservedGeneration: obj.Generation,
				LastTransitionTime: metav1.NewTime(now.Add(-10 * time.Second)),
			},
		}
		return obj
	})
	retryAfter = RetryIfInstancesReadyButNotAvailable[scope.PD]([]*v1alpha1.PD{pd, maxRemainPD}, 60)
	assert.Greater(t, retryAfter, 49*time.Second)
	assert.LessOrEqual(t, retryAfter, 50*time.Second)

	readyLongEnough := fake.FakeObj("pd-1", func(obj *v1alpha1.PD) *v1alpha1.PD {
		obj.Status.Conditions = []metav1.Condition{
			{
				Type:               v1alpha1.CondReady,
				Status:             metav1.ConditionTrue,
				ObservedGeneration: obj.Generation,
				LastTransitionTime: metav1.NewTime(now.Add(-61 * time.Second)),
			},
		}
		return obj
	})
	assert.Zero(t, RetryIfInstancesReadyButNotAvailable[scope.PD]([]*v1alpha1.PD{readyLongEnough}, 60))
}

func TestRetryIfInstancesReadyButNotAvailableReturnsTiKVLeaderGateRemainingTime(t *testing.T) {
	tikv := fakeTiKVWithConditions("tikv-0", func(generation int64) []metav1.Condition {
		return []metav1.Condition{
			readyCondition(generation, time.Now().Add(-2*time.Minute)),
			leadersEvictedCondition(generation, metav1.ConditionFalse, v1alpha1.ReasonNotEvicted, time.Now().Add(-56*time.Second)),
		}
	})

	retryAfter := RetryIfInstancesReadyButNotAvailable[scope.TiKV]([]*v1alpha1.TiKV{tikv}, 60)
	assert.Greater(t, retryAfter, time.Duration(0))
	assert.LessOrEqual(t, retryAfter, 4*time.Second)
}

func TestRetryIfInstancesReadyButNotAvailableWaitsForTiKVLeaderGateEvents(t *testing.T) {
	tests := []struct {
		name            string
		minReadySeconds int64
		conditions      func(generation int64) []metav1.Condition
	}{
		{
			name:            "leader eviction condition missing",
			minReadySeconds: 60,
			conditions: func(generation int64) []metav1.Condition {
				return []metav1.Condition{
					readyCondition(generation, time.Now().Add(-2*time.Minute)),
				}
			},
		},
		{
			name:            "leaders are still evicting",
			minReadySeconds: 60,
			conditions: func(generation int64) []metav1.Condition {
				return []metav1.Condition{
					readyCondition(generation, time.Now().Add(-2*time.Minute)),
					leadersEvictedCondition(generation, metav1.ConditionTrue, v1alpha1.ReasonEvicted, time.Now().Add(-2*time.Minute)),
				}
			},
		},
		{
			name:            "leader eviction reason is invalid",
			minReadySeconds: 60,
			conditions: func(generation int64) []metav1.Condition {
				return []metav1.Condition{
					readyCondition(generation, time.Now().Add(-2*time.Minute)),
					leadersEvictedCondition(generation, metav1.ConditionFalse, v1alpha1.ReasonEvicted, time.Now().Add(-2*time.Minute)),
				}
			},
		},
		{
			name:            "zero min ready seconds and leader eviction condition missing",
			minReadySeconds: 0,
			conditions: func(generation int64) []metav1.Condition {
				return []metav1.Condition{
					readyCondition(generation, time.Now()),
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tikv := fakeTiKVWithConditions("tikv-0", tt.conditions)
			assert.Zero(t, RetryIfInstancesReadyButNotAvailable[scope.TiKV]([]*v1alpha1.TiKV{tikv}, tt.minReadySeconds))
		})
	}
}

func fakeTiKVWithConditions(name string, conditions func(generation int64) []metav1.Condition) *v1alpha1.TiKV {
	return fake.FakeObj(name, func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
		obj.Status.Conditions = conditions(obj.Generation)
		return obj
	})
}

func readyCondition(generation int64, lastTransitionTime time.Time) metav1.Condition {
	return metav1.Condition{
		Type:               v1alpha1.CondReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.NewTime(lastTransitionTime),
	}
}

func leadersEvictedCondition(generation int64, status metav1.ConditionStatus, reason string, lastTransitionTime time.Time) metav1.Condition {
	return metav1.Condition{
		Type:               v1alpha1.TiKVCondLeadersEvicted,
		Status:             status,
		ObservedGeneration: generation,
		Reason:             reason,
		LastTransitionTime: metav1.NewTime(lastTransitionTime),
	}
}

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
