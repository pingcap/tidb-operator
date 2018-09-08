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
	"github.com/pingcap/tidb-operator/pkg/controller"
	mm "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/manager/meta"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
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

	control, setControl, pdControl := newFakeTidbClusterControl()
	pdControl.SetPDClient(tc, pdClient)

	err := syncTidbClusterControl(tc, setControl, control)
	g.Expect(err).NotTo(HaveOccurred())

	newTC, err := setControl.TcLister.TidbClusters(tc.Namespace).Get(tc.Name)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(newTC.Status.PD.StatefulSet).NotTo(Equal(nil))
}

func TestTidbClusterStatusEquality(t *testing.T) {
	g := NewGomegaWithT(t)
	tcStatus := v1alpha1.TidbClusterStatus{}

	tcStatusCopy := tcStatus.DeepCopy()
	tcStatusCopy.PD = v1alpha1.PDStatus{}
	g.Expect(apiequality.Semantic.DeepEqual(&tcStatus, tcStatusCopy)).To(Equal(true))

	tcStatusCopy = tcStatus.DeepCopy()
	tcStatusCopy.PD.Phase = v1alpha1.NormalPhase
	g.Expect(apiequality.Semantic.DeepEqual(&tcStatus, tcStatusCopy)).To(Equal(false))
}

func newFakeTidbClusterControl() (ControlInterface, *controller.FakeStatefulSetControl, *controller.FakePDControl) {
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
	pvcControl := controller.NewRealPVCControl(kubeCli, recorder, pvcInformer.Lister())
	podControl := controller.NewRealPodControl(kubeCli, pdControl, podInformer.Lister(), recorder)
	tcControl := controller.NewFakeTidbClusterControl(tcInformer)
	pdScaler := mm.NewFakePDScaler()
	tikvScaler := mm.NewFakeTiKVScaler()
	autoFailover := true
	pdFailover := mm.NewFakePDFailover()
	pdUpgrader := mm.NewFakePDUpgrader()
	tikvFailover := mm.NewFakeTiKVFailover()
	tikvUpgrader := mm.NewFakeTiKVUpgrader()
	tidbUpgrader := mm.NewFakeTiDBUpgrader()

	pdMemberManager := mm.NewPDMemberManager(pdControl, setControl, svcControl, setInformer.Lister(), svcInformer.Lister(), podInformer.Lister(), podControl, pvcInformer.Lister(), pdScaler, pdUpgrader, autoFailover, pdFailover)
	tikvMemberManager := mm.NewTiKVMemberManager(pdControl, setControl, svcControl, setInformer.Lister(), svcInformer.Lister(), podInformer.Lister(), nodeInformer.Lister(), autoFailover, tikvFailover, tikvScaler, tikvUpgrader)
	tidbMemberManager := mm.NewTiDBMemberManager(setControl, svcControl, setInformer.Lister(), svcInformer.Lister(), tidbUpgrader)
	reclaimPolicyManager := meta.NewReclaimPolicyManager(pvcInformer.Lister(), pvInformer.Lister(), pvControl)
	metaManager := meta.NewMetaManager(pvcInformer.Lister(), pvcControl, pvInformer.Lister(), pvControl, podInformer.Lister(), podControl)
	control := NewDefaultTidbClusterControl(tcControl, pdMemberManager, tikvMemberManager, tidbMemberManager, reclaimPolicyManager, metaManager, recorder)

	return control, setControl, pdControl
}

func syncTidbClusterControl(tc *v1alpha1.TidbCluster, _ *controller.FakeStatefulSetControl, control ControlInterface) error {
	for tc.Status.PD.StatefulSet == nil {
		err := control.UpdateTidbCluster(tc)
		if err != nil {
			return err
		}
	}

	return nil
}
