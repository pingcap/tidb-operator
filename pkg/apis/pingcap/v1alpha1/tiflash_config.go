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

package v1alpha1

// Port from TiFlash configurations till 2020/04/02

// TiFlashConfig is the configuration of TiFlash.
// +k8s:openapi-gen=true
type TiFlashConfig struct {
	// commonConfig is the Configuration of TiFlash process
	// +optional
	CommonConfig *CommonConfig `json:"config,omitempty"`

	// proxyConfig is the Configuration of proxy process
	// +optional
	// +k8s:openapi-gen=false
	ProxyConfig *ProxyConfig `json:"proxy,omitempty"`
}

// FlashServerConfig is the configuration of Proxy server.
// +k8s:openapi-gen=false
type FlashServerConfig struct {
	// +optional
	EngineAddr string `json:"engine-addr,omitempty" toml:"engine-addr,omitempty"`
	// +optional
	StatusAddr       string `json:"status-addr,omitempty" toml:"status-addr,omitempty"`
	TiKVServerConfig `json:",inline"`
}

// ProxyConfig is the configuration of TiFlash proxy process.
// All the configurations are same with those of TiKV except adding `engine-addr` in the TiKVServerConfig
// +k8s:openapi-gen=false
type ProxyConfig struct {
	// Optional: Defaults to info
	// +optional
	LogLevel string `json:"log-level,omitempty" toml:"log-level,omitempty"`
	// +optional
	LogFile string `json:"log-file,omitempty" toml:"log-file,omitempty"`
	// Optional: Defaults to 24h
	// +optional
	LogRotationTimespan string `json:"log-rotation-timespan,omitempty" toml:"log-rotation-timespan,omitempty"`
	// +optional
	PanicWhenUnexpectedKeyOrData *bool `json:"panic-when-unexpected-key-or-data,omitempty" toml:"panic-when-unexpected-key-or-data,omitempty"`
	// +optional
	Server *FlashServerConfig `json:"server,omitempty" toml:"server,omitempty"`
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
}

// CommonConfig is the configuration of TiFlash process.
// +k8s:openapi-gen=true
type CommonConfig struct {
	// Optional: Defaults to "/data0/tmp"
	// +optional
	// +k8s:openapi-gen=false
	TmpPath string `json:"tmp_path,omitempty" toml:"tmp_path,omitempty"`

	// Optional: Defaults to "TiFlash"
	// +optional
	// +k8s:openapi-gen=false
	DisplayName string `json:"display_name,omitempty" toml:"display_name,omitempty"`

	// Optional: Defaults to "default"
	// +optional
	// +k8s:openapi-gen=false
	DefaultProfile string `json:"default_profile,omitempty" toml:"default_profile,omitempty"`

	// Optional: Defaults to "/data0/db"
	// +optional
	// +k8s:openapi-gen=false
	Path string `json:"path,omitempty" toml:"path,omitempty"`

	// Optional: Defaults to false
	// +optional
	PathRealtimeMode *bool `json:"path_realtime_mode,omitempty" toml:"path_realtime_mode,omitempty"`

	// Optional: Defaults to 5368709120
	// +optional
	MarkCacheSize *int64 `json:"mark_cache_size,omitempty" toml:"mark_cache_size,omitempty"`

	// Optional: Defaults to 5368709120
	// +optional
	MinmaxIndexCacheSize *int64 `json:"minmax_index_cache_size,omitempty" toml:"minmax_index_cache_size,omitempty"`

	// Optional: Defaults to "0.0.0.0"
	// +optional
	// +k8s:openapi-gen=false
	ListenHost string `json:"listen_host,omitempty" toml:"listen_host,omitempty"`

	// Optional: Defaults to 9000
	// +optional
	// +k8s:openapi-gen=false
	TCPPort *int32 `json:"tcp_port,omitempty" toml:"tcp_port,omitempty"`
	// Optional: Defaults to 8123
	// +optional
	// +k8s:openapi-gen=false
	HTTPPort *int32 `json:"http_port,omitempty" toml:"http_port,omitempty"`
	// Optional: Defaults to 9009
	// +optional
	// +k8s:openapi-gen=false
	InternalServerHTTPPort *int32 `json:"interserver_http_port,omitempty" toml:"interserver_http_port,omitempty"`
	// +optional
	Flash *Flash `json:"flash,omitempty" toml:"flash,omitempty"`
	// +optional
	FlashLogger *FlashLogger `json:"logger,omitempty" toml:"logger,omitempty"`
	// +optional
	// +k8s:openapi-gen=false
	FlashApplication *FlashApplication `json:"application,omitempty" toml:"application,omitempty"`
	// +optional
	// +k8s:openapi-gen=false
	FlashRaft *FlashRaft `json:"raft,omitempty" toml:"raft,omitempty"`
	// +optional
	// +k8s:openapi-gen=false
	FlashStatus *FlashStatus `json:"status,omitempty" toml:"status,omitempty"`
	// +optional
	// +k8s:openapi-gen=false
	FlashQuota *FlashQuota `json:"quotas,omitempty" toml:"quotas,omitempty"`
	// +optional
	// +k8s:openapi-gen=false
	FlashUser *FlashUser `json:"users,omitempty" toml:"users,omitempty"`
	// +optional
	// +k8s:openapi-gen=false
	FlashProfile *FlashProfile `json:"profiles,omitempty" toml:"profiles,omitempty"`
}

