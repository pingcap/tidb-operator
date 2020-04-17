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
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
func GenerateS3CertEnvVar(s3 *v1alpha1.S3StorageProvider, useKMS bool) ([]corev1.EnvVar, string, error) {
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
			return envVars, "InvalidS3Endpoint", fmt.Errorf("ceph endpoint URI %s must start with http://", s3.Endpoint)
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
	}
	if useKMS {
		envVars = append(envVars, []corev1.EnvVar{
			{
				Name:  "AWS_DEFAULT_REGION",
				Value: s3.Region,
			},
		}...)
	}
	if s3.SecretName != "" {
		envVars = append(envVars, []corev1.EnvVar{
			{
				Name: "AWS_ACCESS_KEY_ID",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: s3.SecretName},
						Key:                  constants.S3AccessKey,
					},
				},
			},
			{
				Name: "AWS_SECRET_ACCESS_KEY",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: s3.SecretName},
						Key:                  constants.S3SecretKey,
					},
				},
			},
		}...)
	}
	return envVars, "", nil
}

// GenerateGcsCertEnvVar generate the env info in order to access google cloud storage
func GenerateGcsCertEnvVar(gcs *v1alpha1.GcsStorageProvider) ([]corev1.EnvVar, string, error) {
	if len(gcs.ProjectId) == 0 {
		return nil, "ProjectIdIsEmpty", fmt.Errorf("the project id is not set")
	}
	envVars := []corev1.EnvVar{
		{
			Name:  "GCS_PROJECT_ID",
			Value: gcs.ProjectId,
		},
		{
			Name:  "GCS_OBJECT_ACL",
			Value: gcs.ObjectAcl,
		},
		{
			Name:  "GCS_BUCKET_ACL",
			Value: gcs.BucketAcl,
		},
		{
			Name:  "GCS_LOCATION",
			Value: gcs.Location,
		},
		{
			Name:  "GCS_STORAGE_CLASS",
			Value: gcs.StorageClass,
		},
		{
			Name: "GCS_SERVICE_ACCOUNT_JSON_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: gcs.SecretName},
					Key:                  constants.GcsCredentialsKey,
				},
			},
		},
	}
	return envVars, "", nil
}

// GenerateStorageCertEnv generate the env info in order to access backend backup storage
func GenerateStorageCertEnv(ns string, useKMS bool, provider v1alpha1.StorageProvider, kubeCli kubernetes.Interface) ([]corev1.EnvVar, string, error) {
	var certEnv []corev1.EnvVar
	var reason string
	var err error
	storageType := GetStorageType(provider)

	switch storageType {
	case v1alpha1.BackupStorageTypeS3:
		if provider.S3 == nil {
			return certEnv, "S3ConfigIsEmpty", errors.New("s3 config is empty")
		}

		s3SecretName := provider.S3.SecretName
		if s3SecretName != "" {
			secret, err := kubeCli.CoreV1().Secrets(ns).Get(s3SecretName, metav1.GetOptions{})
			if err != nil {
				err := fmt.Errorf("get s3 secret %s/%s failed, err: %v", ns, s3SecretName, err)
				return certEnv, "GetS3SecretFailed", err
			}

			keyStr, exist := CheckAllKeysExistInSecret(secret, constants.S3AccessKey, constants.S3SecretKey)
			if !exist {
				err := fmt.Errorf("s3 secret %s/%s missing some keys %s", ns, s3SecretName, keyStr)
				return certEnv, "s3KeyNotExist", err
			}
		}

		certEnv, reason, err = GenerateS3CertEnvVar(provider.S3.DeepCopy(), useKMS)
		if err != nil {
			return certEnv, reason, err
		}
	case v1alpha1.BackupStorageTypeGcs:
		if provider.Gcs == nil {
			return certEnv, "GcsConfigIsEmpty", errors.New("gcs config is empty")
		}
		gcsSecretName := provider.Gcs.SecretName
		secret, err := kubeCli.CoreV1().Secrets(ns).Get(gcsSecretName, metav1.GetOptions{})
		if err != nil {
			err := fmt.Errorf("get gcs secret %s/%s failed, err: %v", ns, gcsSecretName, err)
			return certEnv, "GetGcsSecretFailed", err
		}

		keyStr, exist := CheckAllKeysExistInSecret(secret, constants.GcsCredentialsKey)
		if !exist {
			err := fmt.Errorf("the gcs secret %s/%s missing some keys %s", ns, gcsSecretName, keyStr)
			return certEnv, "gcsKeyNotExist", err
		}

		certEnv, reason, err = GenerateGcsCertEnvVar(provider.Gcs)

		if err != nil {
			return certEnv, reason, err
		}
	default:
		err := fmt.Errorf("unsupported storage type %s", storageType)
		return certEnv, "UnsupportedStorageTye", err
	}
	return certEnv, reason, nil
}

