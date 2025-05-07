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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TestTaskContextObject(t *testing.T) {
	cases := []struct {
		desc          string
		state         *fakeState[v1alpha1.PD]
		objs          []client.Object
		unexpectedErr bool

		expectedResult task.Status
		expectedObj    *v1alpha1.PD
	}{
		{
			desc: "success",
			state: &fakeState[v1alpha1.PD]{
				ns:   "aaa",
				name: "aaa",
			},
			objs: []client.Object{
				fake.FakeObj("aaa", fake.SetNamespace[v1alpha1.PD]("aaa")),
			},
			expectedResult: task.SComplete,
			expectedObj:    fake.FakeObj("aaa", fake.SetNamespace[v1alpha1.PD]("aaa")),
		},
		{
			desc: "not found",
			state: &fakeState[v1alpha1.PD]{
				ns:   "aaa",
				name: "aaa",
			},
			expectedResult: task.SComplete,
		},
		{
			desc: "has unexpected error",
			state: &fakeState[v1alpha1.PD]{
				ns:   "aaa",
				name: "aaa",
			},
			objs: []client.Object{
				fake.FakeObj("aaa", fake.SetNamespace[v1alpha1.PD]("aaa")),
			},
			unexpectedErr:  true,
			expectedResult: task.SFail,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fc := client.NewFakeClient(c.objs...)

			if c.unexpectedErr {
				fc.WithError("*", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			res, done := task.RunTask(context.Background(), TaskContextObject[scope.PD](c.state, fc))
			assert.Equal(tt, c.expectedResult, res.Status(), c.desc)
			assert.False(tt, done, c.desc)
			assert.Equal(tt, c.expectedObj, c.state.obj, c.desc)
		})
	}
}

func TestTaskContextCluster(t *testing.T) {
	const ns = "aaa"
	const name = "bbb"
	cases := []struct {
		desc          string
		state         *fakeObjectState[*v1alpha1.PD]
		objs          []client.Object
		unexpectedErr bool

		expectedResult task.Status
		expectedObj    *v1alpha1.Cluster
	}{
		{
			desc: "success",
			state: newFakeObjectState(fake.FakeObj(name, func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Namespace = ns
				obj.Spec.Cluster.Name = name
				return obj
			})),
			objs: []client.Object{
				fake.FakeObj(name, fake.SetNamespace[v1alpha1.Cluster](ns)),
			},
			expectedResult: task.SComplete,
			expectedObj:    fake.FakeObj(name, fake.SetNamespace[v1alpha1.Cluster](ns)),
		},
		{
			desc: "not found",
			state: newFakeObjectState(fake.FakeObj(name, func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Namespace = ns
				obj.Spec.Cluster.Name = name
				return obj
			})),
			expectedResult: task.SFail,
		},
		{
			desc: "has unexpected error",
			state: newFakeObjectState(fake.FakeObj(name, func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Namespace = ns
				obj.Spec.Cluster.Name = name
				return obj
			})),
			objs: []client.Object{
				fake.FakeObj(name, fake.SetNamespace[v1alpha1.Cluster](ns)),
			},
			unexpectedErr:  true,
			expectedResult: task.SFail,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fc := client.NewFakeClient(c.objs...)

			if c.unexpectedErr {
				fc.WithError("*", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}
			res, done := task.RunTask(context.Background(), TaskContextCluster[scope.PD](c.state, fc))
			assert.Equal(tt, c.expectedResult, res.Status(), c.desc)
			assert.False(tt, done, c.desc)
			assert.Equal(tt, c.expectedObj, c.state.cluster, c.desc)
		})
	}
}

