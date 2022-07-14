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
	corev1 "k8s.io/api/core/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	tcconfig "github.com/pingcap/tidb-operator/pkg/apis/util/config"
	"github.com/pingcap/tidb-operator/pkg/tkctl/util"
	utilimage "github.com/pingcap/tidb-operator/tests/e2e/util/image"
)

var (
	BestEffort     = corev1.ResourceRequirements{}
	BurstableSmall = corev1.ResourceRequirements{

		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1000m"),
			corev1.ResourceMemory: resource.MustParse("2Gi"),
		},
	}
	BurstableMedium = corev1.ResourceRequirements{
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

// ClusterCustomKey for label and ann to test
const ClusterCustomKey = "cluster-test-key"

// ComponentCustomKey for label and ann to test
const ComponentCustomKey = "component-test-key"

// GetTidbCluster returns a TidbCluster resource configured for testing
func GetTidbCluster(ns, name, version string) *v1alpha1.TidbCluster {
	// We assume all unparsable versions are greater or equal to v4.0.0-beta,
	// e.g. nightly.
	tikvConfig := v1alpha1.NewTiKVConfig()
	tikvConfig.Set("log-level", "info")
	// Don't reserve space in e2e tests, see
	// https://github.com/pingcap/tidb-operator/issues/2509.
	tikvConfig.Set("storage.reserve-space", "0MB")
	if v, err := semver.NewVersion(version); err == nil && v.LessThan(tikvV4Beta) {
		tikvConfig.Del("storage")
	}
	deletePVP := corev1.PersistentVolumeReclaimDelete

	tidbConfig := v1alpha1.NewTiDBConfig()
	tidbConfig.Set("log.level", "info")

	return &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: v1alpha1.TidbClusterSpec{
			Version:              version,
			ImagePullPolicy:      corev1.PullIfNotPresent,
			PVReclaimPolicy:      &deletePVP,
			ConfigUpdateStrategy: v1alpha1.ConfigUpdateStrategyRollingUpdate,
			Helper: &v1alpha1.HelperSpec{
				Image: pointer.StringPtr(utilimage.HelperImage),
			},
			SchedulerName: "tidb-scheduler",
			Timezone:      "Asia/Shanghai",
			Labels: map[string]string{
				ClusterCustomKey: "value",
			},
			Annotations: map[string]string{
				ClusterCustomKey: "value",
			},
			PD: &v1alpha1.PDSpec{
				Replicas:             3,
				BaseImage:            "pingcap/pd",
				ResourceRequirements: WithStorage(BurstableSmall, "1Gi"),
				Config: func() *v1alpha1.PDConfigWraper {
					c := v1alpha1.NewPDConfig()
					c.Set("log.level", "info")
					c.Set("schedule.max-store-down-time", "5m")
					return c
				}(),
				ComponentSpec: v1alpha1.ComponentSpec{
					Affinity: buildAffinity(name, ns, v1alpha1.PDMemberType),
					Labels: map[string]string{
						ComponentCustomKey: "value",
					},
					Annotations: map[string]string{
						ComponentCustomKey: "value",
					},
				},
			},

			TiKV: &v1alpha1.TiKVSpec{
				Replicas:             3,
				BaseImage:            "pingcap/tikv",
				ResourceRequirements: WithStorage(BurstableMedium, "10Gi"),
				MaxFailoverCount:     pointer.Int32Ptr(3),
				Config:               tikvConfig,
				ComponentSpec: v1alpha1.ComponentSpec{
					Affinity: buildAffinity(name, ns, v1alpha1.TiKVMemberType),
					Labels: map[string]string{
						ComponentCustomKey: "value",
					},
					Annotations: map[string]string{
						ComponentCustomKey: "value",
					},
				},
				EvictLeaderTimeout: pointer.StringPtr("1m"),
			},

			TiDB: &v1alpha1.TiDBSpec{
				Replicas:             2,
				BaseImage:            "pingcap/tidb",
				ResourceRequirements: BurstableMedium,
				Service: &v1alpha1.TiDBServiceSpec{
					ServiceSpec: v1alpha1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
						Labels: map[string]string{
							ComponentCustomKey: "value",
						},
						Annotations: map[string]string{
							ComponentCustomKey: "value",
						},
					},
					ExposeStatus: pointer.BoolPtr(true),
				},
				SeparateSlowLog:  pointer.BoolPtr(true),
				MaxFailoverCount: pointer.Int32Ptr(3),
				Config:           tidbConfig,
				ComponentSpec: v1alpha1.ComponentSpec{
					Affinity: buildAffinity(name, ns, v1alpha1.TiDBMemberType),
					Labels: map[string]string{
						ComponentCustomKey: "value",
					},
					Annotations: map[string]string{
						ComponentCustomKey: "value",
					},
				},
			},
		},
	}
}

