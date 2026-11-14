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

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/proto"
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

func TestTiKVClient_GetLeaderCountPartialResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.google.protobuf; proto=io.prometheus.client.MetricFamily; encoding=delimited")
		_, err := protodelim.MarshalTo(w, &dto.MetricFamily{
			Name: proto.String("tikv_raftstore_region_count"),
			Type: dto.MetricType_GAUGE.Enum(),
			Metric: []*dto.Metric{{
				Label: []*dto.LabelPair{{Name: proto.String("type"), Value: proto.String("leader")}},
				Gauge: &dto.Gauge{Value: proto.Float64(0)},
			}},
		})
		assert.NoError(t, err)
		// Start another metric family but truncate its payload. The parsed zero
		// above must not make the caller consider leader eviction complete.
		_, err = w.Write([]byte{10, 1})
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewTiKVClient(server.URL, 5*time.Second, nil, false)
	_, err := client.GetLeaderCount()
	require.ErrorContains(t, err, "reading metric family protocol buffer failed")
}

func TestTiKVClient_GetLeaderCountFetchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte("boom"))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewTiKVClient(server.URL, 5*time.Second, nil, false)
	count, err := client.GetLeaderCount()
	require.Error(t, err)
	assert.Equal(t, 0, count)
	assert.Contains(t, err.Error(), "fail to fetch metric families")
}
