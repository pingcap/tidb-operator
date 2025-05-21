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
// +kubebuilder:resource:categories=tc;group,shortName=dbg
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
// +kubebuilder:resource:categories=tc;instance
// +kubebuilder:selectablefield:JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Synced",type=string,JSONPath=`.status.conditions[?(@.type=="Synced")].status`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TiDB defines a TiDB instance.
type TiDB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiDBSpec   `json:"spec,omitempty"`
	Status TiDBStatus `json:"status,omitempty"`
}

// TiDBGroupSpec describes the common attributes of a TiDBGroup.
type TiDBGroupSpec struct {
	Cluster  ClusterReference `json:"cluster"`
	Replicas *int32           `json:"replicas"`

	// Service defines some fields used to override the default service.
	Service *TiDBService `json:"service,omitempty"`

	// +listType=map
	// +listMapKey=type
	SchedulePolicies []SchedulePolicy `json:"schedulePolicies,omitempty"`

	Template TiDBTemplate `json:"template"`
}

type TiDBTemplate struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       TiDBTemplateSpec `json:"spec"`
}

// TiDBTemplateSpec can only be specified in TiDBGroup.
type TiDBTemplateSpec struct {
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
	Config         ConfigFile     `json:"config,omitempty"`
	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`

	Security *TiDBSecurity `json:"security,omitempty"`
	// Volumes defines data volume of TiDB, it is optional.
	// +listType=map
	// +listMapKey=name
	Volumes []Volume `json:"volumes,omitempty"`

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
	// Whether enable the TLS connection between the TiDB server and MySQL client.
	// TODO(liubo02): rename the TiDBTLSClient struct,
	TLS *TiDBTLS `json:"tls,omitempty"`

	// Whether enable `tidb_auth_token` authentication method.
	// To enable this feature, a K8s secret named `<groupName>-tidb-auth-token-jwks-secret` must be created to store the JWKs.
	// ref: https://docs.pingcap.com/tidb/stable/security-compatibility-with-mysql#tidb_auth_token
	// Defaults to false.
	AuthToken *TiDBAuthToken `json:"authToken,omitempty"`
}

type TiDBServer struct {
	// Port defines all ports listened by TiDB.
	Ports TiDBPorts `json:"ports,omitempty"`

	// Labels defines the server labels of the TiDB server.
	// TiDB Operator will ignore `labels` in TiDB's config file and use this field instead.
	// Note these label keys are managed by TiDB Operator, it will be set automatically and you can not modify them:
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

// TiDBService defines some fields used to override the default service.
type TiDBService struct {
	// type determines how the Service is exposed. Defaults to ClusterIP. Valid
	// options are ExternalName, ClusterIP, NodePort, and LoadBalancer.
	// "ClusterIP" allocates a cluster-internal IP address for load-balancing
	// to endpoints. Endpoints are determined by the selector or if that is not
	// specified, by manual construction of an Endpoints object or
	// EndpointSlice objects. If clusterIP is "None", no virtual IP is
	// allocated and the endpoints are published as a set of endpoints rather
	// than a virtual IP.
	// "NodePort" builds on ClusterIP and allocates a port on every node which
	// routes to the same endpoints as the clusterIP.
	// "LoadBalancer" builds on NodePort and creates an external load-balancer
	// (if supported in the current cloud) which routes to the same endpoints
	// as the clusterIP.
	// "ExternalName" aliases this service to the specified externalName.
	// Several other fields do not apply to ExternalName services.
	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types
	// +optional
	Type corev1.ServiceType `json:"type,omitempty"`
}

type TiDBTLS struct {
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
	MySQL *TLS `json:"mysql,omitempty"`

	// TODO(csuzhangxc): usage of the following fields
	// TODO(liubo02): uncomment them after it's worked

	// DisableClientAuthn will skip client's certificate validation from the TiDB server.
	// Optional: defaults to false
	// DisableClientAuthn bool `json:"disableClientAuthn,omitempty"`

	// SkipInternalClientCA will skip TiDB server's certificate validation for internal components like Initializer, Dashboard, etc.
	// Optional: defaults to false
	// SkipInternalClientCA bool `json:"skipInternalClientCA,omitempty"`
}

type TiDBAuthToken struct {
	// Secret name of jwks
	JWKs corev1.LocalObjectReference `json:"jwks"`
}

type TiDBGroupStatus struct {
	CommonStatus `json:",inline"`
	GroupStatus  `json:",inline"`
}

// +kubebuilder:validation:XValidation:rule="(!has(oldSelf.topology) && !has(self.topology)) || (has(oldSelf.topology) && has(self.topology))",fieldPath=".topology",message="topology can only be set when creating"
type TiDBSpec struct {
	Cluster ClusterReference `json:"cluster"`

	// Topology defines the topology domain of this TiDB instance.
	// It will be translated into a node affnity config.
	// Topology cannot be changed.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="topology is immutable"
	Topology Topology `json:"topology,omitempty"`

	// Subdomain means the subdomain of the exported pd dns.
	// A same pd cluster will use a same subdomain
	Subdomain string `json:"subdomain"`

	// TiDBTemplateSpec embeded some fields managed by TiDBGroup.
	TiDBTemplateSpec `json:",inline"`
}

type TiDBStatus struct {
	CommonStatus `json:",inline"`
}
