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
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
)

func TestTiKVFailoverFailover(t *testing.T) {
	tests := []struct {
		name     string
		update   func(*v1alpha1.TidbCluster)
		err      bool
		expectFn func(t *testing.T, tc *v1alpha1.TidbCluster)
	}{
		{
			name: "normal",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"1": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-1",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
					"2": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-2",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-61 * time.Minute)},
					},
				}
			},
			err: false,
			expectFn: func(t *testing.T, tc *v1alpha1.TidbCluster) {
				g := NewGomegaWithT(t)
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(2))
				g.Expect(tc.Status.TiKV.FailoverUID).NotTo(BeEmpty())
			},
		},
		{
			name: "tikv state is not Down",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"1": {State: v1alpha1.TiKVStateUp, PodName: "tikv-1"},
				}
			},
			err: false,
			expectFn: func(t *testing.T, tc *v1alpha1.TidbCluster) {
				g := NewGomegaWithT(t)
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(0))
				g.Expect(tc.Status.TiKV.FailoverUID).To(BeEmpty())
			},
		},
		{
			name: "deadline not exceed",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"1": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-1",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-30 * time.Minute)},
					},
				}
			},
			err: false,
			expectFn: func(t *testing.T, tc *v1alpha1.TidbCluster) {
				g := NewGomegaWithT(t)
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(0))
				g.Expect(tc.Status.TiKV.FailoverUID).To(BeEmpty())
			},
		},
		{
			name: "lastTransitionTime is zero",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"1": {
						State:   v1alpha1.TiKVStateDown,
						PodName: "tikv-1",
					},
				}
			},
			err: false,
			expectFn: func(t *testing.T, tc *v1alpha1.TidbCluster) {
				g := NewGomegaWithT(t)
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(0))
				g.Expect(tc.Status.TiKV.FailoverUID).To(BeEmpty())
			},
		},
		{
			name: "exist in failureStores",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"1": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-1",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
				}
				tc.Status.TiKV.FailureStores = map[string]v1alpha1.TiKVFailureStore{
					"1": {
						PodName: "tikv-1",
						StoreID: "1",
					},
				}
			},
			err: false,
			expectFn: func(t *testing.T, tc *v1alpha1.TidbCluster) {
				g := NewGomegaWithT(t)
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(1))
				g.Expect(tc.Status.TiKV.FailoverUID).NotTo(BeEmpty())
			},
		},
		{
			name: "not exceed max failover count",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"3": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-0",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
					"4": {
						State:              v1alpha1.TiKVStateUp,
						PodName:            "tikv-4",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
					"5": {
						State:              v1alpha1.TiKVStateUp,
						PodName:            "tikv-5",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-61 * time.Minute)},
					},
				}
				tc.Status.TiKV.FailureStores = map[string]v1alpha1.TiKVFailureStore{
					"1": {
						PodName: "tikv-1",
						StoreID: "1",
					},
					"2": {
						PodName: "tikv-2",
						StoreID: "2",
					},
				}
			},
			err: false,
			expectFn: func(t *testing.T, tc *v1alpha1.TidbCluster) {
				g := NewGomegaWithT(t)
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(3))
				g.Expect(tc.Status.TiKV.FailoverUID).NotTo(BeEmpty())
			},
		},
		{
			name: "exceed max failover count1",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"3": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-3",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
					"4": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-4",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
					"5": {
						State:              v1alpha1.TiKVStateUp,
						PodName:            "tikv-5",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-61 * time.Minute)},
					},
				}
				tc.Status.TiKV.FailureStores = map[string]v1alpha1.TiKVFailureStore{
					"1": {
						PodName: "tikv-1",
						StoreID: "1",
					},
					"2": {
						PodName: "tikv-2",
						StoreID: "2",
					},
				}
			},
			err: false,
			expectFn: func(t *testing.T, tc *v1alpha1.TidbCluster) {
				g := NewGomegaWithT(t)
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(3))
				g.Expect(tc.Status.TiKV.FailoverUID).NotTo(BeEmpty())
			},
		},
		{
			name: "exceed max failover count2",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"0": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-0",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
					"4": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-4",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-61 * time.Minute)},
					},
					"5": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-5",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
				}
				tc.Status.TiKV.FailureStores = map[string]v1alpha1.TiKVFailureStore{
					"1": {
						PodName: "tikv-1",
						StoreID: "1",
					},
					"2": {
						PodName: "tikv-2",
						StoreID: "2",
					},
					"3": {
						PodName: "tikv-3",
						StoreID: "3",
					},
				}
			},
			err: false,
			expectFn: func(t *testing.T, tc *v1alpha1.TidbCluster) {
				g := NewGomegaWithT(t)
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(3))
				g.Expect(tc.Status.TiKV.FailoverUID).NotTo(BeEmpty())
			},
		},
		{
			name: "exceed max failover count2 but maxFailoverCount = 0",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Spec.TiKV.MaxFailoverCount = pointer.Int32Ptr(0)
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"12": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-12",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
					"13": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-13",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-61 * time.Minute)},
					},
					"14": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-14",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
				}
				tc.Status.TiKV.FailureStores = map[string]v1alpha1.TiKVFailureStore{
					"1": {
						PodName: "tikv-1",
						StoreID: "1",
					},
					"2": {
						PodName: "tikv-2",
						StoreID: "2",
					},
					"3": {
						PodName: "tikv-3",
						StoreID: "3",
					},
				}
			},
			err: false,
			expectFn: func(t *testing.T, tc *v1alpha1.TidbCluster) {
				g := NewGomegaWithT(t)
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(3))
				g.Expect(tc.Status.TiKV.FailoverUID).To(BeEmpty())
			},
		},
		{
			name: "already have failoverUID",
			update: func(tc *v1alpha1.TidbCluster) {
				tc.Status.TiKV.FailoverUID = "failover-uid-test"
				tc.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"1": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-1",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-70 * time.Minute)},
					},
					"2": {
						State:              v1alpha1.TiKVStateDown,
						PodName:            "tikv-2",
						LastTransitionTime: metav1.Time{Time: time.Now().Add(-61 * time.Minute)},
					},
				}
			},
			err: false,
			expectFn: func(t *testing.T, tc *v1alpha1.TidbCluster) {
				g := NewGomegaWithT(t)
				g.Expect(len(tc.Status.TiKV.FailureStores)).To(Equal(2))
				g.Expect(tc.Status.TiKV.FailoverUID).To(Equal(types.UID("failover-uid-test")))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			tc := newTidbClusterForPD()
			tc.Spec.TiKV.Replicas = 6
			tc.Spec.TiKV.MaxFailoverCount = pointer.Int32Ptr(3)
			tt.update(tc)

			fakeDeps := controller.NewFakeDependencies()
			fakeDeps.CLIConfig.TiKVFailoverPeriod = 1 * time.Hour
			storeAccess := tikvStoreAccess{}
			tikvFailover := &commonStoreFailover{
				storeAccess: &storeAccess,
				deps:        fakeDeps,
				failureRecovery: commonStatefulFailureRecovery{
					deps:                fakeDeps,
					failureObjectAccess: &failureStoreAccess{storeAccess: &storeAccess},
				},
			}

			err := tikvFailover.Failover(tc)
			if tt.err {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
			tt.expectFn(t, tc)
		})
	}
}

