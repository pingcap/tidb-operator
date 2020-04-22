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

package export

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mholt/archiver"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"k8s.io/klog"
)

// Options contains the input arguments to the backup command
type Options struct {
	util.GenericOptions
	Bucket      string
	StorageType string
	Prefix      string
}

func (bo *Options) getBackupFullPath() string {
	return filepath.Join(constants.BackupRootPath, bo.getBackupRelativePath())
}

func (bo *Options) getBackupRelativePath() string {
	backupName := fmt.Sprintf("backup-%s", time.Now().UTC().Format(time.RFC3339))
	return fmt.Sprintf("%s/%s", bo.Bucket, backupName)
}

func (bo *Options) getDestBucketURI(remotePath string) string {
	prefix := bo.Prefix
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}
	return fmt.Sprintf("%s://%s%s", bo.StorageType, prefix, remotePath)
}

func (bo *Options) dumpTidbClusterData() (string, error) {
	bfPath := bo.getBackupFullPath()
	err := util.EnsureDirectoryExist(bfPath)
	if err != nil {
		return "", err
	}
	args := []string{
		fmt.Sprintf("--outputdir=%s", bfPath),
		fmt.Sprintf("--host=%s", bo.Host),
		fmt.Sprintf("--port=%d", bo.Port),
		fmt.Sprintf("--user=%s", bo.User),
		fmt.Sprintf("--password=%s", bo.Password),
		"--long-query-guard=3600",
		"--tidb-force-priority=LOW_PRIORITY",
		"--verbose=3",
		"--regex",
		"^(?!(mysql|test|INFORMATION_SCHEMA|PERFORMANCE_SCHEMA|METRICS_SCHEMA|INSPECTION_SCHEMA))",
	}

	output, err := exec.Command("/mydumper", args...).CombinedOutput()
	if err != nil {
		return bfPath, fmt.Errorf("cluster %s, execute mydumper command %v failed, output: %s, err: %v", bo, args, string(output), err)
	}
	return bfPath, nil
}

func (bo *Options) backupDataToRemote(source, bucketURI string) error {
	destBucket := util.NormalizeBucketURI(bucketURI)
	tmpDestBucket := fmt.Sprintf("%s.tmp", destBucket)
	// TODO: We may need to use exec.CommandContext to control timeouts.
	output, err := exec.Command("rclone", constants.RcloneConfigArg, "copyto", source, tmpDestBucket).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cluster %s, execute rclone copyto command for upload backup data %s failed, output: %s, err: %v", bo, bucketURI, string(output), err)
	}

	klog.Infof("upload cluster %s backup data to %s successfully, now move it to permanent URL %s", bo, tmpDestBucket, destBucket)

	// the backup was a success
	// remove .tmp extension
	output, err = exec.Command("rclone", constants.RcloneConfigArg, "moveto", tmpDestBucket, destBucket).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cluster %s, execute rclone moveto command failed, output: %s, err: %v", bo, string(output), err)
	}
	return nil
}

/*
	getCommitTsFromMetadata get commitTs from mydumper's metadata file

	metadata file format is as follows:

		Started dump at: 2019-06-13 10:00:04
		SHOW MASTER STATUS:
			Log: tidb-binlog
			Pos: 409054741514944513
			GTID:

		Finished dump at: 2019-06-13 10:00:04
*/
func getCommitTsFromMetadata(backupPath string) (string, error) {
	var commitTs string

	metaFile := filepath.Join(backupPath, constants.MetaDataFile)
	if exist := util.IsFileExist(metaFile); !exist {
		return commitTs, fmt.Errorf("file %s does not exist or is not regular file", metaFile)
	}
	contents, err := ioutil.ReadFile(metaFile)
	if err != nil {
		return commitTs, fmt.Errorf("read metadata file %s failed, err: %v", metaFile, err)
	}

	for _, lineStr := range strings.Split(string(contents), "\n") {
		if !strings.Contains(lineStr, "Pos") {
			continue
		}
		lineStrSlice := strings.Split(lineStr, ":")
		if len(lineStrSlice) != 2 {
			return commitTs, fmt.Errorf("parse mydumper's metadata file %s failed, str: %s", metaFile, lineStr)
		}
		commitTs = strings.TrimSpace(lineStrSlice[1])
		break
	}
	return commitTs, nil
}

// getBackupSize get the backup data size
func getBackupSize(backupPath string) (int64, error) {
	var size int64
	if exist := util.IsFileExist(backupPath); !exist {
		return size, fmt.Errorf("file %s does not exist or is not regular file", backupPath)
	}
	out, err := exec.Command("rclone", constants.RcloneConfigArg, "ls", backupPath).CombinedOutput()
	if err != nil {
		return size, fmt.Errorf("failed to get backup %s size, err: %v", backupPath, err)
	}
	sizeStr := strings.Fields(string(out))[0]
	size, err = strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return size, fmt.Errorf("failed to parse size string %s, err: %v", sizeStr, err)
	}
	return size, nil
}

// archiveBackupData archive backup data by destFile's extension name
func archiveBackupData(backupDir, destFile string) error {
	if exist := util.IsDirExist(backupDir); !exist {
		return fmt.Errorf("dir %s does not exist or is not a dir", backupDir)
	}
	destDir := filepath.Dir(destFile)
	if err := util.EnsureDirectoryExist(destDir); err != nil {
		return err
	}
	err := archiver.Archive([]string{backupDir}, destFile)
	if err != nil {
		return fmt.Errorf("archive backup data %s to %s failed, err: %v", backupDir, destFile, err)
	}
	return nil
}
