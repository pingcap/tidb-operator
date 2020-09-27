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
	"encoding/json"
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/informers/externalversions"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager/meta"
	"github.com/pingcap/tidb-operator/pkg/monitor"
	utildiscovery "github.com/pingcap/tidb-operator/pkg/util/discovery"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	discoverycachedmemory "k8s.io/client-go/discovery/cached/memory"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	extensionslister "k8s.io/client-go/listers/extensions/v1beta1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
)

type MonitorManager struct {
	cli                versioned.Interface
	pvManager          monitor.MonitorManager
	discoveryInterface discovery.CachedDiscoveryInterface
	typedControl       controller.TypedControlInterface
	deploymentLister   appslisters.DeploymentLister
	pvLister           corelisters.PersistentVolumeLister
	ingressLister      extensionslister.IngressLister
	pvControl          controller.PVControlInterface
	cmControl          controller.ConfigMapControlInterface
	recorder           record.EventRecorder
}

const (
	FailedSync  = "FailedSync"
	SuccessSync = "SuccessSync"
)

func NewMonitorManager(
	kubeCli kubernetes.Interface,
	cli versioned.Interface,
	informerFactory informers.SharedInformerFactory,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	typedControl controller.TypedControlInterface,
	recorder record.EventRecorder) *MonitorManager {
	pvcLister := kubeInformerFactory.Core().V1().PersistentVolumeClaims().Lister()
	pvLister := kubeInformerFactory.Core().V1().PersistentVolumes().Lister()
	pvControl := controller.NewRealPVControl(kubeCli, pvcLister, pvLister, recorder)
	cmControl := controller.NewRealConfigMapControl(kubeCli, recorder)
	return &MonitorManager{
		cli: cli,
		pvManager: meta.NewReclaimPolicyMonitorManager(
			pvcLister,
			pvLister,
			pvControl),
		discoveryInterface: discoverycachedmemory.NewMemCacheClient(kubeCli.Discovery()),
		typedControl:       typedControl,
		deploymentLister:   kubeInformerFactory.Apps().V1().Deployments().Lister(),
		pvControl:          controller.NewRealPVControl(kubeCli, pvcLister, pvLister, recorder),
		pvLister:           pvLister,
		ingressLister:      kubeInformerFactory.Extensions().V1beta1().Ingresses().Lister(),
		cmControl:          cmControl,
		recorder:           recorder,
	}
}

