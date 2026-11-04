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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestTiProxyGroupReplicasManagedByHPA(t *testing.T) {
	t.Parallel()

	older := metav1.NewTime(time.Now().Add(-time.Hour))
	newer := metav1.NewTime(time.Now())
	replicas := &metav1.FieldsV1{Raw: []byte(`{"f:spec":{"f:replicas":{}}}`)}
	statusOnly := &metav1.FieldsV1{Raw: []byte(`{"f:status":{"f:replicas":{}}}`)}

	tests := []struct {
		name    string
		entries []metav1.ManagedFieldsEntry
		want    bool
	}{
		{
			name: "newest replicas writer is the scale subresource of kube-controller-manager",
			entries: []metav1.ManagedFieldsEntry{
				replicasEntry("kubectl-client-side-apply", "", older, replicas),
				replicasEntry(HPAFieldManager, "scale", newer, replicas),
			},
			want: true,
		},
		{
			name: "a later main-resource write replaces a stale scale owner",
			entries: []metav1.ManagedFieldsEntry{
				replicasEntry(HPAFieldManager, "scale", older, replicas),
				replicasEntry("kubectl-client-side-apply", "", newer, replicas),
			},
		},
		{
			name: "kube-controller-manager outside the scale subresource is not HPA",
			entries: []metav1.ManagedFieldsEntry{
				replicasEntry(HPAFieldManager, "scale", older, replicas),
				replicasEntry(HPAFieldManager, "status", newer, replicas),
			},
		},
		{
			name: "a newer entry that does not own replicas is ignored",
			entries: []metav1.ManagedFieldsEntry{
				replicasEntry(HPAFieldManager, "scale", older, replicas),
				replicasEntry("kubectl-client-side-apply", "", newer, statusOnly),
			},
			want: true,
		},
		{
			name: "a missing time is older than a recorded time",
			entries: []metav1.ManagedFieldsEntry{
				replicasEntry(HPAFieldManager, "scale", metav1.Time{}, replicas),
				replicasEntry("kubectl-scale", "scale", newer, replicas),
			},
		},
		{
			name: "the only replicas writer can omit time",
			entries: []metav1.ManagedFieldsEntry{
				replicasEntry(HPAFieldManager, "scale", metav1.Time{}, replicas),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			proxyg := &v1alpha1.TiProxyGroup{}
			proxyg.ManagedFields = tt.entries
			assert.Equal(t, tt.want, TiProxyGroupReplicasManagedByHPA(proxyg))
		})
	}

	assert.False(t, TiProxyGroupReplicasManagedByHPA(nil))
}

func replicasEntry(manager, subresource string, at metav1.Time, fields *metav1.FieldsV1) metav1.ManagedFieldsEntry {
	entry := metav1.ManagedFieldsEntry{
		Manager:     manager,
		Operation:   metav1.ManagedFieldsOperationUpdate,
		Subresource: subresource,
		FieldsV1:    fields,
	}
	if !at.IsZero() {
		entry.Time = &at
	}
	return entry
}
