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

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	perrors "github.com/pingcap/errors"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
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
		tc.Status.TiKV.BootStrapped = true

		oldSet := newStatefulSetForPDScale()
		oldSet.Name = fmt.Sprintf("%s-tikv", tc.Name)
		newSet := oldSet.DeepCopy()
		newSet.Spec.Replicas = pointer.Int32Ptr(7)

		scaler, _, pvcIndexer, _, pvcControl := newFakeTiKVScaler()

		pvc := newPVCForStatefulSet(oldSet, v1alpha1.TiKVMemberType, tc.Name)
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
			errExpectFn:   errExpectRequeue,
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
		{
			name:          "pvc annotations defer deletion is not nil, pvc delete successfully",
			tikvUpgrading: false,
			hasPVC:        true,
			hasDeferAnn:   true,
			pvcDeleteErr:  false,
			errExpectFn:   errExpectNotNil, // tikv will wait a round more.
			changed:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFn(tt, t)
		})
	}
}

func TestTiKVScalerScaleOutSimultaneously(t *testing.T) {
	g := NewGomegaWithT(t)
	type testcase struct {
		name                string
		tikvUpgrading       bool
		hasPVC              bool
		hasDeferAnn         bool
		pvcDeleteErr        bool
		annoIsNil           bool
		scaleOutParallelism int32
		errExpectFn         func(*GomegaWithT, error)
		newReplicas         int
	}

	testFn := func(test testcase, t *testing.T) {
		tc := newTidbClusterForPD()

		if test.tikvUpgrading {
			tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
		}
		tc.Status.TiKV.BootStrapped = true
		tc.Spec.TiKV.ScalePolicy = v1alpha1.ScalePolicy{
			ScaleOutParallelism: pointer.Int32Ptr(test.scaleOutParallelism),
		}

		oldSet := newStatefulSetForPDScale()
		oldSet.Name = fmt.Sprintf("%s-tikv", tc.Name)
		newSet := oldSet.DeepCopy()
		newSet.Spec.Replicas = pointer.Int32Ptr(7)

		scaler, _, pvcIndexer, _, pvcControl := newFakeTiKVScaler()

		pvc := newPVCForStatefulSet(oldSet, v1alpha1.TiKVMemberType, tc.Name)
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
		g.Expect(int(*newSet.Spec.Replicas)).To(Equal(test.newReplicas))
	}

	tests := []testcase{
		{
			name:                "normal",
			tikvUpgrading:       false,
			hasPVC:              true,
			hasDeferAnn:         false,
			annoIsNil:           true,
			pvcDeleteErr:        false,
			scaleOutParallelism: 1,
			errExpectFn:         errExpectRequeue,
			newReplicas:         5,
		},
		{
			name:                "tikv is upgrading",
			tikvUpgrading:       true,
			hasPVC:              true,
			hasDeferAnn:         false,
			annoIsNil:           true,
			pvcDeleteErr:        false,
			scaleOutParallelism: 1,
			errExpectFn:         errExpectNotNil,
			newReplicas:         5,
		},
		{
			name:                "cache don't have pvc",
			tikvUpgrading:       false,
			hasPVC:              false,
			hasDeferAnn:         false,
			annoIsNil:           true,
			pvcDeleteErr:        false,
			scaleOutParallelism: 1,
			errExpectFn:         errExpectNil,
			newReplicas:         6,
		},
		{
			name:                "pvc annotation is not nil but doesn't contain defer deletion annotation",
			tikvUpgrading:       false,
			hasPVC:              true,
			hasDeferAnn:         false,
			annoIsNil:           false,
			pvcDeleteErr:        false,
			scaleOutParallelism: 1,
			errExpectFn:         errExpectNotNil,
			newReplicas:         5,
		},
		{
			name:                "pvc annotations defer deletion is not nil, pvc delete failed",
			tikvUpgrading:       false,
			hasPVC:              true,
			hasDeferAnn:         true,
			pvcDeleteErr:        true,
			scaleOutParallelism: 1,
			errExpectFn:         errExpectNotNil,
			newReplicas:         5,
		},
		{
			name:                "pvc annotations defer deletion is not nil, pvc delete successfully",
			tikvUpgrading:       false,
			hasPVC:              true,
			hasDeferAnn:         true,
			pvcDeleteErr:        false,
			scaleOutParallelism: 1,
			errExpectFn:         errExpectNotNil, // tikv will wait a round more.
			newReplicas:         5,
		},
		{
			name:                "scaleOutParallelism 2 cache don't have pvc",
			tikvUpgrading:       false,
			hasPVC:              false,
			hasDeferAnn:         false,
			annoIsNil:           true,
			pvcDeleteErr:        false,
			scaleOutParallelism: 2,
			errExpectFn:         errExpectNil,
			newReplicas:         7,
		},
		{
			name:                "scaleOutParallelism 3 cache don't have pvc",
			tikvUpgrading:       false,
			hasPVC:              false,
			hasDeferAnn:         false,
			annoIsNil:           true,
			pvcDeleteErr:        false,
			scaleOutParallelism: 3,
			errExpectFn:         errExpectNil,
			newReplicas:         7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFn(tt, t)
		})
	}
}

