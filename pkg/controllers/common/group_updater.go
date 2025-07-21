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
	"fmt"

	"github.com/go-logr/logr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/updater"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
)

// storeGroupState represents the current state of a store group.
type storeGroupState string

const (
	stateStable               storeGroupState = "Stable"
	stateScalingUp            storeGroupState = "ScalingUp"
	stateCancellingScaleDown  storeGroupState = "CancellingScaleDown"
	stateScalingDown          storeGroupState = "ScalingDown"
	stateAwaitingNodeDeletion storeGroupState = "AwaitingNodeDeletion"
)

// TaskStoreGroupUpdater is a task to scale or update store instances when spec of store group is changed.
func TaskStoreGroupUpdater[
	SG scope.Group[O, G],
	O client.Object,
	G runtime.Group,
	SI runtime.StoreInstance,
](
	ctx context.Context,
	obj O,
	instances []SI,
	cli client.Client,
	executor updater.Executor,
	scaleInSelector updater.Selector[SI],
) task.Result {
	// Calculate the current state
	group := scope.From[SG](obj)
	state := calculateGroupState(group, instances)

	// Handle state transitions
	switch state {
	case stateStable:
		return handleStableState(ctx, cli, instances, executor)
	case stateScalingUp:
		return handleScaleUp(ctx, group, instances, executor)
	case stateCancellingScaleDown:
		return handleCancelScaleDown(ctx, cli, group, instances, executor)
	case stateScalingDown:
		return handleScaleDown(ctx, cli, group, instances, scaleInSelector)
	case stateAwaitingNodeDeletion:
		return handleInstanceDeletion(ctx, cli, instances)
	default:
		return task.Fail().With("unknown group state: %s", state)
	}
}

// calculateGroupState is a pure function that determines the current state of a store group
func calculateGroupState[SI runtime.StoreInstance](group runtime.Group, instances []SI) storeGroupState {
	desiredReplicas := int(group.Replicas())

	// Classify instances by their status
	var (
		onlineInstances, offlineActiveInstances, offlineCompletedInstances []SI
	)
	for _, instance := range instances {
		if instance.IsOffline() {
			condition := instance.GetOfflineCondition()
			if condition != nil && condition.Reason == v1alpha1.OfflineReasonCompleted {
				offlineCompletedInstances = append(offlineCompletedInstances, instance)
			} else if condition != nil && condition.Reason == v1alpha1.OfflineReasonActive {
				offlineActiveInstances = append(offlineActiveInstances, instance)
			}
		} else {
			onlineInstances = append(onlineInstances, instance)
		}
	}

	onlineCount := len(onlineInstances)
	offlineActiveCount := len(offlineActiveInstances)
	offlineCompletedCount := len(offlineCompletedInstances)

	// State decision logic
	if offlineCompletedCount > 0 {
		return stateAwaitingNodeDeletion
	}

	if desiredReplicas > onlineCount {
		// Need more instances
		if offlineActiveCount > 0 {
			// We have offline instances that can be rescued
			return stateCancellingScaleDown
		} else {
			// Pure scale up
			return stateScalingUp
		}
	}

	if desiredReplicas < onlineCount {
		// Need fewer instances - start scale down
		return stateScalingDown
	}

	// desiredReplicas == onlineCount
	if offlineActiveCount > 0 {
		// We have the right number of online instances, but some are still offline
		// This might be a transition state, treat as stable and let normal updates handle it
		return stateStable
	}

	return stateStable
}

// handleStableState handles stable state - performs rolling updates and cleans up offline instances
func handleStableState[SI runtime.StoreInstance](
	ctx context.Context,
	cli client.Client,
	instances []SI,
	executor updater.Executor,
) task.Result {
	// First, try to delete any completed offline instances
	if result := deleteCompletedOfflineInstances(ctx, cli, instances); result != nil {
		return *result
	}

	// Then perform rolling update on normal instances
	return performRollingUpdate(ctx, instances, executor)
}

