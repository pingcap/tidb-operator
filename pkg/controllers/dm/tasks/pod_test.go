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

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/k8s"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TestTaskPod(t *testing.T) {
	cases := []struct {
		desc          string
		dm            *v1alpha1.DM
		pod           *corev1.Pod
		unexpectedErr bool
		wantStatus    task.Status
		wantPod       bool
	}{
		{
			desc:       "no pod exists",
			dm:         newTestDM("aaa-0"),
			wantStatus: task.SComplete,
			wantPod:    true,
		},
		{
			desc:       "pod exists no change",
			dm:         newTestDM("aaa-0"),
			pod:        newPod(fake.FakeObj[v1alpha1.Cluster]("cluster"), newTestDM("aaa-0")),
			wantStatus: task.SComplete,
			wantPod:    true,
		},
		{
			desc: "version triggers pod recreation",
			dm: newTestDM("aaa-0", func(dm *v1alpha1.DM) {
				dm.Spec.Version = "v8.5.3"
			}),
			pod:        newPod(fake.FakeObj[v1alpha1.Cluster]("cluster"), newTestDM("aaa-0")),
			wantStatus: task.SWait,
			wantPod:    false,
		},
		{
			desc: "config overlay is applied in place",
			dm: newTestDM("aaa-0", func(dm *v1alpha1.DM) {
				dm.Spec.Overlay = &v1alpha1.Overlay{Pod: &v1alpha1.PodOverlay{ObjectMeta: v1alpha1.ObjectMeta{
					Labels: map[string]string{"custom": "label"},
				}}}
			}),
			pod:        newPod(fake.FakeObj[v1alpha1.Cluster]("cluster"), newTestDM("aaa-0")),
			wantStatus: task.SComplete,
			wantPod:    true,
		},
		{
			desc:          "pod creation fails",
			dm:            newTestDM("aaa-0"),
			unexpectedErr: true,
			wantStatus:    task.SFail,
		},
		{
			desc:          "pod update fails",
			dm:            newTestDM("aaa-0"),
			pod:           newPod(fake.FakeObj[v1alpha1.Cluster]("cluster"), newTestDM("aaa-0")),
			unexpectedErr: true,
			wantStatus:    task.SFail,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(t *testing.T) {
			t.Parallel()

			cluster := fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
				obj.Status.ID = "cluster-id"
				return obj
			})
			ctx := newDMReconcileContext(c.dm, cluster, c.pod)
			objs := []client.Object{c.dm, cluster}
			if c.pod != nil {
				objs = append(objs, c.pod)
			}
			fc := client.NewFakeClient(objs...)
			if c.unexpectedErr {
				fc.WithError("patch", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			res, done := task.RunTask(context.Background(), TaskPod(ctx, fc))
			assert.Equal(t, c.wantStatus.String(), res.Status().String(), res.Message())
			assert.False(t, done)
			if !c.wantPod {
				return
			}

			require.NotNil(t, ctx.Pod())
			pod := ctx.Pod()
			require.Len(t, pod.Spec.Containers, 1)
			assert.Equal(t, []string{"/dm-master", "--config", "/etc/dm-master/config.toml"}, pod.Spec.Containers[0].Command)
			assert.Equal(t, v1alpha1.LabelValComponentDMMaster, pod.Labels[v1alpha1.LabelKeyComponent])
			assert.Equal(t, k8s.LabelValK8sAppNameDMCluster, pod.Labels[k8s.LabelKeyK8sAppName])
			assert.Len(t, pod.Spec.Containers[0].Ports, 2)
			assert.Equal(t, int32(8261), pod.Spec.Containers[0].Ports[0].ContainerPort)
			assert.Equal(t, int32(8291), pod.Spec.Containers[0].Ports[1].ContainerPort)
			assert.Equal(t, "true", pod.Annotations["prometheus.io/scrape"])
			assert.Equal(t, "8261", pod.Annotations["prometheus.io/port"])
			assert.Equal(t, "/metrics", pod.Annotations["prometheus.io/path"])
		})
	}
}
