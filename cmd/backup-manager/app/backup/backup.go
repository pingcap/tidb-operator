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
	"time"

	"github.com/dustin/go-humanize"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	backupUtil "github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	backupConst "github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// Options contains the input arguments to the backup command
type Options struct {
	backupUtil.GenericOptions
}

// backupData generates br args and runs br binary to do the real backup work
func (bo *Options) backupData(
	ctx context.Context,
	backup *v1alpha1.Backup,
	statusUpdater controller.BackupConditionUpdaterInterface) (bool, error) {
	clusterNamespace := backup.Spec.BR.ClusterNamespace
	if backup.Spec.BR.ClusterNamespace == "" {
		clusterNamespace = backup.Namespace
	}
	args := make([]string, 0)
	args = append(args, fmt.Sprintf("--pd=%s-pd.%s:2379", backup.Spec.BR.Cluster, clusterNamespace))
	if bo.TLSCluster {
		args = append(args, fmt.Sprintf("--ca=%s", path.Join(util.ClusterClientTLSPath, corev1.ServiceAccountRootCAKey)))
		args = append(args, fmt.Sprintf("--cert=%s", path.Join(util.ClusterClientTLSPath, corev1.TLSCertKey)))
		args = append(args, fmt.Sprintf("--key=%s", path.Join(util.ClusterClientTLSPath, corev1.TLSPrivateKeyKey)))
	}

	// CloudSnapBackup is the metadata for backup TiDBCluster,
	// especially TiKV-volumes for BR to take snapshot
	var isCSB bool
	csb := os.Getenv(backupConst.EnvCloudSnapMeta)
	if csb != "" {
		isCSB = true
		klog.Infof("Running cloud-snapshot-backup with metadata: %s", csb)
		csbPath := path.Join(util.BRBinPath, "csb_backup.json")
		err := ioutil.WriteFile(csbPath, []byte(csb), 0644)
		if err != nil {
			return false, err
		}
		args = append(args, fmt.Sprintf("--volume-file=%s", csbPath))
	}

	// `options` in spec are put to the last because we want them to have higher priority than generated arguments
	dataArgs, err := constructOptions(backup)
	if err != nil {
		return false, err
	}
	args = append(args, dataArgs...)

	var backupType string
	if backup.Spec.Type == "" {
		backupType = string(v1alpha1.BackupTypeFull)
	} else {
		backupType = string(backup.Spec.Type)
	}
	fullArgs := []string{
		"backup",
		backupType,
	}
	fullArgs = append(fullArgs, args...)
	klog.Infof("Running br command with args: %v", fullArgs)
	bin := path.Join(util.BRBinPath, "br")
	cmd := exec.CommandContext(ctx, bin, fullArgs...)

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return false, fmt.Errorf("cluster %s, create stdout pipe failed, err: %v", bo, err)
	}
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return false, fmt.Errorf("cluster %s, create stderr pipe failed, err: %v", bo, err)
	}
	err = cmd.Start()
	if err != nil {
		return false, fmt.Errorf("cluster %s, execute br command failed, args: %s, err: %v", bo, fullArgs, err)
	}

	var errMsg string
	if isCSB {
		errMsg = bo.processExecOutputForCSB(backup, backupType, stdOut, statusUpdater)
	} else {
		errMsg = bo.processExecOutput(stdOut)
	}

	tmpErr, _ := ioutil.ReadAll(stdErr)
	if len(tmpErr) > 0 {
		klog.Info(string(tmpErr))
		errMsg += string(tmpErr)
	}
	err = cmd.Wait()
	if err != nil {
		return false, fmt.Errorf("cluster %s, wait pipe message failed, errMsg %s, err: %v", bo, errMsg, err)
	}

	klog.Infof("Backup data for cluster %s successfully", bo)
	return isCSB, nil
}

