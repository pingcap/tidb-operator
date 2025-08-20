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

package updater

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
)

func TestBuilder(t *testing.T) {
	addHook := AddHookFunc[*runtime.PD](func(pd *runtime.PD) *runtime.PD { return pd })
	updateHook := UpdateHookFunc[*runtime.PD](func(update, _ *runtime.PD) *runtime.PD { return update })
	delHook := DelHookFunc[*runtime.PD](func(_ string) {})
	pd0 := fake.FakeObj[v1alpha1.PD]("pd-0")
	pd1 := fake.FakeObj[v1alpha1.PD]("pd-1")
	cli := client.NewFakeClient(pd0, pd1)

	cases := []struct {
		desc         string
		desired      int
		revision     string
		expectedWait bool
	}{
		{
			desc:         "update",
			desired:      2,
			revision:     "v1",
			expectedWait: true,
		},
		{
			desc:         "scale out",
			desired:      3,
			expectedWait: true,
		},
		{
			desc:         "scale in",
			desired:      1,
			expectedWait: true,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			bld := New[runtime.PDTuple]().
				WithInstances(fake.FakeObj[runtime.PD]("pd-0"), fake.FakeObj[runtime.PD]("pd-1")).WithDesired(c.desired).
				WithMaxSurge(1).
				WithMaxUnavailable(1).
				WithRevision(c.revision).
				WithClient(cli).
				WithNewFactory(NewFunc[*runtime.PD](func() *runtime.PD { return &runtime.PD{} })).
				WithAddHooks(addHook).
				WithUpdateHooks(updateHook).
				WithDelHooks(delHook).Build()
			wait, err := bld.Do(context.Background())
			require.NoError(t, err)
			assert.Equal(t, c.expectedWait, wait)
		})
	}
}

