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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	stateutil "github.com/pingcap/tidb-operator/v2/pkg/state"
	tsom "github.com/pingcap/tidb-operator/v2/pkg/timanager/tso"
	"github.com/pingcap/tidb-operator/v2/pkg/tsoapi"
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

		res, _ := task.RunTask(ctx, TaskModeSwitch(state, fc, healthyTSOClientGetter()))
		assert.Equal(t, task.SComplete.String(), res.Status().String())
		assert.False(t, state.ModeSwitchBlocked())
		cond := modeSwitchingCondition(t, pdg)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Equal(t, v1alpha1.ReasonModeSwitchNotNeeded, cond.Reason)
	})

	t.Run("normal to ms waits for single TSOGroup", func(t *testing.T) {
		pdg := fakePDGroupForModeSwitch()
		pdg.Status.Mode = v1alpha1.PDModeNormal
		cluster := fake.FakeObj("cluster", fake.SetNamespace[v1alpha1.Cluster]("ns"))
		pd := fakeSyncedPD(pdg, v1alpha1.PDModeNormal)
		state := newModeSwitchState(pdg, cluster, []*v1alpha1.PD{pd})
		fc := client.NewFakeClient(pdg, cluster, pd)

		res, _ := task.RunTask(ctx, TaskModeSwitch(state, fc, healthyTSOClientGetter()))
		assert.Equal(t, task.SRetry.String(), res.Status().String())
		assert.True(t, state.ModeSwitchBlocked())
		cond := modeSwitchingCondition(t, pdg)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		assert.Equal(t, v1alpha1.ReasonWaitingForSingleTSOGroup, cond.Reason)
	})

	t.Run("same blocked state does not mark status changed again", func(t *testing.T) {
		pdg := fakePDGroupForModeSwitch()
		pdg.Status.Mode = v1alpha1.PDModeNormal
		pdg.Status.Conditions = append(pdg.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.CondModeSwitching,
			Status:             metav1.ConditionTrue,
			Reason:             v1alpha1.ReasonWaitingForSingleTSOGroup,
			Message:            "waiting for exactly one TSOGroup, got 0",
			ObservedGeneration: pdg.Generation,
		})
		cluster := fake.FakeObj("cluster", fake.SetNamespace[v1alpha1.Cluster]("ns"))
		pd := fakeSyncedPD(pdg, v1alpha1.PDModeNormal)
		state := newModeSwitchState(pdg, cluster, []*v1alpha1.PD{pd})
		fc := client.NewFakeClient(pdg, cluster, pd)

		res, _ := task.RunTask(ctx, TaskModeSwitch(state, fc, healthyTSOClientGetter()))
		assert.Equal(t, task.SRetry.String(), res.Status().String())
		assert.True(t, state.ModeSwitchBlocked())
		assert.False(t, state.IsStatusChanged())
	})

	t.Run("normal to ms starts switching after TSOGroup is ready", func(t *testing.T) {
		pdg := fakePDGroupForModeSwitch()
		pdg.Status.Mode = v1alpha1.PDModeNormal
		cluster := fake.FakeObj("cluster", fake.SetNamespace[v1alpha1.Cluster]("ns"))
		pd := fakeSyncedPD(pdg, v1alpha1.PDModeNormal)
		tg := fakeReadyTSOGroup("tso", cluster, 1)
		state := newModeSwitchState(pdg, cluster, []*v1alpha1.PD{pd})
		fc := client.NewFakeClient(pdg, cluster, pd, tg)

		res, _ := task.RunTask(ctx, TaskModeSwitch(state, fc, healthyTSOClientGetter()))
		assert.Equal(t, task.SComplete.String(), res.Status().String())
		assert.False(t, state.ModeSwitchBlocked())
		cond := modeSwitchingCondition(t, pdg)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		assert.Equal(t, v1alpha1.ReasonSwitchingPDInstances, cond.Reason)
	})

	t.Run("normal to ms waits for TSO health API", func(t *testing.T) {
		pdg := fakePDGroupForModeSwitch()
		pdg.Status.Mode = v1alpha1.PDModeNormal
		cluster := fake.FakeObj("cluster", fake.SetNamespace[v1alpha1.Cluster]("ns"))
		pd := fakeSyncedPD(pdg, v1alpha1.PDModeNormal)
		tg := fakeReadyTSOGroup("tso", cluster, 1)
		state := newModeSwitchState(pdg, cluster, []*v1alpha1.PD{pd})
		fc := client.NewFakeClient(pdg, cluster, pd, tg)

		res, _ := task.RunTask(ctx, TaskModeSwitch(state, fc, failingTSOClientGetter(errors.New("connection refused"))))
		assert.Equal(t, task.SRetry.String(), res.Status().String())
		assert.True(t, state.ModeSwitchBlocked())
		cond := modeSwitchingCondition(t, pdg)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		assert.Equal(t, v1alpha1.ReasonWaitingForTSOGroupReady, cond.Reason)
		assert.Contains(t, cond.Message, "health API")
	})

	t.Run("normal to ms waits for TSO health check client", func(t *testing.T) {
		pdg := fakePDGroupForModeSwitch()
		pdg.Status.Mode = v1alpha1.PDModeNormal
		cluster := fake.FakeObj("cluster", fake.SetNamespace[v1alpha1.Cluster]("ns"))
		pd := fakeSyncedPD(pdg, v1alpha1.PDModeNormal)
		tg := fakeReadyTSOGroup("tso", cluster, 1)
		state := newModeSwitchState(pdg, cluster, []*v1alpha1.PD{pd})
		fc := client.NewFakeClient(pdg, cluster, pd, tg)

		res, _ := task.RunTask(ctx, TaskModeSwitch(state, fc, missingTSOClientGetter()))
		assert.Equal(t, task.SRetry.String(), res.Status().String())
		assert.True(t, state.ModeSwitchBlocked())
		cond := modeSwitchingCondition(t, pdg)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		assert.Equal(t, v1alpha1.ReasonWaitingForTSOGroupReady, cond.Reason)
		assert.Contains(t, cond.Message, "health check client")
	})

	t.Run("target changed back to normal while instance is still ms", func(t *testing.T) {
		pdg := fakePDGroupForModeSwitch()
		pdg.Spec.Template.Spec.Mode = v1alpha1.PDModeNormal
		pdg.Status.Mode = v1alpha1.PDModeNormal
		cluster := fake.FakeObj("cluster", fake.SetNamespace[v1alpha1.Cluster]("ns"))
		pd := fakeSyncedPD(pdg, v1alpha1.PDModeMS)
		state := newModeSwitchState(pdg, cluster, []*v1alpha1.PD{pd})
		fc := client.NewFakeClient(pdg, cluster, pd)

		res, _ := task.RunTask(ctx, TaskModeSwitch(state, fc, healthyTSOClientGetter()))
		assert.Equal(t, task.SComplete.String(), res.Status().String())
		assert.False(t, state.ModeSwitchBlocked())
		cond := modeSwitchingCondition(t, pdg)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		assert.Equal(t, v1alpha1.ReasonSwitchingPDInstances, cond.Reason)
	})

	t.Run("switching condition keeps target reversal active until instance is up to date", func(t *testing.T) {
		pdg := fakePDGroupForModeSwitch()
		pdg.Spec.Template.Spec.Mode = v1alpha1.PDModeNormal
		pdg.Status.Mode = v1alpha1.PDModeNormal
		pdg.Status.Conditions = append(pdg.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.CondModeSwitching,
			Status:             metav1.ConditionTrue,
			Reason:             v1alpha1.ReasonSwitchingPDInstances,
			Message:            "switching PD instances to mode \"\"",
			ObservedGeneration: pdg.Generation,
		})
		cluster := fake.FakeObj("cluster", fake.SetNamespace[v1alpha1.Cluster]("ns"))
		pd := fakeSyncedPD(pdg, v1alpha1.PDModeNormal)
		pd.Status.CurrentRevision = oldRevision
		state := newModeSwitchState(pdg, cluster, []*v1alpha1.PD{pd})
		fc := client.NewFakeClient(pdg, cluster, pd)

		res, _ := task.RunTask(ctx, TaskModeSwitch(state, fc, healthyTSOClientGetter()))
		assert.Equal(t, task.SComplete.String(), res.Status().String())
		assert.False(t, state.ModeSwitchBlocked())
		cond := modeSwitchingCondition(t, pdg)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		assert.Equal(t, v1alpha1.ReasonSwitchingPDInstances, cond.Reason)
	})

	t.Run("all instances at target completes mode switch", func(t *testing.T) {
		pdg := fakePDGroupForModeSwitch()
		pdg.Status.Mode = v1alpha1.PDModeNormal
		cluster := fake.FakeObj("cluster", fake.SetNamespace[v1alpha1.Cluster]("ns"))
		pd := fakeSyncedPD(pdg, v1alpha1.PDModeMS)
		rctx := newModeSwitchState(pdg, cluster, []*v1alpha1.PD{pd})
		rctx.State.(*state).updateRevision = newRevision
		fc := client.NewFakeClient(pdg, cluster, pd)

		res, _ := task.RunTask(ctx, TaskModeSwitch(rctx, fc, healthyTSOClientGetter()))
		assert.Equal(t, task.SComplete.String(), res.Status().String())
		assert.Equal(t, v1alpha1.PDModeMS, pdg.Status.Mode)
		cond := modeSwitchingCondition(t, pdg)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Equal(t, v1alpha1.ReasonModeSwitchComplete, cond.Reason)
	})
}

