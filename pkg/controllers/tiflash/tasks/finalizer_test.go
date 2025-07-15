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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TestTaskFinalizerDel(t *testing.T) {
	type testCase struct {
		desc          string
		state         *ReconcileContext
		subresources  []client.Object
		unexpectedErr bool

		expectedStatus task.Status
		expectedObj    *v1alpha1.TiFlash
	}

	cases := []testCase{
		// Original TaskFinalizerDel test cases
		{
			desc: "no sub resources, no finalizer",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						return obj
					}),
				},
			},
			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				return obj
			}),
		},
		{
			desc: "has sub resources (pods), should wait for deletion",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
				},
			},
			subresources:   fakeSubresources("Pod"),
			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "has sub resources (configmaps), should wait for deletion",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
				},
			},
			subresources:   fakeSubresources("ConfigMap"),
			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "has sub resources (pvcs), should wait for deletion",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
				},
			},
			subresources:   fakeSubresources("PersistentVolumeClaim"),
			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "has multiple sub resources, should wait for deletion",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
				},
			},
			subresources:   fakeSubresources("Pod", "ConfigMap", "PersistentVolumeClaim"),
			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "failed to delete subresources (pods)",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
				},
			},
			subresources:   fakeSubresources("Pod"),
			unexpectedErr:  true,
			expectedStatus: task.SFail,
			expectedObj: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "failed to delete subresources (configmaps)",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
				},
			},
			subresources:   fakeSubresources("ConfigMap"),
			unexpectedErr:  true,
			expectedStatus: task.SFail,
			expectedObj: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "failed to delete subresources (pvcs)",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
				},
			},
			subresources:   fakeSubresources("PersistentVolumeClaim"),
			unexpectedErr:  true,
			expectedStatus: task.SFail,
			expectedObj: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "no sub resources, has finalizer, should remove finalizer",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
				},
			},
			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = []string{}
				return obj
			}),
		},
		{
			desc: "no sub resources, has finalizer, failed to remove finalizer",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
				},
			},
			unexpectedErr:  true,
			expectedStatus: task.SFail,
			expectedObj: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "has pods with deletion timestamp, should wait",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
				},
			},
			subresources:   fakeSubresources("PodDeleting"),
			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		{
			desc: "has mixed sub resources with some deleting",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
						return obj
					}),
				},
			},
			subresources: append(
				fakeSubresources("Pod"),
				fakeSubresources("PodDeleting")...,
			),
			expectedStatus: task.SRetry,
			expectedObj: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
				obj.Finalizers = append(obj.Finalizers, meta.Finalizer)
				return obj
			}),
		},
		// Edge cases
		{
			desc: "empty tiflash with multiple finalizers",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = []string{meta.Finalizer, "other.finalizer.com/test", "another.finalizer.com/test"}
						return obj
					}),
				},
			},
			expectedStatus: task.SComplete,
		},
		{
			desc: "tiflash with only other finalizers",
			state: &ReconcileContext{
				State: &state{
					tiflash: fake.FakeObj("test-tiflash", func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
						obj.Finalizers = []string{"other.finalizer.com/test", "another.finalizer.com/test"}
						return obj
					}),
				},
			},
			expectedStatus: task.SComplete,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			var objs []client.Object
			if c.state.TiFlash() != nil {
				objs = append(objs, c.state.TiFlash())
			}
			objs = append(objs, c.subresources...)

			fc := client.NewFakeClient(objs...)
			if c.unexpectedErr {
				// cannot remove finalizer
				fc.WithError("update", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
				// cannot delete sub resources
				fc.WithError("delete", "*", errors.NewInternalError(fmt.Errorf("fake delete error")))
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			res, done := task.RunTask(ctx, TaskFinalizerDel(c.state, fc))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)

			// no need to check update result if there's an unexpected error
			if c.unexpectedErr {
				return
			}

			// For edge cases without expectedObj, skip the object check
			if c.expectedObj != nil {
				tiflash := &v1alpha1.TiFlash{}
				require.NoError(tt, fc.Get(ctx, client.ObjectKey{Name: "test-tiflash"}, tiflash), c.desc)
				assert.Equal(tt, c.expectedObj, tiflash, c.desc)
			}
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
					v1alpha1.LabelKeyInstance:  "test-tiflash",
					v1alpha1.LabelKeyCluster:   "",
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiFlash,
				}
				return obj
			})
		case "PodDeleting":
			obj = fake.FakeObj("pod-deleting-"+strconv.Itoa(i), func(obj *corev1.Pod) *corev1.Pod {
				obj.Labels = map[string]string{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyInstance:  "test-tiflash",
					v1alpha1.LabelKeyCluster:   "",
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiFlash,
				}
				now := metav1.NewTime(time.Now())
				obj.SetDeletionTimestamp(&now)
				return obj
			})
		case "ConfigMap":
			obj = fake.FakeObj(strconv.Itoa(i), func(obj *corev1.ConfigMap) *corev1.ConfigMap {
				obj.Labels = map[string]string{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyInstance:  "test-tiflash",
					v1alpha1.LabelKeyCluster:   "",
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiFlash,
				}
				return obj
			})
		case "PersistentVolumeClaim":
			obj = fake.FakeObj(strconv.Itoa(i), func(obj *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
				obj.Labels = map[string]string{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyInstance:  "test-tiflash",
					v1alpha1.LabelKeyCluster:   "",
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiFlash,
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
