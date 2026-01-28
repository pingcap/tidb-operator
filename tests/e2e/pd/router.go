// Copyright 2026 PingCAP, Inc.
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

var _ = ginkgo.Describe("PD", label.PD, label.FeaturePDMS, func() {
	f := framework.New()
	f.Setup()

	ginkgo.Context("PDMS Router", label.P0, func() {
		ginkgo.It("support create router with 1 replica", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx,
				data.WithVersion[scope.PDGroup](PDMSVersion),
				data.WithMSMode(),
				data.WithReplicas[scope.PDGroup](1),
			)
			rg := data.NewRouterGroup(f.Namespace.Name,
				data.WithVersion[scope.RouterGroup](PDMSVersion),
				data.WithReplicas[scope.RouterGroup](1),
			)
			ginkgo.By("Creating a router group")
			f.Must(f.Client.Create(ctx, rg))

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForRouterGroupReady(ctx, rg)
		})

		ginkgo.It("support create router with 2 replicas", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx,
				data.WithVersion[scope.PDGroup](PDMSVersion),
				data.WithMSMode(),
				data.WithReplicas[scope.PDGroup](1),
			)
			rg := data.NewRouterGroup(f.Namespace.Name,
				data.WithVersion[scope.RouterGroup](PDMSVersion),
				data.WithReplicas[scope.RouterGroup](2),
			)
			ginkgo.By("Creating a router group")
			f.Must(f.Client.Create(ctx, rg))

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForRouterGroupReady(ctx, rg)
		})
	})
})
