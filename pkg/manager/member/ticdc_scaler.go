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
	"time"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	apps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
)

type ticdcScaler struct {
	generalScaler
}

// NewTiCDCScaler returns a TiCDC Scaler.
func NewTiCDCScaler(deps *controller.Dependencies) *ticdcScaler {
	return &ticdcScaler{generalScaler: generalScaler{deps: deps}}
}

// Scale scales in or out of the statefulset.
func (s *ticdcScaler) Scale(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	scaling, _, _, _ := scaleOne(oldSet, newSet)
	if scaling > 0 {
		return s.ScaleOut(meta, oldSet, newSet)
	} else if scaling < 0 {
		return s.ScaleIn(meta, oldSet, newSet)
	}
	// we only sync auto scaler annotations when we are finishing syncing scaling
	return s.SyncAutoScalerAnn(meta, oldSet)
}

// ScaleOut scales out of the statefulset.
func (s *ticdcScaler) ScaleOut(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	obj, ok := meta.(runtime.Object)
	if !ok {
		return fmt.Errorf("cluster[%s/%s] can't conver to runtime.Object", meta.GetNamespace(), meta.GetName())
	}
	klog.Infof("scaling out ticdc statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())
	skipReason, err := s.deleteDeferDeletingPVC(obj, v1alpha1.TiCDCMemberType, ordinal)
	if err != nil {
		return err
	} else if len(skipReason) != 1 || skipReason[ordinalPodName(v1alpha1.TiCDCMemberType, meta.GetName(), ordinal)] != skipReasonScalerPVCNotFound {
		// wait for all PVCs to be deleted
		return controller.RequeueErrorf("ticdc.ScaleOut, cluster %s/%s ready to scale out, wait for next round", meta.GetNamespace(), meta.GetName())
	}
	setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
	return nil
}

// ScaleIn scales in of the statefulset.
func (s *ticdcScaler) ScaleIn(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	ns := meta.GetNamespace()
	tcName := meta.GetName()
	// NOW, we can only remove one member at a time when scaling in
	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)

	klog.Infof("scaling in ticdc statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())
	// We need to remove member from cluster before reducing statefulset replicas
	var podName string
	switch meta.(type) {
	case *v1alpha1.TidbCluster:
		podName = ordinalPodName(v1alpha1.TiCDCMemberType, tcName, ordinal)
	default:
		return fmt.Errorf("ticdcScaler.ScaleIn: failed to convert cluster %s/%s", meta.GetNamespace(), meta.GetName())
	}
	pod, err := s.deps.PodLister.Pods(ns).Get(podName)
	if err != nil {
		return fmt.Errorf("ticdcScaler.ScaleIn: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
	}

	// when scaling in TiCDC pods, we let the "capture info" in PD's etcd to be deleted automatically when shutting down the TiCDC process or after TTL expired.
	// When the TiCDC pod not found in TidbCluster status, there are two possible situations:
	// 1. TiCDC has already joined cluster but status not synced yet.
	//    In this situation return error to wait for another round for safety.
	// 2. TiCDC pod is not ready, such as in pending state.
	//    In this situation we should delete this TiCDC pod immediately to avoid blocking the subsequent operations.
	if !podutil.IsPodReady(pod) {
		safeTimeDeadline := pod.CreationTimestamp.Add(5 * s.deps.CLIConfig.ResyncDuration)
		if time.Now().Before(safeTimeDeadline) {
			// Wait for 5 resync periods to ensure that the following situation does not occur:
			//
			// The ticdc pod starts for a while, but has not synced its status, and then the pod becomes not ready.
			// Here we wait for 5 resync periods to ensure that the status of this ticdc pod has been synced.
			// After this period of time, if there is still no information about this ticdc in TidbCluster status,
			// then we can be sure that this ticdc has never been added to the tidb cluster.
			// So we can scale in this ticdc pod safely.
			resetReplicas(newSet, oldSet)
			return fmt.Errorf("TiCDC %s/%s is not ready, wait for 5 resync periods to sync its status", ns, podName)
		}
	}

	pvcs, err := util.ResolvePVCFromPod(pod, s.deps.PVCLister)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("ticdcScaler.ScaleIn: failed to get pvcs for pod %s/%s in tc %s/%s, error: %s", ns, pod.Name, ns, tcName, err)
	}
	tc, _ := meta.(*v1alpha1.TidbCluster)
	for _, pvc := range pvcs {
		if err := addDeferDeletingAnnoToPVC(tc, pvc, s.deps.PVCControl); err != nil {
			return err
		}
	}

	setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
	return nil
}

// SyncAutoScalerAnn would reclaim the auto-scaling out slots if the target pod is no longer existed
func (s *ticdcScaler) SyncAutoScalerAnn(meta metav1.Object, actual *apps.StatefulSet) error {
	tc, ok := meta.(*v1alpha1.TidbCluster)
	if !ok {
		return nil
	}

	currentScalingSlots := util.GetAutoScalingOutSlots(tc, v1alpha1.TiCDCMemberType)
	if currentScalingSlots.Len() < 1 {
		return nil
	}
	currentOrdinals := helper.GetPodOrdinals(tc.Spec.TiCDC.Replicas, actual)

	// reclaim the auto-scaling out slots if the target pod is no longer existed
	if !currentOrdinals.HasAll(currentScalingSlots.List()...) {
		reclaimedSlots := currentScalingSlots.Difference(currentOrdinals)
		currentScalingSlots = currentScalingSlots.Delete(reclaimedSlots.List()...)
		if currentScalingSlots.Len() < 1 {
			delete(tc.Annotations, label.AnnTiCDCAutoScalingOutOrdinals)
			return nil
		}
		v, err := util.Encode(currentScalingSlots.List())
		if err != nil {
			return err
		}
		tc.Annotations[label.AnnTiCDCAutoScalingOutOrdinals] = v
		return nil
	}
	return nil
}
