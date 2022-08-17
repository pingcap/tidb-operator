// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package tidbcluster

import (
	"context"
	"fmt"
	_ "net/http/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	astsHelper "github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	k8sScheme "k8s.io/client-go/kubernetes/scheme"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	restclient "k8s.io/client-go/rest"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/log"
	e2eskipper "k8s.io/kubernetes/test/e2e/framework/skipper"
	"k8s.io/utils/pointer"
	ctrlCli "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	tcconfig "github.com/pingcap/tidb-operator/pkg/apis/util/config"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/manager/member"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"
	"github.com/pingcap/tidb-operator/pkg/monitor/monitor"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	"github.com/pingcap/tidb-operator/tests"
	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	e2eframework "github.com/pingcap/tidb-operator/tests/e2e/framework"
	remotewrite "github.com/pingcap/tidb-operator/tests/e2e/remotewrite"
	utile2e "github.com/pingcap/tidb-operator/tests/e2e/util"
	utilginkgo "github.com/pingcap/tidb-operator/tests/e2e/util/ginkgo"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	utilpod "github.com/pingcap/tidb-operator/tests/e2e/util/pod"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"github.com/pingcap/tidb-operator/tests/e2e/util/proxiedpdclient"
	utiltc "github.com/pingcap/tidb-operator/tests/e2e/util/tidbcluster"
	"github.com/pingcap/tidb-operator/tests/pkg/fixture"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
)

