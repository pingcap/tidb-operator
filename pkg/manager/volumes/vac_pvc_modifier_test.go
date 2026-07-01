// Copyright 2025 PingCAP, Inc.
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

package volumes

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/stretchr/testify/assert"
)

func newFakeVACModifier() *vacPVCModifier {
	deps := controller.NewFakeDependencies()
	return newVACPVCModifier(deps)
}

func TestVacPVCModifier_tryToModifyOnePodPVCs(t *testing.T) {
	actualVolumes := []ActualVolume{
		{
			PVC: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns-1",
					Name:      "pvc-1",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("100Gi"),
						},
					},
				},
			},
			Desired: &DesiredVolume{
				Size: resource.MustParse("100Gi"),
			},
		},
		{
			PVC: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns-1",
					Name:      "pvc-2",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("100Gi"),
						},
					},
				},
			},
			Desired: &DesiredVolume{
				Size:                      resource.MustParse("100Gi"),
				VolumeAttributesClassName: pointer.String("vac-1"),
			},
		},
		{
			PVC: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns-1",
					Name:      "pvc-3",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("100Gi"),
						},
					},
				},
			},
			Desired: &DesiredVolume{
				Size:                      resource.MustParse("200Gi"),
				VolumeAttributesClassName: pointer.String("vac-1"),
			},
		},
	}

	ctx := &componentVolumeContext{
		tc: &v1alpha1.TidbCluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "test",
			},
		},
		status: &v1alpha1.TiKVStatus{},
	}
	modifier := newFakeVACModifier()

	for _, actual := range actualVolumes {
		_, err := modifier.deps.KubeClientset.CoreV1().PersistentVolumeClaims(actual.PVC.Namespace).Create(ctx, actual.PVC, metav1.CreateOptions{})
		assert.NoError(t, err)
	}

	err := modifier.tryToModifyOnePodPVCs(ctx, actualVolumes)
	assert.NoError(t, err)

	for _, actual := range actualVolumes {
		actualSize := actual.PVC.Spec.Resources.Requests[corev1.ResourceStorage]
		assert.True(t, actual.Desired.Size.Equal(actualSize))
		assert.Equal(t, actual.Desired.VolumeAttributesClassName, actual.PVC.Spec.VolumeAttributesClassName)
	}
}

func TestVacPVCModifier_isStatefulSetSynced(t *testing.T) {
	testcases := []struct {
		name           string
		desiredVolumes []DesiredVolume
		sts            *appsv1.StatefulSet
		err            bool
		synced         bool
	}{
		{
			name:           "desired volumes missing",
			desiredVolumes: []DesiredVolume{},
			sts: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "sts-1",
				},
				Spec: appsv1.StatefulSetSpec{
					VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "pvc-1",
							},
							Spec: corev1.PersistentVolumeClaimSpec{
								Resources: corev1.VolumeResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceStorage: resource.MustParse("100Gi"),
									},
								},
							},
						},
					},
				},
			},
			err:    true,
			synced: false,
		},
		{
			name: "one volume modified",
			desiredVolumes: []DesiredVolume{
				{
					Name: v1alpha1.StorageVolumeName("pvc-1"),
					Size: resource.MustParse("100Gi"),
				},
				{
					Name:                      v1alpha1.StorageVolumeName("pvc-2"),
					Size:                      resource.MustParse("100Gi"),
					VolumeAttributesClassName: pointer.String("vac-1"),
				},
			},
			sts: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "sts-1",
				},
				Spec: appsv1.StatefulSetSpec{
					VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "pvc-1",
							},
							Spec: corev1.PersistentVolumeClaimSpec{
								Resources: corev1.VolumeResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceStorage: resource.MustParse("100Gi"),
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "pvc-2",
							},
							Spec: corev1.PersistentVolumeClaimSpec{
								Resources: corev1.VolumeResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceStorage: resource.MustParse("100Gi"),
									},
								},
							},
						},
					},
				},
			},
			err:    false,
			synced: false,
		},
		{
			name: "sts synced",
			desiredVolumes: []DesiredVolume{
				{
					Name: v1alpha1.StorageVolumeName("pvc-1"),
					Size: resource.MustParse("100Gi"),
				},
				{
					Name:                      v1alpha1.StorageVolumeName("pvc-2"),
					Size:                      resource.MustParse("100Gi"),
					VolumeAttributesClassName: pointer.String("vac-1"),
				},
			},
			sts: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "sts-1",
				},
				Spec: appsv1.StatefulSetSpec{
					VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "pvc-1",
							},
							Spec: corev1.PersistentVolumeClaimSpec{
								Resources: corev1.VolumeResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceStorage: resource.MustParse("100Gi"),
									},
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "pvc-2",
							},
							Spec: corev1.PersistentVolumeClaimSpec{
								Resources: corev1.VolumeResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceStorage: resource.MustParse("100Gi"),
									},
								},
								VolumeAttributesClassName: pointer.String("vac-1"),
							},
						},
					},
				},
			},
			err:    false,
			synced: true,
		},
	}

	ctx := &componentVolumeContext{
		tc: &v1alpha1.TidbCluster{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns-1",
				Name:      "test",
			},
		},
		status: &v1alpha1.TiKVStatus{},
	}
	modifier := newFakeVACModifier()
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			ctx.desiredVolumes = tt.desiredVolumes
			synced, err := modifier.isStatefulSetSynced(ctx, tt.sts)
			if tt.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.synced, synced)
			}
		})
	}
}

