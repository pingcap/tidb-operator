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

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task"
)

func FakeContext(key types.NamespacedName, changes ...fake.ChangeFunc[ReconcileContext, *ReconcileContext]) *ReconcileContext {
	ctx := fake.Fake(changes...)
	ctx.Context = context.TODO()
	ctx.Key = key
	return ctx
}

func TestContext(t *testing.T) {
	cases := []struct {
		desc            string
		key             types.NamespacedName
		objs            []client.Object
		expected        task.Result
		expectedCluster *v1alpha1.Cluster
	}{
		{
			desc: "cluster has been deleted",
			key: types.NamespacedName{
				Name: "test",
			},
			objs:     []client.Object{},
			expected: task.Complete().Break().With(""),
		},
		{
			desc: "new context complete",
			key: types.NamespacedName{
				Name: "test",
			},
			objs: []client.Object{
				fake.FakeObj[v1alpha1.Cluster]("test"),
			},
			expected:        task.Complete().With(""),
			expectedCluster: fake.FakeObj[v1alpha1.Cluster]("test"),
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctx := FakeContext(c.key)
			fc := client.NewFakeClient(c.objs...)
			tk := NewTaskContext(logr.Discard(), fc)
			res := tk.Sync(ctx)

			assert.Equal(tt, c.expected.IsFailed(), res.IsFailed())
			assert.Equal(tt, c.expected.ShouldContinue(), res.ShouldContinue())
			assert.Equal(tt, c.expected.RequeueAfter(), res.RequeueAfter())
			// Ignore message assertion
			// TODO: maybe assert the message format?

			assert.Equal(tt, c.expectedCluster, ctx.Self().Cluster)
		})
	}
}
