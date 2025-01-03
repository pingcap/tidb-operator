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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TestTaskStatus(t *testing.T) {
	cases := []struct {
		desc          string
		state         *ReconcileContext
		unexpectedErr bool

		expectedStatus task.Status
		expectedObj    *v1alpha1.PDGroup
	}{
		{
			desc: "no pds",
			state: &ReconcileContext{
				State: &state{
					pdg: fake.FakeObj("aaa", fake.SetGeneration[v1alpha1.PDGroup](3)),
				},
				UpdateRevision:  newRevision,
				CurrentRevision: oldRevision,
				CollisionCount:  3,
				IsBootstrapped:  true,
			},

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", fake.SetGeneration[v1alpha1.PDGroup](3), func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = append(obj.Status.Conditions, metav1.Condition{
					Type:               v1alpha1.CondSuspended,
					Status:             metav1.ConditionFalse,
					ObservedGeneration: 3,
					Reason:             v1alpha1.ReasonUnsuspended,
					Message:            "group is not suspended",
				})
				obj.Status.ObservedGeneration = 3
				obj.Status.Replicas = 0
				obj.Status.ReadyReplicas = 0
				obj.Status.UpdatedReplicas = 0
				obj.Status.CurrentReplicas = 0
				obj.Status.UpdateRevision = newRevision
				obj.Status.CurrentRevision = newRevision
				obj.Status.CollisionCount = ptr.To[int32](3)
				return obj
			}),
		},
		{
			desc: "no pds and not bootstrapped",
			state: &ReconcileContext{
				State: &state{
					pdg: fake.FakeObj("aaa", fake.SetGeneration[v1alpha1.PDGroup](3)),
				},
				UpdateRevision:  newRevision,
				CurrentRevision: oldRevision,
			},

			expectedStatus: task.SWait,
			expectedObj: fake.FakeObj("aaa", fake.SetGeneration[v1alpha1.PDGroup](3), func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = append(obj.Status.Conditions, metav1.Condition{
					Type:               v1alpha1.CondSuspended,
					Status:             metav1.ConditionFalse,
					ObservedGeneration: 3,
					Reason:             v1alpha1.ReasonUnsuspended,
					Message:            "group is not suspended",
				})
				obj.Status.ObservedGeneration = 3
				obj.Status.Replicas = 0
				obj.Status.ReadyReplicas = 0
				obj.Status.UpdatedReplicas = 0
				obj.Status.CurrentReplicas = 0
				obj.Status.UpdateRevision = newRevision
				obj.Status.CurrentRevision = newRevision
				obj.Status.CollisionCount = nil
				return obj
			}),
		},
		{
			desc: "all pds are outdated and healthy",
			state: &ReconcileContext{
				State: &state{
					pdg: fake.FakeObj("aaa", fake.SetGeneration[v1alpha1.PDGroup](3)),
					pds: []*v1alpha1.PD{
						fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
							obj.Status.CurrentRevision = oldRevision
							obj.Status.Conditions = append(obj.Status.Conditions, metav1.Condition{
								Type:   v1alpha1.PDCondHealth,
								Status: metav1.ConditionTrue,
							})
							return obj
						}),
					},
				},
				UpdateRevision:  newRevision,
				CurrentRevision: oldRevision,
				IsBootstrapped:  true,
			},

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", fake.SetGeneration[v1alpha1.PDGroup](3), func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = append(obj.Status.Conditions, metav1.Condition{
					Type:               v1alpha1.CondSuspended,
					Status:             metav1.ConditionFalse,
					ObservedGeneration: 3,
					Reason:             v1alpha1.ReasonUnsuspended,
					Message:            "group is not suspended",
				})
				obj.Status.ObservedGeneration = 3
				obj.Status.Replicas = 1
				obj.Status.ReadyReplicas = 1
				obj.Status.UpdatedReplicas = 0
				obj.Status.CurrentReplicas = 1
				obj.Status.UpdateRevision = newRevision
				obj.Status.CurrentRevision = oldRevision
				obj.Status.CollisionCount = nil
				return obj
			}),
		},
		{
			desc: "all pds are updated and healthy",
			state: &ReconcileContext{
				State: &state{
					pdg: fake.FakeObj("aaa", fake.SetGeneration[v1alpha1.PDGroup](3)),
					pds: []*v1alpha1.PD{
						fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
							obj.Status.CurrentRevision = newRevision
							obj.Status.Conditions = append(obj.Status.Conditions, metav1.Condition{
								Type:   v1alpha1.PDCondHealth,
								Status: metav1.ConditionTrue,
							})
							return obj
						}),
					},
				},
				UpdateRevision:  newRevision,
				CurrentRevision: oldRevision,
				IsBootstrapped:  true,
			},

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", fake.SetGeneration[v1alpha1.PDGroup](3), func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = append(obj.Status.Conditions, metav1.Condition{
					Type:               v1alpha1.CondSuspended,
					Status:             metav1.ConditionFalse,
					ObservedGeneration: 3,
					Reason:             v1alpha1.ReasonUnsuspended,
					Message:            "group is not suspended",
				})
				obj.Status.ObservedGeneration = 3
				obj.Status.Replicas = 1
				obj.Status.ReadyReplicas = 1
				obj.Status.UpdatedReplicas = 1
				obj.Status.CurrentReplicas = 0
				obj.Status.UpdateRevision = newRevision
				obj.Status.CurrentRevision = newRevision
				obj.Status.CollisionCount = nil
				return obj
			}),
		},
		{
			desc: "all pds are updated but not healthy",
			state: &ReconcileContext{
				State: &state{
					pdg: fake.FakeObj("aaa", fake.SetGeneration[v1alpha1.PDGroup](3)),
					pds: []*v1alpha1.PD{
						fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
							obj.Status.CurrentRevision = newRevision
							return obj
						}),
					},
				},
				UpdateRevision:  newRevision,
				CurrentRevision: oldRevision,
				IsBootstrapped:  true,
			},

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", fake.SetGeneration[v1alpha1.PDGroup](3), func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = append(obj.Status.Conditions, metav1.Condition{
					Type:               v1alpha1.CondSuspended,
					Status:             metav1.ConditionFalse,
					ObservedGeneration: 3,
					Reason:             v1alpha1.ReasonUnsuspended,
					Message:            "group is not suspended",
				})
				obj.Status.ObservedGeneration = 3
				obj.Status.Replicas = 1
				obj.Status.ReadyReplicas = 0
				obj.Status.UpdatedReplicas = 1
				obj.Status.CurrentReplicas = 0
				obj.Status.UpdateRevision = newRevision
				obj.Status.CurrentRevision = newRevision
				obj.Status.CollisionCount = nil
				return obj
			}),
		},
		{
			desc: "status changed but cannot call api",
			state: &ReconcileContext{
				State: &state{
					pdg: fake.FakeObj("aaa", fake.SetGeneration[v1alpha1.PDGroup](3)),
					pds: []*v1alpha1.PD{
						fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
							obj.Status.CurrentRevision = newRevision
							return obj
						}),
					},
				},
				UpdateRevision:  newRevision,
				CurrentRevision: oldRevision,
				IsBootstrapped:  true,
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "status is not changed and cannot call api",
			state: &ReconcileContext{
				State: &state{
					pdg: fake.FakeObj("aaa", fake.SetGeneration[v1alpha1.PDGroup](3), func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
						obj.Status.Conditions = append(obj.Status.Conditions, metav1.Condition{
							Type:               v1alpha1.CondSuspended,
							Status:             metav1.ConditionFalse,
							ObservedGeneration: 3,
							Reason:             v1alpha1.ReasonUnsuspended,
							Message:            "group is not suspended",
						})
						obj.Status.ObservedGeneration = 3
						obj.Status.Replicas = 1
						obj.Status.ReadyReplicas = 0
						obj.Status.UpdatedReplicas = 1
						obj.Status.CurrentReplicas = 0
						obj.Status.UpdateRevision = newRevision
						obj.Status.CurrentRevision = newRevision
						obj.Status.CollisionCount = nil
						return obj
					}),
					pds: []*v1alpha1.PD{
						fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
							obj.Status.CurrentRevision = newRevision
							return obj
						}),
					},
				},
				UpdateRevision:  newRevision,
				CurrentRevision: oldRevision,
				IsBootstrapped:  true,
			},
			unexpectedErr: true,

			expectedStatus: task.SComplete,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fc := client.NewFakeClient(c.state.PDGroup())
			if c.unexpectedErr {
				fc.WithError("*", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			ctx := context.Background()
			res, done := task.RunTask(ctx, TaskStatus(c.state, fc))
			assert.Equal(tt, c.expectedStatus, res.Status(), c.desc)
			assert.False(tt, done, c.desc)

			// no need to check update result
			if c.unexpectedErr {
				return
			}

			pdg := &v1alpha1.PDGroup{}
			require.NoError(tt, fc.Get(ctx, client.ObjectKey{Name: "aaa"}, pdg), c.desc)
			for i := range pdg.Status.Conditions {
				cond := &pdg.Status.Conditions[i]
				cond.LastTransitionTime = metav1.Time{}
			}
			assert.Equal(tt, c.expectedObj, pdg, c.desc)
		})
	}
}
