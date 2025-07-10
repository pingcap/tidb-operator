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

type PreferPolicy[R runtime.Instance] interface {
	Prefer([]R) []R
}

type ScoredPreferPolicy[R runtime.Instance] interface {
	Score() Score
	Prefer([]R) []R
}

type PreferPolicyFunc[R runtime.Instance] func([]R) []R

func (f PreferPolicyFunc[R]) Prefer(in []R) []R {
	return f(in)
}

type Selector[R runtime.Instance] interface {
	Choose([]R) string
}

type Score uint32

type scoredPreferPolicy[R runtime.Instance] struct {
	PreferPolicy[R]

	score Score
}

func scored[R runtime.Instance](s Score, p PreferPolicy[R]) ScoredPreferPolicy[R] {
	return &scoredPreferPolicy[R]{
		PreferPolicy: p,
		score:        s,
	}
}

func (p *scoredPreferPolicy[R]) Score() Score {
	return p.score
}

type selector[R runtime.Instance] struct {
	ps []ScoredPreferPolicy[R]
}

func NewSelector[R runtime.Instance](ps ...PreferPolicy[R]) Selector[R] {
	if len(ps) > maxPreferPolicy {
		// TODO: use a util to panic for unreachable code
		panic(fmt.Sprintf("cannot new selector with too much prefer policy: %d", len(ps)))
	}
	s := selector[R]{}
	for i, p := range ps {
		s.ps = append(s.ps, scored(Score(1<<(i+1)), p))
	}

	return &s
}

func (s *selector[R]) Choose(allowed []R) string {
	nameToIndex := make(map[string]int, len(allowed))
	scores := make([]uint32, len(allowed))
	for index, in := range allowed {
		nameToIndex[in.GetName()] = index
		scores[index] = 1
	}
	for _, p := range s.ps {
		preferred := p.Prefer(allowed)
		for _, ins := range preferred {
			index, ok := nameToIndex[ins.GetName()]
			if !ok {
				panic("prefer should always return items from allowed")
			}
			scores[index] += uint32(p.Score())
		}
	}

	var choosed int
	maximum := uint32(0)
	for index, score := range scores {
		if score > maximum {
			choosed = index
			maximum = score
		}
	}

	return allowed[choosed].GetName()
}

func PreferUnavailable[R runtime.Instance]() PreferPolicy[R] {
	return PreferPolicyFunc[R](func(s []R) []R {
		unavail := []R{}
		for _, in := range s {
			if !in.IsUpToDate() || !in.IsReady() {
				unavail = append(unavail, in)
			}
		}
		return unavail
	})
}
