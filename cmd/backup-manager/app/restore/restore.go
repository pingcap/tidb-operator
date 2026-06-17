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

package restore

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/brlog"
	backupUtil "github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	pkgutil "github.com/pingcap/tidb-operator/pkg/backup/util"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
)

const (
	// replicationStatusSubPrefix is the fixed sub-prefix BR uses to
	// lay out replication status under the log-backup storage.
	// Hardcoded here (not exposed via API) per spec §6: this is a BR
	// call detail and shouldn't leak into the CRD.
	replicationStatusSubPrefix = "crr-checkpoint"

	// replicationPiTRConcurrency is the parallelism BR uses when
	// applying compacted log files. Spec §6.
	replicationPiTRConcurrency = 1024

	replicationMetadataDownloadBatchSize = 512
)

var brBinPath = util.BRBinPath

// replicationBRFlags returns the BR command-line flags that are
// specific to replication restore phases. Returns nil for phase 0
// (= not a replication restore), in which case the caller's
// `append(args, replicationBRFlags(0)...)` is a no-op and standard
// PiTR semantics apply unchanged. Spec §6.
func replicationBRFlags(phase int) []string {
	if phase == 0 {
		return nil
	}
	return []string{
		fmt.Sprintf("--restore-phase=%d", phase),
		fmt.Sprintf("--replication-status-sub-prefix=%s", replicationStatusSubPrefix),
		fmt.Sprintf("--pitr-concurrency=%d", replicationPiTRConcurrency),
		fmt.Sprintf("--metadata-download-batch-size=%d", replicationMetadataDownloadBatchSize),
		"--retain-latest-mvcc-version",
	}
}

type Options struct {
	backupUtil.GenericOptions
	// Prepare to restore data. It's used in volume-snapshot mode.
	Prepare bool
	// TargetAZ indicates which az the volume snapshots restore to. It's used in volume-snapshot mode.
	TargetAZ string
	// UseFSR to indicate if use FSR for TiKV data volumes during EBS snapshot restore
	UseFSR bool
	// Abort indicates whether to abort/cleanup a failed restore operation
	Abort bool
	// ReplicationPhase indicates which phase of a two-phase replication
	// restore this backup-manager invocation handles. Valid values:
	//   1 — snapshot-restore phase (status writes suppressed by Option B
	//       no-op wrapper; controller is sole writer of status.Phase)
	//   2 — log-restore phase (backup-manager directly writes Phase /
	//       Running / Complete / Failed; no wrapper)
	// The zero value (0) means this is NOT a replication restore — the
	// CLI flag was not passed and standard PiTR / snapshot semantics
	// apply. See spec §3 (Option B) and §6 (BR CLI surface).
	ReplicationPhase int
}

