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

	"github.com/pingcap/tidb-operator/pkg/runtime"
)

const (
	maxPreferPolicy = 30
)

type PreferPolicy[PT runtime.Instance] interface {
	Prefer([]PT) []PT
}

type ScoredPreferPolicy[PT runtime.Instance] interface {
	Score() Score
	Prefer([]PT) []PT
}

type PreferPolicyFunc[PT runtime.Instance] func([]PT) []PT

func (f PreferPolicyFunc[PT]) Prefer(in []PT) []PT {
	return f(in)
}

type Selector[PT runtime.Instance] interface {
	Choose([]PT) string
}

type Score uint32

type scoredPreferPolicy[PT runtime.Instance] struct {
	PreferPolicy[PT]

	score Score
}

func scored[PT runtime.Instance](s Score, p PreferPolicy[PT]) ScoredPreferPolicy[PT] {
	return &scoredPreferPolicy[PT]{
		PreferPolicy: p,
		score:        s,
	}
}

func (p *scoredPreferPolicy[PT]) Score() Score {
	return p.score
}

type selector[PT runtime.Instance] struct {
	ps []ScoredPreferPolicy[PT]
}

func NewSelector[PT runtime.Instance](ps ...PreferPolicy[PT]) Selector[PT] {
	if len(ps) > maxPreferPolicy {
		// TODO: use a util to panic for unreachable code
		panic(fmt.Sprintf("cannot new selector with too much prefer policy: %d", len(ps)))
	}
	s := selector[PT]{}
	for i, p := range ps {
		s.ps = append(s.ps, scored(Score(1<<(i+1)), p))
	}

	return &s
}

func (s *selector[PT]) Choose(allowed []PT) string {
	scores := make(map[string]uint32, len(allowed))
	for _, in := range allowed {
		scores[in.GetName()] = 1
	}
	for _, p := range s.ps {
		preferred := p.Prefer(allowed)
		for _, ins := range preferred {
			score, ok := scores[ins.GetName()]
			if !ok {
				score = 0
			}
			score += uint32(p.Score())
			scores[ins.GetName()] = score
		}
	}

	choosed := ""
	maximum := uint32(0)
	for name, score := range scores {
		if score > maximum {
			choosed = name
			maximum = score
		}
	}

	return choosed
}

func PreferUnavailable[PT runtime.Instance]() PreferPolicy[PT] {
	return PreferPolicyFunc[PT](func(s []PT) []PT {
		unavail := []PT{}
		for _, in := range s {
			if !in.IsUpToDate() || !in.IsHealthy() {
				unavail = append(unavail, in)
			}
		}
		return unavail
	})
}
