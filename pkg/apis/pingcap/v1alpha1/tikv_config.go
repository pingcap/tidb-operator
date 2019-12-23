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

// Port from TiKV v3.0.6

// TiKVConfig is the configuration of TiKV.
// +k8s:openapi-gen=true
type TiKVConfig struct {
	// +optional
	LogLevel string `json:"log-level,omitempty" toml:"log-level,omitempty"`
	// +optional
	LogFile string `json:"log-file,omitempty" toml:"log-file,omitempty"`
	// +optional
	LogRotationTimespan string `json:"log-rotation-timespan,omitempty" toml:"log-rotation-timespan,omitempty"`
	// +optional
	PanicWhenUnexpectedKeyOrData *bool `json:"panic-when-unexpected-key-or-data,omitempty" toml:"panic-when-unexpected-key-or-data,omitempty"`
	// +optional
	Addr string `json:"addr,omitempty" toml:"addr,omitempty"`
	// +optional
	AdvertiseAddr string `json:"advertise-addr,omitempty" toml:"advertise-addr,omitempty"`
	// +optional
	StatusAddr string `json:"status-addr,omitempty" toml:"status-addr,omitempty"`
	// +optional
	StatusThreadPoolSize string `json:"status-thread-pool-size,omitempty" toml:"status-thread-pool-size,omitempty"`
	// +optional
	GrpcCompressionType string `json:"grpc-compression-type,omitempty" toml:"grpc-compression-type,omitempty"`
	// +optional
	GrpcConcurrency *uint `json:"grpc-concurrency,omitempty" toml:"grpc-concurrency,omitempty"`
	// +optional
	GrpcConcurrentStream *uint `json:"grpc-concurrent-stream,omitempty" toml:"grpc-concurrent-stream,omitempty"`
	// +optional
	GrpcRaftConnNum *uint `json:"grpc-raft-conn-num,omitempty" toml:"grpc-raft-conn-num,omitempty"`
	// +optional
	GrpcStreamInitialWindowSize string `json:"grpc-stream-initial-window-size,omitempty" toml:"grpc-stream-initial-window-size,omitempty"`
	// +optional
	GrpcKeepaliveTime string `json:"grpc-keepalive-time,omitempty" toml:"grpc-keepalive-time,omitempty"`
	// +optional
	GrpcKeepaliveTimeout string `json:"grpc-keepalive-timeout,omitempty" toml:"grpc-keepalive-timeout,omitempty"`
	// +optional
	ConcurrentSendSnapLimit *uint `json:"concurrent-send-snap-limit,omitempty" toml:"concurrent-send-snap-limit,omitempty"`
	// +optional
	ConcurrentRecvSnapLimit *uint `json:"concurrent-recv-snap-limit,omitempty" toml:"concurrent-recv-snap-limit,omitempty"`
	// +optional
	EndPointRecursionLimit *uint `json:"end-point-recursion-limit,omitempty" toml:"end-point-recursion-limit,omitempty"`
	// +optional
	EndPointStreamChannelSize *uint `json:"end-point-stream-channel-size,omitempty" toml:"end-point-stream-channel-size,omitempty"`
	// +optional
	EndPointBatchRowLimit *uint `json:"end-point-batch-row-limit,omitempty" toml:"end-point-batch-row-limit,omitempty"`
	// +optional
	EndPointStreamBatchRowLimit *uint `json:"end-point-stream-batch-row-limit,omitempty" toml:"end-point-stream-batch-row-limit,omitempty"`
	// +optional
	EndPointEnableBatchIfPossible *uint `json:"end-point-enable-batch-if-possible,omitempty" toml:"end-point-enable-batch-if-possible,omitempty"`
	// +optional
	EndPointRequestMaxHandleDuration string `json:"end-point-request-max-handle-duration,omitempty" toml:"end-point-request-max-handle-duration,omitempty"`
	// +optional
	SnapMaxWriteBytesPerSec string `json:"snap-max-write-bytes-per-sec,omitempty" toml:"snap-max-write-bytes-per-sec,omitempty"`
	// +optional
	SnapMaxTotalSize string `json:"snap-max-total-size,omitempty" toml:"snap-max-total-size,omitempty"`
	// +optional
	StatsConcurrency *uint `json:"stats-concurrency,omitempty" toml:"stats-concurrency,omitempty"`
	// +optional
	HeavyLoadThreshold *uint `json:"heavy-load-threshold,omitempty" toml:"heavy-load-threshold,omitempty"`
	// +optional
	HeavyLoadWaitDuration string `json:"heavy-load-wait-duration,omitempty" toml:"heavy-load-wait-duration,omitempty"`
	// +optional
	Labels map[string]string `json:"labels,omitempty" toml:"labels,omitempty"`
	// +optional
	Server *TiKVServerConfig `json:"server,omitempty" toml:"server,omitempty"`
	// +optional
	Storage *TiKVStorageConfig `json:"storage,omitempty" toml:"storage,omitempty"`
	// +optional
	Raftstore *TiKVRaftstoreConfig `json:"raftstore,omitempty" toml:"raftstore,omitempty"`
	// +optional
	Rocksdb *TiKVDbConfig `json:"rocksdb,omitempty" toml:"rocksdb,omitempty"`
	// +optional
	Coprocessor *TiKVCoprocessorConfig `json:"coprocessor,omitempty" toml:"coprocessor,omitempty"`
	// +optional
	ReadPool *TiKVReadPoolConfig `json:"readpool,omitempty" toml:"readpool,omitempty"`
	// +optional
	RaftDB *TiKVRaftDBConfig `json:"raftdb,omitempty" toml:"raftdb,omitempty"`
	// +optional
	Import *TiKVImportConfig `json:"import,omitempty" toml:"import,omitempty"`
	// +optional
	GC *TiKVGCConfig `json:"gc,omitempty" toml:"gc,omitempty"`
	// +optional
	Security *TiKVSecurityConfig `json:"security,omitempty" toml:"security,omitempty"`
}

