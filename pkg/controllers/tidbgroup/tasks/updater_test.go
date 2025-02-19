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
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
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
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctx := context.Background()
			c.objs = append(c.objs, c.state.TiDBGroup(), c.state.Cluster())
			fc := client.NewFakeClient(c.objs...)
			for _, obj := range c.state.TiDBSlice() {
				require.NoError(tt, fc.Apply(ctx, obj), c.desc)
			}

			if c.unexpectedErr {
				// cannot create or update tidb instance
				fc.WithError("patch", "tidbs", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			res, done := task.RunTask(ctx, TaskUpdater(c.state, fc))
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

func fakeAvailableTiDB(name string, dbg *v1alpha1.TiDBGroup, rev string) *v1alpha1.TiDB {
	return fake.FakeObj(name, func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
		tidb := runtime.ToTiDB(TiDBNewer(dbg, rev).New())
		tidb.Name = ""
		tidb.Status.Conditions = append(tidb.Status.Conditions, metav1.Condition{
			Type:   v1alpha1.CondReady,
			Status: metav1.ConditionTrue,
		})
		tidb.Status.CurrentRevision = rev
		tidb.DeepCopyInto(obj)
		return obj
	})
}
