package tikv

import (
	"context"
	"sync"
	"time"

	"github.com/onsi/ginkgo/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

var _ = ginkgo.Describe("TiKV rolling update", label.TiKV, func() {
	f := framework.New()
	f.Setup()
	ginkgo.Context("Rolling update", label.P0, label.Update, func() {
		workload := f.SetupWorkload()
		ginkgo.It("No error when rolling update tikv", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx, data.WithReplicas[*runtime.TiKVGroup](3))
			dbg := f.MustCreateTiDB(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)

			patch := client.MergeFrom(kvg.DeepCopy())
			kvg.Spec.Template.Spec.Config = `log.level = 'warn'`

			nctx, cancel := context.WithCancel(ctx)
			wg := sync.WaitGroup{}
			wg.Add(2)
			go func() {
				defer wg.Done()
				defer ginkgo.GinkgoRecover()
				f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromTiKVGroup(kvg), 3, 0, waiter.LongTaskTimeout))
			}()
			go func() {
				defer wg.Done()
				defer ginkgo.GinkgoRecover()
				workload.MustRunWorkload(ctx, data.DefaultTiDBServiceName)
			}()

			changeTime := time.Now()
			ginkgo.By("Change config of the TiKVGroup")
			f.Must(f.Client.Patch(ctx, kvg, patch))
			f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiKVGroup(kvg), changeTime, waiter.LongTaskTimeout))
			f.WaitForTiKVGroupReady(ctx, kvg)
			cancel()
			wg.Wait()
		})
	})
})
