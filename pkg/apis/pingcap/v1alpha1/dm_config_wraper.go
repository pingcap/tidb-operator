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
	"github.com/pingcap/tidb-operator/pkg/apis/util/config"
	"k8s.io/apimachinery/pkg/util/json"
)

var _ stdjson.Marshaler = &MasterConfigWraper{}
var _ stdjson.Unmarshaler = &MasterConfigWraper{}
var _ stdjson.Marshaler = &WorkerConfigWraper{}
var _ stdjson.Unmarshaler = &WorkerConfigWraper{}

func NewMasterConfig() *MasterConfigWraper {
	return &MasterConfigWraper{
		GenericConfig: config.New(map[string]interface{}{}),
	}
}

func NewWorkerConfig() *WorkerConfigWraper {
	return &WorkerConfigWraper{
		GenericConfig: config.New(map[string]interface{}{}),
	}
}

type MasterConfigWraper struct {
	*config.GenericConfig `json:",inline"`
}

type WorkerConfigWraper struct {
	*config.GenericConfig `json:",inline"`
}

// MarshalJSON implements stdjson.Marshaler interface.
func (c *MasterConfigWraper) MarshalJSON() ([]byte, error) {
	toml, err := c.GenericConfig.MarshalTOML()
	if err != nil {
		return nil, errors.AddStack(err)
	}

	return json.Marshal(string(toml))
}

// UnmarshalJSON implements stdjson.Unmarshaler interface.
// If the data is a object, we must use the Deprecated MasterConfig to Unmarshal
// for compatibility, if we use a map[string]interface{} to Unmarshal directly,
// we can not distinct the type between integer and float for toml.
func (c *MasterConfigWraper) UnmarshalJSON(data []byte) error {
	deprecated := new(MasterConfig)
	var err error
	c.GenericConfig, err = unmarshalJSON(data, deprecated)
	return err
}

func (c *MasterConfigWraper) MarshalTOML() ([]byte, error) {
	if c == nil {
		return nil, nil
	}

	return c.GenericConfig.MarshalTOML()
}

// MarshalJSON implements stdjson.Marshaler interface.
func (c *WorkerConfigWraper) MarshalJSON() ([]byte, error) {
	toml, err := c.GenericConfig.MarshalTOML()
	if err != nil {
		return nil, errors.AddStack(err)
	}

	return json.Marshal(string(toml))
}

// UnmarshalJSON implements stdjson.Unmarshaler interface.
// If the data is a object, we must use the Deprecated WorkerConfig to Unmarshal
// for compatibility, if we use a map[string]interface{} to Unmarshal directly,
// we can not distinct the type between integer and float for toml.
func (c *WorkerConfigWraper) UnmarshalJSON(data []byte) error {
	deprecated := new(WorkerConfig)
	var err error
	c.GenericConfig, err = unmarshalJSON(data, deprecated)
	return err
}

func (c *WorkerConfigWraper) MarshalTOML() ([]byte, error) {
	if c == nil {
		return nil, nil
	}

	return c.GenericConfig.MarshalTOML()
}