// +k8s:openapi-gen=true
type TiKVReadPoolConfig struct {
	// +optional
	Coprocessor *TiKVComponentReadPoolConfig `json:"coprocessor,omitempty" toml:"coprocessor,omitempty"`
	// +optional
	Storage *TiKVComponentReadPoolConfig `json:"storage,omitempty" toml:"storage,omitempty"`
}

// +k8s:openapi-gen=true
type TiKVComponentReadPoolConfig struct {
	// +optional
	HighConcurrency *int64 `json:"high_concurrency,omitempty" toml:"high_concurrency,omitempty"`
	// +optional
	NormalConcurrency *int64 `json:"normal_concurrency,omitempty" toml:"normal_concurrency,omitempty"`
	// +optional
	LowConcurrency *int64 `json:"low_concurrency,omitempty" toml:"low_concurrency,omitempty"`
	// +optional
	MaxTasksPerWorkerHigh *int64 `json:"max_tasks_per_worker_high,omitempty" toml:"max_tasks_per_worker_high,omitempty"`
	// +optional
	MaxTasksPerWorkerNormal *int64 `json:"max_tasks_per_worker_normal,omitempty" toml:"max_tasks_per_worker_normal,omitempty"`
	// +optional
	MaxTasksPerWorkerLow *int64 `json:"max_tasks_per_worker_low,omitempty" toml:"max_tasks_per_worker_low,omitempty"`
	// +optional
	StackSize string `json:"stack_size,omitempty" toml:"stack_size,omitempty"`
}

// +k8s:openapi-gen=true
type TiKVPDConfig struct {
	// The PD endpoints for the client.
	//
	// Default is empty.
	// +optional
	Endpoints []string `json:"endpoints,omitempty" toml:"endpoints,omitempty"`
	// The interval at which to retry a PD connection initialization.
	//
	// Default is 300ms.
	// +optional
	RetryInterval string `json:"retry_interval,omitempty" toml:"retry_interval,omitempty"`
	// The maximum number of times to retry a PD connection initialization.
	//
	// Default is isize::MAX, represented by -1.
	// +optional
	RetryMaxCount *int64 `json:"retry_max_count,omitempty" toml:"retry_max_count,omitempty"`
	// If the client observes the same error message on retry, it can repeat the message only
	// every `n` times.
	//
	// Default is 10. Set to 1 to disable this feature.
	// +optional
	RetryLogEvery *int64 `json:"retry_log_every,omitempty" toml:"retry_log_every,omitempty"`
}

// +k8s:openapi-gen=true
type TiKVRaftDBConfig struct {
	// +optional
	WalRecoveryMode string `json:"wal_recovery_mode,omitempty" toml:"wal_recovery_mode,omitempty"`
	// +optional
	WalDir string `json:"wal_dir,omitempty" toml:"wal_dir,omitempty"`
	// +optional
	WalTtlSeconds *int64 `json:"wal_ttl_seconds,omitempty" toml:"wal_ttl_seconds,omitempty"`
	// +optional
	WalSizeLimit string `json:"wal_size_limit,omitempty" toml:"wal_size_limit,omitempty"`
	// +optional
	MaxTotalWalSize string `json:"max_total_wal_size,omitempty" toml:"max_total_wal_size,omitempty"`
	// +optional
	MaxBackgroundJobs *int64 `json:"max_background_jobs,omitempty" toml:"max_background_jobs,omitempty"`
	// +optional
	MaxManifestFileSize string `json:"max_manifest_file_size,omitempty" toml:"max_manifest_file_size,omitempty"`
	// +optional
	CreateIfMissing *bool `json:"create_if_missing,omitempty" toml:"create_if_missing,omitempty"`
	// +optional
	MaxOpenFiles *int64 `json:"max_open_files,omitempty" toml:"max_open_files,omitempty"`
	// +optional
	EnableStatistics *bool `json:"enable_statistics,omitempty" toml:"enable_statistics,omitempty"`
	// +optional
	StatsDumpPeriod string `json:"stats_dump_period,omitempty" toml:"stats_dump_period,omitempty"`
	// +optional
	CompactionReadaheadSize string `json:"compaction_readahead_size,omitempty" toml:"compaction_readahead_size,omitempty"`
	// +optional
	InfoLogMaxSize string `json:"info_log_max_size,omitempty" toml:"info_log_max_size,omitempty"`
	// +optional
	FnfoLogRollTime string `json:"info_log_roll_time,omitempty" toml:"info_log_roll_time,omitempty"`
	// +optional
	InfoLogKeepLogFileNum *int64 `json:"info_log_keep_log_file_num,omitempty" toml:"info_log_keep_log_file_num,omitempty"`
	// +optional
	InfoLogDir string `json:"info_log_dir,omitempty" toml:"info_log_dir,omitempty"`
	// +optional
	MaxSubCompactions *int64 `json:"max_sub_compactions,omitempty" toml:"max_sub_compactions,omitempty"`
	// +optional
	WritableFileMaxBufferSize string `json:"writable_file_max_buffer_size,omitempty" toml:"writable_file_max_buffer_size,omitempty"`
	// +optional
	UseDirectIoForFlushAndCompaction *bool `json:"use_direct_io_for_flush_and_compaction,omitempty" toml:"use_direct_io_for_flush_and_compaction,omitempty"`
	// +optional
	EnablePipelinedWrite *bool `json:"enable_pipelined_write,omitempty" toml:"enable_pipelined_write,omitempty"`
	// +optional
	AllowConcurrentMemtableWrite *bool `json:"allow_concurrent_memtable_write,omitempty" toml:"allow_concurrent_memtable_write,omitempty"`
	// +optional
	BytesPerSync string `json:"bytes_per_sync,omitempty" toml:"bytes_per_sync,omitempty"`
	// +optional
	WalBytesPerSync string `json:"wal_bytes_per_sync,omitempty" toml:"wal_bytes_per_sync,omitempty"`
	// +optional
	Defaultcf *TiKVCfConfig `json:"defaultcf,omitempty" toml:"defaultcf,omitempty"`
}

