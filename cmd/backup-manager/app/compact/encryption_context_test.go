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

package compact

import (
	"context"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/stretchr/testify/require"
)

func TestBase64ifyCmdS3ServerSideEncryption(t *testing.T) {
	const keyID = "arn:aws:kms:us-east-1:123456789012:key/log"
	const roleARN = "arn:aws:iam::123456789012:role/compaction"
	s3 := &v1alpha1.S3StorageProvider{
		Bucket: "backup-bucket", Prefix: "log", RoleARN: roleARN, ExternalID: "compaction+log",
		ServerSideEncryption: &v1alpha1.S3ServerSideEncryption{
			Algorithm: "aws:kms", KMSKeyID: keyID,
			KMSEncryptionContext: &v1alpha1.S3EncryptionContext{Values: map[string]string{"tenant": "租户 + log"}},
		},
	}
	manager := &Manager{compact: &v1alpha1.CompactBackup{Spec: v1alpha1.CompactSpec{
		StorageProvider: v1alpha1.StorageProvider{S3: s3},
	}}}
	cmd, err := manager.base64ifyCmd(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{
		manager.brBin(), "operator", "base64ify", "--storage=s3://backup-bucket/log", "--s3.sse=aws:kms",
		"--s3.sse-kms-key-id=" + keyID, "--s3.role-arn=" + roleARN, "--s3.external-id=compaction+log",
		`--s3.sse-kms-context={"tenant":"租户 + log"}`,
	}, cmd.Args)

	s3.SSE = "AES256"
	cmd, err = manager.base64ifyCmd(context.Background())
	require.EqualError(t, err, `s3.sse ("AES256") conflicts with s3.serverSideEncryption.algorithm ("aws:kms")`)
	require.Nil(t, cmd)
}
