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
	"context"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

var (
	refreshCheckpointTsPeriod = time.Minute * 1
	streamKeyPrefix           = "/tidb/br-stream"
	taskCheckpointPath        = "/checkpoint"
)

// BackupTracker implements the logic for tracking log backup progress
type BackupTracker interface {
	StartTrackLogBackupProgress(backup *v1alpha1.Backup) error
}

// the main processes of log backup track:
// a. tracker init will try to find all log backup and add them to the map which key is namespack and cluster.
// b. log backup start will add it to the map
// c. if add log backup to the map successfully, it will start a go routine which has a loop to track log backup's checkpoint ts and will stop when log backup complete.
// d. by the way, add or delete the map has a mutex.
type backupTracker struct {
	deps          *controller.Dependencies
	statusUpdater controller.BackupConditionUpdaterInterface
	operateLock   sync.Mutex
	logBackups    map[string]*trackDepends
}

// trackDepends is the tracker depends, such as tidb cluster info.
type trackDepends struct {
	tc *v1alpha1.TidbCluster
}

// NewBackupTracker returns a BackupTracker
func NewBackupTracker(deps *controller.Dependencies, statusUpdater controller.BackupConditionUpdaterInterface) BackupTracker {
	tracker := &backupTracker{
		deps:          deps,
		statusUpdater: statusUpdater,
		logBackups:    make(map[string]*trackDepends),
	}
	go tracker.initTrackLogBackupsProgress()
	return tracker
}

// initTrackLogBackupsProgress lists all log backups and track their progress.
func (bt *backupTracker) initTrackLogBackupsProgress() {
	var (
		backups *v1alpha1.BackupList
		err     error
	)
	err = retry.OnError(retry.DefaultRetry, func(e error) bool { return e != nil }, func() error {
		backups, err = bt.deps.Clientset.PingcapV1alpha1().Backups("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			klog.Warningf("list backups error %v, will retry", err)
			return err
		}
		return nil
	})
	if err != nil {
		klog.Errorf("list backups error %v after retry, skip track all log backups progress when init, will track when log backup start", err)
		return
	}

	klog.Infof("list backups success, size %d", len(backups.Items))
	for i := range backups.Items {
		backup := backups.Items[i]
		if backup.Spec.Mode == v1alpha1.BackupModeLog {
			err = bt.StartTrackLogBackupProgress(&backup)
			if err != nil {
				klog.Warningf("start track log backup %s/%s error %v, will skip and track when log backup start", backup.Namespace, backup.Name, err)
			}
		}
	}
}

// StartTrackLogBackupProgress starts to track log backup progress.
func (bt *backupTracker) StartTrackLogBackupProgress(backup *v1alpha1.Backup) error {
	if backup.Spec.Mode != v1alpha1.BackupModeLog {
		return nil
	}
	ns := backup.Namespace
	name := backup.Name

	bt.operateLock.Lock()
	defer bt.operateLock.Unlock()

	logkey := genLogBackupKey(ns, name)
	if _, exist := bt.logBackups[logkey]; exist {
		klog.Infof("log backup %s/%s has exist in tracker %s", ns, name, logkey)
		return nil
	}
	klog.Infof("add log backup %s/%s to tracker", ns, name)
	tc, err := bt.getLogBackupTC(backup)
	if err != nil {
		return err
	}
	bt.logBackups[logkey] = &trackDepends{tc: tc}
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
func (bt *backupTracker) getLogBackupTC(backup *v1alpha1.Backup) (*v1alpha1.TidbCluster, error) {
	var (
		ns               = backup.Namespace
		name             = backup.Name
		clusterNamespace = backup.Spec.BR.ClusterNamespace
		tc               *v1alpha1.TidbCluster
		err              error
	)
	if backup.Spec.BR.ClusterNamespace == "" {
		clusterNamespace = ns
	}

	err = retry.OnError(retry.DefaultRetry, func(e error) bool { return e != nil }, func() error {
		tc, err = bt.deps.Clientset.PingcapV1alpha1().TidbClusters(clusterNamespace).Get(context.TODO(), backup.Spec.BR.Cluster, metav1.GetOptions{})
		if err != nil {
			klog.Warningf("get log backup %s/%s tidbcluster %s/%s failed and will retry, err is %v", ns, name, clusterNamespace, backup.Spec.BR.Cluster, err)
			return err
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("get log backup %s/%s tidbcluster %s/%s failed, err is %v", ns, name, clusterNamespace, backup.Spec.BR.Cluster, err)
	}
	return tc, nil
}

// refreshLogBackupCheckpointTs updates log backup progress periodically.
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
			klog.Infof("log backup %s/%s has been deleted, will remove %s from tracker", ns, name, logkey)
			bt.removeLogBackup(ns, name)
			return
		}
		if err != nil {
			klog.Infof("get log backup %s/%s error %v, will skip to the next time", ns, name, err)
			continue
		}
		if backup.DeletionTimestamp != nil || backup.Status.Phase == v1alpha1.BackupComplete {
			klog.Infof("log backup %s/%s is being deleting or complete, will remove %s from tracker", ns, name, logkey)
			bt.removeLogBackup(ns, name)
			return
		}
		if backup.Status.Phase != v1alpha1.BackupRunning {
			klog.Infof("log backup %s/%s is not running, will skip to the next time refresh", ns, name)
			continue
		}
		bt.doRefreshLogBackupCheckpointTs(backup, bt.logBackups[logkey])
	}
}

// doRefreshLogBackupCheckpointTs gets log backup checkpoint ts from pd and updates log backup cr.
func (bt *backupTracker) doRefreshLogBackupCheckpointTs(backup *v1alpha1.Backup, dep *trackDepends) {
	ns := backup.Namespace
	name := backup.Name
	etcdCli, err := bt.deps.PDControl.GetPDEtcdClient(pdapi.Namespace(dep.tc.Namespace), dep.tc.Name, dep.tc.IsTLSClusterEnabled())
	if err != nil {
		klog.Errorf("get log backup %s/%s pd cli error %v", ns, name, err)
		return
	}
	defer etcdCli.Close()
	key := path.Join(streamKeyPrefix, taskCheckpointPath, name)
	klog.Infof("log backup %s/%s checkpointTS key %s", ns, name, key)

	kvs, err := etcdCli.Get(key, true)
	if err != nil {
		klog.Errorf("get log backup %s/%s checkpointTS error %v", ns, name, err)
		return
	}
	if len(kvs) < 1 {
		klog.Errorf("log backup %s/%s checkpointTS not found", ns, name)
		return
	}
	ckTS := strconv.FormatUint(binary.BigEndian.Uint64(kvs[0].Value), 10)

	klog.Infof("update log backup %s/%s checkpointTS %s", ns, name, ckTS)
	updateStatus := &controller.BackupUpdateStatus{
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
