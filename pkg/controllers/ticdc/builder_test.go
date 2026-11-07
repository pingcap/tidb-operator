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

package ticdc

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/tracker"
)

func TestLegacyInvalidInstanceDeletion(t *testing.T) {
	for _, clusterDeleting := range []bool{false, true} {
		name := "instance deletion"
		if clusterDeleting {
			name = "cluster deletion"
		}
		t.Run(name, func(t *testing.T) {
			now := metav1.Now()
			cluster := &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster", Namespace: "test"}}
			cluster.Status.PD = "http://pd:2379"
			instance := &v1alpha1.TiCDC{ObjectMeta: metav1.ObjectMeta{Name: "invalid", Namespace: "test", Finalizers: []string{metav1alpha1.Finalizer}}}
			instance.Spec.Cluster.Name = cluster.Name
			if clusterDeleting {
				cluster.DeletionTimestamp = &now
				cluster.Finalizers = []string{metav1alpha1.Finalizer}
			} else {
				instance.DeletionTimestamp = &now
			}
			c := client.NewFakeClient(cluster, instance)
			r := &Reconciler{Client: c, Tracker: tracker.New().Tracker("ticdc")}
			key := client.ObjectKeyFromObject(instance)
			_, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: key})
			require.NoError(t, err)
			var got v1alpha1.TiCDC
			err = c.Get(t.Context(), key, &got)
			if !errors.IsNotFound(err) {
				require.NoError(t, err)
				require.NotContains(t, got.Finalizers, metav1alpha1.Finalizer)
			}
		})
	}
}
