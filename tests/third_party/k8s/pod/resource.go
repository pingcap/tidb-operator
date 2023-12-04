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

// this file is copiedfrom k8s.io/kubernetes/test/e2e/framework/pod/resource.go @v1.23.17

package pod

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/util/podutils"

	e2elog "github.com/pingcap/tidb-operator/tests/third_party/k8s/log"
)

// errPodCompleted is returned by PodRunning or PodContainerRunning to indicate that
// the pod has already reached completed state.
var errPodCompleted = fmt.Errorf("pod ran to completion")

// LabelLogOnPodFailure can be used to mark which Pods will have their logs logged in the case of
// a test failure. By default, if there are no Pods with this label, only the first 5 Pods will
// have their logs fetched.
const LabelLogOnPodFailure = "log-on-pod-failure"

func podRunning(c clientset.Interface, podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := c.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		switch pod.Status.Phase {
		case v1.PodRunning:
			return true, nil
		case v1.PodFailed, v1.PodSucceeded:
			return false, errPodCompleted
		}
		return false, nil
	}
}

func podRunningAndReady(c clientset.Interface, podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := c.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		switch pod.Status.Phase {
		case v1.PodFailed, v1.PodSucceeded:
			e2elog.Logf("The status of Pod %s is %s which is unexpected", podName, pod.Status.Phase)
			return false, errPodCompleted
		case v1.PodRunning:
			e2elog.Logf("The status of Pod %s is %s (Ready = %v)", podName, pod.Status.Phase, podutils.IsPodReady(pod))
			return podutils.IsPodReady(pod), nil
		}
		e2elog.Logf("The status of Pod %s is %s, waiting for it to be Running (with Ready = true)", podName, pod.Status.Phase)
		return false, nil
	}
}

// LogPodStates logs basic info of provided pods for debugging.
func LogPodStates(pods []v1.Pod) {
	// Find maximum widths for pod, node, and phase strings for column printing.
	maxPodW, maxNodeW, maxPhaseW, maxGraceW := len("POD"), len("NODE"), len("PHASE"), len("GRACE")
	for i := range pods {
		pod := &pods[i]
		if len(pod.ObjectMeta.Name) > maxPodW {
			maxPodW = len(pod.ObjectMeta.Name)
		}
		if len(pod.Spec.NodeName) > maxNodeW {
			maxNodeW = len(pod.Spec.NodeName)
		}
		if len(pod.Status.Phase) > maxPhaseW {
			maxPhaseW = len(pod.Status.Phase)
		}
	}
	// Increase widths by one to separate by a single space.
	maxPodW++
	maxNodeW++
	maxPhaseW++
	maxGraceW++

	// Log pod info. * does space padding, - makes them left-aligned.
	e2elog.Logf("%-[1]*[2]s %-[3]*[4]s %-[5]*[6]s %-[7]*[8]s %[9]s",
		maxPodW, "POD", maxNodeW, "NODE", maxPhaseW, "PHASE", maxGraceW, "GRACE", "CONDITIONS")
	for _, pod := range pods {
		grace := ""
		if pod.DeletionGracePeriodSeconds != nil {
			grace = fmt.Sprintf("%ds", *pod.DeletionGracePeriodSeconds)
		}
		e2elog.Logf("%-[1]*[2]s %-[3]*[4]s %-[5]*[6]s %-[7]*[8]s %[9]s",
			maxPodW, pod.ObjectMeta.Name, maxNodeW, pod.Spec.NodeName, maxPhaseW, pod.Status.Phase, maxGraceW, grace, pod.Status.Conditions)
	}
	e2elog.Logf("") // Final empty line helps for readability.
}

// logPodTerminationMessages logs termination messages for failing pods.  It's a short snippet (much smaller than full logs), but it often shows
// why pods crashed and since it is in the API, it's fast to retrieve.
func logPodTerminationMessages(pods []v1.Pod) {
	for _, pod := range pods {
		for _, status := range pod.Status.InitContainerStatuses {
			if status.LastTerminationState.Terminated != nil && len(status.LastTerminationState.Terminated.Message) > 0 {
				e2elog.Logf("%s[%s].initContainer[%s]=%s", pod.Name, pod.Namespace, status.Name, status.LastTerminationState.Terminated.Message)
			}
		}
		for _, status := range pod.Status.ContainerStatuses {
			if status.LastTerminationState.Terminated != nil && len(status.LastTerminationState.Terminated.Message) > 0 {
				e2elog.Logf("%s[%s].container[%s]=%s", pod.Name, pod.Namespace, status.Name, status.LastTerminationState.Terminated.Message)
			}
		}
	}
}

