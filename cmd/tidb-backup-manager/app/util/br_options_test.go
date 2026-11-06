// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"testing"

	"github.com/stretchr/testify/require"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
)

func TestBRCommandTargetsCannotBeOverridden(t *testing.T) {
	for _, option := range []string{"--pd=other:2379", "--storage=s3://other/backup", "--s3.endpoint=https://other"} {
		config := &br.BRConfig{Cluster: "cluster", Options: []string{option}}
		backup := &br.Backup{Spec: br.BackupSpec{BR: config}}
		backup.Spec.S3 = &br.S3StorageProvider{Bucket: "bucket", Prefix: "backup"}
		_, err := ConstructBRGlobalOptionsForBackup(backup)
		require.ErrorContains(t, err, "spec.br.options")
		restore := &br.Restore{Spec: br.RestoreSpec{BR: config}}
		restore.Spec.S3 = backup.Spec.S3
		_, err = ConstructBRGlobalOptionsForRestore(restore)
		require.ErrorContains(t, err, "spec.br.options")
	}
}
