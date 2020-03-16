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

	"github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/api"
	"github.com/pingcap/tidb-operator/tests/pkg/fault-trigger/manager"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/component-base/logs"
	"k8s.io/klog"
)

var (
	port      int
	pprofPort int
	vmManager string
)

func init() {
	flag.IntVar(&port, "port", 23332, "The port that the fault trigger's http service runs on (default 23332)")
	flag.IntVar(&pprofPort, "pprof-port", 6060, "The port that the pprof's http service runs on (default 6060)")
	flag.StringVar(&vmManager, "vm-manager", "virsh", "the vm manager, virsh/qm (default virsh)")

	flag.Parse()
}

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	mgr := manager.NewManager(vmManager)
	server := api.NewServer(mgr, port)

	go wait.Forever(func() {
		server.StartServer()
	}, 5*time.Second)

	klog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", pprofPort), nil))
}
