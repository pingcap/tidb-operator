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

type tikvOption func(*v1alpha1.TiKV)

func withFinalizers(finalizers ...string) tikvOption {
	return func(tikv *v1alpha1.TiKV) {
		// if the slice is nil (which happens when called with no args),
		// assign an empty slice to avoid nil vs empty slice issues in tests.
		if finalizers == nil {
			tikv.Finalizers = []string{}
			return
		}
		tikv.Finalizers = finalizers
	}
}

func withOfflineSpec() tikvOption {
	return func(tikv *v1alpha1.TiKV) {
		tikv.Spec.Offline = true
	}
}

func withOfflineCondition(reason string) tikvOption {
	return func(tikv *v1alpha1.TiKV) {
		tikv.Status.Conditions = []metav1.Condition{
			{
				Type:   v1alpha1.StoreOfflineConditionType,
				Reason: reason,
			},
		}
	}
}

func newFakeTiKV(opts ...tikvOption) *v1alpha1.TiKV {
	return fake.FakeObj("test-tikv", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
		for _, opt := range opts {
			opt(obj)
		}
		return obj
	})
}

func TestTaskFinalizerDel(t *testing.T) {
	cases := []struct {
		desc          string
		state         *ReconcileContext
		subresources  []client.Object
		unexpectedErr bool

		expectedStatus task.Status
		expectedObj    *v1alpha1.TiKV
	}{
		{
			desc: "tikv.Spec.Offline is false, should wait",
			state: &ReconcileContext{
				State: &state{tikv: newFakeTiKV(withFinalizers(meta.Finalizer))},
			},
			expectedStatus: task.SWait,
			expectedObj:    newFakeTiKV(withFinalizers(meta.Finalizer)),
		},
		{
			desc: "StoreOfflineConditionType condition is missing, should wait",
			state: &ReconcileContext{
				State: &state{tikv: newFakeTiKV(withOfflineSpec(), withFinalizers(meta.Finalizer))},
			},
			expectedStatus: task.SWait,
			expectedObj:    newFakeTiKV(withOfflineSpec(), withFinalizers(meta.Finalizer)),
		},
		{
			desc: "StoreOfflineConditionType status is not Completed, should wait",
			state: &ReconcileContext{
				State: &state{tikv: newFakeTiKV(withOfflineSpec(), withFinalizers(meta.Finalizer), withOfflineCondition(v1alpha1.OfflineReasonActive))},
			},
			expectedStatus: task.SWait,
			expectedObj:    newFakeTiKV(withOfflineSpec(), withFinalizers(meta.Finalizer), withOfflineCondition(v1alpha1.OfflineReasonActive)),
		},
		{
			desc: "no sub resources, no finalizer",
			state: &ReconcileContext{
				State: &state{tikv: newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted))},
			},
			expectedStatus: task.SComplete,
			expectedObj:    newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted)),
		},
		{
			desc: "has sub resources (pods), should wait for deletion",
			state: &ReconcileContext{
				State: &state{tikv: newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer))},
			},
			subresources:   fakeSubresources("Pod"),
			expectedStatus: task.SRetry,
			expectedObj:    newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer)),
		},
		{
			desc: "has sub resources (configmaps), should wait for deletion",
			state: &ReconcileContext{
				State: &state{tikv: newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer))},
			},
			subresources:   fakeSubresources("ConfigMap"),
			expectedStatus: task.SRetry,
			expectedObj:    newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer)),
		},
		{
			desc: "has sub resources (pvcs), should wait for deletion",
			state: &ReconcileContext{
				State: &state{tikv: newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer))},
			},
			subresources:   fakeSubresources("PersistentVolumeClaim"),
			expectedStatus: task.SRetry,
			expectedObj:    newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer)),
		},
		{
			desc: "has multiple sub resources, should wait for deletion",
			state: &ReconcileContext{
				State: &state{tikv: newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer))},
			},
			subresources:   fakeSubresources("Pod", "ConfigMap", "PersistentVolumeClaim"),
			expectedStatus: task.SRetry,
			expectedObj:    newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer)),
		},
		{
			desc: "failed to delete subresources (pods)",
			state: &ReconcileContext{
				State: &state{tikv: newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer))},
			},
			subresources:   fakeSubresources("Pod"),
			unexpectedErr:  true,
			expectedStatus: task.SFail,
			expectedObj:    newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer)),
		},
		{
			desc: "failed to delete subresources (configmaps)",
			state: &ReconcileContext{
				State: &state{tikv: newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer))},
			},
			subresources:   fakeSubresources("ConfigMap"),
			unexpectedErr:  true,
			expectedStatus: task.SFail,
			expectedObj:    newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer)),
		},
		{
			desc: "failed to delete subresources (pvcs)",
			state: &ReconcileContext{
				State: &state{tikv: newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer))},
			},
			subresources:   fakeSubresources("PersistentVolumeClaim"),
			unexpectedErr:  true,
			expectedStatus: task.SFail,
			expectedObj:    newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer)),
		},
		{
			desc: "no sub resources, has finalizer, should remove finalizer",
			state: &ReconcileContext{
				State: &state{tikv: newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer))},
			},
			expectedStatus: task.SComplete,
			expectedObj:    newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers()),
		},
		{
			desc: "no sub resources, has finalizer, failed to remove finalizer",
			state: &ReconcileContext{
				State: &state{tikv: newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer))},
			},
			unexpectedErr:  true,
			expectedStatus: task.SFail,
			expectedObj:    newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer)),
		},
		{
			desc: "has pods with deletion timestamp, should wait",
			state: &ReconcileContext{
				State: &state{tikv: newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer))},
			},
			subresources:   fakeSubresources("PodDeleting"),
			expectedStatus: task.SRetry,
			expectedObj:    newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer)),
		},
		{
			desc: "has mixed sub resources with some deleting",
			state: &ReconcileContext{
				State: &state{tikv: newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer))},
			},
			subresources: append(
				fakeSubresources("Pod"),
				fakeSubresources("PodDeleting")...,
			),
			expectedStatus: task.SRetry,
			expectedObj:    newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer)),
		},
		{
			desc: "empty tikv with multiple finalizers",
			state: &ReconcileContext{
				State: &state{tikv: newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers(meta.Finalizer, "other.finalizer.com/test", "another.finalizer.com/test"))},
			},
			expectedStatus: task.SComplete,
			expectedObj:    newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers("other.finalizer.com/test", "another.finalizer.com/test")),
		},
		{
			desc: "tikv with only other finalizers",
			state: &ReconcileContext{
				State: &state{tikv: newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers("other.finalizer.com/test", "another.finalizer.com/test"))},
			},
			expectedStatus: task.SComplete,
			expectedObj:    newFakeTiKV(withOfflineSpec(), withOfflineCondition(v1alpha1.OfflineReasonCompleted), withFinalizers("other.finalizer.com/test", "another.finalizer.com/test")),
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			objs := []client.Object{
				c.state.TiKV(),
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

			tikv := &v1alpha1.TiKV{}
			require.NoError(tt, fc.Get(ctx, client.ObjectKey{Name: "test-tikv"}, tikv), c.desc)
			assert.Equal(tt, c.expectedObj, tikv, c.desc)
		})
	}
}

