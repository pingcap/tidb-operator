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

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
)

type TestType struct {
	String string `toml:"string"`
}

func TestCodec(t *testing.T) {
	cases := []struct {
		desc   string
		input  []byte
		output []byte
		obj    TestType
		change func(obj *TestType) *TestType
	}{
		{
			desc:  "empty input",
			input: []byte(``),
			output: []byte(`
string = 'changed'
            `),
			change: func(obj *TestType) *TestType {
				obj.String = "changed"
				return obj
			},
		},
		{
			desc: "reserve unknown field",
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
string = 'mmm'
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
				obj.String = "mmm"
				return obj
			},
		},
		{
			desc: "change existing field",
			input: []byte(`
string = 'aaa'
unknown = 'xxx'
            `),
			output: []byte(`
string = 'yyy'
unknown = 'xxx'
            `),
			change: func(obj *TestType) *TestType {
				obj.String = "yyy"
				return obj
			},
		},
	}

	for i := range cases {
		c := &cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			tt.Parallel()

			decoder, encoder := Codec[TestType]()
			require.NoError(t, decoder.Decode(c.input, &c.obj))
			c.change(&c.obj)
			res, err := encoder.Encode(&c.obj)
			require.NoError(tt, err)
			assert.Equal(tt, string(bytes.TrimSpace(c.output)), string(bytes.TrimSpace(res)))
		})
	}
}

func TestGenerateHash(t *testing.T) {
	tests := []struct {
		name                      string
		tomlStr                   v1alpha1.ConfigFile
		semanticallyEquivalentStr v1alpha1.ConfigFile
		wantHash                  string
		wantError                 bool
	}{
		{
			name: "Valid TOML string",
			tomlStr: v1alpha1.ConfigFile(`foo = 'bar'
[log]
k1 = 'v1'
k2 = 'v2'`),
			semanticallyEquivalentStr: v1alpha1.ConfigFile(`foo = 'bar'
[log]
k2 = 'v2'
k1 = 'v1'`),
			wantHash:  "5dbbcf4574",
			wantError: false,
		},
		{
			name: "Different config value",
			tomlStr: v1alpha1.ConfigFile(`foo = 'foo'
[log]
k2 = 'v2'
k1 = 'v1'`),
			wantHash:  "f5bc46cb9",
			wantError: false,
		},
		{
			name: "multiple sections with blank line",
			tomlStr: v1alpha1.ConfigFile(`[a]
k1 = 'v1'
[b]
k2 = 'v2'`),
			semanticallyEquivalentStr: v1alpha1.ConfigFile(`[a]
k1 = 'v1'
[b]

k2 = 'v2'`),
			wantHash:  "79598d5977",
			wantError: false,
		},
		{
			name:      "Empty TOML string",
			tomlStr:   v1alpha1.ConfigFile(``),
			wantHash:  "7d6fc488b7",
			wantError: false,
		},
		{
			name: "Invalid TOML string",
			tomlStr: v1alpha1.ConfigFile(`key1 = "value1"
			key2 = value2`), // Missing quotes around value2
			wantHash:  "",
			wantError: true,
		},
		{
			name: "Nested tables",
			tomlStr: v1alpha1.ConfigFile(`[parent]
child1 = "value1"
child2 = "value2"
[parent.child]
grandchild1 = "value3"
grandchild2 = "value4"`),
			semanticallyEquivalentStr: v1alpha1.ConfigFile(`[parent]
child2 = "value2"
child1 = "value1"
[parent.child]
grandchild2 = "value4"
grandchild1 = "value3"`),
			wantHash:  "7bf645ccb4",
			wantError: false,
		},
		{
			name: "Array of tables",
			tomlStr: v1alpha1.ConfigFile(`[[products]]
name = "Hammer"
sku = 738594937

[[products]]
name = "Nail"
sku = 284758393

color = "gray"`),
			semanticallyEquivalentStr: v1alpha1.ConfigFile(`[[products]]
sku = 738594937
name = "Hammer"

[[products]]
sku = 284758393
name = "Nail"

color = "gray"`),
			wantHash:  "7549cf87f4",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHash, err := GenerateHash(tt.tomlStr)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantHash, gotHash)

				if string(tt.semanticallyEquivalentStr) != "" {
					reorderedHash, err := GenerateHash(tt.semanticallyEquivalentStr)
					require.NoError(t, err)
					assert.Equal(t, tt.wantHash, reorderedHash)
				}
			}
		})
	}
}
