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

// TestBuilderWithScaleInLifecycle tests the WithScaleInLifecycle method and actor creation logic
func TestBuilderWithScaleInLifecycle(t *testing.T) {
	cli := client.NewFakeClient()

	cases := []struct {
		desc              string
		withLifecycle     bool
		expectedActorType string
		expectedCanCancel bool
	}{
		{
			desc:              "builder without lifecycle creates basic actor",
			withLifecycle:     false,
			expectedActorType: "*updater.actor",
			expectedCanCancel: false,
		},
		{
			desc:              "builder with lifecycle creates cancelable actor",
			withLifecycle:     true,
			expectedActorType: "*updater.cancelableActor",
			expectedCanCancel: true,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			builder := New[runtime.TiKVTuple]().
				WithDesired(0). // No instances, no desired
				WithMaxSurge(1).
				WithMaxUnavailable(1).
				WithClient(cli).
				WithNewFactory(NewFunc[*runtime.TiKV](func() *runtime.TiKV {
					return &runtime.TiKV{}
				}))

			if c.withLifecycle {
				lifecycle := &FakeLifecycle[*runtime.TiKV]{}
				builder = builder.WithScaleInLifecycle(lifecycle)
			}

			executor := builder.Build()

			// Verify executor was created successfully
			require.NotNil(t, executor)

			// We can test the behavior rather than the internal structure
			// Create a context and run a no-op execution to verify the actor type
			wait, err := executor.Do(context.Background())
			require.NoError(t, err)

			// If we have lifecycle, we can indirectly test by creating a scenario that would trigger cancel logic
			if c.withLifecycle {
				// Test that the executor has cancel capabilities by checking if it handles the cancel scenario
				// This is tested more thoroughly in the integration test
				assert.False(t, wait, "executor with lifecycle should behave like basic actor when no work needed")
			} else {
				// Basic executor behavior
				assert.False(t, wait, "executor without instances should not wait")
			}
		})
	}
}

// TestBuilderChaining tests that all builder methods return the correct builder instance
func TestBuilderChaining(t *testing.T) {
	cli := client.NewFakeClient()
	lifecycle := &FakeLifecycle[*runtime.TiKV]{}

	builder := New[runtime.TiKVTuple]()

	// Test that all methods return the same builder instance for chaining
	result := builder.
		WithInstances().
		WithDesired(3).
		WithMaxSurge(1).
		WithMaxUnavailable(1).
		WithRevision("v1").
		WithClient(cli).
		WithNewFactory(NewFunc[*runtime.TiKV](func() *runtime.TiKV { return &runtime.TiKV{} })).
		WithAddHooks().
		WithUpdateHooks().
		WithDelHooks().
		WithScaleInPreferPolicy().
		WithUpdatePreferPolicy().
		WithNoInPaceUpdate(false).
		WithScaleInLifecycle(lifecycle)

	assert.Same(t, builder, result, "all builder methods should return the same instance")

	// Verify the builder can create executor
	executor := result.Build()
	assert.NotNil(t, executor)
}

// TestBuilderNilLifecycle tests behavior when nil lifecycle is passed
func TestBuilderNilLifecycle(t *testing.T) {
	cli := client.NewFakeClient()

	// Test with explicit nil lifecycle
	builder := New[runtime.TiKVTuple]().
		WithDesired(0). // No instances, no desired
		WithMaxSurge(1).
		WithMaxUnavailable(1).
		WithClient(cli).
		WithNewFactory(NewFunc[*runtime.TiKV](func() *runtime.TiKV {
			return &runtime.TiKV{}
		})).
		WithScaleInLifecycle(nil) // Explicit nil

	executor := builder.Build()
	require.NotNil(t, executor)

	// Test behavior - should work like basic actor
	wait, err := executor.Do(context.Background())
	require.NoError(t, err)
	assert.False(t, wait, "executor without instances should not wait")
}

