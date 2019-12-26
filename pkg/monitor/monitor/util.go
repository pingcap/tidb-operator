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
	"fmt"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strconv"
)

func getMonitorObjectName(monitor *v1alpha1.TidbMonitor) string {
	return fmt.Sprintf("%s-monitor", monitor.Name)
}

func getMonitorConfigMap(tc *v1alpha1.TidbCluster, monitor *v1alpha1.TidbMonitor) (*core.ConfigMap, error) {

	var releaseNamespaces []string
	for _, cluster := range monitor.Spec.Clusters {
		releaseNamespaces = append(releaseNamespaces, cluster.Namespace)
	}

	model := &MonitorConfigModel{
		AlertmanagerURL:    *monitor.Spec.AlertmanagerURL,
		ReleaseNamespaces:  releaseNamespaces,
		ReleaseTargetRegex: tc.Name,
		EnableTLSCluster:   tc.Spec.EnableTLSCluster,
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

func getMonitorDeployment(sa *core.ServiceAccount, config *core.ConfigMap, secret *core.Secret, monitor *v1alpha1.TidbMonitor, tc *v1alpha1.TidbCluster) *apps.Deployment {
	deployment := getMonitorDeploymentSkeleton(sa, monitor)
	initContainer := getMonitorInitContainer(monitor, tc)
	deployment.Spec.Template.Spec.InitContainers = append(deployment.Spec.Template.Spec.InitContainers, initContainer)
	prometheusContainer := getMonitorPrometheusContainer(monitor, tc)
	reloaderContainer := getMonitorReloaderContainer(monitor, tc)
	deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, prometheusContainer, reloaderContainer)
	if monitor.Spec.Grafana != nil {
		grafanaContainer := getMonitorGrafanaContainer(secret, monitor, tc)
		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, grafanaContainer)
	}
	volumes := getMonitorVolumes(config, monitor, tc)
	deployment.Spec.Template.Spec.Volumes = volumes
	return deployment
}

func getMonitorDeploymentSkeleton(sa *core.ServiceAccount, monitor *v1alpha1.TidbMonitor) *apps.Deployment {
	monitorLabel := label.New().Instance(monitor.Name).Monitor().Labels()
	replicas := int32(1)

	deployment := &apps.Deployment{
		ObjectMeta: meta.ObjectMeta{
			Name:            getMonitorObjectName(monitor),
			Namespace:       monitor.Namespace,
			Labels:          monitorLabel,
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
			Annotations:     monitor.Spec.Annotations,
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

					},
					Containers: []core.Container{

					},
					Volumes: []core.Volume{

					},
					Tolerations:  monitor.Spec.Tolerations,
					NodeSelector: monitor.Spec.NodeSelector,
				},
			},
		},
	}
	return deployment
}

