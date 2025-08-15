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
)

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
// This implements the state machine for offline operations: Processing -> Completed/Failed/Canceled.
func TaskOfflineStoreStateMachine(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.Instance,
) task.Result {
	if store == nil {
		return task.Fail().With("store is nil")
	}

	// Check if offline operation is requested
	if !store.IsOffline() {
		// If offline is false, check if we need to cancel an ongoing operation
		return handleOfflineCancellation(ctx, state, store)
	}

	// Handle offline operation based on current condition state
	return handleOfflineOperation(ctx, state, store)
}

// recoverOfflineStateFromPD recovers the correct offline condition based on PD's actual store state.
// This function is called when local condition is lost (condition == nil) to rebuild the appropriate
// condition based on PD's authoritative state and the current spec.offline setting.
func recoverOfflineStateFromPD(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.Instance,
) *metav1.Condition {
	logger := logr.FromContextOrDiscard(ctx).WithValues("instance", store.GetName())
	logger.Info("Recovering offline state from PD due to lost condition")

	// Check if store exists in PD
	if state.StoreNotExists() {
		// Store doesn't exist, should be completed
		logger.Info("Store does not exist in PD, recovering as completed state")
		return newOfflinedCondition(
			v1alpha1.ReasonOfflineCompleted,
			"Store does not exist, offline operation completed (recovered from PD state)",
			metav1.ConditionTrue,
		)
	}

	// Get current store state from PD
	pdStoreState := state.GetStoreState()
	logger.Info("Recovering state based on PD store state", "pdState", pdStoreState)

	switch pdStoreState {
	case v1alpha1.StoreStateServing:
		// Store is serving in PD, we should start the offline operation
		logger.Info("Store is serving in PD, recovering as need to start offline operation")
		return nil // Return nil to indicate we should start fresh offline operation

	case v1alpha1.StoreStateRemoving:
		// Store is being removed in PD, we should be in processing state
		logger.Info("Store is removing in PD, recovering as processing state")
		return newOfflinedCondition(
			v1alpha1.ReasonOfflineProcessing,
			"Store offline operation is processing, data migration is in progress (recovered from PD state)",
			metav1.ConditionFalse,
		)

	case v1alpha1.StoreStateRemoved:
		// Store has been removed in PD, operation should be completed
		logger.Info("Store is removed in PD, recovering as completed state")
		return newOfflinedCondition(
			v1alpha1.ReasonOfflineCompleted,
			"Store state is removed, offline operation completed (recovered from PD state)",
			metav1.ConditionTrue,
		)

	default:
		// Unknown state, start fresh to be safe
		logger.Info("Unknown store state in PD, recovering as need to start offline operation", "unknownState", pdStoreState)
		return nil // Return nil to indicate we should start fresh offline operation
	}
}

// handleOfflineOperation manages the offline operation state machine.
func handleOfflineOperation(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.Instance,
) task.Result {
	condition := runtime.GetOfflineCondition(store)

	// If no condition exists, recover state from PD first
	if condition == nil {
		recoveredCondition := recoverOfflineStateFromPD(ctx, state, store)
		if recoveredCondition != nil {
			// Update the recovered condition and continue with normal state machine
			updateOfflinedCondition(state, store, recoveredCondition)
			condition = recoveredCondition
		} else {
			// PD state indicates we should start fresh offline operation
			return startOfflineOperation(ctx, state, store)
		}
	}

	switch condition.Reason {
	case v1alpha1.ReasonOfflineProcessing:
		return handleProcessingState(ctx, state, store)
	case v1alpha1.ReasonOfflineCompleted:
		return task.Complete().With("store offline operation is completed")
	case v1alpha1.ReasonOfflineFailed:
		return handleFailedState(ctx, state, store)
	case v1alpha1.ReasonOfflineCancelled:
		// This case is reached if spec.offline is set to true for a store
		// that was previously canceled. It restarts the offline process.
		return startOfflineOperation(ctx, state, store)
	default:
		return task.Fail().With("unknown offline condition reason: %s", condition.Reason)
	}
}

// handleOfflineCancellation handles the cancellation of offline operations when offline=false.
func handleOfflineCancellation(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.Instance,
) task.Result {
	condition := runtime.GetOfflineCondition(store)
	// If no offline condition exists, try to cancel anyway in case PD has ongoing operations
	if condition == nil {
		return cancelOfflineOperation(ctx, state, store)
	}

	switch condition.Reason {
	case v1alpha1.ReasonOfflineCompleted:
		// Cannot cancel completed operations
		return task.Complete().With("cannot cancel completed offline operation")
	case v1alpha1.ReasonOfflineCancelled:
		// Already canceled
		return task.Complete().With("offline operation already canceled")
	case v1alpha1.ReasonOfflineProcessing, v1alpha1.ReasonOfflineFailed:
		// Cancel the operation
		return cancelOfflineOperation(ctx, state, store)
	default:
		return task.Fail().With("unknown offline condition reason: %s", condition.Reason)
	}
}

