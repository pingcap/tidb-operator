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

package pod

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	testutils "k8s.io/kubernetes/test/utils"
)

// PodsAreChanged checks the given pods are changed or not (recreate, update).
func PodsAreChanged(c kubernetes.Interface, pods []v1.Pod) wait.ConditionFunc {
	return func() (bool, error) {
		for _, pod := range pods {
			podNew, err := c.CoreV1().Pods(pod.Namespace).Get(context.TODO(), pod.Name, metav1.GetOptions{})
			if err != nil {
				if testutils.IsRetryableAPIError(err) {
					return false, nil
				}
				return false, err
			}
			changed := IsPodsChanged(pod, *podNew)
			if changed {
				return true, nil
			}
		}
		return false, nil
	}
}

func IsPodsChanged(old v1.Pod, cur v1.Pod) bool {
	if cur.UID != old.UID {
		return true
	}
	if !apiequality.Semantic.DeepEqual(cur.Spec, old.Spec) {
		return true
	}

	return false
}

// WaitForPodsAreChanged waits for given pods are changed.
// It returns wait.ErrWaitTimeout if the given pods are not changed in specified timeout.
func WaitForPodsAreChanged(c kubernetes.Interface, pods []v1.Pod, timeout time.Duration) error {
	return wait.PollImmediate(time.Second*5, timeout, PodsAreChanged(c, pods))
}

func ListPods(labelSelector string, ns string, c clientset.Interface) ([]v1.Pod, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}
	podList, err := c.CoreV1().Pods(ns).List(context.TODO(), listOptions)
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}

func MustListPods(labelSelector string, ns string, c clientset.Interface) []v1.Pod {
	pods, err := ListPods(labelSelector, ns, c)
	framework.ExpectNoError(err, "failed to list pods in ns %s with selector %v", ns, labelSelector)

	return pods
}
