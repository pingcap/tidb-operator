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
	"sort"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager/meta"
	"github.com/pingcap/tidb-operator/pkg/monitor"
	utildiscovery "github.com/pingcap/tidb-operator/pkg/util/discovery"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	discoverycachedmemory "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MonitorManager struct {
	deps               *controller.Dependencies
	pvManager          monitor.MonitorManager
	discoveryInterface discovery.CachedDiscoveryInterface
}

const (
	FailedSync  = "FailedSync"
	SuccessSync = "SuccessSync"
)

func NewMonitorManager(deps *controller.Dependencies) *MonitorManager {
	return &MonitorManager{
		deps:               deps,
		pvManager:          meta.NewReclaimPolicyManager(deps),
		discoveryInterface: discoverycachedmemory.NewMemCacheClient(deps.KubeClientset.Discovery()),
	}
}

func (m *MonitorManager) SyncMonitor(monitor *v1alpha1.TidbMonitor) error {
	// syncing all PVs managed by operator's reclaim policy to Retain
	if err := m.pvManager.SyncMonitor(monitor); err != nil {
		return err
	}

	if monitor.DeletionTimestamp != nil {
		return nil
	}
	if monitor.Spec.Clusters == nil || len(monitor.Spec.Clusters) < 1 {
		err := fmt.Errorf("tm[%s/%s] does not configure the target tidbcluster", monitor.Namespace, monitor.Name)
		return err
	}
	defaultTidbMonitor(monitor)
	var firstTc *v1alpha1.TidbCluster
	for index, tcRef := range monitor.Spec.Clusters {
		tc, err := m.deps.Clientset.PingcapV1alpha1().TidbClusters(tcRef.Namespace).Get(tcRef.Name, metav1.GetOptions{})
		if index == 0 {
			firstTc = tc
		}
		if err != nil {
			rerr := fmt.Errorf("get tm[%s/%s]'s target tc[%s/%s] failed, err: %v", monitor.Namespace, monitor.Name, tcRef.Namespace, tcRef.Name, err)
			return rerr
		}
		if tc.Status.Monitor != nil {
			if tc.Status.Monitor.Name != monitor.Name || tc.Status.Monitor.Namespace != monitor.Namespace {
				err := fmt.Errorf("tm[%s/%s]'s target tc[%s/%s] already referenced by TidbMonitor [%s/%s]", monitor.Namespace, monitor.Name, tc.Namespace, tc.Name, tc.Status.Monitor.Namespace, tc.Status.Monitor.Name)
				m.deps.Recorder.Event(monitor, corev1.EventTypeWarning, FailedSync, err.Error())
				return err
			}
		}
		// TODO: Support validating webhook that forbids the tidbmonitor to update the monitorRef for the tidbcluster whose monitorRef has already
		// been set by another TidbMonitor.
		// Patch tidbcluster status first to avoid multi tidbmonitor monitoring the same tidbcluster
		if err := m.patchTidbClusterStatus(&tcRef, monitor); err != nil {
			message := fmt.Sprintf("Sync TidbMonitorRef into targetCluster[%s/%s] status failed, err:%v", tc.Namespace, tc.Name, err)
			klog.Error(message)
			m.deps.Recorder.Event(monitor, corev1.EventTypeWarning, FailedSync, err.Error())
			return err
		}
	}

	// Sync Service
	if err := m.syncTidbMonitorService(monitor); err != nil {
		message := fmt.Sprintf("Sync TidbMonitor[%s/%s] Service failed, err: %v", monitor.Namespace, monitor.Name, err)
		m.deps.Recorder.Event(monitor, corev1.EventTypeWarning, FailedSync, message)
		return err
	}
	klog.V(4).Infof("tm[%s/%s]'s service synced", monitor.Namespace, monitor.Name)

	// Sync Statefulset
	if err := m.syncTidbMonitorStatefulset(firstTc, monitor); err != nil {
		message := fmt.Sprintf("Sync TidbMonitor[%s/%s] Deployment failed,err:%v", monitor.Namespace, monitor.Name, err)
		m.deps.Recorder.Event(monitor, corev1.EventTypeWarning, FailedSync, message)
		return err
	}

	klog.V(4).Infof("tm[%s/%s]'s deployment synced", monitor.Namespace, monitor.Name)

	// Sync Ingress
	if err := m.syncIngress(monitor); err != nil {
		message := fmt.Sprintf("Sync TidbMonitor[%s/%s] Ingress failed,err:%v", monitor.Namespace, monitor.Name, err)
		m.deps.Recorder.Event(monitor, corev1.EventTypeWarning, FailedSync, message)
		return err
	}
	klog.V(4).Infof("tm[%s/%s]'s ingress synced", monitor.Namespace, monitor.Name)

	return nil
}

