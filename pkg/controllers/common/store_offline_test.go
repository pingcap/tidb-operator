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
	"github.com/pingcap/tidb-operator/pkg/client"
	pdapi "github.com/pingcap/tidb-operator/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

// mockStore provides a mock implementation of the StoreOfflineInstance interface for testing.
type mockStore struct {
	client.Object
	name        string
	offline     bool
	condition   *metav1.Condition
	annotations map[string]string
}

func (m *mockStore) GetName() string                        { return m.name }
func (m *mockStore) IsOffline() bool                        { return m.offline }
func (m *mockStore) SetOffline(offline bool)                { m.offline = offline }
func (m *mockStore) GetOfflineCondition() *metav1.Condition { return m.condition }
func (m *mockStore) SetOfflineCondition(c metav1.Condition) { m.condition = &c }
func (m *mockStore) GetAnnotations() map[string]string      { return m.annotations }
func (m *mockStore) SetAnnotations(a map[string]string)     { m.annotations = a }

// --- Test Helpers ---

type storeBuilder struct {
	store *mockStore
}

func newStore() *storeBuilder {
	return &storeBuilder{
		store: &mockStore{name: "test", annotations: make(map[string]string)},
	}
}

func (b *storeBuilder) Offline(offline bool) *storeBuilder {
	b.store.offline = offline
	return b
}

func (b *storeBuilder) WithCondition(reason string, status metav1.ConditionStatus) *storeBuilder {
	cond := v1alpha1.NewOfflineCondition(reason, "", status)
	b.store.condition = &cond
	return b
}

func (b *storeBuilder) WithAnnotations(a map[string]string) *storeBuilder {
	for k, v := range a {
		b.store.annotations[k] = v
	}
	return b
}

func (b *storeBuilder) Build() *mockStore {
	return b.store
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
	b.ctx.EXPECT().GetStoreNotExists().Return(!b.storeExists).AnyTimes()
	b.ctx.EXPECT().GetLeaderCount().Return(0).AnyTimes()
	b.ctx.EXPECT().GetRegionCount().Return(0).AnyTimes()
	b.ctx.EXPECT().IsStoreUp().Return(true).AnyTimes()
	b.ctx.EXPECT().IsStoreBusy().Return(false).AnyTimes()
	b.ctx.EXPECT().SetStatusChanged().AnyTimes()
	b.ctx.EXPECT().SetLeaderCount(gomock.Any()).AnyTimes()
	b.ctx.EXPECT().SetRegionCount(gomock.Any()).AnyTimes()
	b.ctx.EXPECT().SetStoreBusy(gomock.Any()).AnyTimes()
	b.ctx.EXPECT().SetStoreState(gomock.Any()).AnyTimes()

	// Always set up GetStoreState to return the stored state or empty string
	b.ctx.EXPECT().GetStoreState().Return(b.storeState).AnyTimes()

	if b.pdClient != nil {
		b.ctx.EXPECT().GetPDClient().Return(b.pdClient).AnyTimes()
	}
}

// --- End Test Helpers ---

