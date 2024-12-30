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

const (
	defaultTestClusterName = "cluster"
)

func TestTaskFinalizerDel(t *testing.T) {
	now := metav1.Now()
	cases := []struct {
		desc          string
		state         State
		subresources  []client.Object
		unexpectedErr bool

		expectedStatus task.Status
		expectedObj    *v1alpha1.TiKVGroup
	}{
		{
			desc: "no tikv and no sub resources and no finalizer",
			state: &state{
				kvg: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.TiKVGroup](&now), func(obj *v1alpha1.TiKVGroup) *v1alpha1.TiKVGroup {
					obj.Spec.Cluster.Name = defaultTestClusterName
					return obj
				}),
			},
			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.TiKVGroup](&now), func(obj *v1alpha1.TiKVGroup) *v1alpha1.TiKVGroup {
				obj.Spec.Cluster.Name = defaultTestClusterName
				return obj
			}),
		},
		{
			desc: "no tikv and no sub resources",
			state: &state{
				kvg: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.TiKVGroup](&now), func(obj *v1alpha1.TiKVGroup) *v1alpha1.TiKVGroup {
					obj.Spec.Cluster.Name = defaultTestClusterName
					obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
					return obj
				}),
			},

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.TiKVGroup](&now), func(obj *v1alpha1.TiKVGroup) *v1alpha1.TiKVGroup {
				obj.Spec.Cluster.Name = defaultTestClusterName
				obj.Finalizers = []string{}
				return obj
			}),
		},
		{
			desc: "no tikv and no sub resources but call api failed",
			state: &state{
				kvg: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.TiKVGroup](&now), func(obj *v1alpha1.TiKVGroup) *v1alpha1.TiKVGroup {
					obj.Spec.Cluster.Name = defaultTestClusterName
					obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
					return obj
				}),
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "no tikv but has sub resources",
			state: &state{
				kvg: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.TiKVGroup](&now), func(obj *v1alpha1.TiKVGroup) *v1alpha1.TiKVGroup {
					obj.Spec.Cluster.Name = defaultTestClusterName
					obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
					return obj
				}),
			},
			subresources: []client.Object{
				fake.FakeObj("aaa",
					fake.Label[corev1.Service](v1alpha1.LabelKeyManagedBy, v1alpha1.LabelValManagedByOperator),
					fake.Label[corev1.Service](v1alpha1.LabelKeyCluster, defaultTestClusterName),
					fake.Label[corev1.Service](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentTiKV),
					fake.Label[corev1.Service](v1alpha1.LabelKeyGroup, "aaa"),
				),
			},

			expectedStatus: task.SWait,
			expectedObj: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.TiKVGroup](&now), func(obj *v1alpha1.TiKVGroup) *v1alpha1.TiKVGroup {
				obj.Spec.Cluster.Name = defaultTestClusterName
				obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
				return obj
			}),
		},
		{
			desc: "no tikv but has sub resources and call api failed",
			state: &state{
				kvg: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.TiKVGroup](&now), func(obj *v1alpha1.TiKVGroup) *v1alpha1.TiKVGroup {
					obj.Spec.Cluster.Name = defaultTestClusterName
					obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
					return obj
				}),
			},
			subresources: []client.Object{
				fake.FakeObj("aaa",
					fake.Label[corev1.Service](v1alpha1.LabelKeyManagedBy, v1alpha1.LabelValManagedByOperator),
					fake.Label[corev1.Service](v1alpha1.LabelKeyCluster, defaultTestClusterName),
					fake.Label[corev1.Service](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentTiKV),
					fake.Label[corev1.Service](v1alpha1.LabelKeyGroup, "aaa"),
				),
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "has tikv with finalizer",
			state: &state{
				kvg: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.TiKVGroup](&now), func(obj *v1alpha1.TiKVGroup) *v1alpha1.TiKVGroup {
					obj.Spec.Cluster.Name = defaultTestClusterName
					obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
					return obj
				}),
				kvs: []*v1alpha1.TiKV{
					fake.FakeObj("aaa", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
				},
			},

			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.TiKVGroup](&now), func(obj *v1alpha1.TiKVGroup) *v1alpha1.TiKVGroup {
				obj.Spec.Cluster.Name = defaultTestClusterName
				obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
				return obj
			}),
		},
		{
			desc: "has tikv with finalizer but call api failed",
			state: &state{
				kvg: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.TiKVGroup](&now), func(obj *v1alpha1.TiKVGroup) *v1alpha1.TiKVGroup {
					obj.Spec.Cluster.Name = defaultTestClusterName
					obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
					return obj
				}),
				kvs: []*v1alpha1.TiKV{
					fake.FakeObj("aaa", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
				},
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "has deleting tikv with finalizer but call api failed",
			state: &state{
				kvg: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.TiKVGroup](&now), func(obj *v1alpha1.TiKVGroup) *v1alpha1.TiKVGroup {
					obj.Spec.Cluster.Name = defaultTestClusterName
					obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
					return obj
				}),
				kvs: []*v1alpha1.TiKV{
					fake.FakeObj("aaa", fake.DeleteNow[v1alpha1.TiKV](), func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
				},
			},
			unexpectedErr: true,

			expectedStatus: task.SRetry,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			objs := []client.Object{
				c.state.TiKVGroup(),
			}

			objs = append(objs, c.subresources...)

			fc := client.NewFakeClient(objs...)
			if c.unexpectedErr {
				// cannot remove finalizer
				fc.WithError("update", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
				// cannot delete sub resources
				fc.WithError("delete", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			res, done := task.RunTask(ctx, TaskFinalizerDel(c.state, fc))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)

			// no need to check update result
			if c.unexpectedErr {
				return
			}

			kvg := &v1alpha1.TiKVGroup{}
			require.NoError(tt, fc.Get(ctx, client.ObjectKey{Name: "aaa"}, kvg), c.desc)
			assert.Equal(tt, c.expectedObj, kvg, c.desc)
		})
	}
}
