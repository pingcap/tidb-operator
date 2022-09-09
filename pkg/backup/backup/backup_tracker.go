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

package backup

import (
	"encoding/binary"
	"fmt"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
)

var (
	refreshCheckpointTsPeriod = time.Minute * 1
	streamKeyPrefix           = "/tidb/br-stream"
	taskCheckpointPath        = "/checkpoint"
)

// BackupCleaner implements the logic for cleaning backup
type BackupTracker interface {
	StartTrackLogBackupProgress(backup *v1alpha1.Backup)
	// StopTrackLogBackupProgress(backup *v1alpha1.Backup)
}

type backupTracker struct {
	deps          *controller.Dependencies
	statusUpdater controller.BackupConditionUpdaterInterface
	operateLock   sync.Mutex
	logBackups    map[string]interface{}
}

// NewBackupCleaner returns a BackupCleaner
func NewBackupTracker(deps *controller.Dependencies, statusUpdater controller.BackupConditionUpdaterInterface) BackupTracker {
	return &backupTracker{
		deps:          deps,
		statusUpdater: statusUpdater,
		logBackups:    make(map[string]interface{}, 0),
	}
}

func (bt *backupTracker) StartTrackLogBackupProgress(backup *v1alpha1.Backup) {
	if backup.Spec.Mode != v1alpha1.BackupModeLog {
		return
	}
	bt.operateLock.Lock()
	defer bt.operateLock.Unlock()

	logkey := genLogBackupKey(backup.Namespace, backup.Name)
	if _, exist := bt.logBackups[logkey]; exist {
		return
	}
	bt.logBackups[logkey] = logkey
	go bt.refreshLogBackupCheckpointTs(backup.Namespace, backup.Name)
}

func (bt *backupTracker) removeLogBackup(ns, name string) {
	bt.operateLock.Lock()
	defer bt.operateLock.Unlock()
	delete(bt.logBackups, genLogBackupKey(ns, name))
}

func (bt *backupTracker) refreshLogBackupCheckpointTs(ns, name string) {
	ticker := time.NewTicker(refreshCheckpointTsPeriod)
	defer ticker.Stop()

	for range ticker.C {
		logkey := genLogBackupKey(ns, name)
		if _, exist := bt.logBackups[logkey]; !exist {
			return
		}
		backup, err := bt.deps.BackupLister.Backups(ns).Get(name)
		if errors.IsNotFound(err) {
			klog.Infof("Backup %s/%s has been deleted %v", ns, name, err)
			bt.removeLogBackup(ns, name)
			return
		}
		if err != nil {
			klog.Infof("get Backup has error %v", err)
			continue
		}
		if backup.DeletionTimestamp != nil || backup.Status.Phase != v1alpha1.BackupRunning {
			continue
		}
		bt.doRefreshLogBackupCheckpointTs(backup)
	}
}

func (bt *backupTracker) doRefreshLogBackupCheckpointTs(backup *v1alpha1.Backup) {
	ns := backup.Namespace
	name := backup.Name
	clusterNamespace := backup.Spec.BR.ClusterNamespace
	if backup.Spec.BR.ClusterNamespace == "" {
		clusterNamespace = backup.Namespace
	}
	url := fmt.Sprintf("%s-pd.%s:2379", backup.Spec.BR.Cluster, clusterNamespace)
	klog.Infof("log backup %s/%s pd url %s", ns, name, url)

	etcdCli, err := pdapi.NewPdEtcdClient(url, 30*time.Second, nil)
	if err != nil {
		klog.Errorf("get log backup %s/%s pd cli error %v", ns, name, err)
		return
	}
	key := path.Join(streamKeyPrefix, taskCheckpointPath, name)
	klog.Infof("log backup %s/%s checkpointTS key %s", ns, name, key)

	kvs, err := etcdCli.Get(key, true)
	if err != nil {
		klog.Errorf("get log backup %s/%s checkpointTS error %v", ns, name, err)
		return
	}
	if len(kvs) < 1 {
		klog.Errorf("log backup %s/%s checkpointTS not found", ns, name)
	}
	ckTS := strconv.FormatUint(binary.BigEndian.Uint64(kvs[0].Value), 10)

	updateStatus := &controller.BackupUpdateStatus{
		TimeStarted:     &backup.Status.TimeCompleted,
		LogCheckpointTs: &ckTS,
	}
	err = bt.statusUpdater.Update(backup, nil, updateStatus)
	if err != nil {
		klog.Errorf("update log backup %s/%s checkpointTS %s failed %v", ns, name, ckTS, err)
		return
	}
}

func genLogBackupKey(ns, name string) string {
	return fmt.Sprintf("%s.%s", ns, name)
}
