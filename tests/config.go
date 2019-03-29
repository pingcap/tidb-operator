package tests

import (
	"flag"
	"fmt"
	"io/ioutil"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

// Config defines the config of operator tests
type Config struct {
	configFile string

	TidbVersions     string  `yaml:"tidb_versions" json:"tidb_versions"`
	OperatorTag      string  `yaml:"operator_tag" json:"operator_tag"`
	OperatorImage    string  `yaml:"operator_image" json:"operator_image"`
	LogDir           string  `yaml:"log_dir" json:"log_dir"`
	FaultTriggerPort int     `yaml:"fault_trigger_port" json:"fault_trigger_port"`
	Nodes            []Nodes `yaml:"nodes" json:"nodes"`
	ETCDs            []Nodes `yaml:"etcds" json:"etcds"`
	APIServers       []Nodes `yaml:"apiservers" json:"apiservers"`
}

// Nodes defines a series of nodes that belong to the same physical node.
type Nodes struct {
	PhysicalNode string   `yaml:"physical_node" json:"physical_node"`
	Nodes        []string `yaml:"nodes" json:"nodes"`
}

// NewConfig creates a new config.
func NewConfig() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.configFile, "config", "", "Config file")
	flag.StringVar(&cfg.LogDir, "log-dir", "/logDir", "log directory")
	flag.IntVar(&cfg.FaultTriggerPort, "fault-trigger-port", 23332, "the http port of fault trigger service")
	flag.StringVar(&cfg.TidbVersions, "tidb-versions", "v2.1.3,v2.1.4", "tidb versions")
	flag.StringVar(&cfg.OperatorTag, "operator-tag", "master", "operator tag used to choose charts")
	flag.StringVar(&cfg.OperatorImage, "operator-image", "pingcap/tidb-operator:latest", "operator image")

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

func (c *Config) ParseOrDie() {
	err := c.Parse()
	if err != nil {
		panic(err)
	}
}

func (c *Config) configFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	if err = yaml.Unmarshal(data, c); err != nil {
		return err
	}

	return nil
}

func (c *Config) GetInitTidbVersion() (string, error) {
	tidbVersions := strings.Split(c.TidbVersions, ",")
	if len(tidbVersions) == 0 {
		return "", fmt.Errorf("init tidb versions can not be nil")
	}

	return tidbVersions[0], nil
}

func (c *Config) GetUpgradeTidbVersions() []string {
	tidbVersions := strings.Split(c.TidbVersions, ",")

	return tidbVersions[1:]
}
