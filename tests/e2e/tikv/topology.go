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

package tikv

import (
	"context"

	"github.com/onsi/ginkgo/v2"

	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/label"
)

var _ = ginkgo.Describe("Topology", label.TiKV, label.MultipleAZ, label.P0, func() {
	f := framework.New()
	f.Setup()

	ginkgo.It("Create tikv evenly spread in multiple azs", func(ctx context.Context) {
		ginkgo.By("Creating cluster")
		pdg := f.MustCreatePD(ctx)
		kvg := f.MustCreateTiKV(ctx,
			data.WithReplicas[scope.TiKVGroup](6),
			data.WithTiKVEvenlySpreadPolicy(),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)

		framework.MustEvenlySpread[scope.TiKVGroup](ctx, f, kvg)
	})
})
