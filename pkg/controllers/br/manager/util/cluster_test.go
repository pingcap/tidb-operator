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
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	core "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
)

func TestBackupClusterConsent(t *testing.T) {
	for _, tc := range []struct {
		name, source, target, allowed string
		ok                            bool
	}{
		{"same namespace", "tenant", "tenant", "", true},
		{"absent consent", "tenant", "other", "", false},
		{"explicit consent", "tenant", "other", "tenant", true},
		{"consent list", "tenant", "other", "one, tenant, two", true},
		{"different source", "tenant", "other", "tenant-other", false},
		{"wildcard not supported", "tenant", "other", "*", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cluster := &core.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: tc.target, Annotations: map[string]string{AllowedBackupNamespaces: tc.allowed}}}
			cli := client.NewFakeClient(cluster)
			got, err := GetBackupCluster(context.Background(), cli, tc.source, &br.BRConfig{Cluster: "cluster", ClusterNamespace: tc.target})
			if tc.ok {
				require.NoError(t, err)
				require.Equal(t, tc.target, got.Namespace)
			} else {
				require.True(t, apierrors.IsForbidden(err))
				require.Nil(t, got)
			}
		})
	}
	cli := client.NewFakeClient()
	_, err := GetBackupCluster(context.Background(), cli, "tenant", nil)
	require.Error(t, err)
	_, err = GetBackupCluster(context.Background(), cli, "tenant", &br.BRConfig{Cluster: "missing"})
	require.True(t, apierrors.IsNotFound(err))
}

func TestBackupClusterRejectsTargetOverrides(t *testing.T) {
	cluster := &core.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "tenant"}}
	cli := client.NewFakeClient(cluster)
	for _, options := range [][]string{
		{"--pd=victim-pd.victim:2379"}, {"--pd", "victim-pd.victim:2379"},
		{"-u", "victim-pd.victim:2379"}, {"-cu=victim-pd.victim:2379"},
		{"--storage=s3://other/backup"}, {"--storage", "s3://other/backup"},
		{"-ss3://other/backup"}, {"--s3.endpoint=https://other"},
		{"--gcs.endpoint=https://other"}, {"--azblob.endpoint=https://other"},
		{"--azblob.account-name=other"}, {"--full-backup-storage=s3://other/full"},
	} {
		t.Run(strings.Join(options, " "), func(t *testing.T) {
			_, err := GetBackupCluster(t.Context(), cli, "tenant", &br.BRConfig{Cluster: "cluster", Options: options})
			require.ErrorContains(t, err, "spec.br.options")
		})
	}
}

func TestBROptionsPreserveTuningFlags(t *testing.T) {
	for _, options := range [][]string{
		nil, {"--concurrency=4", "--checksum=false"}, {"--ratelimit", "4"},
		{"--filter=--pd=literal", "--filter=--storage=literal"}, {"--some-number=-1"},
	} {
		require.NoError(t, ValidateBROptions(&br.BRConfig{Options: options}))
	}
}

func TestLogClusterBindingRequiresOriginalHistory(t *testing.T) {
	cluster := &core.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "tenant", UID: "original"}}
	cli := client.NewFakeClient(cluster)
	backup := &br.Backup{ObjectMeta: metav1.ObjectMeta{Namespace: "tenant"},
		Spec: br.BackupSpec{Mode: br.BackupModeLog, BR: &br.BRConfig{Cluster: "cluster"}},
	}
	_, err := GetBackupClusterForBackup(t.Context(), cli, backup)
	require.NoError(t, err)
	backup.Status.CommitTs = "1"
	_, err = GetBackupClusterForBackup(t.Context(), cli, backup)
	require.ErrorContains(t, err, "original log backup cluster identity is unavailable")
	backup.Status.CommitTs = ""
	backup.Status.BackupPath = "s3://bucket/backup"
	backup.Status.LogSubCommandStatuses = map[br.LogSubCommandType]br.LogSubCommandStatus{br.LogStartCommand: {Phase: br.BackupRunning}}
	_, err = GetBackupClusterForBackup(t.Context(), cli, backup)
	require.ErrorContains(t, err, "original log backup cluster identity is unavailable")
	backup.Spec.Mode = br.BackupModeSnapshot
	_, err = GetBackupClusterForBackup(t.Context(), cli, backup)
	require.ErrorContains(t, err, "original log backup cluster identity is unavailable")
	backup.Spec.Mode = br.BackupModeLog
	backup.Status.LogClusterIdentity = LogBackupClusterIdentity(cluster)
	_, err = GetBackupClusterForBackup(t.Context(), cli, backup)
	require.NoError(t, err)
	backup.Spec.BR.ClusterNamespace = "tenant"
	_, err = GetBackupClusterForBackup(t.Context(), cli, backup)
	require.NoError(t, err)
}
