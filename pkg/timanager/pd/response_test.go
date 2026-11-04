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
	"testing"

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
	c.health = &pdapi.HealthInfo{}
	list, err := NewMemberLister("cluster", c).List(context.Background())
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	require.False(t, list.Items[0].IsLeader)
	require.False(t, list.Items[0].IsEtcdLeader)
}
