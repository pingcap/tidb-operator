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

package common

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
	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

func TestTaskFinalizerAdd(t *testing.T) {
	cases := []struct {
		desc          string
		state         ObjectState[*v1alpha1.PD]
		unexpectedErr bool

		expectedStatus task.Status
		expectedObj    *v1alpha1.PD
	}{
		{
			desc: "no finalizer",
			state: newFakeObjectState(
				fake.FakeObj[v1alpha1.PD]("aaa"),
			),
			expectedStatus: task.SComplete,
			expectedObj:    fake.FakeObj("aaa", fake.AddFinalizer[v1alpha1.PD]()),
		},
		{
			desc: "no finalizer and cannot call api",
			state: newFakeObjectState(
				fake.FakeObj[v1alpha1.PD]("aaa"),
			),
			unexpectedErr:  true,
			expectedStatus: task.SFail,
		},
		{
			desc: "has another finalizer",
			state: newFakeObjectState(
				fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.SetFinalizers(append(obj.GetFinalizers(), "xxxx"))
					return obj
				}),
			),
			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.SetFinalizers(append(obj.GetFinalizers(), "xxxx", meta.Finalizer))
				return obj
			}),
		},
		{
			desc: "already has the finalizer",
			state: newFakeObjectState(
				fake.FakeObj("aaa", fake.AddFinalizer[v1alpha1.PD]()),
			),
			expectedStatus: task.SComplete,
			expectedObj:    fake.FakeObj("aaa", fake.AddFinalizer[v1alpha1.PD]()),
		},
		{
			desc: "already has the finalizer and cannot call api",
			state: newFakeObjectState(
				fake.FakeObj("aaa", fake.AddFinalizer[v1alpha1.PD]()),
			),
			unexpectedErr:  true,
			expectedStatus: task.SComplete,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fc := client.NewFakeClient(c.state.Object())
			if c.unexpectedErr {
				fc.WithError("*", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			ctx := context.Background()
			res, done := task.RunTask(ctx, TaskFinalizerAdd[scope.PD](c.state, fc))
			assert.Equal(tt, c.expectedStatus, res.Status(), c.desc)
			assert.False(tt, done, c.desc)

			// no need to check update result
			if c.unexpectedErr {
				return
			}

			obj := &v1alpha1.PD{}
			require.NoError(tt, fc.Get(ctx, client.ObjectKey{Name: "aaa"}, obj), c.desc)
			assert.Equal(tt, c.expectedObj, obj, c.desc)
		})
	}
}

type fakeGroupFinalizerDelState[
	G client.Object,
	I client.Object,
] struct {
	g  G
	is []I
}

func (s *fakeGroupFinalizerDelState[G, I]) Object() G {
	return s.g
}

func (s *fakeGroupFinalizerDelState[G, I]) InstanceSlice() []I {
	return s.is
}

func newFakeGroupFinalizerDelState[
	I client.Object,
	G client.Object,
](g G, is ...I) *fakeGroupFinalizerDelState[G, I] {
	return &fakeGroupFinalizerDelState[G, I]{
		g:  g,
		is: is,
	}
}

