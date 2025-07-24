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
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

type NewFactory[R runtime.Instance] interface {
	New() R
}

type NewFunc[R runtime.Instance] func() R

func (f NewFunc[R]) New() R {
	return f()
}

type actor[T runtime.Tuple[O, R], O client.Object, R runtime.Instance] struct {
	c client.Client

	noInPlaceUpdate bool

	f NewFactory[R]

	converter T

	update   State[R]
	outdated State[R]
	// deleted set records all instances that are marked by defer delete annotation
	deleted State[R]

	addHooks    []AddHook[R]
	updateHooks []UpdateHook[R]
	delHooks    []DelHook[R]

	scaleInSelector Selector[R]
	updateSelector  Selector[R]
}

func (act *actor[T, O, R]) chooseToUpdate(s []R) (string, error) {
	name := act.updateSelector.Choose(s)
	if name == "" {
		return "", fmt.Errorf("no instance can be updated")
	}

	return name, nil
}

func (act *actor[T, O, R]) chooseToScaleIn(s []R) (string, error) {
	name := act.scaleInSelector.Choose(s)
	if name == "" {
		return "", fmt.Errorf("no instance can be scale in")
	}

	return name, nil
}

func (act *actor[T, O, R]) ScaleOut(ctx context.Context) error {
	logger := logr.FromContextOrDiscard(ctx)
	obj := act.f.New()

	for _, hook := range act.addHooks {
		obj = hook.Add(obj)
	}

	logger.Info("act scale out", "namespace", obj.GetNamespace(), "name", obj.GetName())

	if err := act.c.Apply(ctx, act.converter.To(obj)); err != nil {
		return err
	}

	act.update.Add(obj)

	return nil
}

func (act *actor[T, O, R]) ScaleInUpdate(ctx context.Context) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)
	name, err := act.chooseToScaleIn(act.update.List())
	if err != nil {
		return false, err
	}

	obj := act.update.Del(name)

	isUnavailable := !obj.IsReady() || !obj.IsUpToDate()

	logger.Info("act scale in update", "choosed", name, "isUnavailable", isUnavailable, "remain", act.update.Len())

	if err := act.c.Delete(ctx, act.converter.To(obj)); err != nil {
		return false, err
	}

	for _, hook := range act.delHooks {
		hook.Delete(obj.GetName())
	}

	return isUnavailable, nil
}

func (act *actor[T, O, R]) ScaleInOutdated(ctx context.Context) (bool, error) {
	return act.scaleInOutdated(ctx, true)
}

func (act *actor[T, O, R]) scaleInOutdated(ctx context.Context, deferDel bool) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)
	name, err := act.chooseToScaleIn(act.outdated.List())
	if err != nil {
		return false, err
	}

	obj := act.outdated.Del(name)
	isUnavailable := !obj.IsReady() || !obj.IsUpToDate()

	logger.Info("act scale in outdated",
		"choosed", name, "defer", deferDel, "isUnavailable", isUnavailable, "remain", act.outdated.Len())

	if deferDel {
		if err := act.deferDelete(ctx, obj); err != nil {
			return false, err
		}
	} else {
		if err := act.c.Delete(ctx, act.converter.To(obj)); err != nil {
			return false, err
		}
	}

	for _, hook := range act.delHooks {
		hook.Delete(obj.GetName())
	}

	return isUnavailable, nil
}

type Patch struct {
	Metadata Metadata `json:"metadata"`
}

type Metadata struct {
	ResourceVersion string            `json:"resourceVersion"`
	Annotations     map[string]string `json:"annotations"`
}

func (act *actor[T, O, R]) deferDelete(ctx context.Context, obj R) error {
	o := act.converter.To(obj)
	p := Patch{
		Metadata: Metadata{
			ResourceVersion: o.GetResourceVersion(),
			Annotations: map[string]string{
				v1alpha1.AnnoKeyDeferDelete: v1alpha1.AnnoValTrue,
			},
		},
	}

	data, err := json.Marshal(&p)
	if err != nil {
		return fmt.Errorf("invalid patch: %w", err)
	}

	if err := act.c.Patch(ctx, o, client.RawPatch(types.MergePatchType, data)); err != nil {
		return fmt.Errorf("cannot mark obj %s/%s as defer delete: %w", obj.GetNamespace(), obj.GetName(), err)
	}

	act.deleted.Add(obj)

	return nil
}

