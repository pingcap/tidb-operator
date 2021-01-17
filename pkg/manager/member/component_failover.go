// Copyright 2021 PingCAP, Inc.
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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
)

// Failover is used to failover broken pd member
// If there are 3 PD members in a tidb cluster with 1 broken member pd-0, pdFailover will do failover in 3 rounds:
// 1. mark pd-0 as a failure Member with non-deleted state (MemberDeleted=false)
// 2. delete the failure member pd-0, and mark it deleted (MemberDeleted=true)
// 3. PD member manager will add the count of deleted failure members more replicas
// If the count of the failure PD member with the deleted state (MemberDeleted=true) is equal or greater than MaxFailoverCount, we will skip failover.
func ComponentFailover(context *ComponentContext) error {
	component := context.component

	switch component {
	case label.PDLabelVal:
		return componentPDFailover(context)
	case label.TiKVLabelVal:
		return componentTiKVFailover(context)
	case label.TiFlashLabelVal:
		return componentTiFlashFailover(context)
	case label.TiDBLabelVal:
		return componentTiDBFailover(context)
	}

	return nil
}

func componentPDFailover(context *ComponentContext) error {
	tc := context.tc

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if !tc.Status.PD.Synced {
		return fmt.Errorf("TidbCluster: %s/%s's pd status sync failed, can't failover", ns, tcName)
	}
	if tc.Status.PD.FailureMembers == nil {
		tc.Status.PD.FailureMembers = map[string]v1alpha1.PDFailureMember{}
	}

	inQuorum, healthCount := ComponentIsInQuorum(context)
	if !inQuorum {
		return fmt.Errorf("TidbCluster: %s/%s's pd cluster is not health: %d/%d, "+
			"replicas: %d, failureCount: %d, can't failover",
			ns, tcName, healthCount, tc.PDStsDesiredReplicas(), tc.Spec.PD.Replicas, len(tc.Status.PD.FailureMembers))
	}

	pdDeletedFailureReplicas := tc.GetPDDeletedFailureReplicas()
	if pdDeletedFailureReplicas >= *tc.Spec.PD.MaxFailoverCount {
		klog.Errorf("PD failover replicas (%d) reaches the limit (%d), skip failover", pdDeletedFailureReplicas, *tc.Spec.PD.MaxFailoverCount)
		return nil
	}

	notDeletedFailureReplicas := len(tc.Status.PD.FailureMembers) - int(pdDeletedFailureReplicas)

	// we can only failover one at a time
	if notDeletedFailureReplicas == 0 {
		return ComponentTryToMarkAPeerAsFailure(context)
	}

	return ComponentTryToDeleteAFailureMember(context)
}

func componentTiKVFailover(context *ComponentContext) error {
	tc := context.tc
	dependencies := context.dependencies

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	for storeID, store := range tc.Status.TiKV.Stores {
		podName := store.PodName
		if store.LastTransitionTime.IsZero() {
			continue
		}
		if ComponentIsPodDesired(context, podName) {
			// we should ignore the store record of deleted pod, otherwise the
			// record of deleted pod may be added back to failure stores
			// (before it enters into Offline/Tombstone state)
			continue
		}
		deadline := store.LastTransitionTime.Add(dependencies.CLIConfig.TiKVFailoverPeriod)
		exist := false
		for _, failureStore := range tc.Status.TiKV.FailureStores {
			if failureStore.PodName == podName {
				exist = true
				break
			}
		}
		if store.State == v1alpha1.TiKVStateDown && time.Now().After(deadline) && !exist {
			if tc.Status.TiKV.FailureStores == nil {
				tc.Status.TiKV.FailureStores = map[string]v1alpha1.TiKVFailureStore{}
			}
			if tc.Spec.TiKV.MaxFailoverCount != nil && *tc.Spec.TiKV.MaxFailoverCount > 0 {
				maxFailoverCount := *tc.Spec.TiKV.MaxFailoverCount
				if len(tc.Status.TiKV.FailureStores) >= int(maxFailoverCount) {
					klog.Warningf("%s/%s failure stores count reached the limit: %d", ns, tcName, tc.Spec.TiKV.MaxFailoverCount)
					return nil
				}
				tc.Status.TiKV.FailureStores[storeID] = v1alpha1.TiKVFailureStore{
					PodName:   podName,
					StoreID:   store.ID,
					CreatedAt: metav1.Now(),
				}
				msg := fmt.Sprintf("store[%s] is Down", store.ID)
				dependencies.Recorder.Event(tc, corev1.EventTypeWarning, unHealthEventReason, fmt.Sprintf(unHealthEventMsgPattern, "tikv", podName, msg))
			}
		}
	}

	return nil
}

