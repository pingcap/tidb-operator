// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License a
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package statefulset

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1/helper"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"k8s.io/kubernetes/test/e2e/framework"
	e2esset "k8s.io/kubernetes/test/e2e/framework/statefulset"
)

var statefulPodRegex = regexp.MustCompile("(.*)-([0-9]+)$")

func getStatefulPodOrdinal(podName string) int {
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

// WaitForStatusReplicas waits for the ss.Status.Replicas to be equal to expectedReplicas
func WaitForStatusReplicas(c clientset.Interface, ss *appsv1.StatefulSet, expectedReplicas int32) {
	framework.Logf("Waiting for statefulset status.replicas updated to %d", expectedReplicas)

	ns, name := ss.Namespace, ss.Name
	pollErr := wait.PollImmediate(e2esset.StatefulSetPoll, e2esset.StatefulSetTimeout,
		func() (bool, error) {
			ssGet, err := c.AppsV1().StatefulSets(ns).Get(name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if ssGet.Status.ObservedGeneration < ss.Generation {
				return false, nil
			}
			if ssGet.Status.Replicas != expectedReplicas {
				framework.Logf("Waiting for stateful set status.replicas to become %d, currently %d", expectedReplicas, ssGet.Status.Replicas)
				return false, nil
			}
			return true, nil
		})
	if pollErr != nil {
		framework.Failf("Failed waiting for stateful set status.replicas updated to %d: %v", expectedReplicas, pollErr)
	}
}

// WaitForRunning works like e2esset.WaitForRunning except it takes delete slots into account.
func WaitForRunning(c clientset.Interface, numPodsRunning, numPodsReady int32, ss *appsv1.StatefulSet) {
	pollErr := wait.PollImmediate(e2esset.StatefulSetPoll, e2esset.StatefulSetTimeout,
		func() (bool, error) {
			podList := e2esset.GetPodList(c, ss)
			e2esset.SortStatefulPods(podList)
			if int32(len(podList.Items)) < numPodsRunning {
				framework.Logf("Found %d stateful pods, waiting for %d", len(podList.Items), numPodsRunning)
				return false, nil
			}
			if int32(len(podList.Items)) > numPodsRunning {
				return false, fmt.Errorf("too many pods scheduled, expected %d got %d", numPodsRunning, len(podList.Items))
			}
			deleteSlots := helper.GetDeleteSlots(ss)
			replicaCount, deleteSlots := helper.GetMaxReplicaCountAndDeleteSlots(*ss.Spec.Replicas, deleteSlots)
			for _, p := range podList.Items {
				if deleteSlots.Has(int32(getStatefulPodOrdinal(p.Name))) {
					return false, fmt.Errorf("unexpected pod ordinal: %d for stateful set %q", getStatefulPodOrdinal(p.Name), ss.Name)
				}
				shouldBeReady := int32(getStatefulPodOrdinal(p.Name)) < replicaCount
				isReady := podutil.IsPodReady(&p)
				desiredReadiness := shouldBeReady == isReady
				framework.Logf("Waiting for pod %v to enter %v - Ready=%v, currently %v - Ready=%v", p.Name, v1.PodRunning, shouldBeReady, p.Status.Phase, isReady)
				if p.Status.Phase != v1.PodRunning || !desiredReadiness {
					return false, nil
				}
			}
			return true, nil
		})
	if pollErr != nil {
		framework.Failf("Failed waiting for pods to enter running: %v", pollErr)
	}
}

// WaitForRunningAndReady waits for numStatefulPods in ss to be Running and Ready.
func WaitForRunningAndReady(c clientset.Interface, numStatefulPods int32, ss *appsv1.StatefulSet) {
	WaitForRunning(c, numStatefulPods, numStatefulPods, ss)
}
