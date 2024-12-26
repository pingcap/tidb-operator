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

package tikvapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTiKVClient_GetLeaderCount(t *testing.T) {
	jsonStr := `
# HELP tikv_raftstore_region_count Number of regions collected in region_collector
# TYPE tikv_raftstore_region_count gauge
tikv_raftstore_region_count{type="buckets"} 50
tikv_raftstore_region_count{type="leader"} 20
tikv_raftstore_region_count{type="region"} 50
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/metrics", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		_, err := w.Write([]byte(jsonStr))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewTiKVClient(server.URL, 5*time.Second, nil, false)
	count, err := client.GetLeaderCount()
	require.NoError(t, err)
	assert.Equal(t, 20, count)
}
