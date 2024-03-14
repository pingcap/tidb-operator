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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"
)

type tidbScaler struct {
	generalScaler
}

// NewTiDBScaler returns a TiDB Scaler.
func NewTiDBScaler(deps *controller.Dependencies) *tidbScaler {
	return &tidbScaler{generalScaler: generalScaler{deps: deps}}
}

// Scale scales in or out of the statefulset.
func (s *tidbScaler) Scale(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	scaling, _, _, _ := scaleOne(oldSet, newSet)
	if scaling > 0 {
		return s.ScaleOut(meta, oldSet, newSet)
	} else if scaling < 0 {
		return s.ScaleIn(meta, oldSet, newSet)
	}
	return nil
}

// ScaleOut scales out of the statefulset.
func (s *tidbScaler) ScaleOut(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	tc, ok := meta.(*v1alpha1.TidbCluster)
	if !ok {
		klog.Errorf("tidbScaler.ScaleOut: failed to convert cluster %s/%s, scale out will do nothing", meta.GetNamespace(), meta.GetName())
		return nil
	}

	scaleOutParallelism := tc.Spec.TiDB.GetScaleOutParallelism()
	_, ordinals, replicas, deleteSlots := scaleMulti(oldSet, newSet, scaleOutParallelism)
	klog.Infof("scaling out tidb statefulset %s/%s, ordinal: %v (replicas: %d, scale out parallelism: %d, delete slots: %v)",
		oldSet.Namespace, oldSet.Name, ordinals, replicas, scaleOutParallelism, deleteSlots.List())

	var (
		errs                         []error
		finishedOrdinals             = sets.NewInt32()
		updateReplicasAndDeleteSlots bool
	)
	for _, ordinal := range ordinals {
		err := s.scaleOutOne(tc, ordinal)
		if err != nil {
			errs = append(errs, err)
		} else {
			finishedOrdinals.Insert(ordinal)
			updateReplicasAndDeleteSlots = true
		}
	}
	if updateReplicasAndDeleteSlots {
		setReplicasAndDeleteSlotsByFinished(scalingOutFlag, newSet, oldSet, ordinals, finishedOrdinals)
	} else {
		resetReplicas(newSet, oldSet)
	}

	return errorutils.NewAggregate(errs)
}

func (s *tidbScaler) scaleOutOne(tc *v1alpha1.TidbCluster, ordinal int32) error {
	skipReason, err := s.deleteDeferDeletingPVC(tc, v1alpha1.TiDBMemberType, ordinal)
	if err != nil {
		return err
	} else if len(skipReason) != 1 || skipReason[ordinalPodName(
		v1alpha1.TiDBMemberType, tc.GetName(), ordinal)] != skipReasonScalerPVCNotFound {
		// wait for all PVCs to be deleted
		return controller.RequeueErrorf(
			"tidbScaler.ScaleOut, cluster %s/%s ready to scale out, skip reason %v, wait for next round",
			tc.GetNamespace(), tc.GetName(), skipReason)
	}
	return nil
}

// ScaleIn scales in of the statefulset.
func (s *tidbScaler) ScaleIn(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	scaleInTime := time.Now().Format(time.RFC3339Nano)
	tc, ok := meta.(*v1alpha1.TidbCluster)
	if !ok {
		klog.Errorf("tidbScaler.ScaleIn: failed to convert cluster %s/%s, scale in will do nothing", meta.GetNamespace(), meta.GetName())
		return nil
	}

	scaleInParallelism := tc.Spec.TiDB.GetScaleInParallelism()

	_, ordinals, replicas, deleteSlots := scaleMulti(oldSet, newSet, scaleInParallelism)
	klog.Infof("scaling in tidb statefulset %s/%s, ordinals: %v (replicas: %d, delete slots: %v), scaleInParallelism: %v",
		oldSet.Namespace, oldSet.Name, ordinals, replicas, deleteSlots.List(), scaleInParallelism)

	var (
		errs                         []error
		finishedOrdinals             = sets.NewInt32()
		updateReplicasAndDeleteSlots bool
	)
	for _, ordinal := range ordinals {
		err := s.scaleInOne(tc, ordinal, scaleInTime)
		if err != nil {
			errs = append(errs, err)
		} else {
			finishedOrdinals.Insert(ordinal)
			updateReplicasAndDeleteSlots = true
		}
	}

	if updateReplicasAndDeleteSlots {
		setReplicasAndDeleteSlotsByFinished(scalingInFlag, newSet, oldSet, ordinals, finishedOrdinals)
	} else {
		resetReplicas(newSet, oldSet)
	}
	return errorutils.NewAggregate(errs)
}

func (s *tidbScaler) scaleInOne(tc *v1alpha1.TidbCluster, ordinal int32, scaleInTime string) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	// We need to remove member from cluster before reducing statefulset replicas
	podName := ordinalPodName(v1alpha1.TiDBMemberType, tcName, ordinal)
	pod, err := s.deps.PodLister.Pods(ns).Get(podName)
	if err != nil {
		return fmt.Errorf("tidbScaler.ScaleIn: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tcName, err)
	}

	pvcs, err := util.ResolvePVCFromPod(pod, s.deps.PVCLister)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("tidbScaler.ScaleIn: failed to get pvcs for pod %s/%s in tc %s/%s, error: %s", ns, pod.Name, ns, tcName, err)
	}
	for _, pvc := range pvcs {
		if err := addDeferDeletingAnnoToPVC(tc, pvc, s.deps.PVCControl, scaleInTime); err != nil {
			return err
		}
	}
	return nil
}
