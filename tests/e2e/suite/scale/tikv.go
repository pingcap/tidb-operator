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

	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/framework/action"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

var _ = ginkgo.Describe("Scale TiKV", label.TiKV, label.MultipleAZ, label.P0, label.Scale, func() {
	f := framework.New()
	f.Setup()

	ginkgo.It("support scale out from 3 to 4", func(ctx context.Context) {
		pdg := f.MustCreatePD(ctx)
		kvg := f.MustCreateTiKV(ctx,
			data.WithReplicas[scope.TiKVGroup](3),
			data.WithTiKVEvenlySpreadPolicy(),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)

		framework.MustEvenlySpread[scope.TiKVGroup](ctx, f, kvg)

		action.MustScale[scope.TiKVGroup](ctx, f, kvg, 4)

		f.WaitForTiKVGroupReady(ctx, kvg)
		framework.MustEvenlySpread[scope.TiKVGroup](ctx, f, kvg)
	})

	ginkgo.It("support scale in from 4 to 3", func(ctx context.Context) {
		pdg := f.MustCreatePD(ctx)
		kvg := f.MustCreateTiKV(ctx,
			data.WithReplicas[scope.TiKVGroup](4),
			data.WithTiKVEvenlySpreadPolicy(),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)

		framework.MustEvenlySpread[scope.TiKVGroup](ctx, f, kvg)

		action.MustScale[scope.TiKVGroup](ctx, f, kvg, 3)

		f.WaitForTiKVGroupReady(ctx, kvg)
		framework.MustEvenlySpread[scope.TiKVGroup](ctx, f, kvg)
	})

	ginkgo.It("support scale in from 4 to 3 with a pending pod", func(ctx context.Context) {
		pdg := f.MustCreatePD(ctx)
		kvg := f.MustCreateTiKV(ctx,
			data.WithReplicas[scope.TiKVGroup](3),
			data.WithTiKVEvenlySpreadPolicy(),
			data.WithTiKVPodAntiAffinity(),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)

		framework.MustEvenlySpread[scope.TiKVGroup](ctx, f, kvg)

		action.MustScale[scope.TiKVGroup](ctx, f, kvg, 4)

		framework.WaitForInstanceListSynced[scope.TiKVGroup](ctx, f, kvg)

		action.MustScale[scope.TiKVGroup](ctx, f, kvg, 3)

		f.WaitForTiKVGroupReady(ctx, kvg)
		framework.MustEvenlySpread[scope.TiKVGroup](ctx, f, kvg)
	})

	ginkgo.It("support scale from 3 to 6 and rolling update at same time", ginkgo.Serial, label.Update, func(ctx context.Context) {
		pdg := f.MustCreatePD(ctx)
		kvg := f.MustCreateTiKV(ctx,
			data.WithReplicas[scope.TiKVGroup](3),
			data.WithTiKVEvenlySpreadPolicy(),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)
		framework.MustEvenlySpread[scope.TiKVGroup](ctx, f, kvg)

		nctx, cancel := context.WithCancel(ctx)
		done := framework.AsyncWaitPodsRollingUpdateOnce[scope.TiKVGroup](nctx, f, kvg, 6)
		defer func() { <-done }()
		defer cancel()

		changeTime, err := waiter.MaxPodsCreateTimestamp[scope.TiKVGroup](ctx, f.Client, kvg)
		f.Must(err)

		action.MustScaleAndRollingRestart[scope.TiKVGroup](ctx, f, kvg, 6)

		f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiKVGroup(kvg), *changeTime, waiter.LongTaskTimeout))
		f.WaitForTiKVGroupReady(ctx, kvg)

		framework.MustEvenlySpread[scope.TiKVGroup](ctx, f, kvg)
	})

	ginkgo.It("support scale in from 6 to 4 with a failure AZ", func(ctx context.Context) {
		pdg := f.MustCreatePD(ctx)
		kvg := f.MustCreateTiKV(ctx,
			data.WithReplicas[scope.TiKVGroup](6),
			data.WithTiKVEvenlySpreadPolicyOneFailureAZ(),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		framework.WaitForInstanceListSynced[scope.TiKVGroup](ctx, f, kvg)
		framework.MustEvenlySpread[scope.TiKVGroup](ctx, f, kvg)

		action.MustScale[scope.TiKVGroup](ctx, f, kvg, 4)

		framework.WaitForInstanceListSynced[scope.TiKVGroup](ctx, f, kvg)

		framework.MustEvenlySpread[scope.TiKVGroup](ctx, f, kvg)
	})
})
