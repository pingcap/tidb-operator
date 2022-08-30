// Copyright 2018 PingCAP, Inc.
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

package controller

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateBackupStatus(t *testing.T) {
	g := NewGomegaWithT(t)
	tests := []struct {
		name         string
		status       *v1alpha1.BackupStatus
		updateStatus *BackupUpdateStatus
		expectStatus *v1alpha1.BackupStatus
	}{
		{
			name:         "updateStatus is nil",
			status:       newBackupStatus(),
			updateStatus: nil,
			expectStatus: newBackupStatus(),
		},
		{
			name:         "fields are nil",
			status:       newBackupStatus(),
			updateStatus: &BackupUpdateStatus{},
			expectStatus: newBackupStatus(),
		},
		{
			name:         "fields are not nil",
			status:       newBackupStatus(),
			updateStatus: newUpdateBackupStatus(),
			expectStatus: newExpectBackupStatus(),
		},
	}

	for _, test := range tests {
		t.Logf("test: %+v", test.name)
		updateBackupStatus(test.status, test.updateStatus)
		g.Expect(*test.status).Should(Equal(*test.expectStatus))
	}
}

func TestUpdateLogSubCommandConditionOnly(t *testing.T) {
	g := NewGomegaWithT(t)

	// Command LogSubCommandType `json:"command,omitempty"`
	// // TimeStarted is the time at which the command was started.
	// // TODO: remove nullable, https://github.com/kubernetes/kubernetes/issues/86811
	// // +nullable
	// TimeStarted metav1.Time `json:"timeStarted,omitempty"`
	// // TimeCompleted is the time at which the command was completed.
	// // TODO: remove nullable, https://github.com/kubernetes/kubernetes/issues/86811
	// // +nullable
	// TimeCompleted metav1.Time `json:"timeCompleted,omitempty"`
	// // LogTruncateUntil is log backup truncate until timestamp which will be the same as Spec.LogTruncateUntil when truncate is complete.
	// LogTruncateUntil string `json:"logTruncateUntil,omitempty"`
	// // Phase is the command current phase.
	// Phase BackupConditionType `json:"phase,omitempty"`
	// // +nullable
	// Conditions []BackupCondition `json:"conditions,omitempty"`

	// newLogSubCommandStatus := func() *v1alpha1.LogSubCommandStatus {

	// 	start, _ := time.Parse(time.RFC3339, "2020-12-25T12:46:59Z")
	// 	path := "xyz"
	// 	sizeReadable := "1M"
	// 	size := int64(1024)
	// 	return &v1alpha1.BackupStatus{
	// 		CommitTs: "421762809912885249",
	// 		Conditions: []v1alpha1.BackupCondition{
	// 			{
	// 				LastTransitionTime: metav1.Time{Time: start},
	// 				Message:            "",
	// 				Reason:             "",
	// 				Status:             "True",
	// 				Type:               v1alpha1.BackupInvalid,
	// 			},
	// 			{
	// 				LastTransitionTime: metav1.Time{Time: start},
	// 				Message:            "",
	// 				Reason:             "",
	// 				Status:             "True",
	// 				Type:               v1alpha1.BackupComplete,
	// 			},
	// 		},
	// 		Phase:              v1alpha1.BackupComplete,
	// 		TimeCompleted:      metav1.Time{Time: start},
	// 		TimeStarted:        metav1.Time{Time: start},
	// 		BackupPath:         path,
	// 		BackupSizeReadable: sizeReadable,
	// 		BackupSize:         size,
	// 	}

	// 	return nil
	// }

	tests := []struct {
		name            string
		status          *v1alpha1.LogSubCommandStatus
		updateCondition *v1alpha1.BackupCondition
		expectStatus    *v1alpha1.LogSubCommandStatus
	}{
		{
			name: "new condition update",
			status: &v1alpha1.LogSubCommandStatus{
				Command:     v1alpha1.LogStartCommand,
				TimeStarted: metav1.Time{Time: time.Now()},
				Conditions:  make([]v1alpha1.BackupCondition, 0),
			},
			updateCondition: &v1alpha1.BackupCondition{
				Type:               v1alpha1.BackupRunning,
				Status:             "True",
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             "abc",
				Message:            "bcd",
			},
			expectStatus: &v1alpha1.LogSubCommandStatus{
				Command: v1alpha1.LogStartCommand,
				Conditions: []v1alpha1.BackupCondition{
					{
						Reason:  "abc",
						Message: "bcd",
						Status:  "True",
						Type:    v1alpha1.BackupRunning,
					},
				},
				Phase: v1alpha1.BackupRunning,
			},
		},
	}

	for _, test := range tests {
		t.Logf("test: %+v", test.name)
		updateLogSubCommandConditionOnly(test.status, test.updateCondition)
		test.expectStatus.TimeStarted = test.status.TimeStarted
		test.expectStatus.Conditions[0].LastTransitionTime = test.status.Conditions[0].LastTransitionTime
		g.Expect(*test.status).Should(Equal(*test.expectStatus))
	}
}

