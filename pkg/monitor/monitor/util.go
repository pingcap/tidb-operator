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
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/client"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog"
)

const (
	defaultReplicaExternalLabelName = "prometheus_replica"
)

func GetTLSAssetsSecretName(name string) string {
	return fmt.Sprintf("tidbmonitor-%s-tls-assets", name)
}

func GetMonitorObjectName(monitor *v1alpha1.TidbMonitor) string {
	return fmt.Sprintf("%s-monitor", monitor.Name)
}

func GetMonitorFirstPVCName(name string) string {
	return fmt.Sprintf(v1alpha1.TidbMonitorMemberType.String()+"-%s-monitor-0", name)
}

func GetMonitorObjectNameCrossNamespace(monitor *v1alpha1.TidbMonitor) string {
	return fmt.Sprintf("%s-%s-monitor", monitor.Namespace, monitor.Name)
}

func buildTidbMonitorLabel(name string) map[string]string {
	return label.NewMonitor().Instance(name).Monitor().Labels()
}

func getInitCommand(monitor *v1alpha1.TidbMonitor) []string {
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
	return command
}

func getGrafanaVolumeMounts() []core.VolumeMount {
	return []core.VolumeMount{
		{
			MountPath: "/etc/grafana/provisioning/datasources",
			Name:      "datasource",
			ReadOnly:  false,
		}, {
			MountPath: "/grafana-dashboard-definitions/tidb",
			Name:      "grafana-dashboard",
			ReadOnly:  false,
		},
	}
}

func getGrafanaEnvs() []core.EnvVar {
	return []core.EnvVar{
		{
			Name:  "GF_PROVISIONING_PATH",
			Value: "/grafana-dashboard-definitions/tidb",
		},
		{
			Name:  "GF_DATASOURCE_PATH",
			Value: "/etc/grafana/provisioning/datasources",
		},
	}
}

func getAlertManagerRulesVersion(tc *v1alpha1.TidbCluster, monitor *v1alpha1.TidbMonitor) string {
	alertManagerRulesVersion := fmt.Sprintf("tidb:%s", monitor.Spec.Initializer.Version)
	if monitor.Spec.AlertManagerRulesVersion != nil {
		alertManagerRulesVersion = fmt.Sprintf("tidb:%s", *monitor.Spec.AlertManagerRulesVersion)
	}
	return alertManagerRulesVersion
}

