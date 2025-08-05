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

//go:generate ${GOBIN}/mockgen -write_command_comment=false -copyright_file ${BOILERPLATE_FILE} -destination store_offline_mock_generated.go -package=common ${GO_MODULE}/pkg/controllers/common StoreOfflineReconcileContext
package common

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

const (
	// RemovingWaitInterval is the interval to wait for store removal.
	RemovingWaitInterval = 10 * time.Second
	// PendingWaitInterval is the interval to wait before starting offline operation.
	PendingWaitInterval = 2 * time.Second
	// MaxRetryCount is the maximum number of retry attempts for failed operations.
	MaxRetryCount = 3
	// BaseRetryInterval is the base interval for retry attempts.
	BaseRetryInterval = 10 * time.Second
)

type StoreOfflineHook interface {
	BeforeDeleteStore(ctx context.Context) (wait bool, err error)
	AfterStoreIsDeleted(ctx context.Context) (wait bool, err error)
	AfterCancelDeleteStore(ctx context.Context) (wait bool, err error)
}

type DummyStoreOfflineHook struct{}

var _ StoreOfflineHook = &DummyStoreOfflineHook{}

func (DummyStoreOfflineHook) BeforeDeleteStore(_ context.Context) (wait bool, err error) {
	return false, nil
}

func (DummyStoreOfflineHook) AfterStoreIsDeleted(_ context.Context) (wait bool, err error) {
	return false, nil
}

func (DummyStoreOfflineHook) AfterCancelDeleteStore(_ context.Context) (wait bool, err error) {
	return false, nil
}

// StoreOfflineReconcileContext defines the interface for reconcile context with store offline capabilities.
type StoreOfflineReconcileContext interface {
	StoreState
	StatusUpdater

	// GetStoreID returns the store ID for PD operations
	GetStoreID() string
	// StoreNotExists returns true if the store does not exist in PD
	StoreNotExists() bool
	// GetPDClient returns the PD client for API operations
	GetPDClient() pdm.PDClient
}

// TaskOfflineStoreStateMachine handles the two-step store deletion process based on spec.offline field.
// This implements the state machine for offline operations: Pending -> Active -> Completed/Failed/Canceled.
func TaskOfflineStoreStateMachine(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.StoreInstance,
	storeTypeName string,
	hook StoreOfflineHook,
) task.Result {
	if store == nil {
		return task.Fail().With("%s store is nil", storeTypeName)
	}

	// Check if offline operation is requested
	if !store.IsOffline() {
		// If offline is false, check if we need to cancel an ongoing operation
		return handleOfflineCancellation(ctx, state, store, hook)
	}

	// Handle offline operation based on current condition state
	return handleOfflineOperation(ctx, state, store, hook)
}

// handleOfflineOperation manages the offline operation state machine.
func handleOfflineOperation(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.StoreInstance,
	hook StoreOfflineHook,
) task.Result {
	condition := runtime.GetOfflineCondition(store)

	// If no condition exists, start with Pending state
	if condition == nil {
		return transitionToPending(ctx, state, store)
	}

	switch condition.Reason {
	case v1alpha1.OfflineReasonPending:
		return handlePendingState(ctx, state, store, hook)
	case v1alpha1.OfflineReasonActive:
		return handleActiveState(ctx, state, store, hook)
	case v1alpha1.OfflineReasonCompleted:
		return task.Complete().With("store offline operation is completed")
	case v1alpha1.OfflineReasonFailed:
		return handleFailedState(ctx, state, store)
	case v1alpha1.OfflineReasonCancelled:
		// This case is reached if spec.offline is set to true for a store
		// that was previously canceled. It restarts the offline process.
		return transitionToPending(ctx, state, store)
	default:
		return task.Fail().With("unknown offline condition reason: %s", condition.Reason)
	}
}

