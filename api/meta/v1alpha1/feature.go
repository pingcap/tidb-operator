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

// This file defines feature constants and their append-only histories for feature-gen.
// The generated definitions in zz_generated.features.go are used at runtime.
package v1alpha1

const (
	// Support modify feature after cluster creation
	// Now enable/disable any features will update groups by rolling update
	// This feature cannot be disabled
	//
	// +feature:unreloadable=
	// +feature:log=rev=0,stage=ALPHA,default=false
	_FeatureModification = iota

	// Support modify volume by VolumeAttributesClass
	//
	// +feature:unreloadable=
	// +feature:log=rev=0,stage=ALPHA,default=false
	_VolumeAttributesClass

	// Disable PD's default readiness probe
	// Now the pd's default readiness probe use TCP to probe client port
	// It's not useful and will print so many warn logs in PD's stdout/stderr
	//
	// +feature:unreloadable=pd
	// +feature:log=rev=0,stage=ALPHA,default=false
	_DisablePDDefaultReadinessProbe

	// Deprecated: use UsePDReadyAPIV2
	// UsePDReadyAPI means use PD's /ready API as the readiness probe.
	// It requires PD v8.5.2 or later.
	//
	// +feature:unreloadable=pd
	// +feature:log=rev=0,stage=ALPHA,default=false
	_UsePDReadyAPI

	// SessionTokenSigning means tidb operator will always set the two tiproxy related configs for tidb:
	// - `session-token-signing-cert`
	// - `session-token-signing-key`
	// Regardless of whether tiproxy is enabled.
	// You must enable this feature if you are using tiproxy.
	// See: https://docs.pingcap.com/tidb/stable/tidb-configuration-file/#session-token-signing-cert-new-in-v640
	// By default, tidb operator will use the cluster TLS cert as the session token signing cert and key.
	// If different TiDBGroups use different cluster TLS certs, or you want to use custom TLS certs for session token signing,
	// you can specify it via `cluster.spec.security.sessionTokenSigningCertKeyPair`, with this feature enabled.
	//
	// +feature:unreloadable=tidb
	// +feature:log=rev=0,stage=ALPHA,default=false
	_SessionTokenSigning

	// If this feature is enabled, all instances will use a same headless svc as their subdomain
	//
	// +feature:unreloadable=*
	// +feature:log=rev=0,stage=ALPHA,default=false
	_ClusterSubdomain

	// If this feature is enabled, log tailer in sidecar can exit immediately after main container is exited
	//
	// +feature:unreloadable=tidb,tiflash
	// +feature:log=rev=0,stage=ALPHA,default=false
	_TerminableLogTailer

	// UseTSOReadyAPI calls /health api to check readiness for tso pods
	//
	// +feature:unreloadable=tso
	// +feature:log=rev=0,stage=ALPHA,default=false
	_UseTSOReadyAPI

	// UseSchedulingReadyAPI calls /health api to check readiness for scheduling pods.
	//
	// +feature:unreloadable=scheduling
	// +feature:log=rev=0,stage=ALPHA,default=false
	_UseSchedulingReadyAPI

	// UseTiKVReadyAPI means use TiKV's /ready API as the readiness probe.
	//
	// +feature:unreloadable=tikv
	// +feature:log=rev=0,stage=ALPHA,default=false
	_UseTiKVReadyAPI

	// UsePDReadyAPIV2 means use PD's /readyz API as the readiness probe.
	//
	// +feature:unreloadable=pd
	// +feature:log=rev=0,stage=ALPHA,default=false
	_UsePDReadyAPIV2

	// UseTiFlashReadyAPI means use TiFlash's /readyz API as the readiness probe.
	//
	// +feature:unreloadable=tiflash
	// +feature:log=rev=0,stage=ALPHA,default=false
	_UseTiFlashReadyAPI

	// If this feature is enabled
	// - More than one pd group can be created for one cluster.
	// - A new PD service will be created for all PDGroups.
	// - The default internal pd svc of the PDGroup will be not created.
	// - Cannot customize advertised client port
	//
	// +feature:unreloadable=
	// +feature:log=rev=0,stage=ALPHA,default=false
	_MultiPDGroup

	// If this feature is enabled, TiCDC pods can dynamically load secrets with specific labels into pods
	//
	// +feature:unreloadable=ticdc
	// +feature:log=rev=0,stage=ALPHA,default=false
	_TiCDCDynamicSecretSyncer

	// By default, kvengine.remote-worker-addr follows the coprocessor ref.
	// If this feature is enabled, kvengine.remote-worker-addr will use the default worker ref.
	//
	// +feature:unreloadable=tikv
	// +feature:log=rev=0,stage=ALPHA,default=false
	_IndependentKVEngineWorker
)
