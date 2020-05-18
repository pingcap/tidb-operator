// Copyright 2019 PingCAP, Inc.
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
	ForceNewCluster *bool `json:"force-new-cluster,omitempty"`
	// Optional: Defaults to true
	// +optional
	EnableGRPCGateway *bool `json:"enable-grpc-gateway,omitempty"`

	// LeaderLease time, if leader doesn't update its TTL
	// in etcd after lease time, etcd will expire the leader key
	// and other servers can campaign the leader again.
	// Etcd only supports seconds TTL, so here is second too.
	// Optional: Defaults to 3
	// +optional
	LeaderLease *int64 `toml:"lease,omitempty" json:"lease,omitempty"`

	// Log related config.
	// +optional
	Log *PDLogConfig `toml:"log,omitempty" json:"log,omitempty"`

	// Backward compatibility.
	// +optional
	LogFileDeprecated *string `toml:"log-file,omitempty" json:"log-file,omitempty"`
	// +optional
	LogLevelDeprecated *string `toml:"log-level,omitempty" json:"log-level,omitempty"`

	// TsoSaveInterval is the interval to save timestamp.
	// Optional: Defaults to 3s
	// +optional
	TsoSaveInterval *string `toml:"tso-save-interval,omitempty" json:"tso-save-interval,omitempty"`

	// +optional
	Metric *PDMetricConfig `toml:"metric,omitempty" json:"metric,omitempty"`

	// Immutable, change should be made through pd-ctl after cluster creation
	// +optional
	Schedule *PDScheduleConfig `toml:"schedule,omitempty" json:"schedule,omitempty"`

	// Immutable, change should be made through pd-ctl after cluster creation
	// +optional
	Replication *PDReplicationConfig `toml:"replication,omitempty" json:"replication,omitempty"`

	// +optional
	Namespace map[string]PDNamespaceConfig `toml:"namespace,omitempty" json:"namespace,omitempty"`

	// +optional
	PDServerCfg *PDServerConfig `toml:"pd-server,omitempty" json:"pd-server,omitempty"`

	// +optional
	ClusterVersion *string `toml:"cluster-version,omitempty" json:"cluster-version,omitempty"`

	// QuotaBackendBytes Raise alarms when backend size exceeds the given quota. 0 means use the default quota.
	// the default size is 2GB, the maximum is 8GB.
	// +optional
	QuotaBackendBytes *string `toml:"quota-backend-bytes,omitempty" json:"quota-backend-bytes,omitempty"`
	// AutoCompactionMode is either 'periodic' or 'revision'. The default value is 'periodic'.
	// +optional
	AutoCompactionMode *string `toml:"auto-compaction-mode,omitempty" json:"auto-compaction-mode,omitempty"`
	// AutoCompactionRetention is either duration string with time unit
	// (e.g. '5m' for 5-minute), or revision unit (e.g. '5000').
	// If no time unit is provided and compaction mode is 'periodic',
	// the unit defaults to hour. For example, '5' translates into 5-hour.
	// The default retention is 1 hour.
	// Before etcd v3.3.x, the type of retention is int. We add 'v2' suffix to make it backward compatible.
	// +optional
	AutoCompactionRetention *string `toml:"auto-compaction-retention,omitempty" json:"auto-compaction-retention-v2,omitempty"`

	// TickInterval is the interval for etcd Raft tick.
	// +optional
	TickInterval *string `toml:"tick-interval,omitempty" json:"tikv-interval,omitempty"`
	// ElectionInterval is the interval for etcd Raft election.
	// +optional
	ElectionInterval *string `toml:"election-interval,omitempty" json:"election-interval,omitempty"`
	// Prevote is true to enable Raft Pre-Vote.
	// If enabled, Raft runs an additional election phase
	// to check whether it would get enough votes to win
	// an election, thus minimizing disruptions.
	// Optional: Defaults to true
	// +optional
	PreVote *bool `toml:"enable-prevote,omitempty" json:"enable-prevote,omitempty"`

	// +optional
	Security *PDSecurityConfig `toml:"security,omitempty" json:"security,omitempty"`

	// +optional
	LabelProperty *PDLabelPropertyConfig `toml:"label-property,omitempty" json:"label-property,omitempty"`

	// NamespaceClassifier is for classifying stores/regions into different
	// namespaces.
	// Optional: Defaults to true
	// +optional
	NamespaceClassifier *string `toml:"namespace-classifier,omitempty" json:"namespace-classifier,omitempty"`

	// +optional
	Dashboard *DashboardConfig `toml:"dashboard,omitempty" json:"dashboard,omitempty"`
}