func TestTiKVScalerScaleOutSimultaneouslyExtra(t *testing.T) {
	type scaleOp struct {
		preHandler  func(set *apps.StatefulSet, tc *v1alpha1.TidbCluster, pvcIndexer cache.Indexer, pvcControl *controller.FakePVCControl)
		replicas    int32
		deleteSlots sets.Int32
	}
	type testcase struct {
		name           string
		enableAsts     bool
		oldDeleteSlots sets.Int32
		newDeleteSlots sets.Int32
		ops            []scaleOp
	}

	testFn := func(test testcase, t *testing.T) {
		tc := newTidbClusterForPD()

		tc.Status.TiKV.BootStrapped = true
		tc.Spec.TiKV.ScalePolicy = v1alpha1.ScalePolicy{
			ScaleOutParallelism: pointer.Int32Ptr(2),
		}

		oldSet := newStatefulSetForPDScale()
		oldSet.Name = fmt.Sprintf("%s-tikv", tc.Name)
		helper.SetDeleteSlots(oldSet, test.oldDeleteSlots)
		newSet := oldSet.DeepCopy()
		newSet.Spec.Replicas = pointer.Int32Ptr(7)
		helper.SetDeleteSlots(newSet, test.newDeleteSlots)

		scaler, _, pvcIndexer, _, pvcControl := newFakeTiKVScaler()

		features.DefaultFeatureGate.Set(fmt.Sprintf("AdvancedStatefulSet=%v", test.enableAsts))
		actualSet := oldSet.DeepCopy()
		for _, op := range test.ops {
			if op.preHandler != nil {
				op.preHandler(oldSet, tc, pvcIndexer, pvcControl)
			}
			desiredSet := newSet.DeepCopy()
			_ = scaler.ScaleOut(tc, actualSet, desiredSet)
			if diff := cmp.Diff(op.replicas, *desiredSet.Spec.Replicas); diff != "" {
				t.Errorf("unexpected (-want, +got): %s", diff)
			}
			if diff := cmp.Diff(op.deleteSlots, helper.GetDeleteSlots(desiredSet)); diff != "" {
				t.Errorf("unexpected (-want, +got): %s", diff)
			}
			actualSet = desiredSet.DeepCopy()
		}
	}

	tests := []testcase{
		{
			name:           "scale without deleteSlots disable asts",
			enableAsts:     false,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(),
			ops: []scaleOp{
				{
					replicas: 7,
				},
			},
		}, {
			name:           "scale without deleteSlots endable asts",
			enableAsts:     true,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(),
			ops: []scaleOp{
				{
					replicas: 7,
				},
			},
		}, {
			name:           "scale with deleteSlots endable asts",
			enableAsts:     true,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(5),
			ops: []scaleOp{
				{
					replicas:    7,
					deleteSlots: sets.NewInt32(5),
				},
			},
		}, {
			name:           "scale second error without deleteSlots disable asts",
			enableAsts:     false,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(),
			ops: []scaleOp{
				{
					preHandler: func(set *apps.StatefulSet, tc *v1alpha1.TidbCluster, pvcIndexer cache.Indexer, pvcControl *controller.FakePVCControl) {
						pvcIndexer.Add(newPVCWithDeleteAnnotaion(set, v1alpha1.TiKVMemberType, tc.GetName(), 6))
						pvcControl.SetDeletePVCError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
					},
					replicas:    6,
					deleteSlots: sets.NewInt32(),
				}, {
					preHandler: func(set *apps.StatefulSet, tc *v1alpha1.TidbCluster, pvcIndexer cache.Indexer, pvcControl *controller.FakePVCControl) {
						pvcIndexer.Delete(newPVCWithDeleteAnnotaion(set, v1alpha1.TiKVMemberType, tc.GetName(), 6))
						pvcControl.SetDeletePVCError(nil, 0)
					},
					replicas:    7,
					deleteSlots: sets.NewInt32(),
				},
			},
		}, {
			name:           "scale first error without deleteSlots disable asts",
			enableAsts:     false,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(),
			ops: []scaleOp{
				{
					preHandler: func(set *apps.StatefulSet, tc *v1alpha1.TidbCluster, pvcIndexer cache.Indexer, pvcControl *controller.FakePVCControl) {
						pvcIndexer.Add(newPVCWithDeleteAnnotaion(set, v1alpha1.TiKVMemberType, tc.GetName(), 5))
						pvcControl.SetDeletePVCError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
					},
					replicas:    5,
					deleteSlots: sets.NewInt32(),
				}, {
					preHandler: func(set *apps.StatefulSet, tc *v1alpha1.TidbCluster, pvcIndexer cache.Indexer, pvcControl *controller.FakePVCControl) {
						pvcIndexer.Delete(newPVCWithDeleteAnnotaion(set, v1alpha1.TiKVMemberType, tc.GetName(), 5))
						pvcControl.SetDeletePVCError(nil, 0)
					},
					replicas:    7,
					deleteSlots: sets.NewInt32(),
				},
			},
		}, {
			name:           "scale second error without deleteSlots enable asts",
			enableAsts:     true,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(),
			ops: []scaleOp{
				{
					preHandler: func(set *apps.StatefulSet, tc *v1alpha1.TidbCluster, pvcIndexer cache.Indexer, pvcControl *controller.FakePVCControl) {
						pvcIndexer.Add(newPVCWithDeleteAnnotaion(set, v1alpha1.TiKVMemberType, tc.GetName(), 6))
						pvcControl.SetDeletePVCError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
					},
					replicas:    6,
					deleteSlots: sets.NewInt32(),
				}, {
					preHandler: func(set *apps.StatefulSet, tc *v1alpha1.TidbCluster, pvcIndexer cache.Indexer, pvcControl *controller.FakePVCControl) {
						pvcIndexer.Delete(newPVCWithDeleteAnnotaion(set, v1alpha1.TiKVMemberType, tc.GetName(), 6))
						pvcControl.SetDeletePVCError(nil, 0)
					},
					replicas:    7,
					deleteSlots: sets.NewInt32(),
				},
			},
		}, {
			name:           "scale first error without deleteSlots enable asts",
			enableAsts:     true,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(),
			ops: []scaleOp{
				{
					preHandler: func(set *apps.StatefulSet, tc *v1alpha1.TidbCluster, pvcIndexer cache.Indexer, pvcControl *controller.FakePVCControl) {
						pvcIndexer.Add(newPVCWithDeleteAnnotaion(set, v1alpha1.TiKVMemberType, tc.GetName(), 5))
						pvcControl.SetDeletePVCError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
					},
					replicas:    5,
					deleteSlots: sets.NewInt32(),
				}, {
					preHandler: func(set *apps.StatefulSet, tc *v1alpha1.TidbCluster, pvcIndexer cache.Indexer, pvcControl *controller.FakePVCControl) {
						pvcIndexer.Delete(newPVCWithDeleteAnnotaion(set, v1alpha1.TiKVMemberType, tc.GetName(), 5))
						pvcControl.SetDeletePVCError(nil, 0)
					},
					replicas:    7,
					deleteSlots: sets.NewInt32(),
				},
			},
		}, {
			name:           "scale second error with deleteSlots enable asts",
			enableAsts:     true,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(5, 6),
			ops: []scaleOp{
				{
					preHandler: func(set *apps.StatefulSet, tc *v1alpha1.TidbCluster, pvcIndexer cache.Indexer, pvcControl *controller.FakePVCControl) {
						pvcIndexer.Add(newPVCWithDeleteAnnotaion(set, v1alpha1.TiKVMemberType, tc.GetName(), 8))
						pvcControl.SetDeletePVCError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
					},
					replicas:    6,
					deleteSlots: sets.NewInt32(5, 6),
				}, {
					preHandler: func(set *apps.StatefulSet, tc *v1alpha1.TidbCluster, pvcIndexer cache.Indexer, pvcControl *controller.FakePVCControl) {
						pvcIndexer.Delete(newPVCWithDeleteAnnotaion(set, v1alpha1.TiKVMemberType, tc.GetName(), 8))
						pvcControl.SetDeletePVCError(nil, 0)
					},
					replicas:    7,
					deleteSlots: sets.NewInt32(5, 6),
				},
			},
		}, {
			name:           "scale first error with deleteSlots enable asts",
			enableAsts:     true,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(5, 6),
			ops: []scaleOp{
				{
					preHandler: func(set *apps.StatefulSet, tc *v1alpha1.TidbCluster, pvcIndexer cache.Indexer, pvcControl *controller.FakePVCControl) {
						pvcIndexer.Add(newPVCWithDeleteAnnotaion(set, v1alpha1.TiKVMemberType, tc.GetName(), 7))
						pvcControl.SetDeletePVCError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
					},
					replicas:    5,
					deleteSlots: sets.NewInt32(),
				}, {
					preHandler: func(set *apps.StatefulSet, tc *v1alpha1.TidbCluster, pvcIndexer cache.Indexer, pvcControl *controller.FakePVCControl) {
						pvcIndexer.Delete(newPVCWithDeleteAnnotaion(set, v1alpha1.TiKVMemberType, tc.GetName(), 7))
						pvcControl.SetDeletePVCError(nil, 0)
					},
					replicas:    7,
					deleteSlots: sets.NewInt32(5, 6),
				},
			},
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
		getStoresFn   func(action *pdapi.Action) (interface{}, error)
	}

	resyncDuration := time.Duration(0)

	testFn := func(test testcase, t *testing.T) {
		tc := newTidbClusterForPD()
		test.storeFun(tc)

		if test.tikvUpgrading {
			tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
		}
		tc.Status.TiKV.BootStrapped = true

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
			pvc1 := newScaleInPVCForStatefulSet(oldSet, v1alpha1.TiKVMemberType, tc.Name)
			pvc2 := pvc1.DeepCopy()
			pvc1.Name = pvc1.Name + "-1"
			pvc1.UID = pvc1.UID + "-1"
			pvc2.Name = pvc2.Name + "-2"
			pvc2.UID = pvc2.UID + "-2"
			pvcIndexer.Add(pvc1)
			pvcIndexer.Add(pvc2)
			pod.Spec.Volumes = append(pod.Spec.Volumes,
				corev1.Volume{
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvc1.Name,
						},
					},
				}, corev1.Volume{
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvc2.Name,
						},
					},
				})
		}

		pod.Labels = map[string]string{}
		if test.storeIDSynced {
			pod.Labels[label.StoreIDLabelKey] = "1"
		}
		podIndexer.Add(pod)

		pdClient := controller.NewFakePDClient(pdControl, tc)

		pdClient.AddReaction(pdapi.GetConfigActionType, func(action *pdapi.Action) (interface{}, error) {
			var replicas uint64 = 3
			return &pdapi.PDConfigFromAPI{
				Replication: &pdapi.PDReplicationConfig{
					MaxReplicas: &replicas,
				},
			}, nil
		})

		if test.getStoresFn == nil {
			test.getStoresFn = func(action *pdapi.Action) (interface{}, error) {
				store := &pdapi.StoreInfo{
					Store: &pdapi.MetaStore{
						StateName: v1alpha1.TiKVStateUp,
						Store: &metapb.Store{
							Address: fmt.Sprintf("%s-tikv-0", "basic"),
						},
					},
				}
				return &pdapi.StoresInfo{
					Count:  5,
					Stores: []*pdapi.StoreInfo{store, store, store, store, store},
				}, nil
			}
		}
		pdClient.AddReaction(pdapi.GetStoresActionType, test.getStoresFn)

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
			delStoreErr:   true,
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
			name:          "able to scale in while is upgrading",
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
			storeFun:      notReadyStoreFun,
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
			storeFun:      notReadyStoreFun,
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
			name:          "store state is tombstone, update pvc failed",
			tikvUpgrading: false,
			storeFun:      tombstoneStoreFun,
			delStoreErr:   false,
			hasPVC:        true,
			storeIDSynced: true,
			isPodReady:    true,
			hasSynced:     true,
			pvcUpdateErr:  true,
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
		{
			name:          "minimal up stores, scale in TiKV is not allowed",
			tikvUpgrading: false,
			storeFun:      minimalUpStoreFun,
			delStoreErr:   false,
			hasPVC:        true,
			storeIDSynced: true,
			isPodReady:    true,
			hasSynced:     true,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNotNil,
			changed:       false,
			getStoresFn: func(action *pdapi.Action) (interface{}, error) {
				store := &pdapi.StoreInfo{
					Store: &pdapi.MetaStore{
						StateName: v1alpha1.TiKVStateUp,
						Store: &metapb.Store{
							Address: fmt.Sprintf("%s-tikv-0", "basic"),
						},
					},
				}
				return &pdapi.StoresInfo{
					Count:  3,
					Stores: []*pdapi.StoreInfo{store, store, store},
				}, nil
			},
		},
		{
			name:          "minimal up(3) stores with tiflash store, scale in TiKV is not allowed",
			tikvUpgrading: false,
			storeFun:      minimalUpStoreFun,
			delStoreErr:   false,
			hasPVC:        true,
			storeIDSynced: true,
			isPodReady:    true,
			hasSynced:     true,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNotNil,
			changed:       false,
			getStoresFn: func(action *pdapi.Action) (interface{}, error) {
				store := &pdapi.StoreInfo{
					Store: &pdapi.MetaStore{
						StateName: v1alpha1.TiKVStateUp,
						Store: &metapb.Store{
							Address: fmt.Sprintf("%s-tikv-0", "basic"),
						},
					},
				}
				tiflashstore := &pdapi.StoreInfo{
					Store: &pdapi.MetaStore{
						StateName: v1alpha1.TiKVStateUp,
						Store: &metapb.Store{
							Address: fmt.Sprintf("%s-tiflash-0", "basic"),
							Labels: []*metapb.StoreLabel{
								{
									Key:   "engine",
									Value: "tiflash",
								},
							},
						},
					},
				}
				return &pdapi.StoresInfo{
					Count:  4,
					Stores: []*pdapi.StoreInfo{store, store, store, tiflashstore},
				}, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFn(tt, t)
		})
	}
}