func TestTaskContextPod(t *testing.T) {
	const name = "bbb"
	cases := []struct {
		desc          string
		state         *fakeObjectState[*v1alpha1.PD]
		objs          []client.Object
		unexpectedErr bool

		expectedResult task.Status
		expectedObj    *corev1.Pod
	}{
		{
			desc: "success",
			state: newFakeObjectState(fake.FakeObj(name, func(obj *v1alpha1.PD) *v1alpha1.PD {
				return obj
			})),
			objs: []client.Object{
				fake.FakeObj(name, fake.InstanceOwner[scope.PD, corev1.Pod](fake.FakeObj[v1alpha1.PD](name))),
			},
			expectedResult: task.SComplete,
			expectedObj:    fake.FakeObj(name, fake.InstanceOwner[scope.PD, corev1.Pod](fake.FakeObj[v1alpha1.PD](name))),
		},
		{
			desc: "not found",
			state: newFakeObjectState(fake.FakeObj(name, func(obj *v1alpha1.PD) *v1alpha1.PD {
				return obj
			})),
			objs:           []client.Object{},
			expectedResult: task.SComplete,
		},
		{
			desc: "has unexpected error",
			state: newFakeObjectState(fake.FakeObj(name, func(obj *v1alpha1.PD) *v1alpha1.PD {
				return obj
			})),
			objs: []client.Object{
				fake.FakeObj(name, fake.InstanceOwner[scope.PD, corev1.Pod](fake.FakeObj[v1alpha1.PD](name))),
			},
			unexpectedErr:  true,
			expectedResult: task.SFail,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fc := client.NewFakeClient(c.objs...)

			if c.unexpectedErr {
				fc.WithError("*", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			res, done := task.RunTask(context.Background(), TaskContextPod[scope.PD](c.state, fc))
			assert.Equal(tt, c.expectedResult, res.Status(), c.desc)
			assert.False(tt, done, c.desc)
			assert.Equal(tt, c.expectedObj, c.state.pod, c.desc)
		})
	}
}

func TestTaskContextPDSlice(t *testing.T) {
	cases := []struct {
		desc          string
		state         *fakeSliceState[v1alpha1.PD]
		objs          []client.Object
		unexpectedErr bool

		expectedResult task.Status
		expectedObjs   []*v1alpha1.PD
	}{
		{
			desc: "success",
			state: &fakeSliceState[v1alpha1.PD]{
				ns: "aaa",
				labels: map[string]string{
					"xxx": "yyy",
				},
			},
			objs: []client.Object{
				fake.FakeObj("aaa", fake.SetNamespace[v1alpha1.PD]("aaa"), fake.Label[v1alpha1.PD]("xxx", "yyy")),
				fake.FakeObj("bbb", fake.SetNamespace[v1alpha1.PD]("aaa"), fake.Label[v1alpha1.PD]("xxx", "yyy"), fake.Label[v1alpha1.PD]("aaa", "bbb")),
				// ns is mismatched
				fake.FakeObj("ccc", fake.SetNamespace[v1alpha1.PD]("bbb"), fake.Label[v1alpha1.PD]("xxx", "yyy")),
				// labels are mismatched
				fake.FakeObj("ddd", fake.SetNamespace[v1alpha1.PD]("aaa"), fake.Label[v1alpha1.PD]("xxx", "zzz")),
			},
			expectedResult: task.SComplete,
			expectedObjs: []*v1alpha1.PD{
				fake.FakeObj("aaa", fake.SetNamespace[v1alpha1.PD]("aaa"), fake.Label[v1alpha1.PD]("xxx", "yyy")),
				fake.FakeObj("bbb", fake.SetNamespace[v1alpha1.PD]("aaa"), fake.Label[v1alpha1.PD]("xxx", "yyy"), fake.Label[v1alpha1.PD]("aaa", "bbb")),
			},
		},
		{
			desc: "need be sorted",
			state: &fakeSliceState[v1alpha1.PD]{
				ns: "aaa",
				labels: map[string]string{
					"xxx": "yyy",
				},
			},
			objs: []client.Object{
				fake.FakeObj("bbb", fake.SetNamespace[v1alpha1.PD]("aaa"), fake.Label[v1alpha1.PD]("xxx", "yyy"), fake.Label[v1alpha1.PD]("aaa", "bbb")),
				fake.FakeObj("aaa", fake.SetNamespace[v1alpha1.PD]("aaa"), fake.Label[v1alpha1.PD]("xxx", "yyy")),
				// ns is mismatched
				fake.FakeObj("ccc", fake.SetNamespace[v1alpha1.PD]("bbb"), fake.Label[v1alpha1.PD]("xxx", "yyy")),
				// labels are mismatched
				fake.FakeObj("ddd", fake.SetNamespace[v1alpha1.PD]("aaa"), fake.Label[v1alpha1.PD]("xxx", "zzz")),
			},
			expectedResult: task.SComplete,
			expectedObjs: []*v1alpha1.PD{
				fake.FakeObj("aaa", fake.SetNamespace[v1alpha1.PD]("aaa"), fake.Label[v1alpha1.PD]("xxx", "yyy")),
				fake.FakeObj("bbb", fake.SetNamespace[v1alpha1.PD]("aaa"), fake.Label[v1alpha1.PD]("xxx", "yyy"), fake.Label[v1alpha1.PD]("aaa", "bbb")),
			},
		},
		{
			desc: "unexpected err",
			state: &fakeSliceState[v1alpha1.PD]{
				ns: "aaa",
				labels: map[string]string{
					"xxx": "yyy",
				},
			},
			unexpectedErr: true,
			objs: []client.Object{
				fake.FakeObj("bbb", fake.SetNamespace[v1alpha1.PD]("aaa"), fake.Label[v1alpha1.PD]("xxx", "yyy"), fake.Label[v1alpha1.PD]("aaa", "bbb")),
				fake.FakeObj("aaa", fake.SetNamespace[v1alpha1.PD]("aaa"), fake.Label[v1alpha1.PD]("xxx", "yyy")),
				// ns is mismatched
				fake.FakeObj("ccc", fake.SetNamespace[v1alpha1.PD]("bbb"), fake.Label[v1alpha1.PD]("xxx", "yyy")),
				// labels are mismatched
				fake.FakeObj("ddd", fake.SetNamespace[v1alpha1.PD]("aaa"), fake.Label[v1alpha1.PD]("xxx", "zzz")),
			},
			expectedResult: task.SFail,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fc := client.NewFakeClient(c.objs...)

			if c.unexpectedErr {
				fc.WithError("*", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			s := &fakePDSliceState{s: c.state}
			res, done := task.RunTask(context.Background(), TaskContextPDSlice(s, fc))
			assert.Equal(tt, c.expectedResult, res.Status(), c.desc)
			assert.False(tt, done, c.desc)
			assert.Equal(tt, c.expectedObjs, c.state.objs, c.desc)
		})
	}
}

func TestTaskSuspendPod(t *testing.T) {
	now := metav1.Now()
	cases := []struct {
		desc          string
		state         *fakeState[corev1.Pod]
		objs          []client.Object
		unexpectedErr bool

		expectedResult task.Status
		expectedObj    *corev1.Pod
	}{
		{
			desc:           "pod is nil",
			state:          &fakeState[corev1.Pod]{},
			expectedResult: task.SComplete,
		},
		{
			desc: "pod is deleting",
			state: &fakeState[corev1.Pod]{
				obj: fake.FakeObj("aaa", fake.DeleteTimestamp[corev1.Pod](&now)),
			},
			expectedResult: task.SComplete,
			expectedObj:    fake.FakeObj("aaa", fake.DeleteTimestamp[corev1.Pod](&now)),
		},
		{
			// means pod has been fully deleted after it is fetched previously
			desc: "pod is deleted",
			state: &fakeState[corev1.Pod]{
				obj: fake.FakeObj[corev1.Pod]("aaa"),
			},
			expectedResult: task.SComplete,
			expectedObj:    fake.FakeObj[corev1.Pod]("aaa"),
		},
		{
			desc: "delete pod",
			state: &fakeState[corev1.Pod]{
				obj: fake.FakeObj[corev1.Pod]("aaa"),
			},
			objs: []client.Object{
				fake.FakeObj[corev1.Pod]("aaa"),
			},
			expectedResult: task.SRetry,
			expectedObj:    fake.FakeObj[corev1.Pod]("aaa"),
		},
		{
			desc: "delete pod with unexpected err",
			state: &fakeState[corev1.Pod]{
				obj: fake.FakeObj[corev1.Pod]("aaa"),
			},
			objs: []client.Object{
				fake.FakeObj[corev1.Pod]("aaa"),
			},
			unexpectedErr:  true,
			expectedResult: task.SFail,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fc := client.NewFakeClient(c.objs...)

			if c.unexpectedErr {
				fc.WithError("*", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			s := &fakePodState{s: c.state}
			res, done := task.RunTask(context.Background(), TaskSuspendPod(s, fc))
			assert.Equal(tt, c.expectedResult, res.Status(), c.desc)
			assert.False(tt, done, c.desc)
			if !c.unexpectedErr {
				assert.Equal(tt, c.expectedObj, c.state.obj, c.desc)
			}
		})
	}
}

func TestFeatureGates(t *testing.T) {
	cases := []struct {
		desc          string
		state         *fakeState[v1alpha1.Cluster]
		objs          []client.Object
		unexpectedErr bool

		expectedResult task.Status
		expectedObj    *corev1.Pod
	}{
		{
			desc: "up to date",
			state: &fakeState[v1alpha1.Cluster]{
				name: "xxx",
				obj: fake.FakeObj("xxx", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
					obj.Generation = 1
					return obj
				}),
			},
			expectedResult: task.SComplete,
		},
		{
			desc: "generation is changed",
			state: &fakeState[v1alpha1.Cluster]{
				name: "xxx",
				obj: fake.FakeObj("xxx", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
					obj.Generation = 2
					return obj
				}),
			},
			expectedResult: task.SFail,
		},
		{
			desc: "uid is changed",
			state: &fakeState[v1alpha1.Cluster]{
				name: "xxx",
				obj: fake.FakeObj("xxx", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
					obj.Generation = 1
					obj.UID = "newuid"
					return obj
				}),
			},
			expectedResult: task.SFail,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			features.Register(fake.FakeObj("xxx", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
				obj.Generation = 1
				return obj
			}))

			s := &fakeClusterState{s: c.state}
			res, done := task.RunTask(context.Background(), TaskFeatureGates(s))
			assert.Equal(tt, c.expectedResult, res.Status(), c.desc)
			assert.False(tt, done, c.desc)
		})
	}
}
