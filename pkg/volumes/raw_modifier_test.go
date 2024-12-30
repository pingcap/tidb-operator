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

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/go-logr/logr"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/time"
	"github.com/pingcap/tidb-operator/pkg/volumes/cloud"
	"github.com/pingcap/tidb-operator/pkg/volumes/cloud/aws"
)

func withPVCStatus(size string, curVACName *string) fake.ChangeFunc[corev1.PersistentVolumeClaim, *corev1.PersistentVolumeClaim] {
	return func(pvc *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
		pvc.Status.Phase = corev1.ClaimBound
		pvc.Status.Capacity = corev1.ResourceList{}
		pvc.Status.Capacity[corev1.ResourceStorage] = resource.MustParse(size)
		pvc.Status.CurrentVolumeAttributesClassName = curVACName
		return pvc
	}
}

func withPVCSpec(scName, vacName *string, vol, size string) fake.ChangeFunc[corev1.PersistentVolumeClaim, *corev1.PersistentVolumeClaim] {
	return func(pvc *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
		pvc.Spec.StorageClassName = scName
		pvc.Spec.VolumeAttributesClassName = vacName
		pvc.Spec.VolumeName = vol
		pvc.Spec.Resources.Requests = corev1.ResourceList{}
		pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse(size)
		return pvc
	}
}

func withPVCAnnotation(key, value string) fake.ChangeFunc[corev1.PersistentVolumeClaim, *corev1.PersistentVolumeClaim] {
	return func(pvc *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
		if pvc.Annotations == nil {
			pvc.Annotations = map[string]string{}
		}
		pvc.Annotations[key] = value
		return pvc
	}
}

func withParameters(params map[string]string) fake.ChangeFunc[storagev1.StorageClass, *storagev1.StorageClass] {
	return func(sc *storagev1.StorageClass) *storagev1.StorageClass {
		sc.Parameters = params
		return sc
	}
}

func getObjectsFromActualVolume(vol *ActualVolume) []client.Object {
	var objs []client.Object
	if vol != nil {
		if vol.Desired != nil && vol.Desired.StorageClass != nil {
			objs = append(objs, vol.Desired.StorageClass)
		}
		if vol.StorageClass != nil {
			objs = append(objs, vol.StorageClass)
		}
		if vol.PVC != nil {
			objs = append(objs, vol.PVC)
		}
		if vol.PV != nil {
			objs = append(objs, vol.PV)
		}
	}
	return objs
}

func Test_rawModifier_GetActualVolume(t *testing.T) {
	tests := []struct {
		name         string
		existingObjs []client.Object
		desired      *corev1.PersistentVolumeClaim
		current      *corev1.PersistentVolumeClaim
		getState     aws.GetVolumeStateFunc
		expect       func(*WithT, *ActualVolume)
		wantErr      bool
	}{
		{
			name: "happy path: no modification",
			existingObjs: []client.Object{
				fake.FakeObj[corev1.PersistentVolume]("pv-0"),
				fake.FakeObj[storagev1.StorageClass]("sc-0"),
			},
			desired: fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0", withPVCSpec(ptr.To("sc-0"), nil, "pv-0", "10Gi")),
			current: fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0", withPVCStatus("10Gi", nil), withPVCSpec(ptr.To("sc-0"), nil, "pv-0", "10Gi")),
			getState: func(_ string) types.VolumeModificationState {
				return types.VolumeModificationStateFailed
			},
			expect: func(g *WithT, volume *ActualVolume) {
				g.Expect(volume).ShouldNot(BeNil())
				g.Expect(volume.Desired).ShouldNot(BeNil())
				g.Expect(volume.Desired.Size).Should(Equal(resource.MustParse("10Gi")))
				g.Expect(volume.Desired.StorageClassName).Should(Equal(ptr.To("sc-0")))
				g.Expect(volume.Desired.StorageClass).ShouldNot(BeNil())

				g.Expect(volume.PVC).ShouldNot(BeNil())
				g.Expect(volume.PV).ShouldNot(BeNil())
				g.Expect(volume.StorageClass).ShouldNot(BeNil())
				g.Expect(volume.Phase).Should(Equal(VolumePhaseUnknown))
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := client.NewFakeClient(tt.existingObjs...)
			m := NewRawModifier(aws.NewFakeEBSModifier(tt.getState), cli, logr.Discard())
			got, err := m.GetActualVolume(context.TODO(), tt.desired, tt.current)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetActualVolume() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			g := NewGomegaWithT(t)
			if tt.expect != nil {
				tt.expect(g, got)
			}
		})
	}
}

