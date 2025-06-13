// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	brv1alpha1 "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	corev1alpha1 "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controllers/br/manager/constants"
)

const (
	ReasonUnsupportedStorageType = "UnsupportedStorageType"
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
func generateS3CertEnvVar(s3 *brv1alpha1.S3StorageProvider, useKMS bool) ([]corev1.EnvVar, string, error) {
	var envVars []corev1.EnvVar

	switch s3.Provider {
	case brv1alpha1.S3StorageProviderTypeCeph:
		if !strings.Contains(s3.Endpoint, "://") {
			// convert xxx.xxx.xxx.xxx:port to http://xxx.xxx.xxx.xxx:port
			// the endpoint must start with http://
			s3.Endpoint = fmt.Sprintf("http://%s", s3.Endpoint)
			break
		}
		if !strings.HasPrefix(s3.Endpoint, "http://") {
			return envVars, "InvalidS3Endpoint", fmt.Errorf("ceph endpoint URI %s must start with http://", s3.Endpoint)
		}
	case brv1alpha1.S3StorageProviderTypeAWS:
		// TODO: Check the storage class, if it is not a legal storage class, use the default storage class instead
		if s3.StorageClass == "" {
			// The optional storage class reference https://rclone.org/s3
			s3.StorageClass = "STANDARD_IA"
		}
		if s3.Acl == "" {
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
func generateGcsCertEnvVar(gcs *brv1alpha1.GcsStorageProvider) ([]corev1.EnvVar, string, error) {
	if gcs.ProjectId == "" {
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
		// NOTE: it may not be used actually, just keep the compatibility of tidb operator v1
		envVars = append(envVars,
			corev1.EnvVar{
				Name: "GCS_SERVICE_ACCOUNT_JSON_KEY",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: gcs.SecretName},
						Key:                  constants.GcsCredentialsKey,
					},
				},
			},
			// mount credentials to /gcs and specify the default application credentials env
			corev1.EnvVar{
				Name:  "GOOGLE_APPLICATION_CREDENTIALS",
				Value: filepath.Join(constants.GcsCredentialsMountPath, constants.GcsCredentialsKey),
			})
	}
	return envVars, "", nil
}

// generateAzblobCertEnvVar generate the env info in order to access azure blob storage
func generateAzblobCertEnvVar(azblob *brv1alpha1.AzblobStorageProvider, secret *corev1.Secret, useSasToken bool) ([]corev1.EnvVar, string, error) {
	if azblob.AccessTier == "" {
		azblob.AccessTier = "Cool"
	}
	envVars := []corev1.EnvVar{
		{
			Name:  "AZURE_ACCESS_TIER",
			Value: azblob.AccessTier,
		},
		{
			Name:  "AZURE_STORAGE_ACCOUNT",
			Value: azblob.StorageAccount,
		},
	}
	if useSasToken {
		return envVars, "", nil
	}
	_, exist := CheckAllKeysExistInSecret(secret, constants.AzblobClientID, constants.AzblobClientScrt, constants.AzblobTenantID)
	if exist { // using AAD auth
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
		return envVars, "", nil
	}
	_, exist = CheckAllKeysExistInSecret(secret, constants.AzblobAccountKey)
	if exist { // use access key auth
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
		return envVars, "", nil
	}
	return nil, "azblobKeyOrAADMissing", fmt.Errorf("secret %s/%s missing some keys", secret.Namespace, secret.Name)
}

// GenerateStorageCertEnv generate the env info in order to access backend backup storage
// nolint: gocyclo
func GenerateStorageCertEnv(ctx context.Context, ns string, useKMS bool, provider brv1alpha1.StorageProvider, cli client.Client) ([]corev1.EnvVar, string, error) {
	var certEnv []corev1.EnvVar
	var reason string
	var err error
	storageType := GetStorageType(provider)

	switch storageType {
	case brv1alpha1.BackupStorageTypeS3:
		s3SecretName := provider.S3.SecretName
		if s3SecretName != "" {
			secret := &corev1.Secret{}
			if err := cli.Get(ctx, client.ObjectKey{Namespace: ns, Name: s3SecretName}, secret); err != nil {
				err := fmt.Errorf("get s3 secret %s/%s failed, err: %w", ns, s3SecretName, err)
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
	case brv1alpha1.BackupStorageTypeGcs:
		gcsSecretName := provider.Gcs.SecretName
		if gcsSecretName != "" {
			secret := &corev1.Secret{}
			err := cli.Get(ctx, client.ObjectKey{Namespace: ns, Name: gcsSecretName}, secret)
			if err != nil {
				err := fmt.Errorf("get gcs secret %s/%s failed, err: %w", ns, gcsSecretName, err)
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
	case brv1alpha1.BackupStorageTypeAzblob:
		azblobSecretName := provider.Azblob.SecretName
		var secret *corev1.Secret
		if azblobSecretName != "" {
			secret = &corev1.Secret{}
			if err := cli.Get(ctx, client.ObjectKey{Namespace: ns, Name: azblobSecretName}, secret); err != nil {
				err := fmt.Errorf("get azblob secret %s/%s failed, err: %w", ns, azblobSecretName, err)
				return certEnv, "GetAzblobSecretFailed", err
			}
		}
		if provider.Azblob.StorageAccount == "" { // try to get storageAccount from secret
			account := string(secret.Data[constants.AzblobAccountName])
			if account == "" {
				err := fmt.Errorf("secret %s/%s missing some keys, storage account unspecified: %v", ns, azblobSecretName, secret.Data)
				return certEnv, "azblobAccountNotExist", err
			}
			provider.Azblob.StorageAccount = account
		}
		useSasToken := provider.Azblob.SasToken != ""
		certEnv, reason, err = generateAzblobCertEnvVar(provider.Azblob, secret, useSasToken)
		if err != nil {
			return certEnv, reason, err
		}
	case brv1alpha1.BackupStorageTypeLocal:
		return []corev1.EnvVar{}, "", nil
	default:
		err := fmt.Errorf("unsupported storage type %s", storageType)
		return certEnv, ReasonUnsupportedStorageType, err
	}
	return certEnv, reason, nil
}

func GenerateStorageVolumesAndMounts(ctx context.Context, ns string, provider brv1alpha1.StorageProvider) ([]corev1.Volume, []corev1.VolumeMount, error) {
	var vols []corev1.Volume
	var mounts []corev1.VolumeMount
	storageType := GetStorageType(provider)

	switch storageType {
	case brv1alpha1.BackupStorageTypeS3:
	case brv1alpha1.BackupStorageTypeGcs:
		gcsSecretName := provider.Gcs.SecretName
		if gcsSecretName == "" {
			return nil, nil, nil
		}

		volName := constants.GcsCredentialsKey

		vols = append(vols, corev1.Volume{
			Name: volName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: gcsSecretName,
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      volName,
			MountPath: constants.GcsCredentialsMountPath,
		})

		return vols, mounts, nil
	case brv1alpha1.BackupStorageTypeAzblob:
	case brv1alpha1.BackupStorageTypeLocal:
	default:
	}
	return vols, mounts, nil
}

func getPasswordKey(useKMS bool) string {
	if useKMS {
		return fmt.Sprintf("%s_%s_%s", constants.KMSSecretPrefix, constants.BackupManagerEnvVarPrefix, strings.ToUpper(constants.TidbPasswordKey))
	}

	return fmt.Sprintf("%s_%s", constants.BackupManagerEnvVarPrefix, strings.ToUpper(constants.TidbPasswordKey))
}

// GenerateTidbPasswordEnv generate the password EnvVar
func GenerateTidbPasswordEnv(ctx context.Context, ns, tcName, tidbSecretName string, useKMS bool, cli client.Client) ([]corev1.EnvVar, string, error) {
	var certEnv []corev1.EnvVar
	var passwordKey string
	secret := &corev1.Secret{}
	err := cli.Get(ctx, client.ObjectKey{Namespace: ns, Name: tidbSecretName}, secret)
	if err != nil {
		err = fmt.Errorf("backup %s/%s get tidb secret %s failed, err: %w", ns, tcName, tidbSecretName, err)
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
func GetBackupBucketName(backup *brv1alpha1.Backup) (bucket, reason string, err error) {
	ns := backup.GetNamespace()
	name := backup.GetName()
	storageType := GetStorageType(backup.Spec.StorageProvider)

	switch storageType {
	case brv1alpha1.BackupStorageTypeS3:
		bucket = backup.Spec.S3.Bucket
	case brv1alpha1.BackupStorageTypeGcs:
		bucket = backup.Spec.Gcs.Bucket
	default:
		return bucket, ReasonUnsupportedStorageType, fmt.Errorf("backup %s/%s unsupported storage type %s", ns, name, storageType)
	}
	return bucket, "", nil
}

// GetBackupPrefixName return the prefix for remote storage
func GetBackupPrefixName(backup *brv1alpha1.Backup) (prefix, reason string, err error) {
	ns := backup.GetNamespace()
	name := backup.GetName()
	storageType := GetStorageType(backup.Spec.StorageProvider)

	switch storageType {
	case brv1alpha1.BackupStorageTypeS3:
		prefix = backup.Spec.S3.Prefix
	case brv1alpha1.BackupStorageTypeGcs:
		prefix = backup.Spec.Gcs.Prefix
	default:
		return prefix, ReasonUnsupportedStorageType, fmt.Errorf("backup %s/%s unsupported storage type %s", ns, name, storageType)
	}
	return prefix, "", nil
}

// GetStorageType return the backup storage type according to the specified StorageProvider
func GetStorageType(provider brv1alpha1.StorageProvider) brv1alpha1.BackupStorageType {
	// If there are multiple storages in the StorageProvider, the first one found is returned in the following order
	if provider.S3 != nil {
		return brv1alpha1.BackupStorageTypeS3
	}
	if provider.Gcs != nil {
		return brv1alpha1.BackupStorageTypeGcs
	}
	if provider.Azblob != nil {
		return brv1alpha1.BackupStorageTypeAzblob
	}
	if provider.Local != nil {
		return brv1alpha1.BackupStorageTypeLocal
	}
	return brv1alpha1.BackupStorageTypeUnknown
}

// GetBackupDataPath return the full path of backup data
func GetBackupDataPath(provider brv1alpha1.StorageProvider) (backupPath, reason string, err error) {
	storageType := GetStorageType(provider)

	switch storageType {
	case brv1alpha1.BackupStorageTypeS3:
		backupPath = provider.S3.Path
	case brv1alpha1.BackupStorageTypeGcs:
		backupPath = provider.Gcs.Path
	default:
		return backupPath, ReasonUnsupportedStorageType, fmt.Errorf("unsupported storage type %s", storageType)
	}
	protocolPrefix := fmt.Sprintf("%s://", string(storageType))
	if strings.HasPrefix(backupPath, protocolPrefix) {
		// if the full path of backup data start with "<storageType>://", return directly.
		return backupPath, "", nil
	}
	return fmt.Sprintf("%s://%s", string(storageType), backupPath), "", nil
}

// ValidateBackup validates backup sepc
// nolint:gocyclo
func ValidateBackup(backup *brv1alpha1.Backup, tikvVersion string, cluster *corev1alpha1.Cluster) error {
	ns := backup.Namespace
	name := backup.Name

	if backup.Spec.BR == nil {
		return fmt.Errorf(".spec.br is required in spec of %s/%s", ns, name)
	} else {
		if backup.Spec.BR.Cluster == "" {
			return fmt.Errorf("cluster should be configured for BR in spec of %s/%s", ns, name)
		}

		if backup.Spec.Type != "" &&
			backup.Spec.Type != brv1alpha1.BackupTypeFull &&
			backup.Spec.Type != brv1alpha1.BackupTypeDB &&
			backup.Spec.Type != brv1alpha1.BackupTypeTable {
			return fmt.Errorf("invalid backup type %s for BR in spec of %s/%s", backup.Spec.Type, ns, name)
		}

		if (backup.Spec.Type == brv1alpha1.BackupTypeDB || backup.Spec.Type == brv1alpha1.BackupTypeTable) && backup.Spec.BR.DB == "" {
			return fmt.Errorf("DB should be configured for BR with backup type %s in spec of %s/%s", backup.Spec.Type, ns, name)
		}

		if backup.Spec.Type == brv1alpha1.BackupTypeTable && backup.Spec.BR.Table == "" {
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
		if backup.Spec.Mode == brv1alpha1.BackupModeLog {
			var err error
			_, err = brv1alpha1.ParseTSString(backup.Spec.CommitTs)
			if err != nil {
				return err
			}
			if brv1alpha1.ParseLogBackupSubcommand(backup) == brv1alpha1.LogTruncateCommand && backup.Spec.LogTruncateUntil == "" {
				return fmt.Errorf("log backup %s/%s truncate command missing 'logTruncateUntil'", ns, name)
			}
			_, err = brv1alpha1.ParseTSString(backup.Spec.LogTruncateUntil)
			if err != nil {
				return err
			}
		}

		if backup.Spec.BackoffRetryPolicy.MinRetryDuration != "" {
			_, err := time.ParseDuration(backup.Spec.BackoffRetryPolicy.MinRetryDuration)
			if err != nil {
				return fmt.Errorf("fail to parse minRetryDuration %s of backup %s/%s, %w", backup.Spec.BackoffRetryPolicy.MinRetryDuration, backup.Namespace, backup.Name, err)
			}
		}
		if backup.Spec.BackoffRetryPolicy.RetryTimeout != "" {
			_, err := time.ParseDuration(backup.Spec.BackoffRetryPolicy.RetryTimeout)
			if err != nil {
				return fmt.Errorf("fail to parse retryTimeout %s of backup %s/%s, %w", backup.Spec.BackoffRetryPolicy.RetryTimeout, backup.Namespace, backup.Name, err)
			}
		}
	}
	return nil
}

// ValidateRestore checks whether a restore spec is valid.
// nolint:gocyclo
func ValidateRestore(restore *brv1alpha1.Restore, tikvVersion string) error {
	ns := restore.Namespace
	name := restore.Name

	if restore.Spec.BR == nil {
		return fmt.Errorf(".spec.br is required in spec of %s/%s", ns, name)
	} else {
		if restore.Spec.BR.Cluster == "" {
			return fmt.Errorf("cluster should be configured for BR in spec of %s/%s", ns, name)
		}

		if restore.Spec.Type != "" &&
			restore.Spec.Type != brv1alpha1.BackupTypeFull &&
			restore.Spec.Type != brv1alpha1.BackupTypeDB &&
			restore.Spec.Type != brv1alpha1.BackupTypeTable {
			return fmt.Errorf("invalid backup type %s for BR in spec of %s/%s", restore.Spec.Type, ns, name)
		}

		if (restore.Spec.Type == brv1alpha1.BackupTypeDB || restore.Spec.Type == brv1alpha1.BackupTypeTable) && restore.Spec.BR.DB == "" {
			return fmt.Errorf("DB should be configured for BR with restore type %s in spec of %s/%s", restore.Spec.Type, ns, name)
		}

		if restore.Spec.Mode == brv1alpha1.RestoreModePiTR {
			_, err := GetStoragePath(restore.Spec.PitrFullBackupStorageProvider)
			// err is nil when there is a valid storage provider
			if err == nil && restore.Spec.LogRestoreStartTs != "" {
				return fmt.Errorf("pitrFullBackupStorageProvider and logRestoreStartTs option can not co-exists in pitr mode")
			}

			if err != nil && restore.Spec.LogRestoreStartTs == "" {
				return fmt.Errorf("either pitrFullBackupStorageProvider or logRestoreStartTs option needs to be passed in pitr mode")
			}
		}

		if restore.Spec.Type == brv1alpha1.BackupTypeTable && restore.Spec.BR.Table == "" {
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

func validateS3(ns, name string, s3 *brv1alpha1.S3StorageProvider) error {
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

func validateGcs(ns, name string, gcs *brv1alpha1.GcsStorageProvider) error {
	configuredForBR := fmt.Sprintf("configured for BR in spec of %s/%s", ns, name)
	if gcs.ProjectId == "" {
		return fmt.Errorf("projectId should be %s", configuredForBR)
	}
	if gcs.Bucket == "" {
		return fmt.Errorf("bucket should be %s", configuredForBR)
	}
	return nil
}

func validateLocal(ns, name string, local *brv1alpha1.LocalStorageProvider) error {
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
func ParseImage(image string) (name, tag string) {
	colonIdx := strings.LastIndexByte(image, ':')
	if colonIdx >= 0 {
		name = image[:colonIdx]
		tag = image[colonIdx+1:]
	} else {
		name = image
	}
	return name, tag
}

// GetStoragePath generate the path of a specific storage from Restore
func GetStoragePath(privoder brv1alpha1.StorageProvider) (string, error) {
	var urlStr, bucket, prefix string
	st := GetStorageType(privoder)
	switch st {
	case brv1alpha1.BackupStorageTypeS3:
		prefix = privoder.S3.Prefix
		bucket = privoder.S3.Bucket
		urlStr = fmt.Sprintf("s3://%s", path.Join(bucket, prefix))
		return urlStr, nil
	case brv1alpha1.BackupStorageTypeGcs:
		prefix = privoder.Gcs.Prefix
		bucket = privoder.Gcs.Bucket
		urlStr = fmt.Sprintf("gcs://%s/", path.Join(bucket, prefix))
		return urlStr, nil
	case brv1alpha1.BackupStorageTypeAzblob:
		prefix = privoder.Azblob.Prefix
		bucket = privoder.Azblob.Container
		urlStr = fmt.Sprintf("azure://%s/", path.Join(bucket, prefix))
		return urlStr, nil
	case brv1alpha1.BackupStorageTypeLocal:
		prefix = privoder.Local.Prefix
		mountPath := privoder.Local.VolumeMount.MountPath
		urlStr = fmt.Sprintf("local://%s", path.Join(mountPath, prefix))
		return urlStr, nil
	default:
		return "", fmt.Errorf("storage %s not supported yet", st)
	}
}

// GetOptions gets the rclone options
func GetOptions(provider brv1alpha1.StorageProvider) []string {
	st := GetStorageType(provider)
	switch st {
	case brv1alpha1.BackupStorageTypeS3:
		return provider.S3.Options
	default:
		return nil
	}
}

// UpdateBRProgress updates progress for backup/restore.
func UpdateBRProgress(progresses []brv1alpha1.Progress, step *string, progress *int, updateTime *metav1.Time) ([]brv1alpha1.Progress, bool) {
	var oldProgress *brv1alpha1.Progress
	for i, p := range progresses {
		if p.Step == *step {
			oldProgress = &progresses[i]
			break
		}
	}

	makeSureLastProgressOver := func() {
		size := len(progresses)
		if size == 0 || progresses[size-1].Progress >= 100 {
			return
		}
		progresses[size-1].Progress = 100
		progresses[size-1].LastTransitionTime = metav1.Time{Time: time.Now()}
	}

	// no such progress, will new
	if oldProgress == nil {
		makeSureLastProgressOver()
		progresses = append(progresses, brv1alpha1.Progress{
			Step:               *step,
			Progress:           *progress,
			LastTransitionTime: *updateTime,
		})
		return progresses, true
	}

	isUpdate := false
	if oldProgress.Progress < *progress {
		oldProgress.Progress = *progress
		isUpdate = true
	}

	if oldProgress.LastTransitionTime != *updateTime {
		oldProgress.LastTransitionTime = *updateTime
		isUpdate = true
	}

	return progresses, isUpdate
}
