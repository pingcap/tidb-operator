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
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
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
			DefaultProfile: "default",
			DisplayName:    "TiFlash",
			Flash: &v1alpha1.Flash{
				CompactLogMinPeriod: pointer.Int32Ptr(200),
				FlashCluster: &v1alpha1.FlashCluster{
					ClusterManagerPath: "/tiflash/flash_cluster_manager",
					ClusterLog:         "/data0/logs/flash_cluster_manager.log",
					MasterTTL:          pointer.Int32Ptr(60),
					RefreshInterval:    pointer.Int32Ptr(20),
					UpdateRuleInterval: pointer.Int32Ptr(10),
				},
				OverlapThreshold: pointer.Float64Ptr(0.6),
				FlashProxy: &v1alpha1.FlashProxy{
					Addr:          "0.0.0.0:20170",
					AdvertiseAddr: "test-tiflash-POD_NUM.test-tiflash-peer.test.svc:20170",
					Config:        "/data0/proxy.toml",
					DataDir:       "/data0/proxy",
					LogFile:       "/data0/logs/proxy.log",
				},
				ServiceAddr:    "test-tiflash-POD_NUM.test-tiflash-peer.test.svc:3930",
				TiDBStatusAddr: "test-tidb.test.svc:10080",
			},
			HTTPPort:               pointer.Int32Ptr(8123),
			InternalServerHTTPPort: pointer.Int32Ptr(9009),
			ListenHost:             "0.0.0.0",
			FlashLogger: &v1alpha1.FlashLogger{
				Count:     pointer.Int32Ptr(10),
				ErrorLog:  "/data0/logs/error.log",
				Level:     "information",
				ServerLog: "/data0/logs/server.log",
				Size:      "100M",
			},
			MarkCacheSize:        pointer.Int64Ptr(5368709120),
			MinmaxIndexCacheSize: pointer.Int64Ptr(5368709120),
			Path:                 "/data0/db",
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
				KVStorePath:   "/data0/kvstore",
				PDAddr:        "test-pd.test.svc:2379",
				StorageEngine: "dt",
			},
			FlashStatus: &v1alpha1.FlashStatus{
				MetricsPort: pointer.Int32Ptr(8234),
			},
			TCPPort: pointer.Int32Ptr(9000),
			TmpPath: "/data0/tmp",
			FlashUser: &v1alpha1.FlashUser{
				Default: &v1alpha1.User{
					Networks: &v1alpha1.Networks{
						IP: "::/0",
					},
					Profile: "default",
					Quota:   "default",
				},
				Readonly: &v1alpha1.User{
					Networks: &v1alpha1.Networks{
						IP: "::/0",
					},
					Profile: "readonly",
					Quota:   "default",
				},
			},
		},
		ProxyConfig: &v1alpha1.ProxyConfig{
			LogLevel: "info",
			Server: &v1alpha1.FlashServerConfig{
				EngineAddr: "test-tiflash-POD_NUM.test-tiflash-peer.test.svc:3930",
				StatusAddr: "0.0.0.0:20292",
			},
		},
	}
	customTiFlashConfig = v1alpha1.TiFlashConfig{
		CommonConfig: &v1alpha1.CommonConfig{
			FlashApplication: &v1alpha1.FlashApplication{
				RunAsDaemon: pointer.BoolPtr(false),
			},
			DefaultProfile: "defaul",
			DisplayName:    "TiFlah",
			Flash: &v1alpha1.Flash{
				CompactLogMinPeriod: pointer.Int32Ptr(100),
				FlashCluster: &v1alpha1.FlashCluster{
					ClusterManagerPath: "/flash_cluster_manager",
					ClusterLog:         "/data1/logs/flash_cluster_manager.log",
					MasterTTL:          pointer.Int32Ptr(50),
					RefreshInterval:    pointer.Int32Ptr(21),
					UpdateRuleInterval: pointer.Int32Ptr(11),
				},
				OverlapThreshold: pointer.Float64Ptr(0.7),
				FlashProxy: &v1alpha1.FlashProxy{
					Addr:          "0.0.0.0:20171",
					AdvertiseAddr: "test-tiflash-POD_NUM.test-tiflash-peer.test.svc:20171",
					Config:        "/data0/proxy1.toml",
					DataDir:       "/data0/proxy1",
					LogFile:       "/data0/logs/proxy1.log",
				},
				ServiceAddr:    "test-tiflash-POD_NUM.test-tiflash-peer.test.svc:3931",
				TiDBStatusAddr: "test-tidb.test.svc:10081",
			},
			HTTPPort:               pointer.Int32Ptr(8121),
			InternalServerHTTPPort: pointer.Int32Ptr(9001),
			ListenHost:             "0.0.0.1",
			FlashLogger: &v1alpha1.FlashLogger{
				Count:     pointer.Int32Ptr(11),
				ErrorLog:  "/data1/logs/error1.log",
				Level:     "information1",
				ServerLog: "/data0/logs/server1.log",
				Size:      "101M",
			},
			MarkCacheSize:        pointer.Int64Ptr(5368709121),
			MinmaxIndexCacheSize: pointer.Int64Ptr(5368709121),
			Path:                 "/data1/db",
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
				KVStorePath:   "/data1/kvstore",
				PDAddr:        "test-pd.test.svc:2379",
				StorageEngine: "dt",
			},
			FlashStatus: &v1alpha1.FlashStatus{
				MetricsPort: pointer.Int32Ptr(8235),
			},
			TCPPort: pointer.Int32Ptr(9001),
			TmpPath: "/data1/tmp",
			FlashUser: &v1alpha1.FlashUser{
				Default: &v1alpha1.User{
					Networks: &v1alpha1.Networks{
						IP: "::/1",
					},
					Profile: "default1",
					Quota:   "default1",
				},
				Readonly: &v1alpha1.User{
					Networks: &v1alpha1.Networks{
						IP: "::/1",
					},
					Profile: "readonly1",
					Quota:   "default1",
				},
			},
		},
		ProxyConfig: &v1alpha1.ProxyConfig{
			LogLevel: "info1",
			Server: &v1alpha1.FlashServerConfig{
				EngineAddr: "test-tiflash-POD_NUM.test-tiflash-peer.test.svc:3930",
				StatusAddr: "0.0.0.0:20292",
			},
		},
	}
	defaultTiFlashLogConfig = v1alpha1.TiFlashConfig{
		CommonConfig: &v1alpha1.CommonConfig{
			Flash: &v1alpha1.Flash{
				FlashCluster: &v1alpha1.FlashCluster{
					ClusterLog: "/data0/logs/flash_cluster_manager.log",
				},
				FlashProxy: &v1alpha1.FlashProxy{
					LogFile: "/data0/logs/proxy.log",
				},
			},
			FlashLogger: &v1alpha1.FlashLogger{
				ErrorLog:  "/data0/logs/error.log",
				ServerLog: "/data0/logs/server.log",
			},
		},
	}
	customTiFlashLogConfig = v1alpha1.TiFlashConfig{
		CommonConfig: &v1alpha1.CommonConfig{
			Flash: &v1alpha1.Flash{
				FlashCluster: &v1alpha1.FlashCluster{
					ClusterLog: "/data1/logs/flash_cluster_manager.log",
				},
				FlashProxy: &v1alpha1.FlashProxy{
					LogFile: "/data1/logs/proxy.log",
				},
			},
			FlashLogger: &v1alpha1.FlashLogger{
				ErrorLog:  "/data1/logs/error.log",
				ServerLog: "/data1/logs/server.log",
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
			Name:            "proxylog",
			Image:           "busybox:1.26.2",
			ImagePullPolicy: "",
			Resources:       corev1.ResourceRequirements{},
			Command: []string{
				"sh",
				"-c",
				"touch /data0/logs/proxy.log; tail -n0 -F /data0/logs/proxy.log;",
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
			Name:            "proxylog",
			Image:           "busybox:1.26.2",
			ImagePullPolicy: "",
			Resources:       corev1.ResourceRequirements{},
			Command: []string{
				"sh",
				"-c",
				"touch /data1/logs/proxy.log; tail -n0 -F /data1/logs/proxy.log;",
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
			Name:            "proxylog",
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
				"touch /data1/logs/proxy.log; tail -n0 -F /data1/logs/proxy.log;",
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
			PD: v1alpha1.PDSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "pd-test-image",
				},
			},
			TiKV: v1alpha1.TiKVSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "tikv-test-image",
				},
			},
			TiDB: v1alpha1.TiDBSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Image: "tidb-test-image",
				},
			},
			TiFlash: &v1alpha1.TiFlashSpec{},
		},
	}
}

