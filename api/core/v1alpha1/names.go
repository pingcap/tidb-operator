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

package v1alpha1

import meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"

// All volume names
const (
	// VolumeNameConfig defines volume name for main config file
	VolumeNameConfig = meta.NamePrefix + "config"
	// VolumeNamePrestopChecker defines volume name for pre stop checker cmd
	VolumeNamePrestopChecker = meta.NamePrefix + "prestop-checker"
	// VolumeNameBootstrapSQL is the volume name for bootstrap sql
	VolumeNameBootstrapSQL = meta.NamePrefix + "tidb-bootstrap-sql"
	// VolumeNameTiDBAuthToken is the volume name for tidb auth token
	VolumeNameTiDBAuthToken = meta.NamePrefix + "tidb-auth-token"
	// VolumeNameTiDBSlowLogDefault defines default volume name of tidb slowlog
	// Users may claim another volume for tidb slowlog and the default one will be removed
	VolumeNameTiDBSlowLogDefault = meta.NamePrefix + "slowlog"

	// TLS
	//
	// VolumeNameClusterTLS defines volume name for the TLS secret used between components in the TiDB cluster
	VolumeNameClusterTLS = meta.NamePrefix + "tls"
	// VolumeNameClusterClientTLS defines volume name for any one-time job accessing the TiDB cluster components
	VolumeNameClusterClientTLS = meta.NamePrefix + "cluster-client-tls"
	// VolumeNameMySQLTLS is the volume name for the TLS secret used by TLS communication between TiDB server and MySQL client.
	VolumeNameMySQLTLS = meta.NamePrefix + "tidb-sql-tls"
	// VolumeNameTiProxyMySQLTLS is the volume name for the TLS secret used by TLS communication between TiProxy and MySQL client.
	VolumeNameTiProxyMySQLTLS = meta.NamePrefix + "tiproxy-sql-tls"
	// VolumeNameTiProxyHTTPTLS is the volume name for the TLS secret used by TLS communication between TiProxy HTTP server and HTTP client.
	VolumeNameTiProxyHTTPTLS = meta.NamePrefix + "tiproxy-http-tls"
	// VolumeNameTiProxyTiDBTLS is the volume name for the TLS secret used by TLS communication between TiProxy and TiDB server.
	VolumeNameTiProxyTiDBTLS = meta.NamePrefix + "tiproxy-tidb-tls"
)

// All container names
const (
	// Main component containers of the tidb cluster
	// These names are well known so the name prefix is not added.
	ContainerNamePD        = "pd"
	ContainerNameTiKV      = "tikv"
	ContainerNameTiDB      = "tidb"
	ContainerNameTiFlash   = "tiflash"
	ContainerNameTiCDC     = "ticdc"
	ContainerNameTSO       = "tso"
	ContainerNameScheduler = "scheduler"
	ContainerNameTiProxy   = "tiproxy"

	// An init container to copy pre stop checker cmd to main container
	ContainerNamePrestopChecker = meta.NamePrefix + "prestop-checker"

	// TiDB
	//
	// Container to redirect slowlog
	ContainerNameTiDBSlowLog = meta.NamePrefix + "slowlog"

	// TiFlash
	//
	// Container to redirect server log
	ContainerNameTiFlashServerLog = meta.NamePrefix + "serverlog"
	// Container to redirect error log
	ContainerNameTiFlashErrorLog = meta.NamePrefix + "errorlog"
)

// All well known dir path
const (
	// config dir path
	DirPathConfigPD        = "/etc/pd"
	DirPathConfigTiKV      = "/etc/tikv"
	DirPathConfigTiDB      = "/etc/tidb"
	DirPathConfigTiFlash   = "/etc/tiflash"
	DirPathConfigTiCDC     = "/etc/ticdc"
	DirPathConfigTSO       = "/etc/tso"
	DirPathConfigScheduler = "/etc/scheduler"
	DirPathConfigTiProxy   = "/etc/tiproxy"

	// DirPathPrestop defines dir path of pre stop checker cmd
	DirPathPrestop = "/prestop"
	// Dir path of bootstrap sql
	DirPathBootstrapSQL = "/etc/tidb-bootstrap"
	// Dir path of tidb auth token
	DirPathTiDBAuthToken = "/var/lib/tidb-auth-token" // #nosec
	// Default dir path of tidb slowlog
	DirPathTiDBSlowLogDefault = "/var/log/tidb"

	// TLS
	//
	// Dir path of cluster tls file
	DirPathClusterTLSPD        = "/var/lib/pd-tls"
	DirPathClusterTLSTiKV      = "/var/lib/tikv-tls"
	DirPathClusterTLSTiDB      = "/var/lib/tidb-tls"
	DirPathClusterTLSTiFlash   = "/var/lib/tiflash-tls"
	DirPathClusterClientTLS    = "/var/lib/cluster-client-tls"
	DirPathClusterTLSTiCDC     = "/var/lib/ticdc-tls"
	DirPathClusterTLSTSO       = "/var/lib/tso-tls"
	DirPathClusterTLSScheduler = "/var/lib/scheduler-tls"
	DirPathClusterTLSTiProxy   = "/var/lib/tiproxy-tls"
	// DirPathMySQLTLS is the dir path of tls file for tidb and mysql client
	DirPathMySQLTLS = "/var/lib/tidb-sql-tls"
	// DirPathTiProxyMySQLTLS is the dir path of tls file for tiproxy and mysql client
	DirPathTiProxyMySQLTLS = "/var/lib/tiproxy-sql-tls"
	// DirPathTiProxyHTTPTLS is the dir path of tls file for tiproxy http server
	DirPathTiProxyHTTPTLS = "/var/lib/tiproxy-http-tls"
	// DirPathTiProxyTiDBTLS is the dir path of tls file for tiproxy and tidb
	DirPathTiProxyTiDBTLS = "/var/lib/tiproxy-tidb-tls"
)

// All file names
const (
	// FileNameConfig defines default name of config file
	FileNameConfig = "config.toml"
	// FileNameConfigTiFlashProxy defines default name of tiflash proxy config file
	FileNameConfigTiFlashProxy = "proxy.toml"
	// FileNameBootstrapSQL defines default file name of bootstrap sql
	FileNameBootstrapSQL = "bootstrap.sql"
	// FileNameTiDBAuthTokenJWKS defines default file name of auth token jwks
	FileNameTiDBAuthTokenJWKS = "tidb_auth_token_jwks.json" // #nosec
	// FileNameTiDBSlowLog defines default file name of tidb slowlog
	FileNameTiDBSlowLog = "slowlog"
)

// All config map keys
const (
	// Keep compatible with the bootstrap sql in v1
	ConfigMapKeyBootstrapSQL = "bootstrap-sql"
)