func modeSwitchingCondition(t *testing.T, pdg *v1alpha1.PDGroup) *metav1.Condition {
	t.Helper()
	cond := apimeta.FindStatusCondition(pdg.Status.Conditions, v1alpha1.CondModeSwitching)
	require.NotNil(t, cond)
	return cond
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

func fakeSyncedPD(pdg *v1alpha1.PDGroup, mode v1alpha1.PDMode) *v1alpha1.PD {
	pd := fakeAvailablePD("pd-0", pdg, newRevision)
	pd.Spec.Mode = mode
	pd.Status.Conditions = append(pd.Status.Conditions, metav1.Condition{
		Type:   v1alpha1.CondSynced,
		Status: metav1.ConditionTrue,
	})
	return pd
}

func fakeReadyTSOGroup(name string, cluster *v1alpha1.Cluster, replicas int32) *v1alpha1.TSOGroup {
	return fake.FakeObj(name,
		fake.SetNamespace[v1alpha1.TSOGroup](cluster.Namespace),
		func(tg *v1alpha1.TSOGroup) *v1alpha1.TSOGroup {
			tg.Spec.Cluster.Name = cluster.Name
			tg.Spec.Replicas = ptr.To(replicas)
			tg.Status.ObservedGeneration = tg.Generation
			tg.Status.CurrentRevision = newRevision
			tg.Status.UpdateRevision = newRevision
			tg.Status.Replicas = replicas
			tg.Status.ReadyReplicas = replicas
			tg.Status.CurrentReplicas = replicas
			tg.Status.UpdatedReplicas = replicas
			return tg
		})
}

type fakeTSOClientGetter struct {
	client tsom.TSOClient
	ok     bool
}

func (f fakeTSOClientGetter) Get(string) (tsom.TSOClient, bool) {
	return f.client, f.ok
}

type fakeTSOClient struct {
	underlay tsoapi.TSOClient
}

func (f fakeTSOClient) Underlay() tsoapi.TSOClient {
	return f.underlay
}

type fakeTSOUnderlay struct {
	healthy bool
	err     error
}

func (f fakeTSOUnderlay) IsHealthy(context.Context) (bool, error) {
	return f.healthy, f.err
}

func (f fakeTSOUnderlay) TransferTSOLeader(context.Context, string) error {
	return nil
}

func healthyTSOClientGetter() fakeTSOClientGetter {
	return fakeTSOClientGetter{
		client: fakeTSOClient{underlay: fakeTSOUnderlay{healthy: true}},
		ok:     true,
	}
}

func failingTSOClientGetter(err error) fakeTSOClientGetter {
	return fakeTSOClientGetter{
		client: fakeTSOClient{underlay: fakeTSOUnderlay{err: err}},
		ok:     true,
	}
}

func missingTSOClientGetter() fakeTSOClientGetter {
	return fakeTSOClientGetter{}
}
