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
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func TestTaskStatusPersister(t *testing.T) {
	cases := []struct {
		desc          string
		obj           *v1alpha1.PD
		statusChanged bool

		unexpectedErr bool

		expectedStatus task.Status
		expectedObj    *v1alpha1.PD
	}{
		{
			desc: "not changed",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				return obj
			}),
			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				return obj
			}),
		},
		{
			desc: "generation is changed",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Generation = 10
				return obj
			}),
			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Status.IsLeader = true
				obj.Generation = 10
				obj.Status.ObservedGeneration = 10
				return obj
			}),
		},
		{
			desc: "status is changed",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				return obj
			}),
			statusChanged:  true,
			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Status.IsLeader = true
				return obj
			}),
		},
		{
			desc: "status is changed",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				return obj
			}),
			statusChanged:  true,
			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Status.IsLeader = true
				return obj
			}),
		},
		{
			desc: "failed to update status",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				return obj
			}),
			statusChanged:  true,
			unexpectedErr:  true,
			expectedStatus: task.SFail,
		},
	}
	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			fc := client.NewFakeClient(c.obj.DeepCopy())

			if c.unexpectedErr {
				fc.WithError("*", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			c.obj.Status.IsLeader = true

			ctrl := gomock.NewController(tt)
			state := NewMockStatusPersister[*v1alpha1.PD](ctrl)
			state.EXPECT().Object().Return(c.obj)
			state.EXPECT().IsStatusChanged().Return(c.statusChanged)

			ctx := context.Background()
			res, done := task.RunTask(ctx, TaskStatusPersister[scope.PD](state, fc))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)
			if c.unexpectedErr {
				return
			}
			obj := &v1alpha1.PD{}
			require.NoError(tt, fc.Get(ctx, client.ObjectKey{Name: "aaa"}, obj), c.desc)
			assert.Equal(tt, c.expectedObj, obj, c.desc)
		})
	}
}

func TestTaskInstanceConditionSuspended(t *testing.T) {
	cases := []struct {
		desc    string
		obj     *v1alpha1.PD
		cluster *v1alpha1.Cluster
		pod     *corev1.Pod

		expectedStatusChanged bool
		expectedStatus        task.Status
		expectedObj           *v1alpha1.PD
	}{
		{
			desc: "no pod and not suspend",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				return obj
			}),
			cluster: fake.FakeObj("aaa", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
				return obj
			}),
			expectedStatusChanged: true,

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Unsuspended(),
				}
				return obj
			}),
		},
		{
			desc: "no pod and not suspend, not changed",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Unsuspended(),
				}
				return obj
			}),
			cluster: fake.FakeObj("aaa", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
				return obj
			}),

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Unsuspended(),
				}
				return obj
			}),
		},
		{
			desc: "no pod and suspend",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				return obj
			}),
			cluster: fake.FakeObj("aaa", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
				obj.Spec.SuspendAction = &v1alpha1.SuspendAction{
					SuspendCompute: true,
				}
				return obj
			}),
			expectedStatusChanged: true,

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Suspended(),
				}
				return obj
			}),
		},
		{
			desc: "has pod and suspend",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				return obj
			}),
			cluster: fake.FakeObj("aaa", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
				obj.Spec.SuspendAction = &v1alpha1.SuspendAction{
					SuspendCompute: true,
				}
				return obj
			}),
			pod: fake.FakeObj("aaa", func(obj *corev1.Pod) *corev1.Pod {
				return obj
			}),

			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Suspending(),
				}
				return obj
			}),
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctrl := gomock.NewController(tt)
			state := NewMockInstanceCondSuspendedUpdater[*v1alpha1.PD](ctrl)
			state.EXPECT().Object().Return(c.obj)
			state.EXPECT().Cluster().Return(c.cluster)
			state.EXPECT().Pod().Return(c.pod)
			if c.expectedStatusChanged {
				state.EXPECT().SetStatusChanged()
			}

			ctx := context.Background()
			res, done := task.RunTask(ctx, TaskInstanceConditionSuspended[scope.PD](state))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)
			// ignore time
			for i := range c.obj.Status.Conditions {
				cond := &c.obj.Status.Conditions[i]
				cond.LastTransitionTime = metav1.Time{}
			}
			assert.Equal(tt, c.expectedObj, c.obj, c.desc)
		})
	}
}

