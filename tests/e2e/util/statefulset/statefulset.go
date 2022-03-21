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
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"k8s.io/kubernetes/test/e2e/framework/log"
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
		log.Logf("get podlist error in IsAllDesiredPodsRunningAndReady, err:%v", err)
		return false
	}
	actualPodOrdinals := sets.NewInt32()
	for _, pod := range actualPodList.Items {
		actualPodOrdinals.Insert(int32(GetStatefulPodOrdinal(pod.Name)))
	}
	desiredPodOrdinals := helper.GetPodOrdinalsFromReplicasAndDeleteSlots(*sts.Spec.Replicas, deleteSlots)
	if !actualPodOrdinals.Equal(desiredPodOrdinals) {
		log.Logf("pod ordinals of sts %s/%s is %v, expects: %v", sts.Namespace, sts.Name, actualPodOrdinals.List(), desiredPodOrdinals.List())
		return false
	}
	for _, pod := range actualPodList.Items {
		if !podutil.IsPodReady(&pod) {
			log.Logf("pod %s of sts %s/%s is not ready, got: %v", pod.Name, sts.Namespace, sts.Name, podutil.GetPodReadyCondition(pod.Status))
			return false
		}
	}
	// log.Logf("desired pods of sts %s/%s are running and ready (%v)", sts.Namespace, sts.Name, actualPodOrdinals.List())
	return true
}

// GetMemberContainersFromSts returns the member containers of the sts.
//
// If no container is found, return a error.
func GetMemberContainersFromSts(cli kubernetes.Interface, stsGetter typedappsv1.StatefulSetsGetter, namespace, stsName string, component v1alpha1.MemberType) ([]corev1.Container, error) {
	sts, err := stsGetter.StatefulSets(namespace).Get(context.TODO(), stsName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get sts %s/%s: %v", namespace, stsName, err)
	}

	listOption := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(sts.Spec.Selector.MatchLabels).String(),
	}
	podList, err := cli.CoreV1().Pods(sts.Namespace).List(context.TODO(), listOption)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods of sts %s/%s: %v", namespace, stsName, err)
	}
	if len(podList.Items) == 0 {
		return nil, fmt.Errorf("no pod found of sts %s/%s", namespace, stsName)
	}

	containers := getContainersFromPods(podList.Items, string(component))
	if len(containers) == 0 {
		return nil, fmt.Errorf("no container %q found of sts %s/%s", component, namespace, stsName)
	}

	return containers, nil
}

func getContainersFromPods(pods []corev1.Pod, containerName string) []corev1.Container {
	containers := []corev1.Container{}

	for _, pod := range pods {
		for _, c := range pod.Spec.Containers {
			if c.Name == containerName {
				containers = append(containers, c)
			}
		}
	}

	return containers
}

// GetPodList gets the current Pods in ss.
// e2esset.GetPodList(c, sts) would panic while we want return error if something wrong
func getPodList(c kubernetes.Interface, ss *appsv1.StatefulSet) (*corev1.PodList, error) {
	selector, err := metav1.LabelSelectorAsSelector(ss.Spec.Selector)
	if err != nil {
		return nil, err
	}
	podList, err := c.CoreV1().Pods(ss.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	return podList, err
}
