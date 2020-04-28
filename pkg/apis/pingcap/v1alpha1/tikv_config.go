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

// Port from TiKV v3.0.6

// TiKVConfig is the configuration of TiKV.
// +k8s:openapi-gen=true
type TiKVConfig struct {
	// Optional: Defaults to info
	// +optional
	LogLevel string `json:"log-level,omitempty" toml:"log-level,omitempty"`
	// +optional
	LogFile string `json:"log-file,omitempty" toml:"log-file,omitempty"`
	// +optional
	SlowLogFile string `json:"slow-log-file,omitempty" toml:"slow-log-file,omitempty"`
	// +optional
	SlowLogThreshold string `json:"slow-log-threshold,omitempty" toml:"slow-log-threshold,omitempty"`
	// Optional: Defaults to 24h
	// +optional
	LogRotationTimespan string `json:"log-rotation-timespan,omitempty" toml:"log-rotation-timespan,omitempty"`
	// +optional
	LogRotationSize string `json:"log-rotation-size,omitempty" toml:"log-rotation-size,omitempty"`
	// +optional
	RefreshConfigInterval string `json:"refresh-config-interval,omitempty" toml:"refresh-config-interval,omitempty"`
	// +optional
	PanicWhenUnexpectedKeyOrData *bool `json:"panic-when-unexpected-key-or-data,omitempty" toml:"panic-when-unexpected-key-or-data,omitempty"`
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
	PD *TiKVPDConfig `json:"pd,omitempty" toml:"pd,omitempty"`
	// +optional
	Security *TiKVSecurityConfig `json:"security,omitempty" toml:"security,omitempty"`
	// +optional
	Encryption *TiKVEncryptionConfig `json:"encryption,omitempty" toml:"encryption,omitempty"`
	// +optional
	TiKVPessimisticTxn *TiKVPessimisticTxn `json:"pessimistic-txn,omitempty" toml:"pessimistic-txn,omitempty"`
}

// +k8s:openapi-gen=true
type TiKVReadPoolConfig struct {
	// +optional
	Unified *TiKVUnifiedReadPoolConfig `json:"unified,omitempty" toml:"unified,omitempty"`
	// +optional
	Coprocessor *TiKVCoprocessorReadPoolConfig `json:"coprocessor,omitempty" toml:"coprocessor,omitempty"`
	// +optional
	Storage *TiKVStorageReadPoolConfig `json:"storage,omitempty" toml:"storage,omitempty"`
}

// +k8s:openapi-gen=true
type TiKVUnifiedReadPoolConfig struct {
	// +optional
	MinThreadCount int32 `json:"min-thread-count,omitempty" toml:"min-thread-count,omitempty"`
	// +optional
	MaxThreadCount int32 `json:"max-thread-count,omitempty" toml:"max-thread-count,omitempty"`
	// +optional
	StackSize string `json:"stack-size,omitempty" toml:"stack-size,omitempty"`
	// +optional
	MaxTasksPerWorker int32 `json:"max-tasks-per-worker,omitempty" toml:"max-tasks-per-worker,omitempty"`
}

// +k8s:openapi-gen=true
type TiKVStorageReadPoolConfig struct {
	// Optional: Defaults to 4
	// +optional
	HighConcurrency *int64 `json:"high-concurrency,omitempty" toml:"high-concurrency,omitempty"`
	// Optional: Defaults to 4
	// +optional
	NormalConcurrency *int64 `json:"normal-concurrency,omitempty" toml:"normal-concurrency,omitempty"`
	// Optional: Defaults to 4
	// +optional
	LowConcurrency *int64 `json:"low-concurrency,omitempty" toml:"low-concurrency,omitempty"`
	// Optional: Defaults to 2000
	// +optional
	MaxTasksPerWorkerHigh *int64 `json:"max-tasks-per-worker-high,omitempty" toml:"max-tasks-per-worker-high,omitempty"`
	// Optional: Defaults to 2000
	// +optional
	MaxTasksPerWorkerNormal *int64 `json:"max-tasks-per-worker-normal,omitempty" toml:"max-tasks-per-worker-normal,omitempty"`
	// Optional: Defaults to 2000
	// +optional
	MaxTasksPerWorkerLow *int64 `json:"max-tasks-per-worker-low,omitempty" toml:"max-tasks-per-worker-low,omitempty"`
	// Optional: Defaults to 10MB
	// +optional
	StackSize string `json:"stack-size,omitempty" toml:"stack-size,omitempty"`
}

