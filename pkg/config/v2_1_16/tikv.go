package config

type TikvConfig struct {
	LogLevel                     string          `toml:"log-level" json:"log-level"`
	LogFile                      string          `toml:"log-file" json:"log-level"`
	LogRotationTimespan          Duration        `toml:"log-rotation-timespan" json:"log-rotation-timespan"`
	PanicWhenUnexpectedKeyOrData bool            `toml:"panic-when-unexpected-key-or-data" json:"panic-when-unexpected-key-or-data"`
	ReadPool                     ReadPoolConfig  `toml:"readpool" json:"readpool"`
	Server                       ServerConfig    `toml:"server" json:"toml"`
	Storage                      StorageConfig   `toml:"storage" json:"storage"`
	PD                           PDConfig        `toml:"pd" json:"pd"`
	Metric                       MetricConfig    `toml:"metric" json:"metric"`
	RaftStore                    RaftStoreConfig `toml:"raftstore" json:"raftstore"`
	Coprocessor                  CopConfig       `toml:"coprocessor" json:"coprocessor"`
	Rocksdb                      DBConfig        `toml:"rocksdb" json:"rocksdb"`
	Raftdb                       RaftDBConfig    `toml:"raftdb" json:"raftdb"`
	Security                     SecurityConfig  `toml:"security" json:"security"`
	Import                       ImportConfig    `toml:"import" json:"import"`
}

type PDConfig struct {
	Endpoints []string `toml:"endpoints" json:"endpoints"`
}

type DBConfig struct {
	WalRecoveryMode                  DBRecoveryMode `toml:"wal-recovery-mode" json:"wal-recovery-mode"`
	WalDir                           string         `toml:"wal-dir" json:"wal-dir"`
	WalTTLSeconds                    uint64         `toml:"wal-ttl-seconds" json:"wal-ttl-seconds"`
	WalSizeLimit                     ByteSize       `toml:"wal-size-limit" json:"wal-size-limit"`
	MaxTotalWalSize                  ByteSize       `toml:"max-total-wal-size" json:"max-total-wal-size"`
	MaxBackgroundJobs                int32          `toml:"max-background-jobs" json:"max-background-jobs"`
	MaxManifestFileSize              ByteSize       `toml:"max-manifest-file-size" json:"max-manifest-file-size"`
	CreateIfMissing                  bool           `toml:"create-if-missing" json:"create-if-missing"`
	MaxOpenFiles                     int32          `toml:"max-open-files" json:"max-open-files"`
	EnableStatistics                 bool           `toml:"enable-statistics" json:"enable-statistics"`
	StatsDumpPeriod                  Duration       `toml:"stats-dump-period" json:"stats-dump-period"`
	CompactionReadaheadSize          ByteSize       `toml:"compaction-readahead-size" json:"compaction-readhead-size"`
	InfoLogMaxSize                   ByteSize       `toml:"info-log-max-size" json:"info-log-max-size"`
	InfoLogRollTime                  Duration       `toml:"info-log-roll-time" json:"info-log-roll-time"`
	InfoLogKeepLogFileNum            uint64         `toml:"info-log-keep-log-file-num" json:"info-log-keep-log-file-num"`
	InfoLogDir                       string         `toml:"info-log-dir" json:"info-log-dir"`
	RateBytesPerSec                  ByteSize       `toml:"rate-bytes-per-sec" json:"rate-bytes-per-sec"`
	BytesPerSync                     ByteSize       `toml:"bytes-per-sync" json:"bytes-per-sync"`
	WalBytesPerSync                  ByteSize       `toml:"wal-bytes-per-sync" json:"wal-bytes-per-sync"`
	MaxSubCompactions                uint32         `toml:"max-sub-compactions" json:"max-sub-compactions"`
	WritableFileMaxBufferSize        ByteSize       `toml:"writable-file-max-buffer-size" json:"writable-file-max-buffer-size"`
	UseDirectIOForFlushAndCompaction bool           `toml:"use-direct-io-for-flush-and-compaction" json:"use-direct-io-for-flush-and-compaction"`
	EnablePipelinedWrite             bool           `toml:"enable-pipelined-write" json:"enable-pipelined-write"`
	Defaultcf                        CfConfig       `toml:"defaultcf" json:"defaultcf"`
	Writecf                          CfConfig       `toml:"writecf" json:"writecf"`
	Lockcf                           CfConfig       `toml:"lockcf" json:"lockcf"`
	Raftcf                           CfConfig       `toml:"raftcf" json:"raftcf"`
}

