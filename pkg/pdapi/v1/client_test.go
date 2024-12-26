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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestPDClient_GetHealth(t *testing.T) {
	jsonStr := `
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/pd/api/v1/health", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(jsonStr))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewPDClient(server.URL, time.Second, nil)
	healthInfo, err := client.GetHealth(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, healthInfo)
	assert.Len(t, healthInfo.Healths, 3)
	assert.Equal(t, "basic-7axwci", healthInfo.Healths[0].Name)
	assert.Equal(t, uint64(1428427862495950874), healthInfo.Healths[0].MemberID)
	assert.Len(t, healthInfo.Healths[0].ClientUrls, 1)
	assert.Equal(t, "http://basic-pd-7axwci.basic-pd-peer.default:2379", healthInfo.Healths[0].ClientUrls[0])
	assert.True(t, healthInfo.Healths[0].Health)
	assert.Equal(t, "basic-53pe89", healthInfo.Healths[1].Name)
	assert.Equal(t, uint64(2548954049902922308), healthInfo.Healths[1].MemberID)
	assert.Len(t, healthInfo.Healths[1].ClientUrls, 1)
	assert.Equal(t, "http://basic-pd-53pe89.basic-pd-peer.default:2379", healthInfo.Healths[1].ClientUrls[0])
	assert.True(t, healthInfo.Healths[1].Health)
	assert.Equal(t, "basic-0qt9e9", healthInfo.Healths[2].Name)
	assert.Equal(t, uint64(12818309996638325969), healthInfo.Healths[2].MemberID)
	assert.Len(t, healthInfo.Healths[2].ClientUrls, 1)
	assert.Equal(t, "http://basic-pd-0qt9e9.basic-pd-peer.default:2379", healthInfo.Healths[2].ClientUrls[0])
	assert.True(t, healthInfo.Healths[2].Health)
}

func TestPDClient_GetConfig(t *testing.T) {
	// simplified config
	jsonStr := `
{
  "log": {
    "level": "debug"
  },
  "schedule": {
    "leader-schedule-limit": 4,
    "low-space-ratio": 0.8,
    "schedulers-v2": [
      {
        "type": "evict-slow-store",
        "args": null,
        "disable": false,
        "args-payload": ""
      }
    ]
  },
  "replication": {
    "max-replicas": 3,
    "location-labels": "zone,rack"
  }
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/pd/api/v1/config", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(jsonStr))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewPDClient(server.URL, time.Second, nil)
	config, err := client.GetConfig(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.NotNil(t, config.Log)
	assert.Equal(t, "debug", config.Log.Level)
	assert.NotNil(t, config.Schedule)
	assert.Equal(t, uint64(4), *config.Schedule.LeaderScheduleLimit)
	assert.InEpsilon(t, 0.8, *config.Schedule.LowSpaceRatio, 0.0001)
	assert.Len(t, *config.Schedule.Schedulers, 1)
	assert.Equal(t, "evict-slow-store", []PDSchedulerConfig(*config.Schedule.Schedulers)[0].Type)
	assert.NotNil(t, config.Replication)
	assert.Equal(t, uint64(3), *config.Replication.MaxReplicas)
	assert.Len(t, config.Replication.LocationLabels, 2)
	assert.Equal(t, "zone", config.Replication.LocationLabels[0])
	assert.Equal(t, "rack", config.Replication.LocationLabels[1])
}

func TestPDClient_GetCluster(t *testing.T) {
	jsonStr := `
{
  "id": 7452180154224557728,
  "max_peer_count": 3
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/pd/api/v1/cluster", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(jsonStr))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewPDClient(server.URL, time.Second, nil)
	cluster, err := client.GetCluster(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, cluster)
	assert.Equal(t, uint64(7452180154224557728), cluster.Id)
	assert.Equal(t, uint32(3), cluster.MaxPeerCount)
}

func TestPDClient_GetMembers(t *testing.T) {
	jsonStr := `
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/pd/api/v1/members", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(jsonStr))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewPDClient(server.URL, time.Second, nil)
	members, err := client.GetMembers(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(7452180154224557728), members.Header.ClusterId)
	assert.NotNil(t, members)
	assert.Len(t, members.Members, 3)
	assert.Equal(t, "basic-7axwci", members.Members[0].Name)
	assert.Equal(t, uint64(1428427862495950874), members.Members[0].MemberId)
	assert.Len(t, members.Members[0].PeerUrls, 1)
	assert.Len(t, members.Members[0].ClientUrls, 1)
	assert.Equal(t, "http://basic-pd-7axwci.basic-pd-peer.default:2380", members.Members[0].PeerUrls[0])
	assert.Equal(t, "http://basic-pd-7axwci.basic-pd-peer.default:2379", members.Members[0].ClientUrls[0])
	assert.Equal(t, "basic-53pe89", members.Members[1].Name)
	assert.Equal(t, uint64(2548954049902922308), members.Members[1].MemberId)
	assert.Len(t, members.Members[1].PeerUrls, 1)
	assert.Len(t, members.Members[1].ClientUrls, 1)
	assert.Equal(t, "http://basic-pd-53pe89.basic-pd-peer.default:2380", members.Members[1].PeerUrls[0])
	assert.Equal(t, "http://basic-pd-53pe89.basic-pd-peer.default:2379", members.Members[1].ClientUrls[0])
	assert.Equal(t, "basic-0qt9e9", members.Members[2].Name)
	assert.Equal(t, uint64(12818309996638325969), members.Members[2].MemberId)
	assert.Len(t, members.Members[2].PeerUrls, 1)
	assert.Len(t, members.Members[2].ClientUrls, 1)
	assert.Equal(t, "http://basic-pd-0qt9e9.basic-pd-peer.default:2380", members.Members[2].PeerUrls[0])
	assert.Equal(t, "http://basic-pd-0qt9e9.basic-pd-peer.default:2379", members.Members[2].ClientUrls[0])
	assert.Len(t, members.Leader.PeerUrls, 1)
	assert.Len(t, members.Leader.ClientUrls, 1)
	assert.Len(t, members.EtcdLeader.PeerUrls, 1)
	assert.Len(t, members.EtcdLeader.ClientUrls, 1)
	assert.Equal(t, members.Leader.Name, members.EtcdLeader.Name)
	assert.Equal(t, members.Leader.MemberId, members.EtcdLeader.MemberId)
	assert.Equal(t, members.Leader.PeerUrls[0], members.EtcdLeader.PeerUrls[0])
	assert.Equal(t, members.Leader.ClientUrls[0], members.EtcdLeader.ClientUrls[0])
	assert.Equal(t, "http://basic-pd-53pe89.basic-pd-peer.default:2380", members.Leader.PeerUrls[0])
	assert.Equal(t, "http://basic-pd-53pe89.basic-pd-peer.default:2379", members.Leader.ClientUrls[0])
}

func TestPDClient_GetStores(t *testing.T) {
	jsonStr := `
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
		assert.Equal(t, "/pd/api/v1/stores", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(jsonStr))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewPDClient(server.URL, time.Second, nil)
	stores, err := client.GetStores(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, stores)
	assert.Equal(t, 4, stores.Count)
	assert.Len(t, stores.Stores, 4)
	assert.Equal(t, uint64(4), stores.Stores[0].Store.Id)
	assert.Equal(t, "basic-tikv-5gzwgj.basic-tikv-peer.default:20160", stores.Stores[0].Store.Address)
	assert.Equal(t, "8.1.0", stores.Stores[0].Store.Version)
	assert.Equal(t, "basic-tikv-5gzwgj.basic-tikv-peer.default:20160", stores.Stores[0].Store.PeerAddress)
	assert.Equal(t, "basic-tikv-5gzwgj.basic-tikv-peer.default:20180", stores.Stores[0].Store.StatusAddress)
	assert.Equal(t, "Up", stores.Stores[0].Store.StateName)
	assert.Equal(t, 17, stores.Stores[0].Status.LeaderCount)
	assert.Equal(t, 60, stores.Stores[0].Status.RegionCount)
	assert.Equal(t, uint64(20), stores.Stores[1].Store.Id)
	assert.Equal(t, uint64(1), stores.Stores[2].Store.Id)
}

func TestPDClient_GetStore(t *testing.T) {
	jsonStr := `
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
    "last_heartbeat": 1735114832382630989,
    "node_state": 1,
    "state_name": "Up"
  },
  "status": {
    "capacity": "309.4GiB",
    "available": "218.8GiB",
    "used_size": "296.7MiB",
    "leader_count": 0,
    "leader_weight": 1,
    "leader_score": 0,
    "leader_size": 0,
    "region_count": 5,
    "region_weight": 1,
    "region_score": 10.683555392119445,
    "region_size": 5,
    "slow_score": 1,
    "slow_trend": {
      "cause_value": 250064.3282828283,
      "cause_rate": 0,
      "result_value": 0,
      "result_rate": 0
    },
    "start_ts": "2024-12-25T03:05:03Z",
    "last_heartbeat_ts": "2024-12-25T08:20:32.382630989Z",
    "uptime": "5h15m29.382630989s"
  }
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/pd/api/v1/store/4", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(jsonStr))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewPDClient(server.URL, time.Second, nil)
	store, err := client.GetStore(context.Background(), "4")
	require.NoError(t, err)
	assert.NotNil(t, store)
	assert.Equal(t, uint64(4), store.Store.Id)
	assert.Equal(t, "basic-tikv-5gzwgj.basic-tikv-peer.default:20160", store.Store.Address)
	assert.Equal(t, "8.1.0", store.Store.Version)
	assert.Equal(t, "basic-tikv-5gzwgj.basic-tikv-peer.default:20160", store.Store.PeerAddress)
	assert.Equal(t, "basic-tikv-5gzwgj.basic-tikv-peer.default:20180", store.Store.StatusAddress)
	assert.Equal(t, "Up", store.Store.StateName)
}

func TestPDClinet_SetStoreLabels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/pd/api/v1/store/1/label", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{}`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewPDClient(server.URL, time.Second, nil)
	ok, err := client.SetStoreLabels(context.Background(), 1, map[string]string{"zone": "cn", "rack": "1"})
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestPDClient_DeleteStore(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/pd/api/v1/store/1", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{}`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewPDClient(server.URL, time.Second, nil)
	err := client.DeleteStore(context.Background(), "1")
	require.NoError(t, err)
}

func TestPDClient_DeleteMember(t *testing.T) {
	// in DeleteMember, we call GetMembers first to get the member id
	// so we need to mock GetMembers response
	// one server for two requests
	getMembersJSONStr := `
{
  "header": {
    "cluster_id": 7452180154224557728
  },
  "members": [
    {
      "name": "basic-7axwci",
      "member_id": 1428427862495950874
    }
  ]
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pd/api/v1/members" {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(getMembersJSONStr))
			assert.NoError(t, err)
		} else if r.URL.Path == "/pd/api/v1/member/basic-7axwci" {
			assert.Equal(t, http.MethodDelete, r.Method)
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{}`))
			assert.NoError(t, err)
		}
	}))
	defer server.Close()

	client := NewPDClient(server.URL, time.Second, nil)
	err := client.DeleteMember(context.Background(), "basic-7axwci")
	require.NoError(t, err)
}