func (mm *MonitorManager) SyncMonitor(monitor *v1alpha1.TidbMonitor) error {
	if monitor.DeletionTimestamp != nil {
		return nil
	}
	if monitor.Spec.Clusters == nil || len(monitor.Spec.Clusters) < 1 {
		err := fmt.Errorf("tm[%s/%s] does not configure the target tidbcluster", monitor.Namespace, monitor.Name)
		return err
	}
	defaultTidbMonitor(monitor)
	tcRef := monitor.Spec.Clusters[0]
	if len(tcRef.Namespace) < 1 {
		tcRef.Namespace = monitor.Namespace
	}
	tc, err := mm.cli.PingcapV1alpha1().TidbClusters(tcRef.Namespace).Get(tcRef.Name, metav1.GetOptions{})
	if err != nil {
		rerr := fmt.Errorf("get tm[%s/%s]'s target tc[%s/%s] failed, err: %v", monitor.Namespace, monitor.Name, tcRef.Namespace, tcRef.Name, err)
		return rerr
	}
	if tc.Status.Monitor != nil {
		if tc.Status.Monitor.Name != monitor.Name || tc.Status.Monitor.Namespace != monitor.Namespace {
			err := fmt.Errorf("tm[%s/%s]'s target tc[%s/%s] already referenced by TidbMonitor [%s/%s]", monitor.Namespace, monitor.Name, tc.Namespace, tc.Name, tc.Status.Monitor.Namespace, tc.Status.Monitor.Name)
			mm.recorder.Event(monitor, corev1.EventTypeWarning, FailedSync, err.Error())
			return err
		}
	}

	// TODO: Support validating webhook that forbids the tidbmonitor to update the monitorRef for the tidbcluster whose monitorRef has already
	// been set by another TidbMonitor.
	// Patch tidbcluster status first to avoid multi tidbmonitor monitoring the same tidbcluster
	if err := mm.patchTidbClusterStatus(&tcRef, monitor); err != nil {
		message := fmt.Sprintf("Sync TidbMonitorRef into targetCluster[%s/%s] status failed, err:%v", tc.Namespace, tc.Name, err)
		klog.Error(message)
		mm.recorder.Event(monitor, corev1.EventTypeWarning, FailedSync, err.Error())
		return err
	}

	// Sync Service
	if err := mm.syncTidbMonitorService(monitor); err != nil {
		message := fmt.Sprintf("Sync TidbMonitor[%s/%s] Service failed, err: %v", monitor.Namespace, monitor.Name, err)
		mm.recorder.Event(monitor, corev1.EventTypeWarning, FailedSync, message)
		return err
	}
	klog.V(4).Infof("tm[%s/%s]'s service synced", monitor.Namespace, monitor.Name)
	// Sync PVC
	var pvc *corev1.PersistentVolumeClaim
	if monitor.Spec.Persistent {
		var err error
		pvc, err = mm.syncTidbMonitorPVC(monitor)
		if err != nil {
			message := fmt.Sprintf("Sync TidbMonitor[%s/%s] PVC failed,err:%v", monitor.Namespace, monitor.Name, err)
			mm.recorder.Event(monitor, corev1.EventTypeWarning, FailedSync, message)
			return err
		}
		klog.V(4).Infof("tm[%s/%s]'s pvc synced", monitor.Namespace, monitor.Name)

		// syncing all PVs managed by this tidbmonitor
		if err := mm.pvManager.SyncMonitor(monitor); err != nil {
			return err
		}
		klog.V(4).Infof("tm[%s/%s]'s pv synced", monitor.Namespace, monitor.Name)
	}

	// Sync Deployment
	if err := mm.syncTidbMonitorDeployment(tc, monitor); err != nil {
		message := fmt.Sprintf("Sync TidbMonitor[%s/%s] Deployment failed,err:%v", monitor.Namespace, monitor.Name, err)
		mm.recorder.Event(monitor, corev1.EventTypeWarning, FailedSync, message)
		return err
	}

	// After the pvc has consumer, we sync monitor pv's labels
	if monitor.Spec.Persistent {
		if err := mm.syncTidbMonitorPV(monitor, pvc); err != nil {
			return err
		}
	}
	klog.V(4).Infof("tm[%s/%s]'s deployment synced", monitor.Namespace, monitor.Name)

	// Sync Ingress
	if err := mm.syncIngress(monitor); err != nil {
		message := fmt.Sprintf("Sync TidbMonitor[%s/%s] Ingress failed,err:%v", monitor.Namespace, monitor.Name, err)
		mm.recorder.Event(monitor, corev1.EventTypeWarning, FailedSync, message)
		return err
	}
	klog.V(4).Infof("tm[%s/%s]'s ingress synced", monitor.Namespace, monitor.Name)

	return nil
}

func (mm *MonitorManager) syncTidbMonitorService(monitor *v1alpha1.TidbMonitor) error {
	services := getMonitorService(monitor)
	for _, svc := range services {
		_, err := mm.typedControl.CreateOrUpdateService(monitor, svc)
		if err != nil {
			klog.Errorf("tm[%s/%s]'s service[%s] failed to sync,err: %v", monitor.Namespace, monitor.Name, svc.Name, err)
			return controller.RequeueErrorf("tm[%s/%s]'s service[%s] failed to sync,err: %v", monitor.Namespace, monitor.Name, svc.Name, err)
		}
	}
	return nil
}

