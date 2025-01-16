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

package tasks

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/pdapi/v1"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TestTaskFinalizerDel(t *testing.T) {
	cases := []struct {
		desc                   string
		state                  *ReconcileContext
		subresources           []client.Object
		needDelMember          bool
		unexpectedDelMemberErr bool
		unexpectedErr          bool

		expectedStatus task.Status
		expectedObj    *v1alpha1.PD
	}{
		{
			desc: "available, member id is set, no sub resources, no finalizer",
			state: &ReconcileContext{
				State: &state{
					pd: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
						return obj
					}),
				},
				IsAvailable: true,
				MemberID:    "aaa",
			},
			needDelMember: true,

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				return obj
			}),
		},
		{
			desc: "available, member id is set, failed to delete member",
			state: &ReconcileContext{
				State: &state{
					pd: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
						return obj
					}),
				},
				IsAvailable: true,
				MemberID:    "aaa",
			},
			needDelMember:          true,
			unexpectedDelMemberErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "available, member id is set, has sub resources, has finalizer",
			state: &ReconcileContext{
				State: &state{
					pd: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
				},
				IsAvailable: true,
				MemberID:    "aaa",
			},
			subresources: []client.Object{
				fake.FakeObj("aaa", func(obj *corev1.Pod) *corev1.Pod {
					obj.Labels = map[string]string{
						v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
						v1alpha1.LabelKeyInstance:  "aaa",
						v1alpha1.LabelKeyCluster:   "",
						v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
					}
					return obj
				}),
				fake.FakeObj("aaa", func(obj *corev1.ConfigMap) *corev1.ConfigMap {
					obj.Labels = map[string]string{
						v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
						v1alpha1.LabelKeyInstance:  "aaa",
						v1alpha1.LabelKeyCluster:   "",
						v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
					}
					return obj
				}),
				fake.FakeObj("aaa", func(obj *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
					obj.Labels = map[string]string{
						v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
						v1alpha1.LabelKeyInstance:  "aaa",
						v1alpha1.LabelKeyCluster:   "",
						v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
					}
					return obj
				}),
			},
			needDelMember: true,

			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
				return obj
			}),
		},
		{
			desc: "available, member id is set, has sub resources(pod), failed to del sub resource",
			state: &ReconcileContext{
				State: &state{
					pd: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
				},
				IsAvailable: true,
				MemberID:    "aaa",
			},
			subresources: []client.Object{
				fake.FakeObj("aaa", func(obj *corev1.Pod) *corev1.Pod {
					obj.Labels = map[string]string{
						v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
						v1alpha1.LabelKeyInstance:  "aaa",
						v1alpha1.LabelKeyCluster:   "",
						v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
					}
					return obj
				}),
			},
			needDelMember: true,
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "available, member id is set, has sub resources(cm), failed to del sub resource",
			state: &ReconcileContext{
				State: &state{
					pd: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
				},
				IsAvailable: true,
				MemberID:    "aaa",
			},
			subresources: []client.Object{
				fake.FakeObj("aaa", func(obj *corev1.ConfigMap) *corev1.ConfigMap {
					obj.Labels = map[string]string{
						v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
						v1alpha1.LabelKeyInstance:  "aaa",
						v1alpha1.LabelKeyCluster:   "",
						v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
					}
					return obj
				}),
			},
			needDelMember: true,
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "available, member id is set, has sub resources(pvc), failed to del sub resource",
			state: &ReconcileContext{
				State: &state{
					pd: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
				},
				IsAvailable: true,
				MemberID:    "aaa",
			},
			subresources: []client.Object{
				fake.FakeObj("aaa", func(obj *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
					obj.Labels = map[string]string{
						v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
						v1alpha1.LabelKeyInstance:  "aaa",
						v1alpha1.LabelKeyCluster:   "",
						v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
					}
					return obj
				}),
			},
			needDelMember: true,
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "available, member id is set, no sub resources, has finalizer",
			state: &ReconcileContext{
				State: &state{
					pd: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
				},
				IsAvailable: true,
				MemberID:    "aaa",
			},
			needDelMember: true,

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Finalizers = []string{}
				return obj
			}),
		},
		{
			desc: "available, member id is set, no sub resources, has finalizer, failed to del finalizer",
			state: &ReconcileContext{
				State: &state{
					pd: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
				},
				IsAvailable: true,
				MemberID:    "aaa",
			},
			needDelMember: true,
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "available, member id is not set, no sub resources, no finalizer",
			state: &ReconcileContext{
				State: &state{
					pd: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
						return obj
					}),
				},
				IsAvailable: true,
			},
			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				return obj
			}),
		},
		{
			desc: "available, member id is not set, has sub resources, has finalizer",
			state: &ReconcileContext{
				State: &state{
					pd: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
				},
				IsAvailable: true,
			},
			subresources: []client.Object{
				fake.FakeObj("aaa", func(obj *corev1.ConfigMap) *corev1.ConfigMap {
					obj.Labels = map[string]string{
						v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
						v1alpha1.LabelKeyInstance:  "aaa",
						v1alpha1.LabelKeyCluster:   "",
						v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
					}
					return obj
				}),
			},

			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
				return obj
			}),
		},
		{
			desc: "available, member id is not set, has sub resources, failed to del sub resource",
			state: &ReconcileContext{
				State: &state{
					pd: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
				},
				IsAvailable: true,
			},
			subresources: []client.Object{
				fake.FakeObj("aaa", func(obj *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
					obj.Labels = map[string]string{
						v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
						v1alpha1.LabelKeyInstance:  "aaa",
						v1alpha1.LabelKeyCluster:   "",
						v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
					}
					return obj
				}),
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "available, member id is not set, no sub resources, has finalizer",
			state: &ReconcileContext{
				State: &state{
					pd: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
				},
				IsAvailable: true,
			},

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Finalizers = []string{}
				return obj
			}),
		},
		{
			desc: "available, member id is not set, no sub resources, has finalizer, failed to del finalizer",
			state: &ReconcileContext{
				State: &state{
					pd: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
				},
				IsAvailable: true,
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "unavailable",
			state: &ReconcileContext{
				State: &state{
					pd: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
				},
				IsAvailable: false,
			},

			expectedStatus: task.SFail,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
				return obj
			}),
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			objs := []client.Object{
				c.state.PD(),
			}

			objs = append(objs, c.subresources...)

			fc := client.NewFakeClient(objs...)
			if c.unexpectedErr {
				// cannot remove finalizer
				fc.WithError("update", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
				// cannot delete sub resources
				fc.WithError("delete", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var acts []action
			if c.needDelMember {
				var retErr error
				if c.unexpectedDelMemberErr {
					retErr = fmt.Errorf("fake err")
				}
				acts = append(acts, deleteMember(ctx, c.state.MemberID, retErr))
			}

			pdc := NewFakePDClient(tt, acts...)
			c.state.PDClient = pdc

			res, done := task.RunTask(ctx, TaskFinalizerDel(c.state, fc))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)

			// no need to check update result
			if c.unexpectedErr || c.unexpectedDelMemberErr {
				return
			}

			pd := &v1alpha1.PD{}
			require.NoError(tt, fc.Get(ctx, client.ObjectKey{Name: "aaa"}, pd), c.desc)
			assert.Equal(tt, c.expectedObj, pd, c.desc)
		})
	}
}

func deleteMember(ctx context.Context, name string, err error) action {
	return func(ctrl *gomock.Controller, pdc *pdm.MockPDClient) {
		underlay := pdapi.NewMockPDClient(ctrl)
		pdc.EXPECT().Underlay().Return(underlay)
		underlay.EXPECT().DeleteMember(ctx, name).Return(err)
	}
}
