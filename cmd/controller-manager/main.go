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

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/controller/tidbcluster"
	"github.com/pingcap/tidb-operator/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/util/logs"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
)

var (
	printVersion       bool
	workers            int
	autoFailover       bool
	pdFailoverPeriod   time.Duration
	tikvFailoverPeriod time.Duration
	tidbFailoverPeriod time.Duration
	leaseDuration      = 15 * time.Second
	renewDuration      = 5 * time.Second
	retryPeriod        = 3 * time.Second
	resyncDuration     = 30 * time.Second
	waitDuration       = 5 * time.Second
)

func init() {
	flag.BoolVar(&printVersion, "V", false, "Show version and quit")
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.IntVar(&workers, "workers", 5, "The number of workers that are allowed to sync concurrently. Larger number = more responsive management, but more CPU (and network) load")
	flag.BoolVar(&controller.ClusterScoped, "cluster-scoped", true, "Whether tidb-operator should manage kubernetes cluster wide TiDB Clusters")
	flag.StringVar(&controller.DefaultStorageClassName, "default-storage-class-name", "standard", "Default storage class name")
	flag.BoolVar(&autoFailover, "auto-failover", false, "Auto failover")
	flag.DurationVar(&pdFailoverPeriod, "pd-failover-period", time.Duration(5*time.Minute), "PD failover period default(5m)")
	flag.DurationVar(&tikvFailoverPeriod, "tikv-failover-period", time.Duration(5*time.Minute), "TiKV failover period default(5m)")
	flag.DurationVar(&tidbFailoverPeriod, "tidb-failover-period", time.Duration(5*time.Minute), "TiDB failover period")

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

	hostName, err := os.Hostname()
	if err != nil {
		glog.Fatalf("failed to get hostname: %v", err)
	}

	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		glog.Fatal("NAMESPACE environment variable not set")
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("failed to get config: %v", err)
	}

	cli, err := versioned.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("failed to create Clientset: %v", err)
	}
	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("failed to get kubernetes Clientset: %v", err)
	}
	operatorImage := os.Getenv("MY_IMAGE")
	if operatorImage == "" {
		glog.Fatal("MY_IMAGE env variable not set")
	}

	var informerFactory informers.SharedInformerFactory
	var kubeInformerFactory kubeinformers.SharedInformerFactory
	if controller.ClusterScoped {
		informerFactory = informers.NewSharedInformerFactory(cli, resyncDuration)
		kubeInformerFactory = kubeinformers.NewSharedInformerFactory(kubeCli, resyncDuration)
	} else {
		options := []informers.SharedInformerOption{
			informers.WithNamespace(ns),
		}
		informerFactory = informers.NewSharedInformerFactoryWithOptions(cli, resyncDuration, options...)

		kubeoptions := []kubeinformers.SharedInformerOption{
			kubeinformers.WithNamespace(ns),
		}
		kubeInformerFactory = kubeinformers.NewSharedInformerFactoryWithOptions(kubeCli, resyncDuration, kubeoptions...)
	}

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

	tcController := tidbcluster.NewController(kubeCli, cli, informerFactory, kubeInformerFactory, autoFailover, operatorImage, pdFailoverPeriod, tikvFailoverPeriod, tidbFailoverPeriod)
	controllerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go informerFactory.Start(controllerCtx.Done())
	go kubeInformerFactory.Start(controllerCtx.Done())

	onStarted := func(ctx context.Context) {
		tcController.Run(workers, ctx.Done())
	}
	onStopped := func() {
		glog.Fatalf("leader election lost")
	}

	// leader election for multiple tidb-cloud-manager
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

	glog.Fatal(http.ListenAndServe(":6060", nil))
}
