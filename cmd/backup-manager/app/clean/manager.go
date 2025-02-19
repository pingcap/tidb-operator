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
		return fmt.Errorf("can't find cluster %s backup %s CRD object, err: %w", bm, bm.BackupName, err)
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