func TestTaskOfflineStoreStateMachine(t *testing.T) {
	ctx := context.Background()

	type testCase struct {
		name               string
		instance           *mockStore
		contextBuilder     *contextBuilder
		setupMock          func(mockUnderlay *pdapi.MockPDClient)
		expectedResult     task.Result
		expectedCondition  *metav1.Condition
		expectedRetryCount string
		assertFunc         func(t *testing.T, tt testCase)
		skipMockClientInit bool // Flag to explicitly skip mock client initialization
	}

	runTest := func(t *testing.T, tt testCase) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		var context StoreOfflineReconcileContext

		// Initialize the mock context unless explicitly skipped
		if !tt.skipMockClientInit {
			tt.contextBuilder.initMock(ctrl)

			// If no PD client is set and we need to mock it
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
			context = tt.contextBuilder.Build()
		} else {
			// For skipMockClientInit tests, still initialize mock but with nil PD client
			tt.contextBuilder.initMock(ctrl)
			tt.contextBuilder.ctx.EXPECT().GetPDClient().Return(nil).AnyTimes()
			context = tt.contextBuilder.Build()
		}

		result := TaskOfflineStoreStateMachine(ctx, context, tt.instance, "mock")

		// Default assertions
		require.Equal(t, tt.expectedResult.Status(), result.Status(), "Status mismatch")
		require.Equal(t, tt.expectedResult.RequeueAfter(), result.RequeueAfter(), "RequeueAfter mismatch")

		if tt.expectedCondition != nil {
			cond := tt.instance.GetOfflineCondition()
			require.NotNil(t, cond, "Offline condition should not be nil")
			require.Equal(t, tt.expectedCondition.Type, cond.Type, "Condition Type mismatch")
			require.Equal(t, tt.expectedCondition.Status, cond.Status, "Condition Status mismatch")
			require.Equal(t, tt.expectedCondition.Reason, cond.Reason, "Condition Reason mismatch")
		}

		if tt.expectedRetryCount != "" {
			require.Equal(t, tt.expectedRetryCount, tt.instance.GetAnnotations()[AnnotationKeyOfflineRetryCount], "Retry count annotation mismatch")
		} else if tt.instance.GetAnnotations() != nil {
			_, exists := tt.instance.GetAnnotations()[AnnotationKeyOfflineRetryCount]
			require.False(t, exists, "Retry count annotation should be cleared")
		}

		// Custom assertions for specific tests
		if tt.assertFunc != nil {
			tt.assertFunc(t, tt)
		}
	}

	t.Run("Happy Path", func(t *testing.T) {
		tests := []testCase{
			{
				name:              "Start offline operation: no condition exists",
				instance:          newStore().Offline(true).Build(),
				contextBuilder:    newContext(""),
				expectedResult:    task.Retry(PendingWaitInterval).With("offline operation is pending"),
				expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonPending},
			},
			{
				name:           "Pending state: transition to Active",
				instance:       newStore().Offline(true).WithCondition(v1alpha1.OfflineReasonPending, metav1.ConditionTrue).Build(),
				contextBuilder: newContext("1"),
				setupMock: func(mockUnderlay *pdapi.MockPDClient) {
					mockUnderlay.EXPECT().DeleteStore(gomock.Any(), "1").Return(nil)
				},
				expectedResult:    task.Retry(RemovingWaitInterval).With("store offline operation is active"),
				expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonActive},
			},
			{
				name:              "Active state: store still removing",
				instance:          newStore().Offline(true).WithCondition(v1alpha1.OfflineReasonActive, metav1.ConditionTrue).Build(),
				contextBuilder:    newContext("1").WithState(v1alpha1.StoreStateRemoving),
				expectedResult:    task.Retry(RemovingWaitInterval).With("waiting for store to be removed, current state: %s", v1alpha1.StoreStateRemoving),
				expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonActive},
			},
			{
				name:              "Active state: store removed, transition to Completed",
				instance:          newStore().Offline(true).WithCondition(v1alpha1.OfflineReasonActive, metav1.ConditionTrue).Build(),
				contextBuilder:    newContext("1").WithState(v1alpha1.StoreStateRemoved),
				expectedResult:    task.Complete().With("store offline operation completed"),
				expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonCompleted},
			},
			{
				name:           "Completed state should return complete",
				instance:       newStore().Offline(true).WithCondition(v1alpha1.OfflineReasonCompleted, metav1.ConditionTrue).Build(),
				contextBuilder: newContext(""),
				expectedResult: task.Complete().With("store offline operation is completed"),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) { runTest(t, tt) })
		}
	})

	t.Run("Failure and Retry", func(t *testing.T) {
		tests := []testCase{
			{
				name:           "Pending state: PD API call fails, transition to Failed",
				instance:       newStore().Offline(true).WithCondition(v1alpha1.OfflineReasonPending, metav1.ConditionTrue).Build(),
				contextBuilder: newContext("2"),
				setupMock: func(mockUnderlay *pdapi.MockPDClient) {
					mockUnderlay.EXPECT().DeleteStore(gomock.Any(), "2").Return(errors.New("transient pd api error"))
				},
				expectedResult:    task.Retry(RemovingWaitInterval).With("failed to call PD DeleteStore API, will retry"),
				expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonFailed},
			},
			{
				name:           "Pending state: empty store ID",
				instance:       newStore().Offline(true).WithCondition(v1alpha1.OfflineReasonPending, metav1.ConditionTrue).Build(),
				contextBuilder: newContext(""),
				expectedResult: task.Fail().With("store ID is empty"),
			},
			{
				name:               "Failed state: retry operation",
				instance:           newStore().Offline(true).WithCondition(v1alpha1.OfflineReasonFailed, metav1.ConditionTrue).WithAnnotations(map[string]string{AnnotationKeyOfflineRetryCount: "1"}).Build(),
				contextBuilder:     newContext(""),
				expectedResult:     task.Retry(PendingWaitInterval).With("offline operation is pending"),
				expectedCondition:  &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonPending},
				expectedRetryCount: "2",
			},
			{
				name:               "Failed state: max retries reached",
				instance:           newStore().Offline(true).WithCondition(v1alpha1.OfflineReasonFailed, metav1.ConditionTrue).WithAnnotations(map[string]string{AnnotationKeyOfflineRetryCount: strconv.Itoa(MaxRetryCount)}).Build(),
				contextBuilder:     newContext(""),
				expectedResult:     task.Fail().With("offline operation failed after %d retries", MaxRetryCount),
				expectedCondition:  &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonFailed},
				expectedRetryCount: "",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) { runTest(t, tt) })
		}
	})

	t.Run("Cancellation", func(t *testing.T) {
		tests := []testCase{
			{
				name:           "Cancel operation: in Active state",
				instance:       newStore().Offline(false).WithCondition(v1alpha1.OfflineReasonActive, metav1.ConditionTrue).Build(),
				contextBuilder: newContext("3").WithState(v1alpha1.StoreStateRemoving),
				setupMock: func(mockUnderlay *pdapi.MockPDClient) {
					mockUnderlay.EXPECT().CancelDeleteStore(gomock.Any(), "3").Return(nil)
				},
				expectedResult:    task.Complete().With("offline operation canceled"),
				expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.OfflineReasonCancelled},
			},
			{
				name:           "Cancel operation: PD API fails",
				instance:       newStore().Offline(false).WithCondition(v1alpha1.OfflineReasonActive, metav1.ConditionTrue).Build(),
				contextBuilder: newContext("1").WithState(v1alpha1.StoreStateRemoving),
				setupMock: func(mockUnderlay *pdapi.MockPDClient) {
					mockUnderlay.EXPECT().CancelDeleteStore(gomock.Any(), "1").Return(errors.New("cancel api error"))
				},
				expectedResult: task.Fail().With("Failed to cancel store deletion, will retry: cancel api error"),
			},
			{
				name:               "Cancel operation: retry annotation should be cleared",
				instance:           newStore().Offline(false).WithCondition(v1alpha1.OfflineReasonFailed, metav1.ConditionTrue).WithAnnotations(map[string]string{AnnotationKeyOfflineRetryCount: "2"}).Build(),
				contextBuilder:     newContext("1"),
				expectedResult:     task.Complete().With("offline operation canceled"),
				expectedCondition:  &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.OfflineReasonCancelled},
				expectedRetryCount: "",
			},
			{
				name:              "Restart a canceled operation",
				instance:          newStore().Offline(true).WithCondition(v1alpha1.OfflineReasonCancelled, metav1.ConditionFalse).Build(),
				contextBuilder:    newContext(""),
				expectedResult:    task.Retry(PendingWaitInterval).With("offline operation is pending"),
				expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonPending},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) { runTest(t, tt) })
		}
	})

	t.Run("Edge Cases", func(t *testing.T) {
		tests := []testCase{
			{
				name:               "Nil PD client in pending state",
				instance:           newStore().Offline(true).WithCondition(v1alpha1.OfflineReasonPending, metav1.ConditionTrue).Build(),
				contextBuilder:     newContext("1").WithPDClient(nil),
				expectedResult:     task.Fail().With("pd client is not registered"),
				skipMockClientInit: true,
			},
			{
				name:               "Nil PD client in cancel state",
				instance:           newStore().Offline(false).WithCondition(v1alpha1.OfflineReasonActive, metav1.ConditionTrue).Build(),
				contextBuilder:     newContext("1").WithState(v1alpha1.StoreStateRemoving).WithPDClient(nil),
				expectedResult:     task.Fail().With("pd client is not registered"),
				skipMockClientInit: true,
			},
			{
				name:              "Store doesn't exist in pending state",
				instance:          newStore().Offline(true).WithCondition(v1alpha1.OfflineReasonPending, metav1.ConditionTrue).Build(),
				contextBuilder:    newContext("1").StoreExists(false),
				expectedResult:    task.Complete().With("store does not exist, offline operation completed"),
				expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonCompleted},
			},
			{
				name:              "Store doesn't exist in active state",
				instance:          newStore().Offline(true).WithCondition(v1alpha1.OfflineReasonActive, metav1.ConditionTrue).Build(),
				contextBuilder:    newContext("1").StoreExists(false),
				expectedResult:    task.Complete().With("store offline operation completed"),
				expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonCompleted},
			},
			{
				name:           "Unknown condition reason",
				instance:       newStore().Offline(true).WithCondition("UnknownReason", metav1.ConditionTrue).Build(),
				contextBuilder: newContext(""),
				expectedResult: task.Fail().With("unknown offline condition reason: %s", "UnknownReason"),
			},
			{
				name:           "Cancel when no condition exists",
				instance:       newStore().Offline(false).Build(),
				contextBuilder: newContext(""),
				expectedResult: task.Complete().With("no offline operation to cancel"),
			},
			{
				name:           "Cancel when already completed",
				instance:       newStore().Offline(false).WithCondition(v1alpha1.OfflineReasonCompleted, metav1.ConditionTrue).Build(),
				contextBuilder: newContext(""),
				expectedResult: task.Complete().With("cannot cancel completed offline operation"),
			},
			{
				name:           "Cancel when already canceled",
				instance:       newStore().Offline(false).WithCondition(v1alpha1.OfflineReasonCancelled, metav1.ConditionFalse).Build(),
				contextBuilder: newContext(""),
				expectedResult: task.Complete().With("offline operation already canceled"),
			},
			{
				name: "Failed state with nil annotations",
				instance: func() *mockStore {
					s := newStore().Offline(true).WithCondition(v1alpha1.OfflineReasonFailed, metav1.ConditionTrue).Build()
					s.annotations = nil
					return s
				}(),
				contextBuilder:     newContext(""),
				expectedResult:     task.Retry(PendingWaitInterval).With("offline operation is pending"),
				expectedCondition:  &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonPending},
				expectedRetryCount: "1",
			},
			{
				name:               "Failed state with invalid retry count annotation",
				instance:           newStore().Offline(true).WithCondition(v1alpha1.OfflineReasonFailed, metav1.ConditionTrue).WithAnnotations(map[string]string{AnnotationKeyOfflineRetryCount: "invalid"}).Build(),
				contextBuilder:     newContext(""),
				expectedResult:     task.Retry(PendingWaitInterval).With("offline operation is pending"),
				expectedCondition:  &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionTrue, Reason: v1alpha1.OfflineReasonPending},
				expectedRetryCount: "1",
			},
			{
				name:              "Cancel operation clears retry annotation but keeps others",
				instance:          newStore().Offline(false).WithCondition(v1alpha1.OfflineReasonFailed, metav1.ConditionTrue).WithAnnotations(map[string]string{AnnotationKeyOfflineRetryCount: "2", "other": "value"}).Build(),
				contextBuilder:    newContext("1"),
				expectedResult:    task.Complete().With("offline operation canceled"),
				expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.OfflineReasonCancelled},
				assertFunc: func(t *testing.T, tt testCase) {
					require.Contains(t, tt.instance.GetAnnotations(), "other", "Other annotations should remain")
					require.Equal(t, "value", tt.instance.GetAnnotations()["other"], "Other annotation value should be preserved")
				},
			},
			{
				name:               "Cancel operation: empty store ID",
				instance:           newStore().Offline(false).WithCondition(v1alpha1.OfflineReasonActive, metav1.ConditionTrue).Build(),
				contextBuilder:     newContext("").WithState(v1alpha1.StoreStateRemoving),
				expectedResult:     task.Fail().With("store ID is empty"),
				skipMockClientInit: true,
			},
			{
				name:               "Cancel operation: whitespace store ID",
				instance:           newStore().Offline(false).WithCondition(v1alpha1.OfflineReasonActive, metav1.ConditionTrue).Build(),
				contextBuilder:     newContext("   ").WithState(v1alpha1.StoreStateRemoving),
				expectedResult:     task.Fail().With("store ID is empty"),
				skipMockClientInit: true,
			},
			{
				name:              "Cancel operation: store not in removing state",
				instance:          newStore().Offline(false).WithCondition(v1alpha1.OfflineReasonActive, metav1.ConditionTrue).Build(),
				contextBuilder:    newContext("1").WithState(v1alpha1.StoreStateServing),
				expectedResult:    task.Complete().With("offline operation canceled"),
				expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.OfflineReasonCancelled},
			},
			{
				name:              "Cancel operation: store doesn't exist",
				instance:          newStore().Offline(false).WithCondition(v1alpha1.OfflineReasonActive, metav1.ConditionTrue).Build(),
				contextBuilder:    newContext("1").StoreExists(false).WithState(v1alpha1.StoreStateRemoving),
				expectedResult:    task.Complete().With("offline operation canceled"),
				expectedCondition: &metav1.Condition{Type: v1alpha1.StoreOfflineConditionType, Status: metav1.ConditionFalse, Reason: v1alpha1.OfflineReasonCancelled},
			},
			{
				name:           "Cancel unknown condition reason",
				instance:       newStore().Offline(false).WithCondition("UnknownReason", metav1.ConditionTrue).Build(),
				contextBuilder: newContext(""),
				expectedResult: task.Fail().With("unknown offline condition reason: %s", "UnknownReason"),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) { runTest(t, tt) })
		}
	})
}

func TestTaskOfflineStoreStateMachine_NilStore(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	result := TaskOfflineStoreStateMachine(ctx, NewMockStoreOfflineReconcileContext(ctrl), nil, "mock")
	require.Equal(t, task.SFail, result.Status())
	require.Contains(t, result.Message(), "store is nil")
}
