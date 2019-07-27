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
	"github.com/pingcap/tidb-operator/pkg/util"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/scheduler/server"
	"github.com/pingcap/tidb-operator/version"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/client-go/kubernetes"
)

var (
	printVersion bool
	port         int
	kubeConfig   string
	masterURL    string
)

func init() {
	flag.BoolVar(&printVersion, "V", false, "Show version and quit")
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.IntVar(&port, "port", 10262, "The port that the tidb scheduler's http service runs on (default 10262)")
	flag.StringVar(&kubeConfig, "kubeconfig", "", "The path of kubeconfig file, only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", `The url of the Kubernetes API server,will overrides 
		any value in kubeconfig, only required if out-of-cluster.`)
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

	cfg := util.NewClusterConfig(masterURL, kubeConfig)
	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("failed to get kubernetes Clientset: %v", err)
	}
	cli, err := versioned.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("failed to create Clientset: %v", err)
	}

	go wait.Forever(func() {
		server.StartServer(kubeCli, cli, port)
	}, 5*time.Second)
	glog.Fatal(http.ListenAndServe(":6060", nil))
}
