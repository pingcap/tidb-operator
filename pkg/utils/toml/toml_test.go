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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestType struct {
	String string `toml:"string,omitempty"`

	Struct *TestType  `toml:"struct"`
	Array  []TestType `toml:"array,omitempty"`

	unexported string
}

func TestCodec(t *testing.T) {
	cases := []struct {
		desc         string
		input        []byte
		output       []byte
		obj          TestType
		change       func(obj *TestType) *TestType
		hasDecodeErr bool
	}{
		{
			desc:  "empty input",
			input: []byte(``),
			output: []byte(`
string = 'changed'

[[array]]
string = 'arr1'

[[array]]
string = 'arr2'

[struct]
string = 'struct'
            `),
			change: func(obj *TestType) *TestType {
				obj.String = "changed"
				obj.Struct = &TestType{
					String: "struct",
				}
				obj.Array = []TestType{
					{
						String: "arr1",
					},
					{
						String: "arr2",
					},
				}
				return obj
			},
		},
		{
			desc: "reserve unknown fields",
			input: []byte(`
unknown = 'xxx'
unknown_int = 10

[[unknown_arr]]
unknown = 'yyy'

[[unknown_arr]]
unknown = 'yyy'

[unknown_double_map]
[unknown_double_map.map]
unknown = 'yyy'

[unknown_map]
unknown = 'yyy'
            `),
			output: []byte(`
string = 'changed'
unknown = 'xxx'
unknown_int = 10

[[unknown_arr]]
unknown = 'yyy'

[[unknown_arr]]
unknown = 'yyy'

[unknown_double_map]
[unknown_double_map.map]
unknown = 'yyy'

[unknown_map]
unknown = 'yyy'
            `),
			change: func(obj *TestType) *TestType {
				obj.String = "changed"
				return obj
			},
		},
		{
			desc: "change string and reserve unknown fields",
			input: []byte(`
string = 'aaa'
unknown = 'xxx'
            `),
			output: []byte(`
string = 'changed'
unknown = 'xxx'
            `),
			change: func(obj *TestType) *TestType {
				obj.String = "changed"
				return obj
			},
		},
		{
			desc: "add struct",
			input: []byte(`
string = 'aaa'
unknown = 'xxx'
            `),
			output: []byte(`
string = 'aaa'
unknown = 'xxx'

[struct]
string = 'changed'
            `),
			change: func(obj *TestType) *TestType {
				obj.Struct = &TestType{
					String: "changed",
				}
				return obj
			},
		},
		{
			desc: "change struct and reserve unknown fields",
			input: []byte(`
string = 'aaa'
unknown = 'xxx'

[struct]
string = 'aaa'
unknown = 'xxx'
            `),
			output: []byte(`
string = 'aaa'
unknown = 'xxx'

[struct]
string = 'changed'
unknown = 'xxx'
            `),
			change: func(obj *TestType) *TestType {
				obj.Struct = &TestType{
					String: "changed",
				}
				return obj
			},
		},
		{
			desc: "add struct in struct",
			input: []byte(`
string = 'aaa'
unknown = 'xxx'
            `),
			output: []byte(`
string = 'aaa'
unknown = 'xxx'

[struct]
[struct.struct]
string = 'changed'
            `),
			change: func(obj *TestType) *TestType {
				obj.Struct = &TestType{
					Struct: &TestType{
						String: "changed",
					},
				}
				return obj
			},
		},
		{
			desc: "change struct and add struct in struct",
			input: []byte(`
string = 'aaa'
unknown = 'xxx'

[struct]
string = 'aaa'
unknown = 'xxx'
            `),
			output: []byte(`
string = 'aaa'
unknown = 'xxx'

[struct]
string = 'aaa'
unknown = 'xxx'

[struct.struct]
string = 'changed'
            `),
			change: func(obj *TestType) *TestType {
				obj.Struct = &TestType{
					Struct: &TestType{
						String: "changed",
					},
				}
				return obj
			},
		},
		{
			desc: "change struct in struct and reserve unknown fields",
			input: []byte(`
string = 'aaa'
unknown = 'xxx'

[struct]
[struct.struct]
string = 'aaa'
unknown = 'xxx'
            `),
			output: []byte(`
string = 'aaa'
unknown = 'xxx'

[struct]
[struct.struct]
string = 'changed'
unknown = 'xxx'
            `),
			change: func(obj *TestType) *TestType {
				obj.Struct = &TestType{
					Struct: &TestType{
						String: "changed",
					},
				}
				return obj
			},
		},
		{
			desc: "add array in struct",
			input: []byte(`
string = 'aaa'
unknown = 'xxx'
            `),
			output: []byte(`
string = 'aaa'
unknown = 'xxx'

[struct]
[[struct.array]]
string = 'changed'
            `),
			change: func(obj *TestType) *TestType {
				obj.Struct = &TestType{
					Array: []TestType{
						{
							String: "changed",
						},
					},
				}
				return obj
			},
		},
		{
			desc: "change array in struct and reserve unknown fields",
			input: []byte(`
string = 'aaa'
unknown = 'xxx'

[struct]
[[struct.array]]
string = 'aaa'
unknown = 'xxx'
            `),
			output: []byte(`
string = 'aaa'
unknown = 'xxx'

[struct]
[[struct.array]]
string = 'changed'
unknown = 'xxx'
            `),
			change: func(obj *TestType) *TestType {
				obj.Struct = &TestType{
					Array: []TestType{
						{
							String: "changed",
						},
					},
				}
				return obj
			},
		},
		{
			desc: "add array",
			input: []byte(`
string = 'aaa'
unknown = 'xxx'
            `),
			output: []byte(`
string = 'aaa'
unknown = 'xxx'

[[array]]
string = 'changed'
            `),
			change: func(obj *TestType) *TestType {
				obj.Array = []TestType{
					{
						String: "changed",
					},
				}
				return obj
			},
		},
		{
			desc: "change array and reserve unknown fields",
			input: []byte(`
string = 'aaa'
unknown = 'xxx'

[[array]]
string = 'aaa'
unknown = 'xxx'
            `),
			output: []byte(`
string = 'aaa'
unknown = 'xxx'

[[array]]
string = 'changed'
unknown = 'xxx'
            `),
			change: func(obj *TestType) *TestType {
				obj.Array = []TestType{
					{
						String: "changed",
					},
				}
				return obj
			},
		},
		{
			desc: "append new item to array",
			input: []byte(`
string = 'aaa'
unknown = 'xxx'

[[array]]
string = 'aaa'
unknown = 'xxx'
            `),
			output: []byte(`
string = 'aaa'
unknown = 'xxx'

[[array]]
string = 'aaa'
unknown = 'xxx'

[[array]]
string = 'changed'
            `),
			change: func(obj *TestType) *TestType {
				obj.Array = append(obj.Array, TestType{
					String: "changed",
				})
				return obj
			},
		},
		{
			desc: "del existing item in array(useless)",
			input: []byte(`
string = 'aaa'
unknown = 'xxx'

[[array]]
string = 'aaa'
unknown = 'xxx'

[[array]]
string = 'bbb'
unknown = 'xxx'
            `),
			output: []byte(`
string = 'aaa'
unknown = 'xxx'

[[array]]
string = 'aaa'
unknown = 'xxx'

[[array]]
string = 'bbb'
unknown = 'xxx'
            `),
			change: func(obj *TestType) *TestType {
				obj.Array = obj.Array[:1]
				return obj
			},
		},
		{
			desc: "add struct in array",
			input: []byte(`
string = 'aaa'
unknown = 'xxx'

[[array]]
string = 'aaa'
unknown = 'xxx'
            `),
			output: []byte(`
string = 'aaa'
unknown = 'xxx'

[[array]]
string = 'aaa'
unknown = 'xxx'

[array.struct]
string = 'changed'
            `),
			change: func(obj *TestType) *TestType {
				obj.Array[0].Struct = &TestType{
					String: "changed",
				}
				return obj
			},
		},
		{
			desc: "change struct in array",
			input: []byte(`
string = 'aaa'
unknown = 'xxx'

[[array]]
string = 'aaa'
unknown = 'xxx'

[array.struct]
string = 'aaa'
unknown = 'xxx'
            `),
			output: []byte(`
string = 'aaa'
unknown = 'xxx'

[[array]]
string = 'aaa'
unknown = 'xxx'

[array.struct]
string = 'changed'
unknown = 'xxx'
            `),
			change: func(obj *TestType) *TestType {
				obj.Array[0].Struct = &TestType{
					String: "changed",
				}
				return obj
			},
		},
		{
			desc:         "invalid toml",
			input:        []byte(`xxx`),
			hasDecodeErr: true,
		},
		{
			desc: "mismatched type",
			input: []byte(`
struct = 'xxx'
            `),
			hasDecodeErr: true,
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			decoder, encoder := Codec[TestType]()
			err := decoder.Decode(c.input, &c.obj)
			if !c.hasDecodeErr {
				require.NoError(tt, err, c.desc)
			} else {
				assert.Error(tt, err, c.desc)
				return
			}
			c.change(&c.obj)
			res, err := encoder.Encode(&c.obj)
			require.NoError(tt, err, c.desc)
			assert.Equal(tt, string(bytes.TrimSpace(c.output)), string(bytes.TrimSpace(res)))
		})
	}
}
