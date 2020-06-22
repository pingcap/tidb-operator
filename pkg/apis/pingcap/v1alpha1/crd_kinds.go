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

package v1alpha1

import (
	extensionsobj "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

const (
	Version   = "v1alpha1"
	GroupName = "pingcap.com"

	TiDBClusterName    = "tidbclusters"
	TiDBClusterKind    = "TidbCluster"
	TiDBClusterKindKey = "tidbcluster"

	BackupName    = "backups"
	BackupKind    = "Backup"
	BackupKindKey = "backup"

	RestoreName    = "restores"
	RestoreKind    = "Restore"
	RestoreKindKey = "restore"

	BackupScheduleName    = "backupschedules"
	BackupScheduleKind    = "BackupSchedule"
	BackupScheduleKindKey = "backupschedule"

	TiDBMonitorName    = "tidbmonitors"
	TiDBMonitorKind    = "TidbMonitor"
	TiDBMonitorKindKey = "tidbmonitor"

	TiDBInitializerName    = "tidbinitializers"
	TiDBInitializerKind    = "TidbInitializer"
	TiDBInitializerKindKey = "tidbinitializer"

	TidbClusterAutoScalerName    = "tidbclusterautoscalers"
	TidbClusterAutoScalerKind    = "TidbClusterAutoScaler"
	TidbClusterAutoScalerKindKey = "tidbclusterautoscaler"

	TiKVGroupName    = "tikvgroups"
	TiKVGroupKind    = "TiKVGroup"
	TiKVGroupKindKey = "tikvgroup"

	TiDBGroupName    = "tidbgroups"
	TiDBGroupKind    = "TiDBGroup"
	TiDBGroupKindKey = "tidbgroup"

	SpecPath = "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1."
)

type CrdKind struct {
	Kind                    string
	Plural                  string
	SpecName                string
	ShortNames              []string
	AdditionalPrinterColums []extensionsobj.CustomResourceColumnDefinition
}

type CrdKinds struct {
	KindsString           string
	TiDBCluster           CrdKind
	Backup                CrdKind
	Restore               CrdKind
	BackupSchedule        CrdKind
	TiDBMonitor           CrdKind
	TiDBInitializer       CrdKind
	TidbClusterAutoScaler CrdKind
	TiKVGroup             CrdKind
	TiDBGroup             CrdKind
}

var DefaultCrdKinds = CrdKinds{
	KindsString:           "",
	TiDBCluster:           CrdKind{Plural: TiDBClusterName, Kind: TiDBClusterKind, ShortNames: []string{"tc"}, SpecName: SpecPath + TiDBClusterKind},
	Backup:                CrdKind{Plural: BackupName, Kind: BackupKind, ShortNames: []string{"bk"}, SpecName: SpecPath + BackupKind},
	Restore:               CrdKind{Plural: RestoreName, Kind: RestoreKind, ShortNames: []string{"rt"}, SpecName: SpecPath + RestoreKind},
	BackupSchedule:        CrdKind{Plural: BackupScheduleName, Kind: BackupScheduleKind, ShortNames: []string{"bks"}, SpecName: SpecPath + BackupScheduleKind},
	TiDBMonitor:           CrdKind{Plural: TiDBMonitorName, Kind: TiDBMonitorKind, ShortNames: []string{}, SpecName: SpecPath + TiDBMonitorKind},
	TiDBInitializer:       CrdKind{Plural: TiDBInitializerName, Kind: TiDBInitializerKind, ShortNames: []string{"ti"}, SpecName: SpecPath + TiDBInitializerKind},
	TidbClusterAutoScaler: CrdKind{Plural: TidbClusterAutoScalerName, Kind: TidbClusterAutoScalerKind, ShortNames: []string{"ta"}, SpecName: SpecPath + TidbClusterAutoScalerKind},
	TiKVGroup:             CrdKind{Plural: TiKVGroupName, Kind: TiKVGroupKind, ShortNames: []string{}, SpecName: SpecPath + TiKVGroupKind},
	TiDBGroup:             CrdKind{Plural: TiDBGroupName, Kind: TiDBGroupKind, ShortNames: []string{}, SpecName: SpecPath + TiDBGroupKind},
}