func Test_rawModifier_getVolumePhase(t *testing.T) {
	tests := []struct {
		name           string
		volumeModifier cloud.VolumeModifier
		clock          time.Clock
		vol            *ActualVolume
		want           VolumePhase
		wantStr        string
		shouldModify   bool
	}{
		{
			name: "no need to modify",
			vol: &ActualVolume{
				Desired: &DesiredVolume{
					Size:             resource.MustParse("10Gi"),
					StorageClassName: ptr.To("sc-0"),
				},
				PVC: fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0", withPVCSpec(ptr.To("sc-0"), nil, "pv-0", "10Gi"), withPVCStatus("10Gi", nil)),
			},
			volumeModifier: &cloud.FakeVolumeModifier{},
			want:           VolumePhaseModified,
			wantStr:        "Modified",
			shouldModify:   false,
		},
		{
			name: "change storage class",
			vol: &ActualVolume{
				Desired: &DesiredVolume{
					Size:             resource.MustParse("10Gi"),
					StorageClassName: ptr.To("sc-1"),
					StorageClass:     fake.FakeObj[storagev1.StorageClass]("sc-1", withParameters(map[string]string{"iops": "100"})),
				},
				PVC:          fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0", withPVCSpec(ptr.To("sc-0"), nil, "pv-0", "10Gi"), withPVCStatus("10Gi", nil)),
				StorageClass: fake.FakeObj[storagev1.StorageClass]("sc-0"),
				PV:           fake.FakeObj[corev1.PersistentVolume]("pv-0"),
			},
			volumeModifier: &cloud.FakeVolumeModifier{},
			want:           VolumePhasePreparing,
			wantStr:        "Preparing",
			shouldModify:   true,
		},
		{
			name: "increase size",
			vol: &ActualVolume{
				Desired: &DesiredVolume{
					Size:             resource.MustParse("100Gi"),
					StorageClassName: ptr.To("sc-0"),
				},
				PVC: fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0", withPVCSpec(ptr.To("sc-0"), nil, "pv-0", "10Gi"), withPVCStatus("10Gi", nil)),
			},
			volumeModifier: &cloud.FakeVolumeModifier{},
			want:           VolumePhasePreparing,
			wantStr:        "Preparing",
			shouldModify:   true,
		},
		{
			name: "decrease size",
			vol: &ActualVolume{
				Desired: &DesiredVolume{
					Size:             resource.MustParse("1Gi"),
					StorageClassName: ptr.To("sc-0"),
				},
				PVC: fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-1", withPVCSpec(ptr.To("sc-0"), nil, "pv-1", "20Gi"), withPVCStatus("20Gi", nil)),
			},
			volumeModifier: &cloud.FakeVolumeModifier{},
			want:           VolumePhaseCannotModify,
			wantStr:        "CannotModify",
			shouldModify:   false,
		},
		{
			name: "modifying",
			vol: &ActualVolume{
				Desired: &DesiredVolume{
					Size:             resource.MustParse("100Gi"),
					StorageClassName: ptr.To("sc-0"),
				},
				PVC: fake.FakeObj("pvc-1",
					withPVCSpec(ptr.To("sc-0"), nil, "pv-1", "10Gi"), withPVCStatus("10Gi", nil),
					withPVCAnnotation(annoKeyPVCSpecRevision, "2"),
					withPVCAnnotation(annoKeyPVCStatusRevision, "1"),
				),
			},
			volumeModifier: &cloud.FakeVolumeModifier{},
			want:           VolumePhaseModifying,
			wantStr:        "Modifying",
			shouldModify:   true,
		},
		{
			name: "wait for next time",
			vol: &ActualVolume{
				Desired: &DesiredVolume{
					Size:             resource.MustParse("100Gi"),
					StorageClassName: ptr.To("sc-0"),
				},
				PVC: fake.FakeObj("pvc-0",
					withPVCSpec(ptr.To("sc-0"), nil, "pv-0", "10Gi"), withPVCStatus("10Gi", nil),
					withPVCAnnotation(annoKeyPVCLastTransitionTimestamp, "2121-01-01T00:00:00Z"), // a future time
				),
			},
			clock:          time.RealClock{},
			volumeModifier: &cloud.FakeVolumeModifier{},
			want:           VolumePhasePending,
			wantStr:        "Pending",
			shouldModify:   false,
		},
		{
			name: "no need to wait for next time",
			vol: &ActualVolume{
				Desired: &DesiredVolume{
					Size:             resource.MustParse("100Gi"),
					StorageClassName: ptr.To("sc-0"),
				},
				PVC: fake.FakeObj("pvc-0",
					withPVCSpec(ptr.To("sc-0"), nil, "pv-0", "10Gi"), withPVCStatus("10Gi", nil),
					withPVCAnnotation(annoKeyPVCLastTransitionTimestamp, "2021-01-01T00:00:00Z"), // a past time
				),
			},
			clock:          time.RealClock{},
			volumeModifier: &cloud.FakeVolumeModifier{},
			want:           VolumePhasePreparing,
			wantStr:        "Preparing",
			shouldModify:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &rawModifier{
				k8sClient:      client.NewFakeClient(getObjectsFromActualVolume(tt.vol)...),
				logger:         logr.Logger{},
				volumeModifier: tt.volumeModifier,
				clock:          tt.clock,
			}
			if got := m.getVolumePhase(tt.vol); got != tt.want {
				t.Errorf("getVolumePhase() = %v, want %v", got, tt.want)
			}
			if got := tt.want.String(); got != tt.wantStr {
				t.Errorf("VolumePhase.String() = %v, want %v", got, tt.wantStr)
			}
			if got := m.ShouldModify(context.TODO(), tt.vol); got != tt.shouldModify {
				t.Errorf("ShouldModify() = %v, want %v", got, tt.shouldModify)
			}
		})
	}
}

