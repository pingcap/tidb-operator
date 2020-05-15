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

// +k8s:openapi-gen=true
// TidbMonitor spec encode the desired state of tidb monitoring component
type TidbMonitorSpec struct {
	Clusters []TidbClusterRef `json:"clusters"`

	Prometheus PrometheusSpec `json:"prometheus"`
	// +optional
	Grafana     *GrafanaSpec    `json:"grafana,omitempty"`
	Reloader    ReloaderSpec    `json:"reloader"`
	Initializer InitializerSpec `json:"initializer"`

	// Persistent volume reclaim policy applied to the PVs that consumed by TiDB cluster
	// +kubebuilder:default=Recycle
	PVReclaimPolicy corev1.PersistentVolumeReclaimPolicy `json:"pvReclaimPolicy,omitempty"`

	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
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
}

// +k8s:openapi-gen=true
// Config  is the the desired state of Prometheus Configuration
type PrometheusConfiguration struct {

	// user can mount prometheus rule config with external configMap.If use this feature, the external configMap must contain `prometheus-config` key in data.
	ConfigMapRef *ConfigMapRef `json:"configMapRef,omitempty"`

	// user can  use it specify prometheus command options
	CommandOptions []string `json:"command,omitempty"`
}

// ConfigMapRef is the external configMap
type ConfigMapRef struct {
	Name      string  `json:"name,omitempty"`
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

// +k8s:openapi-gen=true
// MonitorContainer is the common attributes of the container of monitoring
type MonitorContainer struct {
	Resources corev1.ResourceRequirements `json:",inline"`

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
}

// TODO: sync status
type TidbMonitorStatus struct {
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
