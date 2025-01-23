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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// StoreList is the list of store
type StoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Store `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// Store is the object of store
type Store struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Invalid means pd svc is unavailable and store info is untrusted
	Invalid bool `json:"invalid,omitempty"`

	ID      string `json:"id,omitempty"`
	Address string `json:"address,omitempty"`
	Version string `json:"version,omitempty"`
	// PeerAddress         string `json:"peer_address,omitempty"`
	// StatusAddress       string `json:"status_address,omitempty"`
	// GitHash             string `json:"git_hash,omitempty"`
	// DeployPath          string `json:"deploy_path,omitempty"`
	PhysicallyDestroyed bool `json:"physically_destroyed,omitempty"`

	State     StoreState `json:"state,omitempty"`
	NodeState NodeState  `json:"node_state,omitempty"`

	StartTimestamp int64 `json:"start_timestamp,omitempty"`
	// LastHeartbeat  int64 `json:"last_heartbeat,omitempty"`

	LeaderCount int `json:"leader_count"`
	RegionCount int `json:"region_count"`
	// SendingSnapCount   uint32      `json:"sending_snap_count"`
	// ReceivingSnapCount uint32      `json:"receiving_snap_count"`
	// ApplyingSnapCount  uint32      `json:"applying_snap_count"`
	// IsBusy             bool        `json:"is_busy"`
}

func (s *Store) Engine() StoreEngine {
	if s == nil {
		return ""
	}
	if e, ok := s.Labels["engine"]; ok {
		return StoreEngine(e)
	}
	return StoreEngineTiKV
}

type StoreEngine string

const (
	StoreEngineTiKV    StoreEngine = "tikv"
	StoreEngineTiFlash StoreEngine = "tiflash"
)

type StoreState string

const (
	StoreStateUp        StoreState = "Up"
	StoreStateOffline   StoreState = "Offline"
	StoreStateTombstore StoreState = "Tombstone"
)

type NodeState string

const (
	NodeStatePreparing NodeState = "Preparing"
	NodeStateServing   NodeState = "Serving"
	NodeStateRemoving  NodeState = "Removing"
	NodeStateRemoved   NodeState = "Removed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// MemberList is the list of pd members
type MemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Member `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// Member is the object of pd member
type Member struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Invalid means pd svc is unavailable and store info is untrusted
	Invalid bool `json:"invalid,omitempty"`

	ClusterID      string   `json:"cluster_id,omitempty"`
	ID             string   `json:"id"`
	PeerUrls       []string `json:"peer_urls,omitempty"`
	ClientUrls     []string `json:"client_urls,omitempty"`
	LeaderPriority int32    `json:"leader_priority,omitempty"`

	IsLeader     bool `json:"is_leader"`
	IsEtcdLeader bool `json:"is_etcd_leader"`
	Health       bool `json:"health"`
}