func TestTaskGroupFinalizerDel(t *testing.T) {
	now := metav1.Now()
	cases := []struct {
		desc          string
		state         GroupFinalizerDelState[*v1alpha1.PDGroup, *v1alpha1.PD]
		subresources  []client.Object
		unexpectedErr bool

		expectedStatus task.Status
		expectedObj    *v1alpha1.PDGroup
	}{
		{
			desc: "no instances and no sub resources and no finalizer",
			state: newFakeGroupFinalizerDelState[*v1alpha1.PD](
				fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now)),
			),
			expectedStatus: task.SComplete,
			expectedObj:    fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now)),
		},
		{
			desc: "no instances and no sub resources",
			state: newFakeGroupFinalizerDelState[*v1alpha1.PD](
				fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), fake.AddFinalizer[v1alpha1.PDGroup]()),
			),
			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.SetFinalizers([]string{})
				return obj
			}),
		},
		{
			desc: "no instances and no sub resources but call api failed",
			state: newFakeGroupFinalizerDelState[*v1alpha1.PD](
				fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), fake.AddFinalizer[v1alpha1.PDGroup]()),
			),
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "no instances but has sub resources",
			state: newFakeGroupFinalizerDelState[*v1alpha1.PD](
				fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), fake.AddFinalizer[v1alpha1.PDGroup]()),
			),
			subresources: []client.Object{
				fake.FakeObj("aaa",
					fake.Label[corev1.Service](v1alpha1.LabelKeyManagedBy, v1alpha1.LabelValManagedByOperator),
					fake.Label[corev1.Service](v1alpha1.LabelKeyCluster, ""),
					fake.Label[corev1.Service](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentPD),
					fake.Label[corev1.Service](v1alpha1.LabelKeyGroup, "aaa"),
				),
			},

			expectedStatus: task.SRetry,
			expectedObj:    fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), fake.AddFinalizer[v1alpha1.PDGroup]()),
		},
		{
			desc: "no instances but has sub resources and call api failed",
			state: newFakeGroupFinalizerDelState[*v1alpha1.PD](
				fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), fake.AddFinalizer[v1alpha1.PDGroup]()),
			),
			subresources: []client.Object{
				fake.FakeObj("aaa",
					fake.Label[corev1.Service](v1alpha1.LabelKeyManagedBy, v1alpha1.LabelValManagedByOperator),
					fake.Label[corev1.Service](v1alpha1.LabelKeyCluster, ""),
					fake.Label[corev1.Service](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentPD),
					fake.Label[corev1.Service](v1alpha1.LabelKeyGroup, "aaa"),
				),
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "has instances with finalizer",
			state: newFakeGroupFinalizerDelState(
				fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), fake.AddFinalizer[v1alpha1.PDGroup]()),
				fake.FakeObj("aaa", fake.AddFinalizer[v1alpha1.PD]()),
			),
			expectedStatus: task.SRetry,
			expectedObj:    fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), fake.AddFinalizer[v1alpha1.PDGroup]()),
		},
		{
			desc: "has instances with finalizer but call api failed",
			state: newFakeGroupFinalizerDelState(
				fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), fake.AddFinalizer[v1alpha1.PDGroup]()),
				fake.FakeObj("aaa", fake.AddFinalizer[v1alpha1.PD]()),
			),
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "has deleting instances with finalizer but call api failed",
			state: newFakeGroupFinalizerDelState(
				fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PDGroup](&now), fake.AddFinalizer[v1alpha1.PDGroup]()),
				fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PD](&now), fake.AddFinalizer[v1alpha1.PD]()),
			),
			unexpectedErr: true,

			expectedStatus: task.SRetry,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			objs := []client.Object{
				c.state.Object(),
			}
			for _, i := range c.state.InstanceSlice() {
				objs = append(objs, i)
			}

			objs = append(objs, c.subresources...)

			fc := client.NewFakeClient(objs...)
			if c.unexpectedErr {
				// cannot remove finalizer
				fc.WithError("update", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
				// cannot delete sub resources
				fc.WithError("delete", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}
			ctx := tt.Context()

			res, done := task.RunTask(ctx, TaskGroupFinalizerDel[scope.PDGroup](c.state, fc))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)

			// no need to check update result
			if c.unexpectedErr {
				return
			}

			obj := &v1alpha1.PDGroup{}
			require.NoError(tt, fc.Get(ctx, client.ObjectKey{Name: "aaa"}, obj), c.desc)
			assert.Equal(tt, c.expectedObj, obj, c.desc)
		})
	}
}

