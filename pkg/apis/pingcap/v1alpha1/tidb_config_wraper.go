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
	"reflect"

	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/util/config"
	"github.com/pingcap/tidb-operator/pkg/util/toml"
	"k8s.io/apimachinery/pkg/util/json"
)

var _ stdjson.Marshaler = &TiDBConfigWraper{}
var _ stdjson.Unmarshaler = &TiDBConfigWraper{}

func NewTiDBConfig() *TiDBConfigWraper {
	return &TiDBConfigWraper{
		GenericConfig: config.New(map[string]interface{}{}),
	}
}

type TiDBConfigWraper struct {
	Deprecated *TiDBConfig
	*config.GenericConfig
}

// MarshalJSON implements stdjson.Marshaler interface.
func (c *TiDBConfigWraper) MarshalJSON() ([]byte, error) {
	toml, err := c.GenericConfig.MarshalTOML()
	if err != nil {
		return nil, errors.AddStack(err)
	}

	return json.Marshal(string(toml))
}

// UnmarshalJSON implements stdjson.Unmarshaler interface.
// If the data is a object, we must use the Deprecated TiDBConfig to Unmarshal
// for compatibility, if we use a map[string]interface{} to Unmarshal directly,
// we can not distinct the type between integer and float for toml.
func (c *TiDBConfigWraper) UnmarshalJSON(data []byte) error {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return errors.AddStack(err)
	}

	var tomlData []byte
	switch s := value.(type) {
	case string:
		tomlData = []byte(s)
	case map[string]interface{}:
		err = json.Unmarshal(data, &c.Deprecated)
		if err != nil {
			return errors.AddStack(err)
		}

		tomlData, err = toml.Marshal(c.Deprecated)
		if err != nil {
			return errors.AddStack(err)
		}

	default:
		return errors.Errorf("unknown type: %v", reflect.TypeOf(value))
	}

	c.GenericConfig = config.New(nil)
	err = c.GenericConfig.UnmarshalTOML(tomlData)
	if err != nil {
		return errors.AddStack(err)
	}
	return nil
}

func (c *TiDBConfigWraper) MarshalTOML() ([]byte, error) {
	if c == nil {
		return nil, nil
	}

	return c.GenericConfig.MarshalTOML()
}