func TestTiKVScalerScaleInSimultaneously(t *testing.T) {
	g := NewGomegaWithT(t)
	type podStatus struct {
		hasPVC        bool
		storeIDSynced bool
		isPodReady    bool
		hasSynced     bool
		ordinal       int
		storeIdLabel  string
	}
	type testcase struct {
		name               string
		tikvUpgrading      bool
		storeFun           func(tc *v1alpha1.TidbCluster)
		delStoreErr        bool
		pvcUpdateErr       bool
		errExpectFn        func(*GomegaWithT, error)
		newReplicas        int
		getStoresFn        func(action *pdapi.Action) (interface{}, error)
		pods               []podStatus
		scaleInParallelism int32
		tikvReplicas       int32
		extraTestFn        func(g *GomegaWithT, tc *v1alpha1.TidbCluster, scaler *tikvScaler, oldSet *apps.StatefulSet, newSet *apps.StatefulSet)
	}

	resyncDuration := time.Duration(0)

	testFn := func(test testcase, t *testing.T) {
		tc := newTidbClusterForPD()
		test.storeFun(tc)
		// set ScaleInParallelism to do scale in simultaneously.
		tc.Spec.TiKV.ScalePolicy = v1alpha1.ScalePolicy{
			ScaleInParallelism: pointer.Int32Ptr(test.scaleInParallelism),
		}
		if test.tikvUpgrading {
			tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
		}
		tc.Status.TiKV.BootStrapped = true
		scaler, pdControl, pvcIndexer, podIndexer, pvcControl := newFakeTiKVScaler(resyncDuration)

		oldSet := newStatefulSetForPDScale()
		if test.tikvReplicas != 0 {
			oldSet.Spec.Replicas = pointer.Int32Ptr(test.tikvReplicas)
		}
		newSet := oldSet.DeepCopy()
		newSet.Spec.Replicas = pointer.Int32Ptr(3)

		createPodFn := func(s podStatus) {
			pod := &corev1.Pod{
				TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{
					Name:              TikvPodName(tc.GetName(), int32(s.ordinal)),
					Namespace:         corev1.NamespaceDefault,
					CreationTimestamp: metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
				},
			}

			readyPodFunc(pod)
			if !s.isPodReady {
				notReadyPodFunc(pod)
			}

			if !s.hasSynced {
				pod.CreationTimestamp = metav1.Time{Time: time.Now().Add(1 * time.Hour)}
			}

			if s.hasPVC {
				pvc1 := _newPVCForStatefulSet(oldSet, v1alpha1.TiKVMemberType, tc.Name, int32(s.ordinal))
				pvc2 := pvc1.DeepCopy()
				pvc1.Name = pvc1.Name + "-1"
				pvc1.UID = pvc1.UID + "-1"
				pvc2.Name = pvc2.Name + "-2"
				pvc2.UID = pvc2.UID + "-2"
				pvcIndexer.Add(pvc1)
				pvcIndexer.Add(pvc2)
				pod.Spec.Volumes = append(pod.Spec.Volumes,
					corev1.Volume{
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: pvc1.Name,
							},
						},
					}, corev1.Volume{
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: pvc2.Name,
							},
						},
					})
			}

			pod.Labels = map[string]string{}
			if s.storeIDSynced {
				pod.Labels[label.StoreIDLabelKey] = s.storeIdLabel
			}
			podIndexer.Add(pod)

		}

		for _, s := range test.pods {
			createPodFn(s)
		}

		pdClient := controller.NewFakePDClient(pdControl, tc)

		pdClient.AddReaction(pdapi.GetConfigActionType, func(action *pdapi.Action) (interface{}, error) {
			var replicas uint64 = 3
			return &pdapi.PDConfigFromAPI{
				Replication: &pdapi.PDReplicationConfig{
					MaxReplicas: &replicas,
				},
			}, nil
		})

		if test.getStoresFn == nil {
			test.getStoresFn = func(action *pdapi.Action) (interface{}, error) {
				store := &pdapi.StoreInfo{
					Store: &pdapi.MetaStore{
						StateName: v1alpha1.TiKVStateUp,
						Store: &metapb.Store{
							Address: fmt.Sprintf("%s-tikv-0", "basic"),
						},
					},
				}
				return &pdapi.StoresInfo{
					Count:  5,
					Stores: []*pdapi.StoreInfo{store, store, store, store, store},
				}, nil
			}
		}
		pdClient.AddReaction(pdapi.GetStoresActionType, test.getStoresFn)

		if test.delStoreErr {
			pdClient.AddReaction(pdapi.DeleteStoreActionType, func(action *pdapi.Action) (interface{}, error) {
				return nil, fmt.Errorf("delete store error")
			})
		} else {
			pdClient.AddReaction(pdapi.DeleteStoreActionType, func(action *pdapi.Action) (interface{}, error) {
				pod := tc.Status.TiKV.Stores[fmt.Sprintf("%v", action.ID)]
				delete(tc.Status.TiKV.Stores, pod.ID)
				pod.State = v1alpha1.TiKVStateTombstone
				if tc.Status.TiKV.TombstoneStores == nil {
					tc.Status.TiKV.TombstoneStores = make(map[string]v1alpha1.TiKVStore)
				}
				tc.Status.TiKV.TombstoneStores[pod.ID] = pod
				return nil, nil
			})
		}
		if test.pvcUpdateErr {
			pvcControl.SetUpdatePVCError(errors.NewInternalError(fmt.Errorf("API server failed")), 0)
		}

		tmpNewSet := newSet.DeepCopy()
		err := scaler.ScaleIn(tc, oldSet, tmpNewSet)
		test.errExpectFn(g, err)
		g.Expect(int(*tmpNewSet.Spec.Replicas)).To(Equal(test.newReplicas))

		if test.extraTestFn != nil {
			test.extraTestFn(g, tc, scaler, tmpNewSet, newSet)
		}
	}

	tests := []testcase{
		{
			name:          "1 scaleInParallelism, store is up, delete store failed",
			tikvUpgrading: false,
			storeFun:      normalStoreFun,
			delStoreErr:   true,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNotNil,
			newReplicas:   5,
			pods: []podStatus{{
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       4,
				storeIdLabel:  "1",
			}, {
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       3,
				storeIdLabel:  "13",
			}},
			scaleInParallelism: 1,
		},
		{
			name:          "1 scaleInParallelism, store is up",
			tikvUpgrading: false,
			storeFun:      normalStoreFun,
			delStoreErr:   false,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectRequeue,
			newReplicas:   5,
			pods: []podStatus{{
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       4,
				storeIdLabel:  "1",
			}, {
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       3,
				storeIdLabel:  "13",
			}},
			scaleInParallelism: 1,
		}, {
			name:          "1 scaleInParallelism, store state is tombstone, update pvc failed",
			tikvUpgrading: false,
			storeFun:      multiTombstoneStoreFun,
			delStoreErr:   false,
			pvcUpdateErr:  true,
			errExpectFn:   errExpectNotNil,
			newReplicas:   5,
			pods: []podStatus{{
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       4,
				storeIdLabel:  "1",
			}, {
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       3,
				storeIdLabel:  "13",
			}},
			scaleInParallelism: 1,
		}, {
			name:          "1 scaleInParallelism, store state is tombstone",
			tikvUpgrading: false,
			storeFun:      multiTombstoneStoreFun,
			delStoreErr:   false,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNil,
			newReplicas:   4,
			pods: []podStatus{{
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       4,
				storeIdLabel:  "1",
			}, {
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       3,
				storeIdLabel:  "13",
			}},
			scaleInParallelism: 1,
		}, {
			name:          "2 scaleInParallelism, store is up, delete store failed",
			tikvUpgrading: false,
			storeFun:      normalStoreFun,
			delStoreErr:   true,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNotNil,
			newReplicas:   5,
			pods: []podStatus{{
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       4,
				storeIdLabel:  "1",
			}, {
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       3,
				storeIdLabel:  "13",
			}},
			scaleInParallelism: 2,
		}, {
			name:          "2 scaleInParallelism, store is up",
			tikvUpgrading: false,
			storeFun:      normalStoreFun,
			delStoreErr:   false,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectAllRequeue,
			newReplicas:   5,
			pods: []podStatus{{
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       4,
				storeIdLabel:  "1",
			}, {
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       3,
				storeIdLabel:  "13",
			}},
			scaleInParallelism: 2,
		}, {
			name:          "2 scaleInParallelism, store state is tombstone",
			tikvUpgrading: false,
			storeFun:      multiTombstoneStoreFun,
			delStoreErr:   false,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNil,
			newReplicas:   3,
			pods: []podStatus{{
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       4,
				storeIdLabel:  "1",
			}, {
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       3,
				storeIdLabel:  "13",
			}},
			scaleInParallelism: 2,
		}, {
			name:          "3 scaleInParallelism, store state is tombstone, scaleInParallelism is bigger than needed",
			tikvUpgrading: false,
			storeFun:      multiTombstoneStoreFun,
			delStoreErr:   false,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNil,
			newReplicas:   3,
			pods: []podStatus{{
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       4,
				storeIdLabel:  "1",
			}, {
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       3,
				storeIdLabel:  "13",
			}},
			scaleInParallelism: 3,
			extraTestFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, scaler *tikvScaler, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) {
				for i := 0; i < 3; i++ {
					podName := ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), int32(i))
					var found bool
					for _, s := range tc.Status.TiKV.Stores {
						found = found || s.PodName == podName
					}
					g.Expect(found).To(Equal(true))
				}
				g.Expect(len(tc.Status.TiKV.Stores)).To(Equal(3))
				for i := 4; i < 6; i++ {
					podName := ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), int32(i))
					var found bool
					for _, s := range tc.Status.TiKV.Stores {
						found = found || s.PodName == podName
					}
					g.Expect(found).To(Equal(false))
				}
				g.Expect(len(tc.Status.TiKV.TombstoneStores)).To(Equal(2))
			},
		}, {
			name:          "2 scaleInParallelism, store state is tombstone, scaleInParallelism is smaller than needed",
			tikvUpgrading: false,
			storeFun: func(tc *v1alpha1.TidbCluster) {
				multiTombstoneStoreFun(tc)
				tc.Status.TiKV.TombstoneStores["14"] = v1alpha1.TiKVStore{
					ID:      "14",
					PodName: ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), 5),
					State:   v1alpha1.TiKVStateUp,
				}
			},
			delStoreErr:  false,
			pvcUpdateErr: false,
			errExpectFn:  errExpectNil,
			newReplicas:  4,
			pods: []podStatus{{
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       5,
				storeIdLabel:  "14",
			}, {
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       4,
				storeIdLabel:  "1",
			}, {
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       3,
				storeIdLabel:  "13",
			}},
			getStoresFn: func(action *pdapi.Action) (interface{}, error) {
				store := &pdapi.StoreInfo{
					Store: &pdapi.MetaStore{
						StateName: v1alpha1.TiKVStateUp,
						Store: &metapb.Store{
							Address: fmt.Sprintf("%s-tikv-0", "basic"),
						},
					},
				}
				return &pdapi.StoresInfo{
					Count:  6,
					Stores: []*pdapi.StoreInfo{store, store, store, store, store, store},
				}, nil
			},
			scaleInParallelism: 2,
			tikvReplicas:       6,
			extraTestFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, scaler *tikvScaler, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) {
				err := scaler.ScaleIn(tc, oldSet, newSet)
				errExpectNil(g, err)
				g.Expect(int(*newSet.Spec.Replicas)).To(Equal(3))
			},
		}, {
			name:          "2 scaleInParallelism, able to scale in simultaneously while is upgrading",
			tikvUpgrading: true,
			storeFun:      multiTombstoneStoreFun,
			delStoreErr:   false,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNil,
			newReplicas:   3,
			pods: []podStatus{{
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       4,
				storeIdLabel:  "1",
			}, {
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       3,
				storeIdLabel:  "13",
			}},
			scaleInParallelism: 2,
		}, {
			name:          "2 maxScaleInReplica, tikv pod is not ready now, not sure if the status has been synced",
			tikvUpgrading: false,
			storeFun: func(tc *v1alpha1.TidbCluster) {
				tombstoneStoreFun(tc)
				delete(tc.Status.TiKV.Stores, "13")
			},
			delStoreErr:  false,
			pvcUpdateErr: false,
			errExpectFn:  errExpectNotNil,
			newReplicas:  4,
			pods: []podStatus{{
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    false,
				hasSynced:     true,
				ordinal:       4,
				storeIdLabel:  "1",
			}, {
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    false,
				hasSynced:     false,
				ordinal:       3,
				storeIdLabel:  "13",
			}},
			scaleInParallelism: 2,
		}, {
			name:          "2 maxScaleInReplica, tikv pod is not ready now, make sure if the status has been synced",
			tikvUpgrading: false,
			storeFun: func(tc *v1alpha1.TidbCluster) {
				tombstoneStoreFun(tc)
				delete(tc.Status.TiKV.Stores, "13")
			},
			delStoreErr:  false,
			pvcUpdateErr: false,
			errExpectFn:  errExpectNil,
			newReplicas:  3,
			pods: []podStatus{{
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    false,
				hasSynced:     true,
				ordinal:       4,
				storeIdLabel:  "1",
			}, {
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    false,
				hasSynced:     true,
				ordinal:       3,
				storeIdLabel:  "13",
			}},
			scaleInParallelism: 2,
		}, {
			name:          "2 maxScaleInReplica, store state is tombstone, don't have pvc",
			tikvUpgrading: false,
			storeFun:      multiTombstoneStoreFun,
			delStoreErr:   false,
			pvcUpdateErr:  false,
			errExpectFn:   errExpectNotNil,
			newReplicas:   4,
			pods: []podStatus{{
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       4,
				storeIdLabel:  "1",
			}, {
				hasPVC:        false,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       3,
				storeIdLabel:  "13",
			}},
			scaleInParallelism: 2,
		}, {
			name:          "2 maxScaleInReplica, 4 up stores, scale in TiKV simultaneously works but only scales one",
			tikvUpgrading: false,
			storeFun: func(tc *v1alpha1.TidbCluster) {
				normalStoreFun(tc)
				tc.Status.TiKV.Stores["12"] = v1alpha1.TiKVStore{
					ID:      "12",
					PodName: ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), 2),
					State:   v1alpha1.TiKVStateDown,
				}
			},
			delStoreErr:  false,
			pvcUpdateErr: false,
			errExpectFn:  errExpectRequeue,
			newReplicas:  5,
			pods: []podStatus{{
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       4,
				storeIdLabel:  "1",
			}, {
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       3,
				storeIdLabel:  "13",
			}},
			getStoresFn: func(action *pdapi.Action) (interface{}, error) {
				store := &pdapi.StoreInfo{
					Store: &pdapi.MetaStore{
						StateName: v1alpha1.TiKVStateUp,
						Store: &metapb.Store{
							Address: fmt.Sprintf("%s-tikv-0", "basic"),
						},
					},
				}
				return &pdapi.StoresInfo{
					Count:  4,
					Stores: []*pdapi.StoreInfo{store, store, store, store},
				}, nil
			},
			scaleInParallelism: 2,
			extraTestFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, scaler *tikvScaler, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) {
				for i := 0; i < 4; i++ {
					podName := ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), int32(i))
					var found bool
					for _, s := range tc.Status.TiKV.Stores {
						found = found || s.PodName == podName
					}
					g.Expect(found).To(Equal(true))
				}
				g.Expect(len(tc.Status.TiKV.Stores)).To(Equal(4))
				for i := 4; i < 5; i++ {
					podName := ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), int32(i))
					var found bool
					for _, s := range tc.Status.TiKV.Stores {
						found = found || s.PodName == podName
					}
					g.Expect(found).To(Equal(false))
				}
				g.Expect(len(tc.Status.TiKV.TombstoneStores)).To(Equal(1))
			},
		}, {
			name:          "2 maxScaleInReplica, 5 up stores with tiflash store, scale in TiKV simultaneously works but only scales one",
			tikvUpgrading: false,
			storeFun: func(tc *v1alpha1.TidbCluster) {
				normalStoreFun(tc)
				tc.Status.TiKV.Stores["12"] = v1alpha1.TiKVStore{
					ID:      "12",
					PodName: ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), 2),
					State:   v1alpha1.TiKVStateDown,
				}
			},
			delStoreErr:  false,
			pvcUpdateErr: false,
			errExpectFn:  errExpectRequeue,
			newReplicas:  5,
			pods: []podStatus{{
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       4,
				storeIdLabel:  "1",
			}, {
				hasPVC:        true,
				storeIDSynced: true,
				isPodReady:    true,
				hasSynced:     true,
				ordinal:       3,
				storeIdLabel:  "13",
			}},
			getStoresFn: func(action *pdapi.Action) (interface{}, error) {
				store := &pdapi.StoreInfo{
					Store: &pdapi.MetaStore{
						StateName: v1alpha1.TiKVStateUp,
						Store: &metapb.Store{
							Address: fmt.Sprintf("%s-tikv-0", "basic"),
						},
					},
				}
				tiflashstore := &pdapi.StoreInfo{
					Store: &pdapi.MetaStore{
						StateName: v1alpha1.TiKVStateUp,
						Store: &metapb.Store{
							Address: fmt.Sprintf("%s-tiflash-0", "basic"),
							Labels: []*metapb.StoreLabel{
								{
									Key:   "engine",
									Value: "tiflash",
								},
							},
						},
					},
				}
				return &pdapi.StoresInfo{
					Count:  5,
					Stores: []*pdapi.StoreInfo{store, store, store, store, tiflashstore},
				}, nil
			},
			scaleInParallelism: 2,
			extraTestFn: func(g *GomegaWithT, tc *v1alpha1.TidbCluster, scaler *tikvScaler, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) {
				for i := 0; i < 4; i++ {
					podName := ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), int32(i))
					var found bool
					for _, s := range tc.Status.TiKV.Stores {
						found = found || s.PodName == podName
					}
					g.Expect(found).To(Equal(true))
				}
				g.Expect(len(tc.Status.TiKV.Stores)).To(Equal(4))
				for i := 4; i < 5; i++ {
					podName := ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), int32(i))
					var found bool
					for _, s := range tc.Status.TiKV.Stores {
						found = found || s.PodName == podName
					}
					g.Expect(found).To(Equal(false))
				}
				g.Expect(len(tc.Status.TiKV.TombstoneStores)).To(Equal(1))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFn(tt, t)
		})
	}
}

