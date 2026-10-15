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
	"fmt"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func TestBackupEncryptionContext(t *testing.T) {
	for _, command := range [][]string{{"backup", "full"}, {"log", "start"}, {"log", "resume"}} {
		for _, options := range [][]string{nil, {`--s3.sse-kms-context={"tenant":"override"}`}, {"--s3.sse-kms-context", `{"tenant":"override"}`}} {
			t.Run(fmt.Sprintf("%s/%s/options-%d", command[0], command[1], len(options)), func(t *testing.T) {
				backup := &v1alpha1.Backup{Spec: v1alpha1.BackupSpec{
					BR: &v1alpha1.BRConfig{Cluster: "test", Options: options},
					StorageProvider: v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
						Bucket: "bucket", SSE: "aws:kms",
						SSEKMSEncryptionContext: &v1alpha1.S3EncryptionContext{Values: map[string]string{"tenant": "字段 \" a"}},
					}},
				}}
				args, err := (&Options{}).backupCommandTemplate(backup, append([]string(nil), command...), false)
				require.NoError(t, err)
				require.Equal(t, command, args[:2])
				flags := pflag.NewFlagSet("br", pflag.ContinueOnError)
				flags.ParseErrorsWhitelist.UnknownFlags = true
				context := flags.String("s3.sse-kms-context", "", "")
				require.NoError(t, flags.Parse(args[2:]))
				expected := `{"tenant":"字段 \" a"}`
				if len(options) > 0 {
					expected = `{"tenant":"override"}`
				}
				require.JSONEq(t, expected, *context)
				require.Contains(t, args, "--s3.sse=aws:kms")
			})
		}
	}
}
