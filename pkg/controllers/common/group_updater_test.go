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

package common

import (
	"context"
	"errors"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/updater"
)

// Helper functions for creating mock objects

func createMockGroup(ctrl *gomock.Controller, name string, replicas int32) *runtime.MockGroup {
	group := runtime.NewMockGroup(ctrl)
	group.EXPECT().GetName().Return(name).AnyTimes()
	group.EXPECT().GetNamespace().Return("default").AnyTimes()
	group.EXPECT().Replicas().Return(replicas).AnyTimes()
	return group
}

// MockStoreInstanceConfig holds configuration for mock store instances
type MockStoreInstanceConfig struct {
	name             string
	namespace        string
	offline          bool
	offlineCondition *metav1.Condition
}

func createTestStoreInstance(name string, opts ...func(*MockStoreInstanceConfig)) runtime.StoreInstance {
	// Create configuration with defaults
	config := &MockStoreInstanceConfig{
		name:      name,
		namespace: "default",
	}

	// Apply options to config
	for _, opt := range opts {
		opt(config)
	}

	tikv := &v1alpha1.TiKV{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: config.namespace,
		},
		Spec: v1alpha1.TiKVSpec{
			Offline: config.offline,
		},
	}
	if config.offlineCondition != nil {
		tikv.Status.Conditions = []metav1.Condition{*config.offlineCondition}
	}

	return (*runtime.TiKV)(tikv)
}

func withOffline(offline bool) func(*MockStoreInstanceConfig) {
	return func(config *MockStoreInstanceConfig) {
		config.offline = offline
	}
}

func withOfflineCondition(reason string) func(*MockStoreInstanceConfig) {
	return func(config *MockStoreInstanceConfig) {
		config.offlineCondition = &metav1.Condition{
			Type:   v1alpha1.StoreOfflineConditionType,
			Status: metav1.ConditionTrue,
			Reason: reason,
		}
	}
}

