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

package k8s

import (
	corev1 "k8s.io/api/core/v1"
)

// a pre-defined mapping that mapping some short label name to K8s well-known labels.
// PD depend on short label name to gain better performance.
// See: https://github.com/pingcap/tidb-operator/issues/4678 for more details.
var shortLabelNameToK8sLabel = map[string][]string{
	"region": {corev1.LabelTopologyRegion},
	"zone":   {corev1.LabelTopologyZone},
	"host":   {corev1.LabelHostname},
}

// GetNodeLabelsForKeys gets the labels of the node for the specified keys.
// This function is used to get the labels for setting store & server labels.
func GetNodeLabelsForKeys(node *corev1.Node, keys []string) map[string]string {
	labels := make(map[string]string)
	for _, key := range keys {
		if value, ok := node.Labels[key]; ok {
			labels[key] = value
			continue
		}
		if k8sLabels, ok := shortLabelNameToK8sLabel[key]; ok {
			for _, kl := range k8sLabels {
				if value, ok := node.Labels[kl]; ok {
					labels[key] = value
					break
				}
			}
		}
	}
	return labels
}
