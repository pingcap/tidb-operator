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

package constants

const (
	// DefaultServiceAccountName is the default name of the ServiceAccount to use to run backup and restore's job pod.
	DefaultServiceAccountName = "tidb-backup-manager"

	// BackupRootPath is the root path to backup data
	BackupRootPath = "/backup"

	// DefaultStorageSize is the default pvc request storage size for backup and restore
	DefaultStorageSize = "100Gi"

	// DefaultBackoffLimit specifies the number of retries before marking this job failed.
	DefaultBackoffLimit = 6

	// TidbPasswordKey represents the password key in tidb secret
	TidbPasswordKey = "password"

	// S3AccessKey represents the S3 compatible access key id in related secret
	S3AccessKey = "access_key"

	// S3SecretKey represents the S3 compatible secret access key in related secret
	S3SecretKey = "secret_key"

	// GcsCredentialsKey represents the gcs service account credentials json key in related secret
	GcsCredentialsKey = "credentials"

	// AzblobAccountName represents the Azure Storage Account using shared key credential in related secret
	AzblobAccountName = "AZURE_STORAGE_ACCOUNT"

	// AzblobAccountKey represents the Azure Storage Access Key using shared key credential in related secret
	AzblobAccountKey = "AZURE_STORAGE_KEY"

	// AzblobClientID represents the Azure Application (client) ID using AAD credtentials in related secret
	AzblobClientID = "AZURE_CLIENT_ID"

	// AzblobClientScrt represents the Azure Application (client) secret using AAD credtentials in related secret
	AzblobClientScrt = "AZURE_CLIENT_SECRET"

	// AzblobTenantID represents the Azure Directory (tenant) ID for the application using AAD credtentials in related secret
	AzblobTenantID = "AZURE_TENANT_ID"

	// BackupManagerEnvVarPrefix represents the environment variable used for tidb-backup-manager must include this prefix
	BackupManagerEnvVarPrefix = "BACKUP_MANAGER"

	// BR certificate storage path
	BRCertPath = "/var/lib/br-tls"

	// ServiceAccountCAPath is where is CABundle of serviceaccount locates
	ServiceAccountCAPath = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"

	// KMS secret env prefix
	KMSSecretPrefix = "KMS_ENCRYPTED"

	// RootKey represents the username in tidb secret
	TidbRootKey = "root"

	// the volumes provisioned by CSI driver on AWSElasticBlockStore
	EbsCSIDriver = "ebs.csi.aws.com"

	// the volumes provisioned by CSI driver on GCEPersistentDisk
	PdCSIDriver = "pd.csi.storage.gke.io"

	// the mount path for TiKV data volume
	TiKVDataVolumeMountPath = "/var/lib/tikv"

	// the configuration for TiKV data volume as type for snapshotter
	TiKVDataVolumeConfType = "storage.data-dir"

	// the metadata for cloud provider to take snapshot
	EnvCloudSnapMeta = "CLOUD_SNAPSHOT_METADATA"

	// the annotation for store temporary volumeID
	AnnTemporaryVolumeID = "temporary/volume-id"

	// These annotations are taken from the Kubernetes persistent volume/persistent volume claim controller.
	// They cannot be directly importing because they are part of the kubernetes/kubernetes package, and importing that package is unsupported.
	// Their values are well-known and slow changing. They're duplicated here as constants to provide compile-time checking.
	// Originals can be found in kubernetes/kubernetes/pkg/controller/volume/persistentvolume/util/util.go.
	KubeAnnBindCompleted          = "pv.kubernetes.io/bind-completed"
	KubeAnnBoundByController      = "pv.kubernetes.io/bound-by-controller"
	KubeAnnDynamicallyProvisioned = "pv.kubernetes.io/provisioned-by"

	LocalTmp           = "/tmp"
	ClusterBackupMeta  = "clustermeta"
	ClusterRestoreMeta = "restoremeta"
)
