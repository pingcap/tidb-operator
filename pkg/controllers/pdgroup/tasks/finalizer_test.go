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
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/pkg/timanager"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
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
		isDeregistered bool
		expectedObj    *v1alpha1.PDGroup
	}{
		{
			desc: "no pd and no sub resources and no finalizer",
			state: &state{
				pdg: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
					obj.Spec.Cluster.Name = defaultTestClusterName
					return obj
				}),
				cluster: fake.FakeObj[v1alpha1.Cluster](defaultTestClusterName),
			},
			isDeregistered: true,
			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Spec.Cluster.Name = defaultTestClusterName
				return obj
			}),
		},
		{
			desc: "no pd and no sub resources",
			state: &state{
				pdg: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
					obj.Spec.Cluster.Name = defaultTestClusterName
					obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
					return obj
				}),
				cluster: fake.FakeObj[v1alpha1.Cluster](defaultTestClusterName),
			},

			isDeregistered: true,
			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Spec.Cluster.Name = defaultTestClusterName
				obj.Finalizers = []string{}
				return obj
			}),
		},
		{
			desc: "no pd and no sub resources but call api failed",
			state: &state{
				pdg: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
					obj.Spec.Cluster.Name = defaultTestClusterName
					obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
					return obj
				}),
				cluster: fake.FakeObj[v1alpha1.Cluster](defaultTestClusterName),
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "no pd but has sub resources",
			state: &state{
				pdg: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
					obj.Spec.Cluster.Name = defaultTestClusterName
					obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
					return obj
				}),
				cluster: fake.FakeObj[v1alpha1.Cluster](defaultTestClusterName),
			},
			subresources: []client.Object{
				fake.FakeObj("aaa",
					fake.Label[corev1.Service](v1alpha1.LabelKeyManagedBy, v1alpha1.LabelValManagedByOperator),
					fake.Label[corev1.Service](v1alpha1.LabelKeyCluster, defaultTestClusterName),
					fake.Label[corev1.Service](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentPD),
					fake.Label[corev1.Service](v1alpha1.LabelKeyGroup, "aaa"),
				),
			},

			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Spec.Cluster.Name = defaultTestClusterName
				obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
				return obj
			}),
		},
		{
			desc: "no pd but has sub resources and call api failed",
			state: &state{
				pdg: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
					obj.Spec.Cluster.Name = defaultTestClusterName
					obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
					return obj
				}),
				cluster: fake.FakeObj[v1alpha1.Cluster](defaultTestClusterName),
			},
			subresources: []client.Object{
				fake.FakeObj("aaa",
					fake.Label[corev1.Service](v1alpha1.LabelKeyManagedBy, v1alpha1.LabelValManagedByOperator),
					fake.Label[corev1.Service](v1alpha1.LabelKeyCluster, defaultTestClusterName),
					fake.Label[corev1.Service](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentPD),
					fake.Label[corev1.Service](v1alpha1.LabelKeyGroup, "aaa"),
				),
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "has pd with finalizer",
			state: &state{
				pdg: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
					obj.Spec.Cluster.Name = defaultTestClusterName
					obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
					return obj
				}),
				cluster: fake.FakeObj[v1alpha1.Cluster](defaultTestClusterName),
				pds: []*v1alpha1.PD{
					fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
				},
			},

			expectedStatus: task.SWait,
			expectedObj: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Spec.Cluster.Name = defaultTestClusterName
				obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
				return obj
			}),
		},
		{
			desc: "has pd with finalizer but call api failed",
			state: &state{
				pdg: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
					obj.Spec.Cluster.Name = defaultTestClusterName
					obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
					return obj
				}),
				cluster: fake.FakeObj[v1alpha1.Cluster](defaultTestClusterName),
				pds: []*v1alpha1.PD{
					fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
				},
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "has deleting pd with finalizer but call api failed",
			state: &state{
				pdg: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
					obj.Spec.Cluster.Name = defaultTestClusterName
					obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
					return obj
				}),
				cluster: fake.FakeObj[v1alpha1.Cluster](defaultTestClusterName),
				pds: []*v1alpha1.PD{
					fake.FakeObj("aaa", fake.DeleteNow[v1alpha1.PD](), func(obj *v1alpha1.PD) *v1alpha1.PD {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
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

			objs := []client.Object{
				c.state.PDGroup(), c.state.Cluster(),
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

			m := NewFakePDClientManager()
			m.Start(ctx)
			require.NoError(tt, m.Register(c.state.PDGroup()), c.desc)

			res, done := task.RunTask(ctx, TaskFinalizerDel(c.state, fc, m))
			assert.Equal(tt, c.expectedStatus, res.Status(), c.desc)
			assert.False(tt, done, c.desc)

			ck := c.state.Cluster()
			_, ok := m.Get(pdm.PrimaryKey(ck.Namespace, ck.Name))
			assert.Equal(tt, c.isDeregistered, !ok, c.desc)

			// no need to check update result
			if c.unexpectedErr {
				return
			}

			pdg := &v1alpha1.PDGroup{}
			require.NoError(tt, fc.Get(ctx, client.ObjectKey{Name: "aaa"}, pdg), c.desc)
			assert.Equal(tt, c.expectedObj, pdg, c.desc)
		})
	}
}

func NewFakePDClientManager() pdm.PDClientManager {
	return timanager.NewManagerBuilder[*v1alpha1.PDGroup, pdapi.PDClient, pdm.PDClient]().
		WithNewUnderlayClientFunc(func(*v1alpha1.PDGroup) (pdapi.PDClient, error) {
			return nil, nil
		}).
		WithNewClientFunc(func(string, pdapi.PDClient, timanager.SharedInformerFactory[pdapi.PDClient]) pdm.PDClient {
			return nil
		}).
		WithCacheKeysFunc(pdm.CacheKeys).
		Build()
}
