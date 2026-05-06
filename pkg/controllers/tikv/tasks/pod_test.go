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

	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/features"
	pdapi "github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/v2/pkg/reloadable"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	stateutil "github.com/pingcap/tidb-operator/v2/pkg/state"
	pdv1 "github.com/pingcap/tidb-operator/v2/pkg/timanager/apis/pd/v1"
	pdm "github.com/pingcap/tidb-operator/v2/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/task/v3"
)

const (
	fakeVersion    = "v1.2.3"
	fakeNewVersion = "v1.3.3"
	changedConfig  = `log.level = 'warn'`
)

func TestTaskPod(t *testing.T) {
	now := metav1.Now()
	cases := []struct {
		desc          string
		state         *ReconcileContext
		objs          []client.Object
		downPeerCount int
		downPeerInfo  *pdapi.RegionsCheckInfo
		pdClientReady bool
		// if true, cannot apply pod
		unexpectedErr bool

		expectUpdatedPod         bool
		expectedPodIsTerminating bool
		expectedStatus           task.Status
		expectedShouldEvict      bool
	}{
		{
			desc: "no pod",
			state: &ReconcileContext{
				State: &state{
					tikv: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
						obj.Spec.Version = fakeVersion
						setLeadersEvicted(obj)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("aaa"),
				},
			},

			expectUpdatedPod: true,
			expectedStatus:   task.SComplete,
			pdClientReady:    false,
		},
		{
			desc: "no pod, failed to apply",
			state: &ReconcileContext{
				State: &state{
					tikv: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
						obj.Spec.Version = fakeVersion
						setLeadersEvicted(obj)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("aaa"),
				},
			},
			unexpectedErr: true,

			expectedStatus: task.SFail,
			pdClientReady:  false,
		},
		{
			desc: "pod is deleting",
			state: &ReconcileContext{
				State: &state{
					tikv: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
						obj.Spec.Version = fakeVersion
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("aaa"),
					pod: fake.FakeObj("aaa-tikv-xxx", func(obj *corev1.Pod) *corev1.Pod {
						obj.SetDeletionTimestamp(&now)
						obj.SetDeletionGracePeriodSeconds(ptr.To[int64](100))
						return obj
					}),
					isPodTerminating: true,
				},
				Store: &pdv1.Store{
					RegionCount: 400,
				},
			},

			expectedPodIsTerminating: true,
			expectedStatus:           task.SWait,
			expectedShouldEvict:      false,
			pdClientReady:            false,
		},
		{
			desc: "version is changed",
			state: &ReconcileContext{
				State: &state{
					tikv: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
						obj.Spec.Version = fakeVersion
						setLeadersEvicted(obj)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("aaa"),
					pod: fakePod(
						fake.FakeObj[v1alpha1.Cluster]("aaa"),
						fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
							obj.Spec.Version = fakeNewVersion
							return obj
						}),
					),
				},
			},

			downPeerCount:            0,
			expectedPodIsTerminating: true,
			expectedStatus:           task.SWait,
			expectedShouldEvict:      true,
			pdClientReady:            true,
		},
		{
			desc: "version is changed, failed to delete",
			state: &ReconcileContext{
				State: &state{
					tikv: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
						obj.Spec.Version = fakeVersion
						setLeadersEvicted(obj)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("aaa"),
					pod: fakePod(
						fake.FakeObj[v1alpha1.Cluster]("aaa"),
						fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
							obj.Spec.Version = fakeNewVersion
							return obj
						}),
					),
				},
			},
			downPeerCount: 0,
			unexpectedErr: true,

			expectedStatus:      task.SFail,
			expectedShouldEvict: true,
			pdClientReady:       true,
		},
		{
			desc: "config changed, hot reload policy",
			state: &ReconcileContext{
				State: &state{
					tikv: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
						obj.Spec.Version = fakeVersion
						obj.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyHotReload
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("aaa"),
					pod: fakePod(
						fake.FakeObj[v1alpha1.Cluster]("aaa"),
						fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
							obj.Spec.Version = fakeVersion
							obj.Spec.Config = changedConfig
							obj.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyHotReload
							return obj
						}),
					),
				},
			},

			expectUpdatedPod: true,
			expectedStatus:   task.SComplete,
			pdClientReady:    false,
		},
		{
			desc: "config changed, restart policy",
			state: &ReconcileContext{
				State: &state{
					tikv: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
						obj.Spec.Version = fakeVersion
						obj.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyRestart
						setLeadersEvicted(obj)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("aaa"),
					pod: fakePod(
						fake.FakeObj[v1alpha1.Cluster]("aaa"),
						fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
							obj.Spec.Version = fakeVersion
							obj.Spec.Config = changedConfig
							obj.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyRestart
							return obj
						}),
					),
				},
			},

			downPeerCount:            0,
			expectedPodIsTerminating: true,
			expectedStatus:           task.SWait,
			expectedShouldEvict:      true,
			pdClientReady:            true,
		},
		{
			desc: "version is changed but restart precheck is blocked by leader count",
			state: &ReconcileContext{
				State: &state{
					tikv: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
						obj.Spec.Version = fakeVersion
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("aaa"),
					pod: fakePod(
						fake.FakeObj[v1alpha1.Cluster]("aaa"),
						fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
							obj.Spec.Version = fakeNewVersion
							return obj
						}),
					),
					leaderCount: 1,
				},
				Store: &pdv1.Store{RegionCount: 400},
			},

			downPeerCount:            0,
			expectedPodIsTerminating: false,
			expectedStatus:           task.SRetry,
			expectedShouldEvict:      true,
			pdClientReady:            true,
		},
		{
			desc: "version is changed but restart precheck is blocked by down peer count",
			state: &ReconcileContext{
				State: &state{
					tikv: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
						obj.Spec.Version = fakeVersion
						setLeadersEvicted(obj)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("aaa"),
					pod: fakePod(
						fake.FakeObj[v1alpha1.Cluster]("aaa"),
						fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
							obj.Spec.Version = fakeNewVersion
							return obj
						}),
					),
				},
				Store: &pdv1.Store{ID: "1", RegionCount: 400},
			},

			downPeerCount: 2,
			downPeerInfo: &pdapi.RegionsCheckInfo{
				Count: 2,
				Regions: []*pdapi.RegionCheckEntry{
					{
						ID: 101,
						DownPeers: []*pdapi.RegionPeerStat{
							{Peer: &metapb.Peer{StoreId: 2}},
							{Peer: &metapb.Peer{StoreId: 3}},
						},
					},
				},
			},
			expectedPodIsTerminating: false,
			expectedStatus:           task.SRetry,
			expectedShouldEvict:      true,
			pdClientReady:            true,
		},
		{
			desc: "version is changed but restart precheck ignores self down peer",
			state: &ReconcileContext{
				State: &state{
					tikv: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
						obj.Spec.Version = fakeVersion
						setLeadersEvicted(obj)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("aaa"),
					pod: fakePod(
						fake.FakeObj[v1alpha1.Cluster]("aaa"),
						fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
							obj.Spec.Version = fakeNewVersion
							return obj
						}),
					),
				},
				Store: &pdv1.Store{ID: "1", RegionCount: 400},
			},

			downPeerCount: 1,
			downPeerInfo: &pdapi.RegionsCheckInfo{
				Count: 1,
				Regions: []*pdapi.RegionCheckEntry{
					{
						ID: 101,
						DownPeers: []*pdapi.RegionPeerStat{
							{Peer: &metapb.Peer{StoreId: 1}},
						},
					},
				},
			},
			expectedPodIsTerminating: true,
			expectedStatus:           task.SWait,
			expectedShouldEvict:      true,
			pdClientReady:            true,
		},
		{
			desc: "version is changed but pd client is not ready",
			state: &ReconcileContext{
				State: &state{
					tikv: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
						obj.Spec.Version = fakeVersion
						setLeadersEvicted(obj)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("aaa"),
					pod: fakePod(
						fake.FakeObj[v1alpha1.Cluster]("aaa"),
						fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
							obj.Spec.Version = fakeNewVersion
							return obj
						}),
					),
				},
			},

			expectedPodIsTerminating: false,
			expectedStatus:           task.SWait,
			expectedShouldEvict:      true,
			pdClientReady:            false,
		},
		{
			desc: "version is changed but down peer details are missing",
			state: &ReconcileContext{
				State: &state{
					tikv: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
						obj.Spec.Version = fakeVersion
						setLeadersEvicted(obj)
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("aaa"),
					pod: fakePod(
						fake.FakeObj[v1alpha1.Cluster]("aaa"),
						fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
							obj.Spec.Version = fakeNewVersion
							return obj
						}),
					),
				},
				Store: &pdv1.Store{ID: "1", RegionCount: 400},
			},

			downPeerCount: 1,
			downPeerInfo: &pdapi.RegionsCheckInfo{
				Count: 1,
			},
			expectedPodIsTerminating: false,
			expectedStatus:           task.SRetry,
			expectedShouldEvict:      true,
			pdClientReady:            true,
		},
		{
			desc: "pod labels changed, config not changed",
			state: &ReconcileContext{
				State: &state{
					tikv: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
						obj.Spec.Version = fakeVersion
						obj.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyRestart
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("aaa"),
					pod: fakePod(
						fake.FakeObj[v1alpha1.Cluster]("aaa"),
						fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
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
					),
				},
			},

			expectUpdatedPod:    true,
			expectedStatus:      task.SComplete,
			expectedShouldEvict: false,
			pdClientReady:       false,
		},
		{
			desc: "pod labels changed, config not changed, apply failed",
			state: &ReconcileContext{
				State: &state{
					tikv: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
						obj.Spec.Version = fakeVersion
						obj.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyRestart
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("aaa"),
					pod: fakePod(
						fake.FakeObj[v1alpha1.Cluster]("aaa"),
						fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
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
					),
				},
			},
			unexpectedErr: true,

			expectedStatus:      task.SFail,
			expectedShouldEvict: false,
			pdClientReady:       false,
		},
		{
			desc: "all are not changed",
			state: &ReconcileContext{
				State: &state{
					tikv: fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
						obj.Spec.Version = fakeVersion
						obj.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyRestart
						return obj
					}),
					cluster: fake.FakeObj[v1alpha1.Cluster]("aaa"),
					pod: fakePod(
						fake.FakeObj[v1alpha1.Cluster]("aaa"),
						fake.FakeObj("aaa-xxx", func(obj *v1alpha1.TiKV) *v1alpha1.TiKV {
							obj.Spec.Version = fakeVersion
							obj.Spec.UpdateStrategy.Config = v1alpha1.ConfigUpdateStrategyRestart
							return obj
						}),
					),
				},
			},

			expectedStatus:      task.SComplete,
			expectedShouldEvict: false,
			pdClientReady:       false,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			ctx := context.Background()
			var objs []client.Object
			objs = append(objs, c.state.TiKV(), c.state.Cluster())
			if c.state.Pod() != nil {
				objs = append(objs, c.state.Pod())
			}
			fc := client.NewFakeClient(objs...)
			for _, obj := range c.objs {
				require.NoError(tt, fc.Apply(ctx, obj), c.desc)
			}
			s := c.state.State.(*state)
			s.IFeatureGates = stateutil.NewFeatureGates[scope.TiKV](s)

			ctrl := gomock.NewController(tt)
			if c.pdClientReady {
				mockPDClient := pdm.NewMockPDClient(ctrl)
				mockUnderlay := pdapi.NewMockPDClient(ctrl)
				mockPDClient.EXPECT().Underlay().Return(mockUnderlay).AnyTimes()
				s.IPDClient = &stubPDClientState{client: mockPDClient}

				shouldQueryDownPeer := c.state.Pod() != nil &&
					c.state.Pod().GetDeletionTimestamp().IsZero() &&
					!reloadable.CheckTiKVPod(c.state.TiKV(), c.state.Pod())
				if shouldQueryDownPeer {
					mockUnderlay.EXPECT().
						GetDownPeerRegions(gomock.Any()).
						Return(func() *pdapi.RegionsCheckInfo {
							if c.downPeerInfo != nil {
								return c.downPeerInfo
							}
							return &pdapi.RegionsCheckInfo{Count: c.downPeerCount}
						}(), nil)
				}
			} else {
				s.IPDClient = &stubPDClientUnavailableState{}
			}

			if c.unexpectedErr {
				// cannot update pod
				fc.WithError("patch", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
				fc.WithError("delete", "*", errors.NewInternalError(fmt.Errorf("fake internal err")))
			}

			res, done := task.RunTask(ctx, TaskPod(c.state, fc, nil))
			assert.Equal(tt, c.expectedStatus.String(), res.Status().String(), res.Message())
			assert.False(tt, done, c.desc)

			assert.Equal(tt, c.expectedPodIsTerminating, c.state.IsPodTerminating(), c.desc)
			assert.Equal(tt, c.expectedShouldEvict, c.state.ShouldEvictLeader, c.desc)

			if c.expectUpdatedPod {
				expectedPod := newPod(c.state.Cluster(), c.state.TiKV(), nil, c.state.FeatureGates())
				actual := c.state.Pod().DeepCopy()
				actual.Kind = ""
				actual.APIVersion = ""
				actual.ManagedFields = nil
				assert.Equal(tt, expectedPod, actual, c.desc)
			}
		})
	}
}

func fakePod(c *v1alpha1.Cluster, tikv *v1alpha1.TiKV) *corev1.Pod {
	return newPod(c, tikv, nil, features.NewFromFeatures(nil))
}

func setLeadersEvicted(tikv *v1alpha1.TiKV) {
	tikv.Status.Conditions = append(tikv.Status.Conditions, metav1.Condition{
		Type:   v1alpha1.TiKVCondLeadersEvicted,
		Status: metav1.ConditionTrue,
		Reason: v1alpha1.ReasonEvicted,
	})
}
