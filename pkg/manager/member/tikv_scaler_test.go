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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
)

func TestTiKVScalerScaleOut(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name          string
		tikvUpgrading bool
		hasPVC        bool
		hasDeferAnn   bool
		pvcDeleteErr  bool
		annoIsNil     bool
		errExpectFn   func(*GomegaWithT, error)
		changed       bool
	}

	testFn := func(test testcase, t *testing.T) {
		tc := newTidbClusterForPD()

		if test.tikvUpgrading {
			tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
		}

		oldSet := newStatefulSetForPDScale()
		oldSet.Name = fmt.Sprintf("%s-tikv", tc.Name)
		newSet := oldSet.DeepCopy()
		newSet.Spec.Replicas = pointer.Int32Ptr(7)

		scaler, _, pvcIndexer, _, pvcControl := newFakeTiKVScaler()

		pvc := newPVCForStatefulSet(oldSet, v1alpha1.TiKVMemberType, tc.Name)
		pvc.Name = ordinalPVCName(v1alpha1.TiKVMemberType, oldSet.GetName(), *oldSet.Spec.Replicas)
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

		err := scaler.ScaleOut(tc, oldSet, newSet)
		test.errExpectFn(g, err)
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
			annoIsNil:     true,
			pvcDeleteErr:  false,
			errExpectFn:   errExpectNotNil,
			changed:       false,
		},
		{
			name:          "tikv is upgrading",
			tikvUpgrading: true,
			hasPVC:        true,
			hasDeferAnn:   false,
			annoIsNil:     true,
			pvcDeleteErr:  false,
			errExpectFn:   errExpectNotNil,
			changed:       false,
		},
		{
			name:          "cache don't have pvc",
			tikvUpgrading: false,
			hasPVC:        false,
			hasDeferAnn:   false,
			annoIsNil:     true,
			pvcDeleteErr:  false,
			errExpectFn:   errExpectNil,
			changed:       true,
		},
		{
			name:          "pvc annotation is not nil but doesn't contain defer deletion annotation",
			tikvUpgrading: false,
			hasPVC:        true,
			hasDeferAnn:   false,
			annoIsNil:     false,
			pvcDeleteErr:  false,
			errExpectFn:   errExpectNotNil,
			changed:       false,
		},
		{
			name:          "pvc annotations defer deletion is not nil, pvc delete failed",
			tikvUpgrading: false,
			hasPVC:        true,
			hasDeferAnn:   true,
			pvcDeleteErr:  true,
			errExpectFn:   errExpectNotNil,
			changed:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFn(tt, t)
		})
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
		storeIDSynced bool
		isPodReady    bool
		hasSynced     bool
		pvcUpdateErr  bool
		errExpectFn   func(*GomegaWithT, error)
		changed       bool
	}

	resyncDuration := time.Duration(0)

	testFn := func(test testcase, t *testing.T) {
		tc := newTidbClusterForPD()
		test.storeFun(tc)

		if test.tikvUpgrading {
			tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
		}

		oldSet := newStatefulSetForPDScale()
		newSet := oldSet.DeepCopy()
		newSet.Spec.Replicas = pointer.Int32Ptr(3)

		pod := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:              TikvPodName(tc.GetName(), 4),
				Namespace:         corev1.NamespaceDefault,
				CreationTimestamp: metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
			},
		}

		readyPodFunc(pod)
		if !test.isPodReady {
			notReadyPodFunc(pod)
		}

		if !test.hasSynced {
			pod.CreationTimestamp = metav1.Time{Time: time.Now().Add(1 * time.Hour)}
		}

		scaler, pdControl, pvcIndexer, podIndexer, pvcControl := newFakeTiKVScaler(resyncDuration)

		if test.hasPVC {
			pvc := newScaleInPVCForStatefulSet(oldSet, v1alpha1.TiKVMemberType, tc.Name)
			pvcIndexer.Add(pvc)
		}

		pod.Labels = map[string]string{}
		if test.storeIDSynced {
			pod.Labels[label.StoreIDLabelKey] = "1"
		}
		podIndexer.Add(pod)

		pdClient := controller.NewFakePDClient(pdControl, tc)

		if test.delStoreErr {
			pdClient.AddReaction(pdapi.DeleteStoreActionType, func(action *pdapi.Action) (interface{}, error) {
				return nil, fmt.Errorf("delete store error")
			})
		}
		if test.pvcUpdateErr {
			pvcControl.SetUpdatePVCError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		err := scaler.ScaleIn(tc, oldSet, newSet)
		test.errExpectFn(g, err)
		if test.changed {
			g.Expect(int(*newSet.Spec.Replicas)).To(Equal(4))
		} else {
			g.Expect(int(*newSet.Spec.Replicas)).To(Equal(5))
		}
	}

	tests := []testcase{
		{
			name:          "store is up, delete store failed",
			tikvUpgrading: false,
			storeFun:      normalStoreFun,
			delStoreErr:   false,
			hasPVC:        true,
			storeIDSynced: true,
			isPodReady:    true,
			hasSynced:     true,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNotNil,
			changed:       false,
		},
		{
			name:          "store state is up, delete store success",
			tikvUpgrading: false,
			storeFun:      normalStoreFun,
			delStoreErr:   false,
			hasPVC:        true,
			storeIDSynced: true,
			isPodReady:    true,
			hasSynced:     true,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectRequeue,
			changed:       false,
		},
		{
			name:          "able to scale in while upgrading",
			tikvUpgrading: true,
			storeFun:      tombstoneStoreFun,
			delStoreErr:   false,
			hasPVC:        true,
			storeIDSynced: true,
			isPodReady:    true,
			hasSynced:     true,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNil,
			changed:       true,
		},
		{
			name:          "status.TiKV.Stores is empty",
			tikvUpgrading: false,
			storeFun: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{}
			},
			delStoreErr:   false,
			hasPVC:        true,
			storeIDSynced: true,
			isPodReady:    true,
			hasSynced:     true,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNotNil,
			changed:       false,
		},
		{
			name:          "tikv pod is not ready now, not sure if the status has been synced",
			tikvUpgrading: false,
			storeFun: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{}
			},
			delStoreErr:   false,
			hasPVC:        true,
			storeIDSynced: true,
			isPodReady:    false,
			hasSynced:     false,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNotNil,
			changed:       false,
		},
		{
			name:          "tikv pod is not ready now, make sure the status has been synced",
			tikvUpgrading: false,
			storeFun: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{}
			},
			delStoreErr:   false,
			hasPVC:        true,
			storeIDSynced: true,
			isPodReady:    false,
			hasSynced:     true,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNil,
			changed:       true,
		},
		{
			name:          "podName not match",
			tikvUpgrading: false,
			storeFun: func(tc *v1alpha1.TidbCluster) {
				normalStoreFun(tc)
				store := tc.Status.TiKV.Stores["1"]
				store.PodName = "xxx"
				tc.Status.TiKV.Stores["1"] = store
			},
			delStoreErr:   false,
			hasPVC:        true,
			storeIDSynced: true,
			isPodReady:    true,
			hasSynced:     true,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNotNil,
			changed:       false,
		},
		{
			name:          "store id is not integer",
			tikvUpgrading: false,
			storeFun: func(tc *v1alpha1.TidbCluster) {
				normalStoreFun(tc)
				store := tc.Status.TiKV.Stores["1"]
				store.ID = "not integer"
				tc.Status.TiKV.Stores["1"] = store
			},
			delStoreErr:   false,
			hasPVC:        true,
			storeIDSynced: true,
			isPodReady:    true,
			hasSynced:     true,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNotNil,
			changed:       false,
		},
		{
			name:          "store state is offline",
			tikvUpgrading: false,
			storeFun: func(tc *v1alpha1.TidbCluster) {
				normalStoreFun(tc)
				store := tc.Status.TiKV.Stores["1"]
				store.State = v1alpha1.TiKVStateOffline
				tc.Status.TiKV.Stores["1"] = store
			},
			delStoreErr:   false,
			hasPVC:        true,
			storeIDSynced: true,
			isPodReady:    true,
			hasSynced:     true,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectRequeue,
			changed:       false,
		},
		{
			name:          "store state is tombstone",
			tikvUpgrading: false,
			storeFun:      tombstoneStoreFun,
			delStoreErr:   false,
			hasPVC:        true,
			storeIDSynced: true,
			isPodReady:    true,
			hasSynced:     true,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNil,
			changed:       true,
		},
		{
			name:          "store state is tombstone and store id not match",
			tikvUpgrading: false,
			storeFun:      normalStoreFun,
			delStoreErr:   false,
			hasPVC:        true,
			storeIDSynced: false,
			isPodReady:    true,
			hasSynced:     true,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNotNil,
			changed:       false,
		},
		{
			name:          "store state is tombstone, id is not integer",
			tikvUpgrading: false,
			storeFun: func(tc *v1alpha1.TidbCluster) {
				tombstoneStoreFun(tc)
				store := tc.Status.TiKV.TombstoneStores["1"]
				store.ID = "not integer"
				tc.Status.TiKV.TombstoneStores["1"] = store
			},
			delStoreErr:   false,
			hasPVC:        true,
			storeIDSynced: true,
			isPodReady:    true,
			hasSynced:     true,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNotNil,
			changed:       false,
		},
		{
			name:          "store state is tombstone, don't have pvc",
			tikvUpgrading: false,
			storeFun:      tombstoneStoreFun,
			delStoreErr:   false,
			hasPVC:        false,
			storeIDSynced: true,
			isPodReady:    true,
			hasSynced:     true,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNotNil,
			changed:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFn(tt, t)
		})
	}
}