func TestTiKVScalerScaleInSimultaneouslyExtra(t *testing.T) {
	type scaleOp struct {
		preHandler  func(tc *v1alpha1.TidbCluster)
		replicas    int32
		deleteSlots sets.Int32
	}
	type testcase struct {
		name           string
		enableAsts     bool
		oldDeleteSlots sets.Int32
		newDeleteSlots sets.Int32
		getStoresFn    func(action *pdapi.Action) (interface{}, error)
		ops            []scaleOp
	}

	resyncDuration := time.Duration(0)

	testFn := func(test testcase, t *testing.T) {
		tc := newTidbCluster()
		allTombstonesStoreFun(tc)
		tc.Spec.TiKV.ScalePolicy = v1alpha1.ScalePolicy{
			ScaleInParallelism: pointer.Int32Ptr(2),
		}
		tc.Status.TiKV.BootStrapped = true
		scaler, pdControl, pvcIndexer, podIndexer, _ := newFakeTiKVScaler(resyncDuration)

		oldSet := newStatefulSetForPDScale()
		oldSet.Spec.Replicas = pointer.Int32Ptr(5)
		helper.SetDeleteSlots(oldSet, test.oldDeleteSlots)
		newSet := oldSet.DeepCopy()
		newSet.Spec.Replicas = pointer.Int32Ptr(3)
		helper.SetDeleteSlots(newSet, test.newDeleteSlots)

		createPodFn := func(ordinal int, storeIdLabel string) {
			pod := &corev1.Pod{
				TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{
					Name:              TikvPodName(tc.GetName(), int32(ordinal)),
					Namespace:         corev1.NamespaceDefault,
					CreationTimestamp: metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
				},
			}
			readyPodFunc(pod)
			pvc1 := _newPVCForStatefulSet(oldSet, v1alpha1.TiKVMemberType, tc.Name, int32(ordinal))
			pvc2 := pvc1.DeepCopy()
			pvc1.Name = pvc1.Name + "-1"
			pvc1.UID = pvc1.UID + "-1"
			pvc2.Name = pvc2.Name + "-2"
			pvc2.UID = pvc2.UID + "-2"
			pvcIndexer.Add(pvc1)
			pvcIndexer.Add(pvc2)
			pod.Spec.Volumes = append(pod.Spec.Volumes,
				corev1.Volume{
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvc1.Name,
						},
					},
				}, corev1.Volume{
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvc2.Name,
						},
					},
				})

			pod.Labels = map[string]string{}
			pod.Labels[label.StoreIDLabelKey] = storeIdLabel
			podIndexer.Add(pod)
		}

		createPodFn(0, "10")
		createPodFn(1, "11")
		createPodFn(2, "12")
		createPodFn(3, "13")
		createPodFn(4, "1")

		pdClient := controller.NewFakePDClient(pdControl, tc)
		pdClient.AddReaction(pdapi.GetConfigActionType, func(action *pdapi.Action) (interface{}, error) {
			var replicas uint64 = 3
			return &pdapi.PDConfigFromAPI{
				Replication: &pdapi.PDReplicationConfig{
					MaxReplicas: &replicas,
				},
			}, nil
		})
		if test.getStoresFn == nil {
			test.getStoresFn = func(action *pdapi.Action) (interface{}, error) {
				store := &pdapi.StoreInfo{
					Store: &pdapi.MetaStore{
						StateName: v1alpha1.TiKVStateUp,
						Store: &metapb.Store{
							Address: fmt.Sprintf("%s-tikv-0", "basic"),
						},
					},
				}
				return &pdapi.StoresInfo{
					Count:  5,
					Stores: []*pdapi.StoreInfo{store, store, store, store, store},
				}, nil
			}
		}
		pdClient.AddReaction(pdapi.GetStoresActionType, test.getStoresFn)

		features.DefaultFeatureGate.Set(fmt.Sprintf("AdvancedStatefulSet=%v", test.enableAsts))
		actualSet := oldSet.DeepCopy()
		for _, op := range test.ops {
			if op.preHandler != nil {
				op.preHandler(tc)
			}
			desiredSet := newSet.DeepCopy()
			_ = scaler.ScaleIn(tc, actualSet, desiredSet)
			if diff := cmp.Diff(op.replicas, *desiredSet.Spec.Replicas); diff != "" {
				t.Errorf("unexpected (-want, +got): %s", diff)
			}
			if diff := cmp.Diff(op.deleteSlots, helper.GetDeleteSlots(desiredSet)); diff != "" {
				t.Errorf("unexpected (-want, +got): %s", diff)
			}
			actualSet = desiredSet.DeepCopy()
		}
	}

	tests := []testcase{
		{
			name:           "scale without deleteSlots disable asts",
			enableAsts:     false,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(),
			ops: []scaleOp{
				{
					replicas: 3,
				},
			},
		}, {
			name:           "scale without deleteSlots endable asts",
			enableAsts:     true,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(),
			ops: []scaleOp{
				{
					replicas: 3,
				},
			},
		}, {
			name:           "scale with deleteSlots endable asts",
			enableAsts:     true,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(3, 2),
			ops: []scaleOp{
				{
					replicas:    3,
					deleteSlots: sets.NewInt32(3, 2),
				},
			},
		}, {
			name:           "scale second error without deleteSlots disable asts",
			enableAsts:     false,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(),
			ops: []scaleOp{
				{
					preHandler: func(tc *v1alpha1.TidbCluster) {
						// hack ID to mock error during scale one
						store := tc.Status.TiKV.TombstoneStores["13"]
						store.ID = "not integer"
						tc.Status.TiKV.TombstoneStores["13"] = store
					},
					replicas:    4,
					deleteSlots: sets.NewInt32(),
				}, {
					preHandler: func(tc *v1alpha1.TidbCluster) {
						// hack ID to mock error during scale one
						store := tc.Status.TiKV.TombstoneStores["13"]
						store.ID = "13"
						tc.Status.TiKV.TombstoneStores["13"] = store
					},
					replicas:    3,
					deleteSlots: sets.NewInt32(),
				},
			},
		}, {
			name:           "scale first error without deleteSlots disable asts",
			enableAsts:     false,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(),
			ops: []scaleOp{
				{
					preHandler: func(tc *v1alpha1.TidbCluster) {
						// hack ID to mock error during scale one
						store := tc.Status.TiKV.TombstoneStores["1"]
						store.ID = "not integer"
						tc.Status.TiKV.TombstoneStores["1"] = store
					},
					replicas:    5,
					deleteSlots: sets.NewInt32(),
				}, {
					preHandler: func(tc *v1alpha1.TidbCluster) {
						// hack ID to mock error during scale one
						store := tc.Status.TiKV.TombstoneStores["1"]
						store.ID = "1"
						tc.Status.TiKV.TombstoneStores["1"] = store
					},
					replicas:    3,
					deleteSlots: sets.NewInt32(),
				},
			},
		}, {
			name:           "scale second error without deleteSlots enable asts",
			enableAsts:     true,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(),
			ops: []scaleOp{
				{
					preHandler: func(tc *v1alpha1.TidbCluster) {
						// hack ID to mock error during scale one
						store := tc.Status.TiKV.TombstoneStores["13"]
						store.ID = "not integer"
						tc.Status.TiKV.TombstoneStores["13"] = store
					},
					replicas:    4,
					deleteSlots: sets.NewInt32(),
				}, {
					preHandler: func(tc *v1alpha1.TidbCluster) {
						// hack ID to mock error during scale one
						store := tc.Status.TiKV.TombstoneStores["13"]
						store.ID = "13"
						tc.Status.TiKV.TombstoneStores["13"] = store
					},
					replicas:    3,
					deleteSlots: sets.NewInt32(),
				},
			},
		}, {
			name:           "scale first error without deleteSlots enable asts",
			enableAsts:     true,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(),
			ops: []scaleOp{
				{
					preHandler: func(tc *v1alpha1.TidbCluster) {
						// hack ID to mock error during scale one
						store := tc.Status.TiKV.TombstoneStores["1"]
						store.ID = "not integer"
						tc.Status.TiKV.TombstoneStores["1"] = store
					},
					replicas:    5,
					deleteSlots: sets.NewInt32(),
				}, {
					preHandler: func(tc *v1alpha1.TidbCluster) {
						// hack ID to mock error during scale one
						store := tc.Status.TiKV.TombstoneStores["1"]
						store.ID = "1"
						tc.Status.TiKV.TombstoneStores["1"] = store
					},
					replicas:    3,
					deleteSlots: sets.NewInt32(),
				},
			},
		}, {
			name:           "scale second error with deleteSlots enable asts",
			enableAsts:     true,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(3, 2),
			ops: []scaleOp{
				{
					preHandler: func(tc *v1alpha1.TidbCluster) {
						// hack ID to mock error during scale one
						store := tc.Status.TiKV.TombstoneStores["12"]
						store.ID = "not integer"
						tc.Status.TiKV.TombstoneStores["12"] = store
					},
					replicas:    4,
					deleteSlots: sets.NewInt32(3),
				}, {
					preHandler: func(tc *v1alpha1.TidbCluster) {
						// hack ID to mock error during scale one
						store := tc.Status.TiKV.TombstoneStores["12"]
						store.ID = "13"
						tc.Status.TiKV.TombstoneStores["12"] = store
					},
					replicas:    3,
					deleteSlots: sets.NewInt32(3, 2),
				},
			},
		}, {
			name:           "scale first error with deleteSlots enable asts",
			enableAsts:     true,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(3, 2),
			ops: []scaleOp{
				{
					preHandler: func(tc *v1alpha1.TidbCluster) {
						// hack ID to mock error during scale one
						store := tc.Status.TiKV.TombstoneStores["13"]
						store.ID = "not integer"
						tc.Status.TiKV.TombstoneStores["13"] = store
					},
					replicas:    5,
					deleteSlots: sets.NewInt32(),
				}, {
					preHandler: func(tc *v1alpha1.TidbCluster) {
						// hack ID to mock error during scale one
						store := tc.Status.TiKV.TombstoneStores["13"]
						store.ID = "13"
						tc.Status.TiKV.TombstoneStores["13"] = store
					},
					replicas:    3,
					deleteSlots: sets.NewInt32(3, 2),
				},
			},
		}, {
			name:           "scale with redundant deleteSlots endable asts",
			enableAsts:     true,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(2, 5),
			ops: []scaleOp{
				{
					replicas:    3,
					deleteSlots: sets.NewInt32(2, 5),
				},
			},
		}, {
			name:           "scale second error with redundant deleteSlots endable asts",
			enableAsts:     true,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(2, 5),
			ops: []scaleOp{
				{
					preHandler: func(tc *v1alpha1.TidbCluster) {
						// hack ID to mock error during scale one
						store := tc.Status.TiKV.TombstoneStores["12"]
						store.ID = "not integer"
						tc.Status.TiKV.TombstoneStores["12"] = store
					},
					replicas:    4,
					deleteSlots: sets.NewInt32(5),
				}, {
					preHandler: func(tc *v1alpha1.TidbCluster) {
						// hack ID to mock error during scale one
						store := tc.Status.TiKV.TombstoneStores["12"]
						store.ID = "12"
						tc.Status.TiKV.TombstoneStores["12"] = store
					},
					replicas:    3,
					deleteSlots: sets.NewInt32(2, 5),
				},
			},
		}, {
			name:           "scale first error with redundant deleteSlots enable asts",
			enableAsts:     true,
			oldDeleteSlots: sets.NewInt32(),
			newDeleteSlots: sets.NewInt32(2, 5),
			ops: []scaleOp{
				{
					preHandler: func(tc *v1alpha1.TidbCluster) {
						// hack ID to mock error during scale one
						store := tc.Status.TiKV.TombstoneStores["1"]
						store.ID = "not integer"
						tc.Status.TiKV.TombstoneStores["1"] = store
					},
					replicas:    5,
					deleteSlots: sets.NewInt32(),
				}, {
					preHandler: func(tc *v1alpha1.TidbCluster) {
						// hack ID to mock error during scale one
						store := tc.Status.TiKV.TombstoneStores["1"]
						store.ID = "1"
						tc.Status.TiKV.TombstoneStores["1"] = store
					},
					replicas:    3,
					deleteSlots: sets.NewInt32(2, 5),
				},
			},
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
	pvcIndexer := fakeDeps.KubeInformerFactory.Core().V1().PersistentVolumeClaims().Informer().GetIndexer()
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
		"10": {
			ID:      "10",
			PodName: ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), 0),
			State:   v1alpha1.TiKVStateUp,
		},
		"11": {
			ID:      "11",
			PodName: ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), 1),
			State:   v1alpha1.TiKVStateUp,
		},
		"12": {
			ID:      "12",
			PodName: ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), 2),
			State:   v1alpha1.TiKVStateUp,
		},
		"13": {
			ID:      "13",
			PodName: ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), 3),
			State:   v1alpha1.TiKVStateUp,
		},
	}
}

