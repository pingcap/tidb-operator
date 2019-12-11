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

package v1alpha1

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pingcap/log"
	"github.com/pingcap/pd/pkg/metricutil"
	"github.com/pingcap/pd/pkg/typeutil"
)

// Maintain a copy of PDConfig to make it more friendly with the kubernetes API:
//
//  - add 'omitempty' json and toml tag to avoid passing the empty value of primitive types to tidb-server, e.g. 0 of int
//  - change all numeric type to pointer, e.g. uint -> *uint, so that 'omitempty' could work properly
//  - add openapi-gen tags so that the kubernetes-style OpenAPI schema could be properly generated
//  - make the whole config struct deepcopy friendly
//
// only adding field is allowed, DO NOT change existing field definition or remove field.
// some fields maybe illegal for certain version of TiDB, this should be addressed in the ValidationWebhook.

// initially copied from PD v3.0.6

// PDConfig is the configuration of pd-server
// +k8s:openapi-gen=true
type PDConfig struct {
	// +optional
	Version *bool `json:"-"`
	// +optional
	ConfigCheck *bool `json:"-"`

	// +optional
	ClientUrls string `toml:"client-urls,omitempty" json:"client-urls,omitempty"`
	// +optional
	PeerUrls string `toml:"peer-urls,omitempty" json:"peer-urls,omitempty"`
	// +optional
	AdvertiseClientUrls string `toml:"advertise-client-urls,omitempty" json:"advertise-client-urls,omitempty"`
	// +optional
	AdvertisePeerUrls string `toml:"advertise-peer-urls,omitempty" json:"advertise-peer-urls,omitempty"`

	// +optional
	Name string `toml:"name,omitempty" json:"name,omitempty"`
	// +optional
	DataDir string `toml:"data-dir,omitempty" json:"data-dir,omitempty"`
	// +optional
	ForceNewCluster *bool `json:"force-new-cluster,omitempty"`
	// +optional
	EnableGRPCGateway *bool `json:"enable-grpc-gateway,omitempty"`

	// +optional
	InitialCluster string `toml:"initial-cluster,omitempty" json:"initial-cluster,omitempty"`
	// +optional
	InitialClusterState string `toml:"initial-cluster-state,omitempty" json:"initial-cluster-state,omitempty"`

	// Join to an existing pd cluster, a string of endpoints.
	// +optional
	Join string `toml:"join,omitempty" json:"join,omitempty"`

	// LeaderLease time, if leader doesn't update its TTL
	// in etcd after lease time, etcd will expire the leader key
	// and other servers can campaign the leader again.
	// Etcd only supports seconds TTL, so here is second too.
	// +optional
	LeaderLease *int64 `toml:"lease,omitempty" json:"lease,omitempty"`

	// Log related config.
	// +optional
	Log *PDLogConfig `toml:"log,omitempty" json:"log,omitempty"`

	// Backward compatibility.
	// +optional
	LogFileDeprecated string `toml:"log-file,omitempty" json:"log-file,omitempty"`
	// +optional
	LogLevelDeprecated string `toml:"log-level,omitempty" json:"log-level,omitempty"`

	// TsoSaveInterval is the interval to save timestamp.
	// +optional
	TsoSaveInterval string `toml:"tso-save-interval,omitempty" json:"tso-save-interval,omitempty"`

	// +optional
	Metric *metricutil.MetricConfig `toml:"metric,omitempty" json:"metric,omitempty"`

	// +optional
	Schedule *PDScheduleConfig `toml:"schedule,omitempty" json:"schedule,omitempty"`

	// +optional
	Replication *PDReplicationConfig `toml:"replication,omitempty" json:"replication,omitempty"`

	// +optional
	Namespace map[string]PDNamespaceConfig `json:"namespace,omitempty"`

	// +optional
	PDServerCfg *PDServerConfig `toml:"pd-server,omitempty" json:"pd-server,omitempty"`

	// +optional
	ClusterVersion string `json:"cluster-version,omitempty"`

	// QuotaBackendBytes Raise alarms when backend size exceeds the given quota. 0 means use the default quota.
	// the default size is 2GB, the maximum is 8GB.
	// +optional
	QuotaBackendBytes string `toml:"quota-backend-bytes,omitempty" json:"quota-backend-bytes,omitempty"`
	// AutoCompactionMode is either 'periodic' or 'revision'. The default value is 'periodic'.
	// +optional
	AutoCompactionMode string `toml:"auto-compaction-mode,omitempty" json:"auto-compaction-mode,omitempty"`
	// AutoCompactionRetention is either duration string with time unit
	// (e.g. '5m' for 5-minute), or revision unit (e.g. '5000').
	// If no time unit is provided and compaction mode is 'periodic',
	// the unit defaults to hour. For example, '5' translates into 5-hour.
	// The default retention is 1 hour.
	// Before etcd v3.3.x, the type of retention is int. We add 'v2' suffix to make it backward compatible.
	// +optional
	AutoCompactionRetention string `toml:"auto-compaction-retention,omitempty" json:"auto-compaction-retention-v2,omitempty"`

	// TickInterval is the interval for etcd Raft tick.
	// +optional
	TickInterval string `toml:"tick-interval,omitempty" json:"tikv-interval,omitempty"`
	// ElectionInterval is the interval for etcd Raft election.
	// +optional
	ElectionInterval string `toml:"election-interval,omitempty" json:"election-interval,omitempty"`
	// Prevote is true to enable Raft Pre-Vote.
	// If enabled, Raft runs an additional election phase
	// to check whether it would get enough votes to win
	// an election, thus minimizing disruptions.
	// +optional
	PreVote *bool `toml:"enable-prevote,omitempty" json:"enable-prevote,omitempty"`

	// +optional
	Security *PDSecurityConfig `toml:"security,omitempty" json:"security,omitempty"`

	// +optional
	LabelProperty *PDLabelPropertyConfig `toml:"label-property,omitempty" json:"label-property,omitempty"`

	// NamespaceClassifier is for classifying stores/regions into different
	// namespaces.
	// +optional
	NamespaceClassifier string `toml:"namespace-classifier,omitempty" json:"namespace-classifier,omitempty"`
}

