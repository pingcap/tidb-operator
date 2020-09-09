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

package tests

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	utiloperator "github.com/pingcap/tidb-operator/tests/e2e/util/operator"
	"github.com/pingcap/tidb-operator/tests/pkg/blockwriter"
	"github.com/pingcap/tidb-operator/tests/slack"
	"gopkg.in/yaml.v2"
	"k8s.io/klog"
)

const (
	defaultTableNum    = 64
	defaultConcurrency = 128
	defaultBatchSize   = 100
	defaultRawSize     = 100
)

// Config defines the config of operator tests
type Config struct {
	configFile string

	TidbVersions         string          `yaml:"tidb_versions" json:"tidb_versions"`
	InstallOperator      bool            `yaml:"install_opeartor" json:"install_opeartor"`
	OperatorTag          string          `yaml:"operator_tag" json:"operator_tag"`
	OperatorImage        string          `yaml:"operator_image" json:"operator_image"`
	BackupImage          string          `yaml:"backup_image" json:"backup_image"`
	OperatorFeatures     map[string]bool `yaml:"operator_features" json:"operator_features"`
	UpgradeOperatorTag   string          `yaml:"upgrade_operator_tag" json:"upgrade_operator_tag"`
	UpgradeOperatorImage string          `yaml:"upgrade_operator_image" json:"upgrade_operator_image"`
	LogDir               string          `yaml:"log_dir" json:"log_dir"`
	FaultTriggerPort     int             `yaml:"fault_trigger_port" json:"fault_trigger_port"`
	Nodes                []Nodes         `yaml:"nodes" json:"nodes"`
	ETCDs                []Nodes         `yaml:"etcds" json:"etcds"`
	APIServers           []Nodes         `yaml:"apiservers" json:"apiservers"`
	CertFile             string
	KeyFile              string

	PDMaxReplicas       int `yaml:"pd_max_replicas" json:"pd_max_replicas"`
	TiKVGrpcConcurrency int `yaml:"tikv_grpc_concurrency" json:"tikv_grpc_concurrency"`
	TiDBTokenLimit      int `yaml:"tidb_token_limit" json:"tidb_token_limit"`

	// old versions of reparo does not support idempotent incremental recover, so we lock the version explicitly
	AdditionalDrainerVersion string `yaml:"file_drainer_version" json:"file_drainer_version"`

	// Block writer
	BlockWriter blockwriter.Config `yaml:"block_writer,omitempty"`

	// For local test
	OperatorRepoUrl string `yaml:"operator_repo_url" json:"operator_repo_url"`
	OperatorRepoDir string `yaml:"operator_repo_dir" json:"operator_repo_dir"`
	// chart dir
	ChartDir string `yaml:"chart_dir" json:"chart_dir"`
	// manifest dir
	ManifestDir string `yaml:"manifest_dir" json:"manifest_dir"`

	E2EImage string `yaml:"e2e_image" json:"e2e_image"`

	PreloadImages bool `yaml:"preload_images" json:"preload_images"`

	OperatorKiller utiloperator.OperatorKillerConfig
}

// Nodes defines a series of nodes that belong to the same physical node.
type Nodes struct {
	PhysicalNode string `yaml:"physical_node" json:"physical_node"`
	Nodes        []Node `yaml:"nodes" json:"nodes"`
}

type Node struct {
	IP   string `yaml:"ip" json:"ip"`
	Name string `yaml:"name" json:"name"`
}

// NewDefaultConfig creates a default configuration.
func NewDefaultConfig() *Config {
	return &Config{
		AdditionalDrainerVersion: "v3.0.8",

		PDMaxReplicas:       5,
		TiDBTokenLimit:      1024,
		TiKVGrpcConcurrency: 8,

		BlockWriter: blockwriter.Config{
			TableNum:    defaultTableNum,
			Concurrency: defaultConcurrency,
			BatchSize:   defaultBatchSize,
			RawSize:     defaultRawSize,
		},
	}
}

