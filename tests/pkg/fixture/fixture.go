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

package fixture

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/tkctl/util"
	tcconfig "github.com/pingcap/tidb-operator/pkg/util/config"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
	corev1 "k8s.io/api/core/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

var (
	tikvConfig = &v1alpha1.TiKVConfig{
		LogLevel: pointer.StringPtr("info"),
		Server:   &v1alpha1.TiKVServerConfig{},
		Storage: &v1alpha1.TiKVStorageConfig{
			// Don't reserve space in e2e tests, see
			// https://github.com/pingcap/tidb-operator/issues/2509.
			ReserveSpace: pointer.StringPtr("0MB"),
		},
	}
)

var (
	BestEffort    = corev1.ResourceRequirements{}
	BurstbleSmall = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1000m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
	}
	BurstbleMedium = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2000m"),
			corev1.ResourceMemory: resource.MustParse("4Gi"),
		},
	}
	// hard-coded region and s3 bucket in our aws account for e2e testing
	// TODO create s3 bucket in current region dynamically
	AWSRegion = "us-west-2"
	Bucket    = "backup.e2e.us-west-2.tidbcloud.com"
	S3Secret  = "s3-secret"
)

func WithStorage(r corev1.ResourceRequirements, size string) corev1.ResourceRequirements {
	if r.Requests == nil {
		r.Requests = corev1.ResourceList{}
	}
	r.Requests[corev1.ResourceStorage] = resource.MustParse(size)
	return r
}

var (
	// the first version which introduces storage.reserve-space config
	// https://github.com/tikv/tikv/pull/6321
	tikvV4Beta = semver.MustParse("v4.0.0-beta")
)

// GetTidbCluster returns a TidbCluster resource configured for testing
func GetTidbCluster(ns, name, version string) *v1alpha1.TidbCluster {
	// We assume all unparsable versions are greater or equal to v4.0.0-beta,
	// e.g. nightly.
	if v, err := semver.NewVersion(version); err == nil && v.LessThan(tikvV4Beta) {
		tikvConfig.Storage = nil
	}
	deletePVP := corev1.PersistentVolumeReclaimDelete
	return &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: v1alpha1.TidbClusterSpec{
			Version:         version,
			ImagePullPolicy: corev1.PullIfNotPresent,
			PVReclaimPolicy: &deletePVP,
			SchedulerName:   "tidb-scheduler",
			Timezone:        "Asia/Shanghai",
			PD: &v1alpha1.PDSpec{
				Replicas:             3,
				BaseImage:            "pingcap/pd",
				ResourceRequirements: WithStorage(BurstbleSmall, "1Gi"),
				Config: &v1alpha1.PDConfig{
					Log: &v1alpha1.PDLogConfig{
						Level: pointer.StringPtr("info"),
					},
					// accelerate failover
					Schedule: &v1alpha1.PDScheduleConfig{
						MaxStoreDownTime: pointer.StringPtr("5m"),
					},
				},
				ComponentSpec: v1alpha1.ComponentSpec{
					Affinity: buildAffinity(name, ns, v1alpha1.PDMemberType),
				},
			},

			TiKV: &v1alpha1.TiKVSpec{
				Replicas:             3,
				BaseImage:            "pingcap/tikv",
				ResourceRequirements: WithStorage(BurstbleMedium, "10Gi"),
				MaxFailoverCount:     pointer.Int32Ptr(3),
				Config:               tikvConfig,
				ComponentSpec: v1alpha1.ComponentSpec{
					Affinity: buildAffinity(name, ns, v1alpha1.TiKVMemberType),
				},
			},

			TiDB: &v1alpha1.TiDBSpec{
				Replicas:             2,
				BaseImage:            "pingcap/tidb",
				ResourceRequirements: BurstbleMedium,
				Service: &v1alpha1.TiDBServiceSpec{
					ServiceSpec: v1alpha1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
					},
					ExposeStatus: pointer.BoolPtr(true),
				},
				SeparateSlowLog:  pointer.BoolPtr(true),
				MaxFailoverCount: pointer.Int32Ptr(3),
				Config: &v1alpha1.TiDBConfig{
					Log: &v1alpha1.Log{
						Level: pointer.StringPtr("info"),
					},
				},
				ComponentSpec: v1alpha1.ComponentSpec{
					Affinity: buildAffinity(name, ns, v1alpha1.TiDBMemberType),
				},
			},
		},
	}
}

