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

package tidb

import (
	"context"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apicall"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
)

var _ = ginkgo.Describe("StandBy", label.TiDB, label.KindNextGen, func() {
	f := framework.New()
	f.Setup()
	f.SetupCluster(data.WithFeatureGates(
		metav1alpha1.FeatureModification,
		metav1alpha1.ClusterSubdomain,
	))

	ginkgo.PIt("Adopt", label.P1, func(ctx context.Context) {
		pdg := f.MustCreatePD(ctx,
			data.WithGroupImage[*runtime.PDGroup]("gcr.io/pingcap-public/dbaas/pd:master-next-gen"),
		)
		kvg := f.MustCreateTiKV(ctx,
			data.WithGroupImage[*runtime.TiKVGroup]("gcr.io/pingcap-public/dbaas/tikv:dedicated-next-gen"),
			data.WithTiKVAPIVersionV2(),
		)
		dbg := f.MustCreateTiDB(ctx,
			data.WithGroupName[*runtime.TiDBGroup]("system"),
			data.WithGroupImage[*runtime.TiDBGroup]("gcr.io/pingcap-public/dbaas/tidb:master-next-gen"),
			data.WithTiDBCommandProbe(),
			// system keyspace
			data.WithKeyspace("SYSTEM"),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)
		f.WaitForTiDBGroupReady(ctx, dbg)

		f.MustScale(ctx, dbg, 0)

		standby := f.MustCreateTiDB(ctx,
			data.WithGroupName[*runtime.TiDBGroup]("standby-tidb"),
			data.WithGroupImage[*runtime.TiDBGroup]("gcr.io/pingcap-public/dbaas/tidb:master-next-gen"),
			data.WithTiDBCommandProbe(),
			data.WithReplicas[*runtime.TiDBGroup](2),
			data.WithTiDBStandbyMode(),
		)

		f.WaitForTiDBGroupReady(ctx, standby)

		another := f.MustCreateTiDB(ctx,
			data.WithGroupName[*runtime.TiDBGroup]("user"),
			data.WithGroupImage[*runtime.TiDBGroup]("gcr.io/pingcap-public/dbaas/tidb:master-next-gen"),
			data.WithTiDBCommandProbe(),
			// system keyspace
			data.WithKeyspace("SYSTEM"),
		)
		f.MustScale(ctx, dbg, 1)

		f.WaitForTiDBGroupReady(ctx, dbg)
		f.WaitForTiDBGroupReady(ctx, another)
		f.WaitForTiDBGroupReady(ctx, standby)

		dbs, err := apicall.ListClusterInstances[scope.TiDB](ctx, f.Client, f.Namespace.Name, f.Cluster.Name)
		f.Must(err)

		createdByStandbyGroup := 0
		for _, db := range dbs {
			if strings.HasPrefix(db.Name, "standby-tidb") {
				createdByStandbyGroup++
			}
		}

		// 2 new + 1 taken by system + 1 taken by user
		gomega.Expect(createdByStandbyGroup).To(gomega.Equal(4))
	})
})
