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

package common

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/features"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/pkg/volumes"
)

type mockTaskPVCState struct {
	cluster *v1alpha1.Cluster
	obj     *v1alpha1.PD
	fg      features.Gates
}

func (m *mockTaskPVCState) Cluster() *v1alpha1.Cluster {
	return m.cluster
}

func (m *mockTaskPVCState) Object() *v1alpha1.PD {
	return m.obj
}

func (m *mockTaskPVCState) FeatureGates() features.Gates {
	return m.fg
}

type mockPVCNewer struct {
	pvcs []*corev1.PersistentVolumeClaim
}

func (m *mockPVCNewer) NewPVCs(c *v1alpha1.Cluster, obj *v1alpha1.PD, fg features.Gates) []*corev1.PersistentVolumeClaim {
	return m.pvcs
}

func TestTaskPVC(t *testing.T) {
	cases := []struct {
		desc           string
		enableVAC      bool
		pvcs           []*corev1.PersistentVolumeClaim
		expectedStatus task.Status
	}{
		{
			desc:           "VAC disabled - no pvcs",
			enableVAC:      false,
			pvcs:           []*corev1.PersistentVolumeClaim{},
			expectedStatus: task.SComplete,
		},
		{
			desc:      "VAC disabled - with pvcs",
			enableVAC: false,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pvc",
						Namespace: "default",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: apiresource.MustParse("10Gi"),
							},
						},
					},
				},
			},
			expectedStatus: task.SComplete,
		},
		{
			desc:           "VAC enabled - no pvcs",
			enableVAC:      true,
			pvcs:           []*corev1.PersistentVolumeClaim{},
			expectedStatus: task.SComplete,
		},
		{
			desc:      "VAC enabled - with pvcs",
			enableVAC: true,
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pvc",
						Namespace: "default",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: apiresource.MustParse("10Gi"),
							},
						},
					},
				},
			},
			expectedStatus: task.SComplete,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctx := logr.NewContext(context.Background(), logr.Discard())

			cluster := fake.FakeObj[v1alpha1.Cluster]("test-cluster")
			obj := fake.FakeObj[v1alpha1.PD]("test-pd")

			fg := features.NewFromFeatures(nil)
			if c.enableVAC {
				fg = features.NewFromFeatures([]meta.Feature{meta.VolumeAttributesClass})
			}
			state := &mockTaskPVCState{
				cluster: cluster,
				obj:     obj,
				fg:      fg,
			}

			fc := client.NewFakeClient()
			ctrl := gomock.NewController(tt)
			defer ctrl.Finish()

			vmFactory := volumes.NewMockModifierFactory(ctrl)
			vm := volumes.NewMockModifier(ctrl)

			if !c.enableVAC {
				vmFactory.EXPECT().New().Return(vm)
			}

			newer := &mockPVCNewer{pvcs: c.pvcs}

			taskFunc := TaskPVC[scope.PD](state, fc, vmFactory, newer)

			result, done := task.RunTask(ctx, taskFunc)

			assert.Equal(tt, c.expectedStatus.String(), result.Status().String(), c.desc)
			assert.False(tt, done, c.desc)
			assert.Contains(tt, result.Message(), "pvcs are synced", c.desc)
		})
	}
}
