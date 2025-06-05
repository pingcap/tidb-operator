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

package tasks

import (
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1br "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
)

func TestDoUpdateConditions(t *testing.T) {
	tests := []struct {
		name            string
		initialConds    []metav1.Condition
		notReadyReasons []string
		unSyncedReasons []string
		expectChanged   bool
		expectReady     metav1.Condition
		expectSynced    metav1.Condition
	}{
		{
			name:            "Set both Ready and Synced to False with reasons",
			notReadyReasons: []string{"Component A not ready"},
			unSyncedReasons: []string{"Config mismatch"},
			expectChanged:   true,
			expectReady: metav1.Condition{
				Type:    v1alpha1br.TiBRCondReady,
				Status:  metav1.ConditionFalse,
				Reason:  v1alpha1br.TiBRReasonNotReady,
				Message: "Component A not ready",
			},
			expectSynced: metav1.Condition{
				Type:    v1alpha1br.TiBRCondSynced,
				Status:  metav1.ConditionFalse,
				Reason:  v1alpha1br.TiBRReasonUnSynced,
				Message: "Config mismatch",
			},
		},
		{
			name:          "Set both Ready and Synced to True",
			expectChanged: true,
			expectReady: metav1.Condition{
				Type:   v1alpha1br.TiBRCondReady,
				Status: metav1.ConditionTrue,
				Reason: v1alpha1br.TiBRReasonReady,
			},
			expectSynced: metav1.Condition{
				Type:   v1alpha1br.TiBRCondSynced,
				Status: metav1.ConditionTrue,
				Reason: v1alpha1br.TiBRReasonSynced,
			},
		},
		{
			name: "Update only Synced condition",
			initialConds: []metav1.Condition{{
				Type:   v1alpha1br.TiBRCondReady,
				Status: metav1.ConditionTrue,
				Reason: v1alpha1br.TiBRReasonReady,
			}},
			unSyncedReasons: []string{"Out of sync"},
			expectChanged:   true,
			expectReady: metav1.Condition{
				Type:   v1alpha1br.TiBRCondReady,
				Status: metav1.ConditionTrue,
				Reason: v1alpha1br.TiBRReasonReady,
			},
			expectSynced: metav1.Condition{
				Type:    v1alpha1br.TiBRCondSynced,
				Status:  metav1.ConditionFalse,
				Reason:  v1alpha1br.TiBRReasonUnSynced,
				Message: "Out of sync",
			},
		},
		{
			name: "No change needed",
			initialConds: []metav1.Condition{
				{
					Type:   v1alpha1br.TiBRCondReady,
					Status: metav1.ConditionTrue,
					Reason: v1alpha1br.TiBRReasonReady,
				},
				{
					Type:   v1alpha1br.TiBRCondSynced,
					Status: metav1.ConditionTrue,
					Reason: v1alpha1br.TiBRReasonSynced,
				},
			},
			expectChanged: false,
			expectReady: metav1.Condition{
				Type:   v1alpha1br.TiBRCondReady,
				Status: metav1.ConditionTrue,
				Reason: v1alpha1br.TiBRReasonReady,
			},
			expectSynced: metav1.Condition{
				Type:   v1alpha1br.TiBRCondSynced,
				Status: metav1.ConditionTrue,
				Reason: v1alpha1br.TiBRReasonSynced,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conds := append([]metav1.Condition{}, tt.initialConds...) // clone
			changed := doUpdateConditions(&conds, tt.notReadyReasons, tt.unSyncedReasons)

			assert.Equal(t, tt.expectChanged, changed)

			gotReady := meta.FindStatusCondition(conds, v1alpha1br.TiBRCondReady)
			gotSynced := meta.FindStatusCondition(conds, v1alpha1br.TiBRCondSynced)

			if assert.NotNil(t, gotReady) {
				assert.Equal(t, tt.expectReady.Type, gotReady.Type)
				assert.Equal(t, tt.expectReady.Status, gotReady.Status)
				assert.Equal(t, tt.expectReady.Reason, gotReady.Reason)
				if tt.expectReady.Message != "" {
					assert.Equal(t, tt.expectReady.Message, gotReady.Message)
				}
			}
			if assert.NotNil(t, gotSynced) {
				assert.Equal(t, tt.expectSynced.Type, gotSynced.Type)
				assert.Equal(t, tt.expectSynced.Status, gotSynced.Status)
				assert.Equal(t, tt.expectSynced.Reason, gotSynced.Reason)
				if tt.expectSynced.Message != "" {
					assert.Equal(t, tt.expectSynced.Message, gotSynced.Message)
				}
			}
		})
	}
}

func TestEvaluateReadyStatus_TableDriven(t *testing.T) {
	testCases := []struct {
		name         string
		ctx          *ReconcileContext
		expectedMsgs []string
	}{
		{
			name:         "All resources missing",
			ctx:          &ReconcileContext{},
			expectedMsgs: []string{"configmap is not existed", "statefulset is not existed", "headless service is not existed"},
		},
		{
			name: "Only statefulset not ready",
			ctx: &ReconcileContext{
				configmap:   &corev1.ConfigMap{},
				sts:         &appsv1.StatefulSet{Status: appsv1.StatefulSetStatus{ReadyReplicas: 0}},
				headlessSvc: &corev1.Service{},
			},
			expectedMsgs: []string{fmt.Sprintf("ready replica of statefulset is not %d", StatefulSetReplica)},
		},
		{
			name: "All resources ready",
			ctx: &ReconcileContext{
				configmap:   &corev1.ConfigMap{},
				sts:         &appsv1.StatefulSet{Status: appsv1.StatefulSetStatus{ReadyReplicas: StatefulSetReplica}},
				headlessSvc: &corev1.Service{},
			},
			expectedMsgs: []string{},
		},
		{
			name: "Only headless service missing",
			ctx: &ReconcileContext{
				configmap: &corev1.ConfigMap{},
				sts:       &appsv1.StatefulSet{Status: appsv1.StatefulSetStatus{ReadyReplicas: StatefulSetReplica}},
			},
			expectedMsgs: []string{"headless service is not existed"},
		},
		{
			name: "Only configmap missing",
			ctx: &ReconcileContext{
				sts:         &appsv1.StatefulSet{Status: appsv1.StatefulSetStatus{ReadyReplicas: StatefulSetReplica}},
				headlessSvc: &corev1.Service{},
			},
			expectedMsgs: []string{"configmap is not existed"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := evaluateReadyStatus(tc.ctx)
			assert.ElementsMatch(t, tc.expectedMsgs, actual)
		})
	}
}
