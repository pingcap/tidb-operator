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

package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/action"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/updater"
	"github.com/pingcap/tidb-operator/pkg/updater/policy"
	"github.com/pingcap/tidb-operator/pkg/utils/task/v3"
	"github.com/pingcap/tidb-operator/pkg/utils/tracker"
)

const (
	defaultUpdateWaitTime = time.Second * 30
)

// TaskUpdater is a task to scale or update TiFlash instances when spec of TiFlashGroup is changed.
func TaskUpdater(state *ReconcileContext, c client.Client, t tracker.Tracker[*v1alpha1.TiFlashGroup, *v1alpha1.TiFlash]) task.Task {
	return task.NameTaskFunc("TiFlashUpdater", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		tfg := state.TiFlashGroup()

		checker := action.NewUpgradeChecker[scope.TiFlashGroup](c, state.Cluster(), logger)

		if needVersionUpgrade(tfg) && !checker.CanUpgrade(ctx, tfg) {
			return task.Retry(defaultUpdateWaitTime).With("wait until preconditions of upgrading is met")
		}

		// Get current state
		updateRevision, _, _ := state.Revision()
		tfs := state.Slice()
		currentReplicas := len(tfs)
		desiredReplicas := int(*tfg.Spec.Replicas)

		// Count current online (non-offline) instances
		var onlineInstances []*runtime.TiFlash
		for _, tf := range tfs {
			if !tf.Spec.Offline {
				onlineInstances = append(onlineInstances, tf)
			}
		}
		currentOnlineReplicas := len(onlineInstances)

		// Handle different scenarios based on replica changes
		switch {
		case desiredReplicas > currentOnlineReplicas:
			// scale out or cancellation scenario
			return handleScaleOutOrCancellation(ctx, state, c, t, updateRevision, tfs, desiredReplicas)
		case desiredReplicas < currentOnlineReplicas:
			// scale in scenario (start offline process)
			return handleScaleIn(ctx, state, c, t, updateRevision, tfs, desiredReplicas)
		case desiredReplicas == currentReplicas && desiredReplicas == currentOnlineReplicas:
			// No scaling needed, handle normal updates and offline state management
			fallthrough
		default:
			// Handle offline state management
			return handleNormalUpdates(ctx, state, c, t, updateRevision, tfs)
		}
	})
}

func needVersionUpgrade(fg *v1alpha1.TiFlashGroup) bool {
	return fg.Spec.Template.Spec.Version != fg.Status.Version && fg.Status.Version != ""
}

func TiFlashNewer(fg *v1alpha1.TiFlashGroup, rev string) updater.NewFactory[*runtime.TiFlash] {
	return updater.NewFunc[*runtime.TiFlash](func() *runtime.TiFlash {
		spec := fg.Spec.Template.Spec.DeepCopy()

		tiflash := &v1alpha1.TiFlash{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: fg.Namespace,
				// Name will be allocated by updater.AllocateName
				Labels:      coreutil.InstanceLabels[scope.TiFlashGroup](fg, rev),
				Annotations: coreutil.InstanceAnnotations[scope.TiFlashGroup](fg),
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(fg, v1alpha1.SchemeGroupVersion.WithKind("TiFlashGroup")),
				},
			},
			Spec: v1alpha1.TiFlashSpec{
				Cluster:             fg.Spec.Cluster,
				Features:            fg.Spec.Features,
				Subdomain:           HeadlessServiceName(fg.Name),
				TiFlashTemplateSpec: *spec,
				// New instances start with offline = false (online)
				Offline: false,
			},
		}

		return runtime.FromTiFlash(tiflash)
	})
}

