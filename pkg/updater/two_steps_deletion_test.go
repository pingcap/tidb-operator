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
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

// Add this mock implementation in your test file
type mockPreferPolicy struct {
	items []*runtime.TiKV
}

func (m *mockPreferPolicy) Prefer(items []*runtime.TiKV) []*runtime.TiKV {
	if len(items) > 0 {
		return items[:1]
	}
	return nil
}

func newMockPreferPolicy(items []*runtime.TiKV) PreferPolicy[*runtime.TiKV] {
	return &mockPreferPolicy{items: items}
}

// assertTiKVState verifies the expected TiKV count and offline status
func assertTiKVState(t *testing.T, cli client.Client, expectedTotal, expectedOffline int) {
	var tikvs v1alpha1.TiKVList
	require.NoError(t, cli.List(context.TODO(), &tikvs), "failed to list TiKVs")
	assert.Len(t, tikvs.Items, expectedTotal, "expected %d TiKVs", expectedTotal)

	offlineCount := 0
	for i := range tikvs.Items {
		tikv := &tikvs.Items[i]
		if tikv.Spec.Offline {
			offlineCount++
		}
	}
	assert.Equal(t, expectedOffline, offlineCount, "expected %d TiKVs to be offline", expectedOffline)
}

// buildObjects converts TiKV instances to client.Object slice
func buildObjects(tikvs ...*runtime.TiKV) []client.Object {
	objs := make([]client.Object, 0, len(tikvs))
	for _, tikv := range tikvs {
		if tikv != nil {
			objs = append(objs, tikv.To())
		}
	}
	return objs
}

type patch func(tikv *runtime.TiKV)

// nolint: unparam
func buildTestTiKV(nameIndex string, gen int64, ready bool, apply ...patch) *runtime.TiKV {
	rev := strconv.FormatInt(gen, 10)
	tikv := &runtime.TiKV{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "tikv-" + nameIndex,
			Generation: gen,
			Labels:     map[string]string{v1alpha1.LabelKeyInstanceRevisionHash: rev},
		},
		Status: v1alpha1.TiKVStatus{
			CommonStatus: v1alpha1.CommonStatus{
				Conditions:         make([]metav1.Condition, 0),
				CurrentRevision:    rev,
				ObservedGeneration: gen,
			},
		},
	}
	condStatus := metav1.ConditionTrue
	if !ready {
		condStatus = metav1.ConditionFalse
	}
	meta.SetStatusCondition(&tikv.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.CondReady,
		Status:             condStatus,
		ObservedGeneration: gen,
	})

	for _, p := range apply {
		p(tikv)
	}
	return tikv
}

func offline() func(tikv *runtime.TiKV) {
	return func(tikv *runtime.TiKV) {
		tikv.Spec.Offline = true
	}
}

func testTiKVNewer() NewFactory[*runtime.TiKV] {
	return NewFunc[*runtime.TiKV](func() *runtime.TiKV {
		tikv := &v1alpha1.TiKV{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("tikv-%d", rand.IntnRange(10, 99)),
			},
		}
		return runtime.FromTiKV(tikv)
	})
}

