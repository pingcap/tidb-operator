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

package pdapi

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncomingStoreLimitsAPI(t *testing.T) {
	var writes []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			assert.Equal(t, "/pd/api/v1/stores/limit", r.URL.Path)
			_, err := w.Write([]byte(`{"1":{"add-peer":15,"remove-peer":99,"transfer-leader-in":100000000},"2":{"add-peer":30}}`))
			assert.NoError(t, err)
		case http.MethodPost:
			assert.Equal(t, "/pd/api/v1/store/1/limit", r.URL.Path)
			var body map[string]any
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			writes = append(writes, body)
		default:
			t.Errorf("unexpected request %s", r.Method)
		}
	}))
	defer server.Close()
	pd := NewPDClient(server.URL, time.Second, nil)
	limits, err := pd.GetStoreLimits(context.Background())
	require.NoError(t, err)
	require.InDelta(t, 15.0, *limits["1"].AddPeer, 0)
	require.InDelta(t, 100000000.0, *limits["1"].TransferLeaderIn, 0)
	require.Nil(t, limits["2"].TransferLeaderIn, "missing capability must not become a zero rate")
	for _, typ := range []StoreLimitType{StoreLimitAddPeer, StoreLimitTransferLeaderIn} {
		require.NoError(t, pd.SetStoreLimit(context.Background(), "1", typ, 0.000001))
	}
	require.Equal(t, []map[string]any{{"type": "add-peer", "rate": 0.000001}, {"type": "transfer-leader-in", "rate": 0.000001}}, writes)
	for _, rate := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		require.Error(t, pd.SetStoreLimit(context.Background(), "1", StoreLimitAddPeer, rate))
	}
	require.Error(t, pd.SetStoreLimit(context.Background(), "1", "", 1))
	require.Error(t, pd.SetStoreLimit(context.Background(), "1", "remove-peer", 1))
	require.Len(t, writes, 2)
}

func TestIncomingStoreLimitsErrors(t *testing.T) {
	for _, body := range []string{"unsupported store limit", "not-json"} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if body == "unsupported store limit" || r.Method == http.MethodPost {
					w.WriteHeader(http.StatusBadRequest)
				}
				_, err := w.Write([]byte(body))
				assert.NoError(t, err)
			}))
			defer server.Close()
			pd := NewPDClient(server.URL, time.Second, nil)
			_, err := pd.GetStoreLimits(context.Background())
			require.Error(t, err)
			require.Error(t, pd.SetStoreLimit(context.Background(), "1", StoreLimitTransferLeaderIn, 0.000001))
		})
	}
}
