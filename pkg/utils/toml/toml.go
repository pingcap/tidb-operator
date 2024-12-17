// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package toml

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"reflect"

	"github.com/mitchellh/mapstructure"
	"github.com/pelletier/go-toml/v2"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	hashutil "github.com/pingcap/tidb-operator/third_party/kubernetes/pkg/util/hash"
)

type Decoder[T any, PT *T] interface {
	Decode(data []byte, obj PT) error
}

type Encoder[T any, PT *T] interface {
	Encode(obj PT) ([]byte, error)
}

type codec[T any, PT *T] struct {
	raw map[string]any
}

func Codec[T any, PT *T]() (Decoder[T, PT], Encoder[T, PT]) {
	c := &codec[T, PT]{}

	return c, c
}

func (c *codec[T, PT]) Decode(data []byte, obj PT) error {
	raw := make(map[string]any)
	if err := toml.NewDecoder(bytes.NewReader(data)).Decode(&raw); err != nil {
		return err
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "toml",
		Result:  obj,
	})
	if err != nil {
		return err
	}
	if err := decoder.Decode(raw); err != nil {
		return err
	}

	c.raw = raw

	return nil
}

func (c *codec[T, PT]) Encode(obj PT) ([]byte, error) {
	if err := Overwrite(obj, c.raw); err != nil {
		return nil, err
	}

	buf := bytes.Buffer{}
	if err := toml.NewEncoder(&buf).Encode(c.raw); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func Overwrite(obj any, m map[string]any) error {
	structVal := reflect.ValueOf(obj).Elem()
	fieldTypes := reflect.VisibleFields(structVal.Type())
	for _, fieldType := range fieldTypes {
		if !fieldType.IsExported() {
			continue
		}

		name := fieldType.Tag.Get("toml")
		src := structVal.FieldByIndex(fieldType.Index)

		for src.Kind() == reflect.Pointer {
			src = src.Elem()
		}

		if src.IsZero() {
			continue
		}

		v, ok := m[name]
		if !ok {
			m[name] = src.Interface()
			continue
		}

		val, err := getField(src, v)
		if err != nil {
			return err
		}
		m[name] = val
	}

	return nil
}

func getField(src reflect.Value, dst any) (any, error) {
	switch src.Kind() {
	case reflect.Struct:
		vm, ok := dst.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("type mismatched, expected map, actual %T", dst)
		}
		if err := Overwrite(src.Addr().Interface(), vm); err != nil {
			return nil, err
		}

		return vm, nil
	case reflect.Slice, reflect.Array:
		vs, ok := dst.([]any)
		if !ok {
			return nil, fmt.Errorf("type mismatched, expected array or slice, actual %T", dst)
		}
		for i := range vs {
			if i >= src.Len() {
				break
			}
			srcIndex := src.Index(i)
			val, err := getField(srcIndex, vs[i])
			if err != nil {
				return nil, err
			}
			vs[i] = val
		}
		if len(vs) < src.Len() {
			for i := len(vs); i < src.Len(); i++ {
				vs = append(vs, src.Index(i).Interface())
			}
		}
		return vs, nil
	default:
		return src.Interface(), nil
	}
}

// GenerateHash takes a TOML string as input, unmarshals it into a map,
// and generates a hash of the resulting configuration. The hash is then
// encoded into a safe string format and returned.
// If the order of keys in the TOML string is different, the hash will be the same.
func GenerateHash(tomlStr v1alpha1.ConfigFile) (string, error) {
	var config map[string]any
	if err := toml.NewDecoder(bytes.NewReader([]byte(tomlStr))).Decode(&config); err != nil {
		return "", fmt.Errorf("failed to unmarshal toml string %s: %w", tomlStr, err)
	}
	hasher := fnv.New32a()
	hashutil.DeepHashObject(hasher, config)
	return rand.SafeEncodeString(fmt.Sprint(hasher.Sum32())), nil
}
