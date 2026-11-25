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

package clean

import (
	"context"
	"net/http"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/tests/pkg/backup/s3auth"
	"github.com/stretchr/testify/require"
)

func TestCleanBRRemoteBackupDataWithS3Role(t *testing.T) {
	for _, roleDenied := range []bool{false, true} {
		t.Run("role access", func(t *testing.T) {
			f := s3auth.Install(t)
			f.AllowSource, f.DenyRole = true, roleDenied
			f.Objects["prefix/backupmeta"] = []byte("metadata")
			f.Objects["prefix/data.sst"] = []byte("data")
			f.Objects["neighbor/keep"] = []byte("unrelated")
			backup := &v1alpha1.Backup{Spec: v1alpha1.BackupSpec{
				BR: &v1alpha1.BRConfig{Options: []string{"--s3.role-arn=" + s3auth.RoleARN}},
				StorageProvider: v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
					Bucket: "bucket", Prefix: "prefix", Endpoint: s3auth.Endpoint, Provider: "aws",
				}},
				CleanOption: &v1alpha1.CleanOption{PageSize: 100, RetryCount: 1},
			}}
			err := (&Options{}).CleanBRRemoteBackupData(context.Background(), backup)
			if roleDenied {
				require.Error(t, err)
				require.Len(t, f.Objects, 3)
			} else {
				require.NoError(t, err)
				require.Equal(t, map[string][]byte{"neighbor/keep": []byte("unrelated")}, f.Objects)
			}
			for _, request := range f.Requests {
				if request.Service == "s3" {
					require.Equal(t, s3auth.RoleKey, request.AccessKey)
				}
			}
		})
	}
}

func TestCleanBRRemoteBackupDataWithS3RoleFields(t *testing.T) {
	f := s3auth.Install(t)
	f.ExpectedExternalID = "cleanup+backup"
	f.Objects["prefix/backupmeta"] = []byte("metadata")
	f.Objects["prefix/data.sst"] = []byte("data")
	f.Objects["neighbor/keep"] = []byte("unrelated")
	backup := &v1alpha1.Backup{Spec: v1alpha1.BackupSpec{
		StorageProvider: v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
			Bucket: "bucket", Prefix: "prefix", Endpoint: s3auth.Endpoint, Provider: "aws", Region: "us-east-1",
			RoleARN: s3auth.RoleARN, ExternalID: f.ExpectedExternalID,
		}},
		CleanOption: &v1alpha1.CleanOption{PageSize: 100, RetryCount: 1},
	}}
	require.NoError(t, (&Options{}).CleanBRRemoteBackupData(context.Background(), backup))
	require.Equal(t, map[string][]byte{"neighbor/keep": []byte("unrelated")}, f.Objects)
	require.Equal(t, []s3auth.Request{
		{Service: "sts", Method: http.MethodPost, Host: "sts.us-east-1.amazonaws.com", Action: "AssumeRole", RoleARN: s3auth.RoleARN,
			ExternalID: f.ExpectedExternalID, AccessKey: s3auth.SourceKey, Region: "us-east-1"},
		{Service: "s3", Method: http.MethodGet, Host: "s3.unit.test", AccessKey: s3auth.RoleKey, Region: "us-east-1"},
		{Service: "s3", Method: http.MethodPost, Host: "s3.unit.test", AccessKey: s3auth.RoleKey, Region: "us-east-1"},
		{Service: "s3", Method: http.MethodGet, Host: "s3.unit.test", AccessKey: s3auth.RoleKey, Region: "us-east-1"},
	}, f.Requests)
}
