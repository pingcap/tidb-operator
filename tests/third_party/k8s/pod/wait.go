/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// this file is copied from k8s.io/kubernetes/test/e2e/framework/pod/wait.go @v1.23.17

package pod

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"text/tabwriter"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"

	podutil "github.com/pingcap/tidb-operator/pkg/third_party/k8s"
	e2elog "github.com/pingcap/tidb-operator/tests/third_party/k8s/log"
)

const (
	// poll is how often to poll pods, nodes and claims.
	poll = 2 * time.Second
)

// errorBadPodsStates create error message of basic info of bad pods for debugging.
func errorBadPodsStates(badPods []v1.Pod, desiredPods int, ns, desiredState string, timeout time.Duration, err error) string {
	errStr := fmt.Sprintf("%d / %d pods in namespace %q are NOT in %s state in %v\n", len(badPods), desiredPods, ns, desiredState, timeout)
	if err != nil {
		errStr += fmt.Sprintf("Last error: %s\n", err)
	}
	// Print bad pods info only if there are fewer than 10 bad pods
	if len(badPods) > 10 {
		return errStr + "There are too many bad pods. Please check log for details."
	}

	buf := bytes.NewBuffer(nil)
	w := tabwriter.NewWriter(buf, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, "POD\tNODE\tPHASE\tGRACE\tCONDITIONS")
	for _, badPod := range badPods {
		grace := ""
		if badPod.DeletionGracePeriodSeconds != nil {
			grace = fmt.Sprintf("%ds", *badPod.DeletionGracePeriodSeconds)
		}
		podInfo := fmt.Sprintf("%s\t%s\t%s\t%s\t%+v",
			badPod.ObjectMeta.Name, badPod.Spec.NodeName, badPod.Status.Phase, grace, badPod.Status.Conditions)
		fmt.Fprintln(w, podInfo)
	}
	w.Flush()
	return errStr + buf.String()
}

// WaitForPodsRunningReady waits up to timeout to ensure that all pods in
// namespace ns are either running and ready, or failed but controlled by a
// controller. Also, it ensures that at least minPods are running and
// ready. It has separate behavior from other 'wait for' pods functions in
// that it requests the list of pods on every iteration. This is useful, for
// example, in cluster startup, because the number of pods increases while
// waiting. All pods that are in SUCCESS state are not counted.
//
// If ignoreLabels is not empty, pods matching this selector are ignored.
//
// If minPods or allowedNotReadyPods are -1, this method returns immediately
// without waiting.
func WaitForPodsRunningReady(c clientset.Interface, ns string, minPods, allowedNotReadyPods int32, timeout time.Duration, ignoreLabels map[string]string) error {
	if minPods == -1 || allowedNotReadyPods == -1 {
		return nil
	}

	ignoreSelector := labels.SelectorFromSet(map[string]string{})
	start := time.Now()
	e2elog.Logf("Waiting up to %v for all pods (need at least %d) in namespace '%s' to be running and ready",
		timeout, minPods, ns)
	var ignoreNotReady bool
	badPods := []v1.Pod{}
	desiredPods := 0
	notReady := int32(0)
	var lastAPIError error

	if wait.PollImmediate(poll, timeout, func() (bool, error) {
		// We get the new list of pods, replication controllers, and
		// replica sets in every iteration because more pods come
		// online during startup and we want to ensure they are also
		// checked.
		replicas, replicaOk := int32(0), int32(0)
		// Clear API error from the last attempt in case the following calls succeed.
		lastAPIError = nil

		rcList, err := c.CoreV1().ReplicationControllers(ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			e2elog.Logf("Error getting replication controllers in namespace '%s': %v", ns, err)
			lastAPIError = err
			return false, err
		}
		for _, rc := range rcList.Items {
			replicas += *rc.Spec.Replicas
			replicaOk += rc.Status.ReadyReplicas
		}

		rsList, err := c.AppsV1().ReplicaSets(ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			lastAPIError = err
			e2elog.Logf("Error getting replication sets in namespace %q: %v", ns, err)
			return false, err
		}
		for _, rs := range rsList.Items {
			replicas += *rs.Spec.Replicas
			replicaOk += rs.Status.ReadyReplicas
		}

		podList, err := c.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			lastAPIError = err
			e2elog.Logf("Error getting pods in namespace '%s': %v", ns, err)
			return false, err
		}
		nOk := int32(0)
		notReady = int32(0)
		badPods = []v1.Pod{}
		desiredPods = len(podList.Items)
		for _, pod := range podList.Items {
			if len(ignoreLabels) != 0 && ignoreSelector.Matches(labels.Set(pod.Labels)) {
				continue
			}
			res, err := PodRunningReady(&pod)
			switch {
			case res && err == nil:
				nOk++
			case pod.Status.Phase == v1.PodSucceeded:
				e2elog.Logf("The status of Pod %s is Succeeded, skipping waiting", pod.ObjectMeta.Name)
				// it doesn't make sense to wait for this pod
				continue
			case pod.Status.Phase != v1.PodFailed:
				e2elog.Logf("The status of Pod %s is %s (Ready = false), waiting for it to be either Running (with Ready = true) or Failed", pod.ObjectMeta.Name, pod.Status.Phase)
				notReady++
				badPods = append(badPods, pod)
			default:
				if metav1.GetControllerOf(&pod) == nil {
					e2elog.Logf("Pod %s is Failed, but it's not controlled by a controller", pod.ObjectMeta.Name)
					badPods = append(badPods, pod)
				}
				//ignore failed pods that are controlled by some controller
			}
		}

		e2elog.Logf("%d / %d pods in namespace '%s' are running and ready (%d seconds elapsed)",
			nOk, len(podList.Items), ns, int(time.Since(start).Seconds()))
		e2elog.Logf("expected %d pod replicas in namespace '%s', %d are Running and Ready.", replicas, ns, replicaOk)

		if replicaOk == replicas && nOk >= minPods && len(badPods) == 0 {
			return true, nil
		}
		ignoreNotReady = (notReady <= allowedNotReadyPods)
		LogPodStates(badPods)
		return false, nil
	}) != nil {
		if !ignoreNotReady {
			return errors.New(errorBadPodsStates(badPods, desiredPods, ns, "RUNNING and READY", timeout, lastAPIError))
		}
		e2elog.Logf("Number of not-ready pods (%d) is below the allowed threshold (%d).", notReady, allowedNotReadyPods)
	}
	return nil
}

