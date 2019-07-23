// Copyright 2018 PingCAP, Inc.
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
	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"k8s.io/apimachinery/pkg/api/errors"
	corelisters "k8s.io/client-go/listers/core/v1"
)

const (
	skipReasonOrphanPodsCleanerIsNotPDOrTiKV  = "orphan pods cleaner: member type is not pd or tikv"
	skipReasonOrphanPodsCleanerPVCNameIsEmpty = "orphan pods cleaner: pvcName is empty"
	skipReasonOrphanPodsCleanerPVCIsFound     = "orphan pods cleaner: pvc is found"
)

// OrphanPodsCleaner implements the logic for cleaning the orphan pods(has no pvc)
type OrphanPodsCleaner interface {
	Clean(*v1alpha1.TidbCluster) (map[string]string, error)
}

type orphanPodsCleaner struct {
	podLister  corelisters.PodLister
	podControl controller.PodControlInterface
	pvcLister  corelisters.PersistentVolumeClaimLister
}

// NewOrphanPodsCleaner returns a OrphanPodsCleaner
func NewOrphanPodsCleaner(podLister corelisters.PodLister,
	podControl controller.PodControlInterface,
	pvcLister corelisters.PersistentVolumeClaimLister) OrphanPodsCleaner {
	return &orphanPodsCleaner{podLister, podControl, pvcLister}
}

func (opc *orphanPodsCleaner) Clean(tc *v1alpha1.TidbCluster) (map[string]string, error) {
	ns := tc.GetNamespace()
	// for unit test
	skipReason := map[string]string{}

	selector, err := label.New().Instance(tc.GetLabels()[label.InstanceLabelKey]).Selector()
	if err != nil {
		return skipReason, err
	}
	pods, err := opc.podLister.Pods(ns).List(selector)
	if err != nil {
		return skipReason, err
	}

	for _, pod := range pods {
		podName := pod.GetName()
		l := label.Label(pod.Labels)
		if !(l.IsPD() || l.IsTiKV()) {
			skipReason[podName] = skipReasonOrphanPodsCleanerIsNotPDOrTiKV
			continue
		}

		var pvcName string
		for _, vol := range pod.Spec.Volumes {
			if vol.PersistentVolumeClaim != nil {
				pvcName = vol.PersistentVolumeClaim.ClaimName
				break
			}
		}
		if pvcName == "" {
			skipReason[podName] = skipReasonOrphanPodsCleanerPVCNameIsEmpty
			continue
		}

		_, err := opc.pvcLister.PersistentVolumeClaims(ns).Get(pvcName)
		if err == nil {
			skipReason[podName] = skipReasonOrphanPodsCleanerPVCIsFound
			continue
		}
		if !errors.IsNotFound(err) {
			return skipReason, err
		}

		err = opc.podControl.DeletePod(tc, pod)
		if err != nil {
			glog.Errorf("orphan pods cleaner: failed to clean orphan pod: %s/%s, %v", ns, podName, err)
			return skipReason, err
		}
		glog.Infof("orphan pods cleaner: clean orphan pod: %s/%s successfully", ns, podName)
	}

	return skipReason, nil
}

type fakeOrphanPodsCleaner struct{}

// NewFakeOrphanPodsCleaner returns a fake orphan pods cleaner
func NewFakeOrphanPodsCleaner() OrphanPodsCleaner {
	return &fakeOrphanPodsCleaner{}
}

func (fopc *fakeOrphanPodsCleaner) Clean(_ *v1alpha1.TidbCluster) (map[string]string, error) {
	return nil, nil
}