type ImportConfig struct {
	ImportDir           string   `toml:"import-dir" json:"import-dir"`
	NumThreads          uint     `toml:"num-threads" json:"num-threads"`
	NumImportJobs       uint     `toml:"num-import-jobs" json:"num-import-jobs"`
	NumImportSstJobs    uint     `toml:"num-import-sst-jobs" json:"num-import-sst-jobs"`
	MaxPrepareDuration  Duration `toml:"max-prepare-duration" json:"max-prepare-duration"`
	RegionSplitSize     ByteSize `toml:"region-split-size" json:"region-split-size"`
	StreamChannelWindow uint     `toml:"stream-channel-window" json:"stream-channel-window"`
	MaxOpenEngines      uint     `toml:"max-open-engines" json:"max-open-engines"`
	UploadSpeedLimit    ByteSize `toml:"upload-speed-limit" json:"upload-speed-limit"`
	MinAvailableRatio   float64  `toml:"min-available-ratio" json:"min-available-ratio"`
}

type RaftDBConfig struct {
	WalRecoveryMode                  DBRecoveryMode `toml:"wal-recovery-mode" json:"wal-recovery-mode"`
	WalDir                           string         `toml:"wal-dir" json:"wal-dir"`
	WalTTLSeconds                    uint64         `toml:"wal-ttl-seconds" json:"wal-ttl-seconds"`
	WalSizeLimit                     ByteSize       `toml:"wal-size-limit" json:"wal-size-limit"`
	MaxTotalWalSize                  ByteSize       `toml:"max-total-wal-size" json:"max-total-wal-size"`
	MaxManifestFileSize              ByteSize       `toml:"max-manifest-file-size" json:"max-manifest-file-size"`
	CreateIfMissing                  bool           `toml:"create-if-missing" json:"create-if-missing"`
	MaxOpenFiles                     int32          `toml:"max-open-files" json:"max-open-files"`
	EnableStatistics                 bool           `toml:"enable-statistics" json:"enable-statistics"`
	StatsDumpPeriod                  Duration       `toml:"stats-dump-period" json:"stats-dump-period"`
	CompactionReadaheadSize          ByteSize       `toml:"compaction-readahead-size" json:"compaction-readhead-size"`
	InfoLogMaxSize                   ByteSize       `toml:"info-log-max-size" json:"info-log-max-size"`
	InfoLogRollTime                  Duration       `toml:"info-log-roll-time" json:"info-log-roll-time"`
	InfoLogKeepLogFileNum            uint64         `toml:"info-log-keep-log-file-num" json:"info-log-keep-log-file-num"`
	InfoLogDir                       string         `toml:"info-log-dir" json:"info-log-dir"`
	MaxSubCompactions                uint32         `toml:"max-sub-compactions" json:"max-sub-compactions"`
	WritableFileMaxBufferSize        ByteSize       `toml:"writable-file-max-buffer-size" json:"writable-file-max-buffer-size"`
	UseDirectIoForFlushAndCompaction bool           `toml:"use-direct-io-for-flush-and-compaction" json:"use-direct-io-for-flush-and-compaction"`
	EnablePipelinedWrite             bool           `toml:"enable-pipelined-write" json:"enable-pipelined-write"`
	AllowConcurrentMemtableWrite     bool           `toml:"allow-concurrent-memtable-write" json:"allow-concurrent-memtable-write"`
	BytesPerSync                     ByteSize       `toml:"bytes-per-sync" json:"bytes-per-sync"`
	WalBytesPerSync                  ByteSize       `toml:"wal-bytes-per-sync" json:"wal-bytes-per-sync"`
	Defaultcf                        CfConfig       `toml:"defaultcf" json:"defaultcf"`
}

