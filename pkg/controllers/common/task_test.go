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

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TestTaskContextPD(t *testing.T) {
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

			s := &fakePDState{s: c.state}
			res, done := task.RunTask(context.Background(), TaskContextPD(s, fc))
			assert.Equal(tt, c.expectedResult, res.Status(), c.desc)
			assert.False(tt, done, c.desc)
			assert.Equal(tt, c.expectedObj, c.state.obj, c.desc)
		})
	}
}

func TestTaskContextCluster(t *testing.T) {
	cases := []struct {
		desc          string
		state         *fakeState[v1alpha1.Cluster]
		objs          []client.Object
		unexpectedErr bool

		expectedResult task.Status
		expectedObj    *v1alpha1.Cluster
	}{
		{
			desc: "success",
			state: &fakeState[v1alpha1.Cluster]{
				ns:   "aaa",
				name: "aaa",
			},
			objs: []client.Object{
				fake.FakeObj("aaa", fake.SetNamespace[v1alpha1.Cluster]("aaa")),
			},
			expectedResult: task.SComplete,
			expectedObj:    fake.FakeObj("aaa", fake.SetNamespace[v1alpha1.Cluster]("aaa")),
		},
		{
			desc: "not found",
			state: &fakeState[v1alpha1.Cluster]{
				ns:   "aaa",
				name: "aaa",
			},
			expectedResult: task.SFail,
		},
		{
			desc: "has unexpected error",
			state: &fakeState[v1alpha1.Cluster]{
				ns:   "aaa",
				name: "aaa",
			},
			objs: []client.Object{
				fake.FakeObj("aaa", fake.SetNamespace[v1alpha1.Cluster]("aaa")),
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
			s := &fakeClusterState{s: c.state}

			res, done := task.RunTask(context.Background(), TaskContextCluster(s, fc))
			assert.Equal(tt, c.expectedResult, res.Status(), c.desc)
			assert.False(tt, done, c.desc)
			assert.Equal(tt, c.expectedObj, c.state.obj, c.desc)
		})
	}
}