// func TestUpdateLogBackupSubcommandStatus(t *testing.T) {
// 	g := NewGomegaWithT(t)

// 	// Command LogSubCommandType `json:"command,omitempty"`
// 	// // TimeStarted is the time at which the command was started.
// 	// // TODO: remove nullable, https://github.com/kubernetes/kubernetes/issues/86811
// 	// // +nullable
// 	// TimeStarted metav1.Time `json:"timeStarted,omitempty"`
// 	// // TimeCompleted is the time at which the command was completed.
// 	// // TODO: remove nullable, https://github.com/kubernetes/kubernetes/issues/86811
// 	// // +nullable
// 	// TimeCompleted metav1.Time `json:"timeCompleted,omitempty"`
// 	// // LogTruncateUntil is log backup truncate until timestamp which will be the same as Spec.LogTruncateUntil when truncate is complete.
// 	// LogTruncateUntil string `json:"logTruncateUntil,omitempty"`
// 	// // Phase is the command current phase.
// 	// Phase BackupConditionType `json:"phase,omitempty"`
// 	// // +nullable
// 	// Conditions []BackupCondition `json:"conditions,omitempty"`

// 	// newLogSubCommandStatus := func() *v1alpha1.LogSubCommandStatus {

// 	// 	start, _ := time.Parse(time.RFC3339, "2020-12-25T12:46:59Z")
// 	// 	path := "xyz"
// 	// 	sizeReadable := "1M"
// 	// 	size := int64(1024)
// 	// 	return &v1alpha1.BackupStatus{
// 	// 		CommitTs: "421762809912885249",
// 	// 		Conditions: []v1alpha1.BackupCondition{
// 	// 			{
// 	// 				LastTransitionTime: metav1.Time{Time: start},
// 	// 				Message:            "",
// 	// 				Reason:             "",
// 	// 				Status:             "True",
// 	// 				Type:               v1alpha1.BackupInvalid,
// 	// 			},
// 	// 			{
// 	// 				LastTransitionTime: metav1.Time{Time: start},
// 	// 				Message:            "",
// 	// 				Reason:             "",
// 	// 				Status:             "True",
// 	// 				Type:               v1alpha1.BackupComplete,
// 	// 			},
// 	// 		},
// 	// 		Phase:              v1alpha1.BackupComplete,
// 	// 		TimeCompleted:      metav1.Time{Time: start},
// 	// 		TimeStarted:        metav1.Time{Time: start},
// 	// 		BackupPath:         path,
// 	// 		BackupSizeReadable: sizeReadable,
// 	// 		BackupSize:         size,
// 	// 	}

// 	// 	return nil
// 	// }

// 	tests := []struct {
// 		name         string
// 		backup       *v1alpha1.Backup
// 		newCondition *v1alpha1.BackupCondition
// 		newStatus    *BackupUpdateStatus
// 		expectBackup *v1alpha1.Backup
// 	}{
// 		{
// 			name: "new status update",
// 			status: &v1alpha1.LogSubCommandStatus{
// 				Command:     v1alpha1.LogStartCommand,
// 				TimeStarted: metav1.Time{Time: time.Now()},
// 				Conditions:  make([]v1alpha1.BackupCondition, 0),
// 			},
// 			updateCondition: &v1alpha1.BackupCondition{
// 				Type:               v1alpha1.BackupRunning,
// 				Status:             "True",
// 				LastTransitionTime: metav1.Time{Time: time.Now()},
// 				Reason:             "abc",
// 				Message:            "bcd",
// 			},
// 			expectStatus: &v1alpha1.LogSubCommandStatus{
// 				Command: v1alpha1.LogStartCommand,
// 				Conditions: []v1alpha1.BackupCondition{
// 					{
// 						Reason:  "abc",
// 						Message: "bcd",
// 						Status:  "True",
// 						Type:    v1alpha1.BackupRunning,
// 					},
// 				},
// 				Phase: v1alpha1.BackupRunning,
// 			},
// 		},
// 	}

// 	for _, test := range tests {
// 		t.Logf("test: %+v", test.name)
// 		updateLogSubCommandConditionOnly(test.status, test.updateCondition)
// 		test.expectStatus.TimeStarted = test.status.TimeStarted
// 		test.expectStatus.Conditions[0].LastTransitionTime = test.status.Conditions[0].LastTransitionTime
// 		g.Expect(*test.status).Should(Equal(*test.expectStatus))
// 	}
// }

// func newBackupForTest(mode v1alpha1.BackupMode, ) *v1alpha1.Backup {
// 	Spec BackupSpec `json:"spec"`
// 	// +k8s:openapi-gen=false
// 	Status BackupStatus `json:"status,omitempty"`

// 	CommitTs string `json:"commitTs,omitempty"`
// 	// LogTruncateUntil is log backup truncate until timestamp.
// 	// Format supports TSO or datetime, e.g. '400036290571534337', '2018-05-11 01:42:23'.
// 	// +optional
// 	LogTruncateUntil string `json:"logTruncateUntil,omitempty"`
// 	// LogStop indicates that will stop the log backup.
// 	// +optional
// 	LogStop bool `json:"logStop,omitempty"`

