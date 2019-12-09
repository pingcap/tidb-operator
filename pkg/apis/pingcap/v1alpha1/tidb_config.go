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
	"time"

	zaplog "github.com/pingcap/log"
)

// Maintain a copy of TiDBConfig to make it more friendly with the kubernetes API:
//
//  - add 'omitempty' json and toml tag to avoid passing the empty value of primitive types to tidb-server, e.g. 0 of int
//  - change all numeric type to pointer, e.g. uint -> *uint, so that 'omitempty' could work properly
//  - add openapi-gen tags so that the kubernetes-style OpenAPI schema could be properly generated
//    necessary because tidb-server add new config fields continuously
//  - make the whole config struct deepcopy friendly
//
// only adding field is allowed, DO NOT change existing field definition or remove field.
// some fields maybe illegal for certain version of TiDB, this should be addressed in the ValidationWebhook.

// initially copied from TiDB v3.0.6

// TiDBConfig is the configuration of tidb-server
// +k8s:openapi-gen=true
type TiDBConfig struct {
	// +optional
	// +kubebuilder:default="0.0.0.0"
	Host string `toml:"host,omitempty" json:"host,omitempty"`
	// +optional
	AdvertiseAddress string `toml:"advertise-address,omitempty" json:"advertise-address,omitempty"`
	// +optional
	Port *uint `toml:"port,omitempty" json:"port,omitempty"`
	// +optional
	Cors string `toml:"cors,omitempty" json:"cors,omitempty"`
	// +optional
	Store string `toml:"store,omitempty" json:"store,omitempty"`
	// +optional
	Path string `toml:"path,omitempty" json:"path,omitempty"`
	// +optional
	Socket string `toml:"socket,omitempty" json:"socket,omitempty"`
	// +optional
	Lease string `toml:"lease,omitempty" json:"lease,omitempty"`
	// +optional
	RunDDL bool `toml:"run-ddl,omitempty" json:"run-ddl,omitempty"`
	// +optional
	SplitTable bool `toml:"split-table,omitempty" json:"split-table,omitempty"`
	// +optional
	TokenLimit *uint `toml:"token-limit,omitempty" json:"token-limit,omitempty"`
	// +optional
	OOMAction string `toml:"oom-action,omitempty" json:"oom-action,omitempty"`
	// +optional
	MemQuotaQuery *int64 `toml:"mem-quota-query,omitempty" json:"mem-quota-query,omitempty"`
	// +optional
	EnableStreaming bool `toml:"enable-streaming,omitempty" json:"enable-streaming,omitempty"`
	// +optional
	EnableBatchDML bool `toml:"enable-batch-dml,omitempty" json:"enable-batch-dml,omitempty"`
	// +optional
	TxnLocalLatches *TxnLocalLatches `toml:"txn-local-latches,omitempty" json:"txn-local-latches,omitempty"`
	// +optional
	LowerCaseTableNames *int `toml:"lower-case-table-names,omitempty" json:"lower-case-table-names,omitempty"`
	// +optional
	Log *Log `toml:"log,omitempty" json:"log,omitempty"`
	// +optional
	Security *Security `toml:"security,omitempty" json:"security,omitempty"`
	// +optional
	Status *Status `toml:"status,omitempty" json:"status,omitempty"`
	// +optional
	Performance *Performance `toml:"performance,omitempty" json:"performance,omitempty"`
	// +optional
	PreparedPlanCache *PreparedPlanCache `toml:"prepared-plan-cache,omitempty" json:"prepared-plan-cache,omitempty"`
	// +optional
	OpenTracing *OpenTracing `toml:"opentracing,omitempty" json:"opentracing,omitempty"`
	// +optional
	ProxyProtocol *ProxyProtocol `toml:"proxy-protocol,omitempty" json:"proxy-protocol,omitempty"`
	// +optional
	TiKVClient *TiKVClient `toml:"tikv-client,omitempty" json:"tikv-client,omitempty"`
	// +optional
	Binlog *Binlog `toml:"binlog,omitempty" json:"binlog,omitempty"`
	// +optional
	CompatibleKillQuery bool `toml:"compatible-kill-query,omitempty" json:"compatible-kill-query,omitempty"`
	// +optional
	Plugin *Plugin `toml:"plugin,omitempty" json:"plugin,omitempty"`
	// +optional
	PessimisticTxn *PessimisticTxn `toml:"pessimistic-txn,omitempty" json:"pessimistic-txn,omitempty"`
	// +optional
	CheckMb4ValueInUTF8 bool `toml:"check-mb4-value-in-utf8,omitempty" json:"check-mb4-value-in-utf8,omitempty"`
	// +optional
	AlterPrimaryKey bool `toml:"alter-primary-key,omitempty" json:"alter-primary-key,omitempty"`
	// +optional
	TreatOldVersionUTF8AsUTF8MB4 bool `toml:"treat-old-version-utf8-as-utf8mb4,omitempty" json:"treat-old-version-utf8-as-utf8mb4,omitempty"`
	// +optional
	SplitRegionMaxNum *uint64 `toml:"split-region-max-num,omitempty" json:"split-region-max-num,omitempty"`
	// +optional
	StmtSummary *StmtSummary `toml:"stmt-summary,omitempty" json:"stmt-summary,omitempty"`
}

