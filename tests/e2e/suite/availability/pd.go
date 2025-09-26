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

	"github.com/onsi/ginkgo/v2"

	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
)

var _ = ginkgo.Describe("PD Availability Test", label.PD, label.KindAvail, label.Update, func() {
	f := framework.New()
	f.Setup(framework.WithSkipClusterDeletionWhenFailed())
	ginkgo.Context("Default", label.P0, func() {
		workload := f.SetupWorkload()
		ginkgo.It("No error when rolling update pd", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx, data.WithReplicas[*runtime.PDGroup](3))
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)

			f.TestPDAvailability(ctx, pdg, workload)
		})
	})

	ginkgo.Context("NextGen PDMS", label.KindNextGen, label.P0, label.Features(metav1alpha1.UseTSOReadyAPI), func() {
		workload := f.SetupWorkload()
		f.SetupCluster(data.WithFeatureGates(metav1alpha1.UseTSOReadyAPI))
		// TODO: wait until the pr is merged
		ginkgo.It("No error when rolling update pd in next-gen", label.KindNextGen, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx, data.WithPDNextGen(), data.WithMSMode(), data.WithReplicas[*runtime.PDGroup](3))
			kvg := f.MustCreateTiKV(ctx, data.WithTiKVNextGen())
			dbg := f.MustCreateTiDB(ctx,
				data.WithTiDBNextGen(),
				data.WithKeyspace("SYSTEM"),
			)
			tg := f.MustCreateTSO(ctx,
				data.WithTSONextGen(),
			)
			sg := f.MustCreateScheduler(ctx,
				data.WithSchedulerNextGen(),
			)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			f.WaitForTSOGroupReady(ctx, tg)
			f.WaitForSchedulerGroupReady(ctx, sg)

			f.TestPDAvailability(ctx, pdg, workload)
		})

		ginkgo.It("No error when rolling update tso in next-gen", label.KindNextGen, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx, data.WithPDNextGen(), data.WithMSMode())
			kvg := f.MustCreateTiKV(ctx, data.WithTiKVNextGen())
			dbg := f.MustCreateTiDB(ctx,
				data.WithTiDBNextGen(),
				data.WithKeyspace("SYSTEM"),
			)
			tg := f.MustCreateTSO(ctx,
				data.WithTSONextGen(),
				data.WithReplicas[*runtime.TSOGroup](2),
			)
			sg := f.MustCreateScheduler(ctx,
				data.WithSchedulerNextGen(),
				data.WithReplicas[*runtime.SchedulerGroup](2),
			)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			f.WaitForTSOGroupReady(ctx, tg)
			f.WaitForSchedulerGroupReady(ctx, sg)

			f.TestTSOAvailability(ctx, tg, workload)
		})
	})
})
