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

package ticdcapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTiCDCClient_GetStatus(t *testing.T) {
	jsonStr := `
{
    "version": "v8.5.0",
    "git_hash": "3703b18c6da785d57236a1452eed43759e51b411",
    "id": "44dd2708-faee-486b-8b4c-260f61bfb2d9",
    "pid": 1,
    "is_owner": true
}
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/status", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		_, err := w.Write([]byte(jsonStr))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewTiCDCClient(server.URL, "basic-ticdc-0", 5*time.Second, nil)
	status, err := client.GetStatus(context.Background())
	require.NoError(t, err)
	assert.True(t, status.IsOwner)
}

func TestTiCDCClient_DrainCapture(t *testing.T) {
	capturesJsonStr := `
[
	{
		"id": "44dd2708-faee-486b-8b4c-260f61bfb2d9",
		"is_owner": true,
		"address": "basic-ticdc-0.basic-ticdc-peer.default.svc:8301",
		"cluster_id": "default"
	}
]
`
	// only one capture
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/captures", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		_, err := w.Write([]byte(capturesJsonStr))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewTiCDCClient(server.URL, "basic-ticdc-0", 5*time.Second, nil)
	tableCount, err := client.DrainCapture(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, tableCount)

	capturesJsonStr = `
[
	{
		"id": "44dd2708-faee-486b-8b4c-260f61bfb2d9",
		"is_owner": true,
		"address": "basic-ticdc-0.basic-ticdc-peer.default.svc:8301",
		"cluster_id": "default"
	},
	{
		"id": "5a5b4c48-8ab0-41f7-bd59-a8bd233f1e38",
		"is_owner": false,
		"address": "basic-ticdc-1.basic-ticdc-peer.default.svc:8301",
		"cluster_id": "default"
	}
]
`
	// two captures, resign owner
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/captures" {
			assert.Equal(t, http.MethodGet, r.Method)
			_, err := w.Write([]byte(capturesJsonStr))
			assert.NoError(t, err)
		} else if r.URL.Path == "/api/v1/captures/drain" {
			assert.Equal(t, http.MethodPut, r.Method)
			_, err := w.Write([]byte(`{"current_table_count": 0}`))
			assert.NoError(t, err)
		}
	}))
	defer server2.Close()

	client2 := NewTiCDCClient(server2.URL, "basic-ticdc-1", 5*time.Second, nil)
	tableCount, err = client2.DrainCapture(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, tableCount)

	// do not support this API
	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/captures" {
			assert.Equal(t, http.MethodGet, r.Method)
			_, err := w.Write([]byte(capturesJsonStr))
			assert.NoError(t, err)
		} else if r.URL.Path == "/api/v1/captures/drain" {
			assert.Equal(t, http.MethodPut, r.Method)
			w.WriteHeader(http.StatusNotImplemented)
		}
	}))
	defer server3.Close()

	client3 := NewTiCDCClient(server3.URL, "basic-ticdc-0", 5*time.Second, nil)
	tableCount, err = client3.DrainCapture(context.Background())
	require.Error(t, err)
	assert.Equal(t, 0, tableCount)
}

func TestTiCDCClient_ResignOwner(t *testing.T) {
	capturesJsonStr := `
[
	{
		"id": "44dd2708-faee-486b-8b4c-260f61bfb2d9",
		"is_owner": true,
		"address": "basic-ticdc-0.basic-ticdc-peer.default.svc:8301",
		"cluster_id": "default"
	}
]
`
	// only one capture
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/captures", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		_, err := w.Write([]byte(capturesJsonStr))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewTiCDCClient(server.URL, "basic-ticdc-0", 5*time.Second, nil)
	ok, err := client.ResignOwner(context.Background())
	require.NoError(t, err)
	assert.True(t, ok)

	capturesJsonStr = `
[
	{
		"id": "44dd2708-faee-486b-8b4c-260f61bfb2d9",
		"is_owner": true,
		"address": "basic-ticdc-0.basic-ticdc-peer.default.svc:8301",
		"cluster_id": "default"
	},
	{
		"id": "5a5b4c48-8ab0-41f7-bd59-a8bd233f1e38",
		"is_owner": false,
		"address": "basic-ticdc-1.basic-ticdc-peer.default.svc:8301",
		"cluster_id": "default"
	}
]
`
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/captures" {
			assert.Equal(t, http.MethodGet, r.Method)
			_, err := w.Write([]byte(capturesJsonStr))
			assert.NoError(t, err)
		} else if r.URL.Path == "/api/v1/owner/resign" {
			assert.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server2.Close()

	// two captures, current capture is owner
	client2 := NewTiCDCClient(server2.URL, "basic-ticdc-0", 5*time.Second, nil)
	ok, err = client2.ResignOwner(context.Background())
	require.NoError(t, err)
	assert.False(t, ok)

	// two captures, current capture is not owner
	client3 := NewTiCDCClient(server2.URL, "basic-ticdc-1", 5*time.Second, nil)
	ok, err = client3.ResignOwner(context.Background())
	require.NoError(t, err)
	assert.True(t, ok)

	// two captures, but this instance is not found
	client4 := NewTiCDCClient(server2.URL, "basic-ticdc-2", 5*time.Second, nil)
	ok, err = client4.ResignOwner(context.Background())
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestTiCDCClient_IsHealthy(t *testing.T) {
	// no way to get capture info, ignore
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/captures", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		_, err := w.Write([]byte("[]"))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewTiCDCClient(server.URL, "basic-ticdc-0", 5*time.Second, nil)
	ok, err := client.IsHealthy(context.Background())
	require.NoError(t, err)
	assert.True(t, ok)

	capturesJsonStr := `
[
	{
		"id": "44dd2708-faee-486b-8b4c-260f61bfb2d9",
		"is_owner": false,
		"address": "basic-ticdc-0.basic-ticdc-peer.default.svc:8301",
		"cluster_id": "default"
	}
]
`
	// no owner
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/captures", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		_, err := w.Write([]byte(capturesJsonStr))
		assert.NoError(t, err)
	}))
	defer server2.Close()

	client2 := NewTiCDCClient(server2.URL, "basic-ticdc-0", 5*time.Second, nil)
	ok, err = client2.IsHealthy(context.Background())
	require.NoError(t, err)
	assert.False(t, ok)

	capturesJsonStr = `
[
	{
		"id": "44dd2708-faee-486b-8b4c-260f61bfb2d9",
		"is_owner": true,
		"address": "basic-ticdc-0.basic-ticdc-peer.default.svc:8301",
		"cluster_id": "default"
	}
]
`
	// owner exists and healthy
	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/captures" {
			assert.Equal(t, http.MethodGet, r.Method)
			_, err := w.Write([]byte(capturesJsonStr))
			assert.NoError(t, err)
		} else if r.URL.Path == "/api/v1/health" {
			assert.Equal(t, http.MethodGet, r.Method)
			_, err := w.Write([]byte("ok"))
			assert.NoError(t, err)
		}
	}))
	defer server3.Close()

	client3 := NewTiCDCClient(server3.URL, "basic-ticdc-0", 5*time.Second, nil)
	ok, err = client3.IsHealthy(context.Background())
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestTiCDCClient_getCaptures(t *testing.T) {
	jsonStr := `
[
    {
        "id": "44dd2708-faee-486b-8b4c-260f61bfb2d9",
        "is_owner": true,
        "address": "basic-ticdc-0.basic-ticdc-peer.default.svc:8301",
        "cluster_id": "default"
    },
	{
        "id": "5a5b4c48-8ab0-41f7-bd59-a8bd233f1e38",
        "is_owner": false,
        "address": "basic-ticdc-1.basic-ticdc-peer.default.svc:8301",
        "cluster_id": "default"
    }
]
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/captures", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		_, err := w.Write([]byte(jsonStr))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewTiCDCClient(server.URL, "basic-ticdc-1", 5*time.Second, nil)
	captures, err := client.(*ticdcClient).getCaptures(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, len(captures))
	assert.Equal(t, "44dd2708-faee-486b-8b4c-260f61bfb2d9", captures[0].ID)
	assert.Equal(t, "5a5b4c48-8ab0-41f7-bd59-a8bd233f1e38", captures[1].ID)
	assert.Equal(t, "basic-ticdc-0.basic-ticdc-peer.default.svc:8301", captures[0].AdvertiseAddr)
	assert.Equal(t, "basic-ticdc-1.basic-ticdc-peer.default.svc:8301", captures[1].AdvertiseAddr)
	assert.True(t, captures[0].IsOwner)
	assert.False(t, captures[1].IsOwner)
}

func Test_getThisAndOwnerCaptureInfo(t *testing.T) {
	captures := []captureInfo{
		{
			ID:            "44dd2708-faee-486b-8b4c-260f61bfb2d9",
			IsOwner:       true,
			AdvertiseAddr: "basic-ticdc-0.basic-ticdc-peer.default.svc:8301",
		},
		{
			ID:            "5a5b4c48-8ab0-41f7-bd59-a8bd233f1e38",
			IsOwner:       false,
			AdvertiseAddr: "basic-ticdc-1.basic-ticdc-peer.default.svc:8301",
		},
	}

	cases := []struct {
		name     string
		instance string
		this     captureInfo
		owner    captureInfo
	}{
		{
			name:     "test owner capture",
			instance: "basic-ticdc-0",
			this:     captures[0],
			owner:    captures[0],
		},
		{
			name:     "test non-owner capture",
			instance: "basic-ticdc-1",
			this:     captures[1],
			owner:    captures[0],
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			this, owner := getThisAndOwnerCaptureInfo(c.instance, captures)
			assert.Equal(t, c.this, *this)
			assert.Equal(t, c.owner, *owner)
		})
	}
}

/*
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
}*/
