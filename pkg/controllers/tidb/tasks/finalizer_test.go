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
	"strconv"
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

func TestTaskFinalizerDel(t *testing.T) {
	now := metav1.Now()
	cases := []struct {
		desc          string
		state         *ReconcileContext
		subresources  []client.Object
		unexpectedErr bool

		expectedStatus task.Status
		expectedObj    *v1alpha1.TiDB
	}{
		{
			desc: "no sub resources, no finalizer",
			state: &ReconcileContext{
				State: &state{
					tidb: fake.FakeObj("aaa", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.SetDeletionTimestamp(&now)
						return obj
					}),
				},
			},

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
				return obj
			}),
		},
		{
			desc: "has sub resources, has finalizer",
			state: &ReconcileContext{
				State: &state{
					tidb: fake.FakeObj("aaa", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.SetDeletionTimestamp(&now)
						return obj
					}),
				},
			},
			subresources: fakeSubresources("Pod", "ConfigMap", "PersistentVolumeClaim"),

			expectedStatus: task.SWait,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
				obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
				return obj
			}),
		},
		{
			desc: "has sub resources, has finalizer, failed to del subresources(pod)",
			state: &ReconcileContext{
				State: &state{
					tidb: fake.FakeObj("aaa", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.SetDeletionTimestamp(&now)
						return obj
					}),
				},
			},
			subresources: fakeSubresources("Pod"),

			unexpectedErr: true,

			expectedStatus: task.SFail,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
				obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
				return obj
			}),
		},
		{
			desc: "has sub resources, has finalizer, failed to del subresources(cm)",
			state: &ReconcileContext{
				State: &state{
					tidb: fake.FakeObj("aaa", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.SetDeletionTimestamp(&now)
						return obj
					}),
				},
			},
			subresources: fakeSubresources("ConfigMap"),

			unexpectedErr: true,

			expectedStatus: task.SFail,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
				obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
				return obj
			}),
		},
		{
			desc: "has sub resources, has finalizer, failed to del subresources(pvc)",
			state: &ReconcileContext{
				State: &state{
					tidb: fake.FakeObj("aaa", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.SetDeletionTimestamp(&now)
						return obj
					}),
				},
			},
			subresources: fakeSubresources("PersistentVolumeClaim"),

			unexpectedErr: true,

			expectedStatus: task.SFail,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
				obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
				return obj
			}),
		},
		{
			desc: "no sub resources, has finalizer",
			state: &ReconcileContext{
				State: &state{
					tidb: fake.FakeObj("aaa", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.SetDeletionTimestamp(&now)
						return obj
					}),
				},
			},

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
				obj.Finalizers = []string{}
				return obj
			}),
		},
		{
			desc: "no sub resources, has finalizer, failed to remove finalizer",
			state: &ReconcileContext{
				State: &state{
					tidb: fake.FakeObj("aaa", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
						obj.Finalizers = append(obj.Finalizers, v1alpha1.Finalizer)
						return obj
					}),
					cluster: fake.FakeObj("cluster", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
						obj.SetDeletionTimestamp(&now)
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
				c.state.TiDB(),
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

			pd := &v1alpha1.TiDB{}
			require.NoError(tt, fc.Get(ctx, client.ObjectKey{Name: "aaa"}, pd), c.desc)
			assert.Equal(tt, c.expectedObj, pd, c.desc)
		})
	}
}

func fakeSubresources(types ...string) []client.Object {
	var objs []client.Object
	for i, t := range types {
		var obj client.Object
		switch t {
		case "Pod":
			obj = fake.FakeObj(strconv.Itoa(i), func(obj *corev1.Pod) *corev1.Pod {
				obj.Labels = map[string]string{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyInstance:  "aaa",
					v1alpha1.LabelKeyCluster:   "",
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiDB,
				}
				return obj
			})
		case "ConfigMap":
			obj = fake.FakeObj(strconv.Itoa(i), func(obj *corev1.ConfigMap) *corev1.ConfigMap {
				obj.Labels = map[string]string{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyInstance:  "aaa",
					v1alpha1.LabelKeyCluster:   "",
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiDB,
				}
				return obj
			})
		case "PersistentVolumeClaim":
			obj = fake.FakeObj(strconv.Itoa(i), func(obj *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
				obj.Labels = map[string]string{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyInstance:  "aaa",
					v1alpha1.LabelKeyCluster:   "",
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiDB,
				}
				return obj
			})
		}
		if obj != nil {
			objs = append(objs, obj)
		}
	}

	return objs
}
