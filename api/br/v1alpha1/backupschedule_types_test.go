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

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBackupScheduleStatusDeepCopy(t *testing.T) {
	now := metav1.NewTime(time.Date(2026, 8, 20, 1, 2, 3, 0, time.UTC))
	original := &BackupSchedule{
		Status: BackupScheduleStatus{
			LastScheduleTime: &now,
			Conditions: []metav1.Condition{{
				Type:               "SchedulingReady",
				Status:             metav1.ConditionTrue,
				Reason:             "Reconciled",
				ObservedGeneration: 2,
				LastTransitionTime: now,
			}},
		},
	}

	copy := original.DeepCopy()
	if original.Status.LastScheduleTime == copy.Status.LastScheduleTime {
		t.Fatal("LastScheduleTime pointer was not deep-copied")
	}
	if &original.Status.Conditions[0] == &copy.Status.Conditions[0] {
		t.Fatal("Conditions slice was not deep-copied")
	}
	copy.Status.LastScheduleTime.Time = copy.Status.LastScheduleTime.Add(time.Hour)
	copy.Status.Conditions[0].Reason = "Changed"

	if want := time.Date(2026, 8, 20, 1, 2, 3, 0, time.UTC); !original.Status.LastScheduleTime.Time.Equal(want) {
		t.Fatalf("original LastScheduleTime changed, got %s, want %s", original.Status.LastScheduleTime, want)
	}
	if original.Status.Conditions[0].Reason != "Reconciled" {
		t.Fatalf("original condition changed, got reason %q", original.Status.Conditions[0].Reason)
	}
}
