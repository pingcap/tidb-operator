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

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/adoption"
	operatorclient "github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/tidbapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/tracker"
)

func TestClassifySmoothUpgrade(t *testing.T) {
	cases := []struct {
		source string
		target string
		want   smoothUpgradeSupport
	}{
		{"v7.1.0", "v7.1.1", smoothUpgradeAutoSupported},
		{"v7.1.1", "v7.3.0", smoothUpgradeAutoSupported},
		{"v7.2.0", "v7.3.0", smoothUpgradeAutoSupported},
		{"v7.1.2", "v7.1.3", smoothUpgradeSwitchControlled},
		{"v7.1.3", "v7.4.0", smoothUpgradeSwitchControlled},
		{"v7.4.0", "v7.5.0", smoothUpgradeSwitchControlled},
		{"v6.5.10", "v8.1.0", smoothUpgradeUnsupported},
		{"v7.3.0", "v7.4.0", smoothUpgradeUnsupported},
		{"v7.5.0", "v7.4.0", smoothUpgradeUnsupported},
		{"v7.5.0", "v7.5.0", smoothUpgradeUnsupported},
		{"nightly", "v7.5.0", smoothUpgradeUnsupported},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, classifySmoothUpgrade(c.source, c.target), "%s -> %s", c.source, c.target)
	}
}

func TestEnsureSmoothUpgradeStarted(t *testing.T) {
	oldFactory := newSmoothUpgradeTiDBClient
	defer func() { newSmoothUpgradeTiDBClient = oldFactory }()

	ctx := context.Background()
	mock := &mockTiDBClient{}
	newSmoothUpgradeTiDBClient = func(context.Context, operatorclient.Client, *v1alpha1.Cluster, *v1alpha1.TiDB) (tidbapi.TiDBClient, error) {
		return mock, nil
	}

	dbg := smoothUpgradeTiDBGroup("db", "v7.5.0", "v7.5.3")
	db := smoothUpgradeTiDB("db-a", "v7.5.0", oldRevision)
	state := &ReconcileContext{State: &state{
		dbg:            dbg,
		cluster:        fake.FakeObj[v1alpha1.Cluster]("cluster"),
		dbs:            []*v1alpha1.TiDB{db},
		updateRevision: newRevision,
	}}
	fc := operatorclient.NewFakeClient(dbg, state.Cluster())

	res := ensureSmoothUpgradeStarted(ctx, state, fc)
	require.Equal(t, task.SComplete, res.Status())
	assert.Equal(t, 1, mock.startCalls)
	assert.Equal(t, v1alpha1.AnnoValTrue, dbg.Annotations[annSmoothUpgradeDDLPaused])
	assert.Equal(t, "v7.5.0", dbg.Annotations[annSmoothUpgradeSourceVersion])
	assert.Equal(t, "v7.5.3", dbg.Annotations[annSmoothUpgradeTargetVersion])
}

func TestEnsureSmoothUpgradeStartedSkipsActiveAnnotation(t *testing.T) {
	oldFactory := newSmoothUpgradeTiDBClient
	defer func() { newSmoothUpgradeTiDBClient = oldFactory }()

	mock := &mockTiDBClient{}
	newSmoothUpgradeTiDBClient = func(context.Context, operatorclient.Client, *v1alpha1.Cluster, *v1alpha1.TiDB) (tidbapi.TiDBClient, error) {
		return mock, nil
	}

	dbg := smoothUpgradeTiDBGroup("db", "v7.5.0", "v7.5.3")
	setSmoothUpgradeAnnotations(dbg, "v7.5.0", "v7.5.3")
	state := &ReconcileContext{State: &state{
		dbg:            dbg,
		cluster:        fake.FakeObj[v1alpha1.Cluster]("cluster"),
		dbs:            []*v1alpha1.TiDB{smoothUpgradeTiDB("db-a", "v7.5.0", oldRevision)},
		updateRevision: newRevision,
	}}

	res := ensureSmoothUpgradeStarted(context.Background(), state, operatorclient.NewFakeClient(dbg, state.Cluster()))
	require.Equal(t, task.SComplete, res.Status())
	assert.Zero(t, mock.startCalls)
	assert.Zero(t, mock.finishCalls)
}

