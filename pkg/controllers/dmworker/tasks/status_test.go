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
	}{
		{desc: "pod not ready", ctx: newDMWorkerReconcileContext(newStatusDMWorker(), fake.FakeObj[v1alpha1.Cluster]("cluster"), nil), wantStatus: task.SRetry},
		{
			desc: "pod ready",
			ctx: newDMWorkerReconcileContext(newStatusDMWorker(func(dw *v1alpha1.DMWorker) {
				dw.Status.Conditions = []metav1.Condition{{Type: v1alpha1.CondReady, Status: metav1.ConditionTrue, ObservedGeneration: 3}}
			}), fake.FakeObj[v1alpha1.Cluster]("cluster"), readyDMWorkerPod()),
			wantStatus: task.SComplete,
		},
		{
			desc: "pod terminating",
			ctx: newDMWorkerReconcileContext(newStatusDMWorker(), fake.FakeObj[v1alpha1.Cluster]("cluster"), fake.FakeObj("aaa-dm-worker-0", func(obj *corev1.Pod) *corev1.Pod {
				obj.SetDeletionTimestamp(&now)
				return obj
			})),
			wantStatus: task.SRetry,
		},
		{desc: "status update fails", ctx: newDMWorkerReconcileContext(newStatusDMWorker(), fake.FakeObj[v1alpha1.Cluster]("cluster"), readyDMWorkerPod()), unexpectedErr: true, wantStatus: task.SFail},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(t *testing.T) {
			t.Parallel()

			fc := client.NewFakeClient(c.ctx.DMWorker())
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

			dw := &v1alpha1.DMWorker{}
			require.NoError(t, fc.Get(context.Background(), client.ObjectKey{Name: "aaa-0"}, dw))
			assert.Equal(t, int64(3), dw.Status.ObservedGeneration)
			assert.Equal(t, "new", dw.Status.UpdateRevision)
			cond := findCondition(dw.Status.Conditions, v1alpha1.CondSuspended)
			require.NotNil(t, cond)
			assert.Equal(t, metav1.ConditionFalse, cond.Status)
		})
	}
}

func newStatusDMWorker(opts ...func(*v1alpha1.DMWorker)) *v1alpha1.DMWorker {
	dw := newTestDMWorker("aaa-0")
	dw.Generation = 3
	dw.Labels = map[string]string{v1alpha1.LabelKeyInstanceRevisionHash: "new"}
	for _, opt := range opts {
		opt(dw)
	}
	return dw
}

func readyDMWorkerPod() *corev1.Pod {
	return fake.FakeObj("aaa-dm-worker-0", func(obj *corev1.Pod) *corev1.Pod {
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
