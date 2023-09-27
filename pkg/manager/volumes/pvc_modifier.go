// Copyright 2022 PingCAP, Inc.
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

package volumes

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errutil "k8s.io/apimachinery/pkg/util/errors"
	klog "k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/utils"
)

const (
	annoKeyPVCSpecRevision     = "spec.tidb.pingcap.com/revision"
	annoKeyPVCSpecStorageClass = "spec.tidb.pingcap.com/storage-class"
	annoKeyPVCSpecStorageSize  = "spec.tidb.pingcap.com/storage-size"

	annoKeyPVCStatusRevision     = "status.tidb.pingcap.com/revision"
	annoKeyPVCStatusStorageClass = "status.tidb.pingcap.com/storage-class"
	annoKeyPVCStatusStorageSize  = "status.tidb.pingcap.com/storage-size"

	annoKeyPVCLastTransitionTimestamp = "status.tidb.pingcap.com/last-transition-timestamp"

	defaultModifyWaitingDuration = time.Minute * 1
)

type PVCModifierInterface interface {
	Sync(tc *v1alpha1.TidbCluster) error
}

type pvcModifier struct {
	deps  *controller.Dependencies
	pm    PodVolumeModifier
	utils *volCompareUtils
}

func NewPVCModifier(deps *controller.Dependencies) PVCModifierInterface {
	return &pvcModifier{
		deps:  deps,
		pm:    NewPodVolumeModifier(deps),
		utils: newVolCompareUtils(deps),
	}
}

func (p *pvcModifier) Sync(tc *v1alpha1.TidbCluster) error {
	components := tc.AllComponentStatus()
	errs := []error{}

	for _, comp := range components {
		ctx, err := p.utils.BuildContextForTC(tc, comp)
		if err != nil {
			errs = append(errs, fmt.Errorf("build ctx used by modifier for %s/%s:%s failed: %w", tc.Namespace, tc.Name, comp.MemberType(), err))
			continue
		}

		err = p.modifyVolumes(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("modify volumes for %s failed: %w", ctx.ComponentID(), err))
			continue
		}
	}

	return errutil.NewAggregate(errs)
}

func (p *pvcModifier) modifyVolumes(ctx *componentVolumeContext) error {
	if ctx.status.GetPhase() != v1alpha1.NormalPhase {
		return fmt.Errorf("component phase is not Normal")
	}

	if err := p.tryToRecreateSTS(ctx); err != nil {
		return err
	}

	if err := p.tryToModifyPVC(ctx); err != nil {
		return err
	}

	return nil
}

func (p *pvcModifier) tryToRecreateSTS(ctx *componentVolumeContext) error {
	ns := ctx.sts.Namespace
	name := ctx.sts.Name

	// Modifier does not support new volumes, trigger error if so and return.
	isSynced, err := p.utils.IsStatefulSetSynced(ctx, ctx.sts)
	if err != nil {
		return fmt.Errorf("change in number of volumes is not supported. For pd, tikv, tidb supported with VolumeReplacing feature. Reconciliation is blocked due to : %s", err.Error())
	}
	if isSynced {
		return nil
	}

	if utils.StatefulSetIsUpgrading(ctx.sts) {
		return fmt.Errorf("component sts %s/%s is upgrading", ctx.sts.Name, ctx.sts.Namespace)
	}

	orphan := metav1.DeletePropagationOrphan
	if err := p.deps.KubeClientset.AppsV1().StatefulSets(ns).Delete(ctx, name, metav1.DeleteOptions{PropagationPolicy: &orphan}); err != nil {
		return fmt.Errorf("delete sts %s/%s for component %s failed: %s", ns, name, ctx.ComponentID(), err)
	}

	klog.Infof("recreate statefulset %s/%s for component %s", ns, name, ctx.ComponentID())

	// component manager will create the sts in next reconciliation
	return nil
}

func (p *pvcModifier) tryToModifyPVC(ctx *componentVolumeContext) error {
	for _, pod := range ctx.pods {
		actual, err := p.pm.GetActualVolumes(pod, ctx.desiredVolumes)
		if err != nil {
			return err
		}

		isNeed := p.pm.ShouldModify(actual)

		if ctx.shouldEvict {
			// ensure leader eviction is finished and tikv store is up
			if !isNeed {
				if err := p.endEvictLeader(ctx.tc, pod); err != nil {
					return err
				}

				if !isTiKVStoreUp(ctx.tc, pod) {
					return fmt.Errorf("wait for tikv store %s/%s up", pod.Namespace, pod.Name)
				}

				continue
			}

			// try to evict leader if need to modify
			isEvicted := isLeaderEvictedOrTimeout(ctx.tc, pod)
			if !isEvicted {
				// do not evict leader when resizing PVC (increasing size)
				// as if the storage size is not enough, the leader eviction will be blocked (never finished)
				if !skipEvictLeaderForSizeModify(actual) {
					if ensureTiKVLeaderEvictionCondition(ctx.tc, metav1.ConditionTrue) {
						// return to sync tc
						return fmt.Errorf("try to evict leader for tidbcluster %s/%s", ctx.tc.Namespace, ctx.tc.Name)
					}
					if err := p.evictLeader(ctx.tc, pod); err != nil {
						return err
					}

					return fmt.Errorf("wait for leader eviction of %s/%s completed", pod.Namespace, pod.Name)
				} else {
					klog.Infof("skip evicting leader for %s/%s as the storage size is changing", pod.Namespace, pod.Name)
				}
			}
		}

		if !isNeed {
			continue
		}

		if err := p.pm.Modify(actual); err != nil {
			return err
		}
	}

	if ctx.shouldEvict {
		if ensureTiKVLeaderEvictionCondition(ctx.tc, metav1.ConditionFalse) {
			// return to sync tc
			return fmt.Errorf("try to stop evicting leader for tidbcluster %s/%s", ctx.tc.Namespace, ctx.tc.Name)
		}
	}

	return nil
}