func TestEnsureSmoothUpgradeStartedRecoversStaleAnnotation(t *testing.T) {
	oldFactory := newSmoothUpgradeTiDBClient
	defer func() { newSmoothUpgradeTiDBClient = oldFactory }()

	mock := &mockTiDBClient{}
	newSmoothUpgradeTiDBClient = func(context.Context, operatorclient.Client, *v1alpha1.Cluster, *v1alpha1.TiDB) (tidbapi.TiDBClient, error) {
		return mock, nil
	}

	dbg := smoothUpgradeTiDBGroup("db", "v7.5.0", "v7.5.3")
	setSmoothUpgradeAnnotations(dbg, "v7.5.0", "v7.5.1")
	state := &ReconcileContext{State: &state{
		dbg:            dbg,
		cluster:        fake.FakeObj[v1alpha1.Cluster]("cluster"),
		dbs:            []*v1alpha1.TiDB{smoothUpgradeTiDB("db-a", "v7.5.0", oldRevision)},
		updateRevision: newRevision,
	}}

	res := ensureSmoothUpgradeStarted(context.Background(), state, operatorclient.NewFakeClient(dbg, state.Cluster()))
	require.Equal(t, task.SRetry, res.Status())
	assert.Zero(t, mock.startCalls)
	assert.Equal(t, 1, mock.finishCalls)
	assert.NotContains(t, dbg.Annotations, annSmoothUpgradeDDLPaused)
}

func TestEnsureSmoothUpgradeStartedUnsupportedPairSkipsHTTP(t *testing.T) {
	oldFactory := newSmoothUpgradeTiDBClient
	defer func() { newSmoothUpgradeTiDBClient = oldFactory }()

	mock := &mockTiDBClient{}
	newSmoothUpgradeTiDBClient = func(context.Context, operatorclient.Client, *v1alpha1.Cluster, *v1alpha1.TiDB) (tidbapi.TiDBClient, error) {
		return mock, nil
	}

	dbg := smoothUpgradeTiDBGroup("db", "v6.5.10", "v8.1.0")
	state := &ReconcileContext{State: &state{
		dbg:            dbg,
		cluster:        fake.FakeObj[v1alpha1.Cluster]("cluster"),
		dbs:            []*v1alpha1.TiDB{smoothUpgradeTiDB("db-a", "v6.5.10", oldRevision)},
		updateRevision: newRevision,
	}}

	res := ensureSmoothUpgradeStarted(context.Background(), state, operatorclient.NewFakeClient(dbg, state.Cluster()))
	require.Equal(t, task.SComplete, res.Status())
	assert.Zero(t, mock.startCalls)
	assert.Zero(t, mock.finishCalls)
	assert.NotContains(t, dbg.Annotations, annSmoothUpgradeDDLPaused)
}

func TestTaskFinishSmoothUpgrade(t *testing.T) {
	oldFactory := newSmoothUpgradeTiDBClient
	defer func() { newSmoothUpgradeTiDBClient = oldFactory }()

	mock := &mockTiDBClient{}
	newSmoothUpgradeTiDBClient = func(context.Context, operatorclient.Client, *v1alpha1.Cluster, *v1alpha1.TiDB) (tidbapi.TiDBClient, error) {
		return mock, nil
	}

	dbg := smoothUpgradeTiDBGroup("db", "v7.5.0", "v7.5.3")
	setSmoothUpgradeAnnotations(dbg, "v7.5.0", "v7.5.3")
	dbg.Status.Version = "v7.5.3"
	dbg.Status.Replicas = 1
	dbg.Status.ReadyReplicas = 1
	dbg.Status.UpdatedReplicas = 1
	dbg.Status.CurrentReplicas = 1
	dbg.Status.UpdateRevision = newRevision
	dbg.Status.CurrentRevision = newRevision
	db := smoothUpgradeTiDB("db-a", "v7.5.3", newRevision)
	state := &ReconcileContext{State: &state{
		dbg:            dbg,
		cluster:        fake.FakeObj[v1alpha1.Cluster]("cluster"),
		dbs:            []*v1alpha1.TiDB{db},
		updateRevision: newRevision,
	}}

	res, done := task.RunTask(context.Background(), TaskFinishSmoothUpgrade(state, operatorclient.NewFakeClient(dbg, state.Cluster())))
	require.False(t, done)
	require.Equal(t, task.SComplete, res.Status())
	assert.Equal(t, 1, mock.finishCalls)
	assert.NotContains(t, dbg.Annotations, annSmoothUpgradeDDLPaused)
}

