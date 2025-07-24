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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
)

// TestCancelableActorCancelScaleIn tests the CancelScaleIn method directly
func TestCancelableActorCancelScaleIn(t *testing.T) {
	cases := []struct {
		desc           string
		updateList     []string
		outdatedList   []string
		offlineList    []string // instances that are offline
		targetReplicas int
		lifecycleError error
		expectedCancel int
		expectedError  bool
	}{
		{
			desc:           "no offline instances to rescue",
			updateList:     []string{"tikv-0", "tikv-1"},
			outdatedList:   []string{},
			offlineList:    []string{},
			targetReplicas: 3,
			expectedCancel: 0,
			expectedError:  false,
		},
		{
			desc:           "rescue one offline instance",
			updateList:     []string{"tikv-0"},
			outdatedList:   []string{"tikv-1"},
			offlineList:    []string{"tikv-2"},
			targetReplicas: 3,
			expectedCancel: 1,
			expectedError:  false,
		},
		{
			desc:           "rescue multiple offline instances",
			updateList:     []string{"tikv-0"},
			outdatedList:   []string{},
			offlineList:    []string{"tikv-1", "tikv-2", "tikv-3"},
			targetReplicas: 4,
			expectedCancel: 3,
			expectedError:  false,
		},
		{
			desc:           "rescue partial offline instances",
			updateList:     []string{"tikv-0", "tikv-1"},
			outdatedList:   []string{},
			offlineList:    []string{"tikv-2", "tikv-3"},
			targetReplicas: 3, // current online = 2, need only 1 more
			expectedCancel: 1,
			expectedError:  false,
		},
		{
			desc:           "no rescue needed - already sufficient replicas",
			updateList:     []string{"tikv-0", "tikv-1", "tikv-2"},
			outdatedList:   []string{},
			offlineList:    []string{"tikv-3"},
			targetReplicas: 3, // current online = 3, no rescue needed
			expectedCancel: 0,
			expectedError:  false,
		},
		{
			desc:           "lifecycle error during rescue",
			updateList:     []string{"tikv-0"},
			outdatedList:   []string{},
			offlineList:    []string{"tikv-1"},
			targetReplicas: 2,
			lifecycleError: assert.AnError,
			expectedCancel: 0,
			expectedError:  true,
		},
		{
			desc:           "rescue when both update and outdated instances exist",
			updateList:     []string{"tikv-0"},
			outdatedList:   []string{"tikv-1"},
			offlineList:    []string{"tikv-2", "tikv-3"},
			targetReplicas: 4, // current online = 2, need 2 more
			expectedCancel: 2,
			expectedError:  false,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			cli := client.NewFakeClient()

			// Create fake lifecycle
			lifecycle := &FakeLifecycle[*runtime.TiKV]{
				isOfflineInstances: make(map[string]bool),
				offlineCompleted:   make(map[string]bool),
			}
			if c.lifecycleError != nil {
				lifecycle.shouldError = true
			}

			// Create instances
			var instances []runtime.TiKV
			for _, name := range c.updateList {
				tikv := fake.FakeObj[v1alpha1.TiKV](name, fake.Label[v1alpha1.TiKV]("core.pingcap.com/instance-revision-hash", "v1"))
				instances = append(instances, *runtime.FromTiKV(tikv))
			}
			for _, name := range c.outdatedList {
				tikv := fake.FakeObj[v1alpha1.TiKV](name, fake.Label[v1alpha1.TiKV]("core.pingcap.com/instance-revision-hash", "v0"))
				instances = append(instances, *runtime.FromTiKV(tikv))
			}
			for _, name := range c.offlineList {
				tikv := fake.FakeObj[v1alpha1.TiKV](name, fake.Label[v1alpha1.TiKV]("core.pingcap.com/instance-revision-hash", "v1"))
				tikv.Spec.Offline = true
				instances = append(instances, *runtime.FromTiKV(tikv))
				lifecycle.isOfflineInstances[name] = true
			}

			// Create basic actor
			update, outdated, deleted := split(toPointers(instances), "v1")
			basicActor := &actor[runtime.TiKVTuple, *v1alpha1.TiKV, *runtime.TiKV]{
				c: cli,
				f: NewFunc[*runtime.TiKV](func() *runtime.TiKV { return &runtime.TiKV{} }),

				update:   NewState(update),
				outdated: NewState(outdated),
				deleted:  NewState(deleted),

				scaleInSelector: NewSelector[*runtime.TiKV](),
				updateSelector:  NewSelector[*runtime.TiKV](),
			}

			// Create cancelable actor
			cancelableActor := &cancelableActor[runtime.TiKVTuple, *v1alpha1.TiKV, *runtime.TiKV]{
				actor:     basicActor,
				lifecycle: lifecycle,
			}

			// Test CancelScaleIn method
			canceled, err := cancelableActor.CancelScaleIn(context.Background(), c.targetReplicas)

			// Verify results
			if c.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, c.expectedCancel, canceled, "unexpected number of instances canceled")

			// Verify lifecycle was called correctly
			if c.expectedCancel > 0 {
				assert.True(t, lifecycle.cancelScaleInCalled, "CancelScaleIn should have been called on lifecycle")
			}
		})
	}
}

