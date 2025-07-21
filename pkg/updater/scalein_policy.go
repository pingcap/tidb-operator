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

// PreferAnnotatedForDeletion returns a policy that prefers instances marked with scale-in deletion annotation.
// This enables users to specify which instances should be selected for deletion during scale-in operations.
func PreferAnnotatedForDeletion[R runtime.Instance]() PreferPolicy[R] {
	return PreferPolicyFunc[R](func(instances []R) []R {
		var annotated []R
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

// PreferNotOfflining returns a policy that prefers instances that are not currently in offline process.
// This prevents selecting instances that are already being offlined for a different scale-in operation.
func PreferNotOfflining[R runtime.Instance]() PreferPolicy[R] {
	return PreferPolicyFunc[R](func(instances []R) []R {
		var notOfflining []R
		for _, instance := range instances {
			// Check if the instance has spec.offline set to true
			// We need to cast to the specific runtime type to access the Spec field
			switch inst := any(instance).(type) {
			case *runtime.TiKV:
				if !inst.Spec.Offline {
					notOfflining = append(notOfflining, instance)
				}
			case *runtime.TiFlash:
				if !inst.Spec.Offline {
					notOfflining = append(notOfflining, instance)
				}
			default:
				// For other types, include them as they don't support offline operations
				notOfflining = append(notOfflining, instance)
			}
		}
		return notOfflining
	})
}

// PreferHealthyForScaleIn returns a policy that prefers healthy instances for scale-in.
// This ensures that unhealthy instances are kept to maintain cluster stability.
func PreferHealthyForScaleIn[R runtime.Instance]() PreferPolicy[R] {
	return PreferPolicyFunc[R](func(instances []R) []R {
		var healthy []R
		for _, instance := range instances {
			// Consider instance healthy if it's both ready and up-to-date
			if instance.IsReady() && instance.IsUpToDate() {
				healthy = append(healthy, instance)
			}
		}
		return healthy
	})
}
