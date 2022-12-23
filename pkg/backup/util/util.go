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
	"net/url"
	"path"
	"strings"
	"unsafe"

	"github.com/Masterminds/semver"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/util/config"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	corev1 "k8s.io/api/core/v1"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
)

var (
	// the first version which allows skipping setting tikv_gc_life_time
	// https://github.com/pingcap/br/pull/553
	tikvLessThanV408, _ = semver.NewConstraint("<v4.0.8-0")
	// the first version which supports log backup
	tikvLessThanV610, _ = semver.NewConstraint("<v6.1.0-0")
)

// CheckAllKeysExistInSecret check if all keys are included in the specific secret
// return the not-exist keys join by ","
func CheckAllKeysExistInSecret(secret *corev1.Secret, keys ...string) (string, bool) {
	var notExistKeys []string

	for _, key := range keys {
		if _, exist := secret.Data[key]; !exist {
			notExistKeys = append(notExistKeys, key)
		}
	}

	return strings.Join(notExistKeys, ","), len(notExistKeys) == 0
}

// generateS3CertEnvVar generate the env info in order to access S3 compliant storage
func generateS3CertEnvVar(s3 *v1alpha1.S3StorageProvider, useKMS bool) ([]corev1.EnvVar, string, error) {
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

// generateGcsCertEnvVar generate the env info in order to access google cloud storage
func generateGcsCertEnvVar(gcs *v1alpha1.GcsStorageProvider) ([]corev1.EnvVar, string, error) {
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
	}
	if gcs.SecretName != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name: "GCS_SERVICE_ACCOUNT_JSON_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: gcs.SecretName},
					Key:                  constants.GcsCredentialsKey,
				},
			},
		})
	}
	return envVars, "", nil
}

// generateAzblobCertEnvVar generate the env info in order to access azure blob storage
func generateAzblobCertEnvVar(azblob *v1alpha1.AzblobStorageProvider, useAAD bool) ([]corev1.EnvVar, string, error) {
	if len(azblob.AccessTier) == 0 {
		azblob.AccessTier = "Cool"
	}
	envVars := []corev1.EnvVar{
		{
			Name:  "AZURE_ACCESS_TIER",
			Value: azblob.AccessTier,
		},
	}
	if azblob.SecretName != "" {
		envVars = append(envVars, []corev1.EnvVar{
			{
				Name: "AZURE_STORAGE_ACCOUNT",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: azblob.SecretName},
						Key:                  constants.AzblobAccountName,
					},
				},
			},
		}...)
		if useAAD {
			envVars = append(envVars, []corev1.EnvVar{
				{
					Name: "AZURE_CLIENT_ID",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: azblob.SecretName},
							Key:                  constants.AzblobClientID,
						},
					},
				},
				{
					Name: "AZURE_CLIENT_SECRET",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: azblob.SecretName},
							Key:                  constants.AzblobClientScrt,
						},
					},
				},
				{
					Name: "AZURE_TENANT_ID",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: azblob.SecretName},
							Key:                  constants.AzblobTenantID,
						},
					},
				},
			}...)
		} else {
			envVars = append(envVars, []corev1.EnvVar{
				{
					Name: "AZURE_STORAGE_KEY",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: azblob.SecretName},
							Key:                  constants.AzblobAccountKey,
						},
					},
				},
			}...)
		}
	}
	return envVars, "", nil
}

