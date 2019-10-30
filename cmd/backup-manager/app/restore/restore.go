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
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
)

// RestoreOpts contains the input arguments to the restore command
type RestoreOpts struct {
	Namespace   string
	TcName      string
	Password    string
	TidbSvc     string
	User        string
	RestoreName string
	BackupPath  string
	BackupName  string
}

func (ro *RestoreOpts) String() string {
	return fmt.Sprintf("%s/%s", ro.Namespace, ro.TcName)
}

func (ro *RestoreOpts) getRestoreDataPath() string {
	backupName := filepath.Base(ro.BackupPath)
	NsClusterName := fmt.Sprintf("%s_%s", ro.Namespace, ro.TcName)
	return filepath.Join(constants.BackupRootPath, NsClusterName, backupName)
}

func (ro *RestoreOpts) downloadBackupData(localPath string) error {
	if err := util.EnsureDirectoryExist(filepath.Dir(localPath)); err != nil {
		return err
	}

	remoteBucket := util.NormalizeBucketURI(ro.BackupPath)
	rcCopy := exec.Command("rclone", constants.RcloneConfigArg, "copyto", remoteBucket, localPath)
	if err := rcCopy.Start(); err != nil {
		return fmt.Errorf("cluster %s, start rclone copyto command for download backup data %s falied, err: %v", ro, ro.BackupPath, err)
	}
	if err := rcCopy.Wait(); err != nil {
		return fmt.Errorf("cluster %s, execute rclone copyto command for download backup data %s failed, err: %v", ro, ro.BackupPath, err)
	}

	return nil
}

func (ro *RestoreOpts) loadTidbClusterData(restorePath string) error {
	if exist := util.IsDirExist(restorePath); !exist {
		return fmt.Errorf("dir %s does not exist or is not a dir", restorePath)
	}
	args := []string{
		fmt.Sprintf("-d=%s", restorePath),
		fmt.Sprintf("-h=%s", ro.TidbSvc),
		"-P=4000",
		fmt.Sprintf("-u=%s", ro.User),
		fmt.Sprintf("-p=%s", ro.Password),
	}

	output, err := exec.Command("/loader", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cluster %s, execute loader command %v failed, output: %s, err: %v", ro, args, string(output), err)
	}
	return nil
}

func (ro *RestoreOpts) getDSN(db string) string {
	return fmt.Sprintf("%s:%s@(%s:4000)/%s?charset=utf8", ro.User, ro.Password, ro.TidbSvc, db)
}

// unarchiveBackupData unarchive backup data to dest dir
func unarchiveBackupData(backupFile, destDir string) (string, error) {
	var unarchiveBackupPath string
	if err := util.EnsureDirectoryExist(destDir); err != nil {
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
