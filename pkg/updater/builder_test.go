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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/utils/fake"
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

func TestBuilderScaleInPrefersNewer(t *testing.T) {
	cases := []struct {
		name   string
		older  *runtime.PD
		custom bool
		want   string
	}{
		{name: "newer by default", older: fakePD("older", true, true), want: "newer"},
		{name: "manual priority wins", older: fakePDWithPriority("older", true, true, "0"), want: "older"},
		{name: "unready wins", older: fakePD("older", true, false), want: "older"},
		{name: "not running wins", older: fakePD("older", false, true), want: "older"},
		{name: "custom policy wins", older: fakePD("older", true, true), custom: true, want: "older"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.older.SetCreationTimestamp(metav1.NewTime(time.Unix(100, 0)))
			newer := fakePD("newer", true, true)
			newer.SetCreationTimestamp(metav1.NewTime(time.Unix(200, 0)))
			b := New[runtime.PDTuple]()
			if tc.custom {
				b.WithScaleInPreferPolicy(PreferPolicyFunc[*runtime.PD](func([]*runtime.PD) []*runtime.PD {
					return []*runtime.PD{tc.older}
				}))
			}
			act := b.Build().(*executor).act.(*actor[runtime.PDTuple, *v1alpha1.PD, *runtime.PD])
			instances := []*runtime.PD{tc.older, newer}
			assert.Equal(t, tc.want, act.scaleInSelector.Choose(instances))
			assert.Equal(t, "older", act.updateSelector.Choose(instances))
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
			name:      "empty input",
			instances: []runtime.Instance{},
			revision:  "v1",
		},
		{
			name: "standard classification - update",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(map[string]string{})
					mock.EXPECT().Conditions().Return([]metav1.Condition{})
					mock.EXPECT().IsOffline().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v1")
					return mock
				}(),
			},
			revision:       "v1",
			expectedUpdate: 1,
		},
		{
			name: "standard classification - outdated",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(map[string]string{})
					mock.EXPECT().Conditions().Return([]metav1.Condition{})
					mock.EXPECT().IsOffline().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v0")
					return mock
				}(),
			},
			revision:         "v1",
			expectedOutdated: 1,
		},
		{
			name: "standard classification - defer delete",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(map[string]string{
						v1alpha1.AnnoKeyDeferDelete: "true",
					})
					return mock
				}(),
			},
			revision:        "v1",
			expectedDeleted: 1,
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
			revision: "v1",
		},
		{
			name: "being offline",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(map[string]string{})
					mock.EXPECT().Conditions().Return([]metav1.Condition{})
					mock.EXPECT().IsOffline().Return(true)
					return mock
				}(),
			},
			revision:        "v1",
			expectedOffline: 1,
		},
		{
			name: "offline completed",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(map[string]string{})
					mock.EXPECT().Conditions().Return([]metav1.Condition{
						{
							Type:   v1alpha1.StoreOfflinedConditionType,
							Status: metav1.ConditionTrue, // This is what meta.IsStatusConditionTrue checks
						},
					})
					return mock
				}(),
			},
			revision:        "v1",
			expectedDeleted: 1,
		},
		// more edge case tests
		{
			name: "empty revision matching",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(map[string]string{})
					mock.EXPECT().Conditions().Return([]metav1.Condition{})
					mock.EXPECT().IsOffline().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("")
					return mock
				}(),
			},
			revision:         "",
			expectedUpdate:   1,
			expectedOutdated: 0,
			expectedDeleted:  0,
		},
		{
			name: "instance with nil annotations",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(nil)
					mock.EXPECT().Conditions().Return([]metav1.Condition{})
					mock.EXPECT().IsOffline().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v0")
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   0,
			expectedOutdated: 1,
			expectedDeleted:  0,
		},
		{
			name: "defer delete annotation exists (value irrelevant)",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(map[string]string{
						v1alpha1.AnnoKeyDeferDelete: "false",
					})
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   0,
			expectedOutdated: 0,
			expectedDeleted:  1,
		},
		{
			name: "zero deletion timestamp",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(&metav1.Time{})
					mock.EXPECT().GetAnnotations().Return(map[string]string{})
					mock.EXPECT().Conditions().Return([]metav1.Condition{})
					mock.EXPECT().IsOffline().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v1")
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   1,
			expectedOutdated: 0,
			expectedDeleted:  0,
		},
		{
			name: "multiple conditions",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(map[string]string{})
					mock.EXPECT().Conditions().Return([]metav1.Condition{
						{
							Type:   "SomeOtherCondition",
							Status: metav1.ConditionTrue,
						},
						{
							Type:   v1alpha1.StoreOfflinedConditionType,
							Status: metav1.ConditionTrue,
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
			expectedDeleted:  1,
		},
		{
			name: "offline condition exists but not true",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(map[string]string{})
					mock.EXPECT().Conditions().Return([]metav1.Condition{
						{
							Type:   v1alpha1.StoreOfflinedConditionType,
							Status: metav1.ConditionFalse, // not True
						},
					})
					mock.EXPECT().IsOffline().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v1")
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   1,
			expectedOutdated: 0,
			expectedDeleted:  0,
		},
		// mixed scenario tests
		{
			name: "mixed instances",
			instances: []runtime.Instance{
				// Update instance
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(map[string]string{})
					mock.EXPECT().Conditions().Return([]metav1.Condition{})
					mock.EXPECT().IsOffline().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v1")
					return mock
				}(),
				// Outdated instance
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(map[string]string{})
					mock.EXPECT().Conditions().Return([]metav1.Condition{})
					mock.EXPECT().IsOffline().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v0")
					return mock
				}(),
				// Defer delete instance
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
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
			expectedDeleted:  1,
		},
		// Additional edge cases
		{
			name: "offline condition with empty reason",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(map[string]string{})
					mock.EXPECT().Conditions().Return([]metav1.Condition{
						{
							Type:   v1alpha1.StoreOfflinedConditionType,
							Status: metav1.ConditionFalse, // not True, so should not be deleted
							Reason: "",                    // empty reason
						},
					})
					mock.EXPECT().IsOffline().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v1")
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   1,
			expectedOutdated: 0,
			expectedDeleted:  0,
		},
		{
			name: "offline condition with different status values",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(map[string]string{})
					mock.EXPECT().Conditions().Return([]metav1.Condition{
						{
							Type:   v1alpha1.StoreOfflinedConditionType,
							Status: metav1.ConditionFalse,
							Reason: "SomeOtherReason",
						},
					})
					mock.EXPECT().GetUpdateRevision().Return("v1")
					mock.EXPECT().IsOffline().Return(false)
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   1,
			expectedOutdated: 0,
			expectedDeleted:  0,
		},
		{
			name: "revision with special characters",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(map[string]string{})
					mock.EXPECT().Conditions().Return([]metav1.Condition{})
					mock.EXPECT().IsOffline().Return(false)
					mock.EXPECT().GetUpdateRevision().Return("v1-beta.1+sha.abc123")
					return mock
				}(),
			},
			revision:         "v1-beta.1+sha.abc123",
			expectedUpdate:   1,
			expectedOutdated: 0,
			expectedDeleted:  0,
		},
		{
			name: "very long revision string",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(map[string]string{})
					mock.EXPECT().Conditions().Return([]metav1.Condition{})
					mock.EXPECT().IsOffline().Return(false)
					longRev := "v1-" + strings.Repeat("a", 200)
					mock.EXPECT().GetUpdateRevision().Return(longRev)
					return mock
				}(),
			},
			revision:         "v1-" + strings.Repeat("a", 200),
			expectedUpdate:   1,
			expectedOutdated: 0,
			expectedDeleted:  0,
		},
		{
			name: "multiple annotations including defer delete",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(map[string]string{
						"some-other-annotation":     "value",
						v1alpha1.AnnoKeyDeferDelete: "", // empty value still triggers deletion
						"another-annotation":        "value2",
					})
					return mock
				}(),
			},
			revision:         "v1",
			expectedUpdate:   0,
			expectedOutdated: 0,
			expectedDeleted:  1,
		},
		// Priority order tests - multiple conditions
		{
			name: "defer delete annotation takes priority over IsOffline()",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(map[string]string{
						v1alpha1.AnnoKeyDeferDelete: "true",
					})
					// IsOffline() should not be called due to defer delete priority
					return mock
				}(),
			},
			revision:        "v1",
			expectedDeleted: 1,
		},
		{
			name: "StoreOfflined condition takes priority over IsOffline()",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(map[string]string{})
					mock.EXPECT().Conditions().Return([]metav1.Condition{
						{
							Type:   v1alpha1.StoreOfflinedConditionType,
							Status: metav1.ConditionTrue,
						},
					})
					// IsOffline() should not be called due to StoreOfflined condition priority
					return mock
				}(),
			},
			revision:        "v1",
			expectedDeleted: 1,
		},
		{
			name: "both defer delete and StoreOfflined condition present",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(map[string]string{
						v1alpha1.AnnoKeyDeferDelete: "true",
					})
					return mock
				}(),
			},
			revision:        "v1",
			expectedDeleted: 1,
		},
		{
			name: "IsOffline() true with version mismatch - should be beingOffline, not outdated",
			instances: []runtime.Instance{
				func() runtime.Instance {
					mock := runtime.NewMockInstance(ctrl)
					mock.EXPECT().GetDeletionTimestamp().Return(nil)
					mock.EXPECT().GetAnnotations().Return(map[string]string{})
					mock.EXPECT().Conditions().Return([]metav1.Condition{})
					mock.EXPECT().IsOffline().Return(true)
					return mock
				}(),
			},
			revision:        "v1",
			expectedOffline: 1,
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

func TestBuilderScalesInExcessCapacity(t *testing.T) {
	small, large := fakePD("small-0", true, true), fakePD("large-0", true, true)
	meta.SetStatusCondition(&large.Status.Conditions, metav1.Condition{
		Type: v1alpha1.CondVolumeCapacityExceedsRequest, Status: metav1.ConditionTrue,
		Reason: v1alpha1.ReasonCapacityExceedsRequest, ObservedGeneration: large.Generation,
	})
	cli := client.NewFakeClient(runtime.ToPD(small), runtime.ToPD(large))
	executor := New[runtime.PDTuple]().WithInstances(small, large).
		WithRevision("test").WithDesired(1).WithMaxUnavailable(1).WithClient(cli).Build()
	_, err := executor.Do(context.Background())
	require.NoError(t, err)
	require.True(t, errors.IsNotFound(cli.Get(context.Background(), client.ObjectKeyFromObject(large), &v1alpha1.PD{})))
	require.NoError(t, cli.Get(context.Background(), client.ObjectKeyFromObject(small), &v1alpha1.PD{}))
}
