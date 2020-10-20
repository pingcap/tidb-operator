package pod

import (
	"strconv"
	"testing"

	"errors"

	. "github.com/onsi/gomega"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	admission "k8s.io/api/admission/v1beta1"
	core "k8s.io/api/core/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func TestAdmitCreateTiKVPod(t *testing.T) {
	g := NewGomegaWithT(t)
	kubeCli := kubefake.NewSimpleClientset()
	pod := &core.Pod{}
	pod.Namespace = "ns"
	pod.Name = "name"
	_, err := kubeCli.CoreV1().Pods(pod.Namespace).Create(pod)
	g.Expect(err).Should(BeNil())
	pdClient := pdapi.NewFakePDClient()

	var resp *admission.AdmissionResponse

	// success if tikv is not bootstrapped
	pdClient.AddReaction(pdapi.GetStoresActionType, func(action *pdapi.Action) (interface{}, error) {
		return nil, errors.New(tikvNotBootstrapped + "\n")
	})
	resp = admitCreateTiKVPod(pod, pdClient)
	g.Expect(resp.Allowed).Should(BeTrue())

	// test should end evict leader and success
	storeID := 1
	pdClient.AddReaction(pdapi.GetStoresActionType, func(action *pdapi.Action) (interface{}, error) {
		store := &pdapi.StoreInfo{
			Store: &pdapi.MetaStore{
				Store: &metapb.Store{
					Id:      uint64(storeID),
					Address: pod.Name + ":8080",
				},
			},
			Status: &pdapi.StoreStatus{LeaderCount: 1},
		}
		return &pdapi.StoresInfo{
			Count:  1,
			Stores: []*pdapi.StoreInfo{store},
		}, nil
	})
	pdClient.AddReaction(pdapi.GetEvictLeaderSchedulersActionType, func(action *pdapi.Action) (interface{}, error) {
		return []string{"a-b-c-" + strconv.Itoa(storeID)}, nil
	})
	endEvictLeader := false
	pdClient.AddReaction(pdapi.EndEvictLeaderActionType, func(action *pdapi.Action) (interface{}, error) {
		endEvictLeader = true
		return nil, nil
	})

	resp = admitCreateTiKVPod(pod, pdClient)
	g.Expect(resp.Allowed).Should(BeTrue())
	g.Expect(endEvictLeader).Should(BeTrue())
}
