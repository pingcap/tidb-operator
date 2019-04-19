package tests

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pingcap/tidb-operator/tests/pkg/blockwriter"

	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
)

const (
	defaultTableNum    int = 64
	defaultConcurrency     = 128
	defaultBatchSize       = 100
	defaultRawSize         = 100
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
	CertFile         string
	KeyFile          string

	// Block writer
	BlockWriter blockwriter.Config `yaml:"block_writer,omitempty"`

	// For local test
	OperatorRepoDir string `yaml:"operator_repo_dir" json:"operator_repo_dir"`
	// chart dir
	ChartDir string `yaml:"chart_dir" json:"chart_dir"`
}

// Nodes defines a series of nodes that belong to the same physical node.
type Nodes struct {
	PhysicalNode string   `yaml:"physical_node" json:"physical_node"`
	Nodes        []string `yaml:"nodes" json:"nodes"`
}

// NewConfig creates a new config.
func NewConfig() (*Config, error) {
	cfg := &Config{
		BlockWriter: blockwriter.Config{
			TableNum:    defaultTableNum,
			Concurrency: defaultConcurrency,
			BatchSize:   defaultBatchSize,
			RawSize:     defaultRawSize,
		},
	}
	flag.StringVar(&cfg.configFile, "config", "", "Config file")
	flag.StringVar(&cfg.LogDir, "log-dir", "/logDir", "log directory")
	flag.IntVar(&cfg.FaultTriggerPort, "fault-trigger-port", 23332, "the http port of fault trigger service")
	flag.StringVar(&cfg.TidbVersions, "tidb-versions", "v2.1.3,v2.1.4", "tidb versions")
	flag.StringVar(&cfg.OperatorTag, "operator-tag", "master", "operator tag used to choose charts")
	flag.StringVar(&cfg.OperatorImage, "operator-image", "pingcap/tidb-operator:latest", "operator image")
	flag.StringVar(&cfg.CertFile, "tls-cert-file", cfg.CertFile, ""+
		"File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated "+
		"after server cert).")
	flag.StringVar(&cfg.KeyFile, "tls-private-key-file", cfg.KeyFile, ""+
		"File containing the default x509 private key matching --tls-cert-file.")
	flag.StringVar(&cfg.OperatorRepoDir, "operator-repo-dir", "/tidb-operator", "local directory to which tidb-operator cloned")
	flag.Parse()

	operatorRepo, err := ioutil.TempDir("", "tidb-operator")
	if err != nil {
		return nil, err
	}
	cfg.OperatorRepoDir = operatorRepo

	chartDir, err := ioutil.TempDir("", "charts")
	if err != nil {
		return nil, err
	}
	cfg.ChartDir = chartDir
	return cfg, nil
}

func ParseConfigOrDie() *Config {
	cfg, err := NewConfig()
	if err != nil {
		panic(err)
	}
	if err := cfg.Parse(); err != nil {
		panic(err)
	}

	glog.Infof("using config: %+v", cfg)
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

	if err = yaml.Unmarshal(data, c); err != nil {
		return err
	}

	return nil
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
		panic(err)
	}

	return v
}

func (c *Config) ConfigTLS() *tls.Config {
	sCert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		glog.Fatal(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{sCert},
	}
}

func (c *Config) GetUpgradeTidbVersions() []string {
	tidbVersions := strings.Split(c.TidbVersions, ",")

	return tidbVersions[1:]
}

func (c *Config) GetUpgradeTidbVersionsOrDie() []string {
	versions := c.GetUpgradeTidbVersions()
	if len(versions) < 1 {
		panic("upgrade tidb verions is empty")
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
