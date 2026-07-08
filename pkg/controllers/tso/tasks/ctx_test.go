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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/timanager"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	tsom "github.com/pingcap/tidb-operator/v2/pkg/timanager/tso"
	"github.com/pingcap/tidb-operator/v2/pkg/tsoapi"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TestTaskContextClient(t *testing.T) {
	ctx := context.Background()

	t.Run("synced pd client without tso member cache is incomplete context", func(t *testing.T) {
		cluster := fake.FakeObj("cluster", fake.SetNamespace[v1alpha1.Cluster]("ns"))
		tso := fake.FakeObj("tso-0",
			fake.SetNamespace[v1alpha1.TSO]("ns"),
			func(tso *v1alpha1.TSO) *v1alpha1.TSO {
				tso.Spec.Cluster.Name = cluster.Name
				return tso
			},
		)
		tg := fake.FakeObj("tso",
			fake.SetNamespace[v1alpha1.TSOGroup]("ns"),
			func(tg *v1alpha1.TSOGroup) *v1alpha1.TSOGroup {
				tg.Spec.Cluster.Name = cluster.Name
				return tg
			},
		)
		state := &ReconcileContext{
			State: &state{
				cluster: cluster,
				tso:     tso,
			},
		}
		pdcm := newTestPDClientManager(t, cluster, &fakePDClient{
			hasSynced:  true,
			tsoMembers: nil,
		})
		tsocm := newTestTSOClientManager(t, tg, fakeTSOClient{})

		res, _ := task.RunTask(ctx, TaskContextClient(state, pdcm, tsocm))

		assert.Equal(t, task.SComplete.String(), res.Status().String())
		assert.NotNil(t, state.TSOClient)
		assert.Nil(t, state.PDClient)
		assert.False(t, state.CacheSynced)
		assert.False(t, state.TSOMemberNotFound)
	})
}

type fakePDClient struct {
	hasSynced  bool
	tsoMembers pdm.TSOMemberCache
}

func (f *fakePDClient) HasSynced() bool {
	return f.hasSynced
}

func (f *fakePDClient) MembersSynced() bool {
	return f.hasSynced
}

func (*fakePDClient) Stores() pdm.StoreCache {
	return nil
}

func (*fakePDClient) Members() pdm.MemberCache {
	return nil
}

func (f *fakePDClient) TSOMembers() pdm.TSOMemberCache {
	return f.tsoMembers
}

func (*fakePDClient) Underlay() pdapi.PDClient {
	return nil
}

type fakeTSOClient struct{}

func (fakeTSOClient) Underlay() tsoapi.TSOClient {
	return nil
}

func newTestPDClientManager(t *testing.T, cluster *v1alpha1.Cluster, pdc pdm.PDClient) pdm.PDClientManager {
	t.Helper()

	m := timanager.NewManagerBuilder[*v1alpha1.Cluster, struct{}, pdm.PDClient]().
		WithLogger(logr.Discard()).
		WithNewUnderlayClientFunc(func(*v1alpha1.Cluster) (struct{}, error) {
			return struct{}{}, nil
		}).
		WithNewClientFunc(func(*v1alpha1.Cluster, struct{}, timanager.SharedInformerFactory[struct{}]) (pdm.PDClient, error) {
			return pdc, nil
		}).
		WithCacheKeysFunc(func(cluster *v1alpha1.Cluster) ([]string, error) {
			return []string{timanager.PrimaryKey(cluster.Namespace, cluster.Name)}, nil
		}).
		Build()
	m.Start(context.Background())
	require.NoError(t, m.Register(cluster))
	return m
}

func newTestTSOClientManager(t *testing.T, tg *v1alpha1.TSOGroup, tsoc tsom.TSOClient) tsom.TSOClientManager {
	t.Helper()

	m := timanager.NewManagerBuilder[*v1alpha1.TSOGroup, struct{}, tsom.TSOClient]().
		WithLogger(logr.Discard()).
		WithNewUnderlayClientFunc(func(*v1alpha1.TSOGroup) (struct{}, error) {
			return struct{}{}, nil
		}).
		WithNewClientFunc(func(*v1alpha1.TSOGroup, struct{}, timanager.SharedInformerFactory[struct{}]) (tsom.TSOClient, error) {
			return tsoc, nil
		}).
		WithCacheKeysFunc(func(tg *v1alpha1.TSOGroup) ([]string, error) {
			return []string{timanager.PrimaryKey(tg.Namespace, tg.Spec.Cluster.Name)}, nil
		}).
		Build()
	m.Start(context.Background())
	require.NoError(t, m.Register(tg))
	return m
}
