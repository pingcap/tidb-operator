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

	"github.com/go-logr/logr"
)

type Actor interface {
	ScaleOut(ctx context.Context) error
	Update(ctx context.Context) error
	ScaleInUpdate(ctx context.Context) (unavailable bool, _ error)
	ScaleInOutdated(ctx context.Context) (unavailable bool, _ error)

	// delete all instances marked as defer deletion
	Cleanup(ctx context.Context) error
}

// CancelableActor extends the basic Actor interface with cancel scale-in capabilities.
// This interface is designed to be optional and only implemented by components that need
// cancel scale-in functionality (currently TiKV/TiFlash store instances).
//
// Components not implementing this interface will continue to work exactly as before,
// ensuring 100% backward compatibility.
type CancelableActor interface {
	Actor // Inherits all basic actor operations

	// CancelScaleIn attempts to cancel ongoing scale-in operations to match the target replica count.
	// It will rescue instances from offline state back to online state.
	//
	// Parameters:
	//   - ctx: Context for the operation
	//   - targetReplicas: The desired number of replicas after cancellation
	//
	// Returns:
	//   - canceled: The number of instances successfully rescued from scale-in
	//   - error: Any error that occurred during the cancellation process
	//
	// This method should only be called when the desired replica count increases
	// during an ongoing scale-in operation.
	CancelScaleIn(ctx context.Context, targetReplicas int) (canceled int, err error)

	// CanCancel indicates whether this actor supports and is ready to perform cancel operations.
	// This allows the executor to safely check capabilities before attempting cancellation.
	CanCancel() bool

	// CountOfflineInstances returns two counts:
	// - beingOffline: instances currently undergoing offline process but not yet completed
	// - offlineCompleted: instances that have completed offline and are no longer in the cluster
	// This provides all the information executor needs for scaling decisions.
	CountOfflineInstances() (beingOffline, offlineCompleted int)

	// CleanupCompletedOffline finds and deletes any instances that have completed the offline
	// process but are still present. This is crucial for handling race conditions during
	// scale-in cancellation where an instance might complete offline before it can be canceled.
	// It returns the number of instances cleaned up.
	CleanupCompletedOffline(ctx context.Context) (cleaned int, err error)
}

// Executor is an executor that updates the instances.
// TODO: return instance list after Do
type Executor interface {
	Do(ctx context.Context) (bool, error)
}

type executor struct {
	update              int
	outdated            int
	desired             int
	unavailableUpdate   int
	unavailableOutdated int
	maxSurge            int
	maxUnavailable      int

	act Actor
}

func NewExecutor(
	act Actor,
	update,
	outdated,
	desired,
	unavailableUpdate,
	unavailableOutdated,
	maxSurge,
	maxUnavailable int,
) Executor {
	return &executor{
		update:              update,
		outdated:            outdated,
		desired:             desired,
		unavailableUpdate:   unavailableUpdate,
		unavailableOutdated: unavailableOutdated,
		maxSurge:            maxSurge,
		maxUnavailable:      maxUnavailable,
		act:                 act,
	}
}

// countOnlineInstances calculates the actual number of online instances,
// excluding those that are currently being offline or have completed offline
func (ex *executor) countOnlineInstances() int {
	total := ex.update + ex.outdated

	// If the actor supports counting offline instances, subtract them
	if cancelableActor, ok := ex.act.(CancelableActor); ok && cancelableActor.CanCancel() {
		beingOffline, offlineCompleted := cancelableActor.CountOfflineInstances()
		return total - beingOffline - offlineCompleted
	}

	// For regular actors, all instances are considered online
	return total
}

// TODO: add scale in/out rate limit
//
//nolint:gocyclo // refactor if possible
func (ex *executor) Do(ctx context.Context) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx).WithName("Updater")

