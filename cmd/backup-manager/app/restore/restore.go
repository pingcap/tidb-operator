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
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	backupUtil "github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type Options struct {
	backupUtil.GenericOptions
}

func (ro *Options) restoreData(ctx context.Context, restore *v1alpha1.Restore, statusUpdater controller.RestoreConditionUpdaterInterface) error {
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

	// gen PiTR args
	if ro.Mode == string(v1alpha1.RestoreModePiTR) {
		// init pitr restore args
		args = append(args, fmt.Sprintf("--restored-ts=%s", ro.PitrRestoredTs))

		if fullBackupArgs, err := backupUtil.GenStorageArgsForFlag(restore.Spec.PitrFullBackupStorageProvider, "full-backup-storage"); err != nil {
			return err
		} else {
			// parse full backup path
			args = append(args, fullBackupArgs...)
		}
		restoreType = "point"
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
	var errMsg string
	reader := bufio.NewReader(stdOut)
	for {
		line, err := reader.ReadString('\n')
		if strings.Contains(line, "[ERROR]") {
			errMsg += line
		} else {
			ro.updateProgressAccordingToBrLog(line, restore, statusUpdater)
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
	klog.Infof("Restore data for cluster %s successfully", ro)
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
