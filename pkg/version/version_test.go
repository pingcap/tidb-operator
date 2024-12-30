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

package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetVersion(t *testing.T) {
	gitVersion = "v2.0.0"
	gitCommit = "1234567"
	gitTreeState = "clean"
	buildDate = "2024-01-01T00:00:00Z"

	info := Get()
	assert.Equal(t, "v2.0.0", info.GitVersion)
	assert.Equal(t, "1234567", info.GitCommit)
	assert.Equal(t, "clean", info.GitTreeState)
	assert.Equal(t, "2024-01-01T00:00:00Z", info.BuildDate)

	kvs := info.KeysAndValues()
	assert.Len(t, kvs, 14)
	assert.Equal(t, "gitVersion", kvs[0])
	assert.Equal(t, "v2.0.0", kvs[1])
	assert.Equal(t, "gitCommit", kvs[2])
	assert.Equal(t, "1234567", kvs[3])
	assert.Equal(t, "gitTreeState", kvs[4])
	assert.Equal(t, "clean", kvs[5])
	assert.Equal(t, "buildDate", kvs[6])
	assert.Equal(t, "2024-01-01T00:00:00Z", kvs[7])

	str := info.String()
	assert.Contains(t, str, "v2.0.0")
	assert.Contains(t, str, "1234567")
	assert.Contains(t, str, "clean")
	assert.Contains(t, str, "2024-01-01T00:00:00Z")
}
