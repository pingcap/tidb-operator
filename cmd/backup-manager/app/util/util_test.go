// Copyright 2020 PingCAP, Inc.
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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	appconstant "github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

func TestConstructDumplingOptionsForBackup(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name              string
		hasBackupFilter   bool
		hasDumplingFilter bool
		hasOptions        bool
	}

	tests := []*testcase{
		{
			name:              "backup filter is false and dumpling config is empty",
			hasOptions:        false,
			hasBackupFilter:   false,
			hasDumplingFilter: false,
		},
		{
			name:              "backup filter is false and customize dumpling options but not set table regex",
			hasOptions:        true,
			hasBackupFilter:   false,
			hasDumplingFilter: false,
		},
		{
			name:              "backup filter is false and customize dumpling table regex but not customize options",
			hasOptions:        false,
			hasBackupFilter:   false,
			hasDumplingFilter: true,
		},
		{
			name:              "backup filter is false and customize dumpling table regex and customize options",
			hasOptions:        true,
			hasBackupFilter:   false,
			hasDumplingFilter: true,
		},
		{
			name:              "customize backup filter and dumpling config is empty",
			hasOptions:        false,
			hasBackupFilter:   true,
			hasDumplingFilter: false,
		},
		{
			name:              "customize backup filter and customize dumpling options but not set table regex",
			hasOptions:        true,
			hasBackupFilter:   true,
			hasDumplingFilter: false,
		},
		{
			name:              "customize backup filter and customize dumpling table regex but not customize options",
			hasOptions:        false,
			hasBackupFilter:   true,
			hasDumplingFilter: true,
		},
		{
			name:              "customize backup filter and customize dumpling table regex and customize options",
			hasOptions:        true,
			hasBackupFilter:   true,
			hasDumplingFilter: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backup := newBackup()

			customBackupFilter := []string{"mysql.*"}
			customDumplingFilter := []string{"mysql2.*"}
			customOptions := []string{"--consistency=snapshot"}

			var expectArgs []string

			if tt.hasBackupFilter || tt.hasDumplingFilter || tt.hasOptions {
				backup.Spec.Dumpling = &v1alpha1.DumplingConfig{}
			}

			if tt.hasBackupFilter {
				backup.Spec.TableFilter = customBackupFilter
				expectArgs = append(expectArgs, "--filter", customBackupFilter[0])
			} else {
				if tt.hasDumplingFilter {
					backup.Spec.Dumpling.TableFilter = customDumplingFilter
					expectArgs = append(expectArgs, "--filter", customDumplingFilter[0])
				} else {
					expectArgs = append(expectArgs, defaultTableFilterOptions...)
				}
			}

			if tt.hasOptions {
				backup.Spec.Dumpling.Options = customOptions
				expectArgs = append(expectArgs, customOptions...)
			} else {
				expectArgs = append(expectArgs, defaultOptions...)
			}

			generateArgs := ConstructDumplingOptionsForBackup(backup)
			g.Expect(apiequality.Semantic.DeepEqual(generateArgs, expectArgs)).To(Equal(true))
		})
	}
}

func TestGetCommitTsFromMetadata(t *testing.T) {
	g := NewGomegaWithT(t)
	tmpdir, err := ioutil.TempDir("", "test-get-commitTs-metadata")
	g.Expect(err).To(Succeed())

	defer os.RemoveAll(tmpdir)
	metaDataFileName := filepath.Join(tmpdir, appconstant.MetaDataFile)

	err = ioutil.WriteFile(metaDataFileName, []byte(`Started dump at: 2019-06-13 10:00:04
		SHOW MASTER STATUS:
			Log: tidb-binlog
			Pos: 409054741514944513
			GTID:

		Finished dump at: 2019-06-13 10:00:04`), 0644)
	g.Expect(err).To(Succeed())

	commitTs, err := GetCommitTsFromMetadata(tmpdir)
	g.Expect(err).To(Succeed())
	g.Expect(commitTs).To(Equal("409054741514944513"))
}

func newBackup() *v1alpha1.Backup {
	return &v1alpha1.Backup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Backup",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-backup",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test-bk"),
		},
		Spec: v1alpha1.BackupSpec{
			From: v1alpha1.TiDBAccessConfig{
				Host:       "10.1.1.2",
				Port:       constants.DefaultTidbPort,
				User:       constants.DefaultTidbUser,
				SecretName: "demo1-tidb-secret",
			},
			StorageProvider: v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{
					Provider:   v1alpha1.S3StorageProviderTypeCeph,
					Endpoint:   "http://10.0.0.1",
					Bucket:     "test1-demo1",
					SecretName: "demo",
				},
			},
			StorageClassName: pointer.StringPtr("local-storage"),
			StorageSize:      "1Gi",
		},
	}
}
