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
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/mohae/deepcopy"
	"github.com/pingcap/errors"
	"k8s.io/apimachinery/pkg/util/json"
)

// Note: un-exported field inside struct won't be copied and should not be included in config
// GenericConfig is a wrapper of go interface{} that makes deepcopy-gen happy
type GenericConfig struct {
	mp map[string]interface{}
}

var _ stdjson.Marshaler = &GenericConfig{}
var _ stdjson.Unmarshaler = &GenericConfig{}

func (c *GenericConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(&c.mp)
}

func (c *GenericConfig) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &c.mp)
}

func New(o map[string]interface{}) *GenericConfig {
	return &GenericConfig{o}
}

func FromJsonObject(x interface{}) (*GenericConfig, error) {
	data, err := json.Marshal(x)
	if err != nil {
		return nil, errors.AddStack(err)
	}

	mp := make(map[string]interface{})
	err = json.Unmarshal(data, &mp)
	if err != nil {
		return nil, errors.AddStack(err)
	}

	return New(mp), nil
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

// JsonPatchDefaults return a new GenericConfig with every item set as the value in defaults if
// it is not setted.
func (c *GenericConfig) JsonPatchDefaults(defaults interface{}) (mergedConfig *GenericConfig, err error) {
	defaultsData, err := json.Marshal(defaults)
	if err != nil {
		return nil, errors.AddStack(err)
	}

	patchData, err := json.Marshal(c.mp)
	if err != nil {
		return nil, errors.AddStack(err)
	}

	mergeData, err := jsonpatch.MergePatch(defaultsData, patchData)
	if err != nil {
		return nil, errors.AddStack(err)
	}

	mp := make(map[string]interface{})
	err = json.Unmarshal(mergeData, &mp)
	if err != nil {
		return nil, errors.AddStack(err)
	}

	tmp := New(mp)
	return tmp, nil
}

func (c *GenericConfig) DeepCopy() *GenericConfig {
	return c.DeepCopyJsonObject()
}

func (c *GenericConfig) DeepCopyInto(out *GenericConfig) {
	*out = *c
	out.mp = c.DeepCopyJsonObject().mp
}

func (c *GenericConfig) UnmarshalToml(v interface{}) error {
	buff := new(bytes.Buffer)
	encoder := toml.NewEncoder(buff)
	err := encoder.Encode(c.mp)
	if err != nil {
		return errors.AddStack(err)
	}

	err = toml.Unmarshal(buff.Bytes(), v)
	if err != nil {
		return errors.AddStack(err)
	}
	return nil
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

func (v *Value) AsString() string {
	s := (v.inner).(string)
	return s
}

func (v *Value) IsString() bool {
	if v == nil {
		return false
	}
	_, ok := (v.inner).(string)
	return ok
}

func (v *Value) AsInt() int64 {
	rv := reflect.ValueOf(v.inner)
	return rv.Int()
}

func (v *Value) AsFloat() float64 {
	rv := reflect.ValueOf(v.inner)
	return rv.Float()
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
