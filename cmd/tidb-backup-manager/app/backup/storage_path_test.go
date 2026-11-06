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
	"errors"
	"testing"

	backupMgr "github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/backup"
	bkutil "github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/util"

	"github.com/stretchr/testify/require"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
)

func TestBackupCannotReplaceRecordedStoragePath(t *testing.T) {
	backup := &v1alpha1.Backup{}
	backup.Spec.S3 = &v1alpha1.S3StorageProvider{Bucket: "bucket", Prefix: "other"}
	backup.Status.BackupPath = "s3://bucket/original"
	manager := &Manager{}
	require.ErrorContains(t, manager.performBackup(context.Background(), backup), "cannot change")
	_, reason, err := manager.startLogBackup(context.Background(), backup)
	require.ErrorContains(t, err, "cannot change")
	require.Equal(t, "BackupStorageTargetChanged", reason)
	require.Equal(t, "s3://bucket/original", backup.Status.BackupPath)
}

type stoppingStatusUpdater struct {
	calls int
	err   error
}

func (u *stoppingStatusUpdater) Update(context.Context, *v1alpha1.Backup, *v1alpha1.BackupCondition, *backupMgr.BackupUpdateStatus) error {
	u.calls++
	return u.err
}
func TestBackupStoragePathAllowsOriginalDestination(t *testing.T) {
	for _, recorded := range []string{"", "s3://bucket/backup"} {
		backup := &v1alpha1.Backup{}
		backup.Spec.S3 = &v1alpha1.S3StorageProvider{Bucket: "bucket", Prefix: "backup"}
		backup.Status.BackupPath = recorded
		identity, err := bkutil.StorageIdentity(backup)
		require.NoError(t, err)
		backup.Status.StorageIdentity = identity
		for _, logMode := range []bool{false, true} {
			updater := &stoppingStatusUpdater{err: errors.New("stop before external backup execution")}
			manager := &Manager{StatusUpdater: updater}
			if logMode {
				_, _, err = manager.startLogBackup(context.Background(), backup)
			} else {
				err = manager.performBackup(context.Background(), backup)
			}
			require.ErrorIs(t, err, updater.err)
			require.Equal(t, 1, updater.calls)
		}
	}
}