var _ = ginkgo.Describe("TiDBCluster", func() {
	f := e2eframework.NewDefaultFramework("tidb-cluster")

	var ns string
	var c clientset.Interface
	var cli versioned.Interface
	var asCli asclientset.Interface
	var aggrCli aggregatorclient.Interface
	var apiExtCli apiextensionsclientset.Interface
	var oa *tests.OperatorActions
	var cfg *tests.Config
	var config *restclient.Config
	var ocfg *tests.OperatorConfig
	var genericCli ctrlCli.Client
	var fwCancel context.CancelFunc
	var fw portforward.PortForward
	var secretLister corelisterv1.SecretLister
	/**
	 * StatefulSet or AdvancedStatefulSet getter interface.
	 */
	var stsGetter typedappsv1.StatefulSetsGetter
	var crdUtil *tests.CrdTestUtil

	ginkgo.BeforeEach(func() {
		ns = f.Namespace.Name
		c = f.ClientSet
		secretLister = tests.GetSecretListerWithCacheSynced(c, 10*time.Second)

		var err error
		config, err = framework.LoadConfig()
		framework.ExpectNoError(err, "failed to load config")
		cli, err = versioned.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset for pingcap")
		asCli, err = asclientset.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset for advanced-statefulset")
		genericCli, err = ctrlCli.New(config, ctrlCli.Options{Scheme: scheme.Scheme})
		framework.ExpectNoError(err, "failed to create clientset for controller-runtime")
		aggrCli, err = aggregatorclient.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset kube-aggregator")
		apiExtCli, err = apiextensionsclientset.NewForConfig(config)
		framework.ExpectNoError(err, "failed to create clientset apiextensions-apiserver")
		clientRawConfig, err := e2econfig.LoadClientRawConfig()
		framework.ExpectNoError(err, "failed to load raw config for tidb-operator")
		ctx, cancel := context.WithCancel(context.Background())
		fw, err = portforward.NewPortForwarder(ctx, e2econfig.NewSimpleRESTClientGetter(clientRawConfig))
		framework.ExpectNoError(err, "failed to create port forwarder")
		fwCancel = cancel
		cfg = e2econfig.TestConfig
		// OperatorFeatures := map[string]bool{"AutoScaling": true}
		// cfg.OperatorFeatures = OperatorFeatures
		ocfg = e2econfig.NewDefaultOperatorConfig(cfg)
		if ocfg.Enabled(features.AdvancedStatefulSet) {
			stsGetter = astsHelper.NewHijackClient(c, asCli).AppsV1()
		} else {
			stsGetter = c.AppsV1()
		}
		oa = tests.NewOperatorActions(cli, c, asCli, aggrCli, apiExtCli, tests.DefaultPollInterval, ocfg, e2econfig.TestConfig, fw, f)
		crdUtil = tests.NewCrdTestUtil(cli, c, asCli, stsGetter)
	})

	ginkgo.AfterEach(func() {
		if fwCancel != nil {
			fwCancel()
		}
	})

	// basic deploy, scale out, scale in, change configuration tests
	ginkgo.Context("[TiDBCluster: Basic]", func() {
		versions := []string{utilimage.TiDBLatest}
		versions = append(versions, utilimage.TiDBPreviousVersions...)
		for _, version := range versions {
			version := version
			versionDashed := strings.ReplaceAll(version, ".", "-")
			ginkgo.Context(fmt.Sprintf("[Version: %s]", version), func() {
				ginkgo.It("should scale out tc successfully", func() {
					ginkgo.By("Deploy a basic tc")
					tc := fixture.GetTidbCluster(ns, fmt.Sprintf("basic-%s", versionDashed), version)
					tc.Spec.TiDB.Replicas = 1
					tc.Spec.TiKV.SeparateRocksDBLog = pointer.BoolPtr(true)
					tc.Spec.TiKV.SeparateRaftLog = pointer.BoolPtr(true)
					tc.Spec.TiKV.LogTailer = &v1alpha1.LogTailerSpec{
						ResourceRequirements: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("100Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("100m"),
								corev1.ResourceMemory: resource.MustParse("100Mi"),
							},
						},
					}
					_, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(context.TODO(), tc, metav1.CreateOptions{})
					framework.ExpectNoError(err, "failed to create TidbCluster: %q", tc.Name)
					err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 30*time.Second)
					framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)
					err = crdUtil.CheckDisasterTolerance(tc)
					framework.ExpectNoError(err, "failed to check disaster tolerance for TidbCluster: %q", tc.Name)

					ginkgo.By("scale out tidb, tikv, pd")
					err = controller.GuaranteedUpdate(genericCli, tc, func() error {
						tc.Spec.TiDB.Replicas = 2
						tc.Spec.TiKV.Replicas = 4
						// this must be 5, or one pd pod will not be scheduled, reference: https://docs.pingcap.com/tidb-in-kubernetes/stable/tidb-scheduler#pd-component
						tc.Spec.PD.Replicas = 5
						return nil
					})
					framework.ExpectNoError(err, "failed to scale out TidbCluster: %q", tc.Name)
					err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
					framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)
					err = crdUtil.CheckDisasterTolerance(tc)
					framework.ExpectNoError(err, "failed to check disaster tolerance for TidbCluster: %q", tc.Name)
				})

				ginkgo.It("should scale in tc successfully", func() {
					ginkgo.By("Deploy a basic tc")
					tc := fixture.GetTidbCluster(ns, fmt.Sprintf("basic-%s", versionDashed), version)
					tc.Spec.TiKV.Replicas = 4
					tc.Spec.PD.Replicas = 5
					_, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(context.TODO(), tc, metav1.CreateOptions{})
					framework.ExpectNoError(err, "failed to create TidbCluster: %q", tc.Name)
					err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 30*time.Second)
					framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)
					err = crdUtil.CheckDisasterTolerance(tc)
					framework.ExpectNoError(err, "failed to check disaster tolerance for TidbCluster: %q", tc.Name)

					ginkgo.By("Scale in tidb, tikv, pd")
					err = controller.GuaranteedUpdate(genericCli, tc, func() error {
						tc.Spec.TiDB.Replicas = 1
						tc.Spec.TiKV.Replicas = 3
						tc.Spec.PD.Replicas = 3
						return nil
					})
					framework.ExpectNoError(err, "failed to scale in TidbCluster: %q", tc.Name)
					err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
					framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)
					err = crdUtil.CheckDisasterTolerance(tc)
					framework.ExpectNoError(err, "failed to check disaster tolerance for TidbCluster: %q", tc.Name)
				})

				// only test with the latest version for avoid too much cases.
				if version == utilimage.TiDBLatest {
					ginkgo.It("should enable and disable binlog normal", func() {
						ginkgo.By("Deploy a basic tc")
						tc := fixture.GetTidbCluster(ns, fmt.Sprintf("basic-%s", versionDashed), version)
						_, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(context.TODO(), tc, metav1.CreateOptions{})
						framework.ExpectNoError(err, "failed to create TidbCluster: %q", tc.Name)
						err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 30*time.Second)
						framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)
						err = crdUtil.CheckDisasterTolerance(tc)
						framework.ExpectNoError(err, "failed to check disaster tolerance for TidbCluster: %q", tc.Name)

						testBinlog(oa, tc, genericCli, cli)
					})
				}

				ginkgo.It("should change configurations successfully", func() {
					ginkgo.By("Deploy a basic tc")
					tc := fixture.GetTidbCluster(ns, fmt.Sprintf("basic-%s", versionDashed), version)
					tc.Spec.TiDB.Replicas = 1
					_, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(context.TODO(), tc, metav1.CreateOptions{})
					framework.ExpectNoError(err, "failed to create TidbCluster: %q", tc.Name)
					err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 30*time.Second)
					framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)
					err = crdUtil.CheckDisasterTolerance(tc)
					framework.ExpectNoError(err, "failed to check disaster tolerance for TidbCluster: %q", tc.Name)

					ginkgo.By("Change configuration")
					err = controller.GuaranteedUpdate(genericCli, tc, func() error {
						tc.Spec.ConfigUpdateStrategy = v1alpha1.ConfigUpdateStrategyRollingUpdate
						tc.Spec.PD.MaxFailoverCount = pointer.Int32Ptr(4)
						tc.Spec.TiKV.MaxFailoverCount = pointer.Int32Ptr(4)
						tc.Spec.TiDB.MaxFailoverCount = pointer.Int32Ptr(4)
						return nil
					})
					framework.ExpectNoError(err, "failed to change configuration of TidbCluster: %q", tc.Name)
					err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
					framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)

					ginkgo.By("Check custom labels and annotations will not lost")
					framework.ExpectNoError(checkCustomLabelAndAnn(tc, c, tc.Spec.Labels[fixture.ClusterCustomKey], time.Minute, 10*time.Second), "failed to check labels and annotations")

					ginkgo.By("Change labels and annotations")
					newValue := "new-value"
					err = controller.GuaranteedUpdate(genericCli, tc, func() error {
						tc.Spec.Labels[fixture.ClusterCustomKey] = newValue
						tc.Spec.Annotations[fixture.ClusterCustomKey] = newValue
						tc.Spec.PD.ComponentSpec.Labels[fixture.ComponentCustomKey] = newValue
						tc.Spec.PD.ComponentSpec.Annotations[fixture.ComponentCustomKey] = newValue
						tc.Spec.TiKV.ComponentSpec.Labels[fixture.ComponentCustomKey] = newValue
						tc.Spec.TiKV.ComponentSpec.Annotations[fixture.ComponentCustomKey] = newValue
						tc.Spec.TiDB.ComponentSpec.Labels[fixture.ComponentCustomKey] = newValue
						tc.Spec.TiDB.ComponentSpec.Annotations[fixture.ComponentCustomKey] = newValue
						tc.Spec.TiDB.Service.Labels[fixture.ComponentCustomKey] = newValue
						tc.Spec.TiDB.Service.Annotations[fixture.ComponentCustomKey] = newValue
						return nil
					})
					framework.ExpectNoError(err, "failed to change configuration of TidbCluster: %q", tc.Name)
					err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
					framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)

					ginkgo.By("Check custom labels and annotations changed")
					framework.ExpectNoError(checkCustomLabelAndAnn(tc, c, newValue, 10*time.Minute, 10*time.Second), "failed to check labels and annotations")
				})
			})
		}
	})

	/**
	 * This test case switches back and forth between pod network and host network of a single cluster.
	 * Note that only one cluster can run in host network mode at the same time.
	 */
	ginkgo.It("should switch between pod network and host network", func() {
		if !ocfg.Enabled(features.AdvancedStatefulSet) {
			serverVersion, err := c.Discovery().ServerVersion()
			framework.ExpectNoError(err, "failed to fetch Kubernetes version")
			sv := utilversion.MustParseSemantic(serverVersion.GitVersion)
			log.Logf("ServerVersion: %v", serverVersion.String())
			if sv.LessThan(utilversion.MustParseSemantic("v1.13.11")) || // < v1.13.11
				(sv.AtLeast(utilversion.MustParseSemantic("v1.14.0")) && sv.LessThan(utilversion.MustParseSemantic("v1.14.7"))) || // >= v1.14.0 and < v1.14.7
				(sv.AtLeast(utilversion.MustParseSemantic("v1.15.0")) && sv.LessThan(utilversion.MustParseSemantic("v1.15.4"))) { // >= v1.15.0 and < v1.15.4
				// https://github.com/pingcap/tidb-operator/issues/1042#issuecomment-547742565
				e2eskipper.Skipf("Skipping HostNetwork test. Kubernetes %v has a bug that StatefulSet may apply revision incorrectly, HostNetwork cannot work well in this cluster", serverVersion)
			}
			log.Logf("Testing HostNetwork feature with Kubernetes %v", serverVersion)
		} else {
			log.Logf("Testing HostNetwork feature with Advanced StatefulSet")
		}

		ginkgo.By("Deploy initial tc")
		clusterName := "host-network"
		tc := fixture.GetTidbCluster(ns, clusterName, utilimage.TiDBLatest)
		tc = fixture.AddTiFlashForTidbCluster(tc)
		tc = fixture.AddTiCDCForTidbCluster(tc)
		tc = fixture.AddPumpForTidbCluster(tc)

		// Set some properties
		tc.Spec.PD.Replicas = 1
		tc.Spec.TiKV.Replicas = 1
		tc.Spec.TiDB.Replicas = 1
		// Create and wait for tidbcluster ready
		utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 6*time.Minute, 5*time.Second)
		ginkgo.By("Switch to host network")
		// TODO: Considering other components?
		err := controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.HostNetwork = pointer.BoolPtr(true)
			tc.Spec.PD.HostNetwork = pointer.BoolPtr(true)
			tc.Spec.TiKV.HostNetwork = pointer.BoolPtr(true)
			tc.Spec.TiDB.HostNetwork = pointer.BoolPtr(true)
			return nil
		})
		framework.ExpectNoError(err, "failed to switch to host network, TidbCluster: %q", tc.Name)
		err = oa.WaitForTidbClusterReady(tc, 3*time.Minute, 5*time.Second)
		framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)

		ginkgo.By("Switch back to pod network")
		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.HostNetwork = pointer.BoolPtr(false)
			tc.Spec.PD.HostNetwork = pointer.BoolPtr(false)
			tc.Spec.TiKV.HostNetwork = pointer.BoolPtr(false)
			tc.Spec.TiDB.HostNetwork = pointer.BoolPtr(false)
			return nil
		})
		framework.ExpectNoError(err, "failed to switch to pod network, TidbCluster: %q", tc.Name)
		err = oa.WaitForTidbClusterReady(tc, 3*time.Minute, 5*time.Second)
		framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)
	})

	ginkgo.It("should direct upgrade tc successfully when PD replicas less than 2.", func() {
		clusterName := "upgrade-cluster-pd-1"
		tc := fixture.GetTidbCluster(ns, clusterName, utilimage.TiDBLatestPrev)
		tc.Spec.PD.Replicas = 1
		tc.Spec.TiDB.Replicas = 1
		tc.Spec.TiKV.Replicas = 1
		tc.Spec.PD.BaseImage = "pingcap/pd-not-exist"
		// Deploy
		err := genericCli.Create(context.TODO(), tc)
		framework.ExpectNoError(err, "failed to create TidbCluster %s/%s", tc.Namespace, tc.Name)

		err = utiltc.WaitForTCCondition(cli, tc.Namespace, tc.Name, time.Minute*5, time.Second*10,
			func(tc *v1alpha1.TidbCluster) (bool, error) {
				if len(tc.Status.Conditions) < 1 {
					return false, nil
				}
				if tc.Status.Conditions[0].Reason == "PDUnhealthy" &&
					tc.Status.PD.StatefulSet.CurrentReplicas == tc.Status.PD.StatefulSet.Replicas {
					return true, nil
				}
				return false, nil
			})
		framework.ExpectNoError(err)

		ginkgo.By("Force Upgrading tidb cluster ignoring PD error")
		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.PD.BaseImage = "pingcap/pd"
			return nil
		})
		framework.ExpectNoError(err, "failed to upgrade TidbCluster: %q", tc.Name)
		err = oa.WaitForTidbClusterReady(tc, 10*time.Minute, 5*time.Second)

		framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)
	})

	ginkgo.It("should direct upgrade tc successfully when TiKV replicas less than 2.", func() {
		clusterName := "upgrade-cluster-tikv-1"
		tc := fixture.GetTidbCluster(ns, clusterName, utilimage.TiDBLatest)
		tc.Spec.PD.Replicas = 1
		tc.Spec.TiDB.Replicas = 1
		tc.Spec.TiKV.Version = pointer.StringPtr(utilimage.TiDBLatestPrev)
		tc.Spec.TiKV.Replicas = 1
		tc.Spec.TiKV.EvictLeaderTimeout = pointer.StringPtr("10m")
		// Deploy
		utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 10*time.Minute, 5*time.Second)
		ginkgo.By(fmt.Sprintf("Upgrading tidb cluster from %s to %s", tc.Spec.Version, utilimage.TiDBLatest))
		err := controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.Version = utilimage.TiDBLatest
			return nil
		})
		framework.ExpectNoError(err, "failed to upgrade TidbCluster: %q", tc.Name)
		err = oa.WaitForTidbClusterReady(tc, 7*time.Minute, 5*time.Second)

		framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)
		pdClient, cancel, err := proxiedpdclient.NewProxiedPDClient(secretLister, fw, ns, tc.Name, false)
		framework.ExpectNoError(err, "failed to create proxied PD client")
		defer cancel()

		evictLeaderSchedulers, err := pdClient.GetEvictLeaderSchedulers()
		framework.ExpectNoError(err, "failed to get EvictLeader")
		res := utiltc.MustPDHasScheduler(evictLeaderSchedulers, "evict-leader-scheduler")
		framework.ExpectEqual(res, false)
	})

	// TODO: move into TiDBMonitor specific group
	ginkgo.It("should manage tidb monitor normally", func() {
		ginkgo.By("Deploy initial tc")
		tc := fixture.GetTidbCluster(ns, "monitor-test", utilimage.TiDBLatest)
		tc.Spec.PD.Replicas = 1
		tc.Spec.TiKV.Replicas = 1
		tc.Spec.TiDB.Replicas = 1
		tc, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(context.TODO(), tc, metav1.CreateOptions{})
		framework.ExpectNoError(err, "Expected create tidbcluster")
		err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
		framework.ExpectNoError(err, "Expected get tidbcluster")

		ginkgo.By("Deploy tidbmonitor")
		tm := fixture.NewTidbMonitor("monitor-test", ns, tc, true, true, false)
		pvpDelete := corev1.PersistentVolumeReclaimDelete
		tm.Spec.PVReclaimPolicy = &pvpDelete
		_, err = cli.PingcapV1alpha1().TidbMonitors(ns).Create(context.TODO(), tm, metav1.CreateOptions{})
		framework.ExpectNoError(err, "Expected tidbmonitor deployed success")
		err = tests.CheckTidbMonitor(tm, cli, c, fw)
		framework.ExpectNoError(err, "Expected tidbmonitor checked success")

		ginkgo.By("Check tidbmonitor pv label")
		err = wait.Poll(5*time.Second, 2*time.Minute, func() (done bool, err error) {
			pvc, err := c.CoreV1().PersistentVolumeClaims(ns).Get(context.TODO(), monitor.GetMonitorFirstPVCName(tm.Name), metav1.GetOptions{})
			if err != nil {
				log.Logf("failed to get monitor pvc")
				return false, nil
			}
			pvName := pvc.Spec.VolumeName
			pv, err := c.CoreV1().PersistentVolumes().Get(context.TODO(), pvName, metav1.GetOptions{})
			if err != nil {
				log.Logf("failed to get monitor pv")
				return false, nil
			}

			value, existed := pv.Labels[label.ComponentLabelKey]
			if !existed || value != label.TiDBMonitorVal {
				log.Logf("label %q not synced", label.ComponentLabelKey)
				return false, nil
			}
			value, existed = pv.Labels[label.InstanceLabelKey]
			if !existed || value != "monitor-test" {
				log.Logf("label %q not synced", label.InstanceLabelKey)
				return false, nil
			}
			value, existed = pv.Labels[label.NameLabelKey]
			if !existed || value != "tidb-cluster" {
				log.Logf("label %q not synced", label.NameLabelKey)
				return false, nil
			}
			value, existed = pv.Labels[label.ManagedByLabelKey]
			if !existed || value != label.TiDBOperator {
				log.Logf("label %q not synced", label.ManagedByLabelKey)
				return false, nil
			}
			if pv.Spec.PersistentVolumeReclaimPolicy != corev1.PersistentVolumeReclaimDelete {
				return false, fmt.Errorf("pv[%s]'s policy is not \"Delete\"", pv.Name)
			}
			return true, nil
		})
		framework.ExpectNoError(err, "tidbmonitor pv label error")

		// update TidbMonitor and check whether portName is updated and the nodePort is unchanged
		ginkgo.By("Update tidbmonitor service")
		err = controller.GuaranteedUpdate(genericCli, tm, func() error {
			tm.Spec.Prometheus.Service.Type = corev1.ServiceTypeNodePort
			retainPVP := corev1.PersistentVolumeReclaimRetain
			tm.Spec.PVReclaimPolicy = &retainPVP
			return nil
		})
		framework.ExpectNoError(err, "update tidbmonitor service type error")

		var targetPort int32
		err = wait.Poll(5*time.Second, 3*time.Minute, func() (done bool, err error) {
			prometheusSvc, err := c.CoreV1().Services(ns).Get(context.TODO(), fmt.Sprintf("%s-prometheus", tm.Name), metav1.GetOptions{})
			if err != nil {
				return false, nil
			}
			if len(prometheusSvc.Spec.Ports) != 3 {
				return false, nil
			}
			if prometheusSvc.Spec.Type != corev1.ServiceTypeNodePort {
				return false, nil
			}
			targetPort = prometheusSvc.Spec.Ports[0].NodePort
			return true, nil
		})
		framework.ExpectNoError(err, "failed to wait for tidbmonitor service to be changed")

		err = controller.GuaranteedUpdate(genericCli, tm, func() error {
			newPortName := "any-other-word"
			tm.Spec.Prometheus.Service.PortName = &newPortName
			return nil
		})
		framework.ExpectNoError(err, "update tidbmonitor service portName error")

		pvc, err := c.CoreV1().PersistentVolumeClaims(ns).Get(context.TODO(), monitor.GetMonitorFirstPVCName(tm.Name), metav1.GetOptions{})
		framework.ExpectNoError(err, "Expected fetch tidbmonitor pvc success")
		pvName := pvc.Spec.VolumeName
		err = wait.Poll(5*time.Second, 3*time.Minute, func() (done bool, err error) {
			prometheusSvc, err := c.CoreV1().Services(ns).Get(context.TODO(), fmt.Sprintf("%s-prometheus", tm.Name), metav1.GetOptions{})
			if err != nil {
				return false, nil
			}
			if len(prometheusSvc.Spec.Ports) != 3 {
				return false, nil
			}
			if prometheusSvc.Spec.Type != corev1.ServiceTypeNodePort {
				log.Logf("prometheus service type haven't be changed")
				return false, nil
			}
			if prometheusSvc.Spec.Ports[0].Name != "any-other-word" {
				log.Logf("prometheus port name haven't be changed")
				return false, nil
			}
			if prometheusSvc.Spec.Ports[0].NodePort != targetPort {
				return false, nil
			}
			pv, err := c.CoreV1().PersistentVolumes().Get(context.TODO(), pvName, metav1.GetOptions{})
			if err != nil {
				return false, nil
			}
			if pv.Spec.PersistentVolumeReclaimPolicy != corev1.PersistentVolumeReclaimRetain {
				log.Logf("prometheus PersistentVolumeReclaimPolicy haven't be changed")
				return false, nil
			}
			return true, nil
		})
		framework.ExpectNoError(err, "second update tidbmonitor service error")

		ginkgo.By("Verify tidbmonitor and tidbcluster PVReclaimPolicy won't affect each other")
		err = wait.Poll(5*time.Second, 3*time.Minute, func() (done bool, err error) {
			tc, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(context.TODO(), tc.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			tm, err = cli.PingcapV1alpha1().TidbMonitors(ns).Get(context.TODO(), tm.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if *tc.Spec.PVReclaimPolicy != corev1.PersistentVolumeReclaimDelete {
				log.Logf("tidbcluster PVReclaimPolicy changed into %v", *tc.Spec.PVReclaimPolicy)
				return true, nil
			}
			if *tm.Spec.PVReclaimPolicy != corev1.PersistentVolumeReclaimRetain {
				log.Logf("tidbmonitor PVReclaimPolicy changed into %v", *tm.Spec.PVReclaimPolicy)
				return true, nil
			}
			return false, nil
		})
		framework.ExpectEqual(err, wait.ErrWaitTimeout, "tidbmonitor and tidbcluster PVReclaimPolicy should not affect each other")

		ginkgo.By("Check custom labels and annotations")
		checkMonitorCustomLabelAndAnn(tm, c)

		ginkgo.By("enable dynamic configuration update")
		err = controller.GuaranteedUpdate(genericCli, tm, func() error {
			tm.Spec.PrometheusReloader = &v1alpha1.PrometheusReloaderSpec{
				MonitorContainer: v1alpha1.MonitorContainer{
					BaseImage: "quay.io/prometheus-operator/prometheus-config-reloader",
					Version:   "v0.49.0",
				},
			}
			return nil
		})
		framework.ExpectNoError(err, "enable tidbmonitor dynamic configuration error")

		secondTc := fixture.GetTidbCluster(ns, "monitor-test-second", utilimage.TiDBLatest)
		secondTc.Spec.PD.Replicas = 1
		secondTc.Spec.TiKV.Replicas = 1
		secondTc.Spec.TiDB.Replicas = 1
		secondTc, err = cli.PingcapV1alpha1().TidbClusters(secondTc.Namespace).Create(context.TODO(), secondTc, metav1.CreateOptions{})
		framework.ExpectNoError(err, "Expected create tidbcluster")
		err = oa.WaitForTidbClusterReady(secondTc, 30*time.Minute, 5*time.Second)
		framework.ExpectNoError(err, "Expected get secondTc tidbcluster")
		ginkgo.By("update tidbmonitor cluster spec")
		err = controller.GuaranteedUpdate(genericCli, tm, func() error {
			tm.Spec.Clusters = []v1alpha1.TidbClusterRef{
				{
					Name: "monitor-test",
				},
				{
					Name: "monitor-test-second",
				},
			}
			return nil
		})
		framework.ExpectNoError(err, "update tidbmonitor cluster spec error")
		err = tests.CheckTidbMonitorConfigurationUpdate(tm, c, fw, 5)
		framework.ExpectNoError(err, "Expected tidbmonitor dynamic configuration checked success")

		ginkgo.By("Delete tidbmonitor")
		err = cli.PingcapV1alpha1().TidbMonitors(tm.Namespace).Delete(context.TODO(), tm.Name, metav1.DeleteOptions{})
		framework.ExpectNoError(err, "delete tidbmonitor failed")
	})

	ginkgo.It("can be paused and resumed", func() {
		ginkgo.By("Deploy initial tc")
		tcName := "paused"
		tc := fixture.GetTidbCluster(ns, tcName, utilimage.TiDBLatestPrev)
		tc.Spec.PD.Replicas = 1
		tc.Spec.TiKV.Replicas = 1
		tc.Spec.TiDB.Replicas = 1
		utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 30*time.Minute, 5*time.Second)

		podListBeforePaused, err := c.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
		framework.ExpectNoError(err, "failed to list pods in ns %s", ns)

		ginkgo.By("Pause the tidb cluster")
		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.Paused = true
			return nil
		})
		framework.ExpectNoError(err, "failed to pause TidbCluster: %q", tc.Name)

		ginkgo.By(fmt.Sprintf("upgrade tc version to %q", utilimage.TiDBLatest))
		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.Version = utilimage.TiDBLatest
			return nil
		})
		framework.ExpectNoError(err, "failed to upgrade TidbCluster version: %q", tc.Name)

		ginkgo.By("Check pods are not changed when the tidb cluster is paused for 3 min")
		err = utilpod.WaitForPodsAreChanged(c, podListBeforePaused.Items, 3*time.Minute)
		framework.ExpectEqual(err, wait.ErrWaitTimeout, "Pods are changed when the tidb cluster is paused")

		ginkgo.By("Resume the tidb cluster")
		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.Paused = false
			return nil
		})
		framework.ExpectNoError(err, "failed to resume TidbCluster: %q", tc.Name)

		ginkgo.By("Check the tidb cluster will be upgraded")
		listOptions := metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(label.New().Instance(tcName).Component(label.TiKVLabelVal).Labels()).String(),
		}
		err = wait.PollImmediate(10*time.Second, 10*time.Minute, func() (bool, error) {
			podList, err := c.CoreV1().Pods(ns).List(context.TODO(), listOptions)
			if err != nil && !apierrors.IsNotFound(err) {
				log.Logf("failed to list pods: %+v", err)
				return false, err
			}
			for _, pod := range podList.Items {
				for _, c := range pod.Spec.Containers {
					if c.Name == v1alpha1.TiKVMemberType.String() {
						log.Logf("c.Image: %s", c.Image)
						if c.Image == tc.TiKVImage() {
							return true, nil
						}
					}
				}
			}
			return false, nil
		})
		framework.ExpectNoError(err, "failed to wait for tikv upgraded: %q", tc.Name)
	})

	ginkgo.Context("[Feature: AutoFailover]", func() {
		// TODO: explain purpose of this case
		ginkgo.It("should clear TiDB failureMembers when scale TiDB to zero", func() {
			ginkgo.By("Deploy initial tc")
			tc := fixture.GetTidbCluster(ns, "tidb-scale", utilimage.TiDBLatest)
			tc.Spec.PD.Replicas = 1
			tc.Spec.TiKV.Replicas = 1
			tc.Spec.TiDB.Replicas = 1
			tc.Spec.TiDB.Config.Set("log.file.max-size", "300")
			// Deploy
			err := genericCli.Create(context.TODO(), tc)
			framework.ExpectNoError(err, "Expected TiDB cluster created")

			ginkgo.By("check tidb failure member count")
			ns := tc.Namespace
			tcName := tc.Name

			err = wait.PollImmediate(5*time.Second, 10*time.Minute, func() (bool, error) {
				var tc *v1alpha1.TidbCluster
				var err error
				if tc, err = cli.PingcapV1alpha1().TidbClusters(ns).Get(context.TODO(), tcName, metav1.GetOptions{}); err != nil {
					log.Logf("failed to get tidbcluster: %s/%s, %v", ns, tcName, err)
					return false, nil
				}
				if len(tc.Status.TiDB.FailureMembers) == 1 {
					return true, nil
				}
				log.Logf("the number of failed tidb member is not 1 (current: %d)", len(tc.Status.TiDB.FailureMembers))
				return false, nil
			})
			framework.ExpectNoError(err, "expect tidb failure member count = 1")

			ginkgo.By("Scale tidb to 0")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.TiDB.Replicas = 0
				return nil
			})
			framework.ExpectNoError(err, "failed to scale tidb to 0, TidbCluster: %q", tc.Name)

			err = wait.PollImmediate(5*time.Second, 3*time.Minute, func() (bool, error) {
				var tc *v1alpha1.TidbCluster
				var err error
				if tc, err = cli.PingcapV1alpha1().TidbClusters(ns).Get(context.TODO(), tcName, metav1.GetOptions{}); err != nil {
					log.Logf("failed to get tidbcluster: %s/%s, %v", ns, tcName, err)
					return false, nil
				}
				if tc.Status.TiDB.StatefulSet.Replicas != 0 {
					log.Logf("failed to scale tidb member to zero (current: %d)", tc.Status.TiDB.StatefulSet.Replicas)
					return false, nil
				}
				if len(tc.Status.TiDB.FailureMembers) != 0 {
					log.Logf("failed to clear fail member (current: %d)", len(tc.Status.TiDB.FailureMembers))
					return false, nil
				}
				log.Logf("scale tidb member to zero successfully")
				return true, nil
			})
			framework.ExpectNoError(err, "not clear TiDB failureMembers when scale TiDB to zero")
		})

		ginkgo.It("should failover and recover by recoverByUID", func() {
			clusterName := "recover-by-uid"
			tc := fixture.GetTidbCluster(ns, clusterName, utilimage.TiDBLatest)
			tc = fixture.AddTiFlashForTidbCluster(tc)
			tc.Spec.PD.Replicas = 1
			// By default, PD set the state of disconnected store to Down
			// after 30 minutes. Use a short time in testing.
			tc.Spec.PD.Config.Set("schedule.max-store-down-time", "1m")
			tc.Spec.TiKV.Replicas = 3
			tc.Spec.TiDB.Replicas = 1
			tc.Spec.TiFlash.Replicas = 3
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 30*time.Minute, 15*time.Second)

			downTiKVPodName := controller.TiKVMemberName(clusterName) + "-1"
			downTiFlashPodName := controller.TiFlashMemberName(clusterName) + "-1"
			failoverTiKVPodName := controller.TiKVMemberName(clusterName) + "-3"
			failoverTiFlashPodName := controller.TiFlashMemberName(clusterName) + "-3"

			ginkgo.By("Down pods tikv-1 and tiflash-1 and waiting to become Down")
			f.ExecCommandInContainer(downTiKVPodName, "tikv", "sh", "-c", "mv /var/lib/tikv/db /var/lib/tikv/db_down")
			oa.RunKubectlOrDie("set", "image", "-n", ns, "pods", downTiFlashPodName, "tiflash=pingcap/tiflash:null")
			err := utiltc.WaitForTCCondition(cli, tc.Namespace, tc.Name, time.Minute*5, time.Second*10,
				func(tc *v1alpha1.TidbCluster) (bool, error) {
					tikvIsDown, tiflashIsDown := false, false
					for _, store := range tc.Status.TiKV.Stores {
						if store.PodName == downTiKVPodName {
							tikvIsDown = store.State == v1alpha1.TiKVStateDown
							break
						}
					}
					for _, store := range tc.Status.TiFlash.Stores {
						if strings.HasPrefix(store.IP, downTiFlashPodName) {
							tiflashIsDown = store.State == v1alpha1.TiKVStateDown
							break
						}
					}
					return tikvIsDown && tiflashIsDown, nil
				})
			framework.ExpectNoError(err, "failed to waiting for stores to become Down")

			ginkgo.By("Waiting for stores to be recorded in failure stores")
			err = utiltc.WaitForTCCondition(cli, tc.Namespace, tc.Name, time.Minute*10, time.Second*10,
				func(tc *v1alpha1.TidbCluster) (bool, error) {
					tikvIsRecorded, tiflashIsRecorded := false, false
					if tc.Status.TiKV.FailoverUID != "" {
						for _, failureStore := range tc.Status.TiKV.FailureStores {
							if failureStore.PodName == downTiKVPodName {
								tikvIsRecorded = true
								break
							}
						}
					}
					if tc.Status.TiFlash.FailoverUID != "" {
						for _, failureStore := range tc.Status.TiFlash.FailureStores {
							if failureStore.PodName == downTiFlashPodName {
								tiflashIsRecorded = true
								break
							}
						}
					}
					return tikvIsRecorded && tiflashIsRecorded, nil
				})
			framework.ExpectNoError(err, "failed to wait for stores to be recorded into failure stores")

			ginkgo.By("Waiting for failover pods to be created")
			err, lastReason := utile2e.PollImmediateWithReason(time.Second*10, time.Minute*5, func() (bool, error, string) {
				pods := []string{failoverTiKVPodName, failoverTiFlashPodName}
				for _, pod := range pods {
					if _, err := c.CoreV1().Pods(ns).Get(context.TODO(), pod, metav1.GetOptions{}); err != nil {
						return false, nil, fmt.Sprintf("failed to get pod %s: %v", pod, err)
					}
				}
				return true, nil, ""
			})
			framework.ExpectNoError(err, "failed to wait for failover pods to be created, last reason: %s", lastReason)

			ginkgo.By("Up down stores and waiting to become Up")
			f.ExecCommandInContainer(downTiKVPodName, "tikv", "sh", "-c", "mv /var/lib/tikv/db_down /var/lib/tikv/db")
			oa.RunKubectlOrDie("set", "image", "-n", ns, "pods", downTiFlashPodName, fmt.Sprintf("tiflash=pingcap/tiflash:%s", utilimage.TiDBLatest))
			err = utiltc.WaitForTCCondition(cli, tc.Namespace, tc.Name, time.Minute*5, time.Second*10,
				func(tc *v1alpha1.TidbCluster) (bool, error) {
					tikvIsUp, tiflashIsUp := false, false
					for _, store := range tc.Status.TiKV.Stores {
						if store.PodName == downTiKVPodName {
							tikvIsUp = store.State == v1alpha1.TiKVStateUp
							break
						}
					}
					for _, store := range tc.Status.TiFlash.Stores {
						if strings.HasPrefix(store.IP, downTiFlashPodName) {
							tiflashIsUp = store.State == v1alpha1.TiKVStateUp
							break
						}
					}
					return tikvIsUp && tiflashIsUp, nil
				})
			framework.ExpectNoError(err, "failed to waiting for down stores to become Up")

			ginkgo.By("Update failover.recoverByUID of tikv and tiflash")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.TiKV.RecoverFailover = false
				tc.Spec.TiFlash.RecoverFailover = false
				tc.Spec.TiKV.Failover = &v1alpha1.Failover{RecoverByUID: tc.Status.TiKV.FailoverUID}
				tc.Spec.TiFlash.Failover = &v1alpha1.Failover{RecoverByUID: tc.Status.TiFlash.FailoverUID}
				return nil
			})
			framework.ExpectNoError(err, "failed to update failover.recoverByUID of tikv and tiflash")

			ginkgo.By("Waiting for failureStore to be empty after recovery")
			err = utiltc.WaitForTCCondition(cli, tc.Namespace, tc.Name, time.Minute*5, time.Second*10,
				func(tc *v1alpha1.TidbCluster) (bool, error) {
					tikvFailureStoreIsEmpty := len(tc.Status.TiKV.FailureStores) == 0 && tc.Status.TiKV.FailoverUID == ""
					tiflasFailureStoreIsEmpty := len(tc.Status.TiFlash.FailureStores) == 0 && tc.Status.TiFlash.FailoverUID == ""
					return tikvFailureStoreIsEmpty && tiflasFailureStoreIsEmpty, nil
				})
			framework.ExpectNoError(err, "failed to wait for failureStore to be empty after recovery")

			ginkgo.By("Waiting for failover pods to be deleted after recovery")
			err, lastReason = utile2e.PollImmediateWithReason(time.Second*10, time.Minute*5, func() (bool, error, string) {
				pods := []string{failoverTiKVPodName, failoverTiFlashPodName}

				for _, pod := range pods {
					_, err := c.CoreV1().Pods(ns).Get(context.TODO(), pod, metav1.GetOptions{})
					if err == nil {
						return false, nil, fmt.Sprintf("pod %s still exists", pod)
					}
					if err != nil && !apierrors.IsNotFound(err) {
						return false, nil, fmt.Sprintf("failed to get pod %s: %v", pod, err)
					}
				}

				return true, nil, ""
			})
			framework.ExpectNoError(err, "failed to wait for failover pods to be deleted after recovery, last reason: %s", lastReason)
		})

		ginkgo.It("should failover and recover by recoverFailover eventually", func() {
			clusterName := "recover"
			tc := fixture.GetTidbCluster(ns, clusterName, utilimage.TiDBLatest)
			tc = fixture.AddTiFlashForTidbCluster(tc)
			tc.Spec.PD.Replicas = 1
			// By default, PD set the state of disconnected store to Down
			// after 30 minutes. Use a short time in testing.
			tc.Spec.PD.Config.Set("schedule.max-store-down-time", "1m")
			tc.Spec.TiKV.Replicas = 3
			tc.Spec.TiDB.Replicas = 1
			tc.Spec.TiFlash.Replicas = 3
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 30*time.Minute, 15*time.Second)

			downTiKVPodName := controller.TiKVMemberName(clusterName) + "-1"
			downTiFlashPodName := controller.TiFlashMemberName(clusterName) + "-1"
			failoverTiKVPodName := controller.TiKVMemberName(clusterName) + "-3"
			failoverTiFlashPodName := controller.TiFlashMemberName(clusterName) + "-3"

			ginkgo.By("Down pods tikv-1 and tiflash-1 and waiting to become Down")
			f.ExecCommandInContainer(downTiKVPodName, "tikv", "sh", "-c", "mv /var/lib/tikv/db /var/lib/tikv/db_down")
			oa.RunKubectlOrDie("set", "image", "-n", ns, "pods", downTiFlashPodName, "tiflash=pingcap/tiflash:null")
			err := utiltc.WaitForTCCondition(cli, tc.Namespace, tc.Name, time.Minute*5, time.Second*10,
				func(tc *v1alpha1.TidbCluster) (bool, error) {
					tikvIsDown, tiflashIsDown := false, false
					for _, store := range tc.Status.TiKV.Stores {
						if store.PodName == downTiKVPodName {
							tikvIsDown = store.State == v1alpha1.TiKVStateDown
							break
						}
					}
					for _, store := range tc.Status.TiFlash.Stores {
						if strings.HasPrefix(store.IP, downTiFlashPodName) {
							tiflashIsDown = store.State == v1alpha1.TiKVStateDown
							break
						}
					}
					return tikvIsDown && tiflashIsDown, nil
				})
			framework.ExpectNoError(err, "failed to waiting for stores to become Down")

			ginkgo.By("Waiting for stores to be recorded in failure stores")
			err = utiltc.WaitForTCCondition(cli, tc.Namespace, tc.Name, time.Minute*10, time.Second*10,
				func(tc *v1alpha1.TidbCluster) (bool, error) {
					tikvIsRecorded, tiflashIsRecorded := false, false
					if tc.Status.TiKV.FailoverUID != "" {
						for _, failureStore := range tc.Status.TiKV.FailureStores {
							if failureStore.PodName == downTiKVPodName {
								tikvIsRecorded = true
								break
							}
						}
					}
					if tc.Status.TiFlash.FailoverUID != "" {
						for _, failureStore := range tc.Status.TiFlash.FailureStores {
							if failureStore.PodName == downTiFlashPodName {
								tiflashIsRecorded = true
								break
							}
						}
					}
					return tikvIsRecorded && tiflashIsRecorded, nil
				})
			framework.ExpectNoError(err, "failed to wait for stores to be recorded into failure stores")

			ginkgo.By("Waiting for failover pods to be created")
			err, lastReason := utile2e.PollImmediateWithReason(time.Second*10, time.Minute*5, func() (bool, error, string) {
				pods := []string{failoverTiKVPodName, failoverTiFlashPodName}
				for _, pod := range pods {
					if _, err := c.CoreV1().Pods(ns).Get(context.TODO(), pod, metav1.GetOptions{}); err != nil {
						return false, nil, fmt.Sprintf("failed to get pod %s: %v", pod, err)
					}
				}
				return true, nil, ""
			})
			framework.ExpectNoError(err, "failed to wait for failover pods to be created, last reason: %s", lastReason)

			ginkgo.By("Up down stores and waiting to become Up")
			f.ExecCommandInContainer(downTiKVPodName, "tikv", "sh", "-c", "mv /var/lib/tikv/db_down /var/lib/tikv/db")
			oa.RunKubectlOrDie("set", "image", "-n", ns, "pods", downTiFlashPodName, fmt.Sprintf("tiflash=pingcap/tiflash:%s", utilimage.TiDBLatest))
			err = utiltc.WaitForTCCondition(cli, tc.Namespace, tc.Name, time.Minute*5, time.Second*10,
				func(tc *v1alpha1.TidbCluster) (bool, error) {
					tikvIsUp, tiflashIsUp := false, false
					for _, store := range tc.Status.TiKV.Stores {
						if store.PodName == downTiKVPodName {
							tikvIsUp = store.State == v1alpha1.TiKVStateUp
							break
						}
					}
					for _, store := range tc.Status.TiFlash.Stores {
						if strings.HasPrefix(store.IP, downTiFlashPodName) {
							tiflashIsUp = store.State == v1alpha1.TiKVStateUp
							break
						}
					}
					return tikvIsUp && tiflashIsUp, nil
				})
			framework.ExpectNoError(err, "failed to waiting for tikv-1 store to be in Up state")

			ginkgo.By("Update TiKV wrong failover.recoverByUID")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.TiKV.RecoverFailover = false
				tc.Spec.TiFlash.RecoverFailover = false
				tc.Spec.TiKV.Failover = &v1alpha1.Failover{RecoverByUID: "11111111-1111-1111-1111-111111111111"}
				tc.Spec.TiFlash.Failover = &v1alpha1.Failover{RecoverByUID: "11111111-1111-1111-1111-111111111111"}
				return nil
			})
			framework.ExpectNoError(err, "failed to update TiKV wrong failover.recoverByUID")

			ginkgo.By("Waiting for failureStore empty timeout")
			err = utiltc.WaitForTCCondition(cli, tc.Namespace, tc.Name, time.Minute*3, time.Second*10,
				func(tc *v1alpha1.TidbCluster) (bool, error) {
					if len(tc.Status.TiKV.FailureStores) == 0 {
						return true, nil
					}
					if len(tc.Status.TiFlash.FailureStores) == 0 {
						return true, nil
					}
					return false, nil
				})
			framework.ExpectEqual(err, wait.ErrWaitTimeout)

			ginkgo.By("Update spec.recoverFailover to be true")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.TiKV.RecoverFailover = true
				tc.Spec.TiFlash.RecoverFailover = true
				// even still wrong recoverByUID
				return nil
			})
			framework.ExpectNoError(err, "failed to update spec.recoverFailover")

			ginkgo.By("Waiting for failureStore to be empty after recovery")
			err = utiltc.WaitForTCCondition(cli, tc.Namespace, tc.Name, time.Minute*5, time.Second*10,
				func(tc *v1alpha1.TidbCluster) (bool, error) {
					tikvFailureStoreIsEmpty := len(tc.Status.TiKV.FailureStores) == 0 && tc.Status.TiKV.FailoverUID == ""
					tiflasFailureStoreIsEmpty := len(tc.Status.TiFlash.FailureStores) == 0 && tc.Status.TiFlash.FailoverUID == ""
					return tikvFailureStoreIsEmpty && tiflasFailureStoreIsEmpty, nil
				})
			framework.ExpectNoError(err, "failed to wait for failureStore to be empty after recovery")

			ginkgo.By("Waiting for failover pods to be deleted after recovery")
			err, lastReason = utile2e.PollImmediateWithReason(time.Second*10, time.Minute*5, func() (bool, error, string) {
				pods := []string{failoverTiKVPodName, failoverTiFlashPodName}

				for _, pod := range pods {
					_, err := c.CoreV1().Pods(ns).Get(context.TODO(), pod, metav1.GetOptions{})
					if err == nil {
						return false, nil, fmt.Sprintf("pod %s still exists", pod)
					}
					if err != nil && !apierrors.IsNotFound(err) {
						return false, nil, fmt.Sprintf("failed to get pod %s: %v", pod, err)
					}
				}

				return true, nil, ""
			})
			framework.ExpectNoError(err, "failed to wait for failover pods to be deleted after recovery, last reason: %s", lastReason)
		})
	})

	ginkgo.Context("[Feature: TLS]", func() {

		type subcase struct {
			name   string
			skipCA bool
		}

		cases := []subcase{
			{
				name:   "should enable TLS for MySQL Client and between TiDB components",
				skipCA: false,
			},
			{
				name:   "should work normally when skipCA configured and ca.crt not present in tidb-client-secret",
				skipCA: true,
			},
		}

		for _, sc := range cases {
			ginkgo.It(sc.name, func() {
				tcName := "tls"

				ginkgo.By("Installing tidb CA certificate")
				err := InstallTiDBIssuer(ns, tcName)
				framework.ExpectNoError(err, "failed to install CA certificate")

				ginkgo.By("Installing tidb server and client certificate")
				err = InstallTiDBCertificates(ns, tcName)
				framework.ExpectNoError(err, "failed to install tidb server and client certificate")

				ginkgo.By("Installing tidbInitializer client certificate")
				err = installTiDBInitializerCertificates(ns, tcName)
				framework.ExpectNoError(err, "failed to install tidbInitializer client certificate")

				ginkgo.By("Installing dashboard client certificate")
				err = installPDDashboardCertificates(ns, tcName)
				framework.ExpectNoError(err, "failed to install dashboard client certificate")

				ginkgo.By("Installing tidb components certificates")
				err = InstallTiDBComponentsCertificates(ns, tcName)
				framework.ExpectNoError(err, "failed to install tidb components certificates")

				if sc.skipCA {
					ginkgo.By("Removing ca.crt in tidb-client-secret, dashboard and tidbInitializer client certificate")
					err = removeCACertFromSecret(genericCli, ns, fmt.Sprintf("%s-tidb-client-secret", tcName))
					framework.ExpectNoError(err, "failed to update tidb-client-secret with ca.crt deleted")
					err = removeCACertFromSecret(genericCli, ns, fmt.Sprintf("%s-dashboard-tls", tcName))
					framework.ExpectNoError(err, "failed to update dashboard-tls secret with ca.crt deleted")
					err = removeCACertFromSecret(genericCli, ns, fmt.Sprintf("%s-initializer-tls", tcName))
					framework.ExpectNoError(err, "failed to update initializer-tls secret with ca.crt deleted")
				}

				ginkgo.By("Creating tidb cluster with TLS enabled")
				dashTLSName := fmt.Sprintf("%s-dashboard-tls", tcName)
				tc := fixture.GetTidbCluster(ns, tcName, utilimage.TiDBLatestPrev)
				tc = fixture.AddTiFlashForTidbCluster(tc)
				tc = fixture.AddTiCDCForTidbCluster(tc)
				tc.Spec.PD.Replicas = 3
				tc.Spec.PD.TLSClientSecretName = &dashTLSName
				tc.Spec.TiKV.Replicas = 3
				tc.Spec.TiDB.Replicas = 2
				tc.Spec.TiFlash.Replicas = 1
				tc.Spec.TiCDC.Replicas = 1
				tc.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
				tc.Spec.TiDB.TLSClient.SkipInternalClientCA = sc.skipCA
				tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
				tc.Spec.Pump = &v1alpha1.PumpSpec{
					Replicas:             1,
					BaseImage:            "pingcap/tidb-binlog",
					ResourceRequirements: fixture.WithStorage(fixture.BurstableSmall, "1Gi"),
					Config: tcconfig.New(map[string]interface{}{
						"addr": "0.0.0.0:8250",
						"storage": map[string]interface{}{
							"stop-write-at-available-space": 0,
						},
					}),
				}

				err = genericCli.Create(context.TODO(), tc)
				framework.ExpectNoError(err, "failed to create TidbCluster: %q", tc.Name)
				err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
				framework.ExpectNoError(err, "wait for TidbCluster ready timeout: %q", tc.Name)

				ginkgo.By("Ensure configs of all components are not changed")
				newTC, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Get(context.TODO(), tc.Name, metav1.GetOptions{})
				tc.Spec.TiDB.Config.Set("log.file.max-backups", int64(3)) // add default configs that ard added by operator
				framework.ExpectNoError(err, "failed to get TidbCluster: %s", tc.Name)
				framework.ExpectEqual(newTC.Spec.PD.Config, tc.Spec.PD.Config, "pd config isn't equal of TidbCluster: %s", tc.Name)
				framework.ExpectEqual(newTC.Spec.TiKV.Config, tc.Spec.TiKV.Config, "tikv config isn't equal of TidbCluster: %s", tc.Name)
				framework.ExpectEqual(newTC.Spec.TiDB.Config, tc.Spec.TiDB.Config, "tidb config isn't equal of TidbCluster: %s", tc.Name)
				framework.ExpectEqual(newTC.Spec.Pump.Config, tc.Spec.Pump.Config, "pump config isn't equal of TidbCluster: %s", tc.Name)
				framework.ExpectEqual(newTC.Spec.TiCDC.Config, tc.Spec.TiCDC.Config, "ticdc config isn't equal of TidbCluster: %s", tc.Name)
				framework.ExpectEqual(newTC.Spec.TiFlash.Config, tc.Spec.TiFlash.Config, "tiflash config isn't equal of TidbCluster: %s", tc.Name)

				ginkgo.By("Ensure Dashboard use custom secret")
				foundSecretName := false
				pdSts, err := stsGetter.StatefulSets(ns).Get(context.TODO(), controller.PDMemberName(tcName), metav1.GetOptions{})
				framework.ExpectNoError(err, "failed to get statefulsets for pd: %q", tc.Name)
				for _, vol := range pdSts.Spec.Template.Spec.Volumes {
					if vol.Name == "tidb-client-tls" {
						foundSecretName = true
						framework.ExpectEqual(vol.Secret.SecretName, dashTLSName)
					}
				}
				framework.ExpectEqual(foundSecretName, true)

				ginkgo.By("Creating tidb initializer")
				passwd := "admin"
				initName := fmt.Sprintf("%s-initializer", tcName)
				initPassWDName := fmt.Sprintf("%s-initializer-passwd", tcName)
				initTLSName := fmt.Sprintf("%s-initializer-tls", tcName)
				initSecret := fixture.GetInitializerSecret(tc, initPassWDName, passwd)
				_, err = c.CoreV1().Secrets(ns).Create(context.TODO(), initSecret, metav1.CreateOptions{})
				framework.ExpectNoError(err, "failed to create secret for TidbInitializer: %v", initSecret)

				ti := fixture.GetTidbInitializer(ns, tcName, initName, initPassWDName, initTLSName)
				err = genericCli.Create(context.TODO(), ti)
				framework.ExpectNoError(err, "failed to create TidbInitializer: %v", ti)

				source := &tests.DrainerSourceConfig{
					Namespace:      ns,
					ClusterName:    tcName,
					OperatorTag:    cfg.OperatorTag,
					ClusterVersion: utilimage.TiDBLatest,
				}
				targetTcName := "tls-target"
				targetTc := fixture.GetTidbCluster(ns, targetTcName, utilimage.TiDBLatest)
				targetTc.Spec.PD.Replicas = 1
				targetTc.Spec.TiKV.Replicas = 1
				targetTc.Spec.TiDB.Replicas = 1
				err = genericCli.Create(context.TODO(), targetTc)
				framework.ExpectNoError(err, "failed to create TidbCluster: %v", targetTc)
				err = oa.WaitForTidbClusterReady(targetTc, 30*time.Minute, 5*time.Second)
				framework.ExpectNoError(err, "wait for TidbCluster timeout: %v", targetTc)

				drainerConfig := &tests.DrainerConfig{
					// Note: DrainerName muse be tcName
					// oa.DeployDrainer will use DrainerName as release name to run "helm install..."
					// in InstallTiDBCertificates, we use `tidbComponentsCertificatesTmpl` to render the certs:
					// ...
					// dnsNames:
					// - "*.{{ .ClusterName }}-{{ .ClusterName }}-drainer"
					// - "*.{{ .ClusterName }}-{{ .ClusterName }}-drainer.{{ .Namespace }}"
					// - "*.{{ .ClusterName }}-{{ .ClusterName }}-drainer.{{ .Namespace }}.svc"
					// ...
					// refer to the 'drainer' part in https://docs.pingcap.com/tidb-in-kubernetes/dev/enable-tls-between-components
					DrainerName:       tcName,
					OperatorTag:       cfg.OperatorTag,
					SourceClusterName: tcName,
					Namespace:         ns,
					DbType:            tests.DbTypeTiDB,
					Host:              fmt.Sprintf("%s-tidb.%s.svc.cluster.local", targetTcName, ns),
					Port:              strconv.Itoa(int(targetTc.Spec.TiDB.GetServicePort())),
					TLSCluster:        true,
					User:              "root",
					Password:          "",
				}

				ginkgo.By("Deploying tidb drainer")
				err = oa.DeployDrainer(drainerConfig, source)
				framework.ExpectNoError(err, "failed to deploy drainer: %v", drainerConfig)
				err = oa.CheckDrainer(drainerConfig, source)
				framework.ExpectNoError(err, "failed to check drainer: %v", drainerConfig)

				ginkgo.By("Inserting data into source db")
				err = wait.PollImmediate(time.Second*5, time.Minute*5, insertIntoDataToSourceDB(fw, c, ns, tcName, passwd, true))
				framework.ExpectNoError(err, "insert data into source db timeout")

				ginkgo.By("Checking tidb-binlog works as expected")
				err = wait.PollImmediate(time.Second*5, time.Minute*5, dataInClusterIsCorrect(fw, c, ns, targetTcName, "", false))
				framework.ExpectNoError(err, "check data correct timeout")

				ginkgo.By("Connecting to tidb server to verify the connection is TLS enabled")
				err = wait.PollImmediate(time.Second*5, time.Minute*5, tidbIsTLSEnabled(fw, c, ns, tcName, passwd))
				framework.ExpectNoError(err, "connect to TLS tidb timeout")

				ginkgo.By("Scaling out tidb cluster")
				err = controller.GuaranteedUpdate(genericCli, tc, func() error {
					tc.Spec.PD.Replicas = 5
					tc.Spec.TiKV.Replicas = 5
					tc.Spec.TiDB.Replicas = 3
					tc.Spec.Pump.Replicas++
					tc.Spec.TiFlash.Replicas = 3
					tc.Spec.TiCDC.Replicas = 3
					return nil
				})
				framework.ExpectNoError(err, "failed to update TidbCluster: %q", tc.Name)
				err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
				framework.ExpectNoError(err, "wait for TidbCluster ready timeout: %q", tc.Name)

				ginkgo.By("Scaling in tidb cluster")
				err = controller.GuaranteedUpdate(genericCli, tc, func() error {
					tc.Spec.PD.Replicas = 3
					tc.Spec.TiKV.Replicas = 3
					tc.Spec.TiDB.Replicas = 2
					tc.Spec.Pump.Replicas--
					tc.Spec.TiFlash.Replicas = 1
					tc.Spec.TiCDC.Replicas = 1
					return nil
				})
				framework.ExpectNoError(err, "failed to update TidbCluster: %q", tc.Name)
				err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
				framework.ExpectNoError(err, "wait for TidbCluster ready timeout: %q", tc.Name)

				ginkgo.By("Upgrading tidb cluster")
				err = controller.GuaranteedUpdate(genericCli, tc, func() error {
					tc.Spec.Version = utilimage.TiDBLatest
					return nil
				})
				framework.ExpectNoError(err, "failed to update TidbCluster: %q", tc.Name)
				err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
				framework.ExpectNoError(err, "wait for TidbCluster ready timeout: %q", tc.Name)

				ginkgo.By("Check custom labels and annotations for TidbInitializer")
				checkInitializerCustomLabelAndAnn(ti, c)
			})
		}

		ginkgo.It("should enable TLS for MySQL Client and between Heterogeneous TiDB components", func() {
			tcName := "origintls"
			heterogeneousTcName := "heterogeneoustls"

			ginkgo.By("Installing tidb CA certificate")
			err := InstallTiDBIssuer(ns, tcName)
			framework.ExpectNoError(err, "failed to generate tidb issuer template")

			ginkgo.By("Installing tidb server and client certificate")
			err = InstallTiDBCertificates(ns, tcName)
			framework.ExpectNoError(err, "failed to install tidb server and client certificate")

			ginkgo.By("Installing heterogeneous tidb server and client certificate")
			err = installHeterogeneousTiDBCertificates(ns, heterogeneousTcName, tcName)
			framework.ExpectNoError(err, "failed to install heterogeneous tidb server and client certificate")

			ginkgo.By("Installing separate tidbInitializer client certificate")
			err = installTiDBInitializerCertificates(ns, tcName)
			framework.ExpectNoError(err, "failed to install separate tidbInitializer client certificate")

			ginkgo.By("Installing separate dashboard client certificate")
			err = installPDDashboardCertificates(ns, tcName)
			framework.ExpectNoError(err, "failed to install separate dashboard client certificate")

			ginkgo.By("Installing tidb components certificates")
			err = InstallTiDBComponentsCertificates(ns, tcName)
			framework.ExpectNoError(err, "failed to install tidb components certificates")
			err = installHeterogeneousTiDBComponentsCertificates(ns, heterogeneousTcName, tcName)
			framework.ExpectNoError(err, "failed to install heterogeneous tidb components certificates")

			ginkgo.By("Creating tidb cluster")
			dashTLSName := fmt.Sprintf("%s-dashboard-tls", tcName)
			tc := fixture.GetTidbCluster(ns, tcName, utilimage.TiDBLatest)
			tc.Spec.PD.Replicas = 1
			tc.Spec.PD.TLSClientSecretName = &dashTLSName
			tc.Spec.TiKV.Replicas = 1
			tc.Spec.TiDB.Replicas = 1
			tc.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
			tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			tc.Spec.Pump = &v1alpha1.PumpSpec{
				Replicas:             1,
				BaseImage:            "pingcap/tidb-binlog",
				ResourceRequirements: fixture.WithStorage(fixture.BurstableSmall, "1Gi"),
				Config: tcconfig.New(map[string]interface{}{
					"addr": "0.0.0.0:8250",
					"storage": map[string]interface{}{
						"stop-write-at-available-space": 0,
					},
				}),
			}
			err = genericCli.Create(context.TODO(), tc)
			framework.ExpectNoError(err, "failed to create TidbCluster: %q", tc.Name)
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "wait for TidbCluster ready timeout: %q", tc.Name)

			ginkgo.By("Creating heterogeneous tidb cluster")
			heterogeneousTc := fixture.GetTidbCluster(ns, heterogeneousTcName, utilimage.TiDBLatest)
			heterogeneousTc = fixture.AddTiFlashForTidbCluster(heterogeneousTc)
			heterogeneousTc.Spec.PD = nil
			heterogeneousTc.Spec.TiKV.Replicas = 1
			heterogeneousTc.Spec.TiDB.Replicas = 1
			heterogeneousTc.Spec.Cluster = &v1alpha1.TidbClusterRef{
				Name: tcName,
			}

			heterogeneousTc.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
			heterogeneousTc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
			err = genericCli.Create(context.TODO(), heterogeneousTc)
			framework.ExpectNoError(err, "failed to create heterogeneous TidbCluster: %v", heterogeneousTc)

			ginkgo.By("Waiting heterogeneous tls tidb cluster ready")
			err = oa.WaitForTidbClusterReady(heterogeneousTc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "wait for TidbCluster ready timeout: %q", tc.Name)

			ginkgo.By("Checking heterogeneous tls tidb cluster status")
			err = wait.PollImmediate(5*time.Second, 10*time.Minute, func() (bool, error) {
				var err error
				if _, err = cli.PingcapV1alpha1().TidbClusters(ns).Get(context.TODO(), heterogeneousTc.Name, metav1.GetOptions{}); err != nil {
					log.Logf("failed to get tidbcluster: %s/%s, %v", ns, heterogeneousTc.Name, err)
					return false, nil
				}
				log.Logf("start check heterogeneous cluster storeInfo: %s/%s", ns, heterogeneousTc.Name)
				secretLister = tests.GetSecretListerWithCacheSynced(c, 5*time.Second)
				pdClient, cancel, err := proxiedpdclient.NewProxiedPDClient(secretLister, fw, ns, tcName, true)
				framework.ExpectNoError(err, "create pdClient error")
				defer cancel()
				storeInfo, err := pdClient.GetStores()
				if err != nil {
					log.Logf("failed to get stores, %v", err)
				}
				if storeInfo.Count != 3 {
					log.Logf("failed to check stores (current: %d)", storeInfo.Count)
					return false, nil
				}
				log.Logf("check heterogeneous tc successfully")
				return true, nil
			})
			framework.ExpectNoError(err, "check heterogeneous TidbCluster timeout: %v", heterogeneousTc)

			ginkgo.By("Ensure Dashboard use custom secret")
			foundSecretName := false
			pdSts, err := stsGetter.StatefulSets(ns).Get(context.TODO(), controller.PDMemberName(tcName), metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get statefulsets for pd")
			for _, vol := range pdSts.Spec.Template.Spec.Volumes {
				if vol.Name == "tidb-client-tls" {
					foundSecretName = true
					framework.ExpectEqual(vol.Secret.SecretName, dashTLSName)
				}
			}
			framework.ExpectEqual(foundSecretName, true)

			ginkgo.By("Creating tidb initializer")
			passwd := "admin"
			initName := fmt.Sprintf("%s-initializer", tcName)
			initPassWDName := fmt.Sprintf("%s-initializer-passwd", tcName)
			initTLSName := fmt.Sprintf("%s-initializer-tls", tcName)
			initSecret := fixture.GetInitializerSecret(tc, initPassWDName, passwd)
			_, err = c.CoreV1().Secrets(ns).Create(context.TODO(), initSecret, metav1.CreateOptions{})
			framework.ExpectNoError(err, "failed to create secret for TidbInitializer: %v", initSecret)

			ti := fixture.GetTidbInitializer(ns, tcName, initName, initPassWDName, initTLSName)
			err = genericCli.Create(context.TODO(), ti)
			framework.ExpectNoError(err, "failed to create TidbInitializer: %v", ti)

			source := &tests.DrainerSourceConfig{
				Namespace:      ns,
				ClusterName:    tcName,
				OperatorTag:    cfg.OperatorTag,
				ClusterVersion: utilimage.TiDBLatest,
			}
			targetTcName := "tls-target"
			targetTc := fixture.GetTidbCluster(ns, targetTcName, utilimage.TiDBLatest)
			targetTc.Spec.PD.Replicas = 1
			targetTc.Spec.TiKV.Replicas = 1
			targetTc.Spec.TiDB.Replicas = 1
			err = genericCli.Create(context.TODO(), targetTc)
			framework.ExpectNoError(err, "failed to create TidbCluster: %v", targetTc)
			err = oa.WaitForTidbClusterReady(targetTc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "wait for TidbCluster ready timeout: %q", tc.Name)

			drainerConfig := &tests.DrainerConfig{
				DrainerName:       tcName,
				OperatorTag:       cfg.OperatorTag,
				SourceClusterName: tcName,
				Namespace:         ns,
				DbType:            tests.DbTypeTiDB,
				Host:              fmt.Sprintf("%s-tidb.%s.svc.cluster.local", targetTcName, ns),
				Port:              strconv.Itoa(int(targetTc.Spec.TiDB.GetServicePort())),
				TLSCluster:        true,
				User:              "root",
				Password:          "",
			}

			ginkgo.By("Deploying tidb drainer")
			err = oa.DeployDrainer(drainerConfig, source)
			framework.ExpectNoError(err, "failed to deploy drainer: %v", drainerConfig)
			err = oa.CheckDrainer(drainerConfig, source)
			framework.ExpectNoError(err, "failed to check drainer: %v", drainerConfig)

			ginkgo.By("Inserting data into source db")
			err = wait.PollImmediate(time.Second*5, time.Minute*5, insertIntoDataToSourceDB(fw, c, ns, tcName, passwd, true))
			framework.ExpectNoError(err, "insert data into source db timeout")

			ginkgo.By("Checking tidb-binlog works as expected")
			err = wait.PollImmediate(time.Second*5, time.Minute*5, dataInClusterIsCorrect(fw, c, ns, targetTcName, "", false))
			framework.ExpectNoError(err, "check data correct timeout")

			ginkgo.By("Connecting to tidb server to verify the connection is TLS enabled")
			err = wait.PollImmediate(time.Second*5, time.Minute*5, tidbIsTLSEnabled(fw, c, ns, tcName, passwd))
			framework.ExpectNoError(err, "connect to TLS tidb timeout")
		})
	})

	// TODO: move into TiDB specific group
	ginkgo.It("should ensure changing TiDB service annotations won't change TiDB service type NodePort", func() {
		ginkgo.By("Deploy initial tc")
		// Create TidbCluster with NodePort to check whether node port would change
		nodeTc := fixture.GetTidbCluster(ns, "nodeport", utilimage.TiDBLatest)
		nodeTc.Spec.PD.Replicas = 1
		nodeTc.Spec.TiKV.Replicas = 1
		nodeTc.Spec.TiDB.Replicas = 1
		nodeTc.Spec.TiDB.Service = &v1alpha1.TiDBServiceSpec{
			ServiceSpec: v1alpha1.ServiceSpec{
				Type: corev1.ServiceTypeNodePort,
			},
		}
		err := genericCli.Create(context.TODO(), nodeTc)
		framework.ExpectNoError(err, "Expected TiDB cluster created")
		err = oa.WaitForTidbClusterReady(nodeTc, 30*time.Minute, 5*time.Second)
		framework.ExpectNoError(err, "Expected TiDB cluster ready")

		// expect tidb service type is Nodeport
		// NOTE: should we poll here or just check for once? because tc is ready
		var s *corev1.Service
		err = wait.Poll(5*time.Second, 1*time.Minute, func() (done bool, err error) {
			s, err = c.CoreV1().Services(ns).Get(context.TODO(), "nodeport-tidb", metav1.GetOptions{})
			if err != nil {
				log.Logf(err.Error())
				return false, nil
			}
			if s.Spec.Type != corev1.ServiceTypeNodePort {
				return false, fmt.Errorf("nodePort tidbcluster tidb service type isn't NodePort")
			}
			return true, nil
		})
		framework.ExpectNoError(err, "wait for tidb service sync timeout")
		oldPorts := s.Spec.Ports

		ensureSvcNodePortUnchangedFor1Min := func() {
			err := wait.Poll(5*time.Second, 1*time.Minute, func() (done bool, err error) {
				s, err := c.CoreV1().Services(ns).Get(context.TODO(), "nodeport-tidb", metav1.GetOptions{})
				if err != nil {
					return false, err
				}
				if s.Spec.Type != corev1.ServiceTypeNodePort {
					return false, err
				}
				for _, dport := range s.Spec.Ports {
					for _, eport := range oldPorts {
						if dport.Port == eport.Port && dport.NodePort != eport.NodePort {
							return false, fmt.Errorf("nodePort tidbcluster tidb service NodePort changed")
						}
					}
				}
				return false, nil
			})
			framework.ExpectEqual(err, wait.ErrWaitTimeout, "service NodePort should not change in 1 minute")
		}

		ginkgo.By("Make sure tidb service NodePort doesn't changed")
		ensureSvcNodePortUnchangedFor1Min()

		ginkgo.By("Change tidb service annotation")
		nodeTc, err = cli.PingcapV1alpha1().TidbClusters(ns).Get(context.TODO(), "nodeport", metav1.GetOptions{})
		framework.ExpectNoError(err, "failed to get TidbCluster")
		err = controller.GuaranteedUpdate(genericCli, nodeTc, func() error {
			nodeTc.Spec.TiDB.Service.Annotations = map[string]string{
				"foo": "bar",
			}
			return nil
		})
		framework.ExpectNoError(err, "failed to change tidb service annotation")

		// check whether the tidb svc have updated
		err = wait.Poll(5*time.Second, 1*time.Minute, func() (done bool, err error) {
			s, err := c.CoreV1().Services(ns).Get(context.TODO(), "nodeport-tidb", metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if s.Annotations == nil {
				return false, nil
			}
			v, ok := s.Annotations["foo"]
			if !ok {
				return false, nil
			}
			if v != "bar" {
				return false, fmt.Errorf("tidb svc annotation foo not equal bar")
			}
			return true, nil
		})
		framework.ExpectNoError(err, "failed wait for tidb service sync")

		ginkgo.By("Make sure tidb service NodePort doesn't changed")
		// check whether NodePort have changed for 1 min
		ensureSvcNodePortUnchangedFor1Min()
		log.Logf("tidbcluster tidb service NodePort haven't changed after update")
	})

	ginkgo.Context("[Feature: Heterogeneous Cluster]", func() {
		ginkgo.It("should join heterogeneous cluster into an existing cluster", func() {
			// Create TidbCluster with NodePort to check whether node port would change
			ginkgo.By("Deploy origin tc")
			originTc := fixture.GetTidbCluster(ns, "origin", utilimage.TiDBLatest)
			originTc.Spec.PD.Replicas = 1
			originTc.Spec.TiKV.Replicas = 1
			originTc.Spec.TiDB.Replicas = 1
			err := genericCli.Create(context.TODO(), originTc)
			framework.ExpectNoError(err, "Expected TiDB cluster created")
			err = oa.WaitForTidbClusterReady(originTc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "Expected TiDB cluster ready")

			ginkgo.By("Deploy heterogeneous tc")
			heterogeneousTc := fixture.GetTidbCluster(ns, "heterogeneous", utilimage.TiDBLatest)
			heterogeneousTc = fixture.AddTiFlashForTidbCluster(heterogeneousTc)
			heterogeneousTc = fixture.AddPumpForTidbCluster(heterogeneousTc)
			heterogeneousTc = fixture.AddTiCDCForTidbCluster(heterogeneousTc)

			heterogeneousTc.Spec.PD = nil
			heterogeneousTc.Spec.TiKV.Replicas = 1
			heterogeneousTc.Spec.TiDB.Replicas = 1
			heterogeneousTc.Spec.TiFlash.Replicas = 1
			heterogeneousTc.Spec.Pump.Replicas = 1
			heterogeneousTc.Spec.TiCDC.Replicas = 1
			heterogeneousTc.Spec.Cluster = &v1alpha1.TidbClusterRef{
				Name: originTc.Name,
			}

			err = genericCli.Create(context.TODO(), heterogeneousTc)
			framework.ExpectNoError(err, "Expected Heterogeneous TiDB cluster created")
			err = oa.WaitForTidbClusterReady(heterogeneousTc, 30*time.Minute, 15*time.Second)
			framework.ExpectNoError(err, "Expected Heterogeneous TiDB cluster ready")

			ginkgo.By("Wait for heterogeneous tc to join origin tc")
			err = wait.PollImmediate(5*time.Second, 10*time.Minute, func() (bool, error) {
				var err error
				if _, err = cli.PingcapV1alpha1().TidbClusters(ns).Get(context.TODO(), heterogeneousTc.Name, metav1.GetOptions{}); err != nil {
					log.Logf("failed to get tidbcluster: %s/%s, %v", ns, heterogeneousTc.Name, err)
					return false, nil
				}
				log.Logf("start check heterogeneous cluster storeInfo: %s/%s", ns, heterogeneousTc.Name)
				pdClient, cancel, err := proxiedpdclient.NewProxiedPDClient(secretLister, fw, ns, originTc.Name, false)
				framework.ExpectNoError(err, "create pdClient error")
				defer cancel()
				storeInfo, err := pdClient.GetStores()
				if err != nil {
					log.Logf("failed to get stores, %v", err)
				}
				if storeInfo.Count != 3 {
					log.Logf("failed to check stores (current: %d)", storeInfo.Count)
					return false, nil
				}
				log.Logf("check heterogeneous tc successfully")
				return true, nil
			})
			framework.ExpectNoError(err, "failed to wait heterogeneous tc to join origin tc")
		})

		ginkgo.It("should join cluster with setting cluster domain", func() {
			// Create TidbCluster with NodePort to check whether node port would change
			ginkgo.By("Deploy origin tc")
			originTc := fixture.GetTidbCluster(ns, "origin", utilimage.TiDBLatest)
			originTc.Spec.PD.Replicas = 1
			originTc.Spec.TiKV.Replicas = 1
			originTc.Spec.TiDB.Replicas = 1
			err := genericCli.Create(context.TODO(), originTc)
			framework.ExpectNoError(err, "Expected TiDB cluster created")
			err = oa.WaitForTidbClusterReady(originTc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "Expected TiDB cluster ready")

			ginkgo.By("Deploy heterogeneous tc")
			heterogeneousTc := fixture.GetTidbCluster(ns, "heterogeneous", utilimage.TiDBLatest)
			heterogeneousTc = fixture.AddTiFlashForTidbCluster(heterogeneousTc)
			heterogeneousTc = fixture.AddPumpForTidbCluster(heterogeneousTc)
			heterogeneousTc = fixture.AddTiCDCForTidbCluster(heterogeneousTc)
			heterogeneousTc.Spec.PD = nil
			heterogeneousTc.Spec.TiKV.Replicas = 1
			heterogeneousTc.Spec.TiDB.Replicas = 1
			heterogeneousTc.Spec.TiFlash.Replicas = 1
			heterogeneousTc.Spec.Pump.Replicas = 1
			heterogeneousTc.Spec.TiCDC.Replicas = 1
			heterogeneousTc.Spec.Cluster = &v1alpha1.TidbClusterRef{
				Name: originTc.Name,
			}
			heterogeneousTc.Spec.ClusterDomain = "cluster.local"

			err = genericCli.Create(context.TODO(), heterogeneousTc)
			framework.ExpectNoError(err, "Expected Heterogeneous TiDB cluster created")
			err = oa.WaitForTidbClusterReady(heterogeneousTc, 30*time.Minute, 15*time.Second)
			framework.ExpectNoError(err, "Expected Heterogeneous TiDB cluster ready")

			ginkgo.By("Wait for heterogeneous tc to join origin tc")
			err = wait.PollImmediate(5*time.Second, 10*time.Minute, func() (bool, error) {
				var err error
				if _, err = cli.PingcapV1alpha1().TidbClusters(ns).Get(context.TODO(), heterogeneousTc.Name, metav1.GetOptions{}); err != nil {
					log.Logf("failed to get tidbcluster: %s/%s, %v", ns, heterogeneousTc.Name, err)
					return false, nil
				}
				log.Logf("start check heterogeneous cluster storeInfo: %s/%s", ns, heterogeneousTc.Name)
				pdClient, cancel, err := proxiedpdclient.NewProxiedPDClient(secretLister, fw, ns, originTc.Name, false)
				framework.ExpectNoError(err, "create pdClient error")
				defer cancel()
				storeInfo, err := pdClient.GetStores()
				if err != nil {
					log.Logf("failed to get stores, %v", err)
				}
				if storeInfo.Count != 3 {
					log.Logf("failed to check stores (current: %d)", storeInfo.Count)
					return false, nil
				}
				log.Logf("check heterogeneous tc successfully")
				return true, nil
			})
			framework.ExpectNoError(err, "failed to wait heterogeneous tc to join origin tc")
		})
	})

	ginkgo.Context("[Feature: CDC]", func() {
		ginkgo.It("should sink data without extra volumes correctly", func() {
			ginkgo.By("Creating cdc cluster")
			fromTc := fixture.GetTidbCluster(ns, "cdc-source", utilimage.TiDBLatest)
			fromTc = fixture.AddTiCDCForTidbCluster(fromTc)

			fromTc.Spec.PD.Replicas = 3
			fromTc.Spec.TiKV.Replicas = 3
			fromTc.Spec.TiDB.Replicas = 2
			fromTc.Spec.TiCDC.Replicas = 3

			err := genericCli.Create(context.TODO(), fromTc)
			framework.ExpectNoError(err, "Expected TiDB cluster created")
			err = oa.WaitForTidbClusterReady(fromTc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "Expected TiDB cluster ready")

			ginkgo.By("Creating cdc-sink cluster")
			toTc := fixture.GetTidbCluster(ns, "cdc-sink", utilimage.TiDBLatest)
			toTc.Spec.PD.Replicas = 1
			toTc.Spec.TiKV.Replicas = 1
			toTc.Spec.TiDB.Replicas = 1
			err = genericCli.Create(context.TODO(), toTc)
			framework.ExpectNoError(err, "Expected TiDB cluster created")
			err = oa.WaitForTidbClusterReady(toTc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "Expected TiDB cluster ready")

			ginkgo.By("Update cdc config to use config file")
			err = controller.GuaranteedUpdate(genericCli, fromTc, func() error {
				fromTc.Spec.TiCDC.Config.Set("capture-session-ttl", 10)
				return nil
			})
			framework.ExpectNoError(err, "failed to update cdc config: %q", fromTc.Name)
			err = oa.WaitForTidbClusterReady(fromTc, 3*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", fromTc.Name)

			ginkgo.By("Check cdc configuration")
			var cdcCmName string
			err = wait.PollImmediate(time.Second*5, time.Minute*5, func() (bool, error) {
				cdcMemberName := controller.TiCDCMemberName(fromTc.Name)
				cdcSts, err := stsGetter.StatefulSets(ns).Get(context.TODO(), cdcMemberName, metav1.GetOptions{})
				if err != nil {
					return false, err
				}

				cdcCmName = mngerutils.FindConfigMapVolume(&cdcSts.Spec.Template.Spec, func(name string) bool {
					return strings.HasPrefix(name, controller.TiCDCMemberName(fromTc.Name))
				})

				if cdcCmName != "" {
					return true, nil
				}

				return false, nil
			})
			framework.ExpectNoError(err, "failed wait to update to use config file")

			cdcCm, err := c.CoreV1().ConfigMaps(ns).Get(context.TODO(), cdcCmName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get ConfigMap %s/%s", ns, cdcCm)
			log.Logf("CDC config:\n%s", cdcCm.Data["config-file"])
			gomega.Expect(cdcCm.Data["config-file"]).To(gomega.ContainSubstring("capture-session-ttl = 10"))

			ginkgo.By("Creating change feed task")
			fromTCName := fromTc.Name
			toTCName := toTc.Name
			args := []string{
				"exec",
				fmt.Sprintf("%s-0", controller.TiCDCMemberName(fromTCName)),
				"--",
				"/cdc", "cli", "changefeed", "create",
				fmt.Sprintf("--sink-uri=tidb://root:@%s:%d/", controller.TiDBMemberName(toTCName), toTc.Spec.TiDB.GetServicePort()),
				fmt.Sprintf("--pd=http://%s:2379", controller.PDMemberName(fromTCName)),
			}
			data, err := framework.RunKubectl(ns, args...)
			framework.ExpectNoError(err, "failed to create change feed task: %s, %v", string(data), err)

			ginkgo.By("Inserting data to cdc cluster")
			err = wait.PollImmediate(time.Second*5, time.Minute*5, insertIntoDataToSourceDB(fw, c, ns, fromTCName, "", false))
			framework.ExpectNoError(err, "insert data to cdc cluster timeout")

			ginkgo.By("Checking cdc works as expected")
			err = wait.PollImmediate(time.Second*5, time.Minute*5, dataInClusterIsCorrect(fw, c, ns, toTCName, "", false))
			framework.ExpectNoError(err, "check cdc timeout")

			log.Logf("CDC works as expected")
		})

		ginkgo.It("should scale in-out and mount storage volumes correctly", func() {
			ginkgo.By("Creating cdc cluster")
			fromTc := fixture.GetTidbCluster(ns, "cdc-source-advance", utilimage.TiDBLatest)
			fromTc = fixture.AddTiCDCForTidbCluster(fromTc)

			fromTc.Spec.PD.Replicas = 3
			fromTc.Spec.TiKV.Replicas = 3
			fromTc.Spec.TiDB.Replicas = 2
			fromTc.Spec.TiCDC.Replicas = 3

			fromTc.Spec.TiCDC.StorageVolumes = []v1alpha1.StorageVolume{
				{
					Name:        "sort-dir",
					StorageSize: "1Gi",
					MountPath:   "/var/lib/sort-dir",
				},
			}

			err := genericCli.Create(context.TODO(), fromTc)
			framework.ExpectNoError(err, "Expected TiDB cluster created")
			err = oa.WaitForTidbClusterReady(fromTc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "Expected TiDB cluster ready")

			ginkgo.By("Creating cdc-sink cluster")
			toTc := fixture.GetTidbCluster(ns, "cdc-sink-advance", utilimage.TiDBLatest)
			toTc.Spec.PD.Replicas = 1
			toTc.Spec.TiKV.Replicas = 1
			toTc.Spec.TiDB.Replicas = 1
			err = genericCli.Create(context.TODO(), toTc)
			framework.ExpectNoError(err, "Expected TiDB cluster created")
			err = oa.WaitForTidbClusterReady(toTc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "Expected TiDB cluster ready")

			ginkgo.By("Scale in TiCDC")
			err = controller.GuaranteedUpdate(genericCli, fromTc, func() error {
				fromTc.Spec.TiCDC.Replicas = 2
				return nil
			})
			framework.ExpectNoError(err, "failed to scale in TiCDC for TidbCluster %s/%s", ns, fromTc.Name)

			ginkgo.By("Wait for fromTC ready")
			err = oa.WaitForTidbClusterReady(fromTc, 3*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbCluster %s/%s ready after scale in TiCDC", ns, fromTc.Name)

			ginkgo.By("Check PVC annotation tidb.pingcap.com/pvc-defer-deleting")
			pvcUIDs := make(map[string]string)
			ordinal := int32(2)
			pvcSelector, err := member.GetPVCSelectorForPod(fromTc, v1alpha1.TiCDCMemberType, ordinal)
			framework.ExpectNoError(err, "failed to get PVC selector for tc %s/%s", fromTc.GetNamespace(), fromTc.GetName())
			err = wait.Poll(10*time.Second, 3*time.Minute, func() (done bool, err error) {
				pvcs, err := c.CoreV1().PersistentVolumeClaims(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: pvcSelector.String()})
				framework.ExpectNoError(err, "failed to list PVCs with selector: %v", pvcSelector)
				if len(pvcs.Items) == 0 {
					log.Logf("no PVC found")
					return false, nil
				}
				for _, pvc := range pvcs.Items {
					annotations := pvc.GetObjectMeta().GetAnnotations()
					log.Logf("pvc annotations: %+v", annotations)
					_, ok := annotations["tidb.pingcap.com/pvc-defer-deleting"]
					if !ok {
						log.Logf("PVC %s/%s does not have annotation tidb.pingcap.com/pvc-defer-deleting", pvc.GetNamespace(), pvc.GetName())
						return false, nil
					}
					pvcUIDs[pvc.Name] = string(pvc.UID)
				}
				return true, nil
			})
			framework.ExpectNoError(err, "expect PVCs of scaled in Pods to have annotation tidb.pingcap.com/pvc-defer-deleting")

			ginkgo.By("Scale out TiCDC")
			err = controller.GuaranteedUpdate(genericCli, fromTc, func() error {
				fromTc.Spec.TiCDC.Replicas = 3
				return nil
			})
			framework.ExpectNoError(err, "failed to scale out TiCDC for TidbCluster %s/%s", ns, fromTc.Name)

			ginkgo.By("Wait for fromTC ready")
			err = oa.WaitForTidbClusterReady(fromTc, 3*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbCluster %s/%s ready after scale out TiCDC", ns, fromTc.Name)

			ginkgo.By("Check PVCs are recreated for newly scaled out TiCDC")
			err = wait.Poll(10*time.Second, 5*time.Minute, func() (done bool, err error) {
				pvcs, err := c.CoreV1().PersistentVolumeClaims(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: pvcSelector.String()})
				framework.ExpectNoError(err, "failed to list PVCs with selector: %v", pvcSelector)
				if len(pvcs.Items) == 0 {
					log.Logf("no PVC found")
					return false, nil
				}
				for _, pvc := range pvcs.Items {
					annotations := pvc.GetObjectMeta().GetAnnotations()
					v, ok := annotations["tidb.pingcap.com/pvc-defer-deleting"]
					if ok {
						log.Logf("PVC %s/%s also have annotation tidb.pingcap.com/pvc-defer-deleting=%s", pvc.GetNamespace(), pvc.GetName(), v)
						return false, nil
					}
					pvcUIDString := pvcUIDs[pvc.Name]
					framework.ExpectNotEqual(string(pvc.UID), pvcUIDString)
				}
				return true, nil
			})
			framework.ExpectNoError(err, "expect PVCs of scaled out Pods to be recreated")

			ginkgo.By("Creating change feed task")
			fromTCName := fromTc.Name
			toTCName := toTc.Name
			args := []string{
				"exec",
				fmt.Sprintf("%s-0", controller.TiCDCMemberName(fromTCName)),
				"--",
				"/cdc", "cli", "changefeed", "create",
				fmt.Sprintf("--sink-uri=tidb://root:@%s:%d/", controller.TiDBMemberName(toTCName), toTc.Spec.TiDB.GetServicePort()),
				fmt.Sprintf("--pd=http://%s:2379", controller.PDMemberName(fromTCName)),
			}
			data, err := framework.RunKubectl(ns, args...)
			framework.ExpectNoError(err, "failed to create change feed task: %s, %v", string(data), err)

			ginkgo.By("Inserting data to cdc cluster")
			err = wait.PollImmediate(time.Second*5, time.Minute*5, insertIntoDataToSourceDB(fw, c, ns, fromTCName, "", false))
			framework.ExpectNoError(err, "insert data to cdc cluster timeout")

			ginkgo.By("Checking cdc works as expected")
			err = wait.PollImmediate(time.Second*5, time.Minute*5, dataInClusterIsCorrect(fw, c, ns, toTCName, "", false))
			framework.ExpectNoError(err, "check cdc timeout")

			log.Logf("CDC works as expected")
		})
	})

	ginkgo.It("TiKV should mount multiple pvc", func() {
		ginkgo.By("Deploy initial tc with addition")
		clusterName := "tidb-multiple-pvc-scale"
		tc := fixture.GetTidbCluster(ns, clusterName, utilimage.TiDBLatest)
		tc.Spec.TiKV.StorageVolumes = []v1alpha1.StorageVolume{
			{
				Name:        "wal",
				StorageSize: "2Gi",
				MountPath:   "/var/lib/wal",
			},
			{
				Name:        "titan",
				StorageSize: "2Gi",
				MountPath:   "/var/lib/titan",
			},
		}
		tc.Spec.TiDB.StorageVolumes = []v1alpha1.StorageVolume{
			{
				Name:        "log",
				StorageSize: "2Gi",
				MountPath:   "/var/log",
			},
		}
		tc.Spec.PD.StorageVolumes = []v1alpha1.StorageVolume{
			{
				Name:        "log",
				StorageSize: "2Gi",
				MountPath:   "/var/log",
			},
		}

		tc.Spec.PD.Config.Set("log.file.filename", "/var/log/tidb/tidb.log")
		tc.Spec.PD.Config.Set("log.level", "warn")
		tc.Spec.TiDB.Config.Set("log.file.max-size", 300)
		tc.Spec.TiDB.Config.Set("log.file.max-days", 1)
		tc.Spec.TiDB.Config.Set("log.file.filename", "/var/log/tidb/tidb.log")
		tc.Spec.TiKV.Config.Set("rocksdb.wal-dir", "/var/lib/wal")
		tc.Spec.TiKV.Config.Set("titan.dirname", "/var/lib/titan")
		tc.Spec.PD.Replicas = 1
		tc.Spec.TiKV.Replicas = 4
		tc.Spec.TiDB.Replicas = 2

		utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 6*time.Minute, 5*time.Second)

		ginkgo.By("Scale in multiple pvc tidb cluster")
		// Scale
		err := controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.TiKV.Replicas = 3
			tc.Spec.TiDB.Replicas = 1
			return nil
		})
		framework.ExpectNoError(err, "failed to scale in TiKV/TiDB, TidbCluster: %q", tc.Name)
		err = oa.WaitForTidbClusterReady(tc, 3*time.Minute, 5*time.Second)
		framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)

		ginkgo.By("Check PVC annotation tidb.pingcap.com/pvc-defer-deleting")
		pvcUIDs := make(map[string]string)
		ordinal := int32(1)
		pvcSelector, err := member.GetPVCSelectorForPod(tc, v1alpha1.TiDBMemberType, ordinal)
		framework.ExpectNoError(err, "failed to get PVC selector for tc %s/%s", tc.GetNamespace(), tc.GetName())
		err = wait.Poll(10*time.Second, 3*time.Minute, func() (done bool, err error) {
			pvcs, err := c.CoreV1().PersistentVolumeClaims(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: pvcSelector.String()})
			framework.ExpectNoError(err, "failed to list PVCs with selector: %v", pvcSelector)
			if len(pvcs.Items) == 0 {
				log.Logf("no PVC found")
				return false, nil
			}
			for _, pvc := range pvcs.Items {
				annotations := pvc.GetObjectMeta().GetAnnotations()
				log.Logf("pvc annotations: %+v", annotations)
				_, ok := annotations["tidb.pingcap.com/pvc-defer-deleting"]
				if !ok {
					log.Logf("PVC %s/%s does not have annotation tidb.pingcap.com/pvc-defer-deleting", pvc.GetNamespace(), pvc.GetName())
					return false, nil
				}
				pvcUIDs[pvc.Name] = string(pvc.UID)
			}
			return true, nil
		})
		framework.ExpectNoError(err, "expect PVCs of scaled in Pods to have annotation tidb.pingcap.com/pvc-defer-deleting")

		ginkgo.By("Scale out multiple pvc tidb cluster")
		// Scale
		err = controller.GuaranteedUpdate(genericCli, tc, func() error {
			tc.Spec.TiKV.Replicas = 4
			tc.Spec.TiDB.Replicas = 2
			return nil
		})
		framework.ExpectNoError(err, "failed to scale out TiKV/TiDB, TidbCluster: %q", tc.Name)
		err = oa.WaitForTidbClusterReady(tc, 3*time.Minute, 5*time.Second)
		framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)

		ginkgo.By("Check PVCs are recreated for newly scaled out TiDB")
		err = wait.Poll(10*time.Second, 3*time.Minute, func() (done bool, err error) {
			pvcs, err := c.CoreV1().PersistentVolumeClaims(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: pvcSelector.String()})
			framework.ExpectNoError(err, "failed to list PVCs with selector: %v", pvcSelector)
			if len(pvcs.Items) == 0 {
				log.Logf("no PVC found")
				return false, nil
			}
			for _, pvc := range pvcs.Items {
				annotations := pvc.GetObjectMeta().GetAnnotations()
				v, ok := annotations["tidb.pingcap.com/pvc-defer-deleting"]
				if ok {
					log.Logf("PVC %s/%s also have annotation tidb.pingcap.com/pvc-defer-deleting=%s", pvc.GetNamespace(), pvc.GetName(), v)
					return false, nil
				}
				pvcUIDString := pvcUIDs[pvc.Name]
				framework.ExpectNotEqual(string(pvc.UID), pvcUIDString)
			}
			return true, nil
		})
		framework.ExpectNoError(err, "expect PVCs of scaled out Pods to be recreated")
	})

	// test cases for tc upgrade
	ginkgo.Context("[Feature: Upgrade]", func() {
		ginkgo.It("for components version", func() {
			ginkgo.By("Deploy initial tc")
			tc := fixture.GetTidbCluster(ns, "upgrade-version", utilimage.TiDBLatestPrev)
			pvRetain := corev1.PersistentVolumeReclaimRetain
			tc.Spec.PVReclaimPolicy = &pvRetain
			tc.Spec.PD.StorageClassName = pointer.StringPtr("local-storage")
			tc.Spec.TiKV.StorageClassName = pointer.StringPtr("local-storage")
			tc.Spec.TiDB.StorageClassName = pointer.StringPtr("local-storage")
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)

			ginkgo.By("Update components version")
			componentVersion := utilimage.TiDBLatest
			err := controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.PD.Version = pointer.StringPtr(componentVersion)
				tc.Spec.TiKV.Version = pointer.StringPtr(componentVersion)
				tc.Spec.TiDB.Version = pointer.StringPtr(componentVersion)
				return nil
			})
			framework.ExpectNoError(err, "failed to update components version to %q", componentVersion)
			err = oa.WaitForTidbClusterReady(tc, 20*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbCluster %s/%s components ready", ns, tc.Name)

			ginkgo.By("Check components version")
			pdMemberName := controller.PDMemberName(tc.Name)
			pdSts, err := stsGetter.StatefulSets(ns).Get(context.TODO(), pdMemberName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get StatefulSet %s/%s", ns, pdMemberName)
			pdImage := fmt.Sprintf("pingcap/pd:%s", componentVersion)
			framework.ExpectEqual(pdSts.Spec.Template.Spec.Containers[0].Image, pdImage, "pd sts image should be %q", pdImage)

			tikvMemberName := controller.TiKVMemberName(tc.Name)
			tikvSts, err := stsGetter.StatefulSets(ns).Get(context.TODO(), tikvMemberName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get StatefulSet %s/%s", ns, tikvMemberName)
			tikvImage := fmt.Sprintf("pingcap/tikv:%s", componentVersion)
			framework.ExpectEqual(tikvSts.Spec.Template.Spec.Containers[0].Image, tikvImage, "tikv sts image should be %q", tikvImage)

			tidbMemberName := controller.TiDBMemberName(tc.Name)
			tidbSts, err := stsGetter.StatefulSets(ns).Get(context.TODO(), tidbMemberName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get StatefulSet %s/%s", ns, tidbMemberName)
			tidbImage := fmt.Sprintf("pingcap/tidb:%s", componentVersion)
			// the 0th container for tidb pod is slowlog, which runs busybox
			framework.ExpectEqual(tidbSts.Spec.Template.Spec.Containers[1].Image, tidbImage, "tidb sts image should be %q", tidbImage)
		})

		ginkgo.It("for configuration update", func() {
			ginkgo.By("Deploy initial tc")
			tc := fixture.GetTidbCluster(ns, "update-config", utilimage.TiDBLatest)
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)

			ginkgo.By("Update components configuration")
			err := controller.GuaranteedUpdate(genericCli, tc, func() error {
				pdCfg := v1alpha1.NewPDConfig()
				tikvCfg := v1alpha1.NewTiKVConfig()
				tidbCfg := v1alpha1.NewTiDBConfig()
				pdCfg.Set("lease", 3)
				tikvCfg.Set("storage.reserve-space", "0MB")
				tikvCfg.Set("status-thread-pool-size", 1)
				tidbCfg.Set("token-limit", 10000)
				tc.Spec.PD.Config = pdCfg
				tc.Spec.TiKV.Config = tikvCfg
				tc.Spec.TiDB.Config = tidbCfg
				return nil
			})
			framework.ExpectNoError(err, "failed to update components configuration")

			ginkgo.By("Wait for PD to be in UpgradePhase")
			utiltc.MustWaitForComponentPhase(cli, tc, v1alpha1.PDMemberType, v1alpha1.UpgradePhase, 3*time.Minute, time.Second*10)
			log.Logf("PD is in UpgradePhase")

			ginkgo.By("Wait for tc ready")
			err = oa.WaitForTidbClusterReady(tc, 10*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbCluster %s/%s components ready", ns, tc.Name)

			ginkgo.By("Check PD configuration")
			pdMemberName := controller.PDMemberName(tc.Name)
			pdSts, err := stsGetter.StatefulSets(ns).Get(context.TODO(), pdMemberName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get StatefulSet %s/%s", ns, pdMemberName)
			pdCmName := mngerutils.FindConfigMapVolume(&pdSts.Spec.Template.Spec, func(name string) bool {
				return strings.HasPrefix(name, controller.PDMemberName(tc.Name))
			})
			pdCm, err := c.CoreV1().ConfigMaps(ns).Get(context.TODO(), pdCmName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get ConfigMap %s/%s", ns, pdCm)
			log.Logf("PD config:\n%s", pdCm.Data["config-file"])
			gomega.Expect(pdCm.Data["config-file"]).To(gomega.ContainSubstring("lease = 3"))

			ginkgo.By("Wait for TiKV to be in UpgradePhase")
			utiltc.MustWaitForComponentPhase(cli, tc, v1alpha1.TiKVMemberType, v1alpha1.UpgradePhase, 3*time.Minute, time.Second*10)
			log.Logf("TiKV is in UpgradePhase")

			ginkgo.By("Wait for tc ready")
			err = oa.WaitForTidbClusterReady(tc, 10*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbCluster %s/%s components ready", ns, tc.Name)

			ginkgo.By("Check TiKV configuration")
			tikvMemberName := controller.TiKVMemberName(tc.Name)
			tikvSts, err := stsGetter.StatefulSets(ns).Get(context.TODO(), tikvMemberName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get StatefulSet %s/%s", ns, tikvMemberName)
			tikvCmName := mngerutils.FindConfigMapVolume(&tikvSts.Spec.Template.Spec, func(name string) bool {
				return strings.HasPrefix(name, controller.TiKVMemberName(tc.Name))
			})
			tikvCm, err := c.CoreV1().ConfigMaps(ns).Get(context.TODO(), tikvCmName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get ConfigMap %s/%s", ns, tikvCmName)
			log.Logf("TiKV config:\n%s", tikvCm.Data["config-file"])
			gomega.Expect(tikvCm.Data["config-file"]).To(gomega.ContainSubstring("status-thread-pool-size = 1"))

			ginkgo.By("Wait for TiDB to be in UpgradePhase")
			utiltc.MustWaitForComponentPhase(cli, tc, v1alpha1.TiDBMemberType, v1alpha1.UpgradePhase, 3*time.Minute, time.Second*10)
			log.Logf("TiDB is in UpgradePhase")

			ginkgo.By("Wait for tc ready")
			err = oa.WaitForTidbClusterReady(tc, 10*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbCluster %s/%s components ready", ns, tc.Name)

			ginkgo.By("Check TiDB configuration")
			tidbMemberName := controller.TiDBMemberName(tc.Name)
			tidbSts, err := stsGetter.StatefulSets(ns).Get(context.TODO(), tidbMemberName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get StatefulSet %s/%s", ns, tidbMemberName)
			tidbCmName := mngerutils.FindConfigMapVolume(&tidbSts.Spec.Template.Spec, func(name string) bool {
				return strings.HasPrefix(name, controller.TiDBMemberName(tc.Name))
			})
			tidbCm, err := c.CoreV1().ConfigMaps(ns).Get(context.TODO(), tidbCmName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get ConfigMap %s/%s", ns, tidbCmName)
			log.Logf("TiDB config:\n%s", tidbCm.Data["config-file"])
			gomega.Expect(tidbCm.Data["config-file"]).To(gomega.ContainSubstring("token-limit = 10000"))
		})

		type upgradeCase struct {
			oldVersion              string
			newVersion              string
			configureOldTiDBCluster func(tc *v1alpha1.TidbCluster)
			configureNewTiDBCluster func(tc *v1alpha1.TidbCluster)
		}
		upgradeTest := func(ucase upgradeCase) {
			ginkgo.By("Deploy initial tc")
			tcName := fmt.Sprintf("upgrade-%s-to-%s", strings.ReplaceAll(ucase.oldVersion, ".", "x"), strings.ReplaceAll(ucase.newVersion, ".", "x"))
			tc := fixture.GetTidbCluster(ns, tcName, ucase.oldVersion)
			tc = fixture.AddTiFlashForTidbCluster(tc)
			tc = fixture.AddTiCDCForTidbCluster(tc)
			tc = fixture.AddPumpForTidbCluster(tc)
			if ucase.configureOldTiDBCluster != nil {
				ucase.configureOldTiDBCluster(tc)
			}
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 15*time.Minute, 10*time.Second)

			ginkgo.By("Update tc version")
			err := controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.Version = ucase.newVersion
				if ucase.configureNewTiDBCluster != nil {
					ucase.configureNewTiDBCluster(tc)
				}
				return nil
			})
			framework.ExpectNoError(err, "failed to update tc version to %q", ucase.newVersion)
			err = oa.WaitForTidbClusterReady(tc, 15*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbCluster %s/%s components ready", ns, tc.Name)

			ginkgo.By("Check components version")
			componentVersion := ucase.newVersion
			pdMemberName := controller.PDMemberName(tc.Name)
			pdSts, err := stsGetter.StatefulSets(ns).Get(context.TODO(), pdMemberName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get StatefulSet %s/%s", ns, pdMemberName)
			pdImage := fmt.Sprintf("pingcap/pd:%s", componentVersion)
			framework.ExpectEqual(pdSts.Spec.Template.Spec.Containers[0].Image, pdImage, "pd sts image should be %q", pdImage)

			tikvMemberName := controller.TiKVMemberName(tc.Name)
			tikvSts, err := stsGetter.StatefulSets(ns).Get(context.TODO(), tikvMemberName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get StatefulSet %s/%s", ns, tikvMemberName)
			tikvImage := fmt.Sprintf("pingcap/tikv:%s", componentVersion)
			framework.ExpectEqual(tikvSts.Spec.Template.Spec.Containers[0].Image, tikvImage, "tikv sts image should be %q", tikvImage)

			tiflashMemberName := controller.TiFlashMemberName(tc.Name)
			tiflashSts, err := stsGetter.StatefulSets(ns).Get(context.TODO(), tiflashMemberName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get StatefulSet %s/%s", ns, tiflashMemberName)
			tiflashImage := fmt.Sprintf("pingcap/tiflash:%s", componentVersion)
			framework.ExpectEqual(tiflashSts.Spec.Template.Spec.Containers[0].Image, tiflashImage, "tiflash sts image should be %q", tiflashImage)

			tidbMemberName := controller.TiDBMemberName(tc.Name)
			tidbSts, err := stsGetter.StatefulSets(ns).Get(context.TODO(), tidbMemberName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get StatefulSet %s/%s", ns, tidbMemberName)
			tidbImage := fmt.Sprintf("pingcap/tidb:%s", componentVersion)
			// the 0th container for tidb pod is slowlog, which runs busybox
			framework.ExpectEqual(tidbSts.Spec.Template.Spec.Containers[1].Image, tidbImage, "tidb sts image should be %q", tidbImage)

			ticdcMemberName := controller.TiCDCMemberName(tc.Name)
			ticdcSts, err := stsGetter.StatefulSets(ns).Get(context.TODO(), ticdcMemberName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get StatefulSet %s/%s", ns, ticdcMemberName)
			ticdcImage := fmt.Sprintf("pingcap/ticdc:%s", componentVersion)
			framework.ExpectEqual(ticdcSts.Spec.Template.Spec.Containers[0].Image, ticdcImage, "ticdc sts image should be %q", ticdcImage)

			pumpMemberName := controller.PumpMemberName(tc.Name)
			pumpSts, err := stsGetter.StatefulSets(ns).Get(context.TODO(), pumpMemberName, metav1.GetOptions{})
			framework.ExpectNoError(err, "failed to get StatefulSet %s/%s", ns, pumpMemberName)
			pumpImage := fmt.Sprintf("pingcap/tidb-binlog:%s", componentVersion)
			framework.ExpectEqual(pumpSts.Spec.Template.Spec.Containers[0].Image, pumpImage, "pump sts image should be %q", pumpImage)
		}

		// upgrdae testing from previous version to latest version
		for _, prevVersion := range utilimage.TiDBPreviousVersions {
			ginkgo.It(fmt.Sprintf("for tc and components version upgrade from %s to latest version %s", prevVersion, utilimage.TiDBLatest), func() {
				upgradeTest(upgradeCase{
					oldVersion: prevVersion,
					newVersion: utilimage.TiDBLatest,
				})
			})
		}

		// upgrdae testing for specific versions
		utilginkgo.ContextWhenFocus("Specific Version", func() {
			configureV5x0x0 := func(tc *v1alpha1.TidbCluster) {
				pdCfg := v1alpha1.NewPDConfig()
				tikvCfg := v1alpha1.NewTiKVConfig()
				tidbCfg := v1alpha1.NewTiDBConfig()
				tiflashCfg := v1alpha1.NewTiFlashConfig()
				pdCfg.Set("log.level", "info")
				pdCfg.SetTable("replication",
					"max-replicas", 3,
					"location-labels", []string{"failure-domain.beta.kubernetes.io/region", "failure-domain.beta.kubernetes.io/zone", "kubernetes.io/hostname"})
				pdCfg.SetTable("dashboard",
					"internal-proxy", true,
					"enable-telemetry", false)
				tidbCfg.Set("new_collations_enabled_on_first_bootstrap", true)
				tidbCfg.Set("enable-telemetry", false)
				tidbCfg.Set("log.level", "info")
				tidbCfg.SetTable("performance",
					"max-procs", 0,
					"tcp-keep-alive", true)
				tikvCfg.Set("log.level", "info")
				tikvCfg.Set("readpool.storage.use-unified-pool", true)
				tikvCfg.Set("readpool.coprocessor.use-unified-pool", true)
				tiflashCfg.Common.Set("mark_cache_size", 2147483648)
				tiflashCfg.Common.Set("minmax_index_cache_size", 2147483648)
				tiflashCfg.Common.Set("max_memory_usage", 10737418240)
				tiflashCfg.Common.Set("max_memory_usage_for_all_queries", 32212254720)
				tc.Spec.PD.Config = pdCfg
				tc.Spec.TiKV.Config = tikvCfg
				tc.Spec.TiDB.Config = tidbCfg
				tc.Spec.TiFlash.Config = tiflashCfg
			}
			configureV5x0x2 := func(tc *v1alpha1.TidbCluster) {
				pdCfg := v1alpha1.NewPDConfig()
				tikvCfg := v1alpha1.NewTiKVConfig()
				tidbCfg := v1alpha1.NewTiDBConfig()
				tiflashCfg := v1alpha1.NewTiFlashConfig()
				pdCfg.Set("log.level", "info")
				pdCfg.SetTable("replication",
					"max-replicas", 3,
					"location-labels", []string{"failure-domain.beta.kubernetes.io/region", "failure-domain.beta.kubernetes.io/zone", "kubernetes.io/hostname"})
				pdCfg.SetTable("dashboard",
					"internal-proxy", true,
					"enable-telemetry", false)
				tidbCfg.Set("new_collations_enabled_on_first_bootstrap", true)
				tidbCfg.Set("enable-telemetry", false)
				tidbCfg.Set("log.level", "info")
				tidbCfg.SetTable("performance",
					"max-procs", 0,
					"tcp-keep-alive", true)
				tikvCfg.Set("log.level", "info")
				tikvCfg.Set("readpool.storage.use-unified-pool", true)
				tikvCfg.Set("readpool.coprocessor.use-unified-pool", true)
				tiflashCfg.Common.Set("mark_cache_size", 2147483648)
				tiflashCfg.Common.Set("minmax_index_cache_size", 2147483648)
				tiflashCfg.Common.Set("max_memory_usage", 10737418240)
				tiflashCfg.Common.Set("max_memory_usage_for_all_queries", 32212254720)
				tc.Spec.PD.Config = pdCfg
				tc.Spec.TiKV.Config = tikvCfg
				tc.Spec.TiDB.Config = tidbCfg
				tc.Spec.TiFlash.Config = tiflashCfg
			}
			configureV5x1x0 := func(tc *v1alpha1.TidbCluster) {
				pdCfg := v1alpha1.NewPDConfig()
				tikvCfg := v1alpha1.NewTiKVConfig()
				tidbCfg := v1alpha1.NewTiDBConfig()
				tiflashCfg := v1alpha1.NewTiFlashConfig()
				pdCfg.Set("replication.location-labels", []string{"failure-domain.beta.kubernetes.io/region", "failure-domain.beta.kubernetes.io/zone", "kubernetes.io/hostname"})
				pdCfg.SetTable("dashboard",
					"internal-proxy", true,
					"enable-telemetry", false)
				tidbCfg.Set("new_collations_enabled_on_first_bootstrap", true)
				tidbCfg.Set("enable-telemetry", false)
				tikvCfg.Set("server.grpc-concurrency", 3)
				tikvCfg.SetTable("rocksdb.defaultcf",
					"soft-pending-compaction-bytes-limit", "1024GB",
					"hard-pending-compaction-bytes-limit", "1024GB",
					"level0-slowdown-writes-trigger", 64,
					"level0-stop-writes-trigger", 128,
					"max-write-buffer-number", 10,
				)
				tikvCfg.SetTable("rocksdb.writecf",
					"soft-pending-compaction-bytes-limit", "1024GB",
					"hard-pending-compaction-bytes-limit", "1024GB",
					"level0-slowdown-writes-trigger", 64,
					"level0-stop-writes-trigger", 128,
					"max-write-buffer-number", 10,
				)
				tikvCfg.SetTable("rocksdb.lockcf",
					"level0-slowdown-writes-trigger", 64,
					"level0-stop-writes-trigger", 128,
					"max-write-buffer-number", 10,
				)
				tiflashCfg.Common.Set("mark_cache_size", 2147483648)
				tiflashCfg.Common.Set("minmax_index_cache_size", 2147483648)
				tiflashCfg.Common.Set("max_memory_usage", 10737418240)
				tiflashCfg.Common.Set("max_memory_usage_for_all_queries", 32212254720)
				tiflashCfg.Common.Set("delta_index_cache_size", 3221225472)
				tc.Spec.PD.Config = pdCfg
				tc.Spec.TiKV.Config = tikvCfg
				tc.Spec.TiDB.Config = tidbCfg
				tc.Spec.TiFlash.Config = tiflashCfg
			}

			cases := []upgradeCase{
				{
					oldVersion:              utilimage.TiDBV5x0x0,
					newVersion:              utilimage.TiDBLatest,
					configureOldTiDBCluster: configureV5x0x0,
					configureNewTiDBCluster: configureV5x1x0,
				},
				{
					oldVersion:              utilimage.TiDBV5x0x2,
					newVersion:              utilimage.TiDBLatest,
					configureOldTiDBCluster: configureV5x0x2,
					configureNewTiDBCluster: configureV5x1x0,
				},
			}
			for i := range cases {
				ucase := cases[i]
				ginkgo.It(fmt.Sprintf("for tc and components version upgrade from %s to %s", ucase.oldVersion, ucase.newVersion), func() {
					upgradeTest(ucase)
				})
			}
		})

		// this case merge scale-in/scale-out into one case, may seems a little bit dense
		// when scale-in, replica is first set to 5 and changed to 3
		// when scale-out, replica is first set to 3 and changed to 5
		ginkgo.Context("while concurrently scale PD", func() {
			operation := []string{"in", "out"}
			for _, op := range operation {
				op := op
				ginkgo.It(op, func() {
					initialReplicas, expectedReplicas := int32(3), int32(5)
					if op == "in" {
						initialReplicas, expectedReplicas = expectedReplicas, initialReplicas
					}

					tcName := fmt.Sprintf("scale-%s-pd-concurrently", op)
					tc := fixture.GetTidbCluster(ns, tcName, utilimage.TiDBLatestPrev)
					tc.Spec.PD.Replicas = initialReplicas

					ginkgo.By("Deploy initial tc")
					utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 10*time.Minute, 10*time.Second)

					ginkgo.By("Upgrade PD version")
					err := controller.GuaranteedUpdate(genericCli, tc, func() error {
						tc.Spec.PD.Version = pointer.StringPtr(utilimage.TiDBLatest)
						return nil
					})
					framework.ExpectNoError(err, "failed to update PD version to %q", utilimage.TiDBLatest)

					ginkgo.By(fmt.Sprintf("Wait for PD phase is %q", v1alpha1.UpgradePhase))
					utiltc.MustWaitForComponentPhase(cli, tc, v1alpha1.PDMemberType, v1alpha1.UpgradePhase, 3*time.Minute, time.Second*10)

					ginkgo.By(fmt.Sprintf("Scale %s PD while in %q phase", op, v1alpha1.UpgradePhase))
					err = controller.GuaranteedUpdate(genericCli, tc, func() error {
						tc.Spec.PD.Replicas = expectedReplicas
						return nil
					})
					framework.ExpectNoError(err, "failed to scale %s PD", op)

					err = oa.WaitForTidbClusterReady(tc, 10*time.Minute, 10*time.Second)
					framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %s/%s", ns, tc.Name)

					ginkgo.By("Check PD replicas")
					tc, err = cli.PingcapV1alpha1().TidbClusters(ns).Get(context.TODO(), tcName, metav1.GetOptions{})
					framework.ExpectNoError(err, "failed to get TidbCluster %s/%s: %v", ns, tcName, err)
					framework.ExpectEqual(tc.Spec.PD.Replicas, expectedReplicas)
				})
			}
		})

		// similar to PD scale-in/scale-out case above, need to check no evict leader scheduler left
		ginkgo.Context("while concurrently scale TiKV", func() {
			operation := []string{"in", "out"}
			for _, op := range operation {
				op := op
				ginkgo.It(op, func() {
					initialReplicas, expectedReplicas := int32(3), int32(4)
					if op == "in" {
						initialReplicas, expectedReplicas = expectedReplicas, initialReplicas
					}

					tcName := fmt.Sprintf("scale-%s-tikv-concurrently", op)
					tc := fixture.GetTidbCluster(ns, tcName, utilimage.TiDBLatestPrev)
					tc.Spec.TiKV.Replicas = initialReplicas

					ginkgo.By("Deploy initial tc")
					utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 10*time.Minute, 10*time.Second)

					ginkgo.By("Upgrade TiKV version")
					err := controller.GuaranteedUpdate(genericCli, tc, func() error {
						tc.Spec.TiKV.Version = pointer.StringPtr(utilimage.TiDBLatest)
						return nil
					})
					framework.ExpectNoError(err, "failed to update TiKV version to %q", utilimage.TiDBLatest)

					ginkgo.By(fmt.Sprintf("Wait for TiKV phase is %q", v1alpha1.UpgradePhase))
					utiltc.MustWaitForComponentPhase(cli, tc, v1alpha1.TiKVMemberType, v1alpha1.UpgradePhase, 3*time.Minute, time.Second*10)

					ginkgo.By(fmt.Sprintf("Scale %s TiKV while in %q phase", op, v1alpha1.UpgradePhase))
					err = controller.GuaranteedUpdate(genericCli, tc, func() error {
						tc.Spec.TiKV.Replicas = expectedReplicas
						return nil
					})
					framework.ExpectNoError(err, "failed to scale %s TiKV", op)

					ginkgo.By("Wait for TiKV to be in ScalePhase")
					utiltc.MustWaitForComponentPhase(cli, tc, v1alpha1.TiKVMemberType, v1alpha1.ScalePhase, time.Minute, time.Second)

					ginkgo.By("Wait for tc ready")
					err = oa.WaitForTidbClusterReady(tc, 10*time.Minute, 10*time.Second)
					framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %s/%s", ns, tc.Name)

					ginkgo.By("Check if TiKV replicas is correct")
					tc, err = cli.PingcapV1alpha1().TidbClusters(ns).Get(context.TODO(), tcName, metav1.GetOptions{})
					framework.ExpectNoError(err, "failed to get TidbCluster %s/%s: %v", ns, tcName)
					framework.ExpectEqual(tc.Spec.TiKV.Replicas, expectedReplicas, "TiKV replicas is not correct")

					ginkgo.By("Check no evict leader scheduler left")
					pdClient, cancel, err := proxiedpdclient.NewProxiedPDClient(secretLister, fw, ns, tc.Name, false)
					framework.ExpectNoError(err, "fail to create proxied pdClient")
					defer cancel()
					err = wait.Poll(5*time.Second, 3*time.Minute, func() (bool, error) {
						schedulers, err := pdClient.GetEvictLeaderSchedulers()
						framework.ExpectNoError(err, "failed to get evict leader schedulers")
						if len(schedulers) != 0 {
							log.Logf("there are %d evict leader left: %v", len(schedulers), schedulers)
							return false, nil
						}
						return true, nil
					})
					framework.ExpectNoError(err, "failed to wait for evict leader schedulers to become 0")
				})
			}
		})

		ginkgo.It("with bad PD config, then recover after force upgrading PD", func() {
			ginkgo.By("Deploy initial tc with incorrect PD image")
			tc := fixture.GetTidbCluster(ns, "force-upgrade-pd", utilimage.TiDBLatestPrev)
			tc.Spec.PD.BaseImage = "wrong-pd-image"
			err := genericCli.Create(context.TODO(), tc)
			framework.ExpectNoError(err, "failed to create TidbCluster %s/%s", ns, tc.Name)

			ginkgo.By("Wait for 1 min and ensure no healthy PD Pod exist")
			err = wait.PollImmediate(5*time.Second, 1*time.Minute, func() (bool, error) {
				listOptions := metav1.ListOptions{
					LabelSelector: labels.SelectorFromSet(label.New().Instance(tc.Name).Component(label.PDLabelVal).Labels()).String(),
				}
				pods, err := c.CoreV1().Pods(ns).List(context.TODO(), listOptions)
				framework.ExpectNoError(err, "failed to list Pods with selector %+v", listOptions)
				for _, pod := range pods.Items {
					framework.ExpectNotEqual(pod.Status.Phase, corev1.PodRunning, "expect PD Pod %s/%s not to be running", ns, pod.Name)
				}
				return false, nil
			})
			framework.ExpectEqual(err, wait.ErrWaitTimeout, "no Pod should be found for PD")

			ginkgo.By("Update PD Pod to correct image")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.PD.BaseImage = "pingcap/pd"
				return nil
			})
			framework.ExpectNoError(err, "failed to change PD Pod image")

			ginkgo.By("Wait for 1 min and ensure no healthy PD Pod exist")
			err = wait.PollImmediate(5*time.Second, 1*time.Minute, func() (bool, error) {
				listOptions := metav1.ListOptions{
					LabelSelector: labels.SelectorFromSet(label.New().Instance(tc.Name).Component(label.PDLabelVal).Labels()).String(),
				}
				pods, err := c.CoreV1().Pods(ns).List(context.TODO(), listOptions)
				framework.ExpectNoError(err, "failed to list Pods with selector %+v", listOptions)
				for _, pod := range pods.Items {
					framework.ExpectNotEqual(pod.Status.Phase, corev1.PodRunning, "expect PD Pod %s/%s not to be running", ns, pod.Name)
				}
				return false, nil
			})
			framework.ExpectEqual(err, wait.ErrWaitTimeout, "no Pod should be found for PD")

			ginkgo.By("Annotate TidbCluster for force upgrade")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				if tc.Annotations == nil {
					tc.Annotations = make(map[string]string)
				}
				tc.Annotations["tidb.pingcap.com/force-upgrade"] = "true"
				tc.Spec.PD.BaseImage = "wrong" // we need to make this right later
				return nil
			})
			framework.ExpectNoError(err, "failed to annotate tc for force upgrade")
			err = wait.PollImmediate(5*time.Second, 1*time.Minute, func() (bool, error) {
				tc, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(context.TODO(), tc.Name, metav1.GetOptions{})
				framework.ExpectNoError(err, "failed to get TidbCluster %s/%s", ns, tc.Name)
				val, exist := tc.Annotations["tidb.pingcap.com/force-upgrade"]
				if !exist {
					log.Logf("annotation tidb.pingcap.com/force-upgrade not exist in tc")
					return false, nil
				}
				framework.ExpectEqual(val, "true", "tc annotation tidb.pingcap.com/force-upgrade is not \"true\", but %q", val)
				return true, nil
			})
			framework.ExpectNoError(err, "failed to wait for annotation tidb.pingcap.com/force-upgrade")

			ginkgo.By("Update PD Pod to correct image")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.PD.BaseImage = "pingcap/pd"
				return nil
			})
			framework.ExpectNoError(err, "failed to change PD Pod image")
			err = oa.WaitForTidbClusterReady(tc, 10*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "failed to wait for TidbCluster ready: %q", tc.Name)
		})
	})

	ginkgo.It("Deleted objects controlled by TidbCluster will be recovered by Operator", func() {
		ginkgo.By("Deploy initial tc")
		tc := fixture.GetTidbCluster(ns, "delete-objects", utilimage.TiDBLatest)
		utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)

		ginkgo.By("Delete StatefulSet/ConfigMap/Service of PD")
		listOptions := metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(label.New().Instance(tc.Name).Labels()).String(),
		}
		stsList, err := stsGetter.StatefulSets(ns).List(context.TODO(), listOptions)
		framework.ExpectNoError(err, "failed to list StatefulSet with option: %+v", listOptions)
		for _, sts := range stsList.Items {
			err := stsGetter.StatefulSets(ns).Delete(context.TODO(), sts.Name, metav1.DeleteOptions{})
			framework.ExpectNoError(err, "failed to delete StatefulSet %s/%s", ns, sts.Name)
		}
		cmList, err := c.CoreV1().ConfigMaps(ns).List(context.TODO(), listOptions)
		framework.ExpectNoError(err, "failed to list ConfigMap with option: %+v", listOptions)
		for _, cm := range cmList.Items {
			err := c.CoreV1().ConfigMaps(ns).Delete(context.TODO(), cm.Name, metav1.DeleteOptions{})
			framework.ExpectNoError(err, "failed to delete ConfigMap %s/%s", ns, cm.Name)
		}
		svcList, err := c.CoreV1().Services(ns).List(context.TODO(), listOptions)
		framework.ExpectNoError(err, "failed to list Service with option: %+v", listOptions)
		for _, svc := range svcList.Items {
			err := c.CoreV1().Services(ns).Delete(context.TODO(), svc.Name, metav1.DeleteOptions{})
			framework.ExpectNoError(err, "failed to delete Service %s/%s", ns, svc.Name)
		}

		ginkgo.By("Wait for TidbCluster components ready again")
		err = oa.WaitForTidbClusterReady(tc, 10*time.Minute, 10*time.Second)
		framework.ExpectNoError(err, "failed to wait for TidbCluster %s/%s components ready", tc.Namespace, tc.Name)
	})

	ginkgo.Context("Scale in", func() {
		ginkgo.Context("and then scale out", func() {
			components := []v1alpha1.MemberType{"pd", "tikv", "tidb"}
			// TODO: refactor fixture.GetTidbCluster to support all the components through parameters more easily
			// components := []string{"PD", "TiKV", "TiFlash", "TiDB", "TiCDC", "Pump"}
			for _, comp := range components {
				comp := comp
				var replicasLarge, replicasSmall int32
				switch comp {
				case v1alpha1.PDMemberType:
					replicasLarge = 5
					replicasSmall = 3
				case v1alpha1.TiKVMemberType:
					replicasLarge = 5
					replicasSmall = 3
				case v1alpha1.TiDBMemberType:
					replicasLarge = 4
					replicasSmall = 2
				}
				ginkgo.It(fmt.Sprintf("should work for %s", comp), func() {
					ginkgo.By("Deploy initial tc")
					tc := fixture.GetTidbCluster(ns, fmt.Sprintf("scale-out-scale-in-%s", comp), utilimage.TiDBLatest)
					switch comp {
					case v1alpha1.PDMemberType:
						tc.Spec.PD.Replicas = replicasLarge
					case v1alpha1.TiKVMemberType:
						tc.Spec.TiKV.Replicas = replicasLarge
					case v1alpha1.TiDBMemberType:
						tc.Spec.TiDB.Replicas = replicasLarge
					}
					utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)

					ginkgo.By(fmt.Sprintf("Scale in %s", comp))
					err := controller.GuaranteedUpdate(genericCli, tc, func() error {
						switch comp {
						case v1alpha1.PDMemberType:
							tc.Spec.PD.Replicas = replicasSmall
						case v1alpha1.TiKVMemberType:
							tc.Spec.TiKV.Replicas = replicasSmall
						case v1alpha1.TiDBMemberType:
							tc.Spec.TiDB.Replicas = replicasSmall
						}
						return nil
					})
					framework.ExpectNoError(err, "failed to scale in %s for TidbCluster %s/%s", comp, ns, tc.Name)
					ginkgo.By(fmt.Sprintf("Wait for %s to be in ScalePhase", comp))
					utiltc.MustWaitForComponentPhase(cli, tc, comp, v1alpha1.ScalePhase, time.Minute, time.Second)
					log.Logf(fmt.Sprintf("%s is in ScalePhase", comp))

					ginkgo.By("Wait for tc ready")
					err = oa.WaitForTidbClusterReady(tc, 5*time.Minute, 10*time.Second)
					framework.ExpectNoError(err, "failed to wait for TidbCluster %s/%s ready after scale in %s", ns, tc.Name, comp)
					log.Logf("tc is ready")

					pvcUIDs := make(map[string]string)
					ginkgo.By("Check PVC annotation tidb.pingcap.com/pvc-defer-deleting")
					err = wait.Poll(10*time.Second, 3*time.Minute, func() (done bool, err error) {
						for ordinal := replicasSmall; ordinal < replicasLarge; ordinal++ {
							var pvcSelector labels.Selector
							pvcSelector, err = member.GetPVCSelectorForPod(tc, comp, int32(ordinal))
							framework.ExpectNoError(err, "failed to get PVC selector for tc %s/%s", tc.GetNamespace(), tc.GetName())
							pvcs, err := c.CoreV1().PersistentVolumeClaims(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: pvcSelector.String()})
							framework.ExpectNoError(err, "failed to list PVCs with selector: %v", pvcSelector)
							if comp == v1alpha1.PDMemberType || comp == v1alpha1.TiKVMemberType {
								framework.ExpectNotEqual(len(pvcs.Items), 0, "expect at least one PVCs for %s", comp)
							}
							for _, pvc := range pvcs.Items {
								annotations := pvc.GetObjectMeta().GetAnnotations()
								log.Logf("pvc annotations: %+v", annotations)
								_, ok := annotations["tidb.pingcap.com/pvc-defer-deleting"]
								if !ok {
									log.Logf("PVC %s/%s does not have annotation tidb.pingcap.com/pvc-defer-deleting", pvc.GetNamespace(), pvc.GetName())
									return false, nil
								}
								pvcUIDs[pvc.Name] = string(pvc.UID)
							}
						}
						return true, nil
					})
					framework.ExpectNoError(err, "expect PVCs of scaled in Pods to have annotation tidb.pingcap.com/pvc-defer-deleting")

					ginkgo.By(fmt.Sprintf("Scale out %s", comp))
					err = controller.GuaranteedUpdate(genericCli, tc, func() error {
						switch comp {
						case v1alpha1.PDMemberType:
							tc.Spec.PD.Replicas = replicasLarge
						case v1alpha1.TiKVMemberType:
							tc.Spec.TiKV.Replicas = replicasLarge
						case v1alpha1.TiDBMemberType:
							tc.Spec.TiDB.Replicas = replicasLarge
						}
						return nil
					})
					framework.ExpectNoError(err, "failed to scale out %s for TidbCluster %s/%s", comp, ns, tc.Name)
					ginkgo.By(fmt.Sprintf("Wait for %s to be in ScalePhase", comp))
					utiltc.MustWaitForComponentPhase(cli, tc, comp, v1alpha1.ScalePhase, time.Minute, time.Second)
					log.Logf(fmt.Sprintf("%s is in ScalePhase", comp))

					ginkgo.By("Wait for tc ready")
					err = oa.WaitForTidbClusterReady(tc, 5*time.Minute, 10*time.Second)
					framework.ExpectNoError(err, "failed to wait for TidbCluster %s/%s ready after scale out %s", ns, tc.Name, comp)

					ginkgo.By(fmt.Sprintf("Check PVCs are recreated for newly scaled out %s", comp))
					for ordinal := replicasSmall; ordinal < replicasLarge; ordinal++ {
						var pvcSelector labels.Selector
						pvcSelector, err = member.GetPVCSelectorForPod(tc, comp, int32(ordinal))
						framework.ExpectNoError(err, "failed to get PVC selector for tc %s/%s", tc.GetNamespace(), tc.GetName())
						pvcs, err := c.CoreV1().PersistentVolumeClaims(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: pvcSelector.String()})
						framework.ExpectNoError(err, "failed to list PVCs with selector: %v", pvcSelector)
						if comp == v1alpha1.PDMemberType || comp == v1alpha1.TiKVMemberType {
							framework.ExpectNotEqual(len(pvcs.Items), 0, "expect at least one PVCs for %s", comp)
						}
						for _, pvc := range pvcs.Items {
							annotations := pvc.GetObjectMeta().GetAnnotations()
							log.Logf("pvc annotations: %+v", annotations)
							_, ok := annotations["tidb.pingcap.com/pvc-defer-deleting"]
							framework.ExpectEqual(ok, false, "expect PVC %s/%s not to have annotation tidb.pingcap.com/pvc-defer-deleting", pvc.GetNamespace(), pvc.GetName())
							pvcUIDString := pvcUIDs[pvc.Name]
							framework.ExpectNotEqual(string(pvc.UID), pvcUIDString)
						}
					}
				})
			}
		})

		ginkgo.Context("while concurrently upgrade", func() {
			components := []v1alpha1.MemberType{"pd", "tikv", "tidb"}
			for _, comp := range components {
				comp := comp
				var replicasLarge, replicasSmall int32
				switch comp {
				case v1alpha1.PDMemberType:
					replicasLarge = 5
					replicasSmall = 3
				case v1alpha1.TiKVMemberType:
					replicasLarge = 5
					replicasSmall = 3
				case v1alpha1.TiDBMemberType:
					replicasLarge = 4
					replicasSmall = 2
				}
				ginkgo.It(fmt.Sprintf("should work for %s", comp), func() {
					ginkgo.By("Deploy initial tc")
					tc := fixture.GetTidbCluster(ns, fmt.Sprintf("scale-in-upgrade-%s", comp), utilimage.TiDBLatestPrev)
					switch comp {
					case v1alpha1.PDMemberType:
						tc.Spec.PD.Replicas = replicasLarge
					case v1alpha1.TiKVMemberType:
						tc.Spec.TiKV.Replicas = replicasLarge
					case v1alpha1.TiDBMemberType:
						tc.Spec.TiDB.Replicas = replicasLarge
					}
					utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)

					ginkgo.By(fmt.Sprintf("Scale in %s to %d replicas", comp, replicasSmall))
					err := controller.GuaranteedUpdate(genericCli, tc, func() error {
						switch comp {
						case v1alpha1.PDMemberType:
							tc.Spec.PD.Replicas = replicasSmall
						case v1alpha1.TiKVMemberType:
							tc.Spec.TiKV.Replicas = replicasSmall
						case v1alpha1.TiDBMemberType:
							tc.Spec.TiDB.Replicas = replicasSmall
						}
						return nil
					})
					framework.ExpectNoError(err, "failed to scale in %s for TidbCluster %s/%s", comp, ns, tc.Name)

					ginkgo.By(fmt.Sprintf("Wait for %s to be in ScalePhase", comp))
					utiltc.MustWaitForComponentPhase(cli, tc, comp, v1alpha1.ScalePhase, time.Minute, time.Second)

					ginkgo.By(fmt.Sprintf("Upgrade %s version concurrently", comp))
					err = controller.GuaranteedUpdate(genericCli, tc, func() error {
						switch comp {
						case v1alpha1.PDMemberType:
							tc.Spec.PD.Version = pointer.StringPtr(utilimage.TiDBLatest)
						case v1alpha1.TiKVMemberType:
							tc.Spec.TiKV.Version = pointer.StringPtr(utilimage.TiDBLatest)
						case v1alpha1.TiDBMemberType:
							tc.Spec.TiDB.Version = pointer.StringPtr(utilimage.TiDBLatest)
						}
						return nil
					})
					framework.ExpectNoError(err, "failed to upgrade %s for TidbCluster %s/%s", comp, ns, tc.Name)

					ginkgo.By("Wait for tc ready")
					err = oa.WaitForTidbClusterReady(tc, 10*time.Minute, 10*time.Second)
					framework.ExpectNoError(err, "failed to wait for TidbCluster %s/%s ready after scale in %s", ns, tc.Name, comp)

					ginkgo.By(fmt.Sprintf("Check %s Pods", comp))
					var labelSelector string
					switch comp {
					case v1alpha1.PDMemberType:
						labelSelector = labels.SelectorFromSet(label.New().Instance(tc.Name).PD().Labels()).String()
					case v1alpha1.TiKVMemberType:
						labelSelector = labels.SelectorFromSet(label.New().Instance(tc.Name).TiKV().Labels()).String()
					case v1alpha1.TiDBMemberType:
						labelSelector = labels.SelectorFromSet(label.New().Instance(tc.Name).TiDB().Labels()).String()
					}
					pods := utilpod.MustListPods(labelSelector, ns, c)
					framework.ExpectEqual(len(pods), int(replicasSmall), "there should be %d %s Pods", replicasSmall, comp)
					for _, pod := range pods {
						// some pods may have multiple containers
						wrongImage := true
						for _, c := range pod.Spec.Containers {
							log.Logf("container image: %s", c.Image)
							if fmt.Sprintf("pingcap/%s:%s", comp, utilimage.TiDBLatest) == c.Image {
								wrongImage = false
								break
							}
						}
						framework.ExpectEqual(wrongImage, false, "%s Pod has wrong image, expected %s", comp, utilimage.TiDBLatest)
					}
				})
			}
		})

		ginkgo.It("PD to 0 is forbidden while other components are running", func() {
			ginkgo.By("Deploy initial tc")
			tc := fixture.GetTidbCluster(ns, "scale-pd-to-0", utilimage.TiDBLatest)
			tc.Spec.PD.Replicas = 1
			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)

			ginkgo.By("Scale in PD to 0 replicas")
			err := controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.PD.Replicas = 0
				return nil
			})
			framework.ExpectNoError(err, "failed to scale in PD for TidbCluster %s/%s", ns, tc.Name)

			ginkgo.By("Wait for PD to be in ScalePhase")
			utiltc.MustWaitForComponentPhase(cli, tc, v1alpha1.PDMemberType, v1alpha1.ScalePhase, time.Minute, time.Second)

			ginkgo.By("Check for FailedScaleIn event")
			// LAST SEEN   TYPE      REASON          OBJECT              MESSAGE
			// 25s         Warning   FailedScaleIn   tidbcluster/basic   The PD is in use by TidbCluster [pingcap/basic], can't scale in PD, podname basic-pd-0
			err = wait.Poll(5*time.Second, 1*time.Minute, func() (done bool, err error) {
				options := metav1.ListOptions{
					FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=TidbCluster", tc.Name),
				}
				event, err := c.CoreV1().Events(ns).List(context.TODO(), options)
				if err != nil {
					log.Logf("failed to list events with options: +%v", options)
					return false, nil
				}
				for _, event := range event.Items {
					log.Logf("found event: %+v", event)
					if event.Reason == "FailedScaleIn" && strings.Contains(event.Message, "PD") {
						return true, nil
					}
				}
				return false, nil
			})
			framework.ExpectNoError(err, "failed to wait for FailedScaleIn event")
		})

		ginkgo.It("TiKV from >=3 replicas to <3 should be forbidden", func() {
			ginkgo.By("Deploy initial tc")
			tc := fixture.GetTidbCluster(ns, "scale-in-tikv", utilimage.TiDBLatest)
			tc, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(context.TODO(), tc, metav1.CreateOptions{})
			framework.ExpectNoError(err, "Expected create tidbcluster")
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "Expected tidbcluster ready")

			ginkgo.By("Scale in tikv to 2 replicas")
			err = controller.GuaranteedUpdate(genericCli, tc, func() error {
				tc.Spec.TiKV.Replicas = 2
				return nil
			})
			framework.ExpectNoError(err, "failed to scale in tikv")

			ginkgo.By("Expect up stores number stays 3")
			pdClient, cancel, err := proxiedpdclient.NewProxiedPDClient(secretLister, fw, ns, tc.Name, false)
			framework.ExpectNoError(err, "create pdClient error")
			defer cancel()

			_ = wait.PollImmediate(15*time.Second, 3*time.Minute, func() (bool, error) {
				storesInfo, err := pdClient.GetStores()
				framework.ExpectNoError(err, "get stores info error")
				framework.ExpectEqual(storesInfo.Count, 3, "Expect number of stores is 3")
				for _, store := range storesInfo.Stores {
					framework.ExpectEqual(store.Store.StateName, "Up", "Expect state of stores are Up")
				}
				return false, nil
			})
		})
	})

	ginkgo.Describe("[Feature]: PodSecurityContext", func() {
		ginkgo.It("TidbCluster global pod security context", func() {
			ginkgo.By("Deploy tidbCluster")
			userID := int64(1000)
			groupID := int64(2000)
			tc := fixture.GetTidbCluster(ns, "run-as-non-root", utilimage.TiDBLatest)
			tc = fixture.AddTiFlashForTidbCluster(tc)
			tc = fixture.AddTiCDCForTidbCluster(tc)
			tc = fixture.AddPumpForTidbCluster(tc)

			tc.Spec.PD.Replicas = 1
			tc.Spec.TiDB.Replicas = 1
			tc.Spec.TiKV.Replicas = 1

			tc.Spec.PodSecurityContext = &corev1.PodSecurityContext{
				RunAsUser:  &userID,
				RunAsGroup: &groupID,
				FSGroup:    &groupID,
			}

			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)
			podList, err := c.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
			framework.ExpectNoError(err, "failed to list pods: %+v")
			for i := range podList.Items {
				framework.ExpectNotEqual(podList.Items[i].Spec.SecurityContext, nil, "Expected security context is not nil")
				framework.ExpectEqual(podList.Items[i].Spec.SecurityContext.RunAsUser, &userID, "Expected run as user ", userID)
				framework.ExpectEqual(podList.Items[i].Spec.SecurityContext.RunAsGroup, &groupID, "Expected run as group ", groupID)
				framework.ExpectEqual(podList.Items[i].Spec.SecurityContext.FSGroup, &groupID, "Expected fs group ", groupID)
			}
		})
	})

	ginkgo.Describe("[Feature]: TopologySpreadConstraint", func() {
		nodeZoneMap := map[string]string{}

		ginkgo.BeforeEach(func() {
			nodeList, err := c.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
			framework.ExpectNoError(err, "failed to list nodes: %+v")
			for i := range nodeList.Items {
				node := &nodeList.Items[i]
				zone, ok := node.Labels[tests.LabelKeyTestingZone]
				framework.ExpectEqual(ok, true, "label %s should exist", tests.LabelKeyTestingZone)
				nodeZoneMap[node.Name] = zone
			}
		})

		ginkgo.It("TidbCluster global topology spread contraint", func() {
			ginkgo.By("Deploy tidbCluster")
			tc := fixture.GetTidbCluster(ns, "topology-test", utilimage.TiDBLatest)
			tc.Spec.TopologySpreadConstraints = []v1alpha1.TopologySpreadConstraint{
				{
					TopologyKey: tests.LabelKeyTestingZone,
				},
			}
			// change to use default scheduler
			tc.Spec.SchedulerName = "default-scheduler"
			tc.Spec.PD.Replicas = 3
			tc.Spec.TiDB.Replicas = 2
			tc.Spec.TiKV.Replicas = 2

			tc = fixture.AddTiFlashForTidbCluster(tc)
			tc = fixture.AddTiCDCForTidbCluster(tc)
			tc = fixture.AddPumpForTidbCluster(tc)

			tc.Spec.TiFlash.Replicas = 2
			tc.Spec.TiCDC.Replicas = 2
			tc.Spec.Pump.Replicas = 2

			utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)
			podList, err := c.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
			framework.ExpectNoError(err, "failed to list pods: %+v")

			err = validatePodSpread(podList.Items, nodeZoneMap, []string{label.PDLabelVal}, 1)
			framework.ExpectNoError(err, "failed even spread pd pods: %v")

			err = validatePodSpread(podList.Items, nodeZoneMap, []string{
				label.TiDBLabelVal,
				label.TiKVLabelVal,
				label.TiFlashLabelVal,
				label.TiCDCLabelVal,
				label.PumpLabelVal,
			}, 0)
			framework.ExpectNoError(err, "failed even spread pods: %v")
		})
	})

	ginkgo.It("deploy tidb monitor specified shards normally", func() {
		ginkgo.By("Deploy initial tc")
		tc := fixture.GetTidbCluster(ns, "monitor-test", utilimage.TiDBLatest)
		tc.Spec.PD.Replicas = 1
		tc.Spec.TiKV.Replicas = 1
		tc.Spec.TiDB.Replicas = 5
		tc, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(context.TODO(), tc, metav1.CreateOptions{})
		framework.ExpectNoError(err, "Expected create tidbcluster")
		err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
		framework.ExpectNoError(err, "Expected get tidbcluster")

		ginkgo.By("Deploy tidbmonitor")
		tm := fixture.NewTidbMonitor("monitor-test", ns, tc, true, true, false)
		tm.Spec.Shards = pointer.Int32Ptr(2)
		pvpDelete := corev1.PersistentVolumeReclaimDelete
		tm.Spec.PVReclaimPolicy = &pvpDelete
		_, err = cli.PingcapV1alpha1().TidbMonitors(ns).Create(context.TODO(), tm, metav1.CreateOptions{})
		framework.ExpectNoError(err, "Expected tidbmonitor deployed success")
		err = tests.CheckTidbMonitor(tm, cli, c, fw)
		framework.ExpectNoError(err, "Expected tidbmonitor checked success")

		ginkgo.By("Delete tidbmonitor")
		err = cli.PingcapV1alpha1().TidbMonitors(tm.Namespace).Delete(context.TODO(), tm.Name, metav1.DeleteOptions{})
		framework.ExpectNoError(err, "delete tidbmonitor failed")
	})

	ginkgo.It("deploy tidb monitor remote write successfully", func() {
		ginkgo.By("Deploy initial tc")
		tc := fixture.GetTidbCluster(ns, "monitor-test", utilimage.TiDBLatest)
		tc.Spec.PD.Replicas = 1
		tc.Spec.TiKV.Replicas = 1
		tc.Spec.TiDB.Replicas = 1
		tc, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(context.TODO(), tc, metav1.CreateOptions{})
		framework.ExpectNoError(err, "Expected create tidbcluster")
		err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
		framework.ExpectNoError(err, "Expected get tidbcluster")

		ginkgo.By("Deploy tidbmonitor")
		tm := fixture.NewTidbMonitor("monitor-test", ns, tc, true, true, false)
		tm.Spec.Prometheus.RemoteWrite = []*v1alpha1.RemoteWriteSpec{
			{
				URL: fmt.Sprintf("http://thanos-receiver-0.thanos-receiver.%s.svc:19291/api/v1/receive", ns),
				QueueConfig: &v1alpha1.QueueConfig{
					MaxSamplesPerSend: 100,
					MaxShards:         100,
				},
				MetadataConfig: &v1alpha1.MetadataConfig{
					SendInterval: "10s",
				},
			},
		}
		pvpDelete := corev1.PersistentVolumeReclaimDelete
		tm.Spec.PVReclaimPolicy = &pvpDelete
		_, err = cli.PingcapV1alpha1().TidbMonitors(ns).Create(context.TODO(), tm, metav1.CreateOptions{})

		framework.ExpectNoError(err, "Expected tidbmonitor deployed success")
		err = tests.CheckTidbMonitor(tm, cli, c, fw)
		framework.ExpectNoError(err, "Expected tidbmonitor checked success")

		decode := k8sScheme.Codecs.UniversalDeserializer().Decode
		thanosReceiverConfigmapObj, _, _ := decode([]byte(fmt.Sprintf(remotewrite.ThanosReceiverConfigmapYaml, ns)), nil, nil)
		thanosReceiverConfigmap := thanosReceiverConfigmapObj.(*corev1.ConfigMap) // This fails
		_, err = c.CoreV1().ConfigMaps(ns).Create(context.TODO(), thanosReceiverConfigmap, metav1.CreateOptions{})
		framework.ExpectNoError(err, "Expected thanos receiver configmap created success")

		decode = k8sScheme.Codecs.UniversalDeserializer().Decode
		thanosReceiverStsObj, _, _ := decode([]byte(remotewrite.ThanosReceiverYaml), nil, nil)
		thanosReceiverSts := thanosReceiverStsObj.(*v1.StatefulSet) // This fails
		_, err = c.AppsV1().StatefulSets(ns).Create(context.TODO(), thanosReceiverSts, metav1.CreateOptions{})
		framework.ExpectNoError(err, "Expected thanos receiver created success")

		err = wait.PollImmediate(15*time.Second, 3*time.Minute, func() (bool, error) {
			receiverSts, err := c.AppsV1().StatefulSets(ns).Get(context.TODO(), thanosReceiverSts.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if receiverSts.Status.ReadyReplicas == 1 {
				return true, nil
			}
			return false, nil
		})
		framework.ExpectNoError(err, "Expected thanos receiver deployed ready success")

		decode = k8sScheme.Codecs.UniversalDeserializer().Decode
		thanosReceiverServiceObj, _, _ := decode([]byte(remotewrite.ThanosReceiverServiceYaml), nil, nil)
		thanosReceiverService := thanosReceiverServiceObj.(*corev1.Service) // This fails
		_, err = c.CoreV1().Services(ns).Create(context.TODO(), thanosReceiverService, metav1.CreateOptions{})
		framework.ExpectNoError(err, "Expected thanos receiver service created success")

		decode = k8sScheme.Codecs.UniversalDeserializer().Decode
		thanosQueryDeploymentObj, _, _ := decode([]byte(remotewrite.ThanosQueryYaml), nil, nil)
		thanosQueryDeployment := thanosQueryDeploymentObj.(*v1.Deployment) // This fails
		_, err = c.AppsV1().Deployments(ns).Create(context.TODO(), thanosQueryDeployment, metav1.CreateOptions{})
		framework.ExpectNoError(err, "Expected thanos query created success")
		err = wait.PollImmediate(15*time.Second, 3*time.Minute, func() (bool, error) {
			queryDeployment, err := c.AppsV1().Deployments(ns).Get(context.TODO(), thanosQueryDeployment.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if queryDeployment.Status.ReadyReplicas == 1 {
				return true, nil
			}
			return false, nil
		})
		framework.ExpectNoError(err, "Expected thanos query deployed ready success")

		decode = k8sScheme.Codecs.UniversalDeserializer().Decode
		thanosQueryServiceObj, _, _ := decode([]byte(remotewrite.ThanosQueryServiceYaml), nil, nil)
		thanosQueryService := thanosQueryServiceObj.(*corev1.Service) // This fails
		_, err = c.CoreV1().Services(ns).Create(context.TODO(), thanosQueryService, metav1.CreateOptions{})
		framework.ExpectNoError(err, "Expected thanos query service created success")

		// Check 3 targets, 1 TiDB + 1 PD + 1 TiKV
		err = tests.CheckThanosQueryData("thanos-query", tm.Namespace, fw, 3)
		framework.ExpectNoError(err, "Expected thanos query check success")

		ginkgo.By("Delete tidbmonitor")
		err = cli.PingcapV1alpha1().TidbMonitors(tm.Namespace).Delete(context.TODO(), tm.Name, metav1.DeleteOptions{})
		framework.ExpectNoError(err, "delete tidbmonitor failed")
		ginkgo.By("Delete thanos query")
		err = c.AppsV1().Deployments(ns).Delete(context.TODO(), thanosQueryService.Name, metav1.DeleteOptions{})
		framework.ExpectNoError(err, "delete thanos query failed")
		err = c.CoreV1().Services(ns).Delete(context.TODO(), thanosQueryService.Name, metav1.DeleteOptions{})
		framework.ExpectNoError(err, "delete thanos query service failed")
		ginkgo.By("Delete thanos receiver")
		err = c.AppsV1().StatefulSets(ns).Delete(context.TODO(), thanosReceiverSts.Name, metav1.DeleteOptions{})
		framework.ExpectNoError(err, "delete thanos receiver failed")
		err = c.CoreV1().Services(ns).Delete(context.TODO(), thanosReceiverService.Name, metav1.DeleteOptions{})
		framework.ExpectNoError(err, "delete thanos receiver service failed")
	})

	ginkgo.Describe("[Feature]: RandomPassword", func() {
		ginkgo.It("deploy tidb cluster with random password", func() {
			ginkgo.By("Deploy initial tc")
			tc := fixture.GetTidbCluster(ns, "random-password", utilimage.TiDBLatest)
			tc.Spec.PD.Replicas = 1
			tc.Spec.TiKV.Replicas = 1
			tc.Spec.TiDB.Replicas = 1
			tc.Spec.TiDB.Initializer = &v1alpha1.TiDBInitializer{CreatePassword: true}

			tc, err := cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(context.TODO(), tc, metav1.CreateOptions{})
			framework.ExpectNoError(err, "Expected create tidbcluster")
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "Expected get tidbcluster")

			err = oa.WaitForTidbClusterInitRandomPassword(tc, fw, 10*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "Expected tidbcluster connect success")
		})
		ginkgo.It("deploy tls tidb cluster with random password", func() {
			tcName := "tls-random-password"

			ginkgo.By("Installing tidb CA certificate")
			err := InstallTiDBIssuer(ns, tcName)
			framework.ExpectNoError(err, "failed to install CA certificate")

			ginkgo.By("Installing tidb server and client certificate")
			err = InstallTiDBCertificates(ns, tcName)
			framework.ExpectNoError(err, "failed to install tidb server and client certificate")

			ginkgo.By("Installing tidbInitializer client certificate")
			err = installTiDBInitializerCertificates(ns, tcName)
			framework.ExpectNoError(err, "failed to install tidbInitializer client certificate")

			ginkgo.By("Installing dashboard client certificate")
			err = installPDDashboardCertificates(ns, tcName)
			framework.ExpectNoError(err, "failed to install dashboard client certificate")

			ginkgo.By("Installing tidb components certificates")
			err = InstallTiDBComponentsCertificates(ns, tcName)
			framework.ExpectNoError(err, "failed to install tidb components certificates")

			ginkgo.By("Creating tidb cluster with TLS enabled")
			dashTLSName := fmt.Sprintf("%s-dashboard-tls", tcName)
			tc := fixture.GetTidbCluster(ns, tcName, utilimage.TiDBLatestPrev)
			tc = fixture.AddTiFlashForTidbCluster(tc)
			tc = fixture.AddTiCDCForTidbCluster(tc)
			tc.Spec.PD.Replicas = 1
			tc.Spec.PD.TLSClientSecretName = &dashTLSName
			tc.Spec.TiKV.Replicas = 1
			tc.Spec.TiDB.Replicas = 1
			tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}

			err = genericCli.Create(context.TODO(), tc)
			framework.ExpectNoError(err, "failed to create TidbCluster: %q", tc.Name)
			err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
			framework.ExpectNoError(err, "wait for TidbCluster ready timeout: %q", tc.Name)

			err = oa.WaitForTidbClusterInitRandomPassword(tc, fw, 10*time.Minute, 10*time.Second)
			framework.ExpectNoError(err, "Expected tidbcluster connect success")
		})
	})

	ginkgo.Describe("Start Script Version", func() {

		type testcase struct {
			nameSuffix string
			tlsEnable  bool
		}

		cases := []testcase{
			{
				nameSuffix: "",
				tlsEnable:  false,
			},
			{
				nameSuffix: "and enable TLS",
				tlsEnable:  true,
			},
		}

		for _, testcase := range cases {
			ginkgo.It("deploy cluster with start script v2 "+testcase.nameSuffix, func() {
				tcName := "start-script-v2"
				tc := fixture.GetTidbCluster(ns, tcName, utilimage.TiDBLatest)
				tc = fixture.AddTiFlashForTidbCluster(tc)
				tc = fixture.AddTiCDCForTidbCluster(tc)
				tc = fixture.AddPumpForTidbCluster(tc)
				tc.Spec.PD.Replicas = 3
				tc.Spec.TiDB.Replicas = 1
				tc.Spec.TiKV.Replicas = 3
				tc.Spec.TiFlash.Replicas = 3
				tc.Spec.Pump.Replicas = 1
				tc.Spec.TiCDC.Replicas = 1
				tc.Spec.StartScriptVersion = v1alpha1.StartScriptV2

				if testcase.tlsEnable {
					tc.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
					tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}

					ginkgo.By("Installing tidb CA certificate")
					err := InstallTiDBIssuer(ns, tcName)
					framework.ExpectNoError(err, "failed to install CA certificate")

					ginkgo.By("Installing tidb server and client certificate")
					err = InstallTiDBCertificates(ns, tcName)
					framework.ExpectNoError(err, "failed to install tidb server and client certificate")

					ginkgo.By("Installing tidbInitializer client certificate")
					err = installTiDBInitializerCertificates(ns, tcName)
					framework.ExpectNoError(err, "failed to install tidbInitializer client certificate")

					ginkgo.By("Installing dashboard client certificate")
					err = installPDDashboardCertificates(ns, tcName)
					framework.ExpectNoError(err, "failed to install dashboard client certificate")

					ginkgo.By("Installing tidb components certificates")
					err = InstallTiDBComponentsCertificates(ns, tcName)
					framework.ExpectNoError(err, "failed to install tidb components certificates")
				}

				ginkgo.By("Deploy tidb cluster")
				utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)
			})

			ginkgo.It("migrate start script from v1 to v2 "+testcase.nameSuffix, func() {
				tcName := "migrate-start-script-v2"
				tc := fixture.GetTidbCluster(ns, tcName, utilimage.TiDBLatest)
				tc = fixture.AddTiFlashForTidbCluster(tc)
				tc = fixture.AddTiCDCForTidbCluster(tc)
				tc = fixture.AddPumpForTidbCluster(tc)
				tc.Spec.PD.Replicas = 3
				tc.Spec.TiDB.Replicas = 1
				tc.Spec.TiKV.Replicas = 3
				tc.Spec.TiFlash.Replicas = 3
				tc.Spec.Pump.Replicas = 1
				tc.Spec.TiCDC.Replicas = 1

				if testcase.tlsEnable {
					tc.Spec.TiDB.TLSClient = &v1alpha1.TiDBTLSClient{Enabled: true}
					tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}

					ginkgo.By("Installing tidb CA certificate")
					err := InstallTiDBIssuer(ns, tcName)
					framework.ExpectNoError(err, "failed to install CA certificate")

					ginkgo.By("Installing tidb server and client certificate")
					err = InstallTiDBCertificates(ns, tcName)
					framework.ExpectNoError(err, "failed to install tidb server and client certificate")

					ginkgo.By("Installing tidbInitializer client certificate")
					err = installTiDBInitializerCertificates(ns, tcName)
					framework.ExpectNoError(err, "failed to install tidbInitializer client certificate")

					ginkgo.By("Installing dashboard client certificate")
					err = installPDDashboardCertificates(ns, tcName)
					framework.ExpectNoError(err, "failed to install dashboard client certificate")

					ginkgo.By("Installing tidb components certificates")
					err = InstallTiDBComponentsCertificates(ns, tcName)
					framework.ExpectNoError(err, "failed to install tidb components certificates")
				}

				ginkgo.By("Deploy tidb cluster")
				utiltc.MustCreateTCWithComponentsReady(genericCli, oa, tc, 5*time.Minute, 10*time.Second)
				oldTC, err := cli.PingcapV1alpha1().TidbClusters(ns).Get(context.TODO(), tcName, metav1.GetOptions{})
				framework.ExpectNoError(err, "failed to get tc %s/%s", ns, tcName)

				ginkgo.By("Update tc to use start script v2")
				err = controller.GuaranteedUpdate(genericCli, tc, func() error {
					tc.Spec.StartScriptVersion = v1alpha1.StartScriptV2
					return nil
				})
				framework.ExpectNoError(err, "failed to start script version to v2")

				ginkgo.By(fmt.Sprintf("Wait for phase is %q", v1alpha1.UpgradePhase))
				utiltc.MustWaitForComponentPhase(cli, tc, v1alpha1.PDMemberType, v1alpha1.UpgradePhase, 3*time.Minute, time.Second*10)

				ginkgo.By("Wait for cluster is ready")
				err = oa.WaitForTidbClusterReady(tc, 15*time.Minute, 10*time.Second)
				framework.ExpectNoError(err, "failed to wait for TidbCluster %s/%s components ready", ns, tc.Name)

				ginkgo.By("Check status of components not changed")
				err = utiltc.CheckComponentStatusNotChanged(cli, oldTC)
				framework.ExpectNoError(err, "failed to check component status of tc %s/%s not changed", ns, tcName)
			})
		}

	})
})

