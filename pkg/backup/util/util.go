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

// GenerateS3CertEnvVar generate the env info in order to access S3 compliant storage
func GenerateS3CertEnvVar(secret *corev1.Secret, s3 *v1alpha1.S3StorageProvider) ([]corev1.EnvVar, error) {
	var envVars []corev1.EnvVar

	switch s3.Provider {
	case v1alpha1.S3StorageProviderTypeCeph:
		if !strings.Contains(s3.Endpoint, "://") {
			// convert xxx.xxx.xxx.xxx:port to http://xxx.xxx.xxx.xxx:port
			// the endpoint must start with http://
			s3.Endpoint = fmt.Sprintf("http://%s", s3.Endpoint)
			break
		}
		if !strings.HasPrefix(s3.Endpoint, "http://") {
			return envVars, fmt.Errorf("cenph endpoint URI %s must start with http://", s3.Endpoint)
		}
	case v1alpha1.S3StorageProviderTypeAWS:
		// TODO: Check the storage class, if it is not a legal storage class, use the default storage class instead
		if len(s3.StorageClass) == 0 {
			// The optional storage class reference https://rclone.org/s3
			s3.StorageClass = "STANDARD_IA"
		}
		if len(s3.Acl) == 0 {
			// The optional acl reference https://rclone.org/s3/
			s3.Acl = "private"
		}
	default:
		return envVars, fmt.Errorf("unknow s3 compliant storage type %s", s3.Provider)
	}

	envVars = []corev1.EnvVar{
		{
			Name:  "S3_PROVIDER",
			Value: string(s3.Provider),
		},
		{
			Name:  "S3_ENDPOINT",
			Value: s3.Endpoint,
		},
		{
			Name:  "AWS_REGION",
			Value: s3.Region,
		},
		{
			Name:  "AWS_ACL",
			Value: s3.Acl,
		},
		{
			Name:  "AWS_STORAGE_CLASS",
			Value: s3.StorageClass,
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
	case v1alpha1.BackupStorageTypeS3:
		s3SecretName := backup.Spec.S3.SecretName
		secret, err := secretLister.Secrets(ns).Get(s3SecretName)
		if err != nil {
			err := fmt.Errorf("backup %s/%s get s3 secret %s failed, err: %v", ns, name, s3SecretName, err)
			return certEnv, "GetS3SecretFailed", err
		}

		keyStr, exist := CheckAllKeysExistInSecret(secret, constants.S3AccessKey, constants.S3SecretKey)
		if !exist {
			err := fmt.Errorf("backup %s/%s, The s3 secret %s missing some keys %s", ns, name, s3SecretName, keyStr)
			return certEnv, "KeyNotExist", err
		}

		certEnv, err = GenerateS3CertEnvVar(secret, backup.Spec.S3.DeepCopy())
		if err != nil {
			return certEnv, "InvalidS3Endpoint", err
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
