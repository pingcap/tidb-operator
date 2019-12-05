// Copyright 2019 PingCAP, Inc.
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

package pod

import (
	"fmt"
	. "github.com/onsi/gomega"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	memberUtils "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	admission "k8s.io/api/admission/v1beta1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"testing"
)

var (
	tiKVStsName = tcName + "-tikv"
)

func TestTiKVDeleterDelete(t *testing.T) {

	g := NewGomegaWithT(t)

	type testcase struct {
		name           string
		isStoreExist   bool
		isOutOfOrdinal bool
		isUpgrading    bool
		storeState     string
		UpdatePVCErr   bool
		expectFn       func(g *GomegaWithT, response *admission.AdmissionResponse)
	}

	testFn := func(test *testcase) {

		t.Log(test.name)

		pod_0 := newTiKVPod(0)
		deleteTiKVPod := newTiKVPod(1)
		pod_2 := newTiKVPod(2)
		pvc_0 := newPVCForTikv(0)
		pvc_1 := newPVCForTikv(1)
		pvc_2 := newPVCForTikv(2)

		ownerStatefulSet := newOwnerStatefulsetForTikv()
		tc := newTidbClusterForPodAdmissionControl()
		kubeCli := kubefake.NewSimpleClientset()

		podAdmissionControl, fakePVCControl, pvcIndexer, podIndexer, _, _ := newPodAdmissionControl()
		pdControl := pdapi.NewFakePDControl(kubeCli)
		fakePDClient := controller.NewFakePDClient(pdControl, tc)

		podIndexer.Add(pod_0)
		podIndexer.Add(deleteTiKVPod)
		podIndexer.Add(pod_2)
		pvcIndexer.Add(pvc_0)
		pvcIndexer.Add(pvc_1)
		pvcIndexer.Add(pvc_2)

		storesInfo := newTiKVStoresInfo()
		if test.isUpgrading {
			ownerStatefulSet.Status.CurrentRevision = "1"
			ownerStatefulSet.Status.UpdateRevision = "2"
		}

		fakePVCControl.SetUpdatePVCError(nil, 0)
		if test.UpdatePVCErr {
			fakePVCControl.SetUpdatePVCError(fmt.Errorf("update pvc error"), 0)
		}

		if test.isOutOfOrdinal {
			pod_3 := newTiKVPod(3)
			pvc_3 := newPVCForTikv(3)
			podIndexer.Add(pod_3)
			pvcIndexer.Add(pvc_3)
			deleteTiKVPod = pod_3
			if test.isStoreExist {
				tc.Status.TiKV.Stores["3"] = v1alpha1.TiKVStore{
					PodName:     memberUtils.TikvPodName(tcName, 3),
					LeaderCount: 1,
					State:       v1alpha1.TiKVStateUp,
				}

				storesInfo.Count = 4
				storesInfo.Stores = append(storesInfo.Stores, &pdapi.StoreInfo{
					Store: &pdapi.MetaStore{
						StateName: v1alpha1.TiKVStateUp,
						Store: &metapb.Store{
							Id:      3,
							Address: fmt.Sprintf("%s-tikv-%d.%s-tikv-peer.%s.svc:20160", tcName, 3, tcName, namespace),
						},
					},
					Status: &pdapi.StoreStatus{
						LeaderCount: 1,
					},
				})
			}
		}

		if !test.isOutOfOrdinal && !test.isStoreExist {
			tc.Status.TiKV = v1alpha1.TiKVStatus{
				Synced: true,
				Phase:  v1alpha1.NormalPhase,
				Stores: map[string]v1alpha1.TiKVStore{
					"0": {
						PodName:     memberUtils.TikvPodName(tcName, 0),
						LeaderCount: 1,
						State:       v1alpha1.TiKVStateUp,
					},
					"2": {
						PodName:     memberUtils.TikvPodName(tcName, 2),
						LeaderCount: 1,
						State:       v1alpha1.TiKVStateUp,
					},
				},
			}
			storesInfo = &pdapi.StoresInfo{
				Count: 2,
				Stores: []*pdapi.StoreInfo{
					{
						Store: &pdapi.MetaStore{
							StateName: v1alpha1.TiKVStateUp,
							Store: &metapb.Store{
								Id:      0,
								Address: fmt.Sprintf("%s-tikv-%d.%s-tikv-peer.%s.svc:20160", tcName, 0, tcName, namespace),
							},
						},
						Status: &pdapi.StoreStatus{
							LeaderCount: 1,
						},
					},
					{
						Store: &pdapi.MetaStore{
							StateName: v1alpha1.TiKVStateUp,
							Store: &metapb.Store{
								Id:      2,
								Address: fmt.Sprintf("%s-tikv-%d.%s-tikv-peer.%s.svc:20160", tcName, 2, tcName, namespace),
							},
						},
						Status: &pdapi.StoreStatus{
							LeaderCount: 1,
						},
					},
				},
			}
		}

		fakePDClient.AddReaction(pdapi.GetStoresActionType, func(action *pdapi.Action) (i interface{}, e error) {
			return storesInfo, nil
		})
		fakePDClient.AddReaction(pdapi.DeleteStoreActionType, func(action *pdapi.Action) (i interface{}, e error) {
			return nil, nil
		})
		fakePDClient.AddReaction(pdapi.BeginEvictLeaderActionType, func(action *pdapi.Action) (i interface{}, e error) {
			return nil, nil
		})

		payload := &admitPayload{
			pod:              deleteTiKVPod,
			ownerStatefulSet: ownerStatefulSet,
			tc:               tc,
			pdClient:         fakePDClient,
		}

		response := podAdmissionControl.admitDeleteTiKVPods(payload)
		test.expectFn(g, response)
	}

	tests := []testcase{
		{
			name:           "no store,no exceed",
			isStoreExist:   false,
			isOutOfOrdinal: false,
			isUpgrading:    false,
			storeState:     "",
			UpdatePVCErr:   false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed, true)
			},
		},
		{
			name:           "no store,out of ordinal",
			isStoreExist:   false,
			isOutOfOrdinal: true,
			isUpgrading:    false,
			storeState:     "",
			UpdatePVCErr:   false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed, true)
			},
		},
		{
			name:           "no store,out of ordinal",
			isStoreExist:   false,
			isOutOfOrdinal: true,
			isUpgrading:    false,
			storeState:     "",
			UpdatePVCErr:   true,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed, true)
				g.Expect(response.Result.Message, "update pvc error")
			},
		},
		{
			name:           "first normal upgraded",
			isStoreExist:   true,
			isOutOfOrdinal: false,
			isUpgrading:    true,
			storeState:     v1alpha1.TiKVStateUp,
			UpdatePVCErr:   false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed, false)
			},
		},
		{
			name:           "first normal scale-in",
			isStoreExist:   true,
			isOutOfOrdinal: true,
			isUpgrading:    false,
			storeState:     v1alpha1.TiKVStateUp,
			UpdatePVCErr:   false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed, false)
			},
		},
		{
			name:           "tombstone Upgrading",
			isStoreExist:   true,
			isOutOfOrdinal: false,
			isUpgrading:    true,
			storeState:     v1alpha1.TiKVStateTombstone,
			UpdatePVCErr:   false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed, true)
			},
		},
	}

	for _, test := range tests {
		testFn(&test)
	}
}

