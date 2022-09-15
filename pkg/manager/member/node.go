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
