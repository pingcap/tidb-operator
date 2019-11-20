package pod

import (
	"strconv"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	"k8s.io/api/admission/v1beta1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/klog"
)

const (
	// EvictLeaderBeginTime is the key of evict Leader begin time
	EvictLeaderBeginTime = "evictLeaderBeginTime"
	// EvictLeaderTimeout is the timeout limit of evict leader
	EvictLeaderTimeout = 3 * time.Minute
)

func (pc *PodAdmissionControl) admitDeleteTiKVPods(pod *core.Pod, ownerStatefulSet *apps.StatefulSet, tc *v1alpha1.TidbCluster, pdClient pdapi.PDClient) *v1beta1.AdmissionResponse {

	name := pod.Name
	namespace := pod.Namespace
	tcName := tc.Name

	isStoreExist := false
	var store v1alpha1.TiKVStore
	for _, info := range tc.Status.TiKV.Stores {
		if name == info.PodName {
			isStoreExist = true
			store = info
			break
		}
	}

	isInOrdinal, err := operatorUtils.IsPodOrdinalNotExceedReplicas(pod, *ownerStatefulSet.Spec.Replicas)
	if err != nil {
		klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", namespace, tcName, namespace, name, err)
		return util.ARFail(err)
	}

	isUpgrading := IsStatefulSetUpgrading(ownerStatefulSet)

	if !isStoreExist {
		return pc.admitDeleteNonStoreTiKVPod(isInOrdinal, pod, ownerStatefulSet, tc, pdClient)
	}

	switch store.State {
	case v1alpha1.TiKVStateTombstone:
		return pc.admitDeleteTombStoneTiKVPod(isInOrdinal, pod, ownerStatefulSet, tc, pdClient)
	case v1alpha1.TiKVStateOffline:
		return pc.admitDeleteOfflineTiKVPod()
	case v1alpha1.TiKVStateDown:
		return pc.admitDeleteDownTiKVPod(isInOrdinal, pod, ownerStatefulSet, tc, pdClient)
	case v1alpha1.TiKVStateUp:
		return pc.admitDeleteUpTiKVPod(isInOrdinal, isUpgrading, pod, ownerStatefulSet, tc, pdClient, store)
	}

	return &v1beta1.AdmissionResponse{
		Allowed: false,
	}
}

func (pc *PodAdmissionControl) admitDeleteNonStoreTiKVPod(isInOrdinal bool, pod *core.Pod, ownerStatefulSet *apps.StatefulSet, tc *v1alpha1.TidbCluster, pdClient pdapi.PDClient) *v1beta1.AdmissionResponse {

	name := pod.Name
	namespace := pod.Namespace
	tcName := tc.Name
	ordinal, err := operatorUtils.GetOrdinalFromPodName(name)
	if err != nil {
		return util.ARFail(err)
	}
	setName := ownerStatefulSet.Name

	if !isInOrdinal {
		err = addDeferDeletingToPVC(v1alpha1.TiKVMemberType, pc, tc, setName, namespace, ordinal)
		if err != nil {
			klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", namespace, tcName, namespace, name, err)
			return util.ARFail(err)
		}
	}

	return util.ARSuccess()
}

func (pc *PodAdmissionControl) admitDeleteTombStoneTiKVPod(isInOrdinal bool, pod *core.Pod, ownerStatefulSet *apps.StatefulSet, tc *v1alpha1.TidbCluster, pdClient pdapi.PDClient) *v1beta1.AdmissionResponse {

	name := pod.Name
	namespace := pod.Namespace
	tcName := tc.Name
	ordinal, err := operatorUtils.GetOrdinalFromPodName(name)
	if err != nil {
		klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", namespace, tcName, namespace, name, err)
		return util.ARFail(err)
	}
	setName := ownerStatefulSet.Name
	if !isInOrdinal {
		err = addDeferDeletingToPVC(v1alpha1.TiKVMemberType, pc, tc, setName, namespace, ordinal)
		if err != nil {
			klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", namespace, tcName, namespace, name, err)
			return util.ARFail(err)
		}
	}
	return util.ARSuccess()
}

