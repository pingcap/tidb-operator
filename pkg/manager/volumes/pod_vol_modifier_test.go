// Copyright 2022 PingCAP, Inc.
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
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/pointer"

	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/volumes/delegation"
)

func newTestPVCForModify(sc *string, specSize, statusSize string, anno map[string]string) *corev1.PersistentVolumeClaim {
	a := resource.MustParse(specSize)
	b := resource.MustParse(statusSize)

	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-pvc",
			Namespace:   "test",
			Annotations: anno,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: a,
				},
			},
			VolumeName:       "test-pv",
			StorageClassName: sc,
		},
		Status: corev1.PersistentVolumeClaimStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: b,
			},
		},
	}
}

func newTestPVForModify() *corev1.PersistentVolume {
	return &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv",
		},
	}
}

func newTestSCForModify(name, provisioner string) *storagev1.StorageClass {
	return &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Provisioner:          provisioner,
		AllowVolumeExpansion: pointer.BoolPtr(true),
	}
}

func TestModify(t *testing.T) {
	oldSize := "10Gi"
	newSize := "20Gi"
	oldSc := "old"
	// newSc := "new"

	provisioner := "test"

	cases := []struct {
		desc string

		pvc  *corev1.PersistentVolumeClaim
		pv   *corev1.PersistentVolume
		sc   *storagev1.StorageClass
		size string

		isModifyVolumeFinished bool

		expectedPVC    *corev1.PersistentVolumeClaim
		expectedHasErr bool
	}{
		{
			desc: "volume is not changed",
			pvc:  newTestPVCForModify(&oldSc, oldSize, oldSize, nil),
			pv:   newTestPVForModify(),
			sc:   newTestSCForModify(oldSc, provisioner),
			size: oldSize,

			expectedPVC: newTestPVCForModify(&oldSc, oldSize, oldSize, nil),
		},
		{
			desc: "volume size is changed, and revision has not been upgraded",

			pvc:  newTestPVCForModify(&oldSc, oldSize, oldSize, nil),
			pv:   newTestPVForModify(),
			sc:   newTestSCForModify(oldSc, provisioner),
			size: newSize,

			isModifyVolumeFinished: false,

			expectedPVC: newTestPVCForModify(&oldSc, oldSize, oldSize, map[string]string{
				annoKeyPVCSpecRevision:     "1",
				annoKeyPVCSpecStorageClass: oldSc,
				annoKeyPVCSpecStorageSize:  newSize,
			}),
			expectedHasErr: true,
		},
		{
			desc: "volume size is changed, and delegate modification is finished",

			pvc: newTestPVCForModify(&oldSc, oldSize, oldSize, map[string]string{
				annoKeyPVCSpecRevision:     "1",
				annoKeyPVCSpecStorageClass: oldSc,
				annoKeyPVCSpecStorageSize:  newSize,
			}),
			pv:   newTestPVForModify(),
			sc:   newTestSCForModify(oldSc, provisioner),
			size: newSize,

			isModifyVolumeFinished: true,

			expectedPVC: newTestPVCForModify(&oldSc, newSize, oldSize, map[string]string{
				annoKeyPVCSpecRevision:     "1",
				annoKeyPVCSpecStorageClass: oldSc,
				annoKeyPVCSpecStorageSize:  newSize,
			}),
			expectedHasErr: true,
		},
		{
			desc: "volume size is changed, and fs resize is finished",

			pvc: newTestPVCForModify(&oldSc, newSize, newSize, map[string]string{
				annoKeyPVCSpecRevision:     "1",
				annoKeyPVCSpecStorageClass: oldSc,
				annoKeyPVCSpecStorageSize:  newSize,
			}),
			pv:   newTestPVForModify(),
			sc:   newTestSCForModify(oldSc, provisioner),
			size: newSize,

			isModifyVolumeFinished: true,

			expectedPVC: newTestPVCForModify(&oldSc, newSize, newSize, map[string]string{
				annoKeyPVCSpecRevision:       "1",
				annoKeyPVCSpecStorageClass:   oldSc,
				annoKeyPVCSpecStorageSize:    newSize,
				annoKeyPVCStatusRevision:     "1",
				annoKeyPVCStatusStorageClass: oldSc,
				annoKeyPVCStatusStorageSize:  newSize,
			}),
		},
	}

	g := NewGomegaWithT(t)
	for _, c := range cases {
		kc := fake.NewSimpleClientset(
			c.pvc,
			c.pv,
			c.sc,
		)
		stopCh := make(chan struct{})

		f := informers.NewSharedInformerFactory(kc, 0)
		pvcLister := f.Core().V1().PersistentVolumeClaims().Lister()
		pvLister := f.Core().V1().PersistentVolumes().Lister()
		scLister := f.Storage().V1().StorageClasses().Lister()

		f.Start(stopCh)
		f.WaitForCacheSync(stopCh)

		m := delegation.NewMockVolumeModifier(provisioner, time.Hour)

		m.ModifyVolumeFunc = func(_ context.Context, pvc *corev1.PersistentVolumeClaim, pv *corev1.PersistentVolume, sc *storagev1.StorageClass) (bool, error) {
			return !c.isModifyVolumeFinished, nil
		}

		pvm := &podVolModifier{
			deps: &controller.Dependencies{
				KubeClientset:      kc,
				PVCLister:          pvcLister,
				PVLister:           pvLister,
				StorageClassLister: scLister,
			},
			modifiers: map[string]delegation.VolumeModifier{
				m.Name(): m,
			},
		}

		actual := ActualVolume{
			Desired: &DesiredVolume{
				Name:         "test",
				Size:         resource.MustParse(c.size),
				StorageClass: c.sc,
			},
			PVC:          c.pvc,
			PV:           c.pv,
			StorageClass: c.sc,
		}

		phase := pvm.getVolumePhase(&actual)
		actual.Phase = phase

		err := pvm.Modify([]ActualVolume{actual})
		if c.expectedHasErr {
			g.Expect(err).Should(HaveOccurred(), c.desc)
		} else {
			g.Expect(err).Should(Succeed(), c.desc)
		}

		resultPVC, err := kc.CoreV1().PersistentVolumeClaims(c.pvc.Namespace).Get(context.TODO(), c.pvc.Name, metav1.GetOptions{})
		g.Expect(err).Should(Succeed(), c.desc)
		delete(resultPVC.Annotations, annoKeyPVCLastTransitionTimestamp)
		g.Expect(resultPVC).Should(Equal(c.expectedPVC), c.desc)
	}
}
