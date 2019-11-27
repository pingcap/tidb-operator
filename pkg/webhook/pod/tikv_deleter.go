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
	"github.com/pingcap/tidb-operator/pkg/label"
	"strings"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	admission "k8s.io/api/admission/v1"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/klog"
)

const (
	// EvictLeaderBeginTime is the key of evict Leader begin time
	EvictLeaderBeginTime = label.AnnEvictLeaderBeginTime
	// EvictLeaderTimeout is the timeout limit of evict leader
	EvictLeaderTimeout = 3 * time.Minute
)

func (pc *PodAdmissionControl) admitDeleteTiKVPods(pod *core.Pod, ownerStatefulSet *apps.StatefulSet, tc *v1alpha1.TidbCluster, pdClient pdapi.PDClient) *admission.AdmissionResponse {

	name := pod.Name
	namespace := pod.Namespace
	tcName := tc.Name

	isStoreExist := false
	storesInfo, err := pdClient.GetStores()
	if err != nil {
		return util.ARFail(err)
	}
	var storeInfo *pdapi.StoreInfo
	for _, info := range storesInfo.Stores {
		ip := strings.Split(info.Store.GetAddress(), ":")[0]
		podName := strings.Split(ip, ".")[0]
		if name == podName {
			isStoreExist = true
			storeInfo = info
			break
		}
	}

	isInOrdinal, err := operatorUtils.IsPodOrdinalNotExceedReplicas(pod, *ownerStatefulSet.Spec.Replicas)
	if err != nil {
		klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", namespace, tcName, namespace, name, err)
		return util.ARFail(err)
	}

	isUpgrading := IsStatefulSetUpgrading(ownerStatefulSet)

	if storeInfo == nil || !isStoreExist {
		return pc.admitDeleteUselessTiKVPod(isInOrdinal, pod, ownerStatefulSet, tc, pdClient)
	}

	if storeInfo.Store == nil {
		return util.ARFail(err)
	}

	switch storeInfo.Store.StateName {
	case v1alpha1.TiKVStateTombstone:
		return pc.admitDeleteUselessTiKVPod(isInOrdinal, pod, ownerStatefulSet, tc, pdClient)
	case v1alpha1.TiKVStateOffline:
		return pc.admitDeleteOfflineTiKVPod()
	case v1alpha1.TiKVStateDown:
		return pc.admitDeleteUselessTiKVPod(isInOrdinal, pod, ownerStatefulSet, tc, pdClient)
	case v1alpha1.TiKVStateUp:
		return pc.admitDeleteUpTiKVPod(isInOrdinal, isUpgrading, pod, ownerStatefulSet, tc, pdClient, storeInfo, storesInfo)
	default:
		klog.Infof("unknown store state[%s] for tikv pod[%/%s]", storeInfo.Store.StateName, namespace, name)
	}

	return &admission.AdmissionResponse{
		Allowed: false,
	}
}

func (pc *PodAdmissionControl) admitDeleteUselessTiKVPod(isInOrdinal bool, pod *core.Pod, ownerStatefulSet *apps.StatefulSet, tc *v1alpha1.TidbCluster, pdClient pdapi.PDClient) *admission.AdmissionResponse {

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

func (pc *PodAdmissionControl) admitDeleteOfflineTiKVPod() *admission.AdmissionResponse {
	return &admission.AdmissionResponse{
		Allowed: false,
	}
}

func (pc *PodAdmissionControl) admitDeleteUpTiKVPod(isInOrdinal, isUpgrading bool, pod *core.Pod, ownerStatefulSet *apps.StatefulSet, tc *v1alpha1.TidbCluster, pdClient pdapi.PDClient, store *pdapi.StoreInfo, storesInfo *pdapi.StoresInfo) *admission.AdmissionResponse {

	name := pod.Name
	namespace := pod.Namespace
	tcName := tc.Name

	ordinal, err := operatorUtils.GetOrdinalFromPodName(name)
	if err != nil {
		klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", namespace, tcName, namespace, name, err)
		return util.ARFail(err)
	}

	if !isInOrdinal {
		err = pdClient.DeleteStore(store.Store.Id)
		if err != nil {
			klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", namespace, tcName, namespace, name, err)
			return util.ARFail(err)
		}
		return &admission.AdmissionResponse{
			Allowed: false,
		}
	}

	if isUpgrading {
		err = checkFormerTiKVPodStatus(pc.podLister, tc, ordinal, *ownerStatefulSet.Spec.Replicas, storesInfo)
		if err != nil {
			klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", namespace, tcName, namespace, name, err)
			return util.ARFail(err)
		}
		return pc.admitDeleteUpTiKVPodDuringUpgrading(ordinal, pod, ownerStatefulSet, tc, pdClient, store)
	}

	return util.ARSuccess()
}

func (pc *PodAdmissionControl) admitDeleteUpTiKVPodDuringUpgrading(ordinal int32, pod *core.Pod, ownerStatefulSet *apps.StatefulSet, tc *v1alpha1.TidbCluster, pdClient pdapi.PDClient, store *pdapi.StoreInfo) *admission.AdmissionResponse {

	name := pod.Name
	namespace := pod.Namespace
	tcName := tc.Name

	_, evicting := pod.Annotations[EvictLeaderBeginTime]
	if !evicting {
		err := beginEvictLeader(pc.kubeCli, store.Store.Id, pod, pdClient)
		if err != nil {
			klog.Infof("tc[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", namespace, tcName, namespace, name, err)
			return util.ARFail(err)
		}
		return &admission.AdmissionResponse{
			Allowed: false,
		}
	}

	if !isTiKVReadyToUpgrade(pod, store) {
		return &admission.AdmissionResponse{
			Allowed: false,
		}
	}

	return util.ARSuccess()
}
