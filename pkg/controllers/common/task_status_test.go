package common

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

const (
	oldRevision = "old"
	newRevision = "new"
)

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

func TestTaskGroupStatus(t *testing.T) {
	t.Run("PDGroup", testTaskGroupStatus[runtime.PDGroupTuple, runtime.PD])
	t.Run("TiKVGroup", testTaskGroupStatus[runtime.TiKVGroupTuple, runtime.TiKV])
	t.Run("TiDBGroup", testTaskGroupStatus[runtime.TiDBGroupTuple, runtime.TiDB])
	t.Run("TiFlashGroup", testTaskGroupStatus[runtime.TiFlashGroupTuple, runtime.TiFlash])
}

func testTaskGroupStatus[
	GT runtime.GroupTuple[OG, RG],
	I runtime.InstanceSet,
	OG client.Object,
	RG runtime.GroupT[G],
	G runtime.GroupSet,
	RI runtime.InstanceT[I],
](t *testing.T) {
	cases := []struct {
		desc          string
		state         GroupAndInstanceSliceAndRevisionState[RG, RI]
		unexpectedErr bool

		expectedStatus task.Status
		expectedObj    RG
	}{
		{
			desc: "no instances",
			state: FakeGroupAndInstanceSliceAndRevisionState[RG, RI](
				newRevision,
				oldRevision,
				3,
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
				obj.SetConditions([]metav1.Condition{
					{
						Type:               v1alpha1.CondSuspended,
						Status:             metav1.ConditionFalse,
						ObservedGeneration: 3,
						Reason:             v1alpha1.ReasonUnsuspended,
						Message:            "group is not suspended",
					},
				})
				obj.SetObservedGeneration(3)
				obj.SetStatusReplicas(0, 0, 0, 0)
				obj.SetStatusRevision(newRevision, newRevision, ptr.To[int32](3))
				return obj
			}),
		},
		{
			desc: "all instances are outdated and healthy",
			state: FakeGroupAndInstanceSliceAndRevisionState(
				newRevision,
				oldRevision,
				0,
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetGeneration(3)
					return obj
				}),
				fake.Fake(func(obj RI) RI {
					obj.SetName("aaa")
					obj.SetCurrentRevision(oldRevision)
					obj.SetConditions([]metav1.Condition{
						{
							Type:   v1alpha1.CondHealth,
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
				obj.SetConditions([]metav1.Condition{
					{
						Type:               v1alpha1.CondSuspended,
						Status:             metav1.ConditionFalse,
						ObservedGeneration: 3,
						Reason:             v1alpha1.ReasonUnsuspended,
						Message:            "group is not suspended",
					},
				})
				obj.SetObservedGeneration(3)
				obj.SetStatusReplicas(1, 1, 0, 1)
				obj.SetStatusRevision(newRevision, oldRevision, nil)
				return obj
			}),
		},
		{
			desc: "all instances are updated and healthy",
			state: FakeGroupAndInstanceSliceAndRevisionState(
				newRevision,
				oldRevision,
				0,
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetGeneration(3)
					return obj
				}),
				fake.Fake(func(obj RI) RI {
					obj.SetName("aaa")
					obj.SetCurrentRevision(newRevision)
					obj.SetConditions([]metav1.Condition{
						{
							Type:   v1alpha1.CondHealth,
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
				obj.SetConditions([]metav1.Condition{
					{
						Type:               v1alpha1.CondSuspended,
						Status:             metav1.ConditionFalse,
						ObservedGeneration: 3,
						Reason:             v1alpha1.ReasonUnsuspended,
						Message:            "group is not suspended",
					},
				})
				obj.SetObservedGeneration(3)
				obj.SetStatusReplicas(1, 1, 1, 0)
				obj.SetStatusRevision(newRevision, newRevision, nil)
				return obj
			}),
		},
		{
			desc: "all instances are updated but not healthy",
			state: FakeGroupAndInstanceSliceAndRevisionState(
				newRevision,
				oldRevision,
				0,
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetGeneration(3)
					return obj
				}),
				fake.Fake(func(obj RI) RI {
					obj.SetName("aaa")
					obj.SetCurrentRevision(newRevision)
					return obj
				}),
			),

			expectedStatus: task.SComplete,
			expectedObj: fake.Fake(func(obj RG) RG {
				obj.SetName("aaa")
				obj.SetGeneration(3)
				obj.SetConditions([]metav1.Condition{
					{
						Type:               v1alpha1.CondSuspended,
						Status:             metav1.ConditionFalse,
						ObservedGeneration: 3,
						Reason:             v1alpha1.ReasonUnsuspended,
						Message:            "group is not suspended",
					},
				})
				obj.SetObservedGeneration(3)
				obj.SetStatusReplicas(1, 0, 1, 0)
				obj.SetStatusRevision(newRevision, newRevision, nil)
				return obj
			}),
		},
		{
			desc: "status changed but cannot call api",
			state: FakeGroupAndInstanceSliceAndRevisionState(
				newRevision,
				oldRevision,
				0,
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetGeneration(3)
					return obj
				}),
				fake.Fake(func(obj RI) RI {
					obj.SetName("aaa")
					obj.SetCurrentRevision(newRevision)
					return obj
				}),
			),
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "status is not changed and cannot call api",
			state: FakeGroupAndInstanceSliceAndRevisionState(
				newRevision,
				oldRevision,
				0,
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetGeneration(3)
					obj.SetConditions([]metav1.Condition{
						{
							Type:               v1alpha1.CondSuspended,
							Status:             metav1.ConditionFalse,
							ObservedGeneration: 3,
							Reason:             v1alpha1.ReasonUnsuspended,
							Message:            "group is not suspended",
						},
					})
					obj.SetObservedGeneration(3)
					obj.SetStatusReplicas(1, 0, 1, 0)
					obj.SetStatusRevision(newRevision, newRevision, nil)
					return obj
				}),
				fake.Fake(func(obj RI) RI {
					obj.SetName("aaa")
					obj.SetCurrentRevision(newRevision)
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
			res, done := task.RunTask(ctx, TaskGroupStatus[GT](c.state, fc))
			assert.Equal(tt, c.expectedStatus, res.Status(), c.desc)
			assert.False(tt, done, c.desc)

			// no need to check update result
			if c.unexpectedErr {
				return
			}

			var g RG = new(G)
			require.NoError(tt, fc.Get(ctx, client.ObjectKey{Name: "aaa"}, gt.To(g)), c.desc)

			conds := g.Conditions()
			for i := range conds {
				cond := &conds[i]
				cond.LastTransitionTime = metav1.Time{}
			}
			assert.Equal(tt, c.expectedObj, g, c.desc)
		})
	}
}
