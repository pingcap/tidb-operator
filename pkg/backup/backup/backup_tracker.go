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
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

var (
	refreshCheckpointTsPeriod = time.Minute * 1
	streamKeyPrefix           = "/tidb/br-stream"
	taskInfoPath              = "/info"
	taskCheckpointPath        = "/checkpoint"
	taskLastErrorPath         = "/last-error"
	taskPausePath             = "/pause"
)

// LogBackupKernelState represents the actual state of log backup in the kernel (etcd)
type LogBackupKernelState string

const (
	// LogBackupKernelRunning means the log backup is running
	LogBackupKernelRunning LogBackupKernelState = "running"
	// LogBackupKernelPaused means the log backup is paused
	LogBackupKernelPaused LogBackupKernelState = "paused"
)

// LogBackupState represents the complete state of a log backup task
type LogBackupState struct {
	// Raw data from etcd queries
	CheckpointTS uint64 // checkpoint timestamp
	IsPaused     bool   // whether the backup is paused
	PauseReason  string // pause reason (manual or error message)
	LastError    string // last error message
	InfoExists   bool   // whether the info key exists in etcd

	// Derived state information
	KernelState     LogBackupKernelState       // kernel state (running/paused)
	ExpectedCommand v1alpha1.LogSubCommandType // expected command to execute
	IsConsistent    bool                       // whether the state is consistent
	NeedSync        bool                       // whether sync is needed

	// Metadata
	LastQueryTime time.Time // last query timestamp
	QueryError    error     // query error if any
}

// BackupTracker implements the logic for tracking log backup progress
type BackupTracker interface {
	StartTrackLogBackupProgress(backup *v1alpha1.Backup) error
	GetLogBackupTC(backup *v1alpha1.Backup) (*v1alpha1.TidbCluster, error)
	
	// New methods for unified state management
	GetLogBackupState(backup *v1alpha1.Backup) (*LogBackupState, error)
	SyncLogBackupState(backup *v1alpha1.Backup) (bool, error)
	RefreshLogBackupState(backup *v1alpha1.Backup) (*LogBackupState, error)
	StopTrackLogBackupProgress(backup *v1alpha1.Backup)
}

// the main processes of log backup track:
// a. tracker init will try to find all log backup and add them to the map which key is namespack and cluster.
// b. log backup start will add it to the map
// c. if add log backup to the map successfully, it will start a go routine which has a loop to track log backup's checkpoint ts and will stop when log backup complete.
// d. by the way, add or delete the map has a mutex.
type backupTracker struct {
	deps          *controller.Dependencies
	statusUpdater controller.BackupConditionUpdaterInterface
	operateLock   sync.RWMutex
	logBackups    map[string]*trackDepends
}