// WaitTimeoutForPodRunningInNamespace waits the given timeout duration for the specified pod to become running.
func WaitTimeoutForPodRunningInNamespace(c clientset.Interface, podName, namespace string, timeout time.Duration) error {
	return wait.PollImmediate(poll, timeout, podRunning(c, podName, namespace))
}

// WaitTimeoutForPodReadyInNamespace waits the given timeout duration for the
// specified pod to be ready and running.
func WaitTimeoutForPodReadyInNamespace(c clientset.Interface, podName, namespace string, timeout time.Duration) error {
	return wait.PollImmediate(poll, timeout, podRunningAndReady(c, podName, namespace))
}

// WaitForPodNotFoundInNamespace returns an error if it takes too long for the pod to fully terminate.
// Unlike `waitForPodTerminatedInNamespace`, the pod's Phase and Reason are ignored. If the pod Get
// api returns IsNotFound then the wait stops and nil is returned. If the Get api returns an error other
// than "not found" then that error is returned and the wait stops.
func WaitForPodNotFoundInNamespace(c clientset.Interface, podName, ns string, timeout time.Duration) error {
	return wait.PollImmediate(poll, timeout, func() (bool, error) {
		_, err := c.CoreV1().Pods(ns).Get(context.TODO(), podName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil // done
		}
		if err != nil {
			return true, err // stop wait with error
		}
		return false, nil
	})
}

// PodRunningReady checks whether pod p's phase is running and it has a ready
// condition of status true.
// This function is copied from k8s.io/kubernetes/test/utils/conditions.go @v1.23.17
func PodRunningReady(p *v1.Pod) (bool, error) {
	// Check the phase is running.
	if p.Status.Phase != v1.PodRunning {
		return false, fmt.Errorf("want pod '%s' on '%s' to be '%v' but was '%v'",
			p.ObjectMeta.Name, p.Spec.NodeName, v1.PodRunning, p.Status.Phase)
	}
	// Check the ready condition is true.
	if !podutil.IsPodReady(p) {
		return false, fmt.Errorf("pod '%s' on '%s' didn't have condition {%v %v}; conditions: %v",
			p.ObjectMeta.Name, p.Spec.NodeName, v1.PodReady, v1.ConditionTrue, p.Status.Conditions)
	}
	return true, nil
}
