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
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/google/go-cmp/cmp"
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
			setTiFlashConfigDefault(test.config, nil, "test", "test", "", false, false, false)
			// g.Expect(test.config).To(Equal(test.expect))
			if diff := cmp.Diff(*test.expect, *test.config); diff != "" {
				t.Fatalf("unexpected configuration (-want, +got): %s", diff)
			}
		})
	}
}

func TestTestGetTiFlashConfig(t *testing.T) {
	t.Run("GetTiFlashConfig", func(t *testing.T) {
		cases := []struct {
			name      string
			tcVersion string
			expectV2  bool
		}{
			{
				name:      "version is larger than v5.4.0",
				tcVersion: "v5.4.1",
				expectV2:  true,
			},
			{
				name:      "version is v5.4.0",
				tcVersion: "v5.4.0",
				expectV2:  true,
			},
			{
				name:      "version is latest",
				tcVersion: "latest",
				expectV2:  true,
			},
			{
				name:      "version is v5.4.0 and dirty",
				tcVersion: "v5.4.0-dev123",
				expectV2:  true,
			},
			{
				name:      "version is less than v5.4.0",
				tcVersion: "v5.3.0",
				expectV2:  false,
			},
		}

		for _, testcase := range cases {
			t.Run(testcase.name, func(t *testing.T) {
				g := NewGomegaWithT(t)

				tc := &v1alpha1.TidbCluster{}
				tc.Name = "test"
				tc.Namespace = "ns"
				tc.Spec.TiFlash = &v1alpha1.TiFlashSpec{}
				tc.Spec.TiFlash.BaseImage = "pingcap/tiflash"
				tc.Spec.Version = testcase.tcVersion

				if testcase.expectV2 {
					patch := gomonkey.ApplyFunc(getTiFlashConfig, func(tc *v1alpha1.TidbCluster) *v1alpha1.TiFlashConfigWraper {
						t.Fatalf("shouldn't call getTiFlashConfig()")
						return nil
					})
					defer patch.Reset()
				} else {
					patch := gomonkey.ApplyFunc(getTiFlashConfigV2, func(tc *v1alpha1.TidbCluster) *v1alpha1.TiFlashConfigWraper {
						t.Fatalf("shouldn't call getTiFlashConfigV2()")
						return nil
					})
					defer patch.Reset()
				}

				cfg := GetTiFlashConfig(tc)
				g.Expect(cfg).ShouldNot(BeNil())
			})
		}
	})

	t.Run("getTiFlashConfig", func(t *testing.T) {
		cases := []struct {
			name            string
			setTC           func(tc *v1alpha1.TidbCluster)
			expectCommonCfg string
			expectProxyCfg  string
		}{
			{
				name: "config is nil with TLS enabled",
				setTC: func(tc *v1alpha1.TidbCluster) {
					tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
				},
				expectCommonCfg: `
				default_profile = "default"
				display_name = "TiFlash"
				https_port = 8123
				interserver_http_port = 9009
				listen_host = "0.0.0.0"
				mark_cache_size = 5368709120
				minmax_index_cache_size = 5368709120
				path = "/data0/db"
				path_realtime_mode = false
				tcp_port_secure = 9000
				tmp_path = "/data0/tmp"
			
				[application]
				  runAsDaemon = true
			
				[flash]
				  compact_log_min_period = 200
				  overlap_threshold = 0.6
				  service_addr = "0.0.0.0:3930"
				  tidb_status_addr = "test-tidb.default.svc:10080"
				  [flash.flash_cluster]
					cluster_manager_path = "/tiflash/flash_cluster_manager"
					log = "/data0/logs/flash_cluster_manager.log"
					master_ttl = 60
					refresh_interval = 20
					update_rule_interval = 10
				  [flash.proxy]
					addr = "0.0.0.0:20170"
					advertise-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20170"
					config = "/data0/proxy.toml"
					data-dir = "/data0/proxy"
			
				[logger]
				  count = 10
				  errorlog = "/data0/logs/error.log"
				  level = "information"
				  log = "/data0/logs/server.log"
				  size = "100M"
			
				[profiles]
				  [profiles.default]
					load_balancing = "random"
					max_memory_usage = 10000000000
					use_uncompressed_cache = 0
				  [profiles.readonly]
					readonly = 1
			
				[quotas]
				  [quotas.default]
					[quotas.default.interval]
					  duration = 3600
					  errors = 0
					  execution_time = 0
					  queries = 0
					  read_rows = 0
					  result_rows = 0
			
				[raft]
				  kvstore_path = "/data0/kvstore"
				  pd_addr = "test-pd.default.svc:2379"
				  storage_engine = "dt"
			
				[status]
				  metrics_port = 8234

				[security]
				  ca_path = "/var/lib/tiflash-tls/ca.crt"
				  cert_path = "/var/lib/tiflash-tls/tls.crt"
				  key_path = "/var/lib/tiflash-tls/tls.key"

				[users]
				  [users.default]
					password = ""
					profile = "default"
					quota = "default"
					[users.default.networks]
					  ip = "::/0"
				  [users.readonly]
					password = ""
					profile = "readonly"
					quota = "default"
					[users.readonly.networks]
					  ip = "::/0"`,
				expectProxyCfg: `
				log-level = "info"

				[security]
				  ca-path = "/var/lib/tiflash-tls/ca.crt"
				  cert-path = "/var/lib/tiflash-tls/tls.crt"
				  key-path = "/var/lib/tiflash-tls/tls.key"

				[server]
				  advertise-status-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20292"
				  engine-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:3930"
				  status-addr = "0.0.0.0:20292"`,
			},
			{
				name: "TiFlash config with cert-allowed-cn and TLS enabled",
				setTC: func(tc *v1alpha1.TidbCluster) {
					tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
					tc.Spec.TiFlash.Config = mustFromOldConfig(&v1alpha1.TiFlashConfig{
						CommonConfig: &v1alpha1.CommonConfig{
							Security: &v1alpha1.FlashSecurity{
								CertAllowedCN: []string{
									"TiDB",
								},
							},
						},
					})
				},
				expectCommonCfg: `
				default_profile = "default"
				display_name = "TiFlash"
				https_port = 8123
				interserver_http_port = 9009
				listen_host = "0.0.0.0"
				mark_cache_size = 5368709120
				minmax_index_cache_size = 5368709120
				path = "/data0/db"
				path_realtime_mode = false
				tcp_port_secure = 9000
				tmp_path = "/data0/tmp"
			
				[application]
				  runAsDaemon = true
			
				[flash]
				  compact_log_min_period = 200
				  overlap_threshold = 0.6
				  service_addr = "0.0.0.0:3930"
				  tidb_status_addr = "test-tidb.default.svc:10080"
				  [flash.flash_cluster]
					cluster_manager_path = "/tiflash/flash_cluster_manager"
					log = "/data0/logs/flash_cluster_manager.log"
					master_ttl = 60
					refresh_interval = 20
					update_rule_interval = 10
				  [flash.proxy]
					addr = "0.0.0.0:20170"
					advertise-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20170"
					config = "/data0/proxy.toml"
					data-dir = "/data0/proxy"
			
				[logger]
				  count = 10
				  errorlog = "/data0/logs/error.log"
				  level = "information"
				  log = "/data0/logs/server.log"
				  size = "100M"
			
				[profiles]
				  [profiles.default]
					load_balancing = "random"
					max_memory_usage = 10000000000
					use_uncompressed_cache = 0
				  [profiles.readonly]
					readonly = 1
			
				[quotas]
				  [quotas.default]
					[quotas.default.interval]
					  duration = 3600
					  errors = 0
					  execution_time = 0
					  queries = 0
					  read_rows = 0
					  result_rows = 0
			
				[raft]
				  kvstore_path = "/data0/kvstore"
				  pd_addr = "test-pd.default.svc:2379"
				  storage_engine = "dt"
			
				[status]
				  metrics_port = 8234

				[security]
				  ca_path = "/var/lib/tiflash-tls/ca.crt"
				  cert_path = "/var/lib/tiflash-tls/tls.crt"
				  key_path = "/var/lib/tiflash-tls/tls.key"
				  cert_allowed_cn = ["TiDB"]

				[users]
				  [users.default]
					password = ""
					profile = "default"
					quota = "default"
					[users.default.networks]
					  ip = "::/0"
				  [users.readonly]
					password = ""
					profile = "readonly"
					quota = "default"
					[users.readonly.networks]
					  ip = "::/0"`,
				expectProxyCfg: `
				log-level = "info"

				[security]
				  ca-path = "/var/lib/tiflash-tls/ca.crt"
				  cert-path = "/var/lib/tiflash-tls/tls.crt"
				  key-path = "/var/lib/tiflash-tls/tls.key"
				  cert-allowed-cn = ["TiDB"]

				[server]
				  advertise-status-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20292"
				  engine-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:3930"
				  status-addr = "0.0.0.0:20292"`,
			},
			{
				name: "config is nil with TLS disabled",
				setTC: func(tc *v1alpha1.TidbCluster) {
					tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: false}
				},
				expectCommonCfg: `
				default_profile = "default"
				display_name = "TiFlash"
				http_port = 8123
				interserver_http_port = 9009
				listen_host = "0.0.0.0"
				mark_cache_size = 5368709120
				minmax_index_cache_size = 5368709120
				path = "/data0/db"
				path_realtime_mode = false
				tcp_port = 9000
				tmp_path = "/data0/tmp"
			
				[application]
				  runAsDaemon = true
			
				[flash]
				  compact_log_min_period = 200
				  overlap_threshold = 0.6
				  service_addr = "0.0.0.0:3930"
				  tidb_status_addr = "test-tidb.default.svc:10080"
				  [flash.flash_cluster]
					cluster_manager_path = "/tiflash/flash_cluster_manager"
					log = "/data0/logs/flash_cluster_manager.log"
					master_ttl = 60
					refresh_interval = 20
					update_rule_interval = 10
				  [flash.proxy]
					addr = "0.0.0.0:20170"
					advertise-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20170"
					config = "/data0/proxy.toml"
					data-dir = "/data0/proxy"
			
				[logger]
				  count = 10
				  errorlog = "/data0/logs/error.log"
				  level = "information"
				  log = "/data0/logs/server.log"
				  size = "100M"
			
				[profiles]
				  [profiles.default]
					load_balancing = "random"
					max_memory_usage = 10000000000
					use_uncompressed_cache = 0
				  [profiles.readonly]
					readonly = 1
			
				[quotas]
				  [quotas.default]
					[quotas.default.interval]
					  duration = 3600
					  errors = 0
					  execution_time = 0
					  queries = 0
					  read_rows = 0
					  result_rows = 0
			
				[raft]
				  kvstore_path = "/data0/kvstore"
				  pd_addr = "test-pd.default.svc:2379"
				  storage_engine = "dt"
			
				[status]
				  metrics_port = 8234

				[users]
				  [users.default]
					password = ""
					profile = "default"
					quota = "default"
					[users.default.networks]
					  ip = "::/0"
				  [users.readonly]
					password = ""
					profile = "readonly"
					quota = "default"
					[users.readonly.networks]
					  ip = "::/0"`,
				expectProxyCfg: `
				log-level = "info"

				[server]
				  advertise-status-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20292"
				  engine-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:3930"
				  status-addr = "0.0.0.0:20292"`,
			},
			{
				name: "config is nil with storageClaim",
				setTC: func(tc *v1alpha1.TidbCluster) {
					tc.Spec.TiFlash.StorageClaims = []v1alpha1.StorageClaim{
						{
							StorageClassName: pointer.StringPtr("local-storage"),
						},
					}
				},
				expectCommonCfg: `
				default_profile = "default"
				display_name = "TiFlash"
				http_port = 8123
				interserver_http_port = 9009
				listen_host = "0.0.0.0"
				mark_cache_size = 5368709120
				minmax_index_cache_size = 5368709120
				path = "/data0/db"
				path_realtime_mode = false
				tcp_port = 9000
				tmp_path = "/data0/tmp"
			
				[application]
				  runAsDaemon = true
			
				[flash]
				  compact_log_min_period = 200
				  overlap_threshold = 0.6
				  service_addr = "0.0.0.0:3930"
				  tidb_status_addr = "test-tidb.default.svc:10080"
				  [flash.flash_cluster]
					cluster_manager_path = "/tiflash/flash_cluster_manager"
					log = "/data0/logs/flash_cluster_manager.log"
					master_ttl = 60
					refresh_interval = 20
					update_rule_interval = 10
				  [flash.proxy]
					addr = "0.0.0.0:20170"
					advertise-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20170"
					config = "/data0/proxy.toml"
					data-dir = "/data0/proxy"
			
				[logger]
				  count = 10
				  errorlog = "/data0/logs/error.log"
				  level = "information"
				  log = "/data0/logs/server.log"
				  size = "100M"
			
				[profiles]
				  [profiles.default]
					load_balancing = "random"
					max_memory_usage = 10000000000
					use_uncompressed_cache = 0
				  [profiles.readonly]
					readonly = 1
			
				[quotas]
				  [quotas.default]
					[quotas.default.interval]
					  duration = 3600
					  errors = 0
					  execution_time = 0
					  queries = 0
					  read_rows = 0
					  result_rows = 0
			
				[raft]
				  kvstore_path = "/data0/kvstore"
				  pd_addr = "test-pd.default.svc:2379"
				  storage_engine = "dt"
			
				[status]
				  metrics_port = 8234

				[users]
				  [users.default]
					password = ""
					profile = "default"
					quota = "default"
					[users.default.networks]
					  ip = "::/0"
				  [users.readonly]
					password = ""
					profile = "readonly"
					quota = "default"
					[users.readonly.networks]
					  ip = "::/0"`,
				expectProxyCfg: `
				log-level = "info"

				[server]
				  advertise-status-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20292"
				  engine-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:3930"
				  status-addr = "0.0.0.0:20292"`,
			},
			{
				name: "config is nil with TLS disabled",
				setTC: func(tc *v1alpha1.TidbCluster) {
					tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: false}
				},
				expectCommonCfg: `
				default_profile = "default"
				display_name = "TiFlash"
				http_port = 8123
				interserver_http_port = 9009
				listen_host = "0.0.0.0"
				mark_cache_size = 5368709120
				minmax_index_cache_size = 5368709120
				path = "/data0/db"
				path_realtime_mode = false
				tcp_port = 9000
				tmp_path = "/data0/tmp"
			
				[application]
				  runAsDaemon = true
			
				[flash]
				  compact_log_min_period = 200
				  overlap_threshold = 0.6
				  service_addr = "0.0.0.0:3930"
				  tidb_status_addr = "test-tidb.default.svc:10080"
				  [flash.flash_cluster]
					cluster_manager_path = "/tiflash/flash_cluster_manager"
					log = "/data0/logs/flash_cluster_manager.log"
					master_ttl = 60
					refresh_interval = 20
					update_rule_interval = 10
				  [flash.proxy]
					addr = "0.0.0.0:20170"
					advertise-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20170"
					config = "/data0/proxy.toml"
					data-dir = "/data0/proxy"
			
				[logger]
				  count = 10
				  errorlog = "/data0/logs/error.log"
				  level = "information"
				  log = "/data0/logs/server.log"
				  size = "100M"
			
				[profiles]
				  [profiles.default]
					load_balancing = "random"
					max_memory_usage = 10000000000
					use_uncompressed_cache = 0
				  [profiles.readonly]
					readonly = 1
			
				[quotas]
				  [quotas.default]
					[quotas.default.interval]
					  duration = 3600
					  errors = 0
					  execution_time = 0
					  queries = 0
					  read_rows = 0
					  result_rows = 0
			
				[raft]
				  kvstore_path = "/data0/kvstore"
				  pd_addr = "test-pd.default.svc:2379"
				  storage_engine = "dt"
			
				[status]
				  metrics_port = 8234

				[users]
				  [users.default]
					password = ""
					profile = "default"
					quota = "default"
					[users.default.networks]
					  ip = "::/0"
				  [users.readonly]
					password = ""
					profile = "readonly"
					quota = "default"
					[users.readonly.networks]
					  ip = "::/0"`,
				expectProxyCfg: `
				log-level = "info"

				[server]
				  advertise-status-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20292"
				  engine-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:3930"
				  status-addr = "0.0.0.0:20292"`,
			},
			{
				name: "heterogeneous cluster without local pd and local tidb",
				setTC: func(tc *v1alpha1.TidbCluster) {
					tc.Spec.PD = nil
					tc.Spec.TiDB = nil
					tc.Spec.Cluster = &v1alpha1.TidbClusterRef{Name: "cluster-1", Namespace: "default"}
				},
				expectCommonCfg: `
				default_profile = "default"
				display_name = "TiFlash"
				http_port = 8123
				interserver_http_port = 9009
				listen_host = "0.0.0.0"
				mark_cache_size = 5368709120
				minmax_index_cache_size = 5368709120
				path = "/data0/db"
				path_realtime_mode = false
				tcp_port = 9000
				tmp_path = "/data0/tmp"
			
				[application]
				  runAsDaemon = true
			
				[flash]
				  compact_log_min_period = 200
				  overlap_threshold = 0.6
				  service_addr = "0.0.0.0:3930"
				  tidb_status_addr = "cluster-1-tidb.default.svc:10080"
				  [flash.flash_cluster]
					cluster_manager_path = "/tiflash/flash_cluster_manager"
					log = "/data0/logs/flash_cluster_manager.log"
					master_ttl = 60
					refresh_interval = 20
					update_rule_interval = 10
				  [flash.proxy]
					addr = "0.0.0.0:20170"
					advertise-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20170"
					config = "/data0/proxy.toml"
					data-dir = "/data0/proxy"
			
				[logger]
				  count = 10
				  errorlog = "/data0/logs/error.log"
				  level = "information"
				  log = "/data0/logs/server.log"
				  size = "100M"
			
				[profiles]
				  [profiles.default]
					load_balancing = "random"
					max_memory_usage = 10000000000
					use_uncompressed_cache = 0
				  [profiles.readonly]
					readonly = 1
			
				[quotas]
				  [quotas.default]
					[quotas.default.interval]
					  duration = 3600
					  errors = 0
					  execution_time = 0
					  queries = 0
					  read_rows = 0
					  result_rows = 0
			
				[raft]
				  kvstore_path = "/data0/kvstore"
				  pd_addr = "cluster-1-pd.default.svc:2379"
				  storage_engine = "dt"
			
				[status]
				  metrics_port = 8234

				[users]
				  [users.default]
					password = ""
					profile = "default"
					quota = "default"
					[users.default.networks]
					  ip = "::/0"
				  [users.readonly]
					password = ""
					profile = "readonly"
					quota = "default"
					[users.readonly.networks]
					  ip = "::/0"`,
				expectProxyCfg: `
				log-level = "info"

				[server]
				  advertise-status-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20292"
				  engine-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:3930"
				  status-addr = "0.0.0.0:20292"`,
			},
			{
				name: "main cluster across kubernetes",
				setTC: func(tc *v1alpha1.TidbCluster) {
					tc.Spec.TiFlash.Config = nil
					tc.Spec.AcrossK8s = true
				},
				expectCommonCfg: `
				default_profile = "default"
				display_name = "TiFlash"
				http_port = 8123
				interserver_http_port = 9009
				listen_host = "0.0.0.0"
				mark_cache_size = 5368709120
				minmax_index_cache_size = 5368709120
				path = "/data0/db"
				path_realtime_mode = false
				tcp_port = 9000
				tmp_path = "/data0/tmp"
			
				[application]
				  runAsDaemon = true
			
				[flash]
				  compact_log_min_period = 200
				  overlap_threshold = 0.6
				  service_addr = "0.0.0.0:3930"
				  tidb_status_addr = "test-tidb.default.svc:10080"
				  [flash.flash_cluster]
					cluster_manager_path = "/tiflash/flash_cluster_manager"
					log = "/data0/logs/flash_cluster_manager.log"
					master_ttl = 60
					refresh_interval = 20
					update_rule_interval = 10
				  [flash.proxy]
					addr = "0.0.0.0:20170"
					advertise-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20170"
					config = "/data0/proxy.toml"
					data-dir = "/data0/proxy"
			
				[logger]
				  count = 10
				  errorlog = "/data0/logs/error.log"
				  level = "information"
				  log = "/data0/logs/server.log"
				  size = "100M"
			
				[profiles]
				  [profiles.default]
					load_balancing = "random"
					max_memory_usage = 10000000000
					use_uncompressed_cache = 0
				  [profiles.readonly]
					readonly = 1
			
				[quotas]
				  [quotas.default]
					[quotas.default.interval]
					  duration = 3600
					  errors = 0
					  execution_time = 0
					  queries = 0
					  read_rows = 0
					  result_rows = 0
			
				[raft]
				  kvstore_path = "/data0/kvstore"
				  pd_addr = "PD_ADDR"
				  storage_engine = "dt"
			
				[status]
				  metrics_port = 8234

				[users]
				  [users.default]
					password = ""
					profile = "default"
					quota = "default"
					[users.default.networks]
					  ip = "::/0"
				  [users.readonly]
					password = ""
					profile = "readonly"
					quota = "default"
					[users.readonly.networks]
					  ip = "::/0"`,
				expectProxyCfg: `
				log-level = "info"

				[server]
				  advertise-status-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20292"
				  engine-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:3930"
				  status-addr = "0.0.0.0:20292"`,
			},
			{
				name: "heterogeneous cluster across kubernetes",
				setTC: func(tc *v1alpha1.TidbCluster) {
					tc.Spec.TiFlash.Config = nil
					tc.Spec.PD = &v1alpha1.PDSpec{}
					tc.Spec.TiDB = &v1alpha1.TiDBSpec{}
					tc.Spec.Cluster = &v1alpha1.TidbClusterRef{Name: "cluster-1", Namespace: "default"}
					tc.Spec.AcrossK8s = true
				},
				expectCommonCfg: `
				default_profile = "default"
				display_name = "TiFlash"
				http_port = 8123
				interserver_http_port = 9009
				listen_host = "0.0.0.0"
				mark_cache_size = 5368709120
				minmax_index_cache_size = 5368709120
				path = "/data0/db"
				path_realtime_mode = false
				tcp_port = 9000
				tmp_path = "/data0/tmp"
			
				[application]
				  runAsDaemon = true
			
				[flash]
				  compact_log_min_period = 200
				  overlap_threshold = 0.6
				  service_addr = "0.0.0.0:3930"
				  tidb_status_addr = "test-tidb.default.svc:10080"
				  [flash.flash_cluster]
					cluster_manager_path = "/tiflash/flash_cluster_manager"
					log = "/data0/logs/flash_cluster_manager.log"
					master_ttl = 60
					refresh_interval = 20
					update_rule_interval = 10
				  [flash.proxy]
					addr = "0.0.0.0:20170"
					advertise-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20170"
					config = "/data0/proxy.toml"
					data-dir = "/data0/proxy"
			
				[logger]
				  count = 10
				  errorlog = "/data0/logs/error.log"
				  level = "information"
				  log = "/data0/logs/server.log"
				  size = "100M"
			
				[profiles]
				  [profiles.default]
					load_balancing = "random"
					max_memory_usage = 10000000000
					use_uncompressed_cache = 0
				  [profiles.readonly]
					readonly = 1
			
				[quotas]
				  [quotas.default]
					[quotas.default.interval]
					  duration = 3600
					  errors = 0
					  execution_time = 0
					  queries = 0
					  read_rows = 0
					  result_rows = 0
			
				[raft]
				  kvstore_path = "/data0/kvstore"
				  pd_addr = "PD_ADDR"
				  storage_engine = "dt"
			
				[status]
				  metrics_port = 8234

				[users]
				  [users.default]
					password = ""
					profile = "default"
					quota = "default"
					[users.default.networks]
					  ip = "::/0"
				  [users.readonly]
					password = ""
					profile = "readonly"
					quota = "default"
					[users.readonly.networks]
					  ip = "::/0"`,
				expectProxyCfg: `
				log-level = "info"

				[server]
				  advertise-status-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20292"
				  engine-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:3930"
				  status-addr = "0.0.0.0:20292"`,
			},
			{
				name: "heterogeneous cluster across kubernetes without pd and tidb",
				setTC: func(tc *v1alpha1.TidbCluster) {
					tc.Spec.TiFlash.Config = nil
					tc.Spec.PD = nil
					tc.Spec.TiDB = nil
					tc.Spec.Cluster = &v1alpha1.TidbClusterRef{Name: "cluster-1", Namespace: "default"}
					tc.Spec.AcrossK8s = true
				},
				expectCommonCfg: `
				default_profile = "default"
				display_name = "TiFlash"
				http_port = 8123
				interserver_http_port = 9009
				listen_host = "0.0.0.0"
				mark_cache_size = 5368709120
				minmax_index_cache_size = 5368709120
				path = "/data0/db"
				path_realtime_mode = false
				tcp_port = 9000
				tmp_path = "/data0/tmp"
			
				[application]
				  runAsDaemon = true
			
				[flash]
				  compact_log_min_period = 200
				  overlap_threshold = 0.6
				  service_addr = "0.0.0.0:3930"
				  tidb_status_addr = "cluster-1-tidb-peer.default.svc:10080"
				  [flash.flash_cluster]
					cluster_manager_path = "/tiflash/flash_cluster_manager"
					log = "/data0/logs/flash_cluster_manager.log"
					master_ttl = 60
					refresh_interval = 20
					update_rule_interval = 10
				  [flash.proxy]
					addr = "0.0.0.0:20170"
					advertise-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20170"
					config = "/data0/proxy.toml"
					data-dir = "/data0/proxy"
			
				[logger]
				  count = 10
				  errorlog = "/data0/logs/error.log"
				  level = "information"
				  log = "/data0/logs/server.log"
				  size = "100M"
			
				[profiles]
				  [profiles.default]
					load_balancing = "random"
					max_memory_usage = 10000000000
					use_uncompressed_cache = 0
				  [profiles.readonly]
					readonly = 1
			
				[quotas]
				  [quotas.default]
					[quotas.default.interval]
					  duration = 3600
					  errors = 0
					  execution_time = 0
					  queries = 0
					  read_rows = 0
					  result_rows = 0
			
				[raft]
				  kvstore_path = "/data0/kvstore"
				  pd_addr = "PD_ADDR"
				  storage_engine = "dt"
			
				[status]
				  metrics_port = 8234

				[users]
				  [users.default]
					password = ""
					profile = "default"
					quota = "default"
					[users.default.networks]
					  ip = "::/0"
				  [users.readonly]
					password = ""
					profile = "readonly"
					quota = "default"
					[users.readonly.networks]
					  ip = "::/0"`,
				expectProxyCfg: `
				log-level = "info"

				[server]
				  advertise-status-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20292"
				  engine-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:3930"
				  status-addr = "0.0.0.0:20292"`,
			},
		}

		for _, testcase := range cases {
			t.Run(testcase.name, func(t *testing.T) {
				g := NewGomegaWithT(t)

				tc := &v1alpha1.TidbCluster{}
				tc.Name = "test"
				tc.Namespace = "default"
				tc.Spec.TiFlash = &v1alpha1.TiFlashSpec{}

				if testcase.setTC != nil {
					testcase.setTC(tc)
				}

				cfg := getTiFlashConfig(tc)

				commonCfgData, err := cfg.Common.MarshalTOML()
				g.Expect(err).Should(Succeed())
				proxyCfgData, err := cfg.Proxy.MarshalTOML()
				g.Expect(err).Should(Succeed())

				outputCfg := v1alpha1.NewTiFlashConfig()
				expectCfg := v1alpha1.NewTiFlashConfig()
				outputCfg.Common.UnmarshalTOML(commonCfgData)
				outputCfg.Proxy.UnmarshalTOML(proxyCfgData)
				expectCfg.Common.UnmarshalTOML([]byte(testcase.expectCommonCfg))
				expectCfg.Proxy.UnmarshalTOML([]byte(testcase.expectProxyCfg))

				diff := cmp.Diff(outputCfg.Common.Inner(), expectCfg.Common.Inner())
				g.Expect(diff).Should(BeEmpty())
				diff = cmp.Diff(outputCfg.Proxy.Inner(), expectCfg.Proxy.Inner())
				g.Expect(diff).Should(BeEmpty())
			})
		}
	})

	t.Run("getTiFlashConfigV2", func(t *testing.T) {
		cases := []struct {
			name            string
			setTC           func(tc *v1alpha1.TidbCluster)
			expectCommonCfg string
			expectProxyCfg  string
		}{
			{
				name: "config is nil",
				setTC: func(tc *v1alpha1.TidbCluster) {
					tc.Spec.TiFlash.Config = nil
				},
				expectCommonCfg: `
					http_port = 8123
					tcp_port = 9000
					[flash]
					  service_addr = "0.0.0.0:3930"
					  tidb_status_addr = "test-tidb.default.svc:10080"
					  [flash.flash_cluster]
						log = "/data0/logs/flash_cluster_manager.log"
					  [flash.proxy]
						addr = "0.0.0.0:20170"
						advertise-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20170"
						config = "/data0/proxy.toml"
						data-dir = "/data0/proxy"
					[logger]
					  errorlog = "/data0/logs/error.log"
					  log = "/data0/logs/server.log"
					[raft]
					  pd_addr = "test-pd.default.svc:2379"
					[storage]
					  [storage.main]
						dir = ["/data0/db"]
					  [storage.raft]
						dir = "/data0/kvstore"`,
				expectProxyCfg: `
					log-level = "info"

					[server]
					advertise-status-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20292"
					engine-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:3930"
					status-addr = "0.0.0.0:20292"`,
			},
			{
				name: "config is nil and cluster enable tls",
				setTC: func(tc *v1alpha1.TidbCluster) {
					tc.Spec.TiFlash.Config = nil
					tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
				},
				expectCommonCfg: `
					https_port = 8123
					tcp_port_secure = 9000
					[flash]
					  service_addr = "0.0.0.0:3930"
					  tidb_status_addr = "test-tidb.default.svc:10080"
					  [flash.flash_cluster]
						log = "/data0/logs/flash_cluster_manager.log"
					  [flash.proxy]
						addr = "0.0.0.0:20170"
						advertise-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20170"
						config = "/data0/proxy.toml"
						data-dir = "/data0/proxy"
					[logger]
					  errorlog = "/data0/logs/error.log"
					  log = "/data0/logs/server.log"
					[raft]
					  pd_addr = "test-pd.default.svc:2379"
					[security]
					  ca_path = "/var/lib/tiflash-tls/ca.crt"
					  cert_path = "/var/lib/tiflash-tls/tls.crt"
					  key_path = "/var/lib/tiflash-tls/tls.key"
					[storage]
					  [storage.main]
						dir = ["/data0/db"]
					  [storage.raft]
						dir = "/data0/kvstore"`,
				expectProxyCfg: `
					log-level = "info"
					[security]
					  ca-path = "/var/lib/tiflash-tls/ca.crt"
					  cert-path = "/var/lib/tiflash-tls/tls.crt"
					  key-path = "/var/lib/tiflash-tls/tls.key"
					[server]
					  advertise-status-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20292"
					  engine-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:3930"
					  status-addr = "0.0.0.0:20292"`,
			},
			{
				name: "config contain cert-allowed-cn and cluster enable tls",
				setTC: func(tc *v1alpha1.TidbCluster) {
					input := `
					[security]
					  cert_allowed_cn = ["TiDB"]`
					tc.Spec.TiFlash.Config = v1alpha1.NewTiFlashConfig()
					tc.Spec.TiFlash.Config.Common.UnmarshalTOML([]byte(input))
					tc.Spec.TLSCluster = &v1alpha1.TLSCluster{Enabled: true}
				},
				expectCommonCfg: `
					https_port = 8123
					tcp_port_secure = 9000
					[flash]
					  service_addr = "0.0.0.0:3930"
					  tidb_status_addr = "test-tidb.default.svc:10080"
					  [flash.flash_cluster]
						log = "/data0/logs/flash_cluster_manager.log"
					  [flash.proxy]
						addr = "0.0.0.0:20170"
						advertise-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20170"
						config = "/data0/proxy.toml"
						data-dir = "/data0/proxy"
					[logger]
					  errorlog = "/data0/logs/error.log"
					  log = "/data0/logs/server.log"
					[raft]
					  pd_addr = "test-pd.default.svc:2379"
					[security]
					  ca_path = "/var/lib/tiflash-tls/ca.crt"
					  cert_path = "/var/lib/tiflash-tls/tls.crt"
					  key_path = "/var/lib/tiflash-tls/tls.key"
					  cert_allowed_cn = ["TiDB"]
					[storage]
					  [storage.main]
						dir = ["/data0/db"]
					  [storage.raft]
						dir = "/data0/kvstore"`,
				expectProxyCfg: `
					log-level = "info"
					[security]
					  ca-path = "/var/lib/tiflash-tls/ca.crt"
					  cert-path = "/var/lib/tiflash-tls/tls.crt"
					  key-path = "/var/lib/tiflash-tls/tls.key"
					  cert-allowed-cn = ["TiDB"]
					[server]
					  advertise-status-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20292"
					  engine-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:3930"
					  status-addr = "0.0.0.0:20292"`,
			},
			{
				name: "config is nil with multi storage",
				setTC: func(tc *v1alpha1.TidbCluster) {
					tc.Spec.TiFlash.Config = nil
					tc.Spec.TiFlash.StorageClaims = []v1alpha1.StorageClaim{
						{
							StorageClassName: pointer.StringPtr("local-storage"),
						},
						{
							StorageClassName: pointer.StringPtr("local-storage-2"),
						},
					}

				},
				expectCommonCfg: `
					http_port = 8123
					tcp_port = 9000
					[flash]
					  service_addr = "0.0.0.0:3930"
					  tidb_status_addr = "test-tidb.default.svc:10080"
					  [flash.flash_cluster]
						log = "/data0/logs/flash_cluster_manager.log"
					  [flash.proxy]
						addr = "0.0.0.0:20170"
						advertise-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20170"
						config = "/data0/proxy.toml"
						data-dir = "/data0/proxy"
					[logger]
					  errorlog = "/data0/logs/error.log"
					  log = "/data0/logs/server.log"
					[raft]
					  pd_addr = "test-pd.default.svc:2379"
					[storage]
					  [storage.main]
						dir = ["/data0/db","/data1/db"]
					  [storage.raft]
						dir = "/data0/kvstore"`,
				expectProxyCfg: `
					log-level = "info"

					[server]
					advertise-status-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20292"
					engine-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:3930"
					status-addr = "0.0.0.0:20292"`,
			},
			{
				name: "heterogeneous cluster without local pd and local tidb",
				setTC: func(tc *v1alpha1.TidbCluster) {
					tc.Spec.TiFlash.Config = nil
					tc.Spec.PD = nil
					tc.Spec.TiDB = nil
					tc.Spec.Cluster = &v1alpha1.TidbClusterRef{Name: "cluster-1", Namespace: "default"}
				},
				expectCommonCfg: `
					http_port = 8123
					tcp_port = 9000
					[flash]
					  service_addr = "0.0.0.0:3930"
					  tidb_status_addr = "cluster-1-tidb.default.svc:10080"
					  [flash.flash_cluster]
						log = "/data0/logs/flash_cluster_manager.log"
					  [flash.proxy]
						addr = "0.0.0.0:20170"
						advertise-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20170"
						config = "/data0/proxy.toml"
						data-dir = "/data0/proxy"
					[logger]
					  errorlog = "/data0/logs/error.log"
					  log = "/data0/logs/server.log"
					[raft]
					  pd_addr = "cluster-1-pd.default.svc:2379"
					[storage]
					  [storage.main]
						dir = ["/data0/db"]
					  [storage.raft]
						dir = "/data0/kvstore"`,
				expectProxyCfg: `
					log-level = "info"

					[server]
					advertise-status-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20292"
					engine-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:3930"
					status-addr = "0.0.0.0:20292"`,
			},
			{
				name: "main cluster across kubernetes",
				setTC: func(tc *v1alpha1.TidbCluster) {
					tc.Spec.TiFlash.Config = nil
					tc.Spec.AcrossK8s = true
				},
				expectCommonCfg: `
					http_port = 8123
					tcp_port = 9000
					[flash]
					  service_addr = "0.0.0.0:3930"
					  tidb_status_addr = "test-tidb.default.svc:10080"
					  [flash.flash_cluster]
						log = "/data0/logs/flash_cluster_manager.log"
					  [flash.proxy]
						addr = "0.0.0.0:20170"
						advertise-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20170"
						config = "/data0/proxy.toml"
						data-dir = "/data0/proxy"
					[logger]
					  errorlog = "/data0/logs/error.log"
					  log = "/data0/logs/server.log"
					[raft]
					  pd_addr = "PD_ADDR"
					[storage]
					  [storage.main]
						dir = ["/data0/db"]
					  [storage.raft]
						dir = "/data0/kvstore"`,
				expectProxyCfg: `
					log-level = "info"

					[server]
					advertise-status-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20292"
					engine-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:3930"
					status-addr = "0.0.0.0:20292"`,
			},
			{
				name: "heterogeneous cluster across kubernetes",
				setTC: func(tc *v1alpha1.TidbCluster) {
					tc.Spec.TiFlash.Config = nil
					tc.Spec.PD = &v1alpha1.PDSpec{}
					tc.Spec.TiDB = &v1alpha1.TiDBSpec{}
					tc.Spec.Cluster = &v1alpha1.TidbClusterRef{Name: "cluster-1", Namespace: "default"}
					tc.Spec.AcrossK8s = true
				},
				expectCommonCfg: `
					http_port = 8123
					tcp_port = 9000
					[flash]
					  service_addr = "0.0.0.0:3930"
					  tidb_status_addr = "test-tidb.default.svc:10080"
					  [flash.flash_cluster]
						log = "/data0/logs/flash_cluster_manager.log"
					  [flash.proxy]
						addr = "0.0.0.0:20170"
						advertise-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20170"
						config = "/data0/proxy.toml"
						data-dir = "/data0/proxy"
					[logger]
					  errorlog = "/data0/logs/error.log"
					  log = "/data0/logs/server.log"
					[raft]
					  pd_addr = "PD_ADDR"
					[storage]
					  [storage.main]
						dir = ["/data0/db"]
					  [storage.raft]
						dir = "/data0/kvstore"`,
				expectProxyCfg: `
					log-level = "info"

					[server]
					advertise-status-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20292"
					engine-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:3930"
					status-addr = "0.0.0.0:20292"`,
			},
			{
				name: "heterogeneous cluster across kubernetes without pd and tidb",
				setTC: func(tc *v1alpha1.TidbCluster) {
					tc.Spec.TiFlash.Config = nil
					tc.Spec.PD = nil
					tc.Spec.TiDB = nil
					tc.Spec.Cluster = &v1alpha1.TidbClusterRef{Name: "cluster-1", Namespace: "default"}
					tc.Spec.AcrossK8s = true
				},
				expectCommonCfg: `
					http_port = 8123
					tcp_port = 9000
					[flash]
					  service_addr = "0.0.0.0:3930"
					  tidb_status_addr = "cluster-1-tidb-peer.default.svc:10080"
					  [flash.flash_cluster]
						log = "/data0/logs/flash_cluster_manager.log"
					  [flash.proxy]
						addr = "0.0.0.0:20170"
						advertise-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20170"
						config = "/data0/proxy.toml"
						data-dir = "/data0/proxy"
					[logger]
					  errorlog = "/data0/logs/error.log"
					  log = "/data0/logs/server.log"
					[raft]
					  pd_addr = "PD_ADDR"
					[storage]
					  [storage.main]
						dir = ["/data0/db"]
					  [storage.raft]
						dir = "/data0/kvstore"`,
				expectProxyCfg: `
					log-level = "info"

					[server]
					advertise-status-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:20292"
					engine-addr = "test-tiflash-POD_NUM.test-tiflash-peer.default.svc:3930"
					status-addr = "0.0.0.0:20292"`,
			},
		}

		for _, testcase := range cases {
			t.Run(testcase.name, func(t *testing.T) {
				g := NewGomegaWithT(t)

				tc := &v1alpha1.TidbCluster{}
				tc.Name = "test"
				tc.Namespace = "default"
				tc.Spec.TiFlash = &v1alpha1.TiFlashSpec{}

				if testcase.setTC != nil {
					testcase.setTC(tc)
				}

				cfg := getTiFlashConfigV2(tc)

				commonCfgData, err := cfg.Common.MarshalTOML()
				g.Expect(err).Should(Succeed())
				proxyCfgData, err := cfg.Proxy.MarshalTOML()
				g.Expect(err).Should(Succeed())

				outputCfg := v1alpha1.NewTiFlashConfig()
				expectCfg := v1alpha1.NewTiFlashConfig()
				outputCfg.Common.UnmarshalTOML(commonCfgData)
				outputCfg.Proxy.UnmarshalTOML(proxyCfgData)
				expectCfg.Common.UnmarshalTOML([]byte(testcase.expectCommonCfg))
				expectCfg.Proxy.UnmarshalTOML([]byte(testcase.expectProxyCfg))

				diff := cmp.Diff(outputCfg.Common.Inner(), expectCfg.Common.Inner())
				g.Expect(diff).Should(BeEmpty())
				diff = cmp.Diff(outputCfg.Proxy.Inner(), expectCfg.Proxy.Inner())
				g.Expect(diff).Should(BeEmpty())
			})
		}
	})

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
