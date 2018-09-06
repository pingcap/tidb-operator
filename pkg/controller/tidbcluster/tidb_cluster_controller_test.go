// Copyright 2018 PingCAP, Inc.
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

package tidbcluster

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	mm "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/manager/meta"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

func TestTidbClusterControllerEnqueueTidbCluster(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()
	tcc, _, _ := newFakeTidbClusterController()

	tcc.enqueueTidbCluster(tc)
	g.Expect(tcc.queue.Len()).To(Equal(1))
}

func TestTidbClusterControllerAddStatefuSet(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                    string
		modifySet               func(*v1alpha1.TidbCluster) *apps.StatefulSet
		addTidbClusterToIndexer bool
		expectedLen             int
	}

	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbCluster()
		set := test.modifySet(tc)

		tcc, tcIndexer, _ := newFakeTidbClusterController()

		if test.addTidbClusterToIndexer {
			err := tcIndexer.Add(tc)
			g.Expect(err).NotTo(HaveOccurred())
		}
		tcc.addStatefulSet(set)
		g.Expect(tcc.queue.Len()).To(Equal(test.expectedLen))
	}

	tests := []testcase{
		{
			name: "normal",
			modifySet: func(tc *v1alpha1.TidbCluster) *apps.StatefulSet {
				return newStatefuSet(tc)
			},
			addTidbClusterToIndexer: true,
			expectedLen:             1,
		},
		{
			name: "have deletionTimestamp",
			modifySet: func(tc *v1alpha1.TidbCluster) *apps.StatefulSet {
				set := newStatefuSet(tc)
				set.DeletionTimestamp = &metav1.Time{Time: time.Now().Add(30 * time.Second)}
				return set
			},
			addTidbClusterToIndexer: true,
			expectedLen:             1,
		},
		{
			name: "without controllerRef",
			modifySet: func(tc *v1alpha1.TidbCluster) *apps.StatefulSet {
				set := newStatefuSet(tc)
				set.OwnerReferences = nil
				return set
			},
			addTidbClusterToIndexer: true,
			expectedLen:             0,
		},
		{
			name: "without tidbcluster",
			modifySet: func(tc *v1alpha1.TidbCluster) *apps.StatefulSet {
				return newStatefuSet(tc)
			},
			addTidbClusterToIndexer: false,
			expectedLen:             0,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestTidbClusterControllerUpdateStatefuSet(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                    string
		initial                 func() *v1alpha1.TidbCluster
		initialSet              func(*v1alpha1.TidbCluster) *apps.StatefulSet
		updateSet               func(*apps.StatefulSet) *apps.StatefulSet
		addTidbClusterToIndexer bool
		expectedLen             int
	}

	testFn := func(test *testcase, t *testing.T) {
		tc := test.initial()
		set1 := test.initialSet(tc)
		set2 := test.updateSet(set1)

		tcc, tcIndexer, _ := newFakeTidbClusterController()

		if test.addTidbClusterToIndexer {
			err := tcIndexer.Add(tc)
			g.Expect(err).NotTo(HaveOccurred())
		}
		tcc.updateStatefuSet(set1, set2)
		g.Expect(tcc.queue.Len()).To(Equal(test.expectedLen))
	}

	tests := []testcase{
		{
			name: "normal",
			initial: func() *v1alpha1.TidbCluster {
				return newTidbCluster()
			},
			initialSet: func(tc *v1alpha1.TidbCluster) *apps.StatefulSet {
				return newStatefuSet(tc)
			},
			updateSet: func(set1 *apps.StatefulSet) *apps.StatefulSet {
				set2 := *set1
				set2.ResourceVersion = "1000"
				return &set2
			},
			addTidbClusterToIndexer: true,
			expectedLen:             1,
		},
		{
			name: "same resouceVersion",
			initial: func() *v1alpha1.TidbCluster {
				return newTidbCluster()
			},
			initialSet: func(tc *v1alpha1.TidbCluster) *apps.StatefulSet {
				return newStatefuSet(tc)
			},
			updateSet: func(set1 *apps.StatefulSet) *apps.StatefulSet {
				set2 := *set1
				return &set2
			},
			addTidbClusterToIndexer: true,
			expectedLen:             0,
		},
		{
			name: "without controllerRef",
			initial: func() *v1alpha1.TidbCluster {
				return newTidbCluster()
			},
			initialSet: func(tc *v1alpha1.TidbCluster) *apps.StatefulSet {
				return newStatefuSet(tc)
			},
			updateSet: func(set1 *apps.StatefulSet) *apps.StatefulSet {
				set2 := *set1
				set2.ResourceVersion = "1000"
				set2.OwnerReferences = nil
				return &set2
			},
			addTidbClusterToIndexer: true,
			expectedLen:             0,
		},
		{
			name: "without tidbcluster",
			initial: func() *v1alpha1.TidbCluster {
				return newTidbCluster()
			},
			initialSet: func(tc *v1alpha1.TidbCluster) *apps.StatefulSet {
				return newStatefuSet(tc)
			},
			updateSet: func(set1 *apps.StatefulSet) *apps.StatefulSet {
				set2 := *set1
				set2.ResourceVersion = "1000"
				return &set2
			},
			addTidbClusterToIndexer: false,
			expectedLen:             0,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func alwaysReady() bool { return true }

func newFakeTidbClusterController() (*Controller, cache.Indexer, cache.Indexer) {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	informerFactory := informers.NewSharedInformerFactory(cli, 0)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeCli, 0)

	setInformer := kubeInformerFactory.Apps().V1beta1().StatefulSets()
	svcInformer := kubeInformerFactory.Core().V1().Services()
	pvcInformer := kubeInformerFactory.Core().V1().PersistentVolumeClaims()
	pvInformer := kubeInformerFactory.Core().V1().PersistentVolumes()
	tcInformer := informerFactory.Pingcap().V1alpha1().TidbClusters()
	podInformer := kubeInformerFactory.Core().V1().Pods()
	nodeInformer := kubeInformerFactory.Core().V1().Nodes()

	tcc := NewController(
		kubeCli,
		cli,
		informerFactory,
		kubeInformerFactory,
	)
	tcc.tcListerSynced = alwaysReady
	tcc.setListerSynced = alwaysReady
	recorder := record.NewFakeRecorder(10)

	pdControl := controller.NewFakePDControl()
	svcControl := controller.NewRealServiceControl(
		kubeCli,
		svcInformer.Lister(),
		recorder,
	)
	setControl := controller.NewRealStatefuSetControl(
		kubeCli,
		setInformer.Lister(),
		recorder,
	)
	pvControl := controller.NewRealPVControl(kubeCli, pvcInformer.Lister(), recorder)
	pvcControl := controller.NewRealPVCControl(kubeCli, recorder, pvcInformer.Lister())
	podControl := controller.NewRealPodControl(kubeCli, pdControl, recorder)
	pdScaler := mm.NewPDScaler(pdControl, pvcInformer.Lister(), pvcControl)
	tikvScaler := mm.NewTiKVScaler(pdControl, pvcInformer.Lister(), pvcControl)
	pdUpgrade := mm.NewPDUpgrader()
	tikvUpgrader := mm.NewTiKVUpgrader()
	tidbUpgrader := mm.NewTiDBUpgrader()

	tcc.control = NewDefaultTidbClusterControl(
		NewRealTidbClusterStatusUpdater(cli, tcInformer.Lister()),
		mm.NewPDMemberManager(
			pdControl,
			setControl,
			svcControl,
			setInformer.Lister(),
			svcInformer.Lister(),
			pdScaler,
			pdUpgrade,
		),
		mm.NewTiKVMemberManager(
			pdControl,
			setControl,
			svcControl,
			setInformer.Lister(),
			svcInformer.Lister(),
			podInformer.Lister(),
			nodeInformer.Lister(),
			tikvScaler,
			tikvUpgrader,
		),
		mm.NewTiDBMemberManager(
			controller.NewRealStatefuSetControl(
				kubeCli,
				setInformer.Lister(),
				recorder,
			),
			setInformer.Lister(),
			tidbUpgrader,
		),
		meta.NewReclaimPolicyManager(
			pvcInformer.Lister(),
			pvInformer.Lister(),
			pvControl,
		),
		meta.NewMetaManager(
			pvcInformer.Lister(),
			pvcControl,
			pvInformer.Lister(),
			pvControl,
			podInformer.Lister(),
			podControl,
		),
		recorder,
	)

	return tcc, tcInformer.Informer().GetIndexer(), setInformer.Informer().GetIndexer()
}

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
				ContainerSpec: v1alpha1.ContainerSpec{
					Image: "pd-test-image",
				},
			},
			TiKV: v1alpha1.TiKVSpec{
				ContainerSpec: v1alpha1.ContainerSpec{
					Image: "tikv-test-image",
				},
			},
			TiDB: v1alpha1.TiDBSpec{
				ContainerSpec: v1alpha1.ContainerSpec{
					Image: "tidb-test-image",
				},
			},
		},
	}
}

func newStatefuSet(tc *v1alpha1.TidbCluster) *apps.StatefulSet {
	return &apps.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-statefuset",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tc, controllerKind),
			},
			ResourceVersion: "1",
		},
		Spec: apps.StatefulSetSpec{
			Replicas: &tc.Spec.PD.Replicas,
		},
	}
}