// trackDepends is the tracker depends, such as tidb cluster info.
type trackDepends struct {
	tc                    *v1alpha1.TidbCluster
	state                 *LogBackupState      // cached state
	etcdClient            pdapi.PDEtcdClient   // reused etcd client
	lastRefresh           time.Time            // last refresh timestamp
	inconsistencyDetected bool                 // flag to track if inconsistency was detected in previous reconcile
	lastCommand           v1alpha1.LogSubCommandType // last synced command to avoid cross-command interference
	mutex                 sync.RWMutex         // protect concurrent access
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
	ns := ""
	if !bt.deps.CLIConfig.ClusterScoped {
		ns = os.Getenv("NAMESPACE")
	}

	err = retry.OnError(retry.DefaultRetry, func(e error) bool { return e != nil }, func() error {
		backups, err = bt.deps.Clientset.PingcapV1alpha1().Backups(ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			klog.Warningf("list backups from namespace %s error %v, will retry", ns, err)
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
	tc, err := bt.GetLogBackupTC(backup)
	if err != nil {
		return err
	}
	bt.logBackups[logkey] = &trackDepends{tc: tc}
	go bt.refreshLogBackupProgress(ns, name)
	return nil
}

// removeLogBackup removes log backup from tracker.
func (bt *backupTracker) removeLogBackup(ns, name string) {
	bt.operateLock.Lock()
	defer bt.operateLock.Unlock()
	
	logkey := genLogBackupKey(ns, name)
	if dep, exist := bt.logBackups[logkey]; exist {
		// Close etcd client if exists
		if dep.etcdClient != nil {
			dep.etcdClient.Close()
		}
		delete(bt.logBackups, logkey)
	}
}

// getLogBackupTC gets log backup's tidb cluster info.
func (bt *backupTracker) GetLogBackupTC(backup *v1alpha1.Backup) (*v1alpha1.TidbCluster, error) {
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

// refreshLogBackupProgress updates log backup progress periodically using the new unified approach.
func (bt *backupTracker) refreshLogBackupProgress(ns, name string) {
	ticker := time.NewTicker(refreshCheckpointTsPeriod)
	defer ticker.Stop()
	
	logkey := genLogBackupKey(ns, name)

	for range ticker.C {
		bt.operateLock.RLock()
		dep, exist := bt.logBackups[logkey]
		bt.operateLock.RUnlock()
		
		if !exist {
			return
		}
		
		// Get Backup CR
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
		
		// Check if should stop tracking
		if backup.DeletionTimestamp != nil || 
		   backup.Status.Phase == v1alpha1.BackupComplete ||
		   backup.Status.Phase == v1alpha1.BackupStopped {
			klog.Infof("log backup %s/%s is being deleted/complete/stopped, will remove %s from tracker", ns, name, logkey)
			bt.removeLogBackup(ns, name)
			return
		}
		
		if backup.Status.Phase != v1alpha1.BackupRunning {
			klog.Infof("log backup %s/%s is not running, will skip to the next time refresh", ns, name)
			continue
		}
		
		// Use the new unified refresh logic
		bt.doRefreshLogBackupState(backup, dep)
	}
}

// doRefreshLogBackupState refreshes the complete log backup state using batch queries
func (bt *backupTracker) doRefreshLogBackupState(backup *v1alpha1.Backup, dep *trackDepends) {
	ns := backup.Namespace
	name := backup.Name
	
	// Get or create etcd client
	if dep.etcdClient == nil {
		etcdCli, err := bt.deps.PDControl.GetPDEtcdClient(pdapi.Namespace(dep.tc.Namespace), dep.tc.Name,
			dep.tc.IsTLSClusterEnabled(), pdapi.ClusterRef(dep.tc.Spec.ClusterDomain))
		if err != nil {
			klog.Errorf("get log backup %s/%s etcd client error %v", ns, name, err)
			return
		}
		dep.etcdClient = etcdCli
	}
	
	// Query all keys from etcd
	state, err := bt.queryAllLogBackupKeys(dep.etcdClient, name)
	if err != nil {
		klog.Errorf("refresh log backup %s/%s state failed: %v", ns, name, err)
		return
	}
	
	// Update cache
	dep.mutex.Lock()
	dep.state = state
	dep.lastRefresh = time.Now()
	dep.mutex.Unlock()
	
	// Only update checkpoint here, state sync will be handled by SyncLogBackupState if needed
	if state.CheckpointTS > 0 {
		ckTS := strconv.FormatUint(state.CheckpointTS, 10)
		klog.V(4).Infof("update log backup %s/%s checkpointTS %s", ns, name, ckTS)
		
		updateStatus := &controller.BackupUpdateStatus{
			LogCheckpointTs: &ckTS,
			TimeSynced:      &metav1.Time{Time: time.Now()},
		}
		err = bt.statusUpdater.Update(backup, nil, updateStatus)
		if err != nil {
			klog.Errorf("update log backup %s/%s checkpointTS %s failed %v", ns, name, ckTS, err)
			return
		}
	}
}

func genLogBackupKey(ns, name string) string {
	return fmt.Sprintf("%s.%s", ns, name)
}

// getDependency gets the trackDepends for a backup, returns nil if not found
func (bt *backupTracker) getDependency(logkey string) *trackDepends {
	bt.operateLock.RLock()
	defer bt.operateLock.RUnlock()
	return bt.logBackups[logkey]
}

// GetLogBackupState gets the cached state or queries from etcd
func (bt *backupTracker) GetLogBackupState(backup *v1alpha1.Backup) (*LogBackupState, error) {
	ns := backup.Namespace
	name := backup.Name
	logkey := genLogBackupKey(ns, name)
	
	bt.operateLock.RLock()
	dep, exist := bt.logBackups[logkey]
	bt.operateLock.RUnlock()
	
	if !exist {
		return nil, fmt.Errorf("log backup %s/%s not found in tracker", ns, name)
	}
	
	// Check cache validity with read lock
	dep.mutex.RLock()
	if dep.state != nil && time.Since(dep.lastRefresh) < refreshCheckpointTsPeriod {
		// Return cached state if it's fresh enough (within the refresh period)
		// This ensures we only query etcd once per minute via refreshLogBackupProgress
		state := dep.state
		dep.mutex.RUnlock()
		return state, nil
	}
	dep.mutex.RUnlock()
	
	// State is stale, need to refresh
	return bt.RefreshLogBackupState(backup)
}

// RefreshLogBackupState forces a refresh of the log backup state
func (bt *backupTracker) RefreshLogBackupState(backup *v1alpha1.Backup) (*LogBackupState, error) {
	ns := backup.Namespace
	name := backup.Name
	logkey := genLogBackupKey(ns, name)
	
	bt.operateLock.Lock()
	dep, exist := bt.logBackups[logkey]
	bt.operateLock.Unlock()
	
	if !exist {
		return nil, fmt.Errorf("log backup %s/%s not found in tracker", ns, name)
	}
	
	// Get or create etcd client
	if dep.etcdClient == nil {
		etcdCli, err := bt.deps.PDControl.GetPDEtcdClient(pdapi.Namespace(dep.tc.Namespace), dep.tc.Name,
			dep.tc.IsTLSClusterEnabled(), pdapi.ClusterRef(dep.tc.Spec.ClusterDomain))
		if err != nil {
			return nil, fmt.Errorf("get etcd client failed: %v", err)
		}
		dep.etcdClient = etcdCli
	}
	
	// Query all keys from etcd
	state, err := bt.queryAllLogBackupKeys(dep.etcdClient, name)
	if err != nil {
		return nil, err
	}
	
	// Update cache
	dep.mutex.Lock()
	dep.state = state
	dep.lastRefresh = time.Now()
	dep.mutex.Unlock()
	
	return state, nil
}

// SyncLogBackupState syncs the log backup state and returns whether the sync can be skipped
func (bt *backupTracker) SyncLogBackupState(backup *v1alpha1.Backup) (bool, error) {
	ns := backup.Namespace
	name := backup.Name
	logkey := genLogBackupKey(ns, name)
	
	// Get dependency
	dep := bt.getDependency(logkey)
	if dep == nil {
		return false, fmt.Errorf("log backup %s/%s not found in tracker", ns, name)
	}
	
	// Get the current state
	state, err := bt.GetLogBackupState(backup)
	if err != nil {
		return false, err
	}
	
	// Check if info key exists
	if !state.InfoExists {
		command := v1alpha1.ParseLogBackupSubcommand(backup)
		if command == v1alpha1.LogStopCommand {
			// After stop, key not existing is normal
			return true, nil
		}
		// For other commands, key not existing is an error
		return false, fmt.Errorf("log backup key not found")
	}
	
	// Handle error pause
	if state.IsPaused && state.PauseReason != "" && state.PauseReason != "manual" {
		command := v1alpha1.ParseLogBackupSubcommand(backup)
		klog.Errorf("log backup %s/%s is paused due to error: %s", ns, name, state.PauseReason)
		
		bt.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Command: command,
			Type:    v1alpha1.BackupRetryTheFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "LogBackupErrorPaused",
			Message: fmt.Sprintf("Log backup paused due to error: %s", state.PauseReason),
		}, &controller.BackupUpdateStatus{
			TimeSynced: &metav1.Time{Time: time.Now()},
		})
		
		return false, fmt.Errorf("log backup paused due to error: %s", state.PauseReason)
	}
	
	// Check state consistency with double-check mechanism
	expectedCommand := v1alpha1.ParseLogBackupSubcommand(backup)
	isConsistent := isCommandConsistentWithKernelState(expectedCommand, state.KernelState)
	
	// Lock to protect inconsistencyDetected flag and command tracking
	dep.mutex.Lock()
	defer dep.mutex.Unlock()
	
	// Reset inconsistency flag if command changed to avoid cross-command interference
	if dep.lastCommand != expectedCommand {
		if dep.inconsistencyDetected {
			klog.V(4).Infof("log backup %s/%s command changed from %s to %s, resetting inconsistency flag", 
				ns, name, dep.lastCommand, expectedCommand)
		}
		dep.inconsistencyDetected = false
		dep.lastCommand = expectedCommand
	}
	
	if !isConsistent {
		if !dep.inconsistencyDetected {
			// First time detecting inconsistency for this command - flag it but don't correct yet
			dep.inconsistencyDetected = true
			klog.V(4).Infof("log backup %s/%s inconsistency detected (expected=%s, kernel=%s), will verify next reconcile", 
				ns, name, expectedCommand, state.KernelState)
			return true, nil // Skip correction this time
		} else {
			// Second time still inconsistent for same command - confirmed real issue, need to correct
			actualCommand := getCommandForKernelState(state.KernelState)
			klog.Infof("log backup %s/%s persistent inconsistency confirmed, correcting from %s to %s", 
				ns, name, expectedCommand, actualCommand)
			
			bt.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Command: actualCommand,
				Type:    v1alpha1.BackupComplete,
				Status:  corev1.ConditionTrue,
				Reason:  "LogBackupKernelSync",
				Message: fmt.Sprintf("Synced with kernel state: %s", state.KernelState),
			}, &controller.BackupUpdateStatus{
				TimeSynced: &metav1.Time{Time: time.Now()},
			})
			
			dep.inconsistencyDetected = false // Clear flag after correction
		}
	} else {
		// State is consistent
		if dep.inconsistencyDetected {
			// Previously inconsistent, now consistent - was likely a cache delay
			klog.V(4).Infof("log backup %s/%s inconsistency resolved, was likely a cache delay", ns, name)
			dep.inconsistencyDetected = false
		}
	}
	
	// Note: Checkpoint update is handled by refreshLogBackupProgress periodically
	// We don't update it here to avoid duplicate updates
	
	return true, nil
}

