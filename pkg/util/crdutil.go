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
	"strings"

	crdutils "github.com/ant31/crd-validation/pkg"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	extensionsobj "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

var (
	tidbClusteradditionalPrinterColumns []extensionsobj.CustomResourceColumnDefinition
	tidbClusterPDColumn                 = extensionsobj.CustomResourceColumnDefinition{
		Name:        "PD",
		Type:        "string",
		Description: "The image for PD cluster",
		JSONPath:    ".spec.pd.image",
	}
	tidbClusterPDStorageColumn = extensionsobj.CustomResourceColumnDefinition{
		Name:        "Storage",
		Type:        "string",
		Description: "The storage size specified for PD node",
		JSONPath:    ".spec.pd.requests.storage",
	}
	tidbClusterPDReadyColumn = extensionsobj.CustomResourceColumnDefinition{
		Name:        "Ready",
		Type:        "integer",
		Description: "The desired replicas number of PD cluster",
		JSONPath:    ".status.pd.statefulSet.readyReplicas",
	}
	tidbClusterPDDesireColumn = extensionsobj.CustomResourceColumnDefinition{
		Name:        "Desire",
		Type:        "integer",
		Description: "The desired replicas number of PD cluster",
		JSONPath:    ".spec.pd.replicas",
	}
	tidbClusterTiKVColumn = extensionsobj.CustomResourceColumnDefinition{
		Name:        "TiKV",
		Type:        "string",
		Description: "The image for TiKV cluster",
		JSONPath:    ".spec.tikv.image",
	}
	tidbClusterTiKVStorageColumn = extensionsobj.CustomResourceColumnDefinition{
		Name:        "Storage",
		Type:        "string",
		Description: "The storage size specified for TiKV node",
		JSONPath:    ".spec.tikv.requests.storage",
	}
	tidbClusterTiKVReadyColumn = extensionsobj.CustomResourceColumnDefinition{
		Name:        "Ready",
		Type:        "integer",
		Description: "The ready replicas number of TiKV cluster",
		JSONPath:    ".status.tikv.statefulSet.readyReplicas",
	}
	tidbClusterTiKVDesireColumn = extensionsobj.CustomResourceColumnDefinition{
		Name:        "Desire",
		Type:        "integer",
		Description: "The desired replicas number of TiKV cluster",
		JSONPath:    ".spec.tikv.replicas",
	}
	tidbClusterTiDBColumn = extensionsobj.CustomResourceColumnDefinition{
		Name:        "TiDB",
		Type:        "string",
		Description: "The image for TiDB cluster",
		JSONPath:    ".spec.tidb.image",
	}
	tidbClusterTiDBReadyColumn = extensionsobj.CustomResourceColumnDefinition{
		Name:        "Ready",
		Type:        "integer",
		Description: "The ready replicas number of TiDB cluster",
		JSONPath:    ".status.tidb.statefulSet.readyReplicas",
	}
	tidbClusterTiDBDesireColumn = extensionsobj.CustomResourceColumnDefinition{
		Name:        "Desire",
		Type:        "integer",
		Description: "The desired replicas number of TiDB cluster",
		JSONPath:    ".spec.tidb.replicas",
	}
	backupAdditionalPrinterColumns []extensionsobj.CustomResourceColumnDefinition
	backupStorageTypeColumn        = extensionsobj.CustomResourceColumnDefinition{
		Name:        "StorageType",
		Type:        "string",
		Description: "The storage type of backup data",
		JSONPath:    ".spec.storageType",
	}
	backupBackupSizeColumn = extensionsobj.CustomResourceColumnDefinition{
		Name:        "BackupSize",
		Type:        "integer",
		Description: "The data size of the backup",
		JSONPath:    ".status.backupSize",
	}
	backupCommitTSColumn = extensionsobj.CustomResourceColumnDefinition{
		Name:        "CommitTS",
		Type:        "string",
		Description: "The commit ts of tidb cluster dump",
		JSONPath:    ".status.commitTs",
	}
	backupStartedColumn = extensionsobj.CustomResourceColumnDefinition{
		Name:        "Started",
		Type:        "date",
		Description: "The time at which the backup was started",
		Priority:    1,
		JSONPath:    ".status.timeStarted",
	}
	backupCompletedColumn = extensionsobj.CustomResourceColumnDefinition{
		Name:        "Completed",
		Type:        "date",
		Description: "The time at which the backup was completed",
		Priority:    1,
		JSONPath:    ".status.timeCompleted",
	}
	restoreAdditionalPrinterColumns []extensionsobj.CustomResourceColumnDefinition
	restoreBackupColumn             = extensionsobj.CustomResourceColumnDefinition{
		Name:        "Backup",
		Type:        "string",
		Description: "The backup that used to restore",
		JSONPath:    ".spec.backup",
	}
	restoreStartedColumn = extensionsobj.CustomResourceColumnDefinition{
		Name:        "Started",
		Type:        "date",
		Description: "The time at which the backup was started",
		Priority:    1,
		JSONPath:    ".status.timeStarted",
	}
	restoreCompletedColumn = extensionsobj.CustomResourceColumnDefinition{
		Name:        "Completed",
		Type:        "date",
		Description: "The time at which the restore was completed",
		Priority:    1,
		JSONPath:    ".status.timeCompleted",
	}
	bksAdditionalPrinterColumns []extensionsobj.CustomResourceColumnDefinition
	bksScheduleColumn           = extensionsobj.CustomResourceColumnDefinition{
		Name:        "Schedule",
		Type:        "string",
		Description: "The cron format string used for backup scheduling.",
		JSONPath:    ".spec.schedule",
	}
	bksMaxBackups = extensionsobj.CustomResourceColumnDefinition{
		Name:        "MaxBackups",
		Type:        "integer",
		Description: "The max number of backups we want to keep.",
		JSONPath:    ".spec.maxBackups",
	}
	bksLastBackup = extensionsobj.CustomResourceColumnDefinition{
		Name:        "LastBackup",
		Type:        "string",
		Description: "The last backup CR name",
		Priority:    1,
		JSONPath:    ".status.lastBackup",
	}
	bksLastBackupTime = extensionsobj.CustomResourceColumnDefinition{
		Name:        "LastBackupTime",
		Type:        "date",
		Description: "The last time the backup was successfully created",
		Priority:    1,
		JSONPath:    ".status.lastBackupTime",
	}
)