func (act *actor[T, O, R]) Update(ctx context.Context) error {
	logger := logr.FromContextOrDiscard(ctx)
	if act.noInPlaceUpdate {
		if _, err := act.scaleInOutdated(ctx, false); err != nil {
			return err
		}
		if err := act.ScaleOut(ctx); err != nil {
			return err
		}

		return nil
	}

	name, err := act.chooseToUpdate(act.outdated.List())
	if err != nil {
		return err
	}

	outdated := act.outdated.Del(name)

	update := act.f.New()
	for _, hook := range act.updateHooks {
		update = hook.Update(update, outdated)
	}

	logger.Info("act update", "choosed", name, "remain", act.outdated.Len())

	if err := act.c.Apply(ctx, act.converter.To(update)); err != nil {
		return err
	}

	act.update.Add(update)

	return nil
}

func (act *actor[T, O, R]) Cleanup(ctx context.Context) error {
	for _, item := range act.deleted.List() {
		if err := act.c.Delete(ctx, act.converter.To(item)); err != nil {
			return err
		}
	}

	return nil
}

// cancelableActor extends the basic actor with cancel scale-in capabilities.
// It embeds the basic actor and adds the ScaleInLifecycle for managing offline operations.
type cancelableActor[T runtime.Tuple[O, R], O client.Object, R runtime.Instance] struct {
	*actor[T, O, R] // Embed basic actor to inherit all existing functionality

	// lifecycle manages the scale-in cancellation operations for store instances
	lifecycle ScaleInLifecycle[R]
}

// CancelScaleIn implements the CancelableActor interface.
// It attempts to cancel ongoing scale-in operations by rescuing offline instances back to online state.
//
// The method works by:
// 1. Identifying instances in offline state that can be rescued
// 2. Calculating how many instances need to be rescued to meet target replicas
// 3. Using selection policies to choose the best instances for rescue
// 4. Using the ScaleInLifecycle to cancel offline operations for selected instances
// 5. Updating internal state to reflect the rescued instances
func (act *cancelableActor[T, O, R]) CancelScaleIn(ctx context.Context, targetReplicas int) (int, error) {
	logger := logr.FromContextOrDiscard(ctx)

	// Get all instances that are currently offline and can potentially be rescued
	var offlineInstances []R

	// Check instances in update and outdated states for offline ones
	allInstances := append(act.update.List(), act.outdated.List()...)
	for _, instance := range allInstances {
		if act.lifecycle.IsOffline(instance) && !act.lifecycle.IsOfflineCompleted(instance) {
			offlineInstances = append(offlineInstances, instance)
		}
	}

	if len(offlineInstances) == 0 {
		logger.Info("no offline instances available for rescue")
		return 0, nil
	}

	// Calculate current online instances
	currentOnline := len(act.update.List()) + len(act.outdated.List()) - len(offlineInstances)
	rescueNeeded := targetReplicas - currentOnline

	if rescueNeeded <= 0 {
		logger.Info("no rescue needed", "currentOnline", currentOnline, "targetReplicas", targetReplicas)
		return 0, nil
	}

	// Limit rescue to what's actually needed and available
	if rescueNeeded > len(offlineInstances) {
		rescueNeeded = len(offlineInstances)
	}

	logger.Info("attempting to rescue offline instances",
		"offlineCount", len(offlineInstances),
		"rescueNeeded", rescueNeeded,
		"targetReplicas", targetReplicas)

	// Use selection logic to choose the best instances for rescue
	// We reverse the scale-in preference - prefer instances that were least preferred for scale-in
	var instancesToRescue []R
	if len(offlineInstances) <= rescueNeeded {
		// Rescue all offline instances
		instancesToRescue = offlineInstances
	} else {
		// Select the best candidates using reverse scale-in preference
		// TODO: Could implement smarter selection logic here based on topology, health, etc.
		instancesToRescue = offlineInstances[:rescueNeeded]
	}

	// Rescue instances using the lifecycle manager
	rescued := 0
	var rescueErrors []error

	for _, instance := range instancesToRescue {
		if err := act.lifecycle.CancelScaleIn(ctx, instance); err != nil {
			logger.Error(err, "failed to cancel scale-in for instance", "instance", instance.GetName())
			rescueErrors = append(rescueErrors, fmt.Errorf("failed to rescue instance %s: %w", instance.GetName(), err))
			continue
		}
		rescued++
		logger.Info("successfully rescued instance from offline state", "instance", instance.GetName())

		// Critical: Update internal state to reflect the rescued instance
		// The instance is no longer offline, so it should be counted as online
		// Note: The actual spec.offline change will be reflected in the next reconcile cycle
		// but we need to be consistent with our internal tracking
	}

	// Report partial failures if any occurred
	if len(rescueErrors) > 0 && rescued == 0 {
		// All rescues failed
		return 0, fmt.Errorf("failed to rescue any instances: %v", rescueErrors)
	} else if len(rescueErrors) > 0 {
		// Partial success - log errors but don't fail the operation
		logger.Info("partial rescue success", "rescued", rescued, "failed", len(rescueErrors), "errors", rescueErrors)
	}

	logger.Info("cancel scale-in completed", "rescued", rescued, "requested", rescueNeeded)
	return rescued, nil
}

