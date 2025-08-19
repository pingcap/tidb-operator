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

package cluster

import (
	"context"

	"github.com/onsi/ginkgo/v2"

	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
)

var _ = ginkgo.Describe("TLS", label.Cluster, label.FeatureTLS, func() {
	f := framework.New()
	f.Setup()
	f.SetupCluster(data.WithClusterTLSEnabled())
	workload := f.SetupWorkload()
	cm := f.SetupCertManager()

	ginkgo.It("should enable internal TLS with same ca secret", func(ctx context.Context) {
		ns := f.Namespace.Name
		cluster := f.Cluster.Name

		ca := f.Cluster.Name + "-ca"
		pdg := f.MustCreatePD(ctx,
			data.WithMSMode(),
			data.WithClusterTLS[*runtime.PDGroup](ca, "pd-internal"),
		)
		tg := f.MustCreateTSO(ctx,
			data.WithClusterTLS[*runtime.TSOGroup](ca, "tso-internal"),
		)
		sg := f.MustCreateScheduler(ctx,
			data.WithClusterTLS[*runtime.SchedulerGroup](ca, "scheduler-internal"),
		)
		kvg := f.MustCreateTiKV(ctx,
			data.WithClusterTLS[*runtime.TiKVGroup](ca, "tikv-internal"),
		)
		dbg := f.MustCreateTiDB(ctx,
			data.WithClusterTLS[*runtime.TiDBGroup](ca, "tidb-internal"),
		)
		fg := f.MustCreateTiFlash(ctx,
			data.WithClusterTLS[*runtime.TiFlashGroup](ca, "tiflash-internal"),
		)
		cg := f.MustCreateTiCDC(ctx,
			data.WithClusterTLS[*runtime.TiCDCGroup](ca, "ticdc-internal"),
		)
		// TODO: Ignore tiproxy until e2e env is fixed
		// pg := f.MustCreateTiProxy(ctx,
		// 	data.WithClusterTLS[*runtime.TiProxyGroup](ca, "tiproxy-internal"),
		// )

		cm.Install(ctx, ns, cluster)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTSOGroupReady(ctx, tg)
		f.WaitForSchedulerGroupReady(ctx, sg)
		f.WaitForTiKVGroupReady(ctx, kvg)
		f.WaitForTiDBGroupReady(ctx, dbg)
		f.WaitForTiFlashGroupReady(ctx, fg)
		f.WaitForTiCDCGroupReady(ctx, cg)
		// f.WaitForTiProxyGroupReady(ctx, pg)

		workload.MustPing(ctx, data.DefaultTiDBServiceName)
		// workload.MustPing(ctx, data.DefaultTiProxyServiceName, wopt.Port(data.DefaultTiProxyServicePort))
	})
})