// Log is the log section of config.
// +k8s:openapi-gen=true
type Log struct {
	// Log level.
	// +optional
	Level string `toml:"level,omitempty" json:"level,omitempty"`
	// Log format. one of json, text, or console.
	// +optional
	Format string `toml:"format,omitempty" json:"format,omitempty"`
	// Disable automatic timestamps in output.
	// +optional
	DisableTimestamp bool `toml:"disable-timestamp,omitempty" json:"disable-timestamp,omitempty"`
	// File log config.
	// +optional
	File zaplog.FileLogConfig `toml:"file,omitempty" json:"file,omitempty"`
	// +optional
	SlowQueryFile string `toml:"slow-query-file,omitempty" json:"slow-query-file,omitempty"`
	// +optional
	SlowThreshold *uint64 `toml:"slow-threshold,omitempty" json:"slow-threshold,omitempty"`
	// +optional
	ExpensiveThreshold *uint `toml:"expensive-threshold,omitempty" json:"expensive-threshold,omitempty"`
	// +optional
	QueryLogMaxLen *uint64 `toml:"query-log-max-len,omitempty" json:"query-log-max-len,omitempty"`
	// +optional
	RecordPlanInSlowLog uint32 `toml:"record-plan-in-slow-log,omitempty" json:"record-plan-in-slow-log,omitempty"`
}

// Security is the security section of the config.
// +k8s:openapi-gen=true
type Security struct {
	// +optional
	SkipGrantTable bool `toml:"skip-grant-table,omitempty" json:"skip-grant-table,omitempty"`
	// +optional
	SSLCA string `toml:"ssl-ca,omitempty" json:"ssl-ca,omitempty"`
	// +optional
	SSLCert string `toml:"ssl-cert,omitempty" json:"ssl-cert,omitempty"`
	// +optional
	SSLKey string `toml:"ssl-key,omitempty" json:"ssl-key,omitempty"`
	// +optional
	ClusterSSLCA string `toml:"cluster-ssl-ca,omitempty" json:"cluster-ssl-ca,omitempty"`
	// +optional
	ClusterSSLCert string `toml:"cluster-ssl-cert,omitempty" json:"cluster-ssl-cert,omitempty"`
	// +optional
	ClusterSSLKey string `toml:"cluster-ssl-key,omitempty" json:"cluster-ssl-key,omitempty"`
}

// Status is the status section of the config.
// +k8s:openapi-gen=true
type Status struct {
	// +optional
	ReportStatus bool `toml:"report-status,omitempty" json:"report-status,omitempty"`
	// +optional
	StatusHost string `toml:"status-host,omitempty" json:"status-host,omitempty"`
	// +optional
	StatusPort *uint `toml:"status-port,omitempty" json:"status-port,omitempty"`
	// +optional
	MetricsAddr string `toml:"metrics-addr,omitempty" json:"metrics-addr,omitempty"`
	// +optional
	MetricsInterval *uint `toml:"metrics-interval,omitempty" json:"metrics-interval,omitempty"`
	// +optional
	RecordQPSbyDB bool `toml:"record-db-qps,omitempty" json:"record-db-qps,omitempty"`
}

// Performance is the performance section of the config.
// +k8s:openapi-gen=true
type Performance struct {
	// +optional
	MaxProcs *uint `toml:"max-procs,omitempty" json:"max-procs,omitempty"`
	// +optional
	MaxMemory *uint64 `toml:"max-memory,omitempty" json:"max-memory,omitempty"`
	// +optional
	TCPKeepAlive bool `toml:"tcp-keep-alive,omitempty" json:"tcp-keep-alive,omitempty"`
	// +optional
	CrossJoin bool `toml:"cross-join,omitempty" json:"cross-join,omitempty"`
	// +optional
	StatsLease string `toml:"stats-lease,omitempty" json:"stats-lease,omitempty"`
	// +optional
	RunAutoAnalyze bool `toml:"run-auto-analyze,omitempty" json:"run-auto-analyze,omitempty"`
	// +optional
	StmtCountLimit *uint `toml:"stmt-count-limit,omitempty" json:"stmt-count-limit,omitempty"`
	// +optional
	FeedbackProbability *float64 `toml:"feedback-probability,omitempty" json:"feedback-probability,omitempty"`
	// +optional
	QueryFeedbackLimit *uint `toml:"query-feedback-limit,omitempty" json:"query-feedback-limit,omitempty"`
	// +optional
	PseudoEstimateRatio *float64 `toml:"pseudo-estimate-ratio,omitempty" json:"pseudo-estimate-ratio,omitempty"`
	// +optional
	ForcePriority string `toml:"force-priority,omitempty" json:"force-priority,omitempty"`
	// +optional
	BindInfoLease string `toml:"bind-info-lease,omitempty" json:"bind-info-lease,omitempty"`
	// +optional
	TxnEntryCountLimit *uint64 `toml:"txn-entry-count-limit,omitempty" json:"txn-entry-count-limit,omitempty"`
	// +optional
	TxnTotalSizeLimit *uint64 `toml:"txn-total-size-limit,omitempty" json:"txn-total-size-limit,omitempty"`
}