func GetTiKVGroup(ns, name, clusterName, version string) *v1alpha1.TiKVGroup {
	// We assume all unparsable versions are greater or equal to v4.0.0-beta,
	// e.g. nightly.
	if v, err := semver.NewVersion(version); err == nil && v.LessThan(tikvV4Beta) {
		tikvConfig.Storage = nil
	}
	tg := &v1alpha1.TiKVGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: v1alpha1.TiKVGroupSpec{
			TiKVSpec: v1alpha1.TiKVSpec{
				Replicas:             3,
				ResourceRequirements: WithStorage(BurstbleMedium, "10Gi"),
				MaxFailoverCount:     pointer.Int32Ptr(3),
				Config:               tikvConfig,
				ComponentSpec: v1alpha1.ComponentSpec{
					Affinity: buildAffinity(name, ns, v1alpha1.TiKVMemberType),
					Image:    fmt.Sprintf("pingcap/tikv:%s", version),
				},
			},
			ClusterName: clusterName,
		},
	}
	return tg
}

func buildAffinity(name, namespace string, memberType v1alpha1.MemberType) *corev1.Affinity {
	return &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: int32(50),
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/component": memberType.String(),
								"app.kubernetes.io/instance":  name,
							},
						},
						Namespaces: []string{
							namespace,
						},
						TopologyKey: "rack",
					},
				},
			},
		},
	}
}

func GetTidbClusterWithTiFlash(ns, name, version string) *v1alpha1.TidbCluster {
	tc := GetTidbCluster(ns, name, version)
	tc.Spec.TiFlash = &v1alpha1.TiFlashSpec{
		Replicas:         1,
		BaseImage:        "pingcap/tiflash",
		MaxFailoverCount: pointer.Int32Ptr(3),
		StorageClaims: []v1alpha1.StorageClaim{
			{
				Resources: WithStorage(BurstbleMedium, "10Gi"),
			},
		},
	}
	return tc
}

func GetTidbInitializer(ns, tcName, initName, initPassWDName, initTLSName string) *v1alpha1.TidbInitializer {
	return &v1alpha1.TidbInitializer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      initName,
			Namespace: ns,
		},
		Spec: v1alpha1.TidbInitializerSpec{
			Image: "tnir/mysqlclient",
			Clusters: v1alpha1.TidbClusterRef{
				Name: tcName,
			},
			PasswordSecret:      &initPassWDName,
			TLSClientSecretName: &initTLSName,
		},
	}
}

func NewTidbMonitor(name, namespace string, tc *v1alpha1.TidbCluster, grafanaEnabled, persist bool) *v1alpha1.TidbMonitor {
	imagePullPolicy := corev1.PullIfNotPresent
	monitor := &v1alpha1.TidbMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.TidbMonitorSpec{
			Clusters: []v1alpha1.TidbClusterRef{
				{
					Name:      tc.Name,
					Namespace: tc.Namespace,
				},
			},
			Prometheus: v1alpha1.PrometheusSpec{
				ReserveDays: 7,
				LogLevel:    "info",
				Service: v1alpha1.ServiceSpec{
					Type:        "ClusterIP",
					Annotations: map[string]string{},
				},
				MonitorContainer: v1alpha1.MonitorContainer{
					BaseImage:            utilimage.PrometheusImage,
					Version:              utilimage.PrometheusVersion,
					ImagePullPolicy:      &imagePullPolicy,
					ResourceRequirements: corev1.ResourceRequirements{},
				},
			},
			Reloader: v1alpha1.ReloaderSpec{
				MonitorContainer: v1alpha1.MonitorContainer{
					BaseImage:            utilimage.TiDBMonitorReloaderImage,
					Version:              utilimage.TiDBMonitorReloaderVersion,
					ImagePullPolicy:      &imagePullPolicy,
					ResourceRequirements: corev1.ResourceRequirements{},
				},
				Service: v1alpha1.ServiceSpec{
					Type:        "ClusterIP",
					Annotations: map[string]string{},
				},
			},
			Initializer: v1alpha1.InitializerSpec{
				MonitorContainer: v1alpha1.MonitorContainer{
					BaseImage:            utilimage.TiDBMonitorInitializerImage,
					Version:              utilimage.TiDBMonitorInitializerVersion,
					ImagePullPolicy:      &imagePullPolicy,
					ResourceRequirements: corev1.ResourceRequirements{},
				},
				Envs: map[string]string{},
			},
			Persistent: persist,
		},
	}
	if grafanaEnabled {
		monitor.Spec.Grafana = &v1alpha1.GrafanaSpec{
			MonitorContainer: v1alpha1.MonitorContainer{
				BaseImage:            utilimage.GrafanaImage,
				Version:              utilimage.GrafanaVersion,
				ImagePullPolicy:      &imagePullPolicy,
				ResourceRequirements: corev1.ResourceRequirements{},
			},
			Username: "admin",
			Password: "admin",
			Service: v1alpha1.ServiceSpec{
				Type:        corev1.ServiceTypeClusterIP,
				Annotations: map[string]string{},
			},
			Envs: map[string]string{
				"A":    "B",
				"foo":  "hello",
				"bar":  "query",
				"some": "any",
			},
		}
	}
	if persist {
		storageClassName := "local-storage"
		monitor.Spec.StorageClassName = &storageClassName
		monitor.Spec.Storage = "2Gi"
	}
	return monitor
}

