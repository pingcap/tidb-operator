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
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateBackupStatus(t *testing.T) {
	g := NewGomegaWithT(t)
	tests := []struct {
		name         string
		status       *v1alpha1.BackupStatus
		updateStatus *BackupUpdateStatus
		expectStatus *v1alpha1.BackupStatus
		expectUpdate bool
	}{
		{
			name:         "updateStatus is nil",
			status:       newBackupStatus(),
			updateStatus: nil,
			expectStatus: newBackupStatus(),
			expectUpdate: false,
		},
		{
			name:         "fields are nil",
			status:       newBackupStatus(),
			updateStatus: &BackupUpdateStatus{},
			expectStatus: newBackupStatus(),
			expectUpdate: false,
		},
		{
			name:         "fields are not nil",
			status:       newBackupStatus(),
			updateStatus: newUpdateBackupStatus(),
			expectStatus: newExpectBackupStatus(),
			expectUpdate: true,
		},
		{
			name: "scalar update is preserved when retry status is unchanged",
			status: &v1alpha1.BackupStatus{
				BackupPath: "old-path",
				BackoffRetryStatus: []v1alpha1.BackoffRetryRecord{
					{
						RetryNum: 1,
					},
				},
			},
			updateStatus: &BackupUpdateStatus{
				BackupPath: stringPtr("new-path"),
				RetryNum:   intPtr(1),
			},
			expectStatus: &v1alpha1.BackupStatus{
				BackupPath: "new-path",
				BackoffRetryStatus: []v1alpha1.BackoffRetryRecord{
					{
						RetryNum: 1,
					},
				},
			},
			expectUpdate: true,
		},
		{
			name: "nil observability update changes nothing",
			status: &v1alpha1.BackupStatus{
				BROperations: makeBROperations(0, 2),
			},
			updateStatus: &BackupUpdateStatus{},
			expectStatus: &v1alpha1.BackupStatus{
				BROperations: makeBROperations(0, 2),
			},
			expectUpdate: false,
		},
		{
			name: "new operation is prepended",
			status: &v1alpha1.BackupStatus{
				BROperations: makeBROperations(0, 2),
			},
			updateStatus: &BackupUpdateStatus{
				BROperation: newBROperation("op-new", "backup"),
			},
			expectStatus: &v1alpha1.BackupStatus{
				BROperations: append([]v1alpha1.BROperation{*newBROperation("op-new", "backup")}, makeBROperations(0, 2)...),
			},
			expectUpdate: true,
		},
		{
			name: "duplicate operation ID refreshes existing entry",
			status: &v1alpha1.BackupStatus{
				BROperations: []v1alpha1.BROperation{
					*newBROperation("op-a", "old-backup"),
					*newBROperation("op-b", "backup"),
				},
			},
			updateStatus: &BackupUpdateStatus{
				BROperation: newBROperation("op-a", "refreshed-backup"),
			},
			expectStatus: &v1alpha1.BackupStatus{
				BROperations: []v1alpha1.BROperation{
					*newBROperation("op-a", "refreshed-backup"),
					*newBROperation("op-b", "backup"),
				},
			},
			expectUpdate: true,
		},
		{
			name: "operation list is capped at 10",
			status: &v1alpha1.BackupStatus{
				BROperations: makeBROperations(0, 10),
			},
			updateStatus: &BackupUpdateStatus{
				BROperation: newBROperation("op-new", "backup"),
			},
			expectStatus: &v1alpha1.BackupStatus{
				BROperations: append([]v1alpha1.BROperation{*newBROperation("op-new", "backup")}, makeBROperations(0, 9)...),
			},
			expectUpdate: true,
		},
	}

	for _, test := range tests {
		t.Logf("test: %+v", test.name)
		updated := updateBackupStatus(test.status, test.updateStatus)
		g.Expect(updated).Should(Equal(test.expectUpdate))
		g.Expect(*test.status).Should(Equal(*test.expectStatus))
	}
}

func TestUpdateBROperations(t *testing.T) {
	g := NewGomegaWithT(t)

	existing := makeBROperations(0, 10)
	operations, updated := updateBROperations(existing, newBROperation("op-new", "backup"))
	g.Expect(updated).Should(BeTrue())
	g.Expect(operations).Should(Equal(append([]v1alpha1.BROperation{*newBROperation("op-new", "backup")}, makeBROperations(0, 9)...)))

	operations, updated = updateBROperations(existing, newBROperation("op-01", "refreshed"))
	g.Expect(updated).Should(BeTrue())
	g.Expect(operations).Should(Equal(append([]v1alpha1.BROperation{*newBROperation("op-01", "refreshed")}, append(makeBROperations(0, 1), makeBROperations(2, 8)...)...)))

	operations, updated = updateBROperations(existing, nil)
	g.Expect(updated).Should(BeFalse())
	g.Expect(operations).Should(Equal(existing))

	operations, updated = updateBROperations(existing, &v1alpha1.BROperation{})
	g.Expect(updated).Should(BeFalse())
	g.Expect(operations).Should(Equal(existing))

	observed := newBROperation("op-deep-copy", "backup")
	operations, updated = updateBROperations(nil, observed)
	g.Expect(updated).Should(BeTrue())
	g.Expect(operations).Should(HaveLen(1))

	originalStartedAt := observed.StartedAt.Time
	observed.StartedAt.Time = observed.StartedAt.Time.Add(time.Hour)
	g.Expect(operations[0].StartedAt.Time).Should(Equal(originalStartedAt))
}

func TestUpdateLogBackupStatusUpdatesBROperationWithoutCondition(t *testing.T) {
	g := NewGomegaWithT(t)
	backup := &v1alpha1.Backup{
		Spec: v1alpha1.BackupSpec{
			Mode: v1alpha1.BackupModeLog,
		},
		Status: v1alpha1.BackupStatus{
			BROperations: makeBROperations(0, 1),
		},
	}
	operation := newBROperation("op-log-truncate", "log truncate")

	updated := updateLogBackupStatus(backup, nil, &BackupUpdateStatus{
		BROperation: operation,
	})

	g.Expect(updated).Should(BeTrue())
	g.Expect(backup.Status.BROperations).Should(Equal(append([]v1alpha1.BROperation{*operation}, makeBROperations(0, 1)...)))
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
	s.TimeTaken = "4m0s"
	s.BackupPath = path
	s.BackupSizeReadable = sizeReadable
	s.BackupSize = size
	return s
}

func newBROperation(operationID, command string) *v1alpha1.BROperation {
	startedAt := metav1.Time{Time: time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC)}
	return &v1alpha1.BROperation{
		OperationID: operationID,
		StartedAt:   &startedAt,
		Command:     command,
		ObservedAt:  metav1.Time{Time: time.Date(2026, 6, 17, 10, 1, 0, 0, time.UTC)},
	}
}

func makeBROperations(start, count int) []v1alpha1.BROperation {
	operations := make([]v1alpha1.BROperation, 0, count)
	for i := start; i < start+count; i++ {
		operations = append(operations, *newBROperation(fmt.Sprintf("op-%02d", i), "backup"))
	}
	return operations
}

func intPtr(value int) *int {
	return &value
}

func stringPtr(value string) *string {
	return &value
}
