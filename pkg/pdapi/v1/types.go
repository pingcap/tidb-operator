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
	"errors"
	"time"

	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/kvproto/pkg/pdpb"

	"github.com/pingcap/tidb-operator/pkg/pdapi/pd"
)

// HealthInfo define PD's healthy info.
type HealthInfo struct {
	Healths []MemberHealth
}

// MemberHealth define a PD member's healthy info.
type MemberHealth struct {
	Name       string   `json:"name"`
	MemberID   uint64   `json:"member_id"`
	ClientUrls []string `json:"client_urls"`
	Health     bool     `json:"health"`
}

// MembersInfo is PD members info returned from PD RESTful interface.
type MembersInfo struct {
	Header     *pdpb.ResponseHeader `json:"header,omitempty"`
	Members    []*pdpb.Member       `json:"members,omitempty"`
	Leader     *pdpb.Member         `json:"leader,omitempty"`
	EtcdLeader *pdpb.Member         `json:"etcd_leader,omitempty"`
}

// ServiceRegistryEntry is the registry entry of PD Micro Service.
type ServiceRegistryEntry struct {
	Name           string `json:"name"`
	ServiceAddr    string `json:"service-addr"`
	Version        string `json:"version"`
	GitHash        string `json:"git-hash"`
	DeployPath     string `json:"deploy-path"`
	StartTimestamp int64  `json:"start-timestamp"`
}

// MetaStore is TiKV store status defined in protobuf.
type MetaStore struct {
	*metapb.Store
	StateName string `json:"state_name"`
}

// StoreStatus is TiKV store status returned from PD RESTful interface.
type StoreStatus struct {
	Capacity           pd.ByteSize `json:"capacity"`
	Available          pd.ByteSize `json:"available"`
	LeaderCount        int         `json:"leader_count"`
	RegionCount        int         `json:"region_count"`
	SendingSnapCount   uint32      `json:"sending_snap_count"`
	ReceivingSnapCount uint32      `json:"receiving_snap_count"`
	ApplyingSnapCount  uint32      `json:"applying_snap_count"`
	IsBusy             bool        `json:"is_busy"`

	StartTS         time.Time   `json:"start_ts"`
	LastHeartbeatTS time.Time   `json:"last_heartbeat_ts"`
	Uptime          pd.Duration `json:"uptime"`
}

// StoreInfo is a single store info returned from PD RESTful interface.
type StoreInfo struct {
	Store  *MetaStore   `json:"store"`
	Status *StoreStatus `json:"status"`
}

// StoresInfo is stores info returned from PD RESTful interface
type StoresInfo struct {
	Count  int          `json:"count"`
	Stores []*StoreInfo `json:"stores"`
}

// SchedulerInfo is a single scheduler info returned from PD RESTful interface.
type SchedulerInfo struct {
	Name    string `json:"name"`
	StoreID uint64 `json:"store_id"`
}

var ErrTiKVNotBootstrapped = errors.New("TiKV is not bootstrapped")

// IsTiKVNotBootstrappedError returns whether err is a TiKVNotBootstrappedError.
func IsTiKVNotBootstrappedError(err error) bool {
	return errors.Is(err, ErrTiKVNotBootstrapped)
}
