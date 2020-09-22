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
	"reflect"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller/autoscaler"
	"github.com/pingcap/tidb-operator/pkg/controller/backup"
	"github.com/pingcap/tidb-operator/pkg/controller/backupschedule"
	"github.com/pingcap/tidb-operator/pkg/controller/dmcluster"
	"github.com/pingcap/tidb-operator/pkg/controller/periodicity"
	"github.com/pingcap/tidb-operator/pkg/controller/restore"
	"github.com/pingcap/tidb-operator/pkg/controller/tidbcluster"
	"github.com/pingcap/tidb-operator/pkg/controller/tidbinitializer"
	"github.com/pingcap/tidb-operator/pkg/controller/tidbmonitor"
	"github.com/pingcap/tidb-operator/pkg/deps"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	"github.com/pingcap/tidb-operator/pkg/upgrader"
	"github.com/pingcap/tidb-operator/pkg/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/component-base/logs"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {
	cliCfg := deps.DefaultCLIConfig()
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

	dependencies := deps.NewDependencies(ns, cliCfg, cli, kubeCli, genericCli)
	controllerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	onStarted := func(ctx context.Context) {
		// Upgrade before running any controller logic. If it fails, we wait
		// for process supervisor to restart it again.
		if err := operatorUpgrader.Upgrade(); err != nil {
			klog.Fatalf("failed to upgrade: %v", err)
		}

		controllers := []interface{ Run(int, <-chan struct{}) }{
			tidbcluster.NewController(dependencies),
			dmcluster.NewController(dependencies),
			backup.NewController(dependencies),
			restore.NewController(dependencies),
			backupschedule.NewController(dependencies),
			tidbinitializer.NewController(dependencies),
			tidbmonitor.NewController(dependencies),
		}
		if cliCfg.PodWebhookEnabled {
			controllers = append(controllers, periodicity.NewController(dependencies))
		}
		if features.DefaultFeatureGate.Enabled(features.AutoScaling) {
			controllers = append(controllers, autoscaler.NewController(dependencies))
		}

		// Start informer factories after all controller are initialized.
		informerFactories := []interface {
			Start(stopCh <-chan struct{})
			WaitForCacheSync(stopCh <-chan struct{}) map[reflect.Type]bool
		}{
			dependencies.InformerFactory, dependencies.KubeInformerFactory,
		}
		for _, factory := range informerFactories {
			factory.Start(ctx.Done())
			for v, synced := range factory.WaitForCacheSync(wait.NeverStop) {
				if !synced {
					klog.Fatalf("error syncing informer for %v", v)
				}
			}
		}
		klog.Infof("cache of informer factories sync successfully")

		for _, controller := range controllers {
			go wait.Forever(func() { controller.Run(cliCfg.Workers, ctx.Done()) }, cliCfg.WaitDuration)
		}
	}
	onStopped := func() {
		klog.Fatalf("leader election lost")
	}

	// leader election for multiple tidb-controller-manager instances
	go wait.Forever(func() {
		leaderelection.RunOrDie(controllerCtx, leaderelection.LeaderElectionConfig{
			Lock: &resourcelock.EndpointsLock{
				EndpointsMeta: metav1.ObjectMeta{
					Namespace: ns,
					Name:      "tidb-controller-manager",
				},
				Client: kubeCli.CoreV1(),
				LockConfig: resourcelock.ResourceLockConfig{
					Identity:      hostName,
					EventRecorder: &record.FakeRecorder{},
				},
			},
			LeaseDuration: cliCfg.LeaseDuration,
			RenewDeadline: cliCfg.RenewDuration,
			RetryPeriod:   cliCfg.RetryPeriod,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: onStarted,
				OnStoppedLeading: onStopped,
			},
		})
	}, cliCfg.WaitDuration)

	klog.Fatal(http.ListenAndServe(":6060", nil))
}
