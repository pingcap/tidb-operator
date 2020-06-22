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
	"time"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/controller/autoscaler"
	"github.com/pingcap/tidb-operator/pkg/controller/backup"
	"github.com/pingcap/tidb-operator/pkg/controller/backupschedule"
	"github.com/pingcap/tidb-operator/pkg/controller/periodicity"
	"github.com/pingcap/tidb-operator/pkg/controller/restore"
	"github.com/pingcap/tidb-operator/pkg/controller/tidbcluster"
	"github.com/pingcap/tidb-operator/pkg/controller/tidbgroup"
	"github.com/pingcap/tidb-operator/pkg/controller/tidbinitializer"
	"github.com/pingcap/tidb-operator/pkg/controller/tidbmonitor"
	"github.com/pingcap/tidb-operator/pkg/controller/tikvgroup"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	"github.com/pingcap/tidb-operator/pkg/upgrader"
	"github.com/pingcap/tidb-operator/pkg/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/component-base/logs"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	printVersion          bool
	workers               int
	autoFailover          bool
	pdFailoverPeriod      time.Duration
	tikvFailoverPeriod    time.Duration
	tidbFailoverPeriod    time.Duration
	tiflashFailoverPeriod time.Duration
	leaseDuration         = 15 * time.Second
	renewDuration         = 5 * time.Second
	retryPeriod           = 3 * time.Second
	waitDuration          = 5 * time.Second
)

func init() {
	flag.BoolVar(&printVersion, "V", false, "Show version and quit")
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.IntVar(&workers, "workers", 5, "The number of workers that are allowed to sync concurrently. Larger number = more responsive management, but more CPU (and network) load")
	flag.BoolVar(&controller.ClusterScoped, "cluster-scoped", true, "Whether tidb-operator should manage kubernetes cluster wide TiDB Clusters")
	flag.BoolVar(&autoFailover, "auto-failover", true, "Auto failover")
	flag.DurationVar(&pdFailoverPeriod, "pd-failover-period", time.Duration(5*time.Minute), "PD failover period default(5m)")
	flag.DurationVar(&tikvFailoverPeriod, "tikv-failover-period", time.Duration(5*time.Minute), "TiKV failover period default(5m)")
	flag.DurationVar(&tiflashFailoverPeriod, "tiflash-failover-period", time.Duration(5*time.Minute), "TiFlash failover period default(5m)")
	flag.DurationVar(&tidbFailoverPeriod, "tidb-failover-period", time.Duration(5*time.Minute), "TiDB failover period")
	flag.DurationVar(&controller.ResyncDuration, "resync-duration", time.Duration(30*time.Second), "Resync time of informer")
	flag.BoolVar(&controller.TestMode, "test-mode", false, "whether tidb-operator run in test mode")
	flag.StringVar(&controller.TidbBackupManagerImage, "tidb-backup-manager-image", "pingcap/tidb-backup-manager:latest", "The image of backup manager tool")
	// TODO: actually we just want to use the same image with tidb-controller-manager, but DownwardAPI cannot get image ID, see if there is any better solution
	flag.StringVar(&controller.TidbDiscoveryImage, "tidb-discovery-image", "pingcap/tidb-operator:latest", "The image of the tidb discovery service")
	flag.BoolVar(&controller.PodWebhookEnabled, "pod-webhook-enabled", false, "Whether Pod admission webhook is enabled")
	features.DefaultFeatureGate.AddFlag(flag.CommandLine)

	flag.Parse()
}

