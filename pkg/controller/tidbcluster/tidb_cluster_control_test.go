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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	mm "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/manager/meta"
	"github.com/pingcap/tidb-operator/pkg/manager/suspender"
	"github.com/pingcap/tidb-operator/pkg/manager/volumes"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
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
		orphanPodCleanerErr      bool
		syncPDMemberManagerErr   bool
		syncTiKVMemberManagerErr bool
		syncTiDBMemberManagerErr bool
		syncMetaManagerErr       bool
		pvcCleanerErr            bool
		updateTCStatusErr        bool
		errExpectFn              func(*GomegaWithT, error)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTidbClusterForTidbClusterControl()
		if test.update != nil {
			test.update(tc)
		}
		control, reclaimPolicyManager, orphanPodCleaner, pdMemberManager, tikvMemberManager, tidbMemberManager, metaManager, pvcCleaner, tcUpdater := newFakeTidbClusterControl()

		if test.syncReclaimPolicyErr {
			reclaimPolicyManager.SetSyncError(fmt.Errorf("reclaim policy sync error"))
		}
		if test.orphanPodCleanerErr {
			orphanPodCleaner.SetnOrphanPodCleanerError(fmt.Errorf("clean orphan pod error"))
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
		if test.pvcCleanerErr {
			pvcCleaner.SetPVCCleanerError(fmt.Errorf("clean PVC error"))
		}

		if test.updateTCStatusErr {
			tcUpdater.SetUpdateTidbClusterError(fmt.Errorf("update tidbcluster status error"), 0)
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
			orphanPodCleanerErr:      false,
			syncPDMemberManagerErr:   false,
			syncTiKVMemberManagerErr: false,
			syncTiDBMemberManagerErr: false,
			syncMetaManagerErr:       false,
			pvcCleanerErr:            false,
			updateTCStatusErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "reclaim policy sync error")).To(Equal(true))
			},
		},
		{
			name:                     "clean orphan pod error",
			update:                   nil,
			syncReclaimPolicyErr:     false,
			orphanPodCleanerErr:      true,
			syncPDMemberManagerErr:   false,
			syncTiKVMemberManagerErr: false,
			syncTiDBMemberManagerErr: false,
			syncMetaManagerErr:       false,
			pvcCleanerErr:            false,
			updateTCStatusErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "clean orphan pod error")).To(Equal(true))
			},
		},
		{
			name:                     "pd member manager sync error",
			update:                   nil,
			syncReclaimPolicyErr:     false,
			orphanPodCleanerErr:      false,
			syncPDMemberManagerErr:   true,
			syncTiKVMemberManagerErr: false,
			syncTiDBMemberManagerErr: false,
			syncMetaManagerErr:       false,
			pvcCleanerErr:            false,
			updateTCStatusErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "pd member manager sync error")).To(Equal(true))
			},
		},
		{
			name:                     "tikv member manager sync error",
			update:                   nil,
			syncReclaimPolicyErr:     false,
			orphanPodCleanerErr:      false,
			syncPDMemberManagerErr:   false,
			syncTiKVMemberManagerErr: true,
			syncTiDBMemberManagerErr: false,
			syncMetaManagerErr:       false,
			pvcCleanerErr:            false,
			updateTCStatusErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "tikv member manager sync error")).To(Equal(true))
			},
		},
		{
			name:                     "tidb member manager sync error",
			update:                   nil,
			syncReclaimPolicyErr:     false,
			orphanPodCleanerErr:      false,
			syncPDMemberManagerErr:   false,
			syncTiKVMemberManagerErr: false,
			syncTiDBMemberManagerErr: true,
			syncMetaManagerErr:       false,
			pvcCleanerErr:            false,
			updateTCStatusErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "tidb member manager sync error")).To(Equal(true))
			},
		},
		{
			name:                     "meta manager sync error",
			update:                   nil,
			syncReclaimPolicyErr:     false,
			orphanPodCleanerErr:      false,
			syncPDMemberManagerErr:   false,
			syncTiKVMemberManagerErr: false,
			syncTiDBMemberManagerErr: false,
			syncMetaManagerErr:       true,
			pvcCleanerErr:            false,
			updateTCStatusErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "meta manager sync error")).To(Equal(true))
			},
		},
		{
			name:                     "clean PVC error",
			update:                   nil,
			syncReclaimPolicyErr:     false,
			orphanPodCleanerErr:      false,
			syncPDMemberManagerErr:   false,
			syncTiKVMemberManagerErr: false,
			syncTiDBMemberManagerErr: false,
			syncMetaManagerErr:       false,
			pvcCleanerErr:            true,
			updateTCStatusErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "clean PVC error")).To(Equal(true))
			},
		},
		{
			name:                     "tidbcluster status is not updated",
			update:                   nil,
			syncReclaimPolicyErr:     false,
			orphanPodCleanerErr:      false,
			syncPDMemberManagerErr:   false,
			syncTiKVMemberManagerErr: false,
			syncTiDBMemberManagerErr: false,
			syncMetaManagerErr:       false,
			pvcCleanerErr:            false,
			updateTCStatusErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name: "tidbcluster status update failed",
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
			orphanPodCleanerErr:      false,
			syncPDMemberManagerErr:   false,
			syncTiKVMemberManagerErr: false,
			syncTiDBMemberManagerErr: false,
			syncMetaManagerErr:       false,
			pvcCleanerErr:            false,
			updateTCStatusErr:        true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "update tidbcluster status error")).To(Equal(true))
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
			orphanPodCleanerErr:      false,
			syncPDMemberManagerErr:   false,
			syncTiKVMemberManagerErr: false,
			syncTiDBMemberManagerErr: false,
			syncMetaManagerErr:       false,
			updateTCStatusErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestTidbClusterControlBootstrapsTiKVWithTiFlash(t *testing.T) {
	g := NewGomegaWithT(t)
	tc := newTidbClusterForTidbClusterControl()
	tc.Spec.Version = "v8.5.0"
	tc.Spec.PD.Mode = "ms"
	pdImage := "pingcap/pd"
	tc.Spec.PDMS = []*v1alpha1.PDMSSpec{
		{Name: "tso", Replicas: 1, BaseImage: &pdImage},
		{Name: "scheduling", Replicas: 1, BaseImage: &pdImage},
	}
	tc.Spec.TiFlash = &v1alpha1.TiFlashSpec{
		Replicas: 1,
		StorageClaims: []v1alpha1.StorageClaim{{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("10Gi")},
			},
		}},
	}
	tc.Status.PD.Members = map[string]v1alpha1.PDMember{
		"pd-0": {Name: "pd-0", Health: true},
		"pd-1": {Name: "pd-1", Health: true},
		"pd-2": {Name: "pd-2", Health: true},
	}
	tc.Status.PD.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 3}

	deps := controller.NewFakeDependencies()
	pdControl := deps.PDControl.(*pdapi.FakePDControl)
	pdClient := controller.NewFakePDClient(pdControl, tc)
	pdClient.AddReaction(pdapi.GetConfigActionType, func(*pdapi.Action) (interface{}, error) {
		return &pdapi.PDConfigFromAPI{Replication: &pdapi.PDReplicationConfig{}}, nil
	})
	pdClient.AddReaction(pdapi.GetStoresActionType, func(*pdapi.Action) (interface{}, error) {
		return nil, pdapi.TiKVNotBootstrappedErrorf("TiKV cluster not bootstrapped")
	})
	pdMSReady := false
	for _, service := range tc.Spec.PDMS {
		client := controller.NewFakePDMSClient(pdControl, tc, service.Name)
		client.AddReaction(pdapi.GetHealthActionType, func(*pdapi.Action) (interface{}, error) {
			if !pdMSReady {
				return nil, fmt.Errorf("PD microservice not ready")
			}
			return nil, nil
		})
	}

	control, _, _, _, _, _, _, _, _ := newFakeTidbClusterControl()
	c := control.(*defaultTidbClusterControl)
	sp := suspender.NewFakeSuspender()
	pvm := &volumes.FakePodVolumeModifier{}
	c.tiflashMemberManager = mm.NewTiFlashMemberManager(deps,
		mm.NewFakeTiFlashFailover(), mm.NewFakeTiFlashScaler(), mm.NewFakeTiFlashUpgrader(), sp, pvm)
	c.tikvMemberManager = mm.NewTiKVMemberManager(deps,
		mm.NewFakeTiKVFailover(), mm.NewFakeTiKVScaler(), mm.NewFakeTiKVUpgrader(), sp, pvm)
	sets := deps.StatefulSetLister.StatefulSets(tc.Namespace)
	tiKVName := controller.TiKVMemberName(tc.Name)

	g.Expect(control.UpdateTidbCluster(tc)).To(MatchError(ContainSubstring("waiting for PD microservice cluster running")))
	_, err := sets.Get(controller.TiFlashMemberName(tc.Name))
	g.Expect(err).NotTo(HaveOccurred())
	_, err = sets.Get(tiKVName)
	g.Expect(apierrors.IsNotFound(err)).To(BeTrue(), "expected TiKV StatefulSet to be absent, got %v", err)

	pdMSReady = true
	g.Expect(control.UpdateTidbCluster(tc)).To(Succeed())
	set, err := sets.Get(tiKVName)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(*set.Spec.Replicas).To(Equal(tc.Spec.TiKV.Replicas))
	g.Expect(tc.Status.TiFlash.Synced).To(BeFalse())
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