// handleOfflineCancellation handles the cancellation of offline operations when offline=false.
func handleOfflineCancellation(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.StoreInstance,
	hook StoreOfflineHook,
) task.Result {
	condition := runtime.GetOfflineCondition(store)
	// If no offline condition exists, nothing to cancel
	if condition == nil {
		return task.Complete().With("no offline operation to cancel")
	}

	switch condition.Reason {
	case v1alpha1.OfflineReasonCompleted:
		// Cannot cancel completed operations
		return task.Complete().With("cannot cancel completed offline operation")
	case v1alpha1.OfflineReasonCancelled:
		// Already canceled
		return task.Complete().With("offline operation already canceled")
	case v1alpha1.OfflineReasonPending, v1alpha1.OfflineReasonActive, v1alpha1.OfflineReasonFailed:
		// Cancel the operation
		return cancelOfflineOperation(ctx, state, store, hook)
	default:
		return task.Fail().With("unknown offline condition reason: %s", condition.Reason)
	}
}

// transitionToPending sets the offline condition to Pending state.
func transitionToPending(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.StoreInstance,
) task.Result {
	logger := logr.FromContextOrDiscard(ctx).WithValues("instance", store.GetName())
	logger.Info("offline operation transitioned to Pending state")
	updateOfflineCondition(state, store, newOfflineCondition(
		v1alpha1.OfflineReasonPending,
		"Store offline operation is requested and pending",
		metav1.ConditionTrue,
	))
	return task.Retry(PendingWaitInterval).With("offline operation is pending")
}

// handlePendingState processes the Pending state and transitions to Active.
func handlePendingState(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.StoreInstance,
	hook StoreOfflineHook,
) task.Result {
	logger := logr.FromContextOrDiscard(ctx).WithValues("instance", store.GetName())
	if state.StoreNotExists() {
		// Store doesn't exist, mark as completed
		logger.Info("Store does not exist, completing offline operation")
		// Reset retry count when operation completes
		resetRetryCount(state, store)
		updateOfflineCondition(state, store, newOfflineCondition(
			v1alpha1.OfflineReasonCompleted,
			"Store does not exist, offline operation completed",
			metav1.ConditionTrue,
		))
		return task.Complete().With("store does not exist, offline operation completed")
	}

	if state.GetPDClient() == nil {
		return task.Fail().With("pd client is not registered")
	}

	storeID := state.GetStoreID()
	if storeID == "" {
		return task.Fail().With("store ID is empty")
	}

	wait, err := hook.BeforeDeleteStore(ctx)
	if err != nil {
		return task.Fail().With("failed to execute the hook before delete store: %v", err)
	}
	if wait {
		return task.Retry(RemovingWaitInterval).With("wait for the hook BeforeDeleteStore")
	}

	logger.Info("Calling PD DeleteStore API", "storeID", storeID)
	if err := state.GetPDClient().Underlay().DeleteStore(ctx, storeID); err != nil {
		// Increment retry count when PD API fails
		incrementRetryCount(state, store)
		// Transition to Failed state for retry
		updateOfflineCondition(state, store, newOfflineCondition(
			v1alpha1.OfflineReasonFailed,
			fmt.Sprintf("Failed to call PD DeleteStore API: %v", err),
			metav1.ConditionTrue,
		))
		// Return a retryable error to re-trigger the failed state handling
		return task.Retry(RemovingWaitInterval).With("failed to call PD DeleteStore API, will retry")
	}

	// Reset retry count on successful PD API call
	resetRetryCount(state, store)

	// Transition to Active state
	logger.Info("Store offline operation transitioned to Active state")
	updateOfflineCondition(state, store, newOfflineCondition(
		v1alpha1.OfflineReasonActive,
		"Store offline operation is active, data migration is in progress",
		metav1.ConditionTrue,
	))
	// Note: We don't set store state to Removing here. Instead, we honor the store state
	// from PD, which will automatically transition from Serving -> Removing -> Removed
	// after calling DeleteStore API. This avoids state conflicts between operator and PD.
	return task.Retry(RemovingWaitInterval).With("store offline operation is active")
}