type CopConfig struct {
	/// When it is true, it will try to split a region with table prefix if
	/// that region crosses tables.
	SplitRegionOnTable bool `toml:"split-region-on-table" json:"split-region-on-table"`

	/// For once split check, there are several split_key produced for batch.
	/// batch_split_limit limits the number of produced split-key for one batch.
	BatchSplitLimit uint64 `toml:"batch-split-limit" json:"batch-split-limit"`

	/// When region [a,e) size meets region_max_size, it will be split into
	/// several regions [a,b), [b,c), [c,d), [d,e). And the size of [a,b),
	/// [b,c), [c,d) will be region_split_size (maybe a little larger).
	RegionMaxSize   ByteSize `toml:"region-max-size" json:"region-max-size"`
	RegionSplitSize ByteSize `toml:"region-split-size" json:"region-split-size"`

	/// When the number of keys in region [a,e) meets the region_max_keys,
	/// it will be split into two several regions [a,b), [b,c), [c,d), [d,e).
	/// And the number of keys in [a,b), [b,c), [c,d) will be region_split_keys.
	RegionMaxKeys   uint64 `toml:"region-max-keys" json:"region-max-keys"`
	RegionSplitKeys uint64 `toml:"region-split-keys" json:"region-split-keys"`
}

type ReadPoolConfig struct {
	Storage     ReadPool `toml:"storage" json:"storage"`
	Coprocessor ReadPool `toml:"coprocessor" json:"coprocessor"`
}

type GRPCCompressionType string
type DBRecoveryMode int
type DBCompactionStyle int
type CompactionPriority int
type DBCompressionType string

const (
	GRPCNoneCompression    GRPCCompressionType = "none"
	GRPCDeflateCompression GRPCCompressionType = "deflate"
	GRPCGzipCompression    GRPCCompressionType = "gzip"

	TolerateCorruptedTailRecords DBRecoveryMode = iota
	AbsoluteConsistency
	PointInTime
	SkipAnyCorruptedRecords

	DBLevelCompaction DBCompactionStyle = iota
	DBUniversalCompaction
	DBFifoCompaction
	DBNoneCompaction

	CompactionByCompensatedSize CompactionPriority = iota
	CompactionOldestLargestSeqFirst
	CompactionOldestSmallestSeqFirst
	CompactionMinOverlappingRatio

	DBNoCompression           DBCompressionType = "no"
	DBSnappyCompression       DBCompressionType = "snappy"
	DBZlibCompression         DBCompressionType = "zlib"
	DBBz2Compression          DBCompressionType = "bzip2"
	DBLz4Compression          DBCompressionType = "lz4"
	DBLz4hcCompression        DBCompressionType = "lz4hc"
	DBZstdCompression         DBCompressionType = "zstd"
	DBZstdNotFinalCompression DBCompressionType = "zstd-not-final"
	DBDisableCompression      DBCompressionType = "disable"
)

type ServerConfig struct {
	// Server listening address.
	Addr string `toml:"addr" json:"addr"`

	// Server advertise listening address for outer communication.
	// If not set, we will use listening address instead.
	AdvertiseAddr string `toml:"advertise-addr" json:"advertise-addr"`

	// These are related to TiKV status.
	StatusAddr           string `toml:"status-addr" json:"status-addr"`
	StatusThreadPoolSize uint   `toml:"status-thread-pool-size" "json:"status-thread-pool-size""`

	// TODO: use CompressionAlgorithms instead once it supports traits like Clone etc.
	GRPCCompressionType         GRPCCompressionType `toml:"grpc-compression-type" json:"grpc-compression-type"`
	GRPCConcurrency             uint
	GRPCConcurrentStream        int32
	GRPCRaftConnNum             uint
	GRPCStreamInitialWindowSize ByteSize
	GRPCKeepaliveTime           Duration
	GRPCKeepaliveTimeout        Duration
	/// How many snapshots can be sent concurrently.
	ConcurrentSendSnapLimit uint
	/// How many snapshots can be recv concurrently.
	ConcurrentRecvSnapLimit          uint
	EndPointRecursionLimit           uint32
	EndPointStreamChannelSize        uint
	EndPointBatchRowLimit            uint
	EndPointStreamBatchRowLimit      uint
	EndPointRequestMaxHandleDuration Duration
	SnapMaxWriteBytesPerSec          ByteSize
	SnapMaxTotalSize                 ByteSize

	// Server labels to specify some attributes about this server.
	Labels map[string]string `toml:"labels" json:"labels"`

	// deprecated. use readpool.coprocessor.xx_concurrency.
	EndPointConcurrency *uint

	// deprecated. use readpool.coprocessor.stack_size.
	EndPointStackSize *ByteSize

	// deprecated. use readpool.coprocessor.max_tasks_per_worker_xx.
	EndPointMaxTasks *uint
}

