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
	dmOldRevision = "old"
	dmNewRevision = "new"
)

func TestTaskUpdater(t *testing.T) {
	cases := []struct {
		desc          string
		state         *ReconcileContext
		unexpectedErr bool
		wantStatus    task.Status
		wantDMs       int
	}{
		{
			desc: "scale up 0 to 3",
			state: newDMGroupReconcileContext(newTestDMGroup("aaa", func(dmg *v1alpha1.DMGroup) {
				dmg.Spec.Replicas = ptr.To[int32](3)
			}), nil, dmNewRevision),
			wantStatus: task.SWait,
			wantDMs:    3,
		},
		{
			desc: "scale down 3 to 1",
			state: newDMGroupReconcileContext(newTestDMGroup("aaa", func(dmg *v1alpha1.DMGroup) {
				dmg.Spec.Replicas = ptr.To[int32](1)
			}), []*v1alpha1.DM{
				fakeAvailableDM("aaa-0", newTestDMGroup("aaa"), dmNewRevision),
				fakeAvailableDM("aaa-1", newTestDMGroup("aaa"), dmNewRevision),
				fakeAvailableDM("aaa-2", newTestDMGroup("aaa"), dmNewRevision),
			}, dmNewRevision),
			wantStatus: task.SComplete,
			wantDMs:    1,
		},
		{
			desc: "rolling update waits after one unavailable change",
			state: newDMGroupReconcileContext(newTestDMGroup("aaa", func(dmg *v1alpha1.DMGroup) {
				dmg.Spec.Replicas = ptr.To[int32](2)
			}), []*v1alpha1.DM{
				fakeAvailableDM("aaa-0", newTestDMGroup("aaa"), dmOldRevision),
				fakeAvailableDM("aaa-1", newTestDMGroup("aaa"), dmOldRevision),
			}, dmNewRevision),
			wantStatus: task.SWait,
			wantDMs:    2,
		},
		{
			desc: "no change needed",
			state: newDMGroupReconcileContext(newTestDMGroup("aaa"), []*v1alpha1.DM{
				fakeAvailableDM("aaa-0", newTestDMGroup("aaa"), dmNewRevision),
			}, dmNewRevision),
			wantStatus: task.SComplete,
			wantDMs:    1,
		},
		{
			desc: "allocate fails",
			state: newDMGroupReconcileContext(newTestDMGroup("aaa", func(dmg *v1alpha1.DMGroup) {
				dmg.Spec.Replicas = ptr.To[int32](2)
			}), nil, dmNewRevision),
			unexpectedErr: true,
			wantStatus:    task.SFail,
		},
		{
			desc: "version upgrade detected",
			state: newDMGroupReconcileContext(newTestDMGroup("aaa", func(dmg *v1alpha1.DMGroup) {
				dmg.Spec.Template.Spec.Version = "v8.5.3"
				dmg.Status.Version = "v8.5.2"
			}), nil, dmNewRevision),
			wantStatus: task.SWait,
			wantDMs:    1,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(t *testing.T) {
			t.Parallel()

			s := c.state.State.(*state)
			s.IFeatureGates = stateutil.NewFeatureGates[scope.DMGroup](s)
			objs := []client.Object{c.state.DMGroup(), c.state.Cluster()}
			fc := client.NewFakeClient(objs...)
			for _, dm := range c.state.DMSlice() {
				require.NoError(t, fc.Apply(context.Background(), dm.DeepCopy()))
				require.NoError(t, fc.Status().Update(context.Background(), dm.DeepCopy()))
			}
			if c.unexpectedErr {
				fc.WithError("patch", "dms", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			res, done := task.RunTask(context.Background(), TaskUpdater(c.state, fc, tracker.New().AllocateFactory("dm")))
			assert.Equal(t, c.wantStatus.String(), res.Status().String(), res.Message())
			assert.False(t, done)
			if c.unexpectedErr {
				return
			}

			dms := v1alpha1.DMList{}
			require.NoError(t, fc.List(context.Background(), &dms))
			assert.Len(t, dms.Items, c.wantDMs)
		})
	}
}

func newTestDMGroup(name string, opts ...func(*v1alpha1.DMGroup)) *v1alpha1.DMGroup { //nolint:unparam
	dmg := fake.FakeObj(name, func(obj *v1alpha1.DMGroup) *v1alpha1.DMGroup {
		obj.Spec.Cluster.Name = "cluster"
		obj.Spec.Template.Spec.Version = "v8.5.2"
		obj.Spec.Template.Spec.DataVolume.Name = "data"
		return obj
	})
	for _, opt := range opts {
		opt(dmg)
	}
	return dmg
}

func newDMGroupReconcileContext(dmg *v1alpha1.DMGroup, dms []*v1alpha1.DM, rev string) *ReconcileContext { //nolint:unparam
	return &ReconcileContext{State: &state{
		dmg:            dmg,
		cluster:        fake.FakeObj[v1alpha1.Cluster]("cluster"),
		dms:            dms,
		updateRevision: rev,
	}}
}

func fakeAvailableDM(name string, dmg *v1alpha1.DMGroup, rev string) *v1alpha1.DM {
	return fake.FakeObj(name, func(obj *v1alpha1.DM) *v1alpha1.DM {
		dm := runtime.ToDM(DMNewer(dmg, rev, nil).New())
		dm.Name = name
		dm.Labels[v1alpha1.LabelKeyInstanceRevisionHash] = rev
		dm.Status.Conditions = []metav1.Condition{{
			Type:               v1alpha1.CondReady,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Unix(1, 0),
		}}
		dm.Status.CurrentRevision = rev
		dm.DeepCopyInto(obj)
		return obj
	})
}
