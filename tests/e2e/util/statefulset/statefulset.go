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

package statefulset

import (
	"regexp"
	"strconv"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
)

var statefulPodRegex = regexp.MustCompile("(.*)-([0-9]+)$")

func GetStatefulPodOrdinal(podName string) int {
	ordinal := -1
	subMatches := statefulPodRegex.FindStringSubmatch(podName)
	if len(subMatches) < 3 {
		return ordinal
	}
	if i, err := strconv.ParseInt(subMatches[2], 10, 32); err == nil {
		ordinal = int(i)
	}
	return ordinal
}

// IsAllDesiredPodsRunningAndReady checks if all desired pods of given statefulset are running and ready
func IsAllDesiredPodsRunningAndReady(c kubernetes.Interface, sts *appsv1.StatefulSet) bool {
	deleteSlots := helper.GetDeleteSlots(sts)
	actualPodList, err := getPodList(c, sts)
	if err != nil {
		klog.Infof("get podlist error in IsAllDesiredPodsRunningAndReady, err:%v", err)
		return false
	}
	actualPodOrdinals := sets.NewInt32()
	for _, pod := range actualPodList.Items {
		actualPodOrdinals.Insert(int32(GetStatefulPodOrdinal(pod.Name)))
	}
	desiredPodOrdinals := helper.GetPodOrdinalsFromReplicasAndDeleteSlots(*sts.Spec.Replicas, deleteSlots)
	if !actualPodOrdinals.Equal(desiredPodOrdinals) {
		klog.Infof("pod ordinals of sts %s/%s is %v, expects: %v", sts.Namespace, sts.Name, actualPodOrdinals.List(), desiredPodOrdinals.List())
		return false
	}
	for _, pod := range actualPodList.Items {
		if !podutil.IsPodReady(&pod) {
			klog.Infof("pod %s of sts %s/%s is not ready, got: %v", pod.Name, sts.Namespace, sts.Name, podutil.GetPodReadyCondition(pod.Status))
			return false
		}
	}
	klog.Infof("desired pods of sts %s/%s are running and ready (%v)", sts.Namespace, sts.Name, actualPodOrdinals.List())
	return true
}

// GetPodList gets the current Pods in ss.
// e2esset.GetPodList(c, sts) would panic while we want return error if something wrong
func getPodList(c kubernetes.Interface, ss *appsv1.StatefulSet) (*corev1.PodList, error) {
	selector, err := metav1.LabelSelectorAsSelector(ss.Spec.Selector)
	if err != nil {
		return nil, err
	}
	podList, err := c.CoreV1().Pods(ss.Namespace).List(metav1.ListOptions{LabelSelector: selector.String()})
	return podList, err
}