// logPodLogs logs the container logs from pods in the given namespace. This can be helpful for debugging
// issues that do not cause the container to fail (e.g.: network connectivity issues)
// We will log the Pods that have the LabelLogOnPodFailure label. If there aren't any, we default to
// logging only the first 5 Pods. This requires the reportDir to be set, and the pods are logged into:
// {report_dir}/pods/{namespace}/{pod}/{container_name}/logs.txt
func logPodLogs(c clientset.Interface, namespace string, pods []v1.Pod, reportDir string) {
	if reportDir == "" {
		return
	}

	var logPods []v1.Pod
	for _, pod := range pods {
		if _, ok := pod.Labels[LabelLogOnPodFailure]; ok {
			logPods = append(logPods, pod)
		}
	}
	maxPods := len(logPods)

	// There are no pods with the LabelLogOnPodFailure label, we default to the first 5 Pods.
	if maxPods == 0 {
		logPods = pods
		maxPods = len(pods)
		if maxPods > 5 {
			maxPods = 5
		}
	}

	tailLen := 42
	for i := 0; i < maxPods; i++ {
		pod := logPods[i]
		for _, container := range pod.Spec.Containers {
			logs, err := getPodLogsInternal(c, namespace, pod.Name, container.Name, false, nil, &tailLen)
			if err != nil {
				e2elog.Logf("Unable to fetch %s/%s/%s logs: %v", pod.Namespace, pod.Name, container.Name, err)
				continue
			}

			logDir := filepath.Join(reportDir, namespace, pod.Name, container.Name)
			err = os.MkdirAll(logDir, 0755)
			if err != nil {
				e2elog.Logf("Unable to create path '%s'. Err: %v", logDir, err)
				continue
			}

			logPath := filepath.Join(logDir, "logs.txt")
			err = os.WriteFile(logPath, []byte(logs), 0644)
			if err != nil {
				e2elog.Logf("Could not write the container logs in: %s. Err: %v", logPath, err)
			}
		}
	}
}

// DumpAllPodInfoForNamespace logs all pod information for a given namespace.
func DumpAllPodInfoForNamespace(c clientset.Interface, namespace, reportDir string) {
	pods, err := c.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		e2elog.Logf("unable to fetch pod debug info: %v", err)
	}
	LogPodStates(pods.Items)
	logPodTerminationMessages(pods.Items)
	logPodLogs(c, namespace, pods.Items, reportDir)
}

// GetPodLogs returns the logs of the specified container (namespace/pod/container).
func GetPodLogs(c clientset.Interface, namespace, podName, containerName string) (string, error) {
	return getPodLogsInternal(c, namespace, podName, containerName, false, nil, nil)
}

// GetPreviousPodLogs returns the logs of the previous instance of the
// specified container (namespace/pod/container).
func GetPreviousPodLogs(c clientset.Interface, namespace, podName, containerName string) (string, error) {
	return getPodLogsInternal(c, namespace, podName, containerName, true, nil, nil)
}

// utility function for gomega Eventually
func getPodLogsInternal(c clientset.Interface, namespace, podName, containerName string, previous bool, sinceTime *metav1.Time, tailLines *int) (string, error) {
	request := c.CoreV1().RESTClient().Get().
		Resource("pods").
		Namespace(namespace).
		Name(podName).SubResource("log").
		Param("container", containerName).
		Param("previous", strconv.FormatBool(previous))
	if sinceTime != nil {
		request.Param("sinceTime", sinceTime.Format(time.RFC3339))
	}
	if tailLines != nil {
		request.Param("tailLines", strconv.Itoa(*tailLines))
	}
	logs, err := request.Do(context.TODO()).Raw()
	if err != nil {
		return "", err
	}
	if strings.Contains(string(logs), "Internal Error") {
		return "", fmt.Errorf("Fetched log contains \"Internal Error\": %q", string(logs))
	}
	return string(logs), err
}
