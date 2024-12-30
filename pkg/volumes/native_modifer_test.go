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

package volumes

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
)

func Test_nativeModifier_GetActualVolume(t *testing.T) {
	tests := []struct {
		name         string
		existingObjs []client.Object
		current      *corev1.PersistentVolumeClaim
		expect       *corev1.PersistentVolumeClaim
		expectFunc   func(*WithT, *ActualVolume)
		wantErr      bool
	}{
		{
			name: "no changes",
			existingObjs: []client.Object{
				fake.FakeObj[corev1.PersistentVolume]("pv-0"),
				fake.FakeObj[storagev1.StorageClass]("sc-0"),
			},
			current: fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0",
				withPVCSpec(ptr.To("sc-0"), nil, "pv-0", "10Gi"),
				withPVCStatus("10Gi", nil)),
			expect: fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0",
				withPVCSpec(ptr.To("sc-0"), nil, "pv-0", "10Gi")),
			expectFunc: func(g *WithT, volume *ActualVolume) {
				g.Expect(volume).ShouldNot(BeNil())
				g.Expect(volume.Desired).ShouldNot(BeNil())
				g.Expect(volume.Desired.Size).Should(Equal(resource.MustParse("10Gi")))
				g.Expect(volume.Desired.StorageClassName).Should(BeNil())
				g.Expect(volume.Desired.StorageClass).Should(BeNil())

				g.Expect(volume.PVC).ShouldNot(BeNil())
				g.Expect(volume.PV).ShouldNot(BeNil())
				g.Expect(volume.StorageClass).ShouldNot(BeNil())
			},
		},
		{
			name: "set the VolumeAttributesClass",
			existingObjs: []client.Object{
				fake.FakeObj[corev1.PersistentVolume]("pv-0"),
				fake.FakeObj[storagev1.StorageClass]("sc-0"),
				fake.FakeObj[storagev1beta1.VolumeAttributesClass]("vac-0"),
			},
			current: fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0",
				withPVCSpec(ptr.To("sc-0"), nil, "pv-0", "10Gi"),
				withPVCStatus("10Gi", nil)),
			expect: fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0",
				withPVCSpec(ptr.To("sc-0"), ptr.To("vac-0"), "pv-0", "10Gi")),
			expectFunc: func(g *WithT, volume *ActualVolume) {
				g.Expect(volume).ShouldNot(BeNil())
				g.Expect(volume.VAC).Should(BeNil())
				g.Expect(volume.Desired).ShouldNot(BeNil())
				g.Expect(volume.Desired.VACName).Should(Equal(ptr.To("vac-0")))
				g.Expect(volume.Desired.VAC).ShouldNot(BeNil())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := client.NewFakeClient(tt.existingObjs...)
			m := NewNativeModifier(cli, logr.Discard())
			got, err := m.GetActualVolume(context.TODO(), tt.expect, tt.current)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetActualVolume() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.expectFunc != nil {
				tt.expectFunc(NewGomegaWithT(t), got)
			}
		})
	}
}