// +k8s:openapi-gen=true
type TiKVCoprocessorReadPoolConfig struct {
	// Optional: Defaults to 8
	// +optional
	HighConcurrency *int64 `json:"high-concurrency,omitempty" toml:"high-concurrency,omitempty"`
	// Optional: Defaults to 8
	// +optional
	NormalConcurrency *int64 `json:"normal-concurrency,omitempty" toml:"normal-concurrency,omitempty"`
	// Optional: Defaults to 8
	// +optional
	LowConcurrency *int64 `json:"low-concurrency,omitempty" toml:"low-concurrency,omitempty"`
	// Optional: Defaults to 2000
	// +optional
	MaxTasksPerWorkerHigh *int64 `json:"max-tasks-per-worker-high,omitempty" toml:"max-tasks-per-worker-high,omitempty"`
	// Optional: Defaults to 2000
	// +optional
	MaxTasksPerWorkerNormal *int64 `json:"max-tasks-per-worker-normal,omitempty" toml:"max-tasks-per-worker-normal,omitempty"`
	// Optional: Defaults to 2000
	// +optional
	MaxTasksPerWorkerLow *int64 `json:"max-tasks-per-worker-low,omitempty" toml:"max-tasks-per-worker-low,omitempty"`
	// Optional: Defaults to 10MB
	// +optional
	StackSize string `json:"stack-size,omitempty" toml:"stack-size,omitempty"`
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
	// Optional: Defaults to 300ms
	// +optional
	RetryInterval string `json:"retry-interval,omitempty" toml:"retry-interval,omitempty"`
	// The maximum number of times to retry a PD connection initialization.
	//
	// Default is isize::MAX, represented by -1.
	// Optional: Defaults to -1
	// +optional
	RetryMaxCount *int64 `json:"retry-max-count,omitempty" toml:"retry-max-count,omitempty"`
	// If the client observes the same error message on retry, it can repeat the message only
	// every `n` times.
	//
	// Default is 10. Set to 1 to disable this feature.
	// Optional: Defaults to 10
	// +optional
	RetryLogEvery *int64 `json:"retry-log-every,omitempty" toml:"retry-log-every,omitempty"`
}

// +k8s:openapi-gen=true
type TiKVRaftDBConfig struct {
	// +optional
	WalRecoveryMode string `json:"wal-recovery-mode,omitempty" toml:"wal-recovery-mode,omitempty"`
	// +optional
	WalDir string `json:"wal-dir,omitempty" toml:"wal-dir,omitempty"`
	// +optional
	WalTtlSeconds *int64 `json:"wal-ttl-seconds,omitempty" toml:"wal-ttl-seconds,omitempty"`
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
	FnfoLogRollTime string `json:"info-log-roll-time,omitempty" toml:"info-log-roll-time,omitempty"`
	// +optional
	InfoLogKeepLogFileNum *int64 `json:"info-log-keep-log-file-num,omitempty" toml:"info-log-keep-log-file-num,omitempty"`
	// +optional
	InfoLogDir string `json:"info-log-dir,omitempty" toml:"info-log-dir,omitempty"`
	// +optional
	MaxSubCompactions *int64 `json:"max-sub-compactions,omitempty" toml:"max-sub-compactions,omitempty"`
	// +optional
	WritableFileMaxBufferSize string `json:"writable-file-max-buffer-size,omitempty" toml:"writable-file-max-buffer-size,omitempty"`
	// +optional
	UseDirectIoForFlushAndCompaction *bool `json:"use-direct-io-for-flush-and-compaction,omitempty" toml:"use-direct-io-for-flush-and-compaction,omitempty"`
	// +optional
	EnablePipelinedWrite *bool `json:"enable-pipelined-write,omitempty" toml:"enable-pipelined-write,omitempty"`
	// +optional
	AllowConcurrentMemtableWrite *bool `json:"allow-concurrent-memtable-write,omitempty" toml:"allow-concurrent-memtable-write,omitempty"`
	// +optional
	BytesPerSync string `json:"bytes-per-sync,omitempty" toml:"bytes-per-sync,omitempty"`
	// +optional
	WalBytesPerSync string `json:"wal-bytes-per-sync,omitempty" toml:"wal-bytes-per-sync,omitempty"`
	// +optional
	Defaultcf *TiKVCfConfig `json:"defaultcf,omitempty" toml:"defaultcf,omitempty"`
}