func notReadyStoreFun(tc *v1alpha1.TidbCluster) {
	normalStoreFun(tc)
	delete(tc.Status.TiKV.Stores, "1")
}

func tombstoneStoreFun(tc *v1alpha1.TidbCluster) {
	notReadyStoreFun(tc)

	tc.Status.TiKV.TombstoneStores = map[string]v1alpha1.TiKVStore{
		"1": {
			ID:      "1",
			PodName: ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), 4),
			State:   v1alpha1.TiKVStateTombstone,
		},
	}
}

func multiTombstoneStoreFun(tc *v1alpha1.TidbCluster) {
	normalStoreFun(tc)
	delete(tc.Status.TiKV.Stores, "1")
	delete(tc.Status.TiKV.Stores, "13")

	tc.Status.TiKV.TombstoneStores = map[string]v1alpha1.TiKVStore{
		"1": {
			ID:      "1",
			PodName: ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), 4),
			State:   v1alpha1.TiKVStateTombstone,
		},
		"13": {
			ID:      "13",
			PodName: ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), 3),
			State:   v1alpha1.TiKVStateTombstone,
		},
	}
}

func minimalUpStoreFun(tc *v1alpha1.TidbCluster) {
	normalStoreFun(tc)

	tc.Status.TiKV.Stores["12"] = v1alpha1.TiKVStore{State: v1alpha1.TiKVStateDown}
	tc.Status.TiKV.Stores["13"] = v1alpha1.TiKVStore{State: v1alpha1.TiKVStateDown}
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
	g.Expect(perrors.Find(err, controller.IsRequeueError) != nil).To(Equal(true))
}

