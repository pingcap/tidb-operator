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
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	"github.com/pingcap/tidb-operator/v2/pkg/apicall"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime"
	"github.com/pingcap/tidb-operator/v2/pkg/runtime/scope"
	tiproxyapi "github.com/pingcap/tidb-operator/v2/pkg/tiproxyapi/v1"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/data"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework/action"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/framework/desc"
	wopt "github.com/pingcap/tidb-operator/v2/tests/e2e/framework/workload"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/label"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/cert"
	utiltidb "github.com/pingcap/tidb-operator/v2/tests/e2e/utils/tidb"
	"github.com/pingcap/tidb-operator/v2/tests/e2e/utils/waiter"
)

const (
	changedConfig      = `log.level = 'warn'`
	gracefulWaitConfig = `[proxy]
graceful-wait-before-shutdown = 30
`
	gracefulScaleInDeleteDelaySeconds int32 = 3600
)

func withTiProxyGracefulShutdownDelay(seconds int32) data.GroupPatch[*v1alpha1.TiProxyGroup] {
	return data.GroupPatchFunc[*v1alpha1.TiProxyGroup](func(obj *v1alpha1.TiProxyGroup) {
		if obj.Spec.Template.Annotations == nil {
			obj.Spec.Template.Annotations = map[string]string{}
		}
		obj.Spec.Template.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds] = strconv.Itoa(int(seconds))
	})
}

func listTiProxyPods(ctx context.Context, f *framework.Framework, proxyg *v1alpha1.TiProxyGroup) (*corev1.PodList, error) {
	pods := &corev1.PodList{}
	if err := f.Client.List(ctx, pods, client.InNamespace(proxyg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyManagedBy: v1alpha1.LabelValManagedByOperator,
		v1alpha1.LabelKeyCluster:   proxyg.Spec.Cluster.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiProxy,
	}); err != nil {
		return nil, err
	}
	return pods, nil
}

func listTiProxies(ctx context.Context, f *framework.Framework, proxyg *v1alpha1.TiProxyGroup) (*v1alpha1.TiProxyList, error) {
	tiproxies := &v1alpha1.TiProxyList{}
	if err := f.Client.List(ctx, tiproxies, client.InNamespace(proxyg.Namespace), client.MatchingLabels{
		v1alpha1.LabelKeyCluster:   proxyg.Spec.Cluster.Name,
		v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiProxy,
	}); err != nil {
		return nil, err
	}
	return tiproxies, nil
}

func podUIDSet(pods []corev1.Pod) map[string]string {
	uids := make(map[string]string, len(pods))
	for i := range pods {
		uids[pods[i].Name] = string(pods[i].UID)
	}
	return uids
}

func offlineTiProxyNames(tiproxies *v1alpha1.TiProxyList) map[string]struct{} {
	names := map[string]struct{}{}
	for i := range tiproxies.Items {
		tp := &tiproxies.Items[i]
		if tp.Spec.Offline != nil && *tp.Spec.Offline && tp.DeletionTimestamp.IsZero() {
			names[tp.Name] = struct{}{}
		}
	}
	return names
}

func readyTiProxyServiceBackends(ctx context.Context, c client.Client, proxyg *v1alpha1.TiProxyGroup) (int, error) {
	endpoints := &corev1.Endpoints{}
	if err := c.Get(ctx, client.ObjectKey{
		Namespace: proxyg.Namespace,
		Name:      proxyg.Name + "-tiproxy",
	}, endpoints); err != nil {
		return 0, err
	}

	count := 0
	for _, subset := range endpoints.Subsets {
		count += len(subset.Addresses)
	}
	return count, nil
}

func tiproxyHealthStatusCode(ctx context.Context, f *framework.Framework, pod *corev1.Pod) (int, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	ports := f.PortForwardPod(probeCtx, pod, []string{":3080"})
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/api/debug/health", ports[0].Local), nil)
	if err != nil {
		return 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func tiproxyConnectionCount(ctx context.Context, f *framework.Framework, pod *corev1.Pod) (float64, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	ports := f.PortForwardPod(probeCtx, pod, []string{fmt.Sprintf(":%d", v1alpha1.DefaultTiProxyPortAPI)})
	tpClient := tiproxyapi.NewTiProxyClient(fmt.Sprintf("127.0.0.1:%d", ports[0].Local), 10*time.Second, nil)
	return tpClient.ConnectionCount(probeCtx)
}

func holdTiProxySQLConnection(ctx context.Context, f *framework.Framework, pod *corev1.Pod) (func(), error) {
	forwardCtx, cancel := context.WithCancel(ctx)
	ports := f.PortForwardPod(forwardCtx, pod, []string{fmt.Sprintf(":%d", v1alpha1.DefaultTiProxyPortClient)})

	db, err := sql.Open("mysql", fmt.Sprintf("root:@tcp(127.0.0.1:%d)/?timeout=10s", ports[0].Local))
	if err != nil {
		cancel()
		return nil, err
	}
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)

	conn, err := db.Conn(ctx)
	if err != nil {
		_ = db.Close()
		cancel()
		return nil, err
	}
	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		_ = db.Close()
		cancel()
		return nil, err
	}

	return func() {
		_ = conn.Close()
		_ = db.Close()
		cancel()
	}, nil
}

