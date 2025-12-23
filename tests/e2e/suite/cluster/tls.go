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

	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework/action"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework/desc"
	wopt "github.com/pingcap/tidb-operator/v2/tests/e2e/framework/workload"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/label"
)

var _ = ginkgo.Describe("TLS", label.Cluster, label.FeatureTLS, func() {
	f := framework.New()
	f.Setup(framework.WithSkipClusterDeletionWhenFailed())

	f.Describe(func(o *desc.Options) {
		f.SetupCluster(desc.ClusterPatches(o)...)
		workload := f.SetupWorkload()
		cm := f.SetupCertManager(o.TLS)

		ginkgo.It("should enable internal TLS with same ca secret", func(ctx context.Context) {
			f.MustCreateS3(ctx)
			ns := f.Namespace.Name
			cluster := f.Cluster.Name

			ca := o.ClusterCA()
			pdg := action.MustCreatePD(ctx, f, o)
			tg := f.MustCreateTSO(ctx,
				data.WithTSONextGen(),
				data.WithClusterTLS[scope.TSOGroup](ca, "tso-internal"),
			)
			sg := f.MustCreateScheduling(ctx,
				data.WithSchedulingNextGen(),
				data.WithClusterTLS[scope.SchedulingGroup](ca, "scheduling-internal"),
			)
			kvg := action.MustCreateTiKV(ctx, f, o)
			dbg := action.MustCreateTiDB(ctx, f, o)
			fgc := f.MustCreateTiFlash(ctx,
				data.WithTiFlashNextGen(),
				data.WithTiFlashComputeMode(),
				data.WithClusterTLS[scope.TiFlashGroup](ca, "tiflash-compute-internal"),
			)
			fgw := f.MustCreateTiFlash(ctx,
				data.WithTiFlashNextGen(),
				data.WithTiFlashWriteMode(),
				data.WithClusterTLS[scope.TiFlashGroup](ca, "tiflash-write-internal"),
			)
			cg := action.MustCreateTiCDC(ctx, f, o)

			pg := action.MustCreateTiProxy(ctx, f, o)

			wg := action.MustCreateTiKVWorker(ctx, f, o)

			cm.Install(ctx, ns, cluster)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTSOGroupReady(ctx, tg)
			f.WaitForSchedulingGroupReady(ctx, sg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			f.WaitForTiFlashGroupReady(ctx, fgc)
			f.WaitForTiFlashGroupReady(ctx, fgw)
			f.WaitForTiCDCGroupReady(ctx, cg)
			f.WaitForTiProxyGroupReady(ctx, pg)
			f.WaitForTiKVWorkerGroupReady(ctx, wg)

			workload.MustPing(ctx, data.DefaultTiDBServiceName, wopt.FromDescOption(o))
			workload.MustPing(ctx, data.DefaultTiProxyServiceName, wopt.Port(data.DefaultTiProxyServicePort))
		})
	},
		[]desc.Option{
			desc.TLS(),
			desc.NextGen(),
		},
		[]desc.Option{
			desc.TLS(),
			desc.NextGen(),
			desc.Features(
				metav1alpha1.UsePDReadyAPIV2,
				metav1alpha1.UseTSOReadyAPI,
				metav1alpha1.UseSchedulingReadyAPI,
				metav1alpha1.UseTiKVReadyAPI,
				metav1alpha1.UseTiFlashReadyAPI,
			),
		},
		[]desc.Option{
			desc.TLS(),
			desc.NextGen(),
			desc.Features(
				metav1alpha1.UsePDReadyAPIV2,
				metav1alpha1.UseTSOReadyAPI,
				metav1alpha1.UseSchedulingReadyAPI,
				metav1alpha1.UseTiKVReadyAPI,
				metav1alpha1.UseTiFlashReadyAPI,
				metav1alpha1.ClusterSubdomain,
			),
		},
	)
})
