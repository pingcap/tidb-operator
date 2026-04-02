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

	"github.com/onsi/ginkgo/v2"

	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework/action"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework/desc"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/label"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/waiter"
)

var _ = ginkgo.Describe("Scale TiDB", label.TiDB, label.P0, label.Scale, func() {
	f := framework.New()
	f.Setup()

	ginkgo.It("support scale from 3 to 5 and rolling restart at same time with maxSurge 2", label.Update, func(ctx context.Context) {
		o := desc.DefaultOptions()

		pdg := action.MustCreatePD(ctx, f, o)
		kvg := action.MustCreateTiKV(ctx, f, o)
		dbg := action.MustCreateTiDB(ctx, f, o,
			data.WithReplicas[scope.TiDBGroup](3),
			data.WithTiDBMaxSurge(2),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)
		f.WaitForTiDBGroupReady(ctx, dbg)

		nctx, cancel := context.WithCancel(ctx)
		done := framework.AsyncWaitPodsRollingUpdateOnce[scope.TiDBGroup](nctx, f, dbg, 5)
		defer func() { <-done }()
		defer cancel()

		// Record the latest old pod creation timestamp so we can verify the rolling restart
		// actually creates new pods instead of only scaling out.
		changeTime, err := waiter.MaxPodsCreateTimestamp[scope.TiDBGroup](ctx, f.Client, dbg)
		f.Must(err)

		// Patch replicas and pod template at the same time to force the updater into the
		// "scale out + rolling restart" path where spec.maxSurge should be honored.
		action.MustScaleAndRollingRestart[scope.TiDBGroup](ctx, f, dbg, 5)

		f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiDBGroup(dbg), *changeTime, waiter.LongTaskTimeout))
		f.WaitForTiDBGroupReady(ctx, dbg)
	})
})
