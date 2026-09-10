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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	tidbapi "github.com/pingcap/tidb-operator/v2/pkg/tidbapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

// fakeTiDBClient is a minimal tidbapi.TiDBClient for upgrade task tests.
type fakeTiDBClient struct {
	upgradeStartCalled bool
	upgradeStartErr    error
	upgradeStartKS     string

	upgradeFinishCalled bool
	upgradeFinishErr    error
}

func (f *fakeTiDBClient) GetHealth(_ context.Context) (bool, error) { return true, nil }
func (f *fakeTiDBClient) GetInfo(_ context.Context) (*tidbapi.ServerInfo, error) {
	return &tidbapi.ServerInfo{}, nil
}
func (f *fakeTiDBClient) SetServerLabels(_ context.Context, _ map[string]string) error { return nil }
func (f *fakeTiDBClient) GetPoolStatus(_ context.Context) (*tidbapi.PoolStatus, error) {
	return &tidbapi.PoolStatus{State: tidbapi.PoolStateActivated}, nil
}
func (f *fakeTiDBClient) Activate(_ context.Context, _ string) error { return nil }

func (f *fakeTiDBClient) UpgradeStart(_ context.Context, keyspace string) error {
	f.upgradeStartCalled = true
	f.upgradeStartKS = keyspace
	return f.upgradeStartErr
}

func (f *fakeTiDBClient) UpgradeFinish(_ context.Context) error {
	f.upgradeFinishCalled = true
	return f.upgradeFinishErr
}

var _ tidbapi.TiDBClient = (*fakeTiDBClient)(nil)

// fakeReadyTiDB creates a TiDB instance with the Ready condition set to true.
func fakeReadyTiDB() *v1alpha1.TiDB {
	return fake.FakeObj("tidb-0", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
		obj.Status.Conditions = append(obj.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.CondReady,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Unix(0, 0),
		})
		return obj
	})
}

