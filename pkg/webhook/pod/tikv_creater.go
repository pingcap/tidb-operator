package pod

import (
	"fmt"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	admission "k8s.io/api/admission/v1beta1"
	core "k8s.io/api/core/v1"
	"k8s.io/klog"
	"strings"
)

func (pc *PodAdmissionControl) admitCreateTiKVPod(pod *core.Pod, tc *v1alpha1.TidbCluster, pdClient pdapi.PDClient) *admission.AdmissionResponse {

	name := pod.Name
	namespace := pod.Namespace

	stores, err := pdClient.GetStores()
	if err != nil {
		klog.Infof("failed to create pod[%s/%s],%v", namespace, name, err)
		return util.ARFail(err)
	}
	evictLeaderSchedulers, err := pdClient.GetEvictLeaderSchedulers()
	if err != nil {
		klog.Infof("failed to create pod[%s/%s],%v", namespace, name, err)
		return util.ARFail(err)
	}

	// if the pod which is going to be created already have a store and was in evictLeaderSchedulers,
	// we should end this evict leader
	for _, store := range stores.Stores {
		ip := strings.Split(store.Store.GetAddress(), ":")[0]
		podName := strings.Split(ip, ".")[0]
		if podName == name {
			for _, s := range evictLeaderSchedulers {
				id := strings.Split(s, "-")[3]
				if id == fmt.Sprintf("%d", store.Store.Id) {
					err := endEvictLeader(store, pdClient)
					if err != nil {
						klog.Infof("failed to create pod[%s/%s],%v", namespace, name, err)
						return util.ARFail(err)
					}
					break
				}
			}
			break
		}
	}

	return util.ARSuccess()
}
