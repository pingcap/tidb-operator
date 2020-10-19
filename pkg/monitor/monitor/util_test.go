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

package monitor

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

func TestGetMonitorConfigMap(t *testing.T) {
	g := NewGomegaWithT(t)
	varTrue := true

	testCases := []struct {
		name     string
		cluster  v1alpha1.TidbCluster
		monitor  v1alpha1.TidbMonitor
		expected *corev1.ConfigMap
	}{
		{
			name: "basic",
			cluster: v1alpha1.TidbCluster{
				Spec: v1alpha1.TidbClusterSpec{
					TLSCluster: &v1alpha1.TLSCluster{Enabled: false},
				},
			},
			monitor: v1alpha1.TidbMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
			},
			expected: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-monitor",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "monitor",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "pingcap.com/v1alpha1",
							Kind:               "TidbMonitor",
							Name:               "foo",
							Controller:         &varTrue,
							BlockOwnerDeletion: &varTrue,
						},
					},
				},
				Data: nil, // tests are in template_test.go
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cm, err := getMonitorConfigMap(&tt.cluster, &tt.monitor)
			g.Expect(err).NotTo(HaveOccurred())
			if tt.expected == nil {
				g.Expect(cm).To(BeNil())
				return
			}
			cm.Data = nil
			if diff := cmp.Diff(&tt.expected, &cm); diff != "" {
				t.Errorf("unexpected plugin configuration (-want, +got): %s", diff)
			}
		})
	}
}

func TestGetMonitorSecret(t *testing.T) {
	g := NewGomegaWithT(t)
	varTrue := true

	testCases := []struct {
		name     string
		monitor  v1alpha1.TidbMonitor
		expected *corev1.Secret
	}{
		{
			name: "basic",
			monitor: v1alpha1.TidbMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbMonitorSpec{
					Grafana: &v1alpha1.GrafanaSpec{
						Username: "admin",
						Password: "passwd",
					},
				},
			},
			expected: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-monitor",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "monitor",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "pingcap.com/v1alpha1",
							Kind:               "TidbMonitor",
							Name:               "foo",
							Controller:         &varTrue,
							BlockOwnerDeletion: &varTrue,
						},
					},
				},
				Data: map[string][]byte{
					"username": []byte("admin"),
					"password": []byte("passwd"),
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			sec := getMonitorSecret(&tt.monitor)
			if tt.expected == nil {
				g.Expect(sec).To(BeNil())
				return
			}
			if diff := cmp.Diff(&tt.expected, &sec); diff != "" {
				t.Errorf("unexpected plugin configuration (-want, +got): %s", diff)
			}
		})
	}
}

func TestGetMonitorServiceAccount(t *testing.T) {
	g := NewGomegaWithT(t)

	testCases := []struct {
		name     string
		monitor  v1alpha1.TidbMonitor
		expected *corev1.ServiceAccount
	}{
		{
			name: "basic",
			monitor: v1alpha1.TidbMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
			},
			expected: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-monitor",
					Namespace: "ns",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "tidb-cluster",
						"app.kubernetes.io/managed-by": "tidb-operator",
						"app.kubernetes.io/instance":   "foo",
						"app.kubernetes.io/component":  "monitor",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "pingcap.com/v1alpha1",
							Kind:               "TidbMonitor",
							Name:               "foo",
							Controller:         pointer.BoolPtr(true),
							BlockOwnerDeletion: pointer.BoolPtr(true),
						},
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			sa := getMonitorServiceAccount(&tt.monitor)
			if tt.expected == nil {
				g.Expect(sa).To(BeNil())
				return
			}
			if diff := cmp.Diff(&tt.expected, &sa); diff != "" {
				t.Errorf("unexpected plugin configuration (-want, +got): %s", diff)
			}
		})
	}
}

