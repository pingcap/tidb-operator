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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/pkg/timanager"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/task"
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
			ctx.PDGroup = c.pdGroup

			m := newFakePDClientManager(tt, getCluster(ctx, &metapb.Cluster{
				Id: c.clusterID,
			}, nil))
			m.Start(ctx)
			require.NoError(tt, m.Register(ctx.PDGroup))

			fc := client.NewFakeClient(c.cluster)
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

func newFakePDClientManager(t *testing.T, acts ...action) pdm.PDClientManager {
	return timanager.NewManagerBuilder[*v1alpha1.PDGroup, pdapi.PDClient, pdm.PDClient]().
		WithNewUnderlayClientFunc(func(*v1alpha1.PDGroup) (pdapi.PDClient, error) {
			return nil, nil
		}).
		WithNewClientFunc(func(string, pdapi.PDClient, timanager.SharedInformerFactory[pdapi.PDClient]) pdm.PDClient {
			return NewFakePDClient(t, acts...)
		}).
		WithCacheKeysFunc(pdm.CacheKeys).
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