func componentTiFlashFailover(context *ComponentContext) error {
	tc := context.tc
	dependencies := context.dependencies

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	for storeID, store := range tc.Status.TiFlash.Stores {
		podName := store.PodName
		if store.LastTransitionTime.IsZero() {
			continue
		}
		if ComponentIsPodDesired(context, podName) {
			// we should ignore the store record of deleted pod, otherwise the
			// record of deleted pod may be added back to failure stores
			// (before it enters into Offline/Tombstone state)
			continue
		}
		deadline := store.LastTransitionTime.Add(dependencies.CLIConfig.TiFlashFailoverPeriod)
		exist := false
		for _, failureStore := range tc.Status.TiFlash.FailureStores {
			if failureStore.PodName == podName {
				exist = true
				break
			}
		}
		if store.State == v1alpha1.TiKVStateDown && time.Now().After(deadline) && !exist {
			if tc.Status.TiFlash.FailureStores == nil {
				tc.Status.TiFlash.FailureStores = map[string]v1alpha1.TiKVFailureStore{}
			}
			if tc.Spec.TiFlash.MaxFailoverCount != nil && *tc.Spec.TiFlash.MaxFailoverCount > 0 {
				maxFailoverCount := *tc.Spec.TiFlash.MaxFailoverCount
				if len(tc.Status.TiFlash.FailureStores) >= int(maxFailoverCount) {
					klog.Warningf("%s/%s TiFlash failure stores count reached the limit: %d", ns, tcName, tc.Spec.TiFlash.MaxFailoverCount)
					return nil
				}
				tc.Status.TiFlash.FailureStores[storeID] = v1alpha1.TiKVFailureStore{
					PodName:   podName,
					StoreID:   store.ID,
					CreatedAt: metav1.Now(),
				}
				msg := fmt.Sprintf("store [%s] is Down", store.ID)
				dependencies.Recorder.Event(tc, corev1.EventTypeWarning, unHealthEventReason, fmt.Sprintf(unHealthEventMsgPattern, "tiflash", podName, msg))
			}
		}
	}
	return nil
}

func componentTiDBFailover(context *ComponentContext) error {
	tc := context.tc
	dependencies := context.dependencies

	if tc.Status.TiDB.FailureMembers == nil {
		tc.Status.TiDB.FailureMembers = map[string]v1alpha1.TiDBFailureMember{}
	}

	for _, tidbMember := range tc.Status.TiDB.Members {
		_, exist := tc.Status.TiDB.FailureMembers[tidbMember.Name]
		if exist && tidbMember.Health {
			delete(tc.Status.TiDB.FailureMembers, tidbMember.Name)
			klog.Infof("tidb failover: delete %s from tidb failoverMembers", tidbMember.Name)
		}
	}

	if tc.Spec.TiDB.MaxFailoverCount == nil || *tc.Spec.TiDB.MaxFailoverCount <= 0 {
		klog.Infof("tidb failover is disabled for %s/%s, skipped", tc.Namespace, tc.Name)
		return nil
	}

	maxFailoverCount := *tc.Spec.TiDB.MaxFailoverCount
	for _, tidbMember := range tc.Status.TiDB.Members {
		_, exist := tc.Status.TiDB.FailureMembers[tidbMember.Name]
		deadline := tidbMember.LastTransitionTime.Add(dependencies.CLIConfig.TiDBFailoverPeriod)
		if !tidbMember.Health && time.Now().After(deadline) && !exist {
			if len(tc.Status.TiDB.FailureMembers) >= int(maxFailoverCount) {
				klog.Warningf("the failover count reaches the limit (%d), no more failover pods will be created", maxFailoverCount)
				break
			}
			pod, err := dependencies.PodLister.Pods(tc.Namespace).Get(tidbMember.Name)
			if err != nil {
				return fmt.Errorf("tidbFailover.Failover: failed to get pods %s for cluster %s/%s, error: %s", tidbMember.Name, tc.GetNamespace(), tc.GetName(), err)
			}
			_, condition := podutil.GetPodCondition(&pod.Status, corev1.PodScheduled)
			if condition == nil || condition.Status != corev1.ConditionTrue {
				// if a member is unheathy because it's not scheduled yet, we
				// should not create failover pod for it
				klog.Warningf("pod %s/%s is not scheduled yet, skipping failover", pod.Namespace, pod.Name)
				continue
			}
			tc.Status.TiDB.FailureMembers[tidbMember.Name] = v1alpha1.TiDBFailureMember{
				PodName:   tidbMember.Name,
				CreatedAt: metav1.Now(),
			}
			msg := fmt.Sprintf("tidb[%s] is unhealthy", tidbMember.Name)
			dependencies.Recorder.Event(tc, corev1.EventTypeWarning, unHealthEventReason, fmt.Sprintf(unHealthEventMsgPattern, "tidb", tidbMember.Name, msg))
			break
		}
	}

	return nil
}