// handleScaleUp handles pure scale up operations
func handleScaleUp[SI runtime.StoreInstance](
	ctx context.Context,
	group runtime.Group,
	instances []SI,
	executor updater.Executor,
) task.Result {
	logger := logr.FromContextOrDiscard(ctx).WithValues("group", group.GetName())

	desiredReplicas := int(group.Replicas())
	currentReplicas := len(instances)
	numToAdd := desiredReplicas - currentReplicas

	logger.Info("scaling up", "currentReplicas", currentReplicas, "desiredReplicas", desiredReplicas, "numToAdd", numToAdd)

	wait, err := executor.Do(ctx)
	if err != nil {
		return task.Fail().With("cannot scale out instances: %v", err)
	}
	if wait {
		return task.Wait().With("wait for new instances ready")
	}

	logger.Info("scale out completed", "added", numToAdd)
	return task.Complete().With("scale out completed")
}

// handleCancelScaleDown handles cancellation of scale down operations
func handleCancelScaleDown[SI runtime.StoreInstance](
	ctx context.Context,
	cli client.Client,
	group runtime.Group,
	instances []SI,
	executor updater.Executor,
) task.Result {
	logger := logr.FromContextOrDiscard(ctx)

	desiredReplicas := int(group.Replicas())

	// Find offline instances that can be rescued
	var onlineInstances []SI
	var offlineActiveInstances []SI

	for _, instance := range instances {
		if instance.IsOffline() {
			condition := instance.GetOfflineCondition()
			if condition != nil && condition.Reason == v1alpha1.OfflineReasonActive {
				offlineActiveInstances = append(offlineActiveInstances, instance)
			}
		} else {
			onlineInstances = append(onlineInstances, instance)
		}
	}

	onlineCount := len(onlineInstances)
	offlineActiveCount := len(offlineActiveInstances)

	// Calculate how many to rescue
	numToRescue := desiredReplicas - onlineCount
	if numToRescue > offlineActiveCount {
		numToRescue = offlineActiveCount
	}

	logger.Info("cancelling scale down",
		"onlineCount", onlineCount,
		"offlineActiveCount", offlineActiveCount,
		"numToRescue", numToRescue)

	// Cancel offline operations by setting `spec.offline = false`
	for i := 0; i < numToRescue && i < len(offlineActiveInstances); i++ {
		instance := offlineActiveInstances[i]
		instance.SetOffline(false)
		if err := cli.Update(ctx, instance); err != nil {
			return task.Fail().With("failed to cancel offline operation for instance %s: %v", instance.GetName(), err)
		}
		logger.Info("canceled offline operation", "instance", instance.GetName())
	}

	// If we still need more instances after canceling, create new ones
	remainingNeeded := desiredReplicas - onlineCount - numToRescue
	if remainingNeeded > 0 {
		wait, err := executor.Do(ctx)
		if err != nil {
			return task.Fail().With("cannot scale out instances: %v", err)
		}
		if wait {
			return task.Wait().With("wait for new instances ready")
		}
	}

	return task.Wait().With("waiting for offline cancellation to complete")
}

