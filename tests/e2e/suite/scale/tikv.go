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

package scale

import (
	"context"
	"time"

	"github.com/onsi/ginkgo/v2"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/pkg/runtime/scope"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

var _ = ginkgo.Describe("Scale TiKV", label.TiKV, label.MultipleAZ, label.P0, label.Scale, func() {
	f := framework.New()
	f.Setup()

	ginkgo.It("support scale out from 3 to 4", func(ctx context.Context) {
		pdg := f.MustCreatePD(ctx)
		kvg := f.MustCreateTiKV(ctx,
			data.WithReplicas[scope.TiKVGroup](3),
			data.WithTiKVEvenlySpreadPolicy(),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)

		f.MustEvenlySpreadTiKV(ctx, kvg)

		patch := client.MergeFrom(kvg.DeepCopy())
		kvg.Spec.Replicas = ptr.To[int32](4)

		f.Must(f.Client.Patch(ctx, kvg, patch))
		f.WaitForTiKVGroupReady(ctx, kvg)
		f.MustEvenlySpreadTiKV(ctx, kvg)
	})

	ginkgo.It("support scale in from 4 to 3", func(ctx context.Context) {
		pdg := f.MustCreatePD(ctx)
		kvg := f.MustCreateTiKV(ctx,
			data.WithReplicas[scope.TiKVGroup](4),
			data.WithTiKVEvenlySpreadPolicy(),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)

		f.MustEvenlySpreadTiKV(ctx, kvg)

		patch := client.MergeFrom(kvg.DeepCopy())
		kvg.Spec.Replicas = ptr.To[int32](3)

		f.Must(f.Client.Patch(ctx, kvg, patch))
		f.WaitForTiKVGroupReady(ctx, kvg)
		f.MustEvenlySpreadTiKV(ctx, kvg)
	})

	ginkgo.FIt("support scale in from 4 to 3 with a pending pod", func(ctx context.Context) {
		pdg := f.MustCreatePD(ctx)
		kvg := f.MustCreateTiKV(ctx,
			data.WithReplicas[scope.TiKVGroup](3),
			data.WithTiKVEvenlySpreadPolicy(),
			data.WithTiKVPodAntiAffinity(),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)

		f.MustEvenlySpreadTiKV(ctx, kvg)

		patch := client.MergeFrom(kvg.DeepCopy())
		kvg.Spec.Replicas = ptr.To[int32](4)
		f.Must(f.Client.Patch(ctx, kvg, patch))

		// just wait a few seconds
		time.Sleep(time.Second * 15)

		patch2 := client.MergeFrom(kvg.DeepCopy())
		kvg.Spec.Replicas = ptr.To[int32](3)

		f.Must(f.Client.Patch(ctx, kvg, patch2))
		f.WaitForTiKVGroupReady(ctx, kvg)
		f.MustEvenlySpreadTiKV(ctx, kvg)
	})

	ginkgo.It("support scale from 3 to 6 and rolling update at same time", ginkgo.Serial, label.Update, func(ctx context.Context) {
		pdg := f.MustCreatePD(ctx)
		kvg := f.MustCreateTiKV(ctx,
			data.WithReplicas[scope.TiKVGroup](3),
			data.WithTiKVEvenlySpreadPolicy(),
		)

		f.WaitForPDGroupReady(ctx, pdg)
		f.WaitForTiKVGroupReady(ctx, kvg)
		f.MustEvenlySpreadTiKV(ctx, kvg)

		patch := client.MergeFrom(kvg.DeepCopy())
		kvg.Spec.Replicas = ptr.To[int32](6)
		kvg.Spec.Template.Annotations = map[string]string{
			"test": "test",
		}

		nctx, cancel := context.WithCancel(ctx)
		ch := make(chan struct{})
		go func() {
			defer close(ch)
			defer ginkgo.GinkgoRecover()
			f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromTiKVGroup(kvg), 3, 0, waiter.LongTaskTimeout))
		}()

		changeTime, err := waiter.MaxPodsCreateTimestamp(ctx, f.Client, runtime.FromTiKVGroup(kvg))
		f.Must(err)

		ginkgo.By("Change config and replicas of the TiKVGroup")
		f.Must(f.Client.Patch(ctx, kvg, patch))
		f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiKVGroup(kvg), *changeTime, waiter.LongTaskTimeout))
		f.WaitForTiKVGroupReady(ctx, kvg)
		cancel()
		<-ch

		f.MustEvenlySpreadTiKV(ctx, kvg)
	})
})
