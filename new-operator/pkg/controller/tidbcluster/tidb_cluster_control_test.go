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
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

func TestTidbClusterControl(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()

	control, setControl, _ := newFakeTidbClusterControl()
	err := syncTidbClusterControl(tc, setControl, control)
	g.Expect(err).NotTo(HaveOccurred())

	newTC, err := setControl.TcLister.TidbClusters(tc.Namespace).Get(tc.Name)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(newTC.Status.PD.StatefulSet).NotTo(Equal(nil))
}

func newFakeTidbClusterControl() (ControlInterface, *controller.FakeStatefulSetControl, StatusUpdaterInterface) {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	setInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Apps().V1beta1().StatefulSets()
	svcInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Services()
	deployInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Apps().V1beta1().Deployments()
	tcInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1().TidbClusters()
	recorder := record.NewFakeRecorder(10)

	setControl := controller.NewFakeStatefulSetControl(setInformer, tcInformer)
	svcControl := controller.NewFakeServiceControl(svcInformer, tcInformer)
	deployControl := controller.NewFakeDeploymentControl(deployInformer, tcInformer)
	statusUpdater := newFakeTidbClusterStatusUpdater(tcInformer)
	pdMemberManager := mm.NewPDMemberManager(setControl, svcControl, setInformer.Lister(), svcInformer.Lister())
	monitorMemberManager := mm.NewMonitorMemberManager(deployControl, svcControl, deployInformer.Lister(), svcInformer.Lister())
	tidbMemberManager := mm.NewTiDBMemberManager(setControl, svcControl, setInformer.Lister(), svcInformer.Lister())
	priTidbMemberManager := mm.NewPriTiDBMemberManager(deployControl, svcControl, deployInformer.Lister(), svcInformer.Lister())
	control := NewDefaultTidbClusterControl(statusUpdater, pdMemberManager, monitorMemberManager, tidbMemberManager, priTidbMemberManager, recorder)

	return control, setControl, statusUpdater
}

func syncTidbClusterControl(tc *v1.TidbCluster, setControl *controller.FakeStatefulSetControl, control ControlInterface) error {
	for tc.Status.PD.StatefulSet == nil {
		err := control.UpdateTidbCluster(tc)
		if err != nil {
			return err
		}
	}

	return nil
}

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
