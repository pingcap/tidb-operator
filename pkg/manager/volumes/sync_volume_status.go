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
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"

	v1 "k8s.io/api/core/v1"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
)

// SyncVolumeStatus lists all pods and pvc for the given component and set the volume status of tc.
func SyncVolumeStatus(pvm PodVolumeModifier, podLister corelisterv1.PodLister, tc *v1alpha1.TidbCluster, mt v1alpha1.MemberType) error {
	status := tc.ComponentStatus(mt)
	if status == nil {
		return fmt.Errorf("component status of %s is nil", mt)
	}

	desiredVolumes, err := pvm.GetDesiredVolumes(tc, mt)
	if err != nil {
		return fmt.Errorf("failed to get desired volumes: %v", err)
	}

	selector, err := MustNewSelectorFactory().NewSelector(tc.GetInstanceName(), mt)
	if err != nil {
		return fmt.Errorf("failed to get selector: %v", err)
	}

	pods, err := podLister.Pods(tc.GetNamespace()).List(selector)
	if err != nil {
		return fmt.Errorf("failed to list pods: %v", err)
	}

	// build observed status
	observedStatus := observeVolumeStatus(pvm, pods, desiredVolumes)
	updateVolumeStatus(status, observedStatus)

	return nil
}

func updateVolumeStatus(status v1alpha1.ComponentStatus, observed map[v1alpha1.StorageVolumeName]*v1alpha1.ObservedStorageVolumeStatus) {
	if status.GetVolumes() == nil {
		status.SetVolumes(map[v1alpha1.StorageVolumeName]*v1alpha1.StorageVolumeStatus{})
	}
	volumeStatus := status.GetVolumes()
	for volName, status := range observed {
		if _, exist := volumeStatus[volName]; !exist {
			volumeStatus[volName] = &v1alpha1.StorageVolumeStatus{
				Name: volName,
			}
		}
		volumeStatus[volName].ObservedStorageVolumeStatus = *status
	}
	for _, status := range volumeStatus {
		if _, exist := observed[status.Name]; !exist {
			delete(volumeStatus, status.Name)
		}
	}
}

func observeVolumeStatus(pvm PodVolumeModifier, pods []*v1.Pod, desiredVolumes []DesiredVolume) map[v1alpha1.StorageVolumeName]*v1alpha1.ObservedStorageVolumeStatus {
	observedStatus := map[v1alpha1.StorageVolumeName]*v1alpha1.ObservedStorageVolumeStatus{}
	for _, pod := range pods {
		actualVolumes, err := pvm.GetActualVolumes(pod, desiredVolumes)
		if err != nil {
			klog.Warningf("skip to observe volume status for pod %s/%s, because failed to get actual volumes: %v", pod.Namespace, pod.Name, err)
			continue
		}

		for _, volume := range actualVolumes {
			volName := volume.Desired.Name
			desiredCap := volume.Desired.GetStorageSize()
			actualCap := volume.GetStorageSize()
			desiredSC := volume.Desired.GetStorageClassName()
			actualSC := volume.GetStorageClassName()

			status, exist := observedStatus[volName]
			if !exist {
				observedStatus[volName] = &v1alpha1.ObservedStorageVolumeStatus{
					BoundCount:    0,
					CurrentCount:  0,
					ModifiedCount: 0,
					// CurrentCapacity is default to same as desired capacity, and maybe changed later if any
					// volume is modifying.
					CurrentCapacity:  desiredCap,
					ModifiedCapacity: desiredCap,
					// CurrentStorageClass is default to same as desired storage class, and maybe changed later if any
					// volume is modifying.
					CurrentStorageClass:  desiredSC,
					ModifiedStorageClass: desiredSC,
				}
				status = observedStatus[volName]
			}

			status.BoundCount++
			capModified := actualCap.Cmp(desiredCap) == 0
			scModified := actualSC == desiredSC
			if capModified && scModified {
				status.ModifiedCount++
			} else {
				status.CurrentCount++
				if !capModified {
					status.CurrentCapacity = actualCap
				}
				if !scModified {
					status.CurrentStorageClass = actualSC
				}
			}

		}
	}

	for _, status := range observedStatus {
		// all volumes are modified, reset the current count
		if status.CurrentCapacity.Cmp(status.ModifiedCapacity) == 0 &&
			status.CurrentStorageClass == status.ModifiedStorageClass {
			status.CurrentCount = status.ModifiedCount
		}

		// TODO(shiori): remove it after removing the deprecated field
		status.ResizedCount = status.ModifiedCount
		status.ResizedCapacity = status.ModifiedCapacity
	}

	return observedStatus
}
