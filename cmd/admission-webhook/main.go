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
	"runtime"
	"time"

	"github.com/openshift/generic-admission-server/pkg/apiserver"
	"github.com/openshift/generic-admission-server/pkg/cmd/server"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/version"
	"github.com/pingcap/tidb-operator/pkg/webhook/statefulset"
	"github.com/pingcap/tidb-operator/pkg/webhook/strategy"

	// Enable FIPS when necessary
	_ "github.com/pingcap/tidb-operator/pkg/fips"
)

var (
	printVersion         bool
	extraServiceAccounts string
	minResyncDuration    time.Duration
)

func init() {
	klog.InitFlags(nil)
	flag.CommandLine.Init(os.Args[0], flag.ContinueOnError)
	// Define the flag "secure-port" to avoid the `flag.Parse()` reporting error
	// TODO: remove this flag after we don't use the lib "github.com/openshift/generic-admission-server"
	flag.Int("secure-port", 6443, "The port on which to serve HTTPS with authentication and authorization. If 0, don't serve HTTPS at all.")
	flag.BoolVar(&printVersion, "V", false, "Show version and quit")
	flag.BoolVar(&printVersion, "version", false, "Show version and quit")
	flag.StringVar(&extraServiceAccounts, "extraServiceAccounts", "", "comma-separated, extra Service Accounts the Webhook should control. The full pattern for each common service account is system:serviceaccount:<namespace>:<serviceaccount-name>")
	flag.DurationVar(&minResyncDuration, "min-resync-duration", 12*time.Hour, "The resync period in reflectors will be random between MinResyncPeriod and 2*MinResyncPeriod.")
	features.DefaultFeatureGate.AddFlag(flag.CommandLine)
}

func main() {

	flag.Parse()

	logs.InitLogs()
	defer logs.FlushLogs()

	if printVersion {
		version.PrintVersionInfo()
		os.Exit(0)
	}
	version.LogVersionInfo()

	flag.CommandLine.VisitAll(func(flag *flag.Flag) {
		klog.V(1).Infof("FLAG: --%s=%q", flag.Name, flag.Value)
	})
	// We choose a random resync period between MinResyncPeriod and 2 *
	// MinResyncPeriod, so that our pods started at the same time don't list the apiserver simultaneously.

	ns := os.Getenv("NAMESPACE")
	if len(ns) < 1 {
		klog.Fatal("ENV NAMESPACE should be set.")
	}

	statefulSetAdmissionHook := statefulset.NewStatefulSetAdmissionControl()
	strategyAdmissionHook := strategy.NewStrategyAdmissionHook(&strategy.Registry)

	runAdmissionServer(statefulSetAdmissionHook, strategyAdmissionHook)
}

// the following code copied from generic-admission-server before the commit
// https://github.com/openshift/generic-admission-server/commit/78b9ae1a90b87aef095c598c2b3315ddbe766ee6
// if using `k8s.io/component-base/cli`, extra flags added by ourself will be lost.

// AdmissionHook is what callers provide, in the mutating, the validating variant or implementing even both interfaces.
// We define it here to limit how much of the import tree callers have to deal with for this plugin. This means that
// callers need to match levels of apimachinery, api, client-go, and apiserver.
type AdmissionHook apiserver.AdmissionHook
type ValidatingAdmissionHook apiserver.ValidatingAdmissionHook
type MutatingAdmissionHook apiserver.MutatingAdmissionHook

func runAdmissionServer(admissionHooks ...AdmissionHook) {
	logs.InitLogs()
	defer logs.FlushLogs()

	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	stopCh := genericapiserver.SetupSignalHandler()

	// done to avoid cannot use admissionHooks (type []AdmissionHook) as type []apiserver.AdmissionHook in argument to "github.com/openshift/kubernetes-namespace-reservation/pkg/genericadmissionserver/cmd/server".NewCommandStartAdmissionServer
	var castSlice []apiserver.AdmissionHook
	for i := range admissionHooks {
		castSlice = append(castSlice, admissionHooks[i])
	}

	cmd := server.NewCommandStartAdmissionServer(os.Stdout, os.Stderr, stopCh, castSlice...)
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	if err := cmd.Execute(); err != nil {
		klog.Fatal(err)
	}
}
