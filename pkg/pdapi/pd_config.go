// Copyright 2020 PingCAP, Inc.
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
	"strconv"
	"strings"
)

// PDConfigFromAPI is the configuration from PD API
// +k8s:openapi-gen=true
type PDConfigFromAPI struct {

	// Log related config.
	Log *PDLogConfig `toml:"log,omitempty" json:"log,omitempty"`

	// Immutable, change should be made through pd-ctl after cluster creation
	Schedule *PDScheduleConfig `toml:"schedule,omitempty" json:"schedule,omitempty"`

	// Immutable, change should be made through pd-ctl after cluster creation
	Replication *PDReplicationConfig `toml:"replication,omitempty" json:"replication,omitempty"`
}

// PDLogConfig serializes log related config in toml/json.
// +k8s:openapi-gen=true
type PDLogConfig struct {
	// Log level.
	// Optional: Defaults to info
	Level string `toml:"level,omitempty" json:"level,omitempty"`
	// Log format. one of json, text, or console.
	Format string `toml:"format,omitempty" json:"format,omitempty"`
	// Disable automatic timestamps in output.
	DisableTimestamp *bool `toml:"disable-timestamp,omitempty" json:"disable-timestamp,omitempty"`
	// File log config.
	File *FileLogConfig `toml:"file,omitempty" json:"file,omitempty"`
	// Development puts the logger in development mode, which changes the
	// behavior of DPanicLevel and takes stacktraces more liberally.
	Development *bool `toml:"development,omitempty" json:"development,omitempty"`
	// DisableCaller stops annotating logs with the calling function's file
	// name and line number. By default, all logs are annotated.
	DisableCaller *bool `toml:"disable-caller,omitempty" json:"disable-caller,omitempty"`
	// DisableStacktrace completely disables automatic stacktrace capturing. By
	// default, stacktraces are captured for WarnLevel and above logs in
	// development and ErrorLevel and above in production.
	DisableStacktrace *bool `toml:"disable-stacktrace,omitempty" json:"disable-stacktrace,omitempty"`
	// DisableErrorVerbose stops annotating logs with the full verbose error
	// message.
	DisableErrorVerbose *bool `toml:"disable-error-verbose,omitempty" json:"disable-error-verbose,omitempty"`
}

// PDReplicationConfig is the replication configuration.
// +k8s:openapi-gen=true
type PDReplicationConfig struct {
	// MaxReplicas is the number of replicas for each region.
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 3
	MaxReplicas *uint64 `toml:"max-replicas,omitempty" json:"max-replicas,omitempty"`

	// The label keys specified the location of a store.
	// The placement priorities is implied by the order of label keys.
	// For example, ["zone", "rack"] means that we should place replicas to
	// different zones first, then to different racks if we don't have enough zones.
	// Immutable, change should be made through pd-ctl after cluster creation
	// +k8s:openapi-gen=false
	LocationLabels StringSlice `toml:"location-labels,omitempty" json:"location-labels,omitempty"`
	// StrictlyMatchLabel strictly checks if the label of TiKV is matched with LocaltionLabels.
	// Immutable, change should be made through pd-ctl after cluster creation.
	// Imported from v3.1.0
	StrictlyMatchLabel *bool `toml:"strictly-match-label,omitempty" json:"strictly-match-label,string,omitempty"`

	// When PlacementRules feature is enabled. MaxReplicas and LocationLabels are not used anymore.
	EnablePlacementRules *bool `toml:"enable-placement-rules" json:"enable-placement-rules,string,omitempty"`
}

