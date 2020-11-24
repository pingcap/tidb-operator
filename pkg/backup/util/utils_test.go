// Copyright 2018 PingCAP, Inc.
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

package util

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCheckAllKeysExistInSecret(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		secret       *corev1.Secret
		keys         []string
		notExistKeys string
		exist        bool
	}{
		{
			secret:       &corev1.Secret{},
			keys:         nil,
			notExistKeys: "",
			exist:        true,
		},
		{
			secret:       &corev1.Secret{},
			keys:         []string{"a"},
			notExistKeys: "a",
			exist:        false,
		},
		{
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"a": nil,
				},
			},
			keys:         []string{"a"},
			notExistKeys: "",
			exist:        true,
		},
		{
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"a": nil,
				},
			},
			keys:         []string{"a", "b", "c"},
			notExistKeys: "b,c",
			exist:        false,
		},
	}

	for _, test := range tests {
		t.Logf("test: %+v", test)
		getKey, exist := CheckAllKeysExistInSecret(test.secret, test.keys...)
		g.Expect(getKey).Should(Equal(test.notExistKeys))
		g.Expect(exist).Should(Equal(test.exist))
	}
}

func TestGenerateS3CertEnvVar(t *testing.T) {
	g := NewGomegaWithT(t)

	var s3 *v1alpha1.S3StorageProvider

	contains := func(envs []corev1.EnvVar, name string, value string) {
		for _, e := range envs {
			if e.Name == name {
				g.Expect(e.Value).Should(Equal(value))
				return
			}
		}
		t.Fatalf("env %s not exist", name)
	}

	// test v1alpha1.S3StorageProviderTypeCeph endpoint append scheme
	s3 = &v1alpha1.S3StorageProvider{
		Provider: v1alpha1.S3StorageProviderTypeCeph,
		Endpoint: "host:80",
	}
	envs, _, err := generateS3CertEnvVar(s3, false)
	g.Expect(err).Should(BeNil())
	contains(envs, "S3_ENDPOINT", "http://host:80")

	// test v1alpha1.S3StorageProviderTypeCeph error endpoint
	s3.Endpoint = "://host:80"
	_, _, err = generateS3CertEnvVar(s3, false)
	g.Expect(err).ShouldNot(BeNil())

	// dummy test
	s3.Endpoint = "http://host:80"
	_, _, err = generateS3CertEnvVar(s3, false)
	g.Expect(err).Should(BeNil())
	s3.Provider = v1alpha1.S3StorageProviderTypeAWS
	_, _, err = generateS3CertEnvVar(s3, true)
	g.Expect(err).Should(BeNil())
}

func TestGetPasswordKey(t *testing.T) {
	g := NewGomegaWithT(t)
	var key string

	key = getPasswordKey(true)
	ar := []string{constants.KMSSecretPrefix, constants.BackupManagerEnvVarPrefix, strings.ToUpper(constants.TidbPasswordKey)}
	g.Expect(key).Should(Equal(strings.Join(ar, "_")))

	key = getPasswordKey(false)
	ar = []string{constants.BackupManagerEnvVarPrefix, strings.ToUpper(constants.TidbPasswordKey)}
	g.Expect(key).Should(Equal(strings.Join(ar, "_")))
}

func TestGenerateGcsCertEnvVar(t *testing.T) {
	g := NewGomegaWithT(t)
	var gcs *v1alpha1.GcsStorageProvider

	// test error case
	gcs = &v1alpha1.GcsStorageProvider{
		ProjectId: "",
	}
	_, _, err := generateGcsCertEnvVar(gcs)
	g.Expect(err).ShouldNot(BeNil())

	// test normal case
	gcs = &v1alpha1.GcsStorageProvider{
		ProjectId: "id",
	}
	envs, _, err := generateGcsCertEnvVar(gcs)
	g.Expect(err).Should(BeNil())
	g.Expect(len(envs)).ShouldNot(Equal(0))
}

func TestGenerateStorageCertEnv(t *testing.T) {
	g := NewGomegaWithT(t)
	ns := "ns"
	secretName := "secretName"

	tests := []struct {
		provider v1alpha1.StorageProvider
		name     string
	}{
		{
			provider: v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{
					SecretName: secretName,
				},
			},
		},
		{
			provider: v1alpha1.StorageProvider{
				Gcs: &v1alpha1.GcsStorageProvider{
					SecretName: secretName,
					ProjectId:  "id",
				},
			},
		},
		{
			provider: v1alpha1.StorageProvider{},
		},
	}

	for _, test := range tests {
		tmp := v1alpha1.StorageProvider{}
		client := fake.NewSimpleClientset()

		// handle unsupported storage type
		if test.provider == tmp {
			_, _, err := GenerateStorageCertEnv(ns, false, test.provider, client)
			g.Expect(err.Error()).Should(MatchRegexp(".*unsupported storage type.*"))
			continue
		}

		// start normal storage type
		_, _, err := GenerateStorageCertEnv(ns, false, test.provider, client)
		g.Expect(err.Error()).Should(MatchRegexp(".*get.*secret.*"))
		// create secret and missing key in secret
		s := &corev1.Secret{}
		s.Namespace = ns
		s.Name = secretName
		_, err = client.CoreV1().Secrets(ns).Create(s)
		g.Expect(err).Should(BeNil())
		_, _, err = GenerateStorageCertEnv(ns, false, test.provider, client)
		g.Expect(err.Error()).Should(MatchRegexp(".*missing some keys.*"))
		// update secret with need key
		s.Data = map[string][]byte{
			constants.TidbPasswordKey:   []byte("dummy"),
			constants.GcsCredentialsKey: []byte("dummy"),
			constants.S3AccessKey:       []byte("dummy"),
			constants.S3SecretKey:       []byte("dummy"),
		}
		_, err = client.CoreV1().Secrets(ns).Update(s)
		g.Expect(err).Should(BeNil())
		_, _, err = GenerateStorageCertEnv(ns, false, test.provider, client)
		g.Expect(err).Should(BeNil())
	}
}

