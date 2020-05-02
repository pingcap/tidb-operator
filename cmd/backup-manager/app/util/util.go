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
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/spf13/pflag"
	"k8s.io/klog"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/util"
)

var (
	cmdHelpMsg        string
	supportedVersions = map[string]struct{}{
		"3.1": {},
		"4.0": {},
	}
	// DefaultVersion is the default tikv and br version
	DefaultVersion = "4.0"
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

// OpenDB opens db
func OpenDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open dsn %s failed, err: %v", dsn, err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("cannot connect to mysql: %s, err: %v", dsn, err)
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
func ConstructBRGlobalOptionsForBackup(backup *v1alpha1.Backup) ([]string, string, error) {
	var args []string
	config := backup.Spec.BR
	if config == nil {
		return nil, "", fmt.Errorf("no config for br in backup %s/%s", backup.Namespace, backup.Name)
	}
	args = append(args, constructBRGlobalOptions(config)...)
	storageArgs, remotePath, err := getRemoteStorage(backup.Spec.StorageProvider)
	if err != nil {
		return nil, "", err
	}
	args = append(args, storageArgs...)
	if (backup.Spec.Type == v1alpha1.BackupTypeDB || backup.Spec.Type == v1alpha1.BackupTypeTable) && config.DB != "" {
		args = append(args, fmt.Sprintf("--db=%s", config.DB))
	}
	if backup.Spec.Type == v1alpha1.BackupTypeTable && config.Table != "" {
		args = append(args, fmt.Sprintf("--table=%s", config.Table))
	}
	return args, remotePath, nil
}

// ConstructBRGlobalOptionsForRestore constructs BR global options for restore.
func ConstructBRGlobalOptionsForRestore(restore *v1alpha1.Restore) ([]string, error) {
	var args []string
	config := restore.Spec.BR
	if config == nil {
		return nil, fmt.Errorf("no config for br in restore %s/%s", restore.Namespace, restore.Name)
	}
	args = append(args, constructBRGlobalOptions(config)...)
	storageArgs, _, err := getRemoteStorage(restore.Spec.StorageProvider)
	if err != nil {
		return nil, err
	}
	args = append(args, storageArgs...)
	if (restore.Spec.Type == v1alpha1.BackupTypeDB || restore.Spec.Type == v1alpha1.BackupTypeTable) && config.DB != "" {
		args = append(args, fmt.Sprintf("--db=%s", config.DB))
	}
	if restore.Spec.Type == v1alpha1.BackupTypeTable && config.Table != "" {
		args = append(args, fmt.Sprintf("--table=%s", config.Table))
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
