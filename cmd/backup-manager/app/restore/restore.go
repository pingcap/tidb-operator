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
	"strings"
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
}

func (ro *Options) restoreData(
	ctx context.Context,
	restore *v1alpha1.Restore,
	started time.Time,
	statusUpdater controller.RestoreConditionUpdaterInterface,
	restoreControl controller.RestoreControlInterface) (bool, error) {
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

	// CloudSnapBackup is the metadata for restore TiDBCluster,
	// especially TiKV-volumes for BR to restore snapshot
	var (
		isCSB   bool
		csbPath string
		ts      string
	)
	switch restore.Spec.Type {
	case v1alpha1.BackupTypeEBS, v1alpha1.BackupTypeGCEPD:
		isCSB = true
		csbPath = path.Join(util.BRBinPath, "csb_restore.json")
		args = append(args, fmt.Sprintf("--output-file=%s", csbPath))
	case v1alpha1.BackupTypeData:
		if restore.Spec.DryRun {
			args = append(args, "--dry-run=true")
		}
		restore.Spec.BR.Options = backupUtil.GetSliceExcludeOneString(restore.Spec.BR.Options, "--skip-aws")
	}

	// `options` in spec are put to the last because we want them to have higher priority than generated arguments
	dataArgs, err := constructBROptions(restore)
	if err != nil {
		return false, err
	}
	args = append(args, dataArgs...)

	var restoreType string
	if restore.Spec.Type == "" {
		restoreType = string(v1alpha1.BackupTypeFull)
	} else {
		restoreType = string(restore.Spec.Type)
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
		return false, fmt.Errorf("cluster %s, create stdout pipe failed, err: %v", ro, err)
	}
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return false, fmt.Errorf("cluster %s, create stderr pipe failed, err: %v", ro, err)
	}
	err = cmd.Start()
	if err != nil {
		return false, fmt.Errorf("cluster %s, execute br command failed, args: %s, err: %v", ro, fullArgs, err)
	}

	var errMsg string
	if isCSB {
		ts, errMsg = ro.processExecOutputForCSB(restoreType, stdOut)
	} else {
		errMsg = ro.processExecOutput(stdOut)
	}

	tmpErr, _ := ioutil.ReadAll(stdErr)
	if len(tmpErr) > 0 {
		klog.Info(string(tmpErr))
		errMsg += string(tmpErr)
	}

	err = cmd.Wait()
	if err != nil {
		return false, fmt.Errorf("cluster %s, wait pipe message failed, errMsg %s, err: %v", ro, errMsg, err)
	}

	return ro.processRestoreResult(csbPath, ts, restore, started, statusUpdater, restoreControl)
}

func (ro *Options) processRestoreResult(
	csbPath, resolvedTS string,
	restore *v1alpha1.Restore,
	started time.Time,
	statusUpdater controller.RestoreConditionUpdaterInterface,
	restoreControl controller.RestoreControlInterface) (bool, error) {
	switch restore.Spec.Type {
	case v1alpha1.BackupTypeEBS, v1alpha1.BackupTypeGCEPD:
		if !backupUtil.IsFileExist(csbPath) {
			return false, fmt.Errorf("cluster %s, the CSB file not found, path: %s", ro, csbPath)
		}
		file, err := os.Open(csbPath)
		if err != nil {
			return false, fmt.Errorf("cluster %s, open the CSB file failed, path: %s, err: %v", ro, csbPath, err)
		}
		defer file.Close()
		if bs, err := ioutil.ReadAll(file); err != nil {
			return false, fmt.Errorf("cluster %s, read the CSB file failed, path: %s, err: %v", ro, csbPath, err)
		} else {
			if len(restore.GetAnnotations()) == 0 {
				restore.Annotations = make(map[string]string)
			}
			klog.Infof("Get restore for cluster %s, annotations: %v", ro, restore.GetAnnotations())
			restore.Annotations[label.AnnBackupCloudSnapKey] = string(bs)
			if _, err = restoreControl.UpdateRestore(restore); err != nil {
				return false, fmt.Errorf("cluster %s, update restore annotation for CSB failed, err: %v", ro, err)
			}

			updateStatus := &controller.RestoreUpdateStatus{
				TimeStarted:   &metav1.Time{Time: started},
				TimeCompleted: &metav1.Time{Time: time.Now()},
				CommitTs:      &resolvedTS,
			}
			if err := statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
				Type:   v1alpha1.RestoreVolumeComplete,
				Status: corev1.ConditionTrue,
			}, updateStatus); err != nil {
				return false, fmt.Errorf("cluster %s, update restore status volume-complete failed, err: %v", ro, err)
			}
			klog.Infof("Restore TiKV-volumes for cluster %s successfully", ro)
			return true, nil
		}

	case v1alpha1.BackupTypeData:
		updateStatus := &controller.RestoreUpdateStatus{
			TimeCompleted: &metav1.Time{Time: time.Now()},
		}
		if restore.Status.TimeStarted.IsZero() {
			updateStatus.TimeStarted = &metav1.Time{Time: started}
		}
		if err := statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:   v1alpha1.RestoreDataComplete,
			Status: corev1.ConditionTrue,
		}, updateStatus); err != nil {
			return false, fmt.Errorf("cluster %s, update restore status data-complete failed, err: %v", ro, err)
		}
		klog.Infof("Restore TiKV-data for cluster %s successfully", ro)
		return true, nil

	default:
		klog.Infof("Restore data for cluster %s successfully", ro)
	}
	return false, nil
}

// processExecOutput processes the output from exec br binary
// NOTE: keep original logic for code
func (ro *Options) processExecOutput(stdOut io.ReadCloser) (errMsg string) {
	var err error
	reader := bufio.NewReader(stdOut)
	for {
		_, errMsg, err = readExecOutputLine(reader)
		if err != nil || io.EOF == err {
			break
		}
	}
	return
}

// processExecOutputForCSB processes the output from exec br binary for CloudSnapBackup
// NOTE: distinguish between previous code logic
func (ro *Options) processExecOutputForCSB(restoreType string, stdOut io.ReadCloser) (ts, errMsg string) {
	var line string
	var err error
	successTag := fmt.Sprintf("[\"%s restore success\"]", strings.ToUpper(restoreType))
	reader := bufio.NewReader(stdOut)
	for {
		line, errMsg, err = readExecOutputLine(reader)

		if strings.Contains(line, successTag) {
			extract := strings.Split(line, successTag)[1]
			tsStr := regexp.MustCompile(`resolved_ts=\d+`).FindString(extract)
			ts = strings.ReplaceAll(tsStr, "resolved_ts=", "")
			klog.Infof("%s resolved_ts: %s", successTag, ts)
		}

		if err != nil || io.EOF == err {
			break
		}
	}
	return
}

func readExecOutputLine(reader *bufio.Reader) (line, errMsg string, err error) {
	line, err = reader.ReadString('\n')
	if strings.Contains(line, "[ERROR]") {
		errMsg += line
	}
	klog.Info(strings.Replace(line, "\n", "", -1))
	return
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