// +k8s:openapi-gen=true
type TiKVSecurityConfig struct {
	// +optional
	CAPath string `json:"ca-path,omitempty" toml:"ca-path,omitempty"`
	// +optional
	CertPath string `json:"cert-path,omitempty" toml:"cert-path,omitempty"`
	// +optional
	KeyPath string `json:"key-path,omitempty" toml:"key-path,omitempty"`
	// CertAllowedCN is the Common Name that allowed
	// +optional
	// +k8s:openapi-gen=false
	CertAllowedCN []string `json:"cert-allowed-cn,omitempty" toml:"cert-allowed-cn,omitempty"`
	// +optional
	OverrideSslTarget string `json:"override-ssl-target,omitempty" toml:"override-ssl-target,omitempty"`
	// +optional
	CipherFile string `json:"cipher-file,omitempty" toml:"cipher-file,omitempty"`
}

// +k8s:openapi-gen=true
type TiKVImportConfig struct {
	// +optional
	ImportDir string `json:"import-dir,omitempty" toml:"import-dir,omitempty"`
	// +optional
	NumThreads *int64 `json:"num-threads,omitempty" toml:"num-threads,omitempty"`
	// +optional
	NumImportJobs *int64 `json:"num-import-jobs,omitempty" toml:"num-import-jobs,omitempty"`
	// +optional
	NumImportSstJobs *int64 `json:"num-import-sst-jobs,omitempty" toml:"num-import-sst-jobs,omitempty"`
	// +optional
	MaxPrepareDuration string `json:"max-prepare-duration,omitempty" toml:"max-prepare-duration,omitempty"`
	// +optional
	RegionSplitSize string `json:"region-split-size,omitempty" toml:"region-split-size,omitempty"`
	// +optional
	StreamChannelWindow *int64 `json:"stream-channel-window,omitempty" toml:"stream-channel-window,omitempty"`
	// +optional
	MaxOpenEngines *int64 `json:"max-open-engines,omitempty" toml:"max-open-engines,omitempty"`
	// +optional
	UploadSpeedLimit string `json:"upload-speed-limit,omitempty" toml:"upload-speed-limit,omitempty"`
}

// +k8s:openapi-gen=true
type TiKVGCConfig struct {
	// +optional
	// Optional: Defaults to 512
	BatchKeys *int64 `json:"	batch-keys,omitempty" toml:"	batch-keys,omitempty"`
	// +optional
	MaxWriteBytesPerSec string `json:"	max-write-bytes-per-sec,omitempty" toml:"	max-write-bytes-per-sec,omitempty"`
}

