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

package storage

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// TestStorage Provide Object Storage Ability in Test
type TestStorage interface {
	// ProvideCredential provide the credential to connect to the TestStorage
	ProvideCredential(ns string) *corev1.Secret
	// ProvideBackup provide the Backup to save data in the TestStorage
	ProvideBackup(tc *v1alpha1.TidbCluster, fromSecret *corev1.Secret) *v1alpha1.Backup
	// ProvideRestore provide the Restore to restore data from the Test Storage
	ProvideRestore(tc *v1alpha1.TidbCluster, toSecret *corev1.Secret) *v1alpha1.Restore
	// CheckDataCleaned check whether TestStorage Clean
	CheckDataCleaned() error
}

