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
				fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
					obj.Spec.Cluster.Name = "aaa"
					return obj
				}),
				fake.FakeObj[v1alpha1.Cluster]("aaa"),
				fake.FakeObj("aaa-tidb-xxx", fake.InstanceOwner[scope.TiDB, corev1.Pod](fake.FakeObj[v1alpha1.TiDB]("aaa-xxx"))),
			},

			expected: &state{
				key: types.NamespacedName{
					Name: "aaa-xxx",
				},
				tidb: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
					obj.Spec.Cluster.Name = "aaa"
					return obj
				}),
				cluster:   fake.FakeObj[v1alpha1.Cluster]("aaa"),
				pod:       fake.FakeObj("aaa-tidb-xxx", fake.InstanceOwner[scope.TiDB, corev1.Pod](fake.FakeObj[v1alpha1.TiDB]("aaa-xxx"))),
				IPDClient: stateutil.NewPDClientState(),
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
			expected.IFeatureGates = stateutil.NewFeatureGates[scope.TiDB](expected)

			ctx := context.Background()
			res, done := task.RunTask(ctx, task.Block(
				common.TaskContextObject[scope.TiDB](s, fc),
				common.TaskContextCluster[scope.TiDB](s, fc),
				common.TaskContextPod[scope.TiDB](s, fc),
			))
			assert.Equal(tt, task.SComplete, res.Status(), c.desc)
			assert.False(tt, done, c.desc)
			assert.Equal(tt, c.expected, s, c.desc)
		})
	}
}
