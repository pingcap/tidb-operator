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

		expectedStatus     task.Status
		expectedTiProxyNum int
	}{
		{
			desc: "no proxies with 1 replicas",
			state: &ReconcileContext{
				State: &state{
					proxyg:  fake.FakeObj[v1alpha1.TiProxyGroup]("aaa"),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				},
			},

			expectedStatus:     task.SWait,
			expectedTiProxyNum: 1,
		},
		{
			desc: "1 updated tiproxy with 1 replicas",
			state: &ReconcileContext{
				State: &state{
					proxyg:  fake.FakeObj[v1alpha1.TiProxyGroup]("aaa"),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					proxies: []*v1alpha1.TiProxy{
						fakeAvailableTiProxy("aaa-xxx", fake.FakeObj[v1alpha1.TiProxyGroup]("aaa"), newRevision),
					},
					updateRevision: newRevision,
				},
			},

			expectedStatus:     task.SComplete,
			expectedTiProxyNum: 1,
		},
		{
			desc: "no proxies with 2 replicas",
			state: &ReconcileContext{
				State: &state{
					proxyg: fake.FakeObj("aaa", func(obj *v1alpha1.TiProxyGroup) *v1alpha1.TiProxyGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				},
			},

			expectedStatus:     task.SWait,
			expectedTiProxyNum: 2,
		},
		{
			desc: "no proxies with 2 replicas and call api failed",
			state: &ReconcileContext{
				State: &state{
					proxyg: fake.FakeObj("aaa", func(obj *v1alpha1.TiProxyGroup) *v1alpha1.TiProxyGroup {
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
			desc: "1 outdated tiproxy with 2 replicas",
			state: &ReconcileContext{
				State: &state{
					proxyg: fake.FakeObj("aaa", func(obj *v1alpha1.TiProxyGroup) *v1alpha1.TiProxyGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					proxies: []*v1alpha1.TiProxy{
						fakeAvailableTiProxy("aaa-xxx", fake.FakeObj[v1alpha1.TiProxyGroup]("aaa"), oldRevision),
					},
					updateRevision: newRevision,
				},
			},

			expectedStatus:     task.SWait,
			expectedTiProxyNum: 2,
		},
		{
			desc: "1 outdated tiproxy with 2 replicas but cannot call api, will fail",
			state: &ReconcileContext{
				State: &state{
					proxyg: fake.FakeObj("aaa", func(obj *v1alpha1.TiProxyGroup) *v1alpha1.TiProxyGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					proxies: []*v1alpha1.TiProxy{
						fakeAvailableTiProxy("aaa-xxx", fake.FakeObj[v1alpha1.TiProxyGroup]("aaa"), oldRevision),
					},
					updateRevision: newRevision,
				},
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "2 updated tiproxy with 2 replicas",
			state: &ReconcileContext{
				State: &state{
					proxyg: fake.FakeObj("aaa", func(obj *v1alpha1.TiProxyGroup) *v1alpha1.TiProxyGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					proxies: []*v1alpha1.TiProxy{
						fakeAvailableTiProxy("aaa-xxx", fake.FakeObj[v1alpha1.TiProxyGroup]("aaa"), newRevision),
						fakeAvailableTiProxy("aaa-yyy", fake.FakeObj[v1alpha1.TiProxyGroup]("aaa"), newRevision),
					},
					updateRevision: newRevision,
				},
			},

			expectedStatus:     task.SComplete,
			expectedTiProxyNum: 2,
		},
		{
			desc: "2 updated tiproxy with 2 replicas and cannot call api, can complete",
			state: &ReconcileContext{
				State: &state{
					proxyg: fake.FakeObj("aaa", func(obj *v1alpha1.TiProxyGroup) *v1alpha1.TiProxyGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					proxies: []*v1alpha1.TiProxy{
						fakeAvailableTiProxy("aaa-xxx", fake.FakeObj[v1alpha1.TiProxyGroup]("aaa"), newRevision),
						fakeAvailableTiProxy("aaa-yyy", fake.FakeObj[v1alpha1.TiProxyGroup]("aaa"), newRevision),
					},
					updateRevision: newRevision,
				},
			},
			unexpectedErr: true,

			expectedStatus:     task.SComplete,
			expectedTiProxyNum: 2,
		},
		{
			// NOTE: it not really check whether the policy is worked
			// It should be tested in /pkg/updater and /pkg/updater/policy package
			desc: "topology evenly spread",
			state: &ReconcileContext{
				State: &state{
					proxyg: fake.FakeObj("aaa", func(obj *v1alpha1.TiProxyGroup) *v1alpha1.TiProxyGroup {
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
			expectedTiProxyNum: 3,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctx := context.Background()
			c.objs = append(c.objs, c.state.TiProxyGroup(), c.state.Cluster())
			fc := client.NewFakeClient(c.objs...)
			for _, obj := range c.state.TiProxySlice() {
				require.NoError(tt, fc.Apply(ctx, obj), c.desc)
			}

			if c.unexpectedErr {
				// cannot create or update tidb instance
				fc.WithError("patch", "tiproxies", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			res, done := task.RunTask(ctx, TaskUpdater(c.state, fc))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)

			if !c.unexpectedErr {
				proxies := v1alpha1.TiProxyList{}
				require.NoError(tt, fc.List(ctx, &proxies), c.desc)
				assert.Len(tt, proxies.Items, c.expectedTiProxyNum, c.desc)
			}
		})
	}
}

func fakeAvailableTiProxy(name string, proxyg *v1alpha1.TiProxyGroup, rev string) *v1alpha1.TiProxy {
	return fake.FakeObj(name, func(obj *v1alpha1.TiProxy) *v1alpha1.TiProxy {
		tiproxy := runtime.ToTiProxy(TiProxyNewer(proxyg, rev).New())
		tiproxy.Name = ""
		tiproxy.Status.Conditions = append(tiproxy.Status.Conditions, metav1.Condition{
			Type:   v1alpha1.CondReady,
			Status: metav1.ConditionTrue,
		})
		tiproxy.Status.CurrentRevision = rev
		tiproxy.DeepCopyInto(obj)
		return obj
	})
}
