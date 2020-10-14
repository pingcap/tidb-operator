// Copyright 2020 PingCAP, Inc.
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

package member

import (
	"encoding/json"
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/util/toml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

var (
	defaultTiFlashConfig = v1alpha1.TiFlashConfig{
		CommonConfig: &v1alpha1.CommonConfig{
			FlashApplication: &v1alpha1.FlashApplication{
				RunAsDaemon: pointer.BoolPtr(true),
			},
			DefaultProfile: pointer.StringPtr("default"),
			DisplayName:    pointer.StringPtr("TiFlash"),
			Flash: &v1alpha1.Flash{
				CompactLogMinPeriod: pointer.Int32Ptr(200),
				FlashCluster: &v1alpha1.FlashCluster{
					ClusterManagerPath: pointer.StringPtr("/tiflash/flash_cluster_manager"),
					ClusterLog:         pointer.StringPtr("/data0/logs/flash_cluster_manager.log"),
					MasterTTL:          pointer.Int32Ptr(60),
					RefreshInterval:    pointer.Int32Ptr(20),
					UpdateRuleInterval: pointer.Int32Ptr(10),
				},
				OverlapThreshold: pointer.Float64Ptr(0.6),
				FlashProxy: &v1alpha1.FlashProxy{
					Addr:          pointer.StringPtr("0.0.0.0:20170"),
					AdvertiseAddr: pointer.StringPtr("test-tiflash-POD_NUM.test-tiflash-peer.test.svc:20170"),
					Config:        pointer.StringPtr("/data0/proxy.toml"),
					DataDir:       pointer.StringPtr("/data0/proxy"),
				},
				ServiceAddr:    pointer.StringPtr("0.0.0.0:3930"),
				TiDBStatusAddr: pointer.StringPtr("test-tidb.test.svc:10080"),
			},
			HTTPPort:               pointer.Int32Ptr(8123),
			HTTPSPort:              pointer.Int32Ptr(8123),
			InternalServerHTTPPort: pointer.Int32Ptr(9009),
			ListenHost:             pointer.StringPtr("0.0.0.0"),
			FlashLogger: &v1alpha1.FlashLogger{
				Count:     pointer.Int32Ptr(10),
				ErrorLog:  pointer.StringPtr("/data0/logs/error.log"),
				Level:     pointer.StringPtr("information"),
				ServerLog: pointer.StringPtr("/data0/logs/server.log"),
				Size:      pointer.StringPtr("100M"),
			},
			MarkCacheSize:        pointer.Int64Ptr(5368709120),
			MinmaxIndexCacheSize: pointer.Int64Ptr(5368709120),
			FlashDataPath:        pointer.StringPtr("/data0/db"),
			PathRealtimeMode:     pointer.BoolPtr(false),
			FlashProfile: &v1alpha1.FlashProfile{
				Default: &v1alpha1.Profile{
					LoadBalancing:        pointer.StringPtr("random"),
					MaxMemoryUsage:       pointer.Int64Ptr(10000000000),
					UseUncompressedCache: pointer.Int32Ptr(0),
				},
				Readonly: &v1alpha1.Profile{
					Readonly: pointer.Int32Ptr(1),
				},
			},
			FlashQuota: &v1alpha1.FlashQuota{
				Default: &v1alpha1.Quota{
					Interval: &v1alpha1.Interval{
						Duration:      pointer.Int32Ptr(3600),
						Errors:        pointer.Int32Ptr(0),
						ExecutionTime: pointer.Int32Ptr(0),
						Queries:       pointer.Int32Ptr(0),
						ReadRows:      pointer.Int32Ptr(0),
						ResultRows:    pointer.Int32Ptr(0),
					},
				},
			},
			FlashRaft: &v1alpha1.FlashRaft{
				KVStorePath:   pointer.StringPtr("/data0/kvstore"),
				PDAddr:        pointer.StringPtr("test-pd.test.svc:2379"),
				StorageEngine: pointer.StringPtr("dt"),
			},
			FlashStatus: &v1alpha1.FlashStatus{
				MetricsPort: pointer.Int32Ptr(8234),
			},
			TCPPort:       pointer.Int32Ptr(9000),
			TCPPortSecure: pointer.Int32Ptr(9000),
			TmpPath:       pointer.StringPtr("/data0/tmp"),
			FlashUser: &v1alpha1.FlashUser{
				Default: &v1alpha1.User{
					Networks: &v1alpha1.Networks{
						IP: pointer.StringPtr("::/0"),
					},
					Profile: pointer.StringPtr("default"),
					Quota:   pointer.StringPtr("default"),
				},
				Readonly: &v1alpha1.User{
					Networks: &v1alpha1.Networks{
						IP: pointer.StringPtr("::/0"),
					},
					Profile: pointer.StringPtr("readonly"),
					Quota:   pointer.StringPtr("default"),
				},
			},
		},
		ProxyConfig: &v1alpha1.ProxyConfig{
			LogLevel: pointer.StringPtr("info"),
			Server: &v1alpha1.FlashServerConfig{
				EngineAddr:          pointer.StringPtr("test-tiflash-POD_NUM.test-tiflash-peer.test.svc:3930"),
				StatusAddr:          pointer.StringPtr("0.0.0.0:20292"),
				AdvertiseStatusAddr: pointer.StringPtr("test-tiflash-POD_NUM.test-tiflash-peer.test.svc:20292"),
			},
		},
	}
	defaultTiFlashNonTLSConfig = v1alpha1.TiFlashConfig{
		CommonConfig: &v1alpha1.CommonConfig{
			FlashApplication: &v1alpha1.FlashApplication{
				RunAsDaemon: pointer.BoolPtr(true),
			},
			DefaultProfile: pointer.StringPtr("default"),
			DisplayName:    pointer.StringPtr("TiFlash"),
			Flash: &v1alpha1.Flash{
				CompactLogMinPeriod: pointer.Int32Ptr(200),
				FlashCluster: &v1alpha1.FlashCluster{
					ClusterManagerPath: pointer.StringPtr("/tiflash/flash_cluster_manager"),
					ClusterLog:         pointer.StringPtr("/data0/logs/flash_cluster_manager.log"),
					MasterTTL:          pointer.Int32Ptr(60),
					RefreshInterval:    pointer.Int32Ptr(20),
					UpdateRuleInterval: pointer.Int32Ptr(10),
				},
				OverlapThreshold: pointer.Float64Ptr(0.6),
				FlashProxy: &v1alpha1.FlashProxy{
					Addr:          pointer.StringPtr("0.0.0.0:20170"),
					AdvertiseAddr: pointer.StringPtr("test-tiflash-POD_NUM.test-tiflash-peer.test.svc:20170"),
					Config:        pointer.StringPtr("/data0/proxy.toml"),
					DataDir:       pointer.StringPtr("/data0/proxy"),
				},
				ServiceAddr:    pointer.StringPtr("0.0.0.0:3930"),
				TiDBStatusAddr: pointer.StringPtr("test-tidb.test.svc:10080"),
			},
			HTTPPort:               pointer.Int32Ptr(8123),
			InternalServerHTTPPort: pointer.Int32Ptr(9009),
			ListenHost:             pointer.StringPtr("0.0.0.0"),
			FlashLogger: &v1alpha1.FlashLogger{
				Count:     pointer.Int32Ptr(10),
				ErrorLog:  pointer.StringPtr("/data0/logs/error.log"),
				Level:     pointer.StringPtr("information"),
				ServerLog: pointer.StringPtr("/data0/logs/server.log"),
				Size:      pointer.StringPtr("100M"),
			},
			MarkCacheSize:        pointer.Int64Ptr(5368709120),
			MinmaxIndexCacheSize: pointer.Int64Ptr(5368709120),
			FlashDataPath:        pointer.StringPtr("/data0/db"),
			PathRealtimeMode:     pointer.BoolPtr(false),
			FlashProfile: &v1alpha1.FlashProfile{
				Default: &v1alpha1.Profile{
					LoadBalancing:        pointer.StringPtr("random"),
					MaxMemoryUsage:       pointer.Int64Ptr(10000000000),
					UseUncompressedCache: pointer.Int32Ptr(0),
				},
				Readonly: &v1alpha1.Profile{
					Readonly: pointer.Int32Ptr(1),
				},
			},
			FlashQuota: &v1alpha1.FlashQuota{
				Default: &v1alpha1.Quota{
					Interval: &v1alpha1.Interval{
						Duration:      pointer.Int32Ptr(3600),
						Errors:        pointer.Int32Ptr(0),
						ExecutionTime: pointer.Int32Ptr(0),
						Queries:       pointer.Int32Ptr(0),
						ReadRows:      pointer.Int32Ptr(0),
						ResultRows:    pointer.Int32Ptr(0),
					},
				},
			},
			FlashRaft: &v1alpha1.FlashRaft{
				KVStorePath:   pointer.StringPtr("/data0/kvstore"),
				PDAddr:        pointer.StringPtr("test-pd.test.svc:2379"),
				StorageEngine: pointer.StringPtr("dt"),
			},
			FlashStatus: &v1alpha1.FlashStatus{
				MetricsPort: pointer.Int32Ptr(8234),
			},
			TCPPort: pointer.Int32Ptr(9000),
			TmpPath: pointer.StringPtr("/data0/tmp"),
			FlashUser: &v1alpha1.FlashUser{
				Default: &v1alpha1.User{
					Networks: &v1alpha1.Networks{
						IP: pointer.StringPtr("::/0"),
					},
					Profile: pointer.StringPtr("default"),
					Quota:   pointer.StringPtr("default"),
				},
				Readonly: &v1alpha1.User{
					Networks: &v1alpha1.Networks{
						IP: pointer.StringPtr("::/0"),
					},
					Profile: pointer.StringPtr("readonly"),
					Quota:   pointer.StringPtr("default"),
				},
			},
		},
		ProxyConfig: &v1alpha1.ProxyConfig{
			LogLevel: pointer.StringPtr("info"),
			Server: &v1alpha1.FlashServerConfig{
				EngineAddr:          pointer.StringPtr("test-tiflash-POD_NUM.test-tiflash-peer.test.svc:3930"),
				StatusAddr:          pointer.StringPtr("0.0.0.0:20292"),
				AdvertiseStatusAddr: pointer.StringPtr("test-tiflash-POD_NUM.test-tiflash-peer.test.svc:20292"),
			},
		},
	}
	defaultTiFlashTLSConfig = v1alpha1.TiFlashConfig{
		CommonConfig: &v1alpha1.CommonConfig{
			FlashApplication: &v1alpha1.FlashApplication{
				RunAsDaemon: pointer.BoolPtr(true),
			},
			DefaultProfile: pointer.StringPtr("default"),
			DisplayName:    pointer.StringPtr("TiFlash"),
			Flash: &v1alpha1.Flash{
				CompactLogMinPeriod: pointer.Int32Ptr(200),
				FlashCluster: &v1alpha1.FlashCluster{
					ClusterManagerPath: pointer.StringPtr("/tiflash/flash_cluster_manager"),
					ClusterLog:         pointer.StringPtr("/data0/logs/flash_cluster_manager.log"),
					MasterTTL:          pointer.Int32Ptr(60),
					RefreshInterval:    pointer.Int32Ptr(20),
					UpdateRuleInterval: pointer.Int32Ptr(10),
				},
				OverlapThreshold: pointer.Float64Ptr(0.6),
				FlashProxy: &v1alpha1.FlashProxy{
					Addr:          pointer.StringPtr("0.0.0.0:20170"),
					AdvertiseAddr: pointer.StringPtr("test-tiflash-POD_NUM.test-tiflash-peer.test.svc:20170"),
					Config:        pointer.StringPtr("/data0/proxy.toml"),
					DataDir:       pointer.StringPtr("/data0/proxy"),
				},
				ServiceAddr:    pointer.StringPtr("0.0.0.0:3930"),
				TiDBStatusAddr: pointer.StringPtr("test-tidb.test.svc:10080"),
			},
			HTTPSPort:              pointer.Int32Ptr(8123),
			InternalServerHTTPPort: pointer.Int32Ptr(9009),
			ListenHost:             pointer.StringPtr("0.0.0.0"),
			FlashLogger: &v1alpha1.FlashLogger{
				Count:     pointer.Int32Ptr(10),
				ErrorLog:  pointer.StringPtr("/data0/logs/error.log"),
				Level:     pointer.StringPtr("information"),
				ServerLog: pointer.StringPtr("/data0/logs/server.log"),
				Size:      pointer.StringPtr("100M"),
			},
			MarkCacheSize:        pointer.Int64Ptr(5368709120),
			MinmaxIndexCacheSize: pointer.Int64Ptr(5368709120),
			FlashDataPath:        pointer.StringPtr("/data0/db"),
			PathRealtimeMode:     pointer.BoolPtr(false),
			FlashProfile: &v1alpha1.FlashProfile{
				Default: &v1alpha1.Profile{
					LoadBalancing:        pointer.StringPtr("random"),
					MaxMemoryUsage:       pointer.Int64Ptr(10000000000),
					UseUncompressedCache: pointer.Int32Ptr(0),
				},
				Readonly: &v1alpha1.Profile{
					Readonly: pointer.Int32Ptr(1),
				},
			},
			FlashQuota: &v1alpha1.FlashQuota{
				Default: &v1alpha1.Quota{
					Interval: &v1alpha1.Interval{
						Duration:      pointer.Int32Ptr(3600),
						Errors:        pointer.Int32Ptr(0),
						ExecutionTime: pointer.Int32Ptr(0),
						Queries:       pointer.Int32Ptr(0),
						ReadRows:      pointer.Int32Ptr(0),
						ResultRows:    pointer.Int32Ptr(0),
					},
				},
			},
			FlashRaft: &v1alpha1.FlashRaft{
				KVStorePath:   pointer.StringPtr("/data0/kvstore"),
				PDAddr:        pointer.StringPtr("test-pd.test.svc:2379"),
				StorageEngine: pointer.StringPtr("dt"),
			},
			FlashStatus: &v1alpha1.FlashStatus{
				MetricsPort: pointer.Int32Ptr(8234),
			},
			Security: &v1alpha1.FlashSecurity{
				CAPath:   pointer.StringPtr(path.Join(tiflashCertPath, corev1.ServiceAccountRootCAKey)),
				CertPath: pointer.StringPtr(path.Join(tiflashCertPath, corev1.TLSCertKey)),
				KeyPath:  pointer.StringPtr(path.Join(tiflashCertPath, corev1.TLSPrivateKeyKey)),
			},
			TCPPortSecure: pointer.Int32Ptr(9000),
			TmpPath:       pointer.StringPtr("/data0/tmp"),
			FlashUser: &v1alpha1.FlashUser{
				Default: &v1alpha1.User{
					Networks: &v1alpha1.Networks{
						IP: pointer.StringPtr("::/0"),
					},
					Profile: pointer.StringPtr("default"),
					Quota:   pointer.StringPtr("default"),
				},
				Readonly: &v1alpha1.User{
					Networks: &v1alpha1.Networks{
						IP: pointer.StringPtr("::/0"),
					},
					Profile: pointer.StringPtr("readonly"),
					Quota:   pointer.StringPtr("default"),
				},
			},
		},
		ProxyConfig: &v1alpha1.ProxyConfig{
			LogLevel: pointer.StringPtr("info"),
			Server: &v1alpha1.FlashServerConfig{
				EngineAddr:          pointer.StringPtr("test-tiflash-POD_NUM.test-tiflash-peer.test.svc:3930"),
				StatusAddr:          pointer.StringPtr("0.0.0.0:20292"),
				AdvertiseStatusAddr: pointer.StringPtr("test-tiflash-POD_NUM.test-tiflash-peer.test.svc:20292"),
			},
			Security: &v1alpha1.TiKVSecurityConfig{
				CAPath:   pointer.StringPtr(path.Join(tiflashCertPath, corev1.ServiceAccountRootCAKey)),
				CertPath: pointer.StringPtr(path.Join(tiflashCertPath, corev1.TLSCertKey)),
				KeyPath:  pointer.StringPtr(path.Join(tiflashCertPath, corev1.TLSPrivateKeyKey)),
			},
		},
	}
	customTiFlashConfig = v1alpha1.TiFlashConfig{
		CommonConfig: &v1alpha1.CommonConfig{
			FlashApplication: &v1alpha1.FlashApplication{
				RunAsDaemon: pointer.BoolPtr(false),
			},
			DefaultProfile: pointer.StringPtr("defaul"),
			DisplayName:    pointer.StringPtr("TiFlah"),
			Flash: &v1alpha1.Flash{
				CompactLogMinPeriod: pointer.Int32Ptr(100),
				FlashCluster: &v1alpha1.FlashCluster{
					ClusterManagerPath: pointer.StringPtr("/flash_cluster_manager"),
					ClusterLog:         pointer.StringPtr("/data1/logs/flash_cluster_manager.log"),
					MasterTTL:          pointer.Int32Ptr(50),
					RefreshInterval:    pointer.Int32Ptr(21),
					UpdateRuleInterval: pointer.Int32Ptr(11),
				},
				OverlapThreshold: pointer.Float64Ptr(0.7),
				FlashProxy: &v1alpha1.FlashProxy{
					Addr:          pointer.StringPtr("0.0.0.0:20171"),
					AdvertiseAddr: pointer.StringPtr("test-tiflash-POD_NUM.test-tiflash-peer.test.svc:20171"),
					Config:        pointer.StringPtr("/data0/proxy1.toml"),
					DataDir:       pointer.StringPtr("/data0/proxy1"),
				},
				ServiceAddr:    pointer.StringPtr("0.0.0.0:3930"),
				TiDBStatusAddr: pointer.StringPtr("test-tidb.test.svc:10081"),
			},
			HTTPPort:               pointer.Int32Ptr(8121),
			HTTPSPort:              pointer.Int32Ptr(9999),
			InternalServerHTTPPort: pointer.Int32Ptr(9001),
			ListenHost:             pointer.StringPtr("0.0.0.1"),
			FlashLogger: &v1alpha1.FlashLogger{
				Count:     pointer.Int32Ptr(11),
				ErrorLog:  pointer.StringPtr("/data1/logs/error1.log"),
				Level:     pointer.StringPtr("information1"),
				ServerLog: pointer.StringPtr("/data0/logs/server1.log"),
				Size:      pointer.StringPtr("101M"),
			},
			MarkCacheSize:        pointer.Int64Ptr(5368709121),
			MinmaxIndexCacheSize: pointer.Int64Ptr(5368709121),
			FlashDataPath:        pointer.StringPtr("/data1/db"),
			PathRealtimeMode:     pointer.BoolPtr(true),
			FlashProfile: &v1alpha1.FlashProfile{
				Default: &v1alpha1.Profile{
					LoadBalancing:        pointer.StringPtr("random1"),
					MaxMemoryUsage:       pointer.Int64Ptr(10000000001),
					UseUncompressedCache: pointer.Int32Ptr(1),
				},
				Readonly: &v1alpha1.Profile{
					Readonly: pointer.Int32Ptr(0),
				},
			},
			FlashQuota: &v1alpha1.FlashQuota{
				Default: &v1alpha1.Quota{
					Interval: &v1alpha1.Interval{
						Duration:      pointer.Int32Ptr(3601),
						Errors:        pointer.Int32Ptr(1),
						ExecutionTime: pointer.Int32Ptr(1),
						Queries:       pointer.Int32Ptr(1),
						ReadRows:      pointer.Int32Ptr(1),
						ResultRows:    pointer.Int32Ptr(1),
					},
				},
			},
			FlashRaft: &v1alpha1.FlashRaft{
				KVStorePath:   pointer.StringPtr("/data1/kvstore"),
				PDAddr:        pointer.StringPtr("test-pd.test.svc:2379"),
				StorageEngine: pointer.StringPtr("dt"),
			},
			FlashStatus: &v1alpha1.FlashStatus{
				MetricsPort: pointer.Int32Ptr(8235),
			},
			TCPPort:       pointer.Int32Ptr(9001),
			TCPPortSecure: pointer.Int32Ptr(9002),
			TmpPath:       pointer.StringPtr("/data1/tmp"),
			FlashUser: &v1alpha1.FlashUser{
				Default: &v1alpha1.User{
					Networks: &v1alpha1.Networks{
						IP: pointer.StringPtr("::/1"),
					},
					Profile: pointer.StringPtr("default1"),
					Quota:   pointer.StringPtr("default1"),
				},
				Readonly: &v1alpha1.User{
					Networks: &v1alpha1.Networks{
						IP: pointer.StringPtr("::/1"),
					},
					Profile: pointer.StringPtr("readonly1"),
					Quota:   pointer.StringPtr("default1"),
				},
			},
		},
		ProxyConfig: &v1alpha1.ProxyConfig{
			LogLevel: pointer.StringPtr("info1"),
			Server: &v1alpha1.FlashServerConfig{
				EngineAddr:          pointer.StringPtr("test-tiflash-POD_NUM.test-tiflash-peer.test.svc:3930"),
				StatusAddr:          pointer.StringPtr("0.0.0.0:20292"),
				AdvertiseStatusAddr: pointer.StringPtr("test-tiflash-POD_NUM.test-tiflash-peer.test.svc:20292"),
			},
		},
	}
	customTiFlashLogConfig = v1alpha1.TiFlashConfig{
		CommonConfig: &v1alpha1.CommonConfig{
			Flash: &v1alpha1.Flash{
				FlashCluster: &v1alpha1.FlashCluster{
					ClusterLog: pointer.StringPtr("/data1/logs/flash_cluster_manager.log"),
				},
				FlashProxy: &v1alpha1.FlashProxy{},
			},
			FlashLogger: &v1alpha1.FlashLogger{
				ErrorLog:  pointer.StringPtr("/data1/logs/error.log"),
				ServerLog: pointer.StringPtr("/data1/logs/server.log"),
			},
		},
	}
	defaultSideCarContainers = []corev1.Container{
		{
			Name:            "serverlog",
			Image:           "busybox:1.26.2",
			ImagePullPolicy: "",
			Resources:       corev1.ResourceRequirements{},
			Command: []string{
				"sh",
				"-c",
				"touch /data0/logs/server.log; tail -n0 -F /data0/logs/server.log;",
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "data0", MountPath: "/data0"},
			},
		},
		{
			Name:            "errorlog",
			Image:           "busybox:1.26.2",
			ImagePullPolicy: "",
			Resources:       corev1.ResourceRequirements{},
			Command: []string{
				"sh",
				"-c",
				"touch /data0/logs/error.log; tail -n0 -F /data0/logs/error.log;",
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "data0", MountPath: "/data0"},
			},
		},
		{
			Name:            "clusterlog",
			Image:           "busybox:1.26.2",
			ImagePullPolicy: "",
			Resources:       corev1.ResourceRequirements{},
			Command: []string{
				"sh",
				"-c",
				"touch /data0/logs/flash_cluster_manager.log; tail -n0 -F /data0/logs/flash_cluster_manager.log;",
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "data0", MountPath: "/data0"},
			},
		},
	}
	customSideCarContainers = []corev1.Container{
		{
			Name:            "serverlog",
			Image:           "busybox:1.26.2",
			ImagePullPolicy: "",
			Resources:       corev1.ResourceRequirements{},
			Command: []string{
				"sh",
				"-c",
				"touch /data1/logs/server.log; tail -n0 -F /data1/logs/server.log;",
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "data1", MountPath: "/data1"},
			},
		},
		{
			Name:            "errorlog",
			Image:           "busybox:1.26.2",
			ImagePullPolicy: "",
			Resources:       corev1.ResourceRequirements{},
			Command: []string{
				"sh",
				"-c",
				"touch /data1/logs/error.log; tail -n0 -F /data1/logs/error.log;",
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "data1", MountPath: "/data1"},
			},
		},
		{
			Name:            "clusterlog",
			Image:           "busybox:1.26.2",
			ImagePullPolicy: "",
			Resources:       corev1.ResourceRequirements{},
			Command: []string{
				"sh",
				"-c",
				"touch /data1/logs/flash_cluster_manager.log; tail -n0 -F /data1/logs/flash_cluster_manager.log;",
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "data1", MountPath: "/data1"},
			},
		},
	}
	customResourceSideCarContainers = []corev1.Container{
		{
			Name:            "serverlog",
			Image:           "busybox:1.26.2",
			ImagePullPolicy: "",
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("2Gi"),
				},
			},
			Command: []string{
				"sh",
				"-c",
				"touch /data1/logs/server.log; tail -n0 -F /data1/logs/server.log;",
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "data1", MountPath: "/data1"},
			},
		},
		{
			Name:            "errorlog",
			Image:           "busybox:1.26.2",
			ImagePullPolicy: "",
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("2Gi"),
				},
			},
			Command: []string{
				"sh",
				"-c",
				"touch /data1/logs/error.log; tail -n0 -F /data1/logs/error.log;",
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "data1", MountPath: "/data1"},
			},
		},
		{
			Name:            "clusterlog",
			Image:           "busybox:1.26.2",
			ImagePullPolicy: "",
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("2Gi"),
				},
			},
			Command: []string{
				"sh",
				"-c",
				"touch /data1/logs/flash_cluster_manager.log; tail -n0 -F /data1/logs/flash_cluster_manager.log;",
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "data1", MountPath: "/data1"},
			},
		},
	}
)

