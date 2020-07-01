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
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	operatorUtils "github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tidb-operator/pkg/webhook/util"
	admission "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

const (
	// EvictLeaderBeginTime is the key of evict Leader begin time
	EvictLeaderBeginTime = label.AnnEvictLeaderBeginTime
)

var (
	// EvictLeaderTimeout is the timeout limit of evict leader
	EvictLeaderTimeout time.Duration
)

func (pc *PodAdmissionControl) admitDeleteTiKVPods(payload *admitPayload) *admission.AdmissionResponse {
	pod := payload.pod
	pdClient := payload.pdClient
	name := pod.Name
	namespace := pod.Namespace
	ordinal, err := operatorUtils.GetOrdinalFromPodName(name)
	if err != nil {
		return util.ARFail(err)
	}
	controllerName := payload.controllerDesc.name
	controllerKind := payload.controllerDesc.kind

	// If the tikv pod is deleted by restarter, it is necessary to check former tikv restart status
	if _, exist := payload.pod.Annotations[label.AnnPodDeferDeleting]; exist {
		existed, err := checkFormerPodRestartStatus(pc.kubeCli, v1alpha1.TiKVMemberType, payload, ordinal)
		if err != nil {
			return util.ARFail(err)
		}
		if existed {
			return &admission.AdmissionResponse{
				Allowed: false,
			}
		}
	}

	storesInfo, err := pdClient.GetStores()
	if err != nil {
		return util.ARFail(err)
	}

	var storeInfo *pdapi.StoreInfo
	var expectedAddress string

	switch controllerKind {
	case v1alpha1.TiDBClusterKind:
		expectedAddress = fmt.Sprintf("%s.%s-tikv-peer.%s.svc:20160", name, controllerName, namespace)
	case v1alpha1.TiKVGroupKind:
		expectedAddress = fmt.Sprintf("%s.%s-tikv-group-peer.%s.svc:20160", name, controllerName, namespace)
	default:
		err := fmt.Errorf("tikv pod[%s/%s] controlled by unknown controllerKind[%s], forbid to delete", namespace, name, controllerKind)
		return util.ARFail(err)
	}

	existed := false
	for _, store := range storesInfo.Stores {
		if store.Store.Address == expectedAddress {
			storeInfo = store
			existed = true
			break
		}
	}

	if !existed || storeInfo == nil || storeInfo.Store == nil {
		klog.Infof("%s[%s/%s]'s tikv pod[%s/%s] can't be found store", controllerKind, namespace, controllerName, namespace, name)
		return pc.admitDeleteUselessTiKVPod(payload)
	}

	switch storeInfo.Store.StateName {
	case v1alpha1.TiKVStateTombstone:
		return pc.admitDeleteUselessTiKVPod(payload)
	case v1alpha1.TiKVStateOffline:
		return pc.rejectDeleteTiKVPod()
	case v1alpha1.TiKVStateDown:
		return pc.admitDeleteDownTikvPod(payload)
	case v1alpha1.TiKVStateUp:
		return pc.admitDeleteUpTiKVPod(payload, storeInfo, storesInfo)
	default:
		klog.Infof("unknown store state[%s] for tikv pod[%s/%s]", storeInfo.Store.StateName, namespace, name)
	}

	return &admission.AdmissionResponse{
		Allowed: false,
	}
}

func (pc *PodAdmissionControl) admitDeleteUselessTiKVPod(payload *admitPayload) *admission.AdmissionResponse {
	name := payload.pod.Name
	namespace := payload.pod.Namespace
	isInOrdinal, err := operatorUtils.IsPodOrdinalNotExceedReplicas(payload.pod, payload.ownerStatefulSet)
	if err != nil {
		return util.ARFail(err)
	}
	ordinal, err := operatorUtils.GetOrdinalFromPodName(name)
	if err != nil {
		return util.ARFail(err)
	}
	controllerName := payload.controllerDesc.name
	controllerKind := payload.controllerDesc.kind

	if !isInOrdinal {
		pvcName := operatorUtils.OrdinalPVCName(v1alpha1.TiKVMemberType, payload.ownerStatefulSet.Name, ordinal)
		pvc, err := pc.kubeCli.CoreV1().PersistentVolumeClaims(namespace).Get(pvcName, meta.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				pc.recorder.Event(payload.controller, corev1.EventTypeNormal, tikvScaleInReason, podDeleteEventMessage(name))
				return util.ARFail(err)
			}
			return util.ARFail(err)
		}
		err = addDeferDeletingToPVC(pvc, pc.kubeCli)
		if err != nil {
			klog.Infof("%s[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", controllerKind, namespace, controllerName, namespace, name, err)
			return util.ARFail(err)
		}
		pc.recorder.Event(payload.controller, corev1.EventTypeNormal, tikvScaleInReason, podDeleteEventMessage(name))
	}

	return util.ARSuccess()
}

func (pc *PodAdmissionControl) rejectDeleteTiKVPod() *admission.AdmissionResponse {
	return &admission.AdmissionResponse{
		Allowed: false,
	}
}

