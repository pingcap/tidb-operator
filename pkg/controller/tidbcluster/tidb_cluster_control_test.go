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
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	mm "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/manager/meta"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
)

func TestTidbClusterControlUpdateTidbCluster(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name                     string
		update                   func(cluster *v1alpha1.TidbCluster)
		syncReclaimPolicyErr     bool
		syncPDMemberManagerErr   bool
		syncTiKVMemberManagerErr bool
		syncTiDBMemberManagerErr bool
		syncMetaManagerErr       bool
		errExpectFn              func(*GomegaWithT, error)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbClusterForTidbClusterControl()
		if test.update != nil {
			test.update(tc)
		}
		control, reclaimPolicyManager, pdMemberManager, tikvMemberManager, tidbMemberManager, metaManager := newFakeTidbClusterControl()

		if test.syncReclaimPolicyErr {
			reclaimPolicyManager.SetSyncError(fmt.Errorf("reclaim policy sync error"))
		}
		if test.syncPDMemberManagerErr {
			pdMemberManager.SetSyncError(fmt.Errorf("pd member manager sync error"))
		}
		if test.syncTiKVMemberManagerErr {
			tikvMemberManager.SetSyncError(fmt.Errorf("tikv member manager sync error"))
		}
		if test.syncTiDBMemberManagerErr {
			tidbMemberManager.SetSyncError(fmt.Errorf("tidb member manager sync error"))
		}
		if test.syncMetaManagerErr {
			metaManager.SetSyncError(fmt.Errorf("meta manager sync error"))
		}

		err := control.UpdateTidbCluster(tc)
		if test.errExpectFn != nil {
			test.errExpectFn(g, err)
		}
	}
	tests := []testcase{
		{
			name:                     "reclaim policy sync error",
			update:                   nil,
			syncReclaimPolicyErr:     true,
			syncPDMemberManagerErr:   false,
			syncTiKVMemberManagerErr: false,
			syncTiDBMemberManagerErr: false,
			syncMetaManagerErr:       false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "reclaim policy sync error")).To(Equal(true))
			},
		},
		{
			name:                     "pd member manager sync error",
			update:                   nil,
			syncReclaimPolicyErr:     false,
			syncPDMemberManagerErr:   true,
			syncTiKVMemberManagerErr: false,
			syncTiDBMemberManagerErr: false,
			syncMetaManagerErr:       false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "pd member manager sync error")).To(Equal(true))
			},
		},
		{
			name: "tikv member manager sync error",
			update: func(cluster *v1alpha1.TidbCluster) {
				cluster.Status.PD.Members = map[string]v1alpha1.PDMember{
					"pd-0": {Name: "pd-0", Health: true},
					"pd-1": {Name: "pd-1", Health: true},
					"pd-2": {Name: "pd-2", Health: true},
				}
				cluster.Status.PD.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 3}
			},
			syncReclaimPolicyErr:     false,
			syncPDMemberManagerErr:   false,
			syncTiKVMemberManagerErr: true,
			syncTiDBMemberManagerErr: false,
			syncMetaManagerErr:       false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "tikv member manager sync error")).To(Equal(true))
			},
		},
		{
			name: "tidb member manager sync error",
			update: func(cluster *v1alpha1.TidbCluster) {
				cluster.Status.PD.Members = map[string]v1alpha1.PDMember{
					"pd-0": {Name: "pd-0", Health: true},
					"pd-1": {Name: "pd-1", Health: true},
					"pd-2": {Name: "pd-2", Health: true},
				}
				cluster.Status.PD.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 3}
				cluster.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"tikv-0": {PodName: "tikv-0", State: v1alpha1.TiKVStateUp},
					"tikv-1": {PodName: "tikv-1", State: v1alpha1.TiKVStateUp},
					"tikv-2": {PodName: "tikv-2", State: v1alpha1.TiKVStateUp},
				}
				cluster.Status.TiKV.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 3}
			},
			syncReclaimPolicyErr:     false,
			syncPDMemberManagerErr:   false,
			syncTiKVMemberManagerErr: false,
			syncTiDBMemberManagerErr: true,
			syncMetaManagerErr:       false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "tidb member manager sync error")).To(Equal(true))
			},
		},
		{
			name: "meta manager sync error",
			update: func(cluster *v1alpha1.TidbCluster) {
				cluster.Status.PD.Members = map[string]v1alpha1.PDMember{
					"pd-0": {Name: "pd-0", Health: true},
					"pd-1": {Name: "pd-1", Health: true},
					"pd-2": {Name: "pd-2", Health: true},
				}
				cluster.Status.PD.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 3}
				cluster.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"tikv-0": {PodName: "tikv-0", State: v1alpha1.TiKVStateUp},
					"tikv-1": {PodName: "tikv-1", State: v1alpha1.TiKVStateUp},
					"tikv-2": {PodName: "tikv-2", State: v1alpha1.TiKVStateUp},
				}
				cluster.Status.TiKV.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 3}
			},
			syncReclaimPolicyErr:     false,
			syncPDMemberManagerErr:   false,
			syncTiKVMemberManagerErr: false,
			syncTiDBMemberManagerErr: false,
			syncMetaManagerErr:       true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "meta manager sync error")).To(Equal(true))
			},
		},
		{
			name: "normal",
			update: func(cluster *v1alpha1.TidbCluster) {
				cluster.Status.PD.Members = map[string]v1alpha1.PDMember{
					"pd-0": {Name: "pd-0", Health: true},
					"pd-1": {Name: "pd-1", Health: true},
					"pd-2": {Name: "pd-2", Health: true},
				}
				cluster.Status.PD.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 3}
				cluster.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"tikv-0": {PodName: "tikv-0", State: v1alpha1.TiKVStateUp},
					"tikv-1": {PodName: "tikv-1", State: v1alpha1.TiKVStateUp},
					"tikv-2": {PodName: "tikv-2", State: v1alpha1.TiKVStateUp},
				}
				cluster.Status.TiKV.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 3}
			},
			syncReclaimPolicyErr:     false,
			syncPDMemberManagerErr:   false,
			syncTiKVMemberManagerErr: false,
			syncTiDBMemberManagerErr: false,
			syncMetaManagerErr:       false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
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