func (ro *Options) restoreData(
	ctx context.Context,
	restore *v1alpha1.Restore,
	statusUpdater controller.RestoreConditionUpdaterInterface,
	restoreControl controller.RestoreControlInterface,
) error {
	clusterNamespace := restore.Spec.BR.ClusterNamespace
	if restore.Spec.BR.ClusterNamespace == "" {
		clusterNamespace = restore.Namespace
	}
	args := make([]string, 0)
	args = append(args, fmt.Sprintf("--pd=%s-pd.%s:%d", restore.Spec.BR.Cluster, clusterNamespace, v1alpha1.DefaultPDClientPort))
	if ro.TLSCluster {
		args = append(args, fmt.Sprintf("--ca=%s", path.Join(util.ClusterClientTLSPath, corev1.ServiceAccountRootCAKey)))
		args = append(args, fmt.Sprintf("--cert=%s", path.Join(util.ClusterClientTLSPath, corev1.TLSCertKey)))
		args = append(args, fmt.Sprintf("--key=%s", path.Join(util.ClusterClientTLSPath, corev1.TLSPrivateKeyKey)))
	}
	// `options` in spec are put to the last because we want them to have higher priority than generated arguments
	dataArgs, err := constructBROptions(restore)
	if err != nil {
		return err
	}
	args = append(args, dataArgs...)

	var restoreType string
	if restore.Spec.Type == "" {
		restoreType = string(v1alpha1.BackupTypeFull)
	} else {
		restoreType = string(restore.Spec.Type)
	}

	var (
		csbPath      string
		progressStep string
	)

	progressFile := "progress.txt"
	useProgressFile := false

	// gen args PiTR and volume-snapshot.
	switch ro.Mode {
	case string(v1alpha1.RestoreModePiTR):
		// init pitr restore args
		args = append(args, fmt.Sprintf("--restored-ts=%s", ro.PitrRestoredTs))

		fullBackupArgs, err := pkgutil.GenStorageArgsForFlag(restore.Spec.PitrFullBackupStorageProvider, "full-backup-storage")
		if err != nil {
			if restore.Spec.LogRestoreStartTs == "" {
				return fmt.Errorf("error: Either pitrFullBackupStorageProvider or logRestoreStartTs option needs to be passed in pitr mode")
			}
			args = append(args, fmt.Sprintf("--start-ts=%s", restore.Spec.LogRestoreStartTs))
		} else {
			args = append(args, fullBackupArgs...)
		}
		args = append(args, replicationBRFlags(ro.ReplicationPhase)...)
		restoreType = "point"
	case string(v1alpha1.RestoreModeVolumeSnapshot):
		// Currently, we only support aws ebs volume snapshot.
		args = append(args, "--type=aws-ebs")
		if ro.Prepare {
			args = append(args, "--prepare")
			csbPath = path.Join(util.BRBinPath, "csb_restore.json")
			args = append(args, fmt.Sprintf("--output-file=%s", csbPath))
			args = append(args, fmt.Sprintf("--target-az=%s", ro.TargetAZ))
			if ro.UseFSR {
				args = append(args, "--use-fsr=true")
			} else {
				args = append(args, "--use-fsr=false")
			}
			progressStep = "Volume Restore"
		} else {
			progressStep = "Data Restore"
		}
		useProgressFile = true
	}

	fullArgs := []string{
		"restore",
		restoreType,
	}
	fullArgs = append(fullArgs, args...)
	fullArgs = append(fullArgs, "--log-format=json")

	var (
		progressWg     sync.WaitGroup
		progressCancel context.CancelFunc
	)
	if useProgressFile {
		progressCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		progressCancel = cancel

		progressWg.Add(1)
		go func() {
			defer progressWg.Done()
			ro.updateProgressFromFile(progressCtx.Done(), restore, progressFile, progressStep, statusUpdater)
		}()
	}

	observer := newRestoreLogObserver(restore, statusUpdater)
	err = ro.brRestoreCommandRunWithLogObserver(ctx, fullArgs, func(line string) {
		if !useProgressFile {
			ro.updateProgressAccordingToBrLog(line, restore, statusUpdater)
		}
		ro.updateResolvedTSForCSB(line, restore, progressStep, statusUpdater)
	}, observer.observeLine)
	if err != nil {
		observer.updateLockBlockerAfterFailure()
		return err
	}
	observer.clearLockBlocker()

	if csbPath != "" {
		err = ro.processCloudSnapBackup(ctx, restore, csbPath, restoreControl)
		if err != nil {
			return err
		}
	}

	// When using progress file, we may not get the last progress update.
	// So we need to update the progress to 100% here since the restore is done.
	if useProgressFile {
		progressCancel()
		progressWg.Wait()

		progress := 100.0
		if err := statusUpdater.Update(restore, nil, &controller.RestoreUpdateStatus{
			ProgressStep:       &progressStep,
			Progress:           &progress,
			ProgressUpdateTime: &metav1.Time{Time: time.Now()},
		}); err != nil {
			klog.Errorf("update restore %s progress error %v", ro, err)
		}
	}

	klog.Infof("Restore data for cluster %s successfully", ro)
	return nil
}

func (ro *Options) brRestoreCommandRunWithLogObserver(
	ctx context.Context,
	fullArgs []string,
	stdoutCallback func(line string),
	logObserver func(line string),
) error {
	if len(fullArgs) == 0 {
		return fmt.Errorf("command is invalid, fullArgs: %v", fullArgs)
	}
	klog.Infof("Running br command with args: %v", fullArgs)
	bin := path.Join(brBinPath, "br")
	cmd := exec.Command(bin, fullArgs...)

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("cluster %s, create stdout pipe failed, err: %v", ro, err)
	}
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("cluster %s, create stderr pipe failed, err: %v", ro, err)
	}
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("cluster %s, execute br command failed, args: %s, err: %v", ro, fullArgs, err)
	}
	go backupUtil.GracefullyShutDownSubProcess(ctx, cmd)

	var errMsg string
	stdOutLineCh := make(chan string)
	stdOutErrCh := make(chan error, 1)
	go backupUtil.ReadLinesToChannel(stdOut, stdOutLineCh, stdOutErrCh)

	stdErrLineCh := make(chan string)
	stdErrErrCh := make(chan error, 1)
	go backupUtil.ReadLinesToChannel(stdErr, stdErrLineCh, stdErrErrCh)

	stdOutOpen, stdErrOpen := true, true
	for stdOutOpen || stdErrOpen {
		select {
		case line, ok := <-stdOutLineCh:
			if !ok {
				stdOutOpen = false
				stdOutLineCh = nil
				continue
			}
			processBRRestoreCommandStdoutLine(line, &errMsg, stdoutCallback, logObserver)
		case line, ok := <-stdErrLineCh:
			if !ok {
				stdErrOpen = false
				stdErrLineCh = nil
				continue
			}
			processBRRestoreCommandLogLine(line, &errMsg, logObserver)
		}
	}

	if err := <-stdOutErrCh; err != nil {
		klog.Errorf("read stdout error: %s", err.Error())
	}
	if err := <-stdErrErrCh; err != nil {
		klog.Errorf("read stderr error: %s", err.Error())
	}

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("cluster %s, wait pipe message failed, errMsg %s, err: %v", ro, errMsg, err)
	}

	klog.Infof("Run br commond %v for cluster %s successfully", fullArgs, ro)
	return nil
}

