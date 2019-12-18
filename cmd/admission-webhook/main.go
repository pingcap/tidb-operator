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
	"os"
	"time"

	"github.com/openshift/generic-admission-server/pkg/apiserver"
	"github.com/openshift/generic-admission-server/pkg/cmd/server"
	"k8s.io/klog"

	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/version"
	"github.com/pingcap/tidb-operator/pkg/webhook"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/logs"
)

var (
	flagset                  *flag.FlagSet
	printVersion             bool
	extraServiceAccounts     string
	evictRegionLeaderTimeout time.Duration
)

func init() {
	flagset = flag.NewFlagSet("tidb-admission-webhook", flag.ExitOnError)
	flag.BoolVar(&printVersion, "V", false, "Show version and quit")
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.StringVar(&extraServiceAccounts, "extraServiceAccounts", "", "comma-separated, extra Service Accounts the Webhook should control. The full pattern for each common service account is system:serviceaccount:<namespace>:<serviceaccount-name>")
	flag.DurationVar(&evictRegionLeaderTimeout, "evictRegionLeaderTimeout", 3*time.Minute, "TiKV evict region leader timeout period, default 3 min")
	features.DefaultFeatureGate.AddFlag(flag.CommandLine)
}

func main() {

	logs.InitLogs()
	defer logs.FlushLogs()

	if printVersion {
		version.PrintVersionInfo()
		os.Exit(0)
	}
	version.LogVersionInfo()

	ah := &webhook.AdmissionHook{
		ExtraServiceAccounts:     extraServiceAccounts,
		EvictRegionLeaderTimeout: evictRegionLeaderTimeout,
	}
	run(flagset, ah)
}

func run(flagset *flag.FlagSet, admissionHooks ...apiserver.AdmissionHook) {

	stopCh := genericapiserver.SetupSignalHandler()

	cmd := server.NewCommandStartAdmissionServer(os.Stdout, os.Stderr, stopCh, admissionHooks...)

	// Add admission hook flags
	cmd.Flags().AddGoFlagSet(flagset)

	// Flags for glog
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	// Fix glog printing "Error: logging before flag.Parse"
	flag.CommandLine.Parse([]string{})

	if err := cmd.Execute(); err != nil {
		klog.Fatal(err)
	}
}
