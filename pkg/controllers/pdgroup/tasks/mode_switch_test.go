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
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	stateutil "github.com/pingcap/tidb-operator/v2/pkg/state"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TestTaskModeSwitch(t *testing.T) {
	ctx := context.Background()

	t.Run("initial ms create is not treated as mode switch", func(t *testing.T) {
		pdg := fakePDGroupForModeSwitch()
		cluster := fake.FakeObj("cluster", fake.SetNamespace[v1alpha1.Cluster]("ns"))
		state := newModeSwitchState(pdg, cluster, nil)
		fc := client.NewFakeClient(pdg, cluster)

		res, _ := task.RunTask(ctx, TaskModeSwitch(state, fc))
		assert.Equal(t, task.SComplete.String(), res.Status().String())
		assert.False(t, state.ModeSwitchBlocked())
		assert.Nil(t, pdg.Status.ModeTransition)
	})

	t.Run("normal to ms waits for single TSOGroup", func(t *testing.T) {
		pdg := fakePDGroupForModeSwitch()
		pdg.Status.Mode = v1alpha1.PDModeNormal
		cluster := fake.FakeObj("cluster", fake.SetNamespace[v1alpha1.Cluster]("ns"))
		pd := fakeSyncedPD("pd-0", pdg, v1alpha1.PDModeNormal)
		state := newModeSwitchState(pdg, cluster, []*v1alpha1.PD{pd})
		fc := client.NewFakeClient(pdg, cluster, pd)

		res, _ := task.RunTask(ctx, TaskModeSwitch(state, fc))
		assert.Equal(t, task.SRetry.String(), res.Status().String())
		assert.True(t, state.ModeSwitchBlocked())
		require.NotNil(t, pdg.Status.ModeTransition)
		assert.Equal(t, v1alpha1.ReasonWaitingForSingleTSOGroup, pdg.Status.ModeTransition.Reason)
	})

	t.Run("same blocked state does not mark status changed again", func(t *testing.T) {
		pdg := fakePDGroupForModeSwitch()
		pdg.Status.Mode = v1alpha1.PDModeNormal
		pdg.Status.ModeTransition = &v1alpha1.PDModeTransition{
			Phase:              modeTransitionPhasePreparing,
			ObservedGeneration: pdg.Generation,
			Reason:             v1alpha1.ReasonWaitingForSingleTSOGroup,
			Message:            "waiting for exactly one TSOGroup, got 0",
		}
		pdg.Status.Conditions = append(pdg.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.CondModeSwitching,
			Status:             metav1.ConditionTrue,
			Reason:             v1alpha1.ReasonWaitingForSingleTSOGroup,
			Message:            "waiting for exactly one TSOGroup, got 0",
			ObservedGeneration: pdg.Generation,
		})
		cluster := fake.FakeObj("cluster", fake.SetNamespace[v1alpha1.Cluster]("ns"))
		pd := fakeSyncedPD("pd-0", pdg, v1alpha1.PDModeNormal)
		state := newModeSwitchState(pdg, cluster, []*v1alpha1.PD{pd})
		fc := client.NewFakeClient(pdg, cluster, pd)

		res, _ := task.RunTask(ctx, TaskModeSwitch(state, fc))
		assert.Equal(t, task.SRetry.String(), res.Status().String())
		assert.True(t, state.ModeSwitchBlocked())
		assert.False(t, state.IsStatusChanged())
	})

	t.Run("all instances at target completes mode switch", func(t *testing.T) {
		pdg := fakePDGroupForModeSwitch()
		pdg.Status.Mode = v1alpha1.PDModeNormal
		cluster := fake.FakeObj("cluster", fake.SetNamespace[v1alpha1.Cluster]("ns"))
		pd := fakeSyncedPD("pd-0", pdg, v1alpha1.PDModeMS)
		rctx := newModeSwitchState(pdg, cluster, []*v1alpha1.PD{pd})
		rctx.State.(*state).updateRevision = newRevision
		fc := client.NewFakeClient(pdg, cluster, pd)

		res, _ := task.RunTask(ctx, TaskModeSwitch(rctx, fc))
		assert.Equal(t, task.SComplete.String(), res.Status().String())
		assert.Equal(t, v1alpha1.PDModeMS, pdg.Status.Mode)
		assert.Nil(t, pdg.Status.ModeTransition)
	})
}

func newModeSwitchState(pdg *v1alpha1.PDGroup, cluster *v1alpha1.Cluster, pds []*v1alpha1.PD) *ReconcileContext {
	s := &state{
		pdg:            pdg,
		cluster:        cluster,
		pds:            pds,
		updateRevision: newRevision,
	}
	s.IFeatureGates = stateutil.NewFeatureGates[scope.PDGroup](s)
	return &ReconcileContext{State: s}
}

func fakePDGroupForModeSwitch() *v1alpha1.PDGroup {
	return fake.FakeObj("pd",
		fake.SetNamespace[v1alpha1.PDGroup]("ns"),
		func(pdg *v1alpha1.PDGroup) *v1alpha1.PDGroup {
			pdg.Spec.Cluster.Name = "cluster"
			pdg.Spec.Replicas = ptr.To[int32](1)
			pdg.Spec.Template.Spec.Version = "v8.3.0"
			pdg.Spec.Template.Spec.Mode = v1alpha1.PDModeMS
			return pdg
		})
}

func fakeSyncedPD(name string, pdg *v1alpha1.PDGroup, mode v1alpha1.PDMode) *v1alpha1.PD {
	pd := fakeAvailablePD(name, pdg, newRevision)
	pd.Spec.Mode = mode
	pd.Status.Conditions = append(pd.Status.Conditions, metav1.Condition{
		Type:   v1alpha1.CondSynced,
		Status: metav1.ConditionTrue,
	})
	return pd
}
