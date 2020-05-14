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

package tidbcluster

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Reasons for TidbCluster conditions.

	// Ready
	Ready = "Ready"
	// StatefulSetNotUpToDate is added when one of statefulsets is not up to date.
	StatfulSetNotUpToDate = "StatefulSetNotUpToDate"
	// PDUnhealthy is added when one of pd members is unhealthy.
	PDUnhealthy = "PDUnhealthy"
	// TiKVStoreNotUp is added when one of tikv stores is not up.
	TiKVStoreNotUp = "TiKVStoreNotUp"
	// TiDBUnhealthy is added when one of tidb pods is unhealthy.
	TiDBUnhealthy = "TiDBUnhealthy"
	// TiFlashStoreNotUp is added when one of tiflash stores is not up.
	TiFlashStoreNotUp = "TiFlashStoreNotUp"
)

// NewTidbClusterCondition creates a new tidbcluster condition.
func NewTidbClusterCondition(condType v1alpha1.TidbClusterConditionType, status v1.ConditionStatus, reason, message string) *v1alpha1.TidbClusterCondition {
	return &v1alpha1.TidbClusterCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// GetTidbClusterCondition returns the condition with the provided type.
func GetTidbClusterCondition(status v1alpha1.TidbClusterStatus, condType v1alpha1.TidbClusterConditionType) *v1alpha1.TidbClusterCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetTidbClusterCondition updates the tidb cluster to include the provided condition. If the condition that
// we are about to add already exists and has the same status and reason then we are not going to update.
func SetTidbClusterCondition(status *v1alpha1.TidbClusterStatus, condition v1alpha1.TidbClusterCondition) {
	currentCond := GetTidbClusterCondition(*status, condition.Type)
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
func filterOutCondition(conditions []v1alpha1.TidbClusterCondition, condType v1alpha1.TidbClusterConditionType) []v1alpha1.TidbClusterCondition {
	var newConditions []v1alpha1.TidbClusterCondition
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}