func newFakeTiKVScaler(resyncDuration ...time.Duration) (*tikvScaler, *pdapi.FakePDControl, cache.Indexer, cache.Indexer, *controller.FakePVCControl) {
	fakeDeps := controller.NewFakeDependencies()
	if len(resyncDuration) > 0 {
		fakeDeps.CLIConfig.ResyncDuration = resyncDuration[0]
	}
	pvcIndexer := fakeDeps.PVCInformer.Informer().GetIndexer()
	podIndexer := fakeDeps.KubeInformerFactory.Core().V1().Pods().Informer().GetIndexer()
	pdControl := fakeDeps.PDControl.(*pdapi.FakePDControl)
	pvcControl := fakeDeps.PVCControl.(*controller.FakePVCControl)
	return &tikvScaler{generalScaler{deps: fakeDeps}}, pdControl, pvcIndexer, podIndexer, pvcControl
}

func normalStoreFun(tc *v1alpha1.TidbCluster) {
	tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
		"1": {
			ID:      "1",
			PodName: ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), 4),
			State:   v1alpha1.TiKVStateUp,
		},
	}
}

func tombstoneStoreFun(tc *v1alpha1.TidbCluster) {
	tc.Status.TiKV.TombstoneStores = map[string]v1alpha1.TiKVStore{
		"1": {
			ID:      "1",
			PodName: ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), 4),
			State:   v1alpha1.TiKVStateTombstone,
		},
	}
}

func readyPodFunc(pod *corev1.Pod) {
	pod.Status.Conditions = []corev1.PodCondition{
		{
			Type:   corev1.PodReady,
			Status: corev1.ConditionTrue,
		},
	}
}

func notReadyPodFunc(pod *corev1.Pod) {
	pod.Status.Conditions = []corev1.PodCondition{
		{
			Type:   corev1.PodReady,
			Status: corev1.ConditionFalse,
		},
	}
}

func errExpectRequeue(g *GomegaWithT, err error) {
	g.Expect(controller.IsRequeueError(err)).To(Equal(true))
}
