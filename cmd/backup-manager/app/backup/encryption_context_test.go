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
	"context"
	"database/sql"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/tests/pkg/backup/s3auth"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBackupServerSideEncryption(t *testing.T) {
	const keyARN = "arn:aws:kms:us-east-1:123456789012:key/backup"
	for _, tc := range []struct {
		name, algorithm, keyID, context string
		options                         []string
	}{
		{name: "typed fields", algorithm: "aws:kms", keyID: keyARN, context: `{"tenant":"字段 \" a"}`},
		{
			name: "equals option overrides", algorithm: "aws:kms:dsse", keyID: "alias/override", context: `{"tenant":"override"}`,
			options: []string{"--s3.sse=aws:kms:dsse", "--s3.sse-kms-key-id=alias/override", `--s3.sse-kms-context={"tenant":"override"}`},
		},
		{
			name: "separate option overrides", algorithm: "aws:kms:dsse", keyID: "alias/override", context: `{"tenant":"override"}`,
			options: []string{"--s3.sse", "aws:kms:dsse", "--s3.sse-kms-key-id", "alias/override", "--s3.sse-kms-context", `{"tenant":"override"}`},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backup := &v1alpha1.Backup{Spec: v1alpha1.BackupSpec{
				BR: &v1alpha1.BRConfig{Cluster: "test", Options: tc.options},
				StorageProvider: v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
					Bucket: "bucket",
					ServerSideEncryption: &v1alpha1.S3ServerSideEncryption{
						Algorithm: "aws:kms", KMSKeyID: keyARN,
						KMSEncryptionContext: &v1alpha1.S3EncryptionContext{Values: map[string]string{"tenant": "字段 \" a"}},
					},
				}},
			}}
			args, err := (&Options{}).backupCommandTemplate(backup, []string{"backup", "full"}, false)
			require.NoError(t, err)
			require.Equal(t, []string{"backup", "full"}, args[:2])
			flags := pflag.NewFlagSet("br", pflag.ContinueOnError)
			flags.ParseErrorsWhitelist.UnknownFlags = true
			algorithm := flags.String("s3.sse", "", "")
			keyID := flags.String("s3.sse-kms-key-id", "", "")
			contextJSON := flags.String("s3.sse-kms-context", "", "")
			require.NoError(t, flags.Parse(args[2:]))
			require.Equal(t, tc.algorithm, *algorithm)
			require.Equal(t, tc.keyID, *keyID)
			require.JSONEq(t, tc.context, *contextJSON)
		})
	}
}

func TestLogBackupRejectsInvalidServerSideEncryption(t *testing.T) {
	previous := brBinPath
	brBinPath = t.TempDir()
	t.Cleanup(func() { brBinPath = previous })
	require.NoError(t, os.WriteFile(filepath.Join(brBinPath, "br"), []byte("#!/bin/sh\nexit 0\n"), 0755))

	for _, tc := range []struct {
		name       string
		encryption v1alpha1.S3ServerSideEncryption
		options    []string
		wantError  string
	}{
		{
			name:       "conflicting algorithms",
			encryption: v1alpha1.S3ServerSideEncryption{Algorithm: "AES256", KMSKeyID: "alias/backup"},
			wantError:  "s3.sse (\"aws:kms\") conflicts with s3.serverSideEncryption.algorithm (\"AES256\")",
		},
		{
			name: "missing context values with task name override",
			encryption: v1alpha1.S3ServerSideEncryption{
				Algorithm: "aws:kms", KMSKeyID: "alias/backup", KMSEncryptionContext: &v1alpha1.S3EncryptionContext{},
			},
			options:   []string{"--task-name", "existing-task"},
			wantError: "s3.serverSideEncryption.kmsEncryptionContext.values must be an object of string key-value pairs",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backup := &v1alpha1.Backup{
				ObjectMeta: metav1.ObjectMeta{Name: "log-backup", Namespace: "backup-ns"},
				Spec: v1alpha1.BackupSpec{
					BR: &v1alpha1.BRConfig{Cluster: "source", Options: tc.options},
					StorageProvider: v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
						Bucket: "bucket", Prefix: "prefix", SSE: "aws:kms", ServerSideEncryption: &tc.encryption,
					}},
				},
			}
			original := backup.DeepCopy()
			options := &Options{}
			require.EqualError(t, options.doStopLogBackup(context.Background(), backup), tc.wantError)
			require.EqualError(t, options.doStartLogBackup(context.Background(), backup), tc.wantError)
			require.Equal(t, original, backup)
		})
	}
}

