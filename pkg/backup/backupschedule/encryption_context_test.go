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

package backupschedule

import (
	"testing"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/stretchr/testify/require"
)

func TestScheduleServerSideEncryption(t *testing.T) {
	for _, tc := range []struct {
		name, algorithm, keyID string
		build                  func(*v1alpha1.BackupSchedule, time.Time) *v1alpha1.Backup
	}{
		{name: "snapshot", algorithm: "aws:kms", keyID: "alias/snapshot", build: buildBackup},
		{name: "log", algorithm: "aws:kms:dsse", keyID: "alias/log", build: buildLogBackup},
	} {
		t.Run(tc.name, func(t *testing.T) {
			encryption := &v1alpha1.S3ServerSideEncryption{
				Algorithm: tc.algorithm, KMSKeyID: tc.keyID,
				KMSEncryptionContext: &v1alpha1.S3EncryptionContext{Values: map[string]string{"tenant": tc.name}},
			}
			template := v1alpha1.BackupSpec{
				BR: &v1alpha1.BRConfig{Cluster: "test"},
				StorageProvider: v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
					Bucket: "bucket", ServerSideEncryption: encryption,
				}},
			}
			schedule := &v1alpha1.BackupSchedule{}
			if tc.name == "snapshot" {
				schedule.Spec.BackupTemplate = template
			} else {
				schedule.Spec.LogBackupTemplate = &template
			}
			backup := tc.build(schedule, time.Now())
			require.NotSame(t, template.S3, backup.Spec.S3)
			copied := backup.Spec.S3.ServerSideEncryption
			require.Equal(t, encryption, copied)
			require.NotSame(t, encryption, copied)
			require.NotSame(t, encryption.KMSEncryptionContext, copied.KMSEncryptionContext)
			copied.KMSEncryptionContext.Values["tenant"] = "changed"
			require.Equal(t, tc.name, encryption.KMSEncryptionContext.Values["tenant"], "scheduled backups must not share the template's map")
		})
	}
}

func TestScheduleS3Role(t *testing.T) {
	snapshot := &v1alpha1.S3StorageProvider{RoleARN: "arn:aws:iam::123456789012:role/snapshot", ExternalID: "snapshot"}
	log := &v1alpha1.S3StorageProvider{RoleARN: "arn:aws:iam::123456789012:role/log", ExternalID: "log"}
	schedule := &v1alpha1.BackupSchedule{Spec: v1alpha1.BackupScheduleSpec{
		BackupTemplate: v1alpha1.BackupSpec{
			BR: &v1alpha1.BRConfig{Cluster: "test"}, StorageProvider: v1alpha1.StorageProvider{S3: snapshot},
		},
		LogBackupTemplate: &v1alpha1.BackupSpec{
			BR: &v1alpha1.BRConfig{Cluster: "test"}, StorageProvider: v1alpha1.StorageProvider{S3: log},
		},
	}}
	now := time.Now()
	for _, tc := range []struct {
		backup   *v1alpha1.Backup
		template *v1alpha1.S3StorageProvider
	}{
		{backup: buildBackup(schedule, now), template: snapshot},
		{backup: buildLogBackup(schedule, now), template: log},
	} {
		require.Equal(t, tc.template.RoleARN, tc.backup.Spec.S3.RoleARN)
		require.Equal(t, tc.template.ExternalID, tc.backup.Spec.S3.ExternalID)
		require.NotSame(t, tc.template, tc.backup.Spec.S3)
	}
}
