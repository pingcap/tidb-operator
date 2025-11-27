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
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	stateutil "github.com/pingcap/tidb-operator/v2/pkg/state"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TestTaskService(t *testing.T) {
	cases := []struct {
		desc          string
		state         State
		objs          []client.Object
		unexpectedErr bool

		expectedStatus task.Status
	}{
		{
			desc:  "no svc exists",
			state: fakeState(),

			expectedStatus: task.SComplete,
		},
		{
			desc:  "headless svc has exists",
			state: fakeState(),
			objs: []client.Object{
				fake.FakeObj[corev1.Service]("aaa-pd-peer"),
			},

			expectedStatus: task.SComplete,
		},
		{
			desc:  "internal svc has exists",
			state: fakeState(),
			objs: []client.Object{
				fake.FakeObj[corev1.Service]("aaa-pd"),
			},

			expectedStatus: task.SComplete,
		},
		{
			desc:          "apply headless svc with unexpected err",
			state:         fakeState(),
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc:  "apply internal svc with unexpected err",
			state: fakeState(),
			objs: []client.Object{
				newHeadlessService(fakePDGroup()),
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc:  "all svcs are updated with unexpected err",
			state: fakeState(),
			objs: []client.Object{
				newHeadlessService(fakePDGroup()),
				newInternalService(fakePDGroup()),
			},
			unexpectedErr: true,

			expectedStatus: task.SComplete,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctx := context.Background()
			fc := client.NewFakeClient(c.state.Cluster(), c.state.PDGroup())
			for _, obj := range c.objs {
				require.NoError(tt, fc.Apply(ctx, obj), c.desc)
			}

			if c.unexpectedErr {
				// cannot update svc
				fc.WithError("patch", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			res, done := task.RunTask(ctx, TaskService(c.state, fc))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)

			if !c.unexpectedErr {
				svcs := corev1.ServiceList{}
				require.NoError(tt, fc.List(ctx, &svcs), c.desc)
				assert.Len(tt, svcs.Items, 2, c.desc)
			}
		})
	}
}

func fakePDGroup() *v1alpha1.PDGroup {
	return fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
		obj.Spec.Cluster.Name = "aaa"
		return obj
	})
}

func fakeState() State {
	s := &state{
		pdg:     fakePDGroup(),
		cluster: fake.FakeObj[v1alpha1.Cluster]("aaa"),
	}
	s.IFeatureGates = stateutil.NewFeatureGates[scope.PDGroup](s)

	return s
}
