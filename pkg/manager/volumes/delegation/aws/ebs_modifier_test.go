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

package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func newTestPVC(size string) *corev1.PersistentVolumeClaim {
	q := resource.MustParse(size)

	return &corev1.PersistentVolumeClaim{
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: q,
				},
			},
		},
	}
}

func newTestPV(volId string) *corev1.PersistentVolume {
	return &corev1.PersistentVolume{
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: &corev1.CSIPersistentVolumeSource{
					VolumeHandle: volId,
				},
			},
		},
	}
}

func newTestStorageClass(typ string, iops string, throughput string) *storagev1.StorageClass {
	return &storagev1.StorageClass{
		Parameters: map[string]string{
			paramKeyIOPS:       iops,
			paramKeyType:       typ,
			paramKeyThroughput: throughput,
		},
	}
}

func TestModifyVolume(t *testing.T) {
	initialPVC := newTestPVC("10Gi")
	initialPV := newTestPV("aaa")
	initialSC := newTestStorageClass("gp3", "1000", "100")

	cases := []struct {
		desc string

		pvc *corev1.PersistentVolumeClaim
		pv  *corev1.PersistentVolume
		sc  *storagev1.StorageClass

		getState GetVolumeStateFunc

		wait   bool
		hasErr bool
	}{
		{
			desc: "volume modification is failed, modify again",
			pvc:  initialPVC,
			pv:   initialPV,
			sc:   initialSC,

			getState: func(id string) types.VolumeModificationState {
				return types.VolumeModificationStateFailed
			},

			wait:   true,
			hasErr: false,
		},
		{
			desc: "volume modification is optimizing, no need to wait to avoid waiting too long time",
			pvc:  initialPVC,
			pv:   initialPV,
			sc:   initialSC,

			getState: func(id string) types.VolumeModificationState {
				return types.VolumeModificationStateOptimizing
			},

			wait:   false,
			hasErr: false,
		},
		{
			desc: "volume modification is completed, no need to wait",
			pvc:  initialPVC,
			pv:   initialPV,
			sc:   initialSC,

			getState: func(id string) types.VolumeModificationState {
				return types.VolumeModificationStateCompleted
			},

			wait:   false,
			hasErr: false,
		},
		{
			desc: "volume modification is modifying, wait",
			pvc:  initialPVC,
			pv:   initialPV,
			sc:   initialSC,

			getState: func(id string) types.VolumeModificationState {
				return types.VolumeModificationStateModifying
			},

			wait:   true,
			hasErr: false,
		},
		{
			desc: "volume has been modified, but size is changed",
			pvc:  newTestPVC("20Gi"),
			pv:   initialPV,
			sc:   initialSC,

			getState: func(id string) types.VolumeModificationState {
				return types.VolumeModificationStateCompleted
			},

			wait:   true,
			hasErr: false,
		},
		{
			desc: "volume has been modified, but sc is changed",
			pvc:  initialPVC,
			pv:   initialPV,
			sc:   newTestStorageClass("gp2", "3000", "300"),

			getState: func(id string) types.VolumeModificationState {
				return types.VolumeModificationStateCompleted
			},

			wait:   true,
			hasErr: false,
		},
		{
			desc: "volume is modifying, but siz/sc is changed",
			pvc:  newTestPVC("20Gi"),
			pv:   initialPV,
			sc:   newTestStorageClass("gp2", "3000", "300"),

			getState: func(id string) types.VolumeModificationState {
				return types.VolumeModificationStateModifying
			},

			wait:   false,
			hasErr: true,
		},
		{
			desc: "volume has not been modified, try to modify",
			pvc:  newTestPVC("10Gi"),
			pv:   newTestPV("bbb"),
			sc:   newTestStorageClass("gp3", "1000", "100"),

			getState: nil,

			wait:   true,
			hasErr: false,
		},
	}

	g := NewGomegaWithT(t)
	for _, c := range cases {
		m := &EBSModifier{
			c: NewFakeEC2VolumeAPI(c.getState),
		}

		wait1, err := m.ModifyVolume(context.TODO(), initialPVC, initialPV, initialSC)
		g.Expect(err).Should(Succeed(), c.desc)
		g.Expect(wait1).Should(BeTrue(), c.desc)

		wait2, err := m.ModifyVolume(context.TODO(), c.pvc, c.pv, c.sc)
		if c.hasErr {
			g.Expect(err).Should(HaveOccurred(), c.desc)
		} else {
			g.Expect(err).Should(Succeed(), c.desc)
		}
		g.Expect(wait2).Should(Equal(c.wait), c.desc)
	}
}
