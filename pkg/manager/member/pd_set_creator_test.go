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

package member

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestPDSetCreatorCreatePDStatefulSet(t *testing.T) {
	g := NewGomegaWithT(t)
	var pdReplicas int32
	type testcase struct {
		name                     string
		update                   func(tc *v1alpha1.TidbCluster)
		hasStatefulSet           bool
		hasOldPVC                bool
		errWhenCreateStatefulSet bool
		errExpectFn              func(*GomegaWithT, error)
		setExpectFn              func(*GomegaWithT, *apps.StatefulSet, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		psc, setIndexer, pvcIndexer, fakeSetControl := newFakePDSetCreator()
		tc := newTidbClusterForPD()
		pdReplicas = tc.Spec.PD.Replicas
		ns := tc.GetNamespace()
		tcName := tc.GetName()
		setName := controller.PDMemberName(tcName)
		if test.update != nil {
			test.update(tc)
		}
		if test.hasStatefulSet {
			set := &apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      setName,
					Namespace: ns,
				},
			}
			setIndexer.Add(set)
		}
		if test.hasOldPVC {
			pvcName := ordinalPVCName(v1alpha1.PDMemberType, setName, tc.Spec.PD.Replicas-int32(1))
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pvcName,
					Namespace: ns,
				},
			}
			pvcIndexer.Add(pvc)
		}
		if test.errWhenCreateStatefulSet {
			fakeSetControl.SetCreateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed when creating StatefulSet")), 0)
		}

		err := psc.CreatePDStatefulSet(tc, tc.Spec.PD)
		test.errExpectFn(g, err)

		set, err := psc.setLister.StatefulSets(ns).Get(setName)
		test.setExpectFn(g, set, err)
	}
	tests := []testcase{
		{
			name: "normal",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD.StorageClassName = ""
			},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(controller.IsRequeueError(err)).To(Equal(true))
				g.Expect(strings.Contains(err.Error(), "TidbCluster: [default/test], StatefulSet: [default/test-pd] created, waiting for PD cluster running")).To(BeTrue())
			},
			setExpectFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(set).NotTo(BeNil())
				g.Expect(*set.Spec.VolumeClaimTemplates[0].Spec.StorageClassName).To(Equal(controller.DefaultStorageClassName))
			},
		},
		{
			name: "storageClassName is abc",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD.StorageClassName = "abc"
			},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(controller.IsRequeueError(err)).To(Equal(true))
				g.Expect(strings.Contains(err.Error(), "TidbCluster: [default/test], StatefulSet: [default/test-pd] created, waiting for PD cluster running")).To(BeTrue())
			},
			setExpectFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(set).NotTo(BeNil())
				g.Expect(*set.Spec.VolumeClaimTemplates[0].Spec.StorageClassName).To(Equal("abc"))
			},
		},
		{
			name: "requests.storage format is wrong",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PD.Requests.Storage = "100Gxxxxi"
			},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "get storage size: 100Gxxxxi for TidbCluster")).To(BeTrue())
			},
			setExpectFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(set).To(BeNil())
			},
		},
		{
			name:           "has statefulset already",
			hasStatefulSet: true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
			setExpectFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(set).NotTo(BeNil())
			},
		},
		{
			name:      "recreated from the old cluster",
			hasOldPVC: true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(controller.IsRequeueError(err)).To(Equal(true))
				g.Expect(strings.Contains(err.Error(), "TidbCluster: [default/test], StatefulSet: [default/test-pd] created, waiting for PD cluster running")).To(BeTrue())
			},
			setExpectFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(set).NotTo(BeNil())
				g.Expect(*set.Spec.Replicas).To(Equal(pdReplicas))
			},
		},
		{
			name:                     "creating statefulset failed",
			errWhenCreateStatefulSet: true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "API server failed when creating StatefulSet")).To(BeTrue())
			},
			setExpectFn: func(g *GomegaWithT, set *apps.StatefulSet, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(set).To(BeNil())
			},
		},
	}
	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestPDSetCreatorCreateExtraPDStatefulSets(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                     string
		update                   func(tc *v1alpha1.TidbCluster)
		pdSets                   []*apps.StatefulSet
		errWhenCreateStatefulSet bool
		errExpectFn              func(*GomegaWithT, error)
		setExpectFn              func(*GomegaWithT, []*apps.StatefulSet, error)
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		psc, setIndexer, _, fakeSetControl := newFakePDSetCreator()
		tc := newTidbClusterForPD()
		ns := tc.GetNamespace()
		if test.update != nil {
			test.update(tc)
		}
		if test.errWhenCreateStatefulSet {
			fakeSetControl.SetCreateStatefulSetError(errors.NewInternalError(fmt.Errorf("API server failed when creating StatefulSet")), 0)
		}
		for _, set := range test.pdSets {
			setIndexer.Add(set)
		}

		err := psc.CreateExtraPDStatefulSets(tc)
		test.errExpectFn(g, err)

		setList, err := psc.setLister.StatefulSets(ns).List(labels.Everything())
		test.setExpectFn(g, setList, err)
	}
	tests := []testcase{
		{
			name: "one extra pd, should create it",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PDs = []v1alpha1.PDSpec{
					{
						Name: "extra-1",
						ContainerSpec: v1alpha1.ContainerSpec{
							Image: "pd-test-image",
						},
						Replicas: 3,
					},
				}
			},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(controller.IsRequeueError(err)).To(Equal(true))
				g.Expect(strings.Contains(err.Error(), "TidbCluster: [default/test], StatefulSet: [default/test-extra-1-pd] created, waiting for PD cluster running")).To(BeTrue())
			},
			setExpectFn: func(g *GomegaWithT, sets []*apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(sets)).To(Equal(1))
				g.Expect(sets[0]).NotTo(BeNil())
				g.Expect(sets[0].GetName()).To(Equal("test-extra-1-pd"))
			},
		},
		{
			name: "two extra pds, should create the first one",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PDs = []v1alpha1.PDSpec{
					{
						Name: "extra-1",
						ContainerSpec: v1alpha1.ContainerSpec{
							Image: "pd-test-image",
						},
						Replicas: 3,
					},
					{
						Name: "extra-2",
						ContainerSpec: v1alpha1.ContainerSpec{
							Image: "pd-test-image",
						},
						Replicas: 3,
					},
				}
			},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(controller.IsRequeueError(err)).To(Equal(true))
				g.Expect(strings.Contains(err.Error(), "TidbCluster: [default/test], StatefulSet: [default/test-extra-1-pd] created, waiting for PD cluster running")).To(BeTrue())
			},
			setExpectFn: func(g *GomegaWithT, sets []*apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(sets)).To(Equal(1))
				g.Expect(sets[0]).NotTo(BeNil())
				g.Expect(sets[0].GetName()).To(Equal("test-extra-1-pd"))
			},
		},
		{
			name: "two extra pds, one statefulset, should create the second one",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PDs = []v1alpha1.PDSpec{
					{
						Name: "extra-1",
						ContainerSpec: v1alpha1.ContainerSpec{
							Image: "pd-test-image",
						},
						Replicas: 3,
					},
					{
						Name: "extra-2",
						ContainerSpec: v1alpha1.ContainerSpec{
							Image: "pd-test-image",
						},
						Replicas: 3,
					},
				}
			},
			pdSets: []*apps.StatefulSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-extra-1-pd",
						Namespace: "default",
					},
				},
			},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(controller.IsRequeueError(err)).To(Equal(true))
				g.Expect(strings.Contains(err.Error(), "TidbCluster: [default/test], StatefulSet: [default/test-extra-2-pd] created, waiting for PD cluster running")).To(BeTrue())
			},
			setExpectFn: func(g *GomegaWithT, sets []*apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(sets)).To(Equal(2))
				names := []string{sets[0].GetName(), sets[1].GetName()}
				sort.Strings(names)
				g.Expect(names).To(Equal([]string{"test-extra-1-pd", "test-extra-2-pd"}))
			},
		},
		{
			name: "two extra pds, two statefulsets, no need to create",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.PDs = []v1alpha1.PDSpec{
					{
						Name: "extra-1",
						ContainerSpec: v1alpha1.ContainerSpec{
							Image: "pd-test-image",
						},
						Replicas: 3,
					},
					{
						Name: "extra-2",
						ContainerSpec: v1alpha1.ContainerSpec{
							Image: "pd-test-image",
						},
						Replicas: 3,
					},
				}
			},
			pdSets: []*apps.StatefulSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-extra-1-pd",
						Namespace: "default",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-extra-2-pd",
						Namespace: "default",
					},
				},
			},
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(BeNil())
			},
			setExpectFn: func(g *GomegaWithT, sets []*apps.StatefulSet, err error) {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(len(sets)).To(Equal(2))
				names := []string{sets[0].GetName(), sets[1].GetName()}
				sort.Strings(names)
				g.Expect(names).To(Equal([]string{"test-extra-1-pd", "test-extra-2-pd"}))
			},
		},
	}
	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newFakePDSetCreator() (*pdSetCreator, cache.Indexer, cache.Indexer, *controller.FakeStatefulSetControl) {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	setInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Apps().V1beta1().StatefulSets()
	pvcInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().PersistentVolumeClaims()
	tcInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1alpha1().TidbClusters()
	setControl := controller.NewFakeStatefulSetControl(setInformer, tcInformer)

	return &pdSetCreator{
		setInformer.Lister(),
		pvcInformer.Lister(),
		setControl}, setInformer.Informer().GetIndexer(), pvcInformer.Informer().GetIndexer(), setControl

}
