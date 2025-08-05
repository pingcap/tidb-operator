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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

const (
	fakeVersion    = "v1.2.3"
	fakeNewVersion = "v1.3.3"
	changedConfig  = `log.level = 'warn'`
)

func TestTaskPod(t *testing.T) {
	cases := []struct {
		desc       string
		setupState func() *ReconcileContext
		objs       []client.Object
		// if true, cannot apply pod
		unexpectedErr bool

		expectUpdatedPod         bool
		expectedPodIsTerminating bool
		expectedStatus           task.Status
	}{
		{
			desc: "no pod",
			setupState: func() *ReconcileContext {
				state := NewState(types.NamespacedName{Namespace: "default", Name: "aaa-xxx"})
				tidb := fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
					obj.Spec.Version = fakeVersion
					return obj
				})
				cluster := fake.FakeObj[v1alpha1.Cluster]("aaa")
				state.SetObject(tidb)
				state.SetCluster(cluster)
				return &ReconcileContext{State: state}
			},

			expectUpdatedPod: true,
			expectedStatus:   task.SComplete,
		},
		{
			desc: "no pod, failed to apply",
			setupState: func() *ReconcileContext {
				state := NewState(types.NamespacedName{Namespace: "default", Name: "aaa-xxx"})
				tidb := fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
					obj.Spec.Version = fakeVersion
					return obj
				})
				cluster := fake.FakeObj[v1alpha1.Cluster]("aaa")
				state.SetObject(tidb)
				state.SetCluster(cluster)
				return &ReconcileContext{State: state}
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "version is changed",
			setupState: func() *ReconcileContext {
				state := NewState(types.NamespacedName{Namespace: "default", Name: "aaa-xxx"})
				tidb := fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
					obj.Spec.Version = fakeVersion
					return obj
				})
				cluster := fake.FakeObj[v1alpha1.Cluster]("aaa")
				pod := fakePod(
					fake.FakeObj[v1alpha1.Cluster]("aaa"),
					fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
						obj.Spec.Version = fakeNewVersion
						return obj
					}),
					features.NewFromFeatures(nil),
				)
				state.SetObject(tidb)
				state.SetCluster(cluster)
				state.SetPod(pod)
				return &ReconcileContext{State: state}
			},

			expectedPodIsTerminating: true,
			expectedStatus:           task.SWait,
		},
		{
			desc: "version is changed, failed to delete",
			setupState: func() *ReconcileContext {
				state := NewState(types.NamespacedName{Namespace: "default", Name: "aaa-xxx"})
				tidb := fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
					obj.Spec.Version = fakeVersion
					return obj
				})
				cluster := fake.FakeObj[v1alpha1.Cluster]("aaa")
				pod := fakePod(
					fake.FakeObj[v1alpha1.Cluster]("aaa"),
					fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
						obj.Spec.Version = fakeNewVersion
						return obj
					}),
					features.NewFromFeatures(nil),
				)
				state.SetObject(tidb)
				state.SetCluster(cluster)
				state.SetPod(pod)
				return &ReconcileContext{State: state}
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "config changed, hot reload policy",
			setupState: func() *ReconcileContext {
				state := NewState(types.NamespacedName{Namespace: "default", Name: "aaa-xxx"})
				tidb := fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
					obj.Spec.Version = fakeVersion
					obj.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyHotReload
					return obj
				})
				cluster := fake.FakeObj[v1alpha1.Cluster]("aaa")
				pod := fakePod(
					fake.FakeObj[v1alpha1.Cluster]("aaa"),
					fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
						obj.Spec.Version = fakeVersion
						obj.Spec.Config = changedConfig
						obj.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyHotReload
						return obj
					}),
					features.NewFromFeatures(nil),
				)
				state.SetObject(tidb)
				state.SetCluster(cluster)
				state.SetPod(pod)
				return &ReconcileContext{State: state}
			},

			expectUpdatedPod: true,
			expectedStatus:   task.SComplete,
		},
		{
			desc: "config changed, restart policy",
			setupState: func() *ReconcileContext {
				state := NewState(types.NamespacedName{Namespace: "default", Name: "aaa-xxx"})
				tidb := fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
					obj.Spec.Version = fakeVersion
					obj.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyRestart
					return obj
				})
				cluster := fake.FakeObj[v1alpha1.Cluster]("aaa")
				pod := fakePod(
					fake.FakeObj[v1alpha1.Cluster]("aaa"),
					fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
						obj.Spec.Version = fakeVersion
						obj.Spec.Config = changedConfig
						obj.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyRestart
						return obj
					}),
					features.NewFromFeatures(nil),
				)
				state.SetObject(tidb)
				state.SetCluster(cluster)
				state.SetPod(pod)
				return &ReconcileContext{State: state}
			},

			expectedPodIsTerminating: true,
			expectedStatus:           task.SWait,
		},
		{
			desc: "pod labels changed, config not changed",
			setupState: func() *ReconcileContext {
				state := NewState(types.NamespacedName{Namespace: "default", Name: "aaa-xxx"})
				tidb := fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
					obj.Spec.Version = fakeVersion
					obj.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyRestart
					return obj
				})
				cluster := fake.FakeObj[v1alpha1.Cluster]("aaa")
				pod := fakePod(
					fake.FakeObj[v1alpha1.Cluster]("aaa"),
					fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
						obj.Spec.Version = fakeVersion
						obj.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyRestart
						obj.Spec.Overlay = &v1alpha1.Overlay{
							Pod: &v1alpha1.PodOverlay{
								ObjectMeta: v1alpha1.ObjectMeta{
									Labels: map[string]string{
										"test": "test",
									},
								},
							},
						}
						return obj
					}),
					features.NewFromFeatures(nil),
				)
				state.SetObject(tidb)
				state.SetCluster(cluster)
				state.SetPod(pod)
				return &ReconcileContext{State: state}
			},

			expectUpdatedPod: true,
			expectedStatus:   task.SComplete,
		},
		{
			desc: "pod labels changed, config not changed, apply failed",
			setupState: func() *ReconcileContext {
				state := NewState(types.NamespacedName{Namespace: "default", Name: "aaa-xxx"})
				tidb := fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
					obj.Spec.Version = fakeVersion
					obj.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyRestart
					return obj
				})
				cluster := fake.FakeObj[v1alpha1.Cluster]("aaa")
				pod := fakePod(
					fake.FakeObj[v1alpha1.Cluster]("aaa"),
					fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
						obj.Spec.Version = fakeVersion
						obj.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyRestart
						obj.Spec.Overlay = &v1alpha1.Overlay{
							Pod: &v1alpha1.PodOverlay{
								ObjectMeta: v1alpha1.ObjectMeta{
									Labels: map[string]string{
										"test": "test",
									},
								},
							},
						}
						return obj
					}),
					features.NewFromFeatures(nil),
				)
				state.SetObject(tidb)
				state.SetCluster(cluster)
				state.SetPod(pod)
				return &ReconcileContext{State: state}
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
		},
		{
			desc: "all are not changed",
			setupState: func() *ReconcileContext {
				state := NewState(types.NamespacedName{Namespace: "default", Name: "aaa-xxx"})
				tidb := fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
					obj.Spec.Version = fakeVersion
					obj.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyRestart
					return obj
				})
				cluster := fake.FakeObj[v1alpha1.Cluster]("aaa")
				pod := fakePod(
					fake.FakeObj[v1alpha1.Cluster]("aaa"),
					fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiDB) *v1alpha1.TiDB {
						obj.Spec.Version = fakeVersion
						obj.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyRestart
						return obj
					}),
					features.NewFromFeatures(nil),
				)
				state.SetObject(tidb)
				state.SetCluster(cluster)
				state.SetPod(pod)
				return &ReconcileContext{State: state}
			},

			expectedStatus: task.SComplete,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctx := context.Background()
			testState := c.setupState()
			var objs []client.Object
			objs = append(objs, testState.TiDB(), testState.Cluster())
			if testState.Pod() != nil {
				objs = append(objs, testState.Pod())
			}
			fc := client.NewFakeClient(objs...)
			for _, obj := range c.objs {
				require.NoError(tt, fc.Apply(ctx, obj), c.desc)
			}

			if c.unexpectedErr {
				// cannot update pod
				fc.WithError("patch", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
				fc.WithError("delete", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			res, done := task.RunTask(ctx, TaskPod(testState, fc))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), res.Message())
			assert.False(tt, done, c.desc)

			assert.Equal(tt, c.expectedPodIsTerminating, testState.IsPodTerminating(), c.desc)

			if c.expectUpdatedPod {
				expectedPod := newPod(testState.Cluster(), testState.TiDB(), testState.FeatureGates())
				actual := testState.Pod().DeepCopy()
				actual.Kind = ""
				actual.APIVersion = ""
				actual.ManagedFields = nil
				assert.Equal(tt, expectedPod, actual, c.desc)
			}
		})
	}
}

func fakePod(c *v1alpha1.Cluster, tidb *v1alpha1.TiDB, g features.Gates) *corev1.Pod {
	return newPod(c, tidb, g)
}
