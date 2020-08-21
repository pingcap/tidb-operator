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
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned/fake"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	memberUtils "github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8sTesting "k8s.io/client-go/testing"
)

var (
	tiKVStsName = tcName + "-tikv"
)

func TestTiKVDeleterDelete(t *testing.T) {
	g := NewGomegaWithT(t)
	testcases := []struct {
		name             string
		controller       runtime.Object
		storesInfo       *pdapi.StoresInfo
		ownerStatefulSet *apps.StatefulSet
		deletePod        *core.Pod
		ownerTc          *v1alpha1.TidbCluster
		allowed          bool
	}{
		{
			name:             "tidbcluster,tombstone,normal-delete",
			deletePod:        newTiKVPod(1, true),
			ownerStatefulSet: newOwnerStatefulsetForTikv(false),
			storesInfo: &pdapi.StoresInfo{
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
							StateName: v1alpha1.TiKVStateTombstone,
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
			},
			controller: newTidbClusterForPodAdmissionControl(pdReplicas, tikvReplicas),
			ownerTc:    newTidbClusterForPodAdmissionControl(pdReplicas, tikvReplicas),
			allowed:    true,
		},
		{
			name:             "tidbcluster,tombstone,scale-in",
			deletePod:        newTiKVPod(3, true),
			ownerStatefulSet: newOwnerStatefulsetForTikv(false),
			storesInfo: &pdapi.StoresInfo{
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
					{
						Store: &pdapi.MetaStore{
							StateName: v1alpha1.TiKVStateTombstone,
							Store: &metapb.Store{
								Id:      3,
								Address: fmt.Sprintf("%s-tikv-%d.%s-tikv-peer.%s.svc:20160", tcName, 3, tcName, namespace),
							},
						},
						Status: &pdapi.StoreStatus{
							LeaderCount: 1,
						},
					},
				},
			},
			controller: newTidbClusterForPodAdmissionControl(pdReplicas, tikvReplicas),
			ownerTc:    newTidbClusterForPodAdmissionControl(pdReplicas, tikvReplicas),
			allowed:    true,
		},
		{
			name:             "tidbcluster,up,upgraded",
			deletePod:        newTiKVPod(1, true),
			ownerStatefulSet: newOwnerStatefulsetForTikv(true),
			storesInfo: &pdapi.StoresInfo{
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
			},
			controller: newTidbClusterForPodAdmissionControl(pdReplicas, tikvReplicas),
			ownerTc:    newTidbClusterForPodAdmissionControl(pdReplicas, tikvReplicas),
			allowed:    false,
		},
		{
			name:             "tidbcluster,up,scaling-in",
			deletePod:        newTiKVPod(3, true),
			ownerStatefulSet: newOwnerStatefulsetForTikv(false),
			storesInfo: &pdapi.StoresInfo{
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
					{
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
					},
				},
			},
			controller: newTidbClusterForPodAdmissionControl(pdReplicas, tikvReplicas),
			ownerTc:    newTidbClusterForPodAdmissionControl(pdReplicas, tikvReplicas),
			allowed:    false,
		},
	}
	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			cli := fake.NewSimpleClientset()
			kubeCli := kubefake.NewSimpleClientset()
			podAdmissionControl := newPodAdmissionControl(nil, kubeCli, cli)
			pdControl := pdapi.NewFakePDControl(kubeCli)
			fakePDClient := controller.NewFakePDClient(pdControl, testcase.ownerTc)

			fakePDClient.AddReaction(pdapi.GetStoresActionType, func(action *pdapi.Action) (i interface{}, e error) {
				return testcase.storesInfo, nil
			})
			fakePDClient.AddReaction(pdapi.DeleteStoreActionType, func(action *pdapi.Action) (i interface{}, e error) {
				return nil, nil
			})
			fakePDClient.AddReaction(pdapi.BeginEvictLeaderActionType, func(action *pdapi.Action) (i interface{}, e error) {
				return nil, nil
			})
			pvc := &core.PersistentVolumeClaim{
				ObjectMeta: meta.ObjectMeta{
					Name:      testcase.deletePod.Name,
					Namespace: namespace,
				},
			}
			kubeCli.PrependReactor("get", "persistentvolumeclaims", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, pvc, nil
			})
			kubeCli.PrependReactor("update", "persistentvolumeclaims", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, pvc, nil
			})

			metaObj := testcase.controller.(meta.Object)
			payload := &admitPayload{
				pod:              testcase.deletePod,
				ownerStatefulSet: testcase.ownerStatefulSet,
				controller:       testcase.controller,
				controllerDesc: controllerDesc{
					name:      metaObj.GetName(),
					namespace: metaObj.GetNamespace(),
					kind:      testcase.controller.GetObjectKind().GroupVersionKind().Kind,
				},
				pdClient: fakePDClient,
			}
			response := podAdmissionControl.admitDeleteTiKVPods(payload)
			g.Expect(response.Allowed).Should(Equal(testcase.allowed))
		})
	}
}

func newTiKVPod(ordinal int32, clusterPod bool) *core.Pod {
	pod := core.Pod{}
	pod.Labels = map[string]string{
		label.ComponentLabelKey: label.TiKVLabelVal,
	}
	if clusterPod {
		pod.Name = memberUtils.TikvPodName(tcName, ordinal)
		pod.Labels[label.NameLabelKey] = "tidb-cluster"
	} else {
		pod.Name = memberUtils.TiKVGroupPodName(tcName, ordinal)
		pod.Labels[label.NameLabelKey] = "tidb-cluster-group"
	}
	pod.Namespace = namespace
	return &pod
}

func newOwnerStatefulsetForTikv(upgrading bool) *apps.StatefulSet {
	sts := apps.StatefulSet{}
	sts.Spec.Replicas = func() *int32 { a := int32(3); return &a }()
	sts.Name = tiKVStsName
	sts.Namespace = namespace
	sts.Status.CurrentRevision = "1"
	sts.Status.UpdateRevision = "1"
	if upgrading {
		sts.Status.UpdateRevision = "2"
	}
	return &sts
}
