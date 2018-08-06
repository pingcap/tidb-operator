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
	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/new-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/new-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/new-operator/pkg/controller"
	mm "github.com/pingcap/tidb-operator/new-operator/pkg/controller/tidbcluster/membermanager"
	"github.com/pingcap/tidb-operator/new-operator/pkg/manager/meta"
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
		modifySet               func(*v1.TidbCluster) *apps.StatefulSet
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
			modifySet: func(tc *v1.TidbCluster) *apps.StatefulSet {
				return newStatefuSet(tc)
			},
			addTidbClusterToIndexer: true,
			expectedLen:             1,
		},
		{
			name: "have deletionTimestamp",
			modifySet: func(tc *v1.TidbCluster) *apps.StatefulSet {
				set := newStatefuSet(tc)
				set.DeletionTimestamp = &metav1.Time{Time: time.Now().Add(30 * time.Second)}
				return set
			},
			addTidbClusterToIndexer: true,
			expectedLen:             1,
		},
		{
			name: "without controllerRef",
			modifySet: func(tc *v1.TidbCluster) *apps.StatefulSet {
				set := newStatefuSet(tc)
				set.OwnerReferences = nil
				return set
			},
			addTidbClusterToIndexer: true,
			expectedLen:             0,
		},
		{
			name: "without tidbcluster",
			modifySet: func(tc *v1.TidbCluster) *apps.StatefulSet {
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
		initial                 func() *v1.TidbCluster
		initialSet              func(*v1.TidbCluster) *apps.StatefulSet
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
			initial: func() *v1.TidbCluster {
				return newTidbCluster()
			},
			initialSet: func(tc *v1.TidbCluster) *apps.StatefulSet {
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
			initial: func() *v1.TidbCluster {
				return newTidbCluster()
			},
			initialSet: func(tc *v1.TidbCluster) *apps.StatefulSet {
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
			initial: func() *v1.TidbCluster {
				return newTidbCluster()
			},
			initialSet: func(tc *v1.TidbCluster) *apps.StatefulSet {
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
			initial: func() *v1.TidbCluster {
				return newTidbCluster()
			},
			initialSet: func(tc *v1.TidbCluster) *apps.StatefulSet {
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

func TestTidbClusterControllerAddPVC(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                    string
		modifyPVC               func(*v1.TidbCluster) *corev1.PersistentVolumeClaim
		addTidbClusterToIndexer bool
		expectedLen             int
	}

	testFn := func(test *testcase, t *testing.T) {
		tc := newTidbCluster()
		pvc := test.modifyPVC(tc)

		tcc, tcIndexer, _ := newFakeTidbClusterController()

		if test.addTidbClusterToIndexer {
			err := tcIndexer.Add(tc)
			g.Expect(err).NotTo(HaveOccurred())
		}
		tcc.addPVC(pvc)
		g.Expect(tcc.queue.Len()).To(Equal(test.expectedLen))
	}

	tests := []testcase{
		{
			name: "normal",
			modifyPVC: func(tc *v1.TidbCluster) *corev1.PersistentVolumeClaim {
				return newPVC(tc)
			},
			addTidbClusterToIndexer: true,
			expectedLen:             1,
		},
		{
			name: "have deletionTimestamp",
			modifyPVC: func(tc *v1.TidbCluster) *corev1.PersistentVolumeClaim {
				pvc := newPVC(tc)
				pvc.DeletionTimestamp = &metav1.Time{Time: time.Now().Add(30 * time.Second)}
				return pvc
			},
			addTidbClusterToIndexer: true,
			expectedLen:             1,
		},
		{
			name: "without label",
			modifyPVC: func(tc *v1.TidbCluster) *corev1.PersistentVolumeClaim {
				pvc := newPVC(tc)
				pvc.Labels = nil
				return pvc
			},
			addTidbClusterToIndexer: true,
			expectedLen:             0,
		},
		{
			name: "without tidbcluster",
			modifyPVC: func(tc *v1.TidbCluster) *corev1.PersistentVolumeClaim {
				return newPVC(tc)
			},
			addTidbClusterToIndexer: false,
			expectedLen:             0,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestTidbClusterControllerUpdatePVC(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                    string
		initial                 func() *v1.TidbCluster
		initialPVC              func(*v1.TidbCluster) *corev1.PersistentVolumeClaim
		updatePVC               func(*corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim
		addTidbClusterToIndexer bool
		expectedLen             int
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		tc := test.initial()
		pvc1 := test.initialPVC(tc)
		pvc2 := test.updatePVC(pvc1)

		tcc, tcIndexer, _ := newFakeTidbClusterController()

		if test.addTidbClusterToIndexer {
			err := tcIndexer.Add(tc)
			g.Expect(err).NotTo(HaveOccurred())
		}
		tcc.updatePVC(pvc1, pvc2)
		g.Expect(tcc.queue.Len()).To(Equal(test.expectedLen))
	}

	tests := []testcase{
		{
			name: "normal",
			initial: func() *v1.TidbCluster {
				return newTidbCluster()
			},
			initialPVC: func(tc *v1.TidbCluster) *corev1.PersistentVolumeClaim {
				return newPVC(tc)
			},
			updatePVC: func(pvc1 *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
				pvc2 := *pvc1
				pvc2.ResourceVersion = "1000"
				pvc2.Spec.VolumeName = "pv-1"
				return &pvc2
			},
			addTidbClusterToIndexer: true,
			expectedLen:             1,
		},
		{
			name: "same resouceVersion",
			initial: func() *v1.TidbCluster {
				return newTidbCluster()
			},
			initialPVC: func(tc *v1.TidbCluster) *corev1.PersistentVolumeClaim {
				return newPVC(tc)
			},
			updatePVC: func(pvc1 *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
				pvc2 := *pvc1
				return &pvc2
			},
			addTidbClusterToIndexer: true,
			expectedLen:             0,
		},
		{
			name: "without label",
			initial: func() *v1.TidbCluster {
				return newTidbCluster()
			},
			initialPVC: func(tc *v1.TidbCluster) *corev1.PersistentVolumeClaim {
				return newPVC(tc)
			},
			updatePVC: func(pvc1 *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
				pvc2 := *pvc1
				pvc2.ResourceVersion = "1000"
				pvc2.Labels = nil
				return &pvc2
			},
			addTidbClusterToIndexer: true,
			expectedLen:             0,
		},
		{
			name: "without tidbcluster",
			initial: func() *v1.TidbCluster {
				return newTidbCluster()
			},
			initialPVC: func(tc *v1.TidbCluster) *corev1.PersistentVolumeClaim {
				return newPVC(tc)
			},
			updatePVC: func(pvc1 *corev1.PersistentVolumeClaim) *corev1.PersistentVolumeClaim {
				pvc2 := *pvc1
				pvc2.ResourceVersion = "1000"
				pvc2.Spec.VolumeName = "pv-1"
				return &pvc2
			},
			addTidbClusterToIndexer: false,
			expectedLen:             0,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

// TODO add deleteStatefulSet and delete PVC specs

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
	tcInformer := informerFactory.Pingcap().V1().TidbClusters()
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

	pdControl := controller.NewDefaultPDControl()
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
	pvControl := controller.NewRealPVControl(kubeCli, recorder)
	tcc.control = NewDefaultTidbClusterControl(
		NewRealTidbClusterStatusUpdater(cli, tcInformer.Lister()),
		mm.NewPDMemberManager(
			pdControl,
			setControl,
			svcControl,
			setInformer.Lister(),
			svcInformer.Lister(),
		),
		mm.NewTiKVMemberManager(
			pdControl,
			setControl,
			svcControl,
			setInformer.Lister(),
			svcInformer.Lister(),
			podInformer.Lister(),
			nodeInformer.Lister(),
		),
		mm.NewTiDBMemberManager(
			controller.NewRealStatefuSetControl(
				kubeCli,
				setInformer.Lister(),
				recorder,
			),
			svcControl,
			setInformer.Lister(),
			svcInformer.Lister(),
		),
		meta.NewReclaimPolicyManager(
			pvcInformer.Lister(),
			pvInformer.Lister(),
			pvControl,
		),
		recorder,
	)

	return tcc, tcInformer.Informer().GetIndexer(), setInformer.Informer().GetIndexer()
}

func newTidbCluster() *v1.TidbCluster {
	return &v1.TidbCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TidbCluster",
			APIVersion: "pingcap.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pd",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
		},
		Spec: v1.TidbClusterSpec{
			PD: v1.PDSpec{
				ContainerSpec: v1.ContainerSpec{
					Image: "pd-test-image",
				},
			},
			TiKV: v1.TiKVSpec{
				ContainerSpec: v1.ContainerSpec{
					Image: "tikv-test-image",
				},
			},
			TiDB: v1.TiDBSpec{
				ContainerSpec: v1.ContainerSpec{
					Image: "tidb-test-image",
				},
			},
		},
	}
}

func newStatefuSet(tc *v1.TidbCluster) *apps.StatefulSet {
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

func newPVC(tc *v1.TidbCluster) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-pvc",
			Namespace:       corev1.NamespaceDefault,
			UID:             types.UID("test"),
			ResourceVersion: "1",
			Labels: map[string]string{
				"cluster.pingcap.com/app":         "pd",
				"cluster.pingcap.com/owner":       "tidbCluster",
				"cluster.pingcap.com/tidbCluster": tc.GetName(),
			},
		},
	}
}
