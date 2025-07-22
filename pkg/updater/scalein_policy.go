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

package updater

import (
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

// PreferAnnotatedForDeletion returns a policy that prefers instances marked with offline store annotation.
// This enables users to specify which instances should be selected for deletion during scale-in operations.
func PreferAnnotatedForDeletion[SI runtime.StoreInstance]() PreferPolicy[SI] {
	return PreferPolicyFunc[SI](func(instances []SI) []SI {
		var annotated []SI
		for _, instance := range instances {
			annotations := instance.GetAnnotations()
			if annotations != nil {
				if val, exists := annotations[v1alpha1.AnnoKeyOfflineStore]; exists && val == "true" {
					annotated = append(annotated, instance)
				}
			}
		}
		return annotated
	})
}

// PreferNotOffline returns a policy that prefers instances that are not currently in offline process.
// This prevents selecting instances that are already being offlined for a different scale-in operation.
func PreferNotOffline[SI runtime.StoreInstance]() PreferPolicy[SI] {
	return PreferPolicyFunc[SI](func(instances []SI) []SI {
		var notOffline []SI
		for _, instance := range instances {
			// Check if the instance has `spec.offline` set to true
			if !instance.IsOffline() {
				notOffline = append(notOffline, instance)
			}
		}
		return notOffline
	})
}