// handleNormalUpdates processes regular updates and manages offline state transitions
func handleNormalUpdates(ctx context.Context, state *ReconcileContext, c client.Client, t tracker.Tracker[*v1alpha1.TiFlashGroup, *v1alpha1.TiFlash], updateRevision string, tfs []*runtime.TiFlash) task.Result {
	logger := logr.FromContextOrDiscard(ctx)
	tfg := state.TiFlashGroup()

	// Check for instances that completed offline process and can be deleted
	var instancesToDelete []*runtime.TiFlash
	var normalInstances []*runtime.TiFlash

	for _, tf := range tfs {
		if tf.Spec.Offline {
			// Check if offline process is completed
			condition := v1alpha1.GetOfflineCondition(tf.Status.Conditions)
			if condition != nil && condition.Reason == v1alpha1.OfflineReasonCompleted {
				instancesToDelete = append(instancesToDelete, tf)
				logger.Info("instance offline completed, ready for deletion", "instance", tf.Name)
			}
		} else {
			normalInstances = append(normalInstances, tf)
		}
	}

	// Delete instances that completed offline process
	if len(instancesToDelete) > 0 {
		for _, instance := range instancesToDelete {
			if err := c.Delete(ctx, instance.To()); err != nil {
				logger.Error(err, "failed to delete offline completed instance", "instance", instance.Name)
				return task.Fail().With("failed to delete offline completed instance %s: %w", instance.Name, err)
			}
			logger.Info("deleted instance after offline completion", "instance", instance.Name)
		}
		return task.Wait().With("waiting for instance deletion to complete")
	}

	// Handle normal updates for non-offline instances
	var topos []v1alpha1.ScheduleTopology
	for _, p := range tfg.Spec.SchedulePolicies {
		switch p.Type {
		case v1alpha1.SchedulePolicyTypeEvenlySpread:
			topos = p.EvenlySpread.Topologies
		}
	}

	topoPolicy, err := policy.NewTopologyPolicy(topos, updateRevision, normalInstances...)
	if err != nil {
		return task.Fail().With("invalid topo policy: %w", err)
	}

	allocator := t.Track(tfg, state.InstanceSlice()...)
	wait, err := updater.New[runtime.TiFlashTuple]().
		WithInstances(normalInstances...).
		WithDesired(len(normalInstances)).
		WithClient(c).
		WithMaxSurge(0).
		WithMaxUnavailable(1).
		WithRevision(updateRevision).
		WithNewFactory(TiFlashNewer(tfg, updateRevision)).
		WithAddHooks(
			updater.AllocateName[*runtime.TiFlash](allocator),
			topoPolicy,
		).
		WithDelHooks(topoPolicy).
		WithUpdateHooks(topoPolicy).
		Build().
		Do(ctx)

	if err != nil {
		return task.Fail().With("cannot update instances: %w", err)
	}
	if wait {
		return task.Wait().With("wait for all instances ready")
	}

	return task.Complete().With("all instances are synced")
}

// handleScaleOutOrCancellation processes scale out requests and cancellation logic
func handleScaleOutOrCancellation(ctx context.Context, state *ReconcileContext, c client.Client, t tracker.Tracker[*v1alpha1.TiFlashGroup, *v1alpha1.TiFlash], updateRevision string, tfs []*runtime.TiFlash, desiredReplicas int) task.Result {
	logger := logr.FromContextOrDiscard(ctx)

	// Count stable instances (not offline and not being deleted)
	var stableInstances []*runtime.TiFlash
	var offlineInstances []*runtime.TiFlash

	for _, tf := range tfs {
		if tf.Spec.Offline {
			// Check offline condition to determine if it's active
			condition := v1alpha1.GetOfflineCondition(tf.Status.Conditions)
			if condition != nil && condition.Reason == v1alpha1.OfflineReasonActive {
				offlineInstances = append(offlineInstances, tf)
			}
		} else {
			stableInstances = append(stableInstances, tf)
		}
	}

	stableCount := len(stableInstances)
	offlineCount := len(offlineInstances)

	logger.Info("scale out or cancellation detected",
		"desiredReplicas", desiredReplicas,
		"stableInstances", stableCount,
		"offlineInstances", offlineCount)

	if offlineCount > 0 && desiredReplicas > stableCount {
		// Cancellation scenario: need to rescue some offline instances
		numToRescue := desiredReplicas - stableCount
		if numToRescue > offlineCount {
			numToRescue = offlineCount // Can't rescue more than what's available
		}

		logger.Info("canceling offline operations", "numToRescue", numToRescue)

		// Cancel offline operations by setting spec.offline = false
		for i := 0; i < numToRescue && i < len(offlineInstances); i++ {
			instance := offlineInstances[i]
			instanceObj := instance.To()

			// Update spec.offline to false
			instanceObj.Spec.Offline = false
			if err := c.Update(ctx, instanceObj); err != nil {
				logger.Error(err, "failed to cancel offline operation", "instance", instance.Name)
				return task.Fail().With("failed to cancel offline operation for instance %s: %w", instance.Name, err)
			}
			logger.Info("canceled offline operation", "instance", instance.Name)
		}

		// If we still need more instances after canceling, create new ones
		remainingNeeded := desiredReplicas - stableCount - numToRescue
		if remainingNeeded > 0 {
			return handleScaleOut(ctx, state, c, t, updateRevision, stableCount+numToRescue, remainingNeeded)
		}

		return task.Wait().With("waiting for offline cancellation to complete")
	}

	// Pure scale out: create new instances
	if desiredReplicas > stableCount {
		return handleScaleOut(ctx, state, c, t, updateRevision, stableCount, desiredReplicas-stableCount)
	}

	// If we reach here, it means desiredReplicas == stableCount after cancellation
	return task.Wait().With("no scaling action needed")
}

