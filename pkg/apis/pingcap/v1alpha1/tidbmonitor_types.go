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
	"time"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TidbMonitor encode the spec and status of the monitoring component of a TiDB cluster
//
// +k8s:openapi-gen=true
// +kubebuilder:resource:shortName="tm"
type TidbMonitor struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	// Spec defines the desired state of TidbMonitor
	Spec TidbMonitorSpec `json:"spec"`

	// +k8s:openapi-gen=false
	// Most recently observed status of the TidbMonitor
	Status TidbMonitorStatus `json:"status,omitempty"`
}

type DMMonitorSpec struct {
	Clusters    []ClusterRef    `json:"clusters"`
	Initializer InitializerSpec `json:"initializer"`
}

// +k8s:openapi-gen=true
// TidbMonitor spec encode the desired state of tidb monitoring component
type TidbMonitorSpec struct {
	// +optional
	// monitored TiDB cluster info
	Clusters []TidbClusterRef `json:"clusters,omitempty"`

	// Prometheus spec
	Prometheus PrometheusSpec `json:"prometheus"`

	// Grafana spec
	// +optional
	Grafana *GrafanaSpec `json:"grafana,omitempty"`

	// Reloader spec
	Reloader ReloaderSpec `json:"reloader"`

	// Initializer spec
	Initializer InitializerSpec `json:"initializer"`

	// monitored DM cluster spec
	// +optional
	DM *DMMonitorSpec `json:"dm,omitempty"`

	// Thanos spec
	// +optional
	Thanos *ThanosSpec `json:"thanos,omitempty"`

	//PrometheusReloader set prometheus reloader configuration
	//+optional
	PrometheusReloader *PrometheusReloaderSpec `json:"prometheusReloader,omitempty"`

	// Persistent volume reclaim policy applied to the PVs that consumed by TiDB cluster
	// +kubebuilder:default=Retain
	PVReclaimPolicy *corev1.PersistentVolumeReclaimPolicy `json:"pvReclaimPolicy,omitempty"`

	// ImagePullPolicy of TidbMonitor Pods
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images.
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// If Persistent enabled, storageClassName must be set to an existing storage.
	// Defaults to false.
	// +optional
	Persistent bool `json:"persistent,omitempty"`

	// The storageClassName of the persistent volume for TidbMonitor data storage.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// Size of the persistent volume.
	// +optional
	Storage string `json:"storage,omitempty"`

	// NodeSelector of the TidbMonitor.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Annotations for the TidbMonitor.
	// Optional: Defaults to cluster-level setting
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels for the TidbMonitor.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Tolerations of the TidbMonitor.
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

	// Additional containers of the TidbMonitor.
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

	// Replicas is the number of desired replicas.
	// Defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// EXPERIMENTAL: Number of shards to distribute targets onto. Number of
	// replicas multiplied by shards is the total number of Pods created. Note
	// that scaling down shards will not reshard data onto remaining instances,
	// it must be manually moved. Increasing shards will not reshard data
	// either but it will continue to be available from the same instances. To
	// query globally use Thanos sidecar and Thanos querier or remote write
	// data to a central location. Sharding is done on the content of the
	// `__address__` target meta-label.
	Shards *int32 `json:"shards,omitempty"`

	// Additional volumes of TidbMonitor pod.
	// +optional
	AdditionalVolumes []corev1.Volume `json:"additionalVolumes,omitempty"`

	// PodSecurityContext of TidbMonitor pod.
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// EnableAlertRules adds alert rules to the Prometheus config even
	// if `AlertmanagerURL` is not configured.
	// +optional
	EnableAlertRules bool `json:"enableAlertRules,omitempty"`

	// Time zone of TidbMonitor
	// Optional: Defaults to UTC
	// +optional
	Timezone string `json:"timezone,omitempty"`
}

// PrometheusReloaderSpec is the desired state of prometheus configuration reloader
type PrometheusReloaderSpec struct {
	MonitorContainer `json:",inline"`
}

