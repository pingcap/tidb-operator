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

package restore

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"net"
	"net/url"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

func TestRestoreServerSideEncryption(t *testing.T) {
	for _, tc := range []struct {
		name, algorithm, keyID, context string
		options                         []string
	}{
		{name: "typed fields", algorithm: "aws:kms", keyID: "alias/log", context: `{"tenant":"typed"}`},
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
			restore := &v1alpha1.Restore{Spec: v1alpha1.RestoreSpec{
				BR: &v1alpha1.BRConfig{Cluster: "test", Options: tc.options},
				StorageProvider: v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
					Bucket: "bucket",
					ServerSideEncryption: &v1alpha1.S3ServerSideEncryption{
						Algorithm: "aws:kms", KMSKeyID: "alias/log",
						KMSEncryptionContext: &v1alpha1.S3EncryptionContext{Values: map[string]string{"tenant": "typed"}},
					},
				}},
			}}
			args, err := constructBROptions(restore)
			require.NoError(t, err)
			flags := pflag.NewFlagSet("br", pflag.ContinueOnError)
			flags.ParseErrorsWhitelist.UnknownFlags = true
			algorithm := flags.String("s3.sse", "", "")
			keyID := flags.String("s3.sse-kms-key-id", "", "")
			contextJSON := flags.String("s3.sse-kms-context", "", "")
			require.NoError(t, flags.Parse(args))
			require.Equal(t, tc.algorithm, *algorithm)
			require.Equal(t, tc.keyID, *keyID)
			require.JSONEq(t, tc.context, *contextJSON)
		})
	}
}

func TestPITRFullBackupServerSideEncryption(t *testing.T) {
	const fullKeyID = "arn:aws:kms:us-east-1:123456789012:key/full"
	restore := &v1alpha1.Restore{Spec: v1alpha1.RestoreSpec{
		BR: &v1alpha1.BRConfig{Cluster: "test"},
		StorageProvider: v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
			Bucket: "bucket", Prefix: "log",
			ServerSideEncryption: &v1alpha1.S3ServerSideEncryption{
				Algorithm: "aws:kms", KMSKeyID: "alias/log",
				KMSEncryptionContext: &v1alpha1.S3EncryptionContext{Values: map[string]string{"purpose": "log backup"}},
			},
		}},
		PitrFullBackupStorageProvider: v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
			Bucket: "bucket", Prefix: "full",
			ServerSideEncryption: &v1alpha1.S3ServerSideEncryption{
				Algorithm: "aws:kms:dsse", KMSKeyID: fullKeyID,
				KMSEncryptionContext: &v1alpha1.S3EncryptionContext{Values: map[string]string{"purpose": "full + backup"}},
			},
		}},
	}}
	options := &Options{}
	options.Mode = string(v1alpha1.RestoreModePiTR)
	options.PitrRestoredTs = "900"
	restoreType, pitrArgs, err := options.restoreTypeAndPITRArgs(restore)
	require.NoError(t, err)
	require.Equal(t, "point", restoreType)
	args, err := constructBROptions(restore, pitrArgs...)
	require.NoError(t, err)
	require.Contains(t, args, "--restored-ts=900")
	require.Contains(t, args, "--storage=s3://bucket/log")
	require.Contains(t, args, "--s3.sse=aws:kms")
	require.Contains(t, args, "--s3.sse-kms-key-id=alias/log")
	require.Contains(t, args, `--s3.sse-kms-context={"purpose":"log backup"}`)
	flags := pflag.NewFlagSet("br", pflag.ContinueOnError)
	flags.ParseErrorsWhitelist.UnknownFlags = true
	fullStorage := flags.String("full-backup-storage", "", "")
	require.NoError(t, flags.Parse(args))
	parsed, err := url.Parse(strings.ReplaceAll(*fullStorage, "+", "%2B"))
	require.NoError(t, err)
	require.Equal(t, "s3", parsed.Scheme)
	require.Equal(t, "bucket", parsed.Host)
	require.Equal(t, "/full", parsed.Path)
	require.Equal(t, url.Values{
		"sse": {"aws:kms:dsse"}, "sse-kms-key-id": {fullKeyID}, "sse-kms-context": {`{"purpose":"full + backup"}`},
	}, parsed.Query())

	full := restore.Spec.PitrFullBackupStorageProvider.S3
	full.ServerSideEncryption = &v1alpha1.S3ServerSideEncryption{Algorithm: "AES256"}
	_, pitrArgs, err = options.restoreTypeAndPITRArgs(restore)
	require.NoError(t, err)
	require.Equal(t, []string{
		"--restored-ts=900",
		"--full-backup-storage=s3://bucket/full?sse-kms-context=&sse-kms-key-id=&sse=AES256",
	}, pitrArgs)

	full.SSE = "aws:kms"
	for _, startTS := range []string{"", "800"} {
		restore.Spec.LogRestoreStartTs = startTS
		_, _, err = options.restoreTypeAndPITRArgs(restore)
		require.EqualError(t, err, `s3.sse ("aws:kms") conflicts with s3.serverSideEncryption.algorithm ("AES256")`)
	}
}

