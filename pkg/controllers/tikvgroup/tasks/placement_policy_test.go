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
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TestTaskPlacementPolicyRefBlock(t *testing.T) {
	ctx := context.Background()
	kvg := newPlacementPolicyBlockTiKVGroup("g1", "c")
	ref := newPlacementPolicyForTiKVGroup("p1", "c", "g1")
	otherRef := newPlacementPolicyForTiKVGroup("p2", "c", "g2")
	crossClusterRef := newPlacementPolicyForTiKVGroup("p3", "other", "g1")
	invalidRef := newPlacementPolicyForTiKVGroup("p4", "c", "g1")
	invalidRef.Spec.GroupRefs[0].Kind = "PDGroup"
	s := &ReconcileContext{
		State: &state{
			key: types.NamespacedName{Namespace: kvg.Namespace, Name: kvg.Name},
			kvg: kvg,
		},
	}
	fc := client.NewFakeClient(kvg, ref, otherRef, crossClusterRef, invalidRef)

	res, done := task.RunTask(ctx, TaskPlacementPolicyRefBlock(s, fc))

	require.Equal(t, task.SWait, res.Status())
	assert.False(t, done)
	assert.Contains(t, res.Message(), "[p1]")
}

func TestTaskPlacementPolicyRefBlockCompletesWhenClusterDeleting(t *testing.T) {
	ctx := context.Background()
	kvg := newPlacementPolicyBlockTiKVGroup("g1", "c")
	policy := newPlacementPolicyForTiKVGroup("p1", "c", "g1")
	s := &ReconcileContext{
		State: &state{
			key:     types.NamespacedName{Namespace: kvg.Namespace, Name: kvg.Name},
			cluster: newDeletingPlacementPolicyBlockCluster("c"),
			kvg:     kvg,
		},
	}
	fc := client.NewFakeClient(kvg, policy)

	res, done := task.RunTask(ctx, TaskPlacementPolicyRefBlock(s, fc))

	require.Equal(t, task.SComplete, res.Status())
	assert.False(t, done)
}

func TestTaskPlacementPolicyRefBlockCompletesWithoutRefs(t *testing.T) {
	ctx := context.Background()
	kvg := newPlacementPolicyBlockTiKVGroup("g1", "c")
	policy := newPlacementPolicyForTiKVGroup("p1", "c", "g2")
	s := &ReconcileContext{
		State: &state{
			key: types.NamespacedName{Namespace: kvg.Namespace, Name: kvg.Name},
			kvg: kvg,
		},
	}
	fc := client.NewFakeClient(kvg, policy)

	res, done := task.RunTask(ctx, TaskPlacementPolicyRefBlock(s, fc))

	require.Equal(t, task.SComplete, res.Status())
	assert.False(t, done)
}

func TestPlacementPolicyRefBlockStopsFinalizerDeletion(t *testing.T) {
	ctx := context.Background()
	kvg := newDeletingPlacementPolicyBlockTiKVGroup("g1", "c")
	kv := &v1alpha1.TiKV{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: kvg.Namespace,
			Name:      "g1-0",
		},
	}
	policy := newPlacementPolicyForTiKVGroup("p1", "c", "g1")
	s := &ReconcileContext{
		State: &state{
			key: types.NamespacedName{Namespace: kvg.Namespace, Name: kvg.Name},
			kvg: kvg,
			kvs: []*v1alpha1.TiKV{kv},
		},
	}
	fc := client.NewFakeClient(kvg, kv, policy)

	res, done := task.RunTask(ctx, task.Block(
		task.BreakOnWait(TaskPlacementPolicyRefBlock(s, fc)),
		common.TaskGroupFinalizerDel[scope.TiKVGroup](s, fc),
	))

	require.Equal(t, task.SWait, res.Status())
	assert.True(t, done)

	var actual v1alpha1.TiKV
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(kv), &actual))
	assert.True(t, actual.DeletionTimestamp.IsZero())
}

func newPlacementPolicyBlockTiKVGroup(name, clusterName string) *v1alpha1.TiKVGroup {
	return &v1alpha1.TiKVGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      name,
		},
		Spec: v1alpha1.TiKVGroupSpec{
			Cluster: v1alpha1.ClusterReference{Name: clusterName},
		},
	}
}

func newDeletingPlacementPolicyBlockCluster(name string) *v1alpha1.Cluster {
	cluster := &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      name,
		},
	}
	now := metav1.Now()
	cluster.DeletionTimestamp = &now
	return cluster
}

func newDeletingPlacementPolicyBlockTiKVGroup(name, clusterName string) *v1alpha1.TiKVGroup {
	kvg := newPlacementPolicyBlockTiKVGroup(name, clusterName)
	now := metav1.Now()
	kvg.DeletionTimestamp = &now
	kvg.Finalizers = []string{metav1alpha1.Finalizer}
	return kvg
}

func newPlacementPolicyForTiKVGroup(name, clusterName string, groups ...string) *v1alpha1.PlacementPolicy {
	refs := make([]v1alpha1.PlacementPolicyGroupRef, 0, len(groups))
	for _, group := range groups {
		refs = append(refs, v1alpha1.PlacementPolicyGroupRef{
			Group: v1alpha1.GroupName,
			Kind:  "TiKVGroup",
			Name:  group,
		})
	}

	return &v1alpha1.PlacementPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      name,
		},
		Spec: v1alpha1.PlacementPolicySpec{
			Cluster:   v1alpha1.ClusterReference{Name: clusterName},
			GroupRefs: refs,
		},
	}
}
