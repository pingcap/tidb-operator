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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/controllers/common"
	tiflashtasks "github.com/pingcap/tidb-operator/v2/pkg/controllers/tiflash/tasks"
	"github.com/pingcap/tidb-operator/v2/pkg/features"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	stateutil "github.com/pingcap/tidb-operator/v2/pkg/state"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/tracker"
)

const (
	oldRevision = "old"
	newRevision = "new"
	newImage    = "pingcap/tiflash:v8.1.1"
)

func TestTaskUpdater(t *testing.T) {
	cases := []struct {
		desc          string
		state         *ReconcileContext
		objs          []client.Object
		unexpectedErr bool

		expectedStatus     task.Status
		expectedTiFlashNum int
	}{
		{
			desc: "no fs with 1 replicas",
			state: &ReconcileContext{
				State: &state{
					fg:      fake.FakeObj[v1alpha1.TiFlashGroup]("aaa"),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				},
			},

			expectedStatus:     task.SWait,
			expectedTiFlashNum: 1,
		},
		{
			desc: "version upgrade check",
			state: &ReconcileContext{
				State: &state{
					fg: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlashGroup) *v1alpha1.TiFlashGroup {
						obj.Spec.Template.Spec.Version = "v8.1.0"
						obj.Spec.Cluster.Name = "cluster"
						obj.Status.Version = "v8.0.0"
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				},
			},
			objs: []client.Object{
				fake.FakeObj("aaa", func(obj *v1alpha1.PDGroup) *v1alpha1.PDGroup {
					obj.Spec.Cluster.Name = "cluster"
					obj.Spec.Replicas = ptr.To[int32](1)
					obj.Spec.Template.Spec.Version = "v8.1.0"
					obj.Status.Version = "v8.0.0"
					return obj
				}),
			},

			expectedStatus: task.SRetry,
		},
		{
			desc: "1 updated tiflash with 1 replicas",
			state: &ReconcileContext{
				State: &state{
					fg:      fake.FakeObj[v1alpha1.TiFlashGroup]("aaa"),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					fs: []*v1alpha1.TiFlash{
						fakeAvailableTiFlash("aaa-xxx", fake.FakeObj[v1alpha1.TiFlashGroup]("aaa"), newRevision),
					},
					updateRevision: newRevision,
				},
			},

			expectedStatus:     task.SComplete,
			expectedTiFlashNum: 1,
		},
		{
			desc: "no fs with 2 replicas",
			state: &ReconcileContext{
				State: &state{
					fg: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlashGroup) *v1alpha1.TiFlashGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				},
			},

			expectedStatus:     task.SWait,
			expectedTiFlashNum: 2,
		},
		{
			desc: "no fs with 2 replicas and call api failed",
			state: &ReconcileContext{
				State: &state{
					fg: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlashGroup) *v1alpha1.TiFlashGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				},
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "1 outdated tiflash with 2 replicas",
			state: &ReconcileContext{
				State: &state{
					fg: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlashGroup) *v1alpha1.TiFlashGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					fs: []*v1alpha1.TiFlash{
						fakeAvailableTiFlash("aaa-xxx", fake.FakeObj[v1alpha1.TiFlashGroup]("aaa"), oldRevision),
					},
					updateRevision: newRevision,
				},
			},

			expectedStatus:     task.SWait,
			expectedTiFlashNum: 2,
		},
		{
			desc: "1 outdated tiflash with 2 replicas but cannot call api, will fail",
			state: &ReconcileContext{
				State: &state{
					fg: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlashGroup) *v1alpha1.TiFlashGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					fs: []*v1alpha1.TiFlash{
						fakeAvailableTiFlash("aaa-xxx", fake.FakeObj[v1alpha1.TiFlashGroup]("aaa"), oldRevision),
					},
					updateRevision: newRevision,
				},
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "2 updated tiflash with 2 replicas",
			state: &ReconcileContext{
				State: &state{
					fg: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlashGroup) *v1alpha1.TiFlashGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					fs: []*v1alpha1.TiFlash{
						fakeAvailableTiFlash("aaa-xxx", fake.FakeObj[v1alpha1.TiFlashGroup]("aaa"), newRevision),
						fakeAvailableTiFlash("aaa-yyy", fake.FakeObj[v1alpha1.TiFlashGroup]("aaa"), newRevision),
					},
					updateRevision: newRevision,
				},
			},

			expectedStatus:     task.SComplete,
			expectedTiFlashNum: 2,
		},
		{
			desc: "2 updated tiflash with 2 replicas and cannot call api, can complete",
			state: &ReconcileContext{
				State: &state{
					fg: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlashGroup) *v1alpha1.TiFlashGroup {
						obj.Spec.Replicas = ptr.To[int32](2)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
					fs: []*v1alpha1.TiFlash{
						fakeAvailableTiFlash("aaa-xxx", fake.FakeObj[v1alpha1.TiFlashGroup]("aaa"), newRevision),
						fakeAvailableTiFlash("aaa-yyy", fake.FakeObj[v1alpha1.TiFlashGroup]("aaa"), newRevision),
					},
					updateRevision: newRevision,
				},
			},
			unexpectedErr: true,

			expectedStatus:     task.SComplete,
			expectedTiFlashNum: 2,
		},
		{
			// NOTE: it not really check whether the policy is worked
			// It should be tested in /pkg/updater and /pkg/updater/policy package
			desc: "topology evenly spread",
			state: &ReconcileContext{
				State: &state{
					fg: fake.FakeObj("aaa", func(obj *v1alpha1.TiFlashGroup) *v1alpha1.TiFlashGroup {
						obj.Spec.Replicas = ptr.To[int32](3)
						obj.Spec.SchedulePolicies = append(obj.Spec.SchedulePolicies, v1alpha1.SchedulePolicy{
							Type: v1alpha1.SchedulePolicyTypeEvenlySpread,
							EvenlySpread: &v1alpha1.SchedulePolicyEvenlySpread{
								Topologies: []v1alpha1.ScheduleTopology{
									{
										Topology: v1alpha1.Topology{
											"zone": "us-west-1a",
										},
									},
									{
										Topology: v1alpha1.Topology{
											"zone": "us-west-1b",
										},
									},
									{
										Topology: v1alpha1.Topology{
											"zone": "us-west-1c",
										},
									},
								},
							},
						})
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"),
				},
			},

			expectedStatus:     task.SWait,
			expectedTiFlashNum: 3,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			s := c.state.State.(*state)
			s.IFeatureGates = stateutil.NewFeatureGates[scope.TiFlashGroup](s)

			ctx := context.Background()
			c.objs = append(c.objs, c.state.TiFlashGroup(), c.state.Cluster())
			fc := client.NewFakeClient(c.objs...)
			for _, obj := range c.state.TiFlashSlice() {
				require.NoError(tt, fc.Apply(ctx, obj.DeepCopy()), c.desc)
				require.NoError(tt, fc.Status().Update(ctx, obj.DeepCopy()), c.desc)
			}

			if c.unexpectedErr {
				// cannot create or update tiflash instance
				fc.WithError("patch", "tiflashs", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			af := tracker.New().AllocateFactory("tiflash")
			res, done := task.RunTask(ctx, TaskUpdater(c.state, fc, af))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), c.desc)
			assert.False(tt, done, c.desc)

			if !c.unexpectedErr {
				fs := v1alpha1.TiFlashList{}
				require.NoError(tt, fc.List(ctx, &fs), c.desc)
				assert.Len(tt, fs.Items, c.expectedTiFlashNum, c.desc)
			}
		})
	}
}

