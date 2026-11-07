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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pingcap/tidb-operator/v2/pkg/pdapi/v1"
)

type responseClient struct {
	pdapi.PDClient
	stores  *pdapi.StoresInfo
	members *pdapi.MembersInfo
	health  *pdapi.HealthInfo
}

func (c *responseClient) GetStores(context.Context) (*pdapi.StoresInfo, error) { return c.stores, nil }
func (c *responseClient) GetMembers(context.Context) (*pdapi.MembersInfo, error) {
	return c.members, nil
}
func (c *responseClient) GetHealth(context.Context) (*pdapi.HealthInfo, error) { return c.health, nil }

func TestStoreListerIncompleteResponse(t *testing.T) {
	for _, body := range []string{
		`null`,
		`{"count":1,"stores":null}`,
		`{"count":1}`,
		`{"count":1,"stores":[]}`,
		`{"count":2,"stores":[{"store":{"id":1,"address":"store"},"status":{}}]}`,
		`{"count":1,"stores":[{"store":{"id":1,"address":"one"},"status":{}},{"store":{"id":2,"address":"two"},"status":{}}]}`,
		`{"stores":[null]}`,
		`{"stores":[{}]}`,
		`{"stores":[{"store":{},"status":{}}]}`,
		`{"stores":[{"store":{"id":1}}]}`,
	} {
		t.Run(body, func(t *testing.T) {
			c := &responseClient{}
			require.NoError(t, json.Unmarshal([]byte(body), &c.stores))
			list, err := NewStoreLister("cluster", c).List(context.Background())
			require.Error(t, err)
			require.Nil(t, list)
		})
	}
	c := &responseClient{}
	require.NoError(t, json.Unmarshal([]byte(`{"stores":[{"store":{"id":1,"address":"store"},"status":{}}]}`), &c.stores))
	list, err := NewStoreLister("cluster", c).List(context.Background())
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	require.Equal(t, "1", list.Items[0].ID)
}

func TestStoreListerEmptyResponse(t *testing.T) {
	for _, body := range []string{`{"count":0,"stores":null}`, `{"count":0,"stores":[]}`} {
		t.Run(body, func(t *testing.T) {
			c := &responseClient{}
			require.NoError(t, json.Unmarshal([]byte(body), &c.stores))
			list, err := NewStoreLister("cluster", c).List(context.Background())
			require.NoError(t, err)
			require.Empty(t, list.Items)
		})
	}
}

func TestMemberListerIncompleteResponse(t *testing.T) {
	for _, body := range []string{`null`, `{}`, `{"header":{},"members":[null]}`} {
		t.Run(body, func(t *testing.T) {
			c := &responseClient{health: &pdapi.HealthInfo{}}
			require.NoError(t, json.Unmarshal([]byte(body), &c.members))
			list, err := NewMemberLister("cluster", c).List(context.Background())
			require.Error(t, err)
			require.Nil(t, list)
		})
	}
	c := &responseClient{}
	require.NoError(t, json.Unmarshal([]byte(`{"header":{"cluster_id":1},"members":[{"member_id":2,"name":"pd"}]}`), &c.members))
	_, err := NewMemberLister("cluster", c).List(context.Background())
	require.Error(t, err)
	c.health = &pdapi.HealthInfo{Healths: []pdapi.MemberHealth{{MemberID: 2, Health: true}}}
	list, err := NewMemberLister("cluster", c).List(context.Background())
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	require.False(t, list.Items[0].IsLeader)
	require.False(t, list.Items[0].IsEtcdLeader)
}

func TestListerHTTPNullResponses(t *testing.T) {
	for _, tc := range []struct {
		name, stores, health string
		storeErr, healthErr  bool
	}{
		{"null stores", "null", "[]", true, false},
		{"null health", `{"count":0,"stores":[]}`, "null", false, true},
		{"empty stores and leaderless member", `{"count":0,"stores":[]}`, `[{"member_id":2,"name":"pd","health":true}]`, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/pd/api/v1/stores":
					_, _ = w.Write([]byte(tc.stores))
				case "/pd/api/v1/health":
					_, _ = w.Write([]byte(tc.health))
				case "/pd/api/v1/members":
					_, _ = w.Write([]byte(`{"header":{"cluster_id":1},"members":[{"member_id":2,"name":"pd"}]}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			c := pdapi.NewPDClient(server.URL, time.Second, nil)
			stores, err := NewStoreLister("cluster", c).List(context.Background())
			if tc.storeErr {
				require.Error(t, err)
				require.Nil(t, stores)
			} else {
				require.NoError(t, err)
				require.Empty(t, stores.Items)
			}
			members, err := NewMemberLister("cluster", c).List(context.Background())
			if tc.healthErr {
				require.Error(t, err)
				require.Nil(t, members)
			} else {
				require.NoError(t, err)
			}
			if !tc.healthErr && tc.name == "empty stores and leaderless member" {
				require.Len(t, members.Items, 1)
				require.True(t, members.Items[0].Health)
				require.False(t, members.Items[0].IsLeader)
				require.False(t, members.Items[0].IsEtcdLeader)
			}
		})
	}
}