func init() {
	tidbClusteradditionalPrinterColumns = append(tidbClusteradditionalPrinterColumns,
		tidbClusterPDColumn, tidbClusterPDStorageColumn, tidbClusterPDReadyColumn, tidbClusterPDDesireColumn,
		tidbClusterTiKVColumn, tidbClusterTiKVStorageColumn, tidbClusterTiKVReadyColumn, tidbClusterTiKVDesireColumn,
		tidbClusterTiDBColumn, tidbClusterTiDBReadyColumn, tidbClusterTiDBDesireColumn)
	backupAdditionalPrinterColumns = append(backupAdditionalPrinterColumns, backupStorageTypeColumn, backupBackupSizeColumn, backupCommitTSColumn, backupStartedColumn, backupCompletedColumn)
	restoreAdditionalPrinterColumns = append(restoreAdditionalPrinterColumns, restoreBackupColumn, restoreStartedColumn, restoreCompletedColumn)
	bksAdditionalPrinterColumns = append(bksAdditionalPrinterColumns, bksScheduleColumn, bksMaxBackups, bksLastBackup, bksLastBackupTime)
}

func NewCustomResourceDefinition(crdKind v1alpha1.CrdKind, group string, labels map[string]string, validation bool) *extensionsobj.CustomResourceDefinition {
	crd := crdutils.NewCustomResourceDefinition(crdutils.Config{
		SpecDefinitionName:    crdKind.SpecName,
		EnableValidation:      validation,
		Labels:                crdutils.Labels{LabelsMap: labels},
		ResourceScope:         string(extensionsobj.NamespaceScoped),
		Group:                 group,
		Kind:                  crdKind.Kind,
		Version:               v1alpha1.Version,
		Plural:                crdKind.Plural,
		ShortNames:            crdKind.ShortNames,
		GetOpenAPIDefinitions: v1alpha1.GetOpenAPIDefinitions,
	})
	addAdditionalPrinterColumnsForCRD(crd, crdKind)
	return crd
}

func GetCrdKindFromKindName(kindName string) (v1alpha1.CrdKind, error) {
	switch strings.ToLower(kindName) {
	case v1alpha1.TiDBClusterKindKey:
		return v1alpha1.DefaultCrdKinds.TiDBCluster, nil
	case v1alpha1.BackupKindKey:
		return v1alpha1.DefaultCrdKinds.Backup, nil
	case v1alpha1.RestoreKindKey:
		return v1alpha1.DefaultCrdKinds.Restore, nil
	case v1alpha1.BackupScheduleKindKey:
		return v1alpha1.DefaultCrdKinds.BackupSchedule, nil
	default:
		return v1alpha1.CrdKind{}, errors.New("unknown CrdKind Name")
	}
}

func addAdditionalPrinterColumnsForCRD(crd *extensionsobj.CustomResourceDefinition, crdKind v1alpha1.CrdKind) {
	switch crdKind.Kind {
	case v1alpha1.DefaultCrdKinds.TiDBCluster.Kind:
		crd.Spec.AdditionalPrinterColumns = tidbClusteradditionalPrinterColumns
		break
	case v1alpha1.DefaultCrdKinds.Backup.Kind:
		crd.Spec.AdditionalPrinterColumns = backupAdditionalPrinterColumns
		break
	case v1alpha1.DefaultCrdKinds.Restore.Kind:
		crd.Spec.AdditionalPrinterColumns = restoreAdditionalPrinterColumns
		break
	case v1alpha1.DefaultCrdKinds.BackupSchedule.Kind:
		crd.Spec.AdditionalPrinterColumns = bksAdditionalPrinterColumns
		break
	default:
		break
	}
}