func errExpectAllRequeue(g *GomegaWithT, err error) {
	if e, ok := err.(errorutils.Aggregate); ok {
		for _, ee := range e.Errors() {
			g.Expect(controller.IsRequeueError(ee)).To(Equal(true))
		}
	} else {
		g.Expect(controller.IsRequeueError(err)).To(Equal(true))
	}
}

func allTombstonesStoreFun(tc *v1alpha1.TidbCluster) {
	tc.Status.TiKV.TombstoneStores = map[string]v1alpha1.TiKVStore{
		"1": {
			ID:      "1",
			PodName: ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), 4),
			State:   v1alpha1.TiKVStateTombstone,
		},
		"10": {
			ID:      "10",
			PodName: ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), 0),
			State:   v1alpha1.TiKVStateTombstone,
		},
		"11": {
			ID:      "11",
			PodName: ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), 1),
			State:   v1alpha1.TiKVStateTombstone,
		},
		"12": {
			ID:      "12",
			PodName: ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), 2),
			State:   v1alpha1.TiKVStateTombstone,
		},
		"13": {
			ID:      "13",
			PodName: ordinalPodName(v1alpha1.TiKVMemberType, tc.GetName(), 3),
			State:   v1alpha1.TiKVStateTombstone,
		},
	}
}