func TestTaskSmoothUpgradeStart(t *testing.T) {
	cases := []struct {
		desc           string
		dbg            *v1alpha1.TiDBGroup
		dbs            []*v1alpha1.TiDB
		apiErr         error
		expectedStatus task.Status
		wantAPICalled  bool
		wantKeyspace   string
		wantAnnotation bool
	}{
		{
			desc: "scale only — no version upgrade",
			dbg: fake.FakeObj("dbg", func(o *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
				o.Spec.Template.Spec.Version = "v8.0.0"
				o.Status.Version = "v8.0.0"
				return o
			}),
			dbs:            []*v1alpha1.TiDB{fakeReadyTiDB()},
			expectedStatus: task.SComplete,
			wantAPICalled:  false,
		},
		{
			desc: "source version < v7.5, skip",
			dbg: fake.FakeObj("dbg", func(o *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
				o.Spec.Template.Spec.Version = "v7.5.0"
				o.Status.Version = "v7.4.0"
				return o
			}),
			dbs:            []*v1alpha1.TiDB{fakeReadyTiDB()},
			expectedStatus: task.SComplete,
			wantAPICalled:  false,
		},
		{
			desc: "target version < v7.5, skip",
			dbg: fake.FakeObj("dbg", func(o *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
				o.Spec.Template.Spec.Version = "v7.4.0"
				o.Status.Version = "v7.5.0"
				return o
			}),
			dbs:            []*v1alpha1.TiDB{fakeReadyTiDB()},
			expectedStatus: task.SComplete,
			wantAPICalled:  false,
		},
		{
			desc: "annotation already set — idempotent",
			dbg: fake.FakeObj("dbg", func(o *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
				o.Spec.Template.Spec.Version = "v8.0.0"
				o.Status.Version = "v7.5.0"
				o.Annotations = map[string]string{
					v1alpha1.AnnoKeySmoothUpgradePhase: v1alpha1.AnnoValSmoothUpgradePhaseInProgress,
				}
				return o
			}),
			dbs:            []*v1alpha1.TiDB{fakeReadyTiDB()},
			expectedStatus: task.SComplete,
			wantAPICalled:  false,
		},
		{
			desc: "no ready instance — retry",
			dbg: fake.FakeObj("dbg", func(o *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
				o.Spec.Template.Spec.Version = "v8.0.0"
				o.Status.Version = "v7.5.0"
				return o
			}),
			dbs:            []*v1alpha1.TiDB{},
			expectedStatus: task.SRetry,
			wantAPICalled:  false,
		},
		{
			desc: "API error — retry",
			dbg: fake.FakeObj("dbg", func(o *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
				o.Spec.Template.Spec.Version = "v8.0.0"
				o.Status.Version = "v7.5.0"
				return o
			}),
			dbs:            []*v1alpha1.TiDB{fakeReadyTiDB()},
			apiErr:         fmt.Errorf("connection refused"),
			expectedStatus: task.SRetry,
			wantAPICalled:  true,
		},
		{
			desc: "Dedicated (empty keyspace) — success",
			dbg: fake.FakeObj("dbg", func(o *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
				o.Spec.Template.Spec.Version = "v8.0.0"
				o.Status.Version = "v7.5.0"
				return o
			}),
			dbs:            []*v1alpha1.TiDB{fakeReadyTiDB()},
			expectedStatus: task.SComplete,
			wantAPICalled:  true,
			wantKeyspace:   "",
			wantAnnotation: true,
		},
		{
			desc: "Premium tenant keyspace — success",
			dbg: fake.FakeObj("dbg", func(o *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
				o.Spec.Template.Spec.Version = "v8.0.0"
				o.Status.Version = "v7.5.0"
				o.Spec.Template.Spec.Keyspace = "tenant1"
				return o
			}),
			dbs:            []*v1alpha1.TiDB{fakeReadyTiDB()},
			expectedStatus: task.SComplete,
			wantAPICalled:  true,
			wantKeyspace:   "tenant1",
			wantAnnotation: true,
		},
		{
			desc: "Premium system keyspace — success",
			dbg: fake.FakeObj("dbg", func(o *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
				o.Spec.Template.Spec.Version = "v8.0.0"
				o.Status.Version = "v7.5.0"
				o.Spec.Template.Spec.Keyspace = "system"
				return o
			}),
			dbs:            []*v1alpha1.TiDB{fakeReadyTiDB()},
			expectedStatus: task.SComplete,
			wantAPICalled:  true,
			wantKeyspace:   "system",
			wantAnnotation: true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fakeClient := &fakeTiDBClient{upgradeStartErr: c.apiErr}

			factory := func(_ context.Context, _ client.Client, _ *v1alpha1.Cluster, _ *v1alpha1.TiDB) (tidbapi.TiDBClient, error) {
				return fakeClient, nil
			}

			ctx := context.Background()
			fc := client.NewFakeClient(c.dbg, fake.FakeObj[v1alpha1.Cluster]("cluster"))

			rc := &ReconcileContext{
				State: &state{
					dbg:     c.dbg,
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					dbs:     c.dbs,
				},
			}

			res, done := task.RunTask(ctx, taskSmoothUpgradeStart(rc, fc, factory))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)
			assert.Equal(tt, c.wantAPICalled, fakeClient.upgradeStartCalled, c.desc)
			if c.wantAPICalled && c.apiErr == nil {
				assert.Equal(tt, c.wantKeyspace, fakeClient.upgradeStartKS, c.desc)
			}

			if c.wantAnnotation {
				updated := &v1alpha1.TiDBGroup{}
				require.NoError(tt, fc.Get(ctx, client.ObjectKeyFromObject(c.dbg), updated))
				assert.Equal(tt, v1alpha1.AnnoValSmoothUpgradePhaseInProgress,
					updated.Annotations[v1alpha1.AnnoKeySmoothUpgradePhase], c.desc)
			}
		})
	}
}