// DashboardConfig is the configuration for tidb-dashboard.
type DashboardConfig struct {
	// +optional
	TiDBCAPath *string `toml:"tidb-cacert-path,omitempty" json:"tidb_cacert_path,omitempty"`
	// +optional
	TiDBCertPath *string `toml:"tidb-cert-path,omitempty" json:"tidb_cert_path,omitempty"`
	// +optional
	TiDBKeyPath *string `toml:"tidb-key-path,omitempty" json:"tidb_key_path,omitempty"`
}

// PDLogConfig serializes log related config in toml/json.
// +k8s:openapi-gen=true
type PDLogConfig struct {
	// Log level.
	// Optional: Defaults to info
	// +optional
	Level *string `toml:"level,omitempty" json:"level,omitempty"`
	// Log format. one of json, text, or console.
	// +optional
	Format *string `toml:"format,omitempty" json:"format,omitempty"`
	// Disable automatic timestamps in output.
	// +optional
	DisableTimestamp *bool `toml:"disable-timestamp,omitempty" json:"disable-timestamp,omitempty"`
	// File log config.
	// +optional
	File *FileLogConfig `toml:"file,omitempty" json:"file,omitempty"`
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
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 3
	// +optional
	MaxReplicas *uint64 `toml:"max-replicas,omitempty" json:"max-replicas,omitempty"`

	// The label keys specified the location of a store.
	// The placement priorities is implied by the order of label keys.
	// For example, ["zone", "rack"] means that we should place replicas to
	// different zones first, then to different racks if we don't have enough zones.
	// Immutable, change should be made through pd-ctl after cluster creation
	// +k8s:openapi-gen=false
	// +optional
	LocationLabels []string `toml:"location-labels,omitempty" json:"location-labels,omitempty"`
	// StrictlyMatchLabel strictly checks if the label of TiKV is matched with LocaltionLabels.
	// Immutable, change should be made through pd-ctl after cluster creation.
	// Imported from v3.1.0
	// +optional
	StrictlyMatchLabel *bool `toml:"strictly-match-label,omitempty" json:"strictly-match-label,string,omitempty"`

	// When PlacementRules feature is enabled. MaxReplicas and LocationLabels are not used anymore.
	// +optional
	EnablePlacementRules *bool `toml:"enable-placement-rules" json:"enable-placement-rules,string,omitempty"`
}

// PDNamespaceConfig is to overwrite the global setting for specific namespace
// +k8s:openapi-gen=true
type PDNamespaceConfig struct {
	// LeaderScheduleLimit is the max coexist leader schedules.
	// +optional
	LeaderScheduleLimit *uint64 `json:"leader-schedule-limit,omitempty" toml:"leader-schedule-limit,omitempty"`
	// RegionScheduleLimit is the max coexist region schedules.
	// +optional
	RegionScheduleLimit *uint64 `json:"region-schedule-limit,omitempty" toml:"region-schedule-limit,omitempty"`
	// ReplicaScheduleLimit is the max coexist replica schedules.
	// +optional
	ReplicaScheduleLimit *uint64 `json:"replica-schedule-limit,omitempty" toml:"replica-schedule-limit,omitempty"`
	// MergeScheduleLimit is the max coexist merge schedules.
	// +optional
	MergeScheduleLimit *uint64 `json:"merge-schedule-limit,omitempty" toml:"merge-schedule-limit,omitempty"`
	// HotRegionScheduleLimit is the max coexist hot region schedules.
	// +optional
	HotRegionScheduleLimit *uint64 `json:"hot-region-schedule-limit,omitempty" toml:"hot-region-schedule-limit,omitempty"`
	// MaxReplicas is the number of replicas for each region.
	// +optional
	MaxReplicas *uint64 `json:"max-replicas,omitempty" toml:"max-replicas,omitempty"`
}

