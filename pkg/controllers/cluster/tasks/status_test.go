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
	"strconv"
	"testing"

	"github.com/go-logr/logr"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/timanager"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task"
)

func TestStatusUpdater(t *testing.T) {
	cases := []struct {
		desc       string
		cluster    *v1alpha1.Cluster
		pdGroup    *v1alpha1.PDGroup
		expected   task.Result
		components []v1alpha1.ComponentStatus
		conditions []metav1.Condition
		clusterID  uint64
	}{
		{
			desc: "creating cluster",
			cluster: fake.FakeObj(
				"test",
				fake.SetGeneration[v1alpha1.Cluster](1),
			),
			pdGroup: fake.FakeObj(
				"pd-group",
				func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
					obj.Spec.Cluster.Name = "test"
					obj.Spec.Replicas = new(int32)
					*obj.Spec.Replicas = 3
					return obj
				},
			),
			expected: task.Complete().With("updated status"),
			components: []v1alpha1.ComponentStatus{
				{
					Kind:     v1alpha1.ComponentKindPD,
					Replicas: 3,
				},
			},
			conditions: []metav1.Condition{
				{
					Type:   v1alpha1.ClusterCondProgressing,
					Status: metav1.ConditionTrue,
				},
				{
					Type:   v1alpha1.ClusterCondAvailable,
					Status: metav1.ConditionFalse,
				},
				{
					Type:   v1alpha1.ClusterCondSuspended,
					Status: metav1.ConditionFalse,
				},
			},
			clusterID: 123,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctx := FakeContext(types.NamespacedName{Name: "test"})
			ctx.Cluster = c.cluster
			ctx.PDGroups = []*v1alpha1.PDGroup{c.pdGroup}

			fc := client.NewFakeClient(c.cluster, c.pdGroup)

			m := newFakePDClientManager(tt, fc, getCluster(ctx, &metapb.Cluster{
				Id: c.clusterID,
			}, nil))
			m.Start(ctx)
			require.NoError(tt, m.Register(ctx.Cluster))

			tk := NewTaskStatus(logr.Discard(), fc, m)
			res := tk.Sync(ctx)
			assert.Equal(tt, c.expected, res)
			assert.Equal(tt, c.cluster.Generation, c.cluster.Status.ObservedGeneration)
			assert.Equal(tt, c.components, c.cluster.Status.Components)

			conditions := make([]metav1.Condition, 0)
			for _, condition := range c.cluster.Status.Conditions {
				conditions = append(conditions, metav1.Condition{
					Type:   condition.Type,
					Status: condition.Status,
				})
			}
			assert.Equal(tt, c.conditions, conditions)
			assert.Equal(tt, strconv.FormatUint(c.clusterID, 10), c.cluster.Status.ID)
		})
	}
}

