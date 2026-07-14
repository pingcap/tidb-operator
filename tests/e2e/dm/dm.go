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

package dm

import (
	"context"

	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/client"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/label"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/waiter"
)

const changedConfig = `log-level = 'warn'`

var _ = ginkgo.Describe("DM", label.DM, func() {
	f := framework.New()
	f.Setup()

	ginkgo.Context("Basic Lifecycle", label.P0, func() {
		ginkgo.It("deploys and reaches Ready state", label.KindBasic, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx)
			dmg := f.MustCreateDM(ctx)
			dwg := f.MustCreateDMWorker(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			f.WaitForDMGroupReady(ctx, dmg)
			f.WaitForDMWorkerGroupReady(ctx, dwg)
		})

		ginkgo.It("deletes groups and owned resources", label.Delete, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dmg := f.MustCreateDM(ctx)
			dwg := f.MustCreateDMWorker(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForDMGroupReady(ctx, dmg)
			f.WaitForDMWorkerGroupReady(ctx, dwg)

			ginkgo.By("Delete DMWorkerGroup")
			f.Must(f.Client.Delete(ctx, dwg))
			f.Must(waiter.WaitForObjectDeleted(ctx, f.Client, dwg, waiter.LongTaskTimeout))
			f.Must(waiter.WaitForPodsDeleted[scope.DMWorkerGroup](ctx, f.Client, dwg, waiter.LongTaskTimeout))
			f.Must(waiter.WaitForListDeleted(ctx, f.Client, &corev1.ConfigMapList{}, waiter.LongTaskTimeout, client.InNamespace(dwg.Namespace), client.MatchingLabels{
				v1alpha1.LabelKeyCluster:   dwg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     dwg.Name,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentDMWorker,
			}))

			ginkgo.By("Delete DMGroup")
			f.Must(f.Client.Delete(ctx, dmg))
			f.Must(waiter.WaitForObjectDeleted(ctx, f.Client, dmg, waiter.LongTaskTimeout))
			f.Must(waiter.WaitForPodsDeleted[scope.DMGroup](ctx, f.Client, dmg, waiter.LongTaskTimeout))
			f.Must(waiter.WaitForListDeleted(ctx, f.Client, &corev1.ConfigMapList{}, waiter.LongTaskTimeout, client.InNamespace(dmg.Namespace), client.MatchingLabels{
				v1alpha1.LabelKeyCluster:   dmg.Spec.Cluster.Name,
				v1alpha1.LabelKeyGroup:     dmg.Name,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentDMMaster,
			}))
		})
	})

	ginkgo.Context("Scale and Update", label.P0, func() {
		ginkgo.It("supports scaling DMWorkerGroup out and in", label.Scale, label.DMWorker, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dmg := f.MustCreateDM(ctx)
			dwg := f.MustCreateDMWorker(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForDMGroupReady(ctx, dmg)
			f.WaitForDMWorkerGroupReady(ctx, dwg)

			ginkgo.By("Scale DMWorkerGroup to 2")
			patch := client.MergeFrom(dwg.DeepCopy())
			dwg.Spec.Replicas = ptr.To[int32](2)
			f.Must(f.Client.Patch(ctx, dwg, patch))
			f.WaitForDMWorkerGroupReady(ctx, dwg)

			ginkgo.By("Scale DMWorkerGroup back to 1")
			patch = client.MergeFrom(dwg.DeepCopy())
			dwg.Spec.Replicas = ptr.To[int32](1)
			f.Must(f.Client.Patch(ctx, dwg, patch))
			f.WaitForDMWorkerGroupReady(ctx, dwg)
		})

		ginkgo.It("supports rolling update of DMGroup", label.Update, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dmg := f.MustCreateDM(ctx, data.WithReplicas[scope.DMGroup](3))
			dwg := f.MustCreateDMWorker(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForDMGroupReady(ctx, dmg)
			f.WaitForDMWorkerGroupReady(ctx, dwg)

			nctx, cancel := context.WithCancel(ctx)
			done := framework.AsyncWaitPodsRollingUpdateOnce[scope.DMGroup](nctx, f, dmg, 3)
			defer func() { <-done }()
			defer cancel()

			changeTime, err := waiter.MaxPodsCreateTimestamp[scope.DMGroup](ctx, f.Client, dmg)
			f.Must(err)

			ginkgo.By("Change config of the DMGroup")
			patch := client.MergeFrom(dmg.DeepCopy())
			dmg.Spec.Template.Spec.Config = changedConfig
			f.Must(f.Client.Patch(ctx, dmg, patch))

			f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromDMGroup(dmg), *changeTime, waiter.LongTaskTimeout))
			f.WaitForDMGroupReady(ctx, dmg)
		})
	})
})
