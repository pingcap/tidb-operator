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
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/mohae/deepcopy"
	"github.com/pingcap/errors"
	"k8s.io/apimachinery/pkg/util/json"
)

type GenericConfig struct {
	// Export this field to make "apiequality.Semantic.DeepEqual" happy now.
	// User of GenericConfig should not directly access this field.
	MP map[string]interface{}
}

var _ stdjson.Marshaler = &GenericConfig{}
var _ stdjson.Unmarshaler = &GenericConfig{}

func (c *GenericConfig) MarshalTOML() ([]byte, error) {
	if c == nil {
		return nil, nil
	}

	buff := new(bytes.Buffer)
	encoder := toml.NewEncoder(buff)
	err := encoder.Encode(c.MP)
	if err != nil {
		return nil, errors.AddStack(err)
	}

	return buff.Bytes(), nil
}

func (c *GenericConfig) UnmarshalTOML(data []byte) error {
	return toml.Unmarshal(data, &c.MP)
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
		err = toml.Unmarshal([]byte(s), &c.MP)
		if err != nil {
			return errors.AddStack(err)
		}
		return nil
	case map[string]interface{}:
		// If v is a *map[string]interface{}, numbers are converted to int64 or float64
		// using s directly all numbers are float type of go
		// Keep the behavior Unmarshal *map[string]interface{} directly we unmarshal again here.
		err = json.Unmarshal(data, &c.MP)
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
	return c.MP
}

func (c *GenericConfig) DeepCopyJsonObject() *GenericConfig {
	if c == nil {
		return nil
	}
	if c.MP == nil {
		return New(nil)
	}

	MP := deepcopy.Copy(c.MP).(map[string]interface{})
	return New(MP)
}

func (c *GenericConfig) DeepCopy() *GenericConfig {
	return c.DeepCopyJsonObject()
}

func (c *GenericConfig) DeepCopyInto(out *GenericConfig) {
	*out = *c
	out.MP = c.DeepCopyJsonObject().MP
}

func (c *GenericConfig) Set(key string, value interface{}) {
	set(c.MP, key, value)
}

// SetTable set multiple KV of a table
//
// For example:
// c.SetTable("root", "key1", "val1", "key2", 10)
//
// Invalid KV will will be ignored
func (c *GenericConfig) SetTable(table string, kvs ...interface{}) {
	for i := 0; i < len(kvs); {
		if key, ok := kvs[i].(string); ok {
			if i+1 < len(kvs) {
				val := kvs[i+1]
				set(c.MP, fmt.Sprintf("%s.%s", table, key), val)
			}
		}

		i += 2
	}
}

func (c *GenericConfig) Del(key string) {
	del(c.MP, key)
}

func (c *GenericConfig) SetIfNil(key string, value interface{}) {
	v := c.Get(key)
	if v != nil {
		return
	}
	set(c.MP, key, value)
}

func (c *GenericConfig) Get(key string) (value *Value) {
	if c == nil {
		return nil
	}

	v := get(c.MP, key)
	if v == nil {
		return nil
	}

	return &Value{inner: v}
}

type Value struct {
	inner interface{}
}

func (v *Value) Interface() interface{} {
	if v == nil {
		return nil
	}
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

func (v *Value) AsStringSlice() ([]string, error) {
	switch s := v.inner.(type) {
	case []string:
		return s, nil
	case []interface{}:
		var slice []string
		for _, item := range s {
			str, ok := item.(string)
			if !ok {
				return nil, errors.Errorf("can not be string slice: %v", v.inner)
			}
			slice = append(slice, str)
		}
		return slice, nil
	default:
		return nil, errors.Errorf("invalid type: %v", reflect.TypeOf(v.inner))
	}
}

func (v *Value) MustStringSlice() []string {
	value, err := v.AsStringSlice()
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

func del(ms map[string]interface{}, key string) {
	ks := strings.SplitN(key, ".", 2)
	if len(ks) == 1 {
		delete(ms, key)
		return
	}

	v := strKeyMap(ms[ks[0]])
	if v == nil {
		return
	}

	vMap, ok := v.(map[string]interface{})
	if !ok {
		panic(v)
	}

	del(vMap, ks[1])
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

// ParseTSString supports TSO or datetime, e.g. '400036290571534337', '2006-01-02 15:04:05'
func ParseTSString(ts string) (uint64, error) {
	if len(ts) == 0 {
		return 0, nil
	}
	if tso, err := strconv.ParseUint(ts, 10, 64); err == nil {
		return tso, nil
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", ts, time.Local)
	if err != nil {
		t, err = time.Parse(time.RFC3339, ts)
		if err != nil {
			return 0, fmt.Errorf("cannot parse ts string %s, err: %v", ts, err)
		}
	}
	return GoTimeToTS(t), nil
}

// GoTimeToTS converts a Go time to uint64 timestamp.
// port from tidb.
func GoTimeToTS(t time.Time) uint64 {
	ts := (t.UnixNano() / int64(time.Millisecond)) << 18
	return uint64(ts)
}