func main() {
	if printVersion {
		version.PrintVersionInfo()
		os.Exit(0)
	}
	version.LogVersionInfo()

	logs.InitLogs()
	defer logs.FlushLogs()

	flag.CommandLine.VisitAll(func(flag *flag.Flag) {
		klog.V(1).Infof("FLAG: --%s=%q", flag.Name, flag.Value)
	})

	hostName, err := os.Hostname()
	if err != nil {
		klog.Fatalf("failed to get hostname: %v", err)
	}

	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		klog.Fatal("NAMESPACE environment variable not set")
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalf("failed to get config: %v", err)
	}

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
	if controller.ClusterScoped {
		operatorUpgrader = upgrader.NewUpgrader(kubeCli, cli, asCli, metav1.NamespaceAll)
	} else {
		operatorUpgrader = upgrader.NewUpgrader(kubeCli, cli, asCli, ns)
	}

	if features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		// If AdvancedStatefulSet is enabled, we hijack the Kubernetes client to use
		// AdvancedStatefulSet.
		kubeCli = helper.NewHijackClient(kubeCli, asCli)
	}

	var informerFactory informers.SharedInformerFactory
	var kubeInformerFactory kubeinformers.SharedInformerFactory
	var options []informers.SharedInformerOption
	var kubeoptions []kubeinformers.SharedInformerOption
	if !controller.ClusterScoped {
		options = append(options, informers.WithNamespace(ns))
		kubeoptions = append(kubeoptions, kubeinformers.WithNamespace(ns))
	}
	informerFactory = informers.NewSharedInformerFactoryWithOptions(cli, controller.ResyncDuration, options...)
	kubeInformerFactory = kubeinformers.NewSharedInformerFactoryWithOptions(kubeCli, controller.ResyncDuration, kubeoptions...)

	rl := resourcelock.EndpointsLock{
		EndpointsMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      "tidb-controller-manager",
		},
		Client: kubeCli.CoreV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity:      hostName,
			EventRecorder: &record.FakeRecorder{},
		},
	}

	controllerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	onStarted := func(ctx context.Context) {
		// Upgrade before running any controller logic. If it fails, we wait
		// for process supervisor to restart it again.
		if err := operatorUpgrader.Upgrade(); err != nil {
			klog.Fatalf("failed to upgrade: %v", err)
		}

		tcController := tidbcluster.NewController(kubeCli, cli, genericCli, informerFactory, kubeInformerFactory, autoFailover, pdFailoverPeriod, tikvFailoverPeriod, tidbFailoverPeriod, tiflashFailoverPeriod)
		backupController := backup.NewController(kubeCli, cli, informerFactory, kubeInformerFactory)
		restoreController := restore.NewController(kubeCli, cli, informerFactory, kubeInformerFactory)
		bsController := backupschedule.NewController(kubeCli, cli, informerFactory, kubeInformerFactory)
		tidbInitController := tidbinitializer.NewController(kubeCli, cli, genericCli, informerFactory, kubeInformerFactory)
		tidbMonitorController := tidbmonitor.NewController(kubeCli, genericCli, cli, informerFactory, kubeInformerFactory)
		tidbGroupController := tidbgroup.NewController(kubeCli, cli, informerFactory, kubeInformerFactory)
		tikvGroupController := tikvgroup.NewController(kubeCli, cli, informerFactory, kubeInformerFactory)

		var periodicityController *periodicity.Controller
		if controller.PodWebhookEnabled {
			periodicityController = periodicity.NewController(kubeCli, informerFactory, kubeInformerFactory)
		}

		var autoScalerController *autoscaler.Controller
		if features.DefaultFeatureGate.Enabled(features.AutoScaling) {
			autoScalerController = autoscaler.NewController(kubeCli, cli, informerFactory, kubeInformerFactory)
		}
		// Start informer factories after all controller are initialized.
		informerFactory.Start(ctx.Done())
		kubeInformerFactory.Start(ctx.Done())

		// Wait for all started informers' cache were synced.
		for v, synced := range informerFactory.WaitForCacheSync(wait.NeverStop) {
			if !synced {
				klog.Fatalf("error syncing informer for %v", v)
			}
		}
		for v, synced := range kubeInformerFactory.WaitForCacheSync(wait.NeverStop) {
			if !synced {
				klog.Fatalf("error syncing informer for %v", v)
			}
		}
		klog.Infof("cache of informer factories sync successfully")

		go wait.Forever(func() { backupController.Run(workers, ctx.Done()) }, waitDuration)
		go wait.Forever(func() { restoreController.Run(workers, ctx.Done()) }, waitDuration)
		go wait.Forever(func() { bsController.Run(workers, ctx.Done()) }, waitDuration)
		go wait.Forever(func() { tidbInitController.Run(workers, ctx.Done()) }, waitDuration)
		go wait.Forever(func() { tidbMonitorController.Run(workers, ctx.Done()) }, waitDuration)
		go wait.Forever(func() { tidbGroupController.Run(workers, ctx.Done()) }, waitDuration)
		go wait.Forever(func() { tikvGroupController.Run(workers, ctx.Done()) }, waitDuration)

		if controller.PodWebhookEnabled {
			go wait.Forever(func() { periodicityController.Run(ctx.Done()) }, waitDuration)
		}
		if features.DefaultFeatureGate.Enabled(features.AutoScaling) {
			go wait.Forever(func() { autoScalerController.Run(workers, ctx.Done()) }, waitDuration)
		}
		wait.Forever(func() { tcController.Run(workers, ctx.Done()) }, waitDuration)
	}
	onStopped := func() {
		klog.Fatalf("leader election lost")
	}

	// leader election for multiple tidb-controller-manager instances
	go wait.Forever(func() {
		leaderelection.RunOrDie(controllerCtx, leaderelection.LeaderElectionConfig{
			Lock:          &rl,
			LeaseDuration: leaseDuration,
			RenewDeadline: renewDuration,
			RetryPeriod:   retryPeriod,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: onStarted,
				OnStoppedLeading: onStopped,
			},
		})
	}, waitDuration)

	klog.Fatal(http.ListenAndServe(":6060", nil))
}