func TestTaskGroupConditionSuspended(t *testing.T) {
	cases := []struct {
		desc      string
		obj       *v1alpha1.PDGroup
		instances []*v1alpha1.PD
		cluster   *v1alpha1.Cluster

		expectedStatusChanged bool
		expectedStatus        task.Status
		expectedObj           *v1alpha1.PDGroup
	}{
		{
			desc: "no instance and not suspend",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				return obj
			}),
			cluster: fake.FakeObj("aaa", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
				return obj
			}),
			expectedStatusChanged: true,

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Unsuspended(),
				}
				return obj
			}),
		},
		{
			desc: "no instance and not suspend, not changed",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Unsuspended(),
				}
				return obj
			}),
			cluster: fake.FakeObj("aaa", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
				return obj
			}),

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Unsuspended(),
				}
				return obj
			}),
		},
		{
			desc: "no instance and suspend",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				return obj
			}),
			cluster: fake.FakeObj("aaa", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
				obj.Spec.SuspendAction = &v1alpha1.SuspendAction{
					SuspendCompute: true,
				}
				return obj
			}),
			expectedStatusChanged: true,

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Suspended(),
				}
				return obj
			}),
		},
		{
			desc: "no instance and suspend, not changed",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Suspended(),
				}
				return obj
			}),
			cluster: fake.FakeObj("aaa", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
				obj.Spec.SuspendAction = &v1alpha1.SuspendAction{
					SuspendCompute: true,
				}
				return obj
			}),

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Suspended(),
				}
				return obj
			}),
		},
		{
			desc: "all instances are suspended",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				return obj
			}),
			cluster: fake.FakeObj("aaa", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
				obj.Spec.SuspendAction = &v1alpha1.SuspendAction{
					SuspendCompute: true,
				}
				return obj
			}),
			instances: []*v1alpha1.PD{
				fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Status.Conditions = []metav1.Condition{
						*coreutil.Suspended(),
					}
					return obj
				}),
				fake.FakeObj("bbb", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Status.Conditions = []metav1.Condition{
						*coreutil.Suspended(),
					}
					return obj
				}),
			},

			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Suspended(),
				}
				return obj
			}),
		},
		{
			desc: "one instance is not suspended",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				return obj
			}),
			cluster: fake.FakeObj("aaa", func(obj *v1alpha1.Cluster) *v1alpha1.Cluster {
				obj.Spec.SuspendAction = &v1alpha1.SuspendAction{
					SuspendCompute: true,
				}
				return obj
			}),
			instances: []*v1alpha1.PD{
				fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Status.Conditions = []metav1.Condition{
						*coreutil.Suspended(),
					}
					return obj
				}),
				fake.FakeObj("bbb", func(obj *v1alpha1.PD) *v1alpha1.PD {
					return obj
				}),
			},

			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Suspending(),
				}
				return obj
			}),
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctrl := gomock.NewController(tt)
			state := NewMockGroupCondSuspendedUpdater[*v1alpha1.PDGroup, *runtime.PD](ctrl)
			state.EXPECT().Object().Return(c.obj)
			state.EXPECT().Cluster().Return(c.cluster)
			state.EXPECT().Slice().Return(runtime.FromPDSlice(c.instances))
			if c.expectedStatusChanged {
				state.EXPECT().SetStatusChanged()
			}

			ctx := context.Background()
			res, done := task.RunTask(ctx, TaskGroupConditionSuspended[scope.PDGroup](state))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)
			// ignore time
			for i := range c.obj.Status.Conditions {
				cond := &c.obj.Status.Conditions[i]
				cond.LastTransitionTime = metav1.Time{}
			}
			assert.Equal(tt, c.expectedObj, c.obj, c.desc)
		})
	}
}