func fakeSubresources(types ...string) []client.Object {
	var objs []client.Object
	for i, t := range types {
		var obj client.Object
		switch t {
		case "Pod":
			obj = fake.FakeObj("pod-"+strconv.Itoa(i), func(obj *corev1.Pod) *corev1.Pod {
				obj.Labels = map[string]string{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyInstance:  "test-tikv",
					v1alpha1.LabelKeyCluster:   "",
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV,
				}
				return obj
			})
		case "PodDeleting":
			obj = fake.FakeObj("pod-deleting-"+strconv.Itoa(i), func(obj *corev1.Pod) *corev1.Pod {
				obj.Labels = map[string]string{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyInstance:  "test-tikv",
					v1alpha1.LabelKeyCluster:   "",
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV,
				}
				now := metav1.NewTime(time.Now())
				obj.SetDeletionTimestamp(&now)
				return obj
			})
		case "ConfigMap":
			obj = fake.FakeObj("cm-"+strconv.Itoa(i), func(obj *corev1.ConfigMap) *corev1.ConfigMap {
				obj.Labels = map[string]string{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyInstance:  "test-tikv",
					v1alpha1.LabelKeyCluster:   "",
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV,
				}
				return obj
			})
		case "PersistentVolumeClaim":
			obj = fake.FakeObj("pvc-"+strconv.Itoa(i), func(obj *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
				obj.Labels = map[string]string{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyInstance:  "test-tikv",
					v1alpha1.LabelKeyCluster:   "",
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV,
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