func TestPDClient_DeleteMemberByID(t *testing.T) {
	// in DeleteMember, we call GetMembers first to get the member id
	// so we need to mock GetMembers response
	// one server for two requests
	getMembersJSONStr := `
{
  "header": {
    "cluster_id": 7452180154224557728
  },
  "members": [
    {
      "name": "basic-7axwci",
      "member_id": 1428427862495950874
    }
  ]
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pd/api/v1/members" {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(getMembersJSONStr))
			assert.NoError(t, err)
		} else if r.URL.Path == "/pd/api/v1/member/1428427862495950874" {
			assert.Equal(t, http.MethodDelete, r.Method)
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{}`))
			assert.NoError(t, err)
		}
	}))
	defer server.Close()

	client := NewPDClient(server.URL, time.Second, nil)
	err := client.DeleteMemberByID(context.Background(), 1428427862495950874)
	require.NoError(t, err)
}

func TestPDConfig_UpdateReplicationConfig(t *testing.T) {
	jsonStr := `
{
  "max-replicas": 3,
  "location-labels": "zone,rack"
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/pd/api/v1/config/replicate", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(jsonStr))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewPDClient(server.URL, time.Second, nil)
	replicationConfig := PDReplicationConfig{
		MaxReplicas:    ptr.To[uint64](3),
		LocationLabels: []string{"zone", "rack"},
	}
	err := client.UpdateReplicationConfig(context.Background(), replicationConfig)
	require.NoError(t, err)
}

func TestPDClient_BeginEvictLeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/pd/api/v1/schedulers", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{}`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewPDClient(server.URL, time.Second, nil)
	err := client.BeginEvictLeader(context.Background(), "4")
	require.NoError(t, err)

	// try to add the evict-leader scheduler, but it already exists
	// it will call the get scheduler API to check if the scheduler exists
	// and then it will can get scheduler config to check if the store id is correct
	schedulersStr := `
[
  "balance-leader-scheduler",
  "balance-hot-region-scheduler",
  "evict-leader-scheduler",
  "evict-slow-store-scheduler",
  "balance-region-scheduler"
]`
	evictLeaderSchedulerListStr := `
{
  "store-id-ranges": {
    "4": [
      {
        "start-key": "",
        "end-key": ""
      }
    ]
  }
}`
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//nolint:goconst // it's ok
		if r.URL.Path == "/pd/api/v1/schedulers" && r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, err = w.Write([]byte(`{"error":"scheduler already exists"}`))
			assert.NoError(t, err)
		} else if r.URL.Path == "/pd/api/v1/schedulers" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write([]byte(schedulersStr))
			assert.NoError(t, err)
		} else if r.URL.Path == "/pd/api/v1/scheduler-config/evict-leader-scheduler/list" {
			assert.Equal(t, http.MethodGet, r.Method)
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write([]byte(evictLeaderSchedulerListStr))
			assert.NoError(t, err)
		}
	}))
	defer server2.Close()

	client2 := NewPDClient(server2.URL, time.Second, nil)
	err = client2.BeginEvictLeader(context.Background(), "4")
	require.NoError(t, err)
}