// checkPumpStatus check there are onlineNum online pump instance running now.
func checkPumpStatus(pcli versioned.Interface, ns string, name string, onlineNum int32) error {
	var checkErr error
	err := wait.PollImmediate(5*time.Second, 10*time.Minute, func() (bool, error) {
		tc, err := pcli.PingcapV1alpha1().TidbClusters(ns).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		var onlines int32
		for _, node := range tc.Status.Pump.Members {
			if node.State == "online" {
				onlines++
			}
		}

		if onlines == onlineNum {
			return true, nil
		}

		checkErr = fmt.Errorf("failed to check %d online pumps, node status: %+v", onlineNum, tc.Status.Pump.Members)
		return false, nil
	})

	if err == wait.ErrWaitTimeout {
		err = checkErr
	}

	return err
}

// testBinlog do the flowing step to test enable and disable binlog
// 1. update tc to deploy pump and enable binlog
// 2. scale in 1 pump
// 3. scale out 1 pump
// 4. remove all pumps and disable binlog
func testBinlog(oa *tests.OperatorActions, tc *v1alpha1.TidbCluster, cli ctrlCli.Client, pcli versioned.Interface) {
	config := tcconfig.New(map[string]interface{}{})
	config.Set("storage.stop-write-at-available-space", 0)
	config.Set("addr", "0.0.0.0:8250")

	pumpSpec := &v1alpha1.PumpSpec{
		BaseImage: "pingcap/tidb-binlog",
		Replicas:  3,
		Config:    config,
		ResourceRequirements: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("1Gi"),
			},
		},
	}

	ns := tc.GetNamespace()
	name := tc.GetName()

	err := controller.GuaranteedUpdate(cli, tc, func() error {
		tc.Spec.Pump = pumpSpec
		return nil
	})
	framework.ExpectNoError(err)
	replicas := pumpSpec.Replicas
	err = checkPumpStatus(pcli, ns, name, replicas)
	framework.ExpectNoError(err)

	// start scale in and check status
	err = controller.GuaranteedUpdate(cli, tc, func() error {
		replicas--
		tc.Spec.Pump.Replicas = replicas
		return nil
	})
	framework.ExpectNoError(err)
	err = checkPumpStatus(pcli, ns, name, replicas)
	framework.ExpectNoError(err)

	// start scale out and check status
	err = controller.GuaranteedUpdate(cli, tc, func() error {
		replicas++
		tc.Spec.Pump.Replicas = replicas
		return nil
	})
	framework.ExpectNoError(err)
	err = checkPumpStatus(pcli, ns, name, replicas)
	framework.ExpectNoError(err)

	// start remove all pumps
	err = controller.GuaranteedUpdate(cli, tc, func() error {
		tc.Spec.TiDB.BinlogEnabled = pointer.BoolPtr(false)
		return nil
	})
	framework.ExpectNoError(err)
	err = oa.WaitForTidbClusterReady(tc, 30*time.Minute, 5*time.Second)
	framework.ExpectNoError(err)

	err = controller.GuaranteedUpdate(cli, tc, func() error {
		replicas = 0
		tc.Spec.Pump.Replicas = replicas
		return nil
	})
	framework.ExpectNoError(err)
	err = checkPumpStatus(pcli, ns, name, replicas)
	framework.ExpectNoError(err)
}