// GetDMCluster returns a DmCluster resource configured for testing.
func GetDMCluster(ns, name, version string) *v1alpha1.DMCluster {
	deletePVP := corev1.PersistentVolumeReclaimDelete
	return &v1alpha1.DMCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: v1alpha1.DMClusterSpec{
			Version:         version,
			ImagePullPolicy: corev1.PullIfNotPresent,
			PVReclaimPolicy: &deletePVP,
			SchedulerName:   "tidb-scheduler",
			Timezone:        "Asia/Shanghai",
			Labels: map[string]string{
				ClusterCustomKey: "value",
			},
			Annotations: map[string]string{
				ClusterCustomKey: "value",
			},
			Master: v1alpha1.MasterSpec{
				Replicas:             3,
				BaseImage:            "pingcap/dm",
				MaxFailoverCount:     pointer.Int32Ptr(3),
				StorageSize:          "1Gi",
				ResourceRequirements: WithStorage(BurstableSmall, "1Gi"),
				Config:               v1alpha1.NewMasterConfig(),
				Service: &v1alpha1.MasterServiceSpec{
					ServiceSpec: v1alpha1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
						Labels: map[string]string{
							ComponentCustomKey: "value",
						},
						Annotations: map[string]string{
							ComponentCustomKey: "value",
						},
					},
				},
				ComponentSpec: v1alpha1.ComponentSpec{
					Affinity: buildAffinity(name, ns, v1alpha1.DMMasterMemberType),
					Labels: map[string]string{
						ComponentCustomKey: "value",
					},
					Annotations: map[string]string{
						ComponentCustomKey: "value",
					},
				},
			},
			Worker: &v1alpha1.WorkerSpec{
				Replicas:             3,
				BaseImage:            "pingcap/dm",
				MaxFailoverCount:     pointer.Int32Ptr(3),
				ResourceRequirements: WithStorage(BurstableSmall, "1Gi"),
				Config:               v1alpha1.NewWorkerConfig(),
				ComponentSpec: v1alpha1.ComponentSpec{
					Affinity: buildAffinity(name, ns, v1alpha1.DMWorkerMemberType),
					Labels: map[string]string{
						ComponentCustomKey: "value",
					},
					Annotations: map[string]string{
						ComponentCustomKey: "value",
					},
				},
			},
		},
	}
}