func GetBackupRole(tc *v1alpha1.TidbCluster, serviceAccountName string) *rbacv1beta1.Role {
	return &rbacv1beta1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: tc.GetNamespace(),
			Labels:    map[string]string{label.ComponentLabelKey: serviceAccountName},
		},
		Rules: []rbacv1beta1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{"pingcap.com"},
				Resources: []string{"backups", "restores"},
				Verbs:     []string{"get", "watch", "list", "update"},
			},
		},
	}
}

func GetBackupServiceAccount(tc *v1alpha1.TidbCluster, serviceAccountName string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: tc.GetNamespace(),
		},
	}
}

func GetBackupRoleBinding(tc *v1alpha1.TidbCluster, serviceAccountName string) *rbacv1beta1.RoleBinding {
	return &rbacv1beta1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: tc.GetNamespace(),
			Labels:    map[string]string{label.ComponentLabelKey: serviceAccountName},
		},
		Subjects: []rbacv1beta1.Subject{
			{
				Kind: rbacv1beta1.ServiceAccountKind,
				Name: serviceAccountName,
			},
		},
		RoleRef: rbacv1beta1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     serviceAccountName,
		},
	}
}

func GetBackupSecret(tc *v1alpha1.TidbCluster, password string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-backup-secret", tc.GetName()),
			Namespace: tc.GetNamespace(),
		},
		Data: map[string][]byte{
			"password": []byte(password),
		},
		Type: corev1.SecretTypeOpaque,
	}
}

func GetInitializerSecret(tc *v1alpha1.TidbCluster, initPassWDName, password string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      initPassWDName,
			Namespace: tc.GetNamespace(),
		},
		Data: map[string][]byte{
			"root": []byte(password),
		},
		Type: corev1.SecretTypeOpaque,
	}
}

func GetS3Secret(namespace, accessKey, secretKey string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      S3Secret,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"access_key": []byte(accessKey),
			"secret_key": []byte(secretKey),
		},
		Type: corev1.SecretTypeOpaque,
	}
}

func GetTidbClusterAutoScaler(name, ns string, tc *v1alpha1.TidbCluster, tm *v1alpha1.TidbMonitor) *v1alpha1.TidbClusterAutoScaler {
	return &v1alpha1.TidbClusterAutoScaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: v1alpha1.TidbClusterAutoScalerSpec{
			Cluster: v1alpha1.TidbClusterRef{
				Name:      tc.Name,
				Namespace: tc.Namespace,
			},
			Monitor: &v1alpha1.TidbMonitorRef{
				Name:      tm.Name,
				Namespace: tm.Namespace,
			},
			TiKV: nil,
			TiDB: nil,
		},
	}
}

const (
	BRType     = "br"
	DumperType = "dumper"
)