func validatePodSpread(pods []corev1.Pod, nodeZoneMap map[string]string, componentLabels []string, maxSkew int) error {
	zones := make([][2]int, len(componentLabels))
	for i := range pods {
		pod := &pods[i]
		framework.ExpectEqual(len(pod.Spec.TopologySpreadConstraints), 1, "Expected pod topology spread constraints are set")

		c, ok := pod.Labels[label.ComponentLabelKey]
		if !ok {
			continue
		}

		zone, ok := nodeZoneMap[pod.Spec.NodeName]
		if !ok {
			return fmt.Errorf("node %s has no zone label", pod.Spec.NodeName)
		}

		zoneId := 0
		switch zone {
		case "zone-0":
			zoneId = 0
		case "zone-1":
			zoneId = 1
		}

		for i, componentLabel := range componentLabels {
			if c == componentLabel {
				zones[i][zoneId]++
			}
		}
	}
	for i, componentLabel := range componentLabels {
		skew := zones[i][0] - zones[i][1]
		if skew < 0 {
			skew = -skew
		}
		if skew > maxSkew {
			return fmt.Errorf("%s pods are not even spread", componentLabel)
		}
	}
	return nil
}

// checkCustomLabelAndAnn check the custom set labels and ann set in `GetTidbCluster`
func checkCustomLabelAndAnn(tc *v1alpha1.TidbCluster, c clientset.Interface, expectValue string, timeout, pollInterval time.Duration) error {
	checkValue := func(k, v1 string, ms ...map[string]string) error {
		for _, m := range ms {
			v2, ok := m[k]
			if !ok {
				return fmt.Errorf("key %s doesn't exist in %v", k, m)
			}
			if v2 != v1 {
				return fmt.Errorf("value %s doesn't equal %s in map", v2, v1)
			}
		}
		return nil
	}

	var checkErr error
	err := wait.PollImmediate(pollInterval, timeout, func() (done bool, err error) {
		listOptions := metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(label.New().Instance(tc.Name).Discovery().Labels()).String(),
		}
		list, err := c.CoreV1().Pods(tc.Namespace).List(context.TODO(), listOptions)
		framework.ExpectNoError(err)
		framework.ExpectNotEqual(len(list.Items), 0, "expect discovery exists")
		for _, pod := range list.Items {
			checkErr = checkValue(fixture.ClusterCustomKey, expectValue, pod.Labels, pod.Annotations)
			if checkErr != nil {
				return false, nil
			}
		}

		if tc.Spec.TiDB != nil {
			listOptions = metav1.ListOptions{
				LabelSelector: labels.SelectorFromSet(label.New().Instance(tc.Name).Component(label.TiDBLabelVal).Labels()).String(),
			}
			list, err = c.CoreV1().Pods(tc.Namespace).List(context.TODO(), listOptions)
			framework.ExpectNoError(err)
			framework.ExpectNotEqual(len(list.Items), 0, "expect tidb pod exists")
			for _, pod := range list.Items {
				for _, k := range []string{fixture.ClusterCustomKey, fixture.ComponentCustomKey} {
					checkErr = checkValue(k, expectValue, pod.Labels, pod.Annotations)
					if checkErr != nil {
						return false, nil
					}
				}
			}

			// check service
			svcList, err := c.CoreV1().Services(tc.Namespace).List(context.TODO(), listOptions)
			framework.ExpectNoError(err)
			framework.ExpectNotEqual(len(list.Items), 0, "expect tidb svc exists")
			for _, svc := range svcList.Items {
				// skip the headless one
				if svc.Spec.ClusterIP == "None" {
					continue
				}

				checkErr = checkValue(fixture.ComponentCustomKey, expectValue, svc.Labels, svc.Annotations)
				if checkErr != nil {
					return false, nil
				}
			}
		}

		if tc.Spec.TiKV != nil {
			listOptions = metav1.ListOptions{
				LabelSelector: labels.SelectorFromSet(label.New().Instance(tc.Name).Component(label.TiKVLabelVal).Labels()).String(),
			}
			list, err = c.CoreV1().Pods(tc.Namespace).List(context.TODO(), listOptions)
			framework.ExpectNoError(err)
			framework.ExpectNotEqual(len(list.Items), 0, "expect tikv pod exists")
			for _, pod := range list.Items {
				for _, k := range []string{fixture.ClusterCustomKey, fixture.ComponentCustomKey} {
					checkErr = checkValue(k, expectValue, pod.Labels, pod.Annotations)
					if checkErr != nil {
						return false, nil
					}
				}
			}
		}

		if tc.Spec.PD != nil {
			listOptions = metav1.ListOptions{
				LabelSelector: labels.SelectorFromSet(label.New().Instance(tc.Name).Component(label.PDLabelVal).Labels()).String(),
			}
			list, err = c.CoreV1().Pods(tc.Namespace).List(context.TODO(), listOptions)
			framework.ExpectNoError(err)
			framework.ExpectNotEqual(len(list.Items), 0, "expect pd pod exists")
			for _, pod := range list.Items {
				for _, k := range []string{fixture.ClusterCustomKey, fixture.ComponentCustomKey} {
					checkErr = checkValue(k, expectValue, pod.Labels, pod.Annotations)
					if checkErr != nil {
						return false, nil
					}
				}
			}
		}

		return true, nil
	})

	if err == wait.ErrWaitTimeout {
		err = checkErr
	}
	return err
}

