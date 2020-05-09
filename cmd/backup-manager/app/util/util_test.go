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
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

func TestConstructMydumperOptionsForBackup(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name       string
		hasRegex   bool
		hasOptions bool
	}

	tests := []*testcase{
		{
			name:       "mydumper config is empty",
			hasOptions: false,
			hasRegex:   false,
		},
		{
			name:       "customize mydumper options but not set table regex",
			hasOptions: true,
			hasRegex:   false,
		},
		{
			name:       "customize mydumper table regex but not customize options",
			hasOptions: false,
			hasRegex:   true,
		},
		{
			name:       "customize mydumper table regex and customize options",
			hasOptions: true,
			hasRegex:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backup := newBackup()

			customRegex := "^mysql"
			customOptions := []string{"--long-query-guard=3000"}

			var expectArgs []string

			if tt.hasOptions {
				backup.Spec.Mydumper = &v1alpha1.MydumperConfig{Options: customOptions}
				expectArgs = append(expectArgs, customOptions...)
			} else {
				expectArgs = append(expectArgs, defaultOptions...)
			}

			if tt.hasRegex {
				if backup.Spec.Mydumper == nil {
					backup.Spec.Mydumper = &v1alpha1.MydumperConfig{TableRegex: &customRegex}
				} else {
					backup.Spec.Mydumper.TableRegex = &customRegex
				}
				expectArgs = append(expectArgs, "--regex", customRegex)
			} else {
				expectArgs = append(expectArgs, defaultTableRegexOptions...)
			}

			generateArgs := ConstructMydumperOptionsForBackup(backup)
			g.Expect(apiequality.Semantic.DeepEqual(generateArgs, expectArgs)).To(Equal(true))
		})
	}
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
