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
	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/updater"
	"github.com/pingcap/tidb-operator/pkg/utils/topology"
)

type topologyPolicy[R runtime.Instance] struct {
	scheduler topology.Scheduler
}

type TopologyPolicy[R runtime.Instance] interface {
	updater.AddHook[R]
	updater.DelHook[R]
	updater.PreferPolicy[R]
}

func NewTopologyPolicy[R runtime.Instance](ts []v1alpha1.ScheduleTopology) (TopologyPolicy[R], error) {
	s, err := topology.New(ts)
	if err != nil {
		return nil, err
	}
	return &topologyPolicy[R]{
		scheduler: s,
	}, nil
}

func (p *topologyPolicy[R]) Add(update R) R {
	topo := p.scheduler.NextAdd()
	update.SetTopology(topo)
	p.scheduler.Add(update.GetName(), update.GetTopology())

	return update
}

func (p *topologyPolicy[R]) Delete(name string) {
	p.scheduler.Del(name)
}

func (p *topologyPolicy[R]) Prefer(allowed []R) []R {
	names := p.scheduler.NextDel()
	preferred := make([]R, 0, len(allowed))
	for _, item := range allowed {
		for _, name := range names {
			if item.GetName() == name {
				preferred = append(preferred, item)
			}
		}
	}

	return preferred
}
