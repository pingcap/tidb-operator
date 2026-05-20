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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateRestoreStatus(t *testing.T) {
	g := NewGomegaWithT(t)
	tests := []struct {
		name         string
		status       *v1alpha1.RestoreStatus
		updateStatus *RestoreUpdateStatus
		expectStatus *v1alpha1.RestoreStatus
	}{
		{
			name:         "updateStatus is nil",
			status:       newRestoreStatus(),
			updateStatus: nil,
			expectStatus: newRestoreStatus(),
		},
		{
			name:         "fields are nil",
			status:       newRestoreStatus(),
			updateStatus: &RestoreUpdateStatus{},
			expectStatus: newRestoreStatus(),
		},
		{
			name:         "fields are not nil",
			status:       newRestoreStatus(),
			updateStatus: newUpdateRestoreStatus(),
			expectStatus: newExpectRestoreStatus(),
		},
	}

	for _, test := range tests {
		t.Logf("test: %+v", test.name)
		updateRestoreStatus(test.status, test.updateStatus)
		g.Expect(*test.status).Should(Equal(*test.expectStatus))
	}
}

func newUpdateRestoreStatus() *RestoreUpdateStatus {
	ts := "421762809912885269"
	start, _ := time.Parse(time.RFC3339, "2020-12-25T21:46:59Z")
	end, _ := time.Parse(time.RFC3339, "2020-12-25T21:50:59Z")
	return &RestoreUpdateStatus{
		CommitTs:      &ts,
		TimeCompleted: &metav1.Time{Time: end},
		TimeStarted:   &metav1.Time{Time: start},
	}
}

func newRestoreStatus() *v1alpha1.RestoreStatus {
	start, _ := time.Parse(time.RFC3339, "2020-12-25T12:46:59Z")
	return &v1alpha1.RestoreStatus{
		CommitTs: "421762809912885249",
		Conditions: []v1alpha1.RestoreCondition{
			{
				LastTransitionTime: metav1.Time{Time: start},
				Message:            "",
				Reason:             "",
				Status:             "True",
				Type:               v1alpha1.RestoreInvalid,
			},
			{
				LastTransitionTime: metav1.Time{Time: start},
				Message:            "",
				Reason:             "",
				Status:             "True",
				Type:               v1alpha1.RestoreComplete,
			},
		},
		Phase:         v1alpha1.RestoreComplete,
		TimeCompleted: metav1.Time{Time: start},
		TimeStarted:   metav1.Time{Time: start},
	}
}
func newExpectRestoreStatus() *v1alpha1.RestoreStatus {
	ts := "421762809912885269"
	start, _ := time.Parse(time.RFC3339, "2020-12-25T21:46:59Z")
	end, _ := time.Parse(time.RFC3339, "2020-12-25T21:50:59Z")
	s := newRestoreStatus()
	s.CommitTs = ts
	s.TimeStarted = metav1.Time{Time: start}
	s.TimeCompleted = metav1.Time{Time: end}
	s.TimeTaken = "4m0s"
	return s
}

func TestReplicationRestoreStatusUpdater_Update_IsNoOp(t *testing.T) {
	g := NewGomegaWithT(t)

	u := NewReplicationRestoreStatusUpdater()

	// Build a Restore with non-empty status so we can assert it was NOT
	// mutated. The whole point of this wrapper is "controller is sole writer";
	// asserting only `err == nil` would let any silent-write impl pass.
	restore := &v1alpha1.Restore{
		Status: v1alpha1.RestoreStatus{
			Phase:         v1alpha1.RestoreSnapshotRestore,
			CommitTs:      "preserved-commit-ts",
			TimeStarted:   metav1.Time{Time: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
			TimeCompleted: metav1.Time{Time: time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)},
			TimeTaken:     "24h0m0s",
			Conditions: []v1alpha1.RestoreCondition{
				{Type: v1alpha1.RestoreRunning, Status: corev1.ConditionTrue, Reason: "preserved"},
			},
		},
	}
	originalConditionsLen := len(restore.Status.Conditions)
	originalPhase := restore.Status.Phase
	originalCommitTs := restore.Status.CommitTs
	originalTimeStarted := restore.Status.TimeStarted
	originalTimeCompleted := restore.Status.TimeCompleted
	originalTimeTaken := restore.Status.TimeTaken

	// Pass non-nil condition + non-nil RestoreUpdateStatus carrying values that
	// a real updater WOULD persist (TimeCompleted, CommitTs). Wrapper must drop them.
	newCommitTs := "new-commit-from-bm"
	newCompleted := metav1.Time{Time: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)}
	err := u.Update(
		restore,
		&v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "bm-tried-to-write-this",
			Message: "should-be-dropped",
		},
		&RestoreUpdateStatus{
			CommitTs:      &newCommitTs,
			TimeCompleted: &newCompleted,
		},
	)
	g.Expect(err).NotTo(HaveOccurred())

	// Critical: nothing on the input Restore was mutated.
	g.Expect(restore.Status.Phase).To(Equal(originalPhase),
		"wrapper must not mutate Phase")
	g.Expect(restore.Status.CommitTs).To(Equal(originalCommitTs),
		"wrapper must not mutate CommitTs (BackupManager-driven write must be dropped)")
	g.Expect(restore.Status.TimeStarted).To(Equal(originalTimeStarted))
	g.Expect(restore.Status.TimeCompleted).To(Equal(originalTimeCompleted),
		"wrapper must not mutate TimeCompleted")
	g.Expect(restore.Status.TimeTaken).To(Equal(originalTimeTaken))
	g.Expect(restore.Status.Conditions).To(HaveLen(originalConditionsLen),
		"wrapper must not append the passed-in condition")
	g.Expect(restore.Status.Conditions[0].Reason).To(Equal("preserved"),
		"wrapper must not modify existing conditions")

	// Nil args must also be safe (no panic).
	err = u.Update(nil, nil, nil)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestReplicationRestoreStatusUpdater_ImplementsInterface(t *testing.T) {
	g := NewGomegaWithT(t)
	var _ RestoreConditionUpdaterInterface = NewReplicationRestoreStatusUpdater()
	g.Expect(true).To(BeTrue()) // compile-time check
}