func TestGenerateTidbPasswordEnv(t *testing.T) {
	g := NewGomegaWithT(t)
	ns := "ns"
	tcName := "tctest"
	secretName := "secretName"
	client := fake.NewSimpleClientset()

	// test fail to get secret
	_, _, err := GenerateTidbPasswordEnv(ns, tcName, secretName, false, client)
	g.Expect(err.Error()).Should(MatchRegexp(".*get tidb secret.*"))

	// create secret and not exist constants.TidbPasswordKey key in secret
	s := &corev1.Secret{}
	s.Namespace = ns
	s.Name = secretName
	_, err = client.CoreV1().Secrets(ns).Create(s)
	g.Expect(err).Should(BeNil())
	_, _, err = GenerateTidbPasswordEnv(ns, tcName, secretName, false, client)
	g.Expect(err.Error()).Should(MatchRegexp(".*missing password key.*"))

	// update secret with need key
	s.Data = map[string][]byte{
		constants.TidbPasswordKey: []byte("dummy"),
	}
	_, err = client.CoreV1().Secrets(ns).Update(s)
	g.Expect(err).Should(BeNil())
	envs, _, err := GenerateTidbPasswordEnv(ns, tcName, secretName, false, client)
	g.Expect(err).Should(BeNil())
	g.Expect(len(envs)).ShouldNot(Equal(0))
}

func TestGetBackupBucketAdnPrefixName(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		backup *v1alpha1.Backup
		name   string
	}{
		{
			backup: &v1alpha1.Backup{
				Spec: v1alpha1.BackupSpec{
					StorageProvider: v1alpha1.StorageProvider{
						S3: &v1alpha1.S3StorageProvider{
							Bucket: "s3",
							Prefix: "s3",
						},
					},
				},
			},
			name: "s3",
		},
		{
			backup: &v1alpha1.Backup{
				Spec: v1alpha1.BackupSpec{
					StorageProvider: v1alpha1.StorageProvider{
						Gcs: &v1alpha1.GcsStorageProvider{
							Bucket: "gcs",
							Prefix: "gcs",
						},
					},
				},
			},
			name: "gcs",
		},
		{
			backup: &v1alpha1.Backup{},
			name:   "",
		},
	}

	for _, test := range tests {
		name, _, err := GetBackupBucketName(test.backup)
		if test.name == "" {
			g.Expect(err).ShouldNot(BeNil())
		} else {
			g.Expect(err).Should(BeNil())
			g.Expect(name).Should(Equal(test.name))
		}

		name, _, err = GetBackupPrefixName(test.backup)
		if test.name == "" {
			g.Expect(err).ShouldNot(BeNil())
		} else {
			g.Expect(err).Should(BeNil())
			g.Expect(name).Should(Equal(test.name))
		}
	}
}

func TestGetBackupDataPath(t *testing.T) {
	g := NewGomegaWithT(t)

	tests := []struct {
		provider v1alpha1.StorageProvider
		name     string
	}{
		{
			provider: v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{
					Path: "s3://host",
				},
			},
			name: "s3://host",
		},
		{
			provider: v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{
					Path: "host",
				},
			},
			name: "s3://host",
		},
		{
			provider: v1alpha1.StorageProvider{
				Gcs: &v1alpha1.GcsStorageProvider{
					Path: "gcs://host",
				},
			},
			name: "gcs://host",
		},
		{
			provider: v1alpha1.StorageProvider{
				Gcs: &v1alpha1.GcsStorageProvider{
					Path: "host",
				},
			},
			name: "gcs://host",
		},
		{
			provider: v1alpha1.StorageProvider{},
			name:     "",
		},
	}

	for _, test := range tests {
		name, _, err := GetBackupDataPath(test.provider)
		if test.name == "" {
			g.Expect(err).ShouldNot(BeNil())
		} else {
			g.Expect(err).Should(BeNil())
			g.Expect(name).Should(Equal(test.name))
		}
	}
}

