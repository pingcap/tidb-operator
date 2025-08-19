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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	stateutil "github.com/pingcap/tidb-operator/pkg/state"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/pkg/utils/tracker"
)

const (
	oldRevision = "old"
	newRevision = "new"
)

func TestTaskUpdater(t *testing.T) {
	cases := []struct {
		desc          string
		state         *ReconcileContext
		unexpectedErr bool

		expectedStatus task.Status
		expectedPDNum  int
	}{
		{
			desc: "no pds with 1 replicas",
			state: &ReconcileContext{
				State: &state{
					pdg:     fake.FakeObj[v1alpha1.PDGroup]("aaa"),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				},
			},

			expectedStatus: task.SWait,
			expectedPDNum:  1,
		},
		{
			desc: "1 updated pd with 1 replicas",
			state: &ReconcileContext{
				State: &state{
					pdg:     fake.FakeObj[v1alpha1.PDGroup]("aaa"),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					pds: []*v1alpha1.PD{
						fakeAvailablePD("aaa-xxx", fake.FakeObj[v1alpha1.PDGroup]("aaa"), newRevision),
					},
					updateRevision: newRevision,
				},
			},

			expectedStatus: task.SComplete,
			expectedPDNum:  1,
		},
		{
			desc: "no pds with 2 replicas",
			state: &ReconcileContext{
				State: &state{
					pdg: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				},
			},

			expectedStatus: task.SWait,
			expectedPDNum:  2,
		},
		{
			desc: "no pds with 2 replicas and call api failed",
			state: &ReconcileContext{
				State: &state{
					pdg: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				},
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "1 outdated pd with 2 replicas",
			state: &ReconcileContext{
				State: &state{
					pdg: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					pds: []*v1alpha1.PD{
						fakeAvailablePD("aaa-xxx", fake.FakeObj[v1alpha1.PDGroup]("aaa"), oldRevision),
					},
					updateRevision: newRevision,
				},
			},

			expectedStatus: task.SWait,
			expectedPDNum:  2,
		},
		{
			desc: "1 outdated pd with 2 replicas but cannot call api, will fail",
			state: &ReconcileContext{
				State: &state{
					pdg: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					pds: []*v1alpha1.PD{
						fakeAvailablePD("aaa-xxx", fake.FakeObj[v1alpha1.PDGroup]("aaa"), oldRevision),
					},
					updateRevision: newRevision,
				},
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "2 updated pd with 2 replicas",
			state: &ReconcileContext{
				State: &state{
					pdg: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					pds: []*v1alpha1.PD{
						fakeAvailablePD("aaa-xxx", fake.FakeObj[v1alpha1.PDGroup]("aaa"), newRevision),
						fakeAvailablePD("aaa-yyy", fake.FakeObj[v1alpha1.PDGroup]("aaa"), newRevision),
					},
					updateRevision: newRevision,
				},
			},

			expectedStatus: task.SComplete,
			expectedPDNum:  2,
		},
		{
			desc: "2 updated pd with 2 replicas and cannot call api, can complete",
			state: &ReconcileContext{
				State: &state{
					pdg: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					pds: []*v1alpha1.PD{
						fakeAvailablePD("aaa-xxx", fake.FakeObj[v1alpha1.PDGroup]("aaa"), newRevision),
						fakeAvailablePD("aaa-yyy", fake.FakeObj[v1alpha1.PDGroup]("aaa"), newRevision),
					},
					updateRevision: newRevision,
				},
			},
			unexpectedErr: true,

			expectedStatus: task.SComplete,
			expectedPDNum:  2,
		},
		{
			// NOTE: it not really check whether the policy is worked
			// It should be tested in /pkg/updater and /pkg/updater/policy package
			desc: "topology evenly spread",
			state: &ReconcileContext{
				State: &state{
					pdg: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
						obj.Spec.Replicas = ptr.To[int32](3)
						obj.Spec.SchedulePolicies = append(obj.Spec.SchedulePolicies, v1alpha1.SchedulePolicy{
							Type: v1alpha1.SchedulePolicyTypeEvenlySpread,
							EvenlySpread: &v1alpha1.SchedulePolicyEvenlySpread{
								Topologies: []v1alpha1.ScheduleTopology{
									{
										Topology: v1alpha1.Topology{
											"zone": "us-west-1a",
										},
									},
									{
										Topology: v1alpha1.Topology{
											"zone": "us-west-1b",
										},
									},
									{
										Topology: v1alpha1.Topology{
											"zone": "us-west-1c",
										},
									},
								},
							},
						})
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				},
			},

			expectedStatus: task.SWait,
			expectedPDNum:  3,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			s := c.state.State.(*state)
			s.IFeatureGates = stateutil.NewFeatureGates[scope.PDGroup](s)

			ctx := context.Background()
			fc := client.NewFakeClient(c.state.PDGroup(), c.state.Cluster())
			for _, obj := range c.state.PDSlice() {
				require.NoError(tt, fc.Apply(ctx, obj.DeepCopy()), c.desc)
				require.NoError(tt, fc.Status().Update(ctx, obj.DeepCopy()), c.desc)
			}

			if c.unexpectedErr {
				// cannot create or update pd instance
				fc.WithError("patch", "pds", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			tr := tracker.New[*v1alpha1.PDGroup, *v1alpha1.PD]()
			res, done := task.RunTask(ctx, TaskUpdater(c.state, fc, tr))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)

			if !c.unexpectedErr {
				pds := v1alpha1.PDList{}
				require.NoError(tt, fc.List(ctx, &pds), c.desc)
				assert.Len(tt, pds.Items, c.expectedPDNum, c.desc)
			}
		})
	}
}

func fakeAvailablePD(name string, pdg *v1alpha1.PDGroup, rev string) *v1alpha1.PD {
	return fake.FakeObj(name, func(obj *v1alpha1.PD) *v1alpha1.PD {
		pd := runtime.ToPD(PDNewer(pdg, rev, features.NewFromFeatures(nil)).New())
		pd.Name = ""
		pd.Status.Conditions = append(pd.Status.Conditions, metav1.Condition{
			Type:   v1alpha1.CondReady,
			Status: metav1.ConditionTrue,
		})
		pd.Status.CurrentRevision = rev
		pd.DeepCopyInto(obj)
		return obj
	})
}
