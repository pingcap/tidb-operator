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
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apicall"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/cert"
)

const (
	changedConfig = `log-level = 'warn'`
)

var _ = ginkgo.Describe("TiCDC", label.TiCDC, func() {
	f := framework.New()
	f.Setup()

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

	ginkgo.Context("Scale and Update", label.P0, func() {
		ginkgo.It("support scale out/in TiCDC", label.Scale, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx)
			cdcg := f.MustCreateTiCDC(ctx)

			ginkgo.By("Wait for Cluster Ready")
			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			f.WaitForTiCDCGroupReady(ctx, cdcg)

			ginkgo.By("Change replica of the TiCDCGroup to 4")
			patch := client.MergeFrom(cdcg.DeepCopy())
			cdcg.Spec.Replicas = ptr.To[int32](4)
			f.Must(f.Client.Patch(ctx, cdcg, patch))
			f.WaitForTiCDCGroupReady(ctx, cdcg)

			ginkgo.By("Change replica of the TiCDCGroup to 2")
			patch = client.MergeFrom(cdcg.DeepCopy())
			cdcg.Spec.Replicas = ptr.To[int32](2)
			f.Must(f.Client.Patch(ctx, cdcg, patch))
			f.WaitForTiCDCGroupReady(ctx, cdcg)
		})

		ginkgo.It("support scale TiCDC from 3 to 2 and rolling update at same time", label.Scale, label.Update, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx)
			cdcg := f.MustCreateTiCDC(ctx, data.WithReplicas[*runtime.TiCDCGroup](3))

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			f.WaitForTiCDCGroupReady(ctx, cdcg)

			patch := client.MergeFrom(cdcg.DeepCopy())
			cdcg.Spec.Replicas = ptr.To[int32](2)
			cdcg.Spec.Template.Spec.Config = changedConfig

			nctx, cancel := context.WithCancel(ctx)
			ch := make(chan struct{})
			go func() {
				defer close(ch)
				defer ginkgo.GinkgoRecover()
				f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromTiCDCGroup(cdcg), 3, 1, waiter.LongTaskTimeout))
			}()

			maxTime, err := waiter.MaxPodsCreateTimestamp(ctx, f.Client, runtime.FromTiCDCGroup(cdcg))
			f.Must(err)
			changeTime := maxTime.Add(time.Second)

			ginkgo.By("Change config and replicas of the TiCDCGroup")
			f.Must(f.Client.Patch(ctx, cdcg, patch))
			f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiCDCGroup(cdcg), changeTime, waiter.LongTaskTimeout))
			f.WaitForTiCDCGroupReady(ctx, cdcg)
			cancel()
			<-ch
		})
	})
})
