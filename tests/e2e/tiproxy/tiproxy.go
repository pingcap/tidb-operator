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
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/runtime"
	"github.com/pingcap/tidb-operator/tests/e2e/data"
	"github.com/pingcap/tidb-operator/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/tests/e2e/label"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/cert"
	utiltidb "github.com/pingcap/tidb-operator/tests/e2e/utils/tidb"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

const (
	changedConfig = `log.level = 'warn'`
)

var _ = ginkgo.Describe("TiProxy", label.TiProxy, func() {
	f := framework.New()
	f.Setup()

	ginkgo.Context("Scale and Update", label.P0, func() {
		ginkgo.It("scale out and in TiProxy", label.Scale, func(ctx context.Context) {
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
			ginkgo.Entry("change config file with hot reload policy", func(g *v1alpha1.TiProxyGroup) { g.Spec.Template.Spec.Config = changedConfig }, data.WithHotReloadPolicyForTiProxy()),
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

	ginkgo.PContext("TLS", label.P0, label.FeatureTLS, func() {
		f.SetupCluster(data.WithClusterTLS())
		workload := f.SetupWorkload()

		ginkgo.It("should enable TLS between MySQL Client and TiProxy", func(ctx context.Context) {
			ns := f.Namespace.Name
			tcName := f.Cluster.Name
			ginkgo.By("Installing the certificates")
			f.Must(cert.InstallTiDBIssuer(ctx, f.Client, ns, tcName))
			f.Must(cert.InstallTiDBCertificates(ctx, f.Client, ns, tcName, "dbg"))
			f.Must(cert.InstallTiDBComponentsCertificates(ctx, f.Client, ns, tcName, "pdg", "kvg", "dbg", "fg", "cg"))

			ginkgo.By("Creating the components with TLS client enabled")
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx)
			pg := f.MustCreateTiProxy(ctx, data.WithTLSForTiProxy())

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			f.WaitForTiProxyGroupReady(ctx, pg)

			ginkgo.By("Checking the status of the cluster and the connection to the TiProxy service")
			checkComponent := func(groupName, componentName string, expectedReplicas *int32) {
				podList := &corev1.PodList{}
				f.Must(f.Client.List(ctx, podList, client.InNamespace(ns), client.MatchingLabels(map[string]string{
					v1alpha1.LabelKeyCluster: tcName,
					v1alpha1.LabelKeyGroup:   groupName,
				})))
				gomega.Expect(len(podList.Items)).To(gomega.Equal(int(*expectedReplicas)))
				for _, pod := range podList.Items {
					gomega.Expect(pod.Status.Phase).To(gomega.Equal(corev1.PodRunning))

					// check for mTLS
					gomega.Expect(pod.Spec.Volumes).To(gomega.ContainElement(corev1.Volume{
						Name: v1alpha1.VolumeNameClusterTLS,
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName:  groupName + "-" + componentName + "-cluster-secret",
								DefaultMode: ptr.To[int32](420),
							},
						},
					}))
					gomega.Expect(pod.Spec.Containers[0].VolumeMounts).To(gomega.ContainElement(corev1.VolumeMount{
						Name:      v1alpha1.VolumeNameClusterTLS,
						MountPath: fmt.Sprintf("/var/lib/%s-tls", componentName),
						ReadOnly:  true,
					}))

					switch componentName {
					case v1alpha1.LabelValComponentTiProxy:
						// check for TiDB server & mysql client TLS
						gomega.Expect(pod.Spec.Volumes).To(gomega.ContainElement(corev1.Volume{
							Name: v1alpha1.VolumeNameTiProxyMySQLTLS,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  dbg.Name + "-tiproxy-server-secret",
									DefaultMode: ptr.To[int32](420),
								},
							},
						}))
						gomega.Expect(pod.Spec.Containers[0].VolumeMounts).To(gomega.ContainElement(corev1.VolumeMount{
							Name:      v1alpha1.VolumeNameTiProxyMySQLTLS,
							MountPath: v1alpha1.DirPathTiProxyMySQLTLS,
							ReadOnly:  true,
						}))
					}
				}
			}

			gomega.Eventually(func(g gomega.Gomega) {
				_, ready := utiltidb.IsClusterReady(f.Client, tcName, ns)
				g.Expect(ready).To(gomega.BeTrue())

				checkComponent(pdg.Name, v1alpha1.LabelValComponentPD, pdg.Spec.Replicas)
				checkComponent(kvg.Name, v1alpha1.LabelValComponentTiKV, kvg.Spec.Replicas)
				checkComponent(dbg.Name, v1alpha1.LabelValComponentTiDB, dbg.Spec.Replicas)
				checkComponent(dbg.Name, v1alpha1.LabelValComponentTiProxy, pg.Spec.Replicas)
			}).WithTimeout(waiter.LongTaskTimeout).WithPolling(waiter.Poll).Should(gomega.Succeed())

			workload.MustPing(ctx, data.DefaultTiProxyServiceName, "root", "", pg.Name+"-tiproxy-client-secret")
		})
	})

	// TODO(Huaxi): Add test for traffic route
})
