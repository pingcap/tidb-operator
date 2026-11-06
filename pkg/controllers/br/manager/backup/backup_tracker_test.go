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
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
)

type checkpointManager struct {
	pdm.PDClientManager
	client pdm.PDClient
}

func (m *checkpointManager) Get(string) (pdm.PDClient, bool) { return m.client, true }

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
				pdcm:          &checkpointManager{client: &checkpointPD{client: &checkpointUnderlay{client: etcd}}},
				statusUpdater: updater,
			}
			backup := &v1alpha1.Backup{ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "backup"}, Spec: v1alpha1.BackupSpec{BR: &v1alpha1.BRConfig{Cluster: "cluster"}}}
			tracker.doRefreshLogBackupCheckpointTs(context.Background(), backup)
			require.True(t, etcd.closed)
			if tc.want == "" {
				require.Nil(t, updater.value)
			} else {
				require.NotNil(t, updater.value)
				require.Equal(t, tc.want, *updater.value)
			}
		})
	}
}

func TestTrackerRefreshConcurrentRemoval(t *testing.T) {
	previousPeriod := refreshCheckpointTsPeriod
	refreshCheckpointTsPeriod = time.Nanosecond
	defer func() { refreshCheckpointTsPeriod = previousPeriod }()
	backup := &v1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "backup"},
		Status:     v1alpha1.BackupStatus{Phase: v1alpha1.BackupPaused},
	}
	tracker := &backupTracker{
		cli:        client.NewFakeClient(backup),
		logBackups: map[string]bool{genLogBackupKey(backup.Namespace, backup.Name): true},
	}
	done := make(chan struct{})
	go func() {
		tracker.refreshLogBackupCheckpointTs(backup.Namespace, backup.Name)
		close(done)
	}()
	for range 100000 {
		tracker.removeLogBackup("test", "other")
	}
	tracker.removeLogBackup(backup.Namespace, backup.Name)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("checkpoint refresh did not stop after removing the tracked backup")
	}
}