func TestPerformRestoreValidatesEncryptionBeforeGC(t *testing.T) {
	const protocol = "restore-gc-encryption-test"
	var connections int
	mysql.RegisterDialContext(protocol, func(context.Context, string) (net.Conn, error) {
		connections++
		return nil, errors.New("unexpected SQL connection")
	})
	t.Cleanup(func() { mysql.DeregisterDialContext(protocol) })
	db, err := sql.Open("mysql", "user@"+protocol+"(unused)/")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	for _, tc := range []struct {
		name      string
		mode      v1alpha1.RestoreMode
		wantError string
	}{
		{name: "main storage", mode: v1alpha1.RestoreModeSnapshot, wantError: `s3.sse ("aws:kms") conflicts with s3.serverSideEncryption.algorithm ("AES256")`},
		{name: "PITR full backup storage", mode: v1alpha1.RestoreModePiTR, wantError: "s3.sseKmsEncryptionContext conflicts with s3.serverSideEncryption.kmsEncryptionContext"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			restore := &v1alpha1.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: "test"}, Spec: v1alpha1.RestoreSpec{
				BR: &v1alpha1.BRConfig{Cluster: "test"}, Mode: tc.mode,
				StorageProvider: v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{Bucket: "main"}},
			}}
			if tc.mode == v1alpha1.RestoreModePiTR {
				restore.Spec.PitrFullBackupStorageProvider = v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
					Bucket: "full", SSEKMSEncryptionContext: &v1alpha1.S3EncryptionContext{Values: map[string]string{"tenant": "legacy"}},
					ServerSideEncryption: &v1alpha1.S3ServerSideEncryption{
						KMSEncryptionContext: &v1alpha1.S3EncryptionContext{Values: map[string]string{"tenant": "nested"}},
					},
				}}
			} else {
				restore.Spec.S3.SSE = "aws:kms"
				restore.Spec.S3.ServerSideEncryption = &v1alpha1.S3ServerSideEncryption{Algorithm: "AES256"}
			}

			updater := &captureRestoreConditionUpdater{}
			manager := &Manager{StatusUpdater: updater}
			manager.Mode = string(tc.mode)
			require.EqualError(t, manager.performRestore(t.Context(), restore, db), tc.wantError)
			require.Zero(t, connections)
			require.Len(t, updater.conditions, 2)
			require.Equal(t, v1alpha1.RestoreRunning, updater.conditions[0].Type)
			require.Equal(t, &v1alpha1.RestoreCondition{
				Type:    v1alpha1.RestoreFailed,
				Status:  corev1.ConditionTrue,
				Reason:  "RestoreDataFromRemoteFailed",
				Message: tc.wantError,
			}, updater.conditions[1])
		})
	}
}

type captureRestoreConditionUpdater struct {
	conditions []*v1alpha1.RestoreCondition
}

func (u *captureRestoreConditionUpdater) Update(_ *v1alpha1.Restore, condition *v1alpha1.RestoreCondition, _ *controller.RestoreUpdateStatus) error {
	if condition != nil {
		u.conditions = append(u.conditions, condition.DeepCopy())
	}
	return nil
}

func TestAbortRestoreRedactsStorageArgs(t *testing.T) {
	restore := &v1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: "test"},
		Spec: v1alpha1.RestoreSpec{
			BR: &v1alpha1.BRConfig{Cluster: "test"},
			StorageProvider: v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{
				Bucket: "bucket",
				ServerSideEncryption: &v1alpha1.S3ServerSideEncryption{
					Algorithm: "aws:kms", KMSKeyID: "alias/restore",
					KMSEncryptionContext: &v1alpha1.S3EncryptionContext{Values: map[string]string{"tenant": "context-probe"}},
				},
			}},
		},
	}
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	require.NoError(t, indexer.Add(restore))
	rm := &Manager{
		restoreLister: listers.NewRestoreLister(indexer),
		StatusUpdater: &controller.FakeRestoreConditionUpdater{RestoreIndexer: indexer},
	}
	rm.Namespace, rm.ResourceName = restore.Namespace, restore.Name
	args, err := rm.constructBRAbortArgs(restore)
	require.NoError(t, err)
	require.Contains(t, args, "--s3.sse=aws:kms")
	require.Contains(t, args, "--s3.sse-kms-key-id=alias/restore")
	require.Contains(t, args, `--s3.sse-kms-context={"tenant":"context-probe"}`)

	state := klog.CaptureState()
	t.Cleanup(state.Restore)
	var output bytes.Buffer
	klog.LogToStderr(false)
	klog.SetOutput(&output)
	// Exercise the real abort log path without starting BR or contacting a cluster.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, rm.performAbort(ctx), context.Canceled)
	klog.Flush()
	require.Contains(t, output.String(), "Running br abort restore command")
	require.NotContains(t, output.String(), "context-probe")
}
