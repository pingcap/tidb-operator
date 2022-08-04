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

	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
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
	return nil
}

// ScaleOut scales out of the statefulset.
func (s *ticdcScaler) ScaleOut(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	obj, ok := meta.(runtime.Object)
	if !ok {
		klog.Errorf("cluster[%s/%s] can't convert to runtime.Object", meta.GetNamespace(), meta.GetName())
		return nil
	}
	klog.Infof("scaling out ticdc statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())
	skipReason, err := s.deleteDeferDeletingPVC(obj, v1alpha1.TiCDCMemberType, ordinal)
	if err != nil {
		return err
	} else if len(skipReason) != 1 || skipReason[ordinalPodName(v1alpha1.TiCDCMemberType, meta.GetName(), ordinal)] != skipReasonScalerPVCNotFound {
		// wait for all PVCs to be deleted
		return controller.RequeueErrorf("ticdc.ScaleOut, cluster %s/%s ready to scale out, skip reason %v, wait for next round", meta.GetNamespace(), meta.GetName(), skipReason)
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
		klog.Errorf("ticdcScaler.ScaleIn: failed to convert cluster %s/%s", meta.GetNamespace(), meta.GetName())
		return nil
	}
	pod, err := s.deps.PodLister.Pods(ns).Get(podName)
	if err != nil {
		return fmt.Errorf("ticdcScaler.ScaleIn: failed to get pods %s for cluster %s/%s, error: %v", podName, ns, tcName, err)
	}
	tc, _ := meta.(*v1alpha1.TidbCluster)

	err = gracefulDrainTiCDC(tc, s.deps.CDCControl, s.deps.PodControl, pod, ordinal, "ScaleIn")
	if err != nil {
		return err
	}
	klog.Infof("ticdcScaler.ScaleIn: %s has graceful shutdown in cluster %s/%s", podName, meta.GetNamespace(), meta.GetName())

	pvcs, err := util.ResolvePVCFromPod(pod, s.deps.PVCLister)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("ticdcScaler.ScaleIn: failed to get pvcs for pod %s/%s in tc %s/%s, error: %v", ns, pod.Name, ns, tcName, err)
	}
	for _, pvc := range pvcs {
		if err := addDeferDeletingAnnoToPVC(tc, pvc, s.deps.PVCControl); err != nil {
			return err
		}
	}

	setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
	return nil
}

func gracefulResignOwnerTiCDC(
	tc *v1alpha1.TidbCluster,
	cdcCtl controller.TiCDCControlInterface,
	podCtl controller.PodControlInterface,
	pod *corev1.Pod,
	ownerPodName string,
	ownerOrdinal int32,
	action string,
) error {
	isTimeout, err := checkTiCDCGracefulShutdownTimeout(tc, podCtl, pod, action)
	if err != nil {
		return err
	}
	if isTimeout {
		return nil
	}

	// To graceful resign the owner from the TiCDC pod, we need to
	//
	// 1. Remove ownership from the capture.
	resigned, err := cdcCtl.ResignOwner(tc, ownerOrdinal)
	if err != nil {
		return err
	}
	if !resigned {
		return controller.RequeueErrorf(
			"ticdc.%s: cluster %s/%s %s is still the owner, try resign owner again",
			action, tc.GetNamespace(), tc.GetName(), ownerPodName)
	}
	// 2. Wait for TiCDC cluster becomes healthy.
	healthy, err := cdcCtl.IsHealthy(tc, ownerOrdinal)
	if err != nil {
		return err
	}
	if !healthy {
		return controller.RequeueErrorf(
			"ticdc.%s: cluster %s/%s %s is resigned, wait for TiCDC cluster become healthy",
			action, tc.GetNamespace(), tc.GetName(), ownerPodName)
	}
	return nil
}

func gracefulDrainTiCDC(
	tc *v1alpha1.TidbCluster,
	cdcCtl controller.TiCDCControlInterface,
	podCtl controller.PodControlInterface,
	pod *corev1.Pod,
	ordinal int32,
	action string,
) error {
	isTimeout, err := checkTiCDCGracefulShutdownTimeout(tc, podCtl, pod, action)
	if err != nil {
		return err
	}
	if isTimeout {
		return nil
	}
	podName := pod.GetName()

	// To graceful shutdown a TiCDC pod, we need to
	//
	// 1. Remove ownership from the capture.
	resigned, err := cdcCtl.ResignOwner(tc, ordinal)
	if err != nil {
		return err
	}
	if !resigned {
		return controller.RequeueErrorf(
			"ticdc.%s: cluster %s/%s %s is still the owner, try resign owner again",
			action, tc.GetNamespace(), tc.GetName(), podName)
	}
	// 2. Drain the capture, move out all its tables.
	tableCount, retry, err := cdcCtl.DrainCapture(tc, ordinal)
	if err != nil {
		return err
	}
	if retry {
		return controller.RequeueErrorf(
			"ticdc.%s: cluster %s/%s %s needs to retry drain capture",
			action, tc.GetNamespace(), tc.GetName(), podName)
	}
	if tableCount != 0 {
		return controller.RequeueErrorf(
			"ticdc.%s: cluster %s/%s %s still has %d tables, wait draining",
			action, tc.GetNamespace(), tc.GetName(), podName, tableCount)
	}
	return nil
}

func checkTiCDCGracefulShutdownTimeout(
	tc *v1alpha1.TidbCluster,
	podCtl controller.PodControlInterface,
	pod *corev1.Pod,
	action string,
) (bool, error) {
	ns := tc.GetNamespace()
	podName := pod.GetName()
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	begin, ok := pod.Annotations[label.AnnTiCDCGracefulShutdownBeginTime]
	if ok {
		// Check graceful shutdown timeout.
		beginTime, err := time.Parse(time.RFC3339, begin)
		if err != nil {
			klog.Errorf("ticdc.%s: parse annotation:[%s] \"%s\" to time failed, skip graceful shutdown",
				action, label.AnnTiCDCGracefulShutdownBeginTime, begin)
			return true, nil
		}

		gracefulShutdownTimeout := tc.TiCDCGracefulShutdownTimeout()
		if time.Now().After(beginTime.Add(gracefulShutdownTimeout)) {
			klog.Infof("ticdc.%s: graceful shutdown timeout (threshold: %v) for Pod %s in cluster %s/%s",
				action, gracefulShutdownTimeout, podName, ns, tc.GetName())
			return true, nil
		}
		return false, nil
	}

	klog.Infof("ticdc.%s: begin graceful shutdown %s in cluster %s/%s",
		action, podName, ns, tc.GetName())

	// Set graceful shutdown begin time.
	now := time.Now().Format(time.RFC3339)
	pod.Annotations[label.AnnTiCDCGracefulShutdownBeginTime] = now
	_, err := podCtl.UpdatePod(tc, pod)
	if err != nil {
		klog.Errorf("ticdc.%s: failed to set pod %s in cluster %s/%s annotation %s to %s, error: %v",
			action, podName, ns, tc.GetName(), label.AnnTiCDCGracefulShutdownBeginTime, now, err)
		return false, err
	}
	klog.Infof("ticdc.%s: set pod %s in cluster %s/%s annotation %s to %s successfully",
		action, podName, ns, tc.GetName(), label.AnnTiCDCGracefulShutdownBeginTime, now)
	return false, nil
}