// getMonitorConfigMap generate the Prometheus config and Grafana config for TidbMonitor,
// If the namespace in ClusterRef is empty, we would set the TidbMonitor's namespace in the default
func getMonitorConfigMap(monitor *v1alpha1.TidbMonitor, monitorClusterInfos []ClusterRegexInfo, dmClusterInfos []ClusterRegexInfo) (*core.ConfigMap, error) {
	model := &MonitorConfigModel{
		AlertmanagerURL: "",
		ClusterInfos:    monitorClusterInfos,
		DMClusterInfos:  dmClusterInfos,
		ExternalLabels:  buildExternalLabels(monitor),
	}

	if len(monitor.Spec.Prometheus.RemoteWrite) > 0 {
		model.RemoteWriteConfigs = generateRemoteWrite(monitor)
	}

	if monitor.Spec.AlertmanagerURL != nil {
		model.AlertmanagerURL = *monitor.Spec.AlertmanagerURL
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

func getMonitorClusterRole(monitor *v1alpha1.TidbMonitor, policyRules []rbac.PolicyRule) *rbac.ClusterRole {
	return &rbac.ClusterRole{
		ObjectMeta: meta.ObjectMeta{
			Name:            GetMonitorObjectNameCrossNamespace(monitor),
			Namespace:       monitor.Namespace,
			Labels:          buildTidbMonitorLabel(monitor.Name),
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
		},
		Rules: policyRules,
	}
}

func getMonitorClusterRoleBinding(sa *core.ServiceAccount, role *rbac.ClusterRole, monitor *v1alpha1.TidbMonitor) *rbac.ClusterRoleBinding {
	return &rbac.ClusterRoleBinding{
		ObjectMeta: meta.ObjectMeta{
			Name:            GetMonitorObjectNameCrossNamespace(monitor),
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
			Name:     role.Name,
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

func getMonitorInitContainer(monitor *v1alpha1.TidbMonitor, tc *v1alpha1.TidbCluster) core.Container {
	command := getInitCommand(monitor)
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
				Value: getAlertManagerRulesVersion(tc, monitor),
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
				Value: tc.Timezone(),
			},
		},
		Command: command,
		VolumeMounts: []core.VolumeMount{
			{
				MountPath: "/prometheus-rules",
				Name:      "prometheus-rules",
				ReadOnly:  false,
			},
			{
				MountPath: "/data",
				Name:      v1alpha1.TidbMonitorMemberType.String(),
			},
		},
		Resources: controller.ContainerResource(monitor.Spec.Initializer.ResourceRequirements),
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
		container.VolumeMounts = append(container.VolumeMounts, getGrafanaVolumeMounts()...)
		container.Env = append(container.Env, getGrafanaEnvs()...)
	}

	var envOverrides []core.EnvVar
	for k, v := range monitor.Spec.Initializer.Envs {
		envOverrides = append(envOverrides, core.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	container.Env = util.AppendOverwriteEnv(container.Env, envOverrides)
	return container
}

func getMonitorDMInitContainer(monitor *v1alpha1.TidbMonitor, dc *v1alpha1.DMCluster, tc *v1alpha1.TidbCluster) core.Container {
	// TODO: Support dm in reloader. Currently dm cluster shares the same persistent rules dir with tidb cluster
	command := getInitCommand(monitor)
	container := core.Container{
		Name:  "dm-initializer",
		Image: fmt.Sprintf("%s:%s", monitor.Spec.DM.Initializer.BaseImage, monitor.Spec.DM.Initializer.Version),
		Env: []core.EnvVar{
			{
				Name:  "DM_CLUSTER_NAME",
				Value: dc.Name,
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
				Name:  "DM_VERSION",
				Value: getAlertManagerRulesVersion(tc, monitor),
			},
			{
				Name:  "GF_DM_PROMETHEUS_URL",
				Value: "http://127.0.0.1:9090",
			},
			{
				Name:  "DM_CLUSTER_NAMESPACE",
				Value: dc.Namespace,
			},
			{
				Name:  "TZ",
				Value: dc.Timezone(),
			},
		},
		Command: command,
		VolumeMounts: []core.VolumeMount{
			{
				MountPath: "/prometheus-rules",
				Name:      "prometheus-rules",
				ReadOnly:  false,
			},
			{
				MountPath: "/data",
				Name:      v1alpha1.TidbMonitorMemberType.String(),
			},
		},
		Resources: controller.ContainerResource(monitor.Spec.DM.Initializer.ResourceRequirements),
	}

	if monitor.Spec.DM.Initializer.ImagePullPolicy != nil {
		container.ImagePullPolicy = *monitor.Spec.DM.Initializer.ImagePullPolicy
	}

	if monitor.Spec.Grafana != nil {
		container.VolumeMounts = append(container.VolumeMounts, getGrafanaVolumeMounts()...)
		container.Env = append(container.Env, getGrafanaEnvs()...)
	}

	var envOverrides []core.EnvVar
	for k, v := range monitor.Spec.DM.Initializer.Envs {
		envOverrides = append(envOverrides, core.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	container.Env = util.AppendOverwriteEnv(container.Env, envOverrides)
	return container
}

func getMonitorPrometheusContainer(monitor *v1alpha1.TidbMonitor, tc *v1alpha1.TidbCluster) core.Container {
	commands := []string{"sed 's/$NAMESPACE/'\"$NAMESPACE\"'/g;s/$POD_NAME/'\"$POD_NAME\"'/g' /etc/prometheus/config/prometheus.yml > /etc/prometheus/config_out/prometheus.yml && /bin/prometheus --web.enable-admin-api --web.enable-lifecycle --config.file=/etc/prometheus/config_out/prometheus.yml --storage.tsdb.path=/data/prometheus " + fmt.Sprintf("--storage.tsdb.retention=%dd", monitor.Spec.Prometheus.ReserveDays)}
	c := core.Container{
		Name:      "prometheus",
		Image:     fmt.Sprintf("%s:%s", monitor.Spec.Prometheus.BaseImage, monitor.Spec.Prometheus.Version),
		Resources: controller.ContainerResource(monitor.Spec.Prometheus.ResourceRequirements),
		Command: []string{
			"/bin/sh",
			"-c",
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
				Value: tc.Timezone(),
			},
			{
				Name: "POD_NAME",
				ValueFrom: &core.EnvVarSource{
					FieldRef: &core.ObjectFieldSelector{FieldPath: "metadata.name"},
				},
			},
			{
				Name: "NAMESPACE",
				ValueFrom: &core.EnvVarSource{
					FieldRef: &core.ObjectFieldSelector{FieldPath: "metadata.namespace"},
				},
			},
		},
		VolumeMounts: []core.VolumeMount{
			{
				Name:      "prometheus-config-out",
				MountPath: "/etc/prometheus/config_out",
				ReadOnly:  false,
			},
			{
				Name:      "prometheus-config",
				MountPath: "/etc/prometheus/config",
				ReadOnly:  true,
			},
			{
				Name:      v1alpha1.TidbMonitorMemberType.String(),
				MountPath: "/data",
			},
			{
				Name:      "prometheus-rules",
				MountPath: "/prometheus-rules",
				ReadOnly:  false,
			},
			{
				Name:      "tls-assets",
				MountPath: util.ClusterAssetsTLSPath,
				ReadOnly:  true,
			},
		},
	}

	if len(monitor.Spec.Prometheus.LogLevel) > 0 {
		commands = append(commands, fmt.Sprintf("--log.level=%s", monitor.Spec.Prometheus.LogLevel))
	}
	if monitor.Spec.Prometheus.Config != nil && len(monitor.Spec.Prometheus.Config.CommandOptions) > 0 {
		commands = append(commands, monitor.Spec.Prometheus.Config.CommandOptions...)
	}
	if monitor.Spec.Prometheus.DisableCompaction || monitor.Spec.Thanos != nil {
		commands = append(commands, "--storage.tsdb.max-block-duration=2h")
		commands = append(commands, "--storage.tsdb.min-block-duration=2h")
	}

	//Add readiness probe. LivenessProbe probe will affect prom wal replay,ref: https://github.com/prometheus-operator/prometheus-operator/pull/3502
	var readinessProbeHandler core.Handler
	{
		readyPath := "/-/ready"
		readinessProbeHandler.HTTPGet = &core.HTTPGetAction{
			Path: readyPath,
			Port: intstr.FromInt(9090),
		}

	}
	readinessProbe := &core.Probe{
		Handler:          readinessProbeHandler,
		TimeoutSeconds:   3,
		PeriodSeconds:    5,
		FailureThreshold: 120, // Allow up to 10m on startup for data recovery
	}
	c.ReadinessProbe = readinessProbe

	c.Command = append(c.Command, strings.Join(commands, " "))
	if monitor.Spec.Prometheus.ImagePullPolicy != nil {
		c.ImagePullPolicy = *monitor.Spec.Prometheus.ImagePullPolicy
	}
	if monitor.Spec.Prometheus.AdditionalVolumeMounts != nil {
		c.VolumeMounts = append(c.VolumeMounts, monitor.Spec.Prometheus.AdditionalVolumeMounts...)
	}
	return c
}

func getMonitorGrafanaContainer(secret *core.Secret, monitor *v1alpha1.TidbMonitor, tc *v1alpha1.TidbCluster) core.Container {
	c := core.Container{
		Name:      "grafana",
		Image:     fmt.Sprintf("%s:%s", monitor.Spec.Grafana.BaseImage, monitor.Spec.Grafana.Version),
		Resources: controller.ContainerResource(monitor.Spec.Grafana.ResourceRequirements),
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
				Name:      v1alpha1.TidbMonitorMemberType.String(),
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

	var probeHandler core.Handler
	{
		readyPath := "/api/health"
		probeHandler.HTTPGet = &core.HTTPGetAction{
			Path: readyPath,
			Port: intstr.FromInt(3000),
		}

	}
	//add readiness probe
	readinessProbe := &core.Probe{
		Handler:          probeHandler,
		TimeoutSeconds:   5,
		PeriodSeconds:    10,
		SuccessThreshold: 1,
	}
	c.ReadinessProbe = readinessProbe

	//add liveness probe
	livenessProbe := &core.Probe{
		Handler:             probeHandler,
		TimeoutSeconds:      5,
		FailureThreshold:    10,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		InitialDelaySeconds: 30,
	}

	c.LivenessProbe = livenessProbe

	if monitor.Spec.Grafana.ImagePullPolicy != nil {
		c.ImagePullPolicy = *monitor.Spec.Grafana.ImagePullPolicy
	}
	var envOverrides []core.EnvVar
	for k, v := range monitor.Spec.Grafana.Envs {
		envOverrides = append(envOverrides, core.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	c.Env = util.AppendOverwriteEnv(c.Env, envOverrides)
	sort.Sort(util.SortEnvByName(c.Env))

	if monitor.Spec.Grafana.AdditionalVolumeMounts != nil {
		c.VolumeMounts = append(c.VolumeMounts, monitor.Spec.Grafana.AdditionalVolumeMounts...)
	}
	return c
}

func getMonitorReloaderContainer(monitor *v1alpha1.TidbMonitor, tc *v1alpha1.TidbCluster) core.Container {
	c := core.Container{
		Name:  "reloader",
		Image: fmt.Sprintf("%s:%s", monitor.Spec.Reloader.BaseImage, monitor.Spec.Reloader.Version),
		Command: []string{
			"/bin/reload",
			"--root-store-path=/data",
			fmt.Sprintf("--sub-store-path=%s", getAlertManagerRulesVersion(tc, monitor)),
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
				Name:      v1alpha1.TidbMonitorMemberType.String(),
				MountPath: "/data",
			},
		},
		Resources: controller.ContainerResource(monitor.Spec.Reloader.ResourceRequirements),
		Env: []core.EnvVar{
			{
				Name:  "TZ",
				Value: tc.Timezone(),
			},
		},
	}
	if monitor.Spec.Reloader.ImagePullPolicy != nil {
		c.ImagePullPolicy = *monitor.Spec.Reloader.ImagePullPolicy
	}
	return c
}

func getMonitorVolumes(config *core.ConfigMap, monitor *v1alpha1.TidbMonitor) []core.Volume {
	volumes := []core.Volume{}
	if !monitor.Spec.Persistent {
		monitorData := core.Volume{
			Name: v1alpha1.TidbMonitorMemberType.String(),
			VolumeSource: core.VolumeSource{
				EmptyDir: &core.EmptyDirVolumeSource{},
			},
		}
		volumes = append(volumes, monitorData)
	}
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

	volumes = append(volumes, core.Volume{
		Name: "prometheus-config-out",
		VolumeSource: core.VolumeSource{
			EmptyDir: &core.EmptyDirVolumeSource{},
		},
	})
	// add additional volumes
	if monitor.Spec.AdditionalVolumes != nil {
		volumes = append(volumes, monitor.Spec.AdditionalVolumes...)
	}

	// add asset tls
	defaultMode := int32(420)
	volumes = append(volumes, core.Volume{
		Name: "tls-assets",
		VolumeSource: core.VolumeSource{
			Secret: &core.SecretVolumeSource{
				SecretName:  GetTLSAssetsSecretName(monitor.Name),
				DefaultMode: &defaultMode,
			},
		},
	})

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

	prometheusName := prometheusName(monitor)
	monitorLabel := label.NewMonitor().Instance(monitor.Name).Monitor()
	promeLabel := monitorLabel.Copy().UsedBy("prometheus")
	grafanaLabel := monitorLabel.Copy().UsedBy("grafana")
	prometheusService := &core.Service{
		ObjectMeta: meta.ObjectMeta{
			Name:            prometheusName,
			Namespace:       monitor.Namespace,
			Labels:          util.CombineStringMap(promeLabel.Labels(), monitor.Spec.Prometheus.Service.Labels, monitor.Spec.Labels),
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
			Annotations:     util.CombineStringMap(monitor.Spec.Prometheus.Service.Annotations, monitor.Spec.Annotations),
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
		if monitor.Spec.Prometheus.Service.LoadBalancerSourceRanges != nil {
			prometheusService.Spec.LoadBalancerSourceRanges = monitor.Spec.Prometheus.Service.LoadBalancerSourceRanges
		}
	}

	if monitor.Spec.Thanos != nil {
		prometheusService.Spec.Ports = append(prometheusService.Spec.Ports, core.ServicePort{
			Name:       "thanos-grpc",
			Protocol:   core.ProtocolTCP,
			Port:       10901,
			TargetPort: intstr.FromInt(10901),
		}, core.ServicePort{
			Name:       "thanos-http",
			Protocol:   core.ProtocolTCP,
			Port:       10902,
			TargetPort: intstr.FromInt(10902),
		})
	}
	reloaderName := reloaderName(monitor)
	reloaderService := &core.Service{
		ObjectMeta: meta.ObjectMeta{
			Name:            reloaderName,
			Namespace:       monitor.Namespace,
			Labels:          util.CombineStringMap(buildTidbMonitorLabel(monitor.Name), monitor.Spec.Reloader.Service.Labels, monitor.Spec.Labels),
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
			Annotations:     util.CombineStringMap(monitor.Spec.Reloader.Service.Annotations, monitor.Spec.Annotations),
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
		if monitor.Spec.Reloader.Service.LoadBalancerSourceRanges != nil {
			reloaderService.Spec.LoadBalancerSourceRanges = monitor.Spec.Reloader.Service.LoadBalancerSourceRanges
		}
	}

	services = append(services, prometheusService, reloaderService)
	if monitor.Spec.Grafana != nil {
		grafanaService := &core.Service{
			ObjectMeta: meta.ObjectMeta{
				Name:            grafanaName(monitor),
				Namespace:       monitor.Namespace,
				Labels:          util.CombineStringMap(grafanaLabel.Labels(), monitor.Spec.Grafana.Service.Labels, monitor.Spec.Labels),
				OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
				Annotations:     util.CombineStringMap(monitor.Spec.Grafana.Service.Annotations, monitor.Spec.Annotations),
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
			if monitor.Spec.Grafana.Service.LoadBalancerSourceRanges != nil {
				grafanaService.Spec.LoadBalancerSourceRanges = monitor.Spec.Grafana.Service.LoadBalancerSourceRanges
			}
		}

		services = append(services, grafanaService)
	}

	return services
}

func getPrometheusIngress(monitor *v1alpha1.TidbMonitor) *extensionsv1beta1.Ingress {
	return getIngress(monitor, monitor.Spec.Prometheus.Ingress, prometheusName(monitor), 9090)
}

func getGrafanaIngress(monitor *v1alpha1.TidbMonitor) *extensionsv1beta1.Ingress {
	return getIngress(monitor, monitor.Spec.Grafana.Ingress, grafanaName(monitor), 3000)
}

func getIngress(monitor *v1alpha1.TidbMonitor, ingressSpec *v1alpha1.IngressSpec, svcName string, port int) *extensionsv1beta1.Ingress {
	monitorLabel := buildTidbMonitorLabel(monitor.Name)
	backend := extensionsv1beta1.IngressBackend{
		ServiceName: svcName,
		ServicePort: intstr.FromInt(port),
	}

	ingress := &extensionsv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       monitor.Namespace,
			Labels:          monitorLabel,
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
			Annotations:     ingressSpec.Annotations,
		},
		Spec: extensionsv1beta1.IngressSpec{
			TLS:   ingressSpec.TLS,
			Rules: []extensionsv1beta1.IngressRule{},
		},
	}

	for _, host := range ingressSpec.Hosts {
		rule := extensionsv1beta1.IngressRule{
			Host: host,
			IngressRuleValue: extensionsv1beta1.IngressRuleValue{
				HTTP: &extensionsv1beta1.HTTPIngressRuleValue{
					Paths: []extensionsv1beta1.HTTPIngressPath{
						{
							Path:    "/",
							Backend: backend,
						},
					},
				},
			},
		}
		ingress.Spec.Rules = append(ingress.Spec.Rules, rule)
	}
	return ingress
}

func prometheusName(monitor *v1alpha1.TidbMonitor) string {
	return fmt.Sprintf("%s-prometheus", monitor.Name)
}

func grafanaName(monitor *v1alpha1.TidbMonitor) string {
	return fmt.Sprintf("%s-grafana", monitor.Name)
}

func reloaderName(monitor *v1alpha1.TidbMonitor) string {
	return fmt.Sprintf("%s-monitor-reloader", monitor.Name)
}

func defaultTidbMonitor(monitor *v1alpha1.TidbMonitor) {
	for id, tcRef := range monitor.Spec.Clusters {
		if len(tcRef.Namespace) < 1 {
			tcRef.Namespace = monitor.Namespace
		}
		monitor.Spec.Clusters[id] = tcRef
	}
	if monitor.Spec.DM != nil {
		for id, dcRef := range monitor.Spec.DM.Clusters {
			if len(dcRef.Namespace) < 1 {
				dcRef.Namespace = monitor.Namespace
			}
			monitor.Spec.DM.Clusters[id] = dcRef
		}
	}
	retainPVP := core.PersistentVolumeReclaimRetain
	if monitor.Spec.PVReclaimPolicy == nil {
		monitor.Spec.PVReclaimPolicy = &retainPVP
	}
}

func getMonitorStatefulSet(sa *core.ServiceAccount, config *core.ConfigMap, secret *core.Secret, monitor *v1alpha1.TidbMonitor, tc *v1alpha1.TidbCluster, dc *v1alpha1.DMCluster) (*apps.StatefulSet, error) {
	statefulSet := getMonitorStatefulSetSkeleton(sa, monitor)
	initContainer := getMonitorInitContainer(monitor, tc)
	statefulSet.Spec.Template.Spec.InitContainers = append(statefulSet.Spec.Template.Spec.InitContainers, initContainer)
	if dc != nil {
		dmInitContainer := getMonitorDMInitContainer(monitor, dc, tc)
		statefulSet.Spec.Template.Spec.InitContainers = append(statefulSet.Spec.Template.Spec.InitContainers, dmInitContainer)
	}
	prometheusContainer := getMonitorPrometheusContainer(monitor, tc)
	reloaderContainer := getMonitorReloaderContainer(monitor, tc)
	statefulSet.Spec.Template.Spec.Containers = append(statefulSet.Spec.Template.Spec.Containers, prometheusContainer, reloaderContainer)
	if monitor.Spec.Thanos != nil {
		thanosSideCarContainer := getThanosSidecarContainer(monitor)
		statefulSet.Spec.Template.Spec.Containers = append(statefulSet.Spec.Template.Spec.Containers, thanosSideCarContainer)
	}
	additionalContainers := monitor.Spec.AdditionalContainers
	if len(additionalContainers) > 0 {
		statefulSet.Spec.Template.Spec.Containers = append(statefulSet.Spec.Template.Spec.Containers, additionalContainers...)
	}
	if monitor.Spec.Grafana != nil {
		grafanaContainer := getMonitorGrafanaContainer(secret, monitor, tc)
		statefulSet.Spec.Template.Spec.Containers = append(statefulSet.Spec.Template.Spec.Containers, grafanaContainer)
	}
	volumes := getMonitorVolumes(config, monitor)
	statefulSet.Spec.Template.Spec.Volumes = volumes

	volumeClaims := getMonitorVolumeClaims(monitor)
	statefulSet.Spec.VolumeClaimTemplates = volumeClaims

	if statefulSet.Annotations == nil {
		statefulSet.Annotations = map[string]string{}
	}

	if monitor.Spec.ImagePullSecrets != nil {
		statefulSet.Spec.Template.Spec.ImagePullSecrets = monitor.Spec.ImagePullSecrets
	}

	return statefulSet, nil
}

func getMonitorStatefulSetSkeleton(sa *core.ServiceAccount, monitor *v1alpha1.TidbMonitor) *apps.StatefulSet {
	replicas := int32(1)
	if monitor.Spec.Replicas != nil {
		replicas = *monitor.Spec.Replicas
	}
	name := GetMonitorObjectName(monitor)
	stsLabels := buildTidbMonitorLabel(monitor.Name)
	podLabels := util.CombineStringMap(stsLabels, monitor.Spec.Labels)
	stsAnnotations := util.CopyStringMap(monitor.Spec.Annotations)
	podAnnotations := util.CopyStringMap(monitor.Spec.Annotations)
	statefulset := &apps.StatefulSet{
		ObjectMeta: meta.ObjectMeta{
			Name:            name,
			Namespace:       monitor.Namespace,
			Labels:          stsLabels,
			OwnerReferences: []meta.OwnerReference{controller.GetTiDBMonitorOwnerRef(monitor)},
			Annotations:     stsAnnotations,
		},
		Spec: apps.StatefulSetSpec{
			ServiceName: name,
			Replicas:    &replicas,
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: apps.RollingUpdateStatefulSetStrategyType,
			},
			Selector: &meta.LabelSelector{
				MatchLabels: stsLabels,
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Labels:      podLabels,
					Annotations: podAnnotations,
				},

				Spec: core.PodSpec{
					SecurityContext:    monitor.Spec.PodSecurityContext,
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
	return statefulset
}

func getMonitorVolumeClaims(monitor *v1alpha1.TidbMonitor) []core.PersistentVolumeClaim {
	if monitor.Spec.Persistent && len(monitor.Spec.Storage) > 0 {
		var storageRequest core.ResourceRequirements
		quantity, err := resource.ParseQuantity(monitor.Spec.Storage)
		if err != nil {
			klog.Errorf("Cannot parse storage size %v in TiDBMonitor %s/%s, error: %v", monitor.Spec.Storage, monitor.Namespace, monitor.Name, err)
			return nil
		}
		storageRequest = core.ResourceRequirements{
			Requests: core.ResourceList{
				core.ResourceStorage: quantity,
			},
		}
		return []core.PersistentVolumeClaim{
			util.VolumeClaimTemplate(storageRequest, v1alpha1.TidbMonitorMemberType.String(), monitor.Spec.StorageClassName),
		}
	}
	return nil
}

func getThanosSidecarContainer(monitor *v1alpha1.TidbMonitor) core.Container {
	bindAddress := "[$(POD_IP)]"
	thanos := monitor.Spec.Thanos
	if thanos.ListenLocal {
		bindAddress = "127.0.0.1"
	}
	thanosArgs := []string{"sidecar",
		fmt.Sprintf("--prometheus.url=http://%s:9090/%s", "localhost", path.Clean(thanos.RoutePrefix)),
		fmt.Sprintf("--grpc-address=%s:10901", bindAddress),
		fmt.Sprintf("--http-address=%s:10902", bindAddress),
	}

	if thanos.GRPCServerTLSConfig != nil {
		tls := thanos.GRPCServerTLSConfig
		if tls.CertFile != "" {
			thanosArgs = append(thanosArgs, "--grpc-server-tls-cert="+tls.CertFile)
		}
		if tls.KeyFile != "" {
			thanosArgs = append(thanosArgs, "--grpc-server-tls-key="+tls.KeyFile)
		}
		if tls.CAFile != "" {
			thanosArgs = append(thanosArgs, "--grpc-server-tls-client-ca="+tls.CAFile)
		}
	}

	container := core.Container{
		Name:      "thanos-sidecar",
		Image:     fmt.Sprintf("%s:%s", thanos.BaseImage, thanos.Version),
		Resources: controller.ContainerResource(thanos.ResourceRequirements),
		Args:      thanosArgs,
		Env: []core.EnvVar{
			{
				Name: "POD_IP",
				ValueFrom: &core.EnvVarSource{
					FieldRef: &core.ObjectFieldSelector{
						FieldPath: "status.podIP",
					},
				},
			},
			{
				Name: "POD_NAME",
				ValueFrom: &core.EnvVarSource{
					FieldRef: &core.ObjectFieldSelector{FieldPath: "metadata.name"},
				},
			},
			{
				Name: "NAMESPACE",
				ValueFrom: &core.EnvVarSource{
					FieldRef: &core.ObjectFieldSelector{FieldPath: "metadata.namespace"},
				},
			},
		},
		Ports: []core.ContainerPort{
			{
				Name:          "http",
				ContainerPort: 10902,
				Protocol:      "TCP",
			},
			{
				Name:          "grpc",
				ContainerPort: 10901,
				Protocol:      "TCP",
			},
		},
	}

	if thanos.ObjectStorageConfig != nil || thanos.ObjectStorageConfigFile != nil {
		if thanos.ObjectStorageConfigFile != nil {
			container.Args = append(container.Args, "--objstore.config-file="+*thanos.ObjectStorageConfigFile)
		} else {
			container.Args = append(container.Args, "--objstore.config=$(OBJSTORE_CONFIG)")
			container.Env = append(container.Env, core.EnvVar{
				Name: "OBJSTORE_CONFIG",
				ValueFrom: &core.EnvVarSource{
					SecretKeyRef: thanos.ObjectStorageConfig,
				},
			})
		}
		storageDir := "/data/prometheus"
		container.Args = append(container.Args, fmt.Sprintf("--tsdb.path=%s", storageDir))
		container.VolumeMounts = append(
			container.VolumeMounts,
			core.VolumeMount{
				Name:      v1alpha1.TidbMonitorMemberType.String(),
				MountPath: "/data",
			},
		)
	}

	if thanos.TracingConfig != nil || thanos.TracingConfigFile != nil {
		if thanos.TracingConfigFile != nil {
			container.Args = append(container.Args, "--tracing.config-file="+*thanos.TracingConfigFile)
		} else {
			container.Args = append(container.Args, "--tracing.config=$(TRACING_CONFIG)")
			container.Env = append(container.Env, core.EnvVar{
				Name: "TRACING_CONFIG",
				ValueFrom: &core.EnvVarSource{
					SecretKeyRef: thanos.TracingConfig,
				},
			})
		}
	}

	if thanos.LogLevel != "" {
		container.Args = append(container.Args, "--log.level="+thanos.LogLevel)
	}
	if thanos.LogFormat != "" {
		container.Args = append(container.Args, "--log.format="+thanos.LogFormat)
	}

	if thanos.MinTime != "" {
		container.Args = append(container.Args, "--min-time="+thanos.MinTime)
	}
	if thanos.AdditionalVolumeMounts != nil {
		container.VolumeMounts = append(container.VolumeMounts, thanos.AdditionalVolumeMounts...)
	}
	return container
}

func buildExternalLabels(monitor *v1alpha1.TidbMonitor) model.LabelSet {
	m := model.LabelSet{}
	// Use defaultReplicaExternalLabelName constant by default if field is missing.
	// Do not add external label if field is set to empty string.
	replicaExternalLabelName := defaultReplicaExternalLabelName
	if monitor.Spec.ReplicaExternalLabelName != nil {
		if *monitor.Spec.ReplicaExternalLabelName != "" {
			replicaExternalLabelName = *monitor.Spec.ReplicaExternalLabelName
		} else {
			replicaExternalLabelName = ""
		}
	}
	if replicaExternalLabelName != "" {
		m[model.LabelName(replicaExternalLabelName)] = "$NAMESPACE_$POD_NAME"
	}
	for n, v := range monitor.Spec.ExternalLabels {
		m[model.LabelName(n)] = model.LabelValue(v)
	}
	return m
}

func generateRemoteWrite(monitor *v1alpha1.TidbMonitor) []*config.RemoteWriteConfig {
	var remoteWriteConfigs []*config.RemoteWriteConfig
	for _, remoteWrite := range monitor.Spec.Prometheus.RemoteWrite {
		url, err := client.ParseHostURL(remoteWrite.URL)
		if err != nil {
			klog.Errorf("remote write url[%s] config fail to parse, err:%v", remoteWrite.URL, err)
			continue
		}
		httpClientConfig := config.HTTPClientConfig{
			BearerTokenFile: remoteWrite.BearerTokenFile,
		}
		if remoteWrite.TLSConfig != nil {
			httpClientConfig.TLSConfig = config.TLSConfig{
				CAFile:             remoteWrite.TLSConfig.CAFile,
				CertFile:           remoteWrite.TLSConfig.CertFile,
				KeyFile:            remoteWrite.TLSConfig.KeyFile,
				ServerName:         remoteWrite.TLSConfig.ServerName,
				InsecureSkipVerify: remoteWrite.TLSConfig.InsecureSkipVerify,
			}
		}
		var writeRelabelConfigs []*config.RelabelConfig
		for _, writeRelabelConfig := range remoteWrite.WriteRelabelConfigs {
			relabelConfig := &config.RelabelConfig{}
			if len(writeRelabelConfig.SourceLabels) > 0 {
				relabelConfig.SourceLabels = writeRelabelConfig.SourceLabels
			}
			if writeRelabelConfig.Separator != "" {
				relabelConfig.Separator = writeRelabelConfig.Separator
			}
			if writeRelabelConfig.TargetLabel != "" {
				relabelConfig.TargetLabel = writeRelabelConfig.TargetLabel
			}
			if writeRelabelConfig.Regex != "" {
				regex, err := config.NewRegexp(writeRelabelConfig.Regex)
				if err != nil {
					continue
				}
				relabelConfig.Regex = regex
			}
			if writeRelabelConfig.Modulus != uint64(0) {
				relabelConfig.Modulus = writeRelabelConfig.Modulus
			}
			if writeRelabelConfig.Replacement != "" {
				relabelConfig.Replacement = writeRelabelConfig.Replacement
			}
			if writeRelabelConfig.Action != "" {
				relabelConfig.Action = writeRelabelConfig.Action
			}
			writeRelabelConfigs = append(writeRelabelConfigs, relabelConfig)
		}

		remoteWriteConfig := &config.RemoteWriteConfig{
			URL:                 &config.URL{URL: url},
			RemoteTimeout:       remoteWrite.RemoteTimeout,
			WriteRelabelConfigs: writeRelabelConfigs,
			HTTPClientConfig:    httpClientConfig,
		}
		if remoteWrite.QueueConfig != nil {
			queueConfig := config.QueueConfig{}

			if remoteWrite.QueueConfig.Capacity != 0 {
				queueConfig.Capacity = remoteWrite.QueueConfig.Capacity
			}
			if remoteWrite.QueueConfig.MaxShards != 0 {
				queueConfig.MaxShards = remoteWrite.QueueConfig.MaxShards
			}
			if remoteWrite.QueueConfig.MaxSamplesPerSend != 0 {
				queueConfig.MaxSamplesPerSend = remoteWrite.QueueConfig.MaxSamplesPerSend
			}
			if remoteWrite.QueueConfig.BatchSendDeadline != time.Duration(0) {
				queueConfig.BatchSendDeadline = remoteWrite.QueueConfig.BatchSendDeadline
			}
			if remoteWrite.QueueConfig.MaxRetries != 0 {
				queueConfig.MaxRetries = remoteWrite.QueueConfig.MaxRetries
			}
			if remoteWrite.QueueConfig.MinBackoff != time.Duration(0) {
				queueConfig.MinBackoff = remoteWrite.QueueConfig.MinBackoff
			}
			if remoteWrite.QueueConfig.MaxBackoff != time.Duration(0) {
				queueConfig.MaxBackoff = remoteWrite.QueueConfig.MaxBackoff
			}
			remoteWriteConfig.QueueConfig = queueConfig
		}
		remoteWriteConfigs = append(remoteWriteConfigs, remoteWriteConfig)
	}
	return remoteWriteConfigs
}