func ComponentRecover(context *ComponentContext) {
	tc := context.tc
	component := context.component

	switch component {
	case label.PDLabelVal:
		tc.Status.PD.FailureMembers = nil
	case label.TiKVLabelVal:
		tc.Status.TiKV.FailureStores = nil
	case label.TiFlashLabelVal:
		tc.Status.TiFlash.FailureStores = nil
	case label.TiDBLabelVal:
		tc.Status.TiDB.FailureMembers = nil
	}

	klog.Infof("%s failover: clearing failoverMembers, %s/%s", component, tc.GetNamespace(), tc.GetName())
}

func ComponentTryToMarkAPeerAsFailure(context *ComponentContext) error {
	tc := context.tc
	dependencies := context.dependencies

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	for pdName, pdMember := range tc.Status.PD.Members {
		if pdMember.LastTransitionTime.IsZero() {
			continue
		}
		podName := strings.Split(pdName, ".")[0]
		if !ComponentIsPodDesired(context, podName) {
			continue
		}

		if tc.Status.PD.FailureMembers == nil {
			tc.Status.PD.FailureMembers = map[string]v1alpha1.PDFailureMember{}
		}
		deadline := pdMember.LastTransitionTime.Add(dependencies.CLIConfig.PDFailoverPeriod)
		_, exist := tc.Status.PD.FailureMembers[pdName]

		if pdMember.Health || time.Now().Before(deadline) || exist {
			continue
		}

		ordinal, err := util.GetOrdinalFromPodName(podName)
		if err != nil {
			return err
		}
		pvcName := ordinalPVCName(v1alpha1.PDMemberType, controller.PDMemberName(tcName), ordinal)
		pvc, err := dependencies.PVCLister.PersistentVolumeClaims(ns).Get(pvcName)
		if err != nil {
			return fmt.Errorf("tryToMarkAPeerAsFailure: failed to get pvc %s for cluster %s/%s, error: %s", pvcName, ns, tcName, err)
		}

		msg := fmt.Sprintf("pd member[%s] is unhealthy", pdMember.ID)
		dependencies.Recorder.Event(tc, apiv1.EventTypeWarning, unHealthEventReason, fmt.Sprintf(unHealthEventMsgPattern, "pd", pdName, msg))

		// mark a peer member failed and return an error to skip reconciliation
		// note that status of tidb cluster will be updated always
		tc.Status.PD.FailureMembers[pdName] = v1alpha1.PDFailureMember{
			PodName:       podName,
			MemberID:      pdMember.ID,
			PVCUID:        pvc.UID,
			MemberDeleted: false,
			CreatedAt:     metav1.Now(),
		}
		return controller.RequeueErrorf("marking Pod: %s/%s pd member: %s as failure", ns, podName, pdMember.Name)
	}

	return nil
}

