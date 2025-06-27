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
	"github.com/pingcap/tidb-operator/tests/e2e/utils/jwt"
	utiltidb "github.com/pingcap/tidb-operator/tests/e2e/utils/tidb"
	"github.com/pingcap/tidb-operator/tests/e2e/utils/waiter"
)

const (
	changedConfig = `log.level = 'warn'`
)

var _ = ginkgo.Describe("TiDB", label.TiDB, func() {
	f := framework.New()
	f.Setup()

	ginkgo.Context("Bootstrap SQL", label.P1, label.FeatureBootstrapSQL, func() {
		sql := "SET PASSWORD FOR 'root'@'%' = 'pingcap';"

		f.SetupBootstrapSQL(sql)
		f.SetupCluster(data.WithBootstrapSQL())
		workload := f.SetupWorkload()

		ginkgo.It("support init a cluster with bootstrap SQL specified", func(ctx context.Context) {
			ginkgo.By("Creating components")
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)

			workload.MustPing(ctx, data.DefaultTiDBServiceName, "root", "pingcap", "")
		})
	})

	ginkgo.Context("Auth token", label.P1, label.FeatureAuthToken, func() {
		const (
			kid   = "the-key-id-0"
			sub   = "user@pingcap.com"
			email = "user@pingcap.com"
			iss   = "issuer-abc"
		)
		sql := fmt.Sprintf(
			`CREATE USER '%s' IDENTIFIED WITH 'tidb_auth_token' REQUIRE TOKEN_ISSUER '%s' ATTRIBUTE '{"email": "%s"}';
GRANT ALL PRIVILEGES ON *.* TO '%s'@'%s';`, sub, iss, email, sub, "%")

		f.SetupBootstrapSQL(sql)
		f.SetupCluster(data.WithBootstrapSQL())
		workload := f.SetupWorkload()

		ginkgo.It("should connect to the TiDB cluster with JWT authentication", func(ctx context.Context) {
			token, err := jwt.GenerateJWT(kid, sub, email, iss)
			if err != nil {
				// ??
				ginkgo.Skip(fmt.Sprintf("failed to generate JWT token: %v", err))
			}
			jwksSecret := jwt.GenerateJWKSSecret(f.Namespace.Name, data.JWKsSecretName)
			f.Must(f.Client.Create(ctx, &jwksSecret))

			ginkgo.By("Creating components")
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx, data.WithAuthToken())

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)

			workload.MustPing(ctx, data.DefaultTiDBServiceName, sub, token, "")
		})
	})

	ginkgo.Context("Scale and Update", label.P0, func() {
		ginkgo.It("support scale TiDB form 3 to 5", label.Scale, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx,
				data.WithReplicas[*runtime.TiDBGroup](3),
			)

			ginkgo.By("Wait for Cluster Ready")
			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)

			patch := client.MergeFrom(dbg.DeepCopy())
			dbg.Spec.Replicas = ptr.To[int32](5)

			ginkgo.By("Change replica of the TiDBGroup")
			f.Must(f.Client.Patch(ctx, dbg, patch))
			f.WaitForTiDBGroupReady(ctx, dbg)
		})

		ginkgo.It("support scale TiDB form 5 to 3", label.Scale, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx,
				data.WithReplicas[*runtime.TiDBGroup](5),
			)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)

			patch := client.MergeFrom(dbg.DeepCopy())
			dbg.Spec.Replicas = ptr.To[int32](3)

			ginkgo.By("Change replica of the TiDBGroup")
			f.Must(f.Client.Patch(ctx, dbg, patch))
			f.WaitForTiDBGroupReady(ctx, dbg)
		})

		ginkgo.DescribeTable("support rolling update TiDB", label.Update,
			func(
				ctx context.Context,
				change func(*v1alpha1.TiDBGroup),
				patches ...data.GroupPatch[*runtime.TiDBGroup],
			) {
				pdg := f.MustCreatePD(ctx)
				kvg := f.MustCreateTiKV(ctx)
				var ps []data.GroupPatch[*runtime.TiDBGroup]
				ps = append(ps, data.WithReplicas[*runtime.TiDBGroup](3))
				ps = append(ps, patches...)
				dbg := f.MustCreateTiDB(ctx,
					ps...,
				)

				f.WaitForPDGroupReady(ctx, pdg)
				f.WaitForTiKVGroupReady(ctx, kvg)
				f.WaitForTiDBGroupReady(ctx, dbg)

				patch := client.MergeFrom(dbg.DeepCopy())
				change(dbg)

				nctx, cancel := context.WithCancel(ctx)
				ch := make(chan struct{})
				go func() {
					defer close(ch)
					defer ginkgo.GinkgoRecover()
					f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromTiDBGroup(dbg), 3, 1, waiter.LongTaskTimeout))
				}()

				maxTime, err := waiter.MaxPodsCreateTimestamp(ctx, f.Client, runtime.FromTiDBGroup(dbg))
				f.Must(err)
				changeTime := maxTime.Add(time.Second)

				ginkgo.By("Patch TiDBGroup")
				f.Must(f.Client.Patch(ctx, dbg, patch))
				f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiDBGroup(dbg), changeTime, waiter.LongTaskTimeout))
				f.WaitForTiDBGroupReady(ctx, dbg)
				cancel()
				<-ch
			},
			ginkgo.Entry("change config file", func(g *v1alpha1.TiDBGroup) { g.Spec.Template.Spec.Config = changedConfig }),
			ginkgo.Entry("change overlay", func(g *v1alpha1.TiDBGroup) {
				g.Spec.Template.Spec.Overlay = &v1alpha1.Overlay{
					Pod: &v1alpha1.PodOverlay{
						Spec: &corev1.PodSpec{
							TerminationGracePeriodSeconds: ptr.To[int64](10),
						},
					},
				}
			}),
			// this case tests a overlay which may contain null creationTimestamp
			ginkgo.Entry("change config file with ephemeral volume", label.OverlayEphemeralVolume, func(g *v1alpha1.TiDBGroup) {
				g.Spec.Template.Spec.Config = changedConfig
			}, data.WithEphemeralVolume()),
		)

		ginkgo.DescribeTable("support hot reload TiDB", label.Update, label.FeatureHotReload,
			func(
				ctx context.Context,
				change func(*v1alpha1.TiDBGroup),
				patches ...data.GroupPatch[*runtime.TiDBGroup],
			) {
				pdg := f.MustCreatePD(ctx)
				kvg := f.MustCreateTiKV(ctx)
				var ps []data.GroupPatch[*runtime.TiDBGroup]
				ps = append(ps, data.WithReplicas[*runtime.TiDBGroup](3))
				ps = append(ps, patches...)
				dbg := f.MustCreateTiDB(ctx,
					ps...,
				)

				f.WaitForPDGroupReady(ctx, pdg)
				f.WaitForTiKVGroupReady(ctx, kvg)
				f.WaitForTiDBGroupReady(ctx, dbg)

				currentRevision := dbg.Status.CurrentRevision

				patch := client.MergeFrom(dbg.DeepCopy())
				change(dbg)

				maxTime, err := waiter.MaxPodsCreateTimestamp(ctx, f.Client, runtime.FromTiDBGroup(dbg))
				f.Must(err)
				changeTime := maxTime.Add(time.Second)

				ginkgo.By("Patch TiDBGroup")
				f.Must(f.Client.Patch(ctx, dbg, patch))
				f.Must(waiter.WaitForPodsCondition(ctx, f.Client, runtime.FromTiDBGroup(dbg), func(pod *corev1.Pod) error {
					revision, ok := pod.Labels[v1alpha1.LabelKeyInstanceRevisionHash]
					if !ok {
						return fmt.Errorf("no revision found for pod %s/%s", pod.Namespace, pod.Name)
					}
					if revision == currentRevision {
						return fmt.Errorf("pod %s/%s is not updated, revision is %s", pod.Namespace, pod.Name, currentRevision)
					}

					return nil
				}, waiter.LongTaskTimeout))
				f.WaitForTiDBGroupReady(ctx, dbg)

				newMaxTime, err := waiter.MaxPodsCreateTimestamp(ctx, f.Client, runtime.FromTiDBGroup(dbg))
				f.Must(err)
				f.True(changeTime.After(*newMaxTime))
			},
			ginkgo.Entry("change config file with hot reload policy", func(g *v1alpha1.TiDBGroup) { g.Spec.Template.Spec.Config = changedConfig }, data.WithHotReloadPolicy()),
			ginkgo.Entry("change pod annotations and labels", func(g *v1alpha1.TiDBGroup) {
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

		ginkgo.It("support scale TiDB form 5 to 3 and rolling update at same time", label.Scale, label.Update, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx,
				data.WithReplicas[*runtime.TiDBGroup](5),
			)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)

			patch := client.MergeFrom(dbg.DeepCopy())
			dbg.Spec.Replicas = ptr.To[int32](3)
			dbg.Spec.Template.Spec.Config = changedConfig

			nctx, cancel := context.WithCancel(ctx)
			ch := make(chan struct{})
			go func() {
				defer close(ch)
				defer ginkgo.GinkgoRecover()
				f.Must(waiter.WaitPodsRollingUpdateOnce(nctx, f.Client, runtime.FromTiDBGroup(dbg), 5, 1, waiter.LongTaskTimeout))
			}()

			maxTime, err := waiter.MaxPodsCreateTimestamp(ctx, f.Client, runtime.FromTiDBGroup(dbg))
			f.Must(err)
			changeTime := maxTime.Add(time.Second)

			ginkgo.By("Change config and replicas of the TiDBGroup")
			f.Must(f.Client.Patch(ctx, dbg, patch))
			f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiDBGroup(dbg), changeTime, waiter.LongTaskTimeout))
			f.WaitForTiDBGroupReady(ctx, dbg)
			cancel()
			<-ch
		})
	})

	ginkgo.Context("TLS", label.P0, label.FeatureTLS, func() {
		f.SetupCluster(data.WithClusterTLS())
		workload := f.SetupWorkload()

		ginkgo.It("should enable TLS for MySQL Client and between TiDB components", func(ctx context.Context) {
			ns := f.Namespace.Name
			tcName := f.Cluster.Name
			ginkgo.By("Installing the certificates")
			f.Must(cert.InstallTiDBIssuer(ctx, f.Client, ns, tcName))
			f.Must(cert.InstallTiDBCertificates(ctx, f.Client, ns, tcName, "dbg"))
			f.Must(cert.InstallTiDBComponentsCertificates(ctx, f.Client, ns, tcName, "pdg", "kvg", "dbg", "fg", "cg"))

			ginkgo.By("Creating the components with TLS client enabled")
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx, data.WithTLS())
			flashg := f.MustCreateTiFlash(ctx)
			cdcg := f.MustCreateTiCDC(ctx)
			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			f.WaitForTiFlashGroupReady(ctx, flashg)
			f.WaitForTiCDCGroupReady(ctx, cdcg)

			ginkgo.By("Checking the status of the cluster and the connection to the TiDB service")
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
					case v1alpha1.LabelValComponentTiDB:
						// check for TiDB server & mysql client TLS
						gomega.Expect(pod.Spec.Volumes).To(gomega.ContainElement(corev1.Volume{
							Name: v1alpha1.VolumeNameMySQLTLS,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  dbg.Name + "-tidb-server-secret",
									DefaultMode: ptr.To[int32](420),
								},
							},
						}))
						gomega.Expect(pod.Spec.Containers[0].VolumeMounts).To(gomega.ContainElement(corev1.VolumeMount{
							Name:      v1alpha1.VolumeNameMySQLTLS,
							MountPath: v1alpha1.DirPathMySQLTLS,
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
				checkComponent(flashg.Name, v1alpha1.LabelValComponentTiFlash, flashg.Spec.Replicas)
				checkComponent(cdcg.Name, v1alpha1.LabelValComponentTiCDC, cdcg.Spec.Replicas)
			}).WithTimeout(waiter.LongTaskTimeout).WithPolling(waiter.Poll).Should(gomega.Succeed())

			workload.MustPing(ctx, data.DefaultTiDBServiceName, "root", "", dbg.Name+"-tidb-client-secret")
		})
	})
})
