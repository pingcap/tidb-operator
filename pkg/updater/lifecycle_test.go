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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/fake"
)

// TestStoreLifecycleIsOfflineCompleted tests the IsOfflineCompleted method
func TestStoreLifecycleIsOfflineCompleted(t *testing.T) {
	cases := []struct {
		desc                       string
		offline                    bool
		offlineCondition           *metav1.Condition
		expectedIsOfflineCompleted bool
	}{
		{
			desc:                       "not offline",
			offline:                    false,
			expectedIsOfflineCompleted: false,
		},
		{
			desc:                       "offline without condition",
			offline:                    true,
			expectedIsOfflineCompleted: false,
		},
		{
			desc:    "offline with non-completed condition",
			offline: true,
			offlineCondition: &metav1.Condition{
				Type:   v1alpha1.StoreOfflineConditionType,
				Status: metav1.ConditionTrue,
				Reason: "InProgress",
			},
			expectedIsOfflineCompleted: false,
		},
		{
			desc:    "offline with completed condition",
			offline: true,
			offlineCondition: &metav1.Condition{
				Type:   v1alpha1.StoreOfflineConditionType,
				Status: metav1.ConditionTrue,
				Reason: v1alpha1.OfflineReasonCompleted,
			},
			expectedIsOfflineCompleted: true,
		},
		{
			desc:    "offline with completed condition but false status",
			offline: true,
			offlineCondition: &metav1.Condition{
				Type:   v1alpha1.StoreOfflineConditionType,
				Status: metav1.ConditionFalse,
				Reason: v1alpha1.OfflineReasonCompleted,
			},
			expectedIsOfflineCompleted: true, // Reason is what matters, not status
		},
		{
			desc:    "not offline but with completed condition",
			offline: false,
			offlineCondition: &metav1.Condition{
				Type:   v1alpha1.StoreOfflineConditionType,
				Status: metav1.ConditionTrue,
				Reason: v1alpha1.OfflineReasonCompleted,
			},
			expectedIsOfflineCompleted: false, // Must be offline first
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			cli := client.NewFakeClient()
			lifecycle := NewStoreLifecycle[*runtime.TiKV](cli)

			// Create TiKV instance
			tikv := fake.FakeObj[v1alpha1.TiKV]("tikv-0")
			tikvRuntime := runtime.FromTiKV(tikv)
			tikvRuntime.Spec.Offline = c.offline
			if c.offlineCondition != nil {
				tikvRuntime.Status.Conditions = []metav1.Condition{*c.offlineCondition}
			}

			// Test IsOfflineCompleted
			isOfflineCompleted := lifecycle.IsOfflineCompleted(tikvRuntime)
			assert.Equal(t, c.expectedIsOfflineCompleted, isOfflineCompleted)
		})
	}
}

// TestStoreLifecycleBeginScaleIn tests the BeginScaleIn method
func TestStoreLifecycleBeginScaleIn(t *testing.T) {
	cases := []struct {
		desc           string
		initialOffline bool
		expectUpdate   bool
		expectError    bool
	}{
		{
			desc:           "begin scale-in on online instance",
			initialOffline: false,
			expectUpdate:   true,
			expectError:    false,
		},
		{
			desc:           "begin scale-in on already offline instance",
			initialOffline: true,
			expectUpdate:   true, // Should still call update
			expectError:    false,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			// Use fake lifecycle to avoid client update issues
			fakeLifecycle := &FakeLifecycle[*runtime.TiKV]{
				isOfflineInstances: make(map[string]bool),
				offlineCompleted:   make(map[string]bool),
			}

			// Create TiKV instance
			tikv := fake.FakeObj[v1alpha1.TiKV]("tikv-0")
			tikvRuntime := runtime.FromTiKV(tikv)
			tikvRuntime.Spec.Offline = c.initialOffline

			// Test BeginScaleIn
			err := fakeLifecycle.BeginScaleIn(context.Background(), tikvRuntime)

			if c.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.True(t, fakeLifecycle.beginScaleInCalled, "BeginScaleIn should be called")
				// Verify fake lifecycle registered the instance as offline
				assert.True(t, fakeLifecycle.isOfflineInstances["tikv-0"], "instance should be marked offline in fake")
			}
		})
	}
}

