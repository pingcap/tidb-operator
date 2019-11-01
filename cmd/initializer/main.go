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
	"github.com/pingcap/tidb-operator/pkg/initializer"
	"github.com/pingcap/tidb-operator/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/component-base/logs"
	glog "k8s.io/klog"
	"os"
	"strconv"
)

var (
	printVersion        bool
	namespace           string
	refreshIntervalHour int
)

func init() {
	flag.BoolVar(&printVersion, "V", false, "Show version and quit")
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.Parse()

	namespace = os.Getenv("NAMESPACE")
	if len(namespace) == 0 {
		glog.Fatalf("ENV NAMESPACE not set")
	}

	refreshIntervalHourENV := os.Getenv("REFRESH_INTERVAL_HOUR")
	if len(refreshIntervalHourENV) == 0 {
		glog.Info("Default RefreshIntervalHour For initializer")
		refreshIntervalHourENV = "336"
	}

	refreshInterval, err := strconv.Atoi(refreshIntervalHourENV)
	if err != nil {
		glog.Fatalf("parse REFRESH_INTERVAL_HOUR failed")
	}
	refreshIntervalHour = refreshInterval

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
		glog.Fatalf("failed to get kubeCli: %v", err)
	}

	init := initializer.NewInitializer(kubeCli)
	err = init.Run(namespace, refreshIntervalHour)
	if err != nil {
		glog.Fatalf("failed to init secret and ValidatingWebhookConfiguration: %v", err)
	}

}
