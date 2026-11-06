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
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"

	br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
)

// StorageIdentity fingerprints the effective destination without recording credentials.
func StorageIdentity(backup *br.Backup) (string, error) {
	if err := ValidateBROptions(backup.Spec.BR); err != nil {
		return "", err
	}
	provider := backup.Spec.StorageProvider
	var identity any
	switch GetStorageType(provider) {
	case br.BackupStorageTypeS3:
		s := provider.S3
		identity = struct {
			Kind, Provider, Endpoint, Region, Bucket, Prefix string
			Options                                          []string
		}{
			"s3", string(s.Provider), s.Endpoint, s.Region, s.Bucket, s.Prefix, s.Options,
		}
	case br.BackupStorageTypeGcs:
		g := provider.Gcs
		universe := os.Getenv("GOOGLE_CLOUD_UNIVERSE_DOMAIN")
		if universe == "googleapis.com" {
			universe = ""
		}
		identity = struct {
			Kind, Bucket, Prefix string
			Endpoint             string `json:",omitempty"`
			UniverseDomain       string `json:",omitempty"`
		}{"gcs", g.Bucket, g.Prefix, os.Getenv("STORAGE_EMULATOR_HOST"), universe}
	case br.BackupStorageTypeAzblob:
		a := provider.Azblob
		account := a.StorageAccount
		environmentAccount := os.Getenv("AZURE_STORAGE_ACCOUNT")
		scopedAccount := os.Getenv(fmt.Sprintf("AZURE_STORAGE_ACCOUNT_%x", sha256.Sum256([]byte(a.SecretName))))
		if scopedAccount != "" {
			if account != "" && account != scopedAccount {
				return "", fmt.Errorf("azure storage account configuration is inconsistent")
			}
			account = scopedAccount
		}
		if environmentAccount != "" {
			if account != "" && account != environmentAccount {
				return "", fmt.Errorf("azure storage account environment does not match its configuration")
			}
			account = environmentAccount
		}
		if account == "" {
			return "", fmt.Errorf("azure storage account is required to identify the backup destination")
		}
		identity = struct{ Kind, Account, Container, Prefix string }{"azure", account, a.Container, a.Prefix}
	case br.BackupStorageTypeLocal:
		identity = struct {
			Kind                   string
			Volume                 any
			Mount                  any
			Prefix                 string
			AdditionalVolumes      []corev1.Volume      `json:",omitempty"`
			AdditionalVolumeMounts []corev1.VolumeMount `json:",omitempty"`
		}{
			"local", provider.Local.Volume, provider.Local.VolumeMount, provider.Local.Prefix,
			backup.Spec.AdditionalVolumes, backup.Spec.AdditionalVolumeMounts,
		}
	default:
		return "", fmt.Errorf("cannot identify unsupported storage destination")
	}
	data, err := json.Marshal(struct {
		BR          bool
		Mode        br.BackupMode
		Destination any
	}{backup.Spec.BR != nil, backup.Spec.Mode, identity})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

// ValidateStorageIdentity refuses to infer an existing backup's history from its mutable spec.
func ValidateStorageIdentity(backup *br.Backup) (string, error) {
	identity, err := StorageIdentity(backup)
	if err != nil {
		return "", err
	}
	if backup.Status.BackupPath != "" && backup.Status.StorageIdentity == "" {
		return "", fmt.Errorf("refusing cleanup or restart: original storage identity is unavailable; administrator verification is required")
	}
	if backup.Status.StorageIdentity != "" && backup.Status.StorageIdentity != identity {
		return "", fmt.Errorf("refusing cleanup or restart: storage identity has changed")
	}
	return identity, nil
}

// StorageIdentityForStart preserves legacy retries without inferring an original destination.
func StorageIdentityForStart(backup *br.Backup) (string, error) {
	if backup.Status.BackupPath != "" && backup.Status.StorageIdentity == "" {
		path, err := GetStoragePath(backup.Spec.StorageProvider)
		if err != nil {
			return "", err
		}
		if path != backup.Status.BackupPath {
			return "", fmt.Errorf("backup storage target cannot change after status.backupPath is recorded")
		}
		if err := ValidateBROptions(backup.Spec.BR); err != nil {
			return "", err
		}
		return "", nil
	}
	return ValidateStorageIdentity(backup)
}
