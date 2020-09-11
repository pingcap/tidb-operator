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

package operator

import (
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/utils/pointer"
)

// OperatorKillerConfig describes configuration for operator killer.
type OperatorKillerConfig struct {
	Enabled bool
	// Interval is time between operator failures.
	Interval time.Duration
	// Operator pods will be deleted between [Interval, Interval * (1.0 + JitterFactor)].
	JitterFactor float64
}

// OperatorKiller deletes pods of tidb-operator to simulate operator failures.
type OperatorKiller struct {
	config    OperatorKillerConfig
	client    kubernetes.Interface
	podLister func() ([]v1.Pod, error)
}

// NewOperatorKiller creates a new operator killer.
func NewOperatorKiller(config OperatorKillerConfig, client kubernetes.Interface, podLister func() ([]v1.Pod, error)) *OperatorKiller {
	return &OperatorKiller{
		config:    config,
		client:    client,
		podLister: podLister,
	}
}

// check the pod has been restarted or not
func hasBeenRestarted(pod *v1.Pod) bool {
	allContainerStatues := append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...)
	for _, containerStatus := range allContainerStatues {
		if containerStatus.RestartCount > 0 {
			return true
		}
	}
	return false
}

// Run starts OperatorKiller until stopCh is closed.
func (k *OperatorKiller) Run(stopCh <-chan struct{}) {
	// wait.JitterUntil starts work immediately, so wait first.
	time.Sleep(wait.Jitter(k.config.Interval, k.config.JitterFactor))
	wait.JitterUntil(func() {
		pods, err := k.podLister()
		if err != nil {
			framework.Logf("failed to list operator pods: %v", err)
			return
		}
		framework.Logf("Trying to kill tidb-operator pods (%d)", len(pods))
		for _, pod := range pods {
			if !podutil.IsPodReady(&pod) || hasBeenRestarted(&pod) {
				// deleting the pod will recreate it, we should skip if the pod
				// is not ready or has been restarted before, otherwise
				// potential errors (e.g. panic) in operator may be hidden.
				framework.Logf("pod %s/%s is not ready or crashed before, skip deleting", pod.Namespace, pod.Name)
				continue
			}
			err = k.client.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{
				GracePeriodSeconds: pointer.Int64Ptr(60),
			})
			if err != nil {
				framework.Logf("failed to delete pod %s/%s: %v", pod.Namespace, pod.Name, err)
			} else {
				framework.Logf("successfully deleted pod %s/%s", pod.Namespace, pod.Name)
			}
		}
	}, k.config.Interval, k.config.JitterFactor, true, stopCh)
}
