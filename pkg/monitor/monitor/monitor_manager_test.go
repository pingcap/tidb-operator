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
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/meta"
	"github.com/prometheus/common/model"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	discoverycachedmemory "k8s.io/client-go/discovery/cached/memory"
	discoveryfake "k8s.io/client-go/discovery/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/utils/pointer"
)

func TestTidbMonitorSyncCreate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name          string
		prepare       func(tmm *MonitorManager, monitor *v1alpha1.TidbMonitor)
		errExpectFn   func(*GomegaWithT, error, *MonitorManager, *v1alpha1.TidbMonitor)
		stsCreated    bool
		svcCreated    bool
		volumeCreated bool
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		tmm := newFakeTidbMonitorManager()
		tc := &v1alpha1.TidbCluster{
			Spec: v1alpha1.TidbClusterSpec{
				TiKV: &v1alpha1.TiKVSpec{
					BaseImage: "pingcap/tikv",
				},
				TiDB: &v1alpha1.TiDBSpec{
					TLSClient: &v1alpha1.TiDBTLSClient{Enabled: true},
				},
			},
		}

		tc.Namespace = "ns"
		tc.Name = "foo"
		err := tmm.deps.TiDBClusterControl.Create(tc)
		g.Expect(err).Should(BeNil())

		if test.name == "enable dm monitor" {
			newFakeDMCluster(tmm)
		}

		tm := newTidbMonitor(v1alpha1.TidbClusterRef{Name: tc.Name, Namespace: tc.Namespace})
		if test.prepare != nil {
			test.prepare(tmm, tm)
		}
		if tm.Spec.Shards == nil {
			tm.Spec.Shards = pointer.Int32Ptr(0)
		}

		err = tmm.SyncMonitor(tm)
		if test.errExpectFn != nil {
			test.errExpectFn(g, err, tmm, tm)
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		if test.svcCreated {
			for shard := int32(0); shard < *tm.Spec.Shards; shard++ {
				_, err = tmm.deps.ServiceLister.Services(tm.Namespace).Get(PrometheusName(tm.Name, shard))
				g.Expect(err).NotTo(HaveOccurred())
				_, err = tmm.deps.ServiceLister.Services(tm.Namespace).Get(reloaderName(tm, shard))
				g.Expect(err).NotTo(HaveOccurred())
			}
		}

		if test.stsCreated {
			for shard := int32(0); shard < *tm.Spec.Shards; shard++ {
				_, err := tmm.deps.StatefulSetLister.StatefulSets(tm.Namespace).Get(GetMonitorShardName(tm.Name, shard))
				g.Expect(err).NotTo(HaveOccurred())
			}
		}
		if test.volumeCreated {
			sts, err := tmm.deps.StatefulSetLister.StatefulSets(tm.Namespace).Get(GetMonitorObjectName(tm))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(sts).NotTo(Equal(nil))
			quantity, err := resource.ParseQuantity(tm.Spec.Storage)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(sts.Spec.VolumeClaimTemplates).To(Equal([]v1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{Name: v1alpha1.TidbMonitorMemberType.String()},
					Spec: v1.PersistentVolumeClaimSpec{
						AccessModes: []v1.PersistentVolumeAccessMode{
							v1.ReadWriteOnce,
						},
						StorageClassName: nil,
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: quantity,
							},
						},
					},
				},
			}))
		}
	}

	tests := []testcase{
		{
			name: "tidbmonitor with remote write configuration",
			prepare: func(tmm *MonitorManager, monitor *v1alpha1.TidbMonitor) {
				secret := &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "auth",
						Namespace: monitor.Namespace,
					},
					Data: map[string][]byte{
						"password": []byte("password"),
						"username": []byte("username"),
					},
				}
				err := tmm.deps.SecretControl.Create(monitor.Namespace, secret)
				g.Expect(err).NotTo(HaveOccurred())

				url := "http://127.0.0.1/a/b/c"
				remoteTimeout := model.Duration(30 * time.Second)
				testTime := 10 * time.Second
				monitor.Spec.Prometheus.RemoteWrite = []*v1alpha1.RemoteWriteSpec{
					{
						URL:           url,
						Name:          "test",
						RemoteTimeout: &remoteTimeout,
						WriteRelabelConfigs: []v1alpha1.RelabelConfig{
							{
								SourceLabels: []model.LabelName{"test1", "test2"},
								Regex:        ",",
								Replacement:  "replace",
								TargetLabel:  "target_label",
								Action:       "test",
							},
						},
						BasicAuth: &v1alpha1.BasicAuth{
							Password: v1.SecretKeySelector{
								LocalObjectReference: v1.LocalObjectReference{Name: "auth"},
								Key:                  "password",
							},
							Username: v1.SecretKeySelector{
								LocalObjectReference: v1.LocalObjectReference{Name: "auth"},
								Key:                  "username",
							},
						},
						BearerToken:     "/test/file",
						BearerTokenFile: "/test/file1",
						ProxyURL:        pointer.StringPtr("test1"),
						TLSConfig: &v1alpha1.TLSConfig{
							CAFile:   "/test/ca.pem",
							CertFile: "/test/cert.key",
							KeyFile:  "/test/key",
						},
						QueueConfig: &v1alpha1.QueueConfig{
							Capacity:          1,
							MinShards:         1,
							MaxShards:         1,
							MaxSamplesPerSend: 1,
							BatchSendDeadline: &testTime,
							MaxRetries:        3,
							MinBackoff:        &testTime,
							MaxBackoff:        &testTime,
						},
						MetadataConfig: &v1alpha1.MetadataConfig{
							Send:         true,
							SendInterval: "10s",
						},
					},
				}
			},
			errExpectFn: func(g *GomegaWithT, err error, tmm *MonitorManager, tm *v1alpha1.TidbMonitor) {
				errExpectRequeuefunc(g, err, tmm, tm)
			},
			stsCreated:    true,
			svcCreated:    true,
			volumeCreated: false,
		},
		{
			name: "tidbmonitor spec prometheus config reloader",
			prepare: func(tmm *MonitorManager, monitor *v1alpha1.TidbMonitor) {

				monitor.Spec.PrometheusReloader = &v1alpha1.PrometheusReloaderSpec{
					MonitorContainer: v1alpha1.MonitorContainer{
						BaseImage: "quay.io/prometheus-operator/prometheus-config-reloaders",
						Version:   "v0.49.0",
					},
				}
			},
			errExpectFn: func(g *GomegaWithT, err error, tmm *MonitorManager, tm *v1alpha1.TidbMonitor) {
				errExpectRequeuefunc(g, err, tmm, tm)
				sts, err := tmm.deps.StatefulSetLister.StatefulSets(tm.Namespace).Get(GetMonitorObjectName(tm))
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(sts.Spec.Template.Spec.Containers).To(HaveLen(3))
			},
			stsCreated:    true,
			svcCreated:    true,
			volumeCreated: false,
		},
		{
			name: "tidbmonitor spec remote write",
			prepare: func(tmm *MonitorManager, monitor *v1alpha1.TidbMonitor) {
				monitor.Spec.Prometheus.RemoteWrite = []*v1alpha1.RemoteWriteSpec{
					{
						URL: "http://localhost:1234/a/b/c",
						WriteRelabelConfigs: []v1alpha1.RelabelConfig{
							{
								SourceLabels: model.LabelNames{
									"__address__",
									portLabel,
								},
								Separator:   ";",
								Regex:       "(.*)",
								TargetLabel: "node",
								Replacement: "$1",
								Action:      "replace",
							},
						},
					},
				}

			},
			errExpectFn: func(g *GomegaWithT, err error, tmm *MonitorManager, tm *v1alpha1.TidbMonitor) {
				errExpectRequeuefunc(g, err, tmm, tm)
			},
			stsCreated:    true,
			svcCreated:    true,
			volumeCreated: false,
		},
		{
			name: "tidbmonitor spec thanos sidecar",
			prepare: func(tmm *MonitorManager, monitor *v1alpha1.TidbMonitor) {

				monitor.Spec.Thanos = &v1alpha1.ThanosSpec{
					MonitorContainer: v1alpha1.MonitorContainer{
						BaseImage: "thanosio/thanos",
						Version:   "v0.17.2",
					},
				}
			},
			errExpectFn: func(g *GomegaWithT, err error, tmm *MonitorManager, tm *v1alpha1.TidbMonitor) {
				errExpectRequeuefunc(g, err, tmm, tm)
				svc, err := tmm.deps.ServiceLister.Services(tm.Namespace).Get(PrometheusName(tm.Name, 0))
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(svc.Spec.Ports).To(Equal([]v1.ServicePort{
					{
						Name:       "http-prometheus",
						Port:       9090,
						Protocol:   v1.ProtocolTCP,
						TargetPort: intstr.FromInt(9090),
					}, {
						Name:       "thanos-grpc",
						Port:       10901,
						Protocol:   v1.ProtocolTCP,
						TargetPort: intstr.FromInt(10901),
					},
					{
						Name:       "thanos-http",
						Port:       10902,
						Protocol:   v1.ProtocolTCP,
						TargetPort: intstr.FromInt(10902),
					},
				}))

				sts, err := tmm.deps.StatefulSetLister.StatefulSets(tm.Namespace).Get(GetMonitorObjectName(tm))
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(sts.Spec.Template.Spec.Containers).To(HaveLen(3))
			},
			stsCreated:    true,
			svcCreated:    true,
			volumeCreated: false,
		},
		{
			name: "tidbmonitor spec thanos sidecar",
			prepare: func(tmm *MonitorManager, monitor *v1alpha1.TidbMonitor) {

				monitor.Spec.Thanos = &v1alpha1.ThanosSpec{
					MonitorContainer: v1alpha1.MonitorContainer{
						BaseImage: "thanosio/thanos",
						Version:   "v0.17.2",
					},
				}
			},
			errExpectFn: func(g *GomegaWithT, err error, tmm *MonitorManager, tm *v1alpha1.TidbMonitor) {
				errExpectRequeuefunc(g, err, tmm, tm)
				svc, err := tmm.deps.ServiceLister.Services(tm.Namespace).Get(PrometheusName(tm.Name, 0))
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(svc.Spec.Ports).To(Equal([]v1.ServicePort{
					{
						Name:       "http-prometheus",
						Port:       9090,
						Protocol:   v1.ProtocolTCP,
						TargetPort: intstr.FromInt(9090),
					}, {
						Name:       "thanos-grpc",
						Port:       10901,
						Protocol:   v1.ProtocolTCP,
						TargetPort: intstr.FromInt(10901),
					},
					{
						Name:       "thanos-http",
						Port:       10902,
						Protocol:   v1.ProtocolTCP,
						TargetPort: intstr.FromInt(10902),
					},
				}))

				sts, err := tmm.deps.StatefulSetLister.StatefulSets(tm.Namespace).Get(GetMonitorObjectName(tm))
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(sts.Spec.Template.Spec.Containers).To(HaveLen(3))
			},
			stsCreated:    true,
			svcCreated:    true,
			volumeCreated: false,
		},
		{
			name: "tidbmonitor enable clusterScope and running normally",
			prepare: func(tmm *MonitorManager, monitor *v1alpha1.TidbMonitor) {
				monitor.Spec.ClusterScoped = true
				monitor.Namespace = "ns2"
			},
			errExpectFn: func(g *GomegaWithT, err error, tmm *MonitorManager, tm *v1alpha1.TidbMonitor) {
				errExpectRequeuefunc(g, err, tmm, tm)
			},
			stsCreated:    true,
			svcCreated:    true,
			volumeCreated: false,
		},
		{
			name: "tidbmonitor enable grafana container and running normally",
			prepare: func(tmm *MonitorManager, monitor *v1alpha1.TidbMonitor) {
				monitor.Spec.Persistent = true
				monitor.Spec.Storage = "10Gi"
				monitor.Spec.Grafana = &v1alpha1.GrafanaSpec{
					MonitorContainer: v1alpha1.MonitorContainer{
						BaseImage: "grafana/grafana",
						Version:   "6.1.6",
					},
				}
			},
			errExpectFn: func(g *GomegaWithT, err error, tmm *MonitorManager, tm *v1alpha1.TidbMonitor) {
				errExpectRequeuefunc(g, err, tmm, tm)
				_, err = tmm.deps.ServiceLister.Services(tm.Namespace).Get(GrafanaName(tm.Name, 0))
				g.Expect(err).NotTo(HaveOccurred())
				sts, err := tmm.deps.StatefulSetLister.StatefulSets(tm.Namespace).Get(GetMonitorObjectName(tm))
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(sts.Spec.Template.Spec.Containers).To(HaveLen(3))

			},
			stsCreated:    true,
			volumeCreated: true,
			svcCreated:    true,
		},
		{
			name: "enable dm monitor",
			prepare: func(tmm *MonitorManager, monitor *v1alpha1.TidbMonitor) {
				monitor.Spec.DM = &v1alpha1.DMMonitorSpec{
					Clusters: []v1alpha1.ClusterRef{
						{
							Namespace: "ns",
							Name:      "dm-test",
						},
					},
					Initializer: v1alpha1.InitializerSpec{
						MonitorContainer: v1alpha1.MonitorContainer{
							BaseImage: "pingcap/tidb-monitor-initializer",
							Version:   "v2.0.0",
						},
					},
				}
			},
			errExpectFn: func(g *GomegaWithT, err error, tmm *MonitorManager, tm *v1alpha1.TidbMonitor) {
				errExpectRequeuefunc(g, err, tmm, tm)
				sts, err := tmm.deps.StatefulSetLister.StatefulSets(tm.Namespace).Get(GetMonitorObjectName(tm))
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(sts.Spec.Template.Spec.Containers).To(HaveLen(2))
				g.Expect(sts.Spec.Template.Spec.InitContainers).To(HaveLen(2))
			},
			stsCreated:    true,
			volumeCreated: false,
			svcCreated:    true,
		},
		{
			name: "tidbmonitor use deployment running without pv and pvc, tidbmonitor can't smooth migrate to statefulset",
			prepare: func(tmm *MonitorManager, monitor *v1alpha1.TidbMonitor) {
				monitor.Status.DeploymentStorageStatus = &v1alpha1.DeploymentStorageStatus{
					PvName: "test-pv",
				}
				monitor.Spec.Persistent = true
				monitor.Spec.Storage = "10Gi"
			},
			errExpectFn: func(g *GomegaWithT, err error, tmm *MonitorManager, tm *v1alpha1.TidbMonitor) {
				g.Expect(err).To(HaveOccurred())
			},
			stsCreated:    false,
			svcCreated:    true,
			volumeCreated: false,
		},
		{
			name: "tidbmonitor use deployment running , tidbmonitor can smooth migrate to statefulset",
			prepare: func(tmm *MonitorManager, monitor *v1alpha1.TidbMonitor) {
				monitor.Status.DeploymentStorageStatus = &v1alpha1.DeploymentStorageStatus{
					PvName: "test-pv",
				}
				monitor.Spec.Persistent = true
				monitor.Spec.Storage = "10Gi"
				quantity, _ := resource.ParseQuantity("10Gi")
				_ = tmm.deps.PVCControl.CreatePVC(monitor, &v1.PersistentVolumeClaim{

					ObjectMeta: metav1.ObjectMeta{
						Name:      GetMonitorObjectName(monitor),
						Namespace: monitor.Namespace,
					},
					Spec: v1.PersistentVolumeClaimSpec{
						VolumeName: "test-pv",
						AccessModes: []v1.PersistentVolumeAccessMode{
							v1.ReadWriteOnce,
						},
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: quantity,
							},
						},
					},
				})
				_ = tmm.deps.PVControl.CreatePV(monitor, &v1.PersistentVolume{
					TypeMeta: metav1.TypeMeta{Kind: "PersistentVolume", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pv",
						Namespace: metav1.NamespaceAll,
					},
					Spec: v1.PersistentVolumeSpec{
						PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimRetain,
					},
				})

			},
			errExpectFn: func(g *GomegaWithT, err error, tmm *MonitorManager, tm *v1alpha1.TidbMonitor) {
				g.Expect(controller.IsRequeueError(err)).To(Equal(true))
				pv, err := tmm.deps.PVControl.GetPV("test-pv")
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(pv.Spec.ClaimRef.Name).To(Equal(GetMonitorFirstPVCName(tm.Name)))
			},
			stsCreated:    true,
			svcCreated:    true,
			volumeCreated: false,
		},
		{
			name: "tidbmonitor enable persistent and running normally",
			prepare: func(tmm *MonitorManager, monitor *v1alpha1.TidbMonitor) {
				monitor.Spec.Persistent = true
				monitor.Spec.Storage = "10Gi"
			},
			errExpectFn:   errExpectRequeuefunc,
			stsCreated:    true,
			volumeCreated: true,
			svcCreated:    true,
		},
		{
			name: "tidbmonitor not spec clusters field",
			prepare: func(tmm *MonitorManager, monitor *v1alpha1.TidbMonitor) {
				monitor.Spec.Clusters = nil
			},
			errExpectFn: func(g *GomegaWithT, err error, tmm *MonitorManager, monitor *v1alpha1.TidbMonitor) {
			},
			stsCreated: false,
			svcCreated: false,
		},
		{
			name: "normal",
			prepare: func(tmm *MonitorManager, monitor *v1alpha1.TidbMonitor) {
			},
			errExpectFn: func(g *GomegaWithT, err error, tmm *MonitorManager, monitor *v1alpha1.TidbMonitor) {

			},
			stsCreated:    true,
			svcCreated:    true,
			volumeCreated: false,
		},
		{
			name: "create with two shards",
			prepare: func(tmm *MonitorManager, monitor *v1alpha1.TidbMonitor) {
				monitor.Spec.Shards = pointer.Int32Ptr(2)
			},
			errExpectFn: func(g *GomegaWithT, err error, tmm *MonitorManager, monitor *v1alpha1.TidbMonitor) {

			},
			stsCreated:    true,
			svcCreated:    true,
			volumeCreated: false,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestTidbMonitorSyncUpdate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name           string
		prepare        func(tmm *MonitorManager, monitor *v1alpha1.TidbMonitor)
		errExpectFn    func(*GomegaWithT, error, *MonitorManager, *v1alpha1.TidbMonitor)
		update         func(tmm *MonitorManager, monitor *v1alpha1.TidbMonitor)
		updateExpectFn func(*GomegaWithT, error, *MonitorManager, *v1alpha1.TidbMonitor)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		tmm := newFakeTidbMonitorManager()
		tc := &v1alpha1.TidbCluster{
			Spec: v1alpha1.TidbClusterSpec{
				TiKV: &v1alpha1.TiKVSpec{
					BaseImage: "pingcap/tikv",
				},
				TiDB: &v1alpha1.TiDBSpec{
					TLSClient: &v1alpha1.TiDBTLSClient{Enabled: true},
				},
			},
		}
		tc.Namespace = "ns"
		tc.Name = "foo"
		err := tmm.deps.TiDBClusterControl.Create(tc)
		g.Expect(err).Should(BeNil())

		tm := newTidbMonitor(v1alpha1.TidbClusterRef{Name: tc.Name, Namespace: tc.Namespace})
		if test.prepare != nil {
			test.prepare(tmm, tm)
		}

		err = tmm.SyncMonitor(tm)

		if test.errExpectFn != nil {
			test.errExpectFn(g, err, tmm, tm)
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}
		if test.update != nil {
			test.update(tmm, tm)
		}
		err = tmm.SyncMonitor(tm)
		if test.updateExpectFn != nil {
			test.updateExpectFn(g, err, tmm, tm)
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

	}

	tests := []testcase{
		{
			name: "enable grafana",
			prepare: func(tmm *MonitorManager, monitor *v1alpha1.TidbMonitor) {
				monitor.Spec.Persistent = true
				monitor.Spec.Storage = "10Gi"
				monitor.Spec.ClusterScoped = true
				monitor.Spec.Grafana = &v1alpha1.GrafanaSpec{
					MonitorContainer: v1alpha1.MonitorContainer{
						BaseImage: "grafana/grafana",
						Version:   "6.1.6",
					},
					Ingress: &v1alpha1.IngressSpec{},
				}
			},
			errExpectFn: func(g *GomegaWithT, err error, tmm *MonitorManager, tm *v1alpha1.TidbMonitor) {
				errExpectRequeuefunc(g, err, tmm, tm)
				_, err = tmm.deps.ServiceLister.Services(tm.Namespace).Get(GrafanaName(tm.Name, 0))
				g.Expect(err).NotTo(HaveOccurred())
				sts, err := tmm.deps.StatefulSetLister.StatefulSets(tm.Namespace).Get(GetMonitorObjectName(tm))
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(sts.Spec.Template.Spec.Containers).To(HaveLen(3))
			},
			update: func(tmm *MonitorManager, monitor *v1alpha1.TidbMonitor) {
				monitor.Spec.Grafana.Service.Type = v1.ServiceTypeLoadBalancer
				monitor.Spec.Grafana.Service.PortName = pointer.StringPtr("test")
				monitor.Spec.Grafana.Service.LoadBalancerIP = pointer.StringPtr("127.0.0.1")
				monitor.Spec.Prometheus.Service.Type = v1.ServiceTypeLoadBalancer
				monitor.Spec.Prometheus.Service.PortName = pointer.StringPtr("test")
				monitor.Spec.Prometheus.Service.LoadBalancerIP = pointer.StringPtr("127.0.0.1")
				monitor.Spec.Reloader.Service.Type = v1.ServiceTypeLoadBalancer
				monitor.Spec.Reloader.Service.LoadBalancerIP = pointer.StringPtr("127.0.0.1")
				monitor.Spec.Reloader.Service.PortName = pointer.StringPtr("test")
			},
			updateExpectFn: func(g *GomegaWithT, err error, tmm *MonitorManager, tm *v1alpha1.TidbMonitor) {
				g.Expect(err).NotTo(HaveOccurred())
				grafanaSvc, err := tmm.deps.ServiceLister.Services(tm.Namespace).Get(GrafanaName(tm.Name, 0))
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(grafanaSvc.Spec.Type).To(Equal(v1.ServiceTypeLoadBalancer))
				g.Expect(grafanaSvc.Spec.Ports[0].Name).To(Equal("test"))
				prometheusSvc, err := tmm.deps.ServiceLister.Services(tm.Namespace).Get(PrometheusName(tm.Name, 0))
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(prometheusSvc.Spec.Type).To(Equal(v1.ServiceTypeLoadBalancer))
				g.Expect(prometheusSvc.Spec.Ports[0].Name).To(Equal("test"))
				reloaderSvc, err := tmm.deps.ServiceLister.Services(tm.Namespace).Get(PrometheusName(tm.Name, 0))
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(reloaderSvc.Spec.Type).To(Equal(v1.ServiceTypeLoadBalancer))
				g.Expect(reloaderSvc.Spec.Ports[0].Name).To(Equal("test"))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newTidbMonitor(cluster v1alpha1.TidbClusterRef) *v1alpha1.TidbMonitor {
	return &v1alpha1.TidbMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "ns",
		},
		Spec: v1alpha1.TidbMonitorSpec{
			Clusters: []v1alpha1.TidbClusterRef{
				cluster,
			},
			Prometheus: v1alpha1.PrometheusSpec{
				MonitorContainer: v1alpha1.MonitorContainer{
					BaseImage: "hub.pingcap.net",
					Version:   "v2.22.2",
				},
				Config: &v1alpha1.PrometheusConfiguration{
					CommandOptions: []string{
						"--web.external-url=https://www.example.com/prometheus/",
					},
				},
			},
		},
	}
}

func newFakeDMCluster(mm *MonitorManager) {
	dmInformer := mm.deps.InformerFactory.Pingcap().V1alpha1().DMClusters()
	dmIndexer := dmInformer.Informer().GetIndexer()
	dc := &v1alpha1.DMCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dm-test",
			Namespace: "ns",
		},
		Spec: v1alpha1.DMClusterSpec{
			Master: v1alpha1.MasterSpec{Replicas: 1},
		},
	}
	dmIndexer.Add(dc)
}

func newFakeTidbMonitorManager() *MonitorManager {
	fakeDeps := controller.NewFakeDependencies()
	fake := &k8stesting.Fake{
		Resources: []*metav1.APIResourceList{
			{
				GroupVersion: "apiextensions.k8s.io/v1beta1",
				APIResources: []metav1.APIResource{
					{
						Name:    "customresourcedefinitions",
						Group:   "apiextensions.k8s.io",
						Version: "v1beta1",
					},
				},
			},
		},
	}
	discoveryClient := &discoveryfake.FakeDiscovery{
		Fake: fake,
	}

	return &MonitorManager{deps: fakeDeps,
		pvManager:          meta.NewReclaimPolicyManager(fakeDeps),
		discoveryInterface: discoverycachedmemory.NewMemCacheClient(discoveryClient),
	}

}

func errExpectRequeuefunc(g *GomegaWithT, err error, tmm *MonitorManager, tm *v1alpha1.TidbMonitor) {
	g.Expect(controller.IsRequeueError(err)).To(Equal(true))
}