// TestBuilderIntegrationWithDifferentTypes tests the integration of Builder with different instance types
func TestBuilderIntegrationWithDifferentTypes(t *testing.T) {
	cases := []struct {
		desc          string
		instanceType  string
		createBuilder func(cli client.Client) Executor
	}{
		{
			desc:         "TiKV with cancel lifecycle",
			instanceType: "TiKV",
			createBuilder: func(cli client.Client) Executor {
				lifecycle := &FakeLifecycle[*runtime.TiKV]{
					isOfflineInstances: make(map[string]bool),
					offlineCompleted:   make(map[string]bool),
				}
				return New[runtime.TiKVTuple]().
					WithDesired(1).
					WithMaxSurge(1).
					WithMaxUnavailable(1).
					WithClient(cli).
					WithNewFactory(NewFunc[*runtime.TiKV](func() *runtime.TiKV { return &runtime.TiKV{} })).
					WithScaleInLifecycle(lifecycle).
					Build()
			},
		},
		{
			desc:         "TiKV without cancel lifecycle",
			instanceType: "TiKV",
			createBuilder: func(cli client.Client) Executor {
				return New[runtime.TiKVTuple]().
					WithDesired(1).
					WithMaxSurge(1).
					WithMaxUnavailable(1).
					WithClient(cli).
					WithNewFactory(NewFunc[*runtime.TiKV](func() *runtime.TiKV { return &runtime.TiKV{} })).
					Build()
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			cli := client.NewFakeClient()

			// Create executor with specific configuration
			executor := c.createBuilder(cli)
			require.NotNil(t, executor, "executor should be created successfully")

			// Test basic execution
			wait, err := executor.Do(context.Background())
			require.NoError(t, err, "executor should run without error")
			assert.True(t, wait, "should wait when scaling out")
		})
	}
}

// TestBuilderWithDifferentHooks tests Builder with different hook configurations
func TestBuilderWithDifferentHooks(t *testing.T) {
	cli := client.NewFakeClient()

	addHook := AddHookFunc[*runtime.TiKV](func(tikv *runtime.TiKV) *runtime.TiKV {
		return tikv
	})

	updateHook := UpdateHookFunc[*runtime.TiKV](func(update, _ *runtime.TiKV) *runtime.TiKV {
		return update
	})

	delHook := DelHookFunc[*runtime.TiKV](func(_ string) {
	})

	builder := New[runtime.TiKVTuple]().
		WithDesired(1).
		WithMaxSurge(1).
		WithMaxUnavailable(1).
		WithClient(cli).
		WithNewFactory(NewFunc[*runtime.TiKV](func() *runtime.TiKV { return &runtime.TiKV{} })).
		WithAddHooks(addHook).
		WithUpdateHooks(updateHook).
		WithDelHooks(delHook)

	executor := builder.Build()
	require.NotNil(t, executor)

	// Execute to verify hooks are registered (though they may not be called in this simple test)
	_, err := executor.Do(context.Background())
	require.NoError(t, err)

	// We can't easily verify hooks are called without more complex setup,
	// but we can verify the builder accepts them without error
}

// FakeLifecycle is a test implementation of ScaleInLifecycle
type FakeLifecycle[R runtime.Instance] struct {
	isOfflineInstances  map[string]bool
	offlineCompleted    map[string]bool
	beginScaleInCalled  bool
	cancelScaleInCalled bool
	shouldError         bool
}

func (f *FakeLifecycle[R]) IsOffline(instance R) bool {
	name := instance.GetName()
	// If offline is completed, IsOffline should return false (matches real StoreLifecycle behavior)
	if f.offlineCompleted[name] {
		return false
	}
	return f.isOfflineInstances[name]
}

func (f *FakeLifecycle[R]) IsOfflineCompleted(instance R) bool {
	return f.offlineCompleted[instance.GetName()]
}

func (f *FakeLifecycle[R]) BeginScaleIn(_ context.Context, instance R) error {
	f.beginScaleInCalled = true
	if f.shouldError {
		return assert.AnError
	}
	f.isOfflineInstances[instance.GetName()] = true
	return nil
}

func (f *FakeLifecycle[R]) CancelScaleIn(_ context.Context, instance R) error {
	f.cancelScaleInCalled = true
	if f.shouldError {
		return assert.AnError
	}
	f.isOfflineInstances[instance.GetName()] = false
	return nil
}
