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

func newUpdateBackupStatus() *BackupUpdateStatus {
	ts := "421762809912885269"
	start, _ := time.Parse(time.RFC3339, "2020-12-25T21:46:59Z")
	end, _ := time.Parse(time.RFC3339, "2020-12-25T21:50:59Z")
	path := "abcd"
	sizeReadable := "5M"
	size := int64(5024)
	progress := "30%"
	return &BackupUpdateStatus{
		CommitTs:           &ts,
		TimeCompleted:      &metav1.Time{Time: end},
		TimeStarted:        &metav1.Time{Time: start},
		BackupPath:         &path,
		BackupSizeReadable: &sizeReadable,
		BackupSize:         &size,
		Progress:           &progress,
	}
}

func newBackupStatus() *v1alpha1.BackupStatus {
	start, _ := time.Parse(time.RFC3339, "2020-12-25T12:46:59Z")
	path := "xyz"
	sizeReadable := "1M"
	size := int64(1024)
	progress := "30%"
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
		Progress:           progress,
	}
}
func newExpectBackupStatus() *v1alpha1.BackupStatus {
	ts := "421762809912885269"
	start, _ := time.Parse(time.RFC3339, "2020-12-25T21:46:59Z")
	end, _ := time.Parse(time.RFC3339, "2020-12-25T21:50:59Z")
	path := "abcd"
	sizeReadable := "5M"
	size := int64(5024)
	progress := "30%"
	s := newBackupStatus()
	s.CommitTs = ts
	s.TimeStarted = metav1.Time{Time: start}
	s.TimeCompleted = metav1.Time{Time: end}
	s.BackupPath = path
	s.BackupSizeReadable = sizeReadable
	s.BackupSize = size
	s.Progress = progress
	return s
}
