// Copyright 2019 PingCAP, Inc.
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

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// TidbMonitor encode the spec and status of the monitoring component of a TiDB cluster
type TidbMonitor struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	// Spec defines the desired state of TidbMonitor
	Spec TidbMonitorSpec `json:"spec"`

	// +k8s:openapi-gen=false
	// Most recently observed status of the TidbMonitor
	Status TidbMonitorStatus `json:"status"`
}

type DMMonitorSpec struct {
	Clusters    []ClusterRef    `json:"clusters"`
	Initializer InitializerSpec `json:"initializer"`
}

// +k8s:openapi-gen=true
// TidbMonitor spec encode the desired state of tidb monitoring component
type TidbMonitorSpec struct {
	Clusters []TidbClusterRef `json:"clusters"`

	Prometheus PrometheusSpec `json:"prometheus"`
	// +optional
	Grafana     *GrafanaSpec    `json:"grafana,omitempty"`
	Reloader    ReloaderSpec    `json:"reloader"`
	Initializer InitializerSpec `json:"initializer"`
	DM          *DMMonitorSpec  `json:"dm,omitempty"`
	// +optional
	Thanos *ThanosSpec `json:"thanos,omitempty"`

	// Persistent volume reclaim policy applied to the PVs that consumed by TiDB cluster
	// +kubebuilder:default=Retain
	PVReclaimPolicy *corev1.PersistentVolumeReclaimPolicy `json:"pvReclaimPolicy,omitempty"`

	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// +optional
	Persistent bool `json:"persistent,omitempty"`
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
	// +optional
	Storage string `json:"storage,omitempty"`
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// kubePrometheusURL is where tidb-monitoring get the  common metrics of kube-prometheus.
	// Ref: https://github.com/coreos/kube-prometheus
	// +optional
	KubePrometheusURL *string `json:"kubePrometheusURL,omitempty"`
	// alertmanagerURL is where tidb-monitoring push alerts to.
	// Ref: https://prometheus.io/docs/alerting/alertmanager/
	// +optional
	AlertmanagerURL *string `json:"alertmanagerURL,omitempty"`
	// alertManagerRulesVersion is the version of the tidb cluster that used for alert rules.
	// default to current tidb cluster version, for example: v3.0.15
	// +optional
	AlertManagerRulesVersion *string `json:"alertManagerRulesVersion,omitempty"`

	// +optional
	AdditionalContainers []corev1.Container `json:"additionalContainers,omitempty"`

	// ClusterScoped indicates whether this monitor should manage Kubernetes cluster-wide TiDB clusters
	ClusterScoped bool `json:"clusterScoped,omitempty"`
	// The labels to add to any time series or alerts when communicating with
	// external systems (federation, remote storage, Alertmanager).
	ExternalLabels map[string]string `json:"externalLabels,omitempty"`
	// Name of Prometheus external label used to denote replica name.
	// Defaults to the value of `prometheus_replica`. External label will
	// _not_ be added when value is set to empty string (`""`).
	ReplicaExternalLabelName *string `json:"replicaExternalLabelName,omitempty"`
}

// PrometheusSpec is the desired state of prometheus
type PrometheusSpec struct {
	MonitorContainer `json:",inline"`

	LogLevel string      `json:"logLevel,omitempty"`
	Service  ServiceSpec `json:"service,omitempty"`
	// +optional
	ReserveDays int `json:"reserveDays,omitempty"`

	// +optional
	Ingress *IngressSpec `json:"ingress,omitempty"`

	// +optional
	Config *PrometheusConfiguration `json:"config,omitempty"`

	// Disable prometheus compaction.
	DisableCompaction bool `json:"disableCompaction,omitempty"`
}

// +k8s:openapi-gen=true
// Config  is the the desired state of Prometheus Configuration
type PrometheusConfiguration struct {

	// user can mount prometheus rule config with external configMap.If use this feature, the external configMap must contain `prometheus-config` key in data.
	ConfigMapRef *ConfigMapRef `json:"configMapRef,omitempty"`

	// user can  use it specify prometheus command options
	CommandOptions []string `json:"commandOptions,omitempty"`
}

// ConfigMapRef is the external configMap
// +k8s:openapi-gen=true
type ConfigMapRef struct {
	Name string `json:"name,omitempty"`

	// +optional
	// if the namespace is omitted, the operator controller would use the Tidbmonitor's namespace instead.
	Namespace *string `json:"namespace,omitempty"`
}

// GrafanaSpec is the desired state of grafana
type GrafanaSpec struct {
	MonitorContainer `json:",inline"`

	LogLevel string      `json:"logLevel,omitempty"`
	Service  ServiceSpec `json:"service,omitempty"`
	Username string      `json:"username,omitempty"`
	Password string      `json:"password,omitempty"`
	// +optional
	Envs map[string]string `json:"envs,omitempty"`

	// +optional
	Ingress *IngressSpec `json:"ingress,omitempty"`
}

// ReloaderSpec is the desired state of reloader
type ReloaderSpec struct {
	MonitorContainer `json:",inline"`
	Service          ServiceSpec `json:"service,omitempty"`
}

// InitializerSpec is the desired state of initializer
type InitializerSpec struct {
	MonitorContainer `json:",inline"`
	// +optional
	Envs map[string]string `json:"envs,omitempty"`
}