// startOfflineOperation initiates the offline operation by performing checks and calling PD API.
func startOfflineOperation(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.Instance,
) task.Result {
	logger := logr.FromContextOrDiscard(ctx).WithValues("instance", store.GetName())
	if state.StoreNotExists() {
		// Store doesn't exist, mark as completed
		logger.Info("Store does not exist, completing offline operation")
		updateOfflinedCondition(state, store, newOfflinedCondition(
			v1alpha1.ReasonOfflineCompleted,
			"Store does not exist, offline operation completed",
			metav1.ConditionTrue,
		))
		return task.Complete().With("store does not exist, offline operation completed")
	}

	if state.GetPDClient() == nil {
		updateOfflinedCondition(state, store, newOfflinedCondition(
			v1alpha1.ReasonOfflineFailed,
			"PD client is not registered",
			metav1.ConditionFalse,
		))
		return task.Retry(RemovingWaitInterval).With("pd client is not registered")
	}

	storeID := state.GetStoreID()
	if storeID == "" {
		updateOfflinedCondition(state, store, newOfflinedCondition(
			v1alpha1.ReasonOfflineFailed,
			"Store ID is empty",
			metav1.ConditionFalse,
		))
		return task.Retry(RemovingWaitInterval).With("store ID is empty")
	}

	logger.Info("Calling PD DeleteStore API", "storeID", storeID)
	if err := state.GetPDClient().Underlay().DeleteStore(ctx, storeID); err != nil {
		// Transition to Failed state and record error in condition message
		updateOfflinedCondition(state, store, newOfflinedCondition(
			v1alpha1.ReasonOfflineFailed,
			fmt.Sprintf("Failed to call PD DeleteStore API: %v", err),
			metav1.ConditionFalse,
		))
		// Return a retryable error to re-trigger the failed state handling
		return task.Retry(RemovingWaitInterval).With("failed to call PD DeleteStore API, will retry")
	}

	// Transition to Processing state
	logger.Info("Store offline operation transitioned to Processing state")
	updateOfflinedCondition(state, store, newOfflinedCondition(
		v1alpha1.ReasonOfflineProcessing,
		"Store offline operation is Processing, data migration is in progress",
		metav1.ConditionFalse,
	))
	// Note: We don't set store state to Removing here. Instead, we honor the store state
	// from PD, which will automatically transition from Serving -> Removing -> Removed
	// after calling DeleteStore API. This avoids state conflicts between operator and PD.
	return task.Retry(RemovingWaitInterval).With("store offline operation is processing")
}

func handleProcessingState(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.Instance,
) task.Result {
	logger := logr.FromContextOrDiscard(ctx).WithValues("instance", store.GetName())
	if state.StoreNotExists() || state.GetStoreState() == v1alpha1.StoreStateRemoved {
		logger.Info("Store has been removed from PD, completing offline operation")
		updateOfflinedCondition(state, store, newOfflinedCondition(
			v1alpha1.ReasonOfflineCompleted,
			"Store state is Removed, offline operation completed",
			metav1.ConditionTrue,
		))
		return task.Complete().With("store offline operation completed")
	}
	return task.Retry(RemovingWaitInterval).With("waiting for store to be removed, current state: %s", state.GetStoreState())
}

// handleFailedState processes the Failed state and directly retries.
func handleFailedState(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.Instance,
) task.Result {
	logger := logr.FromContextOrDiscard(ctx).WithValues("instance", store.GetName())

	// Directly retry by restarting the offline operation
	logger.Info("Store offline operation retrying from failed state")
	return startOfflineOperation(ctx, state, store)
}

// cancelOfflineOperation cancels an ongoing offline operation.
func cancelOfflineOperation(
	ctx context.Context,
	state StoreOfflineReconcileContext,
	store runtime.Instance,
) task.Result {
	logger := logr.FromContextOrDiscard(ctx).WithValues("instance", store.GetName())
	if state.GetPDClient() == nil {
		return task.Fail().With("pd client is not registered")
	}

	if state.StoreNotExists() {
		updateOfflinedCondition(state, store, newOfflinedCondition(
			v1alpha1.ReasonOfflineCompleted,
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
			condition := runtime.GetOfflineCondition(store)
			var reason string
			if condition != nil {
				reason = condition.Reason
			} else {
				// Default reason when condition is nil (status lost scenario)
				reason = v1alpha1.ReasonOfflineProcessing
			}
			status := metav1.ConditionFalse
			updateOfflinedCondition(state, store, newOfflinedCondition(
				reason,
				errMsg,
				status,
			))
			return task.Fail().With(errMsg)
		}
	}

	// Update condition to canceled
	logger.Info("Store offline operation canceled")
	updateOfflinedCondition(state, store, newOfflinedCondition(
		v1alpha1.ReasonOfflineCancelled,
		"Store offline operation has been canceled",
		metav1.ConditionFalse,
	))

	return task.Complete().With("offline operation canceled")
}

func updateOfflinedCondition(
	state StoreOfflineReconcileContext,
	store runtime.Instance,
	condition *metav1.Condition,
) {
	condition.ObservedGeneration = store.GetGeneration()
	runtime.SetOfflineCondition(store, condition)
	state.SetStatusChanged()
}

func newOfflinedCondition(reason, message string, status metav1.ConditionStatus) *metav1.Condition {
	return &metav1.Condition{
		Type:               v1alpha1.StoreOfflinedConditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}
}
