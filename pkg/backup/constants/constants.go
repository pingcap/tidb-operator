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
	// TimeFormat is the time format for generate backup CR name
	TimeFormat = "2006-01-02t15-04-05"

	// DefaultServiceAccountName is the default name of the ServiceAccount to use to run backup and restore's job pod.
	DefaultServiceAccountName = "tidb-backup-manager"

	// DefaultTidbPort is the default tidb cluster port for connecting
	DefaultTidbPort = 4000

	// DefaultTidbUser is the default tidb user for login tidb cluster
	DefaultTidbUser = "root"

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

	// BackupManagerEnvVarPrefix represents the environment variable used for tidb-backup-manager must include this prefix
	BackupManagerEnvVarPrefix = "BACKUP_MANAGER"

	// BR certificate storage path
	BRCertPath = "/var/lib/br-tls"

	// ServiceAccountCAPath is where is CABundle of serviceaccount locates
	ServiceAccountCAPath = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"

	// KMS secret env prefix
	KMSSecretPrefix = "KMS_ENCRYPTED"
)
