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
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

func conditionsFromSingle(condition *metav1.Condition) []metav1.Condition {
	if condition == nil {
		return nil
	}
	return []metav1.Condition{*condition}
}

func createMockInstance(ctrl *gomock.Controller, offline bool, condition []metav1.Condition, annotations map[string]string) *runtime.MockInstance {
	mockStore := runtime.NewMockInstance(ctrl)
	mockStore.EXPECT().GetName().Return("test").AnyTimes()
	mockStore.EXPECT().GetGeneration().AnyTimes()
	mockStore.EXPECT().IsOffline().Return(offline).AnyTimes()

	// Create mutable state for conditions
	currentConditions := condition
	mockStore.EXPECT().Conditions().DoAndReturn(func() []metav1.Condition {
		return currentConditions
	}).AnyTimes()
	mockStore.EXPECT().SetConditions(gomock.Any()).DoAndReturn(func(conditions []metav1.Condition) {
		currentConditions = conditions
	}).AnyTimes()

	if annotations == nil {
		annotations = make(map[string]string)
	}
	// Create mutable state for annotations
	currentAnnotations := annotations
	mockStore.EXPECT().GetAnnotations().DoAndReturn(func() map[string]string {
		return currentAnnotations
	}).AnyTimes()
	mockStore.EXPECT().SetAnnotations(gomock.Any()).DoAndReturn(func(annots map[string]string) {
		currentAnnotations = annots
	}).AnyTimes()
	return mockStore
}

type contextBuilder struct {
	ctx         *MockStoreOfflineReconcileContext
	storeID     string
	storeExists bool
	storeState  string
	pdClient    pd.PDClient
}

func newContext(storeID string) *contextBuilder {
	return &contextBuilder{
		storeID:     storeID,
		storeExists: true,
	}
}

func (b *contextBuilder) StoreExists(exists bool) *contextBuilder {
	b.storeExists = exists
	return b
}

func (b *contextBuilder) WithState(state string) *contextBuilder {
	b.storeState = state
	return b
}

func (b *contextBuilder) WithPDClient(cli pd.PDClient) *contextBuilder {
	b.pdClient = cli
	return b
}

func (b *contextBuilder) Build() *MockStoreOfflineReconcileContext {
	return b.ctx
}

func (b *contextBuilder) initMock(ctrl *gomock.Controller) {
	b.ctx = NewMockStoreOfflineReconcileContext(ctrl)
	b.ctx.EXPECT().GetStoreID().Return(b.storeID).AnyTimes()
	b.ctx.EXPECT().StoreNotExists().Return(!b.storeExists).AnyTimes()
	b.ctx.EXPECT().GetLeaderCount().Return(0).AnyTimes()
	b.ctx.EXPECT().GetRegionCount().Return(0).AnyTimes()
	b.ctx.EXPECT().IsStoreUp().Return(true).AnyTimes()
	b.ctx.EXPECT().IsStoreBusy().Return(false).AnyTimes()
	b.ctx.EXPECT().SetStatusChanged().AnyTimes()

	// Always set up GetStoreState to return the stored state or empty string
	b.ctx.EXPECT().GetStoreState().Return(b.storeState).AnyTimes()

	if b.pdClient != nil {
		b.ctx.EXPECT().GetPDClient().Return(b.pdClient).AnyTimes()
	}
}

