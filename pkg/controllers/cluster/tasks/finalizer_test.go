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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/scheme"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task"
)

func TestFinalizer(t *testing.T) {
	cases := []struct {
		desc         string
		cluster      *v1alpha1.Cluster
		pdGroup      *v1alpha1.PDGroup
		expected     task.Result
		hasFinalizer bool
	}{
		{
			desc:         "ensured finalizer",
			cluster:      fake.FakeObj[v1alpha1.Cluster]("test"),
			expected:     task.Complete().With("ensured finalizer"),
			hasFinalizer: true,
		},
		{
			desc: "removed finalizer",
			cluster: fake.FakeObj("test",
				fake.DeleteNow[v1alpha1.Cluster](), fake.AddFinalizer[v1alpha1.Cluster]()),
			expected: task.Complete().Break().With("removed finalizer"),
		},
		{
			desc: "deleting components",
			cluster: fake.FakeObj("test",
				fake.DeleteNow[v1alpha1.Cluster](), fake.AddFinalizer[v1alpha1.Cluster]()),
			pdGroup:      fake.FakeObj[v1alpha1.PDGroup]("pd-group"),
			expected:     task.Fail().With("deleting components"),
			hasFinalizer: true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctx := FakeContext(types.NamespacedName{Name: "test"})
			ctx.Cluster = c.cluster
			if c.pdGroup != nil {
				ctx.PDGroups = append(ctx.PDGroups, c.pdGroup)
			}

			var objs []client.Object
			objs = append(objs, c.cluster)
			if c.pdGroup != nil {
				objs = append(objs, c.pdGroup)
			}

			fc := client.NewFakeClient(objs...)
			tk := NewTaskFinalizer(logr.Discard(), fc, newFakePDClientManager(tt, fc))
			res := tk.Sync(ctx)
			assert.Equal(tt, c.expected, res)
			assert.Equal(tt, c.hasFinalizer, controllerutil.ContainsFinalizer(c.cluster, meta.Finalizer))
		})
	}
}

func TestFinalizerWaitsForRouterAndResourceManager(t *testing.T) {
	for _, kind := range []string{"router", "resource-manager"} {
		t.Run(kind, func(t *testing.T) {
			ctx := FakeContext(types.NamespacedName{Name: "test"})
			ctx.Cluster = fake.FakeObj("test", fake.DeleteNow[v1alpha1.Cluster](), fake.AddFinalizer[v1alpha1.Cluster]())
			var group client.Object
			if kind == "router" {
				g := fake.FakeObj("router", fake.AddFinalizer[v1alpha1.RouterGroup]())
				ctx.RouterGroups = []*v1alpha1.RouterGroup{g}
				group = g
			} else {
				g := fake.FakeObj("resource-manager", fake.AddFinalizer[v1alpha1.ResourceManagerGroup]())
				ctx.ResourceManagerGroups = []*v1alpha1.ResourceManagerGroup{g}
				group = g
			}
			fc := &finalizerClient{WithWatch: ctrlfake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(ctx.Cluster, group).Build()}
			tk := NewTaskFinalizer(logr.Discard(), fc, newFakePDClientManager(t, fc))
			require.Equal(t, task.Fail().With("deleting components"), tk.Sync(ctx))
			require.True(t, controllerutil.ContainsFinalizer(ctx.Cluster, meta.Finalizer))
			require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(group), group))
			require.NotNil(t, group.GetDeletionTimestamp())
			// A Group still terminating must keep the Cluster available to its reconciler.
			require.Equal(t, task.Fail().With("deleting components"), tk.Sync(ctx))
			require.True(t, controllerutil.ContainsFinalizer(ctx.Cluster, meta.Finalizer))
			require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(group), group))
			controllerutil.RemoveFinalizer(group, meta.Finalizer)
			require.NoError(t, fc.Update(ctx, group))
			require.True(t, errors.IsNotFound(fc.Get(ctx, client.ObjectKeyFromObject(group), group)))
			ctx.RouterGroups = nil
			ctx.ResourceManagerGroups = nil
			require.Equal(t, task.Complete().Break().With("removed finalizer"), tk.Sync(ctx))
			require.False(t, controllerutil.ContainsFinalizer(ctx.Cluster, meta.Finalizer))
		})
	}
}

type finalizerClient struct{ ctrlclient.WithWatch }

func (*finalizerClient) Apply(context.Context, client.Object, ...client.ApplyOption) error {
	panic("unexpected Apply")
}
func (*finalizerClient) ApplyWithResult(context.Context, client.Object, ...client.ApplyOption) (client.ApplyResult, error) {
	panic("unexpected ApplyWithResult")
}
