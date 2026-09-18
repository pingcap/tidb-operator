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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/features"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
)

func TestFeatureGateHash(t *testing.T) {
	for _, tc := range []struct {
		name      string
		previous  string
		updateErr error
		fails     bool
	}{
		{name: "initial recording"},
		{name: "adopt current definitions", previous: "old-hash"},
		{name: "already recorded avoids writes", previous: features.CurrentFeatureGateDefinitionHash, updateErr: errors.New("unexpected write")},
		{name: "write failure", updateErr: errors.New("unavailable"), fails: true},
		{name: "conflict", previous: "old-hash", updateErr: apierrors.NewConflict(schema.GroupResource{Resource: "clusters"}, "test", errors.New("conflict")), fails: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cluster := fake.FakeObj[v1alpha1.Cluster]("test")
			cluster.Status.FeatureGateHash = tc.previous
			cluster.Status.ID = "keep"
			fc := client.NewFakeClient(cluster)
			if tc.updateErr != nil {
				fc.WithError("update", "clusters", tc.updateErr)
			}
			ctx := &ReconcileContext{Context: context.Background(), Cluster: cluster.DeepCopy()}
			result := NewTaskFeatureGateHash(fc).Sync(ctx)
			assert.Equal(t, tc.fails, result.IsFailed())
			assert.Equal(t, !tc.fails, result.ShouldContinue())
			expected := features.CurrentFeatureGateDefinitionHash
			if tc.fails {
				expected = tc.previous
			}
			assert.Equal(t, expected, ctx.Cluster.Status.FeatureGateHash)
			stored := &v1alpha1.Cluster{}
			require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(cluster), stored))
			assert.Equal(t, expected, stored.Status.FeatureGateHash)
			assert.Equal(t, "keep", stored.Status.ID)
			assert.Equal(t, cluster.Spec, stored.Spec)
		})
	}
}

type pruningStatusClient struct{ client.Client }

func (c pruningStatusClient) Status() ctrlclient.SubResourceWriter {
	return pruningStatusWriter{SubResourceWriter: c.Client.Status()}
}

type pruningStatusWriter struct{ ctrlclient.SubResourceWriter }

func (pruningStatusWriter) Update(_ context.Context, obj ctrlclient.Object, _ ...ctrlclient.SubResourceUpdateOption) error {
	obj.(*v1alpha1.Cluster).Status.FeatureGateHash = ""
	return nil
}

func TestFeatureGateHashPruned(t *testing.T) {
	cluster := fake.FakeObj[v1alpha1.Cluster]("test")
	ctx := &ReconcileContext{Context: context.Background(), Cluster: cluster}
	fc := pruningStatusClient{Client: client.NewFakeClient(cluster)}
	result := NewTaskFeatureGateHash(fc).Sync(ctx)
	assert.True(t, result.IsFailed())
	assert.False(t, result.ShouldContinue())
	assert.Empty(t, ctx.Cluster.Status.FeatureGateHash)
}
