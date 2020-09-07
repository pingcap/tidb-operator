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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Reasons for DMCluster conditions.

	// Ready
	Ready = "Ready"
	// StatefulSetNotUpToDate is added when one of statefulsets is not up to date.
	StatfulSetNotUpToDate = "StatefulSetNotUpToDate"
	// MasterUnhealthy is added when one of dm-master members is unhealthy.
	MasterUnhealthy = "DMMasterUnhealthy"
)

// NewDMClusterCondition creates a new dmcluster condition.
func NewDMClusterCondition(condType v1alpha1.DMClusterConditionType, status v1.ConditionStatus, reason, message string) *v1alpha1.DMClusterCondition {
	return &v1alpha1.DMClusterCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// GetDMClusterCondition returns the condition with the provided type.
func GetDMClusterCondition(status v1alpha1.DMClusterStatus, condType v1alpha1.DMClusterConditionType) *v1alpha1.DMClusterCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetDMClusterCondition updates the dm cluster to include the provided condition. If the condition that
// we are about to add already exists and has the same status and reason then we are not going to update.
func SetDMClusterCondition(status *v1alpha1.DMClusterStatus, condition v1alpha1.DMClusterCondition) {
	currentCond := GetDMClusterCondition(*status, condition.Type)
	if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
		return
	}
	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}
	newConditions := filterOutCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

// filterOutCondition returns a new slice of tidbcluster conditions without conditions with the provided type.
func filterOutCondition(conditions []v1alpha1.DMClusterCondition, condType v1alpha1.DMClusterConditionType) []v1alpha1.DMClusterCondition {
	var newConditions []v1alpha1.DMClusterCondition
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}

// GetDMClusterReadyCondition extracts the dmcluster ready condition from the given status and returns that.
// Returns nil if the condition is not present.
func GetDMClusterReadyCondition(status v1alpha1.DMClusterStatus) *v1alpha1.DMClusterCondition {
	return GetDMClusterCondition(status, v1alpha1.DMClusterReady)
}
