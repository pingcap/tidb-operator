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
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
)

// podDeletionCostPolicy prefers the lowest cost on the actual managed Pod.
// Costs are loaded before selection and cached for this updater execution.
// Instance annotations and Pod overlays are deliberately not consulted.
type podDeletionCostPolicy[R runtime.Instance] struct {
	costs map[types.NamespacedName]int32
}

// PreferPodDeletionCost prefers lower Pod deletion costs, after instance priority.
// Like the other preferences, Prefer only reads data already available in memory.
func PreferPodDeletionCost[R runtime.Instance]() *podDeletionCostPolicy[R] {
	return &podDeletionCostPolicy[R]{costs: make(map[types.NamespacedName]int32)}
}

func (p *podDeletionCostPolicy[R]) load(ctx context.Context, c client.Client, instances []R) {
	for _, in := range instances {
		key := types.NamespacedName{Namespace: in.GetNamespace(), Name: in.GetName()}
		if _, ok := p.costs[key]; ok {
			continue
		}
		pod, err := apicall.GetInstancePod(ctx, c, in)
		if err != nil {
			// This preference is best effort. Missing Pods and lookup/ownership
			// errors use the default cost instead of blocking reconciliation.
			p.costs[key] = 0
			continue
		}
		p.costs[key] = podDeletionCost(pod.Annotations)
	}
}

func (p *podDeletionCostPolicy[R]) Prefer(instances []R) []R {
	var chosen []R
	var minimum int32
	for _, in := range instances {
		cost := p.costs[types.NamespacedName{Namespace: in.GetNamespace(), Name: in.GetName()}]
		switch {
		case len(chosen) == 0 || cost < minimum:
			minimum = cost
			chosen = []R{in}
		case cost == minimum:
			chosen = append(chosen, in)
		}
	}
	return chosen
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
