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

package hasher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

func TestGenerateHash(t *testing.T) {
	tests := []struct {
		name              string
		tomlStrings       []v1alpha1.ConfigFile
		comparedStrings   []v1alpha1.ConfigFile
		shouldGetSameHash bool
		wantHash          string
		wantError         bool
	}{
		// Single input
		{
			name: "Valid TOML string",
			tomlStrings: []v1alpha1.ConfigFile{
				v1alpha1.ConfigFile(`foo = 'bar'
[log]
k1 = 'v1'
k2 = 'v2'`)},
			comparedStrings: []v1alpha1.ConfigFile{
				v1alpha1.ConfigFile(`foo = 'bar'
[log]
k2 = 'v2'
k1 = 'v1'`)},
			wantHash:          "6cc5875ff8",
			shouldGetSameHash: true,
			wantError:         false,
		},
		{
			name: "Different config value",
			tomlStrings: []v1alpha1.ConfigFile{
				v1alpha1.ConfigFile(`foo = 'foo'
[log]
k2 = 'v2'
k1 = 'v1'`)},
			wantHash:  "7dd86884c",
			wantError: false,
		},
		{
			name: "multiple sections with blank line",
			tomlStrings: []v1alpha1.ConfigFile{
				v1alpha1.ConfigFile(`[a]
k1 = 'v1'
[b]
k2 = 'v2'`)},
			comparedStrings: []v1alpha1.ConfigFile{
				v1alpha1.ConfigFile(`[a]
k1 = 'v1'
[b]

k2 = 'v2'`)},
			shouldGetSameHash: true,
			wantHash:          "6577c5d475",
			wantError:         false,
		},
		{
			name:        "Empty TOML string",
			tomlStrings: []v1alpha1.ConfigFile{v1alpha1.ConfigFile(``)},
			wantHash:    "76b94dc7f9",
			wantError:   false,
		},
		{
			name: "Invalid TOML string",
			tomlStrings: []v1alpha1.ConfigFile{
				v1alpha1.ConfigFile(`key1 = "value1"
key2 = value2`), // Missing quotes around value2
			},
			wantHash:  "",
			wantError: true,
		},
		{
			name: "Nested tables",
			tomlStrings: []v1alpha1.ConfigFile{
				v1alpha1.ConfigFile(`[parent]
child1 = "value1"
child2 = "value2"
[parent.child]
grandchild1 = "value3"
grandchild2 = "value4"`)},
			comparedStrings: []v1alpha1.ConfigFile{
				v1alpha1.ConfigFile(`[parent]
child2 = "value2"
child1 = "value1"
[parent.child]
grandchild2 = "value4"
grandchild1 = "value3"`)},
			shouldGetSameHash: true,
			wantHash:          "794796f9d",
			wantError:         false,
		},
		{
			name: "Array of tables",
			tomlStrings: []v1alpha1.ConfigFile{
				v1alpha1.ConfigFile(`[[products]]
name = "Hammer"
sku = 738594937

[[products]]
name = "Nail"
sku = 284758393

color = "gray"`)},
			comparedStrings: []v1alpha1.ConfigFile{
				v1alpha1.ConfigFile(`[[products]]
sku = 738594937
name = "Hammer"

[[products]]
sku = 284758393
name = "Nail"

color = "gray"`)},
			shouldGetSameHash: true,
			wantHash:          "7cb6dd5946",
			wantError:         false,
		},
		// Multiple inputs
		{
			name: "both of the inputs change the order of keys",
			tomlStrings: []v1alpha1.ConfigFile{
				v1alpha1.ConfigFile(`[parent]
child1 = "value1"
child2 = "value2"
[parent.child]
grandchild1 = "value3"
grandchild2 = "value4"`),
				v1alpha1.ConfigFile(`[parent]
child1 = "value1"
child2 = "value2"
[parent.child]
grandchild1 = "value3"
grandchild2 = "value4"`)},
			comparedStrings: []v1alpha1.ConfigFile{
				v1alpha1.ConfigFile(`[parent]
child2 = "value2"
child1 = "value1"
[parent.child]
grandchild2 = "value4"
grandchild1 = "value3"`),
				v1alpha1.ConfigFile(`[parent]
child2 = "value2"
child1 = "value1"
[parent.child]
grandchild2 = "value4"
grandchild1 = "value3"`)},
			shouldGetSameHash: true,
			wantHash:          "7c946f9fd6",
			wantError:         false,
		},
		{
			name: "change the order of inputs",
			tomlStrings: []v1alpha1.ConfigFile{
				v1alpha1.ConfigFile(`[parent]
child1 = "value1"
child2 = "value2"
[parent.child]
grandchild1 = "value3"
grandchild2 = "value4"`),
				v1alpha1.ConfigFile(`[foo]
k1 = "value1"`)},
			comparedStrings: []v1alpha1.ConfigFile{
				v1alpha1.ConfigFile(`[foo]
k1 = "value1"`),
				v1alpha1.ConfigFile(`[parent]
child1 = "value1"
child2 = "value2"
[parent.child]
grandchild1 = "value3"
grandchild2 = "value4"`)},
			shouldGetSameHash: false,
			wantHash:          "5c8f696db7",
			wantError:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1, err := GenerateHash(tt.tomlStrings...)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantHash, hash1)

				if len(tt.comparedStrings) > 0 {
					hash2, err := GenerateHash(tt.comparedStrings...)
					require.NoError(t, err)
					if tt.shouldGetSameHash {
						assert.Equal(t, hash1, hash2)
					} else {
						assert.NotEqual(t, hash1, hash2)
					}
				}
			}
		})
	}
}