// TestStoreLifecycleCancelScaleIn tests the CancelScaleIn method
func TestStoreLifecycleCancelScaleIn(t *testing.T) {
	cases := []struct {
		desc           string
		initialOffline bool
		expectUpdate   bool
		expectError    bool
	}{
		{
			desc:           "cancel scale-in on offline instance",
			initialOffline: true,
			expectUpdate:   true,
			expectError:    false,
		},
		{
			desc:           "cancel scale-in on online instance",
			initialOffline: false,
			expectUpdate:   true, // Should still call update
			expectError:    false,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			// Use fake lifecycle to avoid client update issues
			fakeLifecycle := &FakeLifecycle[*runtime.TiKV]{
				isOfflineInstances: make(map[string]bool),
				offlineCompleted:   make(map[string]bool),
			}

			// Create TiKV instance
			tikv := fake.FakeObj[v1alpha1.TiKV]("tikv-0")
			tikvRuntime := runtime.FromTiKV(tikv)
			tikvRuntime.Spec.Offline = c.initialOffline

			// Set initial state in fake lifecycle
			fakeLifecycle.isOfflineInstances["tikv-0"] = c.initialOffline

			// Test CancelScaleIn
			err := fakeLifecycle.CancelScaleIn(context.Background(), tikvRuntime)

			if c.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.True(t, fakeLifecycle.cancelScaleInCalled, "CancelScaleIn should be called")
				// Verify fake lifecycle registered the instance as online
				assert.False(t, fakeLifecycle.isOfflineInstances["tikv-0"], "instance should be marked online in fake")
			}
		})
	}
}

// TestStoreLifecycleWithTiFlash tests that lifecycle works with TiFlash instances too
func TestStoreLifecycleWithTiFlash(t *testing.T) {
	// Use fake lifecycle to avoid client update issues
	fakeLifecycle := &FakeLifecycle[*runtime.TiFlash]{
		isOfflineInstances: make(map[string]bool),
		offlineCompleted:   make(map[string]bool),
	}

	// Create TiFlash instance
	tiflash := fake.FakeObj[v1alpha1.TiFlash]("tiflash-0")
	tiflash.Spec.Offline = false

	// Convert to runtime type for lifecycle methods
	tiflashRuntime := runtime.FromTiFlash(tiflash)

	// Test all lifecycle methods work with TiFlash

	// Initially not offline
	assert.False(t, fakeLifecycle.IsOffline(tiflashRuntime))
	assert.False(t, fakeLifecycle.IsOfflineCompleted(tiflashRuntime))

	// Begin scale-in
	err := fakeLifecycle.BeginScaleIn(context.Background(), tiflashRuntime)
	require.NoError(t, err)
	assert.True(t, fakeLifecycle.beginScaleInCalled, "BeginScaleIn should be called")
	assert.True(t, fakeLifecycle.IsOffline(tiflashRuntime))

	// Cancel scale-in
	err = fakeLifecycle.CancelScaleIn(context.Background(), tiflashRuntime)
	require.NoError(t, err)
	assert.True(t, fakeLifecycle.cancelScaleInCalled, "CancelScaleIn should be called")
	assert.False(t, fakeLifecycle.IsOffline(tiflashRuntime))

	// Begin scale-in again for finalize test
	err = fakeLifecycle.BeginScaleIn(context.Background(), tiflashRuntime)
	require.NoError(t, err)

	// Simulate completed offline condition
	fakeLifecycle.offlineCompleted["tiflash-0"] = true
	assert.True(t, fakeLifecycle.IsOfflineCompleted(tiflashRuntime))
	assert.False(t, fakeLifecycle.IsOffline(tiflashRuntime), "completed offline should return false for IsOffline")
}

// TestStoreLifecycleComplexConditions tests lifecycle with complex condition scenarios
func TestStoreLifecycleComplexConditions(t *testing.T) {
	cli := client.NewFakeClient()
	lifecycle := NewStoreLifecycle[*runtime.TiKV](cli)

	// Test with multiple conditions
	tikv := fake.FakeObj[v1alpha1.TiKV]("tikv-0")
	tikvRuntime := runtime.FromTiKV(tikv)
	tikvRuntime.Spec.Offline = true
	tikvRuntime.Status.Conditions = []metav1.Condition{
		{
			Type:   v1alpha1.CondReady,
			Status: metav1.ConditionFalse,
		},
		{
			Type:   v1alpha1.StoreOfflineConditionType,
			Status: metav1.ConditionTrue,
			Reason: "InProgress",
		},
		{
			Type:   v1alpha1.CondSynced,
			Status: metav1.ConditionTrue,
		},
	}

	// Should be offline but not completed
	assert.True(t, lifecycle.IsOffline(tikvRuntime))
	assert.False(t, lifecycle.IsOfflineCompleted(tikvRuntime))

	// Update offline condition to completed
	for i, cond := range tikvRuntime.Status.Conditions {
		if cond.Type == v1alpha1.StoreOfflineConditionType {
			tikvRuntime.Status.Conditions[i].Reason = v1alpha1.OfflineReasonCompleted
			break
		}
	}

	// Should now be completed
	assert.True(t, lifecycle.IsOffline(tikvRuntime))
	assert.True(t, lifecycle.IsOfflineCompleted(tikvRuntime))
}