func TestGetMonitorService(t *testing.T) {
	g := NewGomegaWithT(t)
	testCases := []struct {
		name     string
		monitor  v1alpha1.TidbMonitor
		expected []*corev1.Service
	}{
		{
			name: "basic",
			monitor: v1alpha1.TidbMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
			},
			expected: []*corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-prometheus",
						Namespace: "ns",
						Labels: map[string]string{
							"app.kubernetes.io/name":       "tidb-cluster",
							"app.kubernetes.io/managed-by": "tidb-operator",
							"app.kubernetes.io/instance":   "foo",
							"app.kubernetes.io/component":  "monitor",
							"app.kubernetes.io/used-by":    "prometheus",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "pingcap.com/v1alpha1",
								Kind:       "TidbMonitor",
								Name:       "foo",
								UID:        "",
								Controller: func(b bool) *bool {
									return &b
								}(true),
								BlockOwnerDeletion: func(b bool) *bool {
									return &b
								}(true),
							},
						},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:       "http-prometheus",
								Protocol:   "TCP",
								Port:       9090,
								TargetPort: intstr.IntOrString{IntVal: 9090},
							},
						},
						Selector: map[string]string{
							"app.kubernetes.io/component": "monitor",
							"app.kubernetes.io/instance":  "foo",
							"app.kubernetes.io/name":      "tidb-cluster",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-monitor-reloader",
						Namespace: "ns",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "monitor",
							"app.kubernetes.io/instance":   "foo",
							"app.kubernetes.io/managed-by": "tidb-operator",
							"app.kubernetes.io/name":       "tidb-cluster",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "pingcap.com/v1alpha1",
								Kind:       "TidbMonitor",
								Name:       "foo",
								Controller: func(b bool) *bool {
									return &b
								}(true),
								BlockOwnerDeletion: func(b bool) *bool {
									return &b
								}(true),
							},
						},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:       "tcp-reloader",
								Port:       9089,
								Protocol:   corev1.ProtocolTCP,
								TargetPort: intstr.FromInt(9089),
							},
						},
						Selector: map[string]string{
							"app.kubernetes.io/component": "monitor",
							"app.kubernetes.io/instance":  "foo",
							"app.kubernetes.io/name":      "tidb-cluster",
						},
					},
				},
			},
		},
		{
			name: "TidbMonitor service in typical public cloud",
			monitor: v1alpha1.TidbMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbMonitorSpec{
					Prometheus: v1alpha1.PrometheusSpec{
						Service: v1alpha1.ServiceSpec{
							Type:           corev1.ServiceTypeLoadBalancer,
							LoadBalancerIP: pointer.StringPtr("78.11.24.19"),
							LoadBalancerSourceRanges: []string{
								"10.0.0.0/8",
								"130.211.204.1/32",
							},
						},
					},
					Reloader: v1alpha1.ReloaderSpec{
						Service: v1alpha1.ServiceSpec{
							Type:           corev1.ServiceTypeLoadBalancer,
							LoadBalancerIP: pointer.StringPtr("78.11.24.19"),
							LoadBalancerSourceRanges: []string{
								"10.0.0.0/8",
								"130.211.204.1/32",
							},
						},
					},
					Grafana: &v1alpha1.GrafanaSpec{
						Service: v1alpha1.ServiceSpec{
							Type:           corev1.ServiceTypeLoadBalancer,
							LoadBalancerIP: pointer.StringPtr("78.11.24.19"),
							LoadBalancerSourceRanges: []string{
								"10.0.0.0/8",
								"130.211.204.1/32",
							},
						},
					},
				},
			},
			expected: []*corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-prometheus",
						Namespace: "ns",
						Labels: map[string]string{
							"app.kubernetes.io/name":       "tidb-cluster",
							"app.kubernetes.io/managed-by": "tidb-operator",
							"app.kubernetes.io/instance":   "foo",
							"app.kubernetes.io/component":  "monitor",
							"app.kubernetes.io/used-by":    "prometheus",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "pingcap.com/v1alpha1",
								Kind:       "TidbMonitor",
								Name:       "foo",
								UID:        "",
								Controller: func(b bool) *bool {
									return &b
								}(true),
								BlockOwnerDeletion: func(b bool) *bool {
									return &b
								}(true),
							},
						},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:       "http-prometheus",
								Protocol:   "TCP",
								Port:       9090,
								TargetPort: intstr.IntOrString{IntVal: 9090},
							},
						},
						Selector: map[string]string{
							"app.kubernetes.io/component": "monitor",
							"app.kubernetes.io/instance":  "foo",
							"app.kubernetes.io/name":      "tidb-cluster",
						},
						Type:           "LoadBalancer",
						LoadBalancerIP: "78.11.24.19",
						LoadBalancerSourceRanges: []string{
							"10.0.0.0/8",
							"130.211.204.1/32",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-monitor-reloader",
						Namespace: "ns",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "monitor",
							"app.kubernetes.io/instance":   "foo",
							"app.kubernetes.io/managed-by": "tidb-operator",
							"app.kubernetes.io/name":       "tidb-cluster",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "pingcap.com/v1alpha1",
								Kind:       "TidbMonitor",
								Name:       "foo",
								Controller: func(b bool) *bool {
									return &b
								}(true),
								BlockOwnerDeletion: func(b bool) *bool {
									return &b
								}(true),
							},
						},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:       "tcp-reloader",
								Port:       9089,
								Protocol:   corev1.ProtocolTCP,
								TargetPort: intstr.FromInt(9089),
							},
						},
						Selector: map[string]string{
							"app.kubernetes.io/component": "monitor",
							"app.kubernetes.io/instance":  "foo",
							"app.kubernetes.io/name":      "tidb-cluster",
						},
						Type:           "LoadBalancer",
						LoadBalancerIP: "78.11.24.19",
						LoadBalancerSourceRanges: []string{
							"10.0.0.0/8",
							"130.211.204.1/32",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-grafana",
						Namespace: "ns",
						Labels: map[string]string{
							"app.kubernetes.io/name":       "tidb-cluster",
							"app.kubernetes.io/managed-by": "tidb-operator",
							"app.kubernetes.io/instance":   "foo",
							"app.kubernetes.io/component":  "monitor",
							"app.kubernetes.io/used-by":    "grafana",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "pingcap.com/v1alpha1",
								Kind:       "TidbMonitor",
								Name:       "foo",
								UID:        "",
								Controller: func(b bool) *bool {
									return &b
								}(true),
								BlockOwnerDeletion: func(b bool) *bool {
									return &b
								}(true),
							},
						},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:       "http-grafana",
								Protocol:   "TCP",
								Port:       3000,
								TargetPort: intstr.IntOrString{IntVal: 3000},
							},
						},
						Selector: map[string]string{
							"app.kubernetes.io/component": "monitor",
							"app.kubernetes.io/instance":  "foo",
							"app.kubernetes.io/name":      "tidb-cluster",
						},
						Type:           "LoadBalancer",
						LoadBalancerIP: "78.11.24.19",
						LoadBalancerSourceRanges: []string{
							"10.0.0.0/8",
							"130.211.204.1/32",
						},
					},
				},
			},
		},
		{
			name: "tidb monitor with grafana",
			monitor: v1alpha1.TidbMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbMonitorSpec{
					Grafana: &v1alpha1.GrafanaSpec{
						Service: v1alpha1.ServiceSpec{
							Type: corev1.ServiceTypeClusterIP,
						},
					},
				},
			},
			expected: []*corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-prometheus",
						Namespace: "ns",
						Labels: map[string]string{
							"app.kubernetes.io/name":       "tidb-cluster",
							"app.kubernetes.io/managed-by": "tidb-operator",
							"app.kubernetes.io/instance":   "foo",
							"app.kubernetes.io/component":  "monitor",
							"app.kubernetes.io/used-by":    "prometheus",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "pingcap.com/v1alpha1",
								Kind:       "TidbMonitor",
								Name:       "foo",
								UID:        "",
								Controller: func(b bool) *bool {
									return &b
								}(true),
								BlockOwnerDeletion: func(b bool) *bool {
									return &b
								}(true),
							},
						},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:       "http-prometheus",
								Protocol:   "TCP",
								Port:       9090,
								TargetPort: intstr.IntOrString{IntVal: 9090},
							},
						},
						Selector: map[string]string{
							"app.kubernetes.io/component": "monitor",
							"app.kubernetes.io/instance":  "foo",
							"app.kubernetes.io/name":      "tidb-cluster",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-monitor-reloader",
						Namespace: "ns",
						Labels: map[string]string{
							"app.kubernetes.io/component":  "monitor",
							"app.kubernetes.io/instance":   "foo",
							"app.kubernetes.io/managed-by": "tidb-operator",
							"app.kubernetes.io/name":       "tidb-cluster",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "pingcap.com/v1alpha1",
								Kind:       "TidbMonitor",
								Name:       "foo",
								Controller: func(b bool) *bool {
									return &b
								}(true),
								BlockOwnerDeletion: func(b bool) *bool {
									return &b
								}(true),
							},
						},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:       "tcp-reloader",
								Port:       9089,
								Protocol:   corev1.ProtocolTCP,
								TargetPort: intstr.FromInt(9089),
							},
						},
						Selector: map[string]string{
							"app.kubernetes.io/component": "monitor",
							"app.kubernetes.io/instance":  "foo",
							"app.kubernetes.io/name":      "tidb-cluster",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-grafana",
						Namespace: "ns",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "pingcap.com/v1alpha1",
								Kind:       "TidbMonitor",
								Name:       "foo",
								Controller: func(b bool) *bool {
									return &b
								}(true),
								BlockOwnerDeletion: func(b bool) *bool {
									return &b
								}(true),
							},
						},
						Labels: map[string]string{
							"app.kubernetes.io/component":  "monitor",
							"app.kubernetes.io/instance":   "foo",
							"app.kubernetes.io/managed-by": "tidb-operator",
							"app.kubernetes.io/name":       "tidb-cluster",
							"app.kubernetes.io/used-by":    "grafana",
						},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Name:       "http-grafana",
								Port:       3000,
								Protocol:   corev1.ProtocolTCP,
								TargetPort: intstr.FromInt(3000),
							},
						},
						Type: "ClusterIP",
						Selector: map[string]string{
							"app.kubernetes.io/component": "monitor",
							"app.kubernetes.io/instance":  "foo",
							"app.kubernetes.io/name":      "tidb-cluster",
						},
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			svc := getMonitorService(&tt.monitor)
			if tt.expected == nil {
				g.Expect(svc).To(BeNil())
				return
			}
			if diff := cmp.Diff(&tt.expected, &svc); diff != "" {
				t.Errorf("unexpected service configuration (-want, +got): %s", diff)
			}
		})
	}
}

