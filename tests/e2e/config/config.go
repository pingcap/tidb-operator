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
	"fmt"
	"io/ioutil"
	"time"

	"github.com/pingcap/tidb-operator/tests"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/kubernetes/test/e2e/framework"
)

// Global Test configuration.
var TestConfig *tests.Config = tests.NewDefaultConfig()

// RegisterTiDBOperatorFlags registers flags for tidb-operator.
func RegisterTiDBOperatorFlags(flags *flag.FlagSet) {
	flags.StringVar(&TestConfig.LogDir, "log-dir", "/logDir", "log directory")
	flags.IntVar(&TestConfig.FaultTriggerPort, "fault-trigger-port", 23332, "the http port of fault trigger service")
	flags.StringVar(&TestConfig.E2EImage, "e2e-image", "", "e2e image")
	flags.BoolVar(&TestConfig.InstallOperator, "install-operator", true, "install a default operator")
	flags.StringVar(&TestConfig.OperatorTag, "operator-tag", "master", "operator tag used to choose charts")
	flags.StringVar(&TestConfig.OperatorImage, "operator-image", "pingcap/tidb-operator:latest", "operator image")
	flags.Var(cliflag.NewMapStringBool(&TestConfig.OperatorFeatures), "operator-features", "a set of key=value pairs that describe feature gates for operator")
	flags.StringVar(&TestConfig.UpgradeOperatorTag, "upgrade-operator-tag", "", "upgrade operator tag used to choose charts")
	flags.StringVar(&TestConfig.UpgradeOperatorImage, "upgrade-operator-image", "", "upgrade operator image")
	flags.StringVar(&TestConfig.OperatorRepoDir, "operator-repo-dir", "/tidb-operator", "local directory to which tidb-operator cloned")
	flags.StringVar(&TestConfig.OperatorRepoUrl, "operator-repo-url", "https://github.com/pingcap/tidb-operator.git", "tidb-operator repo url used")
	flags.StringVar(&TestConfig.ChartDir, "chart-dir", "", "chart dir")
	flags.BoolVar(&TestConfig.PreloadImages, "preload-images", false, "if set, preload images in the bootstrap of e2e process")
	flags.StringVar(&TestConfig.BackupImage, "backup-image", "", "backup image")
	flags.BoolVar(&TestConfig.OperatorKiller.Enabled, "operator-killer", false, "whether to enable operator kill")
	flags.DurationVar(&TestConfig.OperatorKiller.Interval, "operator-killer-interval", 5*time.Minute, "interval between operator kills")
	flags.Float64Var(&TestConfig.OperatorKiller.JitterFactor, "operator-killer-jitter-factor", 1, "factor used to jitter operator kills")
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
	features := []string{}
	for k, v := range cfg.OperatorFeatures {
		t := "false"
		if v {
			t = "true"
		}
		features = append(features, fmt.Sprintf("%s=%s", k, t))
	}
	return &tests.OperatorConfig{
		Namespace:                 "pingcap",
		ReleaseName:               "operator",
		Image:                     cfg.OperatorImage,
		Tag:                       cfg.OperatorTag,
		ControllerManagerReplicas: tests.IntPtr(2),
		SchedulerImage:            "k8s.gcr.io/kube-scheduler",
		SchedulerReplicas:         tests.IntPtr(2),
		Features:                  features,
		LogLevel:                  "4",
		WebhookServiceName:        "webhook-service",
		WebhookSecretName:         "webhook-secret",
		WebhookConfigName:         "webhook-config",
		ImagePullPolicy:           v1.PullIfNotPresent,
		TestMode:                  true,
		WebhookEnabled:            true,
		StsWebhookEnabled:         true,
		PodWebhookEnabled:         false,
		Cabundle:                  "",
		BackupImage:               cfg.BackupImage,
		StringValues: map[string]string{
			"admissionWebhook.failurePolicy.validation": "Fail",
			"admissionWebhook.failurePolicy.mutation":   "Fail",
		},
	}
}

func LoadClientRawConfig() (clientcmdapi.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = framework.TestContext.KubeConfig
	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}
	if framework.TestContext.KubeContext != "" {
		overrides.CurrentContext = framework.TestContext.KubeContext
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).RawConfig()
}
