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

package ticdc

import (
	"context"

	"github.com/onsi/ginkgo/v2"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apicall"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/cert"
)

var _ = ginkgo.Describe("TiCDC", label.TiCDC, func() {
	f := framework.New()
	f.Setup()

	// NOTE(liubo02): this case is failed in e2e env because of the cgroup v2.
	// Enable it if env is fixed.
	ginkgo.DescribeTableSubtree("Leader Eviction", label.P1,
		func(enableTLS bool) {
			if enableTLS {
				f.SetupCluster(data.WithClusterTLSEnabled())
			}

			ginkgo.It("leader evicted when delete ticdc pod directly", func(ctx context.Context) {
				if enableTLS {
					ns := f.Cluster.Namespace
					cn := f.Cluster.Name
					f.Must(cert.InstallTiDBIssuer(ctx, f.Client, ns, cn))
					f.Must(cert.InstallTiDBCertificates(ctx, f.Client, ns, cn, "dbg"))
					f.Must(cert.InstallTiDBComponentsCertificates(ctx, f.Client, ns, cn, "pdg", "kvg", "dbg", "fg", "cg", "pg"))
				}
				pdg := f.MustCreatePD(ctx)
				kvg := f.MustCreateTiKV(ctx)
				cg := f.MustCreateTiCDC(ctx,
					data.WithReplicas[*runtime.TiCDCGroup](3),
				)

				f.WaitForPDGroupReady(ctx, pdg)
				f.WaitForTiKVGroupReady(ctx, kvg)
				f.WaitForTiCDCGroupReady(ctx, cg)

				cdcs, err := apicall.ListInstances[scope.TiCDCGroup](ctx, f.Client, cg)
				f.Must(err)

				var owner *v1alpha1.TiCDC
				for _, cdc := range cdcs {
					if cdc.Status.IsOwner {
						owner = cdc
					}
				}

				nctx, cancel := context.WithCancel(ctx)
				ch := make(chan struct{})
				go func() {
					defer close(ch)
					defer ginkgo.GinkgoRecover()
					f.WaitTiCDCPreStopHookSuccess(nctx, owner)
				}()

				f.RestartTiCDCPod(ctx, owner)

				cancel()
				<-ch
			})
		},
		func(tls bool) string {
			if tls {
				return "TLS"
			}
			return "NO TLS"
		},
		ginkgo.Entry(nil, false),
		ginkgo.Entry(nil, label.FeatureTLS, true),
	)
})
