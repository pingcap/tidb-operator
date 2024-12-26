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

package tidbapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTiDDBControl(t *testing.T) {
	jsonStr := `{"connections":0,"version":"8.0.11-TiDB-v8.1.0","git_hash":"945d07c5d5c7a1ae212f6013adfb187f2de24b23"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/status", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		_, err := w.Write([]byte(jsonStr))
		assert.NoError(t, err)
	}))
	defer server.Close()

	// only test non-tls
	client, err := NewDefaultTiDBControl().GetTiDBPodClient(context.Background(), nil, "", "", "", "", false)
	require.NoError(t, err)
	client.(*tidbClient).url = server.URL // replace the url with the test server
	health, err := client.GetHealth(context.Background())
	require.NoError(t, err)
	assert.True(t, health)
}
