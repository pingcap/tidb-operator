// Copyright 2019. PingCAP, Inc.
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
	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"k8s.io/apiserver/pkg/util/logs"

	"flag"
	"os"
	"time"
)

const (
	timeout = 5 * time.Second
)

func init() {
	flag.Parse()
}

// wait-for-pd waits for 1 PD to be running
func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	oneSecond, err := time.ParseDuration("1s")
	if err != nil {
		glog.Fatalf("Error parsing time %v", err)
	}

	tcName := os.Getenv("CLUSTER_NAME")
	if tcName == "" {
		glog.Fatalf("Expected CLUSTER_NAME env variable to be set")
	}
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		glog.Fatalf("Expected NAMESPACE env variable to be set")
	}

	pdClient := pdapi.NewPDClient(pdapi.PdClientURL(pdapi.Namespace(namespace), tcName), timeout)

	for {
		membersInfo, err := pdClient.GetMembers()
		if err != nil {
			glog.Errorf("Error using pdClient to get members %v", err)
		} else if membersInfo.Leader != nil {
			glog.Infof("Found a PD member. Exiting now.")
			break
		}
		time.Sleep(oneSecond)
	}
}
