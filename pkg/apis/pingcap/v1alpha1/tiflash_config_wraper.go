// Copyright 2020 PingCAP, Inc.
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
	"github.com/pingcap/tidb-operator/pkg/util/config"
	"k8s.io/apimachinery/pkg/util/json"
)

type TiFlashConfigWraper struct {
	Common *TiFlashCommonConfigWraper `json:"config,omitempty"`
	Proxy  *TiFlashProxyConfigWraper  `json:"proxy,omitempty"`
}

func NewTiFlashConfig() *TiFlashConfigWraper {
	return &TiFlashConfigWraper{
		Common: NewTiFlashCommonConfig(),
		Proxy:  NewTiFlashProxyConfig(),
	}
}

var _ stdjson.Marshaler = &TiFlashCommonConfigWraper{}
var _ stdjson.Unmarshaler = &TiFlashCommonConfigWraper{}

func NewTiFlashCommonConfig() *TiFlashCommonConfigWraper {
	return &TiFlashCommonConfigWraper{
		GenericConfig: config.New(map[string]interface{}{}),
	}
}

type TiFlashCommonConfigWraper struct {
	*config.GenericConfig
}

// MarshalJSON implements stdjson.Marshaler interface.
func (c *TiFlashCommonConfigWraper) MarshalJSON() ([]byte, error) {
	toml, err := c.GenericConfig.MarshalTOML()
	if err != nil {
		return nil, errors.AddStack(err)
	}

	return json.Marshal(string(toml))
}

// UnmarshalJSON implements stdjson.Unmarshaler interface.
// If the data is a object, we must use the Deprecated TiFlashCommonConfig to Unmarshal
// for compatibility, if we use a map[string]interface{} to Unmarshal directly,
// we can not distinct the type between integer and float for toml.
func (c *TiFlashCommonConfigWraper) UnmarshalJSON(data []byte) error {
	deprecated := new(CommonConfig)
	var err error
	c.GenericConfig, err = unmarshalJSON(data, deprecated)
	return err
}

func (c *TiFlashCommonConfigWraper) MarshalTOML() ([]byte, error) {
	if c == nil {
		return nil, nil
	}

	return c.GenericConfig.MarshalTOML()
}

var _ stdjson.Marshaler = &TiFlashProxyConfigWraper{}
var _ stdjson.Unmarshaler = &TiFlashProxyConfigWraper{}

func NewTiFlashProxyConfig() *TiFlashProxyConfigWraper {
	return &TiFlashProxyConfigWraper{
		GenericConfig: config.New(map[string]interface{}{}),
	}
}

type TiFlashProxyConfigWraper struct {
	*config.GenericConfig
}

// MarshalJSON implements stdjson.Marshaler interface.
func (c *TiFlashProxyConfigWraper) MarshalJSON() ([]byte, error) {
	toml, err := c.GenericConfig.MarshalTOML()
	if err != nil {
		return nil, errors.AddStack(err)
	}

	return json.Marshal(string(toml))
}

// UnmarshalJSON implements stdjson.Unmarshaler interface.
// If the data is a object, we must use the Deprecated TiFlashProxyConfig to Unmarshal
// for compatibility, if we use a map[string]interface{} to Unmarshal directly,
// we can not distinct the type between integer and float for toml.
func (c *TiFlashProxyConfigWraper) UnmarshalJSON(data []byte) error {
	deprecated := new(ProxyConfig)
	var err error
	c.GenericConfig, err = unmarshalJSON(data, deprecated)
	return err
}

func (c *TiFlashProxyConfigWraper) MarshalTOML() ([]byte, error) {
	if c == nil {
		return nil, nil
	}

	return c.GenericConfig.MarshalTOML()
}
