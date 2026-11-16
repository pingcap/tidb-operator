// Copyright 2026 PingCAP, Inc.
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

package tikvapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetLeaderCountTextMetrics(t *testing.T) {
	for _, leaderCount := range []int{0, 42} {
		t.Run(fmt.Sprintf("leaders=%d", leaderCount), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/metrics" {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "text/plain; version=0.0.4")
				fmt.Fprintf(w, "# TYPE tikv_raftstore_region_count gauge\ntikv_raftstore_region_count{type=\"region\"} 100\ntikv_raftstore_region_count{type=\"leader\"} %d\n", leaderCount)
			}))
			defer server.Close()

			client := NewTiKVClient(TiKVClientOpts{HTTPEndpoint: server.URL, Timeout: DefaultTimeout})
			count, err := client.GetLeaderCount()
			require.NoError(t, err)
			require.Equal(t, leaderCount, count)
		})
	}
}
