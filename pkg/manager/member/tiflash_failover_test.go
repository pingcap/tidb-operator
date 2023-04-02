// Copyright 2022 PingCAP, Inc.
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
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"
)

func TestTiFlashStoreAccess(t *testing.T) {
	g := NewGomegaWithT(t)

	tc := newTidbClusterWithTiFlashFailureMember(false, false, false)
	cliConfig := &controller.CLIConfig{TiFlashFailoverPeriod: 6 * time.Minute}

	storeAccess := &tiflashStoreAccess{}

	g.Expect(storeAccess.GetFailoverPeriod(cliConfig)).To(Equal(6 * time.Minute))

	g.Expect(storeAccess.GetMemberType()).To(Equal(v1alpha1.TiFlashMemberType))

	g.Expect(storeAccess.GetMaxFailoverCount(tc)).To(Equal(pointer.Int32Ptr(int32(4))))

	g.Expect(storeAccess.GetStores(tc)).To(Equal(tc.Status.TiFlash.Stores))

	store, exists := storeAccess.GetStore(tc, "2")
	g.Expect(exists).To(Equal(true))
	g.Expect(store).To(Equal(tc.Status.TiFlash.Stores["2"]))

	storeAccess.SetFailoverUIDIfAbsent(tc)
	g.Expect(tc.Status.TiFlash.FailoverUID).NotTo(BeEmpty())

	storeAccess.CreateFailureStoresIfAbsent(tc)
	g.Expect(tc.Status.TiFlash.FailureStores).NotTo(BeNil())

	tc.Status.TiFlash.FailureStores["2"] = v1alpha1.TiKVFailureStore{StoreID: "2"}
	g.Expect(storeAccess.GetFailureStores(tc)).To(Equal(map[string]v1alpha1.TiKVFailureStore{"2": {StoreID: "2"}}))

	failureStore, exists := storeAccess.GetFailureStore(tc, "2")
	g.Expect(exists).To(Equal(true))
	g.Expect(failureStore).To(Equal(v1alpha1.TiKVFailureStore{StoreID: "2"}))

	g.Expect(storeAccess.IsHostDownForFailurePod(tc)).To(Equal(false))

	storeAccess.SetFailureStore(tc, "2", v1alpha1.TiKVFailureStore{StoreID: "2", HostDown: true})
	failureStore, exists = storeAccess.GetFailureStore(tc, "2")
	g.Expect(exists).To(Equal(true))
	g.Expect(failureStore).To(Equal(v1alpha1.TiKVFailureStore{StoreID: "2", HostDown: true}))
	g.Expect(storeAccess.IsHostDownForFailurePod(tc)).To(Equal(true))

	g.Expect(storeAccess.GetStsDesiredOrdinals(tc, true)).To(Equal(sets.Int32{int32(0): {}, int32(1): {}, int32(2): {}}))

	storeAccess.ClearFailStatus(tc)
	g.Expect(tc.Status.TiFlash.FailoverUID).To(BeEmpty())
	g.Expect(tc.Status.TiFlash.FailureStores).To(BeEmpty())
}

func TestTiFlashFailureStoreAccess(t *testing.T) {
	g := NewGomegaWithT(t)

	tc := newTidbClusterWithTiFlashFailureMember(true, false, false)

	fsAccess := &failureStoreAccess{storeAccess: &tiflashStoreAccess{}}

	g.Expect(fsAccess.GetMemberType()).To(Equal(v1alpha1.TiFlashMemberType))

	g.Expect(fsAccess.GetFailureObjects(tc)).To(Equal(map[string]v1alpha1.EmptyStruct{"2": {}}))

	g.Expect(fsAccess.IsFailing(tc, "1")).To(BeFalse())
	g.Expect(fsAccess.IsFailing(tc, "2")).To(BeTrue())

	g.Expect(fsAccess.GetPodName(tc, "2")).To(Equal("test-tiflash-2"))

	g.Expect(fsAccess.IsHostDownForFailedPod(tc)).To(BeFalse())

	fsAccess.SetHostDown(tc, "2", true)
	g.Expect(fsAccess.IsHostDownForFailedPod(tc)).To(BeTrue())

	g.Expect(fsAccess.GetCreatedAt(tc, "2")).To(Equal(tc.Status.TiFlash.FailureStores["2"].CreatedAt))

	g.Expect(fsAccess.GetLastTransitionTime(tc, "2")).To(Equal(tc.Status.TiFlash.Stores["2"].LastTransitionTime))

	g.Expect(fsAccess.GetPVCUIDSet(tc, "2")).To(Equal(map[types.UID]v1alpha1.EmptyStruct{types.UID("pvc-2-uid-1"): {}}))
}

func newTidbClusterWithTiFlashFailureMember(hasFailureStore, hostDown, storeDeleted bool) *v1alpha1.TidbCluster {
	tc := newTidbClusterForPD()
	tc.Spec.TiFlash.MaxFailoverCount = pointer.Int32Ptr(int32(4))
	pvcUIDSet := map[types.UID]v1alpha1.EmptyStruct{
		types.UID("pvc-2-uid-1"): {},
	}
	time10mAgo := time.Now().Add(-10 * time.Minute)
	time1hAgo := time.Now().Add(-time.Hour)
	tc.Status = v1alpha1.TidbClusterStatus{
		TiKV: v1alpha1.TiKVStatus{},
		TiFlash: v1alpha1.TiFlashStatus{
			Stores: map[string]v1alpha1.TiKVStore{
				"0": {ID: "0", State: v1alpha1.TiKVStateUp},
				"1": {ID: "1", State: v1alpha1.TiKVStateUp},
				"2": {ID: "2", State: v1alpha1.TiKVStateDown, LastTransitionTime: metav1.NewTime(time10mAgo)},
			},
		},
	}
	if hasFailureStore {
		tc.Status.TiFlash.FailureStores = map[string]v1alpha1.TiKVFailureStore{
			"2": {
				PodName:   "test-tiflash-2",
				StoreID:   "2",
				CreatedAt: metav1.NewTime(time1hAgo),
				PVCUIDSet: pvcUIDSet,
			},
		}
	}
	return tc
}