// TestCancelScaleInWithComplexStates tests cancel scale-in with various instance states
func TestCancelScaleInWithComplexStates(t *testing.T) {
	cli := client.NewFakeClient()
	lifecycle := &FakeLifecycle[*runtime.TiKV]{
		isOfflineInstances: make(map[string]bool),
		offlineCompleted:   make(map[string]bool),
	}

	// Create instances in different states
	tikvUpdate := fake.FakeObj[v1alpha1.TiKV]("tikv-update", fake.Label[v1alpha1.TiKV]("core.pingcap.com/instance-revision-hash", "v1"))

	tikvOutdated := fake.FakeObj[v1alpha1.TiKV]("tikv-outdated", fake.Label[v1alpha1.TiKV]("core.pingcap.com/instance-revision-hash", "v0"))

	tikvOfflineNotCompleted := fake.FakeObj[v1alpha1.TiKV]("tikv-offline-not-completed", fake.Label[v1alpha1.TiKV]("core.pingcap.com/instance-revision-hash", "v1"))
	tikvOfflineNotCompleted.Spec.Offline = true
	lifecycle.isOfflineInstances["tikv-offline-not-completed"] = true

	tikvOfflineCompleted := fake.FakeObj[v1alpha1.TiKV]("tikv-offline-completed", fake.Label[v1alpha1.TiKV]("core.pingcap.com/instance-revision-hash", "v1"))
	tikvOfflineCompleted.Spec.Offline = true
	lifecycle.offlineCompleted["tikv-offline-completed"] = true

	instances := []*runtime.TiKV{
		runtime.FromTiKV(tikvUpdate),
		runtime.FromTiKV(tikvOutdated),
		runtime.FromTiKV(tikvOfflineNotCompleted),
		runtime.FromTiKV(tikvOfflineCompleted),
	}

	// Create basic actor
	update, outdated, deleted := split(instances, "v1")
	basicActor := &actor[runtime.TiKVTuple, *v1alpha1.TiKV, *runtime.TiKV]{
		c: cli,
		f: NewFunc[*runtime.TiKV](func() *runtime.TiKV { return &runtime.TiKV{} }),

		update:   NewState(update),
		outdated: NewState(outdated),
		deleted:  NewState(deleted),

		scaleInSelector: NewSelector[*runtime.TiKV](),
		updateSelector:  NewSelector[*runtime.TiKV](),
	}

	// Create cancelable actor
	cancelableActor := &cancelableActor[runtime.TiKVTuple, *v1alpha1.TiKV, *runtime.TiKV]{
		actor:     basicActor,
		lifecycle: lifecycle,
	}

	// Test cancel scale-in - should only rescue the not-completed offline instance
	canceled, err := cancelableActor.CancelScaleIn(context.Background(), 4)

	require.NoError(t, err)
	assert.Equal(t, 1, canceled, "should rescue only the not-completed offline instance")
	assert.True(t, lifecycle.cancelScaleInCalled, "lifecycle CancelScaleIn should have been called")
}

// Helper function to convert slice of instances to slice of pointers
func toPointers(instances []runtime.TiKV) []*runtime.TiKV {
	var result []*runtime.TiKV
	for i := range instances {
		result = append(result, &instances[i])
	}
	return result
}

