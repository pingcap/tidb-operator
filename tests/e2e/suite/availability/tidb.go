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

package availability

import (
	"context"

	"github.com/onsi/ginkgo/v2"

	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework/action"
	wopt "github.com/pingcap/tidb-operator/v2/tests/e2e/framework/workload"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/label"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/cert"
)

var _ = ginkgo.Describe("TiDB Availability Test", label.TiDB, label.KindAvail, label.Update, func() {
	f := framework.New()
	f.Setup()

	ginkgo.Context("NextGen", label.KindNextGen, label.P0, func() {
		f.SetupCluster(data.WithClusterTLSEnabled(), data.WithFeatureGates(metav1alpha1.SessionTokenSigning))
		workload := f.SetupWorkload()
		cm := f.SetupCertManager(true)

		// flaky, fix later
		ginkgo.PIt("Visit tiproxy no error when rolling update tidb in next-gen", func(ctx context.Context) {
			ns := f.Namespace.Name
			cluster := f.Cluster.Name

			// add ns prefix of ca because the bundle is a cluster scope resource
			ca := ns + "-cluster-ca"
			mysqlClientCA := ns + "-mysql-ca"

			mysqlServerCertKeyPair := "mysql-tls"

			f.MustCreateS3(ctx)
			pdg := f.MustCreatePD(ctx,
				data.WithPDNextGen(),
				data.WithClusterTLS[scope.PDGroup](ca, "pd-internal"),
			)
			kvg := f.MustCreateTiKV(ctx,
				data.WithTiKVNextGen(),
				data.WithClusterTLS[scope.TiKVGroup](ca, "tikv-internal"),
			)
			dbg := f.MustCreateTiDB(ctx,
				data.WithTiDBNextGen(),
				data.WithKeyspace("SYSTEM"),
				data.WithReplicas[scope.TiDBGroup](2),
				data.WithClusterTLS[scope.TiDBGroup](ca, "tidb-internal"),
			)
			pg := f.MustCreateTiProxy(ctx,
				data.WithTiProxyNextGen(),
				data.WithClusterTLS[scope.TiProxyGroup](ca, "tiproxy-internal"),
				data.WithTiProxyMySQLTLS(mysqlClientCA, mysqlServerCertKeyPair),
			)

			cm.Install(ctx, ns, cluster)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			f.WaitForTiProxyGroupReady(ctx, pg)

			action.TestTiDBAvailability(ctx,
				f,
				data.DefaultTiProxyServiceName,
				dbg,
				workload,
				wopt.Port(data.DefaultTiProxyServicePort),
				// disable sql conn max life time
				wopt.MaxLifeTime(0),
				wopt.TLS(cert.MySQLClient(mysqlClientCA, mysqlServerCertKeyPair)),
			)
		})
	})
})
