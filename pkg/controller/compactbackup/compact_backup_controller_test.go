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
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/stretchr/testify/require"
)

func TestValidateCompactBackupS3Encryption(t *testing.T) {
	compact := &v1alpha1.CompactBackup{Spec: v1alpha1.CompactSpec{
		StartTs: "100", EndTs: "200", Concurrency: 1,
		StorageProvider: v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
			Bucket: "bucket", SSE: "aws:kms",
			SSEKMSEncryptionContext: &v1alpha1.S3EncryptionContext{Values: map[string]string{"tenant": "legacy"}},
			ServerSideEncryption: &v1alpha1.S3ServerSideEncryption{
				Algorithm: "aws:kms", KMSEncryptionContext: &v1alpha1.S3EncryptionContext{Values: map[string]string{"tenant": "legacy"}},
			},
		}},
	}}
	c := &Controller{}
	require.NoError(t, c.validate(compact))

	encryption := compact.Spec.S3.ServerSideEncryption
	encryption.Algorithm = "AES256"
	require.EqualError(t, c.validate(compact), `s3.sse ("aws:kms") conflicts with s3.serverSideEncryption.algorithm ("AES256")`)

	encryption.Algorithm = "aws:kms"
	encryption.KMSEncryptionContext.Values["tenant"] = "nested"
	require.EqualError(t, c.validate(compact), "s3.sseKmsEncryptionContext conflicts with s3.serverSideEncryption.kmsEncryptionContext")
}
