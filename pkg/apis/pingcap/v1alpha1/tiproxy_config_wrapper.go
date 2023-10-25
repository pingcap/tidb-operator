// Copyright 2022 PingCAP, Inc.
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

package v1alpha1

import (
	stdjson "encoding/json"

	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/util/config"
	tiproxyConfig "github.com/pingcap/tiproxy/lib/config"
	"k8s.io/apimachinery/pkg/util/json"
)

var _ stdjson.Marshaler = &TiProxyConfigWraper{}
var _ stdjson.Unmarshaler = &TiProxyConfigWraper{}

func NewTiProxyConfig() *TiProxyConfigWraper {
	return &TiProxyConfigWraper{
		GenericConfig: config.New(map[string]interface{}{}),
	}
}

type TiProxyConfigWraper struct {
	*config.GenericConfig `json:",inline"`
}

// MarshalJSON implements stdjson.Marshaler interface.
func (c *TiProxyConfigWraper) MarshalJSON() ([]byte, error) {
	toml, err := c.GenericConfig.MarshalTOML()
	if err != nil {
		return nil, errors.AddStack(err)
	}

	return json.Marshal(string(toml))
}

// UnmarshalJSON implements stdjson.Unmarshaler interface.
func (c *TiProxyConfigWraper) UnmarshalJSON(data []byte) error {
	deprecated := new(tiproxyConfig.Config)
	var err error
	c.GenericConfig, err = unmarshalJSON(data, deprecated)
	return err
}

func (c *TiProxyConfigWraper) MarshalTOML() ([]byte, error) {
	if c == nil {
		return nil, nil
	}

	return c.GenericConfig.MarshalTOML()
}
