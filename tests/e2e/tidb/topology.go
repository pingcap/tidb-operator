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

package tidb

import (
	"context"
	"time"

	"github.com/onsi/ginkgo/v2"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

var _ = ginkgo.Describe("Topology", label.TiDB, label.MultipleAZ, label.P0, func() {
	f := framework.New()
	f.Setup()

	ginkgo.It("Create tidb evenly spread in multiple azs", func(ctx context.Context) {
		ginkgo.By("Creating cluster")
		pdg := f.MustCreatePD(ctx)
		kvg := f.MustCreateTiKV(ctx)
		dbg := f.MustCreateTiDB(ctx,
			data.WithReplicas[scope.TiDBGroup](3),
			data.WithTiDBEvenlySpreadPolicy(),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)
		f.WaitForTiDBGroupReady(ctx, dbg)

		f.MustEvenlySpreadTiDB(ctx, dbg)
	})

	ginkgo.It("rolling update and ensure still evenly spread", label.Update, func(ctx context.Context) {
		ginkgo.By("Creating cluster")
		pdg := f.MustCreatePD(ctx)
		kvg := f.MustCreateTiKV(ctx)
		dbg := f.MustCreateTiDB(ctx,
			data.WithReplicas[scope.TiDBGroup](3),
			data.WithTiDBEvenlySpreadPolicy(),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)
		f.WaitForTiDBGroupReady(ctx, dbg)

		f.MustEvenlySpreadTiDB(ctx, dbg)

		patch := client.MergeFrom(dbg.DeepCopy())
		dbg.Spec.Template.Spec.Config = `log.level = 'warn'`

		nctx, cancel := context.WithCancel(ctx)
		ch := make(chan struct{})
		go func() {
			defer close(ch)
			defer ginkgo.GinkgoRecover()
			f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromTiDBGroup(dbg), 3, 1, waiter.LongTaskTimeout))
		}()

		maxTime, err := waiter.MaxPodsCreateTimestamp(ctx, f.Client, runtime.FromTiDBGroup(dbg))
		f.Must(err)
		changeTime := maxTime.Add(time.Second)

		ginkgo.By("rolling update once")
		f.Must(f.Client.Patch(ctx, dbg, patch))
		f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiDBGroup(dbg), changeTime, waiter.LongTaskTimeout))
		f.WaitForTiDBGroupReady(ctx, dbg)
		cancel()
		<-ch
		f.MustEvenlySpreadTiDB(ctx, dbg)
	})

	// a, b -> a, c and then rolling + scale at the same time
	// 1. a, b
	// 2. a(old), b(old), c(new)
	// 3. a(new), b(old), c(new)
	// 4. a, c
	// 5. a(old), c(old), b(new), a(new)
	// 6. a(old), c(new), b(new), a(new)
	// 6. c(new), b(new), a(new)
	ginkgo.It("rolling update and scale + rolling update", label.Scale, label.Update, func(ctx context.Context) {
		ginkgo.By("Creating cluster")
		pdg := f.MustCreatePD(ctx)
		kvg := f.MustCreateTiKV(ctx)
		dbg := f.MustCreateTiDB(ctx,
			data.WithReplicas[scope.TiDBGroup](2),
			data.WithTiDBEvenlySpreadPolicy(),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)
		f.WaitForTiDBGroupReady(ctx, dbg)

		f.MustEvenlySpreadTiDB(ctx, dbg)

		patch := client.MergeFrom(dbg.DeepCopy())
		dbg.Spec.Template.Spec.Config = `log.level = 'warn'`

		nctx, cancel := context.WithCancel(ctx)
		ch := make(chan struct{})
		go func() {
			defer close(ch)
			defer ginkgo.GinkgoRecover()
			f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromTiDBGroup(dbg), 2, 1, waiter.LongTaskTimeout))
		}()

		maxTime, err := waiter.MaxPodsCreateTimestamp(ctx, f.Client, runtime.FromTiDBGroup(dbg))
		f.Must(err)
		changeTime := maxTime.Add(time.Second)

		ginkgo.By("rolling update once")
		f.Must(f.Client.Patch(ctx, dbg, patch))
		f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiDBGroup(dbg), changeTime, waiter.LongTaskTimeout))
		f.WaitForTiDBGroupReady(ctx, dbg)
		cancel()
		<-ch
		f.MustEvenlySpreadTiDB(ctx, dbg)

		patch = client.MergeFrom(dbg.DeepCopy())
		dbg.Spec.Template.Spec.Config = `log.level = 'info'`
		dbg.Spec.Replicas = ptr.To[int32](3)

		nctx, cancel = context.WithCancel(ctx)
		ch = make(chan struct{})
		go func() {
			defer close(ch)
			defer ginkgo.GinkgoRecover()
			f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromTiDBGroup(dbg), 2, 1, waiter.LongTaskTimeout))
		}()

		maxTime, err = waiter.MaxPodsCreateTimestamp(ctx, f.Client, runtime.FromTiDBGroup(dbg))
		f.Must(err)
		changeTime = maxTime.Add(time.Second)

		ginkgo.By("rolling update again")
		f.Must(f.Client.Patch(ctx, dbg, patch))
		f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiDBGroup(dbg), changeTime, waiter.LongTaskTimeout))
		f.WaitForTiDBGroupReady(ctx, dbg)
		cancel()
		<-ch

		f.MustEvenlySpreadTiDB(ctx, dbg)
	})
})