type StorageConfig struct {
	DataDir                        string   `toml:"data-dir" json:"data-dir"`
	GCRatioThreshold               float64  `toml:"gc-ratio-threshold" json:"gc-ratio-threshold"`
	MaxKeySize                     uint     `toml:"max-key-size" json:"max-key-size"`
	SchedulerNotifyCapacity        uint     `toml:"scheduler-notify-capacity" json:"scheduler-notify-capacity"`
	SchedulerConcurrency           uint     `toml:"scheduler-concurrency" json:"scheduler-concurrency"`
	SchedulerWorkerPoolSize        uint     `toml:"scheduler-worker-pool-size" json:"scheduler-worker-pool-size"`
	SchedulerPendingWriteThreshold ByteSize `toml:"scheduler-pending-write-threshold" json:"scheduler-pending-write-threshold"`
}

type RaftStoreConfig struct {
	// true for high reliability, prevent data loss when power failure.
	SyncLog bool `toml:"sync-log" json:"sync-log"`
	// minimizes disruption when a partitioned node rejoins the cluster by using a two phase election.
	Prevote    bool
	RaftdbPath string

	// store capacity. 0 means no limit.
	Capacity ByteSize

	// RaftBaseTickInterval is a base tick interval (ms).
	RaftBaseTickInterval        Duration
	RaftHeartbeatTicks          uint
	RaftElectionTimeoutTicks    uint
	RaftMinElectionTimeoutTicks uint
	RaftMaxElectionTimeoutTicks uint
	RaftMaxSizePerMsg           ByteSize
	RaftMaxInflightMsgs         uint
	// When the entry exceed the max size, reject to propose it.
	RaftEntryMaxSize ByteSize

	// Interval to gc unnecessary raft log (ms).
	RaftLogGcTickInterval Duration
	// A threshold to gc stale raft log, must >= 1.
	RaftLogGcThreshold uint64
	// When entry count exceed this value, gc will be forced trigger.
	RaftLogGcCountLimit uint64
	// When the approximate size of raft log entries exceed this value,
	// gc will be forced trigger.
	RaftLogGcSizeLimit ByteSize
	// When a peer is not responding for this time, leader will not keep entry cache for it.
	RaftEntryCacheLifeTime Duration
	// When a peer is newly added, reject transferring leader to the peer for a while.
	RaftRejectTransferLeaderDuration Duration

	// Interval (ms) to check region whether need to be split or not.
	SplitRegionCheckTickInterval Duration
	/// When size change of region exceed the diff since last check, it
	/// will be checked again whether it should be split.
	RegionSplitCheckDiff ByteSize
	/// Interval (ms) to check whether start compaction for a region.
	RegionCompactCheckInterval Duration
	// delay time before deleting a stale peer
	CleanStalePeerDelay Duration
	/// Number of regions for each time checking.
	RegionCompactCheckStep uint64
	/// Minimum number of tombstones to trigger manual compaction.
	RegionCompactMinTombstones uint64
	/// Minimum percentage of tombstones to trigger manual compaction.
	/// Should between 1 and 100.
	RegionCompactTombstonesPercent uint64
	PdHeartbeatTickInterval        Duration
	PdStoreHeartbeatTickInterval   Duration
	SnapMgrGcTickInterval          Duration
	SnapGcTimeout                  Duration
	LockCfCompactInterval          Duration
	LockCfCompactBytesThreshold    ByteSize

	NotifyCapacity  uint
	MessagesPerTick uint

	/// When a peer is not active for MaxPeerDownDuration,
	/// the peer is considered to be down and is reported to PD.
	MaxPeerDownDuration Duration

	/// If the leader of a peer is missing for longer than MaxLeaderMissingDuration,
	/// the peer would ask pd to confirm whether it is valid in any region.
	/// If the peer is stale and is not valid in any region, it will destroy itself.
	MaxLeaderMissingDuration Duration
	/// Similar to the MaxLeaderMissingDuration, instead it will log warnings and
	/// try to alert monitoring systems, if there is any.
	AbnormalLeaderMissingDuration Duration
	PeerStaleStateCheckInterval   Duration

	LeaderTransferMaxLogLag uint64

	SnapApplyBatchSize ByteSize

	// Interval (ms) to check region whether the data is consistent.
	ConsistencyCheckInterval Duration

	ReportRegionFlowInterval Duration

	// The lease provided by a successfully proposed and applied entry.
	RaftStoreMaxLeaderLease Duration

	// Right region derive origin region id when split.
	RightDeriveWhenSplit bool

	AllowRemoveLeader bool

	/// Max log gap allowed to propose merge.
	MergeMaxLogGap uint64
	/// Interval to repropose merge.
	MergeCheckTickInterval Duration

	UseDeleteRange bool

	CleanupImportSstInterval Duration

	/// Maximum size of every local read task batch.
	LocalReadBatchSize uint64

	// Deprecated! These two configuration has been moved to Coprocessor.
	// They are preserved for compatibility check.
	RegionMaxSize   ByteSize
	RegionSplitSize ByteSize
}

