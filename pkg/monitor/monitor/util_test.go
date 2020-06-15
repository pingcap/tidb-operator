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
	core "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

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
							"kubernetes.io/name":           "foo-prometheus",
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
						Ports: []core.ServicePort{
							{
								Name:       "tcp-reloader",
								Port:       9089,
								Protocol:   core.ProtocolTCP,
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
							"kubernetes.io/name":           "foo-prometheus",
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
						Ports: []core.ServicePort{
							{
								Name:       "tcp-reloader",
								Port:       9089,
								Protocol:   core.ProtocolTCP,
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
							"kubernetes.io/name":           "foo-prometheus",
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
						Ports: []core.ServicePort{
							{
								Name:       "tcp-reloader",
								Port:       9089,
								Protocol:   core.ProtocolTCP,
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
							"kubernetes.io/name":           "foo-grafana",
						},
					},
					Spec: core.ServiceSpec{
						Ports: []core.ServicePort{
							{
								Name:       "http-grafana",
								Port:       3000,
								Protocol:   core.ProtocolTCP,
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
				t.Errorf("unexpected plugin configuration (-want, +got): %s", diff)
			}
		})
	}
}
