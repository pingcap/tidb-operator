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
	"sync"
	"time"

	"github.com/onsi/ginkgo/v2"

	"github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	wopt "github.com/pingcap/tidb-operator/tests/e2e/framework/workload"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

var _ = ginkgo.Describe("TiFlash Availability Test", label.TiFlash, label.Update, label.KindAvail, func() {
	f := framework.New()
	f.Setup()
	f.SetupCluster(data.WithFeatureGates(v1alpha1.TerminableLogTailer))

	ginkgo.Context("Default", label.P0, func() {
		workload := f.SetupWorkload()
		ginkgo.It("No error when rolling update tiflash", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			fg := f.MustCreateTiFlash(ctx,
				// TODO: wait until the https://github.com/pingcap/tiflash/pull/10450 is released
				data.WithGroupImage[*runtime.TiFlashGroup]("gcr.io/pingcap-public/dbaas/tiflash:master-next-gen"),
				data.WithReplicas[*runtime.TiFlashGroup](2),
			)
			dbg := f.MustCreateTiDB(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiFlashGroupReady(ctx, fg)
			f.WaitForTiDBGroupReady(ctx, dbg)

			workload.MustImportData(ctx, data.DefaultTiDBServiceName, wopt.TiFlashReplicas(2), wopt.RegionCount(0))

			done1 := make(chan struct{})
			nctx, cancel := context.WithCancel(ctx)
			go func() {
				defer close(done1)
				defer ginkgo.GinkgoRecover()
				f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromTiFlashGroup(fg), 2, 0, waiter.LongTaskTimeout))
			}()
			done2 := workload.MustRunWorkload(nctx, data.DefaultTiDBServiceName, wopt.TiFlashReplicas(2))

			patch := client.MergeFrom(fg.DeepCopy())
			fg.Spec.Template.Labels = map[string]string{"test": "test"}

			changeTime := time.Now()
			ginkgo.By("Change config of the TiFlashGroup")

			f.Must(f.Client.Patch(ctx, fg, patch))
			f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiFlashGroup(fg), changeTime, waiter.LongTaskTimeout))
			f.WaitForTiFlashGroupReady(ctx, fg)
			cancel()
			<-done1
			<-done2
		})
	})

	ginkgo.Context("NextGen", label.KindNextGen, label.P0, func() {
		workload := f.SetupWorkload()
		// TODO: still not work
		ginkgo.It("No error when rolling update tiflash in next-gen", func(ctx context.Context) {
			f.MustCreateS3(ctx)
			pdg := f.MustCreatePD(ctx, data.WithPDNextGen())
			kvg := f.MustCreateTiKV(ctx, data.WithTiKVNextGen())
			fgc := f.MustCreateTiFlash(ctx,
				data.WithGroupName[*runtime.TiFlashGroup]("fg-compute"),
				data.WithReplicas[*runtime.TiFlashGroup](2),
				data.WithTiFlashNextGen(),
				data.WithTiFlashComputeMode(),
			)
			fgw := f.MustCreateTiFlash(ctx,
				data.WithGroupName[*runtime.TiFlashGroup]("fg-write"),
				data.WithReplicas[*runtime.TiFlashGroup](2),
				data.WithTiFlashNextGen(),
				data.WithTiFlashWriteMode(),
			)
			dbg := f.MustCreateTiDB(ctx,
				data.WithTiDBNextGen(),
				data.WithKeyspace("SYSTEM"),
			)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiFlashGroupReady(ctx, fgc)
			f.WaitForTiFlashGroupReady(ctx, fgw)
			f.WaitForTiDBGroupReady(ctx, dbg)

			workload.MustImportData(ctx, data.DefaultTiDBServiceName, wopt.TiFlashReplicas(2), wopt.RegionCount(0))

			wg := sync.WaitGroup{}
			nctx, cancel := context.WithCancel(ctx)
			wg.Add(2)
			go func() {
				defer wg.Done()
				defer ginkgo.GinkgoRecover()
				f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromTiFlashGroup(fgc), 2, 0, waiter.LongTaskTimeout))
			}()
			go func() {
				defer wg.Done()
				defer ginkgo.GinkgoRecover()
				f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromTiFlashGroup(fgw), 2, 0, waiter.LongTaskTimeout))
			}()

			done := workload.MustRunWorkload(nctx,
				data.DefaultTiDBServiceName,
				wopt.TiFlashReplicas(2),
				// Set max_execution_time to 4s for next-gen tiflash
				// TODO: dig why 2000ms is not enough
				wopt.MaxExecutionTime(4000),
			)

			patch := client.MergeFrom(fgc.DeepCopy())
			fgc.Spec.Template.Labels = map[string]string{"test": "test"}

			changeTime := time.Now()
			ginkgo.By("Change config of the compute TiFlashGroup")

			f.Must(f.Client.Patch(ctx, fgc, patch))
			f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiFlashGroup(fgc), changeTime, waiter.LongTaskTimeout))
			f.WaitForTiFlashGroupReady(ctx, fgc)

			patch = client.MergeFrom(fgw.DeepCopy())
			fgw.Spec.Template.Labels = map[string]string{"test": "test"}

			changeTime = time.Now()
			ginkgo.By("Change config of the write TiFlashGroup")

			f.Must(f.Client.Patch(ctx, fgw, patch))
			f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiFlashGroup(fgw), changeTime, waiter.LongTaskTimeout))
			f.WaitForTiFlashGroupReady(ctx, fgw)

			cancel()
			wg.Wait()
			<-done
		})
	})
})
