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
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
)

type Builder[PT runtime.Instance] interface {
	WithInstances(...PT) Builder[PT]
	WithDesired(desired int) Builder[PT]
	WithMaxSurge(maxSurge int) Builder[PT]
	WithMaxUnavailable(maxUnavailable int) Builder[PT]
	WithRevision(rev string) Builder[PT]
	WithClient(c client.Client) Builder[PT]
	WithNewFactory(NewFactory[PT]) Builder[PT]
	WithAddHooks(hooks ...AddHook[PT]) Builder[PT]
	WithUpdateHooks(hooks ...UpdateHook[PT]) Builder[PT]
	WithDelHooks(hooks ...DelHook[PT]) Builder[PT]
	WithScaleInPreferPolicy(ps ...PreferPolicy[PT]) Builder[PT]
	WithUpdatePreferPolicy(ps ...PreferPolicy[PT]) Builder[PT]
	Build() Executor
}

type builder[PT runtime.Instance] struct {
	instances      []PT
	desired        int
	maxSurge       int
	maxUnavailable int
	rev            string

	c client.Client

	f NewFactory[PT]

	addHooks    []AddHook[PT]
	updateHooks []UpdateHook[PT]
	delHooks    []DelHook[PT]

	scaleInPreferPolicies []PreferPolicy[PT]
	updatePreferPolicies  []PreferPolicy[PT]
}

func (b *builder[PT]) Build() Executor {
	update, outdated := split(b.instances, b.rev)

	updatePolicies := b.updatePreferPolicies
	updatePolicies = append(updatePolicies, PreferUnavailable[PT]())
	actor := &actor[PT]{
		c: b.c,
		f: b.f,

		update:   NewState(update),
		outdated: NewState(outdated),

		addHooks:    b.addHooks,
		updateHooks: b.updateHooks,
		delHooks:    b.delHooks,

		scaleInSelector: NewSelector(b.scaleInPreferPolicies...),
		updateSelector:  NewSelector(updatePolicies...),
	}
	return NewExecutor(actor, len(update), len(outdated), b.desired,
		countUnavailable(update), countUnavailable(outdated), b.maxSurge, b.maxUnavailable)
}

func New[PT runtime.Instance]() Builder[PT] {
	return &builder[PT]{}
}

func (b *builder[PT]) WithInstances(instances ...PT) Builder[PT] {
	b.instances = append(b.instances, instances...)
	return b
}

func (b *builder[PT]) WithDesired(desired int) Builder[PT] {
	b.desired = desired
	return b
}

func (b *builder[PT]) WithMaxSurge(maxSurge int) Builder[PT] {
	b.maxSurge = maxSurge
	return b
}

func (b *builder[PT]) WithMaxUnavailable(maxUnavailable int) Builder[PT] {
	b.maxUnavailable = maxUnavailable
	return b
}

func (b *builder[PT]) WithRevision(revision string) Builder[PT] {
	b.rev = revision
	return b
}

func (b *builder[PT]) WithClient(c client.Client) Builder[PT] {
	b.c = c
	return b
}

func (b *builder[PT]) WithNewFactory(f NewFactory[PT]) Builder[PT] {
	b.f = f
	return b
}

func (b *builder[PT]) WithAddHooks(hooks ...AddHook[PT]) Builder[PT] {
	b.addHooks = append(b.addHooks, hooks...)
	return b
}

func (b *builder[PT]) WithUpdateHooks(hooks ...UpdateHook[PT]) Builder[PT] {
	b.updateHooks = append(b.updateHooks, hooks...)
	return b
}

func (b *builder[PT]) WithDelHooks(hooks ...DelHook[PT]) Builder[PT] {
	b.delHooks = append(b.delHooks, hooks...)
	return b
}

func (b *builder[PT]) WithScaleInPreferPolicy(ps ...PreferPolicy[PT]) Builder[PT] {
	b.scaleInPreferPolicies = append(b.scaleInPreferPolicies, ps...)
	return b
}

func (b *builder[PT]) WithUpdatePreferPolicy(ps ...PreferPolicy[PT]) Builder[PT] {
	b.updatePreferPolicies = append(b.updatePreferPolicies, ps...)
	return b
}

func split[PT runtime.Instance](all []PT, rev string) (update, outdated []PT) {
	for _, instance := range all {
		// if instance is deleting, just ignore it
		// TODO(liubo02): make sure it's ok for PD
		if !instance.GetDeletionTimestamp().IsZero() {
			continue
		}
		fmt.Println("split:", instance.GetName(), instance.GetUpdateRevision(), rev)
		if instance.GetUpdateRevision() == rev {
			update = append(update, instance)
		} else {
			outdated = append(outdated, instance)
		}
	}

	return update, outdated
}

func countUnavailable[PT runtime.Instance](all []PT) int {
	unavailable := 0
	for _, instance := range all {
		if !instance.IsHealthy() || !instance.IsUpToDate() {
			unavailable += 1
		}
	}

	return unavailable
}