// GenerateTidbPasswordEnv generate the password EnvVar
func GenerateTidbPasswordEnv(ns, name, tidbSecretName string, useKMS bool, kubeCli kubernetes.Interface) ([]corev1.EnvVar, string, error) {
	var certEnv []corev1.EnvVar
	var passwordKey string
	secret, err := kubeCli.CoreV1().Secrets(ns).Get(tidbSecretName, metav1.GetOptions{})
	if err != nil {
		err = fmt.Errorf("backup %s/%s get tidb secret %s failed, err: %v", ns, name, tidbSecretName, err)
		return certEnv, "GetTidbSecretFailed", err
	}

	keyStr, exist := CheckAllKeysExistInSecret(secret, constants.TidbPasswordKey)
	if !exist {
		err = fmt.Errorf("backup %s/%s, tidb secret %s missing password key %s", ns, name, tidbSecretName, keyStr)
		return certEnv, "KeyNotExist", err
	}

	if useKMS {
		passwordKey = fmt.Sprintf("%s_%s_%s", constants.KMSSecretPrefix, constants.BackupManagerEnvVarPrefix, strings.ToUpper(constants.TidbPasswordKey))
	} else {
		passwordKey = fmt.Sprintf("%s_%s", constants.BackupManagerEnvVarPrefix, strings.ToUpper(constants.TidbPasswordKey))
	}

	certEnv = []corev1.EnvVar{
		{
			Name: passwordKey,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: tidbSecretName},
					Key:                  constants.TidbPasswordKey,
				},
			},
		},
	}
	return certEnv, "", nil
}

// GetBackupBucketName return the bucket name for remote storage
func GetBackupBucketName(backup *v1alpha1.Backup) (string, string, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()
	storageType := GetStorageType(backup.Spec.StorageProvider)
	var bucketName string

	switch storageType {
	case v1alpha1.BackupStorageTypeS3:
		bucketName = backup.Spec.S3.Bucket
	case v1alpha1.BackupStorageTypeGcs:
		bucketName = backup.Spec.Gcs.Bucket
	default:
		return bucketName, "UnsupportedStorageType", fmt.Errorf("backup %s/%s unsupported storage type %s", ns, name, storageType)
	}
	return bucketName, "", nil
}

// GetStorageType return the backup storage type according to the specified StorageProvider
func GetStorageType(provider v1alpha1.StorageProvider) v1alpha1.BackupStorageType {
	// If there are multiple storages in the StorageProvider, the first one found is returned in the following order
	if provider.S3 != nil {
		return v1alpha1.BackupStorageTypeS3
	}
	if provider.Gcs != nil {
		return v1alpha1.BackupStorageTypeGcs
	}
	return v1alpha1.BackupStorageTypeUnknown
}

// GetBackupDataPath return the full path of backup data
func GetBackupDataPath(provider v1alpha1.StorageProvider) (string, string, error) {
	storageType := GetStorageType(provider)
	var backupPath string

	switch storageType {
	case v1alpha1.BackupStorageTypeS3:
		backupPath = provider.S3.Path
	case v1alpha1.BackupStorageTypeGcs:
		backupPath = provider.Gcs.Path
	default:
		return backupPath, "UnsupportedStorageType", fmt.Errorf("unsupported storage type %s", storageType)
	}
	protocolPrefix := fmt.Sprintf("%s://", string(storageType))
	if strings.HasPrefix(backupPath, protocolPrefix) {
		// if the full path of backup data start with "<storageType>://", return directly.
		return backupPath, "", nil
	}
	return fmt.Sprintf("%s://%s", string(storageType), backupPath), "", nil
}