// TiKVDbConfig is the rocksdb config.
// +k8s:openapi-gen=true
type TiKVDbConfig struct {
	// +optional
	// Optional: Defaults to 2
	WalRecoveryMode *int64 `json:"wal-recovery-mode,omitempty" toml:"wal-recovery-mode,omitempty"`
	// +optional
	WalTTLSeconds *int64 `json:"wal-ttl-seconds,omitempty" toml:"wal-ttl-seconds,omitempty"`
	// +optional
	WalSizeLimit string `json:"wal-size-limit,omitempty" toml:"wal-size-limit,omitempty"`
	// +optional
	// Optional: Defaults to 4GB
	MaxTotalWalSize string `json:"max-total-wal-size,omitempty" toml:"max-total-wal-size,omitempty"`
	// +optional
	// Optional: Defaults to 8
	MaxBackgroundJobs *int64 `json:"max-background-jobs,omitempty" toml:"max-background-jobs,omitempty"`
	// +optional
	// Optional: Defaults to 128MB
	MaxManifestFileSize string `json:"max-manifest-file-size,omitempty" toml:"max-manifest-file-size,omitempty"`
	// +optional
	// Optional: Defaults to true
	CreateIfMissing *bool `json:"create-if-missing,omitempty" toml:"create-if-missing,omitempty"`
	// +optional
	// Optional: Defaults to 40960
	MaxOpenFiles *int64 `json:"max-open-files,omitempty" toml:"max-open-files,omitempty"`
	// +optional
	// Optional: Defaults to true
	EnableStatistics *bool `json:"enable-statistics,omitempty" toml:"enable-statistics,omitempty"`
	// +optional
	// Optional: Defaults to 10m
	StatsDumpPeriod string `json:"stats-dump-period,omitempty" toml:"stats-dump-period,omitempty"`
	// Optional: Defaults to 0
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
	// Optional: Defaults to 3
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
	// +optional
	PinL0FilterAndIndexBlocks *bool `json:"pin-l0-filter-and-index-blocks,omitempty" toml:"pin-l0-filter-and-index-blocks,omitempty"`
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
	TargetFileSizeBase string `json:"target-file-size-base,omitempty" toml:"target-file-size-base,omitempty"`
	// +optional
	Level0FileNumCompactionTrigger *int64 `json:"level0-file-num-compaction-trigger,omitempty" toml:"level0-file-num-compaction-trigger,omitempty"`
	// +optional
	Level0SlowdownWritesTrigger *int64 `json:"level0-slowdown-writes-trigger,omitempty" toml:"level0-slowdown-writes-trigger,omitempty"`
	// +optional
	Level0StopWritesTrigger *int64 `json:"level0-stop-writes-trigger,omitempty" toml:"level0-stop-writes-trigger,omitempty"`
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
	MaxKeySize *int64 `json:"max-key-size,omitempty" toml:"max-key-size,omitempty"`
	// +optional
	SchedulerNotifyCapacity *int64 `json:"scheduler-notify-capacity,omitempty" toml:"scheduler-notify-capacity,omitempty"`
	// +optional
	// Optional: Defaults to 2048000
	SchedulerConcurrency *int64 `json:"scheduler-concurrency,omitempty" toml:"scheduler-concurrency,omitempty"`
	// +optional
	// Optional: Defaults to 4
	SchedulerWorkerPoolSize *int64 `json:"scheduler-worker-pool-size,omitempty" toml:"scheduler-worker-pool-size,omitempty"`
	// +optional
	// Optional: Defaults to 100MB
	SchedulerPendingWriteThreshold string `json:"scheduler-pending-write-threshold,omitempty" toml:"scheduler-pending-write-threshold,omitempty"`
	// +optional
	BlockCache *TiKVBlockCacheConfig `json:"block-cache,omitempty" toml:"block-cache,omitempty"`
}