func Test_nativeModifier_ShouldModify(t *testing.T) {
	tests := []struct {
		name   string
		actual *ActualVolume
		want   bool
	}{
		{
			name: "pvc is being modified",
			actual: &ActualVolume{
				PVC: fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0",
					func(pvc *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
						pvc.Status.ModifyVolumeStatus = &corev1.ModifyVolumeStatus{
							TargetVolumeAttributesClassName: "vac-0",
							Status:                          corev1.PersistentVolumeClaimModifyVolumeInProgress,
						}
						return pvc
					}),
			},
		},
		{
			name: "try to shrink size",
			actual: &ActualVolume{
				Desired: &DesiredVolume{
					Size: resource.MustParse("5Gi"),
				},
				PVC: fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0", withPVCStatus("10Gi", nil)),
			},
		},
		{
			name: "try to expand size, but it's not supported",
			actual: &ActualVolume{
				Desired: &DesiredVolume{
					Size: resource.MustParse("50Gi"),
				},
				PVC: fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0", withPVCStatus("10Gi", nil)),
				StorageClass: fake.FakeObj[storagev1.StorageClass]("sc-0", func(sc *storagev1.StorageClass) *storagev1.StorageClass {
					sc.AllowVolumeExpansion = ptr.To(false)
					return sc
				}),
			},
		},
		{
			name: "try to expand size, and it's supported",
			actual: &ActualVolume{
				Desired: &DesiredVolume{
					Size: resource.MustParse("50Gi"),
				},
				PVC: fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0", withPVCStatus("10Gi", nil)),
				StorageClass: fake.FakeObj[storagev1.StorageClass]("sc-0", func(sc *storagev1.StorageClass) *storagev1.StorageClass {
					sc.AllowVolumeExpansion = ptr.To(true)
					return sc
				}),
			},
			want: true,
		},
		{
			name: "try to change sc",
			actual: &ActualVolume{
				Desired: &DesiredVolume{
					Size:             resource.MustParse("50Gi"),
					StorageClassName: ptr.To("sc-1"),
				},
				PVC:          fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0", withPVCStatus("10Gi", nil)),
				StorageClass: fake.FakeObj[storagev1.StorageClass]("sc-0"),
			},
		},
		{
			name: "modify attributes",
			actual: &ActualVolume{
				Desired: &DesiredVolume{
					Size:             resource.MustParse("10Gi"),
					StorageClassName: ptr.To("sc-0"),
					VACName:          ptr.To("vac-1"),
					VAC:              fake.FakeObj[storagev1beta1.VolumeAttributesClass]("vac-1"),
				},
				PVC:          fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0", withPVCSpec(ptr.To("sc-0"), ptr.To("vac-0"), "pv-0", "10Gi"), withPVCStatus("10Gi", nil)),
				StorageClass: fake.FakeObj[storagev1.StorageClass]("sc-0"),
			},
			want: true,
		},
		{
			name: "try to modify attributes with wrong vac",
			actual: &ActualVolume{
				Desired: &DesiredVolume{
					Size:             resource.MustParse("10Gi"),
					StorageClassName: ptr.To("sc-0"),
					VACName:          ptr.To("vac-1"),
					VAC: fake.FakeObj[storagev1beta1.VolumeAttributesClass]("vac-1", func(vac *storagev1beta1.VolumeAttributesClass) *storagev1beta1.VolumeAttributesClass {
						vac.DriverName = "wrong"
						return vac
					}),
				},
				PVC: fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0", withPVCSpec(ptr.To("sc-0"), ptr.To("vac-0"), "pv-0", "10Gi"), withPVCStatus("10Gi", nil)),
				StorageClass: fake.FakeObj[storagev1.StorageClass]("sc-0", func(sc *storagev1.StorageClass) *storagev1.StorageClass {
					sc.Provisioner = "right"
					return sc
				}),
			},
		},
		{
			name: "nothing is changed",
			actual: &ActualVolume{
				Desired: &DesiredVolume{
					Size:             resource.MustParse("10Gi"),
					StorageClassName: ptr.To("sc-0"),
					VACName:          ptr.To("vac-0"),
					VAC:              fake.FakeObj[storagev1beta1.VolumeAttributesClass]("vac-0"),
				},
				VACName: ptr.To("vac-0"),
				PVC: fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0",
					withPVCSpec(ptr.To("sc-0"), ptr.To("vac-0"), "pv-0", "10Gi"),
					withPVCStatus("10Gi", ptr.To("vac-0"))),
				StorageClass: fake.FakeObj[storagev1.StorageClass]("sc-0"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &nativeModifier{logger: logr.Discard()}
			if got := m.ShouldModify(context.TODO(), tt.actual); got != tt.want {
				t.Errorf("ShouldModify() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_nativeModifier_Modify(t *testing.T) {
	tests := []struct {
		name    string
		prePVC  *corev1.PersistentVolumeClaim
		vol     *ActualVolume
		wantErr bool
	}{
		{
			name: "happy path",
			prePVC: fake.FakeObj[corev1.PersistentVolumeClaim]("pvc",
				withPVCSpec(ptr.To("sc-0"), ptr.To("vac-0"), "pv-0", "10Gi"),
				withPVCStatus("10Gi", ptr.To("vac-0"))),
			vol: &ActualVolume{
				Desired: &DesiredVolume{
					Size:    resource.MustParse("50Gi"),
					VACName: ptr.To("vac-1"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := client.NewFakeClient(tt.prePVC)
			m := &nativeModifier{
				k8sClient: cli,
				logger:    logr.Discard(),
			}
			tt.vol.PVC = tt.prePVC
			if err := m.Modify(context.TODO(), tt.vol); (err != nil) != tt.wantErr {
				t.Errorf("Modify() error = %v, wantErr %v", err, tt.wantErr)
			}

			var pvc corev1.PersistentVolumeClaim
			g := NewGomegaWithT(t)
			g.Expect(cli.Get(context.TODO(), client.ObjectKey{Namespace: tt.vol.PVC.Namespace, Name: tt.vol.PVC.Name}, &pvc)).To(Succeed())
			g.Expect(pvc.Spec.Resources.Requests[corev1.ResourceStorage]).Should(Equal(tt.vol.Desired.Size))
			g.Expect(pvc.Spec.VolumeAttributesClassName).Should(Equal(tt.vol.Desired.VACName))
		})
	}
}