func tiproxySupportsHealthOverrideAPI(ctx context.Context, f *framework.Framework, pod *corev1.Pod) (bool, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	ports := f.PortForwardPod(probeCtx, pod, []string{":3080"})
	req, err := http.NewRequestWithContext(probeCtx, http.MethodDelete, fmt.Sprintf("http://127.0.0.1:%d/api/debug/health", ports[0].Local), nil)
	if err != nil {
		return false, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound, http.StatusMethodNotAllowed:
		return false, nil
	}
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return true, nil
	}
	return false, fmt.Errorf("unexpected status code %d from DELETE /api/debug/health of tiproxy pod %s/%s", resp.StatusCode, pod.Namespace, pod.Name)
}

func controllerTiProxyOwnerName(pod *corev1.Pod) (string, bool) {
	for i := range pod.OwnerReferences {
		owner := &pod.OwnerReferences[i]
		if owner.Kind != "TiProxy" {
			continue
		}
		if owner.Controller == nil || !*owner.Controller {
			continue
		}
		return owner.Name, true
	}
	return "", false
}

func manuallyTriggerTiProxyUnhealthy(
	ctx context.Context,
	f *framework.Framework,
	enabled bool,
	pods []corev1.Pod,
	initialPodUIDs map[string]string,
	deletingTiProxies map[string]struct{},
	terminated map[string]struct{},
) error {
	if !enabled {
		return nil
	}

	for i := range pods {
		pod := &pods[i]
		if _, ok := initialPodUIDs[pod.Name]; !ok {
			continue
		}
		if _, ok := terminated[pod.Name]; ok {
			continue
		}
		ownerName, ok := controllerTiProxyOwnerName(pod)
		if !ok {
			continue
		}
		if _, ok := deletingTiProxies[ownerName]; !ok {
			continue
		}

		stdout, stderr, err := f.ExecPod(ctx, pod, v1alpha1.ContainerNameTiProxy, "/bin/sh", "-c", "kill -TERM 1")
		if err != nil {
			return fmt.Errorf("cannot exec kill TiProxy pod %s/%s: %w, stdout: %s, stderr: %s", pod.Namespace, pod.Name, err, stdout, stderr)
		}
		terminated[pod.Name] = struct{}{}
	}

	return nil
}

// manuallyTriggerOfflineTiProxyUnhealthy kills the TiProxy process in pods owned by offline
// TiProxy instances. This is used when the TiProxy version does not support the health override
// API: ensureTiProxyMarkedUnhealthy treats an already-unhealthy endpoint as success without PUT.
func manuallyTriggerOfflineTiProxyUnhealthy(
	ctx context.Context,
	f *framework.Framework,
	enabled bool,
	pods []corev1.Pod,
	offlineTiProxies map[string]struct{},
	terminated map[string]struct{},
) error {
	if !enabled {
		return nil
	}

	for i := range pods {
		pod := &pods[i]
		if _, ok := terminated[pod.Name]; ok {
			continue
		}
		ownerName, ok := controllerTiProxyOwnerName(pod)
		if !ok {
			continue
		}
		if _, ok := offlineTiProxies[ownerName]; !ok {
			continue
		}

		stdout, stderr, err := f.ExecPod(ctx, pod, v1alpha1.ContainerNameTiProxy, "/bin/sh", "-c", "kill -TERM 1")
		if err != nil {
			return fmt.Errorf("cannot exec kill TiProxy pod %s/%s: %w, stdout: %s, stderr: %s", pod.Namespace, pod.Name, err, stdout, stderr)
		}
		terminated[pod.Name] = struct{}{}
	}

	return nil
}

