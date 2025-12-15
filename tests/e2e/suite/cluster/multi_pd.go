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
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/waiter"
)

var _ = ginkgo.Describe("MultiPDGroup", label.Cluster, func() {
	f := framework.New()
	f.Setup()

	f.Describe(func(o *desc.Options) {
		f.SetupCluster(desc.ClusterPatches(o)...)
		workload := f.SetupWorkload()
		cm := f.SetupCertManager(o.TLS)

		// flaky, fix later
		ginkgo.PIt("should support multiple pd groups", func(ctx context.Context) {
			pdg := action.MustCreatePD(ctx, f, o)
			pdg2 := action.MustCreatePD(ctx, f, o,
				data.WithName[scope.PDGroup]("bootstrapped"),
				data.WithBootstrapped(),
			)
			kvg := action.MustCreateTiKV(ctx, f, o)
			dbg := action.MustCreateTiDB(ctx, f, o)

			cm.Install(ctx, f.Namespace.Name, f.Cluster.Name)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForPDGroupReady(ctx, pdg2)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)

			workload.MustPing(ctx,
				data.DefaultTiDBServiceName,
				wopt.FromDescOption(o),
			)
		})
	},
		[]desc.Option{
			desc.Feature(metav1alpha1.MultiPDGroup),
		},
		[]desc.Option{
			desc.TLS(),
			desc.Feature(metav1alpha1.MultiPDGroup),
		},
	)

	f.Describe(func(o *desc.Options) {
		f.SetupCluster(desc.ClusterPatches(o)...)
		workload := f.SetupWorkload()
		cm := f.SetupCertManager(o.TLS)

		ginkgo.It("should support enable multiple pd groups", func(ctx context.Context) {
			pdg := action.MustCreatePD(ctx, f, o)
			kvg := action.MustCreateTiKV(ctx, f, o)
			dbg := action.MustCreateTiDB(ctx, f, o)

			cm.Install(ctx, f.Namespace.Name, f.Cluster.Name)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)

			fs := []metav1alpha1.Feature{
				metav1alpha1.FeatureModification,
				metav1alpha1.MultiPDGroup,
			}
			fs = append(fs, o.Features...)
			action.MustUpdateCluster(ctx, f,
				data.WithCustomizedPDServiceName(pdg.Name+"-pd"),
				data.WithFeatureGates(fs...),
			)

			pdg2 := action.MustCreatePD(ctx, f, o,
				data.WithName[scope.PDGroup]("bootstrapped"),
				data.WithBootstrapped(),
			)

			cm.Install(ctx, f.Namespace.Name, f.Cluster.Name)

			f.WaitForPDGroupReady(ctx, pdg2)

			action.MustDelete(ctx, f, pdg)
			f.Must(waiter.WaitForObjectDeleted(ctx, f.Client, pdg, waiter.LongTaskTimeout))

			f.WaitForPDGroupReady(ctx, pdg2)

			waiter.WaitForInstanceList[scope.PDGroup](ctx, f.Client, pdg2, waiter.PDHasLeader, waiter.LongTaskTimeout)

			workload.MustPing(ctx,
				data.DefaultTiDBServiceName,
				wopt.FromDescOption(o),
			)
		})
	},
		[]desc.Option{},
		[]desc.Option{
			desc.TLS(),
		},
	)
})
