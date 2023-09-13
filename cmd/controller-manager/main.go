// Copyright 2018 PingCAP, Inc.
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

package main

import (
	"context"
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"reflect"
	"syscall"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/controller/autoscaler"
	"github.com/pingcap/tidb-operator/pkg/controller/backup"
	"github.com/pingcap/tidb-operator/pkg/controller/backupschedule"
	"github.com/pingcap/tidb-operator/pkg/controller/dmcluster"
	"github.com/pingcap/tidb-operator/pkg/controller/restore"
	"github.com/pingcap/tidb-operator/pkg/controller/tidbcluster"
	"github.com/pingcap/tidb-operator/pkg/controller/tidbdashboard"
	"github.com/pingcap/tidb-operator/pkg/controller/tidbinitializer"
	"github.com/pingcap/tidb-operator/pkg/controller/tidbmonitor"
	"github.com/pingcap/tidb-operator/pkg/controller/tidbngmonitoring"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/metrics"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	"github.com/pingcap/tidb-operator/pkg/upgrader"
	"github.com/pingcap/tidb-operator/pkg/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {
	var cfg *rest.Config
	klog.InitFlags(nil)
	cliCfg := controller.DefaultCLIConfig()
	cliCfg.AddFlag(flag.CommandLine)
	features.DefaultFeatureGate.AddFlag(flag.CommandLine)
	flag.Parse()

	if cliCfg.PrintVersion {
		version.PrintVersionInfo()
		os.Exit(0)
	}

	logs.InitLogs()
	defer logs.FlushLogs()

	version.LogVersionInfo()
	flag.VisitAll(func(flag *flag.Flag) {
		klog.V(1).Infof("FLAG: --%s=%q", flag.Name, flag.Value)
	})

	logCustomPorts()

	hostName, err := os.Hostname()
	if err != nil {
		klog.Fatalf("failed to get hostname: %v", err)
	}

	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		klog.Fatal("NAMESPACE environment variable not set")
	}

	helmRelease := os.Getenv("HELM_RELEASE")
	if helmRelease == "" {
		klog.Info("HELM_RELEASE environment variable not set")
	}

	kubconfig := os.Getenv("KUBECONFIG")
	if kubconfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubconfig)
	} else {
		cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		klog.Fatalf("failed to get config: %v", err)
	}

	// If they are zero, the created client will use the default values: 5, 10.
	cfg.QPS = float32(cliCfg.KubeClientQPS)
	cfg.Burst = cliCfg.KubeClientBurst

	cli, err := versioned.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed to create Clientset: %v", err)
	}
	var kubeCli kubernetes.Interface
	kubeCli, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed to get kubernetes Clientset: %v", err)
	}
	asCli, err := asclientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed to get advanced-statefulset Clientset: %v", err)
	}
	// TODO: optimize the read of genericCli with the shared cache
	genericCli, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		klog.Fatalf("failed to get the generic kube-apiserver client: %v", err)
	}

	// note that kubeCli here must not be the hijacked one
	var operatorUpgrader upgrader.Interface
	if cliCfg.ClusterScoped {
		operatorUpgrader = upgrader.NewUpgrader(kubeCli, cli, asCli, metav1.NamespaceAll)
	} else {
		operatorUpgrader = upgrader.NewUpgrader(kubeCli, cli, asCli, ns)
	}

	if features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		// If AdvancedStatefulSet is enabled, we hijack the Kubernetes client to use
		// AdvancedStatefulSet.
		kubeCli = helper.NewHijackClient(kubeCli, asCli)
	}

	deps, err := controller.NewDependencies(ns, cliCfg, cli, kubeCli, genericCli)
	if err != nil {
		klog.Fatalf("failed to create Dependencies: %s", err)
	}

	onStarted := func(ctx context.Context) {
		// Upgrade before running any controller logic. If it fails, we wait
		// for process supervisor to restart it again.
		if err := operatorUpgrader.Upgrade(); err != nil {
			klog.Fatalf("failed to upgrade: %v", err)
		}

		// Define some nested types to simplify the codebase
		type Controller interface {
			Run(int, <-chan struct{})
			Name() string
		}
		type InformerFactory interface {
			Start(stopCh <-chan struct{})
			WaitForCacheSync(stopCh <-chan struct{}) map[reflect.Type]bool
		}

		initMetrics := func(c Controller) {
			metrics.ActiveWorkers.WithLabelValues(c.Name()).Set(0)
		}

		// Initialize all controllers
		controllers := []Controller{
			tidbcluster.NewController(deps),
			tidbcluster.NewPodController(deps),
			dmcluster.NewController(deps),
			backup.NewController(deps),
			restore.NewController(deps),
			backupschedule.NewController(deps),
			tidbinitializer.NewController(deps),
			tidbmonitor.NewController(deps),
			tidbngmonitoring.NewController(deps),
			tidbdashboard.NewController(deps),
		}
		if features.DefaultFeatureGate.Enabled(features.AutoScaling) {
			controllers = append(controllers, autoscaler.NewController(deps))
		}

		// Start informer factories after all controllers are initialized.
		informerFactories := []InformerFactory{
			deps.InformerFactory,
			deps.KubeInformerFactory,
			deps.LabelFilterKubeInformerFactory,
		}
		for _, f := range informerFactories {
			f.Start(ctx.Done())
			for v, synced := range f.WaitForCacheSync(wait.NeverStop) {
				if !synced {
					klog.Fatalf("error syncing informer for %v", v)
				}
			}
		}
		klog.Info("cache of informer factories sync successfully")

		// Start syncLoop for all controllers
		for _, controller := range controllers {
			c := controller
			initMetrics(c)
			go wait.Forever(func() { c.Run(cliCfg.Workers, ctx.Done()) }, cliCfg.WaitDuration)
		}
	}
	onStopped := func() {
		klog.Fatal("leader election lost")
	}

	endPointsName := "tidb-controller-manager"
	if helmRelease != "" {
		endPointsName += "-" + helmRelease
	}
	// leader election for multiple tidb-controller-manager instances
	go wait.Forever(func() {
		leaderelection.RunOrDie(context.TODO(), leaderelection.LeaderElectionConfig{
			Lock: &resourcelock.EndpointsLock{
				EndpointsMeta: metav1.ObjectMeta{
					Namespace: ns,
					Name:      endPointsName,
				},
				Client: kubeCli.CoreV1(),
				LockConfig: resourcelock.ResourceLockConfig{
					Identity:      hostName,
					EventRecorder: &record.FakeRecorder{},
				},
			},
			LeaseDuration: cliCfg.LeaseDuration,
			RenewDeadline: cliCfg.RenewDeadline,
			RetryPeriod:   cliCfg.RetryPeriod,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: onStarted,
				OnStoppedLeading: onStopped,
			},
		})
	}, cliCfg.WaitDuration)

	srv := createHTTPServer()
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	go func() {
		sig := <-sc
		klog.Infof("got signal %s to exit", sig)
		if err2 := srv.Shutdown(context.Background()); err2 != nil {
			klog.Fatal("fail to shutdown the HTTP server", err2)
		}
	}()

	if err = srv.ListenAndServe(); err != http.ErrServerClosed {
		klog.Fatal(err)
	}
	klog.Infof("tidb-controller-manager exited")
}

