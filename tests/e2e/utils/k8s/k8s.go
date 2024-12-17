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

package k8s

import (
	"os"
	"slices"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
)

// LoadConfig loads the k8s config from the default location.
func LoadConfig() (*rest.Config, error) {
	// init k8s client with the current default config
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = os.ExpandEnv("$HOME/.kube/config")
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}

// CheckRollingRestartLogic checks if the rolling restart logic is correctly followed
// based on the provided list of Kubernetes watch events.
// The function first sort events by time, then checks for pod creation events and
// verifies the restart events to ensure that the rolling restart logic is adhered to.
// An example of a rolling restart event sequence is like:
//  1. pod1 added
//  2. pod2 added
//  4. pod2 deleted
//  5. pod2 added
//  8. pod1 deleted
//  9. pod1 added
//
//nolint:gocyclo // refactor if possible
func CheckRollingRestartLogic(events []watch.Event) bool {
	if len(events) == 0 {
		return false
	}

	// Sort eventsBeforeShuffle by time, since the eventsBeforeShuffle are not guaranteed to be in order
	slices.SortFunc(events, func(e1, e2 watch.Event) int {
		t1, t2 := getTimeFromEvent(e1), getTimeFromEvent(e2)
		if t1.Before(t2) {
			return -1
		} else if t1.After(t2) {
			return 1
		}
		return 0
	})

	podCreated := make(map[string]bool)
	restartEvents := make([]watch.Event, 0, len(events))

	// check pod creation events
	creationPhase := true
	for _, e := range events {
		if creationPhase {
			podName := e.Object.(*corev1.Pod).Name
			switch e.Type {
			case watch.Added:
				podCreated[podName] = true
			case watch.Deleted:
				if len(podCreated) == 0 {
					return false
				}
				creationPhase = false
				restartEvents = append(restartEvents, e)
			}
		} else {
			restartEvents = append(restartEvents, e)
		}
	}

	// Check pod restart events
	if len(restartEvents) == 0 {
		return false
	}
	var restartPodName *string
	for _, e := range restartEvents {
		podName := e.Object.(*corev1.Pod).Name

		switch e.Type {
		case watch.Added:
			if restartPodName == nil || *restartPodName != podName || podCreated[podName] {
				return false
			}
			podCreated[podName] = true
			restartPodName = nil
		case watch.Deleted:
			if restartPodName != nil || !podCreated[podName] {
				return false
			}
			restartPodName = ptr.To(podName)
			podCreated[podName] = false
		}
	}
	return true
}

func getTimeFromEvent(e watch.Event) time.Time {
	switch e.Type {
	case watch.Added:
		return e.Object.(*corev1.Pod).CreationTimestamp.Time
	case watch.Deleted:
		return e.Object.(*corev1.Pod).DeletionTimestamp.Time
	}
	return time.Time{}
}