// PrometheusSpec is the desired state of prometheus
type PrometheusSpec struct {
	MonitorContainer `json:",inline"`

	// Prometheus log level
	LogLevel string `json:"logLevel,omitempty"`

	// Service defines a Kubernetes service of Prometheus.
	Service ServiceSpec `json:"service,omitempty"`

	// ReserveDays defines Prometheus Configuration for `--storage.tsdb.retention.time` of units d.
	// reserveDays will be used if retentionTime not defined.
	// +optional
	ReserveDays int `json:"reserveDays,omitempty"`

	// Configuration for `--storage.tsdb.retention.time`, Units Supported: y, w, d, h, m, s, ms.
	// If set to non empty values, it will override the value of `ReserveDays`.
	// +optional
	RetentionTime *string `json:"retentionTime,omitempty"`

	// Ingress configuration of Prometheus
	// +optional
	Ingress *IngressSpec `json:"ingress,omitempty"`

	// Config is the Configuration of Prometheus include Prometheus config/Cli options/custom rules.
	// +optional
	Config *PrometheusConfiguration `json:"config,omitempty"`

	// Disable prometheus compaction.
	// Defaults to false.
	DisableCompaction bool `json:"disableCompaction,omitempty"`

	// If specified, the remote_write spec. This is an experimental feature, it may change in any upcoming release in a breaking way.
	RemoteWrite []*RemoteWriteSpec `json:"remoteWrite,omitempty"`

	// Additional volume mounts of prometheus pod.
	AdditionalVolumeMounts []corev1.VolumeMount `json:"additionalVolumeMounts,omitempty"`
}

// +k8s:openapi-gen=true
// Config  is the the desired state of Prometheus Configuration
type PrometheusConfiguration struct {

	// User can mount prometheus config with external configMap. The external configMap must contain `prometheus-config` key in data.
	ConfigMapRef *ConfigMapRef `json:"configMapRef,omitempty"`

	// user can  use it specify prometheus command options
	CommandOptions []string `json:"commandOptions,omitempty"`

	// User can mount prometheus rule config with external configMap. The external configMap must use the key with suffix `.rules.yml`.
	RuleConfigRef *ConfigMapRef `json:"ruleConfigRef,omitempty"`
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

	// Grafana log level
	LogLevel string `json:"logLevel,omitempty"`

	// Service defines a Kubernetes service of Grafana.
	Service ServiceSpec `json:"service,omitempty"`

	// +optional
	// if `UsernameSecret` is not set, `username` will be used.
	UsernameSecret *corev1.SecretKeySelector `json:"usernameSecret,omitempty"`

	// +optional
	// if `passwordSecret` is not set, `password` will be used.
	PasswordSecret *corev1.SecretKeySelector `json:"passwordSecret,omitempty"`

	// +optional
	// Deprecated in v1.3.0 for security concerns, planned for removal in v1.4.0. Use `usernameSecret` instead.
	// +k8s:openapi-gen=false
	Username string `json:"username,omitempty"`

	// +optional
	// Deprecated in v1.3.0 for security concerns, planned for removal in v1.4.0. Use `passwordSecret` instead.
	// +k8s:openapi-gen=false
	Password string `json:"password,omitempty"`

	// +optional
	Envs map[string]string `json:"envs,omitempty"`

	// Ingress configuration of Prometheus
	// +optional
	Ingress *IngressSpec `json:"ingress,omitempty"`

	// Additional volume mounts of grafana pod.
	AdditionalVolumeMounts []corev1.VolumeMount `json:"additionalVolumeMounts,omitempty"`
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
	TracingConfigFile *string `json:"tracingConfigFile,omitempty"`
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
	// Additional volume mounts of thanos pod.
	AdditionalVolumeMounts []corev1.VolumeMount `json:"additionalVolumeMounts,omitempty"`
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
	// default to the same namespace as TidbMonitor/TidbCluster/TidbNGMonitoring
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

	StatefulSet *apps.StatefulSetStatus `json:"statefulSet,omitempty"`
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
	Secret *corev1.SecretKeySelector `json:"secret,omitempty"`
	// ConfigMap containing data to use for the targets.
	ConfigMap *corev1.ConfigMapKeySelector `json:"configMap,omitempty"`
}

// RemoteWriteSpec defines the remote_write configuration for prometheus.
// +k8s:openapi-gen=true
type RemoteWriteSpec struct {
	// The URL of the endpoint to send samples to.
	URL string `json:"url"`
	// The name of the remote write queue, must be unique if specified. The
	// name is used in metrics and logging in order to differentiate queues.
	// Only valid in Prometheus versions 2.15.0 and newer.
	Name string `json:"name,omitempty"`
	// +optional
	RemoteTimeout *model.Duration `json:"remoteTimeout,omitempty"`
	// The list of remote write relabel configurations.
	// +optional
	WriteRelabelConfigs []RelabelConfig `json:"writeRelabelConfigs,omitempty"`
	//BasicAuth for the URL.
	// +optional
	BasicAuth *BasicAuth `json:"basicAuth,omitempty"`
	// File to read bearer token for remote write.
	// +optional
	BearerToken string `json:"bearerToken,omitempty"`
	// +optional
	// File to read bearer token for remote write.
	// +optional
	BearerTokenFile string `json:"bearerTokenFile,omitempty"`
	// TLS Config to use for remote write.
	// +optional
	TLSConfig *TLSConfig `json:"tlsConfig,omitempty"`
	// Proxy url
	// +optional
	ProxyURL *string `json:"proxyUrl,omitempty"`
	// +optional
	QueueConfig *QueueConfig `json:"queueConfig,omitempty"`
	// MetadataConfig configures the sending of series metadata to remote storage.
	// Only valid in Prometheus versions 2.23.0 and newer.
	// +optional
	MetadataConfig *MetadataConfig `json:"metadataConfig,omitempty"`
	// Custom HTTP headers to be sent along with each remote write request.
	// Be aware that headers that are set by Prometheus itself can't be overwritten.
	// Only valid in Prometheus versions 2.25.0 and newer.
	Headers map[string]string `json:"headers,omitempty"`
}