func TestTaskContextPod(t *testing.T) {
	cases := []struct {
		desc          string
		state         *fakeState[corev1.Pod]
		objs          []client.Object
		unexpectedErr bool

		expectedResult task.Status
		expectedObj    *corev1.Pod
	}{
		{
			desc: "success",
			state: &fakeState[corev1.Pod]{
				ns:   "aaa",
				name: "aaa",
			},
			objs: []client.Object{
				fake.FakeObj("aaa", fake.SetNamespace[corev1.Pod]("aaa")),
			},
			expectedResult: task.SComplete,
			expectedObj:    fake.FakeObj("aaa", fake.SetNamespace[corev1.Pod]("aaa")),
		},
		{
			desc: "not found",
			state: &fakeState[corev1.Pod]{
				ns:   "aaa",
				name: "aaa",
			},
			expectedResult: task.SComplete,
		},
		{
			desc: "has unexpected error",
			state: &fakeState[corev1.Pod]{
				ns:   "aaa",
				name: "aaa",
			},
			objs: []client.Object{
				fake.FakeObj("aaa", fake.SetNamespace[corev1.Pod]("aaa")),
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
			res, done := task.RunTask(context.Background(), TaskContextPod(s, fc))
			assert.Equal(tt, c.expectedResult, res.Status(), c.desc)
			assert.False(tt, done, c.desc)
			assert.Equal(tt, c.expectedObj, c.state.obj, c.desc)
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
			expectedResult: task.SWait,
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

func TestTaskGroupFinalizerAdd(t *testing.T) {
	t.Run("PDGroup", testTaskGroupFinalizerAdd[runtime.PDGroupTuple])
	t.Run("TiKVGroup", testTaskGroupFinalizerAdd[runtime.TiKVGroupTuple])
	t.Run("TiDBGroup", testTaskGroupFinalizerAdd[runtime.TiDBGroupTuple])
	t.Run("TiFlashGroup", testTaskGroupFinalizerAdd[runtime.TiFlashGroupTuple])
}

func testTaskGroupFinalizerAdd[
	GT runtime.GroupTuple[OG, RG],
	OG client.Object,
	RG runtime.GroupT[G],
	G runtime.GroupSet,
](t *testing.T) {
	cases := []struct {
		desc          string
		state         GroupState[RG]
		unexpectedErr bool

		expectedStatus task.Status
		expectedObj    RG
	}{
		{
			desc: "no finalizer",
			state: FakeGroupState(
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					return obj
				}),
			),
			expectedStatus: task.SComplete,
			expectedObj: fake.Fake(func(obj RG) RG {
				obj.SetName("aaa")
				obj.SetFinalizers([]string{v1alpha1.Finalizer})
				return obj
			}),
		},
		{
			desc: "no finalizer and cannot call api",
			state: FakeGroupState(
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					return obj
				}),
			),
			unexpectedErr:  true,
			expectedStatus: task.SFail,
		},
		{
			desc: "has another finalizer",
			state: FakeGroupState(
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetFinalizers(append(obj.GetFinalizers(), "xxxx"))
					return obj
				}),
			),
			expectedStatus: task.SComplete,
			expectedObj: fake.Fake(func(obj RG) RG {
				obj.SetName("aaa")
				obj.SetFinalizers(append(obj.GetFinalizers(), "xxxx", v1alpha1.Finalizer))
				return obj
			}),
		},
		{
			desc: "already has the finalizer",
			state: FakeGroupState(
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetFinalizers(append(obj.GetFinalizers(), v1alpha1.Finalizer))
					return obj
				}),
			),
			expectedStatus: task.SComplete,
			expectedObj: fake.Fake(func(obj RG) RG {
				obj.SetName("aaa")
				obj.SetFinalizers(append(obj.GetFinalizers(), v1alpha1.Finalizer))
				return obj
			}),
		},
		{
			desc: "already has the finalizer and cannot call api",
			state: FakeGroupState(
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetFinalizers(append(obj.GetFinalizers(), v1alpha1.Finalizer))
					return obj
				}),
			),
			unexpectedErr:  true,
			expectedStatus: task.SComplete,
		},
	}

	var gt GT
	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fc := client.NewFakeClient(gt.To(c.state.Group()))
			if c.unexpectedErr {
				fc.WithError("*", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			ctx := context.Background()
			res, done := task.RunTask(ctx, TaskGroupFinalizerAdd[GT](c.state, fc))
			assert.Equal(tt, c.expectedStatus, res.Status(), c.desc)
			assert.False(tt, done, c.desc)

			// no need to check update result
			if c.unexpectedErr {
				return
			}

			obj := gt.To(new(G))
			require.NoError(tt, fc.Get(ctx, client.ObjectKey{Name: "aaa"}, obj), c.desc)
			assert.Equal(tt, c.expectedObj, gt.From(obj), c.desc)
		})
	}
}

func TestTaskGroupStatusSuspend(t *testing.T) {
	t.Run("PDGroup", testTaskGroupStatusSuspend[runtime.PDGroupTuple, runtime.PD])
	t.Run("TiKVGroup", testTaskGroupStatusSuspend[runtime.TiKVGroupTuple, runtime.TiKV])
	t.Run("TiDBGroup", testTaskGroupStatusSuspend[runtime.TiDBGroupTuple, runtime.TiDB])
	t.Run("TiFlashGroup", testTaskGroupStatusSuspend[runtime.TiFlashGroupTuple, runtime.TiFlash])
}