// ScheduleConfig is the schedule configuration.
// +k8s:openapi-gen=true
type PDScheduleConfig struct {
	// If the snapshot count of one store is greater than this value,
	// it will never be used as a source or target store.
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 3
	MaxSnapshotCount *uint64 `toml:"max-snapshot-count,omitempty" json:"max-snapshot-count,omitempty"`
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 16
	MaxPendingPeerCount *uint64 `toml:"max-pending-peer-count,omitempty" json:"max-pending-peer-count,omitempty"`
	// If both the size of region is smaller than MaxMergeRegionSize
	// and the number of rows in region is smaller than MaxMergeRegionKeys,
	// it will try to merge with adjacent regions.
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 20
	MaxMergeRegionSize *uint64 `toml:"max-merge-region-size,omitempty" json:"max-merge-region-size,omitempty"`
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 200000
	MaxMergeRegionKeys *uint64 `toml:"max-merge-region-keys,omitempty" json:"max-merge-region-keys,omitempty"`
	// SplitMergeInterval is the minimum interval time to permit merge after split.
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 1h
	SplitMergeInterval string `toml:"split-merge-interval,omitempty" json:"split-merge-interval,omitempty"`
	// PatrolRegionInterval is the interval for scanning region during patrol.
	// Immutable, change should be made through pd-ctl after cluster creation
	PatrolRegionInterval string `toml:"patrol-region-interval,omitempty" json:"patrol-region-interval,omitempty"`
	// MaxStoreDownTime is the max duration after which
	// a store will be considered to be down if it hasn't reported heartbeats.
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 30m
	MaxStoreDownTime string `toml:"max-store-down-time,omitempty" json:"max-store-down-time,omitempty"`
	// LeaderScheduleLimit is the max coexist leader schedules.
	// Immutable, change should be made through pd-ctl after cluster creation.
	// Optional: Defaults to 4.
	// Imported from v3.1.0
	LeaderScheduleLimit *uint64 `toml:"leader-schedule-limit,omitempty" json:"leader-schedule-limit,omitempty"`
	// RegionScheduleLimit is the max coexist region schedules.
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 2048
	RegionScheduleLimit *uint64 `toml:"region-schedule-limit,omitempty" json:"region-schedule-limit,omitempty"`
	// ReplicaScheduleLimit is the max coexist replica schedules.
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 64
	ReplicaScheduleLimit *uint64 `toml:"replica-schedule-limit,omitempty" json:"replica-schedule-limit,omitempty"`
	// MergeScheduleLimit is the max coexist merge schedules.
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 8
	MergeScheduleLimit *uint64 `toml:"merge-schedule-limit,omitempty" json:"merge-schedule-limit,omitempty"`
	// HotRegionScheduleLimit is the max coexist hot region schedules.
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 4
	HotRegionScheduleLimit *uint64 `toml:"hot-region-schedule-limit,omitempty" json:"hot-region-schedule-limit,omitempty"`
	// HotRegionCacheHitThreshold is the cache hits threshold of the hot region.
	// If the number of times a region hits the hot cache is greater than this
	// threshold, it is considered a hot region.
	// Immutable, change should be made through pd-ctl after cluster creation
	HotRegionCacheHitsThreshold *uint64 `toml:"hot-region-cache-hits-threshold,omitempty" json:"hot-region-cache-hits-threshold,omitempty"`
	// TolerantSizeRatio is the ratio of buffer size for balance scheduler.
	// Immutable, change should be made through pd-ctl after cluster creation.
	// Imported from v3.1.0
	TolerantSizeRatio *float64 `toml:"tolerant-size-ratio,omitempty" json:"tolerant-size-ratio,omitempty"`
	//
	//      high space stage         transition stage           low space stage
	//   |--------------------|-----------------------------|-------------------------|
	//   ^                    ^                             ^                         ^
	//   0       HighSpaceRatio * capacity       LowSpaceRatio * capacity          capacity
	//
	// LowSpaceRatio is the lowest usage ratio of store which regraded as low space.
	// When in low space, store region score increases to very large and varies inversely with available size.
	// Immutable, change should be made through pd-ctl after cluster creation
	LowSpaceRatio *float64 `toml:"low-space-ratio,omitempty" json:"low-space-ratio,omitempty"`
	// HighSpaceRatio is the highest usage ratio of store which regraded as high space.
	// High space means there is a lot of spare capacity, and store region score varies directly with used size.
	// Immutable, change should be made through pd-ctl after cluster creation
	HighSpaceRatio *float64 `toml:"high-space-ratio,omitempty" json:"high-space-ratio,omitempty"`
	// DisableLearner is the option to disable using AddLearnerNode instead of AddNode
	// Immutable, change should be made through pd-ctl after cluster creation
	DisableLearner *bool `toml:"disable-raft-learner,omitempty" json:"disable-raft-learner,string,omitempty"`

	// DisableRemoveDownReplica is the option to prevent replica checker from
	// removing down replicas.
	// Immutable, change should be made through pd-ctl after cluster creation
	DisableRemoveDownReplica *bool `toml:"disable-remove-down-replica,omitempty" json:"disable-remove-down-replica,string,omitempty"`
	// DisableReplaceOfflineReplica is the option to prevent replica checker from
	// repalcing offline replicas.
	// Immutable, change should be made through pd-ctl after cluster creation
	DisableReplaceOfflineReplica *bool `toml:"disable-replace-offline-replica,omitempty" json:"disable-replace-offline-replica,string,omitempty"`
	// DisableMakeUpReplica is the option to prevent replica checker from making up
	// replicas when replica count is less than expected.
	// Immutable, change should be made through pd-ctl after cluster creation
	DisableMakeUpReplica *bool `toml:"disable-make-up-replica,omitempty" json:"disable-make-up-replica,string,omitempty"`
	// DisableRemoveExtraReplica is the option to prevent replica checker from
	// removing extra replicas.
	// Immutable, change should be made through pd-ctl after cluster creation
	DisableRemoveExtraReplica *bool `toml:"disable-remove-extra-replica,omitempty" json:"disable-remove-extra-replica,string,omitempty"`
	// DisableLocationReplacement is the option to prevent replica checker from
	// moving replica to a better location.
	// Immutable, change should be made through pd-ctl after cluster creation
	DisableLocationReplacement *bool `toml:"disable-location-replacement,omitempty" json:"disable-location-replacement,string,omitempty"`
	// DisableNamespaceRelocation is the option to prevent namespace checker
	// from moving replica to the target namespace.
	// Immutable, change should be made through pd-ctl after cluster creation
	DisableNamespaceRelocation *bool `toml:"disable-namespace-relocation,omitempty" json:"disable-namespace-relocation,string,omitempty"`

	// Schedulers support for loding customized schedulers
	// Immutable, change should be made through pd-ctl after cluster creation
	Schedulers *PDSchedulerConfigs `toml:"schedulers,omitempty" json:"schedulers-v2,omitempty"` // json v2 is for the sake of compatible upgrade

	// Only used to display
	SchedulersPayload map[string]interface{} `toml:"schedulers-payload" json:"schedulers-payload,omitempty"`

	// EnableOneWayMerge is the option to enable one way merge. This means a Region can only be merged into the next region of it.
	// Imported from v3.1.0
	EnableOneWayMerge *bool `toml:"enable-one-way-merge" json:"enable-one-way-merge,string,omitempty"`
	// EnableCrossTableMerge is the option to enable cross table merge. This means two Regions can be merged with different table IDs.
	// This option only works when key type is "table".
	// Imported from v3.1.0
	EnableCrossTableMerge *bool `toml:"enable-cross-table-merge" json:"enable-cross-table-merge,string,omitempty"`
}

