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

func TestTaskGroupFinalizerDel(t *testing.T) {
	t.Run("TiKVGroup", testTaskGroupFinalizerDel[runtime.TiKVGroupTuple, runtime.TiKVTuple])
	t.Run("TiDBGroup", testTaskGroupFinalizerDel[runtime.TiDBGroupTuple, runtime.TiDBTuple])
}

func testTaskGroupFinalizerDel[
	GT runtime.GroupTuple[OG, RG],
	IT runtime.InstanceTuple[OI, RI],
	OG client.Object,
	RG runtime.GroupT[G],
	OI client.Object,
	RI runtime.InstanceT[I],
	G runtime.GroupSet,
	I runtime.InstanceSet,
](t *testing.T) {
	now := metav1.Now()
	cases := []struct {
		desc          string
		state         GroupAndInstanceSliceState[RG, RI]
		subresources  []client.Object
		unexpectedErr bool

		expectedStatus task.Status
		expectedObj    RG
	}{
		{
			desc: "no instances and no sub resources and no finalizer",
			state: FakeGroupAndInstanceSliceState[RG, RI](
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetDeletionTimestamp(&now)
					return obj
				}),
			),
			expectedStatus: task.SComplete,
			expectedObj: fake.Fake(func(obj RG) RG {
				obj.SetName("aaa")
				obj.SetDeletionTimestamp(&now)
				return obj
			}),
		},
		{
			desc: "no instances and no sub resources",
			state: FakeGroupAndInstanceSliceState[RG, RI](
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetDeletionTimestamp(&now)
					obj.SetFinalizers([]string{
						v1alpha1.Finalizer,
					})
					return obj
				}),
			),
			expectedStatus: task.SComplete,
			expectedObj: fake.Fake(func(obj RG) RG {
				obj.SetName("aaa")
				obj.SetDeletionTimestamp(&now)
				obj.SetFinalizers([]string{})
				return obj
			}),
		},
		{
			desc: "no instances and no sub resources but call api failed",
			state: FakeGroupAndInstanceSliceState[RG, RI](
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetDeletionTimestamp(&now)
					obj.SetFinalizers([]string{
						v1alpha1.Finalizer,
					})
					return obj
				}),
			),
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "no instances but has sub resources",
			state: FakeGroupAndInstanceSliceState[RG, RI](
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetDeletionTimestamp(&now)
					obj.SetFinalizers([]string{
						v1alpha1.Finalizer,
					})
					return obj
				}),
			),
			subresources: []client.Object{
				fake.FakeObj("aaa",
					fake.Label[corev1.Service](v1alpha1.LabelKeyManagedBy, v1alpha1.LabelValManagedByOperator),
					fake.Label[corev1.Service](v1alpha1.LabelKeyCluster, ""),
					fake.Label[corev1.Service](v1alpha1.LabelKeyComponent, runtime.Component[G, RG]()),
					fake.Label[corev1.Service](v1alpha1.LabelKeyGroup, "aaa"),
				),
			},

			expectedStatus: task.SWait,
			expectedObj: fake.Fake(func(obj RG) RG {
				obj.SetName("aaa")
				obj.SetDeletionTimestamp(&now)
				obj.SetFinalizers([]string{
					v1alpha1.Finalizer,
				})
				return obj
			}),
		},
		{
			desc: "no instances but has sub resources and call api failed",
			state: FakeGroupAndInstanceSliceState[RG, RI](
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetDeletionTimestamp(&now)
					obj.SetFinalizers([]string{
						v1alpha1.Finalizer,
					})
					return obj
				}),
			),
			subresources: []client.Object{
				fake.FakeObj("aaa",
					fake.Label[corev1.Service](v1alpha1.LabelKeyManagedBy, v1alpha1.LabelValManagedByOperator),
					fake.Label[corev1.Service](v1alpha1.LabelKeyCluster, ""),
					fake.Label[corev1.Service](v1alpha1.LabelKeyComponent, runtime.Component[G, RG]()),
					fake.Label[corev1.Service](v1alpha1.LabelKeyGroup, "aaa"),
				),
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "has instances with finalizer",
			state: FakeGroupAndInstanceSliceState(
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetDeletionTimestamp(&now)
					obj.SetFinalizers([]string{
						v1alpha1.Finalizer,
					})
					return obj
				}),
				fake.Fake(func(obj RI) RI {
					obj.SetName("aaa")
					obj.SetFinalizers([]string{
						v1alpha1.Finalizer,
					})
					return obj
				}),
			),
			expectedStatus: task.SRetry,
			expectedObj: fake.Fake(func(obj RG) RG {
				obj.SetName("aaa")
				obj.SetDeletionTimestamp(&now)
				obj.SetFinalizers([]string{
					v1alpha1.Finalizer,
				})
				return obj
			}),
		},
		{
			desc: "has instances with finalizer but call api failed",
			state: FakeGroupAndInstanceSliceState(
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetDeletionTimestamp(&now)
					obj.SetFinalizers([]string{
						v1alpha1.Finalizer,
					})
					return obj
				}),
				fake.Fake(func(obj RI) RI {
					obj.SetName("aaa")
					obj.SetFinalizers([]string{
						v1alpha1.Finalizer,
					})
					return obj
				}),
			),
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "has deleting instances with finalizer but call api failed",
			state: FakeGroupAndInstanceSliceState(
				fake.Fake(func(obj RG) RG {
					obj.SetName("aaa")
					obj.SetDeletionTimestamp(&now)
					obj.SetFinalizers([]string{
						v1alpha1.Finalizer,
					})
					return obj
				}),
				fake.Fake(func(obj RI) RI {
					obj.SetName("aaa")
					obj.SetDeletionTimestamp(&now)
					obj.SetFinalizers([]string{
						v1alpha1.Finalizer,
					})
					return obj
				}),
			),
			unexpectedErr: true,

			expectedStatus: task.SRetry,
		},
	}

	var gt GT
	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			objs := []client.Object{
				gt.To(c.state.Group()),
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

			res, done := task.RunTask(ctx, TaskGroupFinalizerDel[GT, IT](c.state, fc))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
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