// }

// func newBackupStatusForTest() v1alpha1.BackupStatus {

// 	// BackupPath is the location of the backup.
// 	BackupPath string `json:"backupPath,omitempty"`
// 	// TimeStarted is the time at which the backup was started.
// 	// TODO: remove nullable, https://github.com/kubernetes/kubernetes/issues/86811
// 	// +nullable
// 	TimeStarted metav1.Time `json:"timeStarted,omitempty"`
// 	// TimeCompleted is the time at which the backup was completed.
// 	// TODO: remove nullable, https://github.com/kubernetes/kubernetes/issues/86811
// 	// +nullable
// 	TimeCompleted metav1.Time `json:"timeCompleted,omitempty"`
// 	// BackupSizeReadable is the data size of the backup.
// 	// the difference with BackupSize is that its format is human readable
// 	BackupSizeReadable string `json:"backupSizeReadable,omitempty"`
// 	// BackupSize is the data size of the backup.
// 	BackupSize int64 `json:"backupSize,omitempty"`
// 	// CommitTs is the commit ts of the backup, snapshot ts for full backup or start ts for log backup.
// 	CommitTs string `json:"commitTs,omitempty"`
// 	// LogTruncateUntil is log backup truncate until timestamp which will be the same as Spec.LogTruncateUntil when truncate is complete.
// 	LogTruncateUntil string `json:"logTruncateUntil,omitempty"`
// 	// LogCheckpointTs is the ts of log backup process.
// 	LogCheckpointTs string `json:"logCheckpointTs,omitempty"`
// 	// LogStopped indicates whether the log backup has stopped.
// 	LogStopped bool `json:"logStopped,omitempty"`
// 	// Phase is a user readable state inferred from the underlying Backup conditions
// 	Phase BackupConditionType `json:"phase,omitempty"`
// 	// +nullable
// 	Conditions []BackupCondition `json:"conditions,omitempty"`
// 	// LogSubCommandConditions is the detail conditions of log backup subcommands, it is used to debug.
// 	LogSubCommandStatuses map[LogSubCommandType]LogSubCommandStatus `json:"logSubCommandCondition,omitempty"`

// }

func newBackupConditionForTest(conditionType v1alpha1.BackupConditionType, status v1.ConditionStatus, lastTransitionTime metav1.Time, reson, message string) v1alpha1.BackupCondition {
	return v1alpha1.BackupCondition{
		Type:               conditionType,
		Status:             status,
		LastTransitionTime: lastTransitionTime,
		Reason:             reson,
		Message:            message,
	}
}

func newUpdateBackupStatus() *BackupUpdateStatus {
	ts := "421762809912885269"
	start, _ := time.Parse(time.RFC3339, "2020-12-25T21:46:59Z")
	end, _ := time.Parse(time.RFC3339, "2020-12-25T21:50:59Z")
	path := "abcd"
	sizeReadable := "5M"
	size := int64(5024)
	return &BackupUpdateStatus{
		CommitTs:           &ts,
		TimeCompleted:      &metav1.Time{Time: end},
		TimeStarted:        &metav1.Time{Time: start},
		BackupPath:         &path,
		BackupSizeReadable: &sizeReadable,
		BackupSize:         &size,
	}
}

func newBackupStatus() *v1alpha1.BackupStatus {
	start, _ := time.Parse(time.RFC3339, "2020-12-25T12:46:59Z")
	path := "xyz"
	sizeReadable := "1M"
	size := int64(1024)
	return &v1alpha1.BackupStatus{
		CommitTs: "421762809912885249",
		Conditions: []v1alpha1.BackupCondition{
			{
				LastTransitionTime: metav1.Time{Time: start},
				Message:            "",
				Reason:             "",
				Status:             "True",
				Type:               v1alpha1.BackupInvalid,
			},
			{
				LastTransitionTime: metav1.Time{Time: start},
				Message:            "",
				Reason:             "",
				Status:             "True",
				Type:               v1alpha1.BackupComplete,
			},
		},
		Phase:              v1alpha1.BackupComplete,
		TimeCompleted:      metav1.Time{Time: start},
		TimeStarted:        metav1.Time{Time: start},
		BackupPath:         path,
		BackupSizeReadable: sizeReadable,
		BackupSize:         size,
	}
}
func newExpectBackupStatus() *v1alpha1.BackupStatus {
	ts := "421762809912885269"
	start, _ := time.Parse(time.RFC3339, "2020-12-25T21:46:59Z")
	end, _ := time.Parse(time.RFC3339, "2020-12-25T21:50:59Z")
	path := "abcd"
	sizeReadable := "5M"
	size := int64(5024)
	s := newBackupStatus()
	s.CommitTs = ts
	s.TimeStarted = metav1.Time{Time: start}
	s.TimeCompleted = metav1.Time{Time: end}
	s.BackupPath = path
	s.BackupSizeReadable = sizeReadable
	s.BackupSize = size
	return s
}