func TestSyncSuspendedConditionCoversAllGroupTypes(t *testing.T) {
	const (
		groupGeneration   = int64(7)
		clusterGeneration = int64(11)
	)

	newStatus := func() v1alpha1.CommonStatus {
		return v1alpha1.CommonStatus{
			ObservedGeneration:        groupGeneration,
			ObservedClusterGeneration: clusterGeneration,
			Conditions: []metav1.Condition{{
				Type:               v1alpha1.CondSuspended,
				Status:             metav1.ConditionTrue,
				ObservedGeneration: groupGeneration,
			}},
		}
	}
	objectMeta := func() metav1.ObjectMeta {
		return metav1.ObjectMeta{Generation: groupGeneration}
	}

	pdg := &v1alpha1.PDGroup{ObjectMeta: objectMeta(), Status: v1alpha1.PDGroupStatus{CommonStatus: newStatus()}}
	rmg := &v1alpha1.ResourceManagerGroup{ObjectMeta: objectMeta(), Status: v1alpha1.ResourceManagerGroupStatus{CommonStatus: newStatus()}}
	rg := &v1alpha1.RouterGroup{ObjectMeta: objectMeta(), Status: v1alpha1.RouterGroupStatus{CommonStatus: newStatus()}}
	tsog := &v1alpha1.TSOGroup{ObjectMeta: objectMeta(), Status: v1alpha1.TSOGroupStatus{CommonStatus: newStatus()}}
	schedulingg := &v1alpha1.SchedulingGroup{ObjectMeta: objectMeta(), Status: v1alpha1.SchedulingGroupStatus{CommonStatus: newStatus()}}
	schedulerg := &v1alpha1.SchedulerGroup{ObjectMeta: objectMeta(), Status: v1alpha1.SchedulerGroupStatus{CommonStatus: newStatus()}}
	tikvg := &v1alpha1.TiKVGroup{ObjectMeta: objectMeta(), Status: v1alpha1.TiKVGroupStatus{CommonStatus: newStatus()}}
	tiflashg := &v1alpha1.TiFlashGroup{ObjectMeta: objectMeta(), Status: v1alpha1.TiFlashGroupStatus{CommonStatus: newStatus()}}
	tidbg := &v1alpha1.TiDBGroup{ObjectMeta: objectMeta(), Status: v1alpha1.TiDBGroupStatus{CommonStatus: newStatus()}}
	ticdcg := &v1alpha1.TiCDCGroup{ObjectMeta: objectMeta(), Status: v1alpha1.TiCDCGroupStatus{CommonStatus: newStatus()}}
	tiproxyg := &v1alpha1.TiProxyGroup{ObjectMeta: objectMeta(), Status: v1alpha1.TiProxyGroupStatus{CommonStatus: newStatus()}}
	tikvworkerg := &v1alpha1.TiKVWorkerGroup{ObjectMeta: objectMeta(), Status: v1alpha1.TiKVWorkerGroupStatus{CommonStatus: newStatus()}}
	dmg := &v1alpha1.DMGroup{ObjectMeta: objectMeta(), Status: v1alpha1.DMGroupStatus{CommonStatus: newStatus()}}
	dmworkerg := &v1alpha1.DMWorkerGroup{ObjectMeta: objectMeta(), Status: v1alpha1.DMWorkerGroupStatus{CommonStatus: newStatus()}}

	cluster := &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Generation: clusterGeneration},
		Spec: v1alpha1.ClusterSpec{SuspendAction: &v1alpha1.SuspendAction{
			SuspendCompute: true,
		}},
	}
	rtx := &ReconcileContext{
		Cluster:               cluster,
		PDGroups:              []*v1alpha1.PDGroup{pdg},
		ResourceManagerGroups: []*v1alpha1.ResourceManagerGroup{rmg},
		RouterGroups:          []*v1alpha1.RouterGroup{rg},
		TSOGroups:             []*v1alpha1.TSOGroup{tsog},
		SchedulingGroups:      []*v1alpha1.SchedulingGroup{schedulingg},
		SchedulerGroups:       []*v1alpha1.SchedulerGroup{schedulerg},
		TiKVGroups:            []*v1alpha1.TiKVGroup{tikvg},
		TiFlashGroups:         []*v1alpha1.TiFlashGroup{tiflashg},
		TiDBGroups:            []*v1alpha1.TiDBGroup{tidbg},
		TiCDCGroups:           []*v1alpha1.TiCDCGroup{ticdcg},
		TiProxyGroups:         []*v1alpha1.TiProxyGroup{tiproxyg},
		TiKVWorkerGroups:      []*v1alpha1.TiKVWorkerGroup{tikvworkerg},
		DMGroups:              []*v1alpha1.DMGroup{dmg},
		DMWorkerGroups:        []*v1alpha1.DMWorkerGroup{dmworkerg},
	}

	statusByType := map[string]*v1alpha1.CommonStatus{
		"PDGroup":              &pdg.Status.CommonStatus,
		"ResourceManagerGroup": &rmg.Status.CommonStatus,
		"RouterGroup":          &rg.Status.CommonStatus,
		"TSOGroup":             &tsog.Status.CommonStatus,
		"SchedulingGroup":      &schedulingg.Status.CommonStatus,
		"SchedulerGroup":       &schedulerg.Status.CommonStatus,
		"TiKVGroup":            &tikvg.Status.CommonStatus,
		"TiFlashGroup":         &tiflashg.Status.CommonStatus,
		"TiDBGroup":            &tidbg.Status.CommonStatus,
		"TiCDCGroup":           &ticdcg.Status.CommonStatus,
		"TiProxyGroup":         &tiproxyg.Status.CommonStatus,
		"TiKVWorkerGroup":      &tikvworkerg.Status.CommonStatus,
		"DMGroup":              &dmg.Status.CommonStatus,
		"DMWorkerGroup":        &dmworkerg.Status.CommonStatus,
	}

	taskStatus := &TaskStatus{}
	taskStatus.syncConditions(rtx)
	require.True(t, meta.IsStatusConditionTrue(cluster.Status.Conditions, v1alpha1.ClusterCondSuspended))
	suspendedCondition := meta.FindStatusCondition(cluster.Status.Conditions, v1alpha1.ClusterCondSuspended)
	require.NotNil(t, suspendedCondition)
	assert.Equal(t, clusterGeneration, suspendedCondition.ObservedGeneration)

	for groupType, status := range statusByType {
		t.Run(groupType+" stale", func(t *testing.T) {
			status.ObservedClusterGeneration--
			cluster.Status.Conditions = nil
			taskStatus.syncConditions(rtx)
			assert.False(t, meta.IsStatusConditionTrue(cluster.Status.Conditions, v1alpha1.ClusterCondSuspended))
			status.ObservedClusterGeneration++
		})
	}
}

func newFakePDClientManager(t *testing.T, c client.Client, acts ...action) pdm.PDClientManager {
	return timanager.NewManagerBuilder[*v1alpha1.Cluster, pdapi.PDClient, pdm.PDClient]().
		WithNewUnderlayClientFunc(func(*v1alpha1.Cluster) (pdapi.PDClient, error) {
			return nil, nil
		}).
		WithNewClientFunc(func(*v1alpha1.Cluster, pdapi.PDClient, timanager.SharedInformerFactory[pdapi.PDClient]) (pdm.PDClient, error) {
			return NewFakePDClient(t, acts...), nil
		}).
		WithCacheKeysFunc(pdm.CacheKeysFunc(c)).
		Build()
}

func NewFakePDClient(t *testing.T, acts ...action) pdm.PDClient {
	ctrl := gomock.NewController(t)
	pdc := pdm.NewMockPDClient(ctrl)
	for _, act := range acts {
		act(ctrl, pdc)
	}

	return pdc
}

type action func(ctrl *gomock.Controller, pdc *pdm.MockPDClient)

func getCluster(ctx context.Context, cluster *metapb.Cluster, err error) action {
	return func(ctrl *gomock.Controller, pdc *pdm.MockPDClient) {
		underlay := pdapi.NewMockPDClient(ctrl)
		pdc.EXPECT().Underlay().Return(underlay).AnyTimes()
		underlay.EXPECT().GetCluster(ctx).Return(cluster, err).AnyTimes()
	}
}
