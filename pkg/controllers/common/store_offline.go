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
	"strconv"
	"strings"
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
	// AnnotationKeyOfflineRetryCount is the annotation key for tracking retry attempts.
	AnnotationKeyOfflineRetryCount = "pingcap.com/offline-retry-count"
)

// StoreOfflineReconcileContext defines the interface for reconcile context with store offline capabilities.
type StoreOfflineReconcileContext interface {
	StoreState
	StoreStateUpdater
	StatusUpdater

	// GetStoreID returns the store ID for PD operations
	GetStoreID() string
	// GetStoreNotExists returns true if the store does not exist in PD
	GetStoreNotExists() bool
	// GetPDClient returns the PD client for API operations
	GetPDClient() pdm.PDClient
}

// TaskOfflineStoreStateMachine handles the two-step store deletion process based on spec.offline field.
// This implements the state machine for offline operations: Pending -> Active -> Completed/Failed/Canceled.
func TaskOfflineStoreStateMachine(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.Store,
	storeTypeName string,
) task.Result {
	if store == nil {
		return task.Fail().With("%s store store is nil", storeTypeName)
	}

	// Check if offline operation is requested
	if !store.IsOffline() {
		// If offline is false, check if we need to cancel an ongoing operation
		return handleOfflineCancellation(ctx, state, store)
	}

	// Handle offline operation based on current condition state
	return handleOfflineOperation(ctx, state, store)
}

// handleOfflineOperation manages the offline operation state machine.
func handleOfflineOperation(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.Store,
) task.Result {
	condition := store.GetOfflineCondition()

	// If no condition exists, start with Pending state
	if condition == nil {
		return transitionToPending(ctx, state, store)
	}

	switch condition.Reason {
	case v1alpha1.OfflineReasonPending:
		return handlePendingState(ctx, state, store)
	case v1alpha1.OfflineReasonActive:
		return handleActiveState(ctx, state, store)
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
	store runtime.Store,
) task.Result {
	condition := store.GetOfflineCondition()
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
		return cancelOfflineOperation(ctx, state, store)
	default:
		return task.Fail().With("unknown offline condition reason: %s", condition.Reason)
	}
}

