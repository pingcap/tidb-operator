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
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	backupUtil "github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type Options struct {
	backupUtil.GenericOptions
	// Prepare to restore data. It's used in volume-snapshot mode.
	Prepare bool
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
	args = append(args, fmt.Sprintf("--pd=%s-pd.%s:2379", restore.Spec.BR.Cluster, clusterNamespace))
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

		if fullBackupArgs, err := backupUtil.GenStorageArgsForFlag(restore.Spec.PitrFullBackupStorageProvider, "full-backup-storage"); err != nil {
			return err
		} else {
			// parse full backup path
			args = append(args, fullBackupArgs...)
		}
		restoreType = "point"
	case string(v1alpha1.RestoreModeVolumeSnapshot):
		// Currently, we only support aws ebs volume snapshot.
		args = append(args, "--type=aws-ebs")
		if ro.Prepare {
			args = append(args, "--prepare")
			csbPath = path.Join(util.BRBinPath, "csb_restore.json")
			args = append(args, fmt.Sprintf("--output-file=%s", csbPath))
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
	klog.Infof("Running br command with args: %v", fullArgs)
	bin := path.Join(util.BRBinPath, "br")
	cmd := exec.CommandContext(ctx, bin, fullArgs...)

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

	var errMsg string
	reader := bufio.NewReader(stdOut)
	for {
		line, err := reader.ReadString('\n')
		if strings.Contains(line, "[ERROR]") {
			errMsg += line
		} else {
			if !useProgressFile {
				ro.updateProgressAccordingToBrLog(line, restore, statusUpdater)
			}
			ro.updateResolvedTSForCSB(line, restore, progressStep, statusUpdater)
		}
		klog.Info(strings.Replace(line, "\n", "", -1))
		if err != nil || io.EOF == err {
			break
		}
	}
	tmpErr, _ := ioutil.ReadAll(stdErr)
	if len(tmpErr) > 0 {
		klog.Info(string(tmpErr))
		errMsg += string(tmpErr)
	}

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("cluster %s, wait pipe message failed, errMsg %s, err: %v", ro, errMsg, err)
	}

	if csbPath != "" {
		err = ro.processCloudSnapBackup(restore, csbPath, restoreControl)
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

func (ro *Options) processCloudSnapBackup(
	restore *v1alpha1.Restore,
	csbPath string,
	restoreControl controller.RestoreControlInterface,
) error {
	data, err := os.ReadFile(csbPath)
	if err != nil {
		return fmt.Errorf("cluster %s, read the CSB file failed, path: %s, err: %v", ro, csbPath, err)
	}
	if len(restore.GetAnnotations()) == 0 {
		restore.Annotations = make(map[string]string)
	}
	klog.Infof("Get restore for cluster %s, annotations: %v", ro, restore.GetAnnotations())
	restore.Annotations[label.AnnBackupCloudSnapKey] = string(data)
	if _, err = restoreControl.UpdateRestore(restore); err != nil {
		return fmt.Errorf("cluster %s, update restore annotation for CSB failed, err: %v", ro, err)
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