// PlanCache is the PlanCache section of the config.
// +k8s:openapi-gen=true
type PlanCache struct {
	// +optional
	Enabled bool `toml:"enabled,omitempty" json:"enabled,omitempty"`
	// +optional
	Capacity *uint `toml:"capacity,omitempty" json:"capacity,omitempty"`
	// +optional
	Shards *uint `toml:"shards,omitempty" json:"shards,omitempty"`
}

// TxnLocalLatches is the TxnLocalLatches section of the config.
// +k8s:openapi-gen=true
type TxnLocalLatches struct {
	// +optional
	Enabled bool `toml:"enabled,omitempty" json:"enabled,omitempty"`
	// +optional
	Capacity *uint `toml:"capacity,omitempty" json:"capacity,omitempty"`
}

// PreparedPlanCache is the PreparedPlanCache section of the config.
// +k8s:openapi-gen=true
type PreparedPlanCache struct {
	// +optional
	Enabled bool `toml:"enabled,omitempty" json:"enabled,omitempty"`
	// +optional
	Capacity *uint `toml:"capacity,omitempty" json:"capacity,omitempty"`
	// +optional
	MemoryGuardRatio *float64 `toml:"memory-guard-ratio,omitempty" json:"memory-guard-ratio,omitempty"`
}

// OpenTracing is the opentracing section of the config.
// +k8s:openapi-gen=true
type OpenTracing struct {
	// +optional
	Enable bool `toml:"enable,omitempty" json:"enable,omitempty"`
	// +optional
	Sampler OpenTracingSampler `toml:"sampler,omitempty" json:"sampler,omitempty"`
	// +optional
	Reporter OpenTracingReporter `toml:"reporter,omitempty" json:"reporter,omitempty"`
	// +optional
	RPCMetrics bool `toml:"rpc-metrics,omitempty" json:"rpc-metrics,omitempty"`
}

// OpenTracingSampler is the config for opentracing sampler.
// See https://godoc.org/github.com/uber/jaeger-client-go/config#SamplerConfig
// +k8s:openapi-gen=true
type OpenTracingSampler struct {
	// +optional
	Type string `toml:"type,omitempty" json:"type,omitempty"`
	// +optional
	Param *float64 `toml:"param,omitempty" json:"param,omitempty"`
	// +optional
	SamplingServerURL string `toml:"sampling-server-url,omitempty" json:"sampling-server-url,omitempty"`
	// +optional
	MaxOperations *int `toml:"max-operations,omitempty" json:"max-operations,omitempty"`
	// +optional
	SamplingRefreshInterval time.Duration `toml:"sampling-refresh-interval,omitempty" json:"sampling-refresh-interval,omitempty"`
}

// OpenTracingReporter is the config for opentracing reporter.
// See https://godoc.org/github.com/uber/jaeger-client-go/config#ReporterConfig
// +k8s:openapi-gen=true
type OpenTracingReporter struct {
	// +optional
	QueueSize *int `toml:"queue-size,omitempty" json:"queue-size,omitempty"`
	// +optional
	BufferFlushInterval time.Duration `toml:"buffer-flush-interval,omitempty" json:"buffer-flush-interval,omitempty"`
	// +optional
	LogSpans bool `toml:"log-spans,omitempty" json:"log-spans,omitempty"`
	// +optional
	LocalAgentHostPort string `toml:"local-agent-host-port,omitempty" json:"local-agent-host-port,omitempty"`
}

// ProxyProtocol is the PROXY protocol section of the config.
// +k8s:openapi-gen=true
type ProxyProtocol struct {
	// PROXY protocol acceptable client networks.
	// Empty string means disable PROXY protocol,
	// * means all networks.
	// +optional
	Networks string `toml:"networks,omitempty" json:"networks,omitempty"`
	// PROXY protocol header read timeout, Unit is second.
	// +optional
	HeaderTimeout *uint `toml:"header-timeout,omitempty" json:"header-timeout,omitempty"`
}

