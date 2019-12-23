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

package monitor

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"k8s.io/apimachinery/pkg/api/errors"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

type monitorManager struct {
	configMapListers corelisters.ConfigMapLister
	secretLister     corelisters.SecretLister
	deploymentLister appslisters.DeploymentLister
	stsControl       controller.StatefulSetControlInterface
	pvcControl       controller.GeneralPVCControlInterface
}

func (mm *monitorManager) Sync(monitor *v1alpha1.TidbMonitor) error {

	if monitor.DeletionTimestamp != nil {
		return nil
	}
	return nil
}

func (mm *monitorManager) syncTidbMonitor(monitor *v1alpha1.TidbMonitor) error {
	name := monitor.Name
	namespace := monitor.Namespace
	monitorConfigmapName := getMonitorConfigMapName(monitor)

	_, err := mm.configMapListers.ConfigMaps(namespace).Get(monitorConfigmapName)
	if err != nil {
		if errors.IsNotFound(err) {

		}
	}
}