// checkCustomLabelAndAnn check the custom set labels and ann set in `NewTidbMonitor`
func checkMonitorCustomLabelAndAnn(tm *v1alpha1.TidbMonitor, c clientset.Interface) {
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(label.NewMonitor().Instance(tm.Name).Monitor().Labels()).String(),
	}
	pods, err := c.CoreV1().Pods(tm.Namespace).List(context.TODO(), listOptions)
	framework.ExpectNoError(err)
	framework.ExpectNotEqual(len(pods.Items), 0, "expect monitor pod exists")
	for _, pod := range pods.Items {
		_, ok := pod.Labels[fixture.ClusterCustomKey]
		framework.ExpectEqual(ok, true)
		_, ok = pod.Annotations[fixture.ClusterCustomKey]
		framework.ExpectEqual(ok, true)
	}

	svcs, err := c.CoreV1().Services(tm.Namespace).List(context.TODO(), listOptions)
	framework.ExpectNoError(err)
	framework.ExpectNotEqual(len(pods.Items), 0, "expect monitor svc exists")
	for _, svc := range svcs.Items {
		_, ok := svc.Labels[fixture.ClusterCustomKey]
		framework.ExpectEqual(ok, true)
		_, ok = svc.Annotations[fixture.ClusterCustomKey]
		framework.ExpectEqual(ok, true)

		_, ok = svc.Labels[fixture.ComponentCustomKey]
		framework.ExpectEqual(ok, true)
		_, ok = svc.Annotations[fixture.ComponentCustomKey]
		framework.ExpectEqual(ok, true)
	}
}