func (mm *MonitorManager) syncTidbMonitorPVC(monitor *v1alpha1.TidbMonitor) (*corev1.PersistentVolumeClaim, error) {

	pvc := getMonitorPVC(monitor)
	pvc, err := mm.typedControl.CreateOrUpdatePVC(monitor, pvc, false)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s pvc failed to sync,err: %v", monitor.Namespace, monitor.Name, err)
		return nil, err
	}
	return pvc, nil
}

func (mm *MonitorManager) syncTidbMonitorPV(monitor *v1alpha1.TidbMonitor, pvc *corev1.PersistentVolumeClaim) error {
	// update meta info for pv
	pv, err := mm.pvLister.Get(pvc.Spec.VolumeName)
	if err != nil {
		return err
	}
	_, err = mm.pvControl.UpdateMetaInfo(monitor, pv)
	if err != nil {
		return err
	}
	return nil
}

func (mm *MonitorManager) syncTidbMonitorDeployment(tc *v1alpha1.TidbCluster, monitor *v1alpha1.TidbMonitor) error {

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

	deployment, err := getMonitorDeployment(sa, cm, secret, monitor, tc)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s deployment failed to generate,err: %v", monitor.Namespace, monitor.Name, err)
		return err
	}
	_, err = mm.typedControl.CreateOrUpdateDeployment(monitor, deployment)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s deployment failed to sync,err: %v", monitor.Namespace, monitor.Name, err)
		return err
	}
	klog.V(4).Infof("tm[%s/%s]'s deployment synced", monitor.Namespace, monitor.Name)
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
	if features.DefaultFeatureGate.Enabled(features.AutoScaling) {
		// Get all autoscaling clusters for TC, and add them to .Spec.Clusters to
		// generate Prometheus config without modifying the original TidbMonitor
		cloned := monitor.DeepCopy()
		for _, tcRef := range monitor.Spec.Clusters {
			r1, err := labels.NewRequirement(label.AutoInstanceLabelKey, selection.Exists, nil)
			if err != nil {
				klog.Errorf("tm[%s/%s] gets tc[%s/%s]'s autoscaling clusters failed, err: %v", monitor.Namespace, monitor.Name, tcRef.Namespace, tcRef.Name, err)
			}
			r2, err := labels.NewRequirement(label.BaseTCLabelKey, selection.Equals, []string{tcRef.Name})
			if err != nil {
				klog.Errorf("tm[%s/%s] gets tc[%s/%s]'s autoscaling clusters failed, err: %v", monitor.Namespace, monitor.Name, tcRef.Namespace, tcRef.Name, err)
			}
			selector := labels.NewSelector().Add(*r1).Add(*r2)
			tcList, err := mm.cli.PingcapV1alpha1().TidbClusters(tcRef.Namespace).List(metav1.ListOptions{LabelSelector: selector.String()})
			if err != nil {
				klog.Errorf("tm[%s/%s] gets tc[%s/%s]'s autoscaling clusters failed, err: %v", monitor.Namespace, monitor.Name, tcRef.Namespace, tcRef.Name, err)
			}
			for _, autoTc := range tcList.Items {
				cloned.Spec.Clusters = append(cloned.Spec.Clusters, v1alpha1.TidbClusterRef{
					Name:      autoTc.Name,
					Namespace: autoTc.Namespace,
				})
			}
		}
		monitor = cloned
	}

	newCM, err := getMonitorConfigMap(tc, monitor)
	if err != nil {
		return nil, err
	}
	config := monitor.Spec.Prometheus.Config
	if config != nil && config.ConfigMapRef != nil && len(config.ConfigMapRef.Name) > 0 {
		namespace := monitor.Namespace
		if config.ConfigMapRef.Namespace != nil {
			namespace = *config.ConfigMapRef.Namespace
		}
		externalCM, err := mm.cmControl.GetConfigMap(monitor, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.ConfigMapRef.Name,
				Namespace: namespace,
			},
		})
		if err != nil {
			klog.Errorf("tm[%s/%s]'s configMap failed to get,err: %v", namespace, config.ConfigMapRef.Name, err)
			return nil, err
		}
		if externalContent, ok := externalCM.Data["prometheus-config"]; ok {
			newCM.Data["prometheus-config"] = externalContent
		}
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
	policyRules := []rbac.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get", "list", "watch"},
		},
	}
	if supported, err := utildiscovery.IsAPIGroupVersionSupported(mm.discoveryInterface, "security.openshift.io/v1"); err != nil {
		return nil, err
	} else if supported {
		// We must use 'anyuid' SecurityContextConstraint to run our container as root.
		// https://docs.openshift.com/container-platform/4.3/authentication/managing-security-context-constraints.html
		policyRules = append(policyRules, rbac.PolicyRule{
			APIGroups:     []string{"security.openshift.io"},
			ResourceNames: []string{"anyuid"},
			Resources:     []string{"securitycontextconstraints"},
			Verbs:         []string{"use"},
		})
	}

	role := getMonitorRole(monitor, policyRules)
	role, err = mm.typedControl.CreateOrUpdateRole(monitor, role)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s role failed to sync,err: %v", monitor.Namespace, monitor.Name, err)
		return nil, err
	}

	rb := getMonitorRoleBinding(sa, role, monitor)
	_, err = mm.typedControl.CreateOrUpdateRoleBinding(monitor, rb)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s rolebinding failed to sync,err: %v", monitor.Namespace, monitor.Name, err)
		return nil, err
	}

	return sa, nil
}

