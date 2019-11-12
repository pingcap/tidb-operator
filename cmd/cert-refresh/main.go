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
	"github.com/pingcap/tidb-operator/pkg/cert-refresh"
	"github.com/pingcap/tidb-operator/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/component-base/logs"
	"k8s.io/klog"
	"os"
)

var (
	printVersion bool
	namespace    string
	podName      string
	image        string
	config       *cert_refresh.RefreshConfig
)

func init() {
	config = &cert_refresh.RefreshConfig{}
	flag.BoolVar(&printVersion, "V", false, "Show version and quit")
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.IntVar(&config.Timeout, "timeout", 10, "how long would initializer threshold wait")
	flag.IntVar(&config.RefreshIntervalDays, "refresh-interval-days", 7, "Number of days cert should be refreshed before expired.")
	flag.StringVar(&config.WebhookAdmissionName, "admission-webhook-name", "admission-webhook", "deployment name of webhook")
	flag.Parse()

	namespace = os.Getenv("NAMESPACE")
	if len(namespace) == 0 {
		klog.Fatalf("ENV NAMESPACE not set")
	}

	podName = os.Getenv("POD_NAME")
	if len(podName) == 0 {
		klog.Fatalf("ENV POD_NAME not set")
	}

	image = os.Getenv("IMAGE")
	if len(image) == 0 {
		klog.Fatal("ENV IMAGE not set")
	}
	config.Image = image
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
		klog.Fatalf("failed to get config: %v", err)
	}

	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed to get kubeCli: %v", err)
	}

	manager := cert_refresh.NewRefreshManager(kubeCli, config)

	err = manager.Run(podName, namespace)
	if err != nil {
		klog.Fatalf("cert-refresh Manager start running failed,%v", err)
	}
}
