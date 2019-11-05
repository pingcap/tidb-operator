// Copyright 2019 PingCAP, Inc.
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
	"fmt"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	corev1 "k8s.io/api/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// CheckAllKeysExistInSecret check if all keys are included in the specific secret
func CheckAllKeysExistInSecret(secret *corev1.Secret, keys ...string) (string, bool) {
	var notExistKeys []string

	for _, key := range keys {
		if _, exist := secret.Data[key]; !exist {
			notExistKeys = append(notExistKeys, key)
		}
	}

	return strings.Join(notExistKeys, ","), len(notExistKeys) == 0
}

// GenerateCephCertEnvVar generate the env info in order to access ceph
func GenerateCephCertEnvVar(secret *corev1.Secret, endpoint string) ([]corev1.EnvVar, error) {
	var envVars []corev1.EnvVar

	if !strings.Contains(endpoint, "://") {
		// convert xxx.xxx.xxx.xxx:port to http://xxx.xxx.xxx.xxx:port
		// the endpoint must start with http://
		endpoint = fmt.Sprintf("http://%s", endpoint)
	}

	if !strings.HasPrefix(endpoint, "http://") {
		return envVars, fmt.Errorf("cenph endpoint URI %s must start with http:// scheme", endpoint)
	}
	envVars = []corev1.EnvVar{
		{
			Name:  "S3_ENDPOINT",
			Value: endpoint,
		},
		{
			Name:  "AWS_ACCESS_KEY_ID",
			Value: string(secret.Data[constants.S3AccessKey]),
		},
		{
			Name:  "AWS_SECRET_ACCESS_KEY",
			Value: string(secret.Data[constants.S3SecretKey]),
		},
	}
	return envVars, nil
}

// GenerateStorageCertEnv generate the env info in order to access backend backup storage
func GenerateStorageCertEnv(backup *v1alpha1.Backup, secretLister corelisters.SecretLister) ([]corev1.EnvVar, string, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()

	var certEnv []corev1.EnvVar

	switch backup.Spec.StorageType {
	case v1alpha1.BackupStorageTypeCeph:
		cephSecretName := backup.Spec.Ceph.SecretName
		secret, err := secretLister.Secrets(ns).Get(cephSecretName)
		if err != nil {
			err := fmt.Errorf("backup %s/%s get ceph secret %s failed, err: %v", ns, name, cephSecretName, err)
			return certEnv, "GetCephSecretFailed", err
		}

		keyStr, exist := CheckAllKeysExistInSecret(secret, constants.S3AccessKey, constants.S3SecretKey)
		if !exist {
			err := fmt.Errorf("backup %s/%s, The secret %s missing some keys %s", ns, name, cephSecretName, keyStr)
			return certEnv, "KeyNotExist", err
		}

		certEnv, err = GenerateCephCertEnvVar(secret, backup.Spec.Ceph.Endpoint)
		if err != nil {
			return certEnv, "InvalidCephEndpoint", err
		}
	default:
		err := fmt.Errorf("backup %s/%s don't support storage type %s", ns, name, backup.Spec.StorageType)
		return certEnv, "NotSupportStorageType", err
	}
	return certEnv, "", nil
}

// GetTidbUserAndPassword get the tidb user and password from specific secret
func GetTidbUserAndPassword(ns, name, tidbSecretName string, secretLister corelisters.SecretLister) (user, password, reason string, err error) {
	secret, err := secretLister.Secrets(ns).Get(tidbSecretName)
	if err != nil {
		err = fmt.Errorf("backup %s/%s get tidb secret %s failed, err: %v", ns, name, tidbSecretName, err)
		reason = "GetTidbSecretFailed"
		return
	}

	keyStr, exist := CheckAllKeysExistInSecret(secret, constants.TidbUserKey, constants.TidbPasswordKey)
	if !exist {
		err = fmt.Errorf("backup %s/%s, tidb secret %s missing some keys %s", ns, name, tidbSecretName, keyStr)
		reason = "KeyNotExist"
		return
	}

	user = string(secret.Data[constants.TidbUserKey])
	password = string(secret.Data[constants.TidbPasswordKey])
	return
}
