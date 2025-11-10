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

package policy

import (
	"maps"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/updater"
	"github.com/pingcap/tidb-operator/pkg/utils/topology"
)

type topologyPolicy[R runtime.Instance] struct {
	all     topology.Scheduler
	updated topology.Scheduler

	rev string
}

type TopologyPolicy[R runtime.Instance] interface {
	updater.AddHook[R]
	updater.DelHook[R]
	updater.UpdateHook[R]
	PolicyScaleIn() updater.PreferPolicy[R]
	PolicyUpdate() updater.PreferPolicy[R]
}

func NewTopologyPolicy[R runtime.Instance](ts []v1alpha1.ScheduleTopology, rev string, rs ...R) (TopologyPolicy[R], error) {
	all, err := topology.New(ts)
	if err != nil {
		return nil, err
	}
	updated, err := topology.New(ts)
	if err != nil {
		return nil, err
	}
	p := &topologyPolicy[R]{
		all:     all,
		updated: updated,
		rev:     rev,
	}
	for _, r := range rs {
		p.all.Add(r.GetName(), r.GetTopology())
		if r.GetUpdateRevision() == rev {
			p.updated.Add(r.GetName(), r.GetTopology())
		}
	}
	return p, nil
}

func (p *topologyPolicy[R]) Add(update R) R {
	topo := update.GetTopology()
	// topology is not set
	if len(topo) == 0 {
		all := p.all.NextAdd()
		updated := p.updated.NextAdd()
		topo = choose(all, updated)
	}

	update.SetTopology(topo)
	p.all.Add(update.GetName(), update.GetTopology())
	if update.GetUpdateRevision() == p.rev {
		p.updated.Add(update.GetName(), update.GetTopology())
	}

	return update
}

func (p *topologyPolicy[R]) Update(update, outdated R) R {
	update.SetTopology(outdated.GetTopology())

	p.all.Add(update.GetName(), update.GetTopology())
	if update.GetUpdateRevision() == p.rev {
		p.updated.Add(update.GetName(), update.GetTopology())
	}

	return update
}

func (p *topologyPolicy[R]) Delete(name string) {
	p.all.Del(name)
	p.updated.Del(name)
}

func (p *topologyPolicy[R]) PolicyScaleIn() updater.PreferPolicy[R] {
	return &deletePreferPolicy[R]{
		p: p,
	}
}

func (p *topologyPolicy[R]) PolicyUpdate() updater.PreferPolicy[R] {
	return &updatePreferPolicy[R]{
		p: p,
	}
}

type deletePreferPolicy[R runtime.Instance] struct {
	p *topologyPolicy[R]
}

func (p *deletePreferPolicy[R]) Prefer(allowed []R) []R {
	if len(allowed) == 0 {
		return nil
	}
	names := p.p.all.NextDel()
	var preferred []R
	for _, item := range allowed {
		for _, name := range names {
			if item.GetName() == name {
				preferred = append(preferred, item)
			}
		}
	}

	return preferred
}

type updatePreferPolicy[R runtime.Instance] struct {
	p *topologyPolicy[R]
}

// Choose a prefered item to update
// Update will not change topology spreads of "all" set.
// However, spreads of "update" set will be changed.
func (p *updatePreferPolicy[R]) Prefer(allowed []R) []R {
	if len(allowed) == 0 {
		return nil
	}
	topos := p.p.updated.NextAdd()

	var preferred []R
	for _, item := range allowed {
		for _, topo := range topos {
			if maps.Equal(topo, item.GetTopology()) {
				preferred = append(preferred, item)
			}
		}
	}

	return preferred
}

// choose a preferred topology
// - prefer "all" and "update" set are well spread, choose the intersection
// - if no intersection, just return the first one of "all" set
func choose(all, update []v1alpha1.Topology) v1alpha1.Topology {
	// No topology is preferred
	// Normally because of no topology policy is specified
	if len(all) == 0 {
		return nil
	}
	// Only one topology can be chosen
	if len(all) == 1 {
		return all[0]
	}

	// More than one topologies can be chosen
	// Try to find the first topology which is in both all and update
	for _, at := range all {
		for _, bt := range update {
			if maps.Equal(at, bt) {
				return at
			}
		}
	}

	// No intersection of preferred topologies of all and update
	// just return the first preferred topology of all
	return all[0]
}
