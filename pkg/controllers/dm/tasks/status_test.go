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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TestTaskStatus(t *testing.T) {
	now := metav1.Now()
	cases := []struct {
		desc          string
		ctx           *ReconcileContext
		unexpectedErr bool
		wantStatus    task.Status
		wantReady     metav1.ConditionStatus
	}{
		{
			desc:       "pod not ready",
			ctx:        newDMReconcileContext(newStatusDM(), fake.FakeObj[v1alpha1.Cluster]("cluster"), nil),
			wantStatus: task.SRetry,
			wantReady:  metav1.ConditionFalse,
		},
		{
			desc: "pod ready",
			ctx: newDMReconcileContext(newStatusDM(func(dm *v1alpha1.DM) {
				dm.Status.Conditions = []metav1.Condition{{Type: v1alpha1.CondReady, Status: metav1.ConditionTrue, ObservedGeneration: 3}}
			}), fake.FakeObj[v1alpha1.Cluster]("cluster"), readyDMPod()),
			wantStatus: task.SComplete,
			wantReady:  metav1.ConditionTrue,
		},
		{
			desc: "pod terminating",
			ctx: newDMReconcileContext(newStatusDM(), fake.FakeObj[v1alpha1.Cluster]("cluster"), fake.FakeObj("aaa-dm-master-0", func(obj *corev1.Pod) *corev1.Pod {
				obj.SetDeletionTimestamp(&now)
				return obj
			})),
			wantStatus: task.SRetry,
			wantReady:  metav1.ConditionFalse,
		},
		{
			desc:          "status update fails",
			ctx:           newDMReconcileContext(newStatusDM(), fake.FakeObj[v1alpha1.Cluster]("cluster"), readyDMPod()),
			unexpectedErr: true,
			wantStatus:    task.SFail,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(t *testing.T) {
			t.Parallel()

			fc := client.NewFakeClient(c.ctx.DM())
			if c.ctx.Pod() != nil {
				require.NoError(t, fc.Apply(context.Background(), c.ctx.Pod()))
			}
			if c.ctx.Pod() != nil && !c.ctx.Pod().DeletionTimestamp.IsZero() {
				c.ctx.SetPod(c.ctx.Pod())
			}
			if c.unexpectedErr {
				fc.WithError("*", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			res, done := task.RunTask(context.Background(), TaskStatus(c.ctx, fc))
			assert.Equal(t, c.wantStatus.String(), res.Status().String(), res.Message())
			assert.False(t, done)
			if c.unexpectedErr {
				return
			}

			dm := &v1alpha1.DM{}
			require.NoError(t, fc.Get(context.Background(), client.ObjectKey{Name: "aaa-0"}, dm))
			assert.Equal(t, int64(3), dm.Status.ObservedGeneration)
			assert.Equal(t, "new", dm.Status.UpdateRevision)
			cond := findCondition(dm.Status.Conditions, v1alpha1.CondSuspended)
			require.NotNil(t, cond)
			assert.Equal(t, metav1.ConditionFalse, cond.Status)
			if c.wantReady == metav1.ConditionTrue {
				assert.Equal(t, "old", dm.Status.CurrentRevision)
			}
		})
	}
}

func newStatusDM(opts ...func(*v1alpha1.DM)) *v1alpha1.DM {
	dm := newTestDM("aaa-0")
	dm.Generation = 3
	dm.Labels = map[string]string{v1alpha1.LabelKeyInstanceRevisionHash: "new"}
	for _, opt := range opts {
		opt(dm)
	}
	return dm
}

func readyDMPod() *corev1.Pod {
	return fake.FakeObj("aaa-dm-master-0", func(obj *corev1.Pod) *corev1.Pod {
		obj.Labels = map[string]string{v1alpha1.LabelKeyInstanceRevisionHash: "old"}
		obj.Status.Phase = corev1.PodRunning
		obj.Status.Conditions = []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}
		return obj
	})
}

func findCondition(conds []metav1.Condition, typ string) *metav1.Condition {
	for i := range conds {
		if conds[i].Type == typ {
			return &conds[i]
		}
	}
	return nil
}
