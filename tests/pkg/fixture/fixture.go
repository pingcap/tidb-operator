// Copyright 2019. PingCAP, Inc.
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
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("200Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1000m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
	}
	BurstbleMedium = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("200Mi"),
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
				Replicas: 3,
				ComponentSpec: v1alpha1.ComponentSpec{
					BaseImage: "pingcap/pd",
				},
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
				Replicas: 3,
				ComponentSpec: v1alpha1.ComponentSpec{
					BaseImage: "pingcap/tikv",
				},
				ResourceRequirements: WithStorage(BurstbleMedium, "10Gi"),
				MaxFailoverCount:     3,
				Config: &v1alpha1.TiKVConfig{
					LogLevel: "info",
					Server:   &v1alpha1.TiKVServerConfig{},
				},
			},

			TiDB: v1alpha1.TiDBSpec{
				Replicas: 2,
				ComponentSpec: v1alpha1.ComponentSpec{
					BaseImage: "pingcap/tidb",
				},
				ResourceRequirements: BurstbleMedium,
				Service: &v1alpha1.TiDBServiceSpec{
					ServiceSpec: v1alpha1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
					},
					ExposeStatus: true,
				},
				SeparateSlowLog:  true,
				MaxFailoverCount: 3,
				Config: &v1alpha1.TiDBConfig{
					Log: &v1alpha1.Log{
						Level: pointer.StringPtr("info"),
					},
				},
			},
		},
	}
}