func (pc *PodAdmissionControl) admitDeleteUpTiKVPod(payload *admitPayload, store *pdapi.StoreInfo, storesInfo *pdapi.StoresInfo) *admission.AdmissionResponse {
	name := payload.pod.Name
	namespace := payload.pod.Namespace
	isInOrdinal, err := operatorUtils.IsPodOrdinalNotExceedReplicas(payload.pod, payload.ownerStatefulSet)
	if err != nil {
		return util.ARFail(err)
	}
	ordinal, err := operatorUtils.GetOrdinalFromPodName(name)
	if err != nil {
		return util.ARFail(err)
	}
	isUpgrading := operatorUtils.IsStatefulSetUpgrading(payload.ownerStatefulSet)
	controllerName := payload.controllerDesc.name
	controllerKind := payload.controllerDesc.kind

	if !isInOrdinal {
		err = payload.pdClient.DeleteStore(store.Store.Id)
		if err != nil {
			klog.Infof("%s[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", controllerKind, namespace, controllerName, namespace, name, err)
			return util.ARFail(err)
		}
		return &admission.AdmissionResponse{
			Allowed: false,
		}
	}
	var specReplicas int32
	if controllerKind == v1alpha1.TiDBClusterKind {
		tc, ok := payload.controller.(*v1alpha1.TidbCluster)
		if !ok {
			err := fmt.Errorf("tikv pod[%s/%s]'s controller is not tidbcluster,forbid to be deleted", namespace, name)
			return util.ARFail(err)
		}
		specReplicas = tc.Spec.TiKV.Replicas
	} else if controllerKind == v1alpha1.TiKVGroupKind {
		tg, ok := payload.controller.(*v1alpha1.TiKVGroup)
		if !ok {
			err := fmt.Errorf("tikv pod[%s/%s]'s controller is not tikvgroup,forbid to be deleted", namespace, name)
			return util.ARFail(err)
		}
		specReplicas = tg.Spec.Replicas
	} else {
		err := fmt.Errorf("tikv pod[%s/%s] has unknown controller[%s], forbid to be deleted", namespace, name, controllerKind)
		return util.ARFail(err)
	}

	if isUpgrading {
		err = checkFormerTiKVPodStatus(pc.kubeCli, payload.controllerDesc, ordinal, specReplicas, payload.ownerStatefulSet, storesInfo)
		if err != nil {
			klog.Infof("%s[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", controllerKind, namespace, controllerName, namespace, name, err)
			return util.ARFail(err)
		}
		return pc.admitDeleteUpTiKVPodDuringUpgrading(payload, store)
	}

	return util.ARSuccess()
}

func (pc *PodAdmissionControl) admitDeleteUpTiKVPodDuringUpgrading(payload *admitPayload, store *pdapi.StoreInfo) *admission.AdmissionResponse {

	name := payload.pod.Name
	namespace := payload.pod.Namespace
	metaController, ok := payload.controller.(meta.Object)
	if !ok {
		err := fmt.Errorf("tikv pod[%s/%s]'s controller is not a metav1.Object", namespace, name)
		return util.ARFail(err)
	}
	controllerName := metaController.GetName()
	controllerKind := payload.controller.GetObjectKind().GroupVersionKind().Kind

	_, evicting := payload.pod.Annotations[EvictLeaderBeginTime]
	if !evicting {
		err := beginEvictLeader(pc.kubeCli, store.Store.Id, payload.pod, payload.pdClient)
		if err != nil {
			klog.Infof("%s[%s/%s]'s tikv pod[%s/%s] failed to delete,%v", controllerKind, namespace, controllerName, namespace, name, err)
			return util.ARFail(err)
		}
		return &admission.AdmissionResponse{
			Allowed: false,
		}
	}

	if !isTiKVReadyToUpgrade(payload.pod, store) {
		return &admission.AdmissionResponse{
			Allowed: false,
		}
	}

	pc.recorder.Event(payload.controller, corev1.EventTypeNormal, tikvUpgradeReason, podDeleteEventMessage(name))
	return util.ARSuccess()
}

// When the target tikv's store is DOWN, we would not pass the deleting request during scale-in, otherwise it would cause
// the duplicated id problem for the newly tikv pod.
// Users should offline the target tikv into tombstone first, then scale-in it.
// In other cases, we would admit to delete the down tikv pod like upgrading.
func (pc *PodAdmissionControl) admitDeleteDownTikvPod(payload *admitPayload) *admission.AdmissionResponse {

	isInOrdinal, err := operatorUtils.IsPodOrdinalNotExceedReplicas(payload.pod, payload.ownerStatefulSet)
	if err != nil {
		return util.ARFail(err)
	}
	if !isInOrdinal {
		return pc.rejectDeleteTiKVPod()
	}
	name := payload.pod.Name
	isUpgrading := operatorUtils.IsStatefulSetUpgrading(payload.ownerStatefulSet)
	if isUpgrading {
		pc.recorder.Event(payload.controller, corev1.EventTypeNormal, tikvUpgradeReason, podDeleteEventMessage(name))
	}
	return util.ARSuccess()
}