func TestTiKVStoreAccess(t *testing.T) {
	g := NewGomegaWithT(t)

	tc := newTidbClusterWithTiKVFailureMember(false, false, false)
	cliConfig := &controller.CLIConfig{TiKVFailoverPeriod: 6 * time.Minute}

	storeAccess := &tikvStoreAccess{}

	g.Expect(storeAccess.GetFailoverPeriod(cliConfig)).To(Equal(6 * time.Minute))

	g.Expect(storeAccess.GetMemberType()).To(Equal(v1alpha1.TiKVMemberType))

	g.Expect(storeAccess.GetMaxFailoverCount(tc)).To(Equal(pointer.Int32Ptr(int32(2))))

	g.Expect(storeAccess.GetStores(tc)).To(Equal(tc.Status.TiKV.Stores))

	store, exists := storeAccess.GetStore(tc, "1")
	g.Expect(exists).To(Equal(true))
	g.Expect(store).To(Equal(tc.Status.TiKV.Stores["1"]))

	storeAccess.SetFailoverUIDIfAbsent(tc)
	g.Expect(tc.Status.TiKV.FailoverUID).NotTo(BeEmpty())

	storeAccess.CreateFailureStoresIfAbsent(tc)
	g.Expect(tc.Status.TiKV.FailureStores).NotTo(BeNil())

	tc.Status.TiKV.FailureStores["1"] = v1alpha1.TiKVFailureStore{StoreID: "1"}
	g.Expect(storeAccess.GetFailureStores(tc)).To(Equal(map[string]v1alpha1.TiKVFailureStore{"1": {StoreID: "1"}}))

	failureStore, exists := storeAccess.GetFailureStore(tc, "1")
	g.Expect(exists).To(Equal(true))
	g.Expect(failureStore).To(Equal(v1alpha1.TiKVFailureStore{StoreID: "1"}))

	g.Expect(storeAccess.IsHostDownForFailurePod(tc)).To(Equal(false))

	storeAccess.SetFailureStore(tc, "1", v1alpha1.TiKVFailureStore{StoreID: "1", HostDown: true})
	failureStore, exists = storeAccess.GetFailureStore(tc, "1")
	g.Expect(exists).To(Equal(true))
	g.Expect(failureStore).To(Equal(v1alpha1.TiKVFailureStore{StoreID: "1", HostDown: true}))
	g.Expect(storeAccess.IsHostDownForFailurePod(tc)).To(Equal(true))

	g.Expect(storeAccess.GetStsDesiredOrdinals(tc, true)).To(Equal(sets.Int32{int32(0): {}, int32(1): {}, int32(2): {}}))

	storeAccess.ClearFailStatus(tc)
	g.Expect(tc.Status.TiKV.FailoverUID).To(BeEmpty())
	g.Expect(tc.Status.TiKV.FailureStores).To(BeEmpty())
}

