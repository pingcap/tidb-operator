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
	"io/ioutil"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	backupUtil "github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

// Options contains the input arguments to the backup command
type Options struct {
	backupUtil.GenericOptions
}

// backupData generates br args and runs br binary to do the real backup work
func (bo *Options) backupData(ctx context.Context, backup *v1alpha1.Backup) error {
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
	fullArgs, err := bo.backupCommandTemplate(backup, specificArgs)
	if err != nil {
		return err
	}
	return bo.brCommandRun(ctx, fullArgs)
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
	fullArgs, err := bo.backupCommandTemplate(backup, specificArgs)
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
	fullArgs, err := bo.backupCommandTemplate(backup, specificArgs)
	if err != nil {
		return err
	}
	return bo.brCommandRun(ctx, fullArgs)
}

// doTruncatelogBackup generates br args about log backup truncate and runs br binary to do the real backup work.
func (bo *Options) doTruncatelogBackup(ctx context.Context, backup *v1alpha1.Backup) error {
	specificArgs := []string{
		"log",
		"truncate",
	}
	if bo.TruncateUntil != "" && bo.TruncateUntil != "0" {
		specificArgs = append(specificArgs, fmt.Sprintf("--until=%s", bo.TruncateUntil))
	} else {
		return fmt.Errorf("log backup truncate until %s is invalid", bo.TruncateUntil)
	}
	fullArgs, err := bo.backupCommandTemplate(backup, specificArgs)
	if err != nil {
		return err
	}
	return bo.brCommandRun(ctx, fullArgs)
}

// logBackupCommandTemplate is the template to generate br args.
func (bo *Options) backupCommandTemplate(backup *v1alpha1.Backup, specificArgs []string) ([]string, error) {
	if len(specificArgs) == 0 {
		return nil, fmt.Errorf("backup command is invalid, Args: %v", specificArgs)
	}

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
	// `options` in spec are put to the last because we want them to have higher priority than generated arguments
	dataArgs, err := constructOptions(backup)
	if err != nil {
		return nil, err
	}
	args = append(args, dataArgs...)

	fullArgs := append(specificArgs, args...)
	return fullArgs, nil
}

// brCommandRun run br binary to do backup work
func (bo *Options) brCommandRun(ctx context.Context, fullArgs []string) error {
	if len(fullArgs) == 0 {
		return fmt.Errorf("command is invalid, fullArgs: %v", fullArgs)
	}
	klog.Infof("Running br command with args: %v", fullArgs)
	bin := filepath.Join(util.BRBinPath, "br")
	cmd := exec.CommandContext(ctx, bin, fullArgs...)

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("cluster %s, create stdout pipe failed, err: %v", bo, err)
	}
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("cluster %s, create stderr pipe failed, err: %v", bo, err)
	}
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("cluster %s, execute br command failed, args: %s, err: %v", bo, fullArgs, err)
	}
	var errMsg string
	reader := bufio.NewReader(stdOut)
	for {
		line, err := reader.ReadString('\n')
		if strings.Contains(line, "[ERROR]") {
			errMsg += line
		}

		klog.Info(strings.Replace(line, "\n", "", -1))
		if err != nil {
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
		return fmt.Errorf("cluster %s, wait pipe message failed, errMsg %s, err: %v", bo, errMsg, err)
	}

	klog.Infof("Run br commond %v for cluster %s successfully", fullArgs, bo)
	return nil
}
