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
	"github.com/pingcap/tidb-operator/pkg/apicall"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/pkg/utils/topology"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

func (f *Framework) WaitForTiDBGroupReady(ctx context.Context, dbg *v1alpha1.TiDBGroup) {
	// TODO: maybe wait for cluster ready
	ginkgo.By("wait for tidb group ready")
	f.Must(waiter.WaitForObjectCondition[scope.TiDBGroup](
		ctx,
		f.Client,
		dbg,
		v1alpha1.CondReady,
		metav1.ConditionTrue,
		waiter.LongTaskTimeout,
	))
	f.Must(waiter.WaitForTiDBsHealthy(ctx, f.Client, dbg, waiter.LongTaskTimeout))
	f.Must(waiter.WaitForPodsReady(ctx, f.Client, runtime.FromTiDBGroup(dbg), waiter.LongTaskTimeout))
}

func MustEvenlySpread[
	GS scope.GroupInstance[GF, GT, IS],
	IS scope.InstanceList[IF, IT, IL],
	GF client.Object,
	GT runtime.Group,
	IF client.Object,
	IT runtime.Instance,
	IL client.ObjectList,
](ctx context.Context, f *Framework, g GF) {
	ins, err := apicall.ListInstances[GS](ctx, f.Client, g)
	f.Must(err)

	encoder := topology.NewEncoder()
	topo := map[string]int{}

	var st []v1alpha1.ScheduleTopology
	for _, p := range coreutil.SchedulePolicies[GS](g) {
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
	for _, in := range ins {
		key := encoder.Encode(coreutil.Topology[IS](in))
		val, ok := topo[key]

		f.True(ok, "unknown topology %s: %v", in.GetName(), coreutil.Topology[IS](in))

		val += 1
		topo[key] = val

		detail.WriteString(in.GetName())
		detail.WriteString(":\n")
		for k, v := range coreutil.Topology[IS](in) {
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