func TestKVFailureStoreAccess(t *testing.T) {
	g := NewGomegaWithT(t)

	tc := newTidbClusterWithTiKVFailureMember(true, false, false)

	fsAccess := &failureStoreAccess{storeAccess: &tikvStoreAccess{}}

	g.Expect(fsAccess.GetMemberType()).To(Equal(v1alpha1.TiKVMemberType))

	g.Expect(fsAccess.GetFailureObjects(tc)).To(Equal(map[string]v1alpha1.EmptyStruct{"1": {}}))

	g.Expect(fsAccess.IsFailing(tc, "0")).To(BeFalse())
	g.Expect(fsAccess.IsFailing(tc, "1")).To(BeTrue())

	g.Expect(fsAccess.GetPodName(tc, "1")).To(Equal("test-tikv-1"))

	g.Expect(fsAccess.IsHostDownForFailedPod(tc)).To(BeFalse())

	fsAccess.SetHostDown(tc, "1", true)
	g.Expect(fsAccess.IsHostDownForFailedPod(tc)).To(BeTrue())

	g.Expect(fsAccess.GetCreatedAt(tc, "1")).To(Equal(tc.Status.TiKV.FailureStores["1"].CreatedAt))

	g.Expect(fsAccess.GetLastTransitionTime(tc, "1")).To(Equal(tc.Status.TiKV.Stores["1"].LastTransitionTime))

	g.Expect(fsAccess.GetPVCUIDSet(tc, "1")).To(Equal(map[types.UID]v1alpha1.EmptyStruct{types.UID("pvc-1-uid-1"): {}}))
}

func TestInvokeDeleteTiKVFailureStore(t *testing.T) {
	g := NewGomegaWithT(t)

	recorder := record.NewFakeRecorder(100)
	type testcase struct {
		name         string
		storeDeleted bool
		errExpectFn  func(*GomegaWithT, error)
		expectFn     func(*v1alpha1.TidbCluster)
	}

	tests := []testcase{
		{
			name:         "store is deleted, no action",
			storeDeleted: true,
			errExpectFn:  errExpectNil,
			expectFn: func(*v1alpha1.TidbCluster) {
				events := collectEvents(recorder.Events)
				g.Expect(events).To(HaveLen(0))
			},
		},
		{
			name:         "store is not deleted, delete store",
			storeDeleted: false,
			errExpectFn:  errExpectRequeueError,
			expectFn: func(*v1alpha1.TidbCluster) {
				events := collectEvents(recorder.Events)
				g.Expect(events).To(HaveLen(1))
				g.Expect(events[0]).To(ContainSubstring("Invoked delete on tikv store '1' in cluster default/test"))
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tc := newTidbClusterWithTiKVFailureMember(true, true, test.storeDeleted)
			fakeDeps, _, podIndexer, _ := newFakeDependenciesForFailover(true)
			fakeDeps.Recorder = recorder
			pod := newPodForFailover(tc, v1alpha1.TiKVMemberType, 1)
			pod.CreationTimestamp = metav1.NewTime(time.Now().Add(-restartToDeleteStoreGap - time.Second))
			podIndexer.Add(pod)

			fakePDControl := fakeDeps.PDControl.(*pdapi.FakePDControl)
			pdClient := controller.NewFakePDClient(fakePDControl, tc)
			if !test.storeDeleted {
				pdClient.AddReaction(pdapi.DeleteStoreActionType, func(action *pdapi.Action) (interface{}, error) {
					return nil, nil
				})
			}

			storeAccess := tikvStoreAccess{}
			tikvFailover := &commonStoreFailover{
				storeAccess: &storeAccess,
				deps:        fakeDeps,
				failureRecovery: commonStatefulFailureRecovery{
					deps:                fakeDeps,
					failureObjectAccess: &failureStoreAccess{storeAccess: &storeAccess},
				},
			}
			err := tikvFailover.invokeDeleteFailureStore(tc, tc.Status.TiKV.FailureStores["1"])
			test.errExpectFn(g, err)
			test.expectFn(tc)
		})
	}
}

