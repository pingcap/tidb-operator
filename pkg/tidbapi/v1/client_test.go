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
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTiDBClient_GetHealth(t *testing.T) {
	jsonStr := `{"connections":0,"version":"8.0.11-TiDB-v8.1.0","git_hash":"945d07c5d5c7a1ae212f6013adfb187f2de24b23"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/status", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		_, err := w.Write([]byte(jsonStr))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewTiDBClient(server.URL, 5*time.Second, nil)
	health, err := client.GetHealth(context.Background())
	require.NoError(t, err)
	assert.True(t, health)
}

func TestTiDBClient_GetInfo(t *testing.T) {
	jsonStr := `
{
 "is_owner": true,
 "max_procs": 10,
 "gogc": 500,
 "version": "8.0.11-TiDB-v8.1.0",
 "git_hash": "945d07c5d5c7a1ae212f6013adfb187f2de24b23",
 "ddl_id": "fe4de332-a1c5-46ba-a1d0-762c716345d3",
 "ip": "basic-tidb-9lbgl4.basic-tidb-peer.default.svc",
 "listening_port": 4000,
 "status_port": 10080,
 "lease": "45s",
 "binlog_status": "Off",
 "start_timestamp": 1735095910,
 "labels": {},
 "server_id": 85
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/info", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		_, err := w.Write([]byte(jsonStr))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewTiDBClient(server.URL, 5*time.Second, nil)
	info, err := client.GetInfo(context.Background())
	require.NoError(t, err)
	assert.True(t, info.IsOwner)
}

func TestTiDBClient_SetServerLabels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/labels", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewTiDBClient(server.URL, 5*time.Second, nil)
	err := client.SetServerLabels(context.Background(), map[string]string{"region": "us-west-1"})
	require.NoError(t, err)
}

func TestTiDBClient_Activate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/tidb-pool/activate", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.JSONEq(t,
			`{"keyspace_name":"xxx","max_idle_seconds":0,"run_auto_analyze":true,"tidb_enable_ddl":true}`,
			string(body),
		)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewTiDBClient(server.URL, 5*time.Second, nil)
	err := client.Activate(context.Background(), "xxx")
	require.NoError(t, err)
}

func TestTiDBClient_GetPoolStatus(t *testing.T) {
	cases := []struct {
		desc     string
		status   int
		body     string
		expected PoolStatus
	}{
		{
			desc:   "activated",
			status: http.StatusOK,
			body:   `{"state":"activated","keyspace_name":"xxx"}`,
			expected: PoolStatus{
				State: PoolStateActivated,
			},
		},
		{
			desc:   "standby",
			status: http.StatusOK,
			body:   `{"state":"standby","keyspace_name":""}`,
			expected: PoolStatus{
				State: PoolStateStandBy,
			},
		},
		{
			desc:   "not in standby mode or not support standby mode",
			status: http.StatusNotFound,
			expected: PoolStatus{
				State: PoolStateActivated,
			},
		},
	}
	for i := range cases {
		c := cases[i]
		t.Run(c.desc, func(tt *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(tt, "/tidb-pool/status", r.URL.Path)
				assert.Equal(tt, http.MethodGet, r.Method)
				w.WriteHeader(c.status)
				if c.body != "" {
					_, err := w.Write([]byte(c.body))
					assert.NoError(tt, err)
				}
			}))
			defer server.Close()

			client := NewTiDBClient(server.URL, 5*time.Second, nil)
			s, err := client.GetPoolStatus(context.Background())
			require.NoError(tt, err)
			assert.Equal(tt, &c.expected, s)
		})
	}
}