func UpdateTidbMonitorForDM(tm *v1alpha1.TidbMonitor, dc *v1alpha1.DMCluster) {
	imagePullPolicy := *tm.Spec.Initializer.ImagePullPolicy
	tm.Spec.DM = &v1alpha1.DMMonitorSpec{
		Clusters: []v1alpha1.ClusterRef{
			{
				Name:      dc.Name,
				Namespace: dc.Namespace,
			},
		},
		Initializer: v1alpha1.InitializerSpec{
			MonitorContainer: v1alpha1.MonitorContainer{
				BaseImage:            utilimage.DMMonitorInitializerImage,
				Version:              utilimage.DMMonitorInitializerVersion,
				ImagePullPolicy:      &imagePullPolicy,
				ResourceRequirements: corev1.ResourceRequirements{},
			},
			Envs: map[string]string{},
		},
	}
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

func GetTidbInitializer(ns, tcName, initName, initPassWDName, initTLSName string) *v1alpha1.TidbInitializer {
	return &v1alpha1.TidbInitializer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      initName,
			Namespace: ns,
			Labels: map[string]string{
				ClusterCustomKey: "value",
			},
			Annotations: map[string]string{
				ClusterCustomKey: "value",
			},
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

func NewTidbMonitor(name, namespace string, tc *v1alpha1.TidbCluster, grafanaEnabled, persist bool, isRetain bool) *v1alpha1.TidbMonitor {
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
			Labels: map[string]string{
				ClusterCustomKey: "value",
			},
			Annotations: map[string]string{
				ClusterCustomKey: "value",
			},
			Prometheus: v1alpha1.PrometheusSpec{
				ReserveDays: 7,
				LogLevel:    "info",
				Service: v1alpha1.ServiceSpec{
					Type: "ClusterIP",
					Labels: map[string]string{
						ComponentCustomKey: "value",
					},
					Annotations: map[string]string{
						ComponentCustomKey: "value",
					},
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
					Type: "ClusterIP",
					Labels: map[string]string{
						ComponentCustomKey: "value",
					},
					Annotations: map[string]string{
						ComponentCustomKey: "value",
					},
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
			Thanos: &v1alpha1.ThanosSpec{
				MonitorContainer: v1alpha1.MonitorContainer{
					BaseImage:            utilimage.ThanosImage,
					Version:              utilimage.ThanosVersion,
					ImagePullPolicy:      &imagePullPolicy,
					ResourceRequirements: corev1.ResourceRequirements{},
				},
			},
		},
	}
	if isRetain {
		retainPVP := corev1.PersistentVolumeReclaimRetain
		monitor.Spec.PVReclaimPolicy = &retainPVP
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
				Type: corev1.ServiceTypeClusterIP,
				Labels: map[string]string{
					ComponentCustomKey: "value",
				},
				Annotations: map[string]string{
					ComponentCustomKey: "value",
				},
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
		monitor.Spec.Storage = "2Gi"
		monitor.Spec.Persistent = true
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
			From: &v1alpha1.TiDBAccessConfig{
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
		br.Spec.BR = nil
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
			To: &v1alpha1.TiDBAccessConfig{
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
		restore.Spec.BR = nil
		restore.Spec.StorageSize = "1Gi"
		restore.Spec.S3.Path = fmt.Sprintf("s3://%s/%s", s3config.Bucket, s3config.Path)
	}
	return restore
}

func GetTidbNGMonitoring(ns, name string, tc *v1alpha1.TidbCluster) *v1alpha1.TidbNGMonitoring {
	deletePVP := corev1.PersistentVolumeReclaimDelete
	version := utilimage.TiDBNGMonitoringLatest
	cfgUpdateStrategy := v1alpha1.ConfigUpdateStrategyRollingUpdate

	tngm := &v1alpha1.TidbNGMonitoring{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: v1alpha1.TidbNGMonitoringSpec{
			Clusters: []v1alpha1.TidbClusterRef{
				{
					Name:      tc.Name,
					Namespace: tc.Namespace,
				},
			},

			ComponentSpec: v1alpha1.ComponentSpec{
				ConfigUpdateStrategy: &cfgUpdateStrategy,
			},
			PVReclaimPolicy: &deletePVP,
			NGMonitoring: v1alpha1.NGMonitoringSpec{
				ComponentSpec: v1alpha1.ComponentSpec{
					Version: &version,
				},

				BaseImage:            "pingcap/ng-monitoring",
				ResourceRequirements: WithStorage(BurstableSmall, "1Gi"),
			},
		},
	}

	return tngm
}

func AddTiFlashForTidbCluster(tc *v1alpha1.TidbCluster) *v1alpha1.TidbCluster {
	if tc.Spec.TiFlash != nil {
		return tc
	}
	tc.Spec.TiFlash = &v1alpha1.TiFlashSpec{
		Replicas:         1,
		BaseImage:        "pingcap/tiflash",
		MaxFailoverCount: pointer.Int32Ptr(3),
		StorageClaims: []v1alpha1.StorageClaim{
			{
				Resources: WithStorage(BurstableMedium, "10Gi"),
			},
		},
	}
	return tc
}

func AddTiCDCForTidbCluster(tc *v1alpha1.TidbCluster) *v1alpha1.TidbCluster {
	if tc.Spec.TiCDC != nil {
		return tc
	}
	tc.Spec.TiCDC = &v1alpha1.TiCDCSpec{
		BaseImage: "pingcap/ticdc",
		Replicas:  1,
		Config: func() *v1alpha1.CDCConfigWraper {
			cfg := v1alpha1.NewCDCConfig()
			cfg.SetIfNil("log-file", "") // to avoid rolling upgrade due to PR #4494 when upgrading operator
			return cfg
		}(),
	}
	return tc
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
		Replicas: 1,
		ResourceRequirements: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("10Gi"),
			},
		},
		Config: tcconfig.New(map[string]interface{}{
			"addr":               "0.0.0.0:8250",
			"gc":                 7,
			"data-dir":           "/data",
			"heartbeat-interval": 2,
		}),
	}
	return tc
}