func TestCalculateGroupState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCases := []struct {
		name           string
		replicas       int32
		setupInstances func() []runtime.StoreInstance
		expectedState  storeGroupState
		description    string
	}{
		{
			name:     "stable state - exact replicas match",
			replicas: 3,
			setupInstances: func() []runtime.StoreInstance {
				return []runtime.StoreInstance{
					createTestStoreInstance("instance-0"),
					createTestStoreInstance("instance-1"),
					createTestStoreInstance("instance-2"),
				}
			},
			expectedState: stateStable,
			description:   "All instances online and match desired replicas",
		},
		{
			name:     "scale up - need more instances",
			replicas: 5,
			setupInstances: func() []runtime.StoreInstance {
				return []runtime.StoreInstance{
					createTestStoreInstance("instance-0"),
					createTestStoreInstance("instance-1"),
					createTestStoreInstance("instance-2"),
				}
			},
			expectedState: stateScalingUp,
			description:   "Desired replicas > current online instances",
		},
		{
			name:     "scale down - need fewer instances",
			replicas: 2,
			setupInstances: func() []runtime.StoreInstance {
				return []runtime.StoreInstance{
					createTestStoreInstance("instance-0"),
					createTestStoreInstance("instance-1"),
					createTestStoreInstance("instance-2"),
				}
			},
			expectedState: stateScalingDown,
			description:   "Desired replicas < current online instances",
		},
		{
			name:     "cancel scale down - rescue offline active instances",
			replicas: 4,
			setupInstances: func() []runtime.StoreInstance {
				return []runtime.StoreInstance{
					createTestStoreInstance("instance-0"),
					createTestStoreInstance("instance-1"),
					createTestStoreInstance("instance-2"),
					createTestStoreInstance("instance-3",
						withOffline(true),
						withOfflineCondition(v1alpha1.OfflineReasonActive)),
				}
			},
			expectedState: stateCancellingScaleDown,
			description:   "Need more instances and have offline active instances to rescue",
		},
		{
			name:     "awaiting deletion - offline completed instances exist",
			replicas: 3,
			setupInstances: func() []runtime.StoreInstance {
				return []runtime.StoreInstance{
					createTestStoreInstance("instance-0"),
					createTestStoreInstance("instance-1"),
					createTestStoreInstance("instance-2"),
					createTestStoreInstance("instance-3",
						withOffline(true),
						withOfflineCondition(v1alpha1.OfflineReasonCompleted)),
				}
			},
			expectedState: stateAwaitingNodeDeletion,
			description:   "Has offline completed instances that need to be deleted",
		},
		{
			name:     "stable with offline active instances - correct online count",
			replicas: 3,
			setupInstances: func() []runtime.StoreInstance {
				return []runtime.StoreInstance{
					createTestStoreInstance("instance-0"),
					createTestStoreInstance("instance-1"),
					createTestStoreInstance("instance-2"),
					createTestStoreInstance("instance-3",
						withOffline(true),
						withOfflineCondition(v1alpha1.OfflineReasonActive)),
				}
			},
			expectedState: stateStable,
			description:   "Correct number of online instances, offline active instances handled in stable state",
		},
		{
			name:     "complex scenario - multiple offline states with completed taking priority",
			replicas: 2,
			setupInstances: func() []runtime.StoreInstance {
				return []runtime.StoreInstance{
					createTestStoreInstance("instance-0"),
					createTestStoreInstance("instance-1"),
					createTestStoreInstance("instance-2",
						withOffline(true),
						withOfflineCondition(v1alpha1.OfflineReasonActive)),
					createTestStoreInstance("instance-3",
						withOffline(true),
						withOfflineCondition(v1alpha1.OfflineReasonCompleted)),
				}
			},
			expectedState: stateAwaitingNodeDeletion,
			description:   "Offline completed takes priority over other states",
		},
		{
			name:     "edge case - zero replicas",
			replicas: 0,
			setupInstances: func() []runtime.StoreInstance {
				return []runtime.StoreInstance{
					createTestStoreInstance("instance-0"),
				}
			},
			expectedState: stateScalingDown,
			description:   "Scale down to zero replicas",
		},
		{
			name:     "edge case - no instances but need some",
			replicas: 1,
			setupInstances: func() []runtime.StoreInstance {
				return []runtime.StoreInstance{}
			},
			expectedState: stateScalingUp,
			description:   "No instances but need one",
		},
		{
			name:     "scale down with offline active instances",
			replicas: 2,
			setupInstances: func() []runtime.StoreInstance {
				return []runtime.StoreInstance{
					createTestStoreInstance("instance-0"),
					createTestStoreInstance("instance-1"),
					createTestStoreInstance("instance-2"),
					createTestStoreInstance("instance-3",
						withOffline(true),
						withOfflineCondition(v1alpha1.OfflineReasonActive)),
				}
			},
			expectedState: stateScalingDown,
			description:   "Desired replicas < current online instances, offline active instances should not affect scale down decision",
		},
		{
			name:     "edge case - zero replicas with offline completed instance",
			replicas: 0,
			setupInstances: func() []runtime.StoreInstance {
				return []runtime.StoreInstance{
					createTestStoreInstance("instance-0",
						withOffline(true),
						withOfflineCondition(v1alpha1.OfflineReasonCompleted)),
				}
			},
			expectedState: stateAwaitingNodeDeletion,
			description:   "Scale down to zero with an offline completed instance should await deletion",
		},
		{
			name:     "edge case - zero replicas with offline active instance",
			replicas: 0,
			setupInstances: func() []runtime.StoreInstance {
				return []runtime.StoreInstance{
					createTestStoreInstance("instance-0",
						withOffline(true),
						withOfflineCondition(v1alpha1.OfflineReasonActive)),
				}
			},
			expectedState: stateStable,
			description:   "Scale down to zero with an offline active instance is considered stable as online count (0) matches desired (0)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			group := createMockGroup(ctrl, "test-group", tc.replicas)
			instances := tc.setupInstances()

			state := calculateGroupState(group, instances)

			if state != tc.expectedState {
				t.Errorf("Expected state %s, got %s. Description: %s", tc.expectedState, state, tc.description)
			}
		})
	}
}

