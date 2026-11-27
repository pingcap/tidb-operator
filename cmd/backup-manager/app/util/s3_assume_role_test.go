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

package util

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	kvbackup "github.com/pingcap/kvproto/pkg/brpb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/tests/pkg/backup/s3auth"
	"github.com/stretchr/testify/require"
)

func TestGetBRMetaDataWithS3Role(t *testing.T) {
	f := s3auth.Install(t)
	f.ExpectedExternalID = "external-id"
	options := []string{"--s3.role-arn=" + s3auth.RoleARN, "--s3.external-id=" + f.ExpectedExternalID}
	for _, prefix := range []string{"full", "logs"} {
		expected := &kvbackup.BackupMeta{StartVersion: 101, EndVersion: 202, BackupSize: 303}
		if prefix == "logs" {
			expected.StartVersion, expected.EndVersion = 404, 505
		}
		encoded, err := proto.Marshal(expected)
		require.NoError(t, err)
		f.Objects[prefix+"/backupmeta"] = encoded
		provider := v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
			Bucket: "bucket", Prefix: prefix, Region: "us-east-1", Endpoint: s3auth.Endpoint, Provider: "aws",
		}}
		meta, err := GetBRMetaDataWithBROptions(context.Background(), provider, options)
		require.NoError(t, err)
		require.Equal(t, expected, meta)
		require.Equal(t, expected.BackupSize, GetBRArchiveSize(meta))
		commitTS, err := GetCommitTsFromBRMetaDataWithBROptions(context.Background(), provider, options)
		require.NoError(t, err)
		require.Equal(t, expected.EndVersion, commitTS)
	}
	for _, request := range f.Requests {
		if request.Service == "s3" {
			require.Equal(t, s3auth.RoleKey, request.AccessKey)
		}
	}
}

func TestGetBROptionsWithoutBR(t *testing.T) {
	require.Nil(t, GetBROptions(nil))
	options := []string{"--s3.role-arn=" + s3auth.RoleARN}
	require.Equal(t, options, GetBROptions(&v1alpha1.BRConfig{Options: options}))
}
