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

func TestMakeS3ConfigKeepsKMSKeyIDWithLegacyAlgorithm(t *testing.T) {
	const keyID = "alias/backup"
	config := makeS3Config(&v1alpha1.S3StorageProvider{
		SSE:                  "aws:kms",
		ServerSideEncryption: &v1alpha1.S3ServerSideEncryption{Algorithm: "AES256", KMSKeyID: keyID},
	}, false)
	require.Equal(t, "aws:kms", config.sse)
	require.Equal(t, keyID, config.sseKMSKeyID)
}

func TestS3ServerSideEncryptionArgsForMainStorage(t *testing.T) {
	const keyARN = "arn:aws:kms:us-east-1:123456789012:key/backup"
	const keyAlias = "alias/backup"
	encryptionContext := &v1alpha1.S3EncryptionContext{
		Values: map[string]string{"tenant": "租户 a", "punctuation": "\"quoted\" & + = \\ $()\n"},
	}
	contextJSON, err := json.Marshal(encryptionContext.Values)
	require.NoError(t, err)
	contextArg := "--s3.sse-kms-context=" + string(contextJSON)

	for _, tc := range []struct {
		name       string
		legacy     string
		encryption *v1alpha1.S3ServerSideEncryption
		wantFlags  []string
		wantError  string
	}{
		{name: "omitted"},
		{name: "empty encryption", encryption: &v1alpha1.S3ServerSideEncryption{}},
		{name: "legacy algorithm", legacy: "aws:kms", wantFlags: []string{"--s3.sse=aws:kms"}},
		{
			name:       "algorithm only",
			encryption: &v1alpha1.S3ServerSideEncryption{Algorithm: "AES256"},
			wantFlags:  []string{"--s3.sse=AES256"},
		},
		{
			name:       "key ARN only",
			encryption: &v1alpha1.S3ServerSideEncryption{KMSKeyID: keyARN},
			wantFlags:  []string{"--s3.sse-kms-key-id=" + keyARN},
		},
		{
			name:       "key alias only",
			encryption: &v1alpha1.S3ServerSideEncryption{KMSKeyID: keyAlias},
			wantFlags:  []string{"--s3.sse-kms-key-id=" + keyAlias},
		},
		{
			name:       "context only",
			encryption: &v1alpha1.S3ServerSideEncryption{KMSEncryptionContext: encryptionContext},
			wantFlags:  []string{contextArg},
		},
		{
			name: "explicit empty context",
			encryption: &v1alpha1.S3ServerSideEncryption{
				KMSEncryptionContext: &v1alpha1.S3EncryptionContext{Values: map[string]string{}},
			},
			wantFlags: []string{"--s3.sse-kms-context={}"},
		},
		{
			name: "all fields",
			encryption: &v1alpha1.S3ServerSideEncryption{
				Algorithm: "aws:kms:dsse", KMSKeyID: keyARN, KMSEncryptionContext: encryptionContext,
			},
			wantFlags: []string{"--s3.sse=aws:kms:dsse", "--s3.sse-kms-key-id=" + keyARN, contextArg},
		},
		{
			name:       "equal algorithms",
			legacy:     "aws:kms",
			encryption: &v1alpha1.S3ServerSideEncryption{Algorithm: "aws:kms"},
			wantFlags:  []string{"--s3.sse=aws:kms"},
		},
		{
			name:       "conflicting algorithms",
			legacy:     "AES256",
			encryption: &v1alpha1.S3ServerSideEncryption{Algorithm: "aws:kms", KMSEncryptionContext: encryptionContext},
			wantError:  `s3.sse ("AES256") conflicts with s3.serverSideEncryption.algorithm ("aws:kms")`,
		},
		{
			name:   "legacy algorithm with nested key and context",
			legacy: "aws:kms",
			encryption: &v1alpha1.S3ServerSideEncryption{
				KMSKeyID: keyAlias, KMSEncryptionContext: encryptionContext,
			},
			wantFlags: []string{"--s3.sse=aws:kms", "--s3.sse-kms-key-id=" + keyAlias, contextArg},
		},
		{
			name:       "missing context values",
			encryption: &v1alpha1.S3ServerSideEncryption{KMSEncryptionContext: &v1alpha1.S3EncryptionContext{}},
			wantError:  "s3.serverSideEncryption.kmsEncryptionContext.values must be an object of string key-value pairs",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
				Bucket: "backup-bucket", Prefix: "test", SSE: tc.legacy, ServerSideEncryption: tc.encryption,
			}}
			for _, flag := range []string{"", "storage"} {
				args, err := GenStorageArgsForFlag(provider, flag)
				if tc.wantError != "" {
					require.EqualError(t, err, tc.wantError, "storage flag %q", flag)
					require.Nil(t, args)
					continue
				}
				require.NoError(t, err, "storage flag %q", flag)
				require.Equal(t, append([]string{"--storage=s3://backup-bucket/test"}, tc.wantFlags...), args)
			}
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

func TestS3ServerSideEncryptionAndRoleArgsForSecondaryStorage(t *testing.T) {
	const roleARN = "arn:aws:iam::123456789012:role/full"
	const externalID = "customer + backup&scope=full"
	const keyARN = "arn:aws:kms:us-east-1:123456789012:key/backup"
	contextValues := map[string]string{"tenant": "租户 a", "purpose": "\"full\" + backup&scope=PITR\n"}
	contextJSON, err := json.Marshal(contextValues)
	require.NoError(t, err)

	for _, tc := range []struct {
		name, legacy, role, externalID string
		encryption                     *v1alpha1.S3ServerSideEncryption
		wantQuery                      url.Values
		wantQueryKeys                  []string
		wantError                      string
	}{
		{name: "legacy algorithm", legacy: "aws:kms", wantQuery: url.Values{}},
		{
			name:          "algorithm only",
			encryption:    &v1alpha1.S3ServerSideEncryption{Algorithm: "AES256"},
			wantQuery:     url.Values{"sse": {"AES256"}},
			wantQueryKeys: []string{"sse"},
		},
		{
			name:          "key only",
			encryption:    &v1alpha1.S3ServerSideEncryption{KMSKeyID: keyARN},
			wantQuery:     url.Values{"sse-kms-key-id": {keyARN}},
			wantQueryKeys: []string{"sse-kms-key-id"},
		},
		{
			name:          "algorithm and key",
			encryption:    &v1alpha1.S3ServerSideEncryption{Algorithm: "aws:kms", KMSKeyID: keyARN},
			wantQuery:     url.Values{"sse-kms-key-id": {keyARN}, "sse": {"aws:kms"}},
			wantQueryKeys: []string{"sse-kms-key-id", "sse"},
		},
		{
			name:          "role",
			legacy:        "aws:kms",
			role:          roleARN,
			wantQuery:     url.Values{"role-arn": {roleARN}},
			wantQueryKeys: []string{"role-arn"},
		},
		{
			name:          "role and external ID",
			legacy:        "aws:kms",
			role:          roleARN,
			externalID:    externalID,
			wantQuery:     url.Values{"role-arn": {roleARN}, "external-id": {externalID}},
			wantQueryKeys: []string{"role-arn", "external-id"},
		},
		{
			name: "nested encryption with role and external ID", legacy: "aws:kms", role: roleARN, externalID: externalID,
			encryption: &v1alpha1.S3ServerSideEncryption{
				Algorithm: "aws:kms", KMSKeyID: keyARN,
				KMSEncryptionContext: &v1alpha1.S3EncryptionContext{Values: contextValues},
			},
			wantQuery: url.Values{
				"sse-kms-context": {string(contextJSON)}, "sse-kms-key-id": {keyARN}, "sse": {"aws:kms"},
				"role-arn": {roleARN}, "external-id": {externalID},
			},
			wantQueryKeys: []string{"sse-kms-context", "sse-kms-key-id", "sse", "role-arn", "external-id"},
		},
		{
			name:          "legacy algorithm with nested context",
			legacy:        "aws:kms",
			encryption:    &v1alpha1.S3ServerSideEncryption{KMSEncryptionContext: &v1alpha1.S3EncryptionContext{Values: map[string]string{}}},
			wantQuery:     url.Values{"sse-kms-context": {"{}"}, "sse": {"aws:kms"}},
			wantQueryKeys: []string{"sse-kms-context", "sse"},
		},
		{
			name:       "conflicting algorithms",
			legacy:     "AES256",
			encryption: &v1alpha1.S3ServerSideEncryption{Algorithm: "aws:kms"},
			wantError:  `s3.sse ("AES256") conflicts with s3.serverSideEncryption.algorithm ("aws:kms")`,
		},
		{
			name:       "missing context values",
			encryption: &v1alpha1.S3ServerSideEncryption{KMSEncryptionContext: &v1alpha1.S3EncryptionContext{}},
			wantError:  "s3.serverSideEncryption.kmsEncryptionContext.values must be an object of string key-value pairs",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
				Bucket: "backup-bucket", Prefix: "full", SSE: tc.legacy,
				RoleARN: tc.role, ExternalID: tc.externalID, ServerSideEncryption: tc.encryption,
			}}
			args, err := GenStorageArgsForFlag(provider, "full-backup-storage")
			if tc.wantError != "" {
				require.EqualError(t, err, tc.wantError)
				require.Nil(t, args)
				return
			}
			require.NoError(t, err)
			require.Len(t, args, 1)
			rawURL, ok := strings.CutPrefix(args[0], "--full-backup-storage=")
			require.True(t, ok)
			if len(tc.wantQuery) == 0 {
				require.Equal(t, "s3://backup-bucket/full", rawURL)
			} else {
				require.Equal(t, 1, strings.Count(rawURL, "?"))
			}
			// BR escapes literal '+' before parsing the storage URL.
			parsed, err := url.Parse(strings.ReplaceAll(rawURL, "+", "%2B"))
			require.NoError(t, err)
			require.Equal(t, "s3", parsed.Scheme)
			require.Equal(t, "backup-bucket", parsed.Host)
			require.Equal(t, "/full", parsed.Path)
			require.Empty(t, parsed.Fragment)
			query, err := url.ParseQuery(parsed.RawQuery)
			require.NoError(t, err)
			require.Equal(t, tc.wantQuery, query)

			var queryKeys []string
			for _, parameter := range strings.Split(parsed.RawQuery, "&") {
				if key, _, found := strings.Cut(parameter, "="); found {
					queryKeys = append(queryKeys, key)
				}
			}
			require.Equal(t, tc.wantQueryKeys, queryKeys)
		})
	}
}
