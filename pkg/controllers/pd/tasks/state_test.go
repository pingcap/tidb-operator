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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	stateutil "github.com/pingcap/tidb-operator/pkg/state"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

const (
	fakeClusterName = "cluster"
)

func TestState(t *testing.T) {
	cases := []struct {
		desc string
		key  types.NamespacedName
		objs []client.Object

		expected State
	}{
		{
			desc: "normal",
			key: types.NamespacedName{
				Name: "aaa-xxx",
			},
			objs: []client.Object{
				fake.FakeObj("aaa-xxx", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Spec.Cluster.Name = fakeClusterName
					obj.Labels = map[string]string{
						v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
						v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
						v1alpha1.LabelKeyCluster:   fakeClusterName,
					}
					return obj
				}),
				fake.FakeObj[v1alpha1.Cluster](fakeClusterName),
				fake.FakeObj("aaa-yyy", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Spec.Cluster.Name = fakeClusterName
					obj.Labels = map[string]string{
						v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
						v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
						v1alpha1.LabelKeyCluster:   fakeClusterName,
					}
					return obj
				}),
				fake.FakeObj("aaa-pd-xxx", fake.InstanceOwner[scope.PD, corev1.Pod](fake.FakeObj[v1alpha1.PD]("aaa-xxx"))),
			},

			expected: &state{
				key: types.NamespacedName{
					Name: "aaa-xxx",
				},
				pd: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Spec.Cluster.Name = fakeClusterName
					obj.Labels = map[string]string{
						v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
						v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
						v1alpha1.LabelKeyCluster:   fakeClusterName,
					}
					return obj
				}),
				cluster: fake.FakeObj[v1alpha1.Cluster](fakeClusterName),
				pds: []*v1alpha1.PD{
					fake.FakeObj("aaa-xxx", func(obj *v1alpha1.PD) *v1alpha1.PD {
						obj.Spec.Cluster.Name = fakeClusterName
						obj.Labels = map[string]string{
							v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
							v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
							v1alpha1.LabelKeyCluster:   fakeClusterName,
						}
						return obj
					}),
					fake.FakeObj("aaa-yyy", func(obj *v1alpha1.PD) *v1alpha1.PD {
						obj.Spec.Cluster.Name = fakeClusterName
						obj.Labels = map[string]string{
							v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
							v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
							v1alpha1.LabelKeyCluster:   fakeClusterName,
						}
						return obj
					}),
				},
				pod: fake.FakeObj("aaa-pd-xxx", fake.InstanceOwner[scope.PD, corev1.Pod](fake.FakeObj[v1alpha1.PD]("aaa-xxx"))),
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fc := client.NewFakeClient(c.objs...)

			s := NewState(c.key)
			expected := c.expected.(*state)
			expected.IFeatureGates = stateutil.NewFeatureGates[scope.PD](expected)

			ctx := context.Background()
			res, done := task.RunTask(ctx, task.Block(
				common.TaskContextObject[scope.PD](s, fc),
				common.TaskContextCluster[scope.PD](s, fc),
				common.TaskContextPDSlice(s, fc),
				common.TaskContextPod[scope.PD](s, fc),
			))
			assert.Equal(tt, task.SComplete, res.Status(), c.desc)
			assert.False(tt, done, c.desc)
			assert.Equal(tt, c.expected, s, c.desc)
		})
	}
}
