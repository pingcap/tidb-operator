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
	"k8s.io/apimachinery/pkg/api/meta"

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
}

func (b *builder[T, O, R]) Build() Executor {
	update, outdated, beingOffline, deleted := split(b.instances, b.rev)

	updatePolicies := b.updatePreferPolicies
	updatePolicies = append(updatePolicies, PreferUnavailable[R]())
	actor := &actor[T, O, R]{
		c: b.c,
		f: b.f,

		noInPlaceUpdate: b.noInPlaceUpdate,

		update:       NewState(update),
		outdated:     NewState(outdated),
		beingOffline: NewState(beingOffline),
		deleted:      NewState(deleted),

		addHooks:    b.addHooks,
		updateHooks: append(b.updateHooks, KeepName[R](), KeepTopology[R]()),
		delHooks:    b.delHooks,

		scaleInSelector: NewSelector(b.scaleInPreferPolicies...),
		updateSelector:  NewSelector(updatePolicies...),
	}
	return NewExecutor(actor, len(update), len(outdated), b.desired,
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

func split[R runtime.Instance](all []R, rev string) (update, outdated, beingOffline, deleted []R) {
	for _, instance := range all {
		// if instance is deleting, just ignore it
		// TODO(liubo02): make sure it's ok for PD
		if !instance.GetDeletionTimestamp().IsZero() {
			continue
		}

		if _, ok := instance.GetAnnotations()[v1alpha1.AnnoKeyDeferDelete]; ok ||
			meta.IsStatusConditionTrue(instance.Conditions(), v1alpha1.StoreOfflinedConditionType) {
			deleted = append(deleted, instance)
			continue
		}
		if instance.IsOffline() {
			beingOffline = append(beingOffline, instance)
			continue
		}
		if instance.GetUpdateRevision() == rev {
			update = append(update, instance)
		} else {
			outdated = append(outdated, instance)
		}
	}

	return update, outdated, beingOffline, deleted
}

func countUnavailable[R runtime.Instance](all []R) int {
	unavailable := 0
	for _, instance := range all {
		if !instance.IsReady() || !instance.IsUpToDate() {
			unavailable++
		}
	}

	return unavailable
}
