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
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/meta"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	discoverycachedmemory "k8s.io/client-go/discovery/cached/memory"
	discoveryfake "k8s.io/client-go/discovery/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/utils/pointer"
)

func TestTidbMonitorSyncCreate(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name          string
		prepare       func(monitor *v1alpha1.TidbMonitor)
		errExpectFn   func(*GomegaWithT, error)
		volumeCreated bool
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		tmm := newFakeTidbMonitorManager()
		tc := &v1alpha1.TidbCluster{
			Spec: v1alpha1.TidbClusterSpec{
				TLSCluster: &v1alpha1.TLSCluster{Enabled: true},
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
		_, err := tmm.deps.Clientset.PingcapV1alpha1().TidbClusters(tc.Namespace).Create(tc)
		g.Expect(err).Should(BeNil())

		tm := newTidbMonitor(v1alpha1.TidbClusterRef{Name: tc.Name, Namespace: tc.Namespace})
		ns := tm.Namespace
		if test.prepare != nil {
			test.prepare(tm)
		}

		err = tmm.SyncMonitor(tm)

		if test.errExpectFn != nil {
			test.errExpectFn(g, err)
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}

		if test.volumeCreated {
			sts, err := tmm.deps.StatefulSetLister.StatefulSets(ns).Get(GetMonitorObjectName(tm))
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(sts).NotTo(Equal(nil))
			quantity, err := resource.ParseQuantity(tm.Spec.Storage)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(sts.Spec.VolumeClaimTemplates).To(Equal([]v1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "monitor-data"},
					Spec: v1.PersistentVolumeClaimSpec{
						AccessModes: []v1.PersistentVolumeAccessMode{
							v1.ReadWriteOnce,
						},
						StorageClassName: pointer.StringPtr(""),
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
			name: "enable monitor persistent",
			prepare: func(monitor *v1alpha1.TidbMonitor) {
			},
			errExpectFn:   nil,
			volumeCreated: false,
		},
		{
			name: "not set clusters field",
			prepare: func(monitor *v1alpha1.TidbMonitor) {
				monitor.Spec.Clusters = nil
			},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "does not configure the target tidbcluster")).To(BeTrue())
			},
		},
		{
			name: "normal",
			prepare: func(monitor *v1alpha1.TidbMonitor) {
			},
			errExpectFn: func(g *GomegaWithT, err error) {

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
					Version:   "latest",
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
	monitorManager := &MonitorManager{
		deps:               fakeDeps,
		pvManager:          meta.NewReclaimPolicyManager(fakeDeps),
		discoveryInterface: discoverycachedmemory.NewMemCacheClient(discoveryClient),
	}

	return monitorManager
}
