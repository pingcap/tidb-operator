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

package tiflash

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/tracker"
)

func TestSuspendedWithoutPDClient(t *testing.T) {
	cluster := &v1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default", Generation: 1}}
	cluster.Spec.SuspendAction = &v1alpha1.SuspendAction{SuspendCompute: true}
	cluster.Status.PD = "http://pd:2379"
	instance := &v1alpha1.TiFlash{ObjectMeta: metav1.ObjectMeta{Name: "instance", Namespace: "default", Generation: 1}}
	instance.OwnerReferences = []metav1.OwnerReference{{APIVersion: v1alpha1.SchemeGroupVersion.String(), Kind: "TiFlashGroup", Name: "group", Controller: ptr.To(true)}}
	instance.Spec.Cluster.Name = cluster.Name
	instance.Spec.Offline = ptr.To(true)
	offline := metav1.Condition{
		Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionFalse,
		Reason: v1alpha1.ReasonOfflineProcessing, ObservedGeneration: 1,
	}
	instance.Status.Conditions = []metav1.Condition{offline}
	instance.Status.State = v1alpha1.StoreStateRemoving
	c := client.NewFakeClient(cluster, instance)
	r := &Reconciler{Client: c, PDClientManager: pdm.NewPDClientManager(logr.Discard(), c), Tracker: tracker.New().Tracker("tiflash")}
	key := client.ObjectKeyFromObject(instance)
	// A stopped client is absent from the manager. Reconcile twice to also catch
	// deletion driven by an incorrectly reported Offlined condition.
	for range 2 {
		result, err := r.Reconcile(t.Context(), ctrl.Request{NamespacedName: key})
		require.NoError(t, err)
		assert.Zero(t, result.RequeueAfter, "completed suspension must not retry for a missing client")
		assert.False(t, result.Requeue)
		var got v1alpha1.TiFlash
		require.NoError(t, c.Get(t.Context(), key, &got))
		assert.True(t, got.DeletionTimestamp.IsZero(), "missing PD state must not cause Instance deletion")
		assert.Equal(t, &offline, meta.FindStatusCondition(got.Status.Conditions, v1alpha1.StoreOfflinedConditionType))
		assert.True(t, meta.IsStatusConditionTrue(got.Status.Conditions, v1alpha1.CondSuspended))
	}
}
