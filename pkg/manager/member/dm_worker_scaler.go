// Copyright 2020 PingCAP, Inc.
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

	"github.com/pingcap/tidb-operator/pkg/label"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

type workerScaler struct {
	generalScaler
}

// NewWorkerScaler returns a DMScaler
func NewWorkerScaler(deps *controller.Dependencies) Scaler {
	return &workerScaler{
		generalScaler: generalScaler{
			deps: deps,
		},
	}
}

func (wsd *workerScaler) Scale(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	scaling, _, _, _ := scaleOne(oldSet, newSet)
	if scaling > 0 {
		return wsd.ScaleOut(meta, oldSet, newSet)
	} else if scaling < 0 {
		return wsd.ScaleIn(meta, oldSet, newSet)
	}
	return wsd.SyncAutoScalerAnn(meta, oldSet)
}

func (wsd *workerScaler) ScaleOut(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	dc, ok := meta.(*v1alpha1.DMCluster)
	if !ok {
		return nil
	}

	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	ns := dc.GetNamespace()
	dcName := dc.GetName()

	klog.Infof("scaling out dm-worker statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())
	_, err := wsd.deleteDeferDeletingPVC(dc, oldSet.GetName(), v1alpha1.DMWorkerMemberType, ordinal)
	if err != nil {
		return err
	}

	if !dc.Status.Worker.Synced {
		return fmt.Errorf("DMCluster: %s/%s's dm-worker status sync failed, can't scale out now", ns, dcName)
	}

	setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
	return nil
}

// Different from dm-master, we need remove member from cluster **after** reducing statefulset replicas
// Now it will be removed in syncing worker status. For dm-worker we can't remove its register info from dm-master
// when it's still alive. So we delete it later after its keepalive lease is outdated or revoked.
// We can defer deleting dm-worker register info because dm-master will patch replication task through keepalive info.
// only remove one member at a time when scale down
func (wsd *workerScaler) ScaleIn(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	dc, ok := meta.(*v1alpha1.DMCluster)
	if !ok {
		return nil
	}
	ns := dc.GetNamespace()
	dcName := dc.GetName()
	_, ordinal, replicas, deleteSlots := scaleOne(oldSet, newSet)
	resetReplicas(newSet, oldSet)
	setName := oldSet.GetName()

	if !dc.Status.Worker.Synced {
		return fmt.Errorf("DMCluster: %s/%s's dm-worker status sync failed, can't scale in now", ns, dcName)
	}

	klog.Infof("scaling in dm-worker statefulset %s/%s, ordinal: %d (replicas: %d, delete slots: %v)", oldSet.Namespace, oldSet.Name, ordinal, replicas, deleteSlots.List())

	//if controller.PodWebhookEnabled {
	//	setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
	//	return nil
	//}

	pvcName := ordinalPVCName(v1alpha1.DMWorkerMemberType, setName, ordinal)
	pvc, err := wsd.deps.PVCLister.PersistentVolumeClaims(ns).Get(pvcName)
	if err != nil {
		return fmt.Errorf("dm-worker.ScaleIn: failed to get pvc %s for cluster %s/%s, error: %s", pvcName, ns, dcName, err)
	}

	if pvc.Annotations == nil {
		pvc.Annotations = map[string]string{}
	}
	now := time.Now().Format(time.RFC3339)
	pvc.Annotations[label.AnnPVCDeferDeleting] = now

	_, err = wsd.deps.PVCControl.UpdatePVC(dc, pvc)
	if err != nil {
		klog.Errorf("dm-worker scale in: failed to set pvc %s/%s annotation: %s to %s",
			ns, pvcName, label.AnnPVCDeferDeleting, now)
		return err
	}
	klog.Infof("dm-worker scale in: set pvc %s/%s annotation: %s to %s",
		ns, pvcName, label.AnnPVCDeferDeleting, now)

	setReplicasAndDeleteSlots(newSet, replicas, deleteSlots)
	return nil
}

func (wsd *workerScaler) SyncAutoScalerAnn(meta metav1.Object, oldSet *apps.StatefulSet) error {
	return nil
}

type fakeWorkerScaler struct{}

// NewFakeWorkerScaler returns a fake Scaler
func NewFakeWorkerScaler() Scaler {
	return &fakeWorkerScaler{}
}

func (fws *fakeWorkerScaler) Scale(meta metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	if *newSet.Spec.Replicas > *oldSet.Spec.Replicas {
		return fws.ScaleOut(meta, oldSet, newSet)
	} else if *newSet.Spec.Replicas < *oldSet.Spec.Replicas {
		return fws.ScaleIn(meta, oldSet, newSet)
	}
	return nil
}

func (fws *fakeWorkerScaler) ScaleOut(_ metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	setReplicasAndDeleteSlots(newSet, *oldSet.Spec.Replicas+1, nil)
	return nil
}

func (fws *fakeWorkerScaler) ScaleIn(_ metav1.Object, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	setReplicasAndDeleteSlots(newSet, *oldSet.Spec.Replicas-1, nil)
	return nil
}

func (fws *fakeWorkerScaler) SyncAutoScalerAnn(dc metav1.Object, actual *apps.StatefulSet) error {
	return nil
}