func TestBuildTiFlashSidecarContainers(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name        string
		flashConfig *v1alpha1.TiFlashConfig
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
			flashConfig: &v1alpha1.TiFlashConfig{},
			expect:      defaultSideCarContainers,
		},
		{
			name:        "custom config",
			flashConfig: &customTiFlashLogConfig,
			expect:      customSideCarContainers,
		},
		{
			name:        "custom resource config",
			flashConfig: &customTiFlashLogConfig,
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
			cs := buildTiFlashSidecarContainers(tc)
			g.Expect(cs).To(Equal(test.expect))
		})
	}
}
func TestSetTiFlashConfigDefault(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name   string
		config v1alpha1.TiFlashConfig
		expect v1alpha1.TiFlashConfig
	}

	tests := []*testcase{
		{
			name:   "nil config",
			config: v1alpha1.TiFlashConfig{},
			expect: defaultTiFlashConfig,
		},
		{
			name:   "custom config",
			config: customTiFlashConfig,
			expect: customTiFlashConfig,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setTiFlashConfigDefault(&test.config, "test", "test")
			g.Expect(test.config).To(Equal(test.expect))
		})
	}
}

func TestSetTiFlashLogConfigDefault(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name   string
		config v1alpha1.TiFlashConfig
		expect v1alpha1.TiFlashConfig
	}

	tests := []*testcase{
		{
			name:   "nil config",
			config: v1alpha1.TiFlashConfig{},
			expect: defaultTiFlashLogConfig,
		},
		{
			name:   "custom config",
			config: customTiFlashLogConfig,
			expect: customTiFlashLogConfig,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setTiFlashLogConfigDefault(&test.config)
			g.Expect(test.config).To(Equal(test.expect))
		})
	}
}
