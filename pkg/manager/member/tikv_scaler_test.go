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
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
	"k8s.io/apimachinery/pkg/api/errors"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestTiKVScalerScaleOut(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name          string
		tikvUpgrading bool
		hasPVC        bool
		hasDeferAnn   bool
		pvcDeleteErr  bool
		err           bool
		changed       bool
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		tc := newTidbClusterForPD()

		if test.tikvUpgrading {
			tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
		}

		oldSet := newStatefulSetForPDScale()
		newSet := oldSet.DeepCopy()
		newSet.Spec.Replicas = int32Pointer(7)

		scaler, _, pvcIndexer, pvcControl := newFakeTiKVScaler()

		pvc1 := newPVCForStatefulSet(oldSet, v1alpha1.TiKVMemberType)
		pvc2 := pvc1.DeepCopy()
		pvc1.Name = ordinalPVCName(v1alpha1.TiKVMemberType, oldSet.GetName(), *oldSet.Spec.Replicas)
		pvc2.Name = ordinalPVCName(v1alpha1.TiKVMemberType, oldSet.GetName(), *oldSet.Spec.Replicas)
		if test.hasDeferAnn {
			pvc1.Annotations = map[string]string{}
			pvc1.Annotations[label.AnnPVCDeferDeleting] = time.Now().Format(time.RFC3339)
			pvc2.Annotations = map[string]string{}
			pvc2.Annotations[label.AnnPVCDeferDeleting] = time.Now().Format(time.RFC3339)
		}
		if test.hasPVC {
			pvcIndexer.Add(pvc1)
			pvcIndexer.Add(pvc2)
		}

		if test.pvcDeleteErr {
			pvcControl.SetDeletePVCError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err := scaler.ScaleOut(tc, oldSet, newSet)
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
			name:          "normal",
			tikvUpgrading: false,
			hasPVC:        true,
			hasDeferAnn:   false,
			pvcDeleteErr:  false,
			err:           false,
			changed:       true,
		},
		{
			name:          "tikv is upgrading",
			tikvUpgrading: true,
			hasPVC:        true,
			hasDeferAnn:   false,
			pvcDeleteErr:  false,
			err:           false,
			changed:       false,
		},
		{
			name:          "cache don't have pvc",
			tikvUpgrading: false,
			hasPVC:        false,
			hasDeferAnn:   false,
			pvcDeleteErr:  false,
			err:           false,
			changed:       true,
		},
		{
			name:          "pvc annotations defer deletion is not nil, pvc delete failed",
			tikvUpgrading: false,
			hasPVC:        true,
			hasDeferAnn:   true,
			pvcDeleteErr:  true,
			err:           true,
			changed:       false,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestTiKVScalerScaleIn(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name          string
		tikvUpgrading bool
		storeFun      func(tc *v1alpha1.TidbCluster)
		delStoreErr   bool
		hasPVC        bool
		pvcUpdateErr  bool
		err           bool
		changed       bool
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)
		tc := newTidbClusterForPD()
		test.storeFun(tc)

		if test.tikvUpgrading {
			tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
		}

		oldSet := newStatefulSetForPDScale()
		newSet := oldSet.DeepCopy()
		newSet.Spec.Replicas = int32Pointer(3)

		scaler, pdControl, pvcIndexer, pvcControl := newFakeTiKVScaler()

		if test.hasPVC {
			pvc := newPVCForStatefulSet(oldSet, v1alpha1.TiKVMemberType)
			pvcIndexer.Add(pvc)
		}

		pdClient := controller.NewFakePDClient()
		pdControl.SetPDClient(tc, pdClient)

		if test.delStoreErr {
			pdClient.AddReaction(controller.DeleteStoreActionType, func(action *controller.Action) (interface{}, error) {
				return nil, fmt.Errorf("delete store error")
			})
		}
		if test.pvcUpdateErr {
			pvcControl.SetUpdatePVCError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err := scaler.ScaleIn(tc, oldSet, newSet)
		if test.err {
			g.Expect(err).To(HaveOccurred())
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
			name:          "ordinal store is up, delete store success",
			tikvUpgrading: false,
			storeFun:      normalStoreFun,
			delStoreErr:   false,
			hasPVC:        true,
			pvcUpdateErr:  false,
			err:           true,
			changed:       false,
		},
		{
			name:          "tikv is upgrading",
			tikvUpgrading: true,
			storeFun:      normalStoreFun,
			delStoreErr:   false,
			hasPVC:        true,
			pvcUpdateErr:  false,
			err:           false,
			changed:       false,
		},
		{
			name:          "status.TiKV.Stores is empty",
			tikvUpgrading: false,
			storeFun: func(tc *v1alpha1.TidbCluster) {
				normalStoreFun(tc)
				tc.Status.TiKV.Stores = v1alpha1.TiKVStores{}
			},
			delStoreErr:  false,
			hasPVC:       true,
			pvcUpdateErr: false,
			err:          true,
			changed:      false,
		},
		{
			name:          "podName not match",
			tikvUpgrading: false,
			storeFun: func(tc *v1alpha1.TidbCluster) {
				normalStoreFun(tc)
				store := tc.Status.TiKV.Stores.CurrentStores["1"]
				store.PodName = "xxx"
				tc.Status.TiKV.Stores.CurrentStores["1"] = store
			},
			delStoreErr:  false,
			hasPVC:       true,
			pvcUpdateErr: false,
			err:          true,
			changed:      false,
		},
		{
			name:          "store id is not integer",
			tikvUpgrading: false,
			storeFun: func(tc *v1alpha1.TidbCluster) {
				normalStoreFun(tc)
				store := tc.Status.TiKV.Stores.CurrentStores["1"]
				store.ID = "not integer"
				tc.Status.TiKV.Stores.CurrentStores["1"] = store
			},
			delStoreErr:  false,
			hasPVC:       true,
			pvcUpdateErr: false,
			err:          true,
			changed:      false,
		},
		{
			name:          "store state is offline",
			tikvUpgrading: false,
			storeFun: func(tc *v1alpha1.TidbCluster) {
				normalStoreFun(tc)
				store := tc.Status.TiKV.Stores.CurrentStores["1"]
				store.State = util.StoreOfflineState
				tc.Status.TiKV.Stores.CurrentStores["1"] = store
			},
			delStoreErr:  false,
			hasPVC:       true,
			pvcUpdateErr: false,
			err:          true,
			changed:      false,
		},
		{
			name:          "store state is up, delete store success",
			tikvUpgrading: false,
			storeFun:      normalStoreFun,
			delStoreErr:   false,
			hasPVC:        true,
			pvcUpdateErr:  false,
			err:           true,
			changed:       false,
		},
		{
			name:          "store state is tombstone",
			tikvUpgrading: false,
			storeFun:      tombstoneStoreFun,
			delStoreErr:   false,
			hasPVC:        true,
			pvcUpdateErr:  false,
			err:           false,
			changed:       true,
		},
		{
			name:          "store state is tombstone, id is not integer",
			tikvUpgrading: false,
			storeFun: func(tc *v1alpha1.TidbCluster) {
				tombstoneStoreFun(tc)
				store := tc.Status.TiKV.Stores.TombStoneStores["1"]
				store.ID = "not integer"
				tc.Status.TiKV.Stores.TombStoneStores["1"] = store
			},
			delStoreErr:  false,
			hasPVC:       true,
			pvcUpdateErr: false,
			err:          true,
			changed:      false,
		},
		{
			name:          "store state is tombstone, don't have pvc",
			tikvUpgrading: false,
			storeFun:      tombstoneStoreFun,
			delStoreErr:   false,
			hasPVC:        false,
			pvcUpdateErr:  false,
			err:           true,
			changed:       false,
		},
		{
			name:          "store state is tombstone, don't have pvc",
			tikvUpgrading: false,
			storeFun:      tombstoneStoreFun,
			delStoreErr:   false,
			hasPVC:        true,
			pvcUpdateErr:  true,
			err:           true,
			changed:       false,
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func newFakeTiKVScaler() (*tikvScaler, *controller.FakePDControl, cache.Indexer, *controller.FakePVCControl) {
	kubeCli := kubefake.NewSimpleClientset()

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeCli, 0)
	pvcInformer := kubeInformerFactory.Core().V1().PersistentVolumeClaims()
	pdControl := controller.NewFakePDControl()
	pvcControl := controller.NewFakePVCControl(pvcInformer)

	return &tikvScaler{generalScaler{pdControl, pvcInformer.Lister(), pvcControl}},
		pdControl, pvcInformer.Informer().GetIndexer(), pvcControl
}

func normalStoreFun(tc *v1alpha1.TidbCluster) {
	tc.Status.TiKV.Stores = v1alpha1.TiKVStores{
		CurrentStores: map[string]v1alpha1.TiKVStore{
			"1": {
				ID:      "1",
				PodName: ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), 4),
				State:   util.StoreUpState,
			},
		},
	}
}

func tombstoneStoreFun(tc *v1alpha1.TidbCluster) {
	tc.Status.TiKV.Stores = v1alpha1.TiKVStores{
		TombStoneStores: map[string]v1alpha1.TiKVStore{
			"1": {
				ID:      "1",
				PodName: ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), 4),
				State:   util.StoreTombstoneState,
			},
		},
	}
}