func TestGetMonitorVolumes(t *testing.T) {
	g := NewGomegaWithT(t)

	testCases := []struct {
		name     string
		cluster  v1alpha1.TidbCluster
		monitor  v1alpha1.TidbMonitor
		expected []corev1.Volume
	}{
		{
			name: "basic",
			cluster: v1alpha1.TidbCluster{
				Spec: v1alpha1.TidbClusterSpec{
					TLSCluster: &v1alpha1.TLSCluster{Enabled: false},
				},
			},
			monitor: v1alpha1.TidbMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
			},
			expected: []corev1.Volume{
				corev1.Volume{
					Name: "monitor-data",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				corev1.Volume{
					Name: "prometheus-config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "foo-monitor",
							},
							Items: []corev1.KeyToPath{
								corev1.KeyToPath{
									Key:  "prometheus-config",
									Path: "prometheus.yml",
								},
							},
						},
					},
				},
				corev1.Volume{
					Name: "prometheus-rules",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
		{
			name: "tls and persistent",
			cluster: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TLSCluster: &v1alpha1.TLSCluster{Enabled: true},
				},
			},
			monitor: v1alpha1.TidbMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbMonitorSpec{
					Persistent: true,
				},
			},
			expected: []corev1.Volume{
				corev1.Volume{
					Name: "monitor-data",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "foo-monitor",
							ReadOnly:  false,
						},
					},
				},
				corev1.Volume{
					Name: "prometheus-config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "foo-monitor",
							},
							Items: []corev1.KeyToPath{
								corev1.KeyToPath{
									Key:  "prometheus-config",
									Path: "prometheus.yml",
								},
							},
						},
					},
				},
				corev1.Volume{
					Name: "prometheus-rules",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				corev1.Volume{
					Name: "cluster-client-tls",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  "foo-cluster-client-secret",
							DefaultMode: pointer.Int32Ptr(420),
						},
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cm, err := getMonitorConfigMap(&tt.cluster, &tt.monitor)
			g.Expect(err).NotTo(HaveOccurred())
			sa := getMonitorVolumes(cm, &tt.monitor, &tt.cluster)
			if tt.expected == nil {
				g.Expect(sa).To(BeNil())
				return
			}
			if diff := cmp.Diff(&tt.expected, &sa); diff != "" {
				t.Errorf("unexpected volume configuration (-want, +got): %s", diff)
			}
		})
	}
}

