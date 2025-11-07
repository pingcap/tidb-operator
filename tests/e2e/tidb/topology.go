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

package tidb

import (
	"context"

	"github.com/onsi/ginkgo/v2"

	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/framework/action"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

var _ = ginkgo.Describe("Topology", label.TiDB, label.MultipleAZ, label.P0, func() {
	f := framework.New()
	f.Setup()

	ginkgo.It("Create tidb evenly spread in multiple azs", func(ctx context.Context) {
		ginkgo.By("Creating cluster")
		pdg := f.MustCreatePD(ctx)
		kvg := f.MustCreateTiKV(ctx)
		dbg := f.MustCreateTiDB(ctx,
			data.WithReplicas[scope.TiDBGroup](3),
			data.WithTiDBEvenlySpreadPolicy(),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)
		f.WaitForTiDBGroupReady(ctx, dbg)

		f.MustEvenlySpreadTiDB(ctx, dbg)
	})

	ginkgo.It("rolling update and ensure still evenly spread", label.Update, func(ctx context.Context) {
		ginkgo.By("Creating cluster")
		pdg := f.MustCreatePD(ctx)
		kvg := f.MustCreateTiKV(ctx)
		dbg := f.MustCreateTiDB(ctx,
			data.WithReplicas[scope.TiDBGroup](3),
			data.WithTiDBEvenlySpreadPolicy(),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)
		f.WaitForTiDBGroupReady(ctx, dbg)

		f.MustEvenlySpreadTiDB(ctx, dbg)

		nctx, cancel := context.WithCancel(ctx)
		done := framework.AsyncWaitPodsRollingUpdateOnce[scope.TiDBGroup](nctx, f, dbg, 3)
		defer func() { <-done }()
		defer cancel()

		changeTime, err := waiter.MaxPodsCreateTimestamp[scope.TiDBGroup](ctx, f.Client, dbg)
		f.Must(err)

		action.MustRollingRestart[scope.TiDBGroup](ctx, f, dbg)

		f.Must(waiter.WaitForInstanceListRecreated[scope.TiDBGroup](ctx, f.Client, dbg, *changeTime, waiter.LongTaskTimeout))
		f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiDBGroup(dbg), *changeTime, waiter.LongTaskTimeout))
		f.WaitForTiDBGroupReady(ctx, dbg)
		f.MustEvenlySpreadTiDB(ctx, dbg)
	})

	// a, b -> a, c and then rolling + scale at the same time
	// 1. a, b
	// 2. a(old), b(old), c(new)
	// 3. a(new), b(old), c(new)
	// 4. a, c
	// 5. a(old), c(old), b(new), a(new)
	// 6. a(old), c(new), b(new), a(new)
	// 6. c(new), b(new), a(new)
	ginkgo.It("rolling update and scale + rolling update", label.Scale, label.Update, func(ctx context.Context) {
		ginkgo.By("Creating cluster")
		pdg := f.MustCreatePD(ctx)
		kvg := f.MustCreateTiKV(ctx)
		dbg := f.MustCreateTiDB(ctx,
			data.WithReplicas[scope.TiDBGroup](2),
			data.WithTiDBEvenlySpreadPolicy(),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)
		f.WaitForTiDBGroupReady(ctx, dbg)
		f.MustEvenlySpreadTiDB(ctx, dbg)

		nctx, cancel := context.WithCancel(ctx)
		done := framework.AsyncWaitPodsRollingUpdateOnce[scope.TiDBGroup](nctx, f, dbg, 2)

		changeTime, err := waiter.MaxPodsCreateTimestamp[scope.TiDBGroup](ctx, f.Client, dbg)
		f.Must(err)

		action.MustRollingRestart[scope.TiDBGroup](ctx, f, dbg)

		f.Must(waiter.WaitForInstanceListRecreated[scope.TiDBGroup](ctx, f.Client, dbg, *changeTime, waiter.LongTaskTimeout))
		f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiDBGroup(dbg), *changeTime, waiter.LongTaskTimeout))
		f.WaitForTiDBGroupReady(ctx, dbg)
		f.MustEvenlySpreadTiDB(ctx, dbg)

		cancel()
		<-done

		nctx, cancel = context.WithCancel(ctx)
		done = framework.AsyncWaitPodsRollingUpdateOnce[scope.TiDBGroup](nctx, f, dbg, 3)
		defer func() { <-done }()
		defer cancel()

		changeTime, err = waiter.MaxPodsCreateTimestamp[scope.TiDBGroup](ctx, f.Client, dbg)
		f.Must(err)

		action.MustScaleAndRollingRestart[scope.TiDBGroup](ctx, f, dbg, 3)

		f.Must(waiter.WaitForInstanceListRecreated[scope.TiDBGroup](ctx, f.Client, dbg, *changeTime, waiter.LongTaskTimeout))
		f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiDBGroup(dbg), *changeTime, waiter.LongTaskTimeout))
		f.WaitForTiDBGroupReady(ctx, dbg)
		f.MustEvenlySpreadTiDB(ctx, dbg)
	})
})