func processBRRestoreCommandStdoutLine(
	line string,
	errMsg *string,
	stdoutCallback func(line string),
	logObserver func(line string),
) {
	if !brlog.IsErrorLine(line) && stdoutCallback != nil {
		stdoutCallback(line)
	}
	processBRRestoreCommandLogLine(line, errMsg, logObserver)
}

func processBRRestoreCommandLogLine(
	line string,
	errMsg *string,
	logObserver func(line string),
) {
	if brlog.IsErrorLine(line) {
		*errMsg += line + "\n"
	}
	if logObserver != nil {
		logObserver(line)
	}
	klog.Info(line)
}

type restoreLogObserver struct {
	restore       *v1alpha1.Restore
	statusUpdater controller.RestoreConditionUpdaterInterface
	lockBlocker   *v1alpha1.BRLockBlocker
}

func newRestoreLogObserver(
	restore *v1alpha1.Restore,
	statusUpdater controller.RestoreConditionUpdaterInterface,
) *restoreLogObserver {
	return &restoreLogObserver{
		restore:       restore,
		statusUpdater: statusUpdater,
	}
}

func (o *restoreLogObserver) observeLine(line string) {
	event := brlog.ParseLine(line)
	switch event.Type {
	case brlog.EventOperationStarted:
		if event.Operation == nil {
			return
		}
		if event.Operation.ObservedAt.IsZero() {
			event.Operation.ObservedAt = metav1.Now()
		}
		if err := o.statusUpdater.Update(o.restore, nil, &controller.RestoreUpdateStatus{
			BROperation: event.Operation,
		}); err != nil {
			klog.Errorf("Failed to update BR operation status for restore %s/%s, %v", o.restore.Namespace, o.restore.Name, err)
		}
	case brlog.EventLockConflict:
		o.lockBlocker = event.LockBlocker
	}
}

func (o *restoreLogObserver) updateLockBlockerAfterFailure() {
	if o.lockBlocker != nil {
		if err := o.statusUpdater.Update(o.restore, nil, &controller.RestoreUpdateStatus{
			LockBlocker: o.lockBlocker,
		}); err != nil {
			klog.Errorf("Failed to update BR lock blocker status for restore %s/%s, %v", o.restore.Namespace, o.restore.Name, err)
		}
		return
	}
	o.clearLockBlocker()
}

func (o *restoreLogObserver) clearLockBlocker() {
	if err := o.statusUpdater.Update(o.restore, nil, &controller.RestoreUpdateStatus{
		ClearLockBlocker: ptr.To(true),
	}); err != nil {
		klog.Errorf("Failed to clear BR lock blocker status for restore %s/%s, %v", o.restore.Namespace, o.restore.Name, err)
	}
}

// copy the restore meta to remote storage since k8s has limit to handle massive data pass between pods
func (ro *Options) processCloudSnapBackup(
	ctx context.Context,
	restore *v1alpha1.Restore,
	csbPath string,
	restoreControl controller.RestoreControlInterface,
) error {
	klog.Infof("Get restore meta file %s for cluster %s", csbPath, ro)
	contents, err := os.ReadFile(csbPath)
	if err != nil {
		klog.Errorf("read metadata file %s failed, err: %s", csbPath, err)
		return err
	}
	// write a file into external storage
	klog.Infof("save the restore meta to external storage")
	externalStorage, err := pkgutil.NewStorageBackend(restore.Spec.StorageProvider, &pkgutil.StorageCredential{})
	if err != nil {
		return err
	}

	err = externalStorage.WriteAll(ctx, constants.ClusterRestoreMeta, contents, nil)
	if err != nil {
		return err
	}
	return nil
}