func TestExecutorForCancelableScaleIn(t *testing.T) {
	tests := []struct {
		name                  string
		setupStates           func() (update, outdated, beingOffline, deleted []*runtime.TiKV, objs []client.Object)
		scaleInPreferPolicies []PreferPolicy[*runtime.TiKV]

		desired             int
		maxSurge            int
		unavailableUpdate   int
		unavailableOutdated int
		maxUnavailable      int

		expectedActions []action
		expectedWait    bool
		expectedError   bool
		expectedTotal   int
		expectedOffline int
	}{
		// Normal scale in scenarios
		{
			name: "when scale in tikv from 3 to 1, set two of them offline first",
			setupStates: func() (update, outdated, beingOffline, deleted []*runtime.TiKV, objs []client.Object) {
				update = []*runtime.TiKV{
					buildTestTiKV("1", 1, true),
					buildTestTiKV("2", 1, true),
					buildTestTiKV("3", 1, true),
				}
				objs = buildObjects(update...)
				return
			},
			desired: 1,
			expectedActions: []action{
				actionSetOffline,
				actionSetOffline,
			},
			expectedTotal:   3,
			expectedOffline: 2,
			expectedWait:    true,
		},
		{
			name: "when scale in tikv from 3 to 1, wait for offline complete",
			setupStates: func() (update, outdated, beingOffline, deleted []*runtime.TiKV, objs []client.Object) {
				t1 := buildTestTiKV("1", 1, false, offline())
				t2 := buildTestTiKV("2", 1, false, offline())
				t3 := buildTestTiKV("3", 1, true)
				update = []*runtime.TiKV{t3}
				beingOffline = []*runtime.TiKV{t1, t2}
				deleted = []*runtime.TiKV{}
				objs = buildObjects(t1, t2, t3)
				return
			},
			desired:         1,
			expectedActions: []action{},
			expectedTotal:   3,
			expectedOffline: 2,
			expectedWait:    true,
		},
		{
			name: "one completes offline, the other is still being offline",
			setupStates: func() (update, outdated, beingOffline, deleted []*runtime.TiKV, objs []client.Object) {
				// Being offline
				t1 := buildTestTiKV("1", 1, false, offline())
				// Complete offline
				t2 := buildTestTiKV("2", 1, false, offline())
				t3 := buildTestTiKV("3", 1, true)
				update = []*runtime.TiKV{t3}
				beingOffline = []*runtime.TiKV{t1}
				deleted = []*runtime.TiKV{t2}
				objs = buildObjects(t1, t2, t3)
				return
			},
			desired:         1,
			expectedActions: []action{actionCleanup},
			expectedTotal:   2,
			expectedOffline: 1,
			expectedWait:    true,
		},
		{
			name: "both complete offline, one is deleted",
			setupStates: func() (update, outdated, beingOffline, deleted []*runtime.TiKV, objs []client.Object) {
				t1 := buildTestTiKV("1", 1, false, offline())
				t3 := buildTestTiKV("3", 1, true)
				update = []*runtime.TiKV{t3}
				deleted = []*runtime.TiKV{t1}
				objs = buildObjects(t1, t3)
				return
			},
			desired:         1,
			expectedActions: []action{actionCleanup},
			expectedTotal:   1,
			expectedOffline: 0,
			expectedWait:    false,
		},
		{
			name: "both complete offline and deleted",
			setupStates: func() (update, outdated, beingOffline, deleted []*runtime.TiKV, objs []client.Object) {
				t3 := buildTestTiKV("3", 1, true)
				update = []*runtime.TiKV{t3}
				objs = buildObjects(t3)
				return
			},
			desired:         1,
			expectedActions: []action{},
			expectedTotal:   1,
			expectedOffline: 0,
			expectedWait:    false,
		},
		// Cancel scale in scenarios
		{
			name: "scale in from 3 to 1, cancel it by increasing desired to 3",
			setupStates: func() (update, outdated, beingOffline, deleted []*runtime.TiKV, objs []client.Object) {
				t1 := buildTestTiKV("1", 1, false, offline())
				t2 := buildTestTiKV("2", 1, false, offline())
				t3 := buildTestTiKV("3", 1, true)
				update = []*runtime.TiKV{t3}
				beingOffline = []*runtime.TiKV{t1, t2}
				objs = buildObjects(t1, t2, t3)
				return
			},
			desired: 3,
			expectedActions: []action{
				actionCancelOffline,
				actionCancelOffline,
			},
			expectedTotal:   3,
			expectedOffline: 0,
			expectedWait:    true,
		},
		{
			name: "cancel scale in from 3 to 1, one is canceled successfully, the other is offline completed",
			setupStates: func() (update, outdated, beingOffline, deleted []*runtime.TiKV, objs []client.Object) {
				t1 := buildTestTiKV("1", 1, false)
				t2 := buildTestTiKV("2", 1, false, offline())
				t3 := buildTestTiKV("3", 1, true)
				update = []*runtime.TiKV{t1, t3}
				deleted = []*runtime.TiKV{t2}
				objs = buildObjects(t1, t2, t3)
				return
			},
			desired:           3,
			unavailableUpdate: 1,
			expectedActions: []action{
				actionScaleOut,
			},
			expectedTotal:   4,
			expectedOffline: 1,
			expectedWait:    true,
		},
		{
			name: "scale in from 3 to 1, cancel partially it by increasing desired to 2",
			setupStates: func() (update, outdated, beingOffline, deleted []*runtime.TiKV, objs []client.Object) {
				t1 := buildTestTiKV("1", 1, false, offline())
				t2 := buildTestTiKV("2", 1, false, offline())
				t3 := buildTestTiKV("3", 1, true)
				update = []*runtime.TiKV{t3}
				beingOffline = []*runtime.TiKV{t1, t2}
				objs = buildObjects(t1, t2, t3)
				return
			},
			desired: 2,
			expectedActions: []action{
				actionCancelOffline,
			},
			expectedTotal:   3,
			expectedOffline: 1,
			expectedWait:    true,
		},
		{
			name: "scale in from 3 to 1, cancel partially it by increasing desired to 2 (2)",
			setupStates: func() (update, outdated, beingOffline, deleted []*runtime.TiKV, objs []client.Object) {
				t1 := buildTestTiKV("1", 1, false)
				t2 := buildTestTiKV("2", 1, false, offline())
				t3 := buildTestTiKV("3", 1, true)
				update = []*runtime.TiKV{t1, t3}
				deleted = []*runtime.TiKV{t2}
				objs = buildObjects(t1, t2, t3)
				return
			},
			desired: 2,
			expectedActions: []action{
				actionCleanup,
			},
			expectedTotal:   2,
			expectedOffline: 0,
			expectedWait:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update, outdated, beingOffline, deleted, objs := tt.setupStates()
			fakeCli := client.NewFakeClient(objs...)
			act := &actor[runtime.TiKVTuple, *v1alpha1.TiKV, *runtime.TiKV]{
				f:               testTiKVNewer(),
				c:               fakeCli,
				update:          NewState(update),
				outdated:        NewState(outdated),
				beingOffline:    NewState(beingOffline),
				deleted:         NewState(deleted),
				scaleInSelector: NewSelector(newMockPreferPolicy(update)),
				actions:         make([]action, 0),
			}
			e := NewExecutor(act, len(update), len(outdated), len(beingOffline), tt.desired,
				tt.unavailableUpdate, tt.unavailableOutdated, tt.maxSurge, tt.maxUnavailable)

			wait, err := e.Do(context.TODO())
			if tt.expectedError {
				require.Error(t, err, "expected error but got none")
			} else {
				require.NoError(t, err, "unexpected error: %w", err)
				assert.Equal(t, tt.expectedWait, wait, "wait result mismatch")
			}
			assert.Equal(t, tt.expectedActions, act.RecordedActions(), "actions mismatch")

			assertTiKVState(t, fakeCli, tt.expectedTotal, tt.expectedOffline)
		})
	}
}
