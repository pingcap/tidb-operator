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

package scale

import (
	"context"
	"time"

	"github.com/onsi/ginkgo/v2"

	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework/action"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/label"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/waiter"
)

var _ = ginkgo.Describe("Scale PD", label.PD, label.Scale, func() {
	f := framework.New()
	f.Setup()

	f.DescribeFeatureTable(func(fs ...metav1alpha1.Feature) {
		f.SetupCluster(data.WithFeatureGates(fs...))

		ginkgo.DescribeTable("scale",
			func(ctx context.Context, from, to int) {
				pdg := f.MustCreatePD(
					ctx,
					data.WithReplicas[scope.PDGroup](int32(from)),
					data.WithPDFeatures(fs...),
				)
				f.WaitForPDGroupReady(ctx, pdg)

				action.MustScale[scope.PDGroup](ctx, f, pdg, int32(to))

				f.WaitForPDGroupReady(ctx, pdg)
			},
			ginkgo.Entry("1 to 3", 1, 3),
			ginkgo.Entry("3 to 1", 3, 1),
			ginkgo.Entry("3 to 5", 3, 5),
			ginkgo.Entry("5 to 3", 5, 3),
			ginkgo.Entry("1 to 3", 1, 3),
			ginkgo.Entry("1 to 3", 1, 3),
		)

		ginkgo.DescribeTable("scale and rolling restart PD instance",
			func(ctx context.Context, from, to int) {
				pdg := f.MustCreatePD(ctx,
					data.WithReplicas[scope.PDGroup](int32(from)),
					data.WithPDFeatures(fs...),
				)
				f.WaitForPDGroupReady(ctx, pdg)

				nctx, cancel := context.WithCancel(ctx)
				done := framework.AsyncWaitPodsRollingUpdateOnce[scope.PDGroup](nctx, f, pdg, to)
				defer func() { <-done }()
				defer cancel()

				changeTime := time.Now()

				action.MustScaleAndRollingRestart[scope.PDGroup](ctx, f, pdg, int32(to))

				f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromPDGroup(pdg), changeTime, waiter.LongTaskTimeout))
				f.WaitForPDGroupReady(ctx, pdg)
			},
			ginkgo.Entry("3 to 5", 3, 5),
			ginkgo.Entry("5 to 3", 5, 3),
		)

		ginkgo.DescribeTable("scale and rolling restart resource manager",
			func(ctx context.Context, from, to int) {
				pdg := f.MustCreatePD(ctx,
					data.WithReplicas[scope.PDGroup](int32(from)),
					data.WithPDNextGen(),
					data.WithMSMode(),
					data.WithResourceManager(),
				)
				rmg := f.MustCreateResourceManager(ctx,
					data.WithResourceManagerNextGen(),
					data.WithReplicas[scope.ResourceManagerGroup](int32(from)),
				)

				f.WaitForPDGroupReady(ctx, pdg)
				f.WaitForResourceManagerGroupReady(ctx, rmg)

				nctx, cancel := context.WithCancel(ctx)
				done := framework.AsyncWaitPodsRollingUpdateOnce[scope.ResourceManagerGroup](nctx, f, rmg, to)
				defer func() { <-done }()
				defer cancel()

				changeTime := time.Now()

				action.MustScaleAndRollingRestart[scope.ResourceManagerGroup](ctx, f, rmg, int32(to))

				f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromResourceManagerGroup(rmg), changeTime, waiter.LongTaskTimeout))
				f.WaitForResourceManagerGroupReady(ctx, rmg)
			},
			ginkgo.Entry("3 to 5", 3, 5),
			ginkgo.Entry("5 to 3", 5, 3),
		)

		ginkgo.It("wrong image", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx,
				data.WithReplicas[scope.PDGroup](3),
				data.WithPDFeatures(fs...),
				// set image to invalid
				data.WithImage[scope.PDGroup]("invalid"),
			)
			framework.WaitForInstanceListSynced[scope.PDGroup](ctx, f, pdg)

			action.MustUpdate[scope.PDGroup](ctx, f, pdg,
				// reset image
				data.WithPDFeatures(fs...),
			)
			f.WaitForPDGroupReady(ctx, pdg)
		})
	},
		nil,
		[]metav1alpha1.Feature{metav1alpha1.UsePDReadyAPIV2},
	)
})
