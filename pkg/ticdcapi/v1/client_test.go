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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	capturesURL        = "/api/v1/captures"
	capturesOneJSONStr = `
[
	{
		"id": "44dd2708-faee-486b-8b4c-260f61bfb2d9",
		"is_owner": true,
		"address": "basic-ticdc-0.basic-ticdc-peer.default.svc:8301",
		"cluster_id": "default"
	}
]
`

	capturesTwoJSONStr = `
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

	client := NewTiCDCClient("basic-ticdc-0.basic-ticdc-peer.default.svc:8301", withURL(server.URL))
	status, err := client.GetStatus(context.Background())
	require.NoError(t, err)
	assert.True(t, status.IsOwner)
}

func TestTiCDCClient_DrainCapture(t *testing.T) {
	// only one capture
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, capturesURL, r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		_, err := w.Write([]byte(capturesOneJSONStr))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewTiCDCClient("basic-ticdc-0.basic-ticdc-peer.default.svc:8301", withURL(server.URL))
	tableCount, err := client.DrainCapture(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, tableCount)

	// two captures, resign owner
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case capturesURL:
			assert.Equal(t, http.MethodGet, r.Method)
			_, err = w.Write([]byte(capturesTwoJSONStr))
			assert.NoError(t, err)
		case "/api/v1/captures/drain":
			assert.Equal(t, http.MethodPut, r.Method)
			_, err = w.Write([]byte(`{"current_table_count": 0}`))
			assert.NoError(t, err)
		}
	}))
	defer server2.Close()

	client2 := NewTiCDCClient("basic-ticdc-1.basic-ticdc-peer.default.svc:8301", withURL(server2.URL))
	tableCount, err = client2.DrainCapture(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, tableCount)

	// do not support this API
	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case capturesURL:
			assert.Equal(t, http.MethodGet, r.Method)
			_, err = w.Write([]byte(capturesTwoJSONStr))
			assert.NoError(t, err)
		case "/api/v1/captures/drain":
			assert.Equal(t, http.MethodPut, r.Method)
			w.WriteHeader(http.StatusNotImplemented)
		}
	}))
	defer server3.Close()

	client3 := NewTiCDCClient("basic-ticdc-0.basic-ticdc-peer.default.svc:8301", withURL(server3.URL))
	tableCount, err = client3.DrainCapture(context.Background())
	require.Error(t, err)
	assert.Equal(t, 0, tableCount)
}

func TestTiCDCClient_ResignOwner(t *testing.T) {
	// only one capture
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, capturesURL, r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		_, err := w.Write([]byte(capturesOneJSONStr))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewTiCDCClient("basic-ticdc-0.basic-ticdc-peer.default.svc:8301", withURL(server.URL))
	ok, err := client.ResignOwner(context.Background())
	require.NoError(t, err)
	assert.True(t, ok)

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case capturesURL:
			assert.Equal(t, http.MethodGet, r.Method)
			_, err = w.Write([]byte(capturesTwoJSONStr))
			assert.NoError(t, err)
		case "/api/v1/owner/resign":
			assert.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server2.Close()

	// two captures, current capture is owner
	client2 := NewTiCDCClient("basic-ticdc-0.basic-ticdc-peer.default.svc:8301", withURL(server2.URL))
	ok, err = client2.ResignOwner(context.Background())
	require.NoError(t, err)
	assert.False(t, ok)

	// two captures, current capture is not owner
	client3 := NewTiCDCClient("basic-ticdc-1.basic-ticdc-peer.default.svc:8301", withURL(server2.URL))
	ok, err = client3.ResignOwner(context.Background())
	require.NoError(t, err)
	assert.True(t, ok)

	// two captures, but this instance is not found
	client4 := NewTiCDCClient("basic-ticdc-2.basic-ticdc-peer.default.svc:8301", withURL(server2.URL))
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

	client := NewTiCDCClient("basic-ticdc-0.basic-ticdc-peer.default.svc:8301", withURL(server.URL))
	ok, err := client.IsHealthy(context.Background())
	require.NoError(t, err)
	assert.True(t, ok)

	capturesJSONStr := `
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
		_, err = w.Write([]byte(capturesJSONStr))
		assert.NoError(t, err)
	}))
	defer server2.Close()

	client2 := NewTiCDCClient("basic-ticdc-0.basic-ticdc-peer.default.svc:8301", withURL(server2.URL))
	ok, err = client2.IsHealthy(context.Background())
	require.NoError(t, err)
	assert.False(t, ok)

	// owner exists and healthy
	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/captures":
			assert.Equal(t, http.MethodGet, r.Method)
			_, err = w.Write([]byte(capturesOneJSONStr))
			assert.NoError(t, err)
		case "/api/v1/health":
			assert.Equal(t, http.MethodGet, r.Method)
			_, err = w.Write([]byte("ok"))
			assert.NoError(t, err)
		}
	}))
	defer server3.Close()

	client3 := NewTiCDCClient("basic-ticdc-0.basic-ticdc-peer.default.svc:8301", withURL(server3.URL))
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

	client := NewTiCDCClient("basic-ticdc-1.basic-ticdc-peer.default.svc:8301", withURL(server.URL))
	captures, err := client.(*ticdcClient).getCaptures(context.Background())
	require.NoError(t, err)
	assert.Len(t, captures, 2)
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
			instance: "basic-ticdc-0.basic-ticdc-peer.default.svc:8301",
			this:     captures[0],
			owner:    captures[0],
		},
		{
			name:     "test non-owner capture",
			instance: "basic-ticdc-1.basic-ticdc-peer.default.svc:8301",
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