func constructBROptions(restore *v1alpha1.Restore) ([]string, error) {
	args, err := backupUtil.ConstructBRGlobalOptionsForRestore(restore)
	if err != nil {
		return nil, err
	}
	config := restore.Spec.BR
	if config.Concurrency != nil {
		args = append(args, fmt.Sprintf("--concurrency=%d", *config.Concurrency))
	}
	if config.Checksum != nil {
		args = append(args, fmt.Sprintf("--checksum=%t", *config.Checksum))
	}
	if config.CheckRequirements != nil {
		args = append(args, fmt.Sprintf("--check-requirements=%t", *config.CheckRequirements))
	}
	if config.RateLimit != nil {
		args = append(args, fmt.Sprintf("--ratelimit=%d", *config.RateLimit))
	}
	if config.OnLine != nil {
		args = append(args, fmt.Sprintf("--online=%t", *config.OnLine))
	}
	args = append(args, config.Options...)
	return args, nil
}

// updateProgressAccordingToBrLog update restore progress according to the br log.
func (ro *Options) updateProgressAccordingToBrLog(line string, restore *v1alpha1.Restore, statusUpdater controller.RestoreConditionUpdaterInterface) {
	step, progress := backupUtil.ParseRestoreProgress(line)
	if step != "" {
		fvalue, progressUpdateErr := strconv.ParseFloat(progress, 64)
		if progressUpdateErr != nil {
			klog.Errorf("parse restore %s progress string value %s to float error %v", ro, progress, progressUpdateErr)
			fvalue = 0
		}
		klog.Infof("update restore %s step %s progress %s float value %f", ro, step, progress, fvalue)
		progressUpdateErr = statusUpdater.Update(restore, nil, &controller.RestoreUpdateStatus{
			ProgressStep:       &step,
			Progress:           &fvalue,
			ProgressUpdateTime: &metav1.Time{Time: time.Now()},
		})
		if progressUpdateErr != nil {
			klog.Errorf("update restore %s progress error %v", ro, progressUpdateErr)
		}
	}
}

func (ro *Options) updateResolvedTSForCSB(
	line string,
	restore *v1alpha1.Restore,
	progressStep string,
	statusUpdater controller.RestoreConditionUpdaterInterface,
) {
	const successTag = "EBS restore success"

	if strings.Contains(line, successTag) {
		extract := strings.Split(line, successTag)[1]
		tsStr := regexp.MustCompile(`resolved_ts=\d+`).FindString(extract)
		ts := strings.ReplaceAll(tsStr, "resolved_ts=", "")
		klog.Infof("%s resolved_ts: %s", successTag, ts)

		progress := 100.0
		if err := statusUpdater.Update(restore, nil, &controller.RestoreUpdateStatus{
			CommitTs:           &ts,
			ProgressStep:       &progressStep,
			Progress:           &progress,
			ProgressUpdateTime: &metav1.Time{Time: time.Now()},
		}); err != nil {
			klog.Errorf("update restore %s resolved ts error %v", ro, err)
		}
	}
}

func (ro *Options) updateProgressFromFile(
	stopCh <-chan struct{},
	backup *v1alpha1.Restore,
	progressFile string,
	progressStep string,
	statusUpdater controller.RestoreConditionUpdaterInterface,
) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			data, err := os.ReadFile(progressFile)
			if err != nil {
				if !os.IsNotExist(err) {
					klog.Warningf("Failed to read progress file %s: %v", progressFile, err)
				}
				continue
			}
			progressStr := strings.TrimSpace(string(data))
			progressStr = strings.TrimSuffix(progressStr, "%")
			if progressStr == "" {
				continue
			}
			progress, err := strconv.ParseFloat(progressStr, 64)
			if err != nil {
				klog.Warningf("Failed to parse progress %s, err: %v", string(data), err)
				continue
			}
			if err := statusUpdater.Update(backup, nil, &controller.RestoreUpdateStatus{
				ProgressStep:       &progressStep,
				Progress:           &progress,
				ProgressUpdateTime: &metav1.Time{Time: time.Now()},
			}); err != nil {
				klog.Errorf("Failed to update BackupUpdateStatus for cluster %s, %v", ro, err)
			}
		case <-stopCh:
			return
		}
	}
}