func TestTaskOfflineStoreStateMachine(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name               string
		instanceBuilder    func(ctrl *gomock.Controller) *runtime.MockInstance
		contextBuilder     *contextBuilder
		setupMock          func(mockUnderlay *pdapi.MockPDClient)
		expectedResult     task.Status
		expectedCondition  *metav1.Condition
		assertFunc         func(t *testing.T, instance *runtime.MockInstance)
		skipMockClientInit bool // Flag to explicitly skip mock client initialization
	}{
		// Happy path
		{
			name: "Start offline operation: no condition exists, successful transition to Processing",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, true, nil, nil)
			},
			contextBuilder: newContext("1"),
			setupMock: func(mockUnderlay *pdapi.MockPDClient) {
				mockUnderlay.EXPECT().DeleteStore(gomock.Any(), "1").Return(nil)
			},
			expectedResult:    task.SRetry,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonOfflineProcessing},
		},
		{
			name: "Start offline operation: empty store ID causes failure",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, true, nil, nil)
			},
			contextBuilder:    newContext(""),
			expectedResult:    task.SRetry,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonOfflineFailed},
		},
		{
			name: "Processing state: store still removing",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, true, conditionsFromSingle(newOfflinedCondition(v1alpha1.ReasonOfflineProcessing, "", metav1.ConditionFalse)), nil)
			},
			contextBuilder:    newContext("1").WithState(v1alpha1.StoreStateRemoving),
			expectedResult:    task.SRetry,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonOfflineProcessing},
		},
		{
			name: "Processing state: store removed, transition to Completed",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, true, conditionsFromSingle(newOfflinedCondition(v1alpha1.ReasonOfflineProcessing, "", metav1.ConditionFalse)), nil)
			},
			contextBuilder:    newContext("1").WithState(v1alpha1.StoreStateRemoved),
			expectedResult:    task.SComplete,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.ReasonOfflineCompleted},
		},
		{
			name: "Completed state should return complete",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, true, conditionsFromSingle(newOfflinedCondition(v1alpha1.ReasonOfflineCompleted, "", metav1.ConditionTrue)), nil)
			},
			contextBuilder:    newContext(""),
			expectedResult:    task.SComplete,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.ReasonOfflineCompleted},
		},
		// Failure and retry
		{
			name: "Failed state: retry operation with valid store ID",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				// Create a failed condition, retry should succeed with valid store ID
				condition := newOfflinedCondition(v1alpha1.ReasonOfflineFailed, "Failed to call PD DeleteStore API", metav1.ConditionFalse)
				return createMockInstance(ctrl, true, conditionsFromSingle(condition), nil)
			},
			contextBuilder: newContext("1"),
			setupMock: func(mockUnderlay *pdapi.MockPDClient) {
				mockUnderlay.EXPECT().DeleteStore(gomock.Any(), "1").Return(nil)
			},
			expectedResult:    task.SRetry,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonOfflineProcessing},
		},
		{
			name: "Failed state: retry with empty store ID remains failed",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				// With empty store ID, retry should remain in Failed state
				condition := newOfflinedCondition(v1alpha1.ReasonOfflineFailed, "Failed to call PD DeleteStore API", metav1.ConditionFalse)
				return createMockInstance(ctrl, true, conditionsFromSingle(condition), nil)
			},
			contextBuilder:    newContext(""),
			expectedResult:    task.SRetry,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonOfflineFailed},
		},
		// Cancellation
		{
			name: "Cancel operation: in Processing state",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, false, conditionsFromSingle(newOfflinedCondition(v1alpha1.ReasonOfflineProcessing, "", metav1.ConditionFalse)), nil)
			},
			contextBuilder: newContext("3").WithState(v1alpha1.StoreStateRemoving),
			setupMock: func(mockUnderlay *pdapi.MockPDClient) {
				mockUnderlay.EXPECT().CancelDeleteStore(gomock.Any(), "3").Return(nil)
			},
			expectedResult:    task.SComplete,
			expectedCondition: nil, // condition should be removed after cancellation
		},
		{
			name: "Cancel operation: PD API fails",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, false, conditionsFromSingle(newOfflinedCondition(v1alpha1.ReasonOfflineProcessing, "", metav1.ConditionFalse)), nil)
			},
			contextBuilder: newContext("1").WithState(v1alpha1.StoreStateRemoving),
			setupMock: func(mockUnderlay *pdapi.MockPDClient) {
				mockUnderlay.EXPECT().CancelDeleteStore(gomock.Any(), "1").Return(errors.New("cancel api error"))
			},
			expectedResult:    task.SFail,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonOfflineProcessing},
		},
		{
			name: "Cancel operation: retry should be cleared",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, false, conditionsFromSingle(newOfflinedCondition(v1alpha1.ReasonOfflineFailed, "", metav1.ConditionFalse)), nil)
			},
			contextBuilder:    newContext("1"),
			expectedResult:    task.SComplete,
			expectedCondition: nil, // condition should be removed after cancellation
		},
		{
			name: "Restart offline operation with empty store ID after cancellation",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, true, nil, nil) // no condition after cancellation
			},
			contextBuilder:    newContext(""),
			expectedResult:    task.SRetry,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonOfflineFailed},
		},
		// State recovery from PD when condition is lost
		{
			name: "State recovery: PD shows Serving, condition lost, should start fresh",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, true, nil, nil)
			},
			contextBuilder: newContext("1").WithState(v1alpha1.StoreStateServing),
			setupMock: func(mockUnderlay *pdapi.MockPDClient) {
				mockUnderlay.EXPECT().DeleteStore(gomock.Any(), "1").Return(nil)
			},
			expectedResult:    task.SRetry,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonOfflineProcessing},
		},
		{
			name: "State recovery: PD shows Removing, condition lost, should recover as Processing",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, true, nil, nil)
			},
			contextBuilder:    newContext("1").WithState(v1alpha1.StoreStateRemoving),
			expectedResult:    task.SRetry,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonOfflineProcessing},
		},
		{
			name: "State recovery: PD shows Removed, condition lost, should recover as Completed",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, true, nil, nil)
			},
			contextBuilder:    newContext("1").WithState(v1alpha1.StoreStateRemoved),
			expectedResult:    task.SComplete,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.ReasonOfflineCompleted},
		},
		{
			name: "State recovery: Store not exists, condition lost, should recover as Completed",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, true, nil, nil)
			},
			contextBuilder:    newContext("1").StoreExists(false),
			expectedResult:    task.SComplete,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.ReasonOfflineCompleted},
		},
		{
			name: "State recovery: PD shows unknown state, condition lost, should start fresh",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, true, nil, nil)
			},
			contextBuilder: newContext("1").WithState("UnknownState"),
			setupMock: func(mockUnderlay *pdapi.MockPDClient) {
				mockUnderlay.EXPECT().DeleteStore(gomock.Any(), "1").Return(nil)
			},
			expectedResult:    task.SRetry,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonOfflineProcessing},
		},
		// Edge cases
		{
			name: "Nil PD client in cancel state",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, false, conditionsFromSingle(newOfflinedCondition(v1alpha1.ReasonOfflineProcessing, "", metav1.ConditionFalse)), nil)
			},
			contextBuilder:     newContext("1").WithState(v1alpha1.StoreStateRemoving).WithPDClient(nil),
			expectedResult:     task.SFail,
			expectedCondition:  &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonOfflineProcessing},
			skipMockClientInit: true,
		},
		{
			name: "Store doesn't exist in Processing state",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, true, conditionsFromSingle(newOfflinedCondition(v1alpha1.ReasonOfflineProcessing, "", metav1.ConditionFalse)), nil)
			},
			contextBuilder:    newContext("").StoreExists(false),
			expectedResult:    task.SComplete,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.ReasonOfflineCompleted},
		},
		{
			name: "Unknown condition reason",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, true, conditionsFromSingle(newOfflinedCondition("UnknownReason", "", metav1.ConditionTrue)), nil)
			},
			contextBuilder:    newContext(""),
			expectedResult:    task.SFail,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionTrue, Reason: "UnknownReason"},
		},
		{
			name: "Cancel when no condition exists: store not removing",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, false, nil, nil)
			},
			contextBuilder: newContext("1").WithState(v1alpha1.StoreStateServing),
			setupMock: func(mockUnderlay *pdapi.MockPDClient) {
				// Should still call CancelDeleteStore for Serving state
				mockUnderlay.EXPECT().CancelDeleteStore(gomock.Any(), "1").Return(nil)
			},
			expectedResult:    task.SComplete,
			expectedCondition: nil, // condition should be removed after cancellation
		},
		{
			name: "Cancel when no condition exists: store is removing",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, false, nil, nil)
			},
			contextBuilder: newContext("1").WithState(v1alpha1.StoreStateRemoving),
			setupMock: func(mockUnderlay *pdapi.MockPDClient) {
				mockUnderlay.EXPECT().CancelDeleteStore(gomock.Any(), "1").Return(nil)
			},
			expectedResult:    task.SComplete,
			expectedCondition: nil, // condition should be removed after cancellation
		},
		{
			name: "Cancel when no condition exists: cancel API fails but no original condition to preserve",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, false, nil, nil)
			},
			contextBuilder: newContext("1").WithState(v1alpha1.StoreStateRemoving),
			setupMock: func(mockUnderlay *pdapi.MockPDClient) {
				mockUnderlay.EXPECT().CancelDeleteStore(gomock.Any(), "1").Return(errors.New("cancel failed"))
			},
			expectedResult:    task.SFail,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonOfflineProcessing},
		},
		{
			name: "Cancel when already completed",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, false, conditionsFromSingle(newOfflinedCondition(v1alpha1.ReasonOfflineCompleted, "", metav1.ConditionTrue)), nil)
			},
			contextBuilder:    newContext(""),
			expectedResult:    task.SComplete,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.ReasonOfflineCompleted},
		},
		{
			name: "Failed state with empty store ID remains failed",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, true, conditionsFromSingle(newOfflinedCondition(v1alpha1.ReasonOfflineFailed, "", metav1.ConditionFalse)), nil)
			},
			contextBuilder:    newContext(""),
			expectedResult:    task.SRetry,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonOfflineFailed},
		},
		{
			name: "Cancel operation: clears retry but keeps other annotations",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, false, conditionsFromSingle(newOfflinedCondition(v1alpha1.ReasonOfflineFailed, "", metav1.ConditionFalse)), map[string]string{"other": "value"})
			},
			contextBuilder:    newContext("1"),
			expectedResult:    task.SComplete,
			expectedCondition: nil, // condition should be removed after cancellation
			assertFunc: func(t *testing.T, instance *runtime.MockInstance) {
				require.Contains(t, instance.GetAnnotations(), "other", "Other annotations should remain")
				require.Equal(t, "value", instance.GetAnnotations()["other"], "Other annotation value should be preserved")
			},
		},
		{
			name: "Cancel operation: empty store ID",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, false, conditionsFromSingle(newOfflinedCondition(v1alpha1.ReasonOfflineProcessing, "", metav1.ConditionFalse)), nil)
			},
			contextBuilder:     newContext("").WithState(v1alpha1.StoreStateRemoving),
			expectedResult:     task.SFail,
			expectedCondition:  &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonOfflineProcessing},
			skipMockClientInit: true,
		},
		{
			name: "Cancel operation: whitespace store ID",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, false, conditionsFromSingle(newOfflinedCondition(v1alpha1.ReasonOfflineProcessing, "", metav1.ConditionFalse)), nil)
			},
			contextBuilder:     newContext("   ").WithState(v1alpha1.StoreStateRemoving),
			expectedResult:     task.SFail,
			expectedCondition:  &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.ReasonOfflineProcessing},
			skipMockClientInit: true,
		},
		{
			name: "Cancel operation: store not in removing state",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, false, conditionsFromSingle(newOfflinedCondition(v1alpha1.ReasonOfflineProcessing, "", metav1.ConditionFalse)), nil)
			},
			setupMock: func(mockUnderlay *pdapi.MockPDClient) {
				mockUnderlay.EXPECT().CancelDeleteStore(gomock.Any(), "1").Return(nil)
			},
			contextBuilder:    newContext("1").WithState(v1alpha1.StoreStateServing),
			expectedResult:    task.SComplete,
			expectedCondition: nil, // condition should be removed after cancellation
		},
		{
			name: "Cancel operation: store doesn't exist",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, false, conditionsFromSingle(newOfflinedCondition(v1alpha1.ReasonOfflineProcessing, "", metav1.ConditionFalse)), nil)
			},
			contextBuilder:    newContext("").StoreExists(false),
			expectedResult:    task.SComplete,
			expectedCondition: nil,
		},
		{
			name: "Cancel unknown condition reason",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, false, conditionsFromSingle(newOfflinedCondition("UnknownReason", "", metav1.ConditionTrue)), nil)
			},
			contextBuilder:    newContext(""),
			expectedResult:    task.SFail,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflinedConditionType, Status: metav1.ConditionTrue, Reason: "UnknownReason"},
		},
		{
			name: "new instance without PD registration should not get offline condition",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, false, nil, nil)
			},
			contextBuilder:    newContext("").StoreExists(false),
			expectedCondition: nil,
		},
		{
			name: "instance with previous offline condition and no PD registration",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, false, conditionsFromSingle(newOfflinedCondition(v1alpha1.ReasonOfflineProcessing, "Previous offline operation", metav1.ConditionFalse)), nil)
			},
			contextBuilder:    newContext("").StoreExists(false),
			expectedCondition: nil,
		},
		{
			name: "instance with PD registration should proceed normally",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockInstance {
				return createMockInstance(ctrl, false, nil, nil)
			},
			contextBuilder:    newContext("store-123"),
			expectedCondition: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var state StoreOfflineReconcileContext

			// Create instance
			instance := tt.instanceBuilder(ctrl)

			// Initialize the mock context unless explicitly skipped
			if !tt.skipMockClientInit {
				tt.contextBuilder.initMock(ctrl)

				if tt.contextBuilder.pdClient == nil {
					mockPDClient := pd.NewMockPDClient(ctrl)
					mockUnderlayClient := pdapi.NewMockPDClient(ctrl)
					mockPDClient.EXPECT().Underlay().Return(mockUnderlayClient).AnyTimes()
					tt.contextBuilder.pdClient = mockPDClient
					tt.contextBuilder.ctx.EXPECT().GetPDClient().Return(mockPDClient).AnyTimes()

					if tt.setupMock != nil {
						tt.setupMock(mockUnderlayClient)
					}
				}
				state = tt.contextBuilder.Build()
			} else {
				// For skipMockClientInit tests, still initialize mock but with nil PD client
				tt.contextBuilder.initMock(ctrl)
				tt.contextBuilder.ctx.EXPECT().GetPDClient().Return(nil).AnyTimes()
				state = tt.contextBuilder.Build()
			}

			result := TaskOfflineStoreStateMachine(ctx, state, instance)

			// Default assertions
			require.Equal(t, tt.expectedResult.String(), result.Status().String(), "Status mismatch")

			actualCondition := runtime.GetOfflineCondition(instance)
			if tt.expectedCondition == nil {
				require.Nil(t, actualCondition, "Expected no condition but got one")
			} else {
				require.NotNil(t, actualCondition, "Expected a condition but got nil")
				require.Equal(t, tt.expectedCondition.Type, actualCondition.Type, "Condition type mismatch")
				require.Equal(t, tt.expectedCondition.Status, actualCondition.Status, "Condition status mismatch")
				require.Equal(t, tt.expectedCondition.Reason, actualCondition.Reason, "Condition reason mismatch")
				// Skip time and message comparison for most tests as they are dynamic
			}

			if tt.assertFunc != nil {
				tt.assertFunc(t, instance)
			}
		})
	}
}