func TestValidateBackup(t *testing.T) {
	g := NewGomegaWithT(t)

	backup := new(v1alpha1.Backup)
	match := func(sub string) {
		t.Helper()
		err := ValidateBackup(backup, "tikv:v4.0.8")
		if sub == "" {
			g.Expect(err).Should(BeNil())
		} else {
			g.Expect(err).ShouldNot(BeNil())
			g.Expect(err.Error()).Should(MatchRegexp(".*" + sub + ".*"))
		}
	}

	// BR == nil case
	match("missing cluster config in spec of")

	backup.Spec.From = &v1alpha1.TiDBAccessConfig{}
	backup.Spec.From.Host = "localhost"
	match("missing tidbSecretName config in spec")

	backup.Spec.From.SecretName = "secretName"
	match("missing StorageSize config in spec of")
	backup.Spec.StorageSize = "1m"
	match("")

	// start BR != nil case
	backup.Spec.BR = &v1alpha1.BRConfig{}
	match("cluster should be configured for BR in spec")

	backup.Spec.BR.Cluster = "tidb"
	backup.Spec.Type = v1alpha1.BackupType("invalid")
	match("invalid backup type")

	backup.Spec.Type = v1alpha1.BackupTypeDB
	match("DB should be configured for BR with backup type")

	backup.Spec.BR.DB = "dbName"
	backup.Spec.Type = v1alpha1.BackupTypeTable
	match("table should be configured for BR with backup type table in spec of")

	backup.Spec.BR.Table = "tableName"
	backup.Spec.S3 = &v1alpha1.S3StorageProvider{}
	match("bucket should be configured for BR in spec of")

	backup.Spec.S3.Bucket = "bucket"
	backup.Spec.S3.Endpoint = "#$@$#^%**##"
	match("invalid endpoint")

	backup.Spec.S3.Endpoint = "/path"
	match("scheme not found in endpoint")

	backup.Spec.S3.Endpoint = "s3:///"
	match("host not found in endpoint")

	backup.Spec.S3.Endpoint = "s3://localhost:80"
	match("")
}

func TestValidateRestore(t *testing.T) {
	g := NewGomegaWithT(t)

	restore := new(v1alpha1.Restore)
	match := func(sub string) {
		t.Helper()
		err := ValidateRestore(restore, "tikv:v4.0.8")
		if sub == "" {
			g.Expect(err).Should(BeNil())
		} else {
			g.Expect(err).ShouldNot(BeNil())
			g.Expect(err.Error()).Should(MatchRegexp(".*" + sub + ".*"))
		}
	}

	// BR == nil case
	match("missing cluster config in spec of")

	restore.Spec.To = &v1alpha1.TiDBAccessConfig{}
	restore.Spec.To.Host = "localhost"
	match("missing tidbSecretName config in spec")

	restore.Spec.To.SecretName = "secretName"
	match("missing StorageSize config in spec of")
	restore.Spec.StorageSize = "1m"
	match("")

	// start BR != nil case
	restore.Spec.BR = &v1alpha1.BRConfig{}
	match("cluster should be configured for BR in spec")

	restore.Spec.BR.Cluster = "tidb"
	restore.Spec.Type = v1alpha1.BackupType("invalid")
	match("invalid backup type")

	restore.Spec.Type = v1alpha1.BackupTypeDB
	match("DB should be configured for BR with restore type")

	restore.Spec.BR.DB = "dbName"
	restore.Spec.Type = v1alpha1.BackupTypeTable
	match("table should be configured for BR with restore type table in spec of")

	restore.Spec.BR.Table = "tableName"
	restore.Spec.S3 = &v1alpha1.S3StorageProvider{}
	match("bucket should be configured for BR in spec of")

	restore.Spec.S3.Bucket = "bucket"
	restore.Spec.S3.Endpoint = "#$@$#^%**##"
	match("invalid endpoint")

	restore.Spec.S3.Endpoint = "/path"
	match("scheme not found in endpoint")

	restore.Spec.S3.Endpoint = "s3:///"
	match("host not found in endpoint")

	restore.Spec.S3.Endpoint = "s3://localhost:80"
	match("")
}

func TestGetImageTag(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name      string
		image     string
		imageName string
		tag       string
	}

	tests := []*testcase{
		{
			name:      "with repo",
			image:     "localhost:5000/tikv:v3.1.0",
			imageName: "localhost:5000/tikv",
			tag:       "v3.1.0",
		},
		{
			name:      "no colon",
			image:     "tikv",
			imageName: "tikv",
			tag:       "",
		},
		{
			name:      "no repo",
			image:     "tikv:nightly",
			imageName: "tikv",
			tag:       "nightly",
		},
		{
			name:      "start with colon",
			image:     ":v4.0.0",
			imageName: "",
			tag:       "v4.0.0",
		},
		{
			name:      "end with colon",
			image:     "tikv:",
			imageName: "tikv",
			tag:       "",
		},
		{
			name:      "only colon",
			image:     ":",
			imageName: "",
			tag:       "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			name, version := ParseImage(test.image)
			g.Expect(version).To(Equal(test.tag))
			g.Expect(name).To(Equal(test.imageName))
		})
	}
}
