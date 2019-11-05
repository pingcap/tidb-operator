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

	"github.com/spf13/pflag"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var (
	cmdHelpMsg string
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