// FlashProfile is the configuration of [profiles] section.
// +k8s:openapi-gen=false
type FlashProfile struct {
	// +optional
	Readonly *Profile `json:"readonly,omitempty" toml:"readonly,omitempty"`
	// +optional
	Default *Profile `json:"default,omitempty" toml:"default,omitempty"`
}

// Profile is the configuration profiles.
// +k8s:openapi-gen=false
type Profile struct {
	// +optional
	Readonly *int32 `json:"readonly,omitempty" toml:"readonly,omitempty"`
	// +optional
	MaxMemoryUsage *int64 `json:"max_memory_usage,omitempty" toml:"max_memory_usage,omitempty"`
	// +optional
	UseUncompressedCache *int32 `json:"use_uncompressed_cache,omitempty" toml:"use_uncompressed_cache,omitempty"`
	// +optional
	LoadBalancing *string `json:"load_balancing,omitempty" toml:"load_balancing,omitempty"`
}

// FlashUser is the configuration of [users] section.
// +k8s:openapi-gen=false
type FlashUser struct {
	// +optional
	Readonly *User `json:"readonly,omitempty" toml:"readonly,omitempty"`
	Default  *User `json:"default,omitempty" toml:"default,omitempty"`
}

// User is the configuration of users.
// +k8s:openapi-gen=false
type User struct {
	// +optional
	Password string `json:"password,omitempty" toml:"password"`
	// +optional
	Profile string `json:"profile,omitempty" toml:"profile,omitempty"`
	// +optional
	Quota string `json:"quota,omitempty" toml:"quota,omitempty"`
	// +optional
	Networks *Networks `json:"networks,omitempty" toml:"networks,omitempty"`
}

// Networks is the configuration of [users.readonly.networks] section.
// +k8s:openapi-gen=false
type Networks struct {
	// +optional
	IP string `json:"ip,omitempty" toml:"ip,omitempty"`
}

// FlashQuota is the configuration of [quotas] section.
// +k8s:openapi-gen=false
type FlashQuota struct {
	// +optional
	Default *Quota `json:"default,omitempty" toml:"default,omitempty"`
}

// Quota is the configuration of [quotas.default] section.
// +k8s:openapi-gen=false
type Quota struct {
	// +optional
	Interval *Interval `json:"interval,omitempty" toml:"interval,omitempty"`
}

// Interval is the configuration of [quotas.default.interval] section.
// +k8s:openapi-gen=false
type Interval struct {
	// Optional: Defaults to 3600
	// +optional
	Duration *int32 `json:"duration,omitempty" toml:"duration,omitempty"`
	// Optional: Defaults to 0
	// +optional
	Queries *int32 `json:"queries,omitempty" toml:"queries,omitempty"`
	// Optional: Defaults to 0
	// +optional
	Errors *int32 `json:"errors,omitempty" toml:"errors,omitempty"`
	// Optional: Defaults to 0
	// +optional
	ResultRows *int32 `json:"result_rows,omitempty" toml:"result_rows,omitempty"`
	// Optional: Defaults to 0
	// +optional
	ReadRows *int32 `json:"read_rows,omitempty" toml:"read_rows,omitempty"`
	// Optional: Defaults to 0
	// +optional
	ExecutionTime *int32 `json:"execution_time,omitempty" toml:"execution_time,omitempty"`
}

// FlashStatus is the configuration of [status] section.
// +k8s:openapi-gen=false
type FlashStatus struct {
	// Optional: Defaults to 8234
	// +optional
	MetricsPort *int32 `json:"metrics_port,omitempty" toml:"metrics_port,omitempty"`
}

