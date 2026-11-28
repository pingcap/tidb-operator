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
	"net/url"
	"strings"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
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

	restore.Spec.PitrFullBackupStorageProvider.S3.SSE = "AES256"
	_, _, err = options.restoreTypeAndPITRArgs(restore)
	require.ErrorContains(t, err, `s3.sse ("AES256") conflicts with s3.serverSideEncryption.algorithm ("aws:kms:dsse")`)
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