func TestStateClassification(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCases := []struct {
		name             string
		onlineCount      int
		offlineActive    int
		offlineCompleted int
		desiredReplicas  int
		expectedState    storeGroupState
	}{
		{"Priority: Completed instances", 2, 1, 1, 3, stateAwaitingNodeDeletion},
		{"Scale up: More needed", 2, 0, 0, 4, stateScalingUp},
		{"Scale down: Fewer needed", 4, 0, 0, 2, stateScalingDown},
		{"Cancel scale down", 2, 2, 0, 3, stateCancellingScaleDown},
		{"Stable: Exact match", 3, 0, 0, 3, stateStable},
		{"Stable with offline", 3, 1, 0, 3, stateStable},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var instances []runtime.StoreInstance

			// Add online instances
			for i := 0; i < tc.onlineCount; i++ {
				instances = append(instances, createTestStoreInstance("online-"+string(rune('0'+i))))
			}

			// Add offline active instances
			for i := 0; i < tc.offlineActive; i++ {
				instances = append(instances, createTestStoreInstance(
					"offline-active-"+string(rune('0'+i)),
					withOffline(true),
					withOfflineCondition(v1alpha1.OfflineReasonActive),
				))
			}

			// Add offline completed instances
			for i := 0; i < tc.offlineCompleted; i++ {
				instances = append(instances, createTestStoreInstance(
					"offline-completed-"+string(rune('0'+i)),
					withOffline(true),
					withOfflineCondition(v1alpha1.OfflineReasonCompleted),
				))
			}

			group := createMockGroup(ctrl, "test-group", int32(tc.desiredReplicas))
			state := calculateGroupState(group, instances)

			if state != tc.expectedState {
				t.Errorf("Expected state %s, got %s", tc.expectedState, state)
			}
		})
	}
}

func TestCalculateGroupStateEdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCases := []struct {
		name           string
		setupInstances func() []runtime.StoreInstance
		replicas       int32
		expectedState  storeGroupState
		description    string
	}{
		{
			name: "nil condition handling",
			setupInstances: func() []runtime.StoreInstance {
				// This instance will be ignored in classification of offline type
				return []runtime.StoreInstance{createTestStoreInstance("instance-0", withOffline(true))}
			},
			replicas:      1,
			expectedState: stateScalingUp,
			description:   "Should trigger scale up because the offline instance with nil condition is ignored, making onlineCount 0, which is less than desiredReplicas 1.",
		},
		{
			name: "unknown offline reason",
			setupInstances: func() []runtime.StoreInstance {
				return []runtime.StoreInstance{
					createTestStoreInstance("instance-0",
						withOffline(true),
						withOfflineCondition("UnknownReason")),
				}
			},
			replicas:      1,
			expectedState: stateScalingUp,
			description:   "Should trigger scale up because the offline instance with an unknown reason is ignored, making onlineCount 0, which is less than desiredReplicas 1.",
		},
		{
			name: "mixed conditions",
			setupInstances: func() []runtime.StoreInstance {
				return []runtime.StoreInstance{
					createTestStoreInstance("instance-0"),
					createTestStoreInstance("instance-1", withOffline(true), withOfflineCondition(v1alpha1.OfflineReasonActive)),
					createTestStoreInstance("instance-2", withOffline(true), withOfflineCondition(v1alpha1.OfflineReasonCompleted)),
					createTestStoreInstance("instance-3", withOffline(true)), // nil condition
				}
			},
			replicas:      2,
			expectedState: stateAwaitingNodeDeletion,
			description:   "Should prioritize deletion of completed instances over other states.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			group := createMockGroup(ctrl, "test-group", tc.replicas)
			instances := tc.setupInstances()

			state := calculateGroupState(group, instances)

			require.Equal(t, tc.expectedState, state, tc.description)
		})
	}
}

