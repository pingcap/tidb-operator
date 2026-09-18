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

package cluster

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/features"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
)

func TestDMGroupEventsEnqueueCluster(t *testing.T) {
	expected := []reconcile.Request{{NamespacedName: types.NamespacedName{
		Namespace: "test-ns",
		Name:      "test-cluster",
	}}}

	dmGroup := &v1alpha1.DMGroup{
		ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns"},
		Spec: v1alpha1.DMGroupSpec{Cluster: v1alpha1.ClusterReference{
			Name: "test-cluster",
		}},
	}
	dmWorkerGroup := &v1alpha1.DMWorkerGroup{
		ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns"},
		Spec: v1alpha1.DMWorkerGroupSpec{Cluster: v1alpha1.ClusterReference{
			Name: "test-cluster",
		}},
	}

	assert.Equal(t, expected, enqueueForGroupFunc[scope.DMGroup]()(context.Background(), dmGroup))
	assert.Equal(t, expected, enqueueForGroupFunc[scope.DMWorkerGroup]()(context.Background(), dmWorkerGroup))
}

func TestFeatureGateHashBeforeClusterChanges(t *testing.T) {
	for _, fail := range []bool{false, true} {
		name := "persisted"
		if fail {
			name = "write fails"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			cluster := &v1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
				Spec:       v1alpha1.ClusterSpec{FeatureGates: []meta.FeatureGate{{Name: meta.ClusterSubdomain}}},
				Status:     v1alpha1.ClusterStatus{ID: "1"},
			}
			fc := client.NewFakeClient(cluster)
			if fail {
				fc.WithError("update", "clusters", errors.New("status unavailable"))
			}
			reconciler := &Reconciler{Client: fc}
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(cluster)})
			if fail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			stored := &v1alpha1.Cluster{}
			require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(cluster), stored))
			services := &corev1.ServiceList{}
			require.NoError(t, fc.List(ctx, services))
			if fail {
				assert.Empty(t, stored.Status.FeatureGateHash)
				assert.Empty(t, stored.Finalizers)
				assert.Empty(t, services.Items)
			} else {
				assert.Equal(t, features.CurrentFeatureGateDefinitionHash, stored.Status.FeatureGateHash)
				assert.NotEmpty(t, stored.Finalizers)
				assert.Len(t, services.Items, 1)
			}
		})
	}
}