// Configures the sending of series metadata to remote storage.
// +k8s:openapi-gen=true
type MetadataConfig struct {
	// Whether metric metadata is sent to remote storage or not.
	Send bool `json:"send,omitempty"`
	// How frequently metric metadata is sent to remote storage.
	SendInterval string `json:"sendInterval,omitempty"`
}

// BasicAuth allow an endpoint to authenticate over basic authentication
// More info: https://prometheus.io/docs/operating/configuration/#endpoints
// +k8s:openapi-gen=true
type BasicAuth struct {
	// The secret in the service monitor namespace that contains the username
	// for authentication.
	Username corev1.SecretKeySelector `json:"username,omitempty"`
	// The secret in the service monitor namespace that contains the password
	// for authentication.
	Password corev1.SecretKeySelector `json:"password,omitempty"`
}

// RelabelConfig allows dynamic rewriting of the label set, being applied to samples before ingestion.
// It defines `<metric_relabel_configs>`-section of Prometheus configuration.
// More info: https://prometheus.io/docs/prometheus/latest/configuration/configuration/#metric_relabel_configs
// +k8s:openapi-gen=true
type RelabelConfig struct {
	// A list of labels from which values are taken and concatenated
	// with the configured separator in order.
	SourceLabels model.LabelNames `json:"sourceLabels,omitempty"`
	// Separator is the string between concatenated values from the source labels.
	Separator string `json:"separator,omitempty"`
	//Regular expression against which the extracted value is matched. Default is '(.*)'
	Regex string `json:"regex,omitempty"`
	// Modulus to take of the hash of concatenated values from the source labels.
	Modulus uint64 `json:"modulus,omitempty"`
	// TargetLabel is the label to which the resulting string is written in a replacement.
	// Regexp interpolation is allowed for the replace action.
	TargetLabel string `json:"targetLabel,omitempty"`
	// Replacement is the regex replacement pattern to be used.
	Replacement string `json:"replacement,omitempty"`
	// Action is the action to be performed for the relabeling.
	Action config.RelabelAction `json:"action,omitempty"`
}

// QueueConfig allows the tuning of remote_write queue_config parameters. This object
// is referenced in the RemoteWriteSpec object.
// +k8s:openapi-gen=true
type QueueConfig struct {
	// Number of samples to buffer per shard before we start dropping them.
	Capacity int `json:"capacity,omitempty"`

	// MinShards is the minimum number of shards, i.e. amount of concurrency.
	// Only valid in Prometheus versions 2.6.0 and newer.
	MinShards int `json:"minShards,omitempty"`

	// Max number of shards, i.e. amount of concurrency.
	MaxShards int `json:"maxShards,omitempty"`

	// Maximum number of samples per send.
	MaxSamplesPerSend int `json:"maxSamplesPerSend,omitempty"`

	// Maximum time sample will wait in buffer.
	BatchSendDeadline *time.Duration `json:"batchSendDeadline,omitempty"`

	// Max number of times to retry a batch on recoverable errors.
	MaxRetries int `json:"maxRetries,omitempty"`

	// On recoverable errors, backoff exponentially.
	MinBackoff *time.Duration `json:"minBackoff,omitempty"`
	MaxBackoff *time.Duration `json:"maxBackoff,omitempty"`
}

func (tm *TidbMonitor) GetShards() int32 {
	shards := int32(1)
	if tm.Spec.Shards != nil && *tm.Spec.Shards > 1 {
		shards = *tm.Spec.Shards
	}
	return shards
}

func (tm *TidbMonitor) Timezone() string {
	tz := tm.Spec.Timezone
	if len(tz) <= 0 {
		return defaultTimeZone
	}
	return tz
}
