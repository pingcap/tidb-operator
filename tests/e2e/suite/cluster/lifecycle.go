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

	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework/action"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework/desc"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/label"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/waiter"
)

var _ = ginkgo.Describe("Lifecycle", label.Cluster, func() {
	f := framework.New()
	f.Setup()

	f.Describe(func(o *desc.Options) {
		f.SetupCluster(desc.ClusterPatches(o)...)
		cm := f.SetupCertManager(o.TLS)

		ginkgo.FIt("should support deleting tiflash group", func(ctx context.Context) {
			pdg := action.MustCreatePD(ctx, f, o)
			kvg := action.MustCreateTiKV(ctx, f, o, data.WithReplicas[scope.TiKVGroup](3))
			fg := action.MustCreateTiFlash(ctx, f, o)
			dbg := action.MustCreateTiDB(ctx, f, o)

			cm.Install(ctx, f.Namespace.Name, f.Cluster.Name)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiFlashGroupReady(ctx, fg)
			f.WaitForTiDBGroupReady(ctx, dbg)

			action.MustDelete(ctx, f, fg)
			f.Must(waiter.WaitForObjectDeleted(ctx, f.Client, fg, waiter.LongTaskTimeout))
		})
	},
		[]desc.Option{},
	)
})
