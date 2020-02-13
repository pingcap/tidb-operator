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
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	testutils "k8s.io/kubernetes/test/utils"
)

// PodsAreChanged checks the given pods are changed or not (recreate, update).
func PodsAreChanged(c kubernetes.Interface, pods []v1.Pod) wait.ConditionFunc {
	return func() (bool, error) {
		for _, pod := range pods {
			podNew, err := c.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
			if err != nil {
				if testutils.IsRetryableAPIError(err) {
					return false, nil
				}
				return false, err
			}
			if podNew.UID != pod.UID {
				return true, fmt.Errorf("pod %s/%s is recreated (old UID %s, new UID %s)", pod.Namespace, pod.Name, pod.UID, podNew.UID)
			}
			if !apiequality.Semantic.DeepEqual(pod.Spec, podNew.Spec) {
				return true, fmt.Errorf("pod %s/%s spec is changed, diff: %s", pod.Namespace, pod.Name, cmp.Diff(pod.Spec, podNew.Spec))
			}
		}
		return false, nil
	}
}

// WaitForPodsAreChanged waits for given pods are changed.
// It returns wait.ErrWaitTimeout if the given pods are not changed in specified timeout.
func WaitForPodsAreChanged(c kubernetes.Interface, pods []v1.Pod, timeout time.Duration) error {
	return wait.PollImmediate(time.Second*5, timeout, PodsAreChanged(c, pods))
}