func TestRestartedSnapshotBackupValidatesEncryptionBeforeCleanup(t *testing.T) {
	for _, tc := range []struct {
		name        string
		algorithm   string
		wantError   string
		wantCleanup bool
	}{
		{name: "conflicting algorithms", algorithm: "AES256", wantError: "s3.sse (\"aws:kms\") conflicts with s3.serverSideEncryption.algorithm (\"AES256\")"},
		{name: "matching algorithms", algorithm: "aws:kms", wantError: "exit status 42", wantCleanup: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := s3auth.Install(t)
			f.AllowSource = true
			const object = "prefix/data.sst"
			f.Objects[object] = []byte("backup data")

			previous := brBinPath
			brBinPath = t.TempDir()
			t.Cleanup(func() { brBinPath = previous })
			// Exit as soon as BR starts so the control case never reads backup metadata.
			require.NoError(t, os.WriteFile(filepath.Join(brBinPath, "br"), []byte("#!/bin/sh\nexit 42\n"), 0755))

			backup := &v1alpha1.Backup{
				Spec: v1alpha1.BackupSpec{
					Mode: v1alpha1.BackupModeSnapshot,
					BR:   &v1alpha1.BRConfig{Cluster: "source"},
					StorageProvider: v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
						Bucket: "bucket", Prefix: "prefix", Endpoint: s3auth.Endpoint, Provider: "aws", Region: "us-east-1",
						SSE: "aws:kms", ServerSideEncryption: &v1alpha1.S3ServerSideEncryption{Algorithm: tc.algorithm},
					}},
					CleanOption: &v1alpha1.CleanOption{PageSize: 100, RetryCount: 1},
				},
				Status: v1alpha1.BackupStatus{Conditions: []v1alpha1.BackupCondition{
					{Type: v1alpha1.BackupRestart, Status: corev1.ConditionTrue},
				}},
			}
			updater := &recordingBackupStatusUpdater{}
			manager := &Manager{StatusUpdater: updater}
			manager.Mode = string(v1alpha1.BackupModeSnapshot)
			manager.TiKVVersion = "v6.5.0"
			err := manager.performBackup(context.Background(), backup, nil)
			require.ErrorContains(t, err, tc.wantError)
			if tc.wantCleanup {
				require.NotContains(t, f.Objects, object)
				require.Contains(t, f.Requests, s3auth.Request{
					Service: "s3", Method: http.MethodPost, Host: "s3.unit.test", AccessKey: s3auth.SourceKey, Region: "us-east-1",
				})
			} else {
				require.Equal(t, []byte("backup data"), f.Objects[object])
				require.Empty(t, f.Requests)
				require.Len(t, updater.conditions, 2)
				require.Equal(t, v1alpha1.BackupPrepare, updater.conditions[0].Type)
				require.Equal(t, &v1alpha1.BackupCondition{
					Type: v1alpha1.BackupFailed, Status: corev1.ConditionTrue,
					Reason: "BackupDataToRemoteFailed", Message: tc.wantError,
				}, updater.conditions[1])
			}
		})
	}
}

func TestFirstSnapshotBackupValidatesEncryptionBeforeGC(t *testing.T) {
	const protocol = "backup-gc-encryption-test"
	var connections int
	mysql.RegisterDialContext(protocol, func(context.Context, string) (net.Conn, error) {
		connections++
		return nil, errors.New("unexpected SQL connection")
	})
	t.Cleanup(func() { mysql.DeregisterDialContext(protocol) })
	db, err := sql.Open("mysql", "user@"+protocol+"(unused)/")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	backup := &v1alpha1.Backup{Spec: v1alpha1.BackupSpec{
		Mode: v1alpha1.BackupModeSnapshot,
		BR:   &v1alpha1.BRConfig{Cluster: "source"},
		StorageProvider: v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
			Bucket: "bucket", SSE: "aws:kms", ServerSideEncryption: &v1alpha1.S3ServerSideEncryption{Algorithm: "AES256"},
		}},
	}, Status: v1alpha1.BackupStatus{Phase: v1alpha1.BackupScheduled}}
	updater := &recordingBackupStatusUpdater{}
	manager := &Manager{StatusUpdater: updater}
	manager.Mode = string(v1alpha1.BackupModeSnapshot)
	manager.TiKVVersion = "v8.5.7"

	const wantError = `s3.sse ("aws:kms") conflicts with s3.serverSideEncryption.algorithm ("AES256")`
	require.EqualError(t, manager.performBackup(t.Context(), backup, db), wantError)
	require.Zero(t, connections)
	require.Len(t, updater.conditions, 2)
	require.Equal(t, v1alpha1.BackupPrepare, updater.conditions[0].Type)
	require.Equal(t, &v1alpha1.BackupCondition{
		Type: v1alpha1.BackupFailed, Status: corev1.ConditionTrue,
		Reason: "BackupDataToRemoteFailed", Message: wantError,
	}, updater.conditions[1])
}
