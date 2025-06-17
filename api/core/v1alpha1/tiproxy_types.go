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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	meta "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
)

const (
	TiProxyPortNameClient = "mysql-client"
	TiProxyPortNameAPI    = "api"
	TiProxyPortNamePeer   = "peer"

	DefaultTiProxyPortClient = 6000
	DefaultTiProxyPortAPI    = 3080
	DefaultTiProxyPortPeer   = 3081
)

const (
	TiProxyGroupCondAvailable   = "Available"
	TiProxyGroupAvailableReason = "TiProxyGroupAvailable"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// TiProxyGroupList defines a list of TiProxy groups
type TiProxyGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TiProxyGroup `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:resource:categories=group,shortName=pg
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

// TiProxyGroup defines a group of similar TiProxy instances.
type TiProxyGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiProxyGroupSpec   `json:"spec,omitempty"`
	Status TiProxyGroupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// TiProxyList defines a list of TiProxy instances.
type TiProxyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TiProxy `json:"items"`
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

// TiProxy defines a TiProxy instance.
type TiProxy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TiProxySpec   `json:"spec,omitempty"`
	Status TiProxyStatus `json:"status,omitempty"`
}

// TiProxyGroupSpec describes the common attributes of a TiProxyGroup.
type TiProxyGroupSpec struct {
	Cluster ClusterReference `json:"cluster"`
	// Features are enabled feature
	Features []meta.Feature `json:"features,omitempty"`
	// +kubebuilder:validation:Minimum=0
	Replicas *int32 `json:"replicas"`

	// +listType=map
	// +listMapKey=type
	SchedulePolicies []SchedulePolicy `json:"schedulePolicies,omitempty"`

	Template TiProxyTemplate `json:"template"`
}

type TiProxyTemplate struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       TiProxyTemplateSpec `json:"spec"`
}

// TiProxyTemplateSpec can only be specified in TiProxyGroup.
// +kubebuilder:validation:XValidation:rule="!has(self.overlay) || !has(self.overlay.volumeClaims) || (has(self.volumes) && self.overlay.volumeClaims.all(vc, vc.name in self.volumes.map(v, v.name)))",message="overlay volumeClaims names must exist in volumes"
type TiProxyTemplateSpec struct {
	// Version must be a semantic version.
	// It can has a v prefix or not.
	// +kubebuilder:validation:Pattern=`^(v)?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`
	Version string `json:"version"`
	// Image is TiProxy's image
	// If tag is omitted, version will be used as the image tag.
	// Default is pingcap/tiproxy
	Image *string `json:"image,omitempty"`
	// Server defines the server configuration of TiProxy.
	Server TiProxyServer `json:"server,omitempty"`
	// Probes defines probes for TiProxy.
	Probes TiProxyProbes `json:"probes,omitempty"`
	// Resources defines resource required by TiProxy.
	Resources ResourceRequirements `json:"resources,omitempty"`
	// Config defines config file of TiProxy.
	Config         ConfigFile     `json:"config,omitempty"`
	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`

	Security *TiProxySecurity `json:"security,omitempty"`
	// Volumes defines data volume of TiProxy, it is optional.
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=256
	Volumes []Volume `json:"volumes,omitempty"`

	// PreStop defines the preStop config for the TiProxy container.
	PreStop *TiProxyPreStop `json:"preStop,omitempty"`

	// Overlay defines a k8s native resource template patch.
	// All resources(pod, pvcs, ...) managed by TiProxy can be overlayed by this field.
	Overlay *Overlay `json:"overlay,omitempty"`
}

type TiProxyPreStop struct {
	// SleepSeconds is the seconds to sleep before sending the SIGTERM to the TiProxy container.
	// It's useful to achieve a graceful shutdown of the TiProxy container.
	// Default is 10 seconds.
	SleepSeconds int32 `json:"sleepSeconds,omitempty"`
}

type TiProxySecurity struct {
	// Whether enable the TLS connection.
	TLS *TiProxyTLS `json:"tls,omitempty"`
}