func TestTaskInstanceFinalizerDel(t *testing.T) {
	now := metav1.Now()
	cases := []struct {
		desc          string
		state         ObjectState[*v1alpha1.PD]
		subresources  []client.Object
		lister        SubresourceLister
		unexpectedErr bool

		expectedStatus task.Status
		expectedObj    *v1alpha1.PD
	}{
		{
			desc: "no subresources and no finalizer",
			state: newFakeObjectState(
				fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PD](&now)),
			),
			lister: NewSubresourceLister(
				NewSubresource[corev1.PodList](),
				NewSubresource[corev1.ConfigMapList](),
			),
			expectedStatus: task.SComplete,
			expectedObj:    fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PD](&now)),
		},
		{
			desc: "no subresources but has finalizer",
			state: newFakeObjectState(
				fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PD](&now), fake.AddFinalizer[v1alpha1.PD]()),
			),
			lister: NewSubresourceLister(
				NewSubresource[corev1.PodList](),
				NewSubresource[corev1.ConfigMapList](),
			),
			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PD](&now), func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.SetFinalizers([]string{})
				return obj
			}),
		},
		{
			desc: "no subresources but call api failed",
			state: newFakeObjectState(
				fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PD](&now), fake.AddFinalizer[v1alpha1.PD]()),
			),
			lister: NewSubresourceLister(
				NewSubresource[corev1.PodList](),
				NewSubresource[corev1.ConfigMapList](),
			),
			unexpectedErr:  true,
			expectedStatus: task.SFail,
		},
		{
			desc: "has pod subresource",
			state: newFakeObjectState(
				fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PD](&now), fake.AddFinalizer[v1alpha1.PD]()),
			),
			subresources: []client.Object{
				fake.FakeObj("aaa",
					fake.Label[corev1.Pod](v1alpha1.LabelKeyManagedBy, v1alpha1.LabelValManagedByOperator),
					fake.Label[corev1.Pod](v1alpha1.LabelKeyCluster, ""),
					fake.Label[corev1.Pod](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentPD),
					fake.Label[corev1.Pod](v1alpha1.LabelKeyInstance, "aaa"),
				),
			},
			lister: NewSubresourceLister(
				NewSubresource[corev1.PodList](),
				NewSubresource[corev1.ConfigMapList](),
			),
			expectedStatus: task.SRetry,
			expectedObj:    fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PD](&now), fake.AddFinalizer[v1alpha1.PD]()),
		},
		{
			desc: "has configmap subresource",
			state: newFakeObjectState(
				fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PD](&now), fake.AddFinalizer[v1alpha1.PD]()),
			),
			subresources: []client.Object{
				fake.FakeObj("aaa",
					fake.Label[corev1.ConfigMap](v1alpha1.LabelKeyManagedBy, v1alpha1.LabelValManagedByOperator),
					fake.Label[corev1.ConfigMap](v1alpha1.LabelKeyCluster, ""),
					fake.Label[corev1.ConfigMap](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentPD),
					fake.Label[corev1.ConfigMap](v1alpha1.LabelKeyInstance, "aaa"),
				),
			},
			lister: NewSubresourceLister(
				NewSubresource[corev1.PodList](),
				NewSubresource[corev1.ConfigMapList](),
			),
			expectedStatus: task.SRetry,
			expectedObj:    fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PD](&now), fake.AddFinalizer[v1alpha1.PD]()),
		},
		{
			desc: "has pvc subresource",
			state: newFakeObjectState(
				fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PD](&now), fake.AddFinalizer[v1alpha1.PD]()),
			),
			subresources: []client.Object{
				fake.FakeObj("aaa",
					fake.Label[corev1.PersistentVolumeClaim](v1alpha1.LabelKeyManagedBy, v1alpha1.LabelValManagedByOperator),
					fake.Label[corev1.PersistentVolumeClaim](v1alpha1.LabelKeyCluster, ""),
					fake.Label[corev1.PersistentVolumeClaim](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentPD),
					fake.Label[corev1.PersistentVolumeClaim](v1alpha1.LabelKeyInstance, "aaa"),
				),
			},
			lister: NewSubresourceLister(
				NewSubresource[corev1.PersistentVolumeClaimList](),
			),
			expectedStatus: task.SRetry,
			expectedObj:    fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PD](&now), fake.AddFinalizer[v1alpha1.PD]()),
		},
		{
			desc: "has multiple subresources",
			state: newFakeObjectState(
				fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PD](&now), fake.AddFinalizer[v1alpha1.PD]()),
			),
			subresources: []client.Object{
				fake.FakeObj("aaa-pod",
					fake.Label[corev1.Pod](v1alpha1.LabelKeyManagedBy, v1alpha1.LabelValManagedByOperator),
					fake.Label[corev1.Pod](v1alpha1.LabelKeyCluster, ""),
					fake.Label[corev1.Pod](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentPD),
					fake.Label[corev1.Pod](v1alpha1.LabelKeyInstance, "aaa"),
				),
				fake.FakeObj("aaa-cm",
					fake.Label[corev1.ConfigMap](v1alpha1.LabelKeyManagedBy, v1alpha1.LabelValManagedByOperator),
					fake.Label[corev1.ConfigMap](v1alpha1.LabelKeyCluster, ""),
					fake.Label[corev1.ConfigMap](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentPD),
					fake.Label[corev1.ConfigMap](v1alpha1.LabelKeyInstance, "aaa"),
				),
				fake.FakeObj("aaa-pvc",
					fake.Label[corev1.PersistentVolumeClaim](v1alpha1.LabelKeyManagedBy, v1alpha1.LabelValManagedByOperator),
					fake.Label[corev1.PersistentVolumeClaim](v1alpha1.LabelKeyCluster, ""),
					fake.Label[corev1.PersistentVolumeClaim](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentPD),
					fake.Label[corev1.PersistentVolumeClaim](v1alpha1.LabelKeyInstance, "aaa"),
				),
			},
			lister:         DefaultInstanceSubresourceLister,
			expectedStatus: task.SRetry,
			expectedObj:    fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PD](&now), fake.AddFinalizer[v1alpha1.PD]()),
		},
		{
			desc: "has subresources and API call fails",
			state: newFakeObjectState(
				fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PD](&now), fake.AddFinalizer[v1alpha1.PD]()),
			),
			subresources: []client.Object{
				fake.FakeObj("aaa",
					fake.Label[corev1.Pod](v1alpha1.LabelKeyManagedBy, v1alpha1.LabelValManagedByOperator),
					fake.Label[corev1.Pod](v1alpha1.LabelKeyCluster, ""),
					fake.Label[corev1.Pod](v1alpha1.LabelKeyComponent, v1alpha1.LabelValComponentPD),
					fake.Label[corev1.Pod](v1alpha1.LabelKeyInstance, "aaa"),
				),
			},
			lister: NewSubresourceLister(
				NewSubresource[corev1.PodList](),
			),
			unexpectedErr:  true,
			expectedStatus: task.SFail,
		},
		{
			desc: "empty lister",
			state: newFakeObjectState(
				fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PD](&now), fake.AddFinalizer[v1alpha1.PD]()),
			),
			lister:         NewSubresourceLister(),
			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", fake.DeleteTimestamp[v1alpha1.PD](&now), func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.SetFinalizers([]string{})
				return obj
			}),
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			objs := []client.Object{
				c.state.Object(),
			}
			objs = append(objs, c.subresources...)

			fc := client.NewFakeClient(objs...)
			if c.unexpectedErr {
				// cannot remove finalizer
				fc.WithError("update", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
				// cannot delete sub resources
				fc.WithError("delete", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}
			ctx := tt.Context()

			res, done := task.RunTask(ctx, TaskInstanceFinalizerDel[scope.PD](c.state, fc, c.lister))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)

			// no need to check update result
			if c.unexpectedErr {
				return
			}

			obj := &v1alpha1.PD{}
			require.NoError(tt, fc.Get(ctx, client.ObjectKey{Name: "aaa"}, obj), c.desc)
			assert.Equal(tt, c.expectedObj, obj, c.desc)
		})
	}
}