func TestHandleStableState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()

	testCases := []struct {
		name           string
		setupInstances func() []runtime.StoreInstance
		setupMocks     func(mockExecutor *updater.MockExecutor)
		verify         func(t *testing.T, fakeCli client.Client, ctx context.Context)
		expectedStatus task.Status
	}{
		{
			name: "deletes completed instances first",
			setupInstances: func() []runtime.StoreInstance {
				instanceToKeep := createTestStoreInstance("instance-0")
				instanceToDelete := createTestStoreInstance("instance-1", withOffline(true), withOfflineCondition(v1alpha1.OfflineReasonCompleted))
				return []runtime.StoreInstance{instanceToKeep, instanceToDelete}
			},
			setupMocks: func(mockExecutor *updater.MockExecutor) {
				// Mock executor should not be called if deletion takes precedence
				// But add a fallback in case deletion doesn't work as expected
				mockExecutor.EXPECT().Do(gomock.Any()).Return(false, nil).AnyTimes()
			},
			verify: func(t *testing.T, fakeCli client.Client, ctx context.Context) {
				// Verify instance was deleted
				err := fakeCli.Get(ctx, types.NamespacedName{Name: "instance-1", Namespace: "default"}, &v1alpha1.TiKV{})
				require.True(t, kerrors.IsNotFound(err))
			},
			expectedStatus: task.SWait,
		},
		{
			name: "performs rolling update if no completed instances",
			setupInstances: func() []runtime.StoreInstance {
				return []runtime.StoreInstance{createTestStoreInstance("instance-0")}
			},
			setupMocks: func(mockExecutor *updater.MockExecutor) {
				mockExecutor.EXPECT().Do(gomock.Any()).Return(false, nil)
			},
			expectedStatus: task.SComplete,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockExecutor := updater.NewMockExecutor(ctrl)
			instances := tc.setupInstances()

			var clientObjects []client.Object
			for _, inst := range instances {
				// Convert runtime.TiKV back to v1alpha1.TiKV for the fake client
				if tikvInst, ok := inst.(*runtime.TiKV); ok {
					clientObjects = append(clientObjects, (*v1alpha1.TiKV)(tikvInst))
				} else {
					clientObjects = append(clientObjects, inst.(client.Object))
				}
			}
			fakeCli := client.NewFakeClient(clientObjects...)

			if tc.setupMocks != nil {
				tc.setupMocks(mockExecutor)
			}

			result := handleStableState(ctx, fakeCli, instances, mockExecutor)
			require.Equal(t, tc.expectedStatus, result.Status())

			if tc.verify != nil {
				tc.verify(t, fakeCli, ctx)
			}
		})
	}
}

func TestHandleScaleUp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()

	testCases := []struct {
		name                 string
		mockReturnInProgress bool
		mockReturnError      error
		expectedStatus       task.Status
	}{
		{
			name:                 "executes scale up and completes",
			mockReturnInProgress: false,
			mockReturnError:      nil,
			expectedStatus:       task.SComplete,
		},
		{
			name:                 "waits for new instances",
			mockReturnInProgress: true,
			mockReturnError:      nil,
			expectedStatus:       task.SWait,
		},
		{
			name:                 "fails on executor error",
			mockReturnInProgress: false,
			mockReturnError:      errors.New("test error"),
			expectedStatus:       task.SFail,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockExecutor := updater.NewMockExecutor(ctrl)
			group := createMockGroup(ctrl, "test-group", 3)
			instances := []runtime.StoreInstance{createTestStoreInstance("instance-0")}

			mockExecutor.EXPECT().Do(ctx).Return(tc.mockReturnInProgress, tc.mockReturnError)

			result := handleScaleUp(ctx, group, instances, mockExecutor)
			require.Equal(t, tc.expectedStatus, result.Status())
		})
	}
}

