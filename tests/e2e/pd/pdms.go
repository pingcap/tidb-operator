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

	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/label"
)

const PDMSVersion = "v8.3.0"

var _ = ginkgo.Describe("PD", label.PD, label.FeaturePDMS, func() {
	f := framework.New()
	f.Setup()

	ginkgo.Context("PDMS Basic", label.P0, func() {
		ginkgo.It("support create PD, TSO, and scheduling with 1 replica", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx,
				data.WithVersion[scope.PDGroup](PDMSVersion),
				data.WithMSMode(),
				data.WithReplicas[scope.PDGroup](1),
			)
			tg := f.MustCreateTSO(ctx,
				data.WithVersion[scope.TSOGroup](PDMSVersion),
				data.WithReplicas[scope.TSOGroup](1),
			)
			sg := f.MustCreateScheduling(ctx,
				data.WithVersion[scope.SchedulingGroup](PDMSVersion),
				data.WithReplicas[scope.SchedulingGroup](1),
			)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTSOGroupReady(ctx, tg)
			f.WaitForSchedulingGroupReady(ctx, sg)
		})

		ginkgo.It("support create 3 PD instances, 2 TSO instances, and 2 scheduling instances", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx,
				data.WithVersion[scope.PDGroup](PDMSVersion),
				data.WithMSMode(),
				data.WithReplicas[scope.PDGroup](3),
			)
			tg := f.MustCreateTSO(ctx,
				data.WithVersion[scope.TSOGroup](PDMSVersion),
				data.WithReplicas[scope.TSOGroup](2),
			)
			sg := f.MustCreateScheduling(ctx,
				data.WithVersion[scope.SchedulingGroup](PDMSVersion),
				data.WithReplicas[scope.SchedulingGroup](2),
			)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTSOGroupReady(ctx, tg)
			f.WaitForSchedulingGroupReady(ctx, sg)
		})
	})
})
