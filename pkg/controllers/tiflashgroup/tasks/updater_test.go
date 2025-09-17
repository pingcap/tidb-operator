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
		objs          []client.Object
		unexpectedErr bool

		expectedStatus     task.Status
		expectedTiFlashNum int
	}{
		{
			desc: "no fs with 1 replicas",
			state: &ReconcileContext{
				State: &state{
					fg:      fake.FakeObj[v1alpha1.TiFlashGroup]("aaa"),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				},
			},

			expectedStatus:     task.SWait,
			expectedTiFlashNum: 1,
		},
		{
			desc: "version upgrade check",
			state: &ReconcileContext{
				State: &state{
					fg: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlashGroup) *v1alpha1.TiFlashGroup {
						obj.Spec.Template.Spec.Version = "v8.1.0"
						obj.Spec.Cluster.Name = "cluster"
						obj.Status.Version = "v8.0.0"
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				},
			},
			objs: []client.Object{
				fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
					obj.Spec.Cluster.Name = "cluster"
					obj.Spec.Replicas = ptr.To[int32](1)
					obj.Spec.Template.Spec.Version = "v8.1.0"
					obj.Status.Version = "v8.0.0"
					return obj
				}),
			},

			expectedStatus: task.SRetry,
		},
		{
			desc: "1 updated tiflash with 1 replicas",
			state: &ReconcileContext{
				State: &state{
					fg:      fake.FakeObj[v1alpha1.TiFlashGroup]("aaa"),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					fs: []*v1alpha1.TiFlash{
						fakeAvailableTiFlash("aaa-xxx", fake.FakeObj[v1alpha1.TiFlashGroup]("aaa"), newRevision),
					},
					updateRevision: newRevision,
				},
			},

			expectedStatus:     task.SComplete,
			expectedTiFlashNum: 1,
		},
		{
			desc: "no fs with 2 replicas",
			state: &ReconcileContext{
				State: &state{
					fg: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlashGroup) *v1alpha1.TiFlashGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				},
			},

			expectedStatus:     task.SWait,
			expectedTiFlashNum: 2,
		},
		{
			desc: "no fs with 2 replicas and call api failed",
			state: &ReconcileContext{
				State: &state{
					fg: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlashGroup) *v1alpha1.TiFlashGroup {
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
			desc: "1 outdated tiflash with 2 replicas",
			state: &ReconcileContext{
				State: &state{
					fg: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlashGroup) *v1alpha1.TiFlashGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					fs: []*v1alpha1.TiFlash{
						fakeAvailableTiFlash("aaa-xxx", fake.FakeObj[v1alpha1.TiFlashGroup]("aaa"), oldRevision),
					},
					updateRevision: newRevision,
				},
			},

			expectedStatus:     task.SWait,
			expectedTiFlashNum: 2,
		},
		{
			desc: "1 outdated tiflash with 2 replicas but cannot call api, will fail",
			state: &ReconcileContext{
				State: &state{
					fg: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlashGroup) *v1alpha1.TiFlashGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					fs: []*v1alpha1.TiFlash{
						fakeAvailableTiFlash("aaa-xxx", fake.FakeObj[v1alpha1.TiFlashGroup]("aaa"), oldRevision),
					},
					updateRevision: newRevision,
				},
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "2 updated tiflash with 2 replicas",
			state: &ReconcileContext{
				State: &state{
					fg: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlashGroup) *v1alpha1.TiFlashGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					fs: []*v1alpha1.TiFlash{
						fakeAvailableTiFlash("aaa-xxx", fake.FakeObj[v1alpha1.TiFlashGroup]("aaa"), newRevision),
						fakeAvailableTiFlash("aaa-yyy", fake.FakeObj[v1alpha1.TiFlashGroup]("aaa"), newRevision),
					},
					updateRevision: newRevision,
				},
			},

			expectedStatus:     task.SComplete,
			expectedTiFlashNum: 2,
		},
		{
			desc: "2 updated tiflash with 2 replicas and cannot call api, can complete",
			state: &ReconcileContext{
				State: &state{
					fg: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlashGroup) *v1alpha1.TiFlashGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					fs: []*v1alpha1.TiFlash{
						fakeAvailableTiFlash("aaa-xxx", fake.FakeObj[v1alpha1.TiFlashGroup]("aaa"), newRevision),
						fakeAvailableTiFlash("aaa-yyy", fake.FakeObj[v1alpha1.TiFlashGroup]("aaa"), newRevision),
					},
					updateRevision: newRevision,
				},
			},
			unexpectedErr: true,

			expectedStatus:     task.SComplete,
			expectedTiFlashNum: 2,
		},
		{
			// NOTE: it not really check whether the policy is worked
			// It should be tested in /pkg/updater and /pkg/updater/policy package
			desc: "topology evenly spread",
			state: &ReconcileContext{
				State: &state{
					fg: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlashGroup) *v1alpha1.TiFlashGroup {
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

			expectedStatus:     task.SWait,
			expectedTiFlashNum: 3,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			s := c.state.State.(*state)
			s.IFeatureGates = stateutil.NewFeatureGates[scope.TiFlashGroup](s)

			ctx := context.Background()
			c.objs = append(c.objs, c.state.TiFlashGroup(), c.state.Cluster())
			fc := client.NewFakeClient(c.objs...)
			for _, obj := range c.state.TiFlashSlice() {
				require.NoError(tt, fc.Apply(ctx, obj.DeepCopy()), c.desc)
				require.NoError(tt, fc.Status().Update(ctx, obj.DeepCopy()), c.desc)
			}

			if c.unexpectedErr {
				// cannot create or update tiflash instance
				fc.WithError("patch", "tiflashs", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			af := tracker.New().AllocateFactory("tiflash")
			res, done := task.RunTask(ctx, TaskUpdater(c.state, fc, af))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)

			if !c.unexpectedErr {
				fs := v1alpha1.TiFlashList{}
				require.NoError(tt, fc.List(ctx, &fs), c.desc)
				assert.Len(tt, fs.Items, c.expectedTiFlashNum, c.desc)
			}
		})
	}
}

func fakeAvailableTiFlash(name string, fg *v1alpha1.TiFlashGroup, rev string) *v1alpha1.TiFlash {
	return fake.FakeObj(name, func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
		tiflash := runtime.ToTiFlash(TiFlashNewer(fg, rev, features.NewFromFeatures(nil)).New())
		tiflash.Name = ""
		tiflash.Status.Conditions = append(tiflash.Status.Conditions, metav1.Condition{
			Type:   v1alpha1.CondReady,
			Status: metav1.ConditionTrue,
		})
		tiflash.Status.CurrentRevision = rev
		tiflash.DeepCopyInto(obj)
		return obj
	})
}
