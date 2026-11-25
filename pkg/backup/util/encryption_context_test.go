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
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/stretchr/testify/require"
)

func TestS3EncryptionContextArgs(t *testing.T) {
	values := map[string]string{"tenant": "租户 a", "punctuation": "\"quoted\" & + = \\ $()\n"}
	for _, flag := range []string{"", "storage", "full-backup-storage"} {
		t.Run(flag, func(t *testing.T) {
			provider := v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
				Bucket: "backup-bucket", Prefix: "test", SSE: "aws:kms",
			}}
			without, err := GenStorageArgsForFlag(provider, flag)
			require.NoError(t, err)
			if flag == "full-backup-storage" {
				require.Equal(t, []string{"--full-backup-storage=s3://backup-bucket/test"}, without)
			} else {
				require.Equal(t, []string{"--storage=s3://backup-bucket/test", "--s3.sse=aws:kms"}, without)
			}
			for _, contextValues := range []map[string]string{values, {}} {
				provider.S3.SSEKMSEncryptionContext = &v1alpha1.S3EncryptionContext{Values: contextValues}
				args, err := GenStorageArgsForFlag(provider, flag)
				require.NoError(t, err)
				var raw string
				if flag == "full-backup-storage" {
					require.Len(t, args, 1)
					// BR's ParseRawURL preserves literal '+' instead of treating it
					// as a form-encoded space. Check that both spaces and '+' survive.
					rawURL := strings.TrimPrefix(args[0], "--full-backup-storage=")
					u, err := url.Parse(strings.ReplaceAll(rawURL, "+", "%2B"))
					require.NoError(t, err)
					require.Equal(t, "backup-bucket", u.Host)
					require.Equal(t, "/test", u.Path)
					raw = u.Query().Get("sse-kms-context")
					require.Equal(t, "aws:kms", u.Query().Get("sse"))
				} else {
					require.Equal(t, without, args[:len(without)])
					require.Len(t, args, len(without)+1)
					require.True(t, strings.HasPrefix(args[len(without)], "--s3.sse-kms-context="))
					raw = strings.TrimPrefix(args[len(without)], "--s3.sse-kms-context=")
				}
				var actual map[string]string
				require.NoError(t, json.Unmarshal([]byte(raw), &actual), "context must be JSON, not Base64 or shell-quoted JSON")
				require.Equal(t, contextValues, actual)
			}
			provider.S3.SSEKMSEncryptionContext = &v1alpha1.S3EncryptionContext{}
			_, err = GenStorageArgsForFlag(provider, flag)
			require.EqualError(t, err, "s3.sseKmsEncryptionContext.values must be an object of string key-value pairs")
		})
	}
}

func TestS3RoleArgsForMainStorage(t *testing.T) {
	const roleARN = "arn:aws:iam::123456789012:role/backup"
	const externalID = "customer+backup"
	for _, flag := range []string{"", "storage"} {
		for _, tc := range []struct {
			name, role, externalID string
			wantOptions            []string
		}{
			{name: "omitted"},
			{name: "role", role: roleARN, wantOptions: []string{"--s3.role-arn=" + roleARN}},
			{name: "role and external ID", role: roleARN, externalID: externalID, wantOptions: []string{"--s3.role-arn=" + roleARN, "--s3.external-id=" + externalID}},
			{name: "external ID only", externalID: externalID, wantOptions: []string{"--s3.external-id=" + externalID}},
		} {
			t.Run(flag+"/"+tc.name, func(t *testing.T) {
				provider := v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
					Bucket: "backup-bucket", Prefix: "test", RoleARN: tc.role, ExternalID: tc.externalID,
				}}
				args, err := GenStorageArgsForFlag(provider, flag)
				require.NoError(t, err)
				require.Equal(t, append([]string{"--storage=s3://backup-bucket/test"}, tc.wantOptions...), args)
			})
		}
	}
}

func TestS3RoleArgsForSecondaryStorage(t *testing.T) {
	const roleARN = "arn:aws:iam::123456789012:role/full"
	const externalID = "customer + backup&scope=full"
	contextValues := map[string]string{"purpose": "full + PITR"}
	contextJSON, err := json.Marshal(contextValues)
	require.NoError(t, err)
	for _, tc := range []struct {
		name, prefix, wantPath, externalID string
		context                            *v1alpha1.S3EncryptionContext
		wantQuery                          url.Values
	}{
		{name: "role", prefix: "full", wantPath: "/full", wantQuery: url.Values{"role-arn": {roleARN}}},
		{
			name: "role and external ID", prefix: "full", wantPath: "/full", externalID: externalID,
			wantQuery: url.Values{"role-arn": {roleARN}, "external-id": {externalID}},
		},
		{
			name: "role, external ID and encryption context", prefix: "full", wantPath: "/full", externalID: externalID,
			context: &v1alpha1.S3EncryptionContext{Values: contextValues},
			wantQuery: url.Values{
				"sse-kms-context": {string(contextJSON)}, "sse": {"aws:kms"}, "role-arn": {roleARN}, "external-id": {externalID},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
				Bucket: "backup-bucket", Prefix: tc.prefix, SSE: "aws:kms",
				RoleARN: roleARN, ExternalID: tc.externalID, SSEKMSEncryptionContext: tc.context,
			}}
			args, err := GenStorageArgsForFlag(provider, "full-backup-storage")
			require.NoError(t, err)
			require.Len(t, args, 1)
			rawURL, ok := strings.CutPrefix(args[0], "--full-backup-storage=")
			require.True(t, ok)
			require.Equal(t, 1, strings.Count(rawURL, "?"))
			// BR escapes literal '+' before parsing the storage URL.
			parsed, err := url.Parse(strings.ReplaceAll(rawURL, "+", "%2B"))
			require.NoError(t, err)
			require.Equal(t, "s3", parsed.Scheme)
			require.Equal(t, "backup-bucket", parsed.Host)
			require.Equal(t, tc.wantPath, parsed.Path)
			require.Empty(t, parsed.Fragment)
			query, err := url.ParseQuery(parsed.RawQuery)
			require.NoError(t, err)
			require.Equal(t, tc.wantQuery, query)
		})
	}
}
