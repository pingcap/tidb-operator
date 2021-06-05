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
	v1alpha1validation "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1/validation"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/features"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager/member"
	"github.com/pingcap/tidb-operator/pkg/manager/meta"
	"github.com/pingcap/tidb-operator/pkg/monitor"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/util"
	utildiscovery "github.com/pingcap/tidb-operator/pkg/util/discovery"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/discovery"
	discoverycachedmemory "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/klog"
)

type MonitorManager struct {
	deps               *controller.Dependencies
	pvManager          monitor.MonitorManager
	discoveryInterface discovery.CachedDiscoveryInterface
}

const (
	FailedSync          = "FailedSync"
	SuccessSync         = "SuccessSync"
	prometheusComponent = "prometheus"
	grafanaComponent    = "grafana"
	componentPrefix     = "/topology"
)

func NewMonitorManager(deps *controller.Dependencies) *MonitorManager {
	return &MonitorManager{
		deps:               deps,
		pvManager:          meta.NewReclaimPolicyManager(deps),
		discoveryInterface: discoverycachedmemory.NewMemCacheClient(deps.KubeClientset.Discovery()),
	}
}

func (m *MonitorManager) SyncMonitor(monitor *v1alpha1.TidbMonitor) error {
	if monitor.DeletionTimestamp != nil {
		return nil
	}
	if monitor.Spec.Clusters == nil || len(monitor.Spec.Clusters) < 1 {
		klog.Errorf("tm[%s/%s] does not configure the target tidbcluster", monitor.Namespace, monitor.Name)
		return nil
	}

	defaultTidbMonitor(monitor)
	if !m.validate(monitor) {
		return nil // fatal error, no need to retry on invalid object
	}

	var firstTc *v1alpha1.TidbCluster
	assetStore := NewStore(m.deps.SecretLister)

	for _, tcRef := range monitor.Spec.Clusters {
		tc, err := m.deps.TiDBClusterLister.TidbClusters(tcRef.Namespace).Get(tcRef.Name)
		if err != nil {
			rerr := fmt.Errorf("get tm[%s/%s]'s target tc[%s/%s] failed, err: %v", monitor.Namespace, monitor.Name, tcRef.Namespace, tcRef.Name, err)
			return rerr
		}

		// If cluster enable tls
		if tc.IsTLSClusterEnabled() {
			tcTlsSecretName := util.ClusterClientTLSSecretName(tc.Name)
			err := assetStore.addTLSAssets(tc.Namespace, tcTlsSecretName)
			if err != nil {
				return err
			}
		}

		if firstTc == nil && !tc.HeterogeneousWithoutLocalPD() {
			firstTc = tc
		}
		err = m.syncDashboardMetricStorage(tc, monitor)
		if err != nil {
			klog.Errorf("Fail to sync TiDB Dashboard metrics config for TiDB cluster [%s/%s], error: %v", tc.Namespace, tc.Name, err)
			continue
		}
	}

	var firstDc *v1alpha1.DMCluster
	if monitor.Spec.DM != nil {
		for _, dcRef := range monitor.Spec.DM.Clusters {
			dc, err := m.deps.DMClusterLister.DMClusters(dcRef.Namespace).Get(dcRef.Name)
			if err != nil {
				rerr := fmt.Errorf("get tm[%s/%s]'s target dc[%s/%s] failed, err: %v", monitor.Namespace, monitor.Name, dcRef.Namespace, dcRef.Name, err)
				return rerr
			}
			if firstDc == nil {
				firstDc = dc
			}
			// If cluster enable tls
			if dc.IsTLSClusterEnabled() {
				dmTlsSecretName := util.DMClientTLSSecretName(dcRef.Name)
				err := assetStore.addTLSAssets(dcRef.Namespace, dmTlsSecretName)
				if err != nil {
					return err
				}
			}
		}
	}

	// create or update tls asset secret
	err := m.syncAssetSecret(monitor, assetStore)
	if err != nil {
		return err
	}

	// Sync Service
	if err := m.syncTidbMonitorService(monitor); err != nil {
		message := fmt.Sprintf("Sync TidbMonitor[%s/%s] Service failed, err: %v", monitor.Namespace, monitor.Name, err)
		m.deps.Recorder.Event(monitor, corev1.EventTypeWarning, FailedSync, message)
		return err
	}
	klog.V(4).Infof("tm[%s/%s]'s service synced", monitor.Namespace, monitor.Name)

	// Sync Statefulset
	if err := m.syncTidbMonitorStatefulset(firstTc, firstDc, monitor); err != nil {
		message := fmt.Sprintf("Sync TidbMonitor[%s/%s] Statefulset failed, err:%v", monitor.Namespace, monitor.Name, err)
		m.deps.Recorder.Event(monitor, corev1.EventTypeWarning, FailedSync, message)
		return err
	}

	// Sync PV
	if monitor.Spec.Persistent {
		// syncing all PVs managed by this tidbmonitor
		if err := m.pvManager.SyncMonitor(monitor); err != nil {
			return err
		}
		if err := m.syncTidbMonitorPV(monitor); err != nil {
			return err
		}
		klog.V(4).Infof("tm[%s/%s]'s pv synced", monitor.Namespace, monitor.Name)
	}

	klog.V(4).Infof("tm[%s/%s]'s StatefulSet synced", monitor.Namespace, monitor.Name)

	// Sync Ingress
	if err := m.syncIngress(monitor); err != nil {
		message := fmt.Sprintf("Sync TidbMonitor[%s/%s] Ingress failed,err:%v", monitor.Namespace, monitor.Name, err)
		m.deps.Recorder.Event(monitor, corev1.EventTypeWarning, FailedSync, message)
		return err
	}
	klog.V(4).Infof("tm[%s/%s]'s ingress synced", monitor.Namespace, monitor.Name)

	err = m.syncTidbMonitorStatus(monitor)
	if err != nil {
		klog.Errorf("Fail to sync tm[%s/%s]'s status, err: %v", monitor.Namespace, monitor.Name, err)
		return err
	}

	return nil
}

