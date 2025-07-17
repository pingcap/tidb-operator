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

// TaskUpdater is a task to scale or update TiKV instances when spec of TiKVGroup is changed.
func TaskUpdater(state *ReconcileContext, c client.Client, t tracker.Tracker[*v1alpha1.TiKVGroup, *v1alpha1.TiKV]) task.Task {
	return task.NameTaskFunc("TiKVUpdater", func(ctx context.Context) task.Result {
		logger := logr.FromContextOrDiscard(ctx)
		kvg := state.TiKVGroup()

		checker := action.NewUpgradeChecker[scope.TiKVGroup](c, state.Cluster(), logger)

		if needVersionUpgrade(kvg) && !checker.CanUpgrade(ctx, kvg) {
			return task.Retry(defaultUpdateWaitTime).With("wait until preconditions of upgrading is met")
		}

		// Get current state
		updateRevision, _, _ := state.Revision()
		kvs := state.Slice()
		currentReplicas := len(kvs)
		desiredReplicas := int(*kvg.Spec.Replicas)

		// Count current online (non-offline) instances
		var onlineInstances []*runtime.TiKV
		for _, kv := range kvs {
			if !kv.Spec.Offline {
				onlineInstances = append(onlineInstances, kv)
			}
		}
		currentOnlineReplicas := len(onlineInstances)

		// Handle different scenarios based on replica changes
		// No scaling needed, handle normal updates and offline state management
		if desiredReplicas == currentReplicas && desiredReplicas == currentOnlineReplicas {
			return handleNormalUpdates(ctx, state, c, t, updateRevision, kvs)
		}
		// scale out or cancellation scenario
		if desiredReplicas > currentOnlineReplicas {
			return handleScaleOutOrCancellation(ctx, state, c, t, updateRevision, kvs, desiredReplicas)
		}
		// scale in scenario (start offline process)
		if desiredReplicas < currentOnlineReplicas {
			return handleScaleIn(ctx, state, c, t, updateRevision, kvs, desiredReplicas)
		}
		// Handle offline state management
		return handleNormalUpdates(ctx, state, c, t, updateRevision, kvs)
	})
}