// TestCancelableActorScaleInUpdate tests the overridden ScaleInUpdate method
func TestCancelableActorScaleInUpdate(t *testing.T) {
	cases := []struct {
		desc              string
		instanceName      string
		initialOffline    bool
		offlineCompleted  bool
		lifecycleError    error
		expectedWait      bool
		expectedError     bool
		expectedFinalized bool
	}{
		{
			desc:              "begin scale-in on online instance",
			instanceName:      "tikv-0",
			initialOffline:    false,
			offlineCompleted:  false,
			expectedWait:      true, // Should wait after beginning offline
			expectedError:     false,
			expectedFinalized: false,
		},
		{
			desc:              "offline in progress - should wait",
			instanceName:      "tikv-0",
			initialOffline:    true,
			offlineCompleted:  false,
			expectedWait:      true,
			expectedError:     false,
			expectedFinalized: false,
		},
		{
			desc:              "offline completed - should finalize",
			instanceName:      "tikv-0",
			initialOffline:    true,
			offlineCompleted:  true,
			expectedWait:      true, // Instance being deleted is typically unavailable
			expectedError:     false,
			expectedFinalized: true,
		},
		{
			desc:              "lifecycle error during begin",
			instanceName:      "tikv-0",
			initialOffline:    false,
			lifecycleError:    assert.AnError,
			expectedWait:      false,
			expectedError:     true,
			expectedFinalized: false,
		},
		{
			desc:              "lifecycle error during finalize",
			instanceName:      "tikv-0",
			initialOffline:    true,
			offlineCompleted:  true,
			lifecycleError:    assert.AnError,
			expectedWait:      false,
			expectedError:     true,
			expectedFinalized: false,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			// Create instance
			tikv := fake.FakeObj[v1alpha1.TiKV](c.instanceName, fake.Label[v1alpha1.TiKV]("core.pingcap.com/instance-revision-hash", "v1"))
			tikv.Spec.Offline = c.initialOffline
			instance := runtime.FromTiKV(tikv)

			// Add the instance to fake client so deferDelete can work
			// For the "lifecycle error during finalize" case, we don't add the object to simulate delete failure
			var cli client.Client
			if c.desc == "lifecycle error during finalize" {
				cli = client.NewFakeClient() // Empty client - delete will fail
			} else {
				cli = client.NewFakeClient(tikv)
			}

			// Create fake lifecycle
			lifecycle := &FakeLifecycle[*runtime.TiKV]{
				isOfflineInstances: make(map[string]bool),
				offlineCompleted:   make(map[string]bool),
			}
			lifecycle.isOfflineInstances[c.instanceName] = c.initialOffline
			lifecycle.offlineCompleted[c.instanceName] = c.offlineCompleted
			// For "lifecycle error during finalize", the error comes from client.Delete, not lifecycle
			if c.lifecycleError != nil && c.desc != "lifecycle error during finalize" {
				lifecycle.shouldError = true
			}

			// Create basic actor with instance in update state
			basicActor := &actor[runtime.TiKVTuple, *v1alpha1.TiKV, *runtime.TiKV]{
				c: cli,
				f: NewFunc[*runtime.TiKV](func() *runtime.TiKV { return &runtime.TiKV{} }),

				update:   NewState([]*runtime.TiKV{instance}),
				outdated: NewState([]*runtime.TiKV{}),
				deleted:  NewState([]*runtime.TiKV{}),

				scaleInSelector: NewSelector[*runtime.TiKV](),
				updateSelector:  NewSelector[*runtime.TiKV](),
			}

			// Create cancelable actor
			cancelableActor := &cancelableActor[runtime.TiKVTuple, *v1alpha1.TiKV, *runtime.TiKV]{
				actor:     basicActor,
				lifecycle: lifecycle,
			}

			// Test ScaleInUpdate method
			wait, err := cancelableActor.ScaleInUpdate(context.Background())

			// Verify results
			if c.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, c.expectedWait, wait, "unexpected wait result")

			// Verify lifecycle method calls
			if !c.initialOffline && !c.expectedError {
				assert.True(t, lifecycle.beginScaleInCalled, "BeginScaleIn should have been called")
			}

			// Verify instance removal from update state when finalized
			if c.expectedFinalized && !c.expectedError {
				assert.Equal(t, 0, basicActor.update.Len(), "instance should be removed from update state")
			} else if !c.expectedFinalized {
				assert.Equal(t, 1, basicActor.update.Len(), "instance should remain in update state")
			}
		})
	}
}

