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

// Feature defines a supported feature of a tidb cluster.
// NOTE(liubo02): +enum is not supported now, we have to add all enum into comments
// NOTE(liubo02): It's supported by https://github.com/kubernetes-sigs/controller-tools/pull/1179
//
// +kubebuilder:validation:Enum=FeatureModification;VolumeAttributesClass;DisablePDDefaultReadinessProbe;UsePDReadyAPI;SessionTokenSigning;ClusterSubdomain;TerminableLogTailer;UseTSOReadyAPI;UseTiKVReadyAPI
// +enum
type Feature string

type FeatureStage string

const (
	FeatureStageAlpha      FeatureStage = "ALPHA"
	FeatureStageBeta       FeatureStage = "BETA"
	FeatureStageStable     FeatureStage = "STABLE"
	FeatureStageDeprecated FeatureStage = "DEPRECATED"
)

type FeatureGate struct {
	Name Feature `json:"name"`
}

type FeatureGateStatus struct {
	FeatureGate `json:",inline"`
	Stage       FeatureStage `json:"stage"`
}

const (
	// Support modify feature after cluster creation
	// Now enable/disable any features will update groups by rolling update
	// This feature cannot be disabled
	FeatureModification      Feature      = "FeatureModification"
	FeatureModificationStage FeatureStage = FeatureStageAlpha

	// Support modify volume by VolumeAttributesClass
	VolumeAttributesClass      Feature      = "VolumeAttributesClass"
	VolumeAttributesClassStage FeatureStage = FeatureStageAlpha

	// Disable PD's default readiness probe
	// Now the pd's default readiness probe use TCP to probe client port
	// It's not useful and will print so many warn logs in PD's stdout/stderr
	DisablePDDefaultReadinessProbe      Feature      = "DisablePDDefaultReadinessProbe"
	DisablePDDefaultReadinessProbeStage FeatureStage = FeatureStageAlpha

	// UsePDReadyAPI means use PD's /ready API as the readiness probe.
	// It requires PD v8.5.2 or later.
	UsePDReadyAPI      Feature      = "UsePDReadyAPI"
	UsePDReadyAPIStage FeatureStage = FeatureStageAlpha

	// SessionTokenSigning means tidb operator will always set the two tiproxy related configs for tidb:
	// - `session-token-signing-cert`
	// - `session-token-signing-key`
	// Regardless of whether tiproxy is enabled.
	// You must enable this feature if you are using tiproxy.
	// See: https://docs.pingcap.com/tidb/stable/tidb-configuration-file/#session-token-signing-cert-new-in-v640
	// By default, tidb operator will use the cluster TLS cert as the session token signing cert and key.
	// If different TiDBGroups use different cluster TLS certs, or you want to use custom TLS certs for session token signing,
	// you can specify it via `cluster.spec.security.sessionTokenSigningCertKeyPair`, with this feature enabled.
	SessionTokenSigning      Feature      = "SessionTokenSigning"
	SessionTokenSigningStage FeatureStage = FeatureStageAlpha

	// If this feature is enabled, all instances will use a same headless svc as their subdomain
	ClusterSubdomain      Feature      = "ClusterSubdomain"
	ClusterSubdomainStage FeatureStage = FeatureStageAlpha

	// If this feature is enabled, log tailer in sidecar can exit immediately after main container is exited
	TerminableLogTailer      Feature      = "TerminableLogTailer"
	TerminableLogTailerStage FeatureStage = FeatureStageAlpha

	// UseTSOReadyAPI calls /health api to check readiness for tso pods
	UseTSOReadyAPI      Feature      = "UseTSOReadyAPI"
	UseTSOReadyAPIStage FeatureStage = FeatureStageAlpha

	// UseTiKVReadyAPI means use TiKV's /ready API as the readiness probe.
	UseTiKVReadyAPI      Feature      = "UseTiKVReadyAPI"
	UseTiKVReadyAPIStage FeatureStage = FeatureStageAlpha
)