// TiKVBlockCacheConfig is the config of a block cache
// +k8s:openapi-gen=true
type TiKVBlockCacheConfig struct {
	// Optional: Defaults to true
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
	// Optional: Defaults to 1
	// +optional
	StatusThreadPoolSize string `json:"status-thread-pool-size,omitempty" toml:"status-thread-pool-size,omitempty"`
	// Optional: Defaults to none
	// +optional
	GrpcCompressionType string `json:"grpc-compression-type,omitempty" toml:"grpc-compression-type,omitempty"`
	// Optional: Defaults to 4
	// +optional
	GrpcConcurrency *uint `json:"grpc-concurrency,omitempty" toml:"grpc-concurrency,omitempty"`
	// Optional: Defaults to 1024
	// +optional
	GrpcConcurrentStream *uint `json:"grpc-concurrent-stream,omitempty" toml:"grpc-concurrent-stream,omitempty"`
	// Optional: Defaults to 32G
	// +optional
	GrpcMemoryQuota *string `json:"grpc-memory-pool-quota,omitempty" toml:"grpc-memory-pool-quota,omitempty"`
	// Optional: Defaults to 10
	// +optional
	GrpcRaftConnNum *uint `json:"grpc-raft-conn-num,omitempty" toml:"grpc-raft-conn-num,omitempty"`
	// Optional: Defaults to 2MB
	// +optional
	GrpcStreamInitialWindowSize string `json:"grpc-stream-initial-window-size,omitempty" toml:"grpc-stream-initial-window-size,omitempty"`
	// Optional: Defaults to 10s
	// +optional
	GrpcKeepaliveTime string `json:"grpc-keepalive-time,omitempty" toml:"grpc-keepalive-time,omitempty"`
	// Optional: Defaults to 3s
	// +optional
	GrpcKeepaliveTimeout string `json:"grpc-keepalive-timeout,omitempty" toml:"grpc-keepalive-timeout,omitempty"`
	// Optional: Defaults to 32
	// +optional
	ConcurrentSendSnapLimit *uint `json:"concurrent-send-snap-limit,omitempty" toml:"concurrent-send-snap-limit,omitempty"`
	// Optional: Defaults to 32
	// +optional
	ConcurrentRecvSnapLimit *uint `json:"concurrent-recv-snap-limit,omitempty" toml:"concurrent-recv-snap-limit,omitempty"`
	// Optional: Defaults to 1000
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
	// Optional: Defaults to 100MB
	// +optional
	SnapMaxWriteBytesPerSec string `json:"snap-max-write-bytes-per-sec,omitempty" toml:"snap-max-write-bytes-per-sec,omitempty"`
	// +optional
	SnapMaxTotalSize string `json:"snap-max-total-size,omitempty" toml:"snap-max-total-size,omitempty"`
	// +optional
	StatsConcurrency *uint `json:"stats-concurrency,omitempty" toml:"stats-concurrency,omitempty"`
	// +optional
	HeavyLoadThreshold *uint `json:"heavy-load-threshold,omitempty" toml:"heavy-load-threshold,omitempty"`
	// Optional: Defaults to 60s
	// +optional
	HeavyLoadWaitDuration string `json:"heavy-load-wait-duration,omitempty" toml:"heavy-load-wait-duration,omitempty"`
	// +optional
	Labels map[string]string `json:"labels,omitempty" toml:"labels,omitempty"`
	// +optional
	EnableRequestBatch bool `json:"enable-request-batch,omitempty" toml:"enable-request-batch,omitempty"`
	// +optional
	RequestBatchEnableCrossCommand bool `json:"request-batch-enable-cross-command,omitempty" toml:"request-batch-enable-cross-command,omitempty"`
	// +optional
	RequestBatchWaitDuration string `json:"request-batch-wait-duration,omitempty" toml:"request-batch-wait-duration,omitempty"`
}

