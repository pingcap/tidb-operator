// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package _import

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	bkconstants "github.com/pingcap/tidb-operator/pkg/backup/constants"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
)

// RestoreManager mainly used to manage restore related work
type RestoreManager struct {
	restoreLister listers.RestoreLister
	StatusUpdater controller.RestoreConditionUpdaterInterface
	Options
}

// NewRestoreManager return a RestoreManager
func NewRestoreManager(
	restoreLister listers.RestoreLister,
	statusUpdater controller.RestoreConditionUpdaterInterface,
	restoreOpts Options) *RestoreManager {
	return &RestoreManager{
		restoreLister,
		statusUpdater,
		restoreOpts,
	}
}

func (rm *RestoreManager) setOptions(restore *v1alpha1.Restore) {
	rm.Options.Host = restore.Spec.To.Host

	if restore.Spec.To.Port != 0 {
		rm.Options.Port = restore.Spec.To.Port
	} else {
		rm.Options.Port = bkconstants.DefaultTidbPort
	}

	if restore.Spec.To.User != "" {
		rm.Options.User = restore.Spec.To.User
	} else {
		rm.Options.User = bkconstants.DefaultTidbUser
	}

	rm.Options.Password = util.GetOptionValueFromEnv(bkconstants.TidbPasswordKey, bkconstants.BackupManagerEnvVarPrefix)
}

// ProcessRestore used to process the restore logic
func (rm *RestoreManager) ProcessRestore() error {
	var errs []error
	restore, err := rm.restoreLister.Restores(rm.Namespace).Get(rm.ResourceName)
	if err != nil {
		errs = append(errs, err)
		klog.Errorf("can't find cluster %s restore %s CRD object, err: %v", rm, rm.ResourceName, err)
		uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetRestoreCRFailed",
			Message: err.Error(),
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}

	rm.setOptions(restore)

	var db *sql.DB
	var dsn string
	err = wait.PollImmediate(constants.PollInterval, constants.CheckTimeout, func() (done bool, err error) {
		// TLS is not currently supported
		dsn, err = rm.GetDSN(false)
		if err != nil {
			klog.Errorf("can't get dsn of tidb cluster %s, err: %s", rm, err)
			return false, err
		}

		db, err = util.OpenDB(dsn)
		if err != nil {
			klog.Warningf("can't connect to tidb cluster %s, err: %s", rm, err)
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		errs = append(errs, err)
		klog.Errorf("cluster %s connect failed, err: %s", rm, err)
		uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "ConnectTidbFailed",
			Message: err.Error(),
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}

	defer db.Close()
	return rm.performRestore(restore.DeepCopy())
}

func (rm *RestoreManager) performRestore(restore *v1alpha1.Restore) error {
	started := time.Now()

	err := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
		Type:   v1alpha1.RestoreRunning,
		Status: corev1.ConditionTrue,
	})
	if err != nil {
		return err
	}

	var errs []error
	restoreDataPath := rm.getRestoreDataPath()
	opts := util.GetOptions(restore.Spec.StorageProvider)
	if err := rm.downloadBackupData(restoreDataPath, opts); err != nil {
		errs = append(errs, err)
		klog.Errorf("download cluster %s backup %s data failed, err: %s", rm, rm.BackupPath, err)
		uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "DownloadBackupDataFailed",
			Message: fmt.Sprintf("download backup %s data failed, err: %v", rm.BackupPath, err),
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}
	klog.Infof("download cluster %s backup %s data success", rm, rm.BackupPath)

	restoreDataDir := filepath.Dir(restoreDataPath)
	unarchiveDataPath, err := unarchiveBackupData(restoreDataPath, restoreDataDir)
	if err != nil {
		errs = append(errs, err)
		klog.Errorf("unarchive cluster %s backup %s data failed, err: %s", rm, restoreDataPath, err)
		uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "UnarchiveBackupDataFailed",
			Message: fmt.Sprintf("unarchive backup %s data failed, err: %v", restoreDataPath, err),
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}
	klog.Infof("unarchive cluster %s backup %s data success", rm, restoreDataPath)

	err = rm.loadTidbClusterData(unarchiveDataPath)
	if err != nil {
		errs = append(errs, err)
		klog.Errorf("restore cluster %s from backup %s failed, err: %s", rm, rm.BackupPath, err)
		uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "LoaderBackupDataFailed",
			Message: fmt.Sprintf("loader backup %s data failed, err: %v", restoreDataPath, err),
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}
	klog.Infof("restore cluster %s from backup %s success", rm, rm.BackupPath)

	commitTs, err := util.GetCommitTsFromMetadata(unarchiveDataPath)
	if err != nil {
		errs = append(errs, err)
		klog.Errorf("get cluster %s commitTs failed, err: %s", rm, err)
		uerr := rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetCommitTsFailed",
			Message: err.Error(),
		})
		errs = append(errs, uerr)
		return errorutils.NewAggregate(errs)
	}
	klog.Infof("get cluster %s commitTs %s success", rm, commitTs)

	finish := time.Now()

	restore.Status.TimeStarted = metav1.Time{Time: started}
	restore.Status.TimeCompleted = metav1.Time{Time: finish}
	restore.Status.CommitTs = commitTs

	return rm.StatusUpdater.Update(restore, &v1alpha1.RestoreCondition{
		Type:   v1alpha1.RestoreComplete,
		Status: corev1.ConditionTrue,
	})
}
