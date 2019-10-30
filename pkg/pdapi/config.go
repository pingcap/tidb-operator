// Copyright 2019. PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package pdapi

import (
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/pingcap/log"
	"github.com/pingcap/pd/pkg/typeutil"
)

// currently, import pd server.Config will introduce a great deal of dependencies and
// the etcd dependency is conflicted with k8s.io/apiserver, as a workaround, we copy the
// type definition from PD and get rid of the dependency of pd/server
// TODO: use type definition from pd when pd separate the config to proper package

// Copied from pd/server/config.go
type Config struct {
	Version bool `json:"-"`

	ConfigCheck bool `json:"-"`

	ClientUrls          string `toml:"client-urls" json:"client-urls"`
	PeerUrls            string `toml:"peer-urls" json:"peer-urls"`
	AdvertiseClientUrls string `toml:"advertise-client-urls" json:"advertise-client-urls"`
	AdvertisePeerUrls   string `toml:"advertise-peer-urls" json:"advertise-peer-urls"`

	Name    string `toml:"name" json:"name"`
	DataDir string `toml:"data-dir" json:"data-dir"`

	InitialCluster      string `toml:"initial-cluster" json:"initial-cluster"`
	InitialClusterState string `toml:"initial-cluster-state" json:"initial-cluster-state"`

	// Join to an existing pd cluster, a string of endpoints.
	Join string `toml:"join" json:"join"`

	// LeaderLease time, if leader doesn't update its TTL
	// in etcd after lease time, etcd will expire the leader key
	// and other servers can campaign the leader again.
	// Etcd onlys support seoncds TTL, so here is second too.
	LeaderLease int64 `toml:"lease" json:"lease"`

	// Log related config.
	Log log.Config `toml:"log" json:"log"`

	// Backward compatibility.
	LogFileDeprecated  string `toml:"log-file" json:"log-file"`
	LogLevelDeprecated string `toml:"log-level" json:"log-level"`

	// TsoSaveInterval is the interval to save timestamp.
	TsoSaveInterval typeutil.Duration `toml:"tso-save-interval" json:"tso-save-interval"`

	Metric interface{} `toml:"metric" json:"metric"`

	Schedule ScheduleConfig `toml:"schedule" json:"schedule"`

	Replication ReplicationConfig `toml:"replication" json:"replication"`

	Namespace map[string]interface{} `json:"namespace"`

	ClusterVersion semver.Version `json:"cluster-version"`

	// QuotaBackendBytes Raise alarms when backend size exceeds the given quota. 0 means use the default quota.
	// the default size is 2GB, the maximum is 8GB.
	QuotaBackendBytes typeutil.ByteSize `toml:"quota-backend-bytes" json:"quota-backend-bytes"`
	// AutoCompactionMode is either 'periodic' or 'revision'. The default value is 'periodic'.
	AutoCompactionMode string `toml:"auto-compaction-mode" json:"auto-compaction-mode"`
	// AutoCompactionRetention is either duration string with time unit
	// (e.g. '5m' for 5-minute), or revision unit (e.g. '5000').
	// If no time unit is provided and compaction mode is 'periodic',
	// the unit defaults to hour. For example, '5' translates into 5-hour.
	// The default retention is 1 hour.
	// Before etcd v3.3.x, the type of retention is int. We add 'v2' suffix to make it backward compatible.
	AutoCompactionRetention string `toml:"auto-compaction-retention" json:"auto-compaction-retention-v2"`

	// TickInterval is the interval for etcd Raft tick.
	TickInterval typeutil.Duration `toml:"tick-interval"`
	// ElectionInterval is the interval for etcd Raft election.
	ElectionInterval typeutil.Duration `toml:"election-interval"`
	// Prevote is true to enable Raft Pre-Vote.
	// If enabled, Raft runs an additional election phase
	// to check whether it would get enough votes to win
	// an election, thus minimizing disruptions.
	PreVote bool `toml:"enable-prevote"`

	Security interface{} `toml:"security" json:"security"`

	LabelProperty interface{} `toml:"label-property" json:"label-property"`

	configFile string

	// For all warnings during parsing.
	WarningMsgs []string

	// NamespaceClassifier is for classifying stores/regions into different
	// namespaces.
	NamespaceClassifier string `toml:"namespace-classifier" json:"namespace-classifier"`

	// Only test can change them.
	nextRetryDelay             time.Duration
	disableStrictReconfigCheck bool

	heartbeatStreamBindInterval typeutil.Duration

	leaderPriorityCheckInterval typeutil.Duration
}

type ReplicationConfig struct {
	// MaxReplicas is the number of replicas for each region.
	MaxReplicas uint64 `toml:"max-replicas,omitempty" json:"max-replicas"`

	// The label keys specified the location of a store.
	// The placement priorities is implied by the order of label keys.
	// For example, ["zone", "rack"] means that we should place replicas to
	// different zones first, then to different racks if we don't have enough zones.
	LocationLabels typeutil.StringSlice `toml:"location-labels,omitempty" json:"location-labels"`
}

