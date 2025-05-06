package pd

import (
	"context"

	"github.com/onsi/ginkgo/v2"

	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
)

var _ = ginkgo.Describe("PD", label.PD, label.FeaturePDMS, func() {
	f := framework.New()
	f.Setup()

	ginkgo.Context("PDMS Basic", label.P0, func() {
		ginkgo.It("support create PD and TSO with 1 replica", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx, data.WithMSMode(), data.WithReplicas[*runtime.PDGroup](1))
			tg := f.MustCreateTSO(ctx, data.WithReplicas[*runtime.TSOGroup](1))

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTSOGroupReady(ctx, tg)
		})

		ginkgo.It("support create 3 PD instances and 2 TSO instances", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx, data.WithMSMode(), data.WithReplicas[*runtime.PDGroup](3))
			tg := f.MustCreateTSO(ctx, data.WithReplicas[*runtime.TSOGroup](2))

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTSOGroupReady(ctx, tg)
		})
	})
})