// handleActiveState processes the Active state and monitors store removal progress.
func handleActiveState(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.StoreInstance,
	hook StoreOfflineHook,
) task.Result {
	logger := logr.FromContextOrDiscard(ctx).WithValues("instance", store.GetName())
	if state.StoreNotExists() || state.GetStoreState() == v1alpha1.StoreStateRemoved {
		wait, err := hook.AfterStoreIsDeleted(ctx)
		if err != nil {
			return task.Fail().With("failed to execute the hook after store is deleted: %v", err)
		}
		if wait {
			return task.Retry(RemovingWaitInterval).With("wait for the hook AfterStoreIsDeleted")
		}

		logger.Info("Store has been removed from PD, completing offline operation")
		// Reset retry count on successful completion
		resetRetryCount(state, store)
		updateOfflineCondition(state, store, newOfflineCondition(
			v1alpha1.OfflineReasonCompleted,
			"Store state is Removed, offline operation completed",
			metav1.ConditionTrue,
		))
		return task.Complete().With("store offline operation completed")
	}
	return task.Retry(RemovingWaitInterval).With("waiting for store to be removed, current state: %s", state.GetStoreState())
}

// handleFailedState processes the Failed state and implements retry logic.
func handleFailedState(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.StoreInstance,
) task.Result {
	logger := logr.FromContextOrDiscard(ctx).WithValues("instance", store.GetName())

	// Get current retry count from status
	retryCount := getRetryCount(store)

	if !shouldRetry(store) {
		errMsg := fmt.Sprintf("offline operation failed after %d retries", MaxRetryCount)
		logger.Info("offline operation failed permanently", "error", errMsg, "retryCount", retryCount, "maxRetryCount", MaxRetryCount)
		condition := runtime.GetOfflineCondition(store)
		if condition != nil && condition.Message != errMsg {
			updateOfflineCondition(state, store, newOfflineCondition(
				v1alpha1.OfflineReasonFailed,
				errMsg,
				metav1.ConditionTrue,
			))
		}
		return task.Fail().With(errMsg)
	}

	// Calculate retry interval with exponential backoff
	retryInterval := calculateRetryInterval(retryCount)
	logger.Info("Store offline operation retrying", "attempt", retryCount+1, "maxAttempts", MaxRetryCount, "retryInterval", retryInterval)

	// Retry by transitioning back to Pending
	return transitionToPendingWithRetry(ctx, state, store, retryInterval)
}

// getRetryCount gets the current retry count from the store status.
func getRetryCount(store runtime.StoreInstance) int {
	// Use type assertion to get the actual store status
	switch v := any(store).(type) {
	case *runtime.TiKV:
		if v.Status.OfflineRetry != nil {
			return v.Status.OfflineRetry.Count
		}
	case *runtime.TiFlash:
		if v.Status.OfflineRetry != nil {
			return v.Status.OfflineRetry.Count
		}
	}
	return 0
}

// incrementRetryCount increments the retry count in the store status.
func incrementRetryCount(state StoreOfflineReconcileContext, store runtime.StoreInstance) {
	now := metav1.Now()

	// Use type assertion to update the actual store status
	switch v := any(store).(type) {
	case *runtime.TiKV:
		if v.Status.OfflineRetry == nil {
			v.Status.OfflineRetry = &v1alpha1.OfflineRetryStatus{
				Count:            1,
				FirstFailureTime: &now,
				LastRetryTime:    &now,
			}
		} else {
			v.Status.OfflineRetry.Count++
			v.Status.OfflineRetry.LastRetryTime = &now
			if v.Status.OfflineRetry.FirstFailureTime == nil {
				v.Status.OfflineRetry.FirstFailureTime = &now
			}
		}
		state.SetStatusChanged()
	case *runtime.TiFlash:
		if v.Status.OfflineRetry == nil {
			v.Status.OfflineRetry = &v1alpha1.OfflineRetryStatus{
				Count:            1,
				FirstFailureTime: &now,
				LastRetryTime:    &now,
			}
		} else {
			v.Status.OfflineRetry.Count++
			v.Status.OfflineRetry.LastRetryTime = &now
			if v.Status.OfflineRetry.FirstFailureTime == nil {
				v.Status.OfflineRetry.FirstFailureTime = &now
			}
		}
		state.SetStatusChanged()
	}
}

// resetRetryCount resets the retry count in the store status.
func resetRetryCount(state StoreOfflineReconcileContext, store runtime.StoreInstance) {
	// Use type assertion to reset the actual store status
	switch v := any(store).(type) {
	case *runtime.TiKV:
		v.Status.OfflineRetry = nil
		state.SetStatusChanged()
	case *runtime.TiFlash:
		v.Status.OfflineRetry = nil
		state.SetStatusChanged()
	}
}

