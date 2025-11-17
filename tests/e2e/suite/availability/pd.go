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

	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/framework/action"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
)

var _ = ginkgo.Describe("PD Availability Test", label.PD, label.KindAvail, label.Update, func() {
	f := framework.New()
	f.Setup()
	f.SetupCluster(data.WithFeatureGates())
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
})
