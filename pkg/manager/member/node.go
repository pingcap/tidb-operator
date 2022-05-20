// Copyright 2021 PingCAP, Inc.
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

package member

import (
	corev1 "k8s.io/api/core/v1"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
)

func getNodeLabels(nodeLister corelisterv1.NodeLister, nodeName string, storeLabels []string) (map[string]string, error) {
	node, err := nodeLister.Get(nodeName)
	if err != nil {
		return nil, err
	}
	labels := map[string]string{}
	ls := node.GetLabels()
	for _, storeLabel := range storeLabels {
		if value, found := ls[storeLabel]; found {
			labels[storeLabel] = value
			continue
		}

		// TODO after pd supports storeLabel containing slash character, these codes should be deleted
		if storeLabel == "host" {
			if host, found := ls[corev1.LabelHostname]; found {
				labels[storeLabel] = host
			}
		}

	}
	return labels, nil
}

// IsNodeReadyConditionFalse returns true if a pod is not ready; false otherwise.
func IsNodeReadyConditionFalse(status corev1.NodeStatus) bool {
	condition := getNodeReadyCondition(status)
	return condition != nil && condition.Status == corev1.ConditionFalse
}

// GetNodeReadyCondition extracts the node ready condition from the given status and returns that.
// Returns nil if the condition is not present.
func getNodeReadyCondition(status corev1.NodeStatus) *corev1.NodeCondition {
	_, condition := getNodeCondition(&status, corev1.NodeReady)
	return condition
}

// GetNodeCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func getNodeCondition(status *corev1.NodeStatus, conditionType corev1.NodeConditionType) (int, *corev1.NodeCondition) {
	if status == nil {
		return -1, nil
	}
	return getNodeConditionFromList(status.Conditions, conditionType)
}

// GetNodeConditionFromList extracts the provided condition from the given list of condition and
// returns the index of the condition and the condition. Returns -1 and nil if the condition is not present.
func getNodeConditionFromList(conditions []corev1.NodeCondition, conditionType corev1.NodeConditionType) (int, *corev1.NodeCondition) {
	if conditions == nil {
		return -1, nil
	}
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return i, &conditions[i]
		}
	}
	return -1, nil
}
