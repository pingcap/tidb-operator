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

package pd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMBFromText(t *testing.T) {
	value := uint64(512)
	result := ParseMBFromText("1.0 MiB", value)
	assert.Equal(t, uint64(1), result)

	result = ParseMBFromText("invalid", value)
	assert.Equal(t, value, result)
}

func TestByteSize(t *testing.T) {
	b := ByteSize(17 * 1024 * 1024) // 17 MiB
	data, err := b.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `"17MiB"`, string(data))

	var b2 ByteSize
	err = b2.UnmarshalJSON(data)
	require.NoError(t, err)
	assert.Equal(t, b, b2)

	data2 := []byte("17MiB")
	err = b2.UnmarshalText(data2)
	require.NoError(t, err)
	assert.Equal(t, b, b2)
}
