// Copyright 2020 PingCAP, Inc.
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

package dmcluster

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	utildmcluster "github.com/pingcap/tidb-operator/pkg/util/dmcluster"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

func TestDMClusterConditionUpdater_Ready(t *testing.T) {
	tests := []struct {
		name        string
		dc          *v1alpha1.DMCluster
		wantStatus  v1.ConditionStatus
		wantReason  string
		wantMessage string
	}{
		{
			name: "statfulset(s) not up to date",
			dc: &v1alpha1.DMCluster{
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{},
					Worker: &v1alpha1.WorkerSpec{},
				},
				Status: v1alpha1.DMClusterStatus{
					Master: v1alpha1.MasterStatus{
						StatefulSet: &appsv1.StatefulSetStatus{
							CurrentRevision: "1",
							UpdateRevision:  "2",
						},
					},
					Worker: v1alpha1.WorkerStatus{
						StatefulSet: &appsv1.StatefulSetStatus{
							CurrentRevision: "1",
							UpdateRevision:  "2",
						},
					},
				},
			},
			wantStatus:  v1.ConditionFalse,
			wantReason:  utildmcluster.StatfulSetNotUpToDate,
			wantMessage: "Statefulset(s) are in progress",
		},
		{
			name: "dm-master(s) not healthy",
			dc: &v1alpha1.DMCluster{
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{
						Replicas: 1,
					},
					Worker: &v1alpha1.WorkerSpec{},
				},
				Status: v1alpha1.DMClusterStatus{
					Master: v1alpha1.MasterStatus{
						Members: map[string]v1alpha1.MasterMember{
							"dm-master-1": {
								Health: false,
							},
						},
						StatefulSet: &appsv1.StatefulSetStatus{
							CurrentRevision: "2",
							UpdateRevision:  "2",
						},
					},
					Worker: v1alpha1.WorkerStatus{
						StatefulSet: &appsv1.StatefulSetStatus{
							CurrentRevision: "2",
							UpdateRevision:  "2",
						},
					},
				},
			},
			wantStatus:  v1.ConditionFalse,
			wantReason:  utildmcluster.MasterUnhealthy,
			wantMessage: "dm-master(s) are not healthy",
		},
		{
			name: "all ready",
			dc: &v1alpha1.DMCluster{
				Spec: v1alpha1.DMClusterSpec{
					Master: v1alpha1.MasterSpec{
						Replicas: 1,
					},
					Worker: &v1alpha1.WorkerSpec{
						Replicas: 1,
					},
				},
				Status: v1alpha1.DMClusterStatus{
					Master: v1alpha1.MasterStatus{
						Members: map[string]v1alpha1.MasterMember{
							"dm-master-0": {
								Health: true,
							},
						},
						StatefulSet: &appsv1.StatefulSetStatus{
							CurrentRevision: "2",
							UpdateRevision:  "2",
						},
					},
					Worker: v1alpha1.WorkerStatus{
						Members: map[string]v1alpha1.WorkerMember{
							"dm-worker-0": {
								Stage: "free",
							},
						},
						StatefulSet: &appsv1.StatefulSetStatus{
							CurrentRevision: "2",
							UpdateRevision:  "2",
						},
					},
				},
			},
			wantStatus:  v1.ConditionTrue,
			wantReason:  utildmcluster.Ready,
			wantMessage: "DM cluster is fully up and running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conditionUpdater := &dmClusterConditionUpdater{}
			conditionUpdater.Update(tt.dc)
			cond := utildmcluster.GetDMClusterCondition(tt.dc.Status, v1alpha1.DMClusterReady)
			if diff := cmp.Diff(tt.wantStatus, cond.Status); diff != "" {
				t.Errorf("unexpected status (-want, +got): %s", diff)
			}
			if diff := cmp.Diff(tt.wantReason, cond.Reason); diff != "" {
				t.Errorf("unexpected reason (-want, +got): %s", diff)
			}
			if diff := cmp.Diff(tt.wantMessage, cond.Message); diff != "" {
				t.Errorf("unexpected message (-want, +got): %s", diff)
			}
		})
	}
}