// checkInitializerCustomLabelAndAnn check the custom set labels and ann set in `NewTidbMonitor`
func checkInitializerCustomLabelAndAnn(ti *v1alpha1.TidbInitializer, c clientset.Interface) {
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(label.NewInitializer().Instance(ti.Name).Initializer(ti.Name).Labels()).String(),
	}
	list, err := c.CoreV1().Pods(ti.Namespace).List(context.TODO(), listOptions)
	framework.ExpectNoError(err)
	framework.ExpectNotEqual(len(list.Items), 0, "expect initializer exists")
	for _, pod := range list.Items {
		_, ok := pod.Labels[fixture.ClusterCustomKey]
		framework.ExpectEqual(ok, true)
		_, ok = pod.Annotations[fixture.ClusterCustomKey]
		framework.ExpectEqual(ok, true)
	}
}

func removeCACertFromSecret(cli ctrlCli.Client, namespace, name string) error {
	caSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name}}
	var lastErr error
	err := wait.PollImmediate(5*time.Second, 1*time.Minute, func() (bool, error) {
		err := cli.Get(context.TODO(), ctrlCli.ObjectKeyFromObject(caSecret), caSecret)
		if err != nil {
			lastErr = err
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("%s, last error: %v", err, lastErr)
	}
	delete(caSecret.Data, "ca.crt")
	return cli.Update(context.TODO(), caSecret)
}