func newTidbCluster() *v1alpha1.TidbCluster {
	return &v1alpha1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pd",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
		},
		Spec: v1alpha1.TidbClusterSpec{
			PD: &v1alpha1.PDSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "pd-test-image",
				},
			},
			TiKV: &v1alpha1.TiKVSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "tikv-test-image",
				},
			},
			TiDB: &v1alpha1.TiDBSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "tidb-test-image",
				},
			},
			TiFlash: &v1alpha1.TiFlashSpec{},
		},
		Status: v1alpha1.TidbClusterStatus{},
	}
}

func TestBuildTiFlashSidecarContainers(t *testing.T) {
	type testcase struct {
		name        string
		flashConfig *v1alpha1.TiFlashConfigWraper
		expect      []corev1.Container
		resource    bool
	}

	tests := []*testcase{
		{
			name:        "nil config",
			flashConfig: nil,
			expect:      defaultSideCarContainers,
		},
		{
			name:        "empty config",
			flashConfig: v1alpha1.NewTiFlashConfig(),
			expect:      defaultSideCarContainers,
		},
		{
			name:        "custom config",
			flashConfig: mustFromOldConfig(&customTiFlashLogConfig),
			expect:      customSideCarContainers,
		},
		{
			name:        "custom resource config",
			flashConfig: mustFromOldConfig(&customTiFlashLogConfig),
			expect:      customResourceSideCarContainers,
			resource:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tc := newTidbCluster()
			tc.Spec.TiFlash.Config = test.flashConfig
			if test.resource {
				tc.Spec.TiFlash.LogTailer = &v1alpha1.LogTailerSpec{}
				tc.Spec.TiFlash.LogTailer.ResourceRequirements = corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:     resource.MustParse("1"),
						corev1.ResourceMemory:  resource.MustParse("2Gi"),
						corev1.ResourceStorage: resource.MustParse("100Gi"),
					},
				}
			}
			cs, err := buildTiFlashSidecarContainers(tc)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.expect, cs); diff != "" {
				t.Fatalf("test %s unexpected configuration (-want, +got): %s", test.name, diff)
			}
		})
	}
}

