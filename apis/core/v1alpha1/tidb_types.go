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
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
)

var (
	_ GroupList         = &TiDBGroupList{}
	_ Group             = &TiDBGroup{}
	_ ComponentAccessor = &TiDB{}
)

const (
	// VolumeUsageTypeTiDBSlowLog means the data dir of slowlog
	// The default sub path is "slowlog"
	// Users can define a persistent volume for slowlog, or an empty dir will be used.
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
	// TiDBSQLTLSVolumeName is the volume name for the TLS secret used by TLS communication between TiDB server and MySQL client.
	TiDBSQLTLSVolumeName = NamePrefix + "tidb-sql-tls"
	// TiDBSQLTLSMountPath is the volume mount path for the TLS secret used by TLS communication between TiDB server and MySQL client.
	TiDBSQLTLSMountPath = "/var/lib/tidb-sql-tls"
)

const (
	BootstrapSQLVolumeName   = NamePrefix + "tidb-bootstrap-sql"
	BootstrapSQLFilePath     = "/etc/tidb-bootstrap"
	BootstrapSQLFileName     = "bootstrap.sql"
	BootstrapSQLConfigMapKey = "bootstrap-sql"
)

const (
	TiDBAuthTokenVolumeName = NamePrefix + "tidb-auth-token"
	TiDBAuthTokenPath       = "/var/lib/tidb-auth-token"
	TiDBAuthTokenJWKS       = "tidb_auth_token_jwks.json"
)

const (
	TiDBCondHealth   = "Health"
	TiDBHealthReason = "TiDBHealth"

	TiDBCondSuspended = "Suspended"
	TiDBSuspendReason = "TiDBSuspend"

	TiDBGroupCondAvailable   = "Available"
	TiDBGroupAvailableReason = "TiDBGroupAvailable"

	TiDBGroupCondSuspended = "Suspended"
	TiDBGroupSuspendReason = "TiDBGroupSuspend"
)

const (
	TiDBSlowLogContainerName     = NamePrefix + "slowlog"
	TiDBDefaultSlowLogVolumeName = NamePrefix + "slowlog"
	TiDBSlowLogFileName          = "slowlog"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// TiDBGroupList defines a list of TiDB groups
type TiDBGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TiDBGroup `json:"items"`
}

func (l *TiDBGroupList) ToSlice() []Group {
	groups := make([]Group, 0, len(l.Items))
	for i := range l.Items {
		groups = append(groups, &l.Items[i])
	}
	return groups
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=tc
// +kubebuilder:resource:categories=tg
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Available",type=string,JSONPath=`.status.conditions[?(@.type=="Available")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TiDBGroup defines a group of similar TiDB instances.
type TiDBGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiDBGroupSpec   `json:"spec,omitempty"`
	Status TiDBGroupStatus `json:"status,omitempty"`
}

func (in *TiDBGroup) GetClusterName() string {
	return in.Spec.Cluster.Name
}

func (in *TiDBGroup) GetDesiredReplicas() int32 {
	if in.Spec.Replicas == nil {
		return 0
	}
	return *in.Spec.Replicas
}

func (in *TiDBGroup) GetDesiredVersion() string {
	return in.Spec.Version
}

func (in *TiDBGroup) GetActualVersion() string {
	return in.Status.Version
}

func (in *TiDBGroup) GetStatus() GroupStatus {
	return in.Status.GroupStatus
}

func (in *TiDBGroup) GVK() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("TiDBGroup")
}

func (in *TiDBGroup) ObservedGeneration() int64 {
	return in.Status.ObservedGeneration
}

func (in *TiDBGroup) CurrentRevision() string {
	return in.Status.CurrentRevision
}

func (in *TiDBGroup) UpdateRevision() string {
	return in.Status.UpdateRevision
}

func (in *TiDBGroup) CollisionCount() *int32 {
	if in.Status.CollisionCount == nil {
		return nil
	}
	return ptr.To(*in.Status.CollisionCount)
}

func (in *TiDBGroup) ComponentKind() ComponentKind {
	return ComponentKindTiDB
}