// +k8s:openapi-gen=true
type TiKVSecurityConfig struct {
	// +optional
	CAPath string `json:"ca_path,omitempty" toml:"ca_path,omitempty"`
	// +optional
	CertPath string `json:"cert_path,omitempty" toml:"cert_path,omitempty"`
	// +optional
	KeyPath string `json:"key_path,omitempty" toml:"key_path,omitempty"`
	// +optional
	OverrideSslTarget string `json:"override_ssl_target,omitempty" toml:"override_ssl_target,omitempty"`
	// +optional
	CipherFile string `json:"cipher_file,omitempty" toml:"cipher_file,omitempty"`
}

// +k8s:openapi-gen=true
type TiKVImportConfig struct {
	// +optional
	ImportDir string `json:"import_dir,omitempty" toml:"import_dir,omitempty"`
	// +optional
	NumThreads *int64 `json:"num_threads,omitempty" toml:"num_threads,omitempty"`
	// +optional
	NumImportJobs *int64 `json:"num_import_jobs,omitempty" toml:"num_import_jobs,omitempty"`
	// +optional
	NumImportSstJobs *int64 `json:"num_import_sst_jobs,omitempty" toml:"num_import_sst_jobs,omitempty"`
	// +optional
	MaxPrepareDuration string `json:"max_prepare_duration,omitempty" toml:"max_prepare_duration,omitempty"`
	// +optional
	RegionSplitSize string `json:"region_split_size,omitempty" toml:"region_split_size,omitempty"`
	// +optional
	StreamChannelWindow *int64 `json:"stream_channel_window,omitempty" toml:"stream_channel_window,omitempty"`
	// +optional
	MaxOpenEngines *int64 `json:"max_open_engines,omitempty" toml:"max_open_engines,omitempty"`
	// +optional
	UploadSpeedLimit string `json:"upload_speed_limit,omitempty" toml:"upload_speed_limit,omitempty"`
}

// +k8s:openapi-gen=true
type TiKVGCConfig struct {
	// +optional
	BatchKeys *int64 `json:"	batch_keys,omitempty" toml:"	batch_keys,omitempty"`
	// +optional
	MaxWriteBytesPerSec string `json:"	max_write_bytes_per_sec,omitempty" toml:"	max_write_bytes_per_sec,omitempty"`
}

