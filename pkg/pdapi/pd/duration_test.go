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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDuration(t *testing.T) {
	du := NewDuration(72*time.Hour + 3*time.Minute + 500*time.Millisecond)
	data, err := du.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `"72h3m0.5s"`, string(data))
	data2, err := du.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, "72h3m0.5s", string(data2))

	var du2 Duration
	err = du2.UnmarshalJSON(data)
	require.NoError(t, err)
	assert.Equal(t, du, du2)

	err = du2.UnmarshalText(data2)
	require.NoError(t, err)
	assert.Equal(t, du, du2)
}