func (in *TiDBGroup) IsHealthy() bool {
	return meta.IsStatusConditionTrue(in.Status.Conditions, TiDBGroupCondAvailable) && in.DeletionTimestamp.IsZero()
}

func (in *TiDBGroup) GetClientPort() int32 {
	if in.Spec.Template.Spec.Server.Ports.Client != nil {
		return in.Spec.Template.Spec.Server.Ports.Client.Port
	}
	return DefaultTiDBPortClient
}

func (in *TiDBGroup) GetStatusPort() int32 {
	if in.Spec.Template.Spec.Server.Ports.Status != nil {
		return in.Spec.Template.Spec.Server.Ports.Status.Port
	}
	return DefaultTiDBPortStatus
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
// +kubebuilder:resource:categories=tc
// +kubebuilder:resource:categories=tidb
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.cluster.name`
// +kubebuilder:printcolumn:name="Healthy",type=string,JSONPath=`.status.conditions[?(@.type=="Health")].status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TiDB defines a TiDB instance.
type TiDB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiDBSpec   `json:"spec,omitempty"`
	Status TiDBStatus `json:"status,omitempty"`
}

func (in *TiDB) GetClusterName() string {
	return in.Spec.Cluster.Name
}

func (in *TiDB) GetName() string {
	return in.Name
}

func (in *TiDB) ComponentKind() ComponentKind {
	return ComponentKindTiDB
}

func (in *TiDB) GVK() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("TiDB")
}

func (in *TiDB) IsSeparateSlowLogEnabled() bool {
	if in.Spec.SlowLog == nil {
		return true // enabled by default
	}
	return !in.Spec.SlowLog.Disabled
}

func (in *TiDB) ObservedGeneration() int64 {
	return in.Status.ObservedGeneration
}

func (in *TiDB) CurrentRevision() string {
	return in.Status.CurrentRevision
}

func (in *TiDB) UpdateRevision() string {
	return in.Status.UpdateRevision
}

func (in *TiDB) CollisionCount() *int32 {
	if in.Status.CollisionCount == nil {
		return nil
	}
	return ptr.To(*in.Status.CollisionCount)
}

func (in *TiDB) IsHealthy() bool {
	return meta.IsStatusConditionTrue(in.Status.Conditions, TiDBCondHealth) && in.DeletionTimestamp.IsZero()
}

func (in *TiDB) GetClientPort() int32 {
	if in.Spec.Server.Ports.Client != nil {
		return in.Spec.Server.Ports.Client.Port
	}
	return DefaultTiDBPortClient
}

func (in *TiDB) GetStatusPort() int32 {
	if in.Spec.Server.Ports.Status != nil {
		return in.Spec.Server.Ports.Status.Port
	}
	return DefaultTiDBPortStatus
}

// NOTE: name prefix is used to generate all names of underlying resources of this instance
func (in *TiDB) NamePrefixAndSuffix() (prefix, suffix string) {
	index := strings.LastIndexByte(in.Name, '-')
	// TODO(liubo02): validate name to avoid '-' is not found
	if index == -1 {
		panic("cannot get name prefix")
	}
	return in.Name[:index], in.Name[index+1:]
}

// This name is not only for pod, but also configMap, hostname and almost all underlying resources
// TODO(liubo02): rename to more reasonable one
func (in *TiDB) PodName() string {
	prefix, suffix := in.NamePrefixAndSuffix()
	return prefix + "-tidb-" + suffix
}

// TLSClusterSecretName returns the mTLS secret name for a component.
// TODO(liubo02): move to namer
func (in *TiDB) TLSClusterSecretName() string {
	prefix, _ := in.NamePrefixAndSuffix()
	return prefix + "-tidb-cluster-secret"
}

// TiDBGroupSpec describes the common attributes of a TiDBGroup.
type TiDBGroupSpec struct {
	Cluster  ClusterReference `json:"cluster"`
	Replicas *int32           `json:"replicas"`
	Version  string           `json:"version"`

	// Service defines some fields used to override the default service.
	Service *TiDBService `json:"service,omitempty"`

	SchedulePolicies []SchedulePolicy `json:"schedulePolicies,omitempty"`

	Template TiDBTemplate `json:"template"`
}

