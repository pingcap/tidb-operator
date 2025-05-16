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
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	corev1alpha1 "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	backupUtil "github.com/pingcap/tidb-operator/cmd/tidb-backup-manager/app/util"
)

// Options contains the input arguments to the backup command
type Options struct {
	backupUtil.GenericOptions
	// PDAddress is the address of the PD, for example: db-pd.tidb12345:2379
	PDAddress string
}

// backupData generates br args and runs br binary to do the real backup work
func (bo *Options) backupData(
	ctx context.Context,
	backup *v1alpha1.Backup,
) error {
	var backupType string
	if backup.Spec.Type == "" {
		backupType = string(v1alpha1.BackupTypeFull)
	} else {
		backupType = string(backup.Spec.Type)
	}
	specificArgs := []string{
		"backup",
		backupType,
	}

	var logCallback func(line string)
	if bo.Mode == string(v1alpha1.BackupModeSnapshot) {
		if backup.Spec.CommitTs != "" {
			specificArgs = append(specificArgs, fmt.Sprintf("--backupts=%s", backup.Spec.CommitTs))
		}
	}

	fullArgs, err := bo.backupCommandTemplate(backup, specificArgs, false)
	if err != nil {
		return err
	}
	return bo.brCommandRunWithLogCallback(ctx, fullArgs, logCallback)
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

// doStartLogBackup generates br args about log backup start and runs br binary to do the real backup work.
func (bo *Options) doStartLogBackup(ctx context.Context, backup *v1alpha1.Backup) error {
	specificArgs := []string{
		"log",
		"start",
		fmt.Sprintf("--task-name=%s", backup.Name),
	}
	if bo.CommitTS != "" && bo.CommitTS != "0" {
		specificArgs = append(specificArgs, fmt.Sprintf("--start-ts=%s", bo.CommitTS))
	}
	fullArgs, err := bo.backupCommandTemplate(backup, specificArgs, false)
	if err != nil {
		return err
	}
	return bo.brCommandRun(ctx, fullArgs)
}

// doResumeLogBackup generates br args about log backup resume and runs br binary to do the real backup work.
func (bo *Options) doResumeLogBackup(ctx context.Context, backup *v1alpha1.Backup) error {
	specificArgs := []string{
		"log",
		"resume",
		fmt.Sprintf("--task-name=%s", backup.Name),
	}
	fullArgs, err := bo.backupCommandTemplate(backup, specificArgs, false)
	if err != nil {
		return err
	}
	return bo.brCommandRun(ctx, fullArgs)
}

// doStoplogBackup generates br args about log backup stop and runs br binary to do the real backup work.
func (bo *Options) doStopLogBackup(ctx context.Context, backup *v1alpha1.Backup) error {
	specificArgs := []string{
		"log",
		"stop",
		fmt.Sprintf("--task-name=%s", backup.Name),
	}
	fullArgs, err := bo.backupCommandTemplate(backup, specificArgs, false)
	if err != nil {
		return err
	}
	return bo.brCommandRun(ctx, fullArgs)
}

// doPauselogBackup generates br args about log backup pause and runs br binary to do the real backup work.
func (bo *Options) doPauseLogBackup(ctx context.Context, backup *v1alpha1.Backup) error {
	specificArgs := []string{
		"log",
		"pause",
		fmt.Sprintf("--task-name=%s", backup.Name),
	}
	fullArgs, err := bo.backupCommandTemplate(backup, specificArgs, false)
	if err != nil {
		return err
	}
	return bo.brCommandRun(ctx, fullArgs)
}

// doTruncateLogBackup generates br args about log backup truncate and runs br binary to do the real backup work.
func (bo *Options) doTruncateLogBackup(ctx context.Context, backup *v1alpha1.Backup) error {
	specificArgs := []string{
		"log",
		"truncate",
	}
	if bo.TruncateUntil != "" && bo.TruncateUntil != "0" {
		specificArgs = append(specificArgs, fmt.Sprintf("--until=%s", bo.TruncateUntil))
	} else {
		return fmt.Errorf("log backup truncate until %s is invalid", bo.TruncateUntil)
	}
	fullArgs, err := bo.backupCommandTemplate(backup, specificArgs, false)
	if err != nil {
		return err
	}
	return bo.brCommandRun(ctx, fullArgs)
}

// logBackupCommandTemplate is the template to generate br args.
func (bo *Options) backupCommandTemplate(backup *v1alpha1.Backup, specificArgs []string, skipBackupArgs bool) ([]string, error) {
	if len(specificArgs) == 0 {
		return nil, fmt.Errorf("backup command is invalid, Args: %v", specificArgs)
	}

	args := make([]string, 0)
	args = append(args, fmt.Sprintf("--pd=%s", bo.PDAddress))
	if bo.TLSCluster {
		args = append(args, fmt.Sprintf("--ca=%s", path.Join(corev1alpha1.DirPathClusterClientTLS, corev1.ServiceAccountRootCAKey)))
		args = append(args, fmt.Sprintf("--cert=%s", path.Join(corev1alpha1.DirPathClusterClientTLS, corev1.TLSCertKey)))
		args = append(args, fmt.Sprintf("--key=%s", path.Join(corev1alpha1.DirPathClusterClientTLS, corev1.TLSPrivateKeyKey)))
	}

	if skipBackupArgs {
		return append(specificArgs, args...), nil
	}

	// `options` in spec are put to the last because we want them to have higher priority than generated arguments
	dataArgs, err := constructOptions(backup)
	if err != nil {
		return nil, err
	}
	args = append(args, dataArgs...)

	fullArgs := append(specificArgs, args...)
	return fullArgs, nil
}

// brCommandRun run br binary to do backup work.
func (bo *Options) brCommandRun(ctx context.Context, fullArgs []string) error {
	return bo.brCommandRunWithLogCallback(ctx, fullArgs, nil)
}

// brCommandRun run br binary to do backup work with log callback.
func (bo *Options) brCommandRunWithLogCallback(ctx context.Context, fullArgs []string, logCallback func(line string)) error {
	if len(fullArgs) == 0 {
		return fmt.Errorf("command is invalid, fullArgs: %v", fullArgs)
	}
	bin := filepath.Join(v1alpha1.DirPathBRBin, "br")
	klog.Infof("Running br command: %s %v", bin, fullArgs)

	e2eTestSimulate(bo)

	cmd := exec.Command(bin, fullArgs...)
	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("cluster %s, create stdout pipe failed, err: %w", bo, err)
	}
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("cluster %s, create stderr pipe failed, err: %w", bo, err)
	}
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("cluster %s, execute br command failed, args: %s, err: %w", bo, fullArgs, err)
	}

	var errMsg string
	stdErrCh := make(chan []byte, 1)
	go backupUtil.ReadAllStdErrToChannel(stdErr, stdErrCh)

	reader := bufio.NewReader(stdOut)
	for {
		line, err := reader.ReadString('\n')
		if strings.Contains(line, "[ERROR]") {
			errMsg += line
		}
		if logCallback != nil {
			logCallback(line)
		}

		klog.Info(strings.Replace(line, "\n", "", -1))
		if err != nil {
			if err != io.EOF {
				klog.Errorf("read stdout error: %s", err.Error())
			}
			break
		}
	}
	tmpErr := <-stdErrCh
	if len(tmpErr) > 0 {
		klog.Info(string(tmpErr))
		errMsg += string(tmpErr)
	}
	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("cluster %s, wait pipe message failed, errMsg %s, err: %w", bo, errMsg, err)
	}

	klog.Infof("Run br commond %v for cluster %s successfully", fullArgs, bo)
	return nil
}

// TODO use https://github.com/pingcap/failpoint instead e2e test env
func e2eTestSimulate(bo *Options) {
	if backupUtil.IsE2EExtendBackupTime() {
		for i := 0; i < 30; i++ {
			klog.Infof("simulate br running for backup %s", bo)
			time.Sleep(time.Second)
		}
	}
	if backupUtil.IsE2ETestPanic() {
		panic("simulate backup pod unexpected termination for e2e test")
	}
}