func (m *MonitorManager) syncTidbMonitorStatus(monitor *v1alpha1.TidbMonitor) error {
	sts, err := m.deps.StatefulSetLister.StatefulSets(monitor.Namespace).Get(GetMonitorObjectName(monitor))
	if err != nil {
		if errors.IsNotFound(err) {
			klog.V(4).Infof("tm[%s/%s]'s sts not found", monitor.Namespace, monitor.Name)
			return nil
		}
		return err
	}
	monitor.Status.StatefulSet = &sts.Status
	return nil
}

func (m *MonitorManager) syncTidbMonitorService(monitor *v1alpha1.TidbMonitor) error {
	services := getMonitorService(monitor)
	for _, newSvc := range services {
		if err := member.CreateOrUpdateService(m.deps.ServiceLister, m.deps.ServiceControl, newSvc, monitor); err != nil {
			return err
		}
	}
	return nil
}

func (m *MonitorManager) syncTidbMonitorStatefulset(tc *v1alpha1.TidbCluster, dc *v1alpha1.DMCluster, monitor *v1alpha1.TidbMonitor) error {
	ns := monitor.Namespace
	name := monitor.Name
	cm, err := m.syncTidbMonitorConfig(monitor)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s configmap failed to sync,err: %v", ns, name, err)
		return err
	}
	secret, err := m.syncTidbMonitorSecret(monitor)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s secret failed to sync,err: %v", ns, name, err)
		return err
	}

	sa, err := m.syncTidbMonitorRbac(monitor)
	if err != nil {
		klog.Errorf("tm[%s/%s]'s rbac failed to sync,err: %v", ns, name, err)
		return err
	}

	result, err := m.smoothMigrationToStatefulSet(monitor)
	if err != nil {
		klog.Errorf("Fail to migrate from deployment to statefulset for tm [%s/%s], err: %v", ns, name, err)
		return err
	}
	if !result {
		klog.Infof("Wait for the smooth migration to be done successfully for tm [%s/%s]", ns, name)
		return nil
	}

	newMonitorSts, err := getMonitorStatefulSet(sa, cm, secret, monitor, tc, dc)
	if err != nil {
		klog.Errorf("Fail to generate statefulset for tm [%s/%s], err: %v", ns, name, err)
		return err
	}

	oldMonitorSetTmp, err := m.deps.StatefulSetLister.StatefulSets(ns).Get(GetMonitorObjectName(monitor))
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncTidbMonitorStatefulset: fail to get sts %s for cluster %s/%s, error: %s", GetMonitorObjectName(monitor), ns, name, err)
	}
	setNotExist := errors.IsNotFound(err)
	if setNotExist {
		err = member.SetStatefulSetLastAppliedConfigAnnotation(newMonitorSts)
		if err != nil {
			return err
		}
		if err := m.deps.StatefulSetControl.CreateStatefulSet(monitor, newMonitorSts); err != nil {
			return err
		}
		return controller.RequeueErrorf("TidbMonitor: [%s/%s], waiting for tidbmonitor running", ns, name)
	}

	return member.UpdateStatefulSet(m.deps.StatefulSetControl, monitor, newMonitorSts, oldMonitorSetTmp)
}