func (mm *MonitorManager) syncIngress(monitor *v1alpha1.TidbMonitor) error {
	if err := mm.syncPrometheusIngress(monitor); err != nil {
		return err
	}

	return mm.syncGrafanaIngress(monitor)
}

func (mm *MonitorManager) syncPrometheusIngress(monitor *v1alpha1.TidbMonitor) error {
	if monitor.Spec.Prometheus.Ingress == nil {
		return mm.removeIngressIfExist(monitor, prometheusName(monitor))
	}

	ingress := getPrometheusIngress(monitor)
	_, err := mm.typedControl.CreateOrUpdateIngress(monitor, ingress)
	return err
}

func (mm *MonitorManager) syncGrafanaIngress(monitor *v1alpha1.TidbMonitor) error {
	if monitor.Spec.Grafana == nil || monitor.Spec.Grafana.Ingress == nil {
		return mm.removeIngressIfExist(monitor, grafanaName(monitor))
	}
	ingress := getGrafanaIngress(monitor)
	_, err := mm.typedControl.CreateOrUpdateIngress(monitor, ingress)
	return err
}

// removeIngressIfExist removes Ingress if it exists
func (mm *MonitorManager) removeIngressIfExist(monitor *v1alpha1.TidbMonitor, name string) error {
	ingress, err := mm.ingressLister.Ingresses(monitor.Namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return mm.typedControl.Delete(monitor, ingress)
}

func (mm *MonitorManager) patchTidbClusterStatus(tcRef *v1alpha1.TidbClusterRef, monitor *v1alpha1.TidbMonitor) error {
	tc, err := mm.cli.PingcapV1alpha1().TidbClusters(tcRef.Namespace).Get(tcRef.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	var mergePatch []byte
	if tcRef != nil {
		grafanaEnabled := true
		if monitor.Spec.Grafana == nil {
			grafanaEnabled = false
		}
		mergePatch, err = json.Marshal(map[string]interface{}{
			"status": map[string]interface{}{
				"monitor": map[string]interface{}{
					"name":           monitor.Name,
					"namespace":      monitor.Namespace,
					"grafanaEnabled": grafanaEnabled,
				},
			},
		})
	} else {
		mergePatch, err = json.Marshal(map[string]interface{}{
			"status": map[string]interface{}{
				"monitor": nil,
			},
		})
	}
	if err != nil {
		return err
	}
	_, err = mm.cli.PingcapV1alpha1().TidbClusters(tc.Namespace).Patch(tc.Name, types.MergePatchType, mergePatch)
	return err
}
