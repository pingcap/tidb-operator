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

package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1alpha1/helper"
	asclientset "github.com/pingcap/advanced-statefulset/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/version"
	"github.com/pingcap/tidb-operator/pkg/webhook"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/component-base/logs"
	glog "k8s.io/klog"
)

var (
	printVersion bool
	certFile     string
	keyFile      string
)

func init() {
	flag.BoolVar(&printVersion, "V", false, "Show version and quit")
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.StringVar(&certFile, "tlsCertFile", "/etc/webhook/certs/cert.pem", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&keyFile, "tlsKeyFile", "/etc/webhook/certs/key.pem", "File containing the x509 private key to --tlsCertFile.")
	features.DefaultFeatureGate.AddFlag(flag.CommandLine)

	flag.Parse()
}

func main() {

	logs.InitLogs()
	defer logs.FlushLogs()

	if printVersion {
		version.PrintVersionInfo()
		os.Exit(0)
	}
	version.LogVersionInfo()

	cfg, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("failed to get config: %v", err)
	}

	cli, err := versioned.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("failed to create Clientset: %v", err)
	}

	var kubeCli kubernetes.Interface
	kubeCli, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("failed to get kubernetes Clientset: %v", err)
	}

	asCli, err := asclientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("failed to get advanced-statefulset Clientset: %v", err)
	}

	if features.DefaultFeatureGate.Enabled(features.AdvancedStatefulSet) {
		// If AdvancedStatefulSet is enabled, we hijack the Kubernetes client to use
		// AdvancedStatefulSet.
		kubeCli = helper.NewHijackClient(kubeCli, asCli)
	}

	informerFactory := informers.NewSharedInformerFactory(cli, controller.ResyncDuration)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeCli, controller.ResyncDuration)

	webhookServer := webhook.NewWebHookServer(kubeCli, cli, informerFactory, kubeInformerFactory, certFile, keyFile)
	controllerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	informerFactory.Start(controllerCtx.Done())
	kubeInformerFactory.Start(controllerCtx.Done())

	// Wait for all started informers' cache were synced.
	for v, synced := range informerFactory.WaitForCacheSync(wait.NeverStop) {
		if !synced {
			glog.Fatalf("error syncing informer for %v", v)
		}
	}
	for v, synced := range kubeInformerFactory.WaitForCacheSync(wait.NeverStop) {
		if !synced {
			glog.Fatalf("error syncing informer for %v", v)
		}
	}
	glog.Infof("cache of informer factories sync successfully")

	if err := webhookServer.Run(); err != nil {
		glog.Fatalf("stop http server %v", err)
	}

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs

		// Graceful shutdown the server
		if err := webhookServer.Shutdown(); err != nil {
			glog.Errorf("fail to shutdown server %v", err)
		}

		done <- true
	}()

	<-done

	glog.Infof("webhook server terminate safely.")
}