func (m *MonitorManager) syncTidbMonitorSecret(monitor *v1alpha1.TidbMonitor) (*corev1.Secret, error) {
	if monitor.Spec.Grafana == nil {
		return nil, nil
	}
	newSt := getMonitorSecret(monitor)
	return m.deps.TypedControl.CreateOrUpdateSecret(monitor, newSt)
}

func (m *MonitorManager) syncTidbMonitorConfig(monitor *v1alpha1.TidbMonitor) (*corev1.ConfigMap, error) {
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
			tcList, err := m.deps.TiDBClusterLister.TidbClusters(tcRef.Namespace).List(selector)
			if err != nil {
				klog.Errorf("tm[%s/%s] gets tc[%s/%s]'s autoscaling clusters failed, err: %v", monitor.Namespace, monitor.Name, tcRef.Namespace, tcRef.Name, err)
				continue
			}
			for _, autoTc := range tcList {
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

	var monitorClusterInfos []ClusterRegexInfo
	for _, tcRef := range monitor.Spec.Clusters {
		tc, err := m.deps.TiDBClusterLister.TidbClusters(tcRef.Namespace).Get(tcRef.Name)
		if err != nil {
			rerr := fmt.Errorf("get tm[%s/%s]'s target tc[%s/%s] failed, err: %v", monitor.Namespace, monitor.Name, tcRef.Namespace, tcRef.Name, err)
			return nil, rerr
		}
		clusterRegex := ClusterRegexInfo{
			Name:      tcRef.Name,
			Namespace: tcRef.Namespace,
		}
		// If cluster enable tls
		if tc.IsTLSClusterEnabled() {
			clusterRegex.enableTLS = true
		}
		monitorClusterInfos = append(monitorClusterInfos, clusterRegex)
	}

	var dmClusterInfos []ClusterRegexInfo
	if monitor.Spec.DM != nil {
		for _, dmRef := range monitor.Spec.DM.Clusters {
			dm, err := m.deps.DMClusterLister.DMClusters(dmRef.Namespace).Get(dmRef.Name)
			if err != nil {
				rerr := fmt.Errorf("get tm[%s/%s]'s target dm[%s/%s] failed, err: %v", monitor.Namespace, monitor.Name, dmRef.Namespace, dmRef.Name, err)
				return nil, rerr
			}
			clusterRegex := ClusterRegexInfo{
				Name:      dmRef.Name,
				Namespace: dmRef.Namespace,
			}
			// If cluster enable tls
			if dm.IsTLSClusterEnabled() {
				clusterRegex.enableTLS = true
			}
			dmClusterInfos = append(dmClusterInfos, clusterRegex)
		}
	}

	newCM, err := getMonitorConfigMap(monitor, monitorClusterInfos, dmClusterInfos)
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

func (m *MonitorManager) smoothMigrationToStatefulSet(monitor *v1alpha1.TidbMonitor) (bool, error) {
	if m.deps.PVLister == nil {
		klog.V(4).Infof("Persistent volumes lister is unavailable, skip migrating to statefulset for tm[%s/%s]. This may be caused by no relevant permissions",
			monitor.Namespace, monitor.Name)
		return true, nil
	}

	// determine whether there is an old deployment
	oldDeploymentName := GetMonitorObjectName(monitor)
	oldDeployment, err := m.deps.DeploymentLister.Deployments(monitor.Namespace).Get(oldDeploymentName)
	if err == nil {
		klog.Infof("The old deployment exists, start smooth migration for tm [%s/%s]", monitor.Namespace, monitor.Name)
		// if deployment exist, delete it and wait next reconcile.
		err = m.deps.TypedControl.Delete(monitor, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      oldDeployment.Name,
				Namespace: oldDeployment.Namespace,
			},
		})
		if err != nil {
			klog.Errorf("Smooth migration for tm[%s/%s], fail to delete the old deployment, err: %v", monitor.Namespace, monitor.Name, err)
			return false, err
		}
		// If enable persistent,operator need to migrate pvc and pv binding relationship.
		return !monitor.Spec.Persistent, nil
	}

	if !errors.IsNotFound(err) {
		klog.Errorf("Fail to get deployment for tm [%s/%s], err: %v", monitor.Namespace, monitor.Name, err)
		return false, err
	}
	if !monitor.Spec.Persistent {
		return true, nil
	}
	firstStsPvcName := GetMonitorFirstPVCName(monitor.Name)
	deploymentPvcName := GetMonitorObjectName(monitor)
	deploymentPvc, err := m.deps.PVCLister.PersistentVolumeClaims(monitor.Namespace).Get(deploymentPvcName)
	if err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf("Smooth migration for tm[%s/%s], get the PVC of the deployment error: %v", monitor.Namespace, monitor.Name, err)
			return false, err
		}

		// If the PVC of the deployment does not exist and no old PV status, we don't need to migrate.
		if monitor.Status.DeploymentStorageStatus == nil || len(monitor.Status.DeploymentStorageStatus.PvName) <= 0 {
			return true, nil
		}

		deploymentPv, err := m.deps.PVLister.Get(monitor.Status.DeploymentStorageStatus.PvName)
		if err != nil {
			klog.Errorf("Smooth migration for tm[%s/%s], fail to patch PV %s, err: %v", monitor.Namespace, monitor.Name, monitor.Status.DeploymentStorageStatus.PvName, err)
			return false, err
		}

		if deploymentPv.Spec.ClaimRef != nil && deploymentPv.Spec.ClaimRef.Name == firstStsPvcName {
			// smooth migration successfully and clean status
			monitor.Status.DeploymentStorageStatus = nil
			return true, nil
		}

		err = m.patchPVClaimRef(deploymentPv, firstStsPvcName, monitor)
		if err != nil {
			klog.Errorf("Smooth migration for tm[%s/%s], fail to patch PV %s, err: %v", monitor.Namespace, monitor.Name, monitor.Status.DeploymentStorageStatus.PvName, err)
			return false, err
		}
		// smooth migration successfully and clean status
		monitor.Status.DeploymentStorageStatus = nil
		return true, nil
	}

	if len(deploymentPvc.Spec.VolumeName) <= 0 {
		klog.Infof("Smooth migration for tm[%s/%s], old pvc not bind pv and continue create statefulset", monitor.Namespace, monitor.Name)
		return true, nil
	}

	deploymentPv, err := m.deps.PVLister.Get(deploymentPvc.Spec.VolumeName)
	if err != nil {
		klog.Errorf("Smooth migration for tm[%s/%s], fail to get PV %s, err: %v", monitor.Namespace, monitor.Name, deploymentPvc.Spec.VolumeName, err)
		return false, err
	}
	if deploymentPv.Spec.PersistentVolumeReclaimPolicy == corev1.PersistentVolumeReclaimDelete {
		klog.Errorf("Smooth migration for tm[%s/%s], pv[%s] policy is delete, it must be retain", monitor.Namespace, monitor.Name, deploymentPvc.Spec.VolumeName)
		return false, err
	}

	if deploymentPv.Spec.ClaimRef != nil && deploymentPv.Spec.ClaimRef.Name == firstStsPvcName {
		// smooth migration successfully and clean status
		monitor.Status.DeploymentStorageStatus = nil
		return true, nil
	}

	// firstly patch status
	if monitor.Status.DeploymentStorageStatus == nil {
		monitor.Status.DeploymentStorageStatus = &v1alpha1.DeploymentStorageStatus{
			PvName: deploymentPvc.Spec.VolumeName,
		}
		return false, controller.RequeueErrorf("TidbMonitor: [%s/%s] update deploymentStorageStatus requeue", monitor.Namespace, monitor.Name)
		//monitor patch status successfully
	}

	err = m.deps.PVCControl.DeletePVC(monitor, &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentPvcName,
			Namespace: monitor.Namespace,
		},
	})
	if err != nil {
		klog.Errorf("Fail to delete the PVC %s for tm [%s/%s], err: %v", deploymentPvcName, monitor.Namespace, monitor.Name, err)
		return false, err
	}
	err = m.patchPVClaimRef(deploymentPv, firstStsPvcName, monitor)
	if err != nil {
		klog.Errorf("Smooth migration for tm[%s/%s], fail to patch PV %s, err: %v", monitor.Namespace, monitor.Name, deploymentPvc.Spec.VolumeName, err)
		return false, err
	}
	// smooth migration successfully and clean status
	monitor.Status.DeploymentStorageStatus = nil
	return true, nil
}

