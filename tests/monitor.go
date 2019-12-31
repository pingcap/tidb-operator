// Copyright 2019 PingCAP, Inc.
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

package tests

import (
	"fmt"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/monitor/monitor"
	"github.com/pingcap/tidb-operator/tests/slack"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"time"
)

func (oa *operatorActions) DeployTidbMonitor(monitor *v1alpha1.TidbMonitor) error {
	namespace := monitor.Namespace
	_, err := oa.cli.PingcapV1alpha1().TidbMonitors(namespace).Create(monitor)
	if err != nil {
		return err
	}
	return nil
}

func (oa *operatorActions) DeployTidbMonitorOrDie(monitor *v1alpha1.TidbMonitor) {
	if err := oa.DeployTidbMonitor(monitor); err != nil {
		slack.NotifyAndPanic(err)
	}
}

func (oa *operatorActions) CheckTidbMonitor(tm *v1alpha1.TidbMonitor) error {
	namespace := tm.Namespace
	podName := monitor.GetMonitorObjectName(tm)
	svcName := fmt.Sprintf("%s-prometheus", tm.Name)
	return wait.Poll(5*time.Second, 20*time.Minute, func() (done bool, err error) {

		pod, err := oa.kubeCli.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		if err != nil {
			klog.Infof("tm[%s/%s]'s pod is failed to fetch", tm.Namespace, tm.Name)
			return false, nil
		}
		if !podutil.IsPodReady(pod) {
			klog.Infof("tm[%s/%s]'s pod[%s/%s] is not ready", tm.Namespace, tm.Name, pod.Namespace, pod.Name)
			return false, nil
		}
		if tm.Spec.Grafana != nil && len(pod.Spec.Containers) != 3 {
			return false, fmt.Errorf("tm[%s/%s]'s pod didn't have 3 containers with grafana enabled", tm.Namespace, tm.Name)
		}
		if tm.Spec.Grafana == nil && len(pod.Spec.Containers) != 2 {
			return false, fmt.Errorf("tm[%s/%s]'s pod didnt' have 2 containers with grafana disabled", tm.Namespace, tm.Name)
		}
		klog.Infof("tm[%s/%s]'s pod[%s/%s] is ready", tm.Namespace, tm.Name, pod.Namespace, pod.Name)
		_, err = oa.kubeCli.CoreV1().Services(namespace).Get(svcName, metav1.GetOptions{})
		if err != nil {
			klog.Infof("tm[%s/%s]'s service[%s/%s] failed to fetch", tm.Namespace, tm.Name, tm.Namespace, svcName)
			return false, nil
		}
		return true, err
	})
}

func (oa *operatorActions) CheckTidbMonitorOrDie(tm *v1alpha1.TidbMonitor) {
	if err := oa.CheckTidbMonitor(tm); err != nil {
		slack.NotifyAndPanic(err)
	}
}
