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

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

const (
	// VolumeUsageTypeTiDBSlowLog means the data dir of slowlog
	// Users can define a persistent volume for slowlog, or an emptydir will be used.
	VolumeMountTypeTiDBSlowLog VolumeMountType = "slowlog"

	VolumeMountTiDBSlowLogDefaultPath = "/var/log/tidb"
)

const (
	TiDBPortNameClient    = "mysql-client"
	TiDBPortNameStatus    = "status"
	DefaultTiDBPortClient = 4000
	DefaultTiDBPortStatus = 10080

	// DefaultTiDBMinReadySeconds is default min ready seconds of tidb
	DefaultTiDBMinReadySeconds = 10
)

const (
	// TCPProbeType represents the readiness prob method with TCP.
	TCPProbeType string = "tcp"
	// CommandProbeType represents the readiness prob method with arbitrary unix `exec` call format commands.
	CommandProbeType string = "command"
)

const (
	TiDBGroupCondAvailable   = "Available"
	TiDBGroupAvailableReason = "TiDBGroupAvailable"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// TiDBGroupList defines a list of TiDB groups
type TiDBGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TiDBGroup `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:resource:categories=group,shortName=dbg
// +kubebuilder:selectablefield:JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Desired",type=string,JSONPath=`.spec.replicas`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Updated",type=string,JSONPath=`.status.updatedReplicas`
// +kubebuilder:printcolumn:name="UpdateRevision",type=string,JSONPath=`.status.updateRevision`
// +kubebuilder:printcolumn:name="CurrentRevision",type=string,JSONPath=`.status.currentRevision`
// +kubebuilder:printcolumn:name="Synced",type=string,JSONPath=`.status.conditions[?(@.type=="Synced")].status`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TiDBGroup defines a group of similar TiDB instances.
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 40",message="name must not exceed 40 characters"
type TiDBGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiDBGroupSpec   `json:"spec,omitempty"`
	Status TiDBGroupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// TiDBList defines a list of TiDB instances.
type TiDBList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TiDB `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=instance
// +kubebuilder:selectablefield:JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Synced",type=string,JSONPath=`.status.conditions[?(@.type=="Synced")].status`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TiDB defines a TiDB instance.
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 47",message="name must not exceed 47 characters"
type TiDB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiDBSpec   `json:"spec,omitempty"`
	Status TiDBStatus `json:"status,omitempty"`
}

// TiDBGroupSpec describes the common attributes of a TiDBGroup.
type TiDBGroupSpec struct {
	Cluster ClusterReference `json:"cluster"`

	// Features are enabled feature
	Features []meta.Feature `json:"features,omitempty"`

	// +kubebuilder:validation:Minimum=0
	Replicas *int32 `json:"replicas"`

	// +listType=map
	// +listMapKey=type
	SchedulePolicies []SchedulePolicy `json:"schedulePolicies,omitempty"`

	// MinReadySeconds specifies the minimum number of seconds for which a newly created pod be ready without any of its containers crashing, for it to be considered available.
	// +kubebuilder:validation:Minimum=0
	// +optional
	MinReadySeconds *int64 `json:"minReadySeconds,omitempty"`

	// Template is the instance template
	Template TiDBTemplate `json:"template"`
}

type TiDBTemplate struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       TiDBTemplateSpec `json:"spec"`
}