reconcile: // Label to allow restarting the entire reconciliation
	logger.Info("begin to update",
		"update", ex.update, "outdated", ex.outdated, "desired", ex.desired,
		"unavailableUpdate", ex.unavailableUpdate, "unavailableOutdated", ex.unavailableOutdated,
		"maxSurge", ex.maxSurge, "maxUnavailable", ex.maxUnavailable)
	// Track whether we've already attempted cancel in this execution
	cancelAttempted := false

	// Helper function to check if cancellation might be needed
	// This is needed because even when ex.update == ex.desired, we might still
	// need to cancel ongoing offline operations if desired count increased
	needCancelCheck := func() bool {
		if cancelableActor, ok := ex.act.(CancelableActor); ok && cancelableActor.CanCancel() {
			beingOffline, _ := cancelableActor.CountOfflineInstances()
			return beingOffline > 0 && ex.desired > ex.countOnlineInstances()
		}
		return false
	}

	// Main reconciliation loop - handles both normal scaling and cancel operations
	// Extended condition to also run when there are offline instances that might need rescue
	for ex.update != ex.desired || ex.outdated != 0 || needCancelCheck() {
		// Calculate actual instances considering offline state for TiKV/TiFlash
		// For regular actors, this equals ex.update + ex.outdated
		// For CancelableActor (TiKV/TiFlash), this excludes offline instances
		onlineActual := ex.countOnlineInstances()
		actual := ex.update + ex.outdated // Total instances (including offline)
		available := actual - ex.unavailableUpdate - ex.unavailableOutdated
		maximum := ex.desired + min(ex.maxSurge, ex.outdated)
		minimum := ex.desired - ex.maxUnavailable

		logger.Info("loop",
			"update", ex.update, "outdated", ex.outdated, "desired", ex.desired,
			"unavailableUpdate", ex.unavailableUpdate, "unavailableOutdated", ex.unavailableOutdated,
			"onlineActual", onlineActual, "actual", actual, "available", available, "maximum", maximum, "minimum", minimum)

		// Check for cancel scale-in scenario first (when onlineActual < desired)
		// This handles cases where user increases replicas during an ongoing scale-down
		// Only attempt cancel once per execution
		// Extended condition to also trigger cancellation when there are offline instances
		// that need to be rescued even when ex.update == ex.desired
		if (onlineActual < ex.desired || needCancelCheck()) && !cancelAttempted {
			if cancelableActor, ok := ex.act.(CancelableActor); ok && cancelableActor.CanCancel() {
				logger.Info("cancel scale in",
					"onlineActual", onlineActual, "desired", ex.desired)
				// Attempt to rescue offline instances before creating new ones
				canceled, err := cancelableActor.CancelScaleIn(ctx, ex.desired)
				cancelAttempted = true
				if err != nil {
					return false, fmt.Errorf("failed to cancel scale-in: %w", err)
				}

				if canceled > 0 {
					logger.Info("cancel scale in success", "canceled", canceled)
					// Successfully rescued some instances
					// DO NOT update internal counters here - this was the bug!
					// The next reconcile cycle will read the actual state from Kubernetes
					// and re-evaluate based on real instance states.
					//
					// BUG FIX: Previously this code did:
					// ex.update += canceled
					// ex.unavailableUpdate += canceled
					// This caused ex.update to increase, triggering immediate re-scale-in
					// in the same reconcile loop, marking the same instance offline again.
					// Return immediately to allow next reconcile to see the updated state
					return false, nil
				}
			}

			// No rescue possible or needed, proceed with normal scale-out
		}

		switch {
		case actual < maximum:
			logger.Info("scale out")
			if err := ex.act.ScaleOut(ctx); err != nil {
				return false, err
			}
			ex.update += 1
			ex.unavailableUpdate += 1

		case actual == maximum:
			if ex.update < ex.desired {
				// update will always prefer unavailable one so available will not be changed if there are
				// unavailable and outdated instances
				if ex.unavailableOutdated > 0 {
					logger.Info("update unavailable outdated")
					if err := ex.act.Update(ctx); err != nil {
						return false, err
					}
					ex.outdated -= 1
					ex.unavailableOutdated -= 1
					ex.update += 1
					ex.unavailableUpdate += 1
				} else {
					// DON'T decrease available if available is less than minimum
					if available <= minimum {
						logger.Info("wait because available is less than minimum",
							"available", available, "minimum", minimum)
						return true, nil
					}

					logger.Info("update available outdated")
					if err := ex.act.Update(ctx); err != nil {
						return false, err
					}
					ex.outdated -= 1
					ex.update += 1
					ex.unavailableUpdate += 1
				}
			} else {
				// => ex.update + ex.outdated == ex.desired + min(ex.maxSurge, ex.outdated) and ex.update >= ex.desired
				// => ex.outdated <= min(ex.maxSurge, ex.outdated)
				// => ex.outdated <= ex.maxSurge
				// => ex.outdated = min(ex.maxSurge, ex.outdated)
				// => ex.update + ex.outdated >= ex.desired + ex.outdated
				// => ex.update == ex.desired
				// => ex.outdated != 0 (ex.update != ex.desired || ex.outdated != 0 in for loop condition)
				if available <= minimum {
					logger.Info("wait because available is less than minimum",
						"available", available, "minimum", minimum)
					return true, nil
				}

				logger.Info("scale in outdated")
				unavailable, err := ex.act.ScaleInOutdated(ctx)
				if err != nil {
					return false, err
				}
				// scale in may not choose an unavailable outdated so just descrease the outdated
				// and assume we always choose an available outdated.
				// And then wait if next available is less than minimum
				if unavailable {
					ex.outdated -= 1
					ex.unavailableOutdated -= 1
				} else {
					ex.outdated -= 1
				}
			}
		case actual > maximum:
			// For TiKV/TiFlash: check if we're already offlining enough instances
			// Allow parallel offline: continue marking instances offline until we have enough
			if cancelableActor, ok := ex.act.(CancelableActor); ok && cancelableActor.CanCancel() {
				beingOffline, offlineCompleted := cancelableActor.CountOfflineInstances()

				// Calculate current effective instances (what's actually still in the cluster)
				currentEffectiveInstances := (ex.update + ex.outdated) - offlineCompleted

				// Calculate how many more instances we need to start offlining
				stillNeedToStartOfflining := currentEffectiveInstances - ex.desired - beingOffline

				// Ensure we don't get negative values
				if stillNeedToStartOfflining <= 0 {
					stillNeedToStartOfflining = 0
				}

				logger.Info("scale in with cancelable actor",
					"beingOffline", beingOffline, "offlineCompleted", offlineCompleted,
					"currentEffectiveInstances", currentEffectiveInstances,
					"stillNeedToStartOfflining", stillNeedToStartOfflining)

				// If we don't need to start offlining more instances, but some are still in progress, wait for completion.
				if stillNeedToStartOfflining == 0 && beingOffline > 0 {
					logger.Info("wait because some instances are still being offlined",
						"beingOffline", beingOffline, "offlineCompleted", offlineCompleted)
					return true, nil
				}
				// Continue to mark more instances offline
			}

			// Scale in op may choose an available instance.
			// For TiKV/TiFlash with parallel offline support: process one instance per reconcile
			checkAvail := false
			if ex.update > ex.desired {
				logger.Info("scale in update")
				unavailable, err := ex.act.ScaleInUpdate(ctx)
				if err != nil {
					return false, err
				}
				if unavailable {
					ex.update -= 1
					ex.unavailableUpdate -= 1
				} else {
					ex.update -= 1
					available -= 1
					checkAvail = true
				}

				// For CancelableActor (TiKV/TiFlash): return immediately after processing one instance
				// This ensures we process only one instance per reconcile, enabling parallel offline
				if _, ok := ex.act.(CancelableActor); ok {
					logger.Info("return because of cancelable actor")
					return false, nil
				}
			} else {
				// ex.update + ex.outdated > ex.desired + min(ex.maxSurge, ex.outdated) and ex.update <= ex.desired
				// => ex.outdated > min(ex.maxSurge, ex.outdated)
				// => ex.outdated > 0
				logger.Info("scale in outdated")
				unavailable, err := ex.act.ScaleInOutdated(ctx)
				if err != nil {
					return false, err
				}
				if unavailable {
					ex.outdated -= 1
					ex.unavailableOutdated -= 1
				} else {
					ex.outdated -= 1
					available -= 1
					checkAvail = true
				}

				// For CancelableActor (TiKV/TiFlash): return immediately after processing one instance
				if _, ok := ex.act.(CancelableActor); ok {
					logger.Info("return because of cancelable actor")
					return false, nil
				}
			}
			// Wait if available is less than minimum
			if checkAvail && available <= minimum {
				logger.Info("wait because available is less than minimum",
					"available", available, "minimum", minimum)
				return true, nil
			}
		}
	}

	// After the main loop, perform a final check for any zombie instances left over from race conditions.
	if cancelableActor, ok := ex.act.(CancelableActor); ok && cancelableActor.CanCancel() {
		cleaned, err := cancelableActor.CleanupCompletedOffline(ctx)
		if err != nil {
			return false, fmt.Errorf("failed to clean up completed offline instances: %w", err)
		}
		if cleaned > 0 {
			logger.Info("restarting reconciliation after cleaning up completed offline instances", "cleanedCount", cleaned)
			// State has changed significantly, restart the entire reconciliation process.
			ex.update -= cleaned
			// The cleaned instances might have been unavailable.
			// We let the next full reconciliation loop recalculate this accurately.
			goto reconcile
		}
	}

	if ex.unavailableUpdate > 0 {
		// wait until update are all available
		logger.Info("wait because unavailable update is not zero",
			"unavailableUpdate", ex.unavailableUpdate)
		return true, nil
	}

	logger.Info("cleanup")
	if err := ex.act.Cleanup(ctx); err != nil {
		return false, err
	}
	return false, nil
}
