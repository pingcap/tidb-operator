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

package _import

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	backupUtil "github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

// Options contains the input arguments to the restore command
type Options struct {
	backupUtil.GenericOptions
	BackupPath string
}

func (ro *Options) getRestoreDataPath() string {
	// The backupPath format must be like this storageType://backup/to/path, so the split array must have two elements
	backupSuffix := strings.Split(ro.BackupPath, "://")[1]
	return filepath.Join(constants.BackupRootPath, backupSuffix)
}

func (ro *Options) downloadBackupData(localPath string, opts []string) error {
	if err := backupUtil.EnsureDirectoryExist(filepath.Dir(localPath)); err != nil {
		return err
	}

	remoteBucket := backupUtil.NormalizeBucketURI(ro.BackupPath)
	args := backupUtil.ConstructArgs(constants.RcloneConfigArg, opts, "copyto", remoteBucket, localPath)
	rcCopy := exec.Command("rclone", args...)

	stdOut, err := rcCopy.StdoutPipe()
	if err != nil {
		return fmt.Errorf("cluster %s, create stdout pipe failed, err: %v", ro, err)
	}
	stdErr, err := rcCopy.StderrPipe()
	if err != nil {
		return fmt.Errorf("cluster %s, create stderr pipe failed, err: %v", ro, err)
	}

	if err := rcCopy.Start(); err != nil {
		return fmt.Errorf("cluster %s, start rclone copyto command for download backup data %s falied, err: %v", ro, ro.BackupPath, err)
	}

	var errMsg string
	tmpOut, _ := ioutil.ReadAll(stdOut)
	if len(tmpOut) > 0 {
		klog.Infof(string(tmpOut))
	}
	tmpErr, _ := ioutil.ReadAll(stdErr)
	if len(tmpErr) > 0 {
		klog.Infof(string(tmpErr))
		errMsg = string(tmpErr)
	}

	if err := rcCopy.Wait(); err != nil {
		return fmt.Errorf("cluster %s, execute rclone copyto command for download backup data %s failed, errMsg: %v, err: %v", ro, ro.BackupPath, errMsg, err)
	}

	return nil
}

func (ro *Options) loadTidbClusterData(restorePath string) error {
	if exist := backupUtil.IsDirExist(restorePath); !exist {
		return fmt.Errorf("dir %s does not exist or is not a dir", restorePath)
	}
	// args for restore
	args := []string{
		"--status-addr=0.0.0.0:8289",
		"--backend=tidb",
		"--server-mode=false",
		"--log-file=",
		fmt.Sprintf("--tidb-user=%s", ro.User),
		fmt.Sprintf("--tidb-password=%s", ro.Password),
		fmt.Sprintf("--tidb-host=%s", ro.Host),
		fmt.Sprintf("--d=%s", restorePath),
		fmt.Sprintf("--tidb-port=%d", ro.Port),
	}
	if ro.TLSClient {
		args = append(args, fmt.Sprintf("--ca=%s", path.Join(util.TiDBClientTLSPath, corev1.ServiceAccountRootCAKey)))
		args = append(args, fmt.Sprintf("--cert=%s", path.Join(util.TiDBClientTLSPath, corev1.TLSCertKey)))
		args = append(args, fmt.Sprintf("--key=%s", path.Join(util.TiDBClientTLSPath, corev1.TLSPrivateKeyKey)))
	}

	output, err := exec.Command("/tidb-lightning", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cluster %s, execute loader command %v failed, output: %s, err: %v", ro, args, string(output), err)
	}
	return nil
}

// unarchiveBackupData unarchive backup data to dest dir
func unarchiveBackupData(backupFile, destDir string) (string, error) {
	var unarchiveBackupPath string
	if err := backupUtil.EnsureDirectoryExist(destDir); err != nil {
		return unarchiveBackupPath, err
	}
	backupName := strings.TrimSuffix(filepath.Base(backupFile), constants.DefaultArchiveExtention)
	tarGz := archiver.NewTarGz()
	// overwrite if the file already exists
	tarGz.OverwriteExisting = true
	err := tarGz.Unarchive(backupFile, destDir)
	if err != nil {
		return unarchiveBackupPath, fmt.Errorf("unarchive backup data %s to %s failed, err: %v", backupFile, destDir, err)
	}
	unarchiveBackupPath = filepath.Join(destDir, backupName)
	return unarchiveBackupPath, nil
}