func createHTTPServer() *http.Server {
	serverMux := http.NewServeMux()
	// HTTP path for pprof
	serverMux.Handle("/", http.DefaultServeMux)
	// HTTP path for prometheus.
	serverMux.Handle("/metrics", promhttp.Handler())

	return &http.Server{
		Addr:    ":6060",
		Handler: serverMux,
	}
}

func logCustomPorts() {
	if v1alpha1.DefaultTiDBServerPort != 4000 ||
		v1alpha1.DefaultTiDBStatusPort != 10080 ||
		v1alpha1.DefaultPDClientPort != 2379 ||
		v1alpha1.DefaultPDPeerPort != 2380 ||
		v1alpha1.DefaultTiKVServerPort != 20160 ||
		v1alpha1.DefaultTiKVStatusPort != 20180 ||
		v1alpha1.DefaultTiFlashTcpPort != 9000 ||
		v1alpha1.DefaultTiFlashHttpPort != 8123 ||
		v1alpha1.DefaultTiFlashFlashPort != 3930 ||
		v1alpha1.DefaultTiFlashProxyPort != 20170 ||
		v1alpha1.DefaultTiFlashMetricsPort != 8234 ||
		v1alpha1.DefaultTiFlashProxyStatusPort != 20292 ||
		v1alpha1.DefaultTiFlashInternalPort != 9009 ||
		v1alpha1.DefaultPumpPort != 8250 ||
		v1alpha1.DefaultDrainerPort != 8249 ||
		v1alpha1.DefaultTiCDCPort != 8301 {
		klog.Infof("running TiDB Operator with custom ports: %#v", CustomPorts{
			TiDBServerPort: v1alpha1.DefaultTiDBServerPort,
			TiDBStatusPort: v1alpha1.DefaultTiDBStatusPort,

			PDClientPort: v1alpha1.DefaultPDClientPort,
			PDPeerPort:   v1alpha1.DefaultPDPeerPort,

			TiKVServerPort: v1alpha1.DefaultTiKVServerPort,
			TiKVStatusPort: v1alpha1.DefaultTiKVStatusPort,

			TiFlashTcpPort:         v1alpha1.DefaultTiFlashTcpPort,
			TiFlashHttpPort:        v1alpha1.DefaultTiFlashHttpPort,
			TiFlashFlashPort:       v1alpha1.DefaultTiFlashFlashPort,
			TiFlashProxyPort:       v1alpha1.DefaultTiFlashProxyPort,
			TiFlashMetricsPort:     v1alpha1.DefaultTiFlashMetricsPort,
			TiFlashProxyStatusPort: v1alpha1.DefaultTiFlashProxyStatusPort,
			TiFlashInternalPort:    v1alpha1.DefaultTiFlashInternalPort,

			PumpPort: v1alpha1.DefaultPumpPort,

			DrainerPort: v1alpha1.DefaultDrainerPort,

			TiCDCPort: v1alpha1.DefaultTiCDCPort,
		})
	}
}

type CustomPorts struct {
	TiDBServerPort int32
	TiDBStatusPort int32

	PDClientPort int32
	PDPeerPort   int32

	TiKVServerPort int32
	TiKVStatusPort int32

	TiFlashTcpPort         int32
	TiFlashHttpPort        int32
	TiFlashFlashPort       int32
	TiFlashProxyPort       int32
	TiFlashMetricsPort     int32
	TiFlashProxyStatusPort int32
	TiFlashInternalPort    int32

	PumpPort int32

	DrainerPort int32

	TiCDCPort int32
}