// TiKVDbConfig is the rocksdb config.
// +k8s:openapi-gen=true
type TiKVDbConfig struct {
	// +optional
	WalRecoveryMode *int64 `json:"wal-recovery-mode,omitempty" toml:"wal-recovery-mode,omitempty"`
	// +optional
	WalDir string `json:"wal-dir,omitempty" toml:"wal-dir,omitempty"`
	// +optional
	WalTTLSeconds *int64 `json:"wal-ttl-seconds,omitempty" toml:"wal-ttl-seconds,omitempty"`
	// +optional
	WalSizeLimit string `json:"wal-size-limit,omitempty" toml:"wal-size-limit,omitempty"`
	// +optional
	MaxTotalWalSize string `json:"max-total-wal-size,omitempty" toml:"max-total-wal-size,omitempty"`
	// +optional
	MaxBackgroundJobs *int64 `json:"max-background-jobs,omitempty" toml:"max-background-jobs,omitempty"`
	// +optional
	MaxManifestFileSize string `json:"max-manifest-file-size,omitempty" toml:"max-manifest-file-size,omitempty"`
	// +optional
	CreateIfMissing *bool `json:"create-if-missing,omitempty" toml:"create-if-missing,omitempty"`
	// +optional
	MaxOpenFiles *int64 `json:"max-open-files,omitempty" toml:"max-open-files,omitempty"`
	// +optional
	EnableStatistics *bool `json:"enable-statistics,omitempty" toml:"enable-statistics,omitempty"`
	// +optional
	StatsDumpPeriod string `json:"stats-dump-period,omitempty" toml:"stats-dump-period,omitempty"`
	// +optional
	CompactionReadaheadSize string `json:"compaction-readahead-size,omitempty" toml:"compaction-readahead-size,omitempty"`
	// +optional
	InfoLogMaxSize string `json:"info-log-max-size,omitempty" toml:"info-log-max-size,omitempty"`
	// +optional
	InfoLogRollTime string `json:"info-log-roll-time,omitempty" toml:"info-log-roll-time,omitempty"`
	// +optional
	InfoLogKeepLogFileNum *int64 `json:"info-log-keep-log-file-num,omitempty" toml:"info-log-keep-log-file-num,omitempty"`
	// +optional
	InfoLogDir string `json:"info-log-dir,omitempty" toml:"info-log-dir,omitempty"`
	// +optional
	RateBytesPerSec string `json:"rate-bytes-per-sec,omitempty" toml:"rate-bytes-per-sec,omitempty"`
	// +optional
	RateLimiterMode *int64 `json:"rate-limiter-mode,omitempty" toml:"rate-limiter-mode,omitempty"`
	// +optional
	AutoTuned *bool `json:"auto-tuned,omitempty" toml:"auto-tuned,omitempty"`
	// +optional
	BytesPerSync string `json:"bytes-per-sync,omitempty" toml:"bytes-per-sync,omitempty"`
	// +optional
	WalBytesPerSync string `json:"wal-bytes-per-sync,omitempty" toml:"wal-bytes-per-sync,omitempty"`
	// +optional
	MaxSubCompactions *int64 `json:"max-sub-compactions,omitempty" toml:"max-sub-compactions,omitempty"`
	// +optional
	WritableFileMaxBufferSize string `json:"writable-file-max-buffer-size,omitempty" toml:"writable-file-max-buffer-size,omitempty"`
	// +optional
	UseDirectIoForFlushAndCompaction *bool `json:"use-direct-io-for-flush-and-compaction,omitempty" toml:"use-direct-io-for-flush-and-compaction,omitempty"`
	// +optional
	EnablePipelinedWrite *bool `json:"enable-pipelined-write,omitempty" toml:"enable-pipelined-write,omitempty"`
	// +optional
	Defaultcf *TiKVCfConfig `json:"defaultcf,omitempty" toml:"defaultcf,omitempty"`
	// +optional
	Writecf *TiKVCfConfig `json:"writecf,omitempty" toml:"writecf,omitempty"`
	// +optional
	Lockcf *TiKVCfConfig `json:"lockcf,omitempty" toml:"lockcf,omitempty"`
	// +optional
	Raftcf *TiKVCfConfig `json:"raftcf,omitempty" toml:"raftcf,omitempty"`
	// +optional
	Titan *TiKVTitanDBConfig `json:"titan,omitempty" toml:"titan,omitempty"`
}

// TiKVCfConfig is the config of a cf
// +k8s:openapi-gen=true
type TiKVCfConfig struct {
	// +optional
	BlockSize string `json:"block-size,omitempty" toml:"block-size,omitempty"`
	// +optional
	BlockCacheSize string `json:"block-cache-size,omitempty" toml:"block-cache-size,omitempty"`
	// +optional
	DisableBlockCache *bool `json:"disable-block-cache,omitempty" toml:"disable-block-cache,omitempty"`
	// +optional
	CacheIndexAndFilterBlocks *bool `json:"cache-index-and-filter-blocks,omitempty" toml:"cache-index-and-filter-blocks,omitempty"`
	PinL0FilterAndIndexBlocks *bool `toml:"pin-l0-filter-and-index-blocks"`
	// +optional
	UseBloomFilter *bool `json:"use-bloom-filter,omitempty" toml:"use-bloom-filter,omitempty"`
	// +optional
	OptimizeFiltersForHits *bool `json:"optimize-filters-for-hits,omitempty" toml:"optimize-filters-for-hits,omitempty"`
	// +optional
	WholeKeyFiltering *bool `json:"whole-key-filtering,omitempty" toml:"whole-key-filtering,omitempty"`
	// +optional
	BloomFilterBitsPerKey *int64 `json:"bloom-filter-bits-per-key,omitempty" toml:"bloom-filter-bits-per-key,omitempty"`
	// +optional
	BlockBasedBloomFilter *bool `json:"block-based-bloom-filter,omitempty" toml:"block-based-bloom-filter,omitempty"`
	// +optional
	ReadAmpBytesPerBit *int64 `json:"read-amp-bytes-per-bit,omitempty" toml:"read-amp-bytes-per-bit,omitempty"`
	// +optional
	CompressionPerLevel []string `json:"compression-per-level,omitempty" toml:"compression-per-level,omitempty"`
	// +optional
	WriteBufferSize string `json:"write-buffer-size,omitempty" toml:"write-buffer-size,omitempty"`
	// +optional
	MaxWriteBufferNumber *int64 `json:"max-write-buffer-number,omitempty" toml:"max-write-buffer-number,omitempty"`
	// +optional
	MinWriteBufferNumberToMerge *int64 `json:"min-write-buffer-number-to-merge,omitempty" toml:"min-write-buffer-number-to-merge,omitempty"`
	// +optional
	MaxBytesForLevelBase string `json:"max-bytes-for-level-base,omitempty" toml:"max-bytes-for-level-base,omitempty"`
	// +optional
	TargetFileSizeBase             string `json:"target-file-size-base,omitempty" toml:"target-file-size-base,omitempty"`
	Level0FileNumCompactionTrigger *int64 `toml:"level0-file-num-compaction-trigger"`
	Level0SlowdownWritesTrigger    *int64 `toml:"level0-slowdown-writes-trigger"`
	Level0StopWritesTrigger        *int64 `toml:"level0-stop-writes-trigger"`
	// +optional
	MaxCompactionBytes string `json:"max-compaction-bytes,omitempty" toml:"max-compaction-bytes,omitempty"`
	// +optional
	CompactionPri *int64 `json:"compaction-pri,omitempty" toml:"compaction-pri,omitempty"`
	// +optional
	DynamicLevelBytes *bool `json:"dynamic-level-bytes,omitempty" toml:"dynamic-level-bytes,omitempty"`
	// +optional
	NumLevels *int64 `json:"num-levels,omitempty" toml:"num-levels,omitempty"`
	// +optional
	MaxBytesForLevelMultiplier *int64 `json:"max-bytes-for-level-multiplier,omitempty" toml:"max-bytes-for-level-multiplier,omitempty"`
	// +optional
	CompactionStyle *int64 `json:"compaction-style,omitempty" toml:"compaction-style,omitempty"`
	// +optional
	DisableAutoCompactions *bool `json:"disable-auto-compactions,omitempty" toml:"disable-auto-compactions,omitempty"`
	// +optional
	SoftPendingCompactionBytesLimit string `json:"soft-pending-compaction-bytes-limit,omitempty" toml:"soft-pending-compaction-bytes-limit,omitempty"`
	// +optional
	HardPendingCompactionBytesLimit string `json:"hard-pending-compaction-bytes-limit,omitempty" toml:"hard-pending-compaction-bytes-limit,omitempty"`
	// +optional
	ForceConsistencyChecks *bool `json:"force-consistency-checks,omitempty" toml:"force-consistency-checks,omitempty"`
	// +optional
	PropSizeIndexDistance *int64 `json:"prop-size-index-distance,omitempty" toml:"prop-size-index-distance,omitempty"`
	// +optional
	PropKeysIndexDistance *int64 `json:"prop-keys-index-distance,omitempty" toml:"prop-keys-index-distance,omitempty"`
	// +optional
	EnableDoublySkiplist *bool `json:"enable-doubly-skiplist,omitempty" toml:"enable-doubly-skiplist,omitempty"`
	// +optional
	Titan *TiKVTitanCfConfig `json:"titan,omitempty" toml:"titan,omitempty"`
}

