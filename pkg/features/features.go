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

package features

import meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"

// ClusterFeatures returns the effective list to propagate to Groups/Instances.
// Explicit entries retain spec order; default-only entries follow log order.
func ClusterFeatures(fs []meta.FeatureGate) []meta.Feature {
	configured := NewFromCluster(fs)
	enabled := make([]meta.Feature, 0, len(fs))
	seen := make(map[meta.Feature]bool)
	appendEnabled := func(name meta.Feature) {
		if !seen[name] && configured.Enabled(name) {
			enabled = append(enabled, name)
			seen[name] = true
		}
	}
	for _, feature := range fs {
		appendEnabled(feature.Name)
	}
	for _, feature := range logs[len(logs)-1] {
		appendEnabled(feature.Name)
	}
	return enabled
}