func TestTaskFinishSmoothUpgradeKeepsAnnotationOnFailure(t *testing.T) {
	oldFactory := newSmoothUpgradeTiDBClient
	defer func() { newSmoothUpgradeTiDBClient = oldFactory }()

	mock := &mockTiDBClient{finishErr: errors.New("boom")}
	newSmoothUpgradeTiDBClient = func(context.Context, operatorclient.Client, *v1alpha1.Cluster, *v1alpha1.TiDB) (tidbapi.TiDBClient, error) {
		return mock, nil
	}

	dbg := smoothUpgradeTiDBGroup("db", "v7.5.0", "v7.5.3")
	setSmoothUpgradeAnnotations(dbg, "v7.5.0", "v7.5.3")
	dbg.Status.Version = "v7.5.3"
	dbg.Status.Replicas = 1
	dbg.Status.ReadyReplicas = 1
	dbg.Status.UpdatedReplicas = 1
	dbg.Status.CurrentReplicas = 1
	dbg.Status.UpdateRevision = newRevision
	dbg.Status.CurrentRevision = newRevision
	state := &ReconcileContext{State: &state{
		dbg:            dbg,
		cluster:        fake.FakeObj[v1alpha1.Cluster]("cluster"),
		dbs:            []*v1alpha1.TiDB{smoothUpgradeTiDB("db-a", "v7.5.3", newRevision)},
		updateRevision: newRevision,
	}}

	res, _ := task.RunTask(context.Background(), TaskFinishSmoothUpgrade(state, operatorclient.NewFakeClient(dbg, state.Cluster())))
	require.Equal(t, task.SFail, res.Status())
	assert.Equal(t, v1alpha1.AnnoValTrue, dbg.Annotations[annSmoothUpgradeDDLPaused])
}

func TestTaskUpdaterSmoothUpgradeStartFailureBlocksRollout(t *testing.T) {
	oldFactory := newSmoothUpgradeTiDBClient
	defer func() { newSmoothUpgradeTiDBClient = oldFactory }()

	mock := &mockTiDBClient{startErr: errors.New("boom")}
	newSmoothUpgradeTiDBClient = func(context.Context, operatorclient.Client, *v1alpha1.Cluster, *v1alpha1.TiDB) (tidbapi.TiDBClient, error) {
		return mock, nil
	}

	dbg := smoothUpgradeTiDBGroup("db", "v7.5.0", "v7.5.3")
	db := smoothUpgradeTiDB("db-a", "v7.5.0", oldRevision)
	state := &ReconcileContext{State: &state{
		dbg:            dbg,
		cluster:        fake.FakeObj[v1alpha1.Cluster]("cluster"),
		dbs:            []*v1alpha1.TiDB{db},
		updateRevision: newRevision,
	}}
	fc := operatorclient.NewFakeClient(dbg, state.Cluster(), db)

	res, _ := task.RunTask(context.Background(), TaskUpdater(state, fc, tracker.New().AllocateFactory("tidb"), adoption.New(logr.Discard())))
	require.Equal(t, task.SFail, res.Status())
	assert.Equal(t, 1, mock.startCalls)

	var dbs v1alpha1.TiDBList
	require.NoError(t, fc.List(context.Background(), &dbs))
	assert.Len(t, dbs.Items, 1)
	assert.NotContains(t, dbg.Annotations, annSmoothUpgradeDDLPaused)
}

func smoothUpgradeTiDBGroup(name, source, target string) *v1alpha1.TiDBGroup {
	return fake.FakeObj(name, func(obj *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
		obj.Spec.Cluster.Name = "cluster"
		obj.Spec.Replicas = ptr.To[int32](1)
		obj.Spec.Template.Spec.Version = target
		obj.Status.Version = source
		return obj
	})
}

func smoothUpgradeTiDB(name, version, revision string) *v1alpha1.TiDB {
	return fake.FakeObj(name, func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
		obj.Spec.Version = version
		obj.Status.CurrentRevision = revision
		obj.Status.Conditions = append(obj.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.CondReady,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Unix(0, 0),
		})
		return obj
	})
}

type mockTiDBClient struct {
	startCalls  int
	finishCalls int
	startErr    error
	finishErr   error
}

func (m *mockTiDBClient) GetHealth(context.Context) (bool, error) {
	return true, nil
}

func (m *mockTiDBClient) GetInfo(context.Context) (*tidbapi.ServerInfo, error) {
	return nil, nil
}

func (m *mockTiDBClient) SetServerLabels(context.Context, map[string]string) error {
	return nil
}

func (m *mockTiDBClient) GetPoolStatus(context.Context) (*tidbapi.PoolStatus, error) {
	return nil, nil
}

func (m *mockTiDBClient) Activate(context.Context, string) error {
	return nil
}

func (m *mockTiDBClient) StartUpgrade(context.Context) error {
	m.startCalls++
	return m.startErr
}

func (m *mockTiDBClient) FinishUpgrade(context.Context) error {
	m.finishCalls++
	return m.finishErr
}