// handleScaleOut creates new instances for scale out operations
func handleScaleOut(ctx context.Context, state *ReconcileContext, c client.Client, t tracker.Tracker[*v1alpha1.TiFlashGroup, *v1alpha1.TiFlash], updateRevision string, currentStable int, numToAdd int) task.Result {
	logger := logr.FromContextOrDiscard(ctx)
	tfg := state.TiFlashGroup()

	// Get topology policies
	var topos []v1alpha1.ScheduleTopology
	for _, p := range tfg.Spec.SchedulePolicies {
		switch p.Type {
		case v1alpha1.SchedulePolicyTypeEvenlySpread:
			topos = p.EvenlySpread.Topologies
		}
	}

	topoPolicy, err := policy.NewTopologyPolicy(topos, updateRevision, state.Slice()...)
	if err != nil {
		return task.Fail().With("invalid topo policy: %w", err)
	}

	allocator := t.Track(tfg, state.InstanceSlice()...)
	wait, err := updater.New[runtime.TiFlashTuple]().
		WithInstances(state.Slice()...).
		WithDesired(currentStable+numToAdd).
		WithClient(c).
		WithMaxSurge(numToAdd).
		WithMaxUnavailable(0).
		WithRevision(updateRevision).
		WithNewFactory(TiFlashNewer(tfg, updateRevision)).
		WithAddHooks(
			updater.AllocateName[*runtime.TiFlash](allocator),
			topoPolicy,
		).
		Build().
		Do(ctx)

	if err != nil {
		return task.Fail().With("cannot scale up instances: %w", err)
	}
	if wait {
		return task.Wait().With("wait for new instances ready")
	}

	logger.Info("scale out completed", "added", numToAdd)
	return task.Complete().With("scale out completed")
}

// handleScaleIn initiates the offline process for selected instances
func handleScaleIn(ctx context.Context, state *ReconcileContext, c client.Client, t tracker.Tracker[*v1alpha1.TiFlashGroup, *v1alpha1.TiFlash], updateRevision string, tfs []*runtime.TiFlash, desiredReplicas int) task.Result {
	logger := logr.FromContextOrDiscard(ctx)
	tfg := state.TiFlashGroup()

	currentReplicas := len(tfs)
	numToOffline := currentReplicas - desiredReplicas

	logger.Info("scale in detected",
		"currentReplicas", currentReplicas,
		"desiredReplicas", desiredReplicas,
		"numToOffline", numToOffline)

	// Find instances that are not already offline
	var candidatesForOffline []*runtime.TiFlash
	var alreadyOffline []*runtime.TiFlash

	for _, tf := range tfs {
		if tf.Spec.Offline {
			alreadyOffline = append(alreadyOffline, tf)
		} else {
			candidatesForOffline = append(candidatesForOffline, tf)
		}
	}

	alreadyOfflineCount := len(alreadyOffline)
	stillNeedToOffline := numToOffline - alreadyOfflineCount

	if stillNeedToOffline <= 0 {
		// All required instances are already offline, just wait
		return task.Wait().With(fmt.Sprintf("waiting for %d offline instances to complete", alreadyOfflineCount))
	}

	// Select instances for offline using preference policies
	var topos []v1alpha1.ScheduleTopology
	for _, p := range tfg.Spec.SchedulePolicies {
		switch p.Type {
		case v1alpha1.SchedulePolicyTypeEvenlySpread:
			topos = p.EvenlySpread.Topologies
		}
	}

	topoPolicy, err := policy.NewTopologyPolicy(topos, updateRevision, tfs...)
	if err != nil {
		return task.Fail().With("invalid topo policy: %w", err)
	}

	// Create scale-in selector with preference policies
	scaleInSelector := updater.NewSelector(
		updater.PreferAnnotatedForDeletion[*runtime.TiFlash](), // Prioritize annotated instances
		updater.PreferNotOfflining[*runtime.TiFlash](),         // Avoid instances already being offlined
		updater.PreferHealthyForScaleIn[*runtime.TiFlash](),    // Prefer healthy instances for stable data migration
		topoPolicy, // Respect topology constraints
	)

	// Select instances to offline
	var instancesToOffline []*runtime.TiFlash
	remaining := candidatesForOffline

	for i := 0; i < stillNeedToOffline && len(remaining) > 0; i++ {
		selectedName := scaleInSelector.Choose(remaining)

		// Find and remove selected instance from remaining
		for j, instance := range remaining {
			if instance.Name == selectedName {
				instancesToOffline = append(instancesToOffline, instance)
				// Remove from remaining list
				remaining = append(remaining[:j], remaining[j+1:]...)
				break
			}
		}
	}

	// Set spec.offline = true for selected instances
	for _, instance := range instancesToOffline {
		instanceObj := instance.To()
		instanceObj.Spec.Offline = true

		if err := c.Update(ctx, instanceObj); err != nil {
			logger.Error(err, "failed to set instance offline", "instance", instance.Name)
			return task.Fail().With("failed to set instance offline %s: %w", instance.Name, err)
		}

		logger.Info("initiated offline operation", "instance", instance.Name)
	}

	return task.Wait().With(fmt.Sprintf("offline process started for %d instances", len(instancesToOffline)))
}
