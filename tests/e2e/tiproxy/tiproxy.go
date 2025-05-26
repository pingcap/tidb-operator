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

package tiproxy

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

const (
	changedConfig = `log.level = 'warn'`
)

var _ = ginkgo.Describe("TiProxy", label.TiProxy, func() {
	f := framework.New()
	f.Setup()

	ginkgo.Context("Scale and Update", label.P0, func() {
		ginkgo.It("scale out and in", label.Scale, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx)
			proxyg := f.MustCreateTiProxy(ctx, data.WithReplicas[*runtime.TiProxyGroup](2))

			ginkgo.By("Wait for Cluster Ready")
			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			f.WaitForTiProxyGroupReady(ctx, proxyg)

			patch := client.MergeFrom(proxyg.DeepCopy())
			proxyg.Spec.Replicas = ptr.To[int32](4)

			ginkgo.By("Scale out TiProxy")
			f.Must(f.Client.Patch(ctx, proxyg, patch))
			f.WaitForTiProxyGroupReady(ctx, proxyg)

			patch = client.MergeFrom(proxyg.DeepCopy())
			proxyg.Spec.Replicas = ptr.To[int32](2)

			ginkgo.By("Scale in TiProxy")
			f.Must(f.Client.Patch(ctx, proxyg, patch))
			f.WaitForTiProxyGroupReady(ctx, proxyg)
		})

		ginkgo.DescribeTable("support rolling update", label.Update,
			func(
				ctx context.Context,
				change func(*v1alpha1.TiProxyGroup),
				patches ...data.GroupPatch[*runtime.TiProxyGroup],
			) {
				pdg := f.MustCreatePD(ctx)
				kvg := f.MustCreateTiKV(ctx)
				dbg := f.MustCreateTiDB(ctx)
				var ps []data.GroupPatch[*runtime.TiProxyGroup]
				ps = append(ps, data.WithReplicas[*runtime.TiProxyGroup](2))
				ps = append(ps, patches...)
				proxyg := f.MustCreateTiProxy(ctx,
					ps...,
				)

				f.WaitForPDGroupReady(ctx, pdg)
				f.WaitForTiKVGroupReady(ctx, kvg)
				f.WaitForTiDBGroupReady(ctx, dbg)
				f.WaitForTiProxyGroupReady(ctx, proxyg)

				patch := client.MergeFrom(proxyg.DeepCopy())
				change(proxyg)

				nctx, cancel := context.WithCancel(ctx)
				ch := make(chan struct{})
				go func() {
					defer close(ch)
					defer ginkgo.GinkgoRecover()
					f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromTiProxyGroup(proxyg), 2, 1, waiter.LongTaskTimeout))
				}()

				maxTime, err := waiter.MaxPodsCreateTimestamp(ctx, f.Client, runtime.FromTiProxyGroup(proxyg))
				f.Must(err)
				changeTime := maxTime.Add(time.Second)

				ginkgo.By("Patch TiProxyGroup")
				f.Must(f.Client.Patch(ctx, proxyg, patch))
				f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiProxyGroup(proxyg), changeTime, waiter.LongTaskTimeout))
				f.WaitForTiProxyGroupReady(ctx, proxyg)
				cancel()
				<-ch
			},
			ginkgo.Entry("change config file", func(g *v1alpha1.TiProxyGroup) { g.Spec.Template.Spec.Config = changedConfig }),
			ginkgo.Entry("change overlay", func(g *v1alpha1.TiProxyGroup) {
				g.Spec.Template.Spec.Overlay = &v1alpha1.Overlay{
					Pod: &v1alpha1.PodOverlay{
						Spec: &corev1.PodSpec{
							TerminationGracePeriodSeconds: ptr.To[int64](10),
						},
					},
				}
			}),
		)

		ginkgo.DescribeTable("support hot reload", label.Update, label.FeatureHotReload,
			func(
				ctx context.Context,
				change func(*v1alpha1.TiProxyGroup),
				patches ...data.GroupPatch[*runtime.TiProxyGroup],
			) {
				pdg := f.MustCreatePD(ctx)
				kvg := f.MustCreateTiKV(ctx)
				dbg := f.MustCreateTiDB(ctx)
				var ps []data.GroupPatch[*runtime.TiProxyGroup]
				ps = append(ps, data.WithReplicas[*runtime.TiProxyGroup](2))
				ps = append(ps, patches...)
				proxyg := f.MustCreateTiProxy(ctx,
					ps...,
				)

				f.WaitForPDGroupReady(ctx, pdg)
				f.WaitForTiKVGroupReady(ctx, kvg)
				f.WaitForTiDBGroupReady(ctx, dbg)
				f.WaitForTiProxyGroupReady(ctx, proxyg)

				currentRevision := dbg.Status.CurrentRevision

				patch := client.MergeFrom(proxyg.DeepCopy())
				change(proxyg)

				maxTime, err := waiter.MaxPodsCreateTimestamp(ctx, f.Client, runtime.FromTiProxyGroup(proxyg))
				f.Must(err)
				changeTime := maxTime.Add(time.Second)

				ginkgo.By("Patch TiProxyGroup")
				f.Must(f.Client.Patch(ctx, proxyg, patch))
				f.Must(waiter.WaitForPodsCondition(ctx, f.Client, runtime.FromTiProxyGroup(proxyg), func(pod *corev1.Pod) error {
					revision, ok := pod.Labels[v1alpha1.LabelKeyInstanceRevisionHash]
					if !ok {
						return fmt.Errorf("no revision found for pod %s/%s", pod.Namespace, pod.Name)
					}
					if revision == currentRevision {
						return fmt.Errorf("pod %s/%s is not updated, revision is %s", pod.Namespace, pod.Name, currentRevision)
					}

					return nil
				}, waiter.LongTaskTimeout))
				f.WaitForTiProxyGroupReady(ctx, proxyg)

				newMaxTime, err := waiter.MaxPodsCreateTimestamp(ctx, f.Client, runtime.FromTiProxyGroup(proxyg))
				f.Must(err)
				f.True(changeTime.After(*newMaxTime))
			},
			ginkgo.Entry("change config file with hot reload policy", func(g *v1alpha1.TiProxyGroup) { g.Spec.Template.Spec.Config = changedConfig }, data.WithHotReloadPolicy()),
			ginkgo.Entry("change pod annotations and labels", func(g *v1alpha1.TiProxyGroup) {
				g.Spec.Template.Spec.Overlay = &v1alpha1.Overlay{
					Pod: &v1alpha1.PodOverlay{
						ObjectMeta: v1alpha1.ObjectMeta{
							Labels: map[string]string{
								"test": "test",
							},
							Annotations: map[string]string{
								"test": "test",
							},
						},
					},
				}
			}),
		)

		ginkgo.It("support scale in from 4 to 2 and rolling update at same time", label.Scale, label.Update, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx)
			proxyg := f.MustCreateTiProxy(ctx,
				data.WithReplicas[*runtime.TiProxyGroup](4),
			)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			f.WaitForTiProxyGroupReady(ctx, proxyg)

			patch := client.MergeFrom(proxyg.DeepCopy())
			proxyg.Spec.Replicas = ptr.To[int32](2)
			proxyg.Spec.Template.Spec.Config = changedConfig

			nctx, cancel := context.WithCancel(ctx)
			ch := make(chan struct{})
			go func() {
				defer close(ch)
				defer ginkgo.GinkgoRecover()
				f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromTiProxyGroup(proxyg), 4, 1, waiter.LongTaskTimeout))
			}()

			maxTime, err := waiter.MaxPodsCreateTimestamp(ctx, f.Client, runtime.FromTiProxyGroup(proxyg))
			f.Must(err)
			changeTime := maxTime.Add(time.Second)

			ginkgo.By("Change config and replicas of the TiProxyGroup")
			f.Must(f.Client.Patch(ctx, proxyg, patch))
			f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiProxyGroup(proxyg), changeTime, waiter.LongTaskTimeout))
			f.WaitForTiProxyGroupReady(ctx, proxyg)
			cancel()
			<-ch
		})
	})

	// TODO(Huaxi): Add test for traffic route
})
