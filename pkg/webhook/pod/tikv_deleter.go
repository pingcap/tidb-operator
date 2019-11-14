package pod

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	"k8s.io/api/admission/v1beta1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"strconv"
	"time"
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
		return &v1beta1.AdmissionResponse{
			Allowed: false,
		}
	case v1alpha1.TiKVStateDown:
		return pc.admitDeleteDownTiKVPod(isInOrdinal, pod, ownerStatefulSet, tc, pdClient, store)
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

	if isInOrdinal {
		klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] is inOrdinal and nonStore,failed to delete it", namespace, tcName, namespace, name)
		return &v1beta1.AdmissionResponse{
			Allowed: false,
		}
	}

	if podutil.IsPodReady(pod) {
		klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] is Ready and nonStore,failed to delete it", namespace, tcName, namespace, name)
		return &v1beta1.AdmissionResponse{
			Allowed: false,
		}
	}

	safeTimeDeadline := pod.CreationTimestamp.Add(5 * RessyncDuration)
	if time.Now().Before(safeTimeDeadline) {
		// Wait for 5 resync periods to ensure that the following situation does not occur:
		//
		// The tikv pod starts for a while, but has not synced its status, and then the pod becomes not ready.
		// Here we wait for 5 resync periods to ensure that the status of this tikv pod has been synced.
		// After this period of time, if there is still no information about this tikv in TidbCluster status,
		// then we can be sure that this tikv has never been added to the tidb cluster.
		// So we can scale in this tikv pod safely.
		klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] is not Ready and nonStore,still need to be check,failed to delete it", namespace, tcName, namespace, name)
		return &v1beta1.AdmissionResponse{
			Allowed: false,
		}
	}

	err = addDeferDeletingToPVC(v1alpha1.TiKVMemberType, pc, tc, setName, namespace, ordinal)
	if err != nil {
		klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", namespace, tcName, namespace, name, err)
		return util.ARFail(err)
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

func (pc *PodAdmissionControl) admitDeleteDownTiKVPod(isInOrdinal bool, pod *core.Pod, ownerStatefulSet *apps.StatefulSet, tc *v1alpha1.TidbCluster, pdClient pdapi.PDClient, store v1alpha1.TiKVStore) *v1beta1.AdmissionResponse {

	name := pod.Name
	namespace := pod.Namespace
	tcName := tc.Name

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
	}

	return &v1beta1.AdmissionResponse{
		Allowed: false,
	}
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

	err = checkFormerTiKVPodStatus(pc.podLister, tc, ordinal, *ownerStatefulSet.Spec.Replicas)
	if err != nil {
		klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", namespace, tcName, namespace, name, err)
		return util.ARFail(err)
	}

	_, evicting := pod.Annotations[EvictLeaderBeginTime]
	if !evicting {
		err := pc.beginEvictLeader(tc, storeId, pod, pdClient)
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
	err = pc.endEvictLeader(tc, ordinal, store, pdClient)
	if err != nil {
		klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", namespace, tcName, namespace, name, err)
		return util.ARFail(err)
	}

	return util.ARSuccess()
}

func (pc *PodAdmissionControl) beginEvictLeader(tc *v1alpha1.TidbCluster, storeID uint64, pod *core.Pod, pdClient pdapi.PDClient) error {

	name := pod.Name
	namespace := pod.Namespace

	err := pdClient.BeginEvictLeader(storeID)
	if err != nil {
		klog.Errorf("tikv upgrader: failed to begin evict leader: %d, %s/%s, %v",
			storeID, namespace, name, err)
		return err
	}

	klog.Infof("tikv upgrader: begin evict leader: %d, %s/%s successfully", storeID, namespace, name)
	err = addEvictLeaderAnnotation(pc, pod)
	if err != nil {
		return err
	}
	return nil
}

func (pc *PodAdmissionControl) endEvictLeader(tc *v1alpha1.TidbCluster, ordinal int32, store v1alpha1.TiKVStore, pdClient pdapi.PDClient) error {

	storeID, err := strconv.ParseUint(store.ID, 10, 64)
	if err != nil {
		return err
	}

	err = pdClient.EndEvictLeader(storeID)
	if err != nil {
		klog.Errorf("tikv upgrader: failed to end evict leader storeID: %d ordinal: %d, %v", storeID, ordinal, err)
		return err
	}
	klog.Infof("tikv upgrader: end evict leader storeID: %d ordinal: %d successfully", storeID, ordinal)
	return nil
}
