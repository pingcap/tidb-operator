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
	"encoding/binary"
	"fmt"
	"path"
	"strconv"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	corev1alpha1 "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/timanager"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
)

var (
	refreshCheckpointTsPeriod = time.Minute * 1
	streamKeyPrefix           = "/tidb/br-stream"
	taskCheckpointPath        = "/checkpoint"
)

// BackupTracker implements the logic for tracking log backup progress
type BackupTracker interface {
	StartTrackLogBackupProgress(ctx context.Context, backup *v1alpha1.Backup) error
}

// the main processes of log backup track:
// a. tracker init will try to find all log backup and add them to the map which key is namespack and cluster.
// b. log backup start will add it to the map
// c. if add log backup to the map successfully, it will start a go routine which has a loop
// to track log backup's checkpoint ts and will stop when log backup complete.
// d. by the way, add or delete the map has a mutex.
type backupTracker struct {
	cli           client.Client
	pdcm          pdm.PDClientManager
	statusUpdater BackupConditionUpdaterInterface

	operateLock sync.Mutex
	logBackups  map[string]bool
}

// NewBackupTracker returns a BackupTracker
func NewBackupTracker(cli client.Client, pdcm pdm.PDClientManager, statusUpdater BackupConditionUpdaterInterface) BackupTracker {
	tracker := &backupTracker{
		cli:           cli,
		statusUpdater: statusUpdater,
		logBackups:    make(map[string]bool),
		pdcm:          pdcm,
	}
	go tracker.initTrackLogBackupsProgress()
	return tracker
}

// initTrackLogBackupsProgress lists all log backups and track their progress.
func (bt *backupTracker) initTrackLogBackupsProgress() {
	ctx := context.Background()
	logger := log.FromContext(ctx)
	var (
		backups *v1alpha1.BackupList
		err     error
	)
	err = retry.OnError(retry.DefaultRetry, func(e error) bool { return e != nil }, func() error {
		backups = &v1alpha1.BackupList{}
		err = bt.cli.List(ctx, backups)
		if err != nil {
			logger.Error(err, "list backups error, will retry")
			return err
		}
		return nil
	})
	if err != nil {
		logger.Error(err, "list backups error after retry, skip track all log backups progress when init, will track when log backup start")
		return
	}

	logger.Info("list backups success", "size", len(backups.Items))
	for i := range backups.Items {
		backup := backups.Items[i]
		if backup.Spec.Mode == v1alpha1.BackupModeLog {
			err = bt.StartTrackLogBackupProgress(ctx, &backup)
			if err != nil {
				logger.Error(err,
					"start track log backup error, will skip and track when log backup start",
					"namespace", backup.Namespace,
					"backup", backup.Name)
			}
		}
	}
}

// StartTrackLogBackupProgress starts to track log backup progress.
func (bt *backupTracker) StartTrackLogBackupProgress(ctx context.Context, backup *v1alpha1.Backup) error {
	logger := log.FromContext(ctx)
	if backup.Spec.Mode != v1alpha1.BackupModeLog {
		return nil
	}
	ns := backup.Namespace
	name := backup.Name

	bt.operateLock.Lock()
	defer bt.operateLock.Unlock()

	logkey := genLogBackupKey(ns, name)
	if _, exist := bt.logBackups[logkey]; exist {
		logger.Info("log backup has exist in tracker", "namespace", ns, "backup", name, "key", logkey)
		return nil
	}
	logger.Info("add log backup to tracker", "namespace", ns, "backup", name)
	_, err := bt.getLogBackupTC(ctx, backup)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info(
				"log backup cluster not found, will skip",
				"namespace", ns,
				"backup", name,
				"clusterNamespace", backup.Spec.BR.ClusterNamespace,
				"cluster", backup.Spec.BR.Cluster)
			return nil
		}
		return err
	}
	bt.logBackups[logkey] = true
	go bt.refreshLogBackupCheckpointTs(ns, name)
	return nil
}

// removeLogBackup removes log backup from tracker.
func (bt *backupTracker) removeLogBackup(ns, name string) {
	bt.operateLock.Lock()
	defer bt.operateLock.Unlock()
	delete(bt.logBackups, genLogBackupKey(ns, name))
}