func TestCheckAndRemoveTiKVFailurePVC(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name        string
		errExpectFn func(*GomegaWithT, error)
		expectFn    func(*v1alpha1.TidbCluster, cache.Indexer, cache.Indexer)
	}

	tests := []testcase{
		{
			name:        "store is deleted, no action",
			errExpectFn: errExpectNil,
			expectFn: func(tc *v1alpha1.TidbCluster, pvcIndexer cache.Indexer, podIndexer cache.Indexer) {
				g.Expect(tc.Status.TiKV.FailureStores["1"].StoreDeleted).To(BeFalse())
				g.Expect(podIndexer.ListKeys()).To(ContainElement("default/test-tikv-1"))
				g.Expect(pvcIndexer.ListKeys()).To(ContainElement("default/tikv-test-tikv-1-1"))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tc := newTidbClusterWithTiKVFailureMember(true, true, false)

			fakeDeps, pvcIndexer, podIndexer, _ := newFakeDependenciesForFailover(true)
			_, _ = getTestTiKVPodAndPvcs(pvcIndexer, podIndexer, tc, testPodPvcParams{})

			storeAccess := tikvStoreAccess{}
			tikvFailover := &commonStoreFailover{
				storeAccess: &storeAccess,
				deps:        fakeDeps,
				failureRecovery: commonStatefulFailureRecovery{
					deps:                fakeDeps,
					failureObjectAccess: &failureStoreAccess{storeAccess: &storeAccess},
				},
			}
			err := tikvFailover.invokeDeleteFailureStore(tc, tc.Status.TiKV.FailureStores["1"])
			test.errExpectFn(g, err)
			test.expectFn(tc, pvcIndexer, podIndexer)
		})
	}
}

func newTidbClusterWithTiKVFailureMember(hasFailureStore, hostDown, storeDeleted bool) *v1alpha1.TidbCluster {
	tc := newTidbClusterForPD()
	tc.Spec.TiKV.MaxFailoverCount = pointer.Int32Ptr(int32(2))
	pvcUIDSet := map[types.UID]v1alpha1.EmptyStruct{
		types.UID("pvc-1-uid-1"): {},
	}
	time10mAgo := time.Now().Add(-10 * time.Minute)
	time1hAgo := time.Now().Add(-time.Hour)
	tc.Status = v1alpha1.TidbClusterStatus{
		TiKV: v1alpha1.TiKVStatus{
			Stores: map[string]v1alpha1.TiKVStore{
				"0": {ID: "0", State: v1alpha1.TiKVStateUp, LastTransitionTime: metav1.Now()},
				"1": {ID: "1", State: v1alpha1.TiKVStateDown, LastTransitionTime: metav1.NewTime(time10mAgo)},
				"2": {ID: "2", State: v1alpha1.TiKVStateUp, LastTransitionTime: metav1.Now()},
			},
		},
		TiFlash: v1alpha1.TiFlashStatus{},
	}
	if hasFailureStore {
		tc.Status.TiKV.FailureStores = map[string]v1alpha1.TiKVFailureStore{
			"1": {
				PodName:      "test-tikv-1",
				StoreID:      "1",
				CreatedAt:    metav1.NewTime(time1hAgo),
				PVCUIDSet:    pvcUIDSet,
				HostDown:     hostDown,
				StoreDeleted: storeDeleted,
			},
		}
	}
	return tc
}

func getTestTiKVPodAndPvcs(pvcIndexer cache.Indexer, podIndexer cache.Indexer, tc *v1alpha1.TidbCluster, testPodPvcParams testPodPvcParams) (*corev1.Pod, []*corev1.PersistentVolumeClaim) {
	pvc1 := newPVCForTiKVFailover(tc, v1alpha1.TiKVMemberType, 1)
	pvc1.Name = pvc1.Name + "-1"
	pvc1.UID = pvc1.UID + "-1"

	if testPodPvcParams.pvcWithDeletionTimestamp {
		pvc1.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	}
	pvcIndexer.Add(pvc1)
	pod := newPodForFailover(tc, v1alpha1.TiKVMemberType, 1)
	if testPodPvcParams.podWithDeletionTimestamp {
		pod.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	}
	pvc1.ObjectMeta.Labels[label.AnnPodNameKey] = pod.GetName()
	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc1.Name,
				},
			},
		})

	podIndexer.Add(pod)
	return pod, []*corev1.PersistentVolumeClaim{pvc1}
}

func newPVCForTiKVFailover(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType, ordinal int32) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ordinalPVCName(memberType, controller.TiKVMemberName(tc.GetName()), ordinal),
			Namespace: metav1.NamespaceDefault,
			UID:       types.UID("pvc-1-uid"),
			Labels: map[string]string{
				label.NameLabelKey:      "tidb-cluster",
				label.ManagedByLabelKey: label.TiDBOperator,
				label.InstanceLabelKey:  "test",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: fmt.Sprintf("pv-%d", ordinal),
		},
	}
}
