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
	"k8s.io/apimachinery/pkg/api/meta"
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

// action represents an action performed by Actor.
type action int

const (
	actionNone action = iota
	actionScaleOut
	actionScaleInUpdate
	actionScaleInOutdated
	actionUpdate
	actionSetOffline
	actionDeferDelete
	actionDelete
)

type actor[T runtime.Tuple[O, R], O client.Object, R runtime.Instance] struct {
	c client.Client

	noInPlaceUpdate bool

	f NewFactory[R]

	converter T

	update   State[R]
	outdated State[R]
	// deleted set records all instances that are marked by defer delete annotation
	deleted State[R]
	// beingOffline set records all instances that are in the process of going offline.
	// It will be used for cancel offline.
	beingOffline State[R]

	addHooks    []AddHook[R]
	updateHooks []UpdateHook[R]
	delHooks    []DelHook[R]

	scaleInSelector Selector[R]
	updateSelector  Selector[R]

	actions []action
}

// chooseToUpdate selects an outdated instance for update operation.
// Uses the updateSelector to determine which instance should be updated next.
// Returns the name of the selected instance or an error if no instance can be updated.
func (act *actor[T, O, R]) chooseToUpdate(s []R) (string, error) {
	name := act.updateSelector.Choose(s)
	if name == "" {
		return "", fmt.Errorf("no instance can be updated")
	}

	return name, nil
}

// chooseToScaleIn selects an instance for scale-in operation.
// Uses the scaleInSelector to determine which instance should be scaled in next.
// Returns the name of the selected instance or an error if no instance can be scaled in.
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
	act.actions = append(act.actions, actionScaleOut)

	return nil
}

// ScaleInUpdate scales in an updated instance (already running the latest version).
// This is used for cluster scaling down when the total number of instances exceeds the desired count.
// It operates on the "update" state collection which contains instances with the current revision.
func (act *actor[T, O, R]) ScaleInUpdate(ctx context.Context) (bool, error) {
	logger := logr.FromContextOrDiscard(ctx)
	name, err := act.chooseToScaleIn(act.update.List())
	if err != nil {
		return false, err
	}

	obj := act.update.Del(name)

	isUnavailable := !obj.IsReady() || !obj.IsUpToDate()

	logger.Info("act scale in update", "selected", name, "isUnavailable", isUnavailable, "remain", act.update.Len())

	a, err := act.deleteInstance(ctx, obj)
	if err != nil {
		return false, err
	}
	if a == actionDelete {
		act.actions = append(act.actions, actionScaleInUpdate)
	} else {
		act.actions = append(act.actions, a)
	}

	for _, hook := range act.delHooks {
		hook.Delete(obj.GetName())
	}

	return isUnavailable, nil
}

// ScaleInOutdated scales in an outdated instance (running an old version).
// This is used during rolling updates to clean up old instances after new ones are ready.
// It operates on the "outdated" state collection which contains instances with older revisions.
// Uses deferred deletion by default to ensure data safety during rolling updates.
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
		"selected", name, "defer", deferDel, "isUnavailable", isUnavailable, "remain", act.outdated.Len())

	if deferDel {
		if err := act.deferDelete(ctx, obj); err != nil {
			return false, err
		}
	} else {
		a, err := act.deleteInstance(ctx, obj)
		if err != nil {
			return false, err
		}
		if a == actionDelete {
			act.actions = append(act.actions, actionScaleInOutdated)
		} else {
			act.actions = append(act.actions, a)
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

// deferDelete marks an instance with defer-delete annotation instead of immediately deleting it.
// This is a safety mechanism used during rolling updates to prevent data loss and ensure
// cluster stability. The marked instance will be moved to the "deleted" state collection
// and will be actually deleted later by the Cleanup() method after the new instance is ready.
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
	act.actions = append(act.actions, actionDeferDelete)
	return nil
}

// Update performs instance update using either in-place or recreate strategy.
//
// In-place update strategy (default):
// - Selects an outdated instance and updates it to the current revision
// - Preserves instance name and topology through update hooks
//
// Recreate update strategy (noInPlaceUpdate=true):
// - Scales in an outdated instance (immediate deletion)
// - Scales out a new instance to replace it
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

	logger.Info("act update", "selected", name, "remain", act.outdated.Len())

	if err := act.c.Apply(ctx, act.converter.To(update)); err != nil {
		return err
	}

	act.update.Add(update)
	act.actions = append(act.actions, actionUpdate)

	return nil
}

// Cleanup deletes all instances that were marked with defer-delete annotation.
// This is called after rolling updates are complete to remove the old instances
// that were safely marked for deletion. It ensures data safety by only removing
// instances after the new instances are fully operational.
func (act *actor[T, O, R]) Cleanup(ctx context.Context) error {
	for _, item := range act.deleted.List() {
		a, err := act.deleteInstance(ctx, item)
		if err != nil {
			return err
		}
		act.actions = append(act.actions, a)
	}
	return nil
}

// RecordedActions returns all actions recorded by the actor.
// This is used for testing purposes to verify that the actor performed expected actions.
func (act *actor[T, O, R]) RecordedActions() []action {
	return act.actions
}

func (act *actor[T, O, R]) deleteInstance(ctx context.Context, obj R) (action, error) {
	if obj.IsStore() && !obj.IsOffline() && !meta.IsStatusConditionTrue(obj.Conditions(), v1alpha1.StoreOfflinedConditionType) {
		patchData := []byte(fmt.Sprintf(`[{"op": "replace", "path": "/spec/offline", "value": %v}]`, true))
		if err := act.c.Patch(ctx, act.converter.To(obj), client.RawPatch(types.JSONPatchType, patchData)); err != nil {
			return actionNone, fmt.Errorf("failed to set instance %s/%s offline: %w", obj.GetNamespace(), obj.GetName(), err)
		}
		act.beingOffline.Add(obj)
		return actionSetOffline, nil
	}

	if err := act.c.Delete(ctx, act.converter.To(obj)); err != nil {
		return actionNone, err
	}

	return actionDelete, nil
}
