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

package restore

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gogo/protobuf/proto"
	kvbackup "github.com/pingcap/kvproto/pkg/brpb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/tests/pkg/backup/s3auth"
	"github.com/stretchr/testify/require"
)

func TestRestoreManagerUsesS3RoleForMetadata(t *testing.T) {
	for _, mode := range []v1alpha1.RestoreMode{v1alpha1.RestoreModeSnapshot, v1alpha1.RestoreModePiTR} {
		t.Run(string(mode), func(t *testing.T) {
			f := s3auth.Install(t)
			data, err := proto.Marshal(&kvbackup.BackupMeta{EndVersion: 202})
			require.NoError(t, err)
			f.Objects["primary/backupmeta"] = data
			fullData, err := proto.Marshal(&kvbackup.BackupMeta{EndVersion: 999})
			require.NoError(t, err)
			f.Objects["full/backupmeta"] = fullData
			dir := t.TempDir()
			previous := brBinPath
			brBinPath = dir
			t.Cleanup(func() { brBinPath = previous })
			require.NoError(t, os.WriteFile(filepath.Join(dir, "br"), []byte("#!/bin/sh\nexit 0\n"), 0755))
			restore := &v1alpha1.Restore{Spec: v1alpha1.RestoreSpec{
				BR: &v1alpha1.BRConfig{Cluster: "target", Options: []string{"--s3.role-arn=" + s3auth.RoleARN}},
				StorageProvider: v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
					Bucket: "bucket", Prefix: "primary", Endpoint: s3auth.Endpoint, Region: "us-east-1", Provider: "aws",
				}},
			}}
			if mode == v1alpha1.RestoreModePiTR {
				restore.Spec.PitrFullBackupStorageProvider = v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{Bucket: "bucket", Prefix: "full"}}
			}
			updater := &recordingRestoreStatusUpdater{}
			manager := &Manager{StatusUpdater: updater}
			manager.Mode = string(mode)
			require.NoError(t, manager.performRestore(context.Background(), restore, nil))
			last := updater.updates[len(updater.updates)-1]
			require.NotNil(t, last)
			require.Equal(t, "202", *last.CommitTs)
			for _, request := range f.Requests {
				if request.Service == "s3" {
					require.Equal(t, s3auth.RoleKey, request.AccessKey)
				}
			}
		})
	}
}
