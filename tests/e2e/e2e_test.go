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

package e2e

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	e2econfig "github.com/pingcap/tidb-operator/tests/e2e/config"
	"k8s.io/klog"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/config"
	"k8s.io/kubernetes/test/e2e/framework/testfiles"
	"k8s.io/kubernetes/test/e2e/framework/viperconfig"

	// test sources
	_ "github.com/pingcap/tidb-operator/tests/e2e/tidbcluster"
)

var viperConfig = flag.String("viper-config", "", "The name of a viper config file (https://github.com/spf13/viper#what-is-viper). All e2e command line parameters can also be configured in such a file. May contain a path and may or may not contain the file suffix. The default is to look for an optional file with `e2e` as base name. If a file is specified explicitly, it must be present.")

// handleFlags sets up all flags and parses the command line.
func handleFlags() {
	config.CopyFlags(config.Flags, flag.CommandLine)
	framework.RegisterCommonFlags(flag.CommandLine)
	framework.RegisterClusterFlags(flag.CommandLine)
	e2econfig.RegisterTiDBOperatorFlags(flag.CommandLine)
	flag.Parse()
}

func init() {
	framework.RegisterProvider("kind", func() (framework.ProviderInterface, error) {
		return framework.NullProvider{}, nil
	})
	framework.RegisterProvider("openshift", func() (framework.ProviderInterface, error) {
		return framework.NullProvider{}, nil
	})
}

func TestMain(m *testing.M) {
	// Register test flags, then parse flags.
	handleFlags()

	flag.CommandLine.VisitAll(func(flag *flag.Flag) {
		klog.V(1).Infof("FLAG: --%s=%q", flag.Name, flag.Value)
	})

	// Now that we know which Viper config (if any) was chosen,
	// parse it and update those options which weren't already set via command line flags
	// (which have higher priority).
	if err := viperconfig.ViperizeFlags(*viperConfig, "e2e", flag.CommandLine); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	framework.AfterReadingAllFlags(&framework.TestContext)
	e2econfig.AfterReadingAllFlags()

	if framework.TestContext.RepoRoot != "" {
		testfiles.AddFileSource(testfiles.RootFileSource{Root: framework.TestContext.RepoRoot})
	}

	rand.Seed(time.Now().UnixNano())
	os.Exit(m.Run())
}

func TestE2E(t *testing.T) {
	RunE2ETests(t)
}