// skip evict leader if the storage size should be modified or is in modifying phase
func skipEvictLeaderForSizeModify(actual []ActualVolume) bool {
	for _, vol := range actual {
		if vol.PVC == nil || vol.Desired == nil {
			continue
		}

		annoStatusSize, ok := vol.PVC.Annotations[annoKeyPVCStatusStorageSize]
		if ok {
			// modified by the PVC Modifier before (with status size annotation)
			if annoStatusSize == vol.Desired.Size.String() {
				continue // already up to date, no need to modify size
			}
			return true // need to modify size
		}

		// not modified by the PVC modifier before (without status size annotation)
		quantity := vol.GetStorageSize()
		statusSize := quantity.String()
		if statusSize == vol.Desired.Size.String() {
			// special case: skip evict leader (again) as the PVC is in modfiying phase (and the status size annotation is not set yet)
			if vol.Phase == VolumePhaseModifying {
				return true
			}
			// already modified, in fact, the status size annotation should already be set
			continue
		}
		// need to modify size
		return true
	}
	return false
}

func ensureTiKVLeaderEvictionCondition(tc *v1alpha1.TidbCluster, status metav1.ConditionStatus) bool {
	if meta.IsStatusConditionPresentAndEqual(tc.Status.TiKV.Conditions, v1alpha1.ConditionTypeLeaderEvicting, status) {
		return false
	}

	var reason, message string

	switch status {
	case metav1.ConditionTrue:
		reason = "ModifyVolume"
		message = "Evicting leader for volume modification"
	case metav1.ConditionFalse:
		reason = "NoLeaderEviction"
		message = "Leader can be scheduled to all nodes"
	case metav1.ConditionUnknown:
		reason = "Unknown"
		message = "Leader eviction status is unknown"
	}
	cond := metav1.Condition{
		Type:    v1alpha1.ConditionTypeLeaderEvicting,
		Status:  status,
		Reason:  reason,
		Message: message,
	}

	meta.SetStatusCondition(&tc.Status.TiKV.Conditions, cond)
	return true
}

func isTiKVStoreUp(tc *v1alpha1.TidbCluster, pod *corev1.Pod) bool {
	// wait store to be Up
	for _, store := range tc.Status.TiKV.Stores {
		if store.PodName == pod.Name && store.State != v1alpha1.TiKVStateUp {
			return false
		}
	}

	return true
}

func isLeaderEvictionFinished(tc *v1alpha1.TidbCluster, pod *corev1.Pod) bool {
	if _, exist := tc.Status.TiKV.EvictLeader[pod.Name]; exist {
		return false
	}

	return true

}

func isLeaderEvicting(pod *corev1.Pod) bool {
	_, exist := pod.Annotations[v1alpha1.EvictLeaderAnnKeyForResize]
	return exist
}

func (p *pvcModifier) evictLeader(tc *v1alpha1.TidbCluster, pod *corev1.Pod) error {
	if isLeaderEvicting(pod) {
		return nil
	}
	pod = pod.DeepCopy()

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	pod.Annotations[v1alpha1.EvictLeaderAnnKeyForResize] = v1alpha1.EvictLeaderValueNone
	if _, err := p.deps.KubeClientset.CoreV1().Pods(pod.Namespace).Update(context.TODO(), pod, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("add leader eviction annotation to pod %s/%s failed: %s", pod.Namespace, pod.Name, err)
	}

	return nil
}

func (p *pvcModifier) endEvictLeader(tc *v1alpha1.TidbCluster, pod *corev1.Pod) error {
	if isLeaderEvicting(pod) {
		pod = pod.DeepCopy()

		delete(pod.Annotations, v1alpha1.EvictLeaderAnnKeyForResize)
		if _, err := p.deps.KubeClientset.CoreV1().Pods(pod.Namespace).Update(context.TODO(), pod, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("delete leader eviction annotation to pod %s/%s failed: %s", pod.Namespace, pod.Name, err)
		}
		// Return an error once after deleting annotation, to give pod_control a chance to stop the eviction.
		// (but do not block / re-check in case the eviction here maybe manually requested by user via annotation)
		return fmt.Errorf("wait for leader eviction of %s/%s finished", pod.Namespace, pod.Name)
	}

	return nil
}
