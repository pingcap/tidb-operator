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

func TestTiDDBControl(t *testing.T) {
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

	// only test non-tls
	client, err := NewDefaultTiCDCControl().GetTiCDCPodClient(context.Background(), nil, "", "", "", "", false)
	require.NoError(t, err)
	client.(*ticdcClient).url = server.URL // replace the url with the test server
	status, err := client.GetStatus(context.Background())
	require.NoError(t, err)
	assert.True(t, status.IsOwner)
}