// ScheduleConfig is the schedule configuration.
// +k8s:openapi-gen=true
type PDScheduleConfig struct {
	// If the snapshot count of one store is greater than this value,
	// it will never be used as a source or target store.
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 3
	// +optional
	MaxSnapshotCount *uint64 `toml:"max-snapshot-count,omitempty" json:"max-snapshot-count,omitempty"`
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 16
	// +optional
	MaxPendingPeerCount *uint64 `toml:"max-pending-peer-count,omitempty" json:"max-pending-peer-count,omitempty"`
	// If both the size of region is smaller than MaxMergeRegionSize
	// and the number of rows in region is smaller than MaxMergeRegionKeys,
	// it will try to merge with adjacent regions.
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 20
	// +optional
	MaxMergeRegionSize *uint64 `toml:"max-merge-region-size,omitempty" json:"max-merge-region-size,omitempty"`
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 200000
	// +optional
	MaxMergeRegionKeys *uint64 `toml:"max-merge-region-keys,omitempty" json:"max-merge-region-keys,omitempty"`
	// SplitMergeInterval is the minimum interval time to permit merge after split.
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 1h
	// +optional
	SplitMergeInterval *string `toml:"split-merge-interval,omitempty" json:"split-merge-interval,omitempty"`
	// PatrolRegionInterval is the interval for scanning region during patrol.
	// Immutable, change should be made through pd-ctl after cluster creation
	// +optional
	PatrolRegionInterval *string `toml:"patrol-region-interval,omitempty" json:"patrol-region-interval,omitempty"`
	// MaxStoreDownTime is the max duration after which
	// a store will be considered to be down if it hasn't reported heartbeats.
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 30m
	// +optional
	MaxStoreDownTime *string `toml:"max-store-down-time,omitempty" json:"max-store-down-time,omitempty"`
	// LeaderScheduleLimit is the max coexist leader schedules.
	// Immutable, change should be made through pd-ctl after cluster creation.
	// Optional: Defaults to 4.
	// Imported from v3.1.0
	// +optional
	LeaderScheduleLimit *uint64 `toml:"leader-schedule-limit,omitempty" json:"leader-schedule-limit,omitempty"`
	// RegionScheduleLimit is the max coexist region schedules.
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 2048
	// +optional
	RegionScheduleLimit *uint64 `toml:"region-schedule-limit,omitempty" json:"region-schedule-limit,omitempty"`
	// ReplicaScheduleLimit is the max coexist replica schedules.
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 64
	// +optional
	ReplicaScheduleLimit *uint64 `toml:"replica-schedule-limit,omitempty" json:"replica-schedule-limit,omitempty"`
	// MergeScheduleLimit is the max coexist merge schedules.
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 8
	// +optional
	MergeScheduleLimit *uint64 `toml:"merge-schedule-limit,omitempty" json:"merge-schedule-limit,omitempty"`
	// HotRegionScheduleLimit is the max coexist hot region schedules.
	// Immutable, change should be made through pd-ctl after cluster creation
	// Optional: Defaults to 4
	// +optional
	HotRegionScheduleLimit *uint64 `toml:"hot-region-schedule-limit,omitempty" json:"hot-region-schedule-limit,omitempty"`
	// HotRegionCacheHitThreshold is the cache hits threshold of the hot region.
	// If the number of times a region hits the hot cache is greater than this
	// threshold, it is considered a hot region.
	// Immutable, change should be made through pd-ctl after cluster creation
	// +optional
	HotRegionCacheHitsThreshold *uint64 `toml:"hot-region-cache-hits-threshold,omitempty" json:"hot-region-cache-hits-threshold,omitempty"`
	// TolerantSizeRatio is the ratio of buffer size for balance scheduler.
	// Immutable, change should be made through pd-ctl after cluster creation.
	// Imported from v3.1.0
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
	// Immutable, change should be made through pd-ctl after cluster creation
	// +optional
	LowSpaceRatio *float64 `toml:"low-space-ratio,omitempty" json:"low-space-ratio,omitempty"`
	// HighSpaceRatio is the highest usage ratio of store which regraded as high space.
	// High space means there is a lot of spare capacity, and store region score varies directly with used size.
	// Immutable, change should be made through pd-ctl after cluster creation
	// +optional
	HighSpaceRatio *float64 `toml:"high-space-ratio,omitempty" json:"high-space-ratio,omitempty"`
	// DisableLearner is the option to disable using AddLearnerNode instead of AddNode
	// Immutable, change should be made through pd-ctl after cluster creation
	// +optional
	DisableLearner *bool `toml:"disable-raft-learner,omitempty" json:"disable-raft-learner,string,omitempty"`

	// DisableRemoveDownReplica is the option to prevent replica checker from
	// removing down replicas.
	// Immutable, change should be made through pd-ctl after cluster creation
	// +optional
	DisableRemoveDownReplica *bool `toml:"disable-remove-down-replica,omitempty" json:"disable-remove-down-replica,string,omitempty"`
	// DisableReplaceOfflineReplica is the option to prevent replica checker from
	// repalcing offline replicas.
	// Immutable, change should be made through pd-ctl after cluster creation
	// +optional
	DisableReplaceOfflineReplica *bool `toml:"disable-replace-offline-replica,omitempty" json:"disable-replace-offline-replica,string,omitempty"`
	// DisableMakeUpReplica is the option to prevent replica checker from making up
	// replicas when replica count is less than expected.
	// Immutable, change should be made through pd-ctl after cluster creation
	// +optional
	DisableMakeUpReplica *bool `toml:"disable-make-up-replica,omitempty" json:"disable-make-up-replica,string,omitempty"`
	// DisableRemoveExtraReplica is the option to prevent replica checker from
	// removing extra replicas.
	// Immutable, change should be made through pd-ctl after cluster creation
	// +optional
	DisableRemoveExtraReplica *bool `toml:"disable-remove-extra-replica,omitempty" json:"disable-remove-extra-replica,string,omitempty"`
	// DisableLocationReplacement is the option to prevent replica checker from
	// moving replica to a better location.
	// Immutable, change should be made through pd-ctl after cluster creation
	// +optional
	DisableLocationReplacement *bool `toml:"disable-location-replacement,omitempty" json:"disable-location-replacement,string,omitempty"`
	// DisableNamespaceRelocation is the option to prevent namespace checker
	// from moving replica to the target namespace.
	// Immutable, change should be made through pd-ctl after cluster creation
	// +optional
	DisableNamespaceRelocation *bool `toml:"disable-namespace-relocation,omitempty" json:"disable-namespace-relocation,string,omitempty"`

	// Schedulers support for loding customized schedulers
	// Immutable, change should be made through pd-ctl after cluster creation
	// +optional
	Schedulers *PDSchedulerConfigs `toml:"schedulers,omitempty" json:"schedulers-v2,omitempty"` // json v2 is for the sake of compatible upgrade

	// Only used to display
	// +optional
	SchedulersPayload map[string]string `toml:"schedulers-payload,omitempty" json:"schedulers-payload,omitempty"`

	// EnableOneWayMerge is the option to enable one way merge. This means a Region can only be merged into the next region of it.
	// Imported from v3.1.0
	// +optional
	EnableOneWayMerge *bool `toml:"enable-one-way-merge,omitempty" json:"enable-one-way-merge,string,omitempty"`
	// EnableCrossTableMerge is the option to enable cross table merge. This means two Regions can be merged with different table IDs.
	// This option only works when key type is "table".
	// Imported from v3.1.0
	// +optional
	EnableCrossTableMerge *bool `toml:"enable-cross-table-merge,omitempty" json:"enable-cross-table-merge,string,omitempty"`
}

