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

package fixture

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

var (
	BestEffort    = corev1.ResourceRequirements{}
	BurstbleSmall = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1000m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
	}
	BurstbleMedium = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2000m"),
			corev1.ResourceMemory: resource.MustParse("4Gi"),
		},
	}
)

func WithStorage(r corev1.ResourceRequirements, size string) corev1.ResourceRequirements {
	if r.Requests == nil {
		r.Requests = corev1.ResourceList{}
	}
	r.Requests[corev1.ResourceStorage] = resource.MustParse(size)

	return r
}

// GetTidbCluster returns a TidbCluster resource configured for testing
func GetTidbCluster(ns, name, version string) *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: v1alpha1.TidbClusterSpec{
			Version:         version,
			ImagePullPolicy: corev1.PullIfNotPresent,
			PVReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
			SchedulerName:   "tidb-scheduler",
			Timezone:        "Asia/Shanghai",

			PD: v1alpha1.PDSpec{
				Replicas:             3,
				BaseImage:            "pingcap/pd",
				ResourceRequirements: WithStorage(BurstbleSmall, "1Gi"),
				Config: &v1alpha1.PDConfig{
					Log: &v1alpha1.PDLogConfig{
						Level: "info",
					},
					// accelerate failover
					Schedule: &v1alpha1.PDScheduleConfig{
						MaxStoreDownTime: "5m",
					},
				},
			},

			TiKV: v1alpha1.TiKVSpec{
				Replicas:             3,
				BaseImage:            "pingcap/tikv",
				ResourceRequirements: WithStorage(BurstbleMedium, "10Gi"),
				MaxFailoverCount:     pointer.Int32Ptr(3),
				Config: &v1alpha1.TiKVConfig{
					LogLevel: "info",
					Server:   &v1alpha1.TiKVServerConfig{},
				},
			},

			TiDB: v1alpha1.TiDBSpec{
				Replicas:             2,
				BaseImage:            "pingcap/tidb",
				ResourceRequirements: BurstbleMedium,
				Service: &v1alpha1.TiDBServiceSpec{
					ServiceSpec: v1alpha1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
					},
					ExposeStatus: pointer.BoolPtr(true),
				},
				SeparateSlowLog:  pointer.BoolPtr(true),
				MaxFailoverCount: pointer.Int32Ptr(3),
				Config: &v1alpha1.TiDBConfig{
					Log: &v1alpha1.Log{
						Level: pointer.StringPtr("info"),
					},
				},
			},
		},
	}
}

func NewTidbMonitor(name, namespace string, tc *v1alpha1.TidbCluster, grafanaEnabled, persist bool) *v1alpha1.TidbMonitor {
	imagePullPolicy := corev1.PullIfNotPresent
	monitor := &v1alpha1.TidbMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.TidbMonitorSpec{
			Clusters: []v1alpha1.TidbClusterRef{
				{
					Name:      tc.Name,
					Namespace: tc.Namespace,
				},
			},
			Prometheus: v1alpha1.PrometheusSpec{
				ReserveDays: 7,
				LogLevel:    "info",
				Service: v1alpha1.ServiceSpec{
					Type:        "ClusterIP",
					Annotations: map[string]string{},
				},
				MonitorContainer: v1alpha1.MonitorContainer{
					BaseImage:       "prom/prometheus",
					Version:         "v2.11.1",
					ImagePullPolicy: &imagePullPolicy,
					Resources:       corev1.ResourceRequirements{},
				},
			},
			Reloader: v1alpha1.ReloaderSpec{
				MonitorContainer: v1alpha1.MonitorContainer{
					BaseImage:       "pingcap/tidb-monitor-reloader",
					Version:         "v1.0.1",
					ImagePullPolicy: &imagePullPolicy,
					Resources:       corev1.ResourceRequirements{},
				},
				Service: v1alpha1.ServiceSpec{
					Type:        "ClusterIP",
					Annotations: map[string]string{},
				},
			},
			Initializer: v1alpha1.InitializerSpec{
				MonitorContainer: v1alpha1.MonitorContainer{
					BaseImage:       "pingcap/tidb-monitor-initializer",
					Version:         "v3.0.8",
					ImagePullPolicy: &imagePullPolicy,
					Resources:       corev1.ResourceRequirements{},
				},
				Envs: map[string]string{},
			},
			Persistent: persist,
		},
	}
	if grafanaEnabled {
		monitor.Spec.Grafana = &v1alpha1.GrafanaSpec{
			MonitorContainer: v1alpha1.MonitorContainer{
				BaseImage:       "grafana/grafana",
				Version:         "6.0.1",
				ImagePullPolicy: &imagePullPolicy,
				Resources:       corev1.ResourceRequirements{},
			},
			Username: "admin",
			Password: "admin",
			Service: v1alpha1.ServiceSpec{
				Type:        corev1.ServiceTypeClusterIP,
				Annotations: map[string]string{},
			},
			Envs: map[string]string{
				"A":    "B",
				"foo":  "hello",
				"bar":  "query",
				"some": "any",
			},
		}
	}
	if persist {
		storageClassName := "local-storage"
		monitor.Spec.StorageClassName = &storageClassName
		monitor.Spec.Storage = "2Gi"
	}
	return monitor
}
