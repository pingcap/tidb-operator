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

package pd

import (
	"context"
	"time"

	"github.com/onsi/ginkgo/v2"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

var _ = ginkgo.Describe("PD", label.PD, func() {
	f := framework.New()
	f.Setup()

	ginkgo.Context("Basic", label.P0, func() {
		ginkgo.It("support create PD with 1 replica", func(ctx context.Context) {
			pdg := data.NewPDGroup(
				f.Namespace.Name,
				data.WithReplicas[*runtime.PDGroup](1),
			)

			ginkgo.By("Create PDGroup")
			f.Must(f.Client.Create(ctx, pdg))
			f.WaitForPDGroupReady(ctx, pdg)
		})

		ginkgo.It("support create PD with 3 replica", func(ctx context.Context) {
			pdg := data.NewPDGroup(
				f.Namespace.Name,
				data.WithReplicas[*runtime.PDGroup](3),
			)

			ginkgo.By("Create PDGroup")
			f.Must(f.Client.Create(ctx, pdg))
			f.WaitForPDGroupReady(ctx, pdg)
		})
	})

	ginkgo.Context("Scale and Update", label.P0, func() {
		ginkgo.It("support scale PD form 3 to 5", label.Scale, func(ctx context.Context) {
			pdg := data.NewPDGroup(
				f.Namespace.Name,
				data.WithReplicas[*runtime.PDGroup](3),
			)

			ginkgo.By("Create PDGroup")
			f.Must(f.Client.Create(ctx, pdg))
			f.WaitForPDGroupReady(ctx, pdg)

			patch := client.MergeFrom(pdg.DeepCopy())
			pdg.Spec.Replicas = ptr.To[int32](5)

			ginkgo.By("Change replica of the PDGroup")
			f.Must(f.Client.Patch(ctx, pdg, patch))
			f.WaitForPDGroupReady(ctx, pdg)
		})

		ginkgo.It("support scale PD form 5 to 3", label.Scale, func(ctx context.Context) {
			pdg := data.NewPDGroup(
				f.Namespace.Name,
				data.WithReplicas[*runtime.PDGroup](5),
			)

			ginkgo.By("Create PDGroup")
			f.Must(f.Client.Create(ctx, pdg))
			f.WaitForPDGroupReady(ctx, pdg)

			patch := client.MergeFrom(pdg.DeepCopy())
			pdg.Spec.Replicas = ptr.To[int32](3)

			ginkgo.By("Change replica of the PDGroup")
			f.Must(f.Client.Patch(ctx, pdg, patch))
			f.WaitForPDGroupReady(ctx, pdg)
		})

		ginkgo.It("support rolling update PD by change config file", label.Update, func(ctx context.Context) {
			pdg := data.NewPDGroup(
				f.Namespace.Name,
				data.WithReplicas[*runtime.PDGroup](3),
			)

			ginkgo.By("Create PDGroup")
			f.Must(f.Client.Create(ctx, pdg))
			f.WaitForPDGroupReady(ctx, pdg)

			patch := client.MergeFrom(pdg.DeepCopy())
			pdg.Spec.Template.Spec.Config = `log.level = 'warn'`

			nctx, cancel := context.WithCancel(ctx)
			ch := make(chan struct{})
			go func() {
				defer close(ch)
				defer ginkgo.GinkgoRecover()
				f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromPDGroup(pdg), 0, waiter.LongTaskTimeout))
			}()

			changeTime := time.Now()
			ginkgo.By("Change config of the PDGroup")
			f.Must(f.Client.Patch(ctx, pdg, patch))
			f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromPDGroup(pdg), changeTime, waiter.LongTaskTimeout))
			f.WaitForPDGroupReady(ctx, pdg)
			cancel()
			<-ch
		})

		ginkgo.It("support scale PD form 5 to 3 and rolling update at same time", label.Scale, label.Update, func(ctx context.Context) {
			pdg := data.NewPDGroup(
				f.Namespace.Name,
				data.WithReplicas[*runtime.PDGroup](5),
			)

			ginkgo.By("Create PDGroup")
			f.Must(f.Client.Create(ctx, pdg))
			f.WaitForPDGroupReady(ctx, pdg)

			patch := client.MergeFrom(pdg.DeepCopy())
			pdg.Spec.Replicas = ptr.To[int32](3)
			pdg.Spec.Template.Spec.Config = `log.level = 'warn'`

			nctx, cancel := context.WithCancel(ctx)
			ch := make(chan struct{})
			go func() {
				defer close(ch)
				defer ginkgo.GinkgoRecover()
				f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromPDGroup(pdg), -2, waiter.LongTaskTimeout))
			}()

			changeTime := time.Now()
			ginkgo.By("Change config and replicas of the PDGroup")
			f.Must(f.Client.Patch(ctx, pdg, patch))
			f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromPDGroup(pdg), changeTime, waiter.LongTaskTimeout))
			f.WaitForPDGroupReady(ctx, pdg)
			cancel()
			<-ch
		})
	})

	ginkgo.Context("Suspend", label.P0, label.Suspend, func() {
		ginkgo.It("support suspend and resume PD", func(ctx context.Context) {
			pdg := data.NewPDGroup(
				f.Namespace.Name,
				data.WithReplicas[*runtime.PDGroup](3),
			)

			ginkgo.By("Create PDGroup")
			f.Must(f.Client.Create(ctx, pdg))
			f.WaitForPDGroupReadyAndNotSuspended(ctx, pdg)

			patch := client.MergeFrom(f.Cluster.DeepCopy())
			f.Cluster.Spec.SuspendAction = &v1alpha1.SuspendAction{
				SuspendCompute: true,
			}

			ginkgo.By("Suspend cluster")
			f.Must(f.Client.Patch(ctx, f.Cluster, patch))
			f.WaitForPDGroupSuspended(ctx, pdg)

			patch = client.MergeFrom(f.Cluster.DeepCopy())
			f.Cluster.Spec.SuspendAction = nil

			ginkgo.By("Resume cluster")
			f.Must(f.Client.Patch(ctx, f.Cluster, patch))
			f.WaitForPDGroupReadyAndNotSuspended(ctx, pdg)
		})
	})
})
