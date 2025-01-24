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
		unexpectedErr bool

		expectedStatus  task.Status
		expectedTiKVNum int
	}{
		{
			desc: "no kvs with 1 replicas",
			state: &ReconcileContext{
				State: &state{
					kvg:     fake.FakeObj[v1alpha1.TiKVGroup]("aaa"),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				},
			},

			expectedStatus:  task.SWait,
			expectedTiKVNum: 1,
		},
		{
			desc: "version upgrade check",
			state: &ReconcileContext{
				State: &state{
					kvg: fake.FakeObj("aaa", func(obj *v1alpha1.TiKVGroup) *v1alpha1.TiKVGroup {
						// use an wrong version to trigger version check
						// TODO(liubo02): it's not happened actually. Maybe remove whole checking
						obj.Spec.Version = "xxx"
						obj.Status.Version = "yyy"
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				},
			},

			expectedStatus: task.SRetry,
		},
		{
			desc: "1 updated tikv with 1 replicas",
			state: &ReconcileContext{
				State: &state{
					kvg:     fake.FakeObj[v1alpha1.TiKVGroup]("aaa"),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					kvs: []*v1alpha1.TiKV{
						fakeAvailableTiKV("aaa-xxx", fake.FakeObj[v1alpha1.TiKVGroup]("aaa"), newRevision),
					},
					updateRevision: newRevision,
				},
			},

			expectedStatus:  task.SComplete,
			expectedTiKVNum: 1,
		},
		{
			desc: "no kvs with 2 replicas",
			state: &ReconcileContext{
				State: &state{
					kvg: fake.FakeObj("aaa", func(obj *v1alpha1.TiKVGroup) *v1alpha1.TiKVGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				},
			},

			expectedStatus:  task.SWait,
			expectedTiKVNum: 2,
		},
		{
			desc: "no kvs with 2 replicas and call api failed",
			state: &ReconcileContext{
				State: &state{
					kvg: fake.FakeObj("aaa", func(obj *v1alpha1.TiKVGroup) *v1alpha1.TiKVGroup {
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
			desc: "1 outdated tikv with 2 replicas",
			state: &ReconcileContext{
				State: &state{
					kvg: fake.FakeObj("aaa", func(obj *v1alpha1.TiKVGroup) *v1alpha1.TiKVGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					kvs: []*v1alpha1.TiKV{
						fakeAvailableTiKV("aaa-xxx", fake.FakeObj[v1alpha1.TiKVGroup]("aaa"), oldRevision),
					},
					updateRevision: newRevision,
				},
			},

			expectedStatus:  task.SWait,
			expectedTiKVNum: 2,
		},
		{
			desc: "1 outdated tikv with 2 replicas but cannot call api, will fail",
			state: &ReconcileContext{
				State: &state{
					kvg: fake.FakeObj("aaa", func(obj *v1alpha1.TiKVGroup) *v1alpha1.TiKVGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					kvs: []*v1alpha1.TiKV{
						fakeAvailableTiKV("aaa-xxx", fake.FakeObj[v1alpha1.TiKVGroup]("aaa"), oldRevision),
					},
					updateRevision: newRevision,
				},
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "2 updated tikv with 2 replicas",
			state: &ReconcileContext{
				State: &state{
					kvg: fake.FakeObj("aaa", func(obj *v1alpha1.TiKVGroup) *v1alpha1.TiKVGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					kvs: []*v1alpha1.TiKV{
						fakeAvailableTiKV("aaa-xxx", fake.FakeObj[v1alpha1.TiKVGroup]("aaa"), newRevision),
						fakeAvailableTiKV("aaa-yyy", fake.FakeObj[v1alpha1.TiKVGroup]("aaa"), newRevision),
					},
					updateRevision: newRevision,
				},
			},

			expectedStatus:  task.SComplete,
			expectedTiKVNum: 2,
		},
		{
			desc: "2 updated tikv with 2 replicas and cannot call api, can complete",
			state: &ReconcileContext{
				State: &state{
					kvg: fake.FakeObj("aaa", func(obj *v1alpha1.TiKVGroup) *v1alpha1.TiKVGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					kvs: []*v1alpha1.TiKV{
						fakeAvailableTiKV("aaa-xxx", fake.FakeObj[v1alpha1.TiKVGroup]("aaa"), newRevision),
						fakeAvailableTiKV("aaa-yyy", fake.FakeObj[v1alpha1.TiKVGroup]("aaa"), newRevision),
					},
					updateRevision: newRevision,
				},
			},
			unexpectedErr: true,

			expectedStatus:  task.SComplete,
			expectedTiKVNum: 2,
		},
		{
			// NOTE: it not really check whether the policy is worked
			// It should be tested in /pkg/updater and /pkg/updater/policy package
			desc: "topology evenly spread",
			state: &ReconcileContext{
				State: &state{
					kvg: fake.FakeObj("aaa", func(obj *v1alpha1.TiKVGroup) *v1alpha1.TiKVGroup {
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
			expectedTiKVNum: 3,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctx := context.Background()
			fc := client.NewFakeClient(c.state.TiKVGroup(), c.state.Cluster())
			for _, obj := range c.state.TiKVSlice() {
				require.NoError(tt, fc.Apply(ctx, obj), c.desc)
			}

			if c.unexpectedErr {
				// cannot create or update tikv instance
				fc.WithError("patch", "tikvs", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			res, done := task.RunTask(ctx, TaskUpdater(c.state, fc))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)

			if !c.unexpectedErr {
				kvs := v1alpha1.TiKVList{}
				require.NoError(tt, fc.List(ctx, &kvs), c.desc)
				assert.Len(tt, kvs.Items, c.expectedTiKVNum, c.desc)
			}
		})
	}
}

func fakeAvailableTiKV(name string, kvg *v1alpha1.TiKVGroup, rev string) *v1alpha1.TiKV {
	return fake.FakeObj(name, func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
		tikv := runtime.ToTiKV(TiKVNewer(kvg, rev).New())
		tikv.Name = ""
		tikv.Status.Conditions = append(tikv.Status.Conditions, metav1.Condition{
			Type:   v1alpha1.TiKVCondHealth,
			Status: metav1.ConditionTrue,
		})
		tikv.Status.CurrentRevision = rev
		tikv.DeepCopyInto(obj)
		return obj
	})
}