// TestCancelableActorScaleInOutdated tests the overridden ScaleInOutdated method
func TestCancelableActorScaleInOutdated(t *testing.T) {
	cases := []struct {
		desc              string
		instanceName      string
		initialOffline    bool
		offlineCompleted  bool
		lifecycleError    error
		expectedWait      bool
		expectedError     bool
		expectedFinalized bool
	}{
		{
			desc:              "begin scale-in on online outdated instance",
			instanceName:      "tikv-outdated",
			initialOffline:    false,
			offlineCompleted:  false,
			expectedWait:      true, // Should wait after beginning offline
			expectedError:     false,
			expectedFinalized: false,
		},
		{
			desc:              "offline in progress for outdated - should wait",
			instanceName:      "tikv-outdated",
			initialOffline:    true,
			offlineCompleted:  false,
			expectedWait:      true,
			expectedError:     false,
			expectedFinalized: false,
		},
		{
			desc:              "offline completed for outdated - should defer delete",
			instanceName:      "tikv-outdated",
			initialOffline:    true,
			offlineCompleted:  true,
			expectedWait:      true, // Instance being deleted is typically unavailable
			expectedError:     false,
			expectedFinalized: true,
		},
		{
			desc:              "lifecycle error during begin for outdated",
			instanceName:      "tikv-outdated",
			initialOffline:    false,
			lifecycleError:    assert.AnError,
			expectedWait:      false,
			expectedError:     true,
			expectedFinalized: false,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			// Create instance
			tikv := fake.FakeObj[v1alpha1.TiKV](c.instanceName, fake.Label[v1alpha1.TiKV]("core.pingcap.com/instance-revision-hash", "v0"))
			tikv.Spec.Offline = c.initialOffline
			instance := runtime.FromTiKV(tikv)

			// Add the instance to fake client so deferDelete can work
			cli := client.NewFakeClient(tikv)

			// Create fake lifecycle
			lifecycle := &FakeLifecycle[*runtime.TiKV]{
				isOfflineInstances: make(map[string]bool),
				offlineCompleted:   make(map[string]bool),
			}
			lifecycle.isOfflineInstances[c.instanceName] = c.initialOffline
			lifecycle.offlineCompleted[c.instanceName] = c.offlineCompleted
			if c.lifecycleError != nil {
				lifecycle.shouldError = true
			}

			// Create basic actor with instance in outdated state
			basicActor := &actor[runtime.TiKVTuple, *v1alpha1.TiKV, *runtime.TiKV]{
				c: cli,
				f: NewFunc[*runtime.TiKV](func() *runtime.TiKV { return &runtime.TiKV{} }),

				update:   NewState([]*runtime.TiKV{}),
				outdated: NewState([]*runtime.TiKV{instance}),
				deleted:  NewState([]*runtime.TiKV{}),

				scaleInSelector: NewSelector[*runtime.TiKV](),
				updateSelector:  NewSelector[*runtime.TiKV](),
			}

			// Create cancelable actor
			cancelableActor := &cancelableActor[runtime.TiKVTuple, *v1alpha1.TiKV, *runtime.TiKV]{
				actor:     basicActor,
				lifecycle: lifecycle,
			}

			// Test ScaleInOutdated method
			wait, err := cancelableActor.ScaleInOutdated(context.Background())

			// Verify results
			if c.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, c.expectedWait, wait, "unexpected wait result")

			// Verify lifecycle method calls
			if !c.initialOffline && !c.expectedError {
				assert.True(t, lifecycle.beginScaleInCalled, "BeginScaleIn should have been called")
			}

			// Verify instance removal from outdated state when finalized
			if c.expectedFinalized && !c.expectedError {
				assert.Equal(t, 0, basicActor.outdated.Len(), "instance should be removed from outdated state")
				assert.Equal(t, 1, basicActor.deleted.Len(), "instance should be added to deleted state (defer delete)")
			} else if !c.expectedFinalized {
				assert.Equal(t, 1, basicActor.outdated.Len(), "instance should remain in outdated state")
			}
		})
	}
}

