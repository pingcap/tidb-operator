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

package toml

import (
	"bytes"
	"reflect"

	"github.com/BurntSushi/toml"
	"github.com/pingcap/errors"
)

// Equal compare two TOML logically equal.
func Equal(d1 []byte, d2 []byte) (bool, error) {
	m1 := map[string]interface{}{}
	err := Unmarshal(d1, &m1)
	if err != nil {
		return false, err
	}

	m2 := map[string]interface{}{}
	err = Unmarshal(d2, &m2)
	if err != nil {
		return false, err
	}

	return reflect.DeepEqual(m1, m2), nil
}

// Marshal is a template function that try to marshal a go value to toml
func Marshal(v interface{}) ([]byte, error) {
	buff := new(bytes.Buffer)
	encoder := toml.NewEncoder(buff)
	err := encoder.Encode(v)
	if err != nil {
		return nil, errors.AddStack(err)
	}
	data := buff.Bytes()
	return data, nil
}

// Unmarshal decodes the contents of `p` in TOML format into a pointer `v`.
func Unmarshal(p []byte, v interface{}) error {
	err := toml.Unmarshal(p, v)
	if err != nil {
		return errors.AddStack(err)
	}

	return nil
}