func TestTaskGroupConditionReady(t *testing.T) {
	cases := []struct {
		desc      string
		obj       *v1alpha1.PDGroup
		instances []*v1alpha1.PD

		expectedStatusChanged bool
		expectedStatus        task.Status
		expectedObj           *v1alpha1.PDGroup
	}{
		{
			desc: "no instances",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				return obj
			}),

			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Unready(),
				}
				return obj
			}),
		},
		{
			desc: "no instances, not changed",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Unready(),
				}
				return obj
			}),
			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Unready(),
				}
				return obj
			}),
		},
		{
			desc: "all instances are ready",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				return obj
			}),
			instances: []*v1alpha1.PD{
				fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Status.Conditions = []metav1.Condition{
						*coreutil.Ready(),
					}
					return obj
				}),
				fake.FakeObj("bbb", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Status.Conditions = []metav1.Condition{
						*coreutil.Ready(),
					}
					return obj
				}),
			},

			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Ready(),
				}
				return obj
			}),
		},
		{
			desc: "one instance is not ready",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				return obj
			}),
			instances: []*v1alpha1.PD{
				fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Status.Conditions = []metav1.Condition{
						*coreutil.Ready(),
					}
					return obj
				}),
				fake.FakeObj("bbb", func(obj *v1alpha1.PD) *v1alpha1.PD {
					return obj
				}),
			},

			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Unready(),
				}
				return obj
			}),
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctrl := gomock.NewController(tt)
			state := NewMockGroupCondReadyUpdater[*v1alpha1.PDGroup, *runtime.PD](ctrl)
			state.EXPECT().Object().Return(c.obj)
			state.EXPECT().Slice().Return(runtime.FromPDSlice(c.instances))
			if c.expectedStatusChanged {
				state.EXPECT().SetStatusChanged()
			}

			ctx := context.Background()
			res, done := task.RunTask(ctx, TaskGroupConditionReady[scope.PDGroup](state))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)
			// ignore time
			for i := range c.obj.Status.Conditions {
				cond := &c.obj.Status.Conditions[i]
				cond.LastTransitionTime = metav1.Time{}
			}
			assert.Equal(tt, c.expectedObj, c.obj, c.desc)
		})
	}
}

func TestTaskGroupConditionSynced(t *testing.T) {
	const newRevision = "new"
	const oldRevision = "old"
	cases := []struct {
		desc      string
		obj       *v1alpha1.PDGroup
		instances []*v1alpha1.PD

		expectedStatusChanged bool
		expectedStatus        task.Status
		expectedObj           *v1alpha1.PDGroup
	}{
		{
			desc: "no instances and 0 desired replicas",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Spec.Replicas = ptr.To[int32](0)
				return obj
			}),

			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Synced(),
				}
				return obj
			}),
		},
		{
			desc: "no instances and 1 desired replicas",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				return obj
			}),

			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Unsynced(),
				}
				return obj
			}),
		},
		{
			desc: "no instances and 1 desired replicas, not changed",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Unsynced(),
				}
				return obj
			}),
			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Unsynced(),
				}
				return obj
			}),
		},
		{
			desc: "all instances are updated",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Spec.Replicas = ptr.To[int32](2)
				return obj
			}),
			instances: []*v1alpha1.PD{
				fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Status.CurrentRevision = newRevision
					return obj
				}),
				fake.FakeObj("bbb", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Status.CurrentRevision = newRevision
					return obj
				}),
			},

			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Synced(),
				}
				return obj
			}),
		},
		{
			desc: "one instance is not updated",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Spec.Replicas = ptr.To[int32](2)
				return obj
			}),
			instances: []*v1alpha1.PD{
				fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Status.CurrentRevision = newRevision
					return obj
				}),
				fake.FakeObj("bbb", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Status.CurrentRevision = oldRevision
					return obj
				}),
			},

			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Unsynced(),
				}
				return obj
			}),
		},
		{
			desc: "only one instance but desired is two",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Spec.Replicas = ptr.To[int32](2)
				return obj
			}),
			instances: []*v1alpha1.PD{
				fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Status.CurrentRevision = newRevision
					return obj
				}),
			},

			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.Conditions = []metav1.Condition{
					*coreutil.Unsynced(),
				}
				return obj
			}),
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctrl := gomock.NewController(tt)
			state := NewMockGroupCondSyncedUpdater[*v1alpha1.PDGroup, *runtime.PD](ctrl)
			state.EXPECT().Object().Return(c.obj)
			state.EXPECT().Revision().Return(newRevision, oldRevision, int32(0))
			state.EXPECT().Slice().Return(runtime.FromPDSlice(c.instances))
			if c.expectedStatusChanged {
				state.EXPECT().SetStatusChanged()
			}

			ctx := context.Background()
			res, done := task.RunTask(ctx, TaskGroupConditionSynced[scope.PDGroup](state))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)
			// ignore time
			for i := range c.obj.Status.Conditions {
				cond := &c.obj.Status.Conditions[i]
				cond.LastTransitionTime = metav1.Time{}
			}
			assert.Equal(tt, c.expectedObj.Status, c.obj.Status, c.desc)
		})
	}
}

