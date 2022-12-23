// Copyright 2021 PingCAP, Inc.
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

package backup

import (
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/pkg/tkctl/util"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	BRType          = "br"
	DumperType      = "dumper"
	TiDBServicePort = int32(4000)
	PDServicePort   = int32(2379)
)

// GetRole returns a role for br test.
func GetRole(ns string) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.DefaultServiceAccountName,
			Namespace: ns,
			Labels: map[string]string{
				label.ComponentLabelKey: constants.DefaultServiceAccountName,
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{"pingcap.com"},
				Resources: []string{"backups", "restores"},
				Verbs:     []string{"get", "watch", "list", "update"},
			},
		},
	}
}

// GetServiceAccount returns a sa for br test.
func GetServiceAccount(ns string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.DefaultServiceAccountName,
			Namespace: ns,
		},
	}
}

// GetRoleBinding returns a rolebinding for br test.
func GetRoleBinding(ns string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.DefaultServiceAccountName,
			Namespace: ns,
			Labels: map[string]string{
				label.ComponentLabelKey: constants.DefaultServiceAccountName,
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: rbacv1.ServiceAccountKind,
				Name: constants.DefaultServiceAccountName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     constants.DefaultServiceAccountName,
		},
	}
}

// GetSecret returns a secret to visit tidb.
func GetSecret(ns, name, password string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Data: map[string][]byte{
			"password": []byte(password),
		},
		Type: corev1.SecretTypeOpaque,
	}
}

// GetBackup return a basic backup
func GetBackup(ns, name, tcName, typ string, s3Config *v1alpha1.S3StorageProvider) *v1alpha1.Backup {
	if typ != BRType && typ != DumperType {
		return nil
	}
	sendCredToTikv := true
	br := &v1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: v1alpha1.BackupSpec{
			Type: v1alpha1.BackupTypeFull,
			StorageProvider: v1alpha1.StorageProvider{
				S3: s3Config,
			},
			From: &v1alpha1.TiDBAccessConfig{
				Host:       util.GetTidbServiceName(tcName),
				SecretName: name,
				Port:       TiDBServicePort,
				User:       "root",
			},
			BR: &v1alpha1.BRConfig{
				Cluster:          tcName,
				ClusterNamespace: ns,
				SendCredToTikv:   &sendCredToTikv,
			},
			CleanPolicy: v1alpha1.CleanPolicyTypeDelete,
		},
	}
	if typ == DumperType {
		br.Spec.BR = nil
		br.Spec.StorageSize = "1Gi"
	}
	return br
}

func GetRestore(ns, name, tcName, typ string, s3Config *v1alpha1.S3StorageProvider) *v1alpha1.Restore {
	if typ != BRType && typ != DumperType {
		return nil
	}
	sendCredToTikv := true
	restore := &v1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: v1alpha1.RestoreSpec{
			Type: v1alpha1.BackupTypeFull,
			StorageProvider: v1alpha1.StorageProvider{
				S3: s3Config,
			},
			To: &v1alpha1.TiDBAccessConfig{
				Host:       util.GetTidbServiceName(tcName),
				SecretName: name,
				Port:       TiDBServicePort,
				User:       "root",
			},
			BR: &v1alpha1.BRConfig{
				Cluster:           tcName,
				ClusterNamespace:  ns,
				SendCredToTikv:    &sendCredToTikv,
				CheckRequirements: pointer.BoolPtr(false), // workaround for https://docs.pingcap.com/tidb/stable/backup-and-restore-faq#why-does-br-report-new_collations_enabled_on_first_bootstrap-mismatch
			},
		},
	}
	if typ == DumperType {
		restore.Spec.BR = nil
		restore.Spec.StorageSize = "1Gi"
	}
	return restore
}
