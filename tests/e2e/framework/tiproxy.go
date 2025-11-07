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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/topology"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

func (f *Framework) WaitForTiProxyGroupReady(ctx context.Context, pg *v1alpha1.TiProxyGroup) {
	ginkgo.By("wait for tiproxy group ready")
	f.Must(waiter.WaitForObjectCondition[scope.TiProxyGroup](
		ctx,
		f.Client,
		pg,
		v1alpha1.CondReady,
		metav1.ConditionTrue,
		waiter.LongTaskTimeout,
	))
	f.Must(waiter.WaitForTiProxysHealthy(ctx, f.Client, pg, waiter.LongTaskTimeout))
	f.Must(waiter.WaitForPodsReady(ctx, f.Client, runtime.FromTiProxyGroup(pg), waiter.LongTaskTimeout))
}

func (f *Framework) MustEvenlySpreadTiProxy(ctx context.Context, pg *v1alpha1.TiProxyGroup) {
	list := v1alpha1.TiProxyList{}
	f.Must(f.Client.List(ctx, &list, client.InNamespace(pg.GetNamespace()), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   pg.Spec.Cluster.Name,
		v1alpha1.LabelKeyGroup:     pg.GetName(),
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiProxy,
	}))

	encoder := topology.NewEncoder()
	topo := map[string]int{}

	var st []v1alpha1.ScheduleTopology
	for _, p := range pg.Spec.SchedulePolicies {
		switch p.Type {
		case v1alpha1.SchedulePolicyTypeEvenlySpread:
			st = p.EvenlySpread.Topologies
		default:
			// do nothing
		}
	}

	for _, t := range st {
		key := encoder.Encode(t.Topology)
		topo[key] = 0
	}

	detail := strings.Builder{}
	for i := range list.Items {
		item := &list.Items[i]

		key := encoder.Encode(item.Spec.Topology)
		val, ok := topo[key]

		f.True(ok, "unknown topology %s: %v", item.Name, item.Spec.Topology)

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