// transitionToPending sets the offline condition to Pending state.
func transitionToPending(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.Store,
) task.Result {
	logger := logr.FromContextOrDiscard(ctx).WithValues("instance", store.GetName())
	logger.Info("offline operation transitioned to Pending state")
	updateOfflineCondition(state, store, v1alpha1.NewOfflineCondition(
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
	store runtime.Store,
) task.Result {
	logger := logr.FromContextOrDiscard(ctx).WithValues("instance", store.GetName())
	if state.GetStoreNotExists() {
		// Store doesn't exist, mark as completed
		logger.Info("Store does not exist, completing offline operation")
		updateOfflineCondition(state, store, v1alpha1.NewOfflineCondition(
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

	logger.Info("Calling PD DeleteStore API", "storeID", storeID)
	if err := state.GetPDClient().Underlay().DeleteStore(ctx, storeID); err != nil {
		// Transition to Failed state for retry
		updateOfflineCondition(state, store, v1alpha1.NewOfflineCondition(
			v1alpha1.OfflineReasonFailed,
			fmt.Sprintf("Failed to call PD DeleteStore API: %v", err),
			metav1.ConditionTrue,
		))
		// Return a retryable error to re-trigger the failed state handling
		return task.Retry(RemovingWaitInterval).With("failed to call PD DeleteStore API, will retry")
	}

	// Transition to Active state
	logger.Info("Store offline operation transitioned to Active state")
	updateOfflineCondition(state, store, v1alpha1.NewOfflineCondition(
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
	store runtime.Store,
) task.Result {
	logger := logr.FromContextOrDiscard(ctx).WithValues("instance", store.GetName())
	if state.GetStoreNotExists() {
		// Store has been removed, mark as completed
		logger.Info("Store has been removed from PD, completing offline operation")
		updateOfflineCondition(state, store, v1alpha1.NewOfflineCondition(
			v1alpha1.OfflineReasonCompleted,
			"Store has been successfully removed from PD",
			metav1.ConditionTrue,
		))
		return task.Complete().With("store offline operation completed")
	}

	// Check if store is in "Removed" state
	if state.GetStoreState() == v1alpha1.StoreStateRemoved {
		logger.Info("Store is Removed, completing offline operation")
		updateOfflineCondition(state, store, v1alpha1.NewOfflineCondition(
			v1alpha1.OfflineReasonCompleted,
			"Store state is Removed, offline operation completed",
			metav1.ConditionTrue,
		))
		return task.Complete().With("store offline operation completed")
	}

	// Continue monitoring
	return task.Retry(RemovingWaitInterval).With("waiting for store to be removed, current state: %s", state.GetStoreState())
}

// handleFailedState processes the Failed state and implements retry logic.
func handleFailedState(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.Store,
) task.Result {
	logger := logr.FromContextOrDiscard(ctx).WithValues("instance", store.GetName())
	annotations := store.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	retryCountStr := annotations[AnnotationKeyOfflineRetryCount]
	retryCount, _ := strconv.Atoi(retryCountStr)

	if retryCount >= MaxRetryCount {
		errMsg := fmt.Sprintf("offline operation failed after %d retries", MaxRetryCount)
		logger.Info("offline operation failed permanently", "error", errMsg, "retryCount", retryCount, "maxRetryCount", MaxRetryCount)
		condition := store.GetOfflineCondition()
		if condition != nil && condition.Message != errMsg {
			updateOfflineCondition(state, store, v1alpha1.NewOfflineCondition(
				v1alpha1.OfflineReasonFailed,
				errMsg,
				metav1.ConditionTrue,
			))
		}
		// Clean up the retry annotation upon permanent failure.
		delete(annotations, AnnotationKeyOfflineRetryCount)
		store.SetAnnotations(annotations)
		state.SetStatusChanged()
		return task.Fail().With(errMsg)
	}

	// Increment retry count and update annotation
	logger.Info("Store offline operation retrying", "attempt", retryCount+1, "maxAttempts", MaxRetryCount)
	annotations[AnnotationKeyOfflineRetryCount] = strconv.Itoa(retryCount + 1)
	store.SetAnnotations(annotations)
	state.SetStatusChanged() // Annotations are part of metadata, but let's signal a change.

	// Retry by transitioning back to Pending
	return transitionToPending(ctx, state, store)
}

// cancelOfflineOperation cancels an ongoing offline operation.
func cancelOfflineOperation(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.Store,
) task.Result {
	logger := logr.FromContextOrDiscard(ctx).WithValues("instance", store.GetName())
	if state.GetPDClient() == nil {
		return task.Fail().With("pd client is not registered")
	}

	storeID := state.GetStoreID()
	if strings.TrimSpace(storeID) == "" {
		return task.Fail().With("store ID is empty")
	}

	// Only cancel if store exists and is in removing state
	if !state.GetStoreNotExists() && state.GetStoreState() == v1alpha1.StoreStateRemoving {
		logger.Info("Calling PD CancelDeleteStore API", "storeID", storeID)
		if err := state.GetPDClient().Underlay().CancelDeleteStore(ctx, storeID); err != nil {
			// If cancellation fails, update the condition message and retry.
			errMsg := fmt.Sprintf("Failed to cancel store deletion, will retry: %v", err)
			updateOfflineCondition(state, store, v1alpha1.NewOfflineCondition(
				store.GetOfflineCondition().Reason, // Keep the current reason (e.g., Active)
				errMsg,
				metav1.ConditionTrue,
			))
			return task.Fail().With(errMsg)
		}
	}

	// Update condition to canceled
	logger.Info("Store offline operation canceled")
	updateOfflineCondition(state, store, v1alpha1.NewOfflineCondition(
		v1alpha1.OfflineReasonCancelled,
		"Store offline operation has been canceled",
		metav1.ConditionFalse,
	))

	// Reset retry count on cancellation
	annotations := store.GetAnnotations()
	if annotations != nil {
		delete(annotations, AnnotationKeyOfflineRetryCount)
		store.SetAnnotations(annotations)
	}

	return task.Complete().With("offline operation canceled")
}

// updateOfflineCondition updates the offline condition in the store status.
func updateOfflineCondition(
	state StoreOfflineReconcileContext,
	store runtime.Store,
	condition metav1.Condition,
) {
	store.SetOfflineCondition(condition)
	state.SetStatusChanged()
}