func TestTaskStatusRevisionAndReplicas(t *testing.T) {
	const newRevision = "new"
	const oldRevision = "old"
	cases := []struct {
		desc      string
		obj       *v1alpha1.PDGroup
		instances []*v1alpha1.PD

		expectedStatusChanged bool
		expectedStatus        task.Status
		expectedObj           *v1alpha1.PDGroup
	}{
		{
			desc: "no instances and 1 desired",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				return obj
			}),

			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.UpdateRevision = newRevision
				obj.Status.CurrentRevision = oldRevision
				return obj
			}),
		},
		{
			desc: "no instances and 1 desired, not changed",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.UpdateRevision = newRevision
				obj.Status.CurrentRevision = oldRevision
				return obj
			}),

			expectedStatus: task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.UpdateRevision = newRevision
				obj.Status.CurrentRevision = oldRevision
				return obj
			}),
		},
		{
			desc: "no instances and 0 desired",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Spec.Replicas = ptr.To[int32](0)
				return obj
			}),

			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.UpdateRevision = newRevision
				obj.Status.CurrentRevision = newRevision
				return obj
			}),
		},
		{
			desc: "2 updated instances and 1 desired",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				return obj
			}),
			instances: []*v1alpha1.PD{
				fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Status.CurrentRevision = newRevision
					return obj
				}),
				fake.FakeObj("bbb", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Status.CurrentRevision = newRevision
					return obj
				}),
			},

			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.UpdateRevision = newRevision
				obj.Status.CurrentRevision = oldRevision
				obj.Status.Replicas = 2
				obj.Status.ReadyReplicas = 0
				obj.Status.UpdatedReplicas = 2
				obj.Status.CurrentReplicas = 0
				return obj
			}),
		},
		{
			desc: "2 updated instances and 2 desired",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Spec.Replicas = ptr.To[int32](2)
				return obj
			}),
			instances: []*v1alpha1.PD{
				fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Status.CurrentRevision = newRevision
					return obj
				}),
				fake.FakeObj("bbb", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Status.CurrentRevision = newRevision
					return obj
				}),
			},

			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.UpdateRevision = newRevision
				obj.Status.CurrentRevision = newRevision
				obj.Status.Replicas = 2
				obj.Status.ReadyReplicas = 0
				obj.Status.UpdatedReplicas = 2
				obj.Status.CurrentReplicas = 2
				return obj
			}),
		},
		{
			desc: "1 updated instances and 2 desired",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Spec.Replicas = ptr.To[int32](2)
				return obj
			}),
			instances: []*v1alpha1.PD{
				fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Status.CurrentRevision = newRevision
					return obj
				}),
				fake.FakeObj("bbb", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Status.CurrentRevision = oldRevision
					return obj
				}),
			},

			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.UpdateRevision = newRevision
				obj.Status.CurrentRevision = oldRevision
				obj.Status.Replicas = 2
				obj.Status.ReadyReplicas = 0
				obj.Status.UpdatedReplicas = 1
				obj.Status.CurrentReplicas = 1
				return obj
			}),
		},
		{
			desc: "2 updated and ready instances and 2 desired",
			obj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Spec.Replicas = ptr.To[int32](2)
				return obj
			}),
			instances: []*v1alpha1.PD{
				fake.FakeObj("aaa", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Status.CurrentRevision = newRevision
					obj.Status.Conditions = []metav1.Condition{
						*coreutil.Ready(),
					}
					return obj
				}),
				fake.FakeObj("bbb", func(obj *v1alpha1.PD) *v1alpha1.PD {
					obj.Status.CurrentRevision = newRevision
					obj.Status.Conditions = []metav1.Condition{
						*coreutil.Ready(),
					}
					return obj
				}),
			},

			expectedStatusChanged: true,
			expectedStatus:        task.SComplete,
			expectedObj: fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
				obj.Status.UpdateRevision = newRevision
				obj.Status.CurrentRevision = newRevision
				obj.Status.Replicas = 2
				obj.Status.ReadyReplicas = 2
				obj.Status.UpdatedReplicas = 2
				obj.Status.CurrentReplicas = 2
				return obj
			}),
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctrl := gomock.NewController(tt)
			state := NewMockStatusRevisionAndReplicasUpdater[*v1alpha1.PDGroup, *runtime.PD](ctrl)
			state.EXPECT().Object().Return(c.obj)
			state.EXPECT().Revision().Return(newRevision, oldRevision, int32(0))
			state.EXPECT().Slice().Return(runtime.FromPDSlice(c.instances))
			if c.expectedStatusChanged {
				state.EXPECT().SetStatusChanged()
			}

			ctx := context.Background()
			res, done := task.RunTask(ctx, TaskStatusRevisionAndReplicas[scope.PDGroup](state))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)
			// ignore time
			for i := range c.obj.Status.Conditions {
				cond := &c.obj.Status.Conditions[i]
				cond.LastTransitionTime = metav1.Time{}
			}
			assert.Equal(tt, c.expectedObj.Status, c.obj.Status, c.desc)
		})
	}
}
