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
	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	core "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	task "github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

type deletedOnRefreshClient struct {
	client.Client
	calls, deleteAt int
}

func (c *deletedOnRefreshClient) Get(ctx context.Context, key ctrlclient.ObjectKey, obj ctrlclient.Object, opts ...ctrlclient.GetOption) error {
	if _, ok := obj.(*br.TiBR); ok {
		c.calls++
		if c.calls == c.deleteAt {
			return apierrors.NewNotFound(schema.GroupResource{Group: br.SchemeGroupVersion.Group, Resource: "tibrs"}, key.Name)
		}
	}
	return c.Client.Get(ctx, key, obj, opts...)
}
func TestEnsureStopsAfterDeletedCRRefresh(t *testing.T) {
	for _, deleteAt := range []int{1} {
		t.Run(fmt.Sprint(deleteAt), func(t *testing.T) {
			ctx := context.Background()
			obj := &br.TiBR{ObjectMeta: metav1.ObjectMeta{Name: "br", Namespace: "test"}}
			cluster := &core.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "test", UID: "cluster-uid"}}
			base := client.NewFakeClient(obj)
			require.NoError(t, base.Get(ctx, types.NamespacedName{Namespace: "test", Name: "br"}, obj))
			cli := &deletedOnRefreshClient{Client: base, deleteAt: deleteAt}
			rtx := NewReconcileContext(ctx, types.NamespacedName{Namespace: "test", Name: "br"}, cli)
			rtx.tibr = obj
			rtx.cluster = cluster
			continued := false
			result, done := task.RunTask(ctx, task.Block(TaskEnsureSubResources(rtx), task.NameTaskFunc("after ensure", func(context.Context) task.Result { continued = true; return task.Complete().With("continued") })))
			require.Equal(t, task.SComplete, result.Status())
			require.True(t, done)
			require.False(t, continued)
			require.Nil(t, rtx.TiBR())
		})
	}
}