func TestPDClient_EndEvictLeader(t *testing.T) {
	// try to remove the evict-leader scheduler, and then check if the scheduler is removed
	schedulersStr := `
  [
    "balance-leader-scheduler",
    "balance-hot-region-scheduler",
    "evict-slow-store-scheduler",
    "balance-region-scheduler"
  ]`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//nolint:goconst // it's ok
		if r.URL.Path == "/pd/api/v1/schedulers/evict-leader-scheduler-4" && r.Method == http.MethodDelete {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{}`))
			assert.NoError(t, err)
		} else if r.URL.Path == "/pd/api/v1/schedulers" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(schedulersStr))
			assert.NoError(t, err)
		}
	}))
	defer server.Close()

	client := NewPDClient(server.URL, time.Second, nil)
	err := client.EndEvictLeader(context.Background(), "4")
	require.NoError(t, err)

	// exist a evict-leader scheduler for another store
	schedulersStr = `
[
  "balance-leader-scheduler",
  "balance-hot-region-scheduler",
  "evict-leader-scheduler",
  "evict-slow-store-scheduler",
  "balance-region-scheduler"
]`
	evictLeaderSchedulerListStr := `
{
  "store-id-ranges": {
    "10": [
      {
        "start-key": "",
        "end-key": ""
      }
    ]
  }
}`
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pd/api/v1/schedulers/evict-leader-scheduler-4" && r.Method == http.MethodDelete {
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write([]byte(`{}`))
			assert.NoError(t, err)
		} else if r.URL.Path == "/pd/api/v1/schedulers" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write([]byte(schedulersStr))
			assert.NoError(t, err)
		} else if r.URL.Path == "/pd/api/v1/scheduler-config/evict-leader-scheduler/list" {
			assert.Equal(t, http.MethodGet, r.Method)
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write([]byte(evictLeaderSchedulerListStr))
			assert.NoError(t, err)
		}
	}))
	defer server2.Close()
	client2 := NewPDClient(server2.URL, time.Second, nil)
	err = client2.EndEvictLeader(context.Background(), "4")
	require.NoError(t, err)

	// remove the evict-leader scheduler, but it still exists
	schedulersStr = `
  [
    "balance-leader-scheduler",
    "balance-hot-region-scheduler",
    "evict-leader-scheduler",
    "evict-slow-store-scheduler",
    "balance-region-scheduler"
  ]`
	evictLeaderSchedulerListStr = `
  {
    "store-id-ranges": {
      "10": [
        {
          "start-key": "",
          "end-key": ""
        }
      ]
    }
  }`
	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pd/api/v1/schedulers/evict-leader-scheduler-4" && r.Method == http.MethodDelete {
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write([]byte(`{}`))
			assert.NoError(t, err)
		} else if r.URL.Path == "/pd/api/v1/schedulers" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write([]byte(schedulersStr))
			assert.NoError(t, err)
		} else if r.URL.Path == "/pd/api/v1/scheduler-config/evict-leader-scheduler/list" {
			assert.Equal(t, http.MethodGet, r.Method)
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write([]byte(evictLeaderSchedulerListStr))
			assert.NoError(t, err)
		}
	}))
	defer server3.Close()
	client3 := NewPDClient(server3.URL, time.Second, nil)
	err = client3.EndEvictLeader(context.Background(), "10")
	require.Error(t, err)
}

func TestPDClient_GetEvictLeaderScheduler(t *testing.T) {
	schedulersStr := `
  [
    "balance-leader-scheduler",
    "balance-hot-region-scheduler",
    "evict-leader-scheduler",
    "evict-slow-store-scheduler",
    "balance-region-scheduler"
  ]`
	evictLeaderSchedulerListStr := `
  {
    "store-id-ranges": {
      "10": [
        {
          "start-key": "",
          "end-key": ""
        }
      ]
    }
  }`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pd/api/v1/schedulers/evict-leader-scheduler-4" && r.Method == http.MethodDelete {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{}`))
			assert.NoError(t, err)
		} else if r.URL.Path == "/pd/api/v1/schedulers" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(schedulersStr))
			assert.NoError(t, err)
		} else if r.URL.Path == "/pd/api/v1/scheduler-config/evict-leader-scheduler/list" {
			assert.Equal(t, http.MethodGet, r.Method)
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(evictLeaderSchedulerListStr))
			assert.NoError(t, err)
		}
	}))
	defer server.Close()

	client := NewPDClient(server.URL, time.Second, nil)
	scheduler, err := client.GetEvictLeaderScheduler(context.Background(), "10")
	require.NoError(t, err)
	assert.Equal(t, "evict-leader-scheduler-10", scheduler)
}