func newFakeTidbClusterControl() (
	ControlInterface,
	*meta.FakeReclaimPolicyManager,
	*mm.FakeOrphanPodsCleaner,
	*mm.FakePDMemberManager,
	*mm.FakeTiKVMemberManager,
	*mm.FakeTiDBMemberManager,
	*meta.FakeMetaManager,
	*mm.FakePVCCleaner,
	*controller.FakeTidbClusterControl) {
	cli := fake.NewSimpleClientset()
	tcInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1alpha1().TidbClusters()
	recorder := record.NewFakeRecorder(10)

	tcUpdater := controller.NewFakeTidbClusterControl(tcInformer)
	pdMemberManager := mm.NewFakePDMemberManager()
	pdMSMemberManager := mm.NewFakePDMSMemberManager()
	tikvMemberManager := mm.NewFakeTiKVMemberManager()
	tidbMemberManager := mm.NewFakeTiDBMemberManager()
	reclaimPolicyManager := meta.NewFakeReclaimPolicyManager()
	metaManager := meta.NewFakeMetaManager()
	orphanPodCleaner := mm.NewFakeOrphanPodsCleaner()
	pvcCleaner := mm.NewFakePVCCleaner()
	pumpMemberManager := mm.NewFakePumpMemberManager()
	tiflashMemberManager := mm.NewFakeTiFlashMemberManager()
	tiproxyMemberManager := mm.NewFakeTiProxyMemberManager()
	ticdcMemberManager := mm.NewFakeTiCDCMemberManager()
	ticiMemberManager := mm.NewFakeTiCIMemberManager()
	discoveryManager := mm.NewFakeDiscoveryManger()
	statusManager := mm.NewFakeTidbClusterStatusManager()
	pvcResizer := mm.NewFakePVCResizer()
	pvcReplacer := volumes.NewFakePVCReplacer()
	control := NewDefaultTidbClusterControl(
		tcUpdater,
		pdMemberManager,
		pdMSMemberManager,
		tikvMemberManager,
		tidbMemberManager,
		tiproxyMemberManager,
		reclaimPolicyManager,
		metaManager,
		orphanPodCleaner,
		pvcCleaner,
		pvcResizer,
		pvcReplacer,
		pumpMemberManager,
		tiflashMemberManager,
		ticdcMemberManager,
		ticiMemberManager,
		discoveryManager,
		statusManager,
		&tidbClusterConditionUpdater{},
		recorder,
	)

	return control, reclaimPolicyManager, orphanPodCleaner, pdMemberManager, tikvMemberManager, tidbMemberManager, metaManager, pvcCleaner, tcUpdater
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
			Version: "v3.0.8",
			PD: &v1alpha1.PDSpec{
				Replicas:  3,
				BaseImage: "pingcap/pd",
				Config:    v1alpha1.NewPDConfig(),
				ResourceRequirements: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("10G"),
					},
				},
			},
			TiKV: &v1alpha1.TiKVSpec{
				Replicas:  3,
				BaseImage: "pingcap/tikv",
				Config:    v1alpha1.NewTiKVConfig(),
				ResourceRequirements: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("10G"),
					},
				},
			},
			TiDB: &v1alpha1.TiDBSpec{
				Replicas:  2,
				BaseImage: "pingcap/tidb",
				Config:    v1alpha1.NewTiDBConfig(),
			},
		},
	}
}