type PDSchedulerConfigs []PDSchedulerConfig

// PDSchedulerConfig is customized scheduler configuration
// +k8s:openapi-gen=true
type PDSchedulerConfig struct {
	// Immutable, change should be made through pd-ctl after cluster creation
	// +optional
	Type *string `toml:"type,omitempty" json:"type,omitempty"`
	// Immutable, change should be made through pd-ctl after cluster creation
	// +optional
	Args []string `toml:"args,omitempty" json:"args,omitempty"`
	// Immutable, change should be made through pd-ctl after cluster creation
	// +optional
	Disable *bool `toml:"disable,omitempty" json:"disable,omitempty"`
}

// PDStoreLabel is the config item of LabelPropertyConfig.
// +k8s:openapi-gen=true
type PDStoreLabel struct {
	// +optional
	Key *string `toml:"key,omitempty" json:"key,omitempty"`
	// +optional
	Value *string `toml:"value,omitempty" json:"value,omitempty"`
}

type PDStoreLabels []PDStoreLabel

type PDLabelPropertyConfig map[string]PDStoreLabels

// PDSecurityConfig is the configuration for supporting tls.
// +k8s:openapi-gen=true
type PDSecurityConfig struct {
	// CAPath is the path of file that contains list of trusted SSL CAs. if set, following four settings shouldn't be empty
	// +optional
	CAPath *string `toml:"cacert-path,omitempty" json:"cacert-path,omitempty"`
	// CertPath is the path of file that contains X509 certificate in PEM format.
	// +optional
	CertPath *string `toml:"cert-path,omitempty" json:"cert-path,omitempty"`
	// KeyPath is the path of file that contains X509 key in PEM format.
	// +optional
	KeyPath *string `toml:"key-path,omitempty" json:"key-path,omitempty"`
	// CertAllowedCN is the Common Name that allowed
	// +optional
	// +k8s:openapi-gen=false
	CertAllowedCN []string `toml:"cert-allowed-cn,omitempty" json:"cert-allowed-cn,omitempty"`
}

