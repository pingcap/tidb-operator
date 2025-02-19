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

package v1alpha1

// fixme(ideascf): it's copied from tidb-operator/api/core/v1alpha1/common_types.go
const (
	// KeyPrefix defines key prefix of well known labels and annotations
	KeyPrefix = "pingcap.com/"

	// LabelKeyManagedBy means resources are managed by tidb operator
	LabelKeyManagedBy           = KeyPrefix + "managed-by"
	LabelValueManagedByOperator = "tidb-operator"
)

const (
	// ComponentLabelKey is Kubernetes recommended label key, it represents the component within the architecture
	ComponentLabelKey string = "app.kubernetes.io/component"
	// NameLabelKey is Kubernetes recommended label key, it represents the name of the application
	NameLabelKey string = "app.kubernetes.io/name"
	// InstanceLabelKey is Kubernetes recommended label key, it represents a unique name identifying the instance of an application
	// It's set by helm when installing a release
	InstanceLabelKey string = "app.kubernetes.io/instance"

	// BackupLabelKey is backup key
	BackupLabelKey string = "tidb.pingcap.com/backup"
	// RestoreLabelKey is restore key
	RestoreLabelKey string = "tidb.pingcap.com/restore"
	// BackupScheduleLabelKey is backup schedule key
	BackupScheduleLabelKey string = "tidb.pingcap.com/backup-schedule"

	// CleanJobLabelVal is clean job label value
	CleanJobLabelVal string = "clean"
	// RestoreJobLabelVal is restore job label value
	RestoreJobLabelVal string = "restore"
	// RestoreWarmUpJobLabelVal is restore warmup job label value
	RestoreWarmUpJobLabelVal string = "warmup"
	// BackupJobLabelVal is backup job label value
	BackupJobLabelVal string = "backup"
	// BackupScheduleJobLabelVal is backup schedule job label value
	BackupScheduleJobLabelVal string = "backup-schedule"
)

// Label is the label field in metadata
type Label map[string]string

// Instance adds instance kv pair to label
func (l Label) Instance(name string) Label {
	l[InstanceLabelKey] = name
	return l
}

// Component adds component kv pair to label
func (l Label) Component(name string) Label {
	l[ComponentLabelKey] = name
	return l
}

// CleanJob assigns clean to component key in label
func (l Label) CleanJob() Label {
	return l.Component(CleanJobLabelVal)
}

// BackupJob assigns backup to component key in label
func (l Label) BackupJob() Label {
	return l.Component(BackupJobLabelVal)
}

// RestoreJob assigns restore to component key in label
func (l Label) RestoreJob() Label {
	return l.Component(RestoreJobLabelVal)
}

// Backup assigns specific value to backup key in label
func (l Label) Backup(val string) Label {
	l[BackupLabelKey] = val
	return l
}

// BackupSchedule assigns specific value to backup schedule key in label
func (l Label) BackupSchedule(val string) Label {
	l[BackupScheduleLabelKey] = val
	return l
}

// Restore assigns specific value to restore key in label
func (l Label) Restore(val string) Label {
	l[RestoreLabelKey] = val
	return l
}

// NewBackup initialize a new Label for Jobs of bakcup
func NewBackup() Label {
	return Label{
		NameLabelKey:      BackupJobLabelVal,
		LabelKeyManagedBy: LabelValueManagedByOperator,
	}
}

// NewRestore initialize a new Label for Jobs of restore
func NewRestore() Label {
	return Label{
		NameLabelKey:      RestoreJobLabelVal,
		LabelKeyManagedBy: LabelValueManagedByOperator,
	}
}

// NewBackupSchedule initialize a new Label for backups of backup schedule
func NewBackupSchedule() Label {
	return Label{
		NameLabelKey:      BackupScheduleJobLabelVal,
		LabelKeyManagedBy: LabelValueManagedByOperator,
	}
}
