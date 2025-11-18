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

package podutil

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// IsContainerRunning means the main container of this pod is running and deleting this pod may increase an unavailable instance
func IsContainerRunning(pod *corev1.Pod, mainContainerName string) error {
	if pod == nil {
		return fmt.Errorf("pod is not found")
	}
	if pod.Status.Phase != corev1.PodRunning {
		return fmt.Errorf("pod is not running")
	}
	var mc *corev1.ContainerStatus
	for i := range pod.Status.ContainerStatuses {
		cs := &pod.Status.ContainerStatuses[i]
		if cs.Name == mainContainerName {
			mc = cs
		}
	}

	if mc == nil {
		return fmt.Errorf("main container %s is not found", mainContainerName)
	}

	// main container is ready, assume that the pod is running
	if mc.Ready {
		return nil
	}

	// main container is started, assume that pod is running
	if mc.Started != nil && *mc.Started {
		return nil
	}

	// main container is running
	if mc.State.Running != nil {
		return nil
	}

	switch {
	case mc.State.Terminated != nil:
		return fmt.Errorf(
			"main container %s is terminated, reason: %s, message: %s",
			mainContainerName,
			mc.State.Terminated.Reason,
			mc.State.Terminated.Message,
		)
	case mc.State.Waiting != nil:
		return fmt.Errorf(
			"main container %s is waiting, reason: %s, message: %s",
			mainContainerName,
			mc.State.Waiting.Reason,
			mc.State.Waiting.Message,
		)
	default:
		return fmt.Errorf(
			"main container %s is waiting for initialize status",
			mainContainerName,
		)
	}
}
