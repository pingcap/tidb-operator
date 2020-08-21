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
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// TidbClusterConfig indicates which tidb cluster to use.
type TidbClusterConfig struct {
	KubeContext string `json:"context,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	ClusterName string `json:"clusterName,omitempty"`
}

func Load(s string) (*TidbClusterConfig, error) {
	cfg := &TidbClusterConfig{}

	err := yaml.Unmarshal([]byte(s), cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func LoadFile(filename string) (*TidbClusterConfig, error) {
	c, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return Load(string(c))
}
