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

package clean

import (
	"context"
	"fmt"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	backupMgr "github.com/pingcap/tidb-operator/pkg/controllers/br/manager/backup"
)

// Manager mainly used to manage backup related work
type Manager struct {
	cli           client.Client
	StatusUpdater backupMgr.BackupConditionUpdaterInterface
	Options
}

// NewManager return a Manager
func NewManager(
	cli client.Client,
	statusUpdater backupMgr.BackupConditionUpdaterInterface,
	backupOpts Options) *Manager {
	return &Manager{
		cli,
		statusUpdater,
		backupOpts,
	}
}

// ProcessCleanBackup used to clean the specific backup
func (bm *Manager) ProcessCleanBackup() error {
	ctx, cancel := util.GetContextForTerminationSignals(fmt.Sprintf("clean %s", bm.BackupName))
	defer cancel()

	backup := &v1alpha1.Backup{}
	err := bm.cli.Get(ctx, client.ObjectKey{
		Namespace: bm.Namespace,
		Name:      bm.BackupName,
	}, backup)
	if err != nil {
		return fmt.Errorf("can't find cluster %s backup %s CRD object, err: %v", bm, bm.BackupName, err)
	}

	return bm.performCleanBackup(ctx, backup.DeepCopy())
}

func (bm *Manager) performCleanBackup(ctx context.Context, backup *v1alpha1.Backup) error {
	if backup.Status.BackupPath == "" {
		klog.Errorf("cluster %s backup path is empty", bm)
		return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Condition: metav1.Condition{
				Type:    string(v1alpha1.BackupFailed),
				Status:  metav1.ConditionTrue,
				Reason:  "BackupPathIsEmpty",
				Message: fmt.Sprintf("the cluster %s backup path is empty", bm),
			},
		}, nil)
	}

	var errs []error
	var err error
	// TODO(ideascf): remove theses lines, EBS volume snapshot backup is deprecated in v2
	// volume-snapshot backup requires to delete the snapshot firstly, then delete the backup meta file
	// volume-snapshot is incremental snapshot per volume. Any backup deletion will take effects on next volume-snapshot backup
	// we need update backup size of the impacted the volume-snapshot backup.
	// if backup.Spec.Mode == v1alpha1.BackupModeVolumeSnapshot {
	// 	nextNackup := bm.getNextBackup(ctx, backup)
	// 	if nextNackup == nil {
	// 		klog.Errorf("get next backup for cluster %s backup is nil", bm)
	// 	}

	// 	// clean backup will delete all vol snapshots
	// 	err = bm.cleanBackupMetaWithVolSnapshots(ctx, backup)
	// 	if err != nil {
	// 		klog.Errorf("delete backup %s for cluster %s backup failure", backup.Name, bm)
	// 	}

	// } else
	{
		if backup.Spec.BR != nil {
			err = bm.CleanBRRemoteBackupData(ctx, backup)
		} else {
			opts := util.GetOptions(backup.Spec.StorageProvider)
			err = bm.cleanRemoteBackupData(ctx, backup.Status.BackupPath, opts)
		}
	}

	if err != nil {
		errs = append(errs, err)
		klog.Errorf("clean cluster %s backup %s failed, err: %s", bm, backup.Status.BackupPath, err)
		uerr := bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Condition: metav1.Condition{
				Type:    string(v1alpha1.BackupCleanFailed),
				Status:  metav1.ConditionTrue,
				Reason:  "CleanBackupDataFailed",
				Message: err.Error(),
			},
		}, nil)
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}

	klog.Infof("clean cluster %s backup %s success", bm, backup.Status.BackupPath)
	return bm.StatusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Condition: metav1.Condition{
			Type:   string(v1alpha1.BackupClean),
			Status: metav1.ConditionTrue,
		},
	}, nil)
}

// getNextBackup to get next backup sorted by start time
func (bm *Manager) getNextBackup(ctx context.Context, backup *v1alpha1.Backup) *v1alpha1.Backup {
	var err error
	backupList := &v1alpha1.BackupList{}
	err = bm.cli.List(ctx, backupList, client.InNamespace(backup.Namespace))
	if err != nil {
		return nil
	}
	bks := backupList.Items

	// sort the backup list by TimeStarted, since volume snapshot is point-in-time (start time) backup
	sort.Slice(bks, func(i, j int) bool {
		return bks[i].Status.TimeStarted.Before(&bks[j].Status.TimeStarted)
	})

	for i, bk := range bks {
		if backup.Name == bk.Name {
			return bm.getVolumeSnapshotBackup(bks[i+1:])
		}
	}

	return nil
}

// getVolumeSnapshotBackup get the first volume-snapshot backup from backup list, which may contain non-volume snapshot
func (bm *Manager) getVolumeSnapshotBackup(backups []v1alpha1.Backup) *v1alpha1.Backup {
	for _, bk := range backups {
		if bk.Spec.Mode == v1alpha1.BackupModeVolumeSnapshot {
			return &bk
		}
	}

	// reach end of backup list, there is no volume snapshot backups
	return nil
}
