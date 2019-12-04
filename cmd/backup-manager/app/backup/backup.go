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
	"context"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mholt/archiver"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	glog "k8s.io/klog"
)

// BackupOpts contains the input arguments to the backup command
type BackupOpts struct {
	Namespace   string
	BackupName  string
	Bucket      string
	Host        string
	Port        int32
	Password    string
	User        string
	StorageType string
}

func (bo *BackupOpts) String() string {
	return fmt.Sprintf("%s/%s", bo.Namespace, bo.BackupName)
}

func (bo *BackupOpts) getBackupFullPath() string {
	return filepath.Join(constants.BackupRootPath, bo.getBackupRelativePath())
}

func (bo *BackupOpts) getBackupRelativePath() string {
	backupName := fmt.Sprintf("backup-%s", time.Now().UTC().Format(time.RFC3339))
	return fmt.Sprintf("%s/%s", bo.Bucket, backupName)
}

func (bo *BackupOpts) getDestBucketURI(remotePath string) string {
	return fmt.Sprintf("%s://%s", bo.StorageType, remotePath)
}

func (bo *BackupOpts) getTikvGCLifeTime(db *sql.DB) (string, error) {
	var tikvGCTime string
	sql := fmt.Sprintf("select variable_value from %s where variable_name= ?", constants.TidbMetaTable)
	row := db.QueryRow(sql, constants.TikvGCVariable)
	err := row.Scan(&tikvGCTime)
	if err != nil {
		return tikvGCTime, fmt.Errorf("query cluster %s %s failed, sql: %s, err: %v", bo, constants.TikvGCVariable, sql, err)
	}
	return tikvGCTime, nil
}

func (bo *BackupOpts) setTikvGCLifeTime(db *sql.DB, gcTime string) error {
	sql := fmt.Sprintf("update %s set variable_value = ? where variable_name = ?", constants.TidbMetaTable)
	_, err := db.Exec(sql, gcTime, constants.TikvGCVariable)
	if err != nil {
		return fmt.Errorf("set cluster %s %s failed, sql: %s, err: %v", bo, constants.TikvGCVariable, sql, err)
	}
	return nil
}

func (bo *BackupOpts) dumpTidbClusterData() (string, error) {
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
		"^(?!(mysql|test|INFORMATION_SCHEMA|PERFORMANCE_SCHEMA))",
	}

	output, err := exec.Command("/mydumper", args...).CombinedOutput()
	if err != nil {
		return bfPath, fmt.Errorf("cluster %s, execute mydumper command %v failed, output: %s, err: %v", bo, args, string(output), err)
	}
	return bfPath, nil
}

func (bo *BackupOpts) backupDataToRemote(source, bucketURI string) error {
	destBucket := util.NormalizeBucketURI(bucketURI)
	tmpDestBucket := fmt.Sprintf("%s.tmp", destBucket)
	// TODO: We may need to use exec.CommandContext to control timeouts.
	output, err := exec.Command("rclone", constants.RcloneConfigArg, "copyto", source, tmpDestBucket).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cluster %s, execute rclone copyto command for upload backup data %s failed, output: %s, err: %v", bo, bucketURI, string(output), err)
	}

	glog.Infof("upload cluster %s backup data to %s successfully, now move it to permanent URL %s", bo, tmpDestBucket, destBucket)

	// the backup was a success
	// remove .tmp extension
	output, err = exec.Command("rclone", constants.RcloneConfigArg, "moveto", tmpDestBucket, destBucket).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cluster %s, execute rclone moveto command failed, output: %s, err: %v", bo, string(output), err)
	}
	return nil
}

func (bo *BackupOpts) brBackupData(backup *v1alpha1.Backup) (string, error) {
	args, path, err := constructBROptions(backup)
	if err != nil {
		return "", err
	}
	fullArgs := []string{
		"backup",
		fmt.Sprintf("%s", backup.Spec.Type),
	}
	fullArgs = append(fullArgs, args...)
	output, err := exec.Command("br", fullArgs...).CombinedOutput()
	if err != nil {
		return path, fmt.Errorf("cluster %s, execute br command %v failed, output: %s, err: %v", bo, args, string(output), err)
	}
	glog.Infof("backup data for cluster %s successfully, output: %s", bo, string(output))
	return path, nil
}

// cleanBRRemoteBackupData clean the backup data from remote
func (bo *BackupOpts) cleanBRRemoteBackupData(backup *v1alpha1.Backup) error {
	s, err := util.NewRemoteStorage(backup)
	if err != nil {
		return err
	}
	defer s.Close()
	ctx := context.Background()
	iter := s.List(nil)
	for {
		obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		glog.Infof("Prepare to delete %s for cluster %s", obj.Key, bo)
		err = s.Delete(context.Background(), obj.Key)
		if err != nil {
			return err
		}
		glog.Infof("Delete %s for cluster %s successfully", obj.Key, bo)
	}
	return nil
}

func (bo *BackupOpts) cleanRemoteBackupData(bucket string) error {
	destBucket := util.NormalizeBucketURI(bucket)
	output, err := exec.Command("rclone", constants.RcloneConfigArg, "deletefile", destBucket).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cluster %s, execute rclone deletefile command failed, output: %s, err: %v", bo, string(output), err)
	}

	glog.Infof("cluster %s backup %s was deleted successfully", bo, bucket)
	return nil
}

func (bo *BackupOpts) getDSN(db string) string {
	return fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=utf8", bo.User, bo.Password, bo.Host, bo.Port, db)
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

// constructBROptions constructs options for BR and also return the remote path
func constructBROptions(backup *v1alpha1.Backup) ([]string, string, error) {
	args, path, err := util.ConstructBRGlobalOptions(backup)
	if err != nil {
		return args, path, err
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
	return args, path, nil
}

// getBRBackupSize get the backup data size from remote
func getBRBackupSize(backupPath string, backup *v1alpha1.Backup) (int64, error) {
	var size int64
	s, err := util.NewRemoteStorage(backup)
	if err != nil {
		return size, err
	}
	defer s.Close()
	ctx := context.Background()
	iter := s.List(nil)
	for {
		obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return size, err
		}
		size += obj.Size
	}
	return size, nil
}