func ValidateBackup(backup *v1alpha1.Backup) error {
	ns := backup.Namespace
	name := backup.Name

	if backup.Spec.From.Host == "" {
		return fmt.Errorf("missing cluster config in spec of %s/%s", ns, name)
	}
	if backup.Spec.From.SecretName == "" {
		return fmt.Errorf("missing tidbSecretName config in spec of %s/%s", ns, name)
	}
	if backup.Spec.BR == nil {
		if backup.Spec.StorageSize == "" {
			return fmt.Errorf("missing StorageSize config in spec of %s/%s", ns, name)
		}
	} else {
		if backup.Spec.BR.Cluster == "" {
			return fmt.Errorf("cluster should be configured for BR in spec of %s/%s", ns, name)
		}
		if backup.Spec.Type != "" &&
			backup.Spec.Type != v1alpha1.BackupTypeFull &&
			backup.Spec.Type != v1alpha1.BackupTypeDB &&
			backup.Spec.Type != v1alpha1.BackupTypeTable {
			return fmt.Errorf("invalid backup type %s for BR in spec of %s/%s", backup.Spec.Type, ns, name)
		}
		if (backup.Spec.Type == v1alpha1.BackupTypeDB || backup.Spec.Type == v1alpha1.BackupTypeTable) && backup.Spec.BR.DB == "" {
			return fmt.Errorf("DB should be configured for BR with backup type %s in spec of %s/%s", backup.Spec.Type, ns, name)
		}
		if backup.Spec.Type == v1alpha1.BackupTypeTable && backup.Spec.BR.Table == "" {
			return fmt.Errorf("table should be configured for BR with backup type table in spec of %s/%s", ns, name)
		}
		if backup.Spec.S3 != nil {
			if backup.Spec.S3.Bucket == "" {
				return fmt.Errorf("bucket should be configured for BR in spec of %s/%s", ns, name)
			}
			if backup.Spec.S3.Endpoint != "" {
				u, err := url.Parse(backup.Spec.S3.Endpoint)
				if err != nil {
					return fmt.Errorf("invalid endpoint %s is configured for BR in spec of %s/%s", backup.Spec.S3.Endpoint, ns, name)
				}
				if u.Scheme == "" {
					return fmt.Errorf("scheme not found in endpoint %s configured for BR in spec of %s/%s", backup.Spec.S3.Endpoint, ns, name)
				}
				if u.Host == "" {
					return fmt.Errorf("host not found in endpoint %s configured for BR in spec of %s/%s", backup.Spec.S3.Endpoint, ns, name)
				}
			}
		}
	}
	return nil
}

// ValidateRestore checks whether a restore spec is valid.
func ValidateRestore(restore *v1alpha1.Restore) error {
	ns := restore.Namespace
	name := restore.Name

	if restore.Spec.To.Host == "" {
		return fmt.Errorf("missing cluster config in spec of %s/%s", ns, name)
	}
	if restore.Spec.To.SecretName == "" {
		return fmt.Errorf("missing tidbSecretName config in spec of %s/%s", ns, name)
	}
	if restore.Spec.BR == nil {
		if restore.Spec.StorageSize == "" {
			return fmt.Errorf("missing StorageSize config in spec of %s/%s", ns, name)
		}
	} else {
		if restore.Spec.BR.Cluster == "" {
			return fmt.Errorf("cluster should be configured for BR in spec of %s/%s", ns, name)
		}
		if restore.Spec.Type != "" &&
			restore.Spec.Type != v1alpha1.BackupTypeFull &&
			restore.Spec.Type != v1alpha1.BackupTypeDB &&
			restore.Spec.Type != v1alpha1.BackupTypeTable {
			return fmt.Errorf("invalid backup type %s for BR in spec of %s/%s", restore.Spec.Type, ns, name)
		}
		if (restore.Spec.Type == v1alpha1.BackupTypeDB || restore.Spec.Type == v1alpha1.BackupTypeTable) && restore.Spec.BR.DB == "" {
			return fmt.Errorf("DB should be configured for BR with restore type %s in spec of %s/%s", restore.Spec.Type, ns, name)
		}
		if restore.Spec.Type == v1alpha1.BackupTypeTable && restore.Spec.BR.Table == "" {
			return fmt.Errorf("table should be configured for BR with restore type table in spec of %s/%s", ns, name)
		}
		if restore.Spec.S3 != nil {
			if restore.Spec.S3.Bucket == "" {
				return fmt.Errorf("bucket should be configured for BR in spec of %s/%s", ns, name)
			}
			if restore.Spec.S3.Endpoint != "" {
				u, err := url.Parse(restore.Spec.S3.Endpoint)
				if err != nil {
					return fmt.Errorf("invalid endpoint %s is configured for BR in spec of %s/%s", restore.Spec.S3.Endpoint, ns, name)
				}
				if u.Scheme == "" {
					return fmt.Errorf("scheme not found in endpoint %s configured for BR in spec of %s/%s", restore.Spec.S3.Endpoint, ns, name)
				}
				if u.Host == "" {
					return fmt.Errorf("host not found in endpoint %s configured for BR in spec of %s/%s", restore.Spec.S3.Endpoint, ns, name)
				}
			}
		}
	}
	return nil
}

// GetImageTag get the image tag
func GetImageTag(image string) string {
	var tag string
	colonIdx := strings.LastIndexByte(image, ':')
	if colonIdx >= 0 {
		tag = image[colonIdx+1:]
	}
	return tag
}
