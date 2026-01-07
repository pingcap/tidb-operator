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
	"slices"
	"strconv"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
)

type PreferPolicy[R runtime.Instance] interface {
	// Prefer returns the preferred instances
	// If no preferred, just return nil or empty
	Prefer([]R) []R
}

type PreferPolicyFunc[R runtime.Instance] func([]R) []R

func (f PreferPolicyFunc[R]) Prefer(in []R) []R {
	return f(in)
}

type Selector[R runtime.Instance] interface {
	Choose([]R) string
}

type selector[R runtime.Instance] struct {
	ps []PreferPolicy[R]
}

// NewSelector with prefer policies, the last policy has highest priority
func NewSelector[R runtime.Instance](ps ...PreferPolicy[R]) Selector[R] {
	s := selector[R]{
		ps: ps,
	}

	return &s
}

func (s *selector[R]) Choose(allowed []R) string {
	// return early if no allowed
	if len(allowed) == 0 {
		return ""
	}

	for _, p := range slices.Backward(s.ps) {
		preferred := p.Prefer(allowed)
		switch {
		case len(preferred) == 1:
			// only one preferred, return this one
			return preferred[0].GetName()
		case len(preferred) == 0:
			// no preferred, see next policy
			continue
		}

		// more than one preferred, minimize the allowed set
		allowed = preferred
	}

	// no preferred, return first allowed
	if len(allowed) > 0 {
		return allowed[0].GetName()
	}
	return ""
}

func PreferUnready[R runtime.Instance]() PreferPolicy[R] {
	return PreferPolicyFunc[R](func(s []R) []R {
		var unavail []R
		for _, in := range s {
			if !in.IsUpToDate() || !in.IsReady() {
				unavail = append(unavail, in)
			}
		}
		return unavail
	})
}

func PreferNotRunning[R runtime.Instance]() PreferPolicy[R] {
	return PreferPolicyFunc[R](func(s []R) []R {
		var unavail []R
		for _, in := range s {
			if in.IsNotRunning() {
				unavail = append(unavail, in)
			}
		}
		return unavail
	})
}

// PreferPriority returns instances which have a minimal priority val in annotation
// 0 is the highest priority
// unset/invalid is the lowest priority
func PreferPriority[R runtime.Instance]() PreferPolicy[R] {
	return PreferPolicyFunc[R](func(s []R) []R {
		var chosen []R
		var minimal int64 = -1
		for _, in := range s {
			anno := in.GetAnnotations()
			sPrio, ok := anno[v1alpha1.AnnoKeyPriority]
			if !ok {
				continue
			}
			// not a valid priority
			// TODO(liubo02): add a validation
			prio, err := strconv.ParseInt(sPrio, 10, 64)
			if err != nil || prio < 0 {
				continue
			}

			switch {
			case minimal == -1:
				minimal = prio
				chosen = []R{in}
			case prio < minimal:
				minimal = prio
				chosen = []R{in}
			case prio == minimal:
				chosen = append(chosen, in)
			}
		}

		if minimal == -1 {
			return s
		}

		return chosen
	})
}
