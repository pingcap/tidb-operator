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
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	v1alpha1listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	core "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	kubeinformers "k8s.io/client-go/informers"
	appslisters "k8s.io/client-go/listers/apps/v1"
)

type MonitorManager struct {
	typedControl     controller.TypedControlInterface
	deploymentLister appslisters.DeploymentLister
	tcLister         v1alpha1listers.TidbClusterLister
}

func NewMonitorManager(
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	typedControl controller.TypedControlInterface) *MonitorManager {
	return &MonitorManager{
		typedControl:     typedControl,
		deploymentLister: kubeInformerFactory.Apps().V1().Deployments().Lister(),
		tcLister:         informerFactory.Pingcap().V1alpha1().TidbClusters().Lister(),
	}
}

func (mm *MonitorManager) Sync(monitor *v1alpha1.TidbMonitor) error {

	if monitor.DeletionTimestamp != nil {
		return nil
	}
	return mm.syncTidbMonitor(monitor)
}

func (mm *MonitorManager) syncTidbMonitor(monitor *v1alpha1.TidbMonitor) error {
	//name := monitor.Name
	namespace := monitor.Namespace
	targetTcRef := monitor.Spec.Clusters[0]
	tc, err := mm.tcLister.TidbClusters(targetTcRef.Namespace).Get(targetTcRef.Name)
	if err != nil {
		return err
	}

	oldMonitorDeployTmp, err := mm.deploymentLister.Deployments(namespace).Get(getMonitorObjectName(monitor))
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	deployNotExist := errors.IsNotFound(err)

	oldMonitorDeploy := oldMonitorDeployTmp.DeepCopy()

	cm, err := mm.syncTidbMonitorConfig(tc, monitor)
	if err != nil {
		return err
	}
	secret, err := mm.syncTidbMonitorSecret(monitor)
	if err != nil {
		return err
	}

	sa, err := mm.syncTidbMonitorRbac(monitor)
	if err != nil {
		return err
	}
	deployment := getMonitorDeployment(sa, cm, secret, monitor, tc)
	if deployNotExist || !apiequality.Semantic.DeepEqual(oldMonitorDeploy.Spec.Template.Spec, deployment.Spec.Template.Spec) {
		deployment, err = mm.typedControl.CreateOrUpdateDeployment(monitor, deployment)
		if err != nil {
			return err
		}
	}
	return nil
}

func (mm *MonitorManager) syncTidbMonitorSecret(monitor *v1alpha1.TidbMonitor) (*core.Secret, error) {
	if monitor.Spec.Grafana == nil {
		return nil, nil
	}
	newSt := getMonitorSecret(monitor)
	return mm.typedControl.CreateOrUpdateSecret(monitor, newSt)
}

func (mm *MonitorManager) syncTidbMonitorConfig(tc *v1alpha1.TidbCluster, monitor *v1alpha1.TidbMonitor) (*core.ConfigMap, error) {

	newCM, err := getMonitorConfigMap(tc, monitor)
	if err != nil {
		return nil, err
	}
	return mm.typedControl.CreateOrUpdateConfigMap(monitor, newCM)
}

func (mm *MonitorManager) syncTidbMonitorRbac(monitor *v1alpha1.TidbMonitor) (*core.ServiceAccount, error) {
	sa := getMonitorServiceAccount(monitor)
	sa, err := mm.typedControl.CreateOrUpdateServiceAccount(monitor, sa)
	if err != nil {
		return nil, err
	}
	cr := getMonitorClusterRole(monitor)
	cr, err = mm.typedControl.CreateOrUpdateClusterRole(monitor, cr)
	if err != nil {
		return nil, err
	}
	crb := getMonitorClusterRoleBinding(sa, cr, monitor)
	_, err = mm.typedControl.CreateOrUpdateClusterRoleBinding(monitor, crb)
	if err != nil {
		return nil, err
	}
	return sa, nil
}
