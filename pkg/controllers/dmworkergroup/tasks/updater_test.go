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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	stateutil "github.com/pingcap/tidb-operator/v2/pkg/state"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/tracker"
)

const (
	dwOldRevision = "old"
	dwNewRevision = "new"
)

func TestTaskUpdater(t *testing.T) {
	cases := []struct {
		desc          string
		state         *ReconcileContext
		unexpectedErr bool
		wantStatus    task.Status
		wantDWs       int
	}{
		{
			desc: "scale up 0 to 3",
			state: newDMWorkerGroupReconcileContext(newTestDMWorkerGroup("aaa", func(dwg *v1alpha1.DMWorkerGroup) {
				dwg.Spec.Replicas = ptr.To[int32](3)
			}), nil, dwNewRevision),
			wantStatus: task.SWait,
			wantDWs:    3,
		},
		{
			desc: "scale down 3 to 1",
			state: newDMWorkerGroupReconcileContext(newTestDMWorkerGroup("aaa", func(dwg *v1alpha1.DMWorkerGroup) {
				dwg.Spec.Replicas = ptr.To[int32](1)
			}), []*v1alpha1.DMWorker{
				fakeAvailableDMWorker("aaa-0", newTestDMWorkerGroup("aaa"), dwNewRevision),
				fakeAvailableDMWorker("aaa-1", newTestDMWorkerGroup("aaa"), dwNewRevision),
				fakeAvailableDMWorker("aaa-2", newTestDMWorkerGroup("aaa"), dwNewRevision),
			}, dwNewRevision),
			wantStatus: task.SComplete,
			wantDWs:    1,
		},
		{
			desc: "rolling update respects maxSurge zero",
			state: newDMWorkerGroupReconcileContext(newTestDMWorkerGroup("aaa", func(dwg *v1alpha1.DMWorkerGroup) {
				dwg.Spec.Replicas = ptr.To[int32](2)
			}), []*v1alpha1.DMWorker{
				fakeAvailableDMWorker("aaa-0", newTestDMWorkerGroup("aaa"), dwOldRevision),
				fakeAvailableDMWorker("aaa-1", newTestDMWorkerGroup("aaa"), dwOldRevision),
			}, dwNewRevision),
			wantStatus: task.SWait,
			wantDWs:    2,
		},
		{
			desc: "no change needed",
			state: newDMWorkerGroupReconcileContext(newTestDMWorkerGroup("aaa"), []*v1alpha1.DMWorker{
				fakeAvailableDMWorker("aaa-0", newTestDMWorkerGroup("aaa"), dwNewRevision),
			}, dwNewRevision),
			wantStatus: task.SComplete,
			wantDWs:    1,
		},
		{
			desc: "allocate fails",
			state: newDMWorkerGroupReconcileContext(newTestDMWorkerGroup("aaa", func(dwg *v1alpha1.DMWorkerGroup) {
				dwg.Spec.Replicas = ptr.To[int32](2)
			}), nil, dwNewRevision),
			unexpectedErr: true,
			wantStatus:    task.SFail,
		},
		{
			desc: "version upgrade detected",
			state: newDMWorkerGroupReconcileContext(newTestDMWorkerGroup("aaa", func(dwg *v1alpha1.DMWorkerGroup) {
				dwg.Spec.Template.Spec.Version = "v8.5.3"
				dwg.Status.Version = "v8.5.2"
			}), nil, dwNewRevision),
			wantStatus: task.SRetry,
			wantDWs:    0,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(t *testing.T) {
			t.Parallel()

			s := c.state.State.(*state)
			s.IFeatureGates = stateutil.NewFeatureGates[scope.DMWorkerGroup](s)
			fc := client.NewFakeClient(c.state.DMWorkerGroup(), c.state.Cluster())
			for _, dw := range c.state.DMWorkerSlice() {
				require.NoError(t, fc.Apply(context.Background(), dw.DeepCopy()))
				require.NoError(t, fc.Status().Update(context.Background(), dw.DeepCopy()))
			}
			if c.unexpectedErr {
				fc.WithError("patch", "dmworkers", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			res, done := task.RunTask(context.Background(), TaskUpdater(c.state, fc, tracker.New().AllocateFactory("dmworker")))
			assert.Equal(t, c.wantStatus.String(), res.Status().String(), res.Message())
			assert.False(t, done)
			if c.unexpectedErr {
				return
			}

			dws := v1alpha1.DMWorkerList{}
			require.NoError(t, fc.List(context.Background(), &dws))
			assert.Len(t, dws.Items, c.wantDWs)
		})
	}
}

func newTestDMWorkerGroup(name string, opts ...func(*v1alpha1.DMWorkerGroup)) *v1alpha1.DMWorkerGroup { //nolint:unparam
	dwg := fake.FakeObj(name, func(obj *v1alpha1.DMWorkerGroup) *v1alpha1.DMWorkerGroup {
		obj.Spec.Cluster.Name = "cluster"
		obj.Spec.DMGroupRef.Name = "dmg"
		obj.Spec.Template.Spec.Version = "v8.5.2"
		obj.Spec.Template.Spec.RelayVolume = &v1alpha1.Volume{Name: "relay"}
		return obj
	})
	for _, opt := range opts {
		opt(dwg)
	}
	return dwg
}

func newDMWorkerGroupReconcileContext(dwg *v1alpha1.DMWorkerGroup, dws []*v1alpha1.DMWorker, rev string) *ReconcileContext { //nolint:unparam
	return &ReconcileContext{State: &state{
		dwg:            dwg,
		cluster:        fake.FakeObj[v1alpha1.Cluster]("cluster"),
		dws:            dws,
		updateRevision: rev,
	}}
}

func fakeAvailableDMWorker(name string, dwg *v1alpha1.DMWorkerGroup, rev string) *v1alpha1.DMWorker {
	return fake.FakeObj(name, func(obj *v1alpha1.DMWorker) *v1alpha1.DMWorker {
		dw := runtime.ToDMWorker(DMWorkerNewer(dwg, rev).New())
		dw.Name = name
		dw.Labels[v1alpha1.LabelKeyInstanceRevisionHash] = rev
		dw.Status.Conditions = []metav1.Condition{{
			Type:               v1alpha1.CondReady,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Unix(1, 0),
		}}
		dw.Status.CurrentRevision = rev
		dw.DeepCopyInto(obj)
		return obj
	})
}