func fakeAvailableTiFlash(name string, fg *v1alpha1.TiFlashGroup, rev string) *v1alpha1.TiFlash {
	return fake.FakeObj(name, func(obj *v1alpha1.TiFlash) *v1alpha1.TiFlash {
		tiflash := runtime.ToTiFlash(TiFlashNewer(fg, rev, features.NewFromFeatures(nil)).New())
		tiflash.Name = ""
		tiflash.Status.Conditions = append(tiflash.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.CondReady,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Unix(0, 0),
		})
		tiflash.Status.CurrentRevision = rev
		tiflash.DeepCopyInto(obj)
		return obj
	})
}

func TestTaskUpdaterSurgeWaitsForAvailable(t *testing.T) {
	ctx := t.Context()
	fg := fake.FakeObj[v1alpha1.TiFlashGroup]("flash")
	fg.Spec.Cluster.Name = "cluster"
	fg.Spec.Replicas = ptr.To[int32](1)
	fg.Spec.Template.Spec.Version = "v8.1.0"
	fg.Spec.Template.Spec.Volumes = []v1alpha1.Volume{{
		Name: "data", Storage: resource.MustParse("1Gi"),
		Mounts: []v1alpha1.VolumeMount{{Type: v1alpha1.VolumeMountTypeTiFlashData}},
	}}
	fg.Spec.MaxSurge = ptr.To[int32](1)
	fg.Spec.MinReadySeconds = ptr.To[int64](60)
	old := fakeAvailableTiFlash("flash-old", fg, oldRevision)
	fg.Spec.Template.Spec.Image = ptr.To(newImage)
	cluster := fake.FakeObj[v1alpha1.Cluster]("cluster")
	fc := client.NewFakeClient(fg, cluster, old)
	af := tracker.New().AllocateFactory("tiflash")

	reconcile := func() task.Status {
		var list v1alpha1.TiFlashList
		require.NoError(t, fc.List(ctx, &list))
		s := &state{fg: fg, cluster: cluster, updateRevision: newRevision}
		for i := range list.Items {
			s.fs = append(s.fs, &list.Items[i])
		}
		s.IFeatureGates = stateutil.NewFeatureGates[scope.TiFlashGroup](s)
		res, _ := task.RunTask(ctx, TaskUpdater(&ReconcileContext{State: s}, fc, af))
		return res.Status()
	}
	assertOldUnchanged := func() {
		var got v1alpha1.TiFlash
		require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(old), &got))
		assert.Equal(t, old.Spec, got.Spec)
		assert.Equal(t, oldRevision, runtime.FromTiFlash(&got).GetUpdateRevision())
		assert.Empty(t, got.Annotations[v1alpha1.AnnoKeyDeferDelete])
		assert.True(t, got.DeletionTimestamp.IsZero())
	}

	require.Equal(t, task.SWait, reconcile())
	var list v1alpha1.TiFlashList
	require.NoError(t, fc.List(ctx, &list))
	require.Len(t, list.Items, 2, "create a spare before changing the healthy instance")
	assertOldUnchanged()
	var replacement *v1alpha1.TiFlash
	for i := range list.Items {
		if list.Items[i].Name != old.Name {
			replacement = &list.Items[i]
		}
	}
	require.NotNil(t, replacement)
	assert.Equal(t, fg.Spec.Template.Spec.Image, replacement.Spec.Image)
	oldPVCs := tiflashtasks.PVCNewer().NewPVCs(cluster, old, features.NewFromFeatures(nil))
	newPVCs := tiflashtasks.PVCNewer().NewPVCs(cluster, replacement, features.NewFromFeatures(nil))
	require.Len(t, oldPVCs, 1)
	require.Len(t, newPVCs, 1)
	assert.NotEqual(t, oldPVCs[0].Name, newPVCs[0].Name)
	require.NoError(t, fc.Create(ctx, oldPVCs[0]))
	require.NoError(t, fc.Create(ctx, newPVCs[0]))

	// Pending and unhealthy replacements must not disrupt the old instance.
	assert.Equal(t, task.SWait, reconcile())
	assertOldUnchanged()
	replacement.Status.CurrentRevision = newRevision
	replacement.Status.ObservedGeneration = replacement.Generation
	meta.SetStatusCondition(&replacement.Status.Conditions, metav1.Condition{
		Type: v1alpha1.CondReady, Status: metav1.ConditionFalse,
		ObservedGeneration: replacement.Generation, LastTransitionTime: metav1.Now(),
	})
	require.NoError(t, fc.Status().Update(ctx, replacement))
	assert.Equal(t, task.SWait, reconcile())
	assertOldUnchanged()

	meta.SetStatusCondition(&replacement.Status.Conditions, metav1.Condition{
		Type: v1alpha1.CondReady, Status: metav1.ConditionTrue,
		ObservedGeneration: replacement.Generation, LastTransitionTime: metav1.Now(),
	})
	require.NoError(t, fc.Status().Update(ctx, replacement))
	assert.Equal(t, task.SRetry, reconcile(), "wait for minReadySeconds")
	assertOldUnchanged()

	meta.FindStatusCondition(replacement.Status.Conditions, v1alpha1.CondReady).LastTransitionTime = metav1.NewTime(time.Now().Add(-2 * time.Minute))
	require.NoError(t, fc.Status().Update(ctx, replacement))
	reconcile()
	var retiring v1alpha1.TiFlash
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(old), &retiring))
	require.True(t, ptr.Deref(retiring.Spec.Offline, false), "retire the old store only after the replacement is available")
	assert.True(t, retiring.DeletionTimestamp.IsZero(), "wait for the store to finish offlining")
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(oldPVCs[0]), oldPVCs[0]), "keep old storage while offlining")

	// The instance controller reports completion of PD store removal.
	meta.SetStatusCondition(&retiring.Status.Conditions, metav1.Condition{
		Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionTrue,
		ObservedGeneration: retiring.Generation,
	})
	require.NoError(t, fc.Status().Update(ctx, &retiring))
	reconcile()
	assert.True(t, errors.IsNotFound(fc.Get(ctx, client.ObjectKeyFromObject(old), &retiring)))
	// Exercise subresource cleanup separately: the fake client deletes CRs immediately.
	instanceState := tiflashtasks.NewState(client.ObjectKeyFromObject(old))
	instanceState.SetObject(&retiring)
	cleanup, _ := task.RunTask(ctx, common.TaskInstanceFinalizerDel[scope.TiFlash](instanceState, fc, tiflashtasks.SubresourceLister))
	assert.Equal(t, task.SRetry, cleanup.Status(), "wait for subresource deletion before removing the finalizer")
	assert.True(t, errors.IsNotFound(fc.Get(ctx, client.ObjectKeyFromObject(oldPVCs[0]), oldPVCs[0])))
	require.NoError(t, fc.Get(ctx, client.ObjectKeyFromObject(newPVCs[0]), newPVCs[0]), "keep replacement storage")
	assert.Equal(t, task.SComplete, reconcile())
	require.NoError(t, fc.List(ctx, &list))
	require.Len(t, list.Items, 1)
	assert.Equal(t, replacement.Name, list.Items[0].Name)
}

