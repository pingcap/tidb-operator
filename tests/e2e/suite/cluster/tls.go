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
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	wopt "github.com/pingcap/tidb-operator/tests/e2e/framework/workload"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/cert"
)

var _ = ginkgo.Describe("TLS", label.Cluster, label.FeatureTLS, func() {
	f := framework.New()
	f.Setup()

	f.DescribeFeatureTable(func(fs ...metav1alpha1.Feature) {
		f.SetupCluster(data.WithClusterTLSEnabled(), data.WithFeatureGates(fs...))
		workload := f.SetupWorkload()
		cm := f.SetupCertManager()
		ginkgo.It("should enable internal TLS with same ca secret", func(ctx context.Context) {
			f.MustCreateS3(ctx)
			ns := f.Namespace.Name
			cluster := f.Cluster.Name

			// add ns prefix of ca because the bundle is a cluster scope resource
			ca := ns + "-cluster-ca"
			mysqlClientCA, mysqlServerCertKeyPair := ns+"-mysql-ca", "mysql-tls"
			pdg := f.MustCreatePD(ctx,
				data.WithMSMode(),
				data.WithPDNextGen(),
				data.WithClusterTLS[*runtime.PDGroup](ca, "pd-internal"),
			)
			tg := f.MustCreateTSO(ctx,
				data.WithTSONextGen(),
				data.WithClusterTLS[*runtime.TSOGroup](ca, "tso-internal"),
			)
			sg := f.MustCreateScheduling(ctx,
				data.WithSchedulingNextGen(),
				data.WithClusterTLS[*runtime.SchedulingGroup](ca, "scheduling-internal"),
			)
			kvg := f.MustCreateTiKV(ctx,
				data.WithTiKVNextGen(),
				data.WithClusterTLS[*runtime.TiKVGroup](ca, "tikv-internal"),
			)
			dbg := f.MustCreateTiDB(ctx,
				data.WithTiDBNextGen(),
				data.WithKeyspace("SYSTEM"),
				data.WithClusterTLS[*runtime.TiDBGroup](ca, "tidb-internal"),
				data.WithTiDBMySQLTLS(mysqlClientCA, mysqlServerCertKeyPair),
			)
			fgc := f.MustCreateTiFlash(ctx,
				data.WithTiFlashNextGen(),
				data.WithTiFlashComputeMode(),
				data.WithClusterTLS[*runtime.TiFlashGroup](ca, "tiflash-compute-internal"),
			)
			fgw := f.MustCreateTiFlash(ctx,
				data.WithTiFlashNextGen(),
				data.WithTiFlashWriteMode(),
				data.WithClusterTLS[*runtime.TiFlashGroup](ca, "tiflash-write-internal"),
			)
			cg := f.MustCreateTiCDC(ctx,
				data.WithClusterTLS[*runtime.TiCDCGroup](ca, "ticdc-internal"),
			)
			pg := f.MustCreateTiProxy(ctx,
				data.WithTiProxyNextGen(),
				data.WithClusterTLS[*runtime.TiProxyGroup](ca, "tiproxy-internal"),
			)

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

			workload.MustPing(ctx, data.DefaultTiDBServiceName, wopt.TLS(cert.MySQLClient(mysqlClientCA, mysqlServerCertKeyPair)))
			workload.MustPing(ctx, data.DefaultTiProxyServiceName, wopt.Port(data.DefaultTiProxyServicePort))
		})
	},
		[]metav1alpha1.Feature{},
		[]metav1alpha1.Feature{
			metav1alpha1.UsePDReadyAPIV2,
			metav1alpha1.UseTSOReadyAPI,
			metav1alpha1.UseSchedulingReadyAPI,
			metav1alpha1.UseTiKVReadyAPI,
			metav1alpha1.UseTiFlashReadyAPI,
		},
		[]metav1alpha1.Feature{
			metav1alpha1.UsePDReadyAPIV2,
			metav1alpha1.UseTSOReadyAPI,
			metav1alpha1.UseSchedulingReadyAPI,
			metav1alpha1.UseTiKVReadyAPI,
			metav1alpha1.UseTiFlashReadyAPI,
			metav1alpha1.ClusterSubdomain,
		},
	)
})