// getLogBackupTC gets log backup's tidb cluster info.
func (bt *backupTracker) getLogBackupTC(ctx context.Context, backup *v1alpha1.Backup) (*corev1alpha1.Cluster, error) {
	logger := log.FromContext(ctx)
	var (
		ns               = backup.Namespace
		name             = backup.Name
		clusterNamespace = backup.Spec.BR.ClusterNamespace
		cluster          = &corev1alpha1.Cluster{}
		err              error
	)
	if backup.Spec.BR.ClusterNamespace == "" {
		clusterNamespace = ns
	}

	err = retry.OnError(retry.DefaultRetry, func(e error) bool { return e != nil }, func() error {
		err = bt.cli.Get(ctx, types.NamespacedName{Namespace: clusterNamespace, Name: backup.Spec.BR.Cluster}, cluster)
		if err != nil {
			logger.Error(err,
				"get log backup tidbcluster failed and will retry",
				"namespace", ns,
				"backup", name,
				"clusterNamespace", clusterNamespace,
				"cluster", backup.Spec.BR.Cluster)
			return err
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf(
			"get log backup %s/%s tidbcluster %s/%s failed, err is %w",
			ns, name, clusterNamespace, backup.Spec.BR.Cluster, err)
	}
	return cluster, nil
}

// refreshLogBackupCheckpointTs updates log backup progress periodically.
func (bt *backupTracker) refreshLogBackupCheckpointTs(ns, name string) {
	ctx := context.Background()
	logger := log.FromContext(ctx).WithValues("backupTrackerId", uuid.NewUUID())
	ctx = log.IntoContext(ctx, logger)
	ticker := time.NewTicker(refreshCheckpointTsPeriod)
	defer ticker.Stop()

	for range ticker.C {
		logkey := genLogBackupKey(ns, name)
		if _, exist := bt.logBackups[logkey]; !exist {
			return
		}
		backup := &v1alpha1.Backup{}
		err := bt.cli.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, backup)
		if apierrors.IsNotFound(err) {
			logger.Info("log backup has been deleted, will remove from tracker", "namespace", ns, "backup", name, "key", logkey)
			bt.removeLogBackup(ns, name)
			return
		}
		if err != nil {
			logger.Error(err, "get log backup error, will skip to the next time", "namespace", ns, "backup", name)
			continue
		}
		if backup.DeletionTimestamp != nil || backup.Status.Phase == v1alpha1.BackupComplete {
			logger.Info("log backup is being deleting or complete, will remove from tracker", "namespace", ns, "backup", name, "key", logkey)
			bt.removeLogBackup(ns, name)
			return
		}
		if backup.Status.Phase != v1alpha1.BackupRunning {
			logger.Info("log backup is not running, will skip to the next time refresh", "namespace", ns, "backup", name)
			continue
		}
		bt.doRefreshLogBackupCheckpointTs(ctx, backup)
	}
}

// doRefreshLogBackupCheckpointTs gets log backup checkpoint ts from pd and updates log backup cr.
func (bt *backupTracker) doRefreshLogBackupCheckpointTs(ctx context.Context, backup *v1alpha1.Backup) {
	logger := log.FromContext(ctx)
	ns := backup.Namespace
	name := backup.Name
	pc, ok := bt.pdcm.Get(timanager.PrimaryKey(backup.Namespace, backup.Spec.BR.Cluster))
	if !ok {
		logger.Error(nil, "get log backup pd client error", "namespace", ns, "backup", name)
		return
	}
	// Note: `pc` doesn't require close, because it's a shared client.
	etcdCli, err := pc.Underlay().GetPDEtcdClient()
	if err != nil {
		logger.Error(err, "get log backup pd etcd client error", "namespace", ns, "backup", name)
		return
	}
	defer etcdCli.Close() //nolint:errcheck

	key := path.Join(streamKeyPrefix, taskCheckpointPath, name)
	logger.Info("log backup checkpointTS key", "namespace", ns, "backup", name, "key", key)

	kvs, err := etcdCli.Get(key, true)
	if err != nil {
		logger.Error(err, "get log backup checkpointTS error", "namespace", ns, "backup", name)
		return
	}
	if len(kvs) < 1 {
		logger.Error(nil, "log backup checkpointTS not found", "namespace", ns, "backup", name)
		return
	}
	ckTS := strconv.FormatUint(binary.BigEndian.Uint64(kvs[0].Value), 10)

	logger.Info("update log backup checkpointTS", "namespace", ns, "backup", name, "checkpointTS", ckTS)
	updateStatus := &BackupUpdateStatus{
		LogCheckpointTs: &ckTS,
	}
	err = bt.statusUpdater.Update(ctx, backup, nil, updateStatus)
	if err != nil {
		logger.Error(err, "update log backup checkpointTS failed", "namespace", ns, "backup", name, "checkpointTS", ckTS)
		return
	}
}

func genLogBackupKey(ns, name string) string {
	return fmt.Sprintf("%s.%s", ns, name)
}