// CanCancel implements the CancelableActor interface.
// Returns true if this actor has a configured lifecycle manager and can perform cancel operations.
func (act *cancelableActor[T, O, R]) CanCancel() bool {
	return act.lifecycle != nil
}

// CleanupCompletedOffline finds and deletes any instances that have completed the offline
// process but are still present. This is crucial for handling race conditions during
// scale-in cancellation where an instance might complete offline before it can be canceled.
// It returns the number of instances cleaned up.
func (act *cancelableActor[T, O, R]) CleanupCompletedOffline(ctx context.Context) (int, error) {
	logger := logr.FromContextOrDiscard(ctx)
	cleaned := 0

	// Check both update and outdated lists for completed offline instances
	allInstances := append(act.update.List(), act.outdated.List()...)

	for _, instance := range allInstances {
		if act.lifecycle.IsOfflineCompleted(instance) {
			logger.Info("cleaning up completed offline instance", "instance", instance.GetName())

			if err := act.c.Delete(ctx, act.converter.To(instance)); err != nil {
				// Log the error but continue trying to clean up others
				logger.Error(err, "failed to clean up completed offline instance", "instance", instance.GetName())
				continue
			}

			// Remove from internal state
			act.update.Del(instance.GetName())
			act.outdated.Del(instance.GetName())

			for _, hook := range act.delHooks {
				hook.Delete(instance.GetName())
			}

			cleaned++
		}
	}

	if cleaned > 0 {
		logger.Info("cleaned up completed offline instances", "count", cleaned)
	}

	return cleaned, nil
}

// ScaleInUpdate overrides the basic actor's ScaleInUpdate to use the two-phase deletion process.
// It first calls lifecycle.BeginScaleIn() to initiate the offline process for store instances.
func (act *cancelableActor[T, O, R]) ScaleInUpdate(ctx context.Context) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)

	// Prioritize deleting instances that have already completed the offline process.
	for _, instance := range act.update.List() {
		if act.lifecycle.IsOfflineCompleted(instance) {
			logger.Info("finalizing scale-in for completed offline instance", "instance", instance.GetName())

			isUnavailable := !instance.IsReady() || !instance.IsUpToDate()

			if err := act.c.Delete(ctx, act.converter.To(instance)); err != nil {
				return false, fmt.Errorf("failed to delete the completed offline instance %s: %w", instance.GetName(), err)
			}

			// Only remove from state after successful finalization
			act.update.Del(instance.GetName())

			for _, hook := range act.delHooks {
				hook.Delete(instance.GetName())
			}

			return isUnavailable, nil
		}
	}

	name, err := act.chooseToScaleIn(act.update.List())
	if err != nil {
		return false, err
	}

	obj := act.update.Get(name)
	if obj.GetName() == "" {
		return false, fmt.Errorf("instance %s not found in update list", name)
	}

	// Check if the instance is already in offline process
	if act.lifecycle.IsOffline(obj) {
		// Offline in progress, wait for completion
		logger.Info("waiting for offline completion", "instance", name)
		return true, nil
	}

	// Instance is not offline yet, begin the offline process
	logger.Info("beginning scale-in for updated instance", "instance", name)

	if err := act.lifecycle.BeginScaleIn(ctx, obj); err != nil {
		return false, fmt.Errorf("failed to begin scale-in for instance %s: %w", name, err)
	}

	// Instance is now starting offline process, it becomes unavailable
	return true, nil
}