// handleScaleDown handles scale down operations
func handleScaleDown[SI runtime.StoreInstance](
	ctx context.Context,
	cli client.Client,
	group runtime.Group,
	instances []SI,
	scaleInSelector updater.Selector[SI],
) task.Result {
	logger := logr.FromContextOrDiscard(ctx)

	desiredReplicas := int(group.Replicas())
	currentReplicas := len(instances)
	numToOffline := currentReplicas - desiredReplicas

	logger.Info("scale down detected",
		"currentReplicas", currentReplicas,
		"desiredReplicas", desiredReplicas,
		"numToOffline", numToOffline)

	// Find instances that are not already offline
	var candidatesForOffline []SI
	var alreadyOffline []SI

	for _, instance := range instances {
		if instance.IsOffline() {
			alreadyOffline = append(alreadyOffline, instance)
		} else {
			candidatesForOffline = append(candidatesForOffline, instance)
		}
	}

	alreadyOfflineCount := len(alreadyOffline)
	stillNeedToOffline := numToOffline - alreadyOfflineCount

	if stillNeedToOffline <= 0 {
		// All required instances are already offline, just wait
		return task.Wait().With(fmt.Sprintf("waiting for %d offline instances to complete", alreadyOfflineCount))
	}

	// Select instances to offline
	var instancesToOffline []SI
	remaining := candidatesForOffline

	for i := 0; i < stillNeedToOffline && len(remaining) > 0; i++ {
		selectedName := scaleInSelector.Choose(remaining)

		// Find and remove selected instance from remaining
		for j, instance := range remaining {
			if instance.GetName() == selectedName {
				instancesToOffline = append(instancesToOffline, instance)
				// Remove from remaining list
				remaining = append(remaining[:j], remaining[j+1:]...)
				break
			}
		}
	}

	// Set `spec.offline = true` for selected instances
	for _, instance := range instancesToOffline {
		instance.SetOffline(true)
		if err := cli.Update(ctx, instance); err != nil {
			return task.Fail().With("failed to set instance offline %s: %v", instance.GetName(), err)
		}

		logger.Info("initiated offline operation", "instance", instance.GetName())
	}

	return task.Wait().With(fmt.Sprintf("offline process started for %d instances", len(instancesToOffline)))
}

// handleInstanceDeletion handles deletion of completed offline instances.
func handleInstanceDeletion[SI runtime.StoreInstance](ctx context.Context, cli client.Client, instances []SI) task.Result {
	if result := deleteCompletedOfflineInstances(ctx, cli, instances); result != nil {
		return *result
	}

	// If we are in this state, it means there were completed offline instances.
	// After deletion, we should wait for the Kubernetes controller to observe the deletion.
	return task.Wait().With("waiting for instance deletion to complete")
}

// Helper functions

// deleteCompletedOfflineInstances deletes instances that have completed the offline process
func deleteCompletedOfflineInstances[SI runtime.StoreInstance](ctx context.Context, cli client.Client, instances []SI) *task.Result {
	logger := logr.FromContextOrDiscard(ctx)

	var instancesToDelete []SI
	for _, instance := range instances {
		if instance.IsOffline() {
			condition := instance.GetOfflineCondition()
			if condition != nil && condition.Reason == v1alpha1.OfflineReasonCompleted {
				instancesToDelete = append(instancesToDelete, instance)
				logger.Info("instance offline completed, ready for deletion", "instance", instance.GetName())
			}
		}
	}

	if len(instancesToDelete) > 0 {
		for _, instance := range instancesToDelete {
			// Convert runtime type back to v1alpha1 type for client operations
			var clientObj client.Object
			instanceAny := any(instance)
			if tikvInst, ok := instanceAny.(*runtime.TiKV); ok {
				clientObj = tikvInst.To()
			} else if tiflashInst, ok := instanceAny.(*runtime.TiFlash); ok {
				clientObj = tiflashInst.To()
			} else {
				clientObj = instance
			}

			if err := cli.Delete(ctx, clientObj); err != nil {
				logger.Error(err, "failed to delete offline completed instance", "instance", instance.GetName())
				result := task.Fail().With("failed to delete offline completed instance %s: %v", instance.GetName(), err)
				return &result
			}
			logger.Info("deleted instance after offline completion", "instance", instance.GetName())
		}
		result := task.Wait().With("waiting for instance deletion to complete")
		return &result
	}

	return nil
}

// performRollingUpdate performs rolling update on normal instances
func performRollingUpdate[SI runtime.StoreInstance](
	ctx context.Context,
	instances []SI,
	executor updater.Executor,
) task.Result {
	// Filter out offline instances
	var normalInstances []SI
	for _, instance := range instances {
		if !instance.IsOffline() {
			normalInstances = append(normalInstances, instance)
		}
	}

	wait, err := executor.Do(ctx)
	if err != nil {
		return task.Fail().With("cannot update instances: %w", err)
	}
	if wait {
		return task.Wait().With("wait for all instances ready")
	}

	return task.Complete().With("all instances are synced")
}