func (m *MonitorManager) syncTidbMonitorService(monitor *v1alpha1.TidbMonitor) error {
	services := getMonitorService(monitor)
	for _, svc := range services {
		_, err := m.deps.TypedControl.CreateOrUpdateService(monitor, svc)
		if err != nil {
			klog.Errorf("tm[%s/%s]'s service[%s] failed to sync,err: %v", monitor.Namespace, monitor.Name, svc.Name, err)
			return controller.RequeueErrorf("tm[%s/%s]'s service[%s] failed to sync,err: %v", monitor.Namespace, monitor.Name, svc.Name, err)
		}
	}
	return nil
}

func (m *MonitorManager) syncTidbMonitorStatefulset(tc *v1alpha1.TidbCluster, monitor *v1alpha1.TidbMonitor) error {

	cm, err := m.syncTidbMonitorConfig(tc, monitor)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s configmap failed to sync,err: %v", monitor.Namespace, monitor.Name, err)
		return err
	}
	secret, err := m.syncTidbMonitorSecret(monitor)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s secret failed to sync,err: %v", monitor.Namespace, monitor.Name, err)
		return err
	}

	sa, err := m.syncTidbMonitorRbac(monitor)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s rbac failed to sync,err: %v", monitor.Namespace, monitor.Name, err)
		return err
	}

	result, err := m.smoothMigrationToStatefulSet(monitor)
	if err != nil {
		klog.Errorf("Fail to migrate from deployment to statefulset for tm [%s/%s], err: %v", monitor.Namespace, monitor.Name, err)
		return err
	}
	if !result {
		klog.Errorf("Wait for the smooth migration to be done successfully for tm [%s/%s], err: %v", monitor.Namespace, monitor.Name, err)
		return nil
	}
	statefulset, err := getMonitorStatefulSet(sa, cm, secret, monitor, tc)
	if err != nil {
		klog.Errorf("Fail to generate statefulset for tm [%s/%s], err: %v", monitor.Namespace, monitor.Name, err)
		return err
	}

	_, err = m.deps.TypedControl.CreateOrUpdateStatefulSet(monitor, statefulset)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s deployment failed to sync,err: %v", monitor.Namespace, monitor.Name, err)
		return err
	}
	klog.V(4).Infof("tm[%s/%s]'s deployment synced", monitor.Namespace, monitor.Name)
	return nil
}

func (m *MonitorManager) syncTidbMonitorSecret(monitor *v1alpha1.TidbMonitor) (*corev1.Secret, error) {
	if monitor.Spec.Grafana == nil {
		return nil, nil
	}
	newSt := getMonitorSecret(monitor)
	return m.deps.TypedControl.CreateOrUpdateSecret(monitor, newSt)
}