// StopTrackLogBackupProgress stops tracking a log backup
func (bt *backupTracker) StopTrackLogBackupProgress(backup *v1alpha1.Backup) {
	ns := backup.Namespace
	name := backup.Name
	logkey := genLogBackupKey(ns, name)
	
	bt.operateLock.Lock()
	defer bt.operateLock.Unlock()
	
	if dep, exist := bt.logBackups[logkey]; exist {
		// Close etcd client if exists
		if dep.etcdClient != nil {
			dep.etcdClient.Close()
		}
		delete(bt.logBackups, logkey)
		klog.Infof("stopped tracking log backup %s/%s", ns, name)
	}
}

// queryAllLogBackupKeys queries all log backup related keys from etcd in parallel
func (bt *backupTracker) queryAllLogBackupKeys(client pdapi.PDEtcdClient, name string) (*LogBackupState, error) {
	state := &LogBackupState{
		LastQueryTime: time.Now(),
	}
	
	// Define all keys to query
	keys := map[string]string{
		"checkpoint": path.Join(streamKeyPrefix, taskCheckpointPath, name),
		"info":       path.Join(streamKeyPrefix, taskInfoPath, name),
		"pause":      path.Join(streamKeyPrefix, taskPausePath, name),
		"error":      path.Join(streamKeyPrefix, taskLastErrorPath, name),
	}
	
	// Query results channel
	type queryResult struct {
		key   string
		kvs   []*pdapi.KeyValue
		err   error
	}
	
	resultCh := make(chan queryResult, len(keys))
	var wg sync.WaitGroup
	
	// Launch parallel queries
	for keyType, keyPath := range keys {
		wg.Add(1)
		go func(kt, kp string) {
			defer wg.Done()
			kvs, err := client.Get(kp, true)
			resultCh <- queryResult{key: kt, kvs: kvs, err: err}
		}(keyType, keyPath)
	}
	
	wg.Wait()
	close(resultCh)
	
	// Process query results
	for result := range resultCh {
		if result.err != nil {
			klog.Warningf("query etcd key type %s failed: %v", result.key, result.err)
			// Don't fail on individual key errors, continue processing
			continue
		}
		
		switch result.key {
		case "checkpoint":
			if len(result.kvs) > 0 {
				state.CheckpointTS = binary.BigEndian.Uint64(result.kvs[0].Value)
			}
		case "info":
			state.InfoExists = len(result.kvs) > 0
		case "pause":
			if len(result.kvs) > 0 {
				state.IsPaused = true
				// Parse pause reason - this will need to be implemented based on pause info structure
				state.PauseReason = bt.parsePauseReason(result.kvs[0].Value)
			}
		case "error":
			if len(result.kvs) > 0 {
				state.LastError = string(result.kvs[0].Value)
			}
		}
	}
	
	// Derive kernel state
	state.KernelState = getLogBackupKernelState(state.IsPaused)
	
	return state, nil
}

