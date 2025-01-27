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

package framework

import (
	"context"
	"math"
	"strings"

	"github.com/onsi/ginkgo/v2"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/utils/topology"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

func (f *Framework) WaitForTiKVGroupReady(ctx context.Context, kvg *v1alpha1.TiKVGroup) {
	// TODO: maybe wait for cluster ready
	ginkgo.By("wait for tikv group ready")
	f.Must(waiter.WaitForTiKVsHealthy(ctx, f.Client, kvg, waiter.LongTaskTimeout))
	f.Must(waiter.WaitForPodsReady(ctx, f.Client, runtime.FromTiKVGroup(kvg), waiter.LongTaskTimeout))
}

func (f *Framework) MustEvenlySpreadTiKV(ctx context.Context, kvg *v1alpha1.TiKVGroup) {
	list := v1alpha1.TiKVList{}
	f.Must(f.Client.List(ctx, &list, client.InNamespace(kvg.GetNamespace()), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   kvg.Spec.Cluster.Name,
		v1alpha1.LabelKeyGroup:     kvg.GetName(),
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiKV,
	}))

	encoder := topology.NewEncoder()
	topo := map[string]int{}

	detail := strings.Builder{}
	for i := range list.Items {
		item := &list.Items[i]

		key := encoder.Encode(item.Spec.Topology)
		val, ok := topo[key]
		if !ok {
			val = 0
		}
		val += 1
		topo[key] = val

		detail.WriteString(item.Name)
		detail.WriteString(":\n")
		for k, v := range item.Spec.Topology {
			detail.WriteString("    ")
			detail.WriteString(k)
			detail.WriteString(":")
			detail.WriteString(v)
			detail.WriteString(":\n")
		}
	}

	minimum, maximum := math.MaxInt, 0
	for _, val := range topo {
		if val < minimum {
			minimum = val
		}
		if val > maximum {
			maximum = val
		}
	}

	if maximum-minimum > 1 {
		ginkgo.AddReportEntry("TopologyInfo", detail.String())
	}

	f.True(maximum-minimum <= 1)
}
