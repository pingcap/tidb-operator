package tidb

import (
	"context"
	"time"

	"github.com/onsi/ginkgo/v2"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/pkg/runtime"
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
			data.WithReplicas[*runtime.TiDBGroup](3),
			data.WithTiDBEvenlySpreadPolicy(),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)
		f.WaitForTiDBGroupReady(ctx, dbg)

		f.MustEvenlySpreadTiDB(ctx, dbg)
	})

	ginkgo.It("rolling update and scale + rolling update", label.Scale, label.Update, func(ctx context.Context) {
		ginkgo.By("Creating cluster")
		pdg := f.MustCreatePD(ctx)
		kvg := f.MustCreateTiKV(ctx)
		dbg := f.MustCreateTiDB(ctx,
			data.WithReplicas[*runtime.TiDBGroup](2),
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
