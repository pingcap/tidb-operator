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

package pd

import (
	"context"
	"testing"
	"time"

	"net/http"
	"net/http/httptest"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/apis/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/pdapi/v1"
	"github.com/pingcap/tidb-operator/pkg/timanager"
)

func TestPDClientManager(t *testing.T) {
	healthStr := `
[
  {
    "name": "basic-7axwci",
    "member_id": 1428427862495950874,
    "client_urls": [
      "http://basic-pd-7axwci.basic-pd-peer.default:2379"
    ],
    "health": true
  },
  {
    "name": "basic-53pe89",
    "member_id": 2548954049902922308,
    "client_urls": [
      "http://basic-pd-53pe89.basic-pd-peer.default:2379"
    ],
    "health": true
  },
  {
    "name": "basic-0qt9e9",
    "member_id": 12818309996638325969,
    "client_urls": [
      "http://basic-pd-0qt9e9.basic-pd-peer.default:2379"
    ],
    "health": true
  }
]`
	membersStr := `
{
  "header": {
    "cluster_id": 7452180154224557728
  },
  "members": [
    {
      "name": "basic-7axwci",
      "member_id": 1428427862495950874,
      "peer_urls": [
        "http://basic-pd-7axwci.basic-pd-peer.default:2380"
      ],
      "client_urls": [
        "http://basic-pd-7axwci.basic-pd-peer.default:2379"
      ],
      "deploy_path": "/",
      "binary_version": "v8.1.0",
      "git_hash": "fca469ca33eb5d8b5e0891b507c87709a00b0e81"
    },
    {
      "name": "basic-53pe89",
      "member_id": 2548954049902922308,
      "peer_urls": [
        "http://basic-pd-53pe89.basic-pd-peer.default:2380"
      ],
      "client_urls": [
        "http://basic-pd-53pe89.basic-pd-peer.default:2379"
      ],
      "deploy_path": "/",
      "binary_version": "v8.1.0",
      "git_hash": "fca469ca33eb5d8b5e0891b507c87709a00b0e81"
    },
    {
      "name": "basic-0qt9e9",
      "member_id": 12818309996638325969,
      "peer_urls": [
        "http://basic-pd-0qt9e9.basic-pd-peer.default:2380"
      ],
      "client_urls": [
        "http://basic-pd-0qt9e9.basic-pd-peer.default:2379"
      ],
      "deploy_path": "/",
      "binary_version": "v8.1.0",
      "git_hash": "fca469ca33eb5d8b5e0891b507c87709a00b0e81"
    }
  ],
  "leader": {
    "name": "basic-53pe89",
    "member_id": 2548954049902922308,
    "peer_urls": [
      "http://basic-pd-53pe89.basic-pd-peer.default:2380"
    ],
    "client_urls": [
      "http://basic-pd-53pe89.basic-pd-peer.default:2379"
    ],
    "deploy_path": "/",
    "binary_version": "v8.1.0",
    "git_hash": "fca469ca33eb5d8b5e0891b507c87709a00b0e81"
  },
  "etcd_leader": {
    "name": "basic-53pe89",
    "member_id": 2548954049902922308,
    "peer_urls": [
      "http://basic-pd-53pe89.basic-pd-peer.default:2380"
    ],
    "client_urls": [
      "http://basic-pd-53pe89.basic-pd-peer.default:2379"
    ],
    "deploy_path": "/",
    "binary_version": "v8.1.0",
    "git_hash": "fca469ca33eb5d8b5e0891b507c87709a00b0e81"
  }
}`
	storesStr := `
{
	"count": 4,
	"stores": [
	{
		"store": {
		"id": 4,
		"address": "basic-tikv-5gzwgj.basic-tikv-peer.default:20160",
		"version": "8.1.0",
		"peer_address": "basic-tikv-5gzwgj.basic-tikv-peer.default:20160",
		"status_address": "basic-tikv-5gzwgj.basic-tikv-peer.default:20180",
		"git_hash": "ba73b0d92d94463d74543550d0efe61fa6a6f416",
		"start_timestamp": 1735095903,
		"deploy_path": "/",
		"last_heartbeat": 1735098083480549971,
		"node_state": 1,
		"state_name": "Up"
		},
		"status": {
		"capacity": "311.7GiB",
		"available": "222.1GiB",
		"used_size": "290.7MiB",
		"leader_count": 17,
		"leader_weight": 1,
		"leader_score": 17,
		"leader_size": 17,
		"region_count": 60,
		"region_weight": 1,
		"region_score": 127.38727094442186,
		"region_size": 60,
		"slow_score": 1,
		"slow_trend": {
			"cause_value": 250054.47306397307,
			"cause_rate": 0,
			"result_value": 10,
			"result_rate": 0
		},
		"start_ts": "2024-12-25T03:05:03Z",
		"last_heartbeat_ts": "2024-12-25T03:41:23.480549971Z",
		"uptime": "36m20.480549971s"
		}
	},
	{
		"store": {
		"id": 20,
		"address": "basic-tiflash-tphn85.basic-tiflash-peer.default:3930",
		"labels": [
			{
			"key": "engine",
			"value": "tiflash"
			}
		],
		"version": "v8.1.0",
		"peer_address": "basic-tiflash-tphn85.basic-tiflash-peer.default:20170",
		"status_address": "basic-tiflash-tphn85.basic-tiflash-peer.default:20292",
		"git_hash": "c1838001167c8ba83af759085a71ad61e6c2a5af",
		"start_timestamp": 1735095904,
		"deploy_path": "/tiflash",
		"last_heartbeat": 1735098074657264947,
		"node_state": 1,
		"state_name": "Up"
		},
		"status": {
		"capacity": "311.7GiB",
		"available": "222.1GiB",
		"used_size": "1B",
		"leader_count": 0,
		"leader_weight": 1,
		"leader_score": 0,
		"leader_size": 0,
		"region_count": 0,
		"region_weight": 1,
		"region_score": 0,
		"region_size": 0,
		"slow_score": 1,
		"slow_trend": {
			"cause_value": 250039.13468013468,
			"cause_rate": 0,
			"result_value": 0,
			"result_rate": 0
		},
		"start_ts": "2024-12-25T03:05:04Z",
		"last_heartbeat_ts": "2024-12-25T03:41:14.657264947Z",
		"uptime": "36m10.657264947s"
		}
	},
	{
		"store": {
		"id": 1,
		"address": "basic-tikv-06vg39.basic-tikv-peer.default:20160",
		"version": "8.1.0",
		"peer_address": "basic-tikv-06vg39.basic-tikv-peer.default:20160",
		"status_address": "basic-tikv-06vg39.basic-tikv-peer.default:20180",
		"git_hash": "ba73b0d92d94463d74543550d0efe61fa6a6f416",
		"start_timestamp": 1735095903,
		"deploy_path": "/",
		"last_heartbeat": 1735098083478896272,
		"node_state": 1,
		"state_name": "Up"
		},
		"status": {
		"capacity": "311.7GiB",
		"available": "222.1GiB",
		"used_size": "293.5MiB",
		"leader_count": 25,
		"leader_weight": 1,
		"leader_score": 25,
		"leader_size": 25,
		"region_count": 60,
		"region_weight": 1,
		"region_score": 127.38727094442186,
		"region_size": 60,
		"slow_score": 1,
		"slow_trend": {
			"cause_value": 250056.73063973064,
			"cause_rate": 0,
			"result_value": 13,
			"result_rate": 0
		},
		"start_ts": "2024-12-25T03:05:03Z",
		"last_heartbeat_ts": "2024-12-25T03:41:23.478896272Z",
		"uptime": "36m20.478896272s"
		}
	},
	{
		"store": {
		"id": 5,
		"address": "basic-tikv-4q2bgw.basic-tikv-peer.default:20160",
		"version": "8.1.0",
		"peer_address": "basic-tikv-4q2bgw.basic-tikv-peer.default:20160",
		"status_address": "basic-tikv-4q2bgw.basic-tikv-peer.default:20180",
		"git_hash": "ba73b0d92d94463d74543550d0efe61fa6a6f416",
		"start_timestamp": 1735095903,
		"deploy_path": "/",
		"last_heartbeat": 1735098083481308217,
		"node_state": 1,
		"state_name": "Up"
		},
		"status": {
		"capacity": "311.7GiB",
		"available": "222.1GiB",
		"used_size": "290.7MiB",
		"leader_count": 18,
		"leader_weight": 1,
		"leader_score": 18,
		"leader_size": 18,
		"region_count": 60,
		"region_weight": 1,
		"region_score": 127.38727094442186,
		"region_size": 60,
		"slow_score": 1,
		"slow_trend": {
			"cause_value": 250057.9478114478,
			"cause_rate": 0,
			"result_value": 6,
			"result_rate": 0
		},
		"start_ts": "2024-12-25T03:05:03Z",
		"last_heartbeat_ts": "2024-12-25T03:41:23.481308217Z",
		"uptime": "36m20.481308217s"
		}
	}
	]
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/pd/api/v1/health":
			_, err := w.Write([]byte(healthStr))
			assert.NoError(t, err)
		case "/pd/api/v1/members":
			_, err := w.Write([]byte(membersStr))
			assert.NoError(t, err)
		case "/pd/api/v1/stores":
			_, err := w.Write([]byte(storesStr))
			assert.NoError(t, err)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// hack NewUnderlayClientFunc to replace the URL to a test server
	originalNewUnderlayClientFunc := NewUnderlayClientFunc
	defer func() {
		NewUnderlayClientFunc = originalNewUnderlayClientFunc
	}()
	NewUnderlayClientFunc = func(c client.Client) timanager.NewUnderlayClientFunc[*v1alpha1.PDGroup, pdapi.PDClient] {
		return func(pdg *v1alpha1.PDGroup) (pdapi.PDClient, error) {
			_, err := originalNewUnderlayClientFunc(c)(pdg)
			require.NoError(t, err)
			// create a new PDClient with the test server URL
			return pdapi.NewPDClient(server.URL, pdRequestTimeout, nil), nil
		}
	}

	cluster := &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "basic",
			Namespace: "ns1",
		},
	}
	pdg := &v1alpha1.PDGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "basic",
			Namespace: "ns1",
		},
		Spec: v1alpha1.PDGroupSpec{
			Cluster: v1alpha1.ClusterReference{
				Name: cluster.GetName(),
			},
		},
	}
	cli := client.NewFakeClient(cluster, pdg)
	pcm := NewPDClientManager(logr.Discard(), cli)
	ctx, cancel := context.WithCancel(context.Background())
	pcm.Start(ctx)
	defer cancel()

	// not registered yet
	_, ok := pcm.Get(PrimaryKey(pdg.GetNamespace(), pdg.GetName()))
	assert.False(t, ok)

	// register
	require.NoError(t, pcm.Register(pdg))
	pdc, ok := pcm.Get(PrimaryKey(pdg.GetNamespace(), pdg.GetName()))
	assert.True(t, ok)
	// wait for the cache to be synced
	for i := 0; i < 10; i++ {
		if pdc.HasSynced() {
			break
		}
		time.Sleep(time.Second)
	}
	assert.True(t, pdc.HasSynced())

	// check members
	m1, err := pdc.Members().Get("basic-7axwci")
	require.NoError(t, err)
	assert.NotNil(t, m1)
	assert.Equal(t, "1428427862495950874", m1.ID)
	assert.False(t, m1.IsLeader)
	ns, clusterName := SplitPrimaryKey(m1.Namespace) // also test SplitPrimaryKey
	assert.Equal(t, "ns1", ns)
	assert.Equal(t, "basic", clusterName)
	m2, err := pdc.Members().Get("basic-53pe89")
	require.NoError(t, err)
	assert.NotNil(t, m2)
	assert.Equal(t, "2548954049902922308", m2.ID)
	assert.True(t, m2.IsLeader)
	m3, err := pdc.Members().Get("basic-0qt9e9")
	require.NoError(t, err)
	assert.NotNil(t, m3)
	assert.Equal(t, "12818309996638325969", m3.ID)
	assert.False(t, m3.IsLeader)

	// check stores
	s1, err := pdc.Stores().Get("basic-tikv-5gzwgj.basic-tikv-peer.default:20160")
	require.NoError(t, err)
	assert.NotNil(t, s1)
	assert.Equal(t, "4", s1.ID)
	assert.Equal(t, "Up", string(s1.State))
	s2, err := pdc.Stores().Get("basic-tiflash-tphn85.basic-tiflash-peer.default:3930")
	require.NoError(t, err)
	assert.NotNil(t, s2)
	assert.Equal(t, "20", s2.ID)
	assert.Equal(t, "Up", string(s2.State))
	s3, err := pdc.Stores().Get("basic-tikv-06vg39.basic-tikv-peer.default:20160")
	require.NoError(t, err)
	assert.NotNil(t, s3)
	assert.Equal(t, "1", s3.ID)
	assert.Equal(t, "Up", string(s3.State))
	s4, err := pdc.Stores().Get("basic-tikv-4q2bgw.basic-tikv-peer.default:20160")
	require.NoError(t, err)
	assert.NotNil(t, s4)
	assert.Equal(t, "5", s4.ID)
	assert.Equal(t, "Up", string(s4.State))

	// check underlay client
	underlay := pdc.Underlay()
	assert.NotNil(t, underlay)
	health, err := underlay.GetHealth(ctx)
	require.NoError(t, err)
	assert.Len(t, health.Healths, 3)
}
