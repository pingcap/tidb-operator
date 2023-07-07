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