// GenerateStorageCertEnv generate the env info in order to access backend backup storage
func GenerateStorageCertEnv(ns string, useKMS bool, provider v1alpha1.StorageProvider, secretLister corelisterv1.SecretLister) ([]corev1.EnvVar, string, error) {
	var certEnv []corev1.EnvVar
	var reason string
	var err error
	storageType := GetStorageType(provider)

	switch storageType {
	case v1alpha1.BackupStorageTypeS3:
		s3SecretName := provider.S3.SecretName
		if s3SecretName != "" {
			secret, err := secretLister.Secrets(ns).Get(s3SecretName)
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

		certEnv, reason, err = generateS3CertEnvVar(provider.S3.DeepCopy(), useKMS)
		if err != nil {
			return certEnv, reason, err
		}
	case v1alpha1.BackupStorageTypeGcs:
		gcsSecretName := provider.Gcs.SecretName
		if gcsSecretName != "" {
			secret, err := secretLister.Secrets(ns).Get(gcsSecretName)
			if err != nil {
				err := fmt.Errorf("get gcs secret %s/%s failed, err: %v", ns, gcsSecretName, err)
				return certEnv, "GetGcsSecretFailed", err
			}

			keyStr, exist := CheckAllKeysExistInSecret(secret, constants.GcsCredentialsKey)
			if !exist {
				err := fmt.Errorf("the gcs secret %s/%s missing some keys %s", ns, gcsSecretName, keyStr)
				return certEnv, "gcsKeyNotExist", err
			}
		}

		certEnv, reason, err = generateGcsCertEnvVar(provider.Gcs)

		if err != nil {
			return certEnv, reason, err
		}
	case v1alpha1.BackupStorageTypeAzblob:
		useAAD := true
		azblobSecretName := provider.Azblob.SecretName
		if azblobSecretName != "" {
			secret, err := secretLister.Secrets(ns).Get(azblobSecretName)
			if err != nil {
				err := fmt.Errorf("get azblob secret %s/%s failed, err: %v", ns, azblobSecretName, err)
				return certEnv, "GetAzblobSecretFailed", err
			}

			keyStrAAD, exist := CheckAllKeysExistInSecret(secret, constants.AzblobAccountName, constants.AzblobClientID, constants.AzblobClientScrt, constants.AzblobTenantID)
			if !exist {
				keyStrShared, exist := CheckAllKeysExistInSecret(secret, constants.AzblobAccountName, constants.AzblobAccountKey)
				if !exist {
					err := fmt.Errorf("the azblob secret %s/%s missing some keys for AAD %s or shared %s", ns, azblobSecretName, keyStrAAD, keyStrShared)
					return certEnv, "azblobKeyNotExist", err
				}
				useAAD = false
			}
		}

		certEnv, reason, err = generateAzblobCertEnvVar(provider.Azblob, useAAD)

		if err != nil {
			return certEnv, reason, err
		}
	case v1alpha1.BackupStorageTypeLocal:
		return []corev1.EnvVar{}, "", nil
	default:
		err := fmt.Errorf("unsupported storage type %s", storageType)
		return certEnv, "UnsupportedStorageType", err
	}
	return certEnv, reason, nil
}

func getPasswordKey(useKMS bool) string {
	if useKMS {
		return fmt.Sprintf("%s_%s_%s", constants.KMSSecretPrefix, constants.BackupManagerEnvVarPrefix, strings.ToUpper(constants.TidbPasswordKey))
	}

	return fmt.Sprintf("%s_%s", constants.BackupManagerEnvVarPrefix, strings.ToUpper(constants.TidbPasswordKey))
}

// GenerateTidbPasswordEnv generate the password EnvVar
func GenerateTidbPasswordEnv(ns, tcName, tidbSecretName string, useKMS bool, secretLister corelisterv1.SecretLister) ([]corev1.EnvVar, string, error) {
	var certEnv []corev1.EnvVar
	var passwordKey string
	secret, err := secretLister.Secrets(ns).Get(tidbSecretName)
	if err != nil {
		err = fmt.Errorf("backup %s/%s get tidb secret %s failed, err: %v", ns, tcName, tidbSecretName, err)
		return certEnv, "GetTidbSecretFailed", err
	}

	keyStr, exist := CheckAllKeysExistInSecret(secret, constants.TidbPasswordKey)
	if !exist {
		err = fmt.Errorf("backup %s/%s, tidb secret %s missing password key %s", ns, tcName, tidbSecretName, keyStr)
		return certEnv, "KeyNotExist", err
	}

	passwordKey = getPasswordKey(useKMS)

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

// GetBackupPrefixName return the prefix for remote storage
func GetBackupPrefixName(backup *v1alpha1.Backup) (string, string, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()
	storageType := GetStorageType(backup.Spec.StorageProvider)
	var prefix string

	switch storageType {
	case v1alpha1.BackupStorageTypeS3:
		prefix = backup.Spec.S3.Prefix
	case v1alpha1.BackupStorageTypeGcs:
		prefix = backup.Spec.Gcs.Prefix
	default:
		return prefix, "UnsupportedStorageType", fmt.Errorf("backup %s/%s unsupported storage type %s", ns, name, storageType)
	}
	return prefix, "", nil
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
	if provider.Azblob != nil {
		return v1alpha1.BackupStorageTypeAzblob
	}
	if provider.Local != nil {
		return v1alpha1.BackupStorageTypeLocal
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

func validateAccessConfig(config *v1alpha1.TiDBAccessConfig) string {
	if config == nil {
		return "missing cluster config in spec of %s/%s"
	} else {
		if config.Host == "" {
			return "missing cluster config in spec of %s/%s"
		}

		if config.SecretName == "" {
			return "missing tidbSecretName config in spec of %s/%s"
		}
	}
	return ""
}

// ValidateBackup validates backup sepc
func ValidateBackup(backup *v1alpha1.Backup, tikvImage string) error {
	ns := backup.Namespace
	name := backup.Name

	if backup.Spec.BR == nil {
		if reason := validateAccessConfig(backup.Spec.From); reason != "" {
			return fmt.Errorf(reason, ns, name)
		}
		if backup.Spec.StorageSize == "" {
			return fmt.Errorf("missing StorageSize config in spec of %s/%s", ns, name)
		}
	} else {
		if !canSkipSetGCLifeTime(tikvImage) {
			if reason := validateAccessConfig(backup.Spec.From); reason != "" {
				return fmt.Errorf(reason, ns, name)
			}
		}
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

		// validate storage providers
		if backup.Spec.S3 != nil {
			if err := validateS3(ns, name, backup.Spec.S3); err != nil {
				return err
			}
		} else if backup.Spec.Gcs != nil {
			if err := validateGcs(ns, name, backup.Spec.Gcs); err != nil {
				return err
			}
		} else if backup.Spec.Local != nil {
			if err := validateLocal(ns, name, backup.Spec.Local); err != nil {
				return err
			}
		}

		// validate log backup
		if backup.Spec.Mode == v1alpha1.BackupModeLog {
			if !isLogBackSupport(tikvImage) {
				return fmt.Errorf("tikv %s doesn't support log backup in spec of %s/%s, the first version is v6.1.0", tikvImage, ns, name)
			}
			var err error
			_, err = config.ParseTSString(backup.Spec.CommitTs)
			if err != nil {
				return err
			}
			if v1alpha1.ParseLogBackupSubcommand(backup) == v1alpha1.LogTruncateCommand && backup.Spec.LogTruncateUntil == "" {
				return fmt.Errorf("log backup %s/%s truncate command missing 'logTruncateUntil'", ns, name)

			}
			_, err = config.ParseTSString(backup.Spec.LogTruncateUntil)
			if err != nil {
				return err
			}
		}

	}
	return nil
}

// ValidateRestore checks whether a restore spec is valid.
func ValidateRestore(restore *v1alpha1.Restore, tikvImage string) error {
	ns := restore.Namespace
	name := restore.Name

	if restore.Spec.BR == nil {
		if reason := validateAccessConfig(restore.Spec.To); reason != "" {
			return fmt.Errorf(reason, ns, name)
		}
		if restore.Spec.StorageSize == "" {
			return fmt.Errorf("missing StorageSize config in spec of %s/%s", ns, name)
		}
	} else {
		if !canSkipSetGCLifeTime(tikvImage) {
			if reason := validateAccessConfig(restore.Spec.To); reason != "" {
				return fmt.Errorf(reason, ns, name)
			}
		}
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

		// validate storage providers
		if restore.Spec.S3 != nil {
			if err := validateS3(ns, name, restore.Spec.S3); err != nil {
				return err
			}
		} else if restore.Spec.Gcs != nil {
			if err := validateGcs(ns, name, restore.Spec.Gcs); err != nil {
				return err
			}
		} else if restore.Spec.Local != nil {
			if err := validateLocal(ns, name, restore.Spec.Local); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateS3(ns, name string, s3 *v1alpha1.S3StorageProvider) error {
	configuredForBR := fmt.Sprintf("configured for BR in spec of %s/%s", ns, name)
	if s3.Bucket == "" {
		return fmt.Errorf("bucket should be %s", configuredForBR)
	}

	if s3.Endpoint != "" {
		u, err := url.Parse(s3.Endpoint)
		if err != nil {
			return fmt.Errorf("invalid endpoint %s is %s", s3.Endpoint, configuredForBR)
		}
		if u.Scheme == "" {
			return fmt.Errorf("scheme not found in endpoint %s %s", s3.Endpoint, configuredForBR)
		}
		if u.Host == "" {
			return fmt.Errorf("host not found in endpoint %s %s", s3.Endpoint, configuredForBR)
		}
	}
	return nil
}

func validateGcs(ns, name string, gcs *v1alpha1.GcsStorageProvider) error {
	configuredForBR := fmt.Sprintf("configured for BR in spec of %s/%s", ns, name)
	if gcs.ProjectId == "" {
		return fmt.Errorf("projectId should be %s", configuredForBR)
	}
	if gcs.Bucket == "" {
		return fmt.Errorf("bucket should be %s", configuredForBR)
	}
	return nil
}

func validateLocal(ns, name string, local *v1alpha1.LocalStorageProvider) error {
	configuredForBR := fmt.Sprintf("configured for BR in spec of %s/%s", ns, name)
	if local.VolumeMount.Name != local.Volume.Name {
		return fmt.Errorf("Spec.Local.Volume.Name != Spec.Local.VolumeMount.Name is %s", configuredForBR)
	}
	if local.VolumeMount.MountPath == "" {
		return fmt.Errorf("empty Spec.Local.VolumeMount.MountPath is %s", configuredForBR)
	}
	if strings.Contains(local.VolumeMount.MountPath, ":") {
		return fmt.Errorf("Spec.Local.VolumeMount.MountPath cannot contain ':' %s", configuredForBR)
	}
	return nil
}

// ParseImage returns the image name and the tag from the input image string
func ParseImage(image string) (string, string) {
	var name, tag string
	colonIdx := strings.LastIndexByte(image, ':')
	if colonIdx >= 0 {
		name = image[:colonIdx]
		tag = image[colonIdx+1:]
	} else {
		name = image
	}
	return name, tag
}

// canSkipSetGCLifeTime returns if setting tikv_gc_life_time can be skipped based on the TiKV version
func canSkipSetGCLifeTime(image string) bool {
	_, version := ParseImage(image)
	v, err := semver.NewVersion(version)
	if err != nil {
		klog.Errorf("Parse version %s failure, error: %v", version, err)
		return true
	}
	if tikvLessThanV408.Check(v) {
		return false
	}
	return true
}

func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}

// isLogBackSupport returns whether tikv supports log backup
func isLogBackSupport(tikvImage string) bool {
	_, version := ParseImage(tikvImage)
	v, err := semver.NewVersion(version)
	if err != nil {
		klog.Errorf("Parse version %s failure, error: %v", version, err)
		return true
	}
	if tikvLessThanV610.Check(v) {
		return false
	}
	return true
}

// GetStorageRestorePath generate the path of a specific storage from Restore
func GetStoragePath(privoder v1alpha1.StorageProvider) (string, error) {
	var url, bucket, prefix string
	st := GetStorageType(privoder)
	switch st {
	case v1alpha1.BackupStorageTypeS3:
		prefix = privoder.S3.Prefix
		bucket = privoder.S3.Bucket
		url = fmt.Sprintf("s3://%s", path.Join(bucket, prefix))
		return url, nil
	case v1alpha1.BackupStorageTypeGcs:
		prefix = privoder.Gcs.Prefix
		bucket = privoder.Gcs.Bucket
		url = fmt.Sprintf("gcs://%s/", path.Join(bucket, prefix))
		return url, nil
	case v1alpha1.BackupStorageTypeAzblob:
		prefix = privoder.Azblob.Prefix
		bucket = privoder.Azblob.Container
		url = fmt.Sprintf("azure://%s/", path.Join(bucket, prefix))
		return url, nil
	case v1alpha1.BackupStorageTypeLocal:
		prefix = privoder.Local.Prefix
		mountPath := privoder.Local.VolumeMount.MountPath
		url = fmt.Sprintf("local://%s", path.Join(mountPath, prefix))
		return url, nil
	default:
		return "", fmt.Errorf("storage %s not supported yet", st)
	}
}

// GetOptions gets the rclone options
func GetOptions(provider v1alpha1.StorageProvider) []string {
	st := GetStorageType(provider)
	switch st {
	case v1alpha1.BackupStorageTypeS3:
		return provider.S3.Options
	default:
		return nil
	}
}