// TiKVRaftstoreConfig is the configuration of TiKV raftstore component.
// +k8s:openapi-gen=true
type TiKVRaftstoreConfig struct {
	// true for high reliability, prevent data loss when power failure.
	// Optional: Defaults to true
	// +optional
	SyncLog *bool `json:"sync-log,omitempty" toml:"sync-log,omitempty"`

	// Optional: Defaults to true
	// +optional
	Prevote *bool `json:"prevote,omitempty" toml:"prevote,omitempty"`
	// raft-base-tick-interval is a base tick interval (ms).
	// +optional
	RaftBaseTickInterval string `json:"raft-base-tick-interval,omitempty" toml:"raft-base-tick-interval,omitempty"`
	// +optional
	RaftHeartbeatTicks *int64 `json:"raft-heartbeat-ticks,omitempty" toml:"raft-heartbeat-ticks,omitempty"`
	// +optional
	RaftElectionTimeoutTicks *int64 `json:"raft-election-timeout-ticks,omitempty" toml:"raft-election-timeout-ticks,omitempty"`
	// When the entry exceed the max size, reject to propose it.
	// Optional: Defaults to 8MB
	// +optional
	RaftEntryMaxSize string `json:"raft-entry-max-size,omitempty" toml:"raft-entry-max-size,omitempty"`

	// Interval to gc unnecessary raft log (ms).
	// Optional: Defaults to 10s
	// +optional
	RaftLogGCTickInterval string `json:"raft-log-gc-tick-interval,omitempty" toml:"raft-log-gc-tick-interval,omitempty"`
	// A threshold to gc stale raft log, must >= 1.
	// Optional: Defaults to 50
	// +optional
	RaftLogGCThreshold *int64 `json:"raft-log-gc-threshold,omitempty" toml:"raft-log-gc-threshold,omitempty"`
	// When entry count exceed this value, gc will be forced trigger.
	// Optional: Defaults to 72000
	// +optional
	RaftLogGCCountLimit *int64 `json:"raft-log-gc-count-limit,omitempty" toml:"raft-log-gc-count-limit,omitempty"`
	// When the approximate size of raft log entries exceed this value
	// gc will be forced trigger.
	// Optional: Defaults to 72MB
	// +optional
	RaftLogGCSizeLimit string `json:"raft-log-gc-size-limit,omitempty" toml:"raft-log-gc-size-limit,omitempty"`
	// When a peer is not responding for this time, leader will not keep entry cache for it.
	// +optional
	RaftEntryCacheLifeTime string `json:"raft-entry-cache-life-time,omitempty" toml:"raft-entry-cache-life-time,omitempty"`
	// When a peer is newly added, reject transferring leader to the peer for a while.
	// +optional
	RaftRejectTransferLeaderDuration string `json:"raft-reject-transfer-leader-duration,omitempty" toml:"raft-reject-transfer-leader-duration,omitempty"`

	// Interval (ms) to check region whether need to be split or not.
	// Optional: Defaults to 10s
	// +optional
	SplitRegionCheckTickInterval string `json:"split-region-check-tick-interval,omitempty" toml:"split-region-check-tick-interval,omitempty"`
	/// When size change of region exceed the diff since last check, it
	/// will be checked again whether it should be split.
	// Optional: Defaults to 6MB
	// +optional
	RegionSplitCheckDiff string `json:"region-split-check-diff,omitempty" toml:"region-split-check-diff,omitempty"`
	/// Interval (ms) to check whether start compaction for a region.
	// Optional: Defaults to 5m
	// +optional
	RegionCompactCheckInterval string `json:"region-compact-check-interval,omitempty" toml:"region-compact-check-interval,omitempty"`
	// delay time before deleting a stale peer
	// Optional: Defaults to 10m
	// +optional
	CleanStalePeerDelay string `json:"clean-stale-peer-delay,omitempty" toml:"clean-stale-peer-delay,omitempty"`
	/// Number of regions for each time checking.
	// Optional: Defaults to 100
	// +optional
	RegionCompactCheckStep *int64 `json:"region-compact-check-step,omitempty" toml:"region-compact-check-step,omitempty"`
	/// Minimum number of tombstones to trigger manual compaction.
	// Optional: Defaults to 10000
	// +optional
	RegionCompactMinTombstones *int64 `json:"region-compact-min-tombstones,omitempty" toml:"region-compact-min-tombstones,omitempty"`
	/// Minimum percentage of tombstones to trigger manual compaction.
	/// Should between 1 and 100.
	// Optional: Defaults to 30
	// +optional
	RegionCompactTombstonesPercent *int64 `json:"region-compact-tombstones-percent,omitempty" toml:"region-compact-tombstones-percent,omitempty"`
	// Optional: Defaults to 60s
	// +optional
	PdHeartbeatTickInterval string `json:"pd-heartbeat-tick-interval,omitempty" toml:"pd-heartbeat-tick-interval,omitempty"`
	// Optional: Defaults to 10s
	// +optional
	PdStoreHeartbeatTickInterval string `json:"pd-store-heartbeat-tick-interval,omitempty" toml:"pd-store-heartbeat-tick-interval,omitempty"`
	// +optional
	SnapMgrGCTickInterval string `json:"snap-mgr-gc-tick-interval,omitempty" toml:"snap-mgr-gc-tick-interval,omitempty"`
	// +optional
	SnapGCTimeout string `json:"snap-gc-timeout,omitempty" toml:"snap-gc-timeout,omitempty"`
	// +optional
	// Optional: Defaults to 10m
	LockCfCompactInterval string `json:"lock-cf-compact-interval,omitempty" toml:"lock-cf-compact-interval,omitempty"`
	// +optional
	// Optional: Defaults to 256MB
	LockCfCompactBytesThreshold string `json:"lock-cf-compact-bytes-threshold,omitempty" toml:"lock-cf-compact-bytes-threshold,omitempty"`

	// +optional
	NotifyCapacity *int64 `json:"notify-capacity,omitempty" toml:"notify-capacity,omitempty"`
	// +optional
	MessagesPerTick *int64 `json:"messages-per-tick,omitempty" toml:"messages-per-tick,omitempty"`

	/// When a peer is not active for max-peer-down-duration
	/// the peer is considered to be down and is reported to PD.
	// Optional: Defaults to 5m
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
	// Optional: Defaults to 0
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
	// Optional: Defaults to 10m
	CleanupImportSstInterval string `json:"cleanup-import-sst-interval,omitempty" toml:"cleanup-import-sst-interval,omitempty"`

	// +optional
	ApplyMaxBatchSize *int64 `json:"apply-max-batch-size,omitempty" toml:"apply-max-batch-size,omitempty"`
	// Optional: Defaults to 2
	// +optional
	ApplyPoolSize *int64 `json:"apply-pool-size,omitempty" toml:"apply-pool-size,omitempty"`

	// +optional
	StoreMaxBatchSize *int64 `json:"store-max-batch-size,omitempty" toml:"store-max-batch-size,omitempty"`
	// Optional: Defaults to 2
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
	// Optional: Defaults to false
	// optional
	SplitRegionOnTable *bool `json:"split-region-on-table,omitempty" toml:"split-region-on-table,omitempty"`

	// One split check produces several split keys in batch. This config limits the number of produced
	// split keys in one batch.
	// optional
	BatchSplitLimit *int64 `json:"batch-split-limit,omitempty" toml:"batch-split-limit,omitempty"`

	// When Region [a,e) size exceeds `region-max-size`, it will be split into several Regions [a,b),
	// [b,c), [c,d), [d,e) and the size of [a,b), [b,c), [c,d) will be `region-split-size` (or a
	// little larger). See also: region-split-size
	// Optional: Defaults to 144MB
	// optional
	RegionMaxSize string `json:"region-max-size,omitempty" toml:"region-max-size,omitempty"`

	// When Region [a,e) size exceeds `region-max-size`, it will be split into several Regions [a,b),
	// [b,c), [c,d), [d,e) and the size of [a,b), [b,c), [c,d) will be `region-split-size` (or a
	// little larger). See also: region-max-size
	// Optional: Defaults to 96MB
	// optional
	RegionSplitSize string `json:"region-split-size,omitempty" toml:"region-split-size,omitempty"`

	// When the number of keys in Region [a,e) exceeds the `region-max-keys`, it will be split into
	// several Regions [a,b), [b,c), [c,d), [d,e) and the number of keys in [a,b), [b,c), [c,d) will be
	// `region-split-keys`. See also: region-split-keys
	// Optional: Defaults to 1440000
	// optional
	RegionMaxKeys *int64 `json:"region-max-keys,omitempty" toml:"region-max-keys,omitempty"`

	// When the number of keys in Region [a,e) exceeds the `region-max-keys`, it will be split into
	// several Regions [a,b), [b,c), [c,d), [d,e) and the number of keys in [a,b), [b,c), [c,d) will be
	// `region-split-keys`. See also: region-max-keys
	// Optional: Defaults to 960000
	// optional
	RegionSplitKeys *int64 `json:"region-split-keys,omitempty" toml:"region-split-keys,omitempty"`
}

