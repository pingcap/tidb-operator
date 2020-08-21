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
	"github.com/mohae/deepcopy"
)

// Note: un-exported field inside struct won't be copied and should not be included in config
// GenericConfig is a wrapper of go interface{} that makes deepcopy-gen happy
type GenericConfig struct {
	Config map[string]interface{} `json:"config,omitempty"`
}

func New(o map[string]interface{}) GenericConfig {
	return GenericConfig{o}
}

func (c *GenericConfig) Unwrap() interface{} {
	return c.Config
}

func (c *GenericConfig) DeepCopyJsonObject() *GenericConfig {
	// FIXME: mohae/deepcopy is based on reflection, which will lost un-exported field (if any)
	if c == nil {
		return nil
	}
	return deepcopy.Copy(c).(*GenericConfig)
}

func (c *GenericConfig) DeepCopy() *GenericConfig {
	return c.DeepCopyJsonObject()
}

func (c *GenericConfig) DeepCopyInto(out *GenericConfig) {
	*out = *c
	out.Config = c.DeepCopyJsonObject().Config
}