// PDServerConfig is the configuration for pd server.
// +k8s:openapi-gen=true
type PDServerConfig struct {
	// UseRegionStorage enables the independent region storage.
	// +optional
	UseRegionStorage *bool `toml:"use-region-storage,omitempty" json:"use-region-storage,string,omitempty"`
	// MetricStorage is the cluster metric storage.
	// Currently we use prometheus as metric storage, we may use PD/TiKV as metric storage later.
	// Imported from v3.1.0
	// +optional
	MetricStorage *string `toml:"metric-storage,omitempty" json:"metric-storage,omitempty"`
}

// +k8s:openapi-gen=true
type PDMetricConfig struct {
	// +optional
	PushJob *string `toml:"job,omitempty" json:"job,omitempty"`
	// +optional
	PushAddress *string `toml:"address,omitempty" json:"address,omitempty"`
	// +optional
	PushInterval *string `toml:"interval,omitempty" json:"interval,omitempty"`
}

// +k8s:openapi-gen=true
type FileLogConfig struct {
	// Log filename, leave empty to disable file log.
	// +optional
	Filename *string `toml:"filename,omitempty" json:"filename,omitempty"`
	// Is log rotate enabled.
	// +optional
	LogRotate *bool `toml:"log-rotate,omitempty" json:"log-rotate,omitempty"`
	// Max size for a single file, in MB.
	// +optional
	MaxSize *int `toml:"max-size,omitempty" json:"max-size,omitempty"`
	// Max log keep days, default is never deleting.
	// +optional
	MaxDays *int `toml:"max-days,omitempty" json:"max-days,omitempty"`
	// Maximum number of old log files to retain.
	// +optional
	MaxBackups *int `toml:"max-backups,omitempty" json:"max-backups,omitempty"`
}