func TestVacPVCModifier_isPVCModified(t *testing.T) {
	testcases := []struct {
		name           string
		pvc            *corev1.PersistentVolumeClaim
		desiredSize    resource.Quantity
		desiredVACName *string
		expected       bool
	}{
		{
			name: "pvc not modified with vac nil",
			pvc: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pvc-1",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("100Gi"),
						},
					},
				},
			},
			desiredSize:    resource.MustParse("100Gi"),
			desiredVACName: nil,
			expected:       false,
		},
		{
			name: "pvc not modified with vac name",
			pvc: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pvc-1",
					Annotations: map[string]string{
						"vac.pingcap.com/vac-name": "vac-1",
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("100Gi"),
						},
					},
					VolumeAttributesClassName: pointer.String("vac-1"),
				},
			},
			desiredSize:    resource.MustParse("100Gi"),
			desiredVACName: pointer.String("vac-1"),
			expected:       false,
		},
		{
			name: "pvc modified with vac name changed",
			pvc: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pvc-1",
					Annotations: map[string]string{
						"vac.pingcap.com/vac-name": "vac-1",
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("100Gi"),
						},
					},
					VolumeAttributesClassName: pointer.String("vac-1"),
				},
			},
			desiredSize:    resource.MustParse("100Gi"),
			desiredVACName: pointer.String("vac-2"),
			expected:       true,
		},
		{
			name: "pvc modified with vac name added",
			pvc: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pvc-1",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("100Gi"),
						},
					},
				},
			},
			desiredSize:    resource.MustParse("100Gi"),
			desiredVACName: pointer.String("vac-1"),
			expected:       true,
		},
		{
			name: "pvc modified with vac name removed",
			pvc: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pvc-1",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("100Gi"),
						},
					},
					VolumeAttributesClassName: pointer.String("vac-1"),
				},
			},
			desiredSize:    resource.MustParse("100Gi"),
			desiredVACName: nil,
			expected:       true,
		},
		{
			name: "pvc modified with vac size changed",
			pvc: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pvc-1",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("100Gi"),
						},
					},
				},
			},
			desiredSize:    resource.MustParse("200Gi"),
			desiredVACName: nil,
			expected:       true,
		},
	}

	modifier := newFakeVACModifier()
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			actual := modifier.isPVCModified(tt.pvc, tt.desiredSize, tt.desiredVACName)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