type TiProxyServer struct {
	// Port defines all ports listened by TiProxy.
	Ports TiProxyPorts `json:"ports,omitempty"`

	// Labels defines the server labels of the TiProxy.
	// TiDB Operator will ignore `labels` in TiProxy's config file and use this field instead.
	// Note these label keys are managed by TiDB Operator, it will be set automatically and you can not modify them:
	//  - host
	//  - region
	//  - zone
	// +kubebuilder:validation:XValidation:rule="!('host' in self) && !('region' in self) && !('zone' in self)",message="labels cannot contain 'host', 'region', or 'zone' keys"
	Labels map[string]string `json:"labels,omitempty"`
}

type TiProxyPorts struct {
	// Client defines port for TiProxy's SQL service.
	Client *Port `json:"client,omitempty"`
	// API defines port for TiProxy API service.
	API *Port `json:"api,omitempty"`
	// Peer defines port for TiProxy's peer service.
	Peer *Port `json:"peer,omitempty"`
}

type TiProxyProbes struct {
	// Readiness defines the readiness probe for TiProxy.
	// The default handler is a TCP socket on the client port.
	Readiness *TiProxyProb `json:"readiness,omitempty"`
}

type TiProxyProb struct {
	// "tcp" will use TCP socket to connect component port.
	// "command" will probe the HTTP API of TiProxy.
	// +kubebuilder:validation:Enum=tcp;command
	Type *string `json:"type,omitempty"`
}

type TiProxyTLS struct {
	// MySQL defines the TLS configuration for connections between TiProxy and MySQL clients.
	// The steps to enable this feature:
	//   1. Generate a TiProxy server-side certificate for the TiProxy cluster.
	//      There are multiple ways to generate certificates:
	//        - user-provided certificates: https://docs.pingcap.com/TiProxy/stable/generate-self-signed-certificates
	//        - use the K8s built-in certificate signing system signed certificates: https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/
	//        - or use cert-manager signed certificates: https://cert-manager.io/
	//   2. Create a K8s Secret object which contains the TiProxy server-side certificate created above.
	//      The name of this Secret must be: <groupName>-tiproxy-server-secret.
	//        kubectl create secret generic <groupName>-tiproxy-server-secret --namespace=<namespace> --from-file=tls.crt=<path/to/tls.crt> --from-file=tls.key=<path/to/tls.key> --from-file=ca.crt=<path/to/ca.crt>
	//   3. Set Enabled to `true`.
	MySQL *TLS `json:"mysql,omitempty"`

	// Backend defines the TLS configuration for connections between TiProxy and TiDB servers.
	// To enable this feature, the corresponding TiDB server must be configured with TLS enabled.
	Backend *TLS `json:"backend,omitempty"`
}

type TiProxyGroupStatus struct {
	CommonStatus `json:",inline"`
	GroupStatus  `json:",inline"`
}

// +kubebuilder:validation:XValidation:rule="(!has(oldSelf.topology) && !has(self.topology)) || (has(oldSelf.topology) && has(self.topology))",fieldPath=".topology",message="topology can only be set when creating"
type TiProxySpec struct {
	Cluster ClusterReference `json:"cluster"`
	// Features are enabled feature
	Features []meta.Feature `json:"features,omitempty"`

	// Topology defines the topology domain of this TiProxy instance.
	// It will be translated into a node affinity config.
	// Topology cannot be changed.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="topology is immutable"
	Topology Topology `json:"topology,omitempty"`

	// Subdomain means the subdomain of the exported tiproxy dns.
	// A same tiproxy cluster will use a same subdomain
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="subdomain is immutable"
	Subdomain string `json:"subdomain"`

	// TiProxyTemplateSpec embeded some fields managed by TiProxyGroup.
	TiProxyTemplateSpec `json:",inline"`
}

type TiProxyStatus struct {
	CommonStatus `json:",inline"`
}
