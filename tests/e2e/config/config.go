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

package config

import (
	"flag"
	"io/ioutil"

	"github.com/pingcap/tidb-operator/tests"
	v1 "k8s.io/api/core/v1"
)

// Global Test configuration.
var TestConfig *tests.Config = tests.NewDefaultConfig()

// RegisterTiDBOperatorFlags registers flags for tidb-operator.
func RegisterTiDBOperatorFlags(flags *flag.FlagSet) {
	flags.StringVar(&TestConfig.LogDir, "log-dir", "/logDir", "log directory")
	flags.IntVar(&TestConfig.FaultTriggerPort, "fault-trigger-port", 23332, "the http port of fault trigger service")
	flags.StringVar(&TestConfig.TidbVersions, "tidb-versions", "v3.0.2,v3.0.3,v3.0.4", "tidb versions")
	flags.StringVar(&TestConfig.TestApiserverImage, "test-apiserver-image", "pingcap/test-apiserver:latest", "test-apiserver image")
	flags.StringVar(&TestConfig.OperatorTag, "operator-tag", "master", "operator tag used to choose charts")
	flags.StringVar(&TestConfig.OperatorImage, "operator-image", "pingcap/tidb-operator:latest", "operator image")
	flags.StringVar(&TestConfig.UpgradeOperatorTag, "upgrade-operator-tag", "", "upgrade operator tag used to choose charts")
	flags.StringVar(&TestConfig.UpgradeOperatorImage, "upgrade-operator-image", "", "upgrade operator image")
	flags.StringVar(&TestConfig.OperatorRepoDir, "operator-repo-dir", "/tidb-operator", "local directory to which tidb-operator cloned")
	flags.StringVar(&TestConfig.OperatorRepoUrl, "operator-repo-url", "https://github.com/pingcap/tidb-operator.git", "tidb-operator repo url used")
	flags.StringVar(&TestConfig.ChartDir, "chart-dir", "", "chart dir")
}

func AfterReadingAllFlags() error {
	if TestConfig.OperatorRepoDir == "" {
		operatorRepo, err := ioutil.TempDir("", "tidb-operator")
		if err != nil {
			return err
		}
		TestConfig.OperatorRepoDir = operatorRepo
	}

	if TestConfig.ChartDir == "" {
		chartDir, err := ioutil.TempDir("", "charts")
		if err != nil {
			return err
		}
		TestConfig.ChartDir = chartDir
	}

	if TestConfig.ManifestDir == "" {
		manifestDir, err := ioutil.TempDir("", "manifests")
		if err != nil {
			return err
		}
		TestConfig.ManifestDir = manifestDir
	}

	return nil
}

// NewDefaultOperatorConfig creates default operator configuration.
func NewDefaultOperatorConfig(cfg *tests.Config) *tests.OperatorConfig {
	return &tests.OperatorConfig{
		Namespace:      "pingcap",
		ReleaseName:    "operator",
		Image:          cfg.OperatorImage,
		Tag:            cfg.OperatorTag,
		SchedulerImage: "k8s.gcr.io/kube-scheduler",
		Features: []string{
			"StableScheduling=true",
			"AdvancedStatefulSet=true",
		},
		LogLevel:            "4",
		WebhookServiceName:  "webhook-service",
		WebhookSecretName:   "webhook-secret",
		WebhookConfigName:   "webhook-config",
		ImagePullPolicy:     v1.PullIfNotPresent,
		TestMode:            true,
		AdvancedStatefulSet: true,
	}
}