func (m *MonitorManager) syncTidbMonitorConfig(tc *v1alpha1.TidbCluster, monitor *v1alpha1.TidbMonitor) (*corev1.ConfigMap, error) {
	if features.DefaultFeatureGate.Enabled(features.AutoScaling) {
		// TODO: We need to update the status to tell users we are monitoring extra clusters
		// Get all autoscaling clusters for TC, and add them to .Spec.Clusters to
		// generate Prometheus config without modifying the original TidbMonitor
		cloned := monitor.DeepCopy()
		autoTcRefs := []v1alpha1.TidbClusterRef{}
		for _, tcRef := range monitor.Spec.Clusters {
			r1, err := labels.NewRequirement(label.AutoInstanceLabelKey, selection.Exists, nil)
			if err != nil {
				klog.Errorf("tm[%s/%s] gets tc[%s/%s]'s autoscaling clusters failed, err: %v", monitor.Namespace, monitor.Name, tcRef.Namespace, tcRef.Name, err)
				continue
			}
			r2, err := labels.NewRequirement(label.BaseTCLabelKey, selection.Equals, []string{tcRef.Name})
			if err != nil {
				klog.Errorf("tm[%s/%s] gets tc[%s/%s]'s autoscaling clusters failed, err: %v", monitor.Namespace, monitor.Name, tcRef.Namespace, tcRef.Name, err)
				continue
			}
			selector := labels.NewSelector().Add(*r1).Add(*r2)
			tcList, err := m.deps.Clientset.PingcapV1alpha1().TidbClusters(tcRef.Namespace).List(metav1.ListOptions{LabelSelector: selector.String()})
			if err != nil {
				klog.Errorf("tm[%s/%s] gets tc[%s/%s]'s autoscaling clusters failed, err: %v", monitor.Namespace, monitor.Name, tcRef.Namespace, tcRef.Name, err)
				continue
			}
			for _, autoTc := range tcList.Items {
				autoTcRefs = append(autoTcRefs, v1alpha1.TidbClusterRef{
					Name:      autoTc.Name,
					Namespace: autoTc.Namespace,
				})
			}
		}
		// Sort Autoscaling TC for stability
		sort.Slice(autoTcRefs, func(i, j int) bool {
			cmpNS := strings.Compare(autoTcRefs[i].Namespace, autoTcRefs[j].Namespace)
			if cmpNS == 0 {
				return strings.Compare(autoTcRefs[i].Name, autoTcRefs[j].Name) < 0
			}
			return cmpNS < 0
		})

		cloned.Spec.Clusters = append(cloned.Spec.Clusters, autoTcRefs...)
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
		externalCM, err := m.deps.ConfigMapControl.GetConfigMap(monitor, &corev1.ConfigMap{
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
	return m.deps.TypedControl.CreateOrUpdateConfigMap(monitor, newCM)
}

func (m *MonitorManager) syncTidbMonitorRbac(monitor *v1alpha1.TidbMonitor) (*corev1.ServiceAccount, error) {
	sa := getMonitorServiceAccount(monitor)
	sa, err := m.deps.TypedControl.CreateOrUpdateServiceAccount(monitor, sa)
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
	if supported, err := utildiscovery.IsAPIGroupVersionSupported(m.discoveryInterface, "security.openshift.io/v1"); err != nil {
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

	if monitor.Spec.ClusterScoped {
		role := getMonitorClusterRole(monitor, policyRules)
		role, err = m.deps.TypedControl.CreateOrUpdateClusterRole(monitor, role)
		if err != nil {
			klog.Errorf("tm[%s/%s]'s clusterrole failed to sync, err: %v", monitor.Namespace, monitor.Name, err)
			return nil, err
		}

		rb := getMonitorClusterRoleBinding(sa, role, monitor)

		_, err = m.deps.TypedControl.CreateOrUpdateClusterRoleBinding(monitor, rb)
		if err != nil {
			klog.Errorf("tm[%s/%s]'s clusterrolebinding failed to sync, err: %v", monitor.Namespace, monitor.Name, err)
			return nil, err
		}
	} else {
		role := getMonitorRole(monitor, policyRules)
		role, err = m.deps.TypedControl.CreateOrUpdateRole(monitor, role)
		if err != nil {
			klog.Errorf("tm[%s/%s]'s role failed to sync,err: %v", monitor.Namespace, monitor.Name, err)
			return nil, err
		}

		rb := getMonitorRoleBinding(sa, role, monitor)

		_, err = m.deps.TypedControl.CreateOrUpdateRoleBinding(monitor, rb)
		if err != nil {
			klog.Errorf("tm[%s/%s]'s rolebinding failed to sync,err: %v", monitor.Namespace, monitor.Name, err)
			return nil, err
		}
	}

	return sa, nil
}

func (m *MonitorManager) syncIngress(monitor *v1alpha1.TidbMonitor) error {
	if err := m.syncPrometheusIngress(monitor); err != nil {
		return err
	}

	return m.syncGrafanaIngress(monitor)
}

func (m *MonitorManager) syncPrometheusIngress(monitor *v1alpha1.TidbMonitor) error {
	if monitor.Spec.Prometheus.Ingress == nil {
		return m.removeIngressIfExist(monitor, prometheusName(monitor))
	}

	ingress := getPrometheusIngress(monitor)
	_, err := m.deps.TypedControl.CreateOrUpdateIngress(monitor, ingress)
	return err
}

func (m *MonitorManager) syncGrafanaIngress(monitor *v1alpha1.TidbMonitor) error {
	if monitor.Spec.Grafana == nil || monitor.Spec.Grafana.Ingress == nil {
		return m.removeIngressIfExist(monitor, grafanaName(monitor))
	}
	ingress := getGrafanaIngress(monitor)
	_, err := m.deps.TypedControl.CreateOrUpdateIngress(monitor, ingress)
	return err
}

// removeIngressIfExist removes Ingress if it exists
func (m *MonitorManager) removeIngressIfExist(monitor *v1alpha1.TidbMonitor, name string) error {
	ingress, err := m.deps.IngressLister.Ingresses(monitor.Namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return m.deps.TypedControl.Delete(monitor, ingress)
}

func (m *MonitorManager) patchTidbClusterStatus(tcRef *v1alpha1.TidbClusterRef, monitor *v1alpha1.TidbMonitor) error {
	tc, err := m.deps.Clientset.PingcapV1alpha1().TidbClusters(tcRef.Namespace).Get(tcRef.Name, metav1.GetOptions{})
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
	_, err = m.deps.Clientset.PingcapV1alpha1().TidbClusters(tc.Namespace).Patch(tc.Name, types.MergePatchType, mergePatch)
	return err
}

func (m *MonitorManager) smoothMigrationToStatefulSet(monitor *v1alpha1.TidbMonitor) (bool, error) {
	// determine whether there is an old deployment
	deploymentName := GetMonitorObjectName(monitor)
	exist, err := m.deps.TypedControl.Exist(client.ObjectKey{
		Namespace: monitor.Namespace,
		Name:      deploymentName,
	}, &appsv1.Deployment{})
	if err != nil {
		klog.Errorf("Fail to get deployment for tm [%s/%s], err: %v", monitor.Namespace, monitor.Name, err)
		return false, err
	}
	if exist {
		klog.Infof("The deployment exists, start smooth migration for tm [%s/%s]", monitor.Namespace, monitor.Name)
		// if deployment exist, delete it and wait next reconcile.
		err = m.deps.TypedControl.Delete(monitor, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deploymentName,
				Namespace: monitor.Namespace,
			},
		})
		if err != nil {
			klog.Errorf("smoothMigration tm[%s/%s]'s,old deployment failed to delete,err: %v", monitor.Namespace, monitor.Name, err)
			return false, err
		}

		return !monitor.Spec.Persistent, nil
	} else {
		klog.Infof("smoothMigration tm[%s/%s]'s,old deployment is not exist", monitor.Namespace, monitor.Name)
		if monitor.Spec.Persistent {
			deploymentPvcName := GetMonitorObjectName(monitor)
			deploymentPvc, err := m.deps.PVCLister.PersistentVolumeClaims(monitor.Namespace).Get(deploymentPvcName)
			if err != nil {
				// If old deployment pvc not found ,it's not need to migrate.
				if errors.IsNotFound(err) {
					klog.Infof("smoothMigration tm[%s/%s]'s,old deployment pvc is not exist", monitor.Namespace, monitor.Name)
					return true, nil
				}
				return false, err
			}

			err = m.deps.TypedControl.Delete(monitor, &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      deploymentPvcName,
					Namespace: monitor.Namespace,
				},
			})
			if err != nil {
				klog.Errorf("Fail to delete the deployment for tm [%s/%s], err: %v", monitor.Namespace, monitor.Name, err)
				return false, err
			}
			stsPvcName := fmt.Sprintf("monitor-data-%s-0", GetMonitorObjectName(monitor))
			deploymentPv, err := m.deps.PVLister.Get(deploymentPvc.Spec.VolumeName)
			if err != nil && !errors.IsNotFound(err) {
				klog.Errorf("smoothMigration tm[%s/%s]'s,old deployment pv failed to get,err: %v", monitor.Namespace, monitor.Name, err)
				return false, err
			}
			deploymentPv.Spec.ClaimRef.Name = stsPvcName
			err = m.deps.PVControl.PatchPVClaimRef(monitor, deploymentPv, stsPvcName)
			if err != nil {
				klog.Errorf("smoothMigration tm[%s/%s]'s,failed to patch old deployment pv,err: %v", monitor.Namespace, monitor.Name, err)
				return false, err
			}
			//create new statefulSet pvc at advance.
			stsPvc := getMonitorPVC(stsPvcName, monitor)
			stsPvc.Spec.VolumeName = deploymentPv.Name
			_, err = m.deps.TypedControl.CreateOrUpdatePVC(monitor, stsPvc, false)

			if err != nil {
				klog.Errorf("smoothMigration tm[%s/%s]'s,failed to create new sts pvc,err: %v", monitor.Namespace, monitor.Name, err)
				return false, err
			}
			klog.Infof("smoothMigration tm[%s/%s]'s successfully", monitor.Namespace, monitor.Name)
			return true, nil

		}

	}
	return true, nil
}
