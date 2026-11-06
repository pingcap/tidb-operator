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

package backup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	core "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/util"
	"github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/timanager"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
)

type checkpointManager struct {
	pdm.PDClientManager
	client pdm.PDClient
	key    string
}

func (m *checkpointManager) Get(key string) (pdm.PDClient, bool) { m.key = key; return m.client, true }

type checkpointPD struct {
	pdm.PDClient
	client pdapi.PDClient
}

func (p *checkpointPD) Underlay() pdapi.PDClient { return p.client }

type checkpointUnderlay struct {
	pdapi.PDClient
	client pdapi.PDEtcdClient
}

func (p *checkpointUnderlay) GetPDEtcdClient() (pdapi.PDEtcdClient, error) { return p.client, nil }

type checkpointEtcd struct {
	pdapi.PDEtcdClient
	kvs    []*pdapi.KeyValue
	closed bool
}

func (e *checkpointEtcd) Get(string, bool) ([]*pdapi.KeyValue, error) { return e.kvs, nil }
func (e *checkpointEtcd) Close() error                                { e.closed = true; return nil }

type checkpointUpdater struct{ value *string }

func (u *checkpointUpdater) Update(_ context.Context, _ *v1alpha1.Backup, _ *v1alpha1.BackupCondition, status *BackupUpdateStatus) error {
	u.value = status.LogCheckpointTs
	return nil
}

func TestRefreshLogBackupCheckpointEncoding(t *testing.T) {
	for _, tc := range []struct {
		name string
		kvs  []*pdapi.KeyValue
		want string
	}{
		{name: "missing"},
		{name: "nil entry", kvs: []*pdapi.KeyValue{nil}},
		{name: "empty", kvs: []*pdapi.KeyValue{{}}},
		{name: "short", kvs: []*pdapi.KeyValue{{Value: []byte{1, 2, 3, 4, 5, 6, 7}}}},
		{name: "long", kvs: []*pdapi.KeyValue{{Value: make([]byte, 9)}}},
		{name: "valid", kvs: []*pdapi.KeyValue{{Value: []byte{0, 0, 0, 0, 0, 0, 0, 42}}}, want: "42"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			etcd := &checkpointEtcd{kvs: tc.kvs}
			updater := &checkpointUpdater{}
			tracker := &backupTracker{
				cli:           client.NewFakeClient(&core.Cluster{ObjectMeta: metav1.ObjectMeta{Namespace: "target", Name: "cluster", Annotations: map[string]string{util.AllowedBackupNamespaces: "test"}}}),
				pdcm:          &checkpointManager{client: &checkpointPD{client: &checkpointUnderlay{client: etcd}}},
				statusUpdater: updater,
			}
			backup := &v1alpha1.Backup{ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "backup"}, Spec: v1alpha1.BackupSpec{BR: &v1alpha1.BRConfig{Cluster: "cluster", ClusterNamespace: "target"}}}
			tracker.doRefreshLogBackupCheckpointTs(context.Background(), backup)
			require.True(t, etcd.closed)
			require.Equal(t, timanager.PrimaryKey("target", "cluster"), tracker.pdcm.(*checkpointManager).key)
			if tc.want == "" {
				require.Nil(t, updater.value)
			} else {
				require.NotNil(t, updater.value)
				require.Equal(t, tc.want, *updater.value)
			}
		})
	}
}

func TestTrackerRejectsMissingBR(t *testing.T) {
	for _, br := range []*v1alpha1.BRConfig{nil, {}} {
		backup := &v1alpha1.Backup{ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "invalid"}, Spec: v1alpha1.BackupSpec{Mode: v1alpha1.BackupModeLog, BR: br}}
		tracker := &backupTracker{cli: client.NewFakeClient(backup), logBackups: make(map[string]bool)}
		require.ErrorContains(t, tracker.StartTrackLogBackupProgress(context.Background(), backup), "requires spec.br.cluster")
		require.Empty(t, tracker.logBackups)
		// Exercise the startup list path as well as a later checkpoint refresh.
		require.NotPanics(t, func() { tracker.initTrackLogBackupsProgress() })
		require.NotPanics(t, func() { tracker.doRefreshLogBackupCheckpointTs(context.Background(), backup) })
	}
}
