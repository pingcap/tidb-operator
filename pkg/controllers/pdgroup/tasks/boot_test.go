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

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TestTaskBoot(t *testing.T) {
	cases := []struct {
		desc          string
		state         *ReconcileContext
		unexpectedErr bool

		expectedBootstrapped bool
		expectedStatus       task.Status
	}{
		{
			desc: "pd svc is not available, pdg is not bootstrapped",
			state: &ReconcileContext{
				State: &state{
					pdg: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
						obj.Spec.Bootstrapped = false
						return obj
					}),
				},
				IsBootstrapped: false,
			},
			expectedBootstrapped: false,
			expectedStatus:       task.SWait,
		},
		{
			desc: "pd svc is not available, pdg is bootstrapped",
			state: &ReconcileContext{
				State: &state{
					pdg: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
						obj.Spec.Bootstrapped = true
						return obj
					}),
				},
				IsBootstrapped: false,
			},
			expectedBootstrapped: true,
			expectedStatus:       task.SComplete,
		},
		{
			desc: "pd svc is available, pdg is not bootstrapped",
			state: &ReconcileContext{
				State: &state{
					pdg: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
						obj.Spec.Bootstrapped = false
						return obj
					}),
				},
				IsBootstrapped: true,
			},
			expectedBootstrapped: true,
			expectedStatus:       task.SComplete,
		},
		{
			desc: "pd svc is available, pdg is not bootstrapped, but update failed",
			state: &ReconcileContext{
				State: &state{
					pdg: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
						obj.Spec.Bootstrapped = false
						return obj
					}),
				},
				IsBootstrapped: true,
			},
			unexpectedErr:  true,
			expectedStatus: task.SFail,
		},
		{
			desc: "pd svc is available, pdg is bootstrapped",
			state: &ReconcileContext{
				State: &state{
					pdg: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
						obj.Spec.Bootstrapped = true
						return obj
					}),
				},
				IsBootstrapped: true,
			},
			expectedBootstrapped: true,
			expectedStatus:       task.SComplete,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fc := client.NewFakeClient(c.state.PDGroup())
			if c.unexpectedErr {
				fc.WithError("*", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			ctx := context.Background()
			res, done := task.RunTask(ctx, TaskBoot(c.state, fc))
			assert.Equal(tt, c.expectedStatus, res.Status(), c.desc)
			assert.False(tt, done, c.desc)

			// no need to check update result
			if c.unexpectedErr {
				return
			}

			pdg := &v1alpha1.PDGroup{}
			require.NoError(tt, fc.Get(ctx, client.ObjectKey{Name: "aaa"}, pdg), c.desc)
			assert.Equal(tt, c.expectedBootstrapped, pdg.Spec.Bootstrapped, c.desc)
		})
	}
}