// +k8s:openapi-gen=true
type TiKVEncryptionConfig struct {
	// Encrypyion method, use data key encryption raw rocksdb data
	// Possible values: plaintext, aes128-ctr, aes192-ctr, aes256-ctr
	// Optional: Default to plaintext
	// optional
	Method string `json:"method,omitempty" toml:"method,omitempty"`

	// The frequency of datakey rotation, It managered by tikv
	// Optional: default to 7d
	// optional
	DataKeyRotationPeriod string `json:"data-key-rotation-period,omitempty" toml:"data-key-rotation-period,omitempty"`

	// Master key config
	MasterKey *TiKVMasterKeyConfig `json:"master-key,omitempty" toml:"master-key,omitempty"`

	// Previous master key config
	// It used in master key rotation, the data key should decryption by previous master key and  then encrypytion by new master key
	PreviousMasterKey *TiKVMasterKeyConfig `json:"previous-master-key,omitempty" toml:"previoud-master-key,omitempty"`
}

// +k8s:openapi-gen=true
type TiKVMasterKeyConfig struct {
	// Use KMS encryption or use file encryption, possible values: kms, file
	// If set to kms, kms MasterKeyKMSConfig should be filled, if set to file MasterKeyFileConfig should be filled
	// optional
	Type string `json:"type,omitempty" toml:"type,omitempty"`

	// Master key file config
	// If the type set to file, this config should be filled
	MasterKeyFileConfig `json:",inline"`

	// Master key KMS config
	// If the type set to kms, this config should be filled
	MasterKeyKMSConfig `json:",inline"`
}