func newFakeTidbClusterControl() (ControlInterface, *meta.FakeReclaimPolicyManager, *mm.FakePDMemberManager, *mm.FakeTiKVMemberManager, *mm.FakeTiDBMemberManager, *meta.FakeMetaManager) {
	cli := fake.NewSimpleClientset()
	tcInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1alpha1().TidbClusters()
	recorder := record.NewFakeRecorder(10)

	tcControl := controller.NewFakeTidbClusterControl(tcInformer)
	pdMemberManager := mm.NewFakePDMemberManager()
	tikvMemberManager := mm.NewFakeTiKVMemberManager()
	tidbMemberManager := mm.NewFakeTiDBMemberManager()
	reclaimPolicyManager := meta.NewFakeReclaimPolicyManager()
	metaManager := meta.NewFakeMetaManager()
<<<<<<< HEAD
	opc := mm.NewFakeOrphanPodsCleaner()
	pcc := mm.NewFakePVCCleaner()
	control := NewDefaultTidbClusterControl(tcControl, pdMemberManager, tikvMemberManager, tidbMemberManager, reclaimPolicyManager, metaManager, opc, pcc, recorder)
=======
	orphanPodCleaner := mm.NewFakeOrphanPodsCleaner()
	pvcCleaner := mm.NewFakePVCCleaner()
	pumpMemberManager := mm.NewFakePumpMemberManager()
	tiflashMemberManager := mm.NewFakeTiFlashMemberManager()
	ticdcMemberManager := mm.NewFakeTiCDCMemberManager()
	discoveryManager := mm.NewFakeDiscoveryManger()
	statusManager := mm.NewFakeTidbClusterStatusManager()
	pvcResizer := mm.NewFakePVCResizer()
	control := NewDefaultTidbClusterControl(
		tcUpdater,
		pdMemberManager,
		tikvMemberManager,
		tidbMemberManager,
		reclaimPolicyManager,
		metaManager,
		orphanPodCleaner,
		pvcCleaner,
		pvcResizer,
		pumpMemberManager,
		tiflashMemberManager,
		ticdcMemberManager,
		discoveryManager,
		statusManager,
		&tidbClusterConditionUpdater{},
		recorder,
	)
>>>>>>> 18703b9... Remove the PodRestarter controller and `tidb.pingcap.com/pod-defer-deleting` annotation (#3296)

	return control, reclaimPolicyManager, pdMemberManager, tikvMemberManager, tidbMemberManager, metaManager
}

func newTidbClusterForTidbClusterControl() *v1alpha1.TidbCluster {
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
				Replicas: 3,
			},
			TiKV: v1alpha1.TiKVSpec{
				Replicas: 3,
			},
			TiDB: v1alpha1.TiDBSpec{
				Replicas: 1,
			},
		},
	}
}