// tryToDeleteAFailureMember tries to delete a PD member and associated pod &
// pvc. If this succeeds, new pod & pvc will be created by Kubernetes.
// Note that this will fail if the kubelet on the node which failed pod was
// running on is not responding.
func ComponentTryToDeleteAFailureMember(context *ComponentContext) error {
	tc := context.tc
	dependencies := context.dependencies

	ns := tc.GetNamespace()
	tcName := tc.GetName()
	var failureMember *v1alpha1.PDFailureMember
	var failurePodName string
	var failurePdName string

	for pdName, pdMember := range tc.Status.PD.FailureMembers {
		if !pdMember.MemberDeleted {
			failureMember = &pdMember
			failurePodName = strings.Split(pdName, ".")[0]
			failurePdName = pdName
			break
		}
	}
	if failureMember == nil {
		return nil
	}

	memberID, err := strconv.ParseUint(failureMember.MemberID, 10, 64)
	if err != nil {
		return err
	}
	// invoke deleteMember api to delete a member from the pd cluster
	err = controller.GetPDClient(dependencies.PDControl, tc).DeleteMemberByID(memberID)
	if err != nil {
		klog.Errorf("pd failover: failed to delete member: %d, %v", memberID, err)
		return err
	}
	klog.Infof("pd failover: delete member: %d successfully", memberID)
	dependencies.Recorder.Eventf(tc, apiv1.EventTypeWarning, "PDMemberDeleted",
		"%s(%d) deleted from cluster", failurePodName, memberID)

	// The order of old PVC deleting and the new Pod creating is not guaranteed by Kubernetes.
	// If new Pod is created before old PVC deleted, new Pod will reuse old PVC.
	// So we must try to delete the PVC and Pod of this PD peer over and over,
	// and let StatefulSet create the new PD peer with the same ordinal, but don't use the tombstone PV
	pod, err := dependencies.PodLister.Pods(ns).Get(failurePodName)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("tryToDeleteAFailureMember: failed to get pods %s for cluster %s/%s, error: %s", failurePodName, ns, tcName, err)
	}

	ordinal, err := util.GetOrdinalFromPodName(failurePodName)
	if err != nil {
		return err
	}
	pvcName := ordinalPVCName(v1alpha1.PDMemberType, controller.PDMemberName(tcName), ordinal)
	pvc, err := dependencies.PVCLister.PersistentVolumeClaims(ns).Get(pvcName)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("tryToDeleteAFailureMember: failed to get pvc %s for cluster %s/%s, error: %s", pvcName, ns, tcName, err)
	}

	if pod != nil && pod.DeletionTimestamp == nil {
		err := dependencies.PodControl.DeletePod(tc, pod)
		if err != nil {
			return err
		}
	}
	if pvc != nil && pvc.DeletionTimestamp == nil && pvc.GetUID() == failureMember.PVCUID {
		err = dependencies.PVCControl.DeletePVC(tc, pvc)
		if err != nil {
			klog.Errorf("pd failover: failed to delete pvc: %s/%s, %v", ns, pvcName, err)
			return err
		}
		klog.Infof("pd failover: pvc: %s/%s successfully", ns, pvcName)
	}

	setMemberDeleted(tc, failurePdName)
	return nil
}

func ComponentIsPodDesired(context *ComponentContext, podName string) bool {
	ordinals := getComponentDesiredOrdinals(context)
	ordinal, err := util.GetOrdinalFromPodName(podName)
	if err != nil {
		klog.Errorf("unexpected pod name %q: %v", podName, err)
		return false
	}
	return ordinals.Has(ordinal)
}

func ComponentRemoveUndesiredFailures(context *ComponentContext) {
	tc := context.tc
	component := context.component

	if component == label.TiKVLabelVal {
		for key, failureStore := range tc.Status.TiKV.FailureStores {
			if !ComponentIsPodDesired(context, failureStore.PodName) {
				// If we delete the pods, e.g. by using advanced statefulset delete
				// slots feature. We should remove the record of undesired pods,
				// otherwise an extra replacement pod will be created.
				delete(tc.Status.TiKV.FailureStores, key)
			}
		}
	} else if component == label.TiFlashLabelVal {
		for key, failureStore := range tc.Status.TiFlash.FailureStores {
			if !ComponentIsPodDesired(context, failureStore.PodName) {
				// If we delete the pods, e.g. by using advanced statefulset delete
				// slots feature. We should remove the record of undesired pods,
				// otherwise an extra replacement pod will be created.
				delete(tc.Status.TiFlash.FailureStores, key)
			}
		}
	}
}

func ComponentSetMemberDeleted(context *ComponentContext, pdName string) {
	tc := context.tc

	failureMember := tc.Status.PD.FailureMembers[pdName]
	failureMember.MemberDeleted = true
	tc.Status.PD.FailureMembers[pdName] = failureMember
	klog.Infof("pd failover: set pd member: %s/%s deleted", tc.GetName(), pdName)
}

func ComponentIsInQuorum(context *ComponentContext) (bool, int) {
	// context deserialization
	tc := context.tc
	dependencies := context.dependencies

	healthCount := 0
	for podName, pdMember := range tc.Status.PD.Members {
		if pdMember.Health {
			healthCount++
		} else {
			dependencies.Recorder.Eventf(tc, apiv1.EventTypeWarning, "PDMemberUnhealthy",
				"%s(%s) is unhealthy", podName, pdMember.ID)
		}
	}
	for _, pdMember := range tc.Status.PD.PeerMembers {
		if pdMember.Health {
			healthCount++
		}
	}
	return healthCount > (len(tc.Status.PD.Members)+len(tc.Status.PD.PeerMembers))/2, healthCount
}