// handleNormalUpdates processes regular updates and manages offline state transitions
func handleNormalUpdates(ctx context.Context, state *ReconcileContext, c client.Client, t tracker.Tracker[*v1alpha1.TiKVGroup, *v1alpha1.TiKV], updateRevision string, kvs []*runtime.TiKV) task.Result {
	logger := logr.FromContextOrDiscard(ctx)
	kvg := state.TiKVGroup()

	// Check for instances that completed offline process and can be deleted
	var instancesToDelete []*runtime.TiKV
	var normalInstances []*runtime.TiKV

	for _, kv := range kvs {
		if kv.Spec.Offline {
			// Check if offline process is completed
			condition := v1alpha1.GetOfflineCondition(kv.Status.Conditions)
			if condition != nil && condition.Reason == v1alpha1.OfflineReasonCompleted {
				instancesToDelete = append(instancesToDelete, kv)
				logger.Info("instance offline completed, ready for deletion", "instance", kv.Name)
			}
		} else {
			normalInstances = append(normalInstances, kv)
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
	for _, p := range kvg.Spec.SchedulePolicies {
		if p.Type == v1alpha1.SchedulePolicyTypeEvenlySpread {
			topos = p.EvenlySpread.Topologies
		}
	}

	topoPolicy, err := policy.NewTopologyPolicy(topos, updateRevision, normalInstances...)
	if err != nil {
		return task.Fail().With("invalid topo policy: %w", err)
	}

	allocator := t.Track(kvg, state.InstanceSlice()...)
	wait, err := updater.New[runtime.TiKVTuple]().
		WithInstances(normalInstances...).
		WithDesired(len(normalInstances)).
		WithClient(c).
		WithMaxSurge(0).
		WithMaxUnavailable(1).
		WithRevision(updateRevision).
		WithNewFactory(TiKVNewer(kvg, updateRevision)).
		WithAddHooks(
			updater.AllocateName[*runtime.TiKV](allocator),
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
func handleScaleOutOrCancellation(ctx context.Context, state *ReconcileContext, c client.Client, t tracker.Tracker[*v1alpha1.TiKVGroup, *v1alpha1.TiKV], updateRevision string, kvs []*runtime.TiKV, desiredReplicas int) task.Result {
	logger := logr.FromContextOrDiscard(ctx)

	// Count stable instances (not offline and not being deleted)
	var stableInstances []*runtime.TiKV
	var offlineInstances []*runtime.TiKV

	for _, kv := range kvs {
		if kv.Spec.Offline {
			// Check offline condition to determine if it's active
			condition := v1alpha1.GetOfflineCondition(kv.Status.Conditions)
			if condition != nil && condition.Reason == v1alpha1.OfflineReasonActive {
				offlineInstances = append(offlineInstances, kv)
			}
		} else {
			stableInstances = append(stableInstances, kv)
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
			numToRescue = offlineCount
		}

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
func handleScaleOut(ctx context.Context, state *ReconcileContext, c client.Client, t tracker.Tracker[*v1alpha1.TiKVGroup, *v1alpha1.TiKV], updateRevision string, currentStable, numToAdd int) task.Result {
	logger := logr.FromContextOrDiscard(ctx)
	kvg := state.TiKVGroup()

	// Get topology policies
	var topos []v1alpha1.ScheduleTopology
	for _, p := range kvg.Spec.SchedulePolicies {
		if p.Type == v1alpha1.SchedulePolicyTypeEvenlySpread {
			topos = p.EvenlySpread.Topologies
		}
	}

	topoPolicy, err := policy.NewTopologyPolicy(topos, updateRevision, state.Slice()...)
	if err != nil {
		return task.Fail().With("invalid topo policy: %w", err)
	}

	allocator := t.Track(kvg, state.InstanceSlice()...)
	wait, err := updater.New[runtime.TiKVTuple]().
		WithInstances(state.Slice()...).
		WithDesired(currentStable+numToAdd).
		WithClient(c).
		WithMaxSurge(numToAdd).
		WithMaxUnavailable(0).
		WithRevision(updateRevision).
		WithNewFactory(TiKVNewer(kvg, updateRevision)).
		WithAddHooks(
			updater.AllocateName[*runtime.TiKV](allocator),
			topoPolicy,
		).
		Build().
		Do(ctx)

	if err != nil {
		return task.Fail().With("cannot scale out instances: %w", err)
	}
	if wait {
		return task.Wait().With("wait for new instances ready")
	}

	logger.Info("scale out completed", "added", numToAdd)
	return task.Complete().With("scale out completed")
}

// handleScaleIn initiates the offline process for selected instances
func handleScaleIn(ctx context.Context, state *ReconcileContext, c client.Client, t tracker.Tracker[*v1alpha1.TiKVGroup, *v1alpha1.TiKV], updateRevision string, kvs []*runtime.TiKV, desiredReplicas int) task.Result {
	logger := logr.FromContextOrDiscard(ctx)
	kvg := state.TiKVGroup()

	currentReplicas := len(kvs)
	numToOffline := currentReplicas - desiredReplicas

	logger.Info("scale in detected",
		"currentReplicas", currentReplicas,
		"desiredReplicas", desiredReplicas,
		"numToOffline", numToOffline)

	// Find instances that are not already offline
	var candidatesForOffline []*runtime.TiKV
	var alreadyOffline []*runtime.TiKV

	for _, kv := range kvs {
		if kv.Spec.Offline {
			alreadyOffline = append(alreadyOffline, kv)
		} else {
			candidatesForOffline = append(candidatesForOffline, kv)
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
	for _, p := range kvg.Spec.SchedulePolicies {
		if p.Type == v1alpha1.SchedulePolicyTypeEvenlySpread {
			topos = p.EvenlySpread.Topologies
		}
	}

	topoPolicy, err := policy.NewTopologyPolicy(topos, updateRevision, kvs...)
	if err != nil {
		return task.Fail().With("invalid topo policy: %w", err)
	}

	// Create scale-in selector with preference policies
	scaleInSelector := updater.NewSelector(
		updater.PreferAnnotatedForDeletion[*runtime.TiKV](), // Prioritize annotated instances
		updater.PreferNotOfflining[*runtime.TiKV](),         // Avoid instances already being offlined
		updater.PreferHealthyForScaleIn[*runtime.TiKV](),    // Prefer healthy instances for stable data migration
		topoPolicy, // Respect topology constraints
	)

	// Select instances to offline
	var instancesToOffline []*runtime.TiKV
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

func needVersionUpgrade(kvg *v1alpha1.TiKVGroup) bool {
	return kvg.Spec.Template.Spec.Version != kvg.Status.Version && kvg.Status.Version != ""
}

func TiKVNewer(kvg *v1alpha1.TiKVGroup, rev string) updater.NewFactory[*runtime.TiKV] {
	return updater.NewFunc[*runtime.TiKV](func() *runtime.TiKV {
		spec := kvg.Spec.Template.Spec.DeepCopy()

		tikv := &v1alpha1.TiKV{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: kvg.Namespace,
				// Name will be allocated by updater.AllocateName
				Labels:      coreutil.InstanceLabels[scope.TiKVGroup](kvg, rev),
				Annotations: coreutil.InstanceAnnotations[scope.TiKVGroup](kvg),
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(kvg, v1alpha1.SchemeGroupVersion.WithKind("TiKVGroup")),
				},
			},
			Spec: v1alpha1.TiKVSpec{
				Cluster:          kvg.Spec.Cluster,
				Features:         kvg.Spec.Features,
				Subdomain:        HeadlessServiceName(kvg.Name),
				TiKVTemplateSpec: *spec,
				// New instances start with offline = false (online)
				Offline: false,
			},
		}

		return runtime.FromTiKV(tikv)
	})
}
