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
	"strings"
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

func TestTaskConfigMap(t *testing.T) {
	cases := []struct {
		desc          string
		state         *ReconcileContext
		objs          []client.Object
		unexpectedErr bool
		wantStatus    task.Status
		wantConfig    []string
	}{
		{
			desc:       "no config",
			state:      newDMWorkerReconcileContext(newTestDMWorker("aaa-0"), fake.FakeObj[v1alpha1.Cluster]("cluster"), nil),
			wantStatus: task.SComplete,
			wantConfig: []string{
				`worker-addr = '[::]:8262'`,
				`advertise-addr = 'aaa-dm-worker-0.aaa-dm-worker-peer.default:8262'`,
				`join = 'dmg-dm-master.default:8261'`,
			},
		},
		{
			desc: "invalid config",
			state: newDMWorkerReconcileContext(newTestDMWorker("aaa-0", func(dw *v1alpha1.DMWorker) {
				dw.Spec.Config = `invalid`
			}), fake.FakeObj[v1alpha1.Cluster]("cluster"), nil),
			wantStatus: task.SFail,
		},
		{
			desc: "managed field is rejected",
			state: newDMWorkerReconcileContext(newTestDMWorker("aaa-0", func(dw *v1alpha1.DMWorker) {
				dw.Spec.Config = `join = "127.0.0.1:8261"`
			}), fake.FakeObj[v1alpha1.Cluster]("cluster"), nil),
			wantStatus: task.SFail,
		},
		{
			desc:       "has config map",
			state:      newDMWorkerReconcileContext(newTestDMWorker("aaa-0"), fake.FakeObj[v1alpha1.Cluster]("cluster"), nil),
			objs:       []client.Object{&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "aaa-dm-worker-0"}}},
			wantStatus: task.SComplete,
		},
		{
			desc:          "update config map failed",
			state:         newDMWorkerReconcileContext(newTestDMWorker("aaa-0"), fake.FakeObj[v1alpha1.Cluster]("cluster"), nil),
			objs:          []client.Object{&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "aaa-dm-worker-0"}}},
			unexpectedErr: true,
			wantStatus:    task.SFail,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			fc := client.NewFakeClient(c.state.DMWorker(), c.state.Cluster())
			for _, obj := range c.objs {
				require.NoError(t, fc.Apply(ctx, obj))
			}
			if c.unexpectedErr {
				fc.WithError("patch", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			res, done := task.RunTask(ctx, TaskConfigMap(c.state, fc))
			assert.Equal(t, c.wantStatus.String(), res.Status().String(), res.Message())
			assert.False(t, done)
			if c.wantStatus != task.SComplete {
				return
			}

			cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "aaa-dm-worker-0"}}
			require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(cm), cm))
			config := cm.Data[v1alpha1.FileNameConfig]
			assert.NotEmpty(t, config)
			for _, want := range c.wantConfig {
				assert.True(t, strings.Contains(config, want), "config should contain %q:\n%s", want, config)
			}
		})
	}
}

func newDMWorkerReconcileContext(dw *v1alpha1.DMWorker, cluster *v1alpha1.Cluster, pod *corev1.Pod) *ReconcileContext {
	return &ReconcileContext{State: &state{dw: dw, cluster: cluster, pod: pod}}
}

func newTestDMWorker(name string, opts ...func(*v1alpha1.DMWorker)) *v1alpha1.DMWorker {
	dw := fake.FakeObj(name, func(obj *v1alpha1.DMWorker) *v1alpha1.DMWorker {
		obj.Spec.Cluster.Name = "cluster"
		obj.Spec.DMGroupRef.Name = "dmg"
		obj.Spec.Subdomain = "aaa-dm-worker-peer"
		obj.Spec.Version = "v8.5.2"
		obj.Spec.RelayVolume.Name = "relay"
		return obj
	})
	for _, opt := range opts {
		opt(dw)
	}
	return dw
}
