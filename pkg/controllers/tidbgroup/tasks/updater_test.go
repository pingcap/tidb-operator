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

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/adoption"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/features"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	stateutil "github.com/pingcap/tidb-operator/v2/pkg/state"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/tracker"
)

const (
	oldRevision = "old"
	newRevision = "new"
)

func TestTaskUpdater(t *testing.T) {
	t.Run("progressing omitted", func(t *testing.T) {
		testTaskUpdater(t, nil)
	})
	t.Run("progressing true", func(t *testing.T) {
		testTaskUpdater(t, ptr.To(true))
	})
}

func testTaskUpdater(t *testing.T, progressing *bool) {
	t.Helper()
	cases := []struct {
		desc          string
		state         *ReconcileContext
		objs          []client.Object
		unexpectedErr bool

		expectedStatus  task.Status
		expectedTiDBNum int
	}{
		{
			desc: "no dbs with 1 replicas",
			state: &ReconcileContext{
				State: &state{
					dbg:     fake.FakeObj[v1alpha1.TiDBGroup]("aaa"),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				},
			},

			expectedStatus:  task.SWait,
			expectedTiDBNum: 1,
		},
		{
			desc: "version upgrade check",
			state: &ReconcileContext{
				State: &state{
					dbg: fake.FakeObj("aaa", func(obj *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
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
					obj.Spec.Replicas = ptr.To[int32](1)
					obj.Spec.Cluster.Name = "cluster"
					obj.Spec.Template.Spec.Version = "v8.1.0"
					obj.Status.Version = "v8.0.0"
					return obj
				}),
			},

			expectedStatus: task.SRetry,
		},
		{
			desc: "1 updated tidb with 1 replicas",
			state: &ReconcileContext{
				State: &state{
					dbg:     fake.FakeObj[v1alpha1.TiDBGroup]("aaa"),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					dbs: []*v1alpha1.TiDB{
						fakeAvailableTiDB("aaa-xxx", fake.FakeObj[v1alpha1.TiDBGroup]("aaa"), newRevision),
					},
					updateRevision: newRevision,
				},
			},

			expectedStatus:  task.SComplete,
			expectedTiDBNum: 1,
		},
		{
			desc: "no dbs with 2 replicas",
			state: &ReconcileContext{
				State: &state{
					dbg: fake.FakeObj("aaa", func(obj *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				},
			},

			expectedStatus:  task.SWait,
			expectedTiDBNum: 2,
		},
		{
			desc: "no dbs with 2 replicas and call api failed",
			state: &ReconcileContext{
				State: &state{
					dbg: fake.FakeObj("aaa", func(obj *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
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
			desc: "1 outdated tidb with 2 replicas",
			state: &ReconcileContext{
				State: &state{
					dbg: fake.FakeObj("aaa", func(obj *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					dbs: []*v1alpha1.TiDB{
						fakeAvailableTiDB("aaa-xxx", fake.FakeObj[v1alpha1.TiDBGroup]("aaa"), oldRevision),
					},
					updateRevision: newRevision,
				},
			},

			expectedStatus:  task.SWait,
			expectedTiDBNum: 2,
		},
		{
			desc: "1 outdated tidb with 2 replicas but cannot call api, will fail",
			state: &ReconcileContext{
				State: &state{
					dbg: fake.FakeObj("aaa", func(obj *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					dbs: []*v1alpha1.TiDB{
						fakeAvailableTiDB("aaa-xxx", fake.FakeObj[v1alpha1.TiDBGroup]("aaa"), oldRevision),
					},
					updateRevision: newRevision,
				},
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "2 updated tidb with 2 replicas",
			state: &ReconcileContext{
				State: &state{
					dbg: fake.FakeObj("aaa", func(obj *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					dbs: []*v1alpha1.TiDB{
						fakeAvailableTiDB("aaa-xxx", fake.FakeObj[v1alpha1.TiDBGroup]("aaa"), newRevision),
						fakeAvailableTiDB("aaa-yyy", fake.FakeObj[v1alpha1.TiDBGroup]("aaa"), newRevision),
					},
					updateRevision: newRevision,
				},
			},

			expectedStatus:  task.SComplete,
			expectedTiDBNum: 2,
		},
		{
			desc: "2 updated tidb with 2 replicas and cannot call api, can complete",
			state: &ReconcileContext{
				State: &state{
					dbg: fake.FakeObj("aaa", func(obj *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					dbs: []*v1alpha1.TiDB{
						fakeAvailableTiDB("aaa-xxx", fake.FakeObj[v1alpha1.TiDBGroup]("aaa"), newRevision),
						fakeAvailableTiDB("aaa-yyy", fake.FakeObj[v1alpha1.TiDBGroup]("aaa"), newRevision),
					},
					updateRevision: newRevision,
				},
			},
			unexpectedErr: true,

			expectedStatus:  task.SComplete,
			expectedTiDBNum: 2,
		},
		{
			// NOTE: it not really check whether the policy is worked
			// It should be tested in /pkg/updater and /pkg/updater/policy package
			desc: "topology evenly spread",
			state: &ReconcileContext{
				State: &state{
					dbg: fake.FakeObj("aaa", func(obj *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
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

			expectedStatus:  task.SWait,
			expectedTiDBNum: 3,
		},
		{
			desc: "rolling restart honors maxSurge",
			state: &ReconcileContext{
				State: &state{
					dbg: fake.FakeObj("aaa", func(obj *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						obj.Spec.MaxSurge = ptr.To[int32](2)
						obj.Spec.Template.Spec.Image = ptr.To("pingcap/tidb:v8.1.1")
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					dbs: []*v1alpha1.TiDB{
						fakeAvailableTiDB("aaa-xxx", fake.FakeObj[v1alpha1.TiDBGroup]("aaa"), oldRevision),
						fakeAvailableTiDB("aaa-yyy", fake.FakeObj[v1alpha1.TiDBGroup]("aaa"), oldRevision),
					},
					updateRevision: newRevision,
				},
			},

			expectedStatus:  task.SWait,
			expectedTiDBNum: 4,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			s := c.state.State.(*state)
			s.IFeatureGates = stateutil.NewFeatureGates[scope.TiDBGroup](s)

			ctx := context.Background()
			c.objs = append(c.objs, c.state.TiDBGroup(), c.state.Cluster())
			fc := client.NewFakeClient(c.objs...)
			for _, obj := range c.state.TiDBSlice() {
				require.NoError(tt, fc.Apply(ctx, obj.DeepCopy()), c.desc)
				require.NoError(tt, fc.Status().Update(ctx, obj.DeepCopy()), c.desc)
			}

			if c.unexpectedErr {
				// cannot create or update tidb instance
				fc.WithError("patch", "tidbs", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			af := tracker.New().AllocateFactory("tidb")
			// Pausing must preserve instances in every rollout/scaling scenario.
			beforePause := v1alpha1.TiDBList{}
			require.NoError(tt, fc.List(ctx, &beforePause))
			c.state.Object().Spec.Progressing = ptr.To(false)
			paused, stopped := task.RunTask(ctx, TaskUpdater(c.state, fc, af, adoption.New(logr.Discard())))
			require.Equal(tt, task.SWait, paused.Status())
			require.False(tt, stopped, "status tasks must continue while paused")
			afterPause := v1alpha1.TiDBList{}
			require.NoError(tt, fc.List(ctx, &afterPause))
			require.ElementsMatch(tt, beforePause.Items, afterPause.Items)

			// Resuming restores the original updater behavior.
			c.state.Object().Spec.Progressing = progressing
			res, done := task.RunTask(ctx, TaskUpdater(c.state, fc, af, adoption.New(logr.Discard())))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)

			if !c.unexpectedErr {
				dbs := v1alpha1.TiDBList{}
				require.NoError(tt, fc.List(ctx, &dbs), c.desc)
				assert.Len(tt, dbs.Items, c.expectedTiDBNum, c.desc)
			}
		})
	}
}

func TestPausedUpdaterRetainsSurgeUntilResume(t *testing.T) {
	for _, deferred := range []bool{false, true} {
		t.Run(fmt.Sprintf("deferred=%v", deferred), func(t *testing.T) {
			ctx := context.Background()
			group := fake.FakeObj("aaa", func(g *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
				g.Spec.Replicas = ptr.To[int32](1)
				g.Spec.Progressing = ptr.To(false)
				g.Spec.Template.Spec.Image = ptr.To("pingcap/tidb:v8.1.1")
				return g
			})
			old := fakeAvailableTiDB("aaa-old", fake.FakeObj[v1alpha1.TiDBGroup]("aaa"), oldRevision)
			if deferred {
				old.Annotations = map[string]string{v1alpha1.AnnoKeyDeferDelete: v1alpha1.AnnoValTrue}
			}
			replacement := fakeAvailableTiDB("aaa-new", group, newRevision)
			s := &state{
				dbg: group, cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				dbs: []*v1alpha1.TiDB{old, replacement}, updateRevision: newRevision,
			}
			s.IFeatureGates = stateutil.NewFeatureGates[scope.TiDBGroup](s)
			rtx := &ReconcileContext{State: s}
			fc := client.NewFakeClient(group, s.cluster, old, replacement)
			af := tracker.New().AllocateFactory("tidb")
			adopter := adoption.New(logr.Discard())
			res, stopped := task.RunTask(ctx, TaskUpdater(rtx, fc, af, adopter))
			require.Equal(t, task.SWait, res.Status())
			require.False(t, stopped)
			list := &v1alpha1.TiDBList{}
			require.NoError(t, fc.List(ctx, list))
			require.ElementsMatch(t, []v1alpha1.TiDB{*old, *replacement}, list.Items)

			group.Spec.Progressing = ptr.To(true)
			res, _ = task.RunTask(ctx, TaskUpdater(rtx, fc, af, adopter))
			require.Equal(t, task.SComplete, res.Status(), res.Message())
			require.NoError(t, fc.List(ctx, list))
			require.Len(t, list.Items, 1)
			assert.Equal(t, replacement.Name, list.Items[0].Name)
		})
	}
}

func fakeAvailableTiDB(name string, dbg *v1alpha1.TiDBGroup, rev string) *v1alpha1.TiDB {
	f := newFactory(nil, dbg, rev, features.NewFromFeatures(nil))

	return fake.FakeObj(name, func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
		tidb := runtime.ToTiDB(f.New())
		tidb.Name = ""
		tidb.Status.Conditions = append(tidb.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.CondReady,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Unix(0, 0),
		})
		tidb.Status.CurrentRevision = rev
		tidb.DeepCopyInto(obj)
		return obj
	})
}