func (c *MonitorManager) validate(tidbmonitor *v1alpha1.TidbMonitor) bool {
	errs := v1alpha1validation.ValidateTidbMonitor(tidbmonitor)
	if len(errs) > 0 {
		aggregatedErr := errs.ToAggregate()
		klog.Errorf("tidbmonitor %s/%s is not valid and must be fixed first, aggregated error: %v", tidbmonitor.GetNamespace(), tidbmonitor.GetName(), aggregatedErr)
		c.deps.Recorder.Event(tidbmonitor, corev1.EventTypeWarning, "FailedValidation", aggregatedErr.Error())
		return false
	}
	return true
}

func (m *MonitorManager) syncTidbMonitorPV(tm *v1alpha1.TidbMonitor) error {
	ns := tm.GetNamespace()
	instanceName := tm.Name

	if m.deps.PVLister == nil {
		klog.V(4).Infof("Persistent volumes lister is unavailable, skip syncing TidbMonitor %s/%s PVs. This may be caused by no relevant permissions", ns, instanceName)
		return nil
	}

	l, err := label.NewMonitor().Instance(instanceName).Monitor().Selector()
	if err != nil {
		return err
	}
	pods, err := m.deps.PodLister.Pods(ns).List(l)
	if err != nil {
		return fmt.Errorf("fail to list pods for tidbmonitor %s/%s, selector: %s, error: %v", ns, instanceName, l, err)
	}

	for _, pod := range pods {
		// update meta info for pvc
		pvcs, err := util.ResolvePVCFromPod(pod, m.deps.PVCLister)
		if err != nil {
			return err
		}
		for _, pvc := range pvcs {
			if pvc.Spec.VolumeName == "" {
				continue
			}
			// update meta info for pv
			pv, err := m.deps.PVLister.Get(pvc.Spec.VolumeName)
			if err != nil {
				klog.Errorf("Get PV %s error: %v", pvc.Spec.VolumeName, err)
				return err
			}
			_, err = m.deps.PVControl.UpdateMetaInfo(tm, pv)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *MonitorManager) patchPVClaimRef(pv *corev1.PersistentVolume, patchPvcName string, monitor *v1alpha1.TidbMonitor) error {
	if pv.Spec.ClaimRef == nil {
		pv.Spec.ClaimRef = &corev1.ObjectReference{}
	}

	pv.Spec.ClaimRef.Name = patchPvcName
	err := m.deps.PVControl.PatchPVClaimRef(monitor, pv, patchPvcName)
	if err != nil {
		return err
	}
	return nil
}

func (m *MonitorManager) syncAssetSecret(monitor *v1alpha1.TidbMonitor, store *Store) error {
	ns := monitor.Namespace
	name := monitor.Name
	tlsAssetsSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            GetTLSAssetsSecretName(monitor.Name),
			Namespace:       monitor.Namespace,
			Labels:          buildTidbMonitorLabel(monitor.Name),
			OwnerReferences: []metav1.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
		},
		Data: make(map[string][]byte, len(store.TLSAssets)),
	}
	for key, asset := range store.TLSAssets {
		tlsAssetsSecret.Data[key.String()] = []byte(asset)
	}

	_, err := m.deps.TypedControl.CreateOrUpdateSecret(monitor, tlsAssetsSecret)
	if err != nil {
		klog.Errorf("Fail to sync tm[%s/%s]'s secret assets, err: %v", ns, name, err)
		return err
	}
	return nil
}

