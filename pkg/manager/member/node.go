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

const (
	nodeCondRODiskFound = "RODiskFound"
)

// a pre-defined mapping that mapping some short label name to k8s well-known labels.
// PD depend on short label name to gain better performance.
// See: https://github.com/pingcap/tidb-operator/issues/4678 for more details.
var shortLabelNameToK8sLabel = map[string][]string{
	"region": {corev1.LabelZoneRegionStable, corev1.LabelZoneRegion},
	"zone":   {corev1.LabelZoneFailureDomainStable, corev1.LabelZoneFailureDomain},
	"host":   {corev1.LabelHostname},
}

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

		if k8sLabels, ok := shortLabelNameToK8sLabel[storeLabel]; ok {
			for _, name := range k8sLabels {
				if value, ok := ls[name]; ok {
					labels[storeLabel] = value
					break
				}
			}
		}
	}
	return labels, nil
}

// IsNodeReadyConditionFalseOrUnknown returns true if a pod is not ready; false otherwise.
func IsNodeReadyConditionFalseOrUnknown(status corev1.NodeStatus) bool {
	condition := getNodeReadyCondition(status)
	return condition != nil && (condition.Status == corev1.ConditionFalse || condition.Status == corev1.ConditionUnknown)
}

// IsNodeRODiskFoundConditionTrue returns true if a pod has RODiskFound condition set to True; false otherwise.
func IsNodeRODiskFoundConditionTrue(status corev1.NodeStatus) bool {
	_, condition := getNodeCondition(&status, nodeCondRODiskFound)
	return condition != nil && condition.Status == corev1.ConditionTrue
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
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return i, &conditions[i]
		}
	}
	return -1, nil
}
