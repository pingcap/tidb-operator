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
	"bytes"
	stdjson "encoding/json"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/mohae/deepcopy"
	"github.com/pingcap/errors"
	"k8s.io/apimachinery/pkg/util/json"
)

type GenericConfig struct {
	mp map[string]interface{}
}

var _ stdjson.Marshaler = &GenericConfig{}
var _ stdjson.Unmarshaler = &GenericConfig{}

func (c *GenericConfig) MarshalTOML() ([]byte, error) {
	if c == nil {
		return nil, nil
	}

	buff := new(bytes.Buffer)
	encoder := toml.NewEncoder(buff)
	err := encoder.Encode(c.mp)
	if err != nil {
		return nil, errors.AddStack(err)
	}

	return buff.Bytes(), nil
}

func (c *GenericConfig) UnmarshalTOML(data []byte) error {
	return toml.Unmarshal(data, &c.mp)
}

func (c *GenericConfig) MarshalJSON() ([]byte, error) {
	toml, err := c.MarshalTOML()
	if err != nil {
		return nil, err
	}

	return json.Marshal(string(toml))
}

func (c *GenericConfig) UnmarshalJSON(data []byte) error {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return errors.AddStack(err)
	}

	switch s := value.(type) {
	case string:
		err = toml.Unmarshal([]byte(s), &c.mp)
		if err != nil {
			return errors.AddStack(err)
		}
		return nil
	case map[string]interface{}:
		// If v is a *map[string]interface{}, numbers are converted to int64 or float64
		// using s directly all numbers are float type of go
		// Keep the behavior Unmarshal *map[string]interface{} directly we unmarshal again here.
		err = json.Unmarshal(data, &c.mp)
		if err != nil {
			return errors.AddStack(err)
		}
		return nil
	default:
		return errors.Errorf("unknown type: %v", reflect.TypeOf(value))
	}
}

func New(o map[string]interface{}) *GenericConfig {
	return &GenericConfig{o}
}

func (c *GenericConfig) Inner() map[string]interface{} {
	return c.mp
}

func (c *GenericConfig) DeepCopyJsonObject() *GenericConfig {
	if c == nil {
		return nil
	}
	if c.mp == nil {
		return New(nil)
	}

	mp := deepcopy.Copy(c.mp).(map[string]interface{})
	return New(mp)
}

func (c *GenericConfig) DeepCopy() *GenericConfig {
	return c.DeepCopyJsonObject()
}

func (c *GenericConfig) DeepCopyInto(out *GenericConfig) {
	*out = *c
	out.mp = c.DeepCopyJsonObject().mp
}

func (c *GenericConfig) Set(key string, value interface{}) {
	set(c.mp, key, value)
}

func (c *GenericConfig) Get(key string) (value *Value) {
	if c == nil {
		return nil
	}

	v := get(c.mp, key)
	if v == nil {
		return nil
	}

	return &Value{inner: v}
}

type Value struct {
	inner interface{}
}

func (v *Value) Interface() interface{} {
	return v.inner
}

func (v *Value) MustString() string {
	value, err := v.AsString()
	if err != nil {
		panic(err)
	}
	return value
}

func (v *Value) AsString() (string, error) {
	s, ok := (v.inner).(string)
	if !ok {
		return "", errors.Errorf("type is %v not string", reflect.TypeOf(v.inner))
	}
	return s, nil
}

func (v *Value) MustInt() int64 {
	value, err := v.AsInt()
	if err != nil {
		panic(err)
	}
	return value
}

func (v *Value) AsInt() (int64, error) {
	switch value := v.inner.(type) {
	case int:
		return int64(value), nil
	case int8:
		return int64(value), nil
	case int16:
		return int64(value), nil
	case int32:
		return int64(value), nil
	case int64:
		return int64(value), nil
	// return error if overflow ?
	case uint:
		return int64(value), nil
	case uint8:
		return int64(value), nil
	case uint16:
		return int64(value), nil
	case uint32:
		return int64(value), nil
	case uint64:
		return int64(value), nil

	default:
		return 0, errors.Errorf("type is %v not integer", reflect.TypeOf(v.inner))
	}
}

func (v *Value) MustFloat() float64 {
	value, err := v.AsFloat()
	if err != nil {
		panic(err)
	}
	return value
}

func (v *Value) AsFloat() (float64, error) {
	switch value := v.inner.(type) {
	case float32:
		return float64(value), nil
	case float64:
		return value, nil
	default:
		return 0, errors.Errorf("type is %v not float", reflect.TypeOf(v.inner))
	}
}

func set(ms map[string]interface{}, key string, value interface{}) {
	ks := strings.SplitN(key, ".", 2)
	if len(ks) == 1 {
		ms[key] = value
		return
	}

	v := strKeyMap(ms[ks[0]])
	if v == nil {
		v = make(map[string]interface{})
		ms[ks[0]] = v
	}

	vMap, ok := v.(map[string]interface{})
	if !ok {
		panic(v)
	}

	set(vMap, ks[1], value)
}

func get(ms map[string]interface{}, key string) (value interface{}) {
	ks := strings.SplitN(key, ".", 2)
	if len(ks) == 1 {
		value = ms[key]
		return
	}

	v := strKeyMap(ms[ks[0]])
	vMap, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}

	return get(vMap, ks[1])
}

// strKeyMap tries to convert `map[interface{}]interface{}` to `map[string]interface{}`
func strKeyMap(val interface{}) interface{} {
	m, ok := val.(map[interface{}]interface{})
	if ok {
		ret := map[string]interface{}{}
		for k, v := range m {
			kk, ok := k.(string)
			if !ok {
				return val
			}
			ret[kk] = strKeyMap(v)
		}
		return ret
	}

	rv := reflect.ValueOf(val)
	if rv.Kind() == reflect.Slice {
		var ret []interface{}
		for i := 0; i < rv.Len(); i++ {
			ret = append(ret, strKeyMap(rv.Index(i).Interface()))
		}
		return ret
	}

	return val
}
