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
	"k8s.io/client-go/tools/record"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	manager "github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/backup"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/util"
)

type mutatingBackupReader struct {
	ctrlclient.Client
	reads, changeAt int
	options         []string
	mutate          func(*br.Backup)
}

func (c *mutatingBackupReader) Get(ctx context.Context, key ctrlclient.ObjectKey, obj ctrlclient.Object, opts ...ctrlclient.GetOption) error {
	if _, ok := obj.(*br.Backup); ok {
		c.reads++
		if c.reads == c.changeAt {
			current := &br.Backup{}
			if err := c.Client.Get(ctx, key, current); err != nil {
				return err
			}
			if c.mutate != nil {
				c.mutate(current)
			} else if c.options != nil {
				current.Spec.BR.Options = c.options
			} else {
				current.Spec.S3.Endpoint = "https://other"
			}
			if err := c.Update(ctx, current); err != nil {
				return err
			}
		}
	}
	return c.Client.Get(ctx, key, obj, opts...)
}
func TestStorageChangeDuringStatusRefreshIsRejected(t *testing.T) {
	for _, changeAt := range []int{1, 2} {
		backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "tenant"}, Spec: br.BackupSpec{BR: &br.BRConfig{Cluster: "cluster"}}}
		backup.Spec.S3 = &br.S3StorageProvider{Bucket: "bucket", Prefix: "backup", Endpoint: "https://original"}
		reader := &mutatingBackupReader{Client: client.NewFakeClient(backup), changeAt: changeAt}
		updater := manager.NewRealBackupConditionUpdater(reader, record.NewFakeRecorder(100))
		m := &Manager{StatusUpdater: updater}
		require.ErrorContains(t, m.performBackup(context.Background(), backup.DeepCopy()), "storage identity has changed")
		if changeAt == 1 {
			backup.Spec.S3.Endpoint = "https://original"
			reader.reads = 0
			reader.Client = client.NewFakeClient(backup)
			_, _, err := m.startLogBackup(context.Background(), backup.DeepCopy())
			require.ErrorContains(t, err, "storage identity has changed")
		}
	}
}

func TestTargetOverridesDuringStatusRefreshAreRejected(t *testing.T) {
	for _, option := range []string{"--pd=victim:2379", "--storage=s3://other/backup"} {
		for _, changeAt := range []int{1, 2} {
			backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "tenant"}, Spec: br.BackupSpec{BR: &br.BRConfig{Cluster: "cluster"}}}
			backup.Spec.S3 = &br.S3StorageProvider{Bucket: "bucket", Prefix: "backup"}
			reader := &mutatingBackupReader{Client: client.NewFakeClient(backup), changeAt: changeAt, options: []string{option}}
			updater := manager.NewRealBackupConditionUpdater(reader, record.NewFakeRecorder(100))
			m := &Manager{StatusUpdater: updater}
			require.ErrorContains(t, m.performBackup(t.Context(), backup.DeepCopy()), "spec.br.options")
			if changeAt == 1 {
				reader.reads = 0
				reader.Client = client.NewFakeClient(backup)
				_, _, err := m.startLogBackup(t.Context(), backup.DeepCopy())
				require.ErrorContains(t, err, "spec.br.options")
			}
		}
	}
}

func TestProcessTruncateRejectsChangedStorage(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*br.Backup)
	}{
		{"bucket", func(b *br.Backup) { b.Spec.S3.Bucket = "other" }},
		{"prefix", func(b *br.Backup) { b.Spec.S3.Prefix = "other" }},
		{"endpoint", func(b *br.Backup) { b.Spec.S3.Endpoint = "https://other" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, changeAt := range []int{0, 3} {
				backup := &br.Backup{
					ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: "tenant"},
					Spec:       br.BackupSpec{Mode: br.BackupModeLog, BR: &br.BRConfig{Cluster: "cluster"}},
					Status:     br.BackupStatus{CommitTs: "1", BackupPath: "s3://bucket/backup"},
				}
				backup.Spec.S3 = &br.S3StorageProvider{Bucket: "bucket", Prefix: "backup", Endpoint: "https://original"}
				var err error
				backup.Status.StorageIdentity, err = util.StorageIdentity(backup)
				require.NoError(t, err)
				if changeAt == 0 {
					tc.mutate(backup)
				}
				reader := &mutatingBackupReader{Client: client.NewFakeClient(backup), changeAt: changeAt, mutate: tc.mutate}
				updater := manager.NewRealBackupConditionUpdater(reader, record.NewFakeRecorder(100))
				// An invalid until value also prevents an external command on the unfixed path.
				m := NewManager(reader, updater, Options{})
				m.Namespace, m.ResourceName = "tenant", "backup"
				m.Mode, m.SubCommand = string(br.BackupModeLog), string(br.LogTruncateCommand)
				m.TruncateUntil = "0"
				err = m.ProcessBackup()
				require.ErrorContains(t, err, "storage identity has changed")
			}
		})
	}
}