func newTiKVPod(ordinal int32) *core.Pod {

	pod := core.Pod{}
	pod.Labels = map[string]string{
		label.ComponentLabelKey: label.TiKVLabelVal,
	}
	pod.Name = memberUtils.TikvPodName(tcName, ordinal)
	pod.Namespace = namespace
	return &pod
}

func newOwnerStatefulsetForTikv() *apps.StatefulSet {
	sts := apps.StatefulSet{}
	sts.Spec.Replicas = func() *int32 { a := int32(3); return &a }()
	sts.Name = tiKVStsName
	sts.Namespace = namespace
	sts.Status.CurrentRevision = "1"
	sts.Status.UpdateRevision = "1"
	return &sts
}

func newPVCForTikv(ordinal int32) *core.PersistentVolumeClaim {
	return &core.PersistentVolumeClaim{
		ObjectMeta: meta.ObjectMeta{
			Name:      operatorUtils.OrdinalPVCName(v1alpha1.TiKVMemberType, tiKVStsName, ordinal),
			Namespace: namespace,
		},
	}
}

func newTiKVStoresInfo() *pdapi.StoresInfo {
	storesInfo := pdapi.StoresInfo{
		Count: 3,
		Stores: []*pdapi.StoreInfo{
			{
				Store: &pdapi.MetaStore{
					StateName: v1alpha1.TiKVStateUp,
					Store: &metapb.Store{
						Id:      0,
						Address: fmt.Sprintf("%s-tikv-%d.%s-tikv-peer.%s.svc:20160", tcName, 0, tcName, namespace),
					},
				},
				Status: &pdapi.StoreStatus{
					LeaderCount: 1,
				},
			},
			{
				Store: &pdapi.MetaStore{
					StateName: v1alpha1.TiKVStateUp,
					Store: &metapb.Store{
						Id:      1,
						Address: fmt.Sprintf("%s-tikv-%d.%s-tikv-peer.%s.svc:20160", tcName, 1, tcName, namespace),
					},
				},
				Status: &pdapi.StoreStatus{
					LeaderCount: 1,
				},
			},
			{
				Store: &pdapi.MetaStore{
					StateName: v1alpha1.TiKVStateUp,
					Store: &metapb.Store{
						Id:      2,
						Address: fmt.Sprintf("%s-tikv-%d.%s-tikv-peer.%s.svc:20160", tcName, 2, tcName, namespace),
					},
				},
				Status: &pdapi.StoreStatus{
					LeaderCount: 1,
				},
			},
		},
	}

	return &storesInfo
}
