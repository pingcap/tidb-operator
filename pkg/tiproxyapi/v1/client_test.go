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

package tiproxyapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	stringutil "github.com/pingcap/tidb-operator/v2/pkg/utils/string"
)

func TestTiProxyClient_IsHealthy(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedHealth bool
	}{
		{
			name:           "healthy on 200",
			statusCode:     http.StatusOK,
			expectedHealth: true,
		},
		{
			name:           "unhealthy on 502",
			statusCode:     http.StatusBadGateway,
			expectedHealth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonStr := `{"config_checksum":3006078629}`
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/debug/health", r.URL.Path)
				assert.Equal(t, http.MethodGet, r.Method)
				w.WriteHeader(tt.statusCode)
				_, err := w.Write([]byte(jsonStr))
				assert.NoError(t, err)
			}))
			defer server.Close()

			addr := stringutil.RemoveHTTPPrefix(server.URL)
			client := NewTiProxyClient(addr, 5*time.Second, nil)
			health, err := client.IsHealthy(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tt.expectedHealth, health)
		})
	}
}

func TestTiProxyClient_SetLabels(t *testing.T) {
	tests := []struct {
		name           string
		labels         map[string]string
		expectedBody   string
		serverResponse int
		expectError    bool
	}{
		{
			name:   "single label",
			labels: map[string]string{"region": "us-west-1"},
			expectedBody: `[labels]
  region = "us-west-1"
`,
			serverResponse: http.StatusOK,
			expectError:    false,
		},
		{
			name:   "multiple labels",
			labels: map[string]string{"region": "us-west-1", "env": "prod", "version": "v1.0.0"},
			expectedBody: `[labels]
  env = "prod"
  region = "us-west-1"
  version = "v1.0.0"
`,
			serverResponse: http.StatusOK,
			expectError:    false,
		},
		{
			name:   "empty labels",
			labels: map[string]string{},
			expectedBody: `[labels]
`,
			serverResponse: http.StatusOK,
			expectError:    false,
		},
		{
			name:   "server error",
			labels: map[string]string{"region": "us-west-1"},
			expectedBody: `[labels]
  region = "us-west-1"
`,
			serverResponse: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/admin/config", r.URL.Path)
				assert.Equal(t, http.MethodPut, r.Method)
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBody, string(body))
				w.WriteHeader(tt.serverResponse)
			}))
			defer server.Close()

			addr := stringutil.RemoveHTTPPrefix(server.URL)
			client := NewTiProxyClient(addr, 5*time.Second, nil)
			err := client.SetLabels(context.Background(), tt.labels)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTiProxyClient_MarkUnhealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/debug/health", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)
		body, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err) {
			return
		}
		assert.JSONEq(t, `{"healthy":false,"reason":"graceful-shutdown"}`, string(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	addr := stringutil.RemoveHTTPPrefix(server.URL)
	client := NewTiProxyClient(addr, 5*time.Second, nil)
	err := client.MarkUnhealthy(context.Background())
	require.NoError(t, err)
}

func TestTiProxyClient_ConnectionCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/metrics", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		_, err := w.Write([]byte(`# HELP tiproxy_server_connections Number of connections.
# TYPE tiproxy_server_connections gauge
tiproxy_server_connections 7
`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	addr := stringutil.RemoveHTTPPrefix(server.URL)
	client := NewTiProxyClient(addr, 5*time.Second, nil)
	count, err := client.ConnectionCount(context.Background())
	require.NoError(t, err)
	assert.InDelta(t, 7, count, 0)
}