// FlashRaft is the configuration of [raft] section.
// +k8s:openapi-gen=false
type FlashRaft struct {
	// +optional
	PDAddr string `json:"pd_addr,omitempty" toml:"pd_addr,omitempty"`
	// Optional: Defaults to /data0/kvstore
	// +optional
	KVStorePath string `json:"kvstore_path,omitempty" toml:"kvstore_path,omitempty"`
	// Optional: Defaults to dt
	// +optional
	StorageEngine string `json:"storage_engine,omitempty" toml:"storage_engine,omitempty"`
}

// FlashApplication is the configuration of [application] section.
// +k8s:openapi-gen=false
type FlashApplication struct {
	// Optional: Defaults to true
	// +optional
	RunAsDaemon *bool `json:"runAsDaemon,omitempty" toml:"runAsDaemon,omitempty"`
}

// FlashLogger is the configuration of [logger] section.
// +k8s:openapi-gen=true
type FlashLogger struct {
	// Optional: Defaults to /data0/logs/error.log
	// +optional
	// +k8s:openapi-gen=false
	ErrorLog string `json:"errorlog,omitempty" toml:"errorlog,omitempty"`
	// Optional: Defaults to 100M
	// +optional
	Size string `json:"size,omitempty" toml:"size,omitempty"`
	// Optional: Defaults to /data0/logs/server.log
	// +optional
	// +k8s:openapi-gen=false
	ServerLog string `json:"log,omitempty" toml:"log,omitempty"`
	// Optional: Defaults to information
	// +optional
	Level string `json:"level,omitempty" toml:"level,omitempty"`
	// Optional: Defaults to 10
	// +optional
	Count *int32 `json:"count,omitempty" toml:"count,omitempty"`
}

// Flash is the configuration of [flash] section.
// +k8s:openapi-gen=true
type Flash struct {
	// +optional
	// +k8s:openapi-gen=false
	TiDBStatusAddr string `json:"tidb_status_addr,omitempty" toml:"tidb_status_addr,omitempty"`
	// +optional
	// +k8s:openapi-gen=false
	ServiceAddr string `json:"service_addr,omitempty" toml:"service_addr,omitempty"`
	// Optional: Defaults to 0.6
	// +optional
	OverlapThreshold *float64 `json:"overlap_threshold,omitempty" toml:"overlap_threshold,omitempty"`
	// Optional: Defaults to 200
	// +optional
	CompactLogMinPeriod *int32 `json:"compact_log_min_period,omitempty" toml:"compact_log_min_period,omitempty"`
	// +optional
	FlashCluster *FlashCluster `json:"flash_cluster,omitempty" toml:"flash_cluster,omitempty"`
	// +optional
	// +k8s:openapi-gen=false
	FlashProxy *FlashProxy `json:"proxy,omitempty" toml:"proxy,omitempty"`
}

// FlashCluster is the configuration of [flash.flash_cluster] section.
// +k8s:openapi-gen=true
type FlashCluster struct {
	// Optional: Defaults to /tiflash/flash_cluster_manager
	// +optional
	// +k8s:openapi-gen=false
	ClusterManagerPath string `json:"cluster_manager_path,omitempty" toml:"cluster_manager_path,omitempty"`
	// Optional: Defaults to /data0/logs/flash_cluster_manager.log
	// +optional
	// +k8s:openapi-gen=false
	ClusterLog string `json:"log,omitempty" toml:"log,omitempty"`
	// Optional: Defaults to 20
	// +optional
	RefreshInterval *int32 `json:"refresh_interval,omitempty" toml:"refresh_interval,omitempty"`
	// Optional: Defaults to 10
	// +optional
	UpdateRuleInterval *int32 `json:"update_rule_interval,omitempty" toml:"update_rule_interval,omitempty"`
	// Optional: Defaults to 60
	// +optional
	MasterTTL *int32 `json:"master_ttl,omitempty" toml:"master_ttl,omitempty"`
}

// FlashProxy is the configuration of [flash.proxy] section.
// +k8s:openapi-gen=false
type FlashProxy struct {
	// Optional: Defaults to 0.0.0.0:20170
	// +optional
	Addr string `json:"addr,omitempty" toml:"addr,omitempty"`
	// +optional
	AdvertiseAddr string `json:"advertise-addr,omitempty" toml:"advertise-addr,omitempty"`
	// Optional: Defaults to /data0/proxy
	// +optional
	DataDir string `json:"data-dir,omitempty" toml:"data-dir,omitempty"`
	// Optional: Defaults to /data0/proxy.toml
	// +optional
	Config string `json:"config,omitempty" toml:"config,omitempty"`
	// Optional: Defaults to /data0/logs/proxy.log
	// +optional
	LogFile string `json:"log-file,omitempty" toml:"log-file,omitempty"`
}
