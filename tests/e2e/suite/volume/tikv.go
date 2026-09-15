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

package volume

import (
	"context"

	"github.com/onsi/ginkgo/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework/action"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/label"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/waiter"
)

var _ = ginkgo.Describe("TiKV volume", label.TiKV, label.P1, label.Scale,
	label.Features(metav1alpha1.VolumeAttributesClass), func() {
		f := framework.New()
		f.Setup()
		f.SetupCluster(data.WithFeatureGates(
			metav1alpha1.DisablePDDefaultReadinessProbe,
			metav1alpha1.VolumeAttributesClass,
		))

		ginkgo.It("reduces volume requests without restarting and replaces larger volumes through scale out and scale in", func(ctx context.Context) {
			const replicas = 3
			large, small := resource.MustParse("2Gi"), resource.MustParse("1Gi")
			pdg := f.MustCreatePD(ctx)
			// Use a single topology so placement preferences do not restrict which old stores can leave.
			kvg := f.MustCreateTiKV(ctx, data.WithReplicas[scope.TiKVGroup](replicas),
				data.WithTiKVVolumeStorage("data", large),
			)
			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)

			f.Must(waiter.WaitForVolumeCapacityExceedsRequest[scope.TiKVGroup](ctx, f.Client, kvg, 0, waiter.LongTaskTimeout))

			ginkgo.By("Reducing the volume request from 2Gi to 1Gi without restarting instances")
			shrinkCtx, shrinkCancel := context.WithCancel(ctx)
			shrinkDone := framework.AsyncWaitPodsRollingUpdateOnce[scope.TiKVGroup](shrinkCtx, f, kvg, replicas, true)
			defer func() { shrinkCancel(); <-shrinkDone }()
			action.MustUpdate[scope.TiKVGroup](ctx, f, kvg, data.WithTiKVVolumeStorage("data", small))
			f.Must(waiter.WaitForObjectCondition[scope.TiKVGroup](ctx, f.Client, kvg,
				v1alpha1.CondSynced, metav1.ConditionTrue, waiter.LongTaskTimeout))
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.Must(waiter.WaitForVolumeCapacityExceedsRequest[scope.TiKVGroup](ctx, f.Client, kvg, replicas, waiter.LongTaskTimeout))

			shrinkCancel()
			<-shrinkDone

			ginkgo.By("Scaling out to six replicas and verifying three new 1Gi volumes")
			outCtx, outCancel := context.WithCancel(ctx)
			outDone := framework.AsyncWaitPodsRollingUpdateOnce[scope.TiKVGroup](outCtx, f, kvg, 2*replicas, true)
			defer func() { outCancel(); <-outDone }()
			action.MustScale[scope.TiKVGroup](ctx, f, kvg, 2*replicas)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.Must(waiter.WaitForVolumeCapacityExceedsRequest[scope.TiKVGroup](ctx, f.Client, kvg, replicas, waiter.LongTaskTimeout))

			outCancel()
			<-outDone

			ginkgo.By("Scaling back to three replicas and retaining only the new smaller-volume instances")
			inCtx, inCancel := context.WithCancel(ctx)
			inDone := framework.AsyncWaitPodsRollingUpdateOnce[scope.TiKVGroup](inCtx, f, kvg, replicas, true)
			defer func() { inCancel(); <-inDone }()
			action.MustScale[scope.TiKVGroup](ctx, f, kvg, replicas)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.Must(waiter.WaitForVolumeCapacityExceedsRequest[scope.TiKVGroup](ctx, f.Client, kvg, 0, waiter.LongTaskTimeout))
			inCancel()
			<-inDone
			// PVC retention follows the cluster policy; all remaining instances must use the new small volumes.
		})
	})
