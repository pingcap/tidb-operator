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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
)

const (
	skipReasonOrphanPodsCleanerIsNotTarget         = "orphan pods cleaner: member type is not pd, tikv or tiflash"
	skipReasonOrphanPodsCleanerPVCNameIsEmpty      = "orphan pods cleaner: pvcName is empty"
	skipReasonOrphanPodsCleanerPVCIsFound          = "orphan pods cleaner: pvc is found"
	skipReasonOrphanPodsCleanerPodHasBeenScheduled = "orphan pods cleaner: pod has been scheduled"
	skipReasonOrphanPodsCleanerPodIsNotFound       = "orphan pods cleaner: pod does not exist anymore"
	skipReasonOrphanPodsCleanerPodRecreated        = "orphan pods cleaner: pod is recreated before deletion"
)

// OrphanPodsCleaner implements the logic for cleaning the orphan pods(has no pvc)
//
// In scaling out and failover, we will try to delete the old PVC to prevent it
// from being used by the new pod. However, the PVC might not be deleted
// immediately in the apiserver because of finalizers (e.g.
// kubernetes.io/pvc-protection) and the statefulset controller may not have
// received PVC delete event when it tries to create the new replica and the
// new pod will be pending forever because no PVC to use. We need to clean
// these orphan pods and let the statefulset controller to create PVC(s) for
// them.
//
// https://github.com/kubernetes/kubernetes/blob/84fe3db5cf58bf0fc8ff792b885465ceaf70a435/pkg/controller/statefulset/stateful_pod_control.go#L175-L199
//
type OrphanPodsCleaner interface {
	Clean(*v1alpha1.TidbCluster) (map[string]string, error)
}

type orphanPodsCleaner struct {
	podLister  corelisters.PodLister
	podControl controller.PodControlInterface
	pvcLister  corelisters.PersistentVolumeClaimLister
	kubeCli    kubernetes.Interface
}

// NewOrphanPodsCleaner returns a OrphanPodsCleaner
func NewOrphanPodsCleaner(podLister corelisters.PodLister,
	podControl controller.PodControlInterface,
	pvcLister corelisters.PersistentVolumeClaimLister,
	kubeCli kubernetes.Interface) OrphanPodsCleaner {
	return &orphanPodsCleaner{podLister, podControl, pvcLister, kubeCli}
}

func (opc *orphanPodsCleaner) Clean(tc *v1alpha1.TidbCluster) (map[string]string, error) {
	ns := tc.GetNamespace()
	skipReason := map[string]string{}

	selector, err := label.New().Instance(tc.GetInstanceName()).Selector()
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
		if !(l.IsPD() || l.IsTiKV() || l.IsTiFlash()) {
			skipReason[podName] = skipReasonOrphanPodsCleanerIsNotTarget
			continue
		}

		if len(pod.Spec.NodeName) > 0 {
			skipReason[podName] = skipReasonOrphanPodsCleanerPodHasBeenScheduled
			continue
		}

		var pvcNames []string
		for _, vol := range pod.Spec.Volumes {
			if vol.PersistentVolumeClaim != nil {
				if vol.PersistentVolumeClaim.ClaimName != "" {
					pvcNames = append(pvcNames, vol.PersistentVolumeClaim.ClaimName)
				}
			}
		}
		if len(pvcNames) < 1 {
			skipReason[podName] = skipReasonOrphanPodsCleanerPVCNameIsEmpty
			continue
		}

		var err error
		var pvcNotFound bool
		for _, p := range pvcNames {
			// check informer cache
			_, err = opc.pvcLister.PersistentVolumeClaims(ns).Get(p)
			if err == nil {
				continue
			}
			if !errors.IsNotFound(err) {
				return skipReason, err
			}
			// if PVC not found in cache, re-check from apiserver directly to make sure the PVC really not exist
			_, err = opc.kubeCli.CoreV1().PersistentVolumeClaims(ns).Get(p, metav1.GetOptions{})
			if err == nil {
				continue
			}
			if !errors.IsNotFound(err) {
				return skipReason, err
			}
			pvcNotFound = true
			break
		}

		if !pvcNotFound {
			skipReason[podName] = skipReasonOrphanPodsCleanerPVCIsFound
			continue
		}

		// if the PVC is not found in apiserver (also informer cache) and the
		// pod has not been scheduled, delete it and let the stateful
		// controller to create the pod and its PVC(s) again
		apiPod, err := opc.kubeCli.CoreV1().Pods(ns).Get(podName, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			skipReason[podName] = skipReasonOrphanPodsCleanerPodIsNotFound
			continue
		}
		if apiPod.UID != pod.UID {
			skipReason[podName] = skipReasonOrphanPodsCleanerPodRecreated
		}
		if err != nil {
			return skipReason, err
		}
		// In pre-1.14, kube-apiserver does not support
		// deleteOption.Preconditions.ResourceVersion, we fetch the latest
		// version and check again before deletion.
		if len(apiPod.Spec.NodeName) > 0 {
			skipReason[podName] = skipReasonOrphanPodsCleanerPodHasBeenScheduled
		}
		// As the pod may be updated by kube-scheduler or other components
		// frequently, we should use the latest object here to avoid API
		// conflict.
		err = opc.podControl.DeletePod(tc, apiPod)
		if err != nil {
			klog.Errorf("orphan pods cleaner: failed to clean orphan pod: %s/%s, %v", ns, podName, err)
			return skipReason, err
		}
		klog.Infof("orphan pods cleaner: clean orphan pod: %s/%s successfully", ns, podName)
	}

	return skipReason, nil
}

type FakeOrphanPodsCleaner struct {
	err error
}

// NewFakeOrphanPodsCleaner returns a fake orphan pods cleaner
func NewFakeOrphanPodsCleaner() *FakeOrphanPodsCleaner {
	return &FakeOrphanPodsCleaner{}
}

func (fpc *FakeOrphanPodsCleaner) SetnOrphanPodCleanerError(err error) {
	fpc.err = err
}

func (fpc *FakeOrphanPodsCleaner) Clean(_ *v1alpha1.TidbCluster) (map[string]string, error) {
	return nil, fpc.err
}

var _ OrphanPodsCleaner = &FakeOrphanPodsCleaner{}
