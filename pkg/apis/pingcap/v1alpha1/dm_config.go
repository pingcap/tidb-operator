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
	LogLevel *string `toml:"log-level,omitempty" json:"log-level,omitempty"`
	// File log config.
	// +optional
	LogFile *string `toml:"log-file,omitempty" json:"log-file,omitempty"`
	// Log format. one of json or text.
	// +optional
	LogFormat *string `toml:"log-format,omitempty" json:"log-format,omitempty"`

	// RPC timeout when dm-master request to dm-worker
	// Optional: Defaults to 30s
	// +optional
	RPCTimeoutStr *string `toml:"rpc-timeout,omitempty" json:"rpc-timeout,omitempty"`
	// RPC agent rate limit when dm-master request to dm-worker
	// Optional: Defaults to 10
	// +optional
	RPCRateLimit *float64 `toml:"rpc-rate-limit,omitempty" json:"rpc-rate-limit,omitempty"`
	// RPC agent rate burst when dm-master request to dm-worker
	// Optional: Defaults to 40
	// +optional
	RPCRateBurst *int `toml:"rpc-rate-burst,omitempty" json:"rpc-rate-burst,omitempty"`
	// dm-master's security config
	// +optional
	// +k8s:openapi-gen=false
	DMSecurityConfig `toml:",inline" json:",inline"`
}

// WorkerConfig is the configuration of dm-worker-server
// +k8s:openapi-gen=true
type WorkerConfig struct {
	// Log level.
	// Optional: Defaults to info
	// +optional
	LogLevel *string `toml:"log-level,omitempty" json:"log-level,omitempty"`
	// File log config.
	// +optional
	LogFile *string `toml:"log-file,omitempty" json:"log-file,omitempty"`
	// Log format. one of json or text.
	// +optional
	LogFormat *string `toml:"log-format,omitempty" json:"log-format,omitempty"`

	// KeepAliveTTL is the keepalive ttl dm-worker write to dm-master embed etcd
	// Optional: Defaults to 10
	// +optional
	KeepAliveTTL *int64 `toml:"keepalive-ttl,omitempty" json:"keepalive-ttl,omitempty"`
	// dm-worker's security config
	// +optional
	DMSecurityConfig `toml:",inline" json:",inline"`
}

// DM common security config
type DMSecurityConfig struct {
	// SSLCA is the path of file that contains list of trusted SSL CAs. if set, following four settings shouldn't be empty
	// +optional
	SSLCA *string `toml:"ssl-ca,omitempty" json:"ssl-ca,omitempty" yaml:"ssl-ca,omitempty"`
	// SSLCert is the path of file that contains X509 certificate in PEM format.
	// +optional
	SSLCert *string `toml:"ssl-cert,omitempty" json:"ssl-cert,omitempty" yaml:"ssl-cert,omitempty"`
	// SSLKey is the path of file that contains X509 key in PEM format.
	// +optional
	SSLKey *string `toml:"ssl-key,omitempty" json:"ssl-key,omitempty" yaml:"ssl-key,omitempty"`
	// CertAllowedCN is the Common Name that allowed
	// +optional
	CertAllowedCN []string `toml:"cert-allowed-cn,omitempty" json:"cert-allowed-cn,omitempty" yaml:"cert-allowed-cn,omitempty"`
}