// +k8s:openapi-gen=true
type MasterKeyFileConfig struct {
	// Encrypyion method, use master key encryption data key
	// Possible values: plaintext, aes128-ctr, aes192-ctr, aes256-ctr
	// Optional: Default to plaintext
	// optional
	Method string `json:"method,omitempty" toml:"method,omitempty"`

	// Text file containing the key in hex form, end with '\n'
	Path string `json:"path" toml:"path"`
}

// +k8s:openapi-gen=true
type MasterKeyKMSConfig struct {
	// AWS CMK key-id it can be find in AWS Console or use aws cli
	// This field is required
	KeyID string `json:"key-id" toml:"key-id"`

	// AccessKey of AWS user, leave empty if using other authrization method
	// optional
	AccessKey string `json:"access-key,omitempty" toml:"access-key,omitempty"`

	// SecretKey of AWS user, leave empty if using other authrization method
	// optional
	SecretKey string `json:"secret-access-key,omitempty" toml:"access-key,omitempty"`

	// Region of this KMS key
	// Optional: Default to us-east-1
	// optional
	Region string `json:"region,omitempty" toml:"region,omitempty"`

	// Used for KMS compatible KMS, such as Ceph, minio, If use AWS, leave empty
	// optional
	Endpoint string `json:"endpoint,omitempty" toml:"endpoint,omitempty"`
}

// +k8s:openapi-gen=true
type TiKVPessimisticTxn struct {
	// +optional
	Enabled bool `json:"enabled,omitempty" toml:"enabled,omitempty"`
	// +optional
	WaitForLockTimeout int32 `json:"wait-for-lock-timeout,omitempty" toml:"wait-for-lock-timeout,omitempty"`
	// +optional
	WakeUpDelayDuration int32 `json:"wake-up-delay-duration,omitempty" toml:"wake-up-delay-duration,omitempty"`
	// +optional
	Pipelined bool `json:"pipelined,omitempty" toml:"pipelined,omitempty"`
}
