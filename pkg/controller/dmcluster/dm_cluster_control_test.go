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

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
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
		name                       string
		update                     func(cluster *v1alpha1.DMCluster)
		syncReclaimPolicyErr       bool
		orphanPodCleanerErr        bool
		syncMasterMemberManagerErr bool
		syncWorkerMemberManagerErr bool
		pvcCleanerErr              bool
		updateDCStatusErr          bool
		errExpectFn                func(*GomegaWithT, error)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		dc := newDMClusterForDMClusterControl()
		if test.update != nil {
			test.update(dc)
		}
		control, reclaimPolicyManager, orphanPodCleaner, masterMemberManager, workerMemberManager, pvcCleaner, dcControl := newFakeDMClusterControl()

		if test.syncReclaimPolicyErr {
			reclaimPolicyManager.SetSyncError(fmt.Errorf("reclaim policy sync error"))
		}
		if test.orphanPodCleanerErr {
			orphanPodCleaner.SetnOrphanPodCleanerError(fmt.Errorf("clean orphan pod error"))
		}
		if test.syncMasterMemberManagerErr {
			masterMemberManager.SetSyncError(fmt.Errorf("dm-master member manager sync error"))
		}
		if test.syncWorkerMemberManagerErr {
			workerMemberManager.SetSyncError(fmt.Errorf("dm-worker member manager sync error"))
		}
		if test.pvcCleanerErr {
			pvcCleaner.SetPVCCleanerError(fmt.Errorf("clean PVC error"))
		}

		if test.updateDCStatusErr {
			dcControl.SetUpdateDMClusterError(fmt.Errorf("update dmcluster status error"), 0)
		}

		err := control.UpdateDMCluster(dc)
		if test.errExpectFn != nil {
			test.errExpectFn(g, err)
		}
	}
	tests := []testcase{
		{
			name:                       "reclaim policy sync error",
			update:                     nil,
			syncReclaimPolicyErr:       true,
			orphanPodCleanerErr:        false,
			syncMasterMemberManagerErr: false,
			syncWorkerMemberManagerErr: false,
			pvcCleanerErr:              false,
			updateDCStatusErr:          false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "reclaim policy sync error")).To(Equal(true))
			},
		},
		{
			name:                       "clean orphan pod error",
			update:                     nil,
			syncReclaimPolicyErr:       false,
			orphanPodCleanerErr:        true,
			syncMasterMemberManagerErr: false,
			syncWorkerMemberManagerErr: false,
			pvcCleanerErr:              false,
			updateDCStatusErr:          false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "clean orphan pod error")).To(Equal(true))
			},
		},
		{
			name:                       "dm-master member manager sync error",
			update:                     nil,
			syncReclaimPolicyErr:       false,
			orphanPodCleanerErr:        false,
			syncMasterMemberManagerErr: true,
			syncWorkerMemberManagerErr: false,
			pvcCleanerErr:              false,
			updateDCStatusErr:          false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "dm-master member manager sync error")).To(Equal(true))
			},
		},
		{
			name:                       "dm-worker member manager sync error",
			update:                     nil,
			syncReclaimPolicyErr:       false,
			orphanPodCleanerErr:        false,
			syncMasterMemberManagerErr: false,
			syncWorkerMemberManagerErr: true,
			pvcCleanerErr:              false,
			updateDCStatusErr:          false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "dm-worker member manager sync error")).To(Equal(true))
			},
		},
		{
			name:                       "clean PVC error",
			update:                     nil,
			syncReclaimPolicyErr:       false,
			orphanPodCleanerErr:        false,
			syncMasterMemberManagerErr: false,
			syncWorkerMemberManagerErr: false,
			pvcCleanerErr:              true,
			updateDCStatusErr:          false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "clean PVC error")).To(Equal(true))
			},
		},
		{
			name:                       "dmcluster status is not updated",
			update:                     nil,
			syncReclaimPolicyErr:       false,
			orphanPodCleanerErr:        false,
			syncMasterMemberManagerErr: false,
			syncWorkerMemberManagerErr: false,
			pvcCleanerErr:              false,
			updateDCStatusErr:          false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name: "dmcluster status update failed",
			update: func(cluster *v1alpha1.DMCluster) {
				cluster.Status.Master.Members = map[string]v1alpha1.MasterMember{
					"dm-master-0": {Name: "dm-master-0", Health: true},
					"dm-master-1": {Name: "dm-master-1", Health: true},
					"dm-master-2": {Name: "dm-master-2", Health: true},
				}
				cluster.Status.Master.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 3}
				cluster.Status.Worker.Members = map[string]v1alpha1.WorkerMember{
					"dm-worker-0": {Name: "dm-worker-0", Stage: v1alpha1.DMWorkerStateFree},
					"dm-worker-1": {Name: "dm-worker-1", Stage: v1alpha1.DMWorkerStateFree},
					"dm-worker-2": {Name: "dm-worker-2", Stage: v1alpha1.DMWorkerStateFree},
				}
				cluster.Status.Worker.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 3}
			},
			syncReclaimPolicyErr:       false,
			orphanPodCleanerErr:        false,
			syncMasterMemberManagerErr: false,
			syncWorkerMemberManagerErr: false,
			pvcCleanerErr:              false,
			updateDCStatusErr:          true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "update dmcluster status error")).To(Equal(true))
			},
		},
		{
			name: "normal",
			update: func(cluster *v1alpha1.DMCluster) {
				cluster.Status.Master.Members = map[string]v1alpha1.MasterMember{
					"dm-master-0": {Name: "dm-master-0", Health: true},
					"dm-master-1": {Name: "dm-master-1", Health: true},
					"dm-master-2": {Name: "dm-master-2", Health: true},
				}
				cluster.Status.Master.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 3}
				cluster.Status.Worker.Members = map[string]v1alpha1.WorkerMember{
					"dm-worker-0": {Name: "dm-worker-0", Stage: v1alpha1.DMWorkerStateFree},
					"dm-worker-1": {Name: "dm-worker-1", Stage: v1alpha1.DMWorkerStateFree},
					"dm-worker-2": {Name: "dm-worker-2", Stage: v1alpha1.DMWorkerStateFree},
				}
				cluster.Status.Worker.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 3}
			},
			syncReclaimPolicyErr:       false,
			orphanPodCleanerErr:        false,
			syncMasterMemberManagerErr: false,
			syncWorkerMemberManagerErr: false,
			updateDCStatusErr:          false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestDMClusterStatusEquality(t *testing.T) {
	g := NewGomegaWithT(t)
	dcStatus := v1alpha1.DMClusterStatus{}

	tcStatusCopy := dcStatus.DeepCopy()
	tcStatusCopy.Master = v1alpha1.MasterStatus{}
	g.Expect(apiequality.Semantic.DeepEqual(&dcStatus, tcStatusCopy)).To(Equal(true))

	tcStatusCopy = dcStatus.DeepCopy()
	tcStatusCopy.Master.Phase = v1alpha1.NormalPhase
	g.Expect(apiequality.Semantic.DeepEqual(&dcStatus, tcStatusCopy)).To(Equal(false))
}

