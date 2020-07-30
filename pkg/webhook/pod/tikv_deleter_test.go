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
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	memberUtils "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	admission "k8s.io/api/admission/v1beta1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8sTesting "k8s.io/client-go/testing"
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
		PVCNotFound    bool
		expectFn       func(g *GomegaWithT, response *admission.AdmissionResponse)
	}

	testFn := func(test *testcase) {

		t.Log(test.name)
		deleteTiKVPod := newTiKVPod(1)
		ownerStatefulSet := newOwnerStatefulsetForTikv()
		tc := newTidbClusterForPodAdmissionControl(pdReplicas, tikvReplicas)
		kubeCli := kubefake.NewSimpleClientset()

		podAdmissionControl := newPodAdmissionControl(kubeCli)
		pdControl := pdapi.NewFakePDControl(kubeCli)
		fakePDClient := controller.NewFakePDClient(pdControl, tc)

		storesInfo := newTiKVStoresInfo()
		if test.isUpgrading {
			ownerStatefulSet.Status.CurrentRevision = "1"
			ownerStatefulSet.Status.UpdateRevision = "2"
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
		fakePDClient.AddReaction(pdapi.GetStoreActionType, func(action *pdapi.Action) (i interface{}, e error) {
			return &pdapi.StoreInfo{
				Store: &pdapi.MetaStore{
					Store: &metapb.Store{
						Id: action.ID,
					},
					StateName: test.storeState,
				},
			}, nil
		})

		if test.isOutOfOrdinal {
			pod_3 := newTiKVPod(3)
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

		if test.UpdatePVCErr {
			if test.PVCNotFound {
				kubeCli.PrependReactor("get", "persistentvolumeclaims", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.NewNotFound(action.GetResource().GroupResource(), "name")
				})
			} else {
				kubeCli.PrependReactor("get", "persistentvolumeclaims", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, fmt.Errorf("some errors")
				})
			}
		}

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
			storeState:     v1alpha1.TiKVStateDown,
			UpdatePVCErr:   false,
			PVCNotFound:    false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed).Should(Equal(true))
			},
		},
		{
			name:           "no store,out of ordinal",
			isStoreExist:   false,
			isOutOfOrdinal: true,
			isUpgrading:    false,
			storeState:     v1alpha1.TiKVStateDown,
			UpdatePVCErr:   false,
			PVCNotFound:    false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed).Should(Equal(false))
			},
		},
		{
			name:           "no store,out of ordinal,update pvc error",
			isStoreExist:   false,
			isOutOfOrdinal: true,
			isUpgrading:    false,
			storeState:     v1alpha1.TiKVStateDown,
			UpdatePVCErr:   true,
			PVCNotFound:    false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed).Should(Equal(false))
			},
		},
		{
			name:           "no store,out of ordinal,update pvc error, pvc not found",
			isStoreExist:   false,
			isOutOfOrdinal: true,
			isUpgrading:    false,
			storeState:     v1alpha1.TiKVStateDown,
			UpdatePVCErr:   true,
			PVCNotFound:    true,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed).Should(Equal(false))
			},
		},
		{
			name:           "first normal upgraded",
			isStoreExist:   true,
			isOutOfOrdinal: false,
			isUpgrading:    true,
			storeState:     v1alpha1.TiKVStateUp,
			UpdatePVCErr:   false,
			PVCNotFound:    false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed).Should(Equal(false))
			},
		},
		{
			name:           "first normal scale-in",
			isStoreExist:   true,
			isOutOfOrdinal: true,
			isUpgrading:    false,
			storeState:     v1alpha1.TiKVStateUp,
			UpdatePVCErr:   false,
			PVCNotFound:    false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed).Should(Equal(false))
			},
		},
		{
			name:           "tombstone Upgrading",
			isStoreExist:   true,
			isOutOfOrdinal: false,
			isUpgrading:    true,
			storeState:     v1alpha1.TiKVStateTombstone,
			UpdatePVCErr:   false,
			PVCNotFound:    false,
			expectFn: func(g *GomegaWithT, response *admission.AdmissionResponse) {
				g.Expect(response.Allowed).Should(Equal(true))
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
		label.StoreIDLabelKey:   fmt.Sprintf("%d", ordinal),
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
