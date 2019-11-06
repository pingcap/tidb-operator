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
	"github.com/pingcap/tidb-operator/pkg/controller"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/version"
	"github.com/pingcap/tidb-operator/pkg/webhook"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/component-base/logs"
	glog "k8s.io/klog"
)

var (
	printVersion  bool
	certFile      string
	keyFile       string
	leaseDuration = 15 * time.Second
	renewDuration = 5 * time.Second
	retryPeriod   = 3 * time.Second
	waitDuration  = 5 * time.Second
)

func init() {
	flag.BoolVar(&printVersion, "V", false, "Show version and quit")
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.StringVar(&certFile, "tlsCertFile", "/etc/webhook/certs/cert.pem", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&keyFile, "tlsKeyFile", "/etc/webhook/certs/key.pem", "File containing the x509 private key to --tlsCertFile.")
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

	hostName, err := os.Hostname()
	if err != nil {
		glog.Fatalf("failed to get hostname: %v", err)
	}

	ns := os.Getenv("NAMESPACE")
	if len(ns) == 0 {
		glog.Fatalf("env NAMESPACE is not set")
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

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeCli, controller.ResyncDuration)

	webhookServer := webhook.NewWebHookServer(kubeCli, cli, kubeInformerFactory, certFile, keyFile)
	controllerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kubeInformerFactory.Start(controllerCtx.Done())
	for v, synced := range kubeInformerFactory.WaitForCacheSync(wait.NeverStop) {
		if !synced {
			glog.Fatalf("error syncing informer for %v", v)
		}
	}
	glog.Infof("cache of informer factories sync successfully")

	if err := webhookServer.Run(); err != nil {
		glog.Errorf("stop http server %v", err)
	}

	rl := resourcelock.EndpointsLock{
		EndpointsMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      "admission-controller-webhook",
		},
		Client: kubeCli.CoreV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity:      hostName,
			EventRecorder: &record.FakeRecorder{},
		},
	}
	onStarted := func(ctx context.Context) {
		if err := webhookServer.Run(); err != nil {
			glog.Errorf("stop http server %v", err)
		}
	}

	onStopped := func() {
		glog.Fatalf("leader election lost")
	}

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