// TiKVTitanCfConfig is the titian config.
// +k8s:openapi-gen=true
type TiKVTitanCfConfig struct {
	// +optional
	MinBlobSize string `json:"min-blob-size,omitempty" toml:"min-blob-size,omitempty"`
	// +optional
	BlobFileCompression string `json:"blob-file-compression,omitempty" toml:"blob-file-compression,omitempty"`
	// +optional
	BlobCacheSize string `json:"blob-cache-size,omitempty" toml:"blob-cache-size,omitempty"`
	// +optional
	MinGcBatchSize string `json:"min-gc-batch-size,omitempty" toml:"min-gc-batch-size,omitempty"`
	// +optional
	MaxGcBatchSize string `json:"max-gc-batch-size,omitempty" toml:"max-gc-batch-size,omitempty"`
	// +optional
	DiscardableRatio *float64 `json:"discardable-ratio,omitempty" toml:"discardable-ratio,omitempty"`
	// +optional
	SampleRatio *float64 `json:"sample-ratio,omitempty" toml:"sample-ratio,omitempty"`
	// +optional
	MergeSmallFileThreshold string `json:"merge-small-file-threshold,omitempty" toml:"merge-small-file-threshold,omitempty"`
	// +optional
	BlobRunMode string `json:"blob-run-mode,omitempty" toml:"blob-run-mode,omitempty"`
}

// TiKVTitanDBConfig is the config a titian db.
// +k8s:openapi-gen=true
type TiKVTitanDBConfig struct {
	// +optional
	Enabled *bool `json:"enabled,omitempty" toml:"enabled,omitempty"`
	// +optional
	Dirname string `json:"dirname,omitempty" toml:"dirname,omitempty"`
	// +optional
	DisableGc *bool `json:"disable-gc,omitempty" toml:"disable-gc,omitempty"`
	// +optional
	MaxBackgroundGc *int64 `json:"max-background-gc,omitempty" toml:"max-background-gc,omitempty"`
	// The value of this field will be truncated to seconds.
	// +optional
	PurgeObsoleteFilesPeriod string `json:"purge-obsolete-files-period,omitempty" toml:"purge-obsolete-files-period,omitempty"`
}

// TiKVStorageConfig is the config of storage
// +k8s:openapi-gen=true
type TiKVStorageConfig struct {
	// +optional
	DataDir string `json:"data-dir,omitempty" toml:"data-dir,omitempty"`
	// +optional
	MaxKeySize *int64 `json:"max-key-size,omitempty" toml:"max-key-size,omitempty"`
	// +optional
	SchedulerNotifyCapacity *int64 `json:"scheduler-notify-capacity,omitempty" toml:"scheduler-notify-capacity,omitempty"`
	// +optional
	SchedulerConcurrency *int64 `json:"scheduler-concurrency,omitempty" toml:"scheduler-concurrency,omitempty"`
	// +optional
	SchedulerWorkerPoolSize *int64 `json:"scheduler-worker-pool-size,omitempty" toml:"scheduler-worker-pool-size,omitempty"`
	// +optional
	SchedulerPendingWriteThreshold string `json:"scheduler-pending-write-threshold,omitempty" toml:"scheduler-pending-write-threshold,omitempty"`
	// +optional
	BlockCache *TiKVBlockCacheConfig `json:"block-cache,omitempty" toml:"block-cache,omitempty"`
}