// TestCancelableActorScaleInLifecycleFlow tests the complete lifecycle flow
func TestCancelableActorScaleInLifecycleFlow(t *testing.T) {
	// Create instance first
	tikv := fake.FakeObj[v1alpha1.TiKV]("tikv-flow", fake.Label[v1alpha1.TiKV]("core.pingcap.com/instance-revision-hash", "v1"))
	instance := runtime.FromTiKV(tikv)

	// Create client with the instance so deletion can succeed
	cli := client.NewFakeClient(tikv)

	lifecycle := &FakeLifecycle[*runtime.TiKV]{
		isOfflineInstances: make(map[string]bool),
		offlineCompleted:   make(map[string]bool),
	}

	// Create basic actor with instance in update state
	basicActor := &actor[runtime.TiKVTuple, *v1alpha1.TiKV, *runtime.TiKV]{
		c: cli,
		f: NewFunc[*runtime.TiKV](func() *runtime.TiKV { return &runtime.TiKV{} }),

		update:   NewState([]*runtime.TiKV{instance}),
		outdated: NewState([]*runtime.TiKV{}),
		deleted:  NewState([]*runtime.TiKV{}),

		scaleInSelector: NewSelector[*runtime.TiKV](),
		updateSelector:  NewSelector[*runtime.TiKV](),
	}

	cancelableActor := &cancelableActor[runtime.TiKVTuple, *v1alpha1.TiKV, *runtime.TiKV]{
		actor:     basicActor,
		lifecycle: lifecycle,
	}

	// Step 1: Begin scale-in (online -> offline)
	wait, err := cancelableActor.ScaleInUpdate(context.Background())
	require.NoError(t, err)
	assert.True(t, wait, "should wait after beginning scale-in")
	assert.True(t, lifecycle.beginScaleInCalled, "BeginScaleIn should be called")
	assert.Equal(t, 1, basicActor.update.Len(), "instance should remain in update state during offline process")

	// Reset lifecycle flags for next step
	lifecycle.beginScaleInCalled = false

	// Step 2: Offline in progress (should wait)
	lifecycle.isOfflineInstances["tikv-flow"] = true
	wait, err = cancelableActor.ScaleInUpdate(context.Background())
	require.NoError(t, err)
	assert.True(t, wait, "should wait while offline is in progress")
	assert.False(t, lifecycle.beginScaleInCalled, "BeginScaleIn should not be called again")
	assert.Equal(t, 1, basicActor.update.Len(), "instance should remain in update state")

	// Step 3: Offline completed (should finalize)
	lifecycle.offlineCompleted["tikv-flow"] = true
	wait, err = cancelableActor.ScaleInUpdate(context.Background())
	require.NoError(t, err)
	assert.True(t, wait, "offline instances are typically unavailable during deletion")
	assert.Equal(t, 0, basicActor.update.Len(), "instance should be removed from update state")
}

// TestCancelableActorScaleInNotFound tests handling of non-existent instances
func TestCancelableActorScaleInNotFound(t *testing.T) {
	cli := client.NewFakeClient()
	lifecycle := &FakeLifecycle[*runtime.TiKV]{
		isOfflineInstances: make(map[string]bool),
		offlineCompleted:   make(map[string]bool),
	}

	// Create basic actor with empty states
	basicActor := &actor[runtime.TiKVTuple, *v1alpha1.TiKV, *runtime.TiKV]{
		c: cli,
		f: NewFunc[*runtime.TiKV](func() *runtime.TiKV { return &runtime.TiKV{} }),

		update:   NewState([]*runtime.TiKV{}),
		outdated: NewState([]*runtime.TiKV{}),
		deleted:  NewState([]*runtime.TiKV{}),

		scaleInSelector: NewSelector[*runtime.TiKV](),
		updateSelector:  NewSelector[*runtime.TiKV](),
	}

	cancelableActor := &cancelableActor[runtime.TiKVTuple, *v1alpha1.TiKV, *runtime.TiKV]{
		actor:     basicActor,
		lifecycle: lifecycle,
	}

	// Test ScaleInUpdate with no instances
	func() {
		defer func() {
			if r := recover(); r != nil {
				assert.Contains(t, fmt.Sprintf("%v", r), "index out of range")
			}
		}()
		wait, err := cancelableActor.ScaleInUpdate(context.Background())
		// If we reach here without panic, it should be an error
		if err == nil {
			t.Fatal("expected error or panic when no instances available")
		}
		require.Error(t, err)
		assert.False(t, wait)
	}()

	// Test ScaleInOutdated with no instances
	func() {
		defer func() {
			if r := recover(); r != nil {
				assert.Contains(t, fmt.Sprintf("%v", r), "index out of range")
			}
		}()
		wait, err := cancelableActor.ScaleInOutdated(context.Background())
		// If we reach here without panic, it should be an error
		if err == nil {
			t.Fatal("expected error or panic when no instances available")
		}
		require.Error(t, err)
		assert.False(t, wait)
	}()
}