var _ = ginkgo.Describe("TiProxy", label.TiProxy, func() {
	f := framework.New()
	f.Setup()

	ginkgo.Context("Scale and Update", label.P0, func() {
		ginkgo.It("scale out and in TiProxy", label.Scale, func(ctx context.Context) {
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx)
			proxyg := f.MustCreateTiProxy(ctx, data.WithReplicas[scope.TiProxyGroup](2))

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
				patches ...data.GroupPatch[*v1alpha1.TiProxyGroup],
			) {
				pdg := f.MustCreatePD(ctx)
				kvg := f.MustCreateTiKV(ctx)
				dbg := f.MustCreateTiDB(ctx)
				var ps []data.GroupPatch[*v1alpha1.TiProxyGroup]
				ps = append(ps, data.WithReplicas[scope.TiProxyGroup](2))
				ps = append(ps, patches...)
				proxyg := f.MustCreateTiProxy(ctx,
					ps...,
				)

				f.WaitForPDGroupReady(ctx, pdg)
				f.WaitForTiKVGroupReady(ctx, kvg)
				f.WaitForTiDBGroupReady(ctx, dbg)
				f.WaitForTiProxyGroupReady(ctx, proxyg)

				nctx, cancel := context.WithCancel(ctx)
				done := framework.AsyncWaitPodsRollingUpdateOnce[scope.TiProxyGroup](nctx, f, proxyg, 2)
				defer func() { <-done }()
				defer cancel()

				changeTime, err := waiter.MaxPodsCreateTimestamp[scope.TiProxyGroup](ctx, f.Client, proxyg)
				f.Must(err)

				ginkgo.By("Patch TiProxyGroup")
				patch := client.MergeFrom(proxyg.DeepCopy())
				change(proxyg)
				f.Must(f.Client.Patch(ctx, proxyg, patch))

				f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiProxyGroup(proxyg), *changeTime, waiter.LongTaskTimeout))
				f.WaitForTiProxyGroupReady(ctx, proxyg)
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
				patches ...data.GroupPatch[*v1alpha1.TiProxyGroup],
			) {
				pdg := f.MustCreatePD(ctx)
				kvg := f.MustCreateTiKV(ctx)
				dbg := f.MustCreateTiDB(ctx)
				var ps []data.GroupPatch[*v1alpha1.TiProxyGroup]
				ps = append(ps, data.WithReplicas[scope.TiProxyGroup](2))
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

				changeTime, err := waiter.MaxPodsCreateTimestamp[scope.TiProxyGroup](ctx, f.Client, proxyg)
				f.Must(err)

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

				newMaxTime, err := waiter.MaxPodsCreateTimestamp[scope.TiProxyGroup](ctx, f.Client, proxyg)
				f.Must(err)
				f.True(changeTime.Equal(*newMaxTime))
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
				data.WithReplicas[scope.TiProxyGroup](4),
			)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			f.WaitForTiProxyGroupReady(ctx, proxyg)

			nctx, cancel := context.WithCancel(ctx)
			done := framework.AsyncWaitPodsRollingUpdateOnce[scope.TiProxyGroup](nctx, f, proxyg, 2)
			defer func() { <-done }()
			defer cancel()

			changeTime, err := waiter.MaxPodsCreateTimestamp[scope.TiProxyGroup](ctx, f.Client, proxyg)
			f.Must(err)

			ginkgo.By("Change config and replicas of the TiProxyGroup")
			patch := client.MergeFrom(proxyg.DeepCopy())
			proxyg.Spec.Replicas = ptr.To[int32](2)
			proxyg.Spec.Template.Spec.Config = changedConfig
			f.Must(f.Client.Patch(ctx, proxyg, patch))

			f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiProxyGroup(proxyg), *changeTime, waiter.LongTaskTimeout))
			f.WaitForTiProxyGroupReady(ctx, proxyg)
		})

		ginkgo.It("keep service backends during graceful rolling update with large maxSurge", label.Update, func(ctx context.Context) {
			o := desc.DefaultOptions()
			const replicas int32 = 3
			const deleteDelaySeconds int32 = 3600

			pdg := action.MustCreatePD(ctx, f, o)
			kvg := action.MustCreateTiKV(ctx, f, o)
			dbg := action.MustCreateTiDB(ctx, f, o)
			proxyg := action.MustCreateTiProxy(ctx, f, o,
				data.WithReplicas[scope.TiProxyGroup](replicas),
				data.WithTiProxyMaxSurge(5),
				data.GroupPatchFunc[*v1alpha1.TiProxyGroup](func(obj *v1alpha1.TiProxyGroup) {
					obj.Spec.Template.Spec.Config = gracefulWaitConfig
				}),
			)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			f.WaitForTiProxyGroupReady(ctx, proxyg)

			initialPods, err := listTiProxyPods(ctx, f, proxyg)
			f.Must(err)
			gomega.Expect(initialPods.Items).To(gomega.HaveLen(int(replicas)))
			targetOldPod := initialPods.Items[0].DeepCopy()
			initialPodUIDs := map[string]string{}
			for i := range initialPods.Items {
				initialPodUIDs[initialPods.Items[i].Name] = string(initialPods.Items[i].UID)
			}

			ginkgo.By("Patch TiProxyGroup to enable graceful shutdown delay without restarting pods")
			patch := client.MergeFrom(proxyg.DeepCopy())
			if proxyg.Spec.Template.Annotations == nil {
				proxyg.Spec.Template.Annotations = map[string]string{}
			}
			proxyg.Spec.Template.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds] = strconv.Itoa(int(deleteDelaySeconds))
			f.Must(f.Client.Patch(ctx, proxyg, patch))

			gomega.Eventually(func() error {
				tiproxies, err := listTiProxies(ctx, f, proxyg)
				if err != nil {
					return err
				}
				if len(tiproxies.Items) != int(replicas) {
					return fmt.Errorf("got %d tiproxy instances, want %d", len(tiproxies.Items), replicas)
				}
				for i := range tiproxies.Items {
					if tiproxies.Items[i].Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownDeleteDelaySeconds] != strconv.Itoa(int(deleteDelaySeconds)) {
						return fmt.Errorf("tiproxy %s/%s does not have graceful shutdown delete delay annotation set", tiproxies.Items[i].Namespace, tiproxies.Items[i].Name)
					}
				}

				pods, err := listTiProxyPods(ctx, f, proxyg)
				if err != nil {
					return err
				}
				if len(pods.Items) != int(replicas) {
					return fmt.Errorf("got %d tiproxy pods, want %d", len(pods.Items), replicas)
				}
				for i := range pods.Items {
					pod := &pods.Items[i]
					uid, ok := initialPodUIDs[pod.Name]
					if !ok {
						return fmt.Errorf("tiproxy pod %s/%s was recreated unexpectedly", pod.Namespace, pod.Name)
					}
					if string(pod.UID) != uid {
						return fmt.Errorf("tiproxy pod %s/%s uid changed from %s to %s", pod.Namespace, pod.Name, uid, pod.UID)
					}
					if !pod.DeletionTimestamp.IsZero() {
						return fmt.Errorf("tiproxy pod %s/%s is deleting unexpectedly", pod.Namespace, pod.Name)
					}
				}
				return nil
			}).WithTimeout(waiter.LongTaskTimeout).WithPolling(waiter.Poll).Should(gomega.Succeed())

			f.Must(f.Client.Get(ctx, client.ObjectKeyFromObject(proxyg), proxyg))

			ginkgo.By("Probe whether TiProxy supports the health override API")
			probePods, err := listTiProxyPods(ctx, f, proxyg)
			f.Must(err)
			gomega.Expect(probePods.Items).ToNot(gomega.BeEmpty())
			supportsHealthOverrideAPI, err := tiproxySupportsHealthOverrideAPI(ctx, f, &probePods.Items[0])
			f.Must(err)
			manualTiProxyUnhealthyTrigger := !supportsHealthOverrideAPI

			ginkgo.By("Hold one SQL connection on an old TiProxy pod")
			releaseTargetOldPodConnection, err := holdTiProxySQLConnection(ctx, f, targetOldPod)
			f.Must(err)
			defer func() {
				if releaseTargetOldPodConnection != nil {
					releaseTargetOldPodConnection()
				}
			}()
			gomega.Eventually(func() error {
				connectionCount, err := tiproxyConnectionCount(ctx, f, targetOldPod)
				if err != nil {
					return err
				}
				if connectionCount <= 0 {
					return fmt.Errorf("target old tiproxy pod %s/%s has %v connections, want > 0", targetOldPod.Namespace, targetOldPod.Name, connectionCount)
				}
				return nil
			}).WithTimeout(waiter.ShortTaskTimeout).WithPolling(waiter.Poll).Should(gomega.Succeed())

			changeTime, err := waiter.MaxPodsCreateTimestamp[scope.TiProxyGroup](ctx, f.Client, proxyg)
			f.Must(err)

			ginkgo.By("Patch TiProxyGroup to trigger a rolling restart")
			patch = client.MergeFrom(proxyg.DeepCopy())
			proxyg.Spec.Template.Spec.Config = changedConfig
			f.Must(f.Client.Patch(ctx, proxyg, patch))

			var violated error
			terminatedPods := map[string]struct{}{}
			var targetDrainingPod *corev1.Pod
			ginkgo.By("Ensure old pods start graceful shutdown only after enough new TiProxy backends are ready")
			gomega.Eventually(func() error {
				if violated != nil {
					return violated
				}

				pods, err := listTiProxyPods(ctx, f, proxyg)
				if err != nil {
					return err
				}

				tiproxies, err := listTiProxies(ctx, f, proxyg)
				if err != nil {
					return err
				}
				deletingTiProxies := map[string]struct{}{}
				for i := range tiproxies.Items {
					if !tiproxies.Items[i].DeletionTimestamp.IsZero() {
						deletingTiProxies[tiproxies.Items[i].Name] = struct{}{}
					}
				}

				backends, err := readyTiProxyServiceBackends(ctx, f.Client, proxyg)
				if err != nil {
					return err
				}

				oldPodDraining := false
				for i := range pods.Items {
					pod := &pods.Items[i]
					if _, ok := initialPodUIDs[pod.Name]; !ok {
						continue
					}
					rawStartTime := pod.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime]
					if rawStartTime == "" {
						continue
					}
					oldPodDraining = true

					if backends < int(replicas) {
						violated = fmt.Errorf("tiproxy service backends dropped below desired replicas after drain started: got %d, want >= %d", backends, replicas)
						return violated
					}

					statusCode, err := tiproxyHealthStatusCode(ctx, f, pod)
					if err != nil {
						return fmt.Errorf("cannot query health of draining tiproxy pod %s/%s: %w", pod.Namespace, pod.Name, err)
					}
					if statusCode == http.StatusOK {
						return fmt.Errorf("draining tiproxy pod %s/%s is still healthy after graceful shutdown begins", pod.Namespace, pod.Name)
					}

					if pod.Name != targetOldPod.Name {
						continue
					}

					connectionCount, err := tiproxyConnectionCount(ctx, f, pod)
					if err != nil {
						return fmt.Errorf("cannot query connection count of target draining tiproxy pod %s/%s: %w", pod.Namespace, pod.Name, err)
					}
					if connectionCount <= 0 {
						return fmt.Errorf("target draining tiproxy pod %s/%s has %v connections before releasing the held connection, want > 0", pod.Namespace, pod.Name, connectionCount)
					}

					targetDrainingPod = pod.DeepCopy()
					return nil
				}

				if !oldPodDraining && backends < int(replicas) {
					return fmt.Errorf("only %d tiproxy service backends are ready before drain starts, want >= %d", backends, replicas)
				}
				if err := manuallyTriggerTiProxyUnhealthy(ctx, f, manualTiProxyUnhealthyTrigger, pods.Items, initialPodUIDs, deletingTiProxies, terminatedPods); err != nil {
					return err
				}

				return fmt.Errorf("target old tiproxy pod %s/%s has not started graceful shutdown yet", targetOldPod.Namespace, targetOldPod.Name)
			}).WithTimeout(waiter.LongTaskTimeout).WithPolling(waiter.Poll).Should(gomega.Succeed())

			ginkgo.By("Release the held SQL connection and ensure the old TiProxy pod is deleted before the graceful delay limit")
			releaseTargetOldPodConnection()
			releaseTargetOldPodConnection = nil
			const earlyDeleteTimeout = time.Minute
			gomega.Expect(time.Duration(deleteDelaySeconds) * time.Second).To(gomega.BeNumerically(">", waiter.LongTaskTimeout+earlyDeleteTimeout))
			f.Must(waiter.WaitForObjectDeleted(ctx, f.Client, targetDrainingPod, earlyDeleteTimeout))

			f.Must(waiter.WaitForPodsRecreated(ctx, f.Client, runtime.FromTiProxyGroup(proxyg), *changeTime, waiter.LongTaskTimeout))
			f.WaitForTiProxyGroupReady(ctx, proxyg)

			ginkgo.By("Ensure drained TiProxy instances and pods are eventually deleted")
			tiproxies := &v1alpha1.TiProxyList{}
			f.Must(f.Client.List(ctx, tiproxies, client.InNamespace(proxyg.Namespace), client.MatchingLabels{
				v1alpha1.LabelKeyCluster:   proxyg.Spec.Cluster.Name,
				v1alpha1.LabelKeyComponent: v1alpha1.LabelValComponentTiProxy,
			}))
			gomega.Expect(tiproxies.Items).To(gomega.HaveLen(int(replicas)))

			pods, err := apicall.ListPods[scope.TiProxyGroup](ctx, f.Client, proxyg)
			f.Must(err)
			gomega.Expect(pods.Items).To(gomega.HaveLen(int(replicas)))
		})

		ginkgo.It("revive draining TiProxy pods on scale-out instead of creating new ones", label.Scale, func(ctx context.Context) {
			o := desc.DefaultOptions()
			const initialReplicas int32 = 3
			const scaledInReplicas int32 = 1

			pdg := action.MustCreatePD(ctx, f, o)
			kvg := action.MustCreateTiKV(ctx, f, o)
			dbg := action.MustCreateTiDB(ctx, f, o)
			proxyg := action.MustCreateTiProxy(ctx, f, o,
				data.WithReplicas[scope.TiProxyGroup](initialReplicas),
				withTiProxyGracefulShutdownDelay(gracefulScaleInDeleteDelaySeconds),
			)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			f.WaitForTiProxyGroupReady(ctx, proxyg)

			initialPods, err := listTiProxyPods(ctx, f, proxyg)
			f.Must(err)
			gomega.Expect(initialPods.Items).To(gomega.HaveLen(int(initialReplicas)))

			ginkgo.By("Probe whether TiProxy supports the health override API")
			supportsHealthOverrideAPI, err := tiproxySupportsHealthOverrideAPI(ctx, f, &initialPods.Items[0])
			f.Must(err)
			if !supportsHealthOverrideAPI {
				ginkgo.Skip("TiProxy does not support the health override API required for graceful scale-in revive")
			}

			initialPodUIDs := podUIDSet(initialPods.Items)

			ginkgo.By("Hold a SQL connection on every TiProxy pod so drained instances keep connections and stay revivable")
			var releaseFuncs []func()
			defer func() {
				for _, release := range releaseFuncs {
					release()
				}
			}()
			for i := range initialPods.Items {
				release, err := holdTiProxySQLConnection(ctx, f, &initialPods.Items[i])
				f.Must(err)
				releaseFuncs = append(releaseFuncs, release)
			}

			ginkgo.By("Scale in TiProxy and wait for the removed instances to enter graceful scale-in drain")
			patch := client.MergeFrom(proxyg.DeepCopy())
			proxyg.Spec.Replicas = ptr.To(scaledInReplicas)
			f.Must(f.Client.Patch(ctx, proxyg, patch))

			expectedDraining := int(initialReplicas - scaledInReplicas)
			gomega.Eventually(func() error {
				// No pod may be deleted/recreated while draining: held connections must keep them alive.
				pods, err := listTiProxyPods(ctx, f, proxyg)
				if err != nil {
					return err
				}
				if len(pods.Items) != int(initialReplicas) {
					return fmt.Errorf("got %d tiproxy pods during drain, want %d", len(pods.Items), initialReplicas)
				}
				for i := range pods.Items {
					pod := &pods.Items[i]
					uid, ok := initialPodUIDs[pod.Name]
					if !ok || string(pod.UID) != uid {
						return fmt.Errorf("tiproxy pod %s/%s was recreated during drain", pod.Namespace, pod.Name)
					}
					if !pod.DeletionTimestamp.IsZero() {
						return fmt.Errorf("tiproxy pod %s/%s is being deleted during drain", pod.Namespace, pod.Name)
					}
				}

				tiproxies, err := listTiProxies(ctx, f, proxyg)
				if err != nil {
					return err
				}
				draining := 0
				for i := range tiproxies.Items {
					tp := &tiproxies.Items[i]
					offline := tp.Spec.Offline != nil && *tp.Spec.Offline
					if !offline || !tp.DeletionTimestamp.IsZero() {
						continue
					}
					if tp.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime] == "" {
						return fmt.Errorf("tiproxy %s/%s is offline but graceful shutdown has not begun yet", tp.Namespace, tp.Name)
					}
					if tp.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownConnectionsDrained] != "" {
						return fmt.Errorf("tiproxy %s/%s already drained its connections and is no longer revivable", tp.Namespace, tp.Name)
					}
					draining++
				}
				if draining != expectedDraining {
					return fmt.Errorf("got %d draining tiproxy instances, want %d", draining, expectedDraining)
				}
				group := &v1alpha1.TiProxyGroup{}
				if err := f.Client.Get(ctx, client.ObjectKeyFromObject(proxyg), group); err != nil {
					return err
				}
				if group.Status.DrainingReplicas != int32(expectedDraining) {
					return fmt.Errorf("TiProxyGroup status.drainingReplicas = %d, want %d", group.Status.DrainingReplicas, expectedDraining)
				}
				return nil
			}).WithTimeout(waiter.LongTaskTimeout).WithPolling(waiter.Poll).Should(gomega.Succeed())

			ginkgo.By("Scale out TiProxy and expect draining pods to be revived and reused (no new pods)")
			f.Must(f.Client.Get(ctx, client.ObjectKeyFromObject(proxyg), proxyg))
			patch = client.MergeFrom(proxyg.DeepCopy())
			proxyg.Spec.Replicas = ptr.To(initialReplicas)
			f.Must(f.Client.Patch(ctx, proxyg, patch))

			gomega.Eventually(func() error {
				pods, err := listTiProxyPods(ctx, f, proxyg)
				if err != nil {
					return err
				}
				if len(pods.Items) != int(initialReplicas) {
					return fmt.Errorf("got %d tiproxy pods after scale-out, want %d", len(pods.Items), initialReplicas)
				}
				for i := range pods.Items {
					pod := &pods.Items[i]
					uid, ok := initialPodUIDs[pod.Name]
					if !ok {
						return fmt.Errorf("tiproxy pod %s/%s was newly created instead of reviving a draining pod", pod.Namespace, pod.Name)
					}
					if string(pod.UID) != uid {
						return fmt.Errorf("tiproxy pod %s/%s was recreated (uid %s -> %s) instead of revived", pod.Namespace, pod.Name, uid, pod.UID)
					}
				}

				tiproxies, err := listTiProxies(ctx, f, proxyg)
				if err != nil {
					return err
				}
				if len(tiproxies.Items) != int(initialReplicas) {
					return fmt.Errorf("got %d tiproxy instances after scale-out, want %d", len(tiproxies.Items), initialReplicas)
				}
				for i := range tiproxies.Items {
					tp := &tiproxies.Items[i]
					if tp.Spec.Offline != nil && *tp.Spec.Offline {
						return fmt.Errorf("tiproxy %s/%s is still offline after scale-out revive", tp.Namespace, tp.Name)
					}
					if tp.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownBeginTime] != "" {
						return fmt.Errorf("tiproxy %s/%s still has graceful shutdown begin time after revive", tp.Namespace, tp.Name)
					}
					if tp.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownConnectionsDrained] != "" {
						return fmt.Errorf("tiproxy %s/%s still has connections-drained annotation after revive", tp.Namespace, tp.Name)
					}
				}
				return nil
			}).WithTimeout(waiter.LongTaskTimeout).WithPolling(waiter.Poll).Should(gomega.Succeed())

			f.WaitForTiProxyGroupReady(ctx, proxyg)

			ginkgo.By("Ensure revived TiProxy pods report healthy again")
			revivedPods, err := listTiProxyPods(ctx, f, proxyg)
			f.Must(err)
			for i := range revivedPods.Items {
				pod := &revivedPods.Items[i]
				statusCode, err := tiproxyHealthStatusCode(ctx, f, pod)
				f.Must(err)
				f.True(statusCode == http.StatusOK,
					fmt.Sprintf("revived tiproxy pod %s/%s should be healthy after revive, got status code %d", pod.Namespace, pod.Name, statusCode))
			}
		})

		ginkgo.It("create new TiProxy pods on scale-out when drained instances are no longer revivable", label.Scale, func(ctx context.Context) {
			o := desc.DefaultOptions()
			const initialReplicas int32 = 3
			const scaledInReplicas int32 = 1

			pdg := action.MustCreatePD(ctx, f, o)
			kvg := action.MustCreateTiKV(ctx, f, o)
			dbg := action.MustCreateTiDB(ctx, f, o)
			proxyg := action.MustCreateTiProxy(ctx, f, o,
				data.WithReplicas[scope.TiProxyGroup](initialReplicas),
				withTiProxyGracefulShutdownDelay(gracefulScaleInDeleteDelaySeconds),
			)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			f.WaitForTiProxyGroupReady(ctx, proxyg)

			initialPods, err := listTiProxyPods(ctx, f, proxyg)
			f.Must(err)
			gomega.Expect(initialPods.Items).To(gomega.HaveLen(int(initialReplicas)))

			ginkgo.By("Probe whether TiProxy supports the health override API")
			supportsHealthOverrideAPI, err := tiproxySupportsHealthOverrideAPI(ctx, f, &initialPods.Items[0])
			f.Must(err)
			if !supportsHealthOverrideAPI {
				ginkgo.Skip("TiProxy does not support the health override API required for connections-drained scale-in")
			}

			initialPodUIDs := podUIDSet(initialPods.Items)

			ginkgo.By("Scale in TiProxy without holding connections so drained instances become non-revivable")
			patch := client.MergeFrom(proxyg.DeepCopy())
			proxyg.Spec.Replicas = ptr.To(scaledInReplicas)
			f.Must(f.Client.Patch(ctx, proxyg, patch))

			var survivingPodUID string
			gomega.Eventually(func() error {
				pods, err := listTiProxyPods(ctx, f, proxyg)
				if err != nil {
					return err
				}
				if len(pods.Items) != int(scaledInReplicas) {
					return fmt.Errorf("got %d tiproxy pods after scale-in, want %d", len(pods.Items), scaledInReplicas)
				}

				tiproxies, err := listTiProxies(ctx, f, proxyg)
				if err != nil {
					return err
				}
				if len(tiproxies.Items) != int(scaledInReplicas) {
					return fmt.Errorf("got %d tiproxy instances after scale-in, want %d", len(tiproxies.Items), scaledInReplicas)
				}

				for name, uid := range initialPodUIDs {
					found := false
					for i := range pods.Items {
						pod := &pods.Items[i]
						if pod.Name != name {
							continue
						}
						found = true
						if string(pod.UID) != uid {
							return fmt.Errorf("surviving tiproxy pod %s/%s was recreated during scale-in", pod.Namespace, pod.Name)
						}
						survivingPodUID = uid
					}
					if !found {
						// Scaled-in pods must be fully removed once connections are drained.
						continue
					}
				}

				group := &v1alpha1.TiProxyGroup{}
				if err := f.Client.Get(ctx, client.ObjectKeyFromObject(proxyg), group); err != nil {
					return err
				}
				if group.Status.DrainingReplicas != 0 {
					return fmt.Errorf("TiProxyGroup status.drainingReplicas = %d, want 0 after scale-in completes", group.Status.DrainingReplicas)
				}
				return nil
			}).WithTimeout(waiter.LongTaskTimeout).WithPolling(waiter.Poll).Should(gomega.Succeed())
			gomega.Expect(survivingPodUID).ToNot(gomega.BeEmpty())

			ginkgo.By("Scale out TiProxy and expect new pods instead of reviving deleted drained instances")
			f.Must(f.Client.Get(ctx, client.ObjectKeyFromObject(proxyg), proxyg))
			patch = client.MergeFrom(proxyg.DeepCopy())
			proxyg.Spec.Replicas = ptr.To(initialReplicas)
			f.Must(f.Client.Patch(ctx, proxyg, patch))

			gomega.Eventually(func() error {
				pods, err := listTiProxyPods(ctx, f, proxyg)
				if err != nil {
					return err
				}
				if len(pods.Items) != int(initialReplicas) {
					return fmt.Errorf("got %d tiproxy pods after scale-out, want %d", len(pods.Items), initialReplicas)
				}

				newPods := 0
				for i := range pods.Items {
					pod := &pods.Items[i]
					uid, ok := initialPodUIDs[pod.Name]
					if !ok {
						newPods++
						continue
					}
					if string(pod.UID) != uid {
						newPods++
					}
				}
				if newPods < int(initialReplicas-scaledInReplicas) {
					return fmt.Errorf("got %d newly created tiproxy pods, want at least %d", newPods, initialReplicas-scaledInReplicas)
				}

				tiproxies, err := listTiProxies(ctx, f, proxyg)
				if err != nil {
					return err
				}
				if len(tiproxies.Items) != int(initialReplicas) {
					return fmt.Errorf("got %d tiproxy instances after scale-out, want %d", len(tiproxies.Items), initialReplicas)
				}
				for i := range tiproxies.Items {
					tp := &tiproxies.Items[i]
					if tp.Spec.Offline != nil && *tp.Spec.Offline {
						return fmt.Errorf("tiproxy %s/%s is still offline after scale-out", tp.Namespace, tp.Name)
					}
					if tp.Annotations[v1alpha1.AnnoKeyTiProxyGracefulShutdownConnectionsDrained] != "" {
						return fmt.Errorf("tiproxy %s/%s still has connections-drained annotation after scale-out", tp.Namespace, tp.Name)
					}
				}
				return nil
			}).WithTimeout(waiter.LongTaskTimeout).WithPolling(waiter.Poll).Should(gomega.Succeed())

			f.WaitForTiProxyGroupReady(ctx, proxyg)
		})

		ginkgo.It("scale in and out TiProxy when health override API is unsupported", label.Scale, func(ctx context.Context) {
			o := desc.DefaultOptions()
			const initialReplicas int32 = 3
			const scaledInReplicas int32 = 1

			pdg := action.MustCreatePD(ctx, f, o)
			kvg := action.MustCreateTiKV(ctx, f, o)
			dbg := action.MustCreateTiDB(ctx, f, o)
			proxyg := action.MustCreateTiProxy(ctx, f, o,
				data.WithReplicas[scope.TiProxyGroup](initialReplicas),
				withTiProxyGracefulShutdownDelay(gracefulScaleInDeleteDelaySeconds),
			)

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			f.WaitForTiProxyGroupReady(ctx, proxyg)

			initialPods, err := listTiProxyPods(ctx, f, proxyg)
			f.Must(err)
			gomega.Expect(initialPods.Items).To(gomega.HaveLen(int(initialReplicas)))

			ginkgo.By("Probe whether TiProxy supports the health override API")
			supportsHealthOverrideAPI, err := tiproxySupportsHealthOverrideAPI(ctx, f, &initialPods.Items[0])
			f.Must(err)
			if supportsHealthOverrideAPI {
				ginkgo.Skip("TiProxy supports the health override API; this case targets legacy versions without DELETE /api/debug/health")
			}

			initialPodUIDs := podUIDSet(initialPods.Items)

			ginkgo.By("Scale in TiProxy and manually make offline instances unhealthy to unblock graceful drain")
			patch := client.MergeFrom(proxyg.DeepCopy())
			proxyg.Spec.Replicas = ptr.To(scaledInReplicas)
			f.Must(f.Client.Patch(ctx, proxyg, patch))

			terminatedPods := map[string]struct{}{}
			ginkgo.By("Wait for scale-in to finish by deleting drained offline instances")
			gomega.Eventually(func() error {
				tiproxies, err := listTiProxies(ctx, f, proxyg)
				if err != nil {
					return err
				}
				offline := offlineTiProxyNames(tiproxies)
				if len(offline) > 0 {
					pods, err := listTiProxyPods(ctx, f, proxyg)
					if err != nil {
						return err
					}
					if err := manuallyTriggerOfflineTiProxyUnhealthy(ctx, f, true, pods.Items, offline, terminatedPods); err != nil {
						return err
					}
				}

				pods, err := listTiProxyPods(ctx, f, proxyg)
				if err != nil {
					return err
				}
				if len(pods.Items) != int(scaledInReplicas) {
					return fmt.Errorf("got %d tiproxy pods after scale-in, want %d", len(pods.Items), scaledInReplicas)
				}

				if len(tiproxies.Items) != int(scaledInReplicas) {
					return fmt.Errorf("got %d tiproxy instances after scale-in, want %d", len(tiproxies.Items), scaledInReplicas)
				}

				for i := range pods.Items {
					pod := &pods.Items[i]
					uid, ok := initialPodUIDs[pod.Name]
					if !ok || string(pod.UID) != uid {
						return fmt.Errorf("surviving tiproxy pod %s/%s was recreated unexpectedly during scale-in", pod.Namespace, pod.Name)
					}
				}
				return nil
			}).WithTimeout(waiter.LongTaskTimeout).WithPolling(waiter.Poll).Should(gomega.Succeed())

			ginkgo.By("Scale out TiProxy and expect new instances to be created")
			f.Must(f.Client.Get(ctx, client.ObjectKeyFromObject(proxyg), proxyg))
			patch = client.MergeFrom(proxyg.DeepCopy())
			proxyg.Spec.Replicas = ptr.To(initialReplicas)
			f.Must(f.Client.Patch(ctx, proxyg, patch))

			gomega.Eventually(func() error {
				pods, err := listTiProxyPods(ctx, f, proxyg)
				if err != nil {
					return err
				}
				if len(pods.Items) != int(initialReplicas) {
					return fmt.Errorf("got %d tiproxy pods after scale-out, want %d", len(pods.Items), initialReplicas)
				}

				newPods := 0
				for i := range pods.Items {
					pod := &pods.Items[i]
					uid, ok := initialPodUIDs[pod.Name]
					if !ok || string(pod.UID) != uid {
						newPods++
					}
				}
				if newPods < int(initialReplicas-scaledInReplicas) {
					return fmt.Errorf("got %d newly created tiproxy pods, want at least %d", newPods, initialReplicas-scaledInReplicas)
				}
				return nil
			}).WithTimeout(waiter.LongTaskTimeout).WithPolling(waiter.Poll).Should(gomega.Succeed())

			f.WaitForTiProxyGroupReady(ctx, proxyg)
		})
	})

	ginkgo.Context("TLS", label.P0, label.FeatureTLS, func() {
		f.SetupCluster(data.WithClusterTLSEnabled())
		workload := f.SetupWorkload()

		ginkgo.It("use different sql cert from TiDB Server", func(ctx context.Context) {
			ns := f.Namespace.Name
			tcName := f.Cluster.Name
			ginkgo.By("Installing the certificates")
			f.Must(cert.InstallTiDBIssuer(ctx, f.Client, ns, tcName))
			f.Must(cert.InstallTiDBCertificates(ctx, f.Client, ns, tcName, "dbg"))
			f.Must(cert.InstallTiProxyCertificates(ctx, f.Client, ns, tcName, "pg"))
			f.Must(cert.InstallTiDBComponentsCertificates(ctx, f.Client, ns, tcName, "pdg", "kvg", "dbg", "fg", "cg", "pg"))

			ginkgo.By("Creating the components with TLS client enabled")
			pdg := f.MustCreatePD(ctx)
			kvg := f.MustCreateTiKV(ctx)
			dbg := f.MustCreateTiDB(ctx, data.WithTLS())
			pg := f.MustCreateTiProxy(ctx, data.WithTLSForTiProxy())

			f.WaitForPDGroupReady(ctx, pdg)
			f.WaitForTiKVGroupReady(ctx, kvg)
			f.WaitForTiDBGroupReady(ctx, dbg)
			f.WaitForTiProxyGroupReady(ctx, pg)

			ginkgo.By("Checking the status of the cluster and the connection to the TiProxy service")
			checkComponent := func(podList *corev1.PodList, groupName, componentName string, expectedReplicas *int32) {
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
						// check for TiProxy & mysql client TLS
						gomega.Expect(pod.Spec.Volumes).To(gomega.ContainElement(corev1.Volume{
							Name: v1alpha1.VolumeNameTiProxyMySQLTLS,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  pg.Name + "-tiproxy-server-secret",
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

				pdPods, err := apicall.ListPods[scope.PDGroup](ctx, f.Client, pdg)
				g.Expect(err).ToNot(gomega.HaveOccurred())
				checkComponent(pdPods, pdg.Name, v1alpha1.LabelValComponentPD, pdg.Spec.Replicas)
				tikvPods, err := apicall.ListPods[scope.TiKVGroup](ctx, f.Client, kvg)
				g.Expect(err).ToNot(gomega.HaveOccurred())
				checkComponent(tikvPods, kvg.Name, v1alpha1.LabelValComponentTiKV, kvg.Spec.Replicas)
				tidbPods, err := apicall.ListPods[scope.TiDBGroup](ctx, f.Client, dbg)
				g.Expect(err).ToNot(gomega.HaveOccurred())
				checkComponent(tidbPods, dbg.Name, v1alpha1.LabelValComponentTiDB, dbg.Spec.Replicas)
				tiproxyPods, err := apicall.ListPods[scope.TiProxyGroup](ctx, f.Client, pg)
				g.Expect(err).ToNot(gomega.HaveOccurred())
				checkComponent(tiproxyPods, pg.Name, v1alpha1.LabelValComponentTiProxy, pg.Spec.Replicas)
			}).WithTimeout(waiter.LongTaskTimeout).WithPolling(waiter.Poll).Should(gomega.Succeed())

			sec := pg.Name + "-tiproxy-client-secret"
			workload.MustImportData(ctx, data.DefaultTiProxyServiceName, wopt.Port(data.DefaultTiProxyServicePort), wopt.TLS(sec, sec), wopt.RegionCount(100))
		})
	})

	// TODO(Huaxi): Add test for traffic route
})
