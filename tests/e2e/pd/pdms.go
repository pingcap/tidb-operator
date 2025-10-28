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

package pd

import (
	"context"

	"github.com/onsi/ginkgo/v2"

	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
)

const PDMSVersion = "v8.3.0"

var _ = ginkgo.Describe("PD", label.PD, label.FeaturePDMS, func() {
	f := framework.New()
	f.Setup()

	ginkgo.Context("PDMS Basic", label.P0, func() {
		ginkgo.It("support create PD, TSO, and scheduling with 1 replica", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx,
				data.WithGroupVersion[*runtime.PDGroup](PDMSVersion),
				data.WithMSMode(),
				data.WithReplicas[*runtime.PDGroup](1),
			)
			tg := f.MustCreateTSO(ctx,
				data.WithGroupVersion[*runtime.TSOGroup](PDMSVersion),
				data.WithReplicas[*runtime.TSOGroup](1),
			)
			sg := f.MustCreateScheduling(ctx,
				data.WithGroupVersion[*runtime.SchedulingGroup](PDMSVersion),
				data.WithReplicas[*runtime.SchedulingGroup](1),
			)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTSOGroupReady(ctx, tg)
			f.WaitForSchedulingGroupReady(ctx, sg)
		})

		ginkgo.It("support create 3 PD instances, 2 TSO instances, and 2 scheduling instances", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx,
				data.WithGroupVersion[*runtime.PDGroup](PDMSVersion),
				data.WithMSMode(),
				data.WithReplicas[*runtime.PDGroup](3),
			)
			tg := f.MustCreateTSO(ctx,
				data.WithGroupVersion[*runtime.TSOGroup](PDMSVersion),
				data.WithReplicas[*runtime.TSOGroup](2),
			)
			sg := f.MustCreateScheduling(ctx,
				data.WithGroupVersion[*runtime.SchedulingGroup](PDMSVersion),
				data.WithReplicas[*runtime.SchedulingGroup](2),
			)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTSOGroupReady(ctx, tg)
			f.WaitForSchedulingGroupReady(ctx, sg)
		})
	})

	ginkgo.Context("NextGen PDMS",
		label.P0,
		label.KindNextGen,
		label.Features(
			metav1alpha1.UsePDReadyAPIV2,
			metav1alpha1.UseTSOReadyAPI,
			metav1alpha1.UseSchedulingReadyAPI,
		),
		func() {
			f.SetupCluster(data.WithFeatureGates(
				metav1alpha1.UseTSOReadyAPI,
				metav1alpha1.UsePDReadyAPIV2,
				metav1alpha1.UseSchedulingReadyAPI,
			))
			ginkgo.It("support create 3 PD instances, 2 TSO instances, and 2 scheduling instances", func(ctx context.Context) {
				f.MustCreateS3(ctx)
				pdg := f.MustCreatePD(ctx,
					data.WithPDNextGen(),
					data.WithMSMode(),
					data.WithReplicas[*runtime.PDGroup](3),
				)
				tg := f.MustCreateTSO(ctx,
					data.WithTSONextGen(),
					data.WithReplicas[*runtime.TSOGroup](2),
				)
				sg := f.MustCreateScheduling(ctx,
					data.WithSchedulingNextGen(),
					data.WithReplicas[*runtime.SchedulingGroup](2),
				)

				f.WaitForPDGroupReady(ctx, pdg)
				f.WaitForTSOGroupReady(ctx, tg)
				f.WaitForSchedulingGroupReady(ctx, sg)
			})
		})
})
