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
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
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
				Name: "aaa",
			},
			objs: []client.Object{
				fake.FakeObj("aaa", func(obj *v1alpha1.TiCDCGroup) *v1alpha1.TiCDCGroup {
					obj.Spec.Cluster.Name = "aaa"
					return obj
				}),
				fake.FakeObj[v1alpha1.Cluster]("aaa"),
				fake.FakeObj("aaa", func(obj *v1alpha1.TiCDC) *v1alpha1.TiCDC {
					obj.Labels = map[string]string{
						v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
						v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiCDC,
						v1alpha1.LabelKeyCluster:   "aaa",
						v1alpha1.LabelKeyGroup:     "aaa",
					}
					return obj
				}),
			},

			expected: &state{
				key: types.NamespacedName{
					Name: "aaa",
				},
				cdcg: fake.FakeObj("aaa", func(obj *v1alpha1.TiCDCGroup) *v1alpha1.TiCDCGroup {
					obj.Spec.Cluster.Name = "aaa"
					return obj
				}),
				cluster: fake.FakeObj[v1alpha1.Cluster]("aaa"),
				cdcs: []*v1alpha1.TiCDC{
					fake.FakeObj("aaa", func(obj *v1alpha1.TiCDC) *v1alpha1.TiCDC {
						obj.Labels = map[string]string{
							v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
							v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiCDC,
							v1alpha1.LabelKeyCluster:   "aaa",
							v1alpha1.LabelKeyGroup:     "aaa",
						}
						return obj
					}),
				},
				updateRevision:  "aaa-ticdc-5464f78f9d",
				currentRevision: "aaa-ticdc-5464f78f9d",
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fc := client.NewFakeClient(c.objs...)

			s := NewState(c.key)

			ctx := context.Background()
			res, done := task.RunTask(ctx, task.Block(
				common.TaskContextTiCDCGroup(s, fc),
				common.TaskContextCluster(s, fc),
				common.TaskContextTiCDCSlice(s, fc),
				common.TaskRevision[runtime.TiCDCGroupTuple](s, fc),
			))
			assert.Equal(tt, task.SComplete, res.Status(), c.desc)
			assert.False(tt, done, c.desc)
			assert.Equal(tt, c.expected, s, c.desc)
		})
	}
}
