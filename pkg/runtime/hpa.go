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

package runtime

import (
	"encoding/json"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

// HPAFieldManager is the User-Agent recorded when the in-tree HPA controller
// writes a scale subresource. It runs inside kube-controller-manager.
const HPAFieldManager = "kube-controller-manager"

// TiProxyGroupReplicasManagedByHPA reports whether the newest managedFields
// entry that owns spec.replicas was written by the in-tree HPA controller.
// The main resource and the scale subresource keep separate entries, so an
// older scale entry does not count after a later writer updates replicas.
// An entry with no time sorts before any recorded time.
func TiProxyGroupReplicasManagedByHPA(proxyg *v1alpha1.TiProxyGroup) bool {
	if proxyg == nil {
		return false
	}
	var latest metav1.ManagedFieldsEntry
	var found bool
	for _, entry := range proxyg.ManagedFields {
		if !managedFieldsOwnReplicas(entry) {
			continue
		}
		if !found || managedFieldTime(entry).After(managedFieldTime(latest)) {
			latest = entry
			found = true
		}
	}
	return found && latest.Manager == HPAFieldManager && latest.Subresource == "scale"
}

func managedFieldTime(entry metav1.ManagedFieldsEntry) time.Time {
	if entry.Time == nil {
		return time.Time{}
	}
	return entry.Time.Time
}

func managedFieldsOwnReplicas(entry metav1.ManagedFieldsEntry) bool {
	if entry.FieldsV1 == nil {
		return false
	}
	var fields map[string]any
	if err := json.Unmarshal(entry.FieldsV1.Raw, &fields); err != nil {
		return false
	}
	spec, ok := fields["f:spec"].(map[string]any)
	if !ok {
		return false
	}
	_, ok = spec["f:replicas"]
	return ok
}