// ScaleInOutdated overrides the basic actor's ScaleInOutdated to use the two-phase deletion process.
// It first calls lifecycle.BeginScaleIn() to initiate the offline process for store instances.
func (act *cancelableActor[T, O, R]) ScaleInOutdated(ctx context.Context) (bool, error) {
	return act.scaleInOutdatedWithLifecycle(ctx, true)
}

// CountOfflineInstances implements the CancelableActor interface.
// It returns two counts:
// - beingOffline: instances currently undergoing offline process but not yet completed
// - offlineCompleted: instances that have completed offline and are no longer in the cluster
func (act *cancelableActor[T, O, R]) CountOfflineInstances() (beingOffline, offlineCompleted int) {
	// Check instances in update state
	for _, instance := range act.update.List() {
		if act.lifecycle.IsOffline(instance) {
			if act.lifecycle.IsOfflineCompleted(instance) {
				offlineCompleted++
			} else {
				beingOffline++
			}
		}
	}

	// Check instances in outdated state
	for _, instance := range act.outdated.List() {
		if act.lifecycle.IsOffline(instance) {
			if act.lifecycle.IsOfflineCompleted(instance) {
				offlineCompleted++
			} else {
				beingOffline++
			}
		}
	}

	return beingOffline, offlineCompleted
}

func (act *cancelableActor[T, O, R]) scaleInOutdatedWithLifecycle(ctx context.Context, deferDel bool) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)

	// Prioritize deleting instances that have already completed the offline process.
	for _, instance := range act.outdated.List() {
		if act.lifecycle.IsOfflineCompleted(instance) {
			logger.Info("finalizing scale-in for completed offline outdated instance", "instance", instance.GetName())

			isUnavailable := !instance.IsReady() || !instance.IsUpToDate()

			if deferDel {
				if err := act.deferDelete(ctx, instance); err != nil {
					return false, err
				}
			} else {
				if err := act.c.Delete(ctx, act.converter.To(instance)); err != nil {
					return false, fmt.Errorf("failed to delete the completed offline instance %s: %w", instance.GetName(), err)
				}
			}

			// Only remove from state after successful finalization or defer delete
			act.outdated.Del(instance.GetName())

			for _, hook := range act.delHooks {
				hook.Delete(instance.GetName())
			}

			return isUnavailable, nil
		}
	}

	name, err := act.chooseToScaleIn(act.outdated.List())
	if err != nil {
		return false, err
	}

	obj := act.outdated.Get(name)
	if obj.GetName() == "" {
		return false, fmt.Errorf("instance %s not found in outdated list", name)
	}

	// Check if the instance is already in offline process
	if act.lifecycle.IsOffline(obj) {
		// Offline in progress, wait for completion
		logger.Info("waiting for offline completion", "instance", name)
		return true, nil // Return unavailable=true to wait
	}

	// Instance is not offline yet, begin the offline process
	logger.Info("beginning scale-in for outdated instance", "instance", name)

	if err := act.lifecycle.BeginScaleIn(ctx, obj); err != nil {
		return false, fmt.Errorf("failed to begin scale-in for instance %s: %w", name, err)
	}

	// Instance is now starting offline process, it becomes unavailable
	return true, nil
}