func getMonitorInitContainer(monitor *v1alpha1.TidbMonitor, tc *v1alpha1.TidbCluster) core.Container {
	c := "mkdir -p /data/prometheus\nchmod 777 /data/prometheus\n/usr/bin/init.sh"
	if monitor.Spec.Grafana != nil {
		c = "mkdir -p /data/prometheus /data/grafana\nchmod 777 /data/prometheus /data/grafana\n/usr/bin/init.sh"
	}
	command := []string{
		"/bin/sh",
		"-c",
		c,
	}
	secureContext := int64(0)
	return core.Container{
		Name:            "monitor-initializer",
		Image:           fmt.Sprintf("%s:%s", monitor.Spec.Initializer.BaseImage, monitor.Spec.Initializer.Version),
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
			{
				Name:  "TIDB_CLUSTER_NAME",
				Value: tc.Name,
			},
			{
				Name:  "TIDB_ENABLE_BINLOG",
				Value: strconv.FormatBool(tc.Spec.TiDB.BinlogEnabled),
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
				Value: fmt.Sprintf("%s:%s", tc.Spec.TiDB.BaseImage, tc.Spec.TiDB.Version),
			},
			{
				Name:  "GF_K8S_PROMETHEUS_URL",
				Value: *monitor.Spec.KubePrometheusURL,
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
		Resources: util.ResourceRequirement(monitor.Spec.Initializer.Resources),
	}
}

func getMonitorPrometheusContainer(monitor *v1alpha1.TidbMonitor, tc *v1alpha1.TidbCluster) core.Container {
	c := core.Container{
		Name:            "prometheus",
		Image:           fmt.Sprintf("%s:%s", monitor.Spec.Prometheus.BaseImage, monitor.Spec.Prometheus.Version),
		ImagePullPolicy: *monitor.Spec.Prometheus.ImagePullPolicy,
		Resources:       util.ResourceRequirement(monitor.Spec.Prometheus.Resources),
		Command: []string{
			"/bin/prometheus",
			"--web.enable-admin-api",
			"--web.enable-lifecycle",
			fmt.Sprintf("--log.level=%s", monitor.Spec.Prometheus.LogLevel),
			"--config.file=/etc/prometheus/prometheus.yml",
			"--storage.tsdb.path=/data/prometheus",
			fmt.Sprintf("--storage.tsdb.retention=%dd", monitor.Spec.Prometheus.ReserveDays),
		},
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
	if tc.Spec.EnableTLSCluster {
		c.VolumeMounts = append(c.VolumeMounts, core.VolumeMount{
			Name:      "tls-pd-client",
			MountPath: "/var/lib/pd-client-tls",
			ReadOnly:  true,
		})
	}
	return c
}

func getMonitorGrafanaContainer(secret *core.Secret, monitor *v1alpha1.TidbMonitor, tc *v1alpha1.TidbCluster) core.Container {
	c := core.Container{
		Name:            "grafana",
		Image:           fmt.Sprintf("%s:%s", monitor.Spec.Grafana.BaseImage, monitor.Spec.Grafana.Version),
		ImagePullPolicy: *monitor.Spec.Grafana.ImagePullPolicy,
		Resources:       util.ResourceRequirement(monitor.Spec.Grafana.Resources),
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
	return c
}

func getMonitorReloaderContainer(monitor *v1alpha1.TidbMonitor, tc *v1alpha1.TidbCluster) core.Container {
	c := core.Container{
		Image:           fmt.Sprintf("%s:%s", monitor.Spec.Reloader.BaseImage, monitor.Spec.Reloader.Version),
		ImagePullPolicy: *monitor.Spec.Reloader.ImagePullPolicy,
		Command: []string{
			"/bin/reload",
			"--root-store-path=/data",
			fmt.Sprintf("--sub-store-path=%s", fmt.Sprintf("%s:%s", tc.Spec.TiDB.BaseImage, tc.Spec.TiDB.Version)),
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
		Resources: util.ResourceRequirement(monitor.Spec.Reloader.Resources),
		Env: []core.EnvVar{
			{
				Name:  "TZ",
				Value: tc.Spec.Timezone,
			},
		},
	}
	return c
}

func getMonitorVolumes(config *core.ConfigMap, monitor *v1alpha1.TidbMonitor, tc *v1alpha1.TidbCluster) []core.Volume {
	volumes := []core.Volume{}
	monitorData := core.Volume{
		Name: "monitor-data",
		VolumeSource: core.VolumeSource{
			EmptyDir: &core.EmptyDirVolumeSource{

			},
		},
	}
	if monitor.Spec.Persistent {
		monitorData = core.Volume{
			Name: "monitor-data",
			VolumeSource: core.VolumeSource{
				PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
					ClaimName: getMonitorObjectName(monitor),
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
						Name: "dashboards-provisioning",
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
		volumes = append(volumes, dataSource, dashboardsProvisioning)
	}
	prometheusRules := core.Volume{
		Name: "prometheus-rules",
		VolumeSource: core.VolumeSource{
			EmptyDir: &core.EmptyDirVolumeSource{},
		},
	}
	grafanaDashboard := core.Volume{
		Name: "grafana-dashboard",
		VolumeSource: core.VolumeSource{
			EmptyDir: &core.EmptyDirVolumeSource{},
		},
	}
	volumes = append(volumes, prometheusRules, grafanaDashboard)
	if tc.Spec.EnableTLSCluster {
		defaultMode := int32(420)
		tlsPDClient := core.Volume{
			Name: "tls-pd-client",
			VolumeSource: core.VolumeSource{
				Secret: &core.SecretVolumeSource{
					SecretName:  fmt.Sprintf("%s-pd-client", tc.Name),
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
	monitorLabel := label.New().Instance(monitor.Name).Monitor().Labels()
	prometheusService := &core.Service{
		ObjectMeta: meta.ObjectMeta{
			Name:            fmt.Sprintf("%s-prometheus", monitor.Name),
			Namespace:       monitor.Namespace,
			Labels:          monitorLabel,
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
			Annotations:     monitor.Spec.Prometheus.Service.Annotations,
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{
				{
					Name:       "prometheus",
					Port:       9090,
					Protocol:   core.ProtocolTCP,
					TargetPort: intstr.FromInt(9090),
				},
			},
			Type: monitor.Spec.Prometheus.Service.Type,
			Selector: map[string]string{
				label.InstanceLabelKey:  monitor.Name,
				label.ComponentLabelKey: label.TiDBMonitorVal,
			},
		},
	}
	reloaderService := &core.Service{
		ObjectMeta: meta.ObjectMeta{
			Name:            fmt.Sprintf("%s-reloader", monitor.Name),
			Namespace:       monitor.Namespace,
			Labels:          monitorLabel,
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
			Annotations:     monitor.Spec.Prometheus.Service.Annotations,
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{
				{
					Name:       "reloader",
					Port:       9089,
					Protocol:   core.ProtocolTCP,
					TargetPort: intstr.FromInt(9089),
				},
			},
			Type: monitor.Spec.Grafana.Service.Type,
			Selector: map[string]string{
				label.InstanceLabelKey:  monitor.Name,
				label.ComponentLabelKey: label.TiDBMonitorVal,
			},
		},
	}
	services = append(services, prometheusService, reloaderService)
	if monitor.Spec.Grafana != nil {
		grafanaService := &core.Service{
			ObjectMeta: meta.ObjectMeta{
				Name:            fmt.Sprintf("%s-grafana", monitor.Name),
				Namespace:       monitor.Namespace,
				Labels:          monitorLabel,
				OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
				Annotations:     monitor.Spec.Grafana.Service.Annotations,
			},
			Spec: core.ServiceSpec{
				Ports: []core.ServicePort{
					{
						Name:       "grafana",
						Port:       3000,
						Protocol:   core.ProtocolTCP,
						TargetPort: intstr.FromInt(3000),
					},
				},
				Type: monitor.Spec.Grafana.Service.Type,
				Selector: map[string]string{
					label.InstanceLabelKey:  monitor.Name,
					label.ComponentLabelKey: label.TiDBMonitorVal,
				},
			},
		}
		services = append(services, grafanaService)
	}
	return services
}

func getMonitorPVC(monitor *v1alpha1.TidbMonitor) *core.PersistentVolumeClaim {
	monitorLabel := label.New().Instance(monitor.Name).Monitor().Labels()
	return &core.PersistentVolumeClaim{
		ObjectMeta: meta.ObjectMeta{
			Name:            getMonitorObjectName(monitor),
			Namespace:       monitor.Namespace,
			Labels:          monitorLabel,
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
			Annotations:     monitor.Spec.Annotations,
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