func TestGetMonitorPrometheusContainer(t *testing.T) {
	g := NewGomegaWithT(t)

	testCases := []struct {
		name     string
		cluster  v1alpha1.TidbCluster
		monitor  v1alpha1.TidbMonitor
		expected *corev1.Container
	}{
		{
			name: "basic",
			cluster: v1alpha1.TidbCluster{
				Spec: v1alpha1.TidbClusterSpec{
					TLSCluster: &v1alpha1.TLSCluster{Enabled: true},
				},
			},
			monitor: v1alpha1.TidbMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbMonitorSpec{
					Prometheus: v1alpha1.PrometheusSpec{
						MonitorContainer: v1alpha1.MonitorContainer{
							BaseImage: "hub.pingcap.net",
							Version:   "latest",
						},
					},
				},
			},
			expected: &corev1.Container{
				Name:  "prometheus",
				Image: "hub.pingcap.net:latest",
				Command: []string{
					"/bin/prometheus",
					"--web.enable-admin-api",
					"--web.enable-lifecycle",
					"--config.file=/etc/prometheus/prometheus.yml",
					"--storage.tsdb.path=/data/prometheus",
					"--storage.tsdb.retention=0d",
				},
				Ports: []corev1.ContainerPort{
					corev1.ContainerPort{
						Name:          "prometheus",
						ContainerPort: 9090,
						Protocol:      "TCP",
					},
				},
				Env: []corev1.EnvVar{
					corev1.EnvVar{
						Name: "TZ",
					},
				},
				Resources: corev1.ResourceRequirements{},
				VolumeMounts: []corev1.VolumeMount{
					corev1.VolumeMount{
						Name:      "prometheus-config",
						ReadOnly:  true,
						MountPath: "/etc/prometheus",
					},
					corev1.VolumeMount{
						Name:      "monitor-data",
						ReadOnly:  false,
						MountPath: "/data",
					},
					corev1.VolumeMount{
						Name:      "prometheus-rules",
						ReadOnly:  false,
						MountPath: "/prometheus-rules",
					},
					{
						Name:      "cluster-client-tls",
						ReadOnly:  true,
						MountPath: "/var/lib/cluster-client-tls",
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			sa := getMonitorPrometheusContainer(&tt.monitor, &tt.cluster)
			if tt.expected == nil {
				g.Expect(sa).To(BeNil())
				return
			}
			if diff := cmp.Diff(tt.expected, &sa); diff != "" {
				t.Errorf("unexpected plugin configuration (-want, +got): %s", diff)
			}
		})
	}
}

func TestGetMonitorGrafanaContainer(t *testing.T) {
	g := NewGomegaWithT(t)

	testCases := []struct {
		name     string
		secret   corev1.Secret
		cluster  v1alpha1.TidbCluster
		monitor  v1alpha1.TidbMonitor
		expected *corev1.Container
	}{
		{
			name: "basic",
			secret: corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
			},
			cluster: v1alpha1.TidbCluster{
				Spec: v1alpha1.TidbClusterSpec{
					TLSCluster: &v1alpha1.TLSCluster{Enabled: true},
				},
			},
			monitor: v1alpha1.TidbMonitor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "ns",
				},
				Spec: v1alpha1.TidbMonitorSpec{
					Grafana: &v1alpha1.GrafanaSpec{
						MonitorContainer: v1alpha1.MonitorContainer{
							BaseImage: "hub.pingcap.net",
							Version:   "latest",
						},
					},
				},
			},
			expected: &corev1.Container{
				Name:  "grafana",
				Image: "hub.pingcap.net:latest",
				Ports: []corev1.ContainerPort{
					corev1.ContainerPort{
						Name:          "grafana",
						ContainerPort: 3000,
						Protocol:      "TCP",
					},
				},
				Env: []corev1.EnvVar{
					corev1.EnvVar{
						Name:  "GF_PATHS_DATA",
						Value: "/data/grafana",
					},
					corev1.EnvVar{
						Name: "GF_SECURITY_ADMIN_PASSWORD",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "foo",
								},
								Key: "password",
							},
						},
					},
					corev1.EnvVar{
						Name: "GF_SECURITY_ADMIN_USER",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "foo",
								},
								Key: "username",
							},
						},
					},
					corev1.EnvVar{Name: "TZ", Value: "UTC"},
				},
				Resources: corev1.ResourceRequirements{},
				VolumeMounts: []corev1.VolumeMount{
					corev1.VolumeMount{
						Name:      "monitor-data",
						ReadOnly:  false,
						MountPath: "/data",
					},
					corev1.VolumeMount{
						Name:      "datasource",
						ReadOnly:  false,
						MountPath: "/etc/grafana/provisioning/datasources",
					},
					corev1.VolumeMount{
						Name:      "dashboards-provisioning",
						ReadOnly:  false,
						MountPath: "/etc/grafana/provisioning/dashboards",
					},
					corev1.VolumeMount{
						Name:      "grafana-dashboard",
						MountPath: "/grafana-dashboard-definitions/tidb",
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			sa := getMonitorGrafanaContainer(&tt.secret, &tt.monitor, &tt.cluster)
			if tt.expected == nil {
				g.Expect(sa).To(BeNil())
				return
			}
			if diff := cmp.Diff(tt.expected, &sa); diff != "" {
				t.Errorf("unexpected plugin configuration (-want, +got): %s", diff)
			}
		})
	}
}
