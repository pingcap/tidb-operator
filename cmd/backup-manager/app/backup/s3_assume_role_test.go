// Copyright 2026 PingCAP, Inc.
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
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	kvbackup "github.com/pingcap/kvproto/pkg/brpb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/tests/pkg/backup/s3auth"
	"github.com/stretchr/testify/require"
)

func TestBackupManagerUsesS3RoleForMetadata(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mode      v1alpha1.BackupMode
		useFields bool
	}{
		{name: "snapshot BR options", mode: v1alpha1.BackupModeSnapshot},
		{name: "log BR options", mode: v1alpha1.BackupModeLog},
		{name: "snapshot S3 fields", mode: v1alpha1.BackupModeSnapshot, useFields: true},
		{name: "log S3 fields", mode: v1alpha1.BackupModeLog, useFields: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := s3auth.Install(t)
			f.ExpectedExternalID = "external-id"
			data, err := proto.Marshal(&kvbackup.BackupMeta{StartVersion: 101, EndVersion: 202, BackupSize: 303})
			require.NoError(t, err)
			f.Objects["prefix/backupmeta"] = data
			dir := t.TempDir()
			previous := brBinPath
			brBinPath = dir
			t.Cleanup(func() { brBinPath = previous })
			argv := filepath.Join(dir, "argv")
			t.Setenv("BR_ARGV_FILE", argv)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "br"), []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$BR_ARGV_FILE\"\n"), 0755))
			backup := &v1alpha1.Backup{Spec: v1alpha1.BackupSpec{
				Mode: tc.mode,
				BR:   &v1alpha1.BRConfig{Cluster: "source"},
				StorageProvider: v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
					Bucket: "bucket", Prefix: "prefix", Endpoint: s3auth.Endpoint, Region: "us-east-1", Provider: "aws",
				}},
			}}
			roleArgs := []string{"--s3.role-arn=" + s3auth.RoleARN, "--s3.external-id=" + f.ExpectedExternalID}
			if tc.useFields {
				backup.Spec.S3.RoleARN = s3auth.RoleARN
				backup.Spec.S3.ExternalID = f.ExpectedExternalID
			} else {
				backup.Spec.BR.Options = roleArgs
			}
			updater := &recordingBackupStatusUpdater{}
			manager := &Manager{StatusUpdater: updater}
			manager.Mode = string(tc.mode)
			if tc.mode == v1alpha1.BackupModeLog {
				manager.SubCommand = string(v1alpha1.LogStartCommand)
				err = manager.performLogBackup(context.Background(), backup)
			} else {
				err = manager.performBackup(context.Background(), backup, nil)
			}
			require.NoError(t, err)
			args, err := os.ReadFile(argv)
			require.NoError(t, err)
			for _, arg := range roleArgs {
				require.Contains(t, strings.Split(string(args), "\n"), arg)
			}
			require.NotEmpty(t, updater.updates)
			last := updater.updates[len(updater.updates)-1]
			require.NotNil(t, last)
			if tc.mode == v1alpha1.BackupModeLog {
				require.Equal(t, "101", *last.CommitTs)
			} else {
				require.Equal(t, "202", *last.CommitTs)
				require.EqualValues(t, 303, *last.BackupSize)
			}
			require.Equal(t, []s3auth.Request{
				{Service: "sts", Method: http.MethodPost, Host: "sts.us-east-1.amazonaws.com", Action: "AssumeRole", RoleARN: s3auth.RoleARN,
					ExternalID: f.ExpectedExternalID, AccessKey: s3auth.SourceKey, Region: "us-east-1"},
				{Service: "s3", Method: http.MethodHead, Host: "s3.unit.test", AccessKey: s3auth.RoleKey, Region: "us-east-1"},
				{Service: "s3", Method: http.MethodGet, Host: "s3.unit.test", AccessKey: s3auth.RoleKey, Region: "us-east-1"},
			}, f.Requests)
		})
	}
}