// PDLogConfig serializes log related config in toml/json.
// +k8s:openapi-gen=true
type PDLogConfig struct {
	// Log level.
	// +optional
	Level string `toml:"level,omitempty" json:"level,omitempty"`
	// Log format. one of json, text, or console.
	// +optional
	Format string `toml:"format,omitempty" json:"format,omitempty"`
	// Disable automatic timestamps in output.
	// +optional
	DisableTimestamp *bool `toml:"disable-timestamp,omitempty" json:"disable-timestamp,omitempty"`
	// File log config.
	// +optional
	File log.FileLogConfig `toml:"file,omitempty" json:"file,omitempty"`
	// Development puts the logger in development mode, which changes the
	// behavior of DPanicLevel and takes stacktraces more liberally.
	// +optional
	Development *bool `toml:"development,omitempty" json:"development,omitempty"`
	// DisableCaller stops annotating logs with the calling function's file
	// name and line number. By default, all logs are annotated.
	// +optional
	DisableCaller *bool `toml:"disable-caller,omitempty" json:"disable-caller,omitempty"`
	// DisableStacktrace completely disables automatic stacktrace capturing. By
	// default, stacktraces are captured for WarnLevel and above logs in
	// development and ErrorLevel and above in production.
	// +optional
	DisableStacktrace *bool `toml:"disable-stacktrace,omitempty" json:"disable-stacktrace,omitempty"`
	// DisableErrorVerbose stops annotating logs with the full verbose error
	// message.
	// +optional
	DisableErrorVerbose *bool `toml:"disable-error-verbose,omitempty" json:"disable-error-verbose,omitempty"`
}

// PDReplicationConfig is the replication configuration.
// +k8s:openapi-gen=true
type PDReplicationConfig struct {
	// MaxReplicas is the number of replicas for each region.
	// +optional
	MaxReplicas *uint64 `toml:"max-replicas,omitempty" json:"max-replicas,omitempty"`

	// The label keys specified the location of a store.
	// The placement priorities is implied by the order of label keys.
	// For example, ["zone", "rack"] means that we should place replicas to
	// different zones first, then to different racks if we don't have enough zones.
	// +optional
	LocationLabels typeutil.StringSlice `toml:"location-labels,omitempty" json:"location-labels,omitempty"`
	// StrictlyMatchLabel strictly checks if the label of TiKV is matched with LocaltionLabels.
	// +optional
	StrictlyMatchLabel *bool `toml:"strictly-match-label,omitempty" json:"strictly-match-label,string,omitempty"`
}

// PDNamespaceConfig is to overwrite the global setting for specific namespace
// +k8s:openapi-gen=true
type PDNamespaceConfig struct {
	// LeaderScheduleLimit is the max coexist leader schedules.
	// +optional
	LeaderScheduleLimit *uint64 `json:"leader-schedule-limit,omitempty"`
	// RegionScheduleLimit is the max coexist region schedules.
	// +optional
	RegionScheduleLimit *uint64 `json:"region-schedule-limit,omitempty"`
	// ReplicaScheduleLimit is the max coexist replica schedules.
	// +optional
	ReplicaScheduleLimit *uint64 `json:"replica-schedule-limit,omitempty"`
	// MergeScheduleLimit is the max coexist merge schedules.
	// +optional
	MergeScheduleLimit *uint64 `json:"merge-schedule-limit,omitempty"`
	// HotRegionScheduleLimit is the max coexist hot region schedules.
	// +optional
	HotRegionScheduleLimit *uint64 `json:"hot-region-schedule-limit,omitempty"`
	// MaxReplicas is the number of replicas for each region.
	// +optional
	MaxReplicas *uint64 `json:"max-replicas,omitempty"`
}

