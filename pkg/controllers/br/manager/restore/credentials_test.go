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

package restore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	core "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/constants"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/util"
)

func TestPiTRJobUsesIndependentSASSecrets(t *testing.T) {
	cluster := &core.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "tenant"}}
	kv := &core.TiKVGroup{ObjectMeta: metav1.ObjectMeta{Name: "kv", Namespace: "tenant"}}
	kv.Spec.Cluster.Name = "cluster"
	kv.Spec.Template.Spec.Version = "v8.5.0"
	pd := &core.PDGroup{ObjectMeta: metav1.ObjectMeta{Name: "pd", Namespace: "tenant"}}
	pd.Spec.Cluster.Name = "cluster"
	cli := client.NewFakeClient(cluster, kv, pd)
	for _, name := range []string{"logs", "full"} {
		require.NoError(t, cli.Create(context.Background(), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "tenant"}, Data: map[string][]byte{constants.AzblobAccountName: []byte(name + "account"), constants.AzblobSasToken: []byte(name + "token")}}))
	}
	restore := &br.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: "tenant"}}
	restore.Spec.BR = &br.BRConfig{Cluster: "cluster"}
	restore.Spec.Mode = br.RestoreModePiTR
	restore.Spec.Azblob = &br.AzblobStorageProvider{Container: "logs", SecretName: "logs"}
	restore.Spec.PitrFullBackupStorageProvider.Azblob = &br.AzblobStorageProvider{Container: "full", SecretName: "full"}
	manager := &restoreManager{cli: cli, backupManagerImage: "backup-manager:test"}
	job, reason, err := manager.makeRestoreJob(context.Background(), restore)
	require.NoError(t, err, reason)
	envs := map[string]corev1.EnvVar{}
	for _, env := range job.Spec.Template.Spec.Containers[0].Env {
		envs[env.Name] = env
	}
	for _, name := range []string{"logs", "full"} {
		token := envs[util.AzblobSecretEnvName(constants.AzblobSasToken, name)]
		require.Empty(t, token.Value)
		require.NotNil(t, token.ValueFrom)
		require.Equal(t, name, token.ValueFrom.SecretKeyRef.Name)
		require.Equal(t, constants.AzblobSasToken, token.ValueFrom.SecretKeyRef.Key)
		account := envs[util.AzblobSecretEnvName(constants.AzblobAccountName, name)]
		require.Equal(t, name+"account", account.Value)
	}
	require.Equal(t, "logsaccount", envs[constants.AzblobAccountName].Value)
	restore.Spec.Mode = br.RestoreModeSnapshot
	restore.Spec.PitrFullBackupStorageProvider.Azblob.SecretName = "missing-unused-secret"
	_, reason, err = manager.makeRestoreJob(context.Background(), restore)
	require.NoError(t, err, reason)
}
