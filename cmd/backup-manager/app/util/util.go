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

package util

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/gogo/protobuf/proto"
	kvbackup "github.com/pingcap/kvproto/pkg/backup"
	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/util"
	"github.com/spf13/pflag"
	"k8s.io/klog"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var (
	cmdHelpMsg        string
	supportedVersions = map[string]struct{}{
		"3.1": {},
		"4.0": {},
	}
	// DefaultVersion is the default tikv and br version
	DefaultVersion = "4.0"
	defaultOptions = []string{
		// "--tidb-force-priority=LOW_PRIORITY",
		"--threads=16",
		"--rows=10000",
	}
	defaultTableFilterOptions = []string{
		"--filter", "*.*",
		"--filter", constants.DefaultTableFilter,
	}
)

func validCmdFlagFunc(flag *pflag.Flag) {
	if len(flag.Value.String()) > 0 {
		return
	}

	cmdutil.CheckErr(fmt.Errorf(cmdHelpMsg, flag.Name))
}

// ValidCmdFlags verify that all flags are set
func ValidCmdFlags(cmdPath string, flagSet *pflag.FlagSet) {
	cmdHelpMsg = "error: some flags [--%s] are missing.\nSee '" + cmdPath + " -h for' help."
	flagSet.VisitAll(validCmdFlagFunc)
}

// EnsureDirectoryExist create directory if does not exist
func EnsureDirectoryExist(dirName string) error {
	src, err := os.Stat(dirName)

	if os.IsNotExist(err) {
		errDir := os.MkdirAll(dirName, os.ModePerm)
		if errDir != nil {
			return fmt.Errorf("create dir %s failed. err: %v", dirName, err)
		}
		return nil
	}

	if src.Mode().IsRegular() {
		return fmt.Errorf("%s already exist as a file", dirName)
	}

	return nil
}

func GetRemotePath(backup *v1alpha1.Backup) (string, error) {
	var path, bucket, prefix string
	st := util.GetStorageType(backup.Spec.StorageProvider)
	switch st {
	case v1alpha1.BackupStorageTypeS3:
		prefix = backup.Spec.StorageProvider.S3.Prefix
		bucket = backup.Spec.StorageProvider.S3.Bucket
		prefix = strings.Trim(prefix, "/")
		prefix += "/"
		if prefix == "/" {
			path = fmt.Sprintf("s3://%s%s", bucket, prefix)
		} else {
			path = fmt.Sprintf("s3://%s/%s", bucket, prefix)
		}
		return path, nil
	case v1alpha1.BackupStorageTypeGcs:
		prefix = backup.Spec.StorageProvider.Gcs.Prefix
		bucket = backup.Spec.StorageProvider.Gcs.Bucket
		prefix = strings.Trim(prefix, "/")
		prefix += "/"
		if prefix == "/" {
			path = fmt.Sprintf("gcs://%s%s", bucket, prefix)
		} else {
			path = fmt.Sprintf("gcs://%s/%s", bucket, prefix)
		}
		return path, nil
	default:
		return "", fmt.Errorf("storage %s not support yet", st)
	}
}

// OpenDB opens db
func OpenDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open datasource failed, err: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("cannot connect to mysql, err: %v", err)
	}
	return db, nil
}

// IsFileExist return true if file exist and is a regular file, other cases return false
func IsFileExist(file string) bool {
	fi, err := os.Stat(file)
	if err != nil || !fi.Mode().IsRegular() {
		return false
	}
	return true
}

// IsDirExist return true if path exist and is a dir, other cases return false
func IsDirExist(path string) bool {
	fi, err := os.Stat(path)
	if err != nil || !fi.IsDir() {
		return false
	}
	return true
}

// NormalizeBucketURI normal bucket URL for rclone, e.g. s3://bucket -> s3:bucket
func NormalizeBucketURI(bucket string) string {
	return strings.Replace(bucket, "://", ":", 1)
}

// GetOptionValueFromEnv get option's value from environment variable. If unset, return empty string.
func GetOptionValueFromEnv(option, envPrefix string) string {
	envVar := envPrefix + "_" + strings.Replace(strings.ToUpper(option), "-", "_", -1)
	return os.Getenv(envVar)
}

// ConstructBRGlobalOptionsForBackup constructs BR global options for backup and also return the remote path.
func ConstructBRGlobalOptionsForBackup(backup *v1alpha1.Backup) ([]string, error) {
	var args []string
	config := backup.Spec
	if config.BR == nil {
		return nil, fmt.Errorf("no config for br in backup %s/%s", backup.Namespace, backup.Name)
	}
	args = append(args, constructBRGlobalOptions(config.BR)...)
	storageArgs, err := getRemoteStorage(backup.Spec.StorageProvider)
	if err != nil {
		return nil, err
	}
	args = append(args, storageArgs...)

	if config.TableFilter != nil && len(config.TableFilter) > 0 {
		for _, tableFilter := range config.TableFilter {
			args = append(args, "--filter", tableFilter)
		}
		return args, nil
	}

	switch backup.Spec.Type {
	case v1alpha1.BackupTypeTable:
		if config.BR.Table != "" {
			args = append(args, fmt.Sprintf("--table=%s", config.BR.Table))
		}
		if config.BR.DB != "" {
			args = append(args, fmt.Sprintf("--db=%s", config.BR.DB))
		}
	case v1alpha1.BackupTypeDB:
		if config.BR.DB != "" {
			args = append(args, fmt.Sprintf("--db=%s", config.BR.DB))
		}
	}

	return args, nil
}