// TiKVBlockCacheConfig is the config of a block cache
// +k8s:openapi-gen=true
type TiKVBlockCacheConfig struct {
	// +optional
	Shared *bool `json:"shared,omitempty" toml:"shared,omitempty"`
	// +optional
	Capacity string `json:"capacity,omitempty" toml:"capacity,omitempty"`
	// +optional
	NumShardBits *int64 `json:"num-shard-bits,omitempty" toml:"num-shard-bits,omitempty"`
	// +optional
	StrictCapacityLimit *bool `json:"strict-capacity-limit,omitempty" toml:"strict-capacity-limit,omitempty"`
	// +optional
	HighPriPoolRatio *float64 `json:"high-pri-pool-ratio,omitempty" toml:"high-pri-pool-ratio,omitempty"`
	// +optional
	MemoryAllocator string `json:"memory-allocator,omitempty" toml:"memory-allocator,omitempty"`
}

// TiKVServerConfig is the configuration of TiKV server.
// +k8s:openapi-gen=true
type TiKVServerConfig struct {
	// +optional
	GrpcCompressionType string `json:"grpc-compression-type,omitempty" toml:"grpc-compression-type,omitempty"`
	// +optional
	GrpcConcurrency *int64 `json:"grpc-concurrency,omitempty" toml:"grpc-concurrency,omitempty"`
	// +optional
	GrpcConcurrentStream *int64 `json:"grpc-concurrent-stream,omitempty" toml:"grpc-concurrent-stream,omitempty"`
	// +optional
	GrpcRaftConnNum *int64 `json:"grpc-raft-conn-num,omitempty" toml:"grpc-raft-conn-num,omitempty"`
	// +optional
	GrpcStreamInitialWindowSize string `json:"grpc-stream-initial-window-size,omitempty" toml:"grpc-stream-initial-window-size,omitempty"`
	// +optional
	GrpcKeepaliveTime string `json:"grpc-keepalive-time,omitempty" toml:"grpc-keepalive-time,omitempty"`
	// +optional
	GrpcKeepaliveTimeout string `json:"grpc-keepalive-timeout,omitempty" toml:"grpc-keepalive-timeout,omitempty"`
	// +optional
	ConcurrentSendSnapLimit *int64 `json:"concurrent-send-snap-limit,omitempty" toml:"concurrent-send-snap-limit,omitempty"`
	// +optional
	ConcurrentRecvSnapLimit *int64 `json:"concurrent-recv-snap-limit,omitempty" toml:"concurrent-recv-snap-limit,omitempty"`
	// +optional
	EndPointRecursionLimit *int64 `json:"end-point-recursion-limit,omitempty" toml:"end-point-recursion-limit,omitempty"`
	// +optional
	EndPointStreamChannelSize *int64 `json:"end-point-stream-channel-size,omitempty" toml:"end-point-stream-channel-size,omitempty"`
	// +optional
	EndPointBatchRowLimit *int64 `json:"end-point-batch-row-limit,omitempty" toml:"end-point-batch-row-limit,omitempty"`
	// +optional
	EndPointStreamBatchRowLimit *int64 `json:"end-point-stream-batch-row-limit,omitempty" toml:"end-point-stream-batch-row-limit,omitempty"`
	// +optional
	EndPointEnableBatchIfPossible *bool `json:"end-point-enable-batch-if-possible,omitempty" toml:"end-point-enable-batch-if-possible,omitempty"`
	// +optional
	EndPointRequestMaxHandleDuration string `json:"end-point-request-max-handle-duration,omitempty" toml:"end-point-request-max-handle-duration,omitempty"`
	// +optional
	SnapMaxWriteBytesPerSec string `json:"snap-max-write-bytes-per-sec,omitempty" toml:"snap-max-write-bytes-per-sec,omitempty"`
	// +optional
	SnapMaxTotalSize string `json:"snap-max-total-size,omitempty" toml:"snap-max-total-size,omitempty"`
	// +optional
	StatsConcurrency *int64 `json:"stats-concurrency,omitempty" toml:"stats-concurrency,omitempty"`
	// +optional
	HeavyLoadThreshold *int64 `json:"heavy-load-threshold,omitempty" toml:"heavy-load-threshold,omitempty"`
	// +optional
	HeavyLoadWaitDuration string `json:"heavy-load-wait-duration,omitempty" toml:"heavy-load-wait-duration,omitempty"`
	// +optional
	Labels map[string]string `json:"labels,omitempty" toml:"labels,omitempty"`
}

