// Copyright 2026 PingCAP, Inc.
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
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
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