// ConstructDumplingOptionsForBackup constructs dumpling options for backup
func ConstructDumplingOptionsForBackup(backup *v1alpha1.Backup) []string {
	var args []string
	config := backup.Spec

	if config.TableFilter != nil && len(config.TableFilter) > 0 {
		for _, tableFilter := range config.TableFilter {
			args = append(args, "--filter", tableFilter)
		}
	} else if config.Dumpling != nil && config.Dumpling.TableFilter != nil && len(config.Dumpling.TableFilter) > 0 {
		for _, tableFilter := range config.Dumpling.TableFilter {
			args = append(args, "--filter", tableFilter)
		}
	} else {
		args = append(args, defaultTableFilterOptions...)
	}

	if config.Dumpling == nil {
		args = append(args, defaultOptions...)
		return args
	}

	if len(config.Dumpling.Options) != 0 {
		args = append(args, config.Dumpling.Options...)
	} else {
		args = append(args, defaultOptions...)
	}

	return args
}

// ConstructBRGlobalOptionsForRestore constructs BR global options for restore.
func ConstructBRGlobalOptionsForRestore(restore *v1alpha1.Restore) ([]string, error) {
	var args []string
	config := restore.Spec
	if config.BR == nil {
		return nil, fmt.Errorf("no config for br in restore %s/%s", restore.Namespace, restore.Name)
	}
	args = append(args, constructBRGlobalOptions(config.BR)...)
	storageArgs, err := getRemoteStorage(restore.Spec.StorageProvider)
	if err != nil {
		return nil, err
	}
	args = append(args, storageArgs...)

	if config.TableFilter != nil && len(config.TableFilter) > 0 {
		for _, tableFilter := range config.TableFilter {
			args = append(args, "--filter", tableFilter)
		}
		return args, nil
	}

	switch restore.Spec.Type {
	case v1alpha1.BackupTypeTable:
		if config.BR.Table != "" {
			args = append(args, fmt.Sprintf("--table=%s", config.BR.Table))
		}
		if config.BR.DB != "" {
			args = append(args, fmt.Sprintf("--db=%s", config.BR.DB))
		}
	case v1alpha1.BackupTypeDB:
		if config.BR.DB != "" {
			args = append(args, fmt.Sprintf("--db=%s", config.BR.DB))
		}
	}

	return args, nil
}

// constructBRGlobalOptions constructs BR basic global options.
func constructBRGlobalOptions(config *v1alpha1.BRConfig) []string {
	var args []string
	if config.LogLevel != "" {
		args = append(args, fmt.Sprintf("--log-level=%s", config.LogLevel))
	}
	if config.StatusAddr != "" {
		args = append(args, fmt.Sprintf("--status-addr=%s", config.StatusAddr))
	}
	if config.SendCredToTikv != nil {
		args = append(args, fmt.Sprintf("--send-credentials-to-tikv=%t", *config.SendCredToTikv))
	}
	return args
}

// Suffix parses the major and minor version from the string and return the suffix
func Suffix(version string) string {
	numS := strings.Split(DefaultVersion, ".")
	defaultSuffix := numS[0] + numS[1]

	v, err := semver.NewVersion(version)
	if err != nil {
		klog.Errorf("Parse version %s failure, error: %v", version, err)
		return defaultSuffix
	}
	parsed := fmt.Sprintf("%d.%d", v.Major(), v.Minor())
	if _, ok := supportedVersions[parsed]; ok {
		return fmt.Sprintf("%d%d", v.Major(), v.Minor())
	}
	return defaultSuffix
}

// GetOptions gets the rclone options
func GetOptions(provider v1alpha1.StorageProvider) []string {
	st := util.GetStorageType(provider)
	switch st {
	case v1alpha1.BackupStorageTypeS3:
		return provider.S3.Options
	default:
		return nil
	}
}

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
	if exist := IsFileExist(metaFile); !exist {
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

// GetBRArchiveSize returns the total size of the backup archive.
func GetBRArchiveSize(meta *kvbackup.BackupMeta) uint64 {
	total := uint64(meta.Size())
	for _, file := range meta.Files {
		total += file.Size_
	}
	return total
}

// GetBRMetaData get backup metadata from cloud storage
func GetBRMetaData(provider v1alpha1.StorageProvider) (*kvbackup.BackupMeta, error) {
	s, err := NewRemoteStorage(provider)
	if err != nil {
		return nil, err
	}
	defer s.Close()
	ctx := context.Background()
	exist, err := s.Exists(ctx, constants.MetaFile)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, fmt.Errorf("%s not exist", constants.MetaFile)

	}
	metaData, err := s.ReadAll(ctx, constants.MetaFile)
	if err != nil {
		return nil, err
	}
	backupMeta := &kvbackup.BackupMeta{}
	err = proto.Unmarshal(metaData, backupMeta)
	if err != nil {
		return nil, err
	}
	return backupMeta, nil
}

// GetCommitTsFromBRMetaData get backup position from `EndVersion` in BR backup meta
func GetCommitTsFromBRMetaData(provider v1alpha1.StorageProvider) (uint64, error) {
	backupMeta, err := GetBRMetaData(provider)
	if err != nil {
		return 0, err
	}
	return backupMeta.EndVersion, nil
}

// ConstructArgs constructs the rclone args
func ConstructArgs(conf string, opts []string, command, source, dest string) []string {
	var args []string
	if conf != "" {
		args = append(args, conf)
	}
	if len(opts) > 0 {
		args = append(args, opts...)
	}
	if command != "" {
		args = append(args, command)
	}
	if source != "" {
		args = append(args, source)
	}
	if dest != "" {
		args = append(args, dest)
	}
	return args
}
