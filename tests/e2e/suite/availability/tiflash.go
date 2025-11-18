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

package availability

import (
	"context"
	"time"

	"github.com/onsi/ginkgo/v2"

	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework/action"
	wopt "github.com/pingcap/tidb-operator/v2/tests/e2e/framework/workload"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/label"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/waiter"
)

var _ = ginkgo.Describe("TiFlash Availability Test",
	label.TiFlash,
	label.Update,
	label.KindAvail,
	label.Features(),
	func() {
		f := framework.New()
		f.Setup()
		f.SetupCluster(data.WithFeatureGates())

		ginkgo.Context("Default", label.P0, func() {
			workload := f.SetupWorkload()
			ginkgo.It("No error when rolling update tiflash", func(ctx context.Context) {
				pdg := f.MustCreatePD(ctx)
				kvg := f.MustCreateTiKV(ctx)
				fg := f.MustCreateTiFlash(ctx,
					// TODO: wait until the https://github.com/pingcap/tiflash/pull/10450 is released
					data.WithImage[scope.TiFlashGroup]("gcr.io/pingcap-public/dbaas/tiflash:master-next-gen"),
					data.WithReplicas[scope.TiFlashGroup](2),
				)
				dbg := f.MustCreateTiDB(ctx)

				f.WaitForPDGroupReady(ctx, pdg)
				f.WaitForTiKVGroupReady(ctx, kvg)
				f.WaitForTiFlashGroupReady(ctx, fg)
				f.WaitForTiDBGroupReady(ctx, dbg)

				workload.MustImportData(ctx, data.DefaultTiDBServiceName, wopt.TiFlashReplicas(2), wopt.RegionCount(0))

				nctx, cancel := context.WithCancel(ctx)
				done1 := framework.AsyncWaitPodsRollingUpdateOnce[scope.TiFlashGroup](nctx, f, fg, 2)
				defer func() { <-done1 }()
				done2 := workload.MustRunWorkload(
					nctx,
					data.DefaultTiDBServiceName,
					wopt.TiFlashReplicas(2),
					wopt.WorkloadType(wopt.WorkloadTypeSelectCount),
				)
				defer func() { <-done2 }()
				defer cancel()

				changeTime := time.Now()

				action.MustRollingRestart[scope.TiFlashGroup](ctx, f, fg)

				f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiFlashGroup(fg), changeTime, waiter.LongTaskTimeout))
				f.WaitForTiFlashGroupReady(ctx, fg)
			})
		})
	})