// TiKVRaftstoreConfig is the configuration of TiKV raftstore component.
// +k8s:openapi-gen=true
type TiKVRaftstoreConfig struct {
	// true for high reliability, prevent data loss when power failure.
	// +optional
	SyncLog *bool `json:"sync-log,omitempty" toml:"sync-log,omitempty"`

	// raft-base-tick-interval is a base tick interval (ms).
	// +optional
	RaftBaseTickInterval string `json:"raft-base-tick-interval,omitempty" toml:"raft-base-tick-interval,omitempty"`
	// +optional
	RaftHeartbeatTicks *int64 `json:"raft-heartbeat-ticks,omitempty" toml:"raft-heartbeat-ticks,omitempty"`
	// +optional
	RaftElectionTimeoutTicks *int64 `json:"raft-election-timeout-ticks,omitempty" toml:"raft-election-timeout-ticks,omitempty"`
	// When the entry exceed the max size, reject to propose it.
	// +optional
	RaftEntryMaxSize string `json:"raft-entry-max-size,omitempty" toml:"raft-entry-max-size,omitempty"`

	// Interval to gc unnecessary raft log (ms).
	// +optional
	RaftLogGCTickInterval string `json:"raft-log-gc-tick-interval,omitempty" toml:"raft-log-gc-tick-interval,omitempty"`
	// A threshold to gc stale raft log, must >= 1.
	// +optional
	RaftLogGCThreshold *int64 `json:"raft-log-gc-threshold,omitempty" toml:"raft-log-gc-threshold,omitempty"`
	// When entry count exceed this value, gc will be forced trigger.
	// +optional
	RaftLogGCCountLimit *int64 `json:"raft-log-gc-count-limit,omitempty" toml:"raft-log-gc-count-limit,omitempty"`
	// When the approximate size of raft log entries exceed this value
	// gc will be forced trigger.
	// +optional
	RaftLogGCSizeLimit string `json:"raft-log-gc-size-limit,omitempty" toml:"raft-log-gc-size-limit,omitempty"`
	// When a peer is not responding for this time, leader will not keep entry cache for it.
	// +optional
	RaftEntryCacheLifeTime string `json:"raft-entry-cache-life-time,omitempty" toml:"raft-entry-cache-life-time,omitempty"`
	// When a peer is newly added, reject transferring leader to the peer for a while.
	// +optional
	RaftRejectTransferLeaderDuration string `json:"raft-reject-transfer-leader-duration,omitempty" toml:"raft-reject-transfer-leader-duration,omitempty"`

	// Interval (ms) to check region whether need to be split or not.
	// +optional
	SplitRegionCheckTickInterval string `json:"split-region-check-tick-interval,omitempty" toml:"split-region-check-tick-interval,omitempty"`
	/// When size change of region exceed the diff since last check, it
	/// will be checked again whether it should be split.
	// +optional
	RegionSplitCheckDiff string `json:"region-split-check-diff,omitempty" toml:"region-split-check-diff,omitempty"`
	/// Interval (ms) to check whether start compaction for a region.
	// +optional
	RegionCompactCheckInterval string `json:"region-compact-check-interval,omitempty" toml:"region-compact-check-interval,omitempty"`
	// delay time before deleting a stale peer
	// +optional
	CleanStalePeerDelay string `json:"clean-stale-peer-delay,omitempty" toml:"clean-stale-peer-delay,omitempty"`
	/// Number of regions for each time checking.
	// +optional
	RegionCompactCheckStep *int64 `json:"region-compact-check-step,omitempty" toml:"region-compact-check-step,omitempty"`
	/// Minimum number of tombstones to trigger manual compaction.
	// +optional
	RegionCompactMinTombstones *int64 `json:"region-compact-min-tombstones,omitempty" toml:"region-compact-min-tombstones,omitempty"`
	/// Minimum percentage of tombstones to trigger manual compaction.
	/// Should between 1 and 100.
	// +optional
	RegionCompactTombstonesPercent *int64 `json:"region-compact-tombstones-percent,omitempty" toml:"region-compact-tombstones-percent,omitempty"`
	// +optional
	PdHeartbeatTickInterval string `json:"pd-heartbeat-tick-interval,omitempty" toml:"pd-heartbeat-tick-interval,omitempty"`
	// +optional
	PdStoreHeartbeatTickInterval string `json:"pd-store-heartbeat-tick-interval,omitempty" toml:"pd-store-heartbeat-tick-interval,omitempty"`
	// +optional
	SnapMgrGCTickInterval string `json:"snap-mgr-gc-tick-interval,omitempty" toml:"snap-mgr-gc-tick-interval,omitempty"`
	// +optional
	SnapGCTimeout string `json:"snap-gc-timeout,omitempty" toml:"snap-gc-timeout,omitempty"`
	// +optional
	LockCfCompactInterval string `json:"lock-cf-compact-interval,omitempty" toml:"lock-cf-compact-interval,omitempty"`
	// +optional
	LockCfCompactBytesThreshold string `json:"lock-cf-compact-bytes-threshold,omitempty" toml:"lock-cf-compact-bytes-threshold,omitempty"`

	// +optional
	NotifyCapacity *int64 `json:"notify-capacity,omitempty" toml:"notify-capacity,omitempty"`
	// +optional
	MessagesPerTick *int64 `json:"messages-per-tick,omitempty" toml:"messages-per-tick,omitempty"`

	/// When a peer is not active for max-peer-down-duration
	/// the peer is considered to be down and is reported to PD.
	// +optional
	MaxPeerDownDuration string `json:"max-peer-down-duration,omitempty" toml:"max-peer-down-duration,omitempty"`

	/// If the leader of a peer is missing for longer than max-leader-missing-duration
	/// the peer would ask pd to confirm whether it is valid in any region.
	/// If the peer is stale and is not valid in any region, it will destroy itself.
	// +optional
	MaxLeaderMissingDuration string `json:"max-leader-missing-duration,omitempty" toml:"max-leader-missing-duration,omitempty"`
	/// Similar to the max-leader-missing-duration, instead it will log warnings and
	/// try to alert monitoring systems, if there is any.
	// +optional
	AbnormalLeaderMissingDuration string `json:"abnormal-leader-missing-duration,omitempty" toml:"abnormal-leader-missing-duration,omitempty"`
	// +optional
	PeerStaleStateCheckInterval string `json:"peer-stale-state-check-interval,omitempty" toml:"peer-stale-state-check-interval,omitempty"`

	// +optional
	LeaderTransferMaxLogLag *int64 `json:"leader-transfer-max-log-lag,omitempty" toml:"leader-transfer-max-log-lag,omitempty"`

	// +optional
	SnapApplyBatchSize string `json:"snap-apply-batch-size,omitempty" toml:"snap-apply-batch-size,omitempty"`

	// Interval (ms) to check region whether the data is consistent.
	// +optional
	ConsistencyCheckInterval string `json:"consistency-check-interval,omitempty" toml:"consistency-check-interval,omitempty"`

	// +optional
	ReportRegionFlowInterval string `json:"report-region-flow-interval,omitempty" toml:"report-region-flow-interval,omitempty"`

	// The lease provided by a successfully proposed and applied entry.
	// +optional
	RaftStoreMaxLeaderLease string `json:"raft-store-max-leader-lease,omitempty" toml:"raft-store-max-leader-lease,omitempty"`

	// Right region derive origin region id when split.
	// +optional
	RightDeriveWhenSplit *bool `json:"right-derive-when-split,omitempty" toml:"right-derive-when-split,omitempty"`

	// +optional
	AllowRemoveLeader *bool `json:"allow-remove-leader,omitempty" toml:"allow-remove-leader,omitempty"`

	/// Max log gap allowed to propose merge.
	// +optional
	MergeMaxLogGap *int64 `json:"merge-max-log-gap,omitempty" toml:"merge-max-log-gap,omitempty"`
	/// Interval to re-propose merge.
	// +optional
	MergeCheckTickInterval string `json:"merge-check-tick-interval,omitempty" toml:"merge-check-tick-interval,omitempty"`

	// +optional
	UseDeleteRange *bool `json:"use-delete-range,omitempty" toml:"use-delete-range,omitempty"`

	// +optional
	CleanupImportSstInterval string `json:"cleanup-import-sst-interval,omitempty" toml:"cleanup-import-sst-interval,omitempty"`

	// +optional
	ApplyMaxBatchSize *int64 `json:"apply-max-batch-size,omitempty" toml:"apply-max-batch-size,omitempty"`
	// +optional
	ApplyPoolSize *int64 `json:"apply-pool-size,omitempty" toml:"apply-pool-size,omitempty"`

	// +optional
	StoreMaxBatchSize *int64 `json:"store-max-batch-size,omitempty" toml:"store-max-batch-size,omitempty"`
	// +optional
	StorePoolSize *int64 `json:"store-pool-size,omitempty" toml:"store-pool-size,omitempty"`
	// +optional
	HibernateRegions *bool `json:"hibernate-regions,omitempty" toml:"hibernate-regions,omitempty"`
}