// ScheduleConfig is the schedule configuration.
// +k8s:openapi-gen=true
type PDScheduleConfig struct {
	// If the snapshot count of one store is greater than this value,
	// it will never be used as a source or target store.
	// +optional
	MaxSnapshotCount *uint64 `toml:"max-snapshot-count,omitempty" json:"max-snapshot-count,omitempty"`
	// +optional
	MaxPendingPeerCount *uint64 `toml:"max-pending-peer-count,omitempty" json:"max-pending-peer-count,omitempty"`
	// If both the size of region is smaller than MaxMergeRegionSize
	// and the number of rows in region is smaller than MaxMergeRegionKeys,
	// it will try to merge with adjacent regions.
	// +optional
	MaxMergeRegionSize *uint64 `toml:"max-merge-region-size,omitempty" json:"max-merge-region-size,omitempty"`
	// +optional
	MaxMergeRegionKeys *uint64 `toml:"max-merge-region-keys,omitempty" json:"max-merge-region-keys,omitempty"`
	// SplitMergeInterval is the minimum interval time to permit merge after split.
	// +optional
	SplitMergeInterval string `toml:"split-merge-interval,omitempty" json:"split-merge-interval,omitempty"`
	// PatrolRegionInterval is the interval for scanning region during patrol.
	// +optional
	PatrolRegionInterval string `toml:"patrol-region-interval,omitempty" json:"patrol-region-interval,omitempty"`
	// MaxStoreDownTime is the max duration after which
	// a store will be considered to be down if it hasn't reported heartbeats.
	// +optional
	MaxStoreDownTime Duration `toml:"max-store-down-time,omitempty" json:"max-store-down-time,omitempty"`
	// LeaderScheduleLimit is the max coexist leader schedules.
	// +optional
	LeaderScheduleLimit *uint64 `toml:"leader-schedule-limit,omitempty" json:"leader-schedule-limit,omitempty"`
	// RegionScheduleLimit is the max coexist region schedules.
	// +optional
	RegionScheduleLimit *uint64 `toml:"region-schedule-limit,omitempty" json:"region-schedule-limit,omitempty"`
	// ReplicaScheduleLimit is the max coexist replica schedules.
	// +optional
	ReplicaScheduleLimit *uint64 `toml:"replica-schedule-limit,omitempty" json:"replica-schedule-limit,omitempty"`
	// MergeScheduleLimit is the max coexist merge schedules.
	// +optional
	MergeScheduleLimit *uint64 `toml:"merge-schedule-limit,omitempty" json:"merge-schedule-limit,omitempty"`
	// HotRegionScheduleLimit is the max coexist hot region schedules.
	// +optional
	HotRegionScheduleLimit *uint64 `toml:"hot-region-schedule-limit,omitempty" json:"hot-region-schedule-limit,omitempty"`
	// HotRegionCacheHitThreshold is the cache hits threshold of the hot region.
	// If the number of times a region hits the hot cache is greater than this
	// threshold, it is considered a hot region.
	// +optional
	HotRegionCacheHitsThreshold *uint64 `toml:"hot-region-cache-hits-threshold,omitempty" json:"hot-region-cache-hits-threshold,omitempty"`
	// TolerantSizeRatio is the ratio of buffer size for balance scheduler.
	// +optional
	TolerantSizeRatio *float64 `toml:"tolerant-size-ratio,omitempty" json:"tolerant-size-ratio,omitempty"`
	//
	//      high space stage         transition stage           low space stage
	//   |--------------------|-----------------------------|-------------------------|
	//   ^                    ^                             ^                         ^
	//   0       HighSpaceRatio * capacity       LowSpaceRatio * capacity          capacity
	//
	// LowSpaceRatio is the lowest usage ratio of store which regraded as low space.
	// When in low space, store region score increases to very large and varies inversely with available size.
	// +optional
	LowSpaceRatio *float64 `toml:"low-space-ratio,omitempty" json:"low-space-ratio,omitempty"`
	// HighSpaceRatio is the highest usage ratio of store which regraded as high space.
	// High space means there is a lot of spare capacity, and store region score varies directly with used size.
	// +optional
	HighSpaceRatio *float64 `toml:"high-space-ratio,omitempty" json:"high-space-ratio,omitempty"`
	// DisableLearner is the option to disable using AddLearnerNode instead of AddNode
	// +optional
	DisableLearner *bool `toml:"disable-raft-learner,omitempty" json:"disable-raft-learner,string,omitempty"`

	// DisableRemoveDownReplica is the option to prevent replica checker from
	// removing down replicas.
	// +optional
	DisableRemoveDownReplica *bool `toml:"disable-remove-down-replica,omitempty" json:"disable-remove-down-replica,string,omitempty"`
	// DisableReplaceOfflineReplica is the option to prevent replica checker from
	// repalcing offline replicas.
	// +optional
	DisableReplaceOfflineReplica *bool `toml:"disable-replace-offline-replica,omitempty" json:"disable-replace-offline-replica,string,omitempty"`
	// DisableMakeUpReplica is the option to prevent replica checker from making up
	// replicas when replica count is less than expected.
	// +optional
	DisableMakeUpReplica *bool `toml:"disable-make-up-replica,omitempty" json:"disable-make-up-replica,string,omitempty"`
	// DisableRemoveExtraReplica is the option to prevent replica checker from
	// removing extra replicas.
	// +optional
	DisableRemoveExtraReplica *bool `toml:"disable-remove-extra-replica,omitempty" json:"disable-remove-extra-replica,string,omitempty"`
	// DisableLocationReplacement is the option to prevent replica checker from
	// moving replica to a better location.
	// +optional
	DisableLocationReplacement *bool `toml:"disable-location-replacement,omitempty" json:"disable-location-replacement,string,omitempty"`
	// DisableNamespaceRelocation is the option to prevent namespace checker
	// from moving replica to the target namespace.
	// +optional
	DisableNamespaceRelocation *bool `toml:"disable-namespace-relocation,omitempty" json:"disable-namespace-relocation,string,omitempty"`

	// Schedulers support for loding customized schedulers
	// +optional
	Schedulers *PDSchedulerConfigs `toml:"schedulers,omitempty" json:"schedulers-v2,omitempty"` // json v2 is for the sake of compatible upgrade
}

