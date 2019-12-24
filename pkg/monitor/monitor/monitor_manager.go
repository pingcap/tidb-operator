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
	"github.com/pingcap/tidb-operator/pkg/label"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

type MonitorManager struct {
	typedControl     controller.TypedControlInterface
	ConfigMapListers corelisters.ConfigMapLister
	SecretLister     corelisters.SecretLister
	DeploymentLister appslisters.DeploymentLister
	StsControl       controller.StatefulSetControlInterface
	PvcControl       controller.GeneralPVCControlInterface
}

func (mm *MonitorManager) Sync(monitor *v1alpha1.TidbMonitor) error {

	if monitor.DeletionTimestamp != nil {
		return nil
	}
	return nil
}

func (mm *MonitorManager) syncTidbMonitor(monitor *v1alpha1.TidbMonitor) error {
	//name := monitor.Name
	namespace := monitor.Namespace

	oldMonitorDeployTmp, err := mm.DeploymentLister.Deployments(namespace).Get(getMonitorObjectName(monitor))
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	deployNotExist := errors.IsNotFound(err)

	oldMonitorDeploy := oldMonitorDeployTmp.DeepCopy()

	cm, err := mm.syncTidbMonitorConfig(monitor)
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
	deployment := getMonitorDeployment(sa, cm, secret, monitor)
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

func (mm *MonitorManager) syncTidbMonitorConfig(monitor *v1alpha1.TidbMonitor) (*core.ConfigMap, error) {

	newCM, err := getMonitorConfigMap(monitor)
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

func getMonitorConfigMap(monitor *v1alpha1.TidbMonitor) (*core.ConfigMap, error) {

	var releaseNamespaces []string
	for _, cluster := range monitor.Spec.Clusters {
		releaseNamespaces = append(releaseNamespaces, cluster.Namespace)
	}

	model := &MonitorConfigModel{
		AlertmanagerURL:    *monitor.Spec.AlertmanagerURL,
		ReleaseNamespaces:  releaseNamespaces,
		ReleaseTargetRegex: getMonitorTargetRegex(monitor),
		EnableTLSCluster:   false,
	}

	content, err := RenderPrometheusConfig(model)
	if err != nil {
		return nil, err
	}

	monitorLabel := label.New().Instance(monitor.Name).Monitor().Labels()
	cm := &core.ConfigMap{
		ObjectMeta: meta.ObjectMeta{
			Name:            getMonitorObjectName(monitor),
			Namespace:       monitor.Namespace,
			Labels:          monitorLabel,
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
		},
		Data: map[string]string{
			"prometheus-config": content,
		},
	}
	if monitor.Spec.Grafana != nil {
		cm.Data["dashboard-config"] = dashBoardConfig
	}
	return cm, nil
}

func getMonitorSecret(monitor *v1alpha1.TidbMonitor) *core.Secret {
	monitorLabel := label.New().Instance(monitor.Name).Monitor().Labels()
	return &core.Secret{
		ObjectMeta: meta.ObjectMeta{
			Name:            getMonitorObjectName(monitor),
			Namespace:       monitor.Namespace,
			Labels:          monitorLabel,
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
		},
		Data: map[string][]byte{
			"username": []byte(monitor.Spec.Grafana.Username),
			"password": []byte(monitor.Spec.Grafana.Password),
		},
	}
}

func getMonitorServiceAccount(monitor *v1alpha1.TidbMonitor) *core.ServiceAccount {
	monitorLabel := label.New().Instance(monitor.Name).Monitor().Labels()
	sa := &core.ServiceAccount{
		ObjectMeta: meta.ObjectMeta{
			Name:            getMonitorObjectName(monitor),
			Namespace:       monitor.Namespace,
			Labels:          monitorLabel,
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
		},
	}
	return sa
}

func getMonitorClusterRole(monitor *v1alpha1.TidbMonitor) *rbac.ClusterRole {
	monitorLabel := label.New().Instance(monitor.Name).Monitor().Labels()
	return &rbac.ClusterRole{
		ObjectMeta: meta.ObjectMeta{
			Name:            getMonitorObjectName(monitor),
			Namespace:       monitor.Namespace,
			Labels:          monitorLabel,
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
		},
		Rules: []rbac.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				NonResourceURLs: []string{"/metrics"},
				Verbs:           []string{"get"},
			},
		},
	}
}