// TiKVCoprocessorConfig is the configuration of TiKV Coprocessor component.
// +k8s:openapi-gen=true
type TiKVCoprocessorConfig struct {
	// When it is set to `true`, TiKV will try to split a Region with table prefix if that Region
	// crosses tables.
	// It is recommended to turn off this option if there will be a large number of tables created.
	// optional
	SplitRegionOnTable *bool `json:"split-region-on-table,omitempty" toml:"split-region-on-table,omitempty"`

	// One split check produces several split keys in batch. This config limits the number of produced
	// split keys in one batch.
	// optional
	BatchSplitLimit *int64 `json:"batch-split-limit,omitempty" toml:"batch-split-limit,omitempty"`

	// When Region [a,e) size exceeds `region_max_size`, it will be split into several Regions [a,b),
	// [b,c), [c,d), [d,e) and the size of [a,b), [b,c), [c,d) will be `region_split_size` (or a
	// little larger). See also: region-split-size
	// optional
	RegionMaxSize string `json:"region-max-size,omitempty" toml:"region-max-size,omitempty"`

	// When Region [a,e) size exceeds `region_max_size`, it will be split into several Regions [a,b),
	// [b,c), [c,d), [d,e) and the size of [a,b), [b,c), [c,d) will be `region_split_size` (or a
	// little larger). See also: region-max-size
	// optional
	RegionSplitSize string `json:"region-split-size,omitempty" toml:"region-split-size,omitempty"`

	// When the number of keys in Region [a,e) exceeds the `region_max_keys`, it will be split into
	// several Regions [a,b), [b,c), [c,d), [d,e) and the number of keys in [a,b), [b,c), [c,d) will be
	// `region_split_keys`. See also: region-split-keys
	// optional
	RegionMaxKeys *int64 `json:"region-max-keys,omitempty" toml:"region-max-keys,omitempty"`

	// When the number of keys in Region [a,e) exceeds the `region_max_keys`, it will be split into
	// several Regions [a,b), [b,c), [c,d), [d,e) and the number of keys in [a,b), [b,c), [c,d) will be
	// `region_split_keys`. See also: region-max-keys
	// optional
	RegionSplitKeys *int64 `json:"region-split-keys,omitempty" toml:"region-split-keys,omitempty"`
}
