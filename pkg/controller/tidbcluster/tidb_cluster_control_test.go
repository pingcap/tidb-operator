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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	tcinformers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions/pingcap.com/v1alpha1"
	v1listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	mm "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/manager/meta"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

func TestTidbClusterControl(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbCluster()

	pdClient := controller.NewFakePDClient()
	pdClient.AddReaction(controller.GetHealthActionType, func(action *controller.Action) (interface{}, error) {
		return &controller.HealthInfo{
			Healths: []controller.MemberHealth{
				{Name: "pd1", MemberID: 1, ClientUrls: []string{"http://pd1:2379"}, Health: true},
				{Name: "pd2", MemberID: 2, ClientUrls: []string{"http://pd2:2379"}, Health: true},
				{Name: "pd3", MemberID: 3, ClientUrls: []string{"http://pd3:2379"}, Health: false},
			},
		}, nil
	})

	control, setControl, _, pdControl := newFakeTidbClusterControl()
	pdControl.SetPDClient(tc, pdClient)

	err := syncTidbClusterControl(tc, setControl, control)
	g.Expect(err).NotTo(HaveOccurred())

	newTC, err := setControl.TcLister.TidbClusters(tc.Namespace).Get(tc.Name)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(newTC.Status.PD.StatefulSet).NotTo(Equal(nil))
}

func newFakeTidbClusterControl() (ControlInterface, *controller.FakeStatefulSetControl, StatusUpdaterInterface, *controller.FakePDControl) {
	cli := fake.NewSimpleClientset()
	kubeCli := kubefake.NewSimpleClientset()
	setInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Apps().V1beta1().StatefulSets()
	svcInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Services()
	tcInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1alpha1().TidbClusters()
	pvcInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().PersistentVolumeClaims()
	pvInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().PersistentVolumes()
	recorder := record.NewFakeRecorder(10)
	podInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Pods()
	nodeInformer := kubeinformers.NewSharedInformerFactory(kubeCli, 0).Core().V1().Nodes()

	pdControl := controller.NewFakePDControl()
	setControl := controller.NewFakeStatefulSetControl(setInformer, tcInformer)
	svcControl := controller.NewFakeServiceControl(svcInformer, tcInformer)
	pvControl := controller.NewRealPVControl(kubeCli, pvcInformer.Lister(), recorder)
	pvcControl := controller.NewRealPVCControl(kubeCli, recorder)
	podControl := controller.NewRealPodControl(kubeCli, pdControl, recorder)
	statusUpdater := newFakeTidbClusterStatusUpdater(tcInformer)

	pdMemberManager := mm.NewPDMemberManager(pdControl, setControl, svcControl, setInformer.Lister(), svcInformer.Lister())
	tikvMemberManager := mm.NewTiKVMemberManager(pdControl, setControl, svcControl, setInformer.Lister(), svcInformer.Lister(), podInformer.Lister(), nodeInformer.Lister())
	tidbMemberManager := mm.NewTiDBMemberManager(setControl, svcControl, setInformer.Lister(), svcInformer.Lister())
	reclaimPolicyManager := meta.NewReclaimPolicyManager(pvcInformer.Lister(), pvInformer.Lister(), pvControl)
	metaManager := meta.NewMetaManager(pvcInformer.Lister(), pvcControl, pvInformer.Lister(), pvControl, podInformer.Lister(), podControl)
	control := NewDefaultTidbClusterControl(statusUpdater, pdMemberManager, tikvMemberManager, tidbMemberManager, reclaimPolicyManager, metaManager, recorder)

	return control, setControl, statusUpdater, pdControl
}

func syncTidbClusterControl(tc *v1alpha1.TidbCluster, setControl *controller.FakeStatefulSetControl, control ControlInterface) error {
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

func (tsu *fakeTidbClusterStatusUpdater) UpdateTidbClusterStatus(tc *v1alpha1.TidbCluster, status *v1alpha1.TidbClusterStatus) error {
	tc.Status = *status
	return tsu.tcIndexer.Update(tc)
}

var _ StatusUpdaterInterface = &fakeTidbClusterStatusUpdater{}
