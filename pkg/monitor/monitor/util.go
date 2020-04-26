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
	"strconv"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/prometheus/prometheus/config"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

func GetMonitorObjectName(monitor *v1alpha1.TidbMonitor) string {
	return fmt.Sprintf("%s-monitor", monitor.Name)
}

func buildTidbMonitorLabel(name string) map[string]string {
	return label.NewMonitor().Instance(name).Monitor().Labels()
}

// getMonitorConfigMap generate the Prometheus config and Grafana config for TidbMonitor,
// If the namespace in ClusterRef is empty, we would set the TidbMonitor's namespace in the default
func getMonitorConfigMap(tc *v1alpha1.TidbCluster, monitor *v1alpha1.TidbMonitor) (*core.ConfigMap, error) {

	var releaseNamespaces []string
	for _, cluster := range monitor.Spec.Clusters {
		releaseNamespaces = append(releaseNamespaces, cluster.Namespace)
	}

	targetPattern, err := config.NewRegexp(tc.Name)
	if err != nil {
		return nil, err
	}
	model := &MonitorConfigModel{
		AlertmanagerURL:    "",
		ReleaseNamespaces:  releaseNamespaces,
		ReleaseTargetRegex: &targetPattern,
		EnableTLSCluster:   tc.IsTLSClusterEnabled(),
	}

	if monitor.Spec.AlertmanagerURL != nil {
		model.AlertmanagerURL = *monitor.Spec.AlertmanagerURL
	}

	if len(model.ReleaseNamespaces) < 1 {
		model.ReleaseNamespaces = append(model.ReleaseNamespaces, monitor.Namespace)
	}

	content, err := RenderPrometheusConfig(model)
	if err != nil {
		return nil, err
	}

	cm := &core.ConfigMap{
		ObjectMeta: meta.ObjectMeta{
			Name:            GetMonitorObjectName(monitor),
			Namespace:       monitor.Namespace,
			Labels:          buildTidbMonitorLabel(monitor.Name),
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
	return &core.Secret{
		ObjectMeta: meta.ObjectMeta{
			Name:            GetMonitorObjectName(monitor),
			Namespace:       monitor.Namespace,
			Labels:          buildTidbMonitorLabel(monitor.Name),
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
		},
		Data: map[string][]byte{
			"username": []byte(monitor.Spec.Grafana.Username),
			"password": []byte(monitor.Spec.Grafana.Password),
		},
	}
}

func getMonitorServiceAccount(monitor *v1alpha1.TidbMonitor) *core.ServiceAccount {
	sa := &core.ServiceAccount{
		ObjectMeta: meta.ObjectMeta{
			Name:            GetMonitorObjectName(monitor),
			Namespace:       monitor.Namespace,
			Labels:          buildTidbMonitorLabel(monitor.Name),
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
		},
	}
	return sa
}

func getMonitorClusterRole(monitor *v1alpha1.TidbMonitor, policyRules []rbac.PolicyRule) *rbac.ClusterRole {
	return &rbac.ClusterRole{
		ObjectMeta: meta.ObjectMeta{
			Name:            GetMonitorObjectName(monitor),
			Namespace:       monitor.Namespace,
			Labels:          buildTidbMonitorLabel(monitor.Name),
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
		},
		Rules: policyRules,
	}
}

func getMonitorRole(monitor *v1alpha1.TidbMonitor, policyRules []rbac.PolicyRule) *rbac.Role {
	return &rbac.Role{
		ObjectMeta: meta.ObjectMeta{
			Name:            GetMonitorObjectName(monitor),
			Namespace:       monitor.Namespace,
			Labels:          buildTidbMonitorLabel(monitor.Name),
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
		},
		Rules: policyRules,
	}
}

func getMonitorClusterRoleBinding(sa *core.ServiceAccount, cr *rbac.ClusterRole, monitor *v1alpha1.TidbMonitor) *rbac.ClusterRoleBinding {
	return &rbac.ClusterRoleBinding{
		ObjectMeta: meta.ObjectMeta{
			Name:            GetMonitorObjectName(monitor),
			Namespace:       monitor.Namespace,
			Labels:          buildTidbMonitorLabel(monitor.Name),
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
		},
		Subjects: []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa.Name,
				Namespace: sa.Namespace,
				APIGroup:  "",
			},
		},
		RoleRef: rbac.RoleRef{
			Kind:     "ClusterRole",
			Name:     cr.Name,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

func getMonitorRoleBinding(sa *core.ServiceAccount, role *rbac.Role, monitor *v1alpha1.TidbMonitor) *rbac.RoleBinding {
	return &rbac.RoleBinding{
		ObjectMeta: meta.ObjectMeta{
			Name:            GetMonitorObjectName(monitor),
			Namespace:       monitor.Namespace,
			Labels:          buildTidbMonitorLabel(monitor.Name),
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
		},
		Subjects: []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa.Name,
				Namespace: sa.Namespace,
				APIGroup:  "",
			},
		},
		RoleRef: rbac.RoleRef{
			Kind:     "Role",
			Name:     role.Name,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

func getMonitorDeployment(sa *core.ServiceAccount, config *core.ConfigMap, secret *core.Secret, monitor *v1alpha1.TidbMonitor, tc *v1alpha1.TidbCluster) (*apps.Deployment, error) {
	deployment := getMonitorDeploymentSkeleton(sa, monitor)
	initContainer := getMonitorInitContainer(monitor, tc)
	deployment.Spec.Template.Spec.InitContainers = append(deployment.Spec.Template.Spec.InitContainers, initContainer)
	prometheusContainer := getMonitorPrometheusContainer(monitor, tc, config)
	reloaderContainer := getMonitorReloaderContainer(monitor, tc)
	deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, prometheusContainer, reloaderContainer)
	if monitor.Spec.Grafana != nil {
		grafanaContainer := getMonitorGrafanaContainer(secret, monitor, tc)
		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, grafanaContainer)
	}
	volumes := getMonitorVolumes(config, monitor, tc)
	deployment.Spec.Template.Spec.Volumes = volumes
	b, err := json.Marshal(deployment.Spec.Template.Spec)
	if err != nil {
		return nil, err
	}
	if deployment.Annotations == nil {
		deployment.Annotations = map[string]string{}
	}
	deployment.Annotations[controller.LastAppliedPodTemplate] = string(b)
	return deployment, nil
}

func getMonitorDeploymentSkeleton(sa *core.ServiceAccount, monitor *v1alpha1.TidbMonitor) *apps.Deployment {
	replicas := int32(1)

	deployment := &apps.Deployment{
		ObjectMeta: meta.ObjectMeta{
			Name:            GetMonitorObjectName(monitor),
			Namespace:       monitor.Namespace,
			Labels:          buildTidbMonitorLabel(monitor.Name),
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
			Annotations:     monitor.Spec.Annotations,
		},
		Spec: apps.DeploymentSpec{
			Replicas: &replicas,
			Strategy: apps.DeploymentStrategy{
				Type: apps.RecreateDeploymentStrategyType,
			},
			Selector: &meta.LabelSelector{
				MatchLabels: buildTidbMonitorLabel(monitor.Name),
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Labels: buildTidbMonitorLabel(monitor.Name),
				},

				Spec: core.PodSpec{
					ServiceAccountName: sa.Name,
					InitContainers:     []core.Container{},
					Containers:         []core.Container{},
					Volumes:            []core.Volume{},
					Tolerations:        monitor.Spec.Tolerations,
					NodeSelector:       monitor.Spec.NodeSelector,
				},
			},
		},
	}
	return deployment
}

func getMonitorInitContainer(monitor *v1alpha1.TidbMonitor, tc *v1alpha1.TidbCluster) core.Container {
	c := `mkdir -p /data/prometheus
chmod 777 /data/prometheus
/usr/bin/init.sh`
	if monitor.Spec.Grafana != nil {
		c = `mkdir -p /data/prometheus /data/grafana
chmod 777 /data/prometheus /data/grafana
/usr/bin/init.sh`
	}
	command := []string{
		"/bin/sh",
		"-c",
		c,
	}
	container := core.Container{
		Name:  "monitor-initializer",
		Image: fmt.Sprintf("%s:%s", monitor.Spec.Initializer.BaseImage, monitor.Spec.Initializer.Version),
		Env: []core.EnvVar{
			{
				Name:  "TIDB_CLUSTER_NAME",
				Value: tc.Name,
			},
			{
				Name:  "TIDB_ENABLE_BINLOG",
				Value: strconv.FormatBool(tc.IsTiDBBinlogEnabled()),
			},
			{
				Name:  "PROM_CONFIG_PATH",
				Value: "/prometheus-rules",
			},
			{
				Name:  "PROM_PERSISTENT_DIR",
				Value: "/data",
			},
			{
				Name:  "TIDB_VERSION",
				Value: tc.TiDBImage(),
			},
			{
				Name:  "GF_TIDB_PROMETHEUS_URL",
				Value: "http://127.0.0.1:9090",
			},
			{
				Name:  "TIDB_CLUSTER_NAMESPACE",
				Value: tc.Namespace,
			},
			{
				Name:  "TZ",
				Value: tc.Spec.Timezone,
			},
		},
		Command: command,
		SecurityContext: &core.SecurityContext{
			RunAsUser: pointer.Int64Ptr(0),
		},
		VolumeMounts: []core.VolumeMount{
			{
				MountPath: "/prometheus-rules",
				Name:      "prometheus-rules",
				ReadOnly:  false,
			},
			{
				MountPath: "/data",
				Name:      "monitor-data",
			},
		},
		Resources: controller.ContainerResource(monitor.Spec.Initializer.Resources),
	}

	if monitor.Spec.Initializer.ImagePullPolicy != nil {
		container.ImagePullPolicy = *monitor.Spec.Initializer.ImagePullPolicy
	}

	if monitor.Spec.KubePrometheusURL != nil {
		container.Env = append(container.Env, core.EnvVar{
			Name:  "GF_K8S_PROMETHEUS_URL",
			Value: *monitor.Spec.KubePrometheusURL,
		})
	}

	if monitor.Spec.Grafana != nil {
		container.VolumeMounts = append(container.VolumeMounts, core.VolumeMount{

			MountPath: "/etc/grafana/provisioning/datasources",
			Name:      "datasource",
			ReadOnly:  false,
		}, core.VolumeMount{

			MountPath: "/grafana-dashboard-definitions/tidb",
			Name:      "grafana-dashboard",
			ReadOnly:  false,
		})
		container.Env = append(container.Env,
			core.EnvVar{
				Name:  "GF_PROVISIONING_PATH",
				Value: "/grafana-dashboard-definitions/tidb",
			},
			core.EnvVar{
				Name:  "GF_DATASOURCE_PATH",
				Value: "/etc/grafana/provisioning/datasources",
			})

	}
	return container
}

func getMonitorPrometheusContainer(monitor *v1alpha1.TidbMonitor, tc *v1alpha1.TidbCluster, config *core.ConfigMap) core.Container {
	command := []string{
		"/bin/prometheus",
		"--web.enable-admin-api",
		"--web.enable-lifecycle",
		"--config.file=/etc/prometheus/prometheus.yml",
		"--storage.tsdb.path=/data/prometheus",
		fmt.Sprintf("--storage.tsdb.retention=%dd", monitor.Spec.Prometheus.ReserveDays),
	}

	if startCommand, ok := config.Data["start-command"]; ok {
		command = strings.Split(startCommand, "\n")
	}
	c := core.Container{
		Name:      "prometheus",
		Image:     fmt.Sprintf("%s:%s", monitor.Spec.Prometheus.BaseImage, monitor.Spec.Prometheus.Version),
		Resources: controller.ContainerResource(monitor.Spec.Prometheus.Resources),
		Command:   command,
		Ports: []core.ContainerPort{
			{
				Name:          "prometheus",
				ContainerPort: 9090,
				Protocol:      core.ProtocolTCP,
			},
		},
		Env: []core.EnvVar{
			{
				Name:  "TZ",
				Value: tc.Spec.Timezone,
			},
		},
		VolumeMounts: []core.VolumeMount{
			{
				Name:      "prometheus-config",
				MountPath: "/etc/prometheus",
				ReadOnly:  true,
			},
			{
				Name:      "monitor-data",
				MountPath: "/data",
			},
			{
				Name:      "prometheus-rules",
				MountPath: "/prometheus-rules",
				ReadOnly:  false,
			},
		},
	}
	if len(monitor.Spec.Prometheus.LogLevel) > 0 {
		c.Command = append(c.Command, fmt.Sprintf("--log.level=%s", monitor.Spec.Prometheus.LogLevel))
	}

	if tc.IsTLSClusterEnabled() {
		c.VolumeMounts = append(c.VolumeMounts, core.VolumeMount{
			Name:      "cluster-client-tls",
			MountPath: util.ClusterClientTLSPath,
			ReadOnly:  true,
		})
	}
	if monitor.Spec.Prometheus.ImagePullPolicy != nil {
		c.ImagePullPolicy = *monitor.Spec.Prometheus.ImagePullPolicy
	}
	return c
}

func getMonitorGrafanaContainer(secret *core.Secret, monitor *v1alpha1.TidbMonitor, tc *v1alpha1.TidbCluster) core.Container {
	c := core.Container{
		Name:      "grafana",
		Image:     fmt.Sprintf("%s:%s", monitor.Spec.Grafana.BaseImage, monitor.Spec.Grafana.Version),
		Resources: controller.ContainerResource(monitor.Spec.Grafana.Resources),
		Ports: []core.ContainerPort{
			{
				Name:          "grafana",
				ContainerPort: 3000,
				Protocol:      core.ProtocolTCP,
			},
		},
		Env: []core.EnvVar{
			{
				Name:  "GF_PATHS_DATA",
				Value: "/data/grafana",
			},
			{
				Name: "GF_SECURITY_ADMIN_USER",
				ValueFrom: &core.EnvVarSource{
					SecretKeyRef: &core.SecretKeySelector{
						LocalObjectReference: core.LocalObjectReference{
							Name: secret.Name,
						},
						Key: "username",
					},
				},
			},
			{
				Name: "GF_SECURITY_ADMIN_PASSWORD",
				ValueFrom: &core.EnvVarSource{
					SecretKeyRef: &core.SecretKeySelector{
						LocalObjectReference: core.LocalObjectReference{
							Name: secret.Name,
						},
						Key: "password",
					},
				},
			},
			{
				Name:  "TZ",
				Value: tc.Timezone(),
			},
		},
		VolumeMounts: []core.VolumeMount{
			{
				Name:      "monitor-data",
				MountPath: "/data",
			},
			{
				Name:      "datasource",
				MountPath: "/etc/grafana/provisioning/datasources",
				ReadOnly:  false,
			},
			{
				Name:      "dashboards-provisioning",
				MountPath: "/etc/grafana/provisioning/dashboards",
				ReadOnly:  false,
			},
			{
				Name:      "grafana-dashboard",
				MountPath: "/grafana-dashboard-definitions/tidb",
				ReadOnly:  false,
			},
		},
	}
	for k, v := range monitor.Spec.Grafana.Envs {
		c.Env = append(c.Env, core.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	if monitor.Spec.Grafana.ImagePullPolicy != nil {
		c.ImagePullPolicy = *monitor.Spec.Grafana.ImagePullPolicy
	}
	sort.Sort(util.SortEnvByName(c.Env))
	return c
}

func getMonitorReloaderContainer(monitor *v1alpha1.TidbMonitor, tc *v1alpha1.TidbCluster) core.Container {
	c := core.Container{
		Name:  "reloader",
		Image: fmt.Sprintf("%s:%s", monitor.Spec.Reloader.BaseImage, monitor.Spec.Reloader.Version),
		Command: []string{
			"/bin/reload",
			"--root-store-path=/data",
			fmt.Sprintf("--sub-store-path=%s", tc.TiDBImage()),
			"--watch-path=/prometheus-rules/rules",
			"--prometheus-url=http://127.0.0.1:9090",
		},
		Ports: []core.ContainerPort{
			{
				Name:          "reloader",
				ContainerPort: 9089,
				Protocol:      core.ProtocolTCP,
			},
		},
		VolumeMounts: []core.VolumeMount{
			{
				Name:      "prometheus-rules",
				MountPath: "/prometheus-rules",
				ReadOnly:  false,
			},
			{
				Name:      "monitor-data",
				MountPath: "/data",
			},
		},
		Resources: controller.ContainerResource(monitor.Spec.Reloader.Resources),
		Env: []core.EnvVar{
			{
				Name:  "TZ",
				Value: tc.Spec.Timezone,
			},
		},
	}
	if monitor.Spec.Reloader.ImagePullPolicy != nil {
		c.ImagePullPolicy = *monitor.Spec.Reloader.ImagePullPolicy
	}
	return c
}

func getMonitorVolumes(config *core.ConfigMap, monitor *v1alpha1.TidbMonitor, tc *v1alpha1.TidbCluster) []core.Volume {
	volumes := []core.Volume{}
	monitorData := core.Volume{
		Name: "monitor-data",
		VolumeSource: core.VolumeSource{
			EmptyDir: &core.EmptyDirVolumeSource{},
		},
	}
	if monitor.Spec.Persistent {
		monitorData = core.Volume{
			Name: "monitor-data",
			VolumeSource: core.VolumeSource{
				PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
					ClaimName: GetMonitorObjectName(monitor),
				},
			},
		}
	}
	volumes = append(volumes, monitorData)
	prometheusConfig := core.Volume{
		Name: "prometheus-config",
		VolumeSource: core.VolumeSource{
			ConfigMap: &core.ConfigMapVolumeSource{
				LocalObjectReference: core.LocalObjectReference{
					Name: config.Name,
				},
				Items: []core.KeyToPath{
					{
						Key:  "prometheus-config",
						Path: "prometheus.yml",
					},
				},
			},
		},
	}
	volumes = append(volumes, prometheusConfig)
	if monitor.Spec.Grafana != nil {
		dataSource := core.Volume{
			Name: "datasource",
			VolumeSource: core.VolumeSource{
				EmptyDir: &core.EmptyDirVolumeSource{},
			},
		}
		dashboardsProvisioning := core.Volume{
			Name: "dashboards-provisioning",
			VolumeSource: core.VolumeSource{
				ConfigMap: &core.ConfigMapVolumeSource{
					LocalObjectReference: core.LocalObjectReference{
						Name: GetMonitorObjectName(monitor),
					},
					Items: []core.KeyToPath{
						{
							Key:  "dashboard-config",
							Path: "dashboards.yaml",
						},
					},
				},
			},
		}
		grafanaDashboard := core.Volume{
			Name: "grafana-dashboard",
			VolumeSource: core.VolumeSource{
				EmptyDir: &core.EmptyDirVolumeSource{},
			},
		}
		volumes = append(volumes, dataSource, dashboardsProvisioning, grafanaDashboard)
	}
	prometheusRules := core.Volume{
		Name: "prometheus-rules",
		VolumeSource: core.VolumeSource{
			EmptyDir: &core.EmptyDirVolumeSource{},
		},
	}
	volumes = append(volumes, prometheusRules)
	if tc.IsTLSClusterEnabled() {
		defaultMode := int32(420)
		tlsPDClient := core.Volume{
			Name: "cluster-client-tls",
			VolumeSource: core.VolumeSource{
				Secret: &core.SecretVolumeSource{
					SecretName:  util.ClusterClientTLSSecretName(tc.Name),
					DefaultMode: &defaultMode,
				},
			},
		}
		volumes = append(volumes, tlsPDClient)
	}
	return volumes
}

func getMonitorService(monitor *v1alpha1.TidbMonitor) []*core.Service {
	var services []*core.Service

	reloaderPortName := "tcp-reloader"
	prometheusPortName := "http-prometheus"
	grafanaPortName := "http-grafana"

	// currently monitor label haven't managedBy label due to 1.0 historical problem.
	// In order to be compatible with 1.0 release monitor, we have removed managedBy label for now.
	// We would add managedBy label key during released 1.2 version
	selector := map[string]string{
		label.InstanceLabelKey:  monitor.Name,
		label.NameLabelKey:      "tidb-cluster",
		label.ComponentLabelKey: label.TiDBMonitorVal,
	}

	if monitor.BaseReloaderSpec().PortName() != nil {
		reloaderPortName = *monitor.BaseReloaderSpec().PortName()
	}
	if monitor.BasePrometheusSpec().PortName() != nil {
		prometheusPortName = *monitor.BasePrometheusSpec().PortName()
	}
	if monitor.BaseGrafanaSpec() != nil && monitor.BaseGrafanaSpec().PortName() != nil {
		grafanaPortName = *monitor.BaseGrafanaSpec().PortName()
	}

	promethuesName := fmt.Sprintf("%s-prometheus", monitor.Name)
	prometheusService := &core.Service{
		ObjectMeta: meta.ObjectMeta{
			Name:            promethuesName,
			Namespace:       monitor.Namespace,
			Labels:          buildTidbMonitorLabel(monitor.Name),
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
			Annotations:     monitor.Spec.Prometheus.Service.Annotations,
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{
				{
					Name:       prometheusPortName,
					Port:       9090,
					Protocol:   core.ProtocolTCP,
					TargetPort: intstr.FromInt(9090),
				},
			},
			Type:     monitor.Spec.Prometheus.Service.Type,
			Selector: selector,
		},
	}
	if monitor.BasePrometheusSpec().ServiceType() == core.ServiceTypeLoadBalancer {
		if monitor.Spec.Prometheus.Service.LoadBalancerIP != nil {
			prometheusService.Spec.LoadBalancerIP = *monitor.Spec.Prometheus.Service.LoadBalancerIP
		}
	}

	reloaderService := &core.Service{
		ObjectMeta: meta.ObjectMeta{
			Name:            fmt.Sprintf("%s-monitor-reloader", monitor.Name),
			Namespace:       monitor.Namespace,
			Labels:          buildTidbMonitorLabel(monitor.Name),
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
			Annotations:     monitor.Spec.Prometheus.Service.Annotations,
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{
				{
					Name:       reloaderPortName,
					Port:       9089,
					Protocol:   core.ProtocolTCP,
					TargetPort: intstr.FromInt(9089),
				},
			},
			Type:     monitor.Spec.Reloader.Service.Type,
			Selector: selector,
		},
	}

	if monitor.BaseReloaderSpec().ServiceType() == core.ServiceTypeLoadBalancer {
		if monitor.Spec.Reloader.Service.LoadBalancerIP != nil {
			reloaderService.Spec.LoadBalancerIP = *monitor.Spec.Reloader.Service.LoadBalancerIP
		}
	}

	services = append(services, prometheusService, reloaderService)
	if monitor.Spec.Grafana != nil {
		grafanaService := &core.Service{
			ObjectMeta: meta.ObjectMeta{
				Name:            fmt.Sprintf("%s-grafana", monitor.Name),
				Namespace:       monitor.Namespace,
				Labels:          buildTidbMonitorLabel(monitor.Name),
				OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
				Annotations:     monitor.Spec.Grafana.Service.Annotations,
			},
			Spec: core.ServiceSpec{
				Ports: []core.ServicePort{
					{
						Name:       grafanaPortName,
						Port:       3000,
						Protocol:   core.ProtocolTCP,
						TargetPort: intstr.FromInt(3000),
					},
				},
				Type:     monitor.Spec.Grafana.Service.Type,
				Selector: selector,
			},
		}

		if monitor.BaseGrafanaSpec().ServiceType() == core.ServiceTypeLoadBalancer {
			if monitor.Spec.Grafana.Service.LoadBalancerIP != nil {
				grafanaService.Spec.LoadBalancerIP = *monitor.Spec.Grafana.Service.LoadBalancerIP
			}
		}

		services = append(services, grafanaService)
	}
	return services
}

func getMonitorPVC(monitor *v1alpha1.TidbMonitor) *core.PersistentVolumeClaim {
	l := buildTidbMonitorLabel(monitor.Name)
	return &core.PersistentVolumeClaim{
		ObjectMeta: meta.ObjectMeta{
			Name:        GetMonitorObjectName(monitor),
			Namespace:   monitor.Namespace,
			Labels:      l,
			Annotations: monitor.Spec.Annotations,
		},

		Spec: core.PersistentVolumeClaimSpec{
			AccessModes: []core.PersistentVolumeAccessMode{
				core.ReadWriteOnce,
			},

			Resources: core.ResourceRequirements{
				Requests: core.ResourceList{
					core.ResourceStorage: resource.MustParse(monitor.Spec.Storage),
				},
			},
			StorageClassName: monitor.Spec.StorageClassName,
		},
	}
}