type PDSchedulerConfigs []PDSchedulerConfig

// PDSchedulerConfig is customized scheduler configuration
// +k8s:openapi-gen=true
type PDSchedulerConfig struct {
	// +optional
	Type string `toml:"type,omitempty" json:"type,omitempty"`
	// +optional
	Args []string `toml:"args,omitempty" json:"args,omitempty"`
	// +optional
	Disable *bool `toml:"disable,omitempty" json:"disable,omitempty"`
}

// PDStoreLabel is the config item of LabelPropertyConfig.
// +k8s:openapi-gen=true
type PDStoreLabel struct {
	// +optional
	Key string `toml:"key,omitempty" json:"key,omitempty"`
	// +optional
	Value string `toml:"value,omitempty" json:"value,omitempty"`
}

type PDStoreLabels []PDStoreLabel

type PDLabelPropertyConfig map[string]PDStoreLabels

// PDSecurityConfig is the configuration for supporting tls.
// +k8s:openapi-gen=true
type PDSecurityConfig struct {
	// CAPath is the path of file that contains list of trusted SSL CAs. if set, following four settings shouldn't be empty
	// +optional
	CAPath string `toml:"cacert-path,omitempty" json:"cacert-path,omitempty"`
	// CertPath is the path of file that contains X509 certificate in PEM format.
	// +optional
	CertPath string `toml:"cert-path,omitempty" json:"cert-path,omitempty"`
	// KeyPath is the path of file that contains X509 key in PEM format.
	// +optional
	KeyPath string `toml:"key-path,omitempty" json:"key-path,omitempty"`
}

// PDServerConfig is the configuration for pd server.
// +k8s:openapi-gen=true
type PDServerConfig struct {
	// UseRegionStorage enables the independent region storage.
	// +optional
	UseRegionStorage *bool `toml:"use-region-storage,omitempty" json:"use-region-storage,string,omitempty"`
}

// Duration is a wrapper of time.Duration for TOML and JSON.
// Copied from pingcap typeutil, add marshal to TOML support
type Duration struct {
	time.Duration
}

// NewDuration creates a Duration from time.Duration.
func NewDuration(duration time.Duration) Duration {
	return Duration{Duration: duration}
}

// MarshalJSON returns the duration as a JSON string.
func (d *Duration) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}

// UnmarshalJSON parses a JSON string into the duration.
func (d *Duration) UnmarshalJSON(text []byte) error {
	s, err := strconv.Unquote(string(text))
	if err != nil {
		return err
	}
	duration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = duration
	return nil
}

// UnmarshalText parses a TOML string into the duration.
func (d *Duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

// MarshalText marshal duration to a TOML string
func (d *Duration) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}
