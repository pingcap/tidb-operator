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

package action

import (
	"context"
	"time"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework/workload"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/waiter"
)

func TestPDAvailability(ctx context.Context, f *framework.Framework, pdg *v1alpha1.PDGroup, w *framework.Workload) {
	// Prepare PD endpoints for region API access
	pdEndpoints := pdg.Name + "-pd." + f.Namespace.Name + ":2379"

	nctx, cancel := context.WithCancel(ctx)
	done := framework.AsyncWaitPodsRollingUpdateOnce[scope.PDGroup](nctx, f, pdg, int(*pdg.Spec.Replicas))
	defer func() { <-done }()

	done2 := w.MustRunPDRegionAccess(nctx, pdEndpoints)
	defer func() { <-done2 }()
	defer cancel()

	changeTime := time.Now()

	MustRollingRestart[scope.PDGroup](ctx, f, pdg)

	f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromPDGroup(pdg), changeTime, waiter.LongTaskTimeout))
	f.WaitForPDGroupReady(ctx, pdg)
}

func TestTSOAvailability(ctx context.Context, f *framework.Framework, tg *v1alpha1.TSOGroup, w *framework.Workload) {
	f.Must(waiter.WaitForClusterPDRegistered(ctx, f.Client, f.Cluster, waiter.LongTaskTimeout))
	// Prepare PD endpoints for region API access
	pdEndpoints := f.Cluster.Status.PD

	nctx, cancel := context.WithCancel(ctx)
	rolling := framework.AsyncWaitPodsRollingUpdateOnce[scope.TSOGroup](nctx, f, tg, int(*tg.Spec.Replicas))
	defer func() { <-rolling }()

	done := w.MustRunPDRegionAccess(nctx, pdEndpoints)
	defer func() { <-done }()

	defer cancel()

	changeTime := time.Now()
	MustRollingRestart[scope.TSOGroup](ctx, f, tg)

	f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTSOGroup(tg), changeTime, waiter.LongTaskTimeout))
	f.WaitForTSOGroupReady(ctx, tg)
}

func TestResourceManagerAvailability(ctx context.Context, f *framework.Framework, rmg *v1alpha1.ResourceManagerGroup, w *framework.Workload) {
	f.Must(waiter.WaitForClusterPDRegistered(ctx, f.Client, f.Cluster, waiter.LongTaskTimeout))
	// Prepare PD endpoints for region API access
	pdEndpoints := f.Cluster.Status.PD

	nctx, cancel := context.WithCancel(ctx)
	rolling := framework.AsyncWaitPodsRollingUpdateOnce[scope.ResourceManagerGroup](nctx, f, rmg, int(*rmg.Spec.Replicas))
	defer func() { <-rolling }()

	done := w.MustRunPDRegionAccess(nctx, pdEndpoints)
	defer func() { <-done }()

	defer cancel()

	changeTime := time.Now()
	MustRollingRestart[scope.ResourceManagerGroup](ctx, f, rmg)

	f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromResourceManagerGroup(rmg), changeTime, waiter.LongTaskTimeout))
	f.WaitForResourceManagerGroupReady(ctx, rmg)
}

func TestTiDBAvailability(ctx context.Context, f *framework.Framework, ep string, dbg *v1alpha1.TiDBGroup, w *framework.Workload, opts ...workload.Option) {
	nctx, cancel := context.WithCancel(ctx)
	rolling := framework.AsyncWaitPodsRollingUpdateOnce[scope.TiDBGroup](nctx, f, dbg, int(*dbg.Spec.Replicas))
	defer func() { <-rolling }()

	done := w.MustRunWorkload(nctx, ep, opts...)
	defer func() { <-done }()
	defer cancel()

	changeTime := time.Now()
	MustRollingRestart[scope.TiDBGroup](ctx, f, dbg)

	f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiDBGroup(dbg), changeTime, waiter.LongTaskTimeout))
	f.WaitForTiDBGroupReady(ctx, dbg)
}