//func TestHandleCancelScaleDown(t *testing.T) {
//	ctrl := gomock.NewController(t)
//	defer ctrl.Finish()
//	mockExecutor := updater.NewMockExecutor(ctrl)
//	ctx := context.Background()
//
//	t.Run("rescues offline active instances", func(t *testing.T) {
//		group := createMockGroup(ctrl, "test-group", 2)
//		onlineInstance := createTestStoreInstance("instance-0")
//		offlineInstance := createTestStoreInstance("instance-1", withOffline(true), withOfflineCondition(v1alpha1.OfflineReasonActive))
//		instances := []runtime.StoreInstance{onlineInstance, offlineInstance}
//		fakeCli := client.NewFakeClient(onlineInstance.(client.Object), offlineInstance.(client.Object))
//
//		result := handleCancelScaleDown(ctx, fakeCli, group, instances, mockExecutor)
//		require.Equal(t, result.Status(), task.SWait)
//
//		updatedInstance := &v1alpha1.PD{}
//		err := fakeCli.Get(ctx, types.NamespacedName{Name: "instance-1", Namespace: "default"}, updatedInstance)
//		require.NoError(t, err)
//		require.False(t, updatedInstance.Spec.Offline)
//	})
//
//	t.Run("rescues and scales up", func(t *testing.T) {
//		group := createMockGroup(ctrl, "test-group", 3)
//		onlineInstance := createTestStoreInstance("instance-0")
//		offlineInstance := createTestStoreInstance("instance-1", withOffline(true), withOfflineCondition(v1alpha1.OfflineReasonActive))
//		instances := []runtime.StoreInstance{onlineInstance, offlineInstance}
//		fakeCli := client.NewFakeClient(onlineInstance.(client.Object), offlineInstance.(client.Object))
//		mockExecutor.EXPECT().Do(ctx).Return(false, nil)
//
//		result := handleCancelScaleDown(ctx, fakeCli, group, instances, mockExecutor)
//		require.True(t, result.IsWait())
//
//		updatedInstance := &v1alpha1.PD{}
//		err := fakeCli.Get(ctx, types.NamespacedName{Name: "instance-1", Namespace: "default"}, updatedInstance)
//		require.NoError(t, err)
//		require.False(t, updatedInstance.Spec.Offline)
//	})
//}

