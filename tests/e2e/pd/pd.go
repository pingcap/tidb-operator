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
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/cert"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

var _ = ginkgo.Describe("PD", label.PD, func() {
	f := framework.New()
	f.Setup()

	ginkgo.Context("Basic", label.P0, func() {
		ginkgo.It("support create PD with 1 replica", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx, data.WithReplicas[*runtime.PDGroup](1))
			_ = f.MustCreateTiKV(ctx)
			f.WaitForPDGroupReady(ctx, pdg)
		})

		ginkgo.It("support create PD with 3 replica", func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx, data.WithReplicas[*runtime.PDGroup](3))
			_ = f.MustCreateTiKV(ctx)
			f.WaitForPDGroupReady(ctx, pdg)
		})
	})

	ginkgo.Context("Scale and Update", label.P0, func() {
		ginkgo.It("support scale PD from 1 to 3", label.Scale, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx, data.WithReplicas[*runtime.PDGroup](1))
			_ = f.MustCreateTiKV(ctx)
			f.WaitForPDGroupReady(ctx, pdg)

			patch := client.MergeFrom(pdg.DeepCopy())
			pdg.Spec.Replicas = ptr.To[int32](3)

			ginkgo.By("Change replica of the PDGroup")
			f.Must(f.Client.Patch(ctx, pdg, patch))
			f.WaitForPDGroupReady(ctx, pdg)
		})

		ginkgo.It("support scale PD from 3 to 1", label.Scale, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx, data.WithReplicas[*runtime.PDGroup](3))
			_ = f.MustCreateTiKV(ctx)
			f.WaitForPDGroupReady(ctx, pdg)

			patch := client.MergeFrom(pdg.DeepCopy())
			pdg.Spec.Replicas = ptr.To[int32](1)

			ginkgo.By("Change replica of the PDGroup")
			f.Must(f.Client.Patch(ctx, pdg, patch))
			f.WaitForPDGroupReady(ctx, pdg)
		})

		ginkgo.It("support scale PD from 3 to 5", label.Scale, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx, data.WithReplicas[*runtime.PDGroup](3))
			_ = f.MustCreateTiKV(ctx)
			f.WaitForPDGroupReady(ctx, pdg)

			patch := client.MergeFrom(pdg.DeepCopy())
			pdg.Spec.Replicas = ptr.To[int32](5)

			ginkgo.By("Change replica of the PDGroup")
			f.Must(f.Client.Patch(ctx, pdg, patch))
			f.WaitForPDGroupReady(ctx, pdg)
		})

		ginkgo.It("support scale PD from 5 to 3", label.Scale, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx, data.WithReplicas[*runtime.PDGroup](5))
			_ = f.MustCreateTiKV(ctx)
			f.WaitForPDGroupReady(ctx, pdg)

			patch := client.MergeFrom(pdg.DeepCopy())
			pdg.Spec.Replicas = ptr.To[int32](3)

			ginkgo.By("Change replica of the PDGroup")
			f.Must(f.Client.Patch(ctx, pdg, patch))
			f.WaitForPDGroupReady(ctx, pdg)
		})

		ginkgo.It("support rolling update PD by change config file", label.Update, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx, data.WithReplicas[*runtime.PDGroup](3))
			_ = f.MustCreateTiKV(ctx)
			f.WaitForPDGroupReady(ctx, pdg)

			patch := client.MergeFrom(pdg.DeepCopy())
			pdg.Spec.Template.Spec.Config = `log.level = 'warn'`

			nctx, cancel := context.WithCancel(ctx)
			ch := make(chan struct{})
			go func() {
				defer close(ch)
				defer ginkgo.GinkgoRecover()
				f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromPDGroup(pdg), 3, 0, waiter.LongTaskTimeout))
			}()

			changeTime := time.Now()
			ginkgo.By("Change config of the PDGroup")
			f.Must(f.Client.Patch(ctx, pdg, patch))
			f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromPDGroup(pdg), changeTime, waiter.LongTaskTimeout))
			f.WaitForPDGroupReady(ctx, pdg)
			cancel()
			<-ch
		})

		ginkgo.It("support scale PD from 5 to 3 and rolling update at same time", label.Scale, label.Update, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx, data.WithReplicas[*runtime.PDGroup](5))
			_ = f.MustCreateTiKV(ctx)
			f.WaitForPDGroupReady(ctx, pdg)

			patch := client.MergeFrom(pdg.DeepCopy())
			pdg.Spec.Replicas = ptr.To[int32](3)
			pdg.Spec.Template.Spec.Config = `log.level = 'warn'`

			nctx, cancel := context.WithCancel(ctx)
			ch := make(chan struct{})
			go func() {
				defer close(ch)
				defer ginkgo.GinkgoRecover()
				f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromPDGroup(pdg), 5, 0, waiter.LongTaskTimeout))
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
			pdg := f.MustCreatePD(ctx, data.WithReplicas[*runtime.PDGroup](3))
			_ = f.MustCreateTiKV(ctx)
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

	ginkgo.DescribeTableSubtree("PDReadyAPI", label.P1,
		func(tls bool) {
			// Setup cluster with UsePDReadyAPI feature gate and optionally TLS
			if tls {
				f.SetupCluster(
					data.WithClusterTLS(),
					data.WithFeatureGates(metav1alpha1.UsePDReadyAPI),
				)
			} else {
				f.SetupCluster(data.WithFeatureGates(metav1alpha1.UsePDReadyAPI))
			}

			ginkgo.It("support create PD with UsePDReadyAPI feature gate enabled", func(ctx context.Context) {
				if tls {
					ginkgo.By("Installing TLS certificates")
					ns := f.Namespace.Name
					clusterName := f.Cluster.Name
					f.Must(cert.InstallTiDBIssuer(ctx, f.Client, ns, clusterName))
					f.Must(cert.InstallTiDBComponentsCertificates(ctx, f.Client, ns, clusterName, "pdg", "kvg", "dbg", "fg", "cg"))
				}

				ginkgo.By("Creating PD with UsePDReadyAPI feature gate enabled")
				pdg := f.MustCreatePD(ctx, data.WithReplicas[*runtime.PDGroup](3))
				kvg := f.MustCreateTiKV(ctx)
				f.WaitForPDGroupReady(ctx, pdg)
				f.WaitForTiKVGroupReady(ctx, kvg)

				ginkgo.By("Verifying PD pods have correct readiness probe configuration")
				pods := &corev1.PodList{}
				f.Must(f.Client.List(ctx, pods, client.InNamespace(f.Namespace.Name), client.MatchingLabels(map[string]string{
					v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
					v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentPD,
				})))

				gomega.Expect(len(pods.Items)).To(gomega.Equal(3), "Should have 3 PD pods")
				for _, pod := range pods.Items {
					gomega.Expect(pod.Spec.Containers).To(gomega.HaveLen(1), "PD pod should have 1 container")
					probe := pod.Spec.Containers[0].ReadinessProbe
					gomega.Expect(probe).ToNot(gomega.BeNil(), "PD container should have readiness probe")
					gomega.Expect(probe.Exec).ToNot(gomega.BeNil(), "Readiness probe should use Exec action")
					gomega.Expect(probe.Exec.Command).To(gomega.ContainElement("curl"), "Should use curl command")
					gomega.Expect(probe.Exec.Command).To(gomega.ContainElement(gomega.ContainSubstring("/pd/api/v2/ready")), "Should probe /pd/api/v2/ready endpoint")
				}

				ginkgo.By("Trigger a rolling update")
				patch := client.MergeFrom(pdg.DeepCopy())
				pdg.Spec.Template.Spec.Config = `log.level = 'warn'`
				f.Must(f.Client.Patch(ctx, pdg, patch))
				f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromPDGroup(pdg), time.Now(), waiter.LongTaskTimeout))
				f.WaitForPDGroupReady(ctx, pdg)
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