func Test_rawModifier_Modify(t *testing.T) {
	tests := []struct {
		name     string
		vol      *ActualVolume
		getState aws.GetVolumeStateFunc
		wantErr  bool
	}{
		{
			name: "can not modify",
			vol: &ActualVolume{
				PVC:   fake.FakeObj[corev1.PersistentVolumeClaim]("pvc-0"),
				Phase: VolumePhaseModified,
			},
			wantErr: true,
		},
		{
			name: "preparing, wait for fs to be resized",
			vol: &ActualVolume{
				PVC: fake.FakeObj("pvc-0",
					withPVCSpec(ptr.To("sc-0"), nil, "pv-0", "10Gi"),
					withPVCStatus("10Gi", nil),
				),
				Phase: VolumePhasePreparing,
				Desired: &DesiredVolume{
					Size:             resource.MustParse("20Gi"),
					StorageClassName: ptr.To("sc-0"),
				},
			},
			wantErr: true,
		},
		{
			name: "modifying, wait for fs to be resized",
			vol: &ActualVolume{
				PVC: fake.FakeObj("pvc-0",
					withPVCSpec(ptr.To("sc-0"), nil, "pv-0", "10Gi"),
					withPVCStatus("10Gi", nil),
				),
				Phase: VolumePhaseModifying,
				Desired: &DesiredVolume{
					Size:             resource.MustParse("20Gi"),
					StorageClassName: ptr.To("sc-0"),
				},
			},
			wantErr: true,
		},
		{
			name: "modifying, synced with desired",
			vol: &ActualVolume{
				PVC: fake.FakeObj("pvc-0",
					withPVCSpec(ptr.To("sc-0"), nil, "pv-0", "20Gi"),
					withPVCStatus("20Gi", nil),
				),
				Phase: VolumePhaseModifying,
				Desired: &DesiredVolume{
					Size:             resource.MustParse("20Gi"),
					StorageClassName: ptr.To("sc-0"),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &rawModifier{
				k8sClient:      client.NewFakeClient(getObjectsFromActualVolume(tt.vol)...),
				logger:         logr.Discard(),
				volumeModifier: aws.NewFakeEBSModifier(tt.getState),
				clock:          &time.RealClock{},
			}
			if err := m.Modify(context.TODO(), tt.vol); (err != nil) != tt.wantErr {
				t.Errorf("Modify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
