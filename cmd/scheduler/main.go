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
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/scheduler/server"
	"github.com/pingcap/tidb-operator/version"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	printVersion bool
	port         int
	pdReplicas   int
	tikvReplicas int
)

func init() {
	flag.BoolVar(&printVersion, "V", false, "Show version and quit")
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.IntVar(&port, "port", 10262, "The port that the tidb scheduler's http service runs on (default 10262)")
	flag.IntVar(&pdReplicas, "pd-replicas", 3, "The pd replicas (default 3)")
	flag.IntVar(&tikvReplicas, "tikv-replicas", 3, "The tikv replicas (default 3)")
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

	cfg, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("failed to get config: %v", err)
	}
	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("failed to get kubernetes Clientset: %v", err)
	}

	go wait.Forever(func() {
		server.StartServer(kubeCli, port, int32(pdReplicas), int32(tikvReplicas))
	}, 5*time.Second)
	glog.Fatal(http.ListenAndServe(":6060", nil))
}
