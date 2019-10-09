package v1alpha1

import (
	"fmt"
	"strings"
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

	SpecPath = "github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1."
)

type CrdKind struct {
	Kind       string
	Plural     string
	SpecName   string
	ShortNames []string
}

type CrdKinds struct {
	KindsString    string
	TiDBCluster    CrdKind
	Backup         CrdKind
	Restore        CrdKind
	BackupSchedule CrdKind
}

var DefaultCrdKinds = CrdKinds{
	KindsString:    "",
	TiDBCluster:    CrdKind{Plural: TiDBClusterName, Kind: TiDBClusterKind, ShortNames: []string{"tc"}, SpecName: SpecPath + TiDBClusterKind},
	Backup:         CrdKind{Plural: BackupName, Kind: BackupKind, ShortNames: []string{"bk"}, SpecName: SpecPath + BackupKind},
	Restore:        CrdKind{Plural: RestoreName, Kind: RestoreKind, ShortNames: []string{"rt"}, SpecName: SpecPath + RestoreKind},
	BackupSchedule: CrdKind{Plural: BackupScheduleName, Kind: BackupScheduleKind, ShortNames: []string{"bks"}, SpecName: SpecPath + BackupScheduleKind},
}

// Implement the flag.Value interface
func (crdkinds *CrdKinds) String() string {
	return crdkinds.KindsString
}

func (crdkinds *CrdKinds) Set(value string) error {
	*crdkinds = DefaultCrdKinds
	if value == "" {
		value = fmt.Sprintf("%s=%s:%s,%s=%s:%s,%s=%s:%s,%s=%s:%s",
			TiDBClusterKindKey, TiDBClusterKind, TiDBClusterName,
			BackupKindKey, BackupKind, BackupName,
			RestoreKindKey, RestoreKind, RestoreName,
			BackupScheduleKindKey, BackupScheduleKind, BackupScheduleName,
		)
	}
	splited := strings.Split(value, ",")
	for _, pair := range splited {
		sp := strings.Split(pair, "=")
		kind := strings.Split(sp[1], ":")
		crdKind := CrdKind{Plural: kind[1], Kind: kind[0]}
		switch kindKey := sp[0]; kindKey {
		case TiDBClusterKindKey:
			(*crdkinds).TiDBCluster = crdKind
		case BackupKindKey:
			(*crdkinds).Backup = crdKind
		case RestoreKindKey:
			(*crdkinds).Restore = crdKind
		case BackupScheduleKindKey:
			(*crdkinds).BackupSchedule = crdKind
		default:
			fmt.Printf("Warning: unknown kind: %s... ignoring", kindKey)
		}
	}
	(*crdkinds).KindsString = value
	return nil
}
