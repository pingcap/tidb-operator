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

package dmcluster

import (
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/scheme"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	controllerfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDMClusterControllerEnqueueDMCluster(t *testing.T) {
	g := NewGomegaWithT(t)
	dc := newDMCluster()
	dcc, _, _ := newFakeDMClusterController()

	dcc.enqueueDMCluster(dc)
	g.Expect(dcc.queue.Len()).To(Equal(1))
}

func TestDMClusterControllerEnqueueDMClusterFailed(t *testing.T) {
	g := NewGomegaWithT(t)
	dcc, _, _ := newFakeDMClusterController()

	dcc.enqueueDMCluster(struct{}{})
	g.Expect(dcc.queue.Len()).To(Equal(0))
}

func TestDMClusterControllerAddStatefulSet(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                  string
		modifySet             func(*v1alpha1.DMCluster) *apps.StatefulSet
		addDMClusterToIndexer bool
		expectedLen           int
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log("test: ", test.name)

		dc := newDMCluster()
		set := test.modifySet(dc)

		dcc, dcIndexer, _ := newFakeDMClusterController()

		if test.addDMClusterToIndexer {
			err := dcIndexer.Add(dc)
			g.Expect(err).NotTo(HaveOccurred())
		}
		dcc.addStatefulSet(set)
		g.Expect(dcc.queue.Len()).To(Equal(test.expectedLen))
	}

	tests := []testcase{
		{
			name: "normal",
			modifySet: func(dc *v1alpha1.DMCluster) *apps.StatefulSet {
				return newStatefulSet(dc)
			},
			addDMClusterToIndexer: true,
			expectedLen:           1,
		},
		{
			name: "have deletionTimestamp",
			modifySet: func(dc *v1alpha1.DMCluster) *apps.StatefulSet {
				set := newStatefulSet(dc)
				set.DeletionTimestamp = &metav1.Time{Time: time.Now().Add(30 * time.Second)}
				return set
			},
			addDMClusterToIndexer: true,
			expectedLen:           1,
		},
		{
			name: "without controllerRef",
			modifySet: func(dc *v1alpha1.DMCluster) *apps.StatefulSet {
				set := newStatefulSet(dc)
				set.OwnerReferences = nil
				return set
			},
			addDMClusterToIndexer: true,
			expectedLen:           0,
		},
		{
			name: "without dmcluster",
			modifySet: func(dc *v1alpha1.DMCluster) *apps.StatefulSet {
				return newStatefulSet(dc)
			},
			addDMClusterToIndexer: false,
			expectedLen:           0,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestDMClusterControllerUpdateStatefulSet(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                  string
		updateSet             func(*apps.StatefulSet) *apps.StatefulSet
		addDMClusterToIndexer bool
		expectedLen           int
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log("test: ", test.name)

		dc := newDMCluster()
		set1 := newStatefulSet(dc)
		set2 := test.updateSet(set1)

		dcc, dcIndexer, _ := newFakeDMClusterController()

		if test.addDMClusterToIndexer {
			err := dcIndexer.Add(dc)
			g.Expect(err).NotTo(HaveOccurred())
		}
		dcc.updateStatefulSet(set1, set2)
		g.Expect(dcc.queue.Len()).To(Equal(test.expectedLen))
	}

	tests := []testcase{
		{
			name: "normal",
			updateSet: func(set1 *apps.StatefulSet) *apps.StatefulSet {
				set2 := *set1
				set2.ResourceVersion = "1000"
				return &set2
			},
			addDMClusterToIndexer: true,
			expectedLen:           1,
		},
		{
			name: "same resouceVersion",
			updateSet: func(set1 *apps.StatefulSet) *apps.StatefulSet {
				set2 := *set1
				return &set2
			},
			addDMClusterToIndexer: true,
			expectedLen:           0,
		},
		{
			name: "without controllerRef",
			updateSet: func(set1 *apps.StatefulSet) *apps.StatefulSet {
				set2 := *set1
				set2.ResourceVersion = "1000"
				set2.OwnerReferences = nil
				return &set2
			},
			addDMClusterToIndexer: true,
			expectedLen:           0,
		},
		{
			name: "without dmcluster",
			updateSet: func(set1 *apps.StatefulSet) *apps.StatefulSet {
				set2 := *set1
				set2.ResourceVersion = "1000"
				return &set2
			},
			addDMClusterToIndexer: false,
			expectedLen:           0,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestDMClusterControllerSync(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                   string
		addDcToIndexer         bool
		errWhenUpdateDMCluster bool
		errExpectFn            func(*GomegaWithT, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		dc := newDMCluster()
		dcc, dcIndexer, dcControl := newFakeDMClusterController()

		if test.addDcToIndexer {
			err := dcIndexer.Add(dc)
			g.Expect(err).NotTo(HaveOccurred())
		}
		key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(dc)
		g.Expect(err).NotTo(HaveOccurred())

		if test.errWhenUpdateDMCluster {
			dcControl.SetUpdateDCError(fmt.Errorf("update dm cluster failed"))
		}

		err = dcc.sync(key)

		if test.errExpectFn != nil {
			test.errExpectFn(g, err)
		}
	}

	tests := []testcase{
		{
			name:                   "normal",
			addDcToIndexer:         true,
			errWhenUpdateDMCluster: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name:                   "can't found dm cluster",
			addDcToIndexer:         false,
			errWhenUpdateDMCluster: false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name:                   "update dm cluster failed",
			addDcToIndexer:         true,
			errWhenUpdateDMCluster: true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "update dm cluster failed")).To(Equal(true))
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}

}

func alwaysReady() bool { return true }

func newFakeDMClusterController() (*Controller, cache.Indexer, *FakeDMClusterControlInterface) {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	genericCli := controllerfake.NewFakeClientWithScheme(scheme.Scheme)
	informerFactory := informers.NewSharedInformerFactory(cli, 0)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeCli, 0)

	dcInformer := informerFactory.Pingcap().V1alpha1().DMClusters()
	autoFailover := true
	dcControl := NewFakeDMClusterControlInterface()

	dcc := NewController(
		kubeCli,
		cli,
		genericCli,
		informerFactory,
		kubeInformerFactory,
		autoFailover,
		5*time.Minute,
		5*time.Minute,
	)
	dcc.dcListerSynced = alwaysReady
	dcc.setListerSynced = alwaysReady

	dcc.control = dcControl
	return dcc, dcInformer.Informer().GetIndexer(), dcControl
}

func newDMCluster() *v1alpha1.DMCluster {
	return &v1alpha1.DMCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DMCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dm-master",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
		},
		Spec: v1alpha1.DMClusterSpec{
			Version:   "v2.0.0-rc.2",
			Discovery: v1alpha1.DMDiscoverySpec{Address: "http://basic-discovery.demo:10261"},
			Master: v1alpha1.MasterSpec{
				Replicas:    3,
				BaseImage:   "pingcap/dm",
				Config:      &v1alpha1.MasterConfig{},
				StorageSize: "10Gi",
			},
			Worker: &v1alpha1.WorkerSpec{
				Replicas:    3,
				BaseImage:   "pingcap/dm",
				Config:      &v1alpha1.WorkerConfig{},
				StorageSize: "10Gi",
			},
		},
	}
}

func newStatefulSet(dc *v1alpha1.DMCluster) *apps.StatefulSet {
	return &apps.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-statefuset",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(dc, controller.DMControllerKind),
			},
			ResourceVersion: "1",
		},
		Spec: apps.StatefulSetSpec{
			Replicas: &dc.Spec.Master.Replicas,
		},
	}
}