func testTaskGroupStatusSuspend[
	GT runtime.GroupTuple[OG, RG],
	I runtime.InstanceSet,
	OG client.Object,
	RG runtime.GroupT[G],
	G runtime.GroupSet,
	RI runtime.InstanceT[I],
](t *testing.T) {
	cases := []struct {
		desc          string
		state         GroupAndInstanceSliceState[RG, RI]
		unexpectedErr bool

		expectedStatus task.Status
		expectedObj    RG
	}{
		{
			desc: "no instance",
			state: FakeGroupAndInstanceSliceState[RG, RI](
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetGeneration(3)
					return obj
				}),
			),

			expectedStatus: task.SComplete,
			expectedObj: fake.Fake(func(obj RG) RG {
				obj.SetName("aaa")
				obj.SetGeneration(3)
				obj.SetObservedGeneration(3)
				obj.SetConditions([]metav1.Condition{
					{
						Type:               v1alpha1.CondSuspended,
						Status:             metav1.ConditionTrue,
						ObservedGeneration: 3,
						Reason:             v1alpha1.ReasonSuspended,
						Message:            "group is suspended",
					},
				})
				return obj
			}),
		},
		{
			desc: "all instances are suspended",
			state: FakeGroupAndInstanceSliceState(
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetGeneration(3)
					return obj
				}),
				fake.Fake(func(obj RI) RI {
					obj.SetName("aaa")
					obj.SetConditions([]metav1.Condition{
						{
							Type:   v1alpha1.CondSuspended,
							Status: metav1.ConditionTrue,
						},
					})
					return obj
				}),
			),

			expectedStatus: task.SComplete,
			expectedObj: fake.Fake(func(obj RG) RG {
				obj.SetName("aaa")
				obj.SetGeneration(3)
				obj.SetObservedGeneration(3)
				obj.SetConditions([]metav1.Condition{
					{
						Type:               v1alpha1.CondSuspended,
						Status:             metav1.ConditionTrue,
						ObservedGeneration: 3,
						Reason:             v1alpha1.ReasonSuspended,
						Message:            "group is suspended",
					},
				})
				return obj
			}),
		},
		{
			desc: "one instance is not suspended",
			state: FakeGroupAndInstanceSliceState(
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetGeneration(3)
					return obj
				}),
				fake.Fake(func(obj RI) RI {
					obj.SetName("aaa")
					obj.SetConditions([]metav1.Condition{
						{
							Type:   v1alpha1.CondSuspended,
							Status: metav1.ConditionFalse,
						},
					})
					return obj
				}),
			),

			expectedStatus: task.SComplete,
			expectedObj: fake.Fake(func(obj RG) RG {
				obj.SetName("aaa")
				obj.SetGeneration(3)
				obj.SetObservedGeneration(3)
				obj.SetConditions([]metav1.Condition{
					{
						Type:               v1alpha1.CondSuspended,
						Status:             metav1.ConditionFalse,
						ObservedGeneration: 3,
						Reason:             v1alpha1.ReasonSuspending,
						Message:            "group is suspending",
					},
				})
				return obj
			}),
		},
		{
			desc: "all instances are suspended but cannot call api",
			state: FakeGroupAndInstanceSliceState(
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetGeneration(3)
					return obj
				}),
				fake.Fake(func(obj RI) RI {
					obj.SetName("aaa")
					obj.SetConditions([]metav1.Condition{
						{
							Type:   v1alpha1.CondSuspended,
							Status: metav1.ConditionTrue,
						},
					})
					return obj
				}),
			),
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "all instances are suspended and group is up to date and cannot call api",
			state: FakeGroupAndInstanceSliceState(
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetGeneration(3)
					obj.SetObservedGeneration(3)
					obj.SetConditions([]metav1.Condition{
						{
							Type:               v1alpha1.CondSuspended,
							Status:             metav1.ConditionTrue,
							ObservedGeneration: 3,
							Reason:             v1alpha1.ReasonSuspended,
							Message:            "group is suspended",
						},
					})
					return obj
				}),
				fake.Fake(func(obj RI) RI {
					obj.SetName("aaa")
					obj.SetConditions([]metav1.Condition{
						{
							Type:   v1alpha1.CondSuspended,
							Status: metav1.ConditionTrue,
						},
					})
					return obj
				}),
			),
			unexpectedErr: true,

			expectedStatus: task.SComplete,
		},
	}

	var gt GT
	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fc := client.NewFakeClient(gt.To(c.state.Group()))
			if c.unexpectedErr {
				fc.WithError("*", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			ctx := context.Background()
			res, done := task.RunTask(ctx, TaskGroupStatusSuspend[GT](c.state, fc))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)

			// no need to check update result
			if c.unexpectedErr {
				return
			}

			obj := gt.To(new(G))
			require.NoError(tt, fc.Get(ctx, client.ObjectKey{Name: "aaa"}, obj), c.desc)
			rg := gt.From(obj)
			conds := rg.Conditions()
			for i := range conds {
				cond := &conds[i]
				cond.LastTransitionTime = metav1.Time{}
			}
			assert.Equal(tt, c.expectedObj, rg, c.desc)
		})
	}
}