// ScheduleConfig is the schedule configuration.
type ScheduleConfig struct {
	// If the snapshot count of one store is greater than this value,
	// it will never be used as a source or target store.
	MaxSnapshotCount    uint64 `toml:"max-snapshot-count,omitempty" json:"max-snapshot-count"`
	MaxPendingPeerCount uint64 `toml:"max-pending-peer-count,omitempty" json:"max-pending-peer-count"`
	// If both the size of region is smaller than MaxMergeRegionSize
	// and the number of rows in region is smaller than MaxMergeRegionKeys,
	// it will try to merge with adjacent regions.
	MaxMergeRegionSize uint64 `toml:"max-merge-region-size,omitempty" json:"max-merge-region-size"`
	MaxMergeRegionKeys uint64 `toml:"max-merge-region-keys,omitempty" json:"max-merge-region-keys"`
	// SplitMergeInterval is the minimum interval time to permit merge after split.
	SplitMergeInterval typeutil.Duration `toml:"split-merge-interval,omitempty" json:"split-merge-interval"`
	// PatrolRegionInterval is the interval for scanning region during patrol.
	PatrolRegionInterval typeutil.Duration `toml:"patrol-region-interval,omitempty" json:"patrol-region-interval"`
	// MaxStoreDownTime is the max duration after which
	// a store will be considered to be down if it hasn't reported heartbeats.
	MaxStoreDownTime typeutil.Duration `toml:"max-store-down-time,omitempty" json:"max-store-down-time"`
	// LeaderScheduleLimit is the max coexist leader schedules.
	LeaderScheduleLimit uint64 `toml:"leader-schedule-limit,omitempty" json:"leader-schedule-limit"`
	// RegionScheduleLimit is the max coexist region schedules.
	RegionScheduleLimit uint64 `toml:"region-schedule-limit,omitempty" json:"region-schedule-limit"`
	// ReplicaScheduleLimit is the max coexist replica schedules.
	ReplicaScheduleLimit uint64 `toml:"replica-schedule-limit,omitempty" json:"replica-schedule-limit"`
	// MergeScheduleLimit is the max coexist merge schedules.
	MergeScheduleLimit uint64 `toml:"merge-schedule-limit,omitempty" json:"merge-schedule-limit"`
	// HotRegionScheduleLimit is the max coexist hot region schedules.
	HotRegionScheduleLimit uint64 `toml:"hot-region-schedule-limit,omitempty" json:"hot-region-schedule-limit"`
	// HotRegionCacheHitThreshold is the cache hits threshold of the hot region.
	// If the number of times a region hits the hot cache is greater than this
	// threshold, it is considered a hot region.
	HotRegionCacheHitsThreshold uint64 `toml:"hot-region-cache-hits-threshold,omitempty" json:"hot-region-cache-hits-threshold"`
	// TolerantSizeRatio is the ratio of buffer size for balance scheduler.
	TolerantSizeRatio float64 `toml:"tolerant-size-ratio,omitempty" json:"tolerant-size-ratio"`
	//
	//      high space stage         transition stage           low space stage
	//   |--------------------|-----------------------------|-------------------------|
	//   ^                    ^                             ^                         ^
	//   0       HighSpaceRatio * capacity       LowSpaceRatio * capacity          capacity
	//
	// LowSpaceRatio is the lowest usage ratio of store which regraded as low space.
	// When in low space, store region score increases to very large and varies inversely with available size.
	LowSpaceRatio float64 `toml:"low-space-ratio,omitempty" json:"low-space-ratio"`
	// HighSpaceRatio is the highest usage ratio of store which regraded as high space.
	// High space means there is a lot of spare capacity, and store region score varies directly with used size.
	HighSpaceRatio float64 `toml:"high-space-ratio,omitempty" json:"high-space-ratio"`
	// DisableLearner is the option to disable using AddLearnerNode instead of AddNode
	DisableLearner bool `toml:"disable-raft-learner" json:"disable-raft-learner,string"`

	// DisableRemoveDownReplica is the option to prevent replica checker from
	// removing down replicas.
	DisableRemoveDownReplica bool `toml:"disable-remove-down-replica" json:"disable-remove-down-replica,string"`
	// DisableReplaceOfflineReplica is the option to prevent replica checker from
	// repalcing offline replicas.
	DisableReplaceOfflineReplica bool `toml:"disable-replace-offline-replica" json:"disable-replace-offline-replica,string"`
	// DisableMakeUpReplica is the option to prevent replica checker from making up
	// replicas when replica count is less than expected.
	DisableMakeUpReplica bool `toml:"disable-make-up-replica" json:"disable-make-up-replica,string"`
	// DisableRemoveExtraReplica is the option to prevent replica checker from
	// removing extra replicas.
	DisableRemoveExtraReplica bool `toml:"disable-remove-extra-replica" json:"disable-remove-extra-replica,string"`
	// DisableLocationReplacement is the option to prevent replica checker from
	// moving replica to a better location.
	DisableLocationReplacement bool `toml:"disable-location-replacement" json:"disable-location-replacement,string"`
	// DisableNamespaceRelocation is the option to prevent namespace checker
	// from moving replica to the target namespace.
	DisableNamespaceRelocation bool `toml:"disable-namespace-relocation" json:"disable-namespace-relocation,string"`

	// Schedulers support for loding customized schedulers
	Schedulers SchedulerConfigs `toml:"schedulers,omitempty" json:"schedulers-v2"` // json v2 is for the sake of compatible upgrade
}

type SchedulerConfigs []SchedulerConfig

// SchedulerConfig is customized scheduler configuration
type SchedulerConfig struct {
	Type    string   `toml:"type" json:"type"`
	Args    []string `toml:"args,omitempty" json:"args"`
	Disable bool     `toml:"disable" json:"disable"`
}