func TestTaskUpdaterSurgePolicy(t *testing.T) {
	cases := []struct {
		name       string
		maxSurge   *int32
		replicas   int32
		change     func(*v1alpha1.TiFlashGroup)
		unready    bool
		wantCount  int
		wantStatus task.Status
	}{
		{name: "unset uses in-place restart", replicas: 1, change: func(g *v1alpha1.TiFlashGroup) {
			g.Spec.Template.Spec.Image = ptr.To(newImage)
		}, wantCount: 1, wantStatus: task.SWait},
		{name: "zero uses in-place restart", replicas: 1, maxSurge: ptr.To[int32](0), change: func(g *v1alpha1.TiFlashGroup) {
			g.Spec.Template.Spec.Image = ptr.To(newImage)
		}, wantCount: 1, wantStatus: task.SWait},
		{name: "two surge instances", replicas: 3, maxSurge: ptr.To[int32](2), change: func(g *v1alpha1.TiFlashGroup) {
			g.Spec.Template.Spec.Image = ptr.To(newImage)
		}, wantCount: 5, wantStatus: task.SWait},
		{name: "one spare for three replicas", replicas: 3, maxSurge: ptr.To[int32](1), change: func(g *v1alpha1.TiFlashGroup) {
			g.Spec.Template.Spec.Image = ptr.To(newImage)
		}, wantCount: 4, wantStatus: task.SWait},
		{name: "restart config uses surge", replicas: 1, maxSurge: ptr.To[int32](1), change: func(g *v1alpha1.TiFlashGroup) {
			g.Spec.Template.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyRestart
			g.Spec.Template.Spec.Config = "log-level = 'debug'"
		}, wantCount: 2, wantStatus: task.SWait},
		{name: "reloadable config updates in place", replicas: 1, maxSurge: ptr.To[int32](1), change: func(g *v1alpha1.TiFlashGroup) {
			g.Spec.Template.Spec.Config = "log-level = 'debug'"
		}, wantCount: 1, wantStatus: task.SWait},
		{name: "max surge alone does not restart", replicas: 1, maxSurge: ptr.To[int32](1), wantCount: 1, wantStatus: task.SComplete},
		{name: "current revision still waits for readiness", replicas: 1, maxSurge: ptr.To[int32](1), unready: true, wantCount: 1, wantStatus: task.SWait},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			fg := fake.FakeObj[v1alpha1.TiFlashGroup]("flash")
			fg.Spec.Replicas = ptr.To(tt.replicas)
			fg.Spec.SchedulePolicies = []v1alpha1.SchedulePolicy{{
				Type: v1alpha1.SchedulePolicyTypeEvenlySpread,
				EvenlySpread: &v1alpha1.SchedulePolicyEvenlySpread{Topologies: []v1alpha1.ScheduleTopology{
					{Topology: v1alpha1.Topology{"zone": "a"}},
					{Topology: v1alpha1.Topology{"zone": "b"}},
					{Topology: v1alpha1.Topology{"zone": "c"}},
				}},
			}}
			revision := oldRevision
			if tt.change == nil {
				revision = newRevision
			}
			s := &state{fg: fg, cluster: fake.FakeObj[v1alpha1.Cluster]("cluster"), updateRevision: newRevision}
			objs := []client.Object{fg, s.cluster}
			for i := range tt.replicas {
				instance := fakeAvailableTiFlash(fmt.Sprintf("flash-old-%d", i), fg, revision)
				instance.Spec.Topology = fg.Spec.SchedulePolicies[0].EvenlySpread.Topologies[i].Topology
				if tt.unready {
					instance.Status.Conditions = nil
				}
				s.fs = append(s.fs, instance)
				objs = append(objs, instance)
			}
			fg.Spec.MaxSurge = tt.maxSurge
			if tt.change != nil {
				tt.change(fg)
			}
			fc := client.NewFakeClient(objs...)
			s.IFeatureGates = stateutil.NewFeatureGates[scope.TiFlashGroup](s)
			res, _ := task.RunTask(t.Context(), TaskUpdater(&ReconcileContext{State: s}, fc, tracker.New().AllocateFactory("tiflash")))
			assert.Equal(t, tt.wantStatus, res.Status())
			var list v1alpha1.TiFlashList
			require.NoError(t, fc.List(t.Context(), &list))
			assert.Len(t, list.Items, tt.wantCount)
			if tt.wantCount > int(tt.replicas) {
				zones := map[string]int{}
				for _, instance := range list.Items {
					zones[instance.Spec.Topology["zone"]]++
				}
				for _, count := range zones {
					assert.LessOrEqual(t, count, 2, "surge instances remain evenly spread")
				}
				for _, old := range s.fs {
					var got v1alpha1.TiFlash
					require.NoError(t, fc.Get(t.Context(), client.ObjectKeyFromObject(old), &got))
					assert.Equal(t, old.Spec, got.Spec, "pending surge instances must not disrupt old instances")
				}
			}
			if tt.replicas > 1 {
				verifyMultiReplicaSurge(t, s, fc)
			}
		})
	}
}

