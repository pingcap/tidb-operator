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
	"k8s.io/kubernetes/test/e2e/framework"
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
		for _, pod := range pods {
			err = k.client.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
			if err != nil {
				framework.Logf("failed to delete pod %s/%s: %v", pod.Namespace, pod.Name, err)
			} else {
				framework.Logf("successfully deleted tidb-operator pod %s/%s", pod.Namespace, pod.Name)
			}
		}
	}, k.config.Interval, k.config.JitterFactor, true, stopCh)
}
