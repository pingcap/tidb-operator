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

package tasks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TestPDMSProtection(t *testing.T) {
	ctx := context.Background()

	t.Run("replicas zero is clamped when PD involves ms", func(t *testing.T) {
		tg := fakeTSOGroupForPDMSProtection("tso", 0)
		pdg := fakePDGroupForPDMSProtection()
		fc := client.NewFakeClient(tg, pdg)

		desired, protected, err := effectiveReplicas(ctx, fc, tg)
		require.NoError(t, err)
		assert.True(t, protected)
		assert.Equal(t, int32(1), desired)
	})

	t.Run("condition is true only for destructive protected state", func(t *testing.T) {
		tg := fakeTSOGroupForPDMSProtection("tso", 0)
		pdg := fakePDGroupForPDMSProtection()
		state := &ReconcileContext{State: &state{tg: tg}}
		fc := client.NewFakeClient(tg, pdg)

		res, _ := task.RunTask(ctx, TaskPDMSProtection(state, fc))
		assert.Equal(t, task.SComplete.String(), res.Status().String())
		assert.True(t, state.PDMSProtected())
		require.Len(t, tg.Status.Conditions, 1)
		assert.Equal(t, v1alpha1.CondPDMSProtected, tg.Status.Conditions[0].Type)
		assert.Equal(t, metav1.ConditionTrue, tg.Status.Conditions[0].Status)
	})

	t.Run("deleting extra zero-replica TSOGroup is not protected", func(t *testing.T) {
		tg := fakeTSOGroupForPDMSProtection("tso-extra", 0)
		markDeleting(tg)
		otherTG := fakeTSOGroupForPDMSProtection("tso", 1)
		pdg := fakePDGroupForPDMSProtection()
		state := &ReconcileContext{State: &state{tg: tg}}
		fc := client.NewFakeClient(tg, otherTG, pdg)

		res, _ := task.RunTask(ctx, TaskPDMSProtection(state, fc))
		assert.Equal(t, task.SComplete.String(), res.Status().String())
		assert.False(t, state.PDMSProtected())
		require.Len(t, tg.Status.Conditions, 1)
		assert.Equal(t, v1alpha1.CondPDMSProtected, tg.Status.Conditions[0].Type)
		assert.Equal(t, metav1.ConditionFalse, tg.Status.Conditions[0].Status)
	})

	t.Run("concurrent TSOGroup deletes keep last non-deleting topology protected", func(t *testing.T) {
		tg := fakeTSOGroupForPDMSProtection("tso-0", 1)
		markDeleting(tg)
		otherTG := fakeTSOGroupForPDMSProtection("tso-1", 1)
		markDeleting(otherTG)
		pdg := fakePDGroupForPDMSProtection()
		state := &ReconcileContext{State: &state{tg: tg}}
		fc := client.NewFakeClient(tg, otherTG, pdg)

		res, _ := task.RunTask(ctx, TaskPDMSProtection(state, fc))
		assert.Equal(t, task.SComplete.String(), res.Status().String())
		assert.True(t, state.PDMSProtected())
		require.Len(t, tg.Status.Conditions, 1)
		assert.Equal(t, v1alpha1.CondPDMSProtected, tg.Status.Conditions[0].Type)
		assert.Equal(t, metav1.ConditionTrue, tg.Status.Conditions[0].Status)
	})

	t.Run("protected deletion asks runner to retry", func(t *testing.T) {
		res, _ := task.RunTask(ctx, TaskPDMSProtectedRetry())
		assert.Equal(t, task.SRetry.String(), res.Status().String())
		assert.NotZero(t, res.RequeueAfter())
	})
}

func fakeTSOGroupForPDMSProtection(name string, replicas int32) *v1alpha1.TSOGroup {
	return fake.FakeObj(name,
		fake.SetNamespace[v1alpha1.TSOGroup]("ns"),
		func(tg *v1alpha1.TSOGroup) *v1alpha1.TSOGroup {
			tg.Spec.Cluster.Name = "cluster"
			tg.Spec.Replicas = ptr.To(replicas)
			return tg
		})
}

func markDeleting(tg *v1alpha1.TSOGroup) {
	now := metav1.Now()
	tg.DeletionTimestamp = &now
	tg.Finalizers = []string{"test.finalizer"}
}

func fakePDGroupForPDMSProtection() *v1alpha1.PDGroup {
	return fake.FakeObj("pd",
		fake.SetNamespace[v1alpha1.PDGroup]("ns"),
		func(pdg *v1alpha1.PDGroup) *v1alpha1.PDGroup {
			pdg.Spec.Cluster.Name = "cluster"
			pdg.Spec.Replicas = ptr.To[int32](1)
			pdg.Spec.Template.Spec.Mode = v1alpha1.PDModeMS
			return pdg
		})
}