type TiDBTemplate struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       TiDBTemplateSpec `json:"spec"`
}

// TiDBTemplateSpec can only be specified in TiDBGroup.
type TiDBTemplateSpec struct {
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
	Config         ConfigFile     `json:"config"`
	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`

	Security *TiDBSecurity `json:"security,omitempty"`
	// Volumes defines data volume of TiDB, it is optional.
	Volumes []Volume `json:"volumes,omitempty"`

	// SlowLog defines the separate slow log configuration for TiDB.
	// When enabled, a sidecar container will be created to output the slow log to its stdout.
	SlowLog *TiDBSlowLog `json:"slowLog,omitempty"`

	// Overlay defines a k8s native resource template patch.
	// All resources(pod, pvcs, ...) managed by TiDB can be overlayed by this field.
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	Overlay *Overlay `json:"overlay,omitempty"`
}

type TiDBSecurity struct {
	// Whether enable the TLS connection between the TiDB server and MySQL client.
	// TODO(liubo02): rename the TiDBTLSClient struct,
	TLS *TiDBTLS `json:"tls,omitempty"`

	// BootstrapSQL refer to a configmap which contains the bootstrap SQL file with the key `bootstrap-sql`,
	// which will only be executed when a TiDB cluster bootstrap on the first time.
	// The field should be set ONLY when create the first TiDB group for a cluster, since it only take effect on the first time bootstrap.
	// Only v6.5.1+ supports this feature.
	// TODO(liubo02): move to cluster spec
	BootstrapSQL *corev1.LocalObjectReference `json:"bootstrapSQL,omitempty"`

	// Whether enable `tidb_auth_token` authentication method.
	// To enable this feature, a K8s secret named `<groupName>-tidb-auth-token-jwks-secret` must be created to store the JWKs.
	// ref: https://docs.pingcap.com/tidb/stable/security-compatibility-with-mysql#tidb_auth_token
	// Defaults to false.
	AuthToken *TiDBAuthToken `json:"authToken,omitempty"`
}

type TiDBServer struct {
	// Port defines all ports listened by TiDB.
	Ports TiDBPorts `json:"ports,omitempty"`
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
	// Disabled indicates whether the separate slow log is disabled.
	// Defaults to false. In other words, the separate slow log is enabled by default.
	Disabled bool `json:"disable,omitempty"`

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

type TiDBSpec struct {
	Cluster ClusterReference `json:"cluster"`

	// Topology defines the topology domain of this TiDB instance.
	// It will be translated into a node affnity config.
	// Topology cannot be changed.
	Topology Topology `json:"topology,omitempty"`

	// Version specifies the TiDB version.
	Version string `json:"version"`

	// Subdomain means the subdomain of the exported pd dns.
	// A same pd cluster will use a same subdomain
	Subdomain string `json:"subdomain"`

	// TiDBTemplateSpec embeded some fields managed by TiDBGroup.
	TiDBTemplateSpec `json:",inline"`
}

type TiDBStatus struct {
	CommonStatus `json:",inline"`
}

// IsMySQLTLSEnabled returns whether the TLS between TiDB server and MySQL client is enabled.
func (in *TiDB) IsMySQLTLSEnabled() bool {
	return in.Spec.Security != nil && in.Spec.Security.TLS != nil && in.Spec.Security.TLS.MySQL != nil && in.Spec.Security.TLS.MySQL.Enabled
}

// MySQLTLSSecretName returns the secret name used in TiDB server for the TLS between TiDB server and MySQL client.
func (in *TiDB) MySQLTLSSecretName() string {
	prefix, _ := in.NamePrefixAndSuffix()
	return prefix + "-tidb-server-secret"
}

func (in *TiDB) IsBootstrapSQLEnabled() bool {
	return in.Spec.Security != nil && in.Spec.Security.BootstrapSQL != nil
}

func (in *TiDB) IsTokenBasedAuthEnabled() bool {
	return in.Spec.Security != nil && in.Spec.Security.AuthToken != nil
}

func (in *TiDB) AuthTokenJWKSSecretName() string {
	if in.IsTokenBasedAuthEnabled() {
		return in.Spec.Security.AuthToken.JWKs.Name
	}
	return ""
}
