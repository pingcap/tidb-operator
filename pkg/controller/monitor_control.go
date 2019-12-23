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

package controller

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	glog "k8s.io/klog"
)

type TidbMonitorControlInterface interface {
	CreateTidbMonitor(monitor *v1alpha1.TidbMonitor) (*v1alpha1.TidbMonitor, error)
	DeleteTidbMonitor(monitor *v1alpha1.TidbMonitor) error
}

type realTidbMonitorControl struct {
	cli      versioned.Interface
	recorder record.EventRecorder
}

func NewTidbMonitorControl(
	cli versioned.Interface,
	recorder record.EventRecorder,
) TidbMonitorControlInterface {
	return &realTidbMonitorControl{cli: cli, recorder: recorder}
}

func (rmc *realTidbMonitorControl) CreateTidbMonitor(monitor *v1alpha1.TidbMonitor) (*v1alpha1.TidbMonitor, error) {
	name := monitor.Name
	namespace := monitor.Namespace

	monitor, err := rmc.cli.PingcapV1alpha1().TidbMonitors(namespace).Create(monitor)
	if err != nil {
		klog.Errorf("failed to create TidbMonitor: [%s/%s], err: %v", namespace, name, err)
	} else {
		klog.V(4).Infof("create Monitor: [%s/%s] successfully", namespace, name)
	}
	rmc.recordTidbMonitorEvent("create", monitor, err)
	return monitor, err
}

func (rmc *realTidbMonitorControl) DeleteTidbMonitor(monitor *v1alpha1.TidbMonitor) error {
	name := monitor.Name
	namespace := monitor.Namespace

	err := rmc.cli.PingcapV1alpha1().TidbMonitors(namespace).Delete(name, nil)
	if err != nil {
		klog.Errorf("failed to delete TidbMonitor: [%s/%s], err: %v", namespace, name, err)
	} else {
		glog.V(4).Infof("delete TidbMonitor: [%s/%s] successfully", namespace, name)
	}

	rmc.recordTidbMonitorEvent("delete", monitor, err)
	return err
}

func (rmc *realTidbMonitorControl) recordTidbMonitorEvent(verb string, monitor *v1alpha1.TidbMonitor, err error) {
	name := monitor.Name
	namespace := monitor.Namespace
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s TidbMonitor %s/%s for successful",
			strings.ToLower(verb), namespace, name)
		rmc.recorder.Event(monitor, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s TidbMonitor %s/%s for failed error: %s",
			strings.ToLower(verb), namespace, name, err)
		rmc.recorder.Event(monitor, corev1.EventTypeWarning, reason, msg)
	}
}