// shouldRetry determines whether the operation should retry based on retry count.
func shouldRetry(store runtime.StoreInstance) bool {
	return getRetryCount(store) < MaxRetryCount
}

// calculateRetryInterval calculates the retry interval with exponential backoff
func calculateRetryInterval(retryCount int) time.Duration {
	if retryCount <= 0 {
		return BaseRetryInterval
	}
	// Exponential backoff: 10s, 30s, 90s, etc.
	return BaseRetryInterval * time.Duration(math.Pow(3, float64(retryCount-1)))
}

// transitionToPendingWithRetry sets the offline condition to Pending state with a custom retry interval
func transitionToPendingWithRetry(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.StoreInstance,
	retryInterval time.Duration,
) task.Result {
	logger := logr.FromContextOrDiscard(ctx).WithValues("instance", store.GetName())
	logger.Info("offline operation transitioned to Pending state", "retryInterval", retryInterval)
	updateOfflineCondition(state, store, newOfflineCondition(
		v1alpha1.OfflineReasonPending,
		"Store offline operation is requested and pending",
		metav1.ConditionTrue,
	))
	return task.Retry(retryInterval).With("offline operation is pending")
}

// cancelOfflineOperation cancels an ongoing offline operation.
func cancelOfflineOperation(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.StoreInstance,
	hook StoreOfflineHook,
) task.Result {
	logger := logr.FromContextOrDiscard(ctx).WithValues("instance", store.GetName())
	if state.GetPDClient() == nil {
		return task.Fail().With("pd client is not registered")
	}

	if state.StoreNotExists() {
		// Reset retry count when operation completes
		resetRetryCount(state, store)
		updateOfflineCondition(state, store, newOfflineCondition(
			v1alpha1.OfflineReasonCompleted,
			"Store does not exist, offline operation completed",
			metav1.ConditionTrue,
		))
		return task.Complete().With("store does not exist, offline operation completed")
	}

	storeID := state.GetStoreID()
	if storeID == "" {
		return task.Fail().With("store ID is empty")
	}

	// Only cancel if store exists and is in removing state
	if !state.StoreNotExists() &&
		(state.GetStoreState() == v1alpha1.StoreStateServing || state.GetStoreState() == v1alpha1.StoreStateRemoving) {
		logger.Info("Calling PD CancelDeleteStore API", "storeID", storeID)
		if err := state.GetPDClient().Underlay().CancelDeleteStore(ctx, storeID); err != nil {
			// If cancellation fails, update the condition message and retry.
			errMsg := fmt.Sprintf("Failed to cancel store deletion, will retry: %v", err)
			updateOfflineCondition(state, store, newOfflineCondition(
				runtime.GetOfflineCondition(store).Reason, // Keep the current reason (e.g., Active)
				errMsg,
				metav1.ConditionTrue,
			))
			return task.Fail().With(errMsg)
		}

		wait, err := hook.AfterCancelDeleteStore(ctx)
		if err != nil {
			return task.Fail().With("failed to execute the hook after cancel delete store")
		}
		if wait {
			return task.Retry(RemovingWaitInterval).With("wait for the hook AfterCancelDeleteStore")
		}
	}

	// Update condition to canceled
	logger.Info("Store offline operation canceled")
	updateOfflineCondition(state, store, newOfflineCondition(
		v1alpha1.OfflineReasonCancelled,
		"Store offline operation has been canceled",
		metav1.ConditionFalse,
	))

	return task.Complete().With("offline operation canceled")
}

// updateOfflineCondition updates the offline condition in the store status.
func updateOfflineCondition(
	state StoreOfflineReconcileContext,
	store runtime.StoreInstance,
	condition *metav1.Condition,
) {
	runtime.SetOfflineCondition(store, condition)
	state.SetStatusChanged()
}

func newOfflineCondition(reason, message string, status metav1.ConditionStatus) *metav1.Condition {
	return &metav1.Condition{
		Type:               v1alpha1.StoreOfflineConditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}
}
