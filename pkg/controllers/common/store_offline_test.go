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
	"strconv"
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

func createMockStoreInstance(ctrl *gomock.Controller, offline bool, condition []metav1.Condition, annotations map[string]string) *runtime.MockStoreInstance {
	mockStore := runtime.NewMockStoreInstance(ctrl)
	mockStore.EXPECT().GetName().Return("test").AnyTimes()
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
		instanceBuilder    func(ctrl *gomock.Controller) *runtime.MockStoreInstance
		contextBuilder     *contextBuilder
		setupMock          func(mockUnderlay *pdapi.MockPDClient)
		expectedResult     task.Status
		expectedCondition  *metav1.Condition
		expectedRetryCount string
		assertFunc         func(t *testing.T, instance *runtime.MockStoreInstance)
		skipMockClientInit bool // Flag to explicitly skip mock client initialization
	}{
		// Happy path
		{
			name: "Start offline operation: no condition exists",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, true, nil, nil)
			},
			contextBuilder:    newContext(""),
			expectedResult:    task.SRetry,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonPending},
		},
		{
			name: "Pending state: transition to Active",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, true, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonPending, "", metav1.ConditionTrue)), nil)
			},
			contextBuilder: newContext("1"),
			setupMock: func(mockUnderlay *pdapi.MockPDClient) {
				mockUnderlay.EXPECT().DeleteStore(gomock.Any(), "1").Return(nil)
			},
			expectedResult:    task.SRetry,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonActive},
		},
		{
			name: "Active state: store still removing",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, true, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonActive, "", metav1.ConditionTrue)), nil)
			},
			contextBuilder:    newContext("1").WithState(v1alpha1.StoreStateRemoving),
			expectedResult:    task.SRetry,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonActive},
		},
		{
			name: "Active state: store removed, transition to Completed",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, true, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonActive, "", metav1.ConditionTrue)), nil)
			},
			contextBuilder:    newContext("1").WithState(v1alpha1.StoreStateRemoved),
			expectedResult:    task.SComplete,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonCompleted},
		},
		{
			name: "Completed state should return complete",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, true, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonCompleted, "", metav1.ConditionTrue)), nil)
			},
			contextBuilder:    newContext(""),
			expectedResult:    task.SComplete,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonCompleted},
		},
		// Failure and retry
		{
			name: "Pending state: PD API call fails, transition to Failed",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, true, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonPending, "", metav1.ConditionTrue)), nil)
			},
			contextBuilder: newContext("2"),
			setupMock: func(mockUnderlay *pdapi.MockPDClient) {
				mockUnderlay.EXPECT().DeleteStore(gomock.Any(), "2").Return(errors.New("transient pd api error"))
			},
			expectedResult:    task.SRetry,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonFailed},
		},
		{
			name: "Pending state: empty store ID",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, true, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonPending, "", metav1.ConditionTrue)), nil)
			},
			contextBuilder:    newContext(""),
			expectedResult:    task.SFail,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonPending},
		},
		{
			name: "Failed state: retry operation",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, true, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonFailed, "", metav1.ConditionTrue)), map[string]string{AnnotationKeyOfflineRetryCount: "1"})
			},
			contextBuilder:     newContext(""),
			expectedResult:     task.SRetry,
			expectedCondition:  &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonPending},
			expectedRetryCount: "2",
		},
		{
			name: "Failed state: max retries reached",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, true, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonFailed, "", metav1.ConditionTrue)), map[string]string{AnnotationKeyOfflineRetryCount: strconv.Itoa(MaxRetryCount)})
			},
			contextBuilder:     newContext(""),
			expectedResult:     task.SFail,
			expectedCondition:  &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonFailed},
			expectedRetryCount: "",
		},
		// Cancellation
		{
			name: "Cancel operation: in Active state",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, false, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonActive, "", metav1.ConditionTrue)), nil)
			},
			contextBuilder: newContext("3").WithState(v1alpha1.StoreStateRemoving),
			setupMock: func(mockUnderlay *pdapi.MockPDClient) {
				mockUnderlay.EXPECT().CancelDeleteStore(gomock.Any(), "3").Return(nil)
			},
			expectedResult:    task.SComplete,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.OfflineReasonCancelled},
		},
		{
			name: "Cancel operation: PD API fails",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, false, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonActive, "", metav1.ConditionTrue)), nil)
			},
			contextBuilder: newContext("1").WithState(v1alpha1.StoreStateRemoving),
			setupMock: func(mockUnderlay *pdapi.MockPDClient) {
				mockUnderlay.EXPECT().CancelDeleteStore(gomock.Any(), "1").Return(errors.New("cancel api error"))
			},
			expectedResult:    task.SFail,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonActive},
		},
		{
			name: "Cancel operation: retry annotation should be cleared",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, false, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonFailed, "", metav1.ConditionTrue)), map[string]string{AnnotationKeyOfflineRetryCount: "2"})
			},
			contextBuilder:     newContext("1"),
			expectedResult:     task.SComplete,
			expectedCondition:  &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.OfflineReasonCancelled},
			expectedRetryCount: "",
		},
		{
			name: "Restart a canceled operation",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, true, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonCancelled, "", metav1.ConditionFalse)), nil)
			},
			contextBuilder:    newContext(""),
			expectedResult:    task.SRetry,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonPending},
		},
		// Edge cases
		{
			name: "Nil PD client in pending state",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, true, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonPending, "", metav1.ConditionTrue)), nil)
			},
			contextBuilder:     newContext("1").WithPDClient(nil),
			expectedResult:     task.SFail,
			expectedCondition:  &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonPending},
			skipMockClientInit: true,
		},
		{
			name: "Nil PD client in cancel state",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, false, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonActive, "", metav1.ConditionTrue)), nil)
			},
			contextBuilder:     newContext("1").WithState(v1alpha1.StoreStateRemoving).WithPDClient(nil),
			expectedResult:     task.SFail,
			expectedCondition:  &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonActive},
			skipMockClientInit: true,
		},
		{
			name: "Store doesn't exist in pending state",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, true, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonPending, "", metav1.ConditionTrue)), nil)
			},
			contextBuilder:    newContext("").StoreExists(false),
			expectedResult:    task.SComplete,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonCompleted},
		},
		{
			name: "Store doesn't exist in active state",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, true, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonActive, "", metav1.ConditionTrue)), nil)
			},
			contextBuilder:    newContext("").StoreExists(false),
			expectedResult:    task.SComplete,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonCompleted},
		},
		{
			name: "Unknown condition reason",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, true, conditionsFromSingle(newOfflineCondition("UnknownReason", "", metav1.ConditionTrue)), nil)
			},
			contextBuilder:    newContext(""),
			expectedResult:    task.SFail,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: "UnknownReason"},
		},
		{
			name: "Cancel when no condition exists",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, false, nil, nil)
			},
			contextBuilder: newContext(""),
			expectedResult: task.SComplete,
		},
		{
			name: "Cancel when already completed",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, false, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonCompleted, "", metav1.ConditionTrue)), nil)
			},
			contextBuilder:    newContext(""),
			expectedResult:    task.SComplete,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonCompleted},
		},
		{
			name: "Cancel when already canceled",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, false, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonCancelled, "", metav1.ConditionFalse)), nil)
			},
			contextBuilder:    newContext(""),
			expectedResult:    task.SComplete,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.OfflineReasonCancelled},
		},
		{
			name: "Failed state with nil annotations",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				// Use the helper function for consistency
				return createMockStoreInstance(ctrl, true, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonFailed, "", metav1.ConditionTrue)), nil)
			},
			contextBuilder:     newContext(""),
			expectedResult:     task.SRetry,
			expectedCondition:  &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonPending},
			expectedRetryCount: "1",
		},
		{
			name: "Failed state with invalid retry count annotation",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, true, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonFailed, "", metav1.ConditionTrue)), map[string]string{AnnotationKeyOfflineRetryCount: "invalid"})
			},
			contextBuilder:     newContext(""),
			expectedResult:     task.SRetry,
			expectedCondition:  &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonPending},
			expectedRetryCount: "1",
		},
		{
			name: "Cancel operation clears retry annotation but keeps others",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, false, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonFailed, "", metav1.ConditionTrue)), map[string]string{AnnotationKeyOfflineRetryCount: "2", "other": "value"})
			},
			contextBuilder:    newContext("1"),
			expectedResult:    task.SComplete,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.OfflineReasonCancelled},
			assertFunc: func(t *testing.T, instance *runtime.MockStoreInstance) {
				require.Contains(t, instance.GetAnnotations(), "other", "Other annotations should remain")
				require.Equal(t, "value", instance.GetAnnotations()["other"], "Other annotation value should be preserved")
			},
		},
		{
			name: "Cancel operation: empty store ID",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, false, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonActive, "", metav1.ConditionTrue)), nil)
			},
			contextBuilder:     newContext("").WithState(v1alpha1.StoreStateRemoving),
			expectedResult:     task.SFail,
			expectedCondition:  &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonActive},
			skipMockClientInit: true,
		},
		{
			name: "Cancel operation: whitespace store ID",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, false, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonActive, "", metav1.ConditionTrue)), nil)
			},
			contextBuilder:     newContext("   ").WithState(v1alpha1.StoreStateRemoving),
			expectedResult:     task.SFail,
			expectedCondition:  &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonActive},
			skipMockClientInit: true,
		},
		{
			name: "Cancel operation: store not in removing state",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, false, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonActive, "", metav1.ConditionTrue)), nil)
			},
			setupMock: func(mockUnderlay *pdapi.MockPDClient) {
				mockUnderlay.EXPECT().CancelDeleteStore(gomock.Any(), "1").Return(nil)
			},
			contextBuilder:    newContext("1").WithState(v1alpha1.StoreStateServing),
			expectedResult:    task.SComplete,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.OfflineReasonCancelled},
		},
		{
			name: "Cancel operation: store doesn't exist",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, false, conditionsFromSingle(newOfflineCondition(v1alpha1.OfflineReasonActive, "", metav1.ConditionTrue)), nil)
			},
			contextBuilder:    newContext("").StoreExists(false),
			expectedResult:    task.SComplete,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonCompleted},
		},
		{
			name: "Cancel unknown condition reason",
			instanceBuilder: func(ctrl *gomock.Controller) *runtime.MockStoreInstance {
				return createMockStoreInstance(ctrl, false, conditionsFromSingle(newOfflineCondition("UnknownReason", "", metav1.ConditionTrue)), nil)
			},
			contextBuilder:    newContext(""),
			expectedResult:    task.SFail,
			expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: "UnknownReason"},
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

			result := TaskOfflineStoreStateMachine(ctx, state, instance, "mock", &DummyStoreOfflineHook{})

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

			// Check retry count if expected
			if tt.expectedRetryCount != "" {
				annotations := instance.GetAnnotations()
				if annotations == nil {
					require.Fail(t, "Expected retry count annotation but no annotations found")
				} else {
					actualRetryCount := annotations[AnnotationKeyOfflineRetryCount]
					require.Equal(t, tt.expectedRetryCount, actualRetryCount, "Retry count mismatch")
				}
			}

			// Special handling for tests that explicitly expect retry count to be cleared (empty string)
			if tt.name == "Cancel operation: retry annotation should be cleared" ||
				tt.name == "Failed state: max retries reached" {
				annotations := instance.GetAnnotations()
				if annotations != nil {
					_, exists := annotations[AnnotationKeyOfflineRetryCount]
					require.False(t, exists, "Expected retry count annotation to be cleared")
				}
			}

			if tt.assertFunc != nil {
				tt.assertFunc(t, instance)
			}
		})
	}
}