// TiKVClient is the config for tikv client.
// +k8s:openapi-gen=true
type TiKVClient struct {
	// GrpcConnectionCount is the max gRPC connections that will be established
	// with each tikv-server.
	// +optional
	GrpcConnectionCount *uint `toml:"grpc-connection-count,omitempty" json:"grpc-connection-count,omitempty"`
	// After a duration of this time in seconds if the client doesn't see any activity it pings
	// the server to see if the transport is still alive.
	// +optional
	GrpcKeepAliveTime *uint `toml:"grpc-keepalive-time,omitempty" json:"grpc-keepalive-time,omitempty"`
	// After having pinged for keepalive check, the client waits for a duration of Timeout in seconds
	// and if no activity is seen even after that the connection is closed.
	// +optional
	GrpcKeepAliveTimeout *uint `toml:"grpc-keepalive-timeout,omitempty" json:"grpc-keepalive-timeout,omitempty"`
	// CommitTimeout is the max time which command 'commit' will wait.
	// +optional
	CommitTimeout string `toml:"commit-timeout,omitempty" json:"commit-timeout,omitempty"`
	// MaxTxnTimeUse is the max time a Txn may use (in seconds) from its startTS to commitTS.
	// +optional
	MaxTxnTimeUse *uint `toml:"max-txn-time-use,omitempty" json:"max-txn-time-use,omitempty"`
	// MaxBatchSize is the max batch size when calling batch commands API.
	// +optional
	MaxBatchSize *uint `toml:"max-batch-size,omitempty" json:"max-batch-size,omitempty"`
	// If TiKV load is greater than this, TiDB will wait for a while to avoid little batch.
	// +optional
	OverloadThreshold *uint `toml:"overload-threshold,omitempty" json:"overload-threshold,omitempty"`
	// MaxBatchWaitTime in nanosecond is the max wait time for batch.
	// +optional
	MaxBatchWaitTime time.Duration `toml:"max-batch-wait-time,omitempty" json:"max-batch-wait-time,omitempty"`
	// BatchWaitSize is the max wait size for batch.
	// +optional
	BatchWaitSize *uint `toml:"batch-wait-size,omitempty" json:"batch-wait-size,omitempty"`
	// If a Region has not been accessed for more than the given duration (in seconds), it
	// will be reloaded from the PD.
	// +optional
	RegionCacheTTL *uint `toml:"region-cache-ttl,omitempty" json:"region-cache-ttl,omitempty"`
	// If a store has been up to the limit, it will return error for successive request to
	// prevent the store occupying too much token in dispatching level.
	// +optional
	StoreLimit int64 `toml:"store-limit,omitempty" json:"store-limit,omitempty"`
}

// Binlog is the config for binlog.
// +k8s:openapi-gen=true
type Binlog struct {
	// +optional
	Enable bool `toml:"enable,omitempty" json:"enable,omitempty"`
	// +optional
	WriteTimeout string `toml:"write-timeout,omitempty" json:"write-timeout,omitempty"`
	// If IgnoreError is true, when writing binlog meets error, TiDB would
	// ignore the error.
	// +optional
	IgnoreError bool `toml:"ignore-error,omitempty" json:"ignore-error,omitempty"`
	// Use socket file to write binlog, for compatible with kafka version tidb-binlog.
	// +optional
	BinlogSocket string `toml:"binlog-socket,omitempty" json:"binlog-socket,omitempty"`
	// The strategy for sending binlog to pump, value can be "range,omitempty" or "hash,omitempty" now.
	// +optional
	Strategy string `toml:"strategy,omitempty" json:"strategy,omitempty"`
}

// Plugin is the config for plugin
// +k8s:openapi-gen=true
type Plugin struct {
	// +optional
	Dir string `toml:"dir,omitempty" json:"dir,omitempty"`
	// +optional
	Load string `toml:"load,omitempty" json:"load,omitempty"`
}

// PessimisticTxn is the config for pessimistic transaction.
// +k8s:openapi-gen=true
type PessimisticTxn struct {
	// Enable must be true for 'begin lock' or session variable to start a pessimistic transaction.
	// +optional
	Enable bool `toml:"enable,omitempty" json:"enable,omitempty"`
	// The max count of retry for a single statement in a pessimistic transaction.
	// +optional
	MaxRetryCount *uint `toml:"max-retry-count,omitempty" json:"max-retry-count,omitempty"`
}

// StmtSummary is the config for statement summary.
// +k8s:openapi-gen=true
type StmtSummary struct {
	// The maximum number of statements kept in memory.
	// +optional
	MaxStmtCount *uint `toml:"max-stmt-count,omitempty" json:"max-stmt-count,omitempty"`
	// The maximum length of displayed normalized SQL and sample SQL.
	// +optional
	MaxSQLLength *uint `toml:"max-sql-length,omitempty" json:"max-sql-length,omitempty"`
}
