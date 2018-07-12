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

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/new-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/new-operator/pkg/client/informers/externalversions"
	tcinformers "github.com/pingcap/tidb-operator/new-operator/pkg/client/informers/externalversions/pingcap.com/v1"
	v1listers "github.com/pingcap/tidb-operator/new-operator/pkg/client/listers/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/new-operator/pkg/controller"
	mm "github.com/pingcap/tidb-operator/new-operator/pkg/controller/tidbcluster/membermanager"
	apps "k8s.io/api/apps/v1beta1"
	kubeinformers "k8s.io/client-go/informers"
	appsinformers "k8s.io/client-go/informers/apps/v1beta1"
	kubefake "k8s.io/client-go/kubernetes/fake"
	appslisters "k8s.io/client-go/listers/apps/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

func TestTidbClusterControl(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()

	control, setControl, _ := newFakeTidbClusterControl()
	err := syncTidbClusterControl(tc, setControl, control)
	g.Expect(err).NotTo(HaveOccurred())

	newTC, err := setControl.tcLister.TidbClusters(tc.Namespace).Get(tc.Name)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(newTC.Status.PDStatus.StatefulSet.Name).NotTo(Equal(""))
}

func newFakeTidbClusterControl() (ControlInterface, *fakeStatefulSetControl, StatusUpdaterInterface) {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	setInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Apps().V1beta1().StatefulSets()
	tcInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1().TidbClusters()
	recorder := record.NewFakeRecorder(10)

	setControl := newFakeStatefulSetControl(setInformer, tcInformer)
	statusUpdater := newFakeTidbClusterStatusUpdater(tcInformer)
	pdMemberManager := mm.NewPDMemberManager(setControl, setInformer.Lister())
	control := NewDefaultTidbClusterControl(statusUpdater, pdMemberManager, recorder)

	return control, setControl, statusUpdater
}

func syncTidbClusterControl(tc *v1.TidbCluster, setControl *fakeStatefulSetControl, control ControlInterface) error {
	for tc.Status.PDStatus.StatefulSet.Name == "" {
		err := control.UpdateTidbCluster(tc)
		if err != nil {
			return err
		}
	}

	return nil
}

type fakeStatefulSetControl struct {
	setLister  appslisters.StatefulSetLister
	setIndexer cache.Indexer
	tcLister   v1listers.TidbClusterLister
	tcIndexer  cache.Indexer
}

func newFakeStatefulSetControl(setInformer appsinformers.StatefulSetInformer, tcInformer tcinformers.TidbClusterInformer) *fakeStatefulSetControl {
	return &fakeStatefulSetControl{
		setInformer.Lister(),
		setInformer.Informer().GetIndexer(),
		tcInformer.Lister(),
		tcInformer.Informer().GetIndexer(),
	}
}

func (ssc *fakeStatefulSetControl) CreateStatefulSet(tc *v1.TidbCluster, set *apps.StatefulSet) error {
	return ssc.setIndexer.Add(set)
}

func (ssc *fakeStatefulSetControl) UpdateStatefulSet(tc *v1.TidbCluster, set *apps.StatefulSet) error {
	return nil
}

func (ssc *fakeStatefulSetControl) DeleteStatefulSet(tc *v1.TidbCluster, set *apps.StatefulSet) error {
	return nil
}

var _ controller.StatefulSetControlInterface = &fakeStatefulSetControl{}

type fakeTidbClusterStatusUpdater struct {
	tcLister  v1listers.TidbClusterLister
	tcIndexer cache.Indexer
}

func newFakeTidbClusterStatusUpdater(tcInformer tcinformers.TidbClusterInformer) *fakeTidbClusterStatusUpdater {
	return &fakeTidbClusterStatusUpdater{
		tcInformer.Lister(),
		tcInformer.Informer().GetIndexer(),
	}
}

func (tsu *fakeTidbClusterStatusUpdater) UpdateTidbClusterStatus(tc *v1.TidbCluster, status *v1.TidbClusterStatus) error {
	tc.Status = *status
	return tsu.tcIndexer.Update(tc)
}

var _ StatusUpdaterInterface = &fakeTidbClusterStatusUpdater{}
