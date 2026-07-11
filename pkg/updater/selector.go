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

// FilterPolicy narrows the allowed instances before prefer policies run.
type FilterPolicy[R runtime.Instance] interface {
	// Filter returns eligible instances.
	// Return nil to abstain and keep the current allowed set.
	// Return a non-nil empty slice when no instance is eligible.
	Filter([]R) []R
}

type FilterPolicyFunc[R runtime.Instance] func([]R) []R

func (f FilterPolicyFunc[R]) Filter(in []R) []R {
	return f(in)
}

type Selector[R runtime.Instance] interface {
	Choose([]R) string
}

type selector[R runtime.Instance] struct {
	filters []FilterPolicy[R]
	ps      []PreferPolicy[R]
}

// NewSelector with prefer policies, the last policy has highest priority
func NewSelector[R runtime.Instance](ps ...PreferPolicy[R]) Selector[R] {
	return &selector[R]{ps: ps}
}

// NewSelectorWithFilter creates a selector with filter and prefer policies.
func NewSelectorWithFilter[R runtime.Instance](filters []FilterPolicy[R], ps ...PreferPolicy[R]) Selector[R] {
	return &selector[R]{filters: filters, ps: ps}
}

func applyFilters[R runtime.Instance](allowed []R, filters []FilterPolicy[R]) []R {
	for _, f := range filters {
		filtered := f.Filter(allowed)
		if filtered == nil {
			continue
		}
		if len(filtered) == 0 {
			return filtered
		}
		allowed = filtered
	}
	return allowed
}

func (s *selector[R]) Choose(allowed []R) string {
	allowed = applyFilters(allowed, s.filters)
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

// FilterReviveAbandoned excludes instances whose scale-in revive was abandoned.
func FilterReviveAbandoned[R runtime.Instance]() FilterPolicy[R] {
	return FilterPolicyFunc[R](func(s []R) []R {
		var eligible []R
		for _, in := range s {
			if in.GetAnnotations()[v1alpha1.AnnoKeyTiProxyReviveAbandoned] == v1alpha1.AnnoValTrue {
				continue
			}
			eligible = append(eligible, in)
		}
		if len(eligible) == 0 {
			return []R{}
		}
		return eligible
	})
}

// FilterOutdated excludes instances not on the target revision.
func FilterOutdated[R runtime.Instance](revision string) FilterPolicy[R] {
	return FilterPolicyFunc[R](func(s []R) []R {
		var updated []R
		for _, in := range s {
			if in.GetUpdateRevision() == revision {
				updated = append(updated, in)
			}
		}
		if len(updated) == 0 {
			return []R{}
		}
		return updated
	})
}
