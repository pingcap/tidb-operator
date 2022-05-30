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
)