// ThanosSpec is the desired state of thanos sidecar
type ThanosSpec struct {
	MonitorContainer `json:",inline"`
	// ObjectStorageConfig configures object storage in Thanos.
	// Alternative to ObjectStorageConfigFile, and lower order priority.
	ObjectStorageConfig *corev1.SecretKeySelector `json:"objectStorageConfig,omitempty"`
	// ObjectStorageConfigFile specifies the path of the object storage configuration file.
	// When used alongside with ObjectStorageConfig, ObjectStorageConfigFile takes precedence.
	ObjectStorageConfigFile *string `json:"objectStorageConfigFile,omitempty"`
	// ListenLocal makes the Thanos sidecar listen on loopback, so that it
	// does not bind against the Pod IP.
	ListenLocal bool `json:"listenLocal,omitempty"`
	// TracingConfig configures tracing in Thanos. This is an experimental feature, it may change in any upcoming release in a breaking way.
	TracingConfig *corev1.SecretKeySelector `json:"tracingConfig,omitempty"`
	// TracingConfig specifies the path of the tracing configuration file.
	// When used alongside with TracingConfig, TracingConfigFile takes precedence.
	TracingConfigFile *string `json:"objectStorageConfigFile,omitempty"`
	// GRPCServerTLSConfig configures the gRPC server from which Thanos Querier reads
	// recorded rule data.
	// Note: Currently only the CAFile, CertFile, and KeyFile fields are supported.
	// Maps to the '--grpc-server-tls-*' CLI args.
	GRPCServerTLSConfig *TLSConfig `json:"grpcServerTlsConfig,omitempty"`
	// LogLevel for Thanos sidecar to be configured with.
	LogLevel string `json:"logLevel,omitempty"`
	// LogFormat for Thanos sidecar to be configured with.
	LogFormat string `json:"logFormat,omitempty"`
	// MinTime for Thanos sidecar to be configured with. Option can be a constant time in RFC3339 format or time duration relative to current time, such as -1d or 2h45m. Valid duration units are ms, s, m, h, d, w, y.
	MinTime string `json:"minTime,omitempty"`
	// RoutePrefix is prometheus prefix url
	RoutePrefix string `json:"routePrefix,omitempty"`
}

// +k8s:openapi-gen=true
// MonitorContainer is the common attributes of the container of monitoring
type MonitorContainer struct {
	corev1.ResourceRequirements `json:",inline"`

	BaseImage string `json:"baseImage,omitempty"`
	Version   string `json:"version,omitempty"`
	// +optional
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

// +k8s:openapi-gen=true
// TidbClusterRef reference to a TidbCluster
type TidbClusterRef struct {
	// Namespace is the namespace that TidbCluster object locates,
	// default to the same namespace with TidbMonitor
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Name is the name of TidbCluster object
	Name string `json:"name"`

	// ClusterDomain is the domain of TidbCluster object
	// +optional
	ClusterDomain string `json:"clusterDomain,omitempty"`
}

// +k8s:openapi-gen=true
// ClusterRef reference to a TidbCluster
type ClusterRef TidbClusterRef

type TidbMonitorStatus struct {
	// Storage status for deployment
	DeploymentStorageStatus *DeploymentStorageStatus `json:"deploymentStorageStatus,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// TidbMonitorList is TidbMonitor list
type TidbMonitorList struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ListMeta `json:"metadata"`

	Items []TidbMonitor `json:"items"`
}

// DeploymentStorageStatus is the storage information of the deployment
type DeploymentStorageStatus struct {
	// PV name
	PvName string `json:"pvName,omitempty"`
}

// TLSConfig extends the safe TLS configuration with file parameters.
// +k8s:openapi-gen=true
type TLSConfig struct {
	SafeTLSConfig `json:",inline"`
	// Path to the CA cert in the Prometheus container to use for the targets.
	CAFile string `json:"caFile,omitempty"`
	// Path to the client cert file in the Prometheus container for the targets.
	CertFile string `json:"certFile,omitempty"`
	// Path to the client key file in the Prometheus container for the targets.
	KeyFile string `json:"keyFile,omitempty"`
}

// SafeTLSConfig specifies safe TLS configuration parameters.
// +k8s:openapi-gen=true
type SafeTLSConfig struct {
	// Struct containing the CA cert to use for the targets.
	CA SecretOrConfigMap `json:"ca,omitempty"`
	// Struct containing the client cert file for the targets.
	Cert SecretOrConfigMap `json:"cert,omitempty"`
	// Secret containing the client key file for the targets.
	KeySecret *corev1.SecretKeySelector `json:"keySecret,omitempty"`
	// Used to verify the hostname for the targets.
	ServerName string `json:"serverName,omitempty"`
	// Disable target certificate validation.
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`
}

// SecretOrConfigMap allows to specify data as a Secret or ConfigMap. Fields are mutually exclusive.
type SecretOrConfigMap struct {
	// Secret containing data to use for the targets.
	Secret *v1.SecretKeySelector `json:"secret,omitempty"`
	// ConfigMap containing data to use for the targets.
	ConfigMap *v1.ConfigMapKeySelector `json:"configMap,omitempty"`
}
