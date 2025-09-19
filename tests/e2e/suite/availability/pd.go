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

	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

var _ = ginkgo.Describe("PD Availability Test", label.PD, label.KindAvail, label.Update, func() {
	f := framework.New()
	f.Setup()
	ginkgo.Context("Default", label.P0, func() {
		workload := f.SetupWorkload()
		ginkgo.It("No error when rolling update pd", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx, data.WithReplicas[*runtime.PDGroup](3))
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)

			// Prepare PD endpoints for region API access
			pdEndpoints := pdg.Name + "-pd." + f.Namespace.Name + ":2379"

			patch := client.MergeFrom(pdg.DeepCopy())
			pdg.Spec.Template.Annotations = map[string]string{
				"test": "test",
			}

			nctx, cancel := context.WithCancel(ctx)
			wg := sync.WaitGroup{}
			wg.Add(2)
			go func() {
				defer wg.Done()
				defer ginkgo.GinkgoRecover()
				f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromPDGroup(pdg), 3, 0, waiter.LongTaskTimeout))
			}()
			go func() {
				defer wg.Done()
				defer ginkgo.GinkgoRecover()
				workload.MustRunPDRegionAccess(ctx, pdEndpoints)
			}()

			changeTime := time.Now()
			ginkgo.By("Rolling udpate the PDGroup")
			f.Must(f.Client.Patch(ctx, pdg, patch))
			f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromPDGroup(pdg), changeTime, waiter.LongTaskTimeout))
			f.WaitForPDGroupReady(ctx, pdg)
			cancel()
			wg.Wait()
		})
	})
})
