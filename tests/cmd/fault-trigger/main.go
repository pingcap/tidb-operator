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
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/pingcap/tidb-operator/tests/pkg/util"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/api"
	"github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/manager"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/util/logs"
)

var (
	port             int
	pprofPort        int
	kubeProxyImage   string
	hostnameOverride string
)

func init() {
	flag.IntVar(&port, "port", 23332, "The port that the fault trigger's http service runs on (default 23332)")
	flag.IntVar(&pprofPort, "pprof-port", 6060, "The port that the pprof's http service runs on (default 6060)")
	flag.StringVar(&kubeProxyImage, "kube-proxy-image", "k8s.gcr.io/kube-proxy:v1.12.2", "The kube proxy image (default k8s.gcr.io/kube-proxy:v1.12.2)")
	flag.StringVar(&hostnameOverride, "hostname-override", "", "If non-empty, will use this string as identification instead of the actual hostname")

	flag.Parse()
}

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	mgr := manager.NewManager(kubeProxyImage)
	hostname, err := util.GetHostname(hostnameOverride)
	if err != nil {
		glog.Fatalf("get hostname failed, err: %v", err)
	}
	err = mgr.UpdateKubeProxyDaemonset(hostname)
	if err != nil {
		glog.Fatalf("update kube-proxy daemonset failed, err: %v", err)
	}

	server := api.NewServer(mgr, port)

	go wait.Forever(func() {
		server.StartServer()
	}, 5*time.Second)

	glog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", pprofPort), nil))
}
