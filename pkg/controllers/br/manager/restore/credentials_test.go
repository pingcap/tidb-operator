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
	"os"
	"strings"
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

// TestPiTRMixedAzureCredentials exercises actual Job injection and storage argument generation.
func TestPiTRMixedAzureCredentials(t *testing.T) {
	for _, pair := range [][2]string{{"sas", "key"}, {"sas", "aad"}, {"key", "sas"}, {"aad", "key"}, {"aad", "aad"}} {
		t.Run(pair[0]+"/"+pair[1], func(t *testing.T) {
			cluster := &core.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "tenant"}}
			kv := &core.TiKVGroup{ObjectMeta: metav1.ObjectMeta{Name: "kv", Namespace: "tenant"}}
			kv.Spec.Cluster.Name = "cluster"
			pd := &core.PDGroup{ObjectMeta: metav1.ObjectMeta{Name: "pd", Namespace: "tenant"}}
			pd.Spec.Cluster.Name = "cluster"
			cli := client.NewFakeClient(cluster, kv, pd)
			secrets := map[string]*corev1.Secret{}
			for i, name := range []string{"logs", "full"} {
				data := map[string][]byte{constants.AzblobAccountName: []byte(name + "account")}
				switch pair[i] {
				case "sas":
					data[constants.AzblobSasToken] = []byte(name + "-sas")
				case "key":
					data[constants.AzblobAccountKey] = []byte(name + "-key")
				case "aad":
					data[constants.AzblobClientID] = []byte(name + "-client")
					data[constants.AzblobClientScrt] = []byte(name + "-password")
					data[constants.AzblobTenantID] = []byte(name + "-tenant")
				}
				secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "tenant"}, Data: data}
				secrets[name] = secret
				require.NoError(t, cli.Create(context.Background(), secret))
			}
			restore := &br.Restore{ObjectMeta: metav1.ObjectMeta{Name: "restore", Namespace: "tenant"}}
			restore.Spec.BR = &br.BRConfig{Cluster: "cluster"}
			restore.Spec.Mode = br.RestoreModePiTR
			restore.Spec.Azblob = &br.AzblobStorageProvider{Container: "logs", SecretName: "logs"}
			restore.Spec.PitrFullBackupStorageProvider.Azblob = &br.AzblobStorageProvider{Container: "full", SecretName: "full"}
			manager := &restoreManager{cli: cli}
			job, reason, err := manager.makeRestoreJob(context.Background(), restore)
			if pair[0] == "aad" && pair[1] == "aad" {
				require.Error(t, err)
				require.Equal(t, "ConflictingAzureAADCredentials", reason)
				return
			}
			require.NoError(t, err, reason)
			for _, env := range job.Spec.Template.Spec.Containers[0].Env {
				value := env.Value
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
					ref := env.ValueFrom.SecretKeyRef
					value = string(secrets[ref.Name].Data[ref.Key])
				}
				t.Setenv(env.Name, value)
			}
			for i, provider := range []br.StorageProvider{restore.Spec.StorageProvider, restore.Spec.PitrFullBackupStorageProvider} {
				args, err := util.GenStorageArgsForFlag(provider, "storage")
				require.NoError(t, err)
				name := []string{"logs", "full"}[i]
				joined := strings.Join(args, " ")
				require.Contains(t, joined, "account-name="+name+"account")
				switch pair[i] {
				case "sas":
					require.Contains(t, joined, "sas-token="+name+"-sas")
					require.NotContains(t, joined, "account-key=")
				case "key":
					require.Contains(t, joined, "account-key="+name+"-key")
				case "aad":
					require.Equal(t, name+"-client", os.Getenv(constants.AzblobClientID))
					require.Equal(t, name+"-password", os.Getenv(constants.AzblobClientScrt))
				}
			}
		})
	}
}