func (m *MonitorManager) syncDashboardMetricStorage(tc *v1alpha1.TidbCluster, tm *v1alpha1.TidbMonitor) error {
	if tc.Spec.PD == nil {
		return nil
	}
	pdEtcdClient, err := m.deps.PDControl.GetPDEtcdClient(pdapi.Namespace(tc.Namespace), tc.Name, tc.IsTLSClusterEnabled())

	if err != nil {
		return err
	}
	defer pdEtcdClient.Close()
	// sync prometheus key
	err = syncComponent(tm, prometheusComponent, 9090, pdEtcdClient)
	if err != nil {
		return err
	}

	// sync grafana key
	if tm.Spec.Grafana != nil {
		err = syncComponent(tm, grafanaComponent, 3000, pdEtcdClient)
		if err != nil {
			return err
		}
	}

	return nil
}

func syncComponent(tm *v1alpha1.TidbMonitor, componentName string, port int, etcdClient pdapi.PDEtcdClient) error {
	key := buildComponentKey(componentName)
	v, err := buildComponentValue(tm, componentName, port)
	if err != nil {
		klog.Error(err.Error())
		return err
	}
	//10 min
	err = etcdClient.PutTTLKey(key, v, 600)
	if err != nil {
		klog.Error(err.Error())
		return err
	}
	return nil
}

func buildComponentKey(component string) string {
	return fmt.Sprintf("%s/%s", componentPrefix, component)
}

func buildComponentValue(tm *v1alpha1.TidbMonitor, componentName string, port int) (string, error) {
	return buildEtcdValue(fmt.Sprintf("%s-%s.%s.svc", tm.Name, componentName, tm.Namespace), port)
}

type componentTopology struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

func buildEtcdValue(host string, port int) (string, error) {
	topology := componentTopology{
		IP:   host,
		Port: port,
	}
	data, err := json.Marshal(topology)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