// TiDBTemplateSpec can only be specified in TiDBGroup.
// +kubebuilder:validation:XValidation:rule="!has(self.overlay) || !has(self.overlay.volumeClaims) || (has(self.volumes) && self.overlay.volumeClaims.all(vc, vc.name in self.volumes.map(v, v.name)))",message="overlay volumeClaims names must exist in volumes"
type TiDBTemplateSpec struct {
	// Version must be a semantic version.
	// It can has a v prefix or not.
	// +kubebuilder:validation:Pattern=`^(v)?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`
	Version string `json:"version"`

	// Image is tidb's image
	// If tag is omitted, version will be used as the image tag.
	// Default is pingcap/tidb
	Image *string `json:"image,omitempty"`

	// Server defines the server configuration of TiDB.
	Server TiDBServer `json:"server,omitempty"`

	// Probes defines probes for TiDB.
	Probes TiDBProbes `json:"probes,omitempty"`

	// Resources defines resource required by TiDB.
	Resources ResourceRequirements `json:"resources,omitempty"`

	// Config defines config file of TiDB.
	Config ConfigFile `json:"config,omitempty"`

	// UpdateStrategy defines strategy when some fields are updated
	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`

	// Security defines security configs of TiDB
	Security *TiDBSecurity `json:"security,omitempty"`

	// Volumes defines data volume of TiDB, it is optional.
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=256
	Volumes []Volume `json:"volumes,omitempty"`

	// Keyspace defines the keyspace name of a TiDB instance
	// Keyspace can only be set when mode is changed from StandBy to Normal
	// If mode is Normal, it cannot be changed again
	// For classic tidb, keyspace name is not supported and should not be set
	Keyspace string `json:"keyspace,omitempty"`

	// Mode defines the mode of a TiDB instance
	// +kubebuilder:default="Normal"
	Mode TiDBMode `json:"mode,omitempty"`

	// SlowLog defines the separate slow log configuration for TiDB.
	// When enabled, a sidecar container will be created to output the slow log to its stdout.
	SlowLog *TiDBSlowLog `json:"slowLog,omitempty"`

	// PreStop defines the preStop config for the tidb container.
	PreStop *TiDBPreStop `json:"preStop,omitempty"`

	// Overlay defines a k8s native resource template patch.
	// All resources(pod, pvcs, ...) managed by TiDB can be overlayed by this field.
	Overlay *Overlay `json:"overlay,omitempty"`
}

type TiDBPreStop struct {
	// SleepSeconds is the seconds to sleep before sending the SIGTERM to the tidb container.
	// It's useful to achieve a graceful shutdown of the tidb container.
	// Operator will calculate the tidb pod's `terminationGracePeriod` based on this field:
	// `terminationGracePeriod` = `preStopHookSleepSeconds` + 15(gracefulCloseConnectionsTimeout) + 5(buffer)
	// Default is 10 seconds.
	SleepSeconds int32 `json:"sleepSeconds,omitempty"`
}

type TiDBSecurity struct {
	// TLS defines the tls configs of TiDB
	TLS *TiDBTLSConfig `json:"tls,omitempty"`

	// Whether enable `tidb_auth_token` authentication method.
	// To enable this feature, a secret named `<groupName>-tidb-auth-token-jwks-secret` must be created to store the JWKs.
	// ref: https://docs.pingcap.com/tidb/stable/security-compatibility-with-mysql#tidb_auth_token
	// Defaults to false.
	AuthToken *TiDBAuthToken `json:"authToken,omitempty"`

	// SEM defines the security enhancement mode config of TiDB
	// If this field is set, security.enable-sem = true will be set automatically
	// Users still can disable sem in config file explicitly
	SEM *SEM `json:"sem,omitempty"`
}

type SEM struct {
	// Config defines the configmap reference of the sem config
	// By default, we use the 'sem.json' as the file key of configmap
	Config corev1.LocalObjectReference `json:"config"`
}

type TiDBServer struct {
	// Port defines all ports listened by TiDB.
	Ports TiDBPorts `json:"ports,omitempty"`

	// Labels defines the server labels of the TiDB server.
	// Operator will set these `labels` by API.
	// If a label in this field is conflict with the config file, this field takes precedence.
	// NOTE: Different from other components, TiDB will replace all labels but not only add/update.
	// NOTE: these label keys are managed by TiDB Operator, it will be set automatically and you can not modify them:
	//  - host
	//  - region
	//  - zone
	// +kubebuilder:validation:XValidation:rule="!('host' in self) && !('region' in self) && !('zone' in self)",message="labels cannot contain 'host', 'region', or 'zone' keys"
	Labels map[string]string `json:"labels,omitempty"`
}

type TiDBPorts struct {
	// Client defines port for TiDB's SQL service.
	Client *Port `json:"client,omitempty"`
	// Status defines port for TiDB status API.
	Status *Port `json:"status,omitempty"`
}

type TiDBProbes struct {
	// Readiness defines the readiness probe for TiDB.
	// The default handler is a TCP socket on the client port.
	Readiness *TiDBProb `json:"readiness,omitempty"`
}

type TiDBProb struct {
	// "tcp" will use TCP socket to connect component port.
	// "command" will probe the status api of tidb.
	// +kubebuilder:validation:Enum=tcp;command
	Type *string `json:"type,omitempty"`
}

type TiDBSlowLog struct {
	// Image to tail slowlog to stdout
	// Default is busybox:1.37.0
	Image *string `json:"image,omitempty"`

	// ResourceRequirements defines the resource requirements for the slow log sidecar.
	Resources ResourceRequirements `json:"resources,omitempty"`
}

type TiDBTLSConfig struct {
	// When enabled, TiDB will accept TLS encrypted connections from MySQL clients.
	// The steps to enable this feature:
	//   1. Generate a TiDB server-side certificate and a client-side certifiacete for the TiDB cluster.
	//      There are multiple ways to generate certificates:
	//        - user-provided certificates: https://docs.pingcap.com/tidb/stable/generate-self-signed-certificates
	//        - use the K8s built-in certificate signing system signed certificates: https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/
	//        - or use cert-manager signed certificates: https://cert-manager.io/
	//   2. Create a K8s Secret object which contains the TiDB server-side certificate created above.
	//      The name of this Secret must be: <groupName>-tidb-server-secret.
	//        kubectl create secret generic <groupName>-tidb-server-secret --namespace=<namespace> --from-file=tls.crt=<path/to/tls.crt> --from-file=tls.key=<path/to/tls.key> --from-file=ca.crt=<path/to/ca.crt>
	//   3. Create a K8s Secret object which contains the TiDB client-side certificate created above which will be used by TiDB Operator.
	//      The name of this Secret must be: <groupName>-tidb-client-secret.
	//        kubectl create secret generic <groupName>-tidb-client-secret --namespace=<namespace> --from-file=tls.crt=<path/to/tls.crt> --from-file=tls.key=<path/to/tls.key> --from-file=ca.crt=<path/to/ca.crt>
	//   4. Set Enabled to `true`.
	// +kubebuilder:validation:XValidation:rule="oldSelf == null || self.enabled == oldSelf.enabled",message="field .mysql.enabled is immutable"
	MySQL *TLS `json:"mysql,omitempty"`

	// ComponentTLSConfig is tls config to access internal components
	ComponentTLSConfig `json:",inline"`
}

type TiDBAuthToken struct {
	// Secret name of jwks
	JWKs corev1.LocalObjectReference `json:"jwks"`
}

type TiDBGroupStatus struct {
	CommonStatus `json:",inline"`
	GroupStatus  `json:",inline"`
}

// TiDBSpec is the spec of a TiDB instance
// +kubebuilder:validation:XValidation:rule="(!has(oldSelf.topology) && !has(self.topology)) || (has(oldSelf.topology) && has(self.topology))",fieldPath=".topology",message="topology can only be set when creating"
type TiDBSpec struct {
	Cluster ClusterReference `json:"cluster"`

	// Features are enabled feature
	Features []meta.Feature `json:"features,omitempty"`

	// Topology defines the topology domain of this TiDB instance.
	// It will be translated into a node affnity config.
	// Topology cannot be changed.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="topology is immutable"
	Topology Topology `json:"topology,omitempty"`

	// Subdomain means the subdomain of the exported TiDB dns.
	// A same TiDB cluster will use a same subdomain
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="subdomain is immutable"
	Subdomain string `json:"subdomain"`

	// TiDBTemplateSpec embedded some fields managed by TiDBGroup.
	// +kubebuilder:validation:XValidation:rule="!has(self.mode) || self.mode == 'Normal' || !has(self.keyspace) || self.keyspace.size() == 0",message="keyspace cannot be set if mode is StandBy"
	TiDBTemplateSpec `json:",inline"`
}

// TiDBMode defines the mode of a TiDB instance
// +kubebuilder:validation:Enum=Normal;StandBy
type TiDBMode string

const (
	// TiDBModeNormal means the tidb is in the default normal mode
	TiDBModeNormal TiDBMode = "Normal"
	// TiDBModeStandBy means the tidb is waiting for activating
	TiDBModeStandBy TiDBMode = "StandBy"
)

type TiDBStatus struct {
	CommonStatus `json:",inline"`
}
