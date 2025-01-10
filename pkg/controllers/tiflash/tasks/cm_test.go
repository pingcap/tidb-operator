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

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

const fakePDAddr = "any string, useless in test"

func TestTaskConfigMap(t *testing.T) {
	cases := []struct {
		desc          string
		state         *ReconcileContext
		objs          []client.Object
		unexpectedErr bool
		invalidConfig bool

		expectedStatus task.Status
	}{
		{
			desc: "no config",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj[v1alpha1.TiFlash]("aaa-xxx"),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.Status.PD = fakePDAddr
						return obj
					}),
				},
			},
			expectedStatus: task.SComplete,
		},
		{
			desc: "invalid config",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Spec.Config = `invalid`
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.Status.PD = fakePDAddr
						return obj
					}),
				},
			},
			invalidConfig:  true,
			expectedStatus: task.SFail,
		},
		{
			desc: "invalid proxy config",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Spec.ProxyConfig = `invalid`
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.Status.PD = fakePDAddr
						return obj
					}),
				},
			},
			invalidConfig:  true,
			expectedStatus: task.SFail,
		},
		{
			desc: "config with managed field",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Spec.Config = `tmp_path = 'xxx'`
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.Status.PD = fakePDAddr
						return obj
					}),
				},
			},
			invalidConfig:  true,
			expectedStatus: task.SFail,
		},
		{
			desc: "proxy config with managed field",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Spec.ProxyConfig = `server.status-addr = 'xxx'`
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.Status.PD = fakePDAddr
						return obj
					}),
				},
			},
			invalidConfig:  true,
			expectedStatus: task.SFail,
		},
		{
			desc: "has config map",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.Status.PD = fakePDAddr
						return obj
					}),
				},
			},
			objs: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aaa-tiflash-xxx",
					},
				},
			},
			expectedStatus: task.SComplete,
		},
		{
			desc: "update config map failed",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.Status.PD = fakePDAddr
						return obj
					}),
				},
			},
			objs: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aaa-tiflash-xxx",
					},
				},
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctx := context.Background()
			var objs []client.Object
			objs = append(objs, c.state.TiFlash(), c.state.Cluster())
			fc := client.NewFakeClient(objs...)
			for _, obj := range c.objs {
				require.NoError(tt, fc.Apply(ctx, obj), c.desc)
			}

			if c.unexpectedErr {
				// cannot update svc
				fc.WithError("patch", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			res, done := task.RunTask(ctx, TaskConfigMap(c.state, fc))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), res.Message())
			assert.False(tt, done, c.desc)

			if !c.invalidConfig {
				// config hash should be set
				assert.NotEmpty(tt, c.state.ConfigHash, c.desc)
			}

			if c.expectedStatus == task.SComplete {
				cm := corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "aaa-tiflash-xxx",
					},
				}
				require.NoError(tt, fc.Get(ctx, client.ObjectKeyFromObject(&cm), &cm), c.desc)
				assert.Equal(tt, c.state.ConfigHash, cm.Labels[v1alpha1.LabelKeyConfigHash], c.desc)
			}
		})
	}
}