func TestSetTiFlashConfigDefault(t *testing.T) {
	type testcase struct {
		name   string
		config *v1alpha1.TiFlashConfigWraper
		expect *v1alpha1.TiFlashConfigWraper
	}

	tests := []*testcase{
		{
			name:   "nil config",
			config: v1alpha1.NewTiFlashConfig(),
			expect: mustFromOldConfig(&defaultTiFlashConfig),
		},
		{
			name:   "custom config",
			config: mustFromOldConfig(&customTiFlashConfig),
			expect: mustFromOldConfig(&customTiFlashConfig),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// g := NewGomegaWithT(t)
			setTiFlashConfigDefault(test.config, "", "test", "test")
			// g.Expect(test.config).To(Equal(test.expect))
			if diff := cmp.Diff(*test.expect, *test.config); diff != "" {
				t.Fatalf("unexpected configuration (-want, +got): %s", diff)
			}
		})
	}
}

func TestGetTiFlashConfig(t *testing.T) {
	cnConfig := defaultTiFlashTLSConfig.DeepCopy()
	cnConfig.ProxyConfig.Security.CertAllowedCN = append(cnConfig.ProxyConfig.Security.CertAllowedCN, "TiDB")
	cnConfig.CommonConfig.Security.CertAllowedCN = append(cnConfig.CommonConfig.Security.CertAllowedCN, "TiDB")
	testCases := []struct {
		name     string
		tc       v1alpha1.TidbCluster
		expected *v1alpha1.TiFlashConfig
	}{
		{
			name: "TiFlash config is nil with TLS enabled",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiFlash: &v1alpha1.TiFlashSpec{},
					TLSCluster: &v1alpha1.TLSCluster{
						Enabled: true,
					},
				},
			},
			expected: &defaultTiFlashTLSConfig,
		},
		{
			name: "TiFlash config with cert-allowed-cn and TLS enabled",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiFlash: &v1alpha1.TiFlashSpec{
						Config: mustFromOldConfig(&v1alpha1.TiFlashConfig{
							CommonConfig: &v1alpha1.CommonConfig{
								Security: &v1alpha1.FlashSecurity{
									CertAllowedCN: []string{
										"TiDB",
									},
								},
							},
						}),
					},
					TLSCluster: &v1alpha1.TLSCluster{
						Enabled: true,
					},
				},
			},
			expected: cnConfig,
		},
		{
			name: "TiFlash config is nil with TLS disabled",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiFlash: &v1alpha1.TiFlashSpec{},
				},
			},
			expected: &defaultTiFlashNonTLSConfig,
		},
		{
			name: "TiFlash config is nil with storageClaim",
			tc: v1alpha1.TidbCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: v1alpha1.TidbClusterSpec{
					TiFlash: &v1alpha1.TiFlashSpec{
						StorageClaims: []v1alpha1.StorageClaim{
							{
								StorageClassName: pointer.StringPtr("local-storage"),
							},
						},
					},
				},
			},
			expected: &defaultTiFlashNonTLSConfig,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			config := getTiFlashConfig(&tt.tc)
			flashConfig := &v1alpha1.TiFlashConfig{
				CommonConfig: new(v1alpha1.CommonConfig),
				ProxyConfig:  new(v1alpha1.ProxyConfig),
			}

			data, err := config.Common.MarshalTOML()
			g.Expect(err).Should(BeNil())
			err = toml.Unmarshal(data, flashConfig.CommonConfig)
			g.Expect(err).Should(BeNil())

			data, err = config.Proxy.MarshalTOML()
			g.Expect(err).Should(BeNil())
			err = toml.Unmarshal(data, flashConfig.ProxyConfig)
			g.Expect(err).Should(BeNil())

			if diff := cmp.Diff(*tt.expected, *flashConfig); diff != "" {
				t.Fatalf("unexpected configuration (-want, +got): %s", diff)
			}
		})
	}
}

func mustFromOldConfig(old *v1alpha1.TiFlashConfig) *v1alpha1.TiFlashConfigWraper {
	config := v1alpha1.NewTiFlashConfig()

	data, err := json.Marshal(old)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(data, config)
	if err != nil {
		panic(err)
	}

	return config
}
