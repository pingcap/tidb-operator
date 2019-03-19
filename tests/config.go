package tests

import (
	"flag"
	"io/ioutil"

	"github.com/pingcap/errors"
	yaml "gopkg.in/yaml.v2"
)

// Config defines the config of operator tests
type Config struct {
	*flag.FlagSet `json:"-"`

	configFile string

	LogDir     string  `yaml:"log-dir" json:"log-dir"`
	Nodes      []Nodes `yaml:"nodes" json:"nodes"`
	ETCDs      []Nodes `yaml:"etcds" json:"etcds"`
	APIServers []Nodes `yaml:"apiservers" json:"apiservers"`
}

// Nodes defines a series of nodes that belong to the same physical node.
type Nodes struct {
	PhysicalNode string   `yaml:"physical_node" json:"physical_node"`
	Nodes        []string `yaml:"nodes" json:"nodes"`
}

// NewConfig creates a new config.
func NewConfig() *Config {
	cfg := &Config{}
	cfg.FlagSet = flag.NewFlagSet("e2e", flag.ContinueOnError)

	fs := cfg.FlagSet

	fs.StringVar(&cfg.configFile, "config", "", "Config file")
	fs.StringVar(&cfg.LogDir, "log-dir", "/logDir", "log directory")

	return cfg
}

// Parse parses flag definitions from the argument list.
func (c *Config) Parse(args []string) error {
	// Parse first to get config file
	err := c.FlagSet.Parse(args)
	if err != nil {
		return err
	}

	if c.configFile != "" {
		if err = c.configFromFile(c.configFile); err != nil {
			return err
		}
	}

	// Parse again to replace with command line options.
	err = c.FlagSet.Parse(args)
	if err != nil {
		return err
	}

	if len(c.FlagSet.Args()) != 0 {
		return errors.Errorf("'%s' is an invalid flag", c.FlagSet.Arg(0))
	}

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