// processExecOutput processes the output from exec br binary
// NOTE: keep original logic for code
func (bo *Options) processExecOutput(stdOut io.ReadCloser) (errMsg string) {
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
func (bo *Options) processExecOutputForCSB(
	backup *v1alpha1.Backup,
	backupType string,
	stdOut io.ReadCloser,
	statusUpdater controller.BackupConditionUpdaterInterface) (errMsg string) {
	type progressUpdate struct {
		progress, backupSize, resolvedTs *string
	}

	started := time.Now()
	quit := make(chan struct{})
	update := make(chan *progressUpdate)
	progressFile := path.Join(util.BRBinPath, "progress.txt")
	go func() {
		ticker := time.NewTicker(constants.ProgressInterval)
		var lastUpdate string
		var completed bool
		for {
			select {
			case <-quit:
				ticker.Stop()
				return
			case val := <-update:
				if completed {
					continue
				}
				updateStatus := &controller.BackupUpdateStatus{
					Progress: val.progress,
				}
				if val.resolvedTs != nil || val.backupSize != nil {
					updateStatus.CommitTs = val.resolvedTs
					backupSize, err := strconv.ParseInt(*val.backupSize, 10, 64)
					if err == nil {
						updateStatus.BackupSize = &backupSize
						backupSizeReadable := humanize.Bytes(uint64(backupSize))
						updateStatus.BackupSizeReadable = &backupSizeReadable
					} else {
						klog.Warningf("Failed to parse BackupSize %s, %s", *val.backupSize, err.Error())
					}

					updateStatus.TimeStarted = &metav1.Time{Time: started}
					updateStatus.TimeCompleted = &metav1.Time{Time: time.Now()}
					err = statusUpdater.Update(backup, &v1alpha1.BackupCondition{
						Type:   v1alpha1.BackupComplete,
						Status: corev1.ConditionTrue,
					}, updateStatus)
					if err == nil {
						completed = true
					} else {
						klog.Warningf("Failed to update BackupUpdateStatus-Complete for cluster %s, %s", bo, err.Error())
					}
				} else {
					err := statusUpdater.Update(backup, &v1alpha1.BackupCondition{
						Type:   v1alpha1.BackupRunning,
						Status: corev1.ConditionTrue,
					}, updateStatus)
					if err != nil {
						klog.Warningf("Failed to update BackupUpdateStatus-Running for cluster %s, %s", bo, err.Error())
					}
				}
			case <-ticker.C:
				if !backupUtil.IsFileExist(progressFile) {
					continue
				}
				bs, err := ioutil.ReadFile(progressFile)
				if err != nil {
					return
				}
				go func() {
					progress := string(bs)
					klog.Infof("update progress for current: %s and last: %s", progress, lastUpdate)
					if lastUpdate != progress {
						update <- &progressUpdate{
							progress: &progress,
						}
						lastUpdate = progress
					}
				}()
			}
		}
	}()

	var line string
	var err error
	successTag := fmt.Sprintf("[\"%s backup success\"]", strings.ToUpper(backupType))
	reader := bufio.NewReader(stdOut)
	for {
		line, errMsg, err = readExecOutputLine(reader)

		if strings.Contains(line, successTag) {
			extract := strings.Split(line, successTag)[1]
			sizeStr := regexp.MustCompile(`size=\d+`).FindString(extract)
			size := strings.ReplaceAll(sizeStr, "size=", "")
			tsStr := regexp.MustCompile(`resolved_ts=\d+`).FindString(extract)
			ts := strings.ReplaceAll(tsStr, "resolved_ts=", "")
			klog.Infof("%s size: %s, resolved_ts: %s", successTag, size, ts)
			update <- &progressUpdate{
				progress:   func(s string) *string { return &s }("100%"),
				resolvedTs: &ts,
				backupSize: &size,
			}
		}

		if err != nil || io.EOF == err {
			quit <- struct{}{}
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

// constructOptions constructs options for BR
func constructOptions(backup *v1alpha1.Backup) ([]string, error) {
	args, err := backupUtil.ConstructBRGlobalOptionsForBackup(backup)
	if err != nil {
		return args, err
	}
	config := backup.Spec.BR
	if config.Concurrency != nil {
		args = append(args, fmt.Sprintf("--concurrency=%d", *config.Concurrency))
	}
	if config.RateLimit != nil {
		args = append(args, fmt.Sprintf("--ratelimit=%d", *config.RateLimit))
	}
	if config.TimeAgo != "" {
		args = append(args, fmt.Sprintf("--timeago=%s", config.TimeAgo))
	}
	if config.Checksum != nil {
		args = append(args, fmt.Sprintf("--checksum=%t", *config.Checksum))
	}
	if config.CheckRequirements != nil {
		args = append(args, fmt.Sprintf("--check-requirements=%t", *config.CheckRequirements))
	}
	args = append(args, config.Options...)
	return args, nil
}