func GetBackupCRDWithS3(tc *v1alpha1.TidbCluster, fromSecretName, brType string, s3config *v1alpha1.S3StorageProvider) *v1alpha1.Backup {
	if brType != BRType && brType != DumperType {
		return nil
	}
	sendCredToTikv := true
	br := &v1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-backup", tc.Name),
			Namespace: tc.Namespace,
		},
		Spec: v1alpha1.BackupSpec{
			Type: v1alpha1.BackupTypeFull,
			StorageProvider: v1alpha1.StorageProvider{
				S3: s3config,
			},
			From: v1alpha1.TiDBAccessConfig{
				Host:       util.GetTidbServiceName(tc.Name),
				SecretName: fromSecretName,
				Port:       4000,
				User:       "root",
			},
			BR: &v1alpha1.BRConfig{
				Cluster:          tc.GetName(),
				ClusterNamespace: tc.GetNamespace(),
				SendCredToTikv:   &sendCredToTikv,
			},
			CleanPolicy: v1alpha1.CleanPolicyTypeDelete,
		},
	}
	if brType == DumperType {
		storage := "local-storage"
		br.Spec.BR = nil
		br.Spec.StorageClassName = &storage
		br.Spec.StorageSize = "1Gi"
	}
	return br
}

func GetRestoreCRDWithS3(tc *v1alpha1.TidbCluster, toSecretName, restoreType string, s3config *v1alpha1.S3StorageProvider) *v1alpha1.Restore {
	if restoreType != BRType && restoreType != DumperType {
		return nil
	}
	sendCredToTikv := true
	restore := &v1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-restore", tc.GetName()),
			Namespace: tc.GetNamespace(),
		},
		Spec: v1alpha1.RestoreSpec{
			Type: v1alpha1.BackupTypeFull,
			StorageProvider: v1alpha1.StorageProvider{
				S3: s3config,
			},
			To: v1alpha1.TiDBAccessConfig{
				Host:       util.GetTidbServiceName(tc.Name),
				SecretName: toSecretName,
				Port:       4000,
				User:       "root",
			},
			BR: &v1alpha1.BRConfig{
				Cluster:          tc.GetName(),
				ClusterNamespace: tc.GetNamespace(),
				SendCredToTikv:   &sendCredToTikv,
			},
		},
	}
	if restoreType == DumperType {
		storage := "local-storage"
		restore.Spec.BR = nil
		restore.Spec.StorageClassName = &storage
		restore.Spec.StorageSize = "1Gi"
		restore.Spec.S3.Path = fmt.Sprintf("s3://%s/%s", s3config.Bucket, s3config.Path)
	}
	return restore
}

func AddPumpForTidbCluster(tc *v1alpha1.TidbCluster) *v1alpha1.TidbCluster {
	if tc.Spec.Pump != nil {
		return tc
	}
	policy := corev1.PullIfNotPresent
	tc.Spec.Pump = &v1alpha1.PumpSpec{
		BaseImage: "pingcap/tidb-binlog",
		ComponentSpec: v1alpha1.ComponentSpec{
			Version:         &tc.Spec.Version,
			ImagePullPolicy: &policy,
			Affinity: &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						{
							PodAffinityTerm: corev1.PodAffinityTerm{
								Namespaces:  []string{tc.Namespace},
								TopologyKey: "rack",
							},
							Weight: 50,
						},
					},
				},
			},
			Tolerations: []corev1.Toleration{
				{
					Effect:   corev1.TaintEffectNoSchedule,
					Key:      "node-role",
					Operator: corev1.TolerationOpEqual,
					Value:    "tidb",
				},
			},
			SchedulerName:        pointer.StringPtr("default-scheduler"),
			ConfigUpdateStrategy: &tc.Spec.ConfigUpdateStrategy,
		},
		Replicas:         1,
		StorageClassName: pointer.StringPtr("local-storage"),
		ResourceRequirements: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("10Gi"),
			},
		},
		GenericConfig: tcconfig.New(map[string]interface{}{
			"addr":               "0.0.0.0:8250",
			"gc":                 7,
			"data-dir":           "/data",
			"heartbeat-interval": 2,
		}),
	}
	return tc
}
