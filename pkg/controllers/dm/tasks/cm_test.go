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

func TestTaskConfigMap(t *testing.T) {
	bootDM := newTestDM("aaa-0", func(dm *v1alpha1.DM) {
		dm.Annotations = map[string]string{v1alpha1.AnnoKeyInitialClusterNum: "1"}
	})

	cases := []struct {
		desc          string
		state         *ReconcileContext
		objs          []client.Object
		unexpectedErr bool
		wantStatus    task.Status
		wantConfig    []string
	}{
		{
			desc: "no config joins existing cluster",
			state: newDMReconcileContext(newTestDM("aaa-0"), fake.FakeObj[v1alpha1.Cluster]("cluster"), nil,
				newTestDM("aaa-0")),
			wantStatus: task.SComplete,
			wantConfig: []string{
				`master-addr = '[::]:8261'`,
				`advertise-addr = 'aaa-dm-master-0.aaa-dm-master-peer.default:8261'`,
				`join = 'aaa-dm-master-peer.default:8261'`,
			},
		},
		{
			desc:       "bootstrap initial cluster",
			state:      newDMReconcileContext(bootDM, fake.FakeObj[v1alpha1.Cluster]("cluster"), nil, bootDM),
			wantStatus: task.SComplete,
			wantConfig: []string{
				`initial-cluster = 'aaa-0=http://aaa-dm-master-0.aaa-dm-master-peer.default:8291'`,
				`initial-cluster-state = 'new'`,
				`openapi = true`,
			},
		},
		{
			desc: "invalid config",
			state: newDMReconcileContext(newTestDM("aaa-0", func(dm *v1alpha1.DM) {
				dm.Spec.Config = `invalid`
			}), fake.FakeObj[v1alpha1.Cluster]("cluster"), nil),
			wantStatus: task.SFail,
		},
		{
			desc: "managed field is rejected",
			state: newDMReconcileContext(newTestDM("aaa-0", func(dm *v1alpha1.DM) {
				dm.Spec.Config = `master-addr = "127.0.0.1:8261"`
			}), fake.FakeObj[v1alpha1.Cluster]("cluster"), nil),
			wantStatus: task.SFail,
		},
		{
			desc: "has config map",
			state: newDMReconcileContext(newTestDM("aaa-0"), fake.FakeObj[v1alpha1.Cluster]("cluster"), nil,
				newTestDM("aaa-0")),
			objs:       []client.Object{&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "aaa-dm-master-0"}}},
			wantStatus: task.SComplete,
		},
		{
			desc: "update config map failed",
			state: newDMReconcileContext(newTestDM("aaa-0"), fake.FakeObj[v1alpha1.Cluster]("cluster"), nil,
				newTestDM("aaa-0")),
			objs:          []client.Object{&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "aaa-dm-master-0"}}},
			unexpectedErr: true,
			wantStatus:    task.SFail,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			fc := client.NewFakeClient(c.state.DM(), c.state.Cluster())
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
			cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "aaa-dm-master-0"}}
			require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(cm), cm))
			config := cm.Data[v1alpha1.FileNameConfig]
			assert.NotEmpty(t, config)
			for _, want := range c.wantConfig {
				assert.Contains(t, config, want)
			}
		})
	}
}

func newDMReconcileContext(dm *v1alpha1.DM, cluster *v1alpha1.Cluster, pod *corev1.Pod, dms ...*v1alpha1.DM) *ReconcileContext {
	return &ReconcileContext{State: &state{dm: dm, cluster: cluster, pod: pod, dms: dms}}
}

func newTestDM(name string, opts ...func(*v1alpha1.DM)) *v1alpha1.DM { //nolint:unparam
	dm := fake.FakeObj(name, func(obj *v1alpha1.DM) *v1alpha1.DM {
		obj.Spec.Cluster.Name = "cluster"
		obj.Spec.Subdomain = "aaa-dm-master-peer"
		obj.Spec.Version = "v8.5.2"
		obj.Spec.DataVolume.Name = "data"
		return obj
	})
	for _, opt := range opts {
		opt(dm)
	}
	return dm
}