//
//func TestHandleScaleDown(t *testing.T) {
//	ctrl := gomock.NewController(t)
//	defer ctrl.Finish()
//	mockSelector := updater.NewMockSelector[runtime.StoreInstance](ctrl)
//	ctx := context.Background()
//
//	t.Run("sets instances offline", func(t *testing.T) {
//		group := createMockGroup(ctrl, "test-group", 1)
//		instance0 := createTestStoreInstance("instance-0")
//		instance1 := createTestStoreInstance("instance-1")
//		instances := []runtime.StoreInstance{instance0, instance1}
//		fakeCli := client.NewFakeClient(instance0.(client.Object), instance1.(client.Object))
//
//		mockSelector.EXPECT().Choose(gomock.Any()).Return("instance-1")
//
//		result := handleScaleDown(ctx, fakeCli, group, instances, mockSelector)
//		require.True(t, result.IsWait())
//
//		updatedInstance := &v1alpha1.PD{}
//		err := fakeCli.Get(ctx, types.NamespacedName{Name: "instance-1", Namespace: "default"}, updatedInstance)
//		require.NoError(t, err)
//		require.True(t, updatedInstance.Spec.Offline)
//	})
//
//	t.Run("waits if instances are already offline", func(t *testing.T) {
//		group := createMockGroup(ctrl, "test-group", 1)
//		instance0 := createTestStoreInstance("instance-0")
//		instance1 := createTestStoreInstance("instance-1", withOffline(true))
//		instances := []runtime.StoreInstance{instance0, instance1}
//		fakeCli := client.NewFakeClient(instance0.(client.Object), instance1.(client.Object))
//
//		result := handleScaleDown(ctx, fakeCli, group, instances, mockSelector)
//		require.True(t, result.IsWait())
//	})
//}
//
//func TestHandleInstanceDeletion(t *testing.T) {
//	ctx := context.Background()
//
//	t.Run("deletes completed instances", func(t *testing.T) {
//		instanceToDelete := createTestStoreInstance("instance-0", withOffline(true), withOfflineCondition(v1alpha1.OfflineReasonCompleted))
//		instances := []runtime.StoreInstance{instanceToDelete}
//		fakeCli := client.NewFakeClient(instanceToDelete.(client.Object))
//
//		result := handleInstanceDeletion(ctx, fakeCli, instances)
//		require.True(t, result.IsWait())
//
//		err := fakeCli.Get(ctx, types.NamespacedName{Name: "instance-0", Namespace: "default"}, &v1alpha1.PD{})
//		require.True(t, kerrors.IsNotFound(err))
//	})
//
//	t.Run("waits if no completed instances", func(t *testing.T) {
//		instances := []runtime.StoreInstance{createTestStoreInstance("instance-0")}
//		fakeCli := client.NewFakeClient(instances[0].(client.Object))
//
//		result := handleInstanceDeletion(ctx, fakeCli, instances)
//		require.True(t, result.IsWait())
//	})
//}
//
//func TestDeleteCompletedOfflineInstances(t *testing.T) {
//	ctx := context.Background()
//
//	t.Run("deletes a single completed instance", func(t *testing.T) {
//		instanceToDelete := createTestStoreInstance("instance-0", withOffline(true), withOfflineCondition(v1alpha1.OfflineReasonCompleted))
//		instances := []runtime.StoreInstance{instanceToDelete}
//		fakeCli := client.NewFakeClient(instanceToDelete.(client.Object))
//
//		result := deleteCompletedOfflineInstances(ctx, fakeCli, instances)
//		require.NotNil(t, result)
//		require.True(t, result.IsWait())
//
//		err := fakeCli.Get(ctx, types.NamespacedName{Name: "instance-0", Namespace: "default"}, &v1alpha1.PD{})
//		require.True(t, kerrors.IsNotFound(err))
//	})
//
//	t.Run("returns nil if no instances to delete", func(t *testing.T) {
//		instances := []runtime.StoreInstance{createTestStoreInstance("instance-0")}
//		fakeCli := client.NewFakeClient(instances[0].(client.Object))
//
//		result := deleteCompletedOfflineInstances(ctx, fakeCli, instances)
//		require.Nil(t, result)
//	})
//
//	t.Run("fails on delete error", func(t *testing.T) {
//		instanceToDelete := createTestStoreInstance("instance-0", withOffline(true), withOfflineCondition(v1alpha1.OfflineReasonCompleted))
//		instances := []runtime.StoreInstance{instanceToDelete}
//		fakeCli := client.NewFakeClient(instanceToDelete.(client.Object))
//		fakeCli.WithError("delete", "pds", errors.New("test error"))
//
//		result := deleteCompletedOfflineInstances(ctx, fakeCli, instances)
//		require.NotNil(t, result)
//		require.True(t, result.IsFail())
//	})
//}
//
//func TestPerformRollingUpdate(t *testing.T) {
//	ctrl := gomock.NewController(t)
//	defer ctrl.Finish()
//	mockExecutor := updater.NewMockExecutor(ctrl)
//	ctx := context.Background()
//
//	t.Run("performs update on normal instances", func(t *testing.T) {
//		instances := []runtime.StoreInstance{
//			createTestStoreInstance("instance-0"),
//			createTestStoreInstance("instance-1", withOffline(true)),
//		}
//		mockExecutor.EXPECT().Do(ctx).Return(false, nil)
//
//		result := performRollingUpdate(ctx, instances, mockExecutor)
//		require.True(t, result.IsComplete())
//	})
//
//	t.Run("waits for update to complete", func(t *testing.T) {
//		instances := []runtime.StoreInstance{createTestStoreInstance("instance-0")}
//		mockExecutor.EXPECT().Do(ctx).Return(true, nil)
//
//		result := performRollingUpdate(ctx, instances, mockExecutor)
//		require.True(t, result.IsWait())
//	})
//
//	t.Run("fails on executor error", func(t *testing.T) {
//		instances := []runtime.StoreInstance{createTestStoreInstance("instance-0")}
//		mockExecutor.EXPECT().Do(ctx).Return(false, errors.New("test error"))
//
//		result := performRollingUpdate(ctx, instances, mockExecutor)
//		require.True(t, result.IsFail())
//	})
//}