func TestPDClient_GetPDLeader(t *testing.T) {
	jsonStr := `
{
  "name": "basic-53pe89",
  "member_id": 2548954049902922308,
  "peer_urls": [
    "http://basic-pd-53pe89.basic-pd-peer.default:2380"
  ],
  "client_urls": [
    "http://basic-pd-53pe89.basic-pd-peer.default:2379"
  ]
}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/pd/api/v1/leader", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(jsonStr))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewPDClient(server.URL, time.Second, nil)
	leader, err := client.GetPDLeader(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, leader)
	assert.Equal(t, "basic-53pe89", leader.Name)
	assert.Equal(t, uint64(2548954049902922308), leader.MemberId)
	assert.Len(t, leader.PeerUrls, 1)
	assert.Len(t, leader.ClientUrls, 1)
	assert.Equal(t, "http://basic-pd-53pe89.basic-pd-peer.default:2380", leader.PeerUrls[0])
	assert.Equal(t, "http://basic-pd-53pe89.basic-pd-peer.default:2379", leader.ClientUrls[0])
}

func TestPDClient_TransferPDLeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/pd/api/v1/leader/transfer/basic-7axwci", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{}`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	client := NewPDClient(server.URL, time.Second, nil)
	err := client.TransferPDLeader(context.Background(), "basic-7axwci")
	require.NoError(t, err)
}
