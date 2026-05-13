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
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TestTaskService(t *testing.T) {
	cases := []struct {
		desc          string
		objs          []client.Object
		unexpectedErr bool
		wantStatus    task.Status
		wantServices  int
	}{
		{desc: "no svc exists", wantStatus: task.SComplete, wantServices: 1},
		{desc: "svc already exists", objs: []client.Object{newHeadlessService(newTestDMWorkerGroup("aaa"))}, wantStatus: task.SComplete, wantServices: 1},
		{desc: "apply fails", unexpectedErr: true, wantStatus: task.SFail},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(t *testing.T) {
			t.Parallel()

			dwg := newTestDMWorkerGroup("aaa")
			fc := client.NewFakeClient(dwg)
			for _, obj := range c.objs {
				require.NoError(t, fc.Apply(context.Background(), obj))
			}
			if c.unexpectedErr {
				fc.WithError("patch", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			res, done := task.RunTask(context.Background(), TaskService(&state{dwg: dwg}, fc))
			assert.Equal(t, c.wantStatus, res.Status(), res.Message())
			assert.False(t, done)
			if c.wantStatus != task.SComplete {
				return
			}

			svcs := corev1.ServiceList{}
			require.NoError(t, fc.List(context.Background(), &svcs))
			assert.Len(t, svcs.Items, c.wantServices)
			assert.Equal(t, v1alpha1.LabelValComponentDMWorker, svcs.Items[0].Labels[v1alpha1.LabelKeyComponent])
		})
	}
}
