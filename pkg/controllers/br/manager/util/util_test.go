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
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha1 "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/br/manager/constants"
)

func TestGenerateStorageCertEnvAzblob(t *testing.T) {
	const namespace = "test"
	const secretName = "storage"
	sensitiveValue := []byte("test-sensitive-value")
	tests := []struct {
		name           string
		account        string
		sasToken       string
		secretData     map[string][]byte
		omitSecretName bool
		wantReason     string
		wantEnvKey     string
	}{
		{name: "missing account in populated secret", secretData: map[string][]byte{"unrelated-password": sensitiveValue}, wantReason: "azblobAccountNotExist"},
		{name: "empty account in populated secret", secretData: map[string][]byte{constants.AzblobAccountName: nil, constants.AzblobAccountKey: sensitiveValue}, wantReason: "azblobAccountNotExist"},
		{name: "empty secret", secretData: map[string][]byte{}, wantReason: "azblobAccountNotExist"},
		{name: "missing account and secret", omitSecretName: true, wantReason: "azblobAccountNotExist"},
		{name: "missing account with SAS and no secret", sasToken: "test-sas", omitSecretName: true, wantReason: "azblobAccountNotExist"},
		{name: "explicit account without authentication secret", account: "test-account", omitSecretName: true, wantReason: "azblobKeyOrAADMissing"},
		{name: "named secret does not exist", account: "test-account", wantReason: "GetAzblobSecretFailed"},
		{name: "explicit account with SAS and no secret", account: "test-account", sasToken: "test-sas", omitSecretName: true},
		{name: "account from secret with SAS", sasToken: "test-sas", secretData: map[string][]byte{constants.AzblobAccountName: []byte("test-account")}},
		{name: "shared key with account from secret", secretData: map[string][]byte{constants.AzblobAccountName: []byte("test-account"), constants.AzblobAccountKey: sensitiveValue}, wantEnvKey: constants.AzblobAccountKey},
		{name: "shared key with explicit account", account: "test-account", secretData: map[string][]byte{constants.AzblobAccountKey: sensitiveValue}, wantEnvKey: constants.AzblobAccountKey},
		{name: "AAD with account from secret", secretData: map[string][]byte{constants.AzblobAccountName: []byte("test-account"), constants.AzblobClientID: []byte("test-client"), constants.AzblobClientScrt: sensitiveValue, constants.AzblobTenantID: []byte("test-tenant")}, wantEnvKey: constants.AzblobClientScrt},
		{name: "missing authentication keys", account: "test-account", secretData: map[string][]byte{"unrelated-password": sensitiveValue}, wantReason: "azblobKeyOrAADMissing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := v1alpha1.StorageProvider{Azblob: &v1alpha1.AzblobStorageProvider{StorageAccount: tt.account, SasToken: tt.sasToken}}
			if !tt.omitSecretName {
				provider.Azblob.SecretName = secretName
			}
			builder := fake.NewClientBuilder()
			if tt.secretData != nil {
				builder = builder.WithObjects(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: secretName}, Data: tt.secretData})
			}
			cli := builder.Build()
			envs, reason, err := GenerateStorageCertEnv(context.Background(), namespace, false, provider, cli)
			g := NewGomegaWithT(t)
			g.Expect(reason).To(Equal(tt.wantReason))
			if tt.wantReason != "" {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).NotTo(ContainSubstring(string(sensitiveValue)))
				g.Expect(err.Error()).NotTo(ContainSubstring(fmt.Sprintf("%d", sensitiveValue)))
				return
			}
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(envs).To(ContainElement(corev1.EnvVar{Name: constants.AzblobAccountName, Value: "test-account"}))
			if tt.wantEnvKey != "" {
				g.Expect(envs).To(ContainElement(corev1.EnvVar{
					Name: tt.wantEnvKey,
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
							Key:                  tt.wantEnvKey,
						},
					},
				}))
			}
			for _, env := range envs {
				g.Expect(env.Value).NotTo(Equal(string(sensitiveValue)))
			}
		})
	}
}
