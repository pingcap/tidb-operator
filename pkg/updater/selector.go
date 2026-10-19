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
	"slices"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
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
	// Filter returns eligible instances. An empty slice means no instance is eligible.
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
		allowed = f.Filter(allowed)
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

// PreferNewer returns instances with the latest creation timestamp.
// Instances with equal timestamps remain candidates for subsequent policies.
func PreferNewer[R runtime.Instance]() PreferPolicy[R] {
	return PreferPolicyFunc[R](func(s []R) []R {
		var chosen []R
		for _, in := range s {
			if len(chosen) == 0 {
				chosen = append(chosen, in)
				continue
			}
			latest := chosen[0].GetCreationTimestamp()
			created := in.GetCreationTimestamp()
			switch {
			case latest.Before(&created):
				chosen = []R{in}
			case latest.Equal(&created):
				chosen = append(chosen, in)
			}
		}
		return chosen
	})
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

// PreferPodDeletionCost lets applications express which managed Pods are cheaper
// to remove using the standard controller.kubernetes.io/pod-deletion-cost annotation.
// Group controllers manage instances directly, so they must honor this hint themselves
// instead of relying on ReplicaSet downscaling. Read the actual Pod annotation, not
// instance CR metadata, so applications can use the same convention as Kubernetes.
// See https://kubernetes.io/docs/concepts/workloads/controllers/replicaset/#pod-deletion-cost
//
// Lower signed int32 costs are preferred; negative values are allowed and an unset
// annotation means zero. This is a best-effort preference, not a deletion-order
// guarantee: higher-priority policies run first.
func PreferPodDeletionCost[R runtime.Instance](c client.Client) PreferPolicy[R] {
	// Build creates fresh policies for each executor. Reuse costs across its selections,
	// including zero fallbacks, to avoid repeatedly listing the same Pods.
	costs := make(map[string]int32)
	return PreferPolicyFunc[R](func(instances []R) []R {
		if c == nil || len(instances) < 2 {
			return instances
		}
		var chosen []R
		var minimum int32
		for _, in := range instances {
			key := in.GetNamespace() + "/" + in.Component() + "/" + in.GetName() + "/" + string(in.GetUID())
			cost, ok := costs[key]
			if !ok {
				// Prefer has no reconcile context; the manager client reads Pods from cache.
				pod, err := apicall.GetInstancePod(context.Background(), c, in)
				if err == nil {
					cost = podDeletionCost(pod.Annotations)
				}
				costs[key] = cost
			}
			switch {
			case len(chosen) == 0 || cost < minimum:
				minimum = cost
				chosen = []R{in}
			case cost == minimum:
				chosen = append(chosen, in)
			}
		}
		return chosen
	})
}

// podDeletionCost follows Kubernetes' signed int32 annotation semantics:
// absent or invalid values cost zero; a leading '+' or positive leading zeros
// are invalid. Negative costs are allowed and preferred over zero.
func podDeletionCost(annotations map[string]string) int32 {
	value, ok := annotations[corev1.PodDeletionCost]
	if !ok || value == "" || value[0] == '+' || (value[0] == '0' && value != "0") {
		return 0
	}
	cost, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0
	}
	return int32(cost)
}

// FilterAnnotationAbsent excludes instances that have the given annotation key set.
func FilterAnnotationAbsent[R runtime.Instance](key string) FilterPolicy[R] {
	return FilterPolicyFunc[R](func(s []R) []R {
		var eligible []R
		for _, in := range s {
			if _, ok := in.GetAnnotations()[key]; ok {
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

// PreferVolumeCapacityExceedsRequest prefers instances with observed excess volume capacity.
func PreferVolumeCapacityExceedsRequest[R runtime.Instance]() PreferPolicy[R] {
	return PreferPolicyFunc[R](func(instances []R) []R {
		var preferred []R
		for _, in := range instances {
			cond := meta.FindStatusCondition(in.Conditions(), v1alpha1.CondVolumeCapacityExceedsRequest)
			if cond != nil && cond.Status == metav1.ConditionTrue && cond.ObservedGeneration == in.GetGeneration() {
				preferred = append(preferred, in)
			}
		}
		return preferred
	})
}
