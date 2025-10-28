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
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	wopt "github.com/pingcap/tidb-operator/tests/e2e/framework/workload"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

// We use tidb workload to test whether the tikv is always available
// TODO(liubo02): maybe call tikv directly?
var _ = ginkgo.Describe("TiKV Availability Test", label.TiKV, label.KindAvail, label.Update, label.P0, func() {
	f := framework.New()
	f.Setup()

	f.DescribeFeatureTable(func(fs ...metav1alpha1.Feature) {
		f.SetupCluster(data.WithFeatureGates(fs...))
		workload := f.SetupWorkload()
		ginkgo.PIt("No error when rolling update tikv", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx, data.WithReplicas[*runtime.TiKVGroup](3))
			dbg := f.MustCreateTiDB(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)

			patch := client.MergeFrom(kvg.DeepCopy())
			kvg.Spec.Template.Labels = map[string]string{"test": "test"}

			nctx, cancel := context.WithCancel(ctx)
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer ginkgo.GinkgoRecover()
				f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromTiKVGroup(kvg), 3, 0, waiter.LongTaskTimeout))
			}()

			done := workload.MustRunWorkload(
				nctx,
				data.DefaultTiDBServiceName,
				// TODO: 500ms is still not worked, dig why
				wopt.MaxExecutionTime(1000),
				wopt.WorkloadType(wopt.WorkloadTypeSelectCount),
			)

			changeTime := time.Now()
			ginkgo.By("Change config of the TiKVGroup")
			f.Must(f.Client.Patch(ctx, kvg, patch))
			f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiKVGroup(kvg), changeTime, waiter.LongTaskTimeout))
			f.WaitForTiKVGroupReady(ctx, kvg)
			cancel()
			wg.Wait()
			<-done
		})
	},
		nil,
	)

	f.DescribeFeatureTable(func(fs ...metav1alpha1.Feature) {
		f.SetupCluster(data.WithFeatureGates(fs...))
		workload := f.SetupWorkload()
		// 500ms is still not worked
		ginkgo.PIt("No error when rolling update tikv", ginkgo.Serial, label.KindNextGen, func(ctx context.Context) {
			f.MustCreateS3(ctx)
			pdg := f.MustCreatePD(ctx, data.WithPDNextGen())
			kvg := f.MustCreateTiKV(ctx, data.WithTiKVNextGen(), data.WithReplicas[*runtime.TiKVGroup](3))
			dbg := f.MustCreateTiDB(ctx,
				data.WithTiDBNextGen(),
				data.WithKeyspace("SYSTEM"),
			)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)

			patch := client.MergeFrom(kvg.DeepCopy())
			kvg.Spec.Template.Labels = map[string]string{"test": "test"}

			nctx, cancel := context.WithCancel(ctx)
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer ginkgo.GinkgoRecover()
				f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromTiKVGroup(kvg), 3, 0, waiter.LongTaskTimeout))
			}()

			done := workload.MustRunWorkload(
				nctx,
				data.DefaultTiDBServiceName,
				// TODO: 500ms is still not worked, dig why
				wopt.MaxExecutionTime(1000),
				wopt.WorkloadType(wopt.WorkloadTypeSelectCount),
			)

			changeTime := time.Now()
			ginkgo.By("Change config of the TiKVGroup")
			f.Must(f.Client.Patch(ctx, kvg, patch))
			f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiKVGroup(kvg), changeTime, waiter.LongTaskTimeout))
			f.WaitForTiKVGroupReady(ctx, kvg)
			cancel()
			wg.Wait()
			<-done
		})
	},
		[]metav1alpha1.Feature{metav1alpha1.UseTiKVReadyAPI},
	)
})