func getMonitorClusterRoleBinding(sa *core.ServiceAccount, cr *rbac.ClusterRole, monitor *v1alpha1.TidbMonitor) *rbac.ClusterRoleBinding {
	monitorLabel := label.New().Instance(monitor.Name).Monitor().Labels()
	return &rbac.ClusterRoleBinding{
		ObjectMeta: meta.ObjectMeta{
			Name:            getMonitorObjectName(monitor),
			Namespace:       monitor.Namespace,
			Labels:          monitorLabel,
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
		},
		Subjects: []rbac.Subject{
			{
				Kind:      sa.Kind,
				Name:      sa.Name,
				Namespace: sa.Namespace,
			},
		},
		RoleRef: rbac.RoleRef{
			Kind:     cr.Kind,
			Name:     cr.Name,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

func getMonitorDeployment(sa *core.ServiceAccount, config *core.ConfigMap, secret *core.Secret, monitor *v1alpha1.TidbMonitor) *apps.Deployment {
	monitorLabel := label.New().Instance(monitor.Name).Monitor().Labels()
	replicas := int32(1)

	c := "mkdir -p /data/prometheus\nchmod 777 /data/prometheus\n/usr/bin/init.sh"
	if monitor.Spec.Grafana != nil {
		c = "mkdir -p /data/prometheus /data/grafana\nchmod 777 /data/prometheus /data/grafana\n/usr/bin/init.sh"
	}
	commands := []string{
		"/bin/sh",
		"-c",
		c,
	}
	secureContext := int64(0)
	deployment := &apps.Deployment{
		ObjectMeta: meta.ObjectMeta{
			Name:            getMonitorObjectName(monitor),
			Namespace:       monitor.Namespace,
			Labels:          monitorLabel,
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
		},
		Spec: apps.DeploymentSpec{
			Replicas: &replicas,
			Strategy: apps.DeploymentStrategy{
				Type:          apps.RecreateDeploymentStrategyType,
				RollingUpdate: nil,
			},
			Selector: &meta.LabelSelector{
				MatchLabels: map[string]string{
					label.InstanceLabelKey:  monitor.Name,
					label.ComponentLabelKey: label.TiDBMonitorVal,
				},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Labels: map[string]string{
						label.InstanceLabelKey:  monitor.Name,
						label.ComponentLabelKey: label.TiDBMonitorVal,
					},
				},
				Spec: core.PodSpec{
					ServiceAccountName: sa.Name,
					InitContainers: []core.Container{
						{
							Name:            "monitor-initializer",
							Image:           monitor.Spec.Initializer.BaseImage,
							ImagePullPolicy: *monitor.Spec.Initializer.ImagePullPolicy,
							Env: []core.EnvVar{
								{
									Name:  "GF_PROVISIONING_PATH",
									Value: "/grafana-dashboard-definitions/tidb",
								},
								{
									Name:  "GF_DATASOURCE_PATH",
									Value: "/etc/grafana/provisioning/datasources",
								},
								//{
								//	Name: "TIDB_CLUSTER_NAME",
								//},
								//{
								//	Name:"TIDB_ENABLE_BINLOG",
								//},
								{
									Name:  "PROM_CONFIG_PATH",
									Value: "/prometheus-rules",
								},
								{
									Name:  "PROM_PERSISTENT_DIR",
									Value: "/data",
								},
								//{
								//	Name: "TIDB_VERSION",
								//},
								{
									Name:  "GF_K8S_PROMETHEUS_URL",
									Value: *monitor.Spec.KubePrometheusURL,
								},
								{
									Name:  "GF_TIDB_PROMETHEUS_URL",
									Value: "http://127.0.0.1:9090",
								},
								//{
								//	Name: "TIDB_CLUSTER_NAMESPACE",
								//},
								//{
								//	Name:	"TZ",
								//},
							},
							Command: commands,
							SecurityContext: &core.SecurityContext{
								RunAsUser: &secureContext,
							},
							VolumeMounts: []core.VolumeMount{
								{
									MountPath: "/grafana-dashboard-definitions/tidb",
									Name:      "grafana-dashboard",
									ReadOnly:  false,
								},
								{
									MountPath: "/prometheus-rules",
									Name:      "prometheus-rules",
									ReadOnly:  false,
								},
								{
									MountPath: "/data",
									Name:      "monitor-data",
								},
								{
									MountPath: "/etc/grafana/provisioning/datasources",
									Name:      "datasource",
									ReadOnly:  false,
								},
							},

							Resources: core.ResourceRequirements{},
						},
					},
				},
			},
		},
	}
	return deployment
}