func TestTaskSmoothUpgradeFinish(t *testing.T) {
	cases := []struct {
		desc               string
		dbg                *v1alpha1.TiDBGroup
		dbs                []*v1alpha1.TiDB
		apiErr             error
		expectedStatus     task.Status
		wantAPICalled      bool
		wantAnnotationGone bool
	}{
		{
			desc: "no annotation — no-op",
			dbg: fake.FakeObj("dbg", func(o *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
				o.Spec.Template.Spec.Version = "v8.0.0"
				o.Status.Version = "v8.0.0"
				return o
			}),
			dbs:            []*v1alpha1.TiDB{fakeReadyTiDB()},
			expectedStatus: task.SComplete,
			wantAPICalled:  false,
		},
		{
			desc: "annotation set but upgrade still in progress",
			dbg: fake.FakeObj("dbg", func(o *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
				o.Spec.Template.Spec.Version = "v8.0.0"
				o.Status.Version = "v7.5.0"
				o.Annotations = map[string]string{
					v1alpha1.AnnoKeySmoothUpgradePhase: v1alpha1.AnnoValSmoothUpgradePhaseInProgress,
				}
				return o
			}),
			dbs:            []*v1alpha1.TiDB{fakeReadyTiDB()},
			expectedStatus: task.SComplete,
			wantAPICalled:  false,
		},
		{
			desc: "no ready instance — retry",
			dbg: fake.FakeObj("dbg", func(o *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
				o.Spec.Template.Spec.Version = "v8.0.0"
				o.Status.Version = "v8.0.0"
				o.Annotations = map[string]string{
					v1alpha1.AnnoKeySmoothUpgradePhase: v1alpha1.AnnoValSmoothUpgradePhaseInProgress,
				}
				return o
			}),
			dbs:            []*v1alpha1.TiDB{},
			expectedStatus: task.SRetry,
			wantAPICalled:  false,
		},
		{
			desc: "API error — retry",
			dbg: fake.FakeObj("dbg", func(o *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
				o.Spec.Template.Spec.Version = "v8.0.0"
				o.Status.Version = "v8.0.0"
				o.Annotations = map[string]string{
					v1alpha1.AnnoKeySmoothUpgradePhase: v1alpha1.AnnoValSmoothUpgradePhaseInProgress,
				}
				return o
			}),
			dbs:            []*v1alpha1.TiDB{fakeReadyTiDB()},
			apiErr:         fmt.Errorf("connection refused"),
			expectedStatus: task.SRetry,
			wantAPICalled:  true,
		},
		{
			desc: "success — annotation removed",
			dbg: fake.FakeObj("dbg", func(o *v1alpha1.TiDBGroup) *v1alpha1.TiDBGroup {
				o.Spec.Template.Spec.Version = "v8.0.0"
				o.Status.Version = "v8.0.0"
				o.Annotations = map[string]string{
					v1alpha1.AnnoKeySmoothUpgradePhase: v1alpha1.AnnoValSmoothUpgradePhaseInProgress,
				}
				return o
			}),
			dbs:                []*v1alpha1.TiDB{fakeReadyTiDB()},
			expectedStatus:     task.SComplete,
			wantAPICalled:      true,
			wantAnnotationGone: true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fakeClient := &fakeTiDBClient{upgradeFinishErr: c.apiErr}

			factory := func(_ context.Context, _ client.Client, _ *v1alpha1.Cluster, _ *v1alpha1.TiDB) (tidbapi.TiDBClient, error) {
				return fakeClient, nil
			}

			ctx := context.Background()
			fc := client.NewFakeClient(c.dbg, fake.FakeObj[v1alpha1.Cluster]("cluster"))

			rc := &ReconcileContext{
				State: &state{
					dbg:     c.dbg,
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					dbs:     c.dbs,
				},
			}

			res, done := task.RunTask(ctx, taskSmoothUpgradeFinish(rc, fc, factory))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)
			assert.Equal(tt, c.wantAPICalled, fakeClient.upgradeFinishCalled, c.desc)

			if c.wantAnnotationGone {
				updated := &v1alpha1.TiDBGroup{}
				require.NoError(tt, fc.Get(ctx, client.ObjectKeyFromObject(c.dbg), updated))
				_, hasAnno := updated.Annotations[v1alpha1.AnnoKeySmoothUpgradePhase]
				assert.False(tt, hasAnno, c.desc)
			}
		})
	}
}
