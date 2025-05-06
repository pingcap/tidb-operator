package framework

import (
	"context"

	"github.com/onsi/ginkgo/v2"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

func (f *Framework) WaitForTSOGroupReady(ctx context.Context, tg *v1alpha1.TSOGroup) {
	// TODO: maybe wait for cluster ready
	ginkgo.By("wait for tso group ready")
	f.Must(waiter.WaitForTSOsHealthy(ctx, f.Client, tg, waiter.LongTaskTimeout))
	f.Must(waiter.WaitForPodsReady(ctx, f.Client, runtime.FromTSOGroup(tg), waiter.LongTaskTimeout))
}
