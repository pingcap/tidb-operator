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
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/discovery/server"
	"github.com/pingcap/tidb-operator/pkg/dmapi"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/version"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"
)

var (
	printVersion bool
	port         int
	proxyPort    int
)

func init() {
	flag.BoolVar(&printVersion, "V", false, "Show version and quit")
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.IntVar(&port, "port", 10261, "The port that the tidb discovery's http service runs on (default 10261)")
	flag.IntVar(&proxyPort, "proxy-port", 10262, "The port that the tidb discovery's proxy service runs on (default 10262)")
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

	cfg, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalf("failed to get config: %v", err)
	}
	cli, err := versioned.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed to create Clientset: %v", err)
	}
	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed to get kubernetes Clientset: %v", err)
	}

	tcName := os.Getenv("TC_NAME")
	if len(tcName) < 1 {
		klog.Fatal("ENV TC_NAME is not set")
	}
	tcTls := false
	tlsEnabled := os.Getenv("TC_TLS_ENABLED")
	if tlsEnabled == strconv.FormatBool(true) {
		tcTls = true
	}
	// informers
	options := []informers.SharedInformerOption{
		informers.WithNamespace(os.Getenv("MY_POD_NAMESPACE")),
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(kubeCli, 30*time.Minute, options...)
	secretInformer := kubeInformerFactory.Core().V1().Secrets()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go kubeInformerFactory.Start(ctx.Done())

	// waiting for the shared informer's store has synced.
	cache.WaitForCacheSync(ctx.Done(), secretInformer.Informer().HasSynced)

	go wait.Forever(func() {
		addr := fmt.Sprintf("0.0.0.0:%d", port)
		klog.Infof("starting TiDB Discovery server, listening on %s", addr)
		discoveryServer := server.NewServer(pdapi.NewDefaultPDControl(secretInformer.Lister()), dmapi.NewDefaultMasterControl(secretInformer.Lister()), cli, kubeCli)
		discoveryServer.ListenAndServe(addr)
	}, 5*time.Second)
	go wait.Forever(func() {
		addr := fmt.Sprintf("0.0.0.0:%d", proxyPort)
		klog.Infof("starting TiDB Proxy server, listening on %s", addr)
		proxyServer := server.NewProxyServer(tcName, tcTls)
		proxyServer.ListenAndServe(addr)
	}, 5*time.Second)

	srv := http.Server{Addr: ":6060"}
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
	klog.Infof("tidb-discovery exited")
}
