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
	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

type Builder[R runtime.Instance] interface {
	WithInstances(...R) Builder[R]
	WithDesired(desired int) Builder[R]
	WithMaxSurge(maxSurge int) Builder[R]
	WithMaxUnavailable(maxUnavailable int) Builder[R]
	WithRevision(rev string) Builder[R]
	WithClient(c client.Client) Builder[R]
	WithNewFactory(NewFactory[R]) Builder[R]
	WithAddHooks(hooks ...AddHook[R]) Builder[R]
	WithUpdateHooks(hooks ...UpdateHook[R]) Builder[R]
	WithDelHooks(hooks ...DelHook[R]) Builder[R]
	WithScaleInPreferPolicy(ps ...PreferPolicy[R]) Builder[R]
	WithUpdatePreferPolicy(ps ...PreferPolicy[R]) Builder[R]
	// NoInPlaceUpdate if true, actor will use Scale in and Scale out to replace Update operation
	WithNoInPaceUpdate(noUpdate bool) Builder[R]

	// WithScaleInLifecycle configures the builder to create a CancelableActor with scale-in lifecycle management.
	// This is optional and only needed for components that support cancel scale-in operations (like TiKV/TiFlash).
	//
	// When configured:
	//   - The builder will create a CancelableActor instead of a basic Actor
	//   - The resulting executor will support cancel scale-in operations
	//   - Components can rescue instances from offline state back to online state
	//
	// Components not calling this method will continue to work exactly as before with basic Actor functionality.
	WithScaleInLifecycle(lifecycle ScaleInLifecycle[R]) Builder[R]

	Build() Executor
}

type builder[T runtime.Tuple[O, R], O client.Object, R runtime.Instance] struct {
	instances      []R
	desired        int
	maxSurge       int
	maxUnavailable int
	rev            string

	noInPlaceUpdate bool

	c client.Client

	f NewFactory[R]

	addHooks    []AddHook[R]
	updateHooks []UpdateHook[R]
	delHooks    []DelHook[R]

	scaleInPreferPolicies []PreferPolicy[R]
	updatePreferPolicies  []PreferPolicy[R]

	// scaleInLifecycle is optional and when set, creates a CancelableActor instead of basic Actor
	scaleInLifecycle ScaleInLifecycle[R]
}

func (b *builder[T, O, R]) Build() Executor {
	update, outdated, deleted := split(b.instances, b.rev)

	updatePolicies := b.updatePreferPolicies
	updatePolicies = append(updatePolicies, PreferUnavailable[R]())

	// Create basic actor first
	basicActor := &actor[T, O, R]{
		c: b.c,
		f: b.f,

		noInPlaceUpdate: b.noInPlaceUpdate,

		update:   NewState(update),
		outdated: NewState(outdated),
		deleted:  NewState(deleted),

		addHooks:    b.addHooks,
		updateHooks: append(b.updateHooks, KeepName[R](), KeepTopology[R]()),
		delHooks:    b.delHooks,

		scaleInSelector: NewSelector(b.scaleInPreferPolicies...),
		updateSelector:  NewSelector(updatePolicies...),
	}

	// Create appropriate actor based on whether ScaleInLifecycle is configured
	var actorInterface Actor
	if b.scaleInLifecycle != nil {
		// Create cancelable actor when lifecycle is configured
		actorInterface = &cancelableActor[T, O, R]{
			actor:     basicActor,
			lifecycle: b.scaleInLifecycle,
		}
	} else {
		// Use basic actor when no lifecycle is configured (existing behavior)
		actorInterface = basicActor
	}

	return NewExecutor(actorInterface, len(update), len(outdated), b.desired,
		countUnavailable(update), countUnavailable(outdated), b.maxSurge, b.maxUnavailable)
}

func New[T runtime.Tuple[O, R], O client.Object, R runtime.Instance]() Builder[R] {
	return &builder[T, O, R]{}
}

func (b *builder[T, O, R]) WithInstances(instances ...R) Builder[R] {
	b.instances = append(b.instances, instances...)
	return b
}

func (b *builder[T, O, R]) WithDesired(desired int) Builder[R] {
	b.desired = desired
	return b
}

func (b *builder[T, O, R]) WithMaxSurge(maxSurge int) Builder[R] {
	b.maxSurge = maxSurge
	return b
}

func (b *builder[T, O, R]) WithMaxUnavailable(maxUnavailable int) Builder[R] {
	b.maxUnavailable = maxUnavailable
	return b
}

func (b *builder[T, O, R]) WithRevision(revision string) Builder[R] {
	b.rev = revision
	return b
}

func (b *builder[T, O, R]) WithClient(c client.Client) Builder[R] {
	b.c = c
	return b
}

func (b *builder[T, O, R]) WithNewFactory(f NewFactory[R]) Builder[R] {
	b.f = f
	return b
}

func (b *builder[T, O, R]) WithAddHooks(hooks ...AddHook[R]) Builder[R] {
	b.addHooks = append(b.addHooks, hooks...)
	return b
}

func (b *builder[T, O, R]) WithUpdateHooks(hooks ...UpdateHook[R]) Builder[R] {
	b.updateHooks = append(b.updateHooks, hooks...)
	return b
}

func (b *builder[T, O, R]) WithDelHooks(hooks ...DelHook[R]) Builder[R] {
	b.delHooks = append(b.delHooks, hooks...)
	return b
}

func (b *builder[T, O, R]) WithScaleInPreferPolicy(ps ...PreferPolicy[R]) Builder[R] {
	b.scaleInPreferPolicies = append(b.scaleInPreferPolicies, ps...)
	return b
}

func (b *builder[T, O, R]) WithUpdatePreferPolicy(ps ...PreferPolicy[R]) Builder[R] {
	b.updatePreferPolicies = append(b.updatePreferPolicies, ps...)
	return b
}

// NoInPlaceUpdate if true, actor will use Scale in and Scale out to replace Update operation
func (b *builder[T, O, R]) WithNoInPaceUpdate(noUpdate bool) Builder[R] {
	b.noInPlaceUpdate = noUpdate
	return b
}

// WithScaleInLifecycle configures the builder to create a CancelableActor with scale-in lifecycle management.
// This method is optional and only needed for components that support cancel scale-in operations.
//
// When a ScaleInLifecycle is configured:
//   - The builder will create a cancelableActor that implements CancelableActor interface
//   - The resulting executor will support detecting and handling cancel scale-in scenarios
//   - The lifecycle manager will be used for canceling offline operations on instances
//
// If this method is not called, the builder creates a basic Actor with existing behavior.
func (b *builder[T, O, R]) WithScaleInLifecycle(lifecycle ScaleInLifecycle[R]) Builder[R] {
	b.scaleInLifecycle = lifecycle
	return b
}

func split[R runtime.Instance](all []R, rev string) (update, outdated, deleted []R) {
	for _, instance := range all {
		// if instance is deleting, just ignore it
		// TODO(liubo02): make sure it's ok for PD
		if !instance.GetDeletionTimestamp().IsZero() {
			continue
		}
		if instance.GetUpdateRevision() == rev {
			update = append(update, instance)
		} else if _, ok := instance.GetAnnotations()[v1alpha1.AnnoKeyDeferDelete]; ok {
			deleted = append(deleted, instance)
		} else {
			outdated = append(outdated, instance)
		}
	}

	return update, outdated, deleted
}

func countUnavailable[R runtime.Instance](all []R) int {
	unavailable := 0
	for _, instance := range all {
		if !instance.IsReady() || !instance.IsUpToDate() {
			unavailable += 1
		}
	}

	return unavailable
}
