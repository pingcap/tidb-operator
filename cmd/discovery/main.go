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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/component-base/logs"
	"k8s.io/klog"
)

var (
	printVersion bool
	port         int
	proxyPort    int
	flagSet      *flag.FlagSet
)

func init() {
	flagSet = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagSet.BoolVar(&printVersion, "V", false, "Show version and quit")
	flagSet.BoolVar(&printVersion, "version", false, "Show version and quit")
	flagSet.IntVar(&port, "port", 10261, "The port that the tidb discovery's http service runs on (default 10261)")
	flagSet.IntVar(&proxyPort, "proxy-port", 10262, "The port that the tidb discovery's proxy service runs on (default 10262)")

	i := func() int {
		for i, f := range os.Args {
			if f == "--" {
				return i
			}
		}
		return 0
	}()

	flagSet.Parse(os.Args[i+1:])
}

func main() {
	if printVersion {
		version.PrintVersionInfo()
		return
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
	tcTLS := false
	tlsEnabled := os.Getenv("TC_TLS_ENABLED")
	if tlsEnabled == strconv.FormatBool(true) {
		tcTLS = true
	}

	go wait.Forever(func() {
		addr := fmt.Sprintf("0.0.0.0:%d", port)
		klog.Infof("starting TiDB Discovery server, listening on %s", addr)
		discoveryServer := server.NewServer(pdapi.NewDefaultPDControl(kubeCli), dmapi.NewDefaultMasterControl(kubeCli), cli, kubeCli)
		discoveryServer.ListenAndServe(addr)
	}, 5*time.Second)
	go wait.Forever(func() {
		addr := fmt.Sprintf("0.0.0.0:%d", proxyPort)
		klog.Infof("starting TiDB Proxy server, listening on %s", addr)
		proxyServer := server.NewProxyServer(tcName, tcTLS)
		proxyServer.ListenAndServe(addr)
	}, 5*time.Second)

	srv := http.Server{Addr: ":6060"}
	closeSignalChan := make(chan os.Signal, 1)
	signal.Notify(
		closeSignalChan,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	// gracefully shutdown the server
	go func() {
		sig := <-closeSignalChan
		klog.V(1).Infof("got %s signal, exit.\n", sig)
		if err := srv.Shutdown(context.Background()); err != nil {
			klog.Fatal(err)
		}
		close(closeSignalChan)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		klog.Fatal(err)
	}
}
