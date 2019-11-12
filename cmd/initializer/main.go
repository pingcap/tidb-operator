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
	"k8s.io/klog"
	"os"
	"strings"
	"time"
)

var (
	printVersion     bool
	namespace        string
	componentList    string
	podName          string
	config           *initializer.InitializerConfig
	timeout          int
)

func init() {
	config = &initializer.InitializerConfig{}
	flag.BoolVar(&printVersion, "V", false, "Show version and quit")
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.StringVar(&componentList, "component", "all", "The component to initializer resource for")
	flag.BoolVar(&config.WebhookEnabled, "webhook-enabled", false, "switch to enable init webhook resource")
	flag.StringVar(&config.AdmissionWebhookName, "admission-webhook-name", "admission-webhook", "deployment name of webhook")
	flag.IntVar(&timeout, "timeout", 10, "how long would initializer threshold wait")
	flag.Parse()

	namespace = os.Getenv("NAMESPACE")
	if len(namespace) == 0 {
		klog.Fatalf("ENV NAMESPACE not set")
	}

	podName = os.Getenv("POD_NAME")
	if len(podName) == 0 {
		klog.Fatalf("ENV POD_NAME not set")
	}
	config.OwnerPodName = podName
	config.Timeout = time.Duration(timeout) * time.Second

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

	init := initializer.NewInitializer(kubeCli, config)

	components := strings.Split(componentList, ",")
	err = init.Run(namespace, components)
	if err != nil {
		klog.Fatalf("failed to init resources for %s: %v", components, err)
	}
}
