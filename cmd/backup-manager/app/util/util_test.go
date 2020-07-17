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
		name       string
		hasFilter  bool
		hasOptions bool
	}

	tests := []*testcase{
		{
			name:       "dumpling config is empty",
			hasOptions: false,
			hasFilter:  false,
		},
		{
			name:       "customize dumpling options but not set table regex",
			hasOptions: true,
			hasFilter:  false,
		},
		{
			name:       "customize dumpling table regex but not customize options",
			hasOptions: false,
			hasFilter:  true,
		},
		{
			name:       "customize dumpling table regex and customize options",
			hasOptions: true,
			hasFilter:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backup := newBackup()

			customFilter := []string{"mysql.*"}
			customOptions := []string{"--consistency=snapshot"}

			var expectArgs []string

			if tt.hasOptions {
				backup.Spec.Dumpling = &v1alpha1.DumplingConfig{Options: customOptions}
				expectArgs = append(expectArgs, customOptions...)
			} else {
				expectArgs = append(expectArgs, defaultOptions...)
			}

			if tt.hasFilter {
				if backup.Spec.Dumpling == nil {
					backup.Spec.Dumpling = &v1alpha1.DumplingConfig{TableFilter: customFilter}
				} else {
					backup.Spec.Dumpling.TableFilter = customFilter
				}
				expectArgs = append(expectArgs, "--filter", customFilter[0])
			} else {
				expectArgs = append(expectArgs, defaultTableFilterOptions...)
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