type ReadPool struct {
	HighConcurrency         uint
	NormalConcurrency       uint
	LowConcurrency          uint
	MaxTasksPerWorkerHigh   uint
	MaxTasksPerWorkerNormal uint
	MaxTasksPerWorkerLow    uint
	StackSize               ByteSize
}

type CfConfig struct {
	BlockSize                       ByteSize             `toml:"block-size" json:"block-size"`
	BlockCacheSize                  ByteSize             `toml:"block-cache-size" json:"block-cache-size"`
	DisableBlockCache               bool                 `toml:"disable-block-cache" json:"disable-block-cache"`
	CacheIndexAndFilterBlocks       bool                 `toml:"cache-index-and-filter-blocks" json:"cache-index-and-filter-blocks"`
	PinL0FilterAndIndexBlocks       bool                 `toml:"pin-l0-filter-and-index-blocks" json:"pin-l0-filter-and-index-blocks"`
	UseBloomFilter                  bool                 `toml:"use-bloom-filter" json:"use-bloom-filter"`
	WholeKeyFiltering               bool                 `toml:"whole-key-filtering" json:"whole-key-filtering"`
	BloomFilterBitsPerKey           int32                `toml:"bloom-filter-bits-per-key" json:"bloom-filter-bits-per-key"`
	BlockBasedBloomFilter           bool                 `toml:"block-based-bloom-filter" json:"block-based-bloom-filter"`
	ReadAmpBytesPerBit              uint32               `toml:"read-amp-bytes-per-bit" json:"read-amp-bytes-per-bit"`
	CompressionPerLevel             [7]DBCompressionType `toml:"compression-per-level" json:"compression-per-level"`
	WriteBufferSize                 ByteSize             `toml:"write-buffer-size" json:"write-buffer-size"`
	MaxWriteBufferNumber            int32                `toml:"max-write-buffer-number" json:"max-write-buffer-number"`
	MinWriteBufferNumberToMerge     int32                `toml:"min-write-buffer-number-to-merge" json:"min-write-buffer-number-to-merge"`
	MaxBytesForLevelBase            ByteSize             `toml:"max-bytes-for-level-base" json:"max-bytes-for-level-base"`
	TargetFileSizeBase              ByteSize             `toml:"target-file-size-base" json:"target-file-size-base"`
	Level0FileNumCompactionTrigger  int32                `toml:"level0-file-num-compaction-trigger" json:"level0-file-num-compaction-trigger"`
	Level0SlowdownWritesTrigger     int32                `toml:"level0-slowdown-writes-trigger" json:"level0-slowdown-writes-trigger"`
	Level0StopWritesTrigger         int32                `toml:"level0-stop-writes-trigger" json:"level0-stop-writes-trigger"`
	MaxCompactionBytes              ByteSize             `toml:"max-compaction-bytes" json:"max-compaction-bytes"`
	CompactionPri                   CompactionPriority   `toml:"compaction-pri" json:"compaction-pri"`
	DynamicLevelBytes               bool                 `toml:"dynamic-level-bytes" json:"dynamic-level-bytes"`
	NumLevels                       int32                `toml:"num-levels" json:"num-levels"`
	MaxBytesForLevelMultiplier      int32                `toml:"max-bytes-for-level-multiplier" json:"max-bytes-for-level-multiplier"`
	CompactionStyle                 DBCompactionStyle    `toml:"compaction-style" json:"compaction-style"`
	DisableAutoCompactions          bool                 `toml:"disable-auto-compactions" json:"disable-auto-compactions"`
	SoftPendingCompactionBytesLimit ByteSize             `toml:"soft-pending-compaction-bytes-limit" json:"soft-pending-compaction-bytes-limit"`
	HardPendingCompactionBytesLimit ByteSize             `toml:"hard-pending-compaction-bytes-limit" json:"hard-pending-compaction-bytes-limit"`
}
