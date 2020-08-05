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

// Maintain a copy of MasterConfig to make it more friendly with the kubernetes API:
//
//  - add 'omitempty' json and toml tag to avoid passing the empty value of primitive types to dm-master-server, e.g. 0 of int
//  - change all numeric type to pointer, e.g. uint -> *uint, so that 'omitempty' could work properly
//  - add openapi-gen tags so that the kubernetes-style OpenAPI schema could be properly generated
//  - make the whole config struct deepcopy friendly
//
// only adding field is allowed, DO NOT change existing field definition or remove field.
// some fields maybe illegal for certain version of DM, this should be addressed in the ValidationWebhook.

// initially copied from dm-master v2.0.0-rc

// MasterConfig is the configuration of dm-master-server
// +k8s:openapi-gen=true
type MasterConfig struct {
	// Log level.
	// Optional: Defaults to info
	// +optional
	LogLevel *string `toml:"log-level" json:"log-level"`
	// File log config.
	// +optional
	LogFile *string `toml:"log-file" json:"log-file"`
	// Log format. one of json or text.
	// +optional
	LogFormat *string `toml:"log-format" json:"log-format"`

	// RPC timeout when dm-master request to dm-worker
	// Optional: Defaults to 30s
	// +optional
	RPCTimeoutStr *string `toml:"rpc-timeout" json:"rpc-timeout"`
	// RPC agent rate limit when dm-master request to dm-worker
	// Optional: Defaults to 10
	// +optional
	RPCRateLimit *float64 `toml:"rpc-rate-limit" json:"rpc-rate-limit"`
	// RPC agent rate burst when dm-master request to dm-worker
	// Optional: Defaults to 40
	// +optional
	RPCRateBurst *int `toml:"rpc-rate-burst" json:"rpc-rate-burst"`
	// dm-master's security config
	// +optional
	// +k8s:openapi-gen=false
	DMSecurityConfig
}

// WorkerConfig is the configuration of dm-worker-server
// +k8s:openapi-gen=true
type WorkerConfig struct {
	// Log level.
	// Optional: Defaults to info
	// +optional
	LogLevel *string `toml:"log-level" json:"log-level"`
	// File log config.
	// +optional
	LogFile *string `toml:"log-file" json:"log-file"`
	// Log format. one of json or text.
	// +optional
	LogFormat *string `toml:"log-format" json:"log-format"`

	// KeepAliveTTL is the keepalive ttl dm-worker write to dm-master embed etcd
	// Optional: Defaults to 10
	// +optional
	KeepAliveTTL *int64 `toml:"keepalive-ttl" json:"keepalive-ttl"`
	// dm-worker's security config
	// +optional
	DMSecurityConfig
}

// DM common security config
type DMSecurityConfig struct {
	// SSLCA is the path of file that contains list of trusted SSL CAs. if set, following four settings shouldn't be empty
	// +optional
	SSLCA *string `toml:"ssl-ca" json:"ssl-ca" yaml:"ssl-ca"`
	// SSLCert is the path of file that contains X509 certificate in PEM format.
	// +optional
	SSLCert *string `toml:"ssl-cert" json:"ssl-cert" yaml:"ssl-cert"`
	// SSLKey is the path of file that contains X509 key in PEM format.
	// +optional
	SSLKey *string `toml:"ssl-key" json:"ssl-key" yaml:"ssl-key"`
	// CertAllowedCN is the Common Name that allowed
	// +optional
	// +k8s:openapi-gen=false
	CertAllowedCN []string `toml:"cert-allowed-cn" json:"cert-allowed-cn" yaml:"cert-allowed-cn"`
}
