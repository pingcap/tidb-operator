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
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/task"
)

func FakeContext(changes ...fake.ChangeFunc[ReconcileContext, *ReconcileContext]) *ReconcileContext {
	ctx := fake.Fake(changes...)
	ctx.Context = context.TODO()
	return ctx
}

func WithCluster(cluster *v1alpha1.Cluster) fake.ChangeFunc[ReconcileContext, *ReconcileContext] {
	return func(obj *ReconcileContext) *ReconcileContext {
		obj.Cluster = cluster
		return obj
	}
}

func WithTiDBGroup(dbg *v1alpha1.TiDBGroup) fake.ChangeFunc[ReconcileContext, *ReconcileContext] {
	return func(obj *ReconcileContext) *ReconcileContext {
		obj.TiDBGroup = dbg
		return obj
	}
}

func TestUpdater(t *testing.T) {
	tests := []struct {
		name       string
		ctx        *ReconcileContext
		objs       []client.Object
		expected   task.Result
		expectFunc func(t *testing.T, ctx *ReconcileContext, cli client.Client)
	}{
		{
			name: "first time to sync",
			ctx: FakeContext(
				WithCluster(fake.FakeObj[v1alpha1.Cluster]("test")),
				WithTiDBGroup(fake.FakeObj("test-tidbgroup",
					func(dbg *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
						dbg.Spec.Cluster = v1alpha1.ClusterReference{Name: "test"}
						dbg.Spec.Replicas = ptr.To(int32(1))
						return dbg
					},
				)),
			),
			expected: task.Complete().With(""),
			expectFunc: func(t *testing.T, _ *ReconcileContext, cli client.Client) {
				var crList appsv1.ControllerRevisionList
				require.NoError(t, cli.List(context.TODO(), &crList))
				assert.Len(t, crList.Items, 1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := client.NewFakeClient(tt.objs...)
			updaterTask := NewTaskUpdater(logr.Discard(), fc)
			got := updaterTask.Sync(tt.ctx)
			assert.Equal(t, tt.expected.IsFailed(), got.IsFailed())
			assert.Equal(t, tt.expected.ShouldContinue(), got.ShouldContinue())
			assert.Equal(t, tt.expected.RequeueAfter(), got.RequeueAfter())

			if tt.expectFunc != nil {
				tt.expectFunc(t, tt.ctx, fc)
			}
		})
	}
}
