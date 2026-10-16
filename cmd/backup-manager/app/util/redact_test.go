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
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactBRArgs(t *testing.T) {
	context := `{"tenant":"sensitive value"}`
	for _, test := range []struct {
		name           string
		args, expected []string
	}{
		{"equals", []string{"backup", "full", "--s3.sse-kms-context=" + context, "--s3.sse=aws:kms"}, []string{"backup", "full", "--s3.sse-kms-context=<redacted>", "--s3.sse=aws:kms"}},
		{"separate and duplicate", []string{"--s3.sse-kms-context", context, "--s3.sse-kms-context={}"}, []string{"--s3.sse-kms-context", "<redacted>", "--s3.sse-kms-context=<redacted>"}},
		{"storage URL", []string{"--storage=s3://bucket/path?region=us-east-1&sse-kms-context=" + url.QueryEscape(context)}, []string{"--storage=s3://bucket/path?region=us-east-1&sse-kms-context=%3Credacted%3E"}},
		{"secondary URL with alias", []string{"--full-backup-storage", "s3://bucket/path?SSE_KMS_CONTEXT=" + url.QueryEscape(context)}, []string{"--full-backup-storage", "s3://bucket/path?SSE_KMS_CONTEXT=%3Credacted%3E"}},
		{"duplicate URL values", []string{"--storage=s3://bucket?sse-kms-context=a&sse-kms-context=b"}, []string{"--storage=s3://bucket?sse-kms-context=%3Credacted%3E"}},
		{"malformed query", []string{"--storage=s3://bucket?sse-kms-context=%zz"}, []string{"--storage=<redacted>"}},
		{"malformed URL", []string{"--full-backup-storage=s3://[?sse-kms-context=value"}, []string{"--full-backup-storage=<redacted>"}},
		{"absent", []string{"--storage=s3://bucket/path?region=us-east-1", "--s3.sse=aws:kms"}, []string{"--storage=s3://bucket/path?region=us-east-1", "--s3.sse=aws:kms"}},
		{"short storage flag", []string{"-s", "s3://bucket?sse-kms-context=value"}, []string{"-s", "s3://bucket?sse-kms-context=%3Credacted%3E"}},
		{"attached short flag", []string{"-ss3://bucket?sse-kms-context=value"}, []string{"-ss3://bucket?sse-kms-context=%3Credacted%3E"}},
		{"checkpoint URL", []string{"--checkpoint-storage=s3://bucket?sse-kms-context=value"}, []string{"--checkpoint-storage=s3://bucket?sse-kms-context=%3Credacted%3E"}},
		{"missing storage value", []string{"--storage", "--s3.sse-kms-context", context}, []string{"--storage", "--s3.sse-kms-context", "<redacted>"}},
		{"credentials with context", []string{"--storage=s3://user:password@bucket/path?access-key=access&secret-access-key=secret&session-token=token&sse-kms-context=value"}, []string{"--storage=s3://bucket/path?access-key=%3Credacted%3E&secret-access-key=%3Credacted%3E&session-token=%3Credacted%3E&sse-kms-context=%3Credacted%3E"}},
		{"credentials without context", []string{"--full-backup-storage", "s3://bucket/path?SESSION_TOKEN=token&region=us-east-1"}, []string{"--full-backup-storage", "s3://bucket/path?SESSION_TOKEN=%3Credacted%3E&region=us-east-1"}},
		{"userinfo only", []string{"-s=s3://user:password@bucket/path"}, []string{"-s=s3://bucket/path"}},
		{"missing argument", []string{"--s3.sse-kms-context"}, []string{"--s3.sse-kms-context"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := append([]string(nil), test.args...)
			require.Equal(t, test.expected, RedactBRArgs(test.args))
			require.Equal(t, before, test.args, "redaction must not alter the process arguments")
		})
	}
}
