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
	corev1 "k8s.io/api/core/v1"
	kubeinformers "k8s.io/client-go/informers"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	// Sync Service
	if err := mm.syncTidbMonitorService(monitor); err != nil {
		return err
	}
	klog.Infof("tm[%s/%s]'s service synced", monitor.Namespace, monitor.Name)
	// Sync PVC
	if monitor.Spec.Persistent {
		if err := mm.syncTidbMonitorPVC(monitor); err != nil {
			return err
		}
	}
	klog.Infof("tm[%s/%s]'s pvc synced", monitor.Namespace, monitor.Name)
	// Sync Deployment
	return mm.syncTidbMonitorDeployment(monitor)
}

func (mm *MonitorManager) syncTidbMonitorService(monitor *v1alpha1.TidbMonitor) error {
	service := getMonitorService(monitor)
	for _, svc := range service {
		_, err := mm.typedControl.CreateOrUpdateService(monitor, svc)
		if err != nil {
			klog.Errorf("tm[%s/%s]'s service[%s] failed to sync,err: %v", monitor.Namespace, monitor.Name, svc.Name, err)
			return err
		}
	}
	return nil
}

func (mm *MonitorManager) syncTidbMonitorPVC(monitor *v1alpha1.TidbMonitor) error {
	pvcName := getMonitorObjectName(monitor)
	pvc := &corev1.PersistentVolumeClaim{}
	exist, err := mm.typedControl.Exist(client.ObjectKey{
		Name:      pvcName,
		Namespace: monitor.Namespace,
	}, pvc)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s pvc[%s] failed to sync,err: %v", monitor.Namespace, monitor.Name, pvc.Name, err)
		return err
	}
	if exist {
		return nil
	}
	pvc = getMonitorPVC(monitor)
	err = mm.typedControl.Create(monitor, pvc)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s pvc[%s] failed to sync,err: %v", monitor.Namespace, monitor.Name, pvc.Name, err)
		return err
	}
	return nil
}

func (mm *MonitorManager) syncTidbMonitorDeployment(monitor *v1alpha1.TidbMonitor) error {
	targetTcRef := monitor.Spec.Clusters[0]
	tc, err := mm.tcLister.TidbClusters(targetTcRef.Namespace).Get(targetTcRef.Name)
	if err != nil {
		return err
	}

	cm, err := mm.syncTidbMonitorConfig(tc, monitor)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s configmap failed to sync,err: %v", monitor.Namespace, monitor.Name, err)
		return err
	}
	secret, err := mm.syncTidbMonitorSecret(monitor)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s secret failed to sync,err: %v", monitor.Namespace, monitor.Name, err)
		return err
	}

	sa, err := mm.syncTidbMonitorRbac(monitor)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s rbac failed to sync,err: %v", monitor.Namespace, monitor.Name, err)
		return err
	}

	deployment := getMonitorDeployment(sa, cm, secret, monitor, tc)
	_, err = mm.typedControl.CreateOrUpdateDeployment(monitor, deployment)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s deployment failed to sync,err: %v", monitor.Namespace, monitor.Name, err)
		return err
	}
	klog.Infof("tm[%s/%s]'s deployment synced", monitor.Namespace, monitor.Name)
	return nil
}

func (mm *MonitorManager) syncTidbMonitorSecret(monitor *v1alpha1.TidbMonitor) (*corev1.Secret, error) {
	if monitor.Spec.Grafana == nil {
		return nil, nil
	}
	newSt := getMonitorSecret(monitor)
	return mm.typedControl.CreateOrUpdateSecret(monitor, newSt)
}

func (mm *MonitorManager) syncTidbMonitorConfig(tc *v1alpha1.TidbCluster, monitor *v1alpha1.TidbMonitor) (*corev1.ConfigMap, error) {

	newCM, err := getMonitorConfigMap(tc, monitor)
	if err != nil {
		return nil, err
	}
	return mm.typedControl.CreateOrUpdateConfigMap(monitor, newCM)
}

func (mm *MonitorManager) syncTidbMonitorRbac(monitor *v1alpha1.TidbMonitor) (*corev1.ServiceAccount, error) {
	sa := getMonitorServiceAccount(monitor)
	sa, err := mm.typedControl.CreateOrUpdateServiceAccount(monitor, sa)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s serviceaccount failed to sync,err: %v", monitor.Namespace, monitor.Name, err)
		return nil, err
	}
	cr := getMonitorClusterRole(monitor)
	cr, err = mm.typedControl.CreateOrUpdateClusterRole(monitor, cr)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s clusterrole failed to sync,err: %v", monitor.Namespace, monitor.Name, err)
		return nil, err
	}
	crb := getMonitorClusterRoleBinding(sa, cr, monitor)
	_, err = mm.typedControl.CreateOrUpdateClusterRoleBinding(monitor, crb)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s clusterRile failed to sync,err: %v", monitor.Namespace, monitor.Name, err)
		return nil, err
	}
	return sa, nil
}