func verifyMultiReplicaSurge(t *testing.T, s *state, fc client.Client) {
	t.Helper()
	replicas := *s.fg.Spec.Replicas
	var list v1alpha1.TiFlashList
	require.NoError(t, fc.List(t.Context(), &list))
	// Drive readiness and store removal through the multi-replica rollout.
	// Allow a readiness and cleanup pass for each desired instance.
	for range replicas * 2 {
		for i := range list.Items {
			instance := &list.Items[i]
			instance.Status.CurrentRevision = runtime.FromTiFlash(instance).GetUpdateRevision()
			instance.Status.ObservedGeneration = instance.Generation
			meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
				Type: v1alpha1.CondReady, Status: metav1.ConditionTrue,
				ObservedGeneration: instance.Generation, LastTransitionTime: metav1.Unix(0, 0),
			})
			if ptr.Deref(instance.Spec.Offline, false) {
				meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
					Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionTrue,
					ObservedGeneration: instance.Generation,
				})
			}
			require.NoError(t, fc.Status().Update(t.Context(), instance))
		}
		s.fs = nil
		for i := range list.Items {
			s.fs = append(s.fs, &list.Items[i])
		}
		res, _ := task.RunTask(t.Context(), TaskUpdater(&ReconcileContext{State: s}, fc, tracker.New().AllocateFactory("tiflash")))
		require.NotEqual(t, task.SFail, res.Status())
		require.NoError(t, fc.List(t.Context(), &list))
		available := 0
		for i := range list.Items {
			instance := runtime.FromTiFlash(&list.Items[i])
			if !instance.IsOffline() && instance.IsReady() && instance.IsUpToDate() {
				available++
			}
		}
		assert.GreaterOrEqual(t, available, int(replicas))
		if res.Status() == task.SComplete && len(list.Items) == int(replicas) {
			break
		}
	}
	require.Len(t, list.Items, int(replicas))
	for i := range list.Items {
		assert.Equal(t, newRevision, runtime.FromTiFlash(&list.Items[i]).GetUpdateRevision())
	}
}
