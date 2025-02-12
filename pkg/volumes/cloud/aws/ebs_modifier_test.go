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

package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/go-logr/logr"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/pkg/volumes/cloud"
)

func newTestStorageClass(provisioner, typ, iops, throughput string) *storagev1.StorageClass {
	return &storagev1.StorageClass{
		Provisioner: provisioner,
		Parameters: map[string]string{
			paramKeyIOPS:       iops,
			paramKeyType:       typ,
			paramKeyThroughput: throughput,
		},
	}
}

func TestModifyVolume(t *testing.T) {
	initialPVC := cloud.NewTestPVC("10Gi")
	initialPV := cloud.NewTestPV("aaa")
	initialSC := newTestStorageClass("", "gp3", "3000", "125")

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

			getState: func(_ string) types.VolumeModificationState {
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

			getState: func(_ string) types.VolumeModificationState {
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

			getState: func(_ string) types.VolumeModificationState {
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

			getState: func(_ string) types.VolumeModificationState {
				return types.VolumeModificationStateModifying
			},

			wait:   true,
			hasErr: false,
		},
		{
			desc: "volume has been modified, but size is changed",
			pvc:  cloud.NewTestPVC("20Gi"),
			pv:   initialPV,
			sc:   initialSC,

			getState: func(_ string) types.VolumeModificationState {
				return types.VolumeModificationStateCompleted
			},

			wait:   true,
			hasErr: false,
		},
		{
			desc: "volume has been modified, but sc is changed",
			pvc:  initialPVC,
			pv:   initialPV,
			sc:   newTestStorageClass("", "io2", "3000", "300"),

			getState: func(_ string) types.VolumeModificationState {
				return types.VolumeModificationStateCompleted
			},

			wait:   true,
			hasErr: false,
		},
		{
			desc: "volume is modifying, but size/sc is changed",
			pvc:  cloud.NewTestPVC("20Gi"),
			pv:   initialPV,
			sc:   newTestStorageClass("", "gp2", "3000", "300"),

			getState: func(_ string) types.VolumeModificationState {
				return types.VolumeModificationStateModifying
			},

			wait:   false,
			hasErr: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.desc, func(t *testing.T) {
			g := NewGomegaWithT(t)
			m := NewFakeEBSModifier(tt.getState)

			wait1, err := m.Modify(context.TODO(), initialPVC, initialPV, initialSC)
			g.Expect(err).Should(Succeed(), tt.desc)
			g.Expect(wait1).Should(BeTrue(), tt.desc)

			wait2, err := m.Modify(context.TODO(), tt.pvc, tt.pv, tt.sc)
			if tt.hasErr {
				g.Expect(err).Should(HaveOccurred(), tt.desc)
			} else {
				g.Expect(err).Should(Succeed(), tt.desc)
			}
			g.Expect(wait2).Should(Equal(tt.wait), tt.desc)
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		ssc     *storagev1.StorageClass
		dsc     *storagev1.StorageClass
		wantErr bool
	}{
		{
			name:    "same provisioner, but not ebs",
			ssc:     newTestStorageClass("foo", "gp3", "1000", "100"),
			dsc:     newTestStorageClass("foo", "gp3", "2000", "200"),
			wantErr: true,
		},
		{
			name:    "different provisioner",
			ssc:     newTestStorageClass("foo", "gp3", "1000", "100"),
			dsc:     newTestStorageClass("bar", "gp3", "2000", "200"),
			wantErr: true,
		},
		{
			name:    "happy path",
			ssc:     newTestStorageClass("ebs.csi.aws.com", "gp3", "1000", "100"),
			dsc:     newTestStorageClass("ebs.csi.aws.com", "gp3", "2000", "200"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EBSModifier{logger: logr.Logger{}}
			if err := m.Validate(nil, nil, tt.ssc, tt.dsc); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_validateVolume(t *testing.T) {
	tests := []struct {
		name    string
		desired *Volume
		wantErr bool
	}{
		{
			name:    "empty volume type",
			desired: &Volume{},
		},
		{
			name:    "nil volume",
			desired: &Volume{},
		},
		{
			name: "valid volume",
			desired: &Volume{
				Size:       ptr.To(int32(10)),
				IOPS:       ptr.To(int32(3000)),
				Throughput: ptr.To(int32(125)),
				Type:       types.VolumeTypeGp3,
			},
			wantErr: false,
		},
		{
			name: "invalid size",
			desired: &Volume{
				Size: ptr.To(int32(100000)),
				Type: types.VolumeTypeGp3,
			},
			wantErr: true,
		},
		{
			name: "invalid IOPS",
			desired: &Volume{
				IOPS: ptr.To(int32(20000)),
				Type: types.VolumeTypeGp3,
			},
			wantErr: true,
		},
		{
			name: "invalid throughput",
			desired: &Volume{
				Throughput: ptr.To(int32(2000)),
				Type:       types.VolumeTypeGp3,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateVolume(tt.desired); (err != nil) != tt.wantErr {
				t.Errorf("validateVolume() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