type PDSchedulerConfigs []PDSchedulerConfig

// PDSchedulerConfig is customized scheduler configuration
// +k8s:openapi-gen=true
type PDSchedulerConfig struct {
	// Immutable, change should be made through pd-ctl after cluster creation
	Type string `toml:"type,omitempty" json:"type,omitempty"`
	// Immutable, change should be made through pd-ctl after cluster creation
	Args []string `toml:"args,omitempty" json:"args,omitempty"`
	// Immutable, change should be made through pd-ctl after cluster creation
	Disable *bool `toml:"disable,omitempty" json:"disable,omitempty"`
}

// PDStoreLabel is the config item of LabelPropertyConfig.
// +k8s:openapi-gen=true
type PDStoreLabel struct {
	Key   string `toml:"key,omitempty" json:"key,omitempty"`
	Value string `toml:"value,omitempty" json:"value,omitempty"`
}

type PDStoreLabels []PDStoreLabel

type PDLabelPropertyConfig map[string]PDStoreLabels

// +k8s:openapi-gen=true
type FileLogConfig struct {
	// Log filename, leave empty to disable file log.
	Filename string `toml:"filename,omitempty" json:"filename,omitempty"`
	// Is log rotate enabled.
	LogRotate bool `toml:"log-rotate,omitempty" json:"log-rotate,omitempty"`
	// Max size for a single file, in MB.
	MaxSize int `toml:"max-size,omitempty" json:"max-size,omitempty"`
	// Max log keep days, default is never deleting.
	MaxDays int `toml:"max-days,omitempty" json:"max-days,omitempty"`
	// Maximum number of old log files to retain.
	MaxBackups int `toml:"max-backups,omitempty" json:"max-backups,omitempty"`
}

//StringSlice is more friendly to json encode/decode
type StringSlice []string

// MarshalJSON returns the size as a JSON string.
func (s StringSlice) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(strings.Join(s, ","))), nil
}

// UnmarshalJSON parses a JSON string into the bytesize.
func (s *StringSlice) UnmarshalJSON(text []byte) error {
	data, err := strconv.Unquote(string(text))
	if err != nil {
		return err
	}
	if len(data) == 0 {
		*s = nil
		return nil
	}
	*s = strings.Split(data, ",")
	return nil
}

// evictLeaderSchedulerConfig holds configuration for evict leader
// https://github.com/pingcap/pd/blob/b21855a3aeb787c71b0819743059e432be217dcd/server/schedulers/evict_leader.go#L81-L86
// note that we use `interface{}` as the type of value because we don't care
// about the value for now
type evictLeaderSchedulerConfig struct {
	StoreIDWithRanges map[uint64]interface{} `json:"store-id-ranges"`
}
