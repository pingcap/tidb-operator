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
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"

	"github.com/mholt/archiver"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/util"
)

/*
	GetCommitTsFromMetadata get commitTs from mydumper's metadata file

	metadata file format is as follows:

		Started dump at: 2019-06-13 10:00:04
		SHOW MASTER STATUS:
		Log: tidb-binlog
		Pos: 409054741514944513
		GTID:

		Finished dump at: 2019-06-13 10:00:04
*/
func GetCommitTsFromMetadata(backupPath string) (string, error) {
	var commitTs string

	metaFile := filepath.Join(backupPath, constants.MetaDataFile)
	if exist := util.IsFileExist(metaFile); !exist {
		return commitTs, fmt.Errorf("file %s does not exist or is not regular file", metaFile)
	}
	contents, err := ioutil.ReadFile(metaFile)
	if err != nil {
		return commitTs, fmt.Errorf("read metadata file %s failed, err: %v", err)
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

// GetBackupSize get the backup data size
func GetBackupSize(backupPath string) (string, error) {
	var size string
	if exist := util.IsFileExist(backupPath); !exist {
		return size, fmt.Errorf("file %s does not exist or is not regular file", backupPath)
	}
	// Uses the same niceness level as cadvisor.fs does when running du
	// Uses -B 1 to always scale to a blocksize of 1 byte
	out, err := exec.Command("nice", "-n", "19", "du", "-s", "-B", "1", backupPath).CombinedOutput()
	if err != nil {
		return size, fmt.Errorf("failed to get backup %s size, err: %v", backupPath, err)
	}
	size = strings.Fields(string(out))[0]
	return size, nil
}

// ArchiveBackupData archive backup data by destFile's extension name
func ArchiveBackupData(backupDir, destFile string) error {
	if exist := util.IsDirExist(backupDir); !exist {
		return fmt.Errorf("dir %s does not exist or is not a dir", backupDir)
	}
	destDir := filepath.Dir(destFile)
	if err := util.EnsureDirectoryExist(destDir); err != nil {
		return err
	}
	err := archiver.Archive([]string{backupDir}, destFile)
	if err != nil {
		return fmt.Errorf("archive backup data %s failed, err: %s", backupDir, destFile)
	}
	return nil
}
