// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package framework

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/util/podutils"

	"github.com/pingcap/tidb-operator/pkg/client"
)

const (
	// poll is how often to poll pods, nodes and claims.
	poll = 2 * time.Second
)

// errPodCompleted is returned by PodRunning or PodContainerRunning to indicate that
// the pod has already reached completed state.
var errPodCompleted = fmt.Errorf("pod ran to completion")

// WaitTimeoutForPodReadyInNamespace waits the given timeout duration for the
// specified pod to be ready and running.
func WaitTimeoutForPodReadyInNamespace(c client.Client, podName, namespace string, timeout time.Duration) error {
	return wait.PollUntilContextCancel(context.TODO(), poll, true, podRunningAndReady(c, podName, namespace))
}

func podRunningAndReady(c client.Client, podName, namespace string) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		pod := &v1.Pod{}
		err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: podName}, pod)
		if err != nil {
			return false, err
		}
		switch pod.Status.Phase {
		case v1.PodFailed, v1.PodSucceeded:
			klog.Errorf("The status of Pod %s is %s which is unexpected", podName, pod.Status.Phase)
			return false, errPodCompleted
		case v1.PodRunning:
			klog.Infof("The status of Pod %s is %s (Ready = %v)", podName, pod.Status.Phase, podutils.IsPodReady(pod))
			return podutils.IsPodReady(pod), nil
		}
		klog.Infof("The status of Pod %s is %s, waiting for it to be Running (with Ready = true)", podName, pod.Status.Phase)
		return false, nil
	}
}