// parsePauseReason parses the pause reason from raw pause data
func (bt *backupTracker) parsePauseReason(rawData []byte) string {
	if len(rawData) == 0 {
		// V1 pause format - manual pause
		return "manual"
	}
	
	// V2 pause format - parse JSON
	pauseInfo, err := NewPauseV2Info(rawData)
	if err != nil {
		klog.Warningf("failed to parse pause info: %v", err)
		return "unknown"
	}
	
	if pauseInfo.Severity == SeverityManual {
		return "manual"
	}
	
	// Parse error message for error pause
	errMsg, err := pauseInfo.ParseError()
	if err != nil {
		klog.Warningf("failed to parse pause error: %v", err)
		return "error"
	}
	
	return errMsg
}

// Helper functions for kernel state management

// getLogBackupKernelState returns the kernel state based on pause status
func getLogBackupKernelState(paused bool) LogBackupKernelState {
	if paused {
		return LogBackupKernelPaused
	}
	return LogBackupKernelRunning
}

// isCommandConsistentWithKernelState checks if the command is consistent with kernel state
func isCommandConsistentWithKernelState(command v1alpha1.LogSubCommandType, kernelState LogBackupKernelState) bool {
	switch kernelState {
	case LogBackupKernelRunning:
		// Running state is consistent with Start or Resume commands
		return command == v1alpha1.LogStartCommand || command == v1alpha1.LogResumeCommand
	case LogBackupKernelPaused:
		// Paused state is consistent with Pause command
		return command == v1alpha1.LogPauseCommand
	default:
		return false
	}
}

// getCommandForKernelState returns the appropriate command for a kernel state
func getCommandForKernelState(kernelState LogBackupKernelState) v1alpha1.LogSubCommandType {
	switch kernelState {
	case LogBackupKernelPaused:
		return v1alpha1.LogPauseCommand
	case LogBackupKernelRunning:
		return v1alpha1.LogResumeCommand
	default:
		return v1alpha1.LogUnknownCommand
	}
}