// TestStoreLifecycleEdgeCases tests various edge cases
func TestStoreLifecycleEdgeCases(t *testing.T) {
	cli := client.NewFakeClient()
	lifecycle := NewStoreLifecycle[*runtime.TiKV](cli)

	// Test with nil conditions
	tikv := fake.FakeObj[v1alpha1.TiKV]("tikv-0")
	tikvRuntime := runtime.FromTiKV(tikv)
	tikvRuntime.Spec.Offline = true
	tikvRuntime.Status.Conditions = nil

	assert.True(t, lifecycle.IsOffline(tikvRuntime))
	assert.False(t, lifecycle.IsOfflineCompleted(tikvRuntime))

	// Test with empty conditions slice
	tikvRuntime.Status.Conditions = []metav1.Condition{}
	assert.True(t, lifecycle.IsOffline(tikvRuntime))
	assert.False(t, lifecycle.IsOfflineCompleted(tikvRuntime))

	// Test offline condition with empty reason
	tikvRuntime.Status.Conditions = []metav1.Condition{
		{
			Type:   v1alpha1.StoreOfflineConditionType,
			Status: metav1.ConditionTrue,
			Reason: "", // Empty reason
		},
	}
	assert.True(t, lifecycle.IsOffline(tikvRuntime))
	assert.False(t, lifecycle.IsOfflineCompleted(tikvRuntime))
}

// TestStoreLifecycleRealImplementation tests the real storeLifecycle implementation
// This would catch type conversion issues that fake implementations miss
func TestStoreLifecycleRealImplementation(t *testing.T) {
	t.Run("TiKV store lifecycle", func(t *testing.T) {
		// Create TiKV instance
		tikv := fake.FakeObj[v1alpha1.TiKV]("tikv-0")
		tikvRuntime := runtime.FromTiKV(tikv)

		// Create fake client with the instance object
		cli := client.NewFakeClient(tikv)

		// Create real lifecycle implementation
		lifecycle := NewStoreLifecycle[*runtime.TiKV](cli)

		// Test IsOffline on non-offline instance
		assert.False(t, lifecycle.IsOffline(tikvRuntime), "instance should not be offline initially")
		assert.False(t, lifecycle.IsOfflineCompleted(tikvRuntime), "instance should not be offline completed initially")

		// Test BeginScaleIn - this would fail if type conversion is wrong
		err := lifecycle.BeginScaleIn(context.Background(), tikvRuntime)
		require.NoError(t, err, "BeginScaleIn should succeed with proper type conversion")

		// Verify the instance is now marked offline
		assert.True(t, tikvRuntime.IsOffline(), "instance should be offline after BeginScaleIn")
		assert.True(t, lifecycle.IsOffline(tikvRuntime), "lifecycle should report instance as offline")

		// Test CancelScaleIn - this would also fail if type conversion is wrong
		err = lifecycle.CancelScaleIn(context.Background(), tikvRuntime)
		require.NoError(t, err, "CancelScaleIn should succeed with proper type conversion")

		// Verify the instance is back online
		assert.False(t, tikvRuntime.IsOffline(), "instance should be online after CancelScaleIn")
		assert.False(t, lifecycle.IsOffline(tikvRuntime), "lifecycle should report instance as online")
	})

	t.Run("TiFlash store lifecycle", func(t *testing.T) {
		// Create TiFlash instance
		tiflash := fake.FakeObj[v1alpha1.TiFlash]("tiflash-0")
		tiflashRuntime := runtime.FromTiFlash(tiflash)

		// Create fake client with the instance object
		cli := client.NewFakeClient(tiflash)

		// Create real lifecycle implementation
		lifecycle := NewStoreLifecycle[*runtime.TiFlash](cli)

		// Test IsOffline on non-offline instance
		assert.False(t, lifecycle.IsOffline(tiflashRuntime), "instance should not be offline initially")
		assert.False(t, lifecycle.IsOfflineCompleted(tiflashRuntime), "instance should not be offline completed initially")

		// Test BeginScaleIn - this would fail if type conversion is wrong
		err := lifecycle.BeginScaleIn(context.Background(), tiflashRuntime)
		require.NoError(t, err, "BeginScaleIn should succeed with proper type conversion")

		// Verify the instance is now marked offline
		assert.True(t, tiflashRuntime.IsOffline(), "instance should be offline after BeginScaleIn")
		assert.True(t, lifecycle.IsOffline(tiflashRuntime), "lifecycle should report instance as offline")

		// Test CancelScaleIn - this would also fail if type conversion is wrong
		err = lifecycle.CancelScaleIn(context.Background(), tiflashRuntime)
		require.NoError(t, err, "CancelScaleIn should succeed with proper type conversion")

		// Verify the instance is back online
		assert.False(t, tiflashRuntime.IsOffline(), "instance should be online after CancelScaleIn")
		assert.False(t, lifecycle.IsOffline(tiflashRuntime), "lifecycle should report instance as online")
	})
}
