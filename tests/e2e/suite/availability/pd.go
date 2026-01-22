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
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework/action"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/label"
)

var _ = ginkgo.Describe("PD Availability Test", label.PD, label.KindAvail, label.Update, func() {
	f := framework.New()
	f.Setup()
	f.SetupCluster(data.WithFeatureGates(metav1alpha1.UsePDReadyAPIV2))
	ginkgo.Context("Default", label.P0, func() {
		workload := f.SetupWorkload()
		// TODO: wait until the ready api pr is released
		// https://github.com/tikv/pd/pull/9851
		ginkgo.PIt("No error when rolling update pd", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx,
				data.WithReplicas[scope.PDGroup](3),
			)
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)

			action.TestPDAvailability(ctx, f, pdg, workload)
		})
	})

	ginkgo.Context("NextGen PDMS",
		ginkgo.Serial,
		label.KindNextGen,
		label.P0,
		label.Features(
			metav1alpha1.UsePDReadyAPIV2,
			metav1alpha1.UseTSOReadyAPI,
			metav1alpha1.UseSchedulingReadyAPI,
		), func() {
			workload := f.SetupWorkload()
			f.SetupCluster(data.WithFeatureGates(
				metav1alpha1.UsePDReadyAPIV2,
				metav1alpha1.UseTSOReadyAPI,
				metav1alpha1.UseSchedulingReadyAPI,
			))
			ginkgo.It("No error when rolling update pd in next-gen", func(ctx context.Context) {
				f.MustCreateS3(ctx)
				pdg := f.MustCreatePD(ctx, data.WithPDNextGen(), data.WithMSMode(), data.WithReplicas[scope.PDGroup](3))
				kvg := f.MustCreateTiKV(ctx, data.WithTiKVNextGen())
				dbg := f.MustCreateTiDB(ctx,
					data.WithTiDBNextGen(),
					data.WithKeyspace("SYSTEM"),
				)
				tg := f.MustCreateTSO(ctx,
					data.WithTSONextGen(),
				)
				sg := f.MustCreateScheduling(ctx,
					data.WithSchedulingNextGen(),
				)

				f.WaitForPDGroupReady(ctx, pdg)
				f.WaitForTiKVGroupReady(ctx, kvg)
				f.WaitForTiDBGroupReady(ctx, dbg)
				f.WaitForTSOGroupReady(ctx, tg)
				f.WaitForSchedulingGroupReady(ctx, sg)

				action.TestPDAvailability(ctx, f, pdg, workload)
			})

			ginkgo.It("No error when rolling update tso in next-gen", func(ctx context.Context) {
				f.MustCreateS3(ctx)
				pdg := f.MustCreatePD(ctx, data.WithPDNextGen(), data.WithMSMode())
				kvg := f.MustCreateTiKV(ctx, data.WithTiKVNextGen())
				dbg := f.MustCreateTiDB(ctx,
					data.WithTiDBNextGen(),
					data.WithKeyspace("SYSTEM"),
				)
				tg := f.MustCreateTSO(ctx,
					data.WithTSONextGen(),
					data.WithReplicas[scope.TSOGroup](2),
				)
				sg := f.MustCreateScheduling(ctx,
					data.WithSchedulingNextGen(),
					data.WithReplicas[scope.SchedulingGroup](2),
				)

				f.WaitForPDGroupReady(ctx, pdg)
				f.WaitForTiKVGroupReady(ctx, kvg)
				f.WaitForTiDBGroupReady(ctx, dbg)
				f.WaitForTSOGroupReady(ctx, tg)
				f.WaitForSchedulingGroupReady(ctx, sg)

				action.TestTSOAvailability(ctx, f, tg, workload)
			})

			// TODO: enable ResourceManager test after supporting latest image
			ginkgo.PIt("No error when rolling update resource manager in next-gen", func(ctx context.Context) {
				f.MustCreateS3(ctx)
				pdg := f.MustCreatePD(ctx,
					data.WithPDNextGen(),
					data.WithMSMode(),
					data.WithResourceManager(),
				)
				kvg := f.MustCreateTiKV(ctx, data.WithTiKVNextGen())
				dbg := f.MustCreateTiDB(ctx,
					data.WithTiDBNextGen(),
					data.WithKeyspace("SYSTEM"),
				)
				tg := f.MustCreateTSO(ctx,
					data.WithTSONextGen(),
					data.WithReplicas[scope.TSOGroup](2),
				)
				sg := f.MustCreateScheduling(ctx,
					data.WithSchedulingNextGen(),
					data.WithReplicas[scope.SchedulingGroup](2),
				)
				rmg := f.MustCreateResourceManager(ctx,
					data.WithResourceManagerNextGen(),
					data.WithReplicas[scope.ResourceManagerGroup](2),
				)

				f.WaitForPDGroupReady(ctx, pdg)
				f.WaitForTiKVGroupReady(ctx, kvg)
				f.WaitForTiDBGroupReady(ctx, dbg)
				f.WaitForTSOGroupReady(ctx, tg)
				f.WaitForSchedulingGroupReady(ctx, sg)
				f.WaitForResourceManagerGroupReady(ctx, rmg)

				action.TestResourceManagerAvailability(ctx, f, rmg, workload)
			})
		})
})