func (pc *PodAdmissionControl) admitDeleteOfflineTiKVPod() *v1beta1.AdmissionResponse {
	return &v1beta1.AdmissionResponse{
		Allowed: false,
	}
}

func (pc *PodAdmissionControl) admitDeleteDownTiKVPod(isInOrdinal bool, pod *core.Pod, ownerStatefulSet *apps.StatefulSet, tc *v1alpha1.TidbCluster, pdClient pdapi.PDClient) *v1beta1.AdmissionResponse {

	name := pod.Name
	namespace := pod.Namespace
	tcName := tc.Name
	ordinal, err := operatorUtils.GetOrdinalFromPodName(name)
	if err != nil {
		klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", namespace, tcName, namespace, name, err)
		return util.ARFail(err)
	}
	setName := ownerStatefulSet.Name
	if !isInOrdinal {
		err = addDeferDeletingToPVC(v1alpha1.TiKVMemberType, pc, tc, setName, namespace, ordinal)
		if err != nil {
			klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", namespace, tcName, namespace, name, err)
			return util.ARFail(err)
		}
	}
	return util.ARSuccess()
}

func (pc *PodAdmissionControl) admitDeleteUpTiKVPod(isInOrdinal, isUpgrading bool, pod *core.Pod, ownerStatefulSet *apps.StatefulSet, tc *v1alpha1.TidbCluster, pdClient pdapi.PDClient, store v1alpha1.TiKVStore) *v1beta1.AdmissionResponse {

	name := pod.Name
	namespace := pod.Namespace
	tcName := tc.Name

	ordinal, err := operatorUtils.GetOrdinalFromPodName(name)
	if err != nil {
		klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", namespace, tcName, namespace, name, err)
		return util.ARFail(err)
	}

	if !isInOrdinal {
		id, err := strconv.ParseUint(store.ID, 10, 64)
		if err != nil {
			klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", namespace, tcName, namespace, name, err)
			return util.ARFail(err)
		}
		err = pdClient.DeleteStore(id)
		if err != nil {
			klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", namespace, tcName, namespace, name, err)
			return util.ARFail(err)
		}
		return &v1beta1.AdmissionResponse{
			Allowed: false,
		}
	}

	if isUpgrading {
		err = checkFormerTiKVPodStatus(pc.podLister, tc, ordinal, *ownerStatefulSet.Spec.Replicas)
		if err != nil {
			klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", namespace, tcName, namespace, name, err)
			return util.ARFail(err)
		}
		return pc.admitDeleteUPTiKVPodDuringUpgrading(ordinal, pod, ownerStatefulSet, tc, pdClient, store)
	}

	return util.ARSuccess()
}

func (pc *PodAdmissionControl) admitDeleteUPTiKVPodDuringUpgrading(ordinal int32, pod *core.Pod, ownerStatefulSet *apps.StatefulSet, tc *v1alpha1.TidbCluster, pdClient pdapi.PDClient, store v1alpha1.TiKVStore) *v1beta1.AdmissionResponse {

	name := pod.Name
	namespace := pod.Namespace
	tcName := tc.Name
	storeId, err := strconv.ParseUint(store.ID, 10, 64)
	if err != nil {
		klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", namespace, tcName, namespace, name, err)
		return util.ARFail(err)
	}

	_, evicting := pod.Annotations[EvictLeaderBeginTime]
	if !evicting {
		err := beginEvictLeader(pc.kubeCli, tc, storeId, pod, pdClient)
		if err != nil {
			klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", namespace, tcName, namespace, name, err)
			return util.ARFail(err)
		}
	}

	if !isTiKVReadyToUpgrade(pod, store) {
		return &v1beta1.AdmissionResponse{
			Allowed: false,
		}
	}
	return util.ARSuccess()
}