func TestSplit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name             string
		instances        []runtime.Instance
		revision         string
		expectedUpdate   int
		expectedOutdated int
		expectedOffline  int
		expectedDeleted  int
	}{
		{
			name:             "empty input",
			instances:        []runtime.Instance{},
			revision:         "v1",
			expectedUpdate:   0,
			expectedOutdated: 0,
			expectedOffline:  0,
			expectedDeleted:  0,
		},
		{
			name: "standard classification - update",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().IsStore().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v1")
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   1,
			expectedOutdated: 0,
			expectedOffline:  0,
			expectedDeleted:  0,
		},
		{
			name: "standard classification - outdated",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().IsStore().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v0")
					mock.EXPECT().GetAnnotations().Return(map[string]string{})
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   0,
			expectedOutdated: 1,
			expectedOffline:  0,
			expectedDeleted:  0,
		},
		{
			name: "standard classification - defer delete",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().IsStore().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v0")
					mock.EXPECT().GetAnnotations().Return(map[string]string{
						v1alpha1.AnnoKeyDeferDelete: "true",
					})
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   0,
			expectedOutdated: 0,
			expectedOffline:  0,
			expectedDeleted:  1,
		},
		{
			name: "being deleted instance should be skipped",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					now := metav1.NewTime(time.Now())
					mock.EXPECT().GetDeletionTimestamp().Return(&now)
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   0,
			expectedOutdated: 0,
			expectedOffline:  0,
			expectedDeleted:  0,
		},
		// StoreInstance specific tests
		{
			name: "store instance - offline completed",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().IsStore().Return(true)
					mock.EXPECT().Conditions().Return([]metav1.Condition{
						{
							Type:   v1alpha1.StoreOfflinedConditionType,
							Reason: v1alpha1.ReasonOfflineCompleted,
						},
					})
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   0,
			expectedOutdated: 0,
			expectedOffline:  0,
			expectedDeleted:  1,
		},
		{
			name: "store instance - being offline",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().IsStore().Return(true)
					mock.EXPECT().Conditions().Return([]metav1.Condition{})
					mock.EXPECT().IsOffline().Return(true)
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   0,
			expectedOutdated: 0,
			expectedOffline:  1,
			expectedDeleted:  0,
		},
		{
			name: "store instance - normal online classification",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().IsStore().Return(true)
					mock.EXPECT().Conditions().Return([]metav1.Condition{})
					mock.EXPECT().IsOffline().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v1")
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   1,
			expectedOutdated: 0,
			expectedOffline:  0,
			expectedDeleted:  0,
		},
		{
			name: "store instance - online outdated (not classified as outdated)",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().IsStore().Return(true)
					mock.EXPECT().Conditions().Return([]metav1.Condition{})
					mock.EXPECT().IsOffline().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v0")
					mock.EXPECT().GetAnnotations().Return(map[string]string{})
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   0,
			expectedOutdated: 1,
			expectedOffline:  0,
			expectedDeleted:  0,
		},
		{
			name: "store instance - online with defer delete annotation (not classified)",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().IsStore().Return(true)
					mock.EXPECT().Conditions().Return([]metav1.Condition{})
					mock.EXPECT().IsOffline().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v0")
					mock.EXPECT().GetAnnotations().Return(map[string]string{
						v1alpha1.AnnoKeyDeferDelete: "true",
					})
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   0,
			expectedOutdated: 0,
			expectedOffline:  0,
			expectedDeleted:  1,
		},
		// more edge case tests
		{
			name: "empty revision matching",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().IsStore().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("")
					return mock
				}(),
			},
			revision:         "",
			expectedUpdate:   1,
			expectedOutdated: 0,
			expectedOffline:  0,
			expectedDeleted:  0,
		},
		{
			name: "instance with nil annotations",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().IsStore().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v0")
					mock.EXPECT().GetAnnotations().Return(nil)
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   0,
			expectedOutdated: 1,
			expectedOffline:  0,
			expectedDeleted:  0,
		},
		{
			name: "defer delete annotation with non-true value",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().IsStore().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v0")
					mock.EXPECT().GetAnnotations().Return(map[string]string{
						v1alpha1.AnnoKeyDeferDelete: "false",
					})
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   0,
			expectedOutdated: 0,
			expectedOffline:  0,
			expectedDeleted:  1,
		},
		{
			name: "zero deletion timestamp",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(&metav1.Time{})
					mock.EXPECT().IsStore().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v1")
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   1,
			expectedOutdated: 0,
			expectedOffline:  0,
			expectedDeleted:  0,
		},
		{
			name: "store instance with multiple conditions",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().IsStore().Return(true)
					mock.EXPECT().Conditions().Return([]metav1.Condition{
						{
							Type:   "SomeOtherCondition",
							Status: metav1.ConditionTrue,
						},
						{
							Type:   v1alpha1.StoreOfflinedConditionType,
							Reason: v1alpha1.ReasonOfflineCompleted,
						},
						{
							Type:   "AnotherCondition",
							Status: metav1.ConditionFalse,
						},
					})
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   0,
			expectedOutdated: 0,
			expectedOffline:  0,
			expectedDeleted:  1,
		},
		{
			name: "store instance - offline condition exists but not completed",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().IsStore().Return(true)
					mock.EXPECT().Conditions().Return([]metav1.Condition{
						{
							Type:   v1alpha1.StoreOfflinedConditionType,
							Reason: v1alpha1.ReasonOfflineFailed, // not Completed
						},
					})
					mock.EXPECT().IsOffline().Return(true)
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   0,
			expectedOutdated: 0,
			expectedOffline:  1,
			expectedDeleted:  0,
		},
		{
			name: "store instance - offline completed with spec.offline=false",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().IsStore().Return(true)
					mock.EXPECT().Conditions().Return([]metav1.Condition{
						{
							Type:   v1alpha1.StoreOfflinedConditionType,
							Reason: v1alpha1.ReasonOfflineCompleted,
						},
					})
					// Note: this tests the case mentioned in comment - spec.offline=false but has completed condition
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   0,
			expectedOutdated: 0,
			expectedOffline:  0,
			expectedDeleted:  1,
		},
		// mixed scenario tests
		{
			name: "mixed instances",
			instances: []runtime.Instance{
				// Update instance
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().IsStore().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v1")
					return mock
				}(),
				// Outdated instance
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().IsStore().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v0")
					mock.EXPECT().GetAnnotations().Return(map[string]string{})
					return mock
				}(),
				// Being offline store instance
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().IsStore().Return(true)
					mock.EXPECT().Conditions().Return([]metav1.Condition{})
					mock.EXPECT().IsOffline().Return(true)
					return mock
				}(),
				// Defer delete instance
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().IsStore().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v0")
					mock.EXPECT().GetAnnotations().Return(map[string]string{
						v1alpha1.AnnoKeyDeferDelete: "true",
					})
					return mock
				}(),
				// Being deleted instance (should be skipped)
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					now := metav1.NewTime(time.Now())
					mock.EXPECT().GetDeletionTimestamp().Return(&now)
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   1,
			expectedOutdated: 1,
			expectedOffline:  1,
			expectedDeleted:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update, outdated, beingOffline, deleted := split(tt.instances, tt.revision)

			assert.Len(t, update, tt.expectedUpdate, "update count mismatch")
			assert.Len(t, outdated, tt.expectedOutdated, "outdated count mismatch")
			assert.Len(t, beingOffline, tt.expectedOffline, "beingOffline count mismatch")
			assert.Len(t, deleted, tt.expectedDeleted, "deleted count mismatch")
		})
	}
}