func newFakeDMClusterControl() (
	ControlInterface,
	*meta.FakeReclaimPolicyManager,
	*mm.FakeOrphanPodsCleaner,
	*mm.FakeMasterMemberManager,
	*mm.FakeWorkerMemberManager,
	*mm.FakePVCCleaner,
	*controller.FakeDMClusterControl) {
	cli := fake.NewSimpleClientset()
	dcInformer := informers.NewSharedInformerFactory(cli, 0).Pingcap().V1alpha1().DMClusters()
	recorder := record.NewFakeRecorder(10)

	dcControl := controller.NewFakeDMClusterControl(dcInformer)
	masterMemberManager := mm.NewFakeMasterMemberManager()
	workerMemberManager := mm.NewFakeWorkerMemberManager()
	reclaimPolicyManager := meta.NewFakeReclaimPolicyManager()
	orphanPodCleaner := mm.NewFakeOrphanPodsCleaner()
	pvcCleaner := mm.NewFakePVCCleaner()
	podRestarter := mm.NewFakePodRestarter()
	pvcResizer := mm.NewFakePVCResizer()
	control := NewDefaultDMClusterControl(
		dcControl,
		masterMemberManager,
		workerMemberManager,
		reclaimPolicyManager,
		orphanPodCleaner,
		pvcCleaner,
		pvcResizer,
		podRestarter,
		&dmClusterConditionUpdater{},
		recorder,
	)

	return control, reclaimPolicyManager, orphanPodCleaner, masterMemberManager, workerMemberManager, pvcCleaner, dcControl
}

func newDMClusterForDMClusterControl() *v1alpha1.DMCluster {
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