// NewConfig creates a new config.
func NewConfig() (*Config, error) {
	cfg := NewDefaultConfig()
	flag.StringVar(&cfg.configFile, "config", "", "Config file")
	flag.StringVar(&cfg.LogDir, "log-dir", "/logDir", "log directory")
	flag.IntVar(&cfg.FaultTriggerPort, "fault-trigger-port", 23332, "the http port of fault trigger service")
	flag.StringVar(&cfg.TidbVersions, "tidb-versions", "v3.0.6,v3.0.7,v3.0.8", "tidb versions")
	flag.StringVar(&cfg.OperatorTag, "operator-tag", "master", "operator tag used to choose charts")
	flag.StringVar(&cfg.OperatorImage, "operator-image", "pingcap/tidb-operator:latest", "operator image")
	flag.StringVar(&cfg.E2EImage, "e2e-image", "pingcap/tidb-operator-e2e:latest", "operator-e2e image")
	flag.StringVar(&cfg.UpgradeOperatorTag, "upgrade-operator-tag", "", "upgrade operator tag used to choose charts")
	flag.StringVar(&cfg.UpgradeOperatorImage, "upgrade-operator-image", "", "upgrade operator image")
	flag.StringVar(&cfg.OperatorRepoDir, "operator-repo-dir", "/tidb-operator", "local directory to which tidb-operator cloned")
	flag.StringVar(&cfg.OperatorRepoUrl, "operator-repo-url", "https://github.com/pingcap/tidb-operator.git", "tidb-operator repo url used")
	flag.StringVar(&cfg.ChartDir, "chart-dir", "", "chart dir")
	flag.StringVar(&slack.WebhookURL, "slack-webhook-url", "", "slack webhook url")
	flag.StringVar(&slack.TestName, "test-name", "operator-test", "the stability test name")
	flag.Parse()

	operatorRepo, err := ioutil.TempDir("", "tidb-operator")
	if err != nil {
		return nil, err
	}
	cfg.OperatorRepoDir = operatorRepo

	if strings.TrimSpace(cfg.ChartDir) == "" {
		chartDir, err := ioutil.TempDir("", "charts")
		if err != nil {
			return nil, err
		}
		cfg.ChartDir = chartDir
	}

	manifestDir, err := ioutil.TempDir("", "manifests")
	if err != nil {
		return nil, err
	}
	cfg.ManifestDir = manifestDir

	return cfg, nil
}

func ParseConfigOrDie() *Config {
	cfg, err := NewConfig()
	if err != nil {
		slack.NotifyAndPanic(err)
	}
	if err := cfg.Parse(); err != nil {
		slack.NotifyAndPanic(err)
	}

	klog.Infof("using config: %+v", cfg)
	return cfg
}

// Parse parses flag definitions from the argument list.
func (c *Config) Parse() error {
	// Parse first to get config file
	flag.Parse()

	if c.configFile != "" {
		if err := c.configFromFile(c.configFile); err != nil {
			return err
		}
	}

	// Parse again to replace with command line options.
	flag.Parse()

	return nil
}

func (c *Config) configFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, c)
}

func (c *Config) GetTiDBVersion() (string, error) {
	tidbVersions := strings.Split(c.TidbVersions, ",")
	if len(tidbVersions) == 0 {
		return "", fmt.Errorf("init tidb versions can not be nil")
	}

	return tidbVersions[0], nil
}

func (c *Config) GetTiDBVersionOrDie() string {
	v, err := c.GetTiDBVersion()
	if err != nil {
		slack.NotifyAndPanic(err)
	}

	return v
}

func (c *Config) GetUpgradeTidbVersions() []string {
	tidbVersions := strings.Split(c.TidbVersions, ",")

	return tidbVersions[1:]
}

func (c *Config) GetUpgradeTidbVersionsOrDie() []string {
	versions := c.GetUpgradeTidbVersions()
	if len(versions) < 1 {
		slack.NotifyAndPanic(fmt.Errorf("upgrade tidb verions is empty"))
	}

	return versions
}

func (c *Config) CleanTempDirs() error {
	if c.OperatorRepoDir != "" {
		err := os.RemoveAll(c.OperatorRepoDir)
		if err != nil {
			return err
		}
	}
	if c.ChartDir != "" {
		err := os.RemoveAll(c.ChartDir)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) PrettyPrintJSON() (string, error) {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (c *Config) MustPrettyPrintJSON() string {
	s, err := c.PrettyPrintJSON()
	if err != nil {
		panic(err)
	}
	return s
}
