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

package member

import (
	"fmt"
	"testing"
	"time"

	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/dmapi"
	"k8s.io/client-go/tools/cache"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/pointer"
)

func TestWorkerScalerScaleOut(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name             string
		update           func(cluster *v1alpha1.DMCluster)
		hasPVC           bool
		hasDeferAnn      bool
		annoIsNil        bool
		pvcDeleteErr     bool
		statusSyncFailed bool
		err              bool
		changed          bool
	}

	testFn := func(test testcase, t *testing.T) {
		dc := newDMClusterForWorker()
		test.update(dc)

		oldSet := newStatefulSetForDMScale()
		newSet := oldSet.DeepCopy()
		newSet.Spec.Replicas = pointer.Int32Ptr(7)

		scaler, _, pvcIndexer, pvcControl := newFakeWorkerScaler()

		pvc := newPVCForStatefulSet(oldSet, v1alpha1.DMWorkerMemberType, dc.Name)
		pvc.Name = ordinalPVCName(v1alpha1.DMWorkerMemberType, oldSet.GetName(), *oldSet.Spec.Replicas)
		if !test.annoIsNil {
			pvc.Annotations = map[string]string{}
		}

		if test.hasDeferAnn {
			pvc.Annotations = map[string]string{}
			pvc.Annotations[label.AnnPVCDeferDeleting] = time.Now().Format(time.RFC3339)
		}
		if test.hasPVC {
			pvcIndexer.Add(pvc)
		}

		if test.pvcDeleteErr {
			pvcControl.SetDeletePVCError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		dc.Status.Worker.Synced = !test.statusSyncFailed

		err := scaler.ScaleOut(dc, oldSet, newSet)
		if test.err {
			g.Expect(err).To(HaveOccurred())
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}
		if test.changed {
			g.Expect(int(*newSet.Spec.Replicas)).To(Equal(6))
		} else {
			g.Expect(int(*newSet.Spec.Replicas)).To(Equal(5))
		}
	}

	tests := []testcase{
		{
			name:             "normal",
			update:           normalWorkerMember,
			hasPVC:           true,
			hasDeferAnn:      false,
			annoIsNil:        true,
			pvcDeleteErr:     false,
			statusSyncFailed: false,
			err:              false,
			changed:          true,
		},
		{
			name:             "cache don't have pvc",
			update:           normalWorkerMember,
			hasPVC:           false,
			hasDeferAnn:      false,
			annoIsNil:        true,
			pvcDeleteErr:     false,
			statusSyncFailed: false,
			err:              false,
			changed:          true,
		},
		{
			name:             "pvc annotation is not nil but doesn't contain defer deletion annotation",
			update:           normalWorkerMember,
			hasPVC:           true,
			hasDeferAnn:      false,
			annoIsNil:        false,
			pvcDeleteErr:     false,
			statusSyncFailed: false,
			err:              false,
			changed:          true,
		},
		{
			name:             "pvc annotations defer deletion is not nil, pvc delete failed",
			update:           normalWorkerMember,
			hasPVC:           true,
			hasDeferAnn:      true,
			annoIsNil:        false,
			pvcDeleteErr:     true,
			statusSyncFailed: false,
			err:              true,
			changed:          false,
		},
		{
			name:             "dm-worker status sync failed",
			update:           normalWorkerMember,
			hasPVC:           true,
			hasDeferAnn:      false,
			annoIsNil:        true,
			pvcDeleteErr:     false,
			statusSyncFailed: true,
			err:              true,
			changed:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFn(tt, t)
		})
	}
}

func TestWorkerScalerScaleIn(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name             string
		hasPVC           bool
		pvcUpdateErr     bool
		statusSyncFailed bool
		err              bool
		changed          bool
		isLeader         bool
	}

	testFn := func(test testcase, t *testing.T) {
		dc := newDMClusterForWorker()
		oldSet := newStatefulSetForDMScale()
		newSet := oldSet.DeepCopy()
		newSet.Spec.Replicas = pointer.Int32Ptr(3)

		scaler, _, pvcIndexer, pvcControl := newFakeWorkerScaler()

		if test.hasPVC {
			pvc := newScaleInPVCForStatefulSet(oldSet, v1alpha1.DMWorkerMemberType, dc.Name)
			pvcIndexer.Add(pvc)
		}

		if test.pvcUpdateErr {
			pvcControl.SetUpdatePVCError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		dc.Status.Worker.Synced = !test.statusSyncFailed

		err := scaler.ScaleIn(dc, oldSet, newSet)
		if test.err {
			g.Expect(err).To(HaveOccurred())
			if test.isLeader {
				g.Expect(controller.IsRequeueError(err)).To(BeTrue())
			}
		} else {
			g.Expect(err).NotTo(HaveOccurred())
		}
		if test.changed {
			g.Expect(int(*newSet.Spec.Replicas)).To(Equal(4))
		} else {
			g.Expect(int(*newSet.Spec.Replicas)).To(Equal(5))
		}
	}

	tests := []testcase{
		{
			name:             "normal",
			hasPVC:           true,
			pvcUpdateErr:     false,
			statusSyncFailed: false,
			err:              false,
			changed:          true,
		},
		{
			name:             "cache don't have pvc",
			hasPVC:           false,
			pvcUpdateErr:     false,
			statusSyncFailed: false,
			err:              true,
			changed:          false,
		},
		{
			name:             "error when update pvc",
			hasPVC:           true,
			pvcUpdateErr:     true,
			statusSyncFailed: false,
			err:              true,
			changed:          false,
		},
		{
			name:             "dm-worker status sync failed",
			hasPVC:           true,
			pvcUpdateErr:     false,
			statusSyncFailed: true,
			err:              true,
			changed:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFn(tt, t)
		})
	}
}

func newFakeWorkerScaler() (*workerScaler, *dmapi.FakeMasterControl, cache.Indexer, *controller.FakePVCControl) {
	fakeDeps := controller.NewFakeDependencies()
	scaler := &workerScaler{generalScaler{deps: fakeDeps}}
	masterControl := fakeDeps.DMMasterControl.(*dmapi.FakeMasterControl)
	pvcIndexer := fakeDeps.KubeInformerFactory.Core().V1().PersistentVolumeClaims().Informer().GetIndexer()
	pvcControl := fakeDeps.PVCControl.(*controller.FakePVCControl)
	return scaler, masterControl, pvcIndexer, pvcControl
}

func normalWorkerMember(dc *v1alpha1.DMCluster) {
	dcName := dc.GetName()
	dc.Status.Worker.Members = map[string]v1alpha1.WorkerMember{
		ordinalPodName(v1alpha1.DMWorkerMemberType, dcName, 0): {Stage: v1alpha1.DMWorkerStateFree},
		ordinalPodName(v1alpha1.DMWorkerMemberType, dcName, 1): {Stage: v1alpha1.DMWorkerStateFree},
		ordinalPodName(v1alpha1.DMWorkerMemberType, dcName, 2): {Stage: v1alpha1.DMWorkerStateFree},
		ordinalPodName(v1alpha1.DMWorkerMemberType, dcName, 3): {Stage: v1alpha1.DMWorkerStateFree},
		ordinalPodName(v1alpha1.DMWorkerMemberType, dcName, 4): {Stage: v1alpha1.DMWorkerStateFree},
	}
}
