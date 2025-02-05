// Copyright 2022 PingCAP, Inc.
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

package member

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/manager/member/startscript"
	"github.com/pingcap/tidb-operator/pkg/manager/suspender"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"
	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tidb-operator/pkg/util/cmpver"

	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
)

const (
	tiproxyVolumeMountPath = "/var/lib/tiproxy"

	// [security.sql-tls]
	tiproxySQLPath = "/var/lib/tiproxy-sql-tls"
	// [security.server-tls]
	tiproxyServerPath = "/var/lib/tiproxy-server-tls"
	// [security.server-http-tls]
	tiproxyHTTPServerPath = "/var/lib/tiproxy-http-server-tls"
	// [security.cluster-tls]
	tiproxyClusterCertPath        = "/var/lib/tiproxy-tls"
	tiProxyClusterCertVolumeMount = "tiproxy-tls"
)

func labelTiProxy(tc *v1alpha1.TidbCluster) label.Label {
	instanceName := tc.GetInstanceName()
	return label.New().Instance(instanceName).TiProxy()
}

// tiproxyMemberManager implements manager.Manager.
type tiproxyMemberManager struct {
	deps      *controller.Dependencies
	scaler    Scaler
	upgrader  Upgrader
	suspender suspender.Suspender
}

// NewTiProxyMemberManager returns a *tiproxyMemberManager
func NewTiProxyMemberManager(deps *controller.Dependencies, scaler Scaler, upgrader Upgrader, spder suspender.Suspender) manager.Manager {
	m := &tiproxyMemberManager{
		deps:      deps,
		scaler:    scaler,
		upgrader:  upgrader,
		suspender: spder,
	}
	return m
}

// Sync fulfills the manager.Manager interface
func (m *tiproxyMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	if tc.Spec.TiProxy == nil {
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if tc.Spec.Paused {
		klog.Infof("TidbCluster %s/%s is paused, skip syncing tiproxy service", ns, tcName)
		return nil
	}

	// skip sync if tiproxy is suspended
	component := v1alpha1.TiProxyMemberType
	needSuspend, err := m.suspender.SuspendComponent(tc, component)
	if err != nil {
		return fmt.Errorf("suspend %s failed: %v", component, err)
	}
	if needSuspend {
		klog.Infof("component %s for cluster %s/%s is suspended, skip syncing", component, ns, tcName)
		return nil
	}

	if err := m.syncProxyService(tc, false); err != nil {
		return err
	}
	if err := m.syncProxyService(tc, true); err != nil {
		return err
	}

	return m.syncStatefulSet(tc)
}

func (m *tiproxyMemberManager) syncConfigMap(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) (*corev1.ConfigMap, error) {
	PDAddr := fmt.Sprintf("%s:%d", controller.PDMemberName(tc.Name), v1alpha1.DefaultPDClientPort)
	// TODO: support it
	if tc.AcrossK8s() {
		return nil, fmt.Errorf("across k8s is not supported")
	}
	if tc.Heterogeneous() && tc.WithoutLocalPD() {
		PDAddr = fmt.Sprintf("%s:%d", controller.PDMemberName(tc.Spec.Cluster.Name), v1alpha1.DefaultPDClientPort) // use pd of reference cluster
	}

	var cfgWrapper *v1alpha1.TiProxyConfigWraper
	if tc.Spec.TiProxy.Config != nil {
		cfgWrapper = tc.Spec.TiProxy.Config.DeepCopy()
	} else {
		cfgWrapper = v1alpha1.NewTiProxyConfig()
	}

	cfgWrapper.Set("workdir", filepath.Join(tiproxyVolumeMountPath, "work"))
	cfgWrapper.Set("proxy.pd-addrs", PDAddr)
	if cfgWrapper.Get("proxy.require-backend-tls") == nil {
		cfgWrapper.Set("proxy.require-backend-tls", false)
	}

	if tc.Spec.TiProxy.CertLayout == v1alpha1.TiProxyCertLayoutV1 {
		m.modifyConfigMapForTLSV1(tc, cfgWrapper)
	} else {
		m.modifyConfigMapForTLSLegacy(tc, cfgWrapper)
	}

	cfgBytes, err := cfgWrapper.MarshalTOML()
	if err != nil {
		return nil, fmt.Errorf("render start-script for tc %s/%s failed: %v", tc.Namespace, tc.Name, err)
	}

	startScript, err := startscript.RenderTiProxyStartScript(tc)
	if err != nil {
		return nil, fmt.Errorf("render start-script for tc %s/%s failed: %v", tc.Namespace, tc.Name, err)
	}

	newCm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            controller.TiProxyMemberName(tc.Name),
			Namespace:       tc.Namespace,
			Labels:          labelTiProxy(tc),
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Data: map[string]string{
			"config-file":    string(cfgBytes),
			"startup-script": startScript,
		},
	}

	var inUseName string
	if set != nil {
		inUseName = mngerutils.FindConfigMapVolume(&set.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, controller.TiProxyMemberName(tc.Name))
		})
	} else {
		inUseName, err = mngerutils.FindConfigMapNameFromTCAnno(context.Background(), m.deps.ConfigMapLister, tc, v1alpha1.TiProxyMemberType, newCm)
		if err != nil {
			return nil, err
		}
	}

	klog.V(4).Info("get tiproxy in use config map name: ", inUseName)

	err = mngerutils.UpdateConfigMapIfNeed(m.deps.ConfigMapLister, v1alpha1.ConfigUpdateStrategyInPlace, inUseName, newCm)
	if err != nil {
		return nil, err
	}

	return m.deps.TypedControl.CreateOrUpdateConfigMap(tc, newCm)
}

func (m *tiproxyMemberManager) syncStatefulSet(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	oldStsTmp, err := m.deps.StatefulSetLister.StatefulSets(ns).Get(controller.TiProxyMemberName(tcName))
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncStatefulSet: failed to get sts %s for cluster %s/%s, error: %s", controller.TiProxyMemberName(tcName), ns, tcName, err)
	}

	stsNotExist := errors.IsNotFound(err)
	oldStatefulSet := oldStsTmp.DeepCopy()

	// failed to sync tiproxy status will not affect subsequent logic, just print the errors.
	if err := m.syncStatus(tc, oldStatefulSet); err != nil {
		klog.Errorf("failed to sync TidbCluster: [%s/%s]'s tiproxy status, error: %v",
			ns, tcName, err)
	}

	cm, err := m.syncConfigMap(tc, oldStatefulSet)
	if err != nil {
		return err
	}

	newSts, err := m.getNewStatefulSet(tc, cm)
	if err != nil {
		return err
	}

	if stsNotExist {
		err = mngerutils.SetStatefulSetLastAppliedConfigAnnotation(newSts)
		if err != nil {
			return err
		}
		err = m.deps.StatefulSetControl.CreateStatefulSet(tc, newSts)
		if err != nil {
			return err
		}
		return nil
	}

	// If setting labels fails, log and continue.
	if _, err := m.setLabelsForTiProxy(tc); err != nil {
		klog.Errorf("set labels for TiProxy sts %s/%s failed, error: %v", ns, tcName, err)
	}

	// Scaling takes precedence over upgrading because:
	// - if a pod fails in the upgrading, users may want to delete it or add
	//   new replicas
	// - it's ok to scale in the middle of upgrading (in statefulset controller
	//   scaling takes precedence over upgrading too)
	if err := m.scaler.Scale(tc, oldStatefulSet, newSts); err != nil {
		return err
	}

	if !templateEqual(newSts, oldStatefulSet) || tc.Status.TiProxy.Phase == v1alpha1.UpgradePhase {
		if err := m.upgrader.Upgrade(tc, oldStatefulSet, newSts); err != nil {
			return err
		}
	}

	return mngerutils.UpdateStatefulSetWithPrecheck(m.deps, tc, "FailedUpdateTiProxySTS", newSts, oldStatefulSet)
}

func (m *tiproxyMemberManager) syncStatus(tc *v1alpha1.TidbCluster, sts *apps.StatefulSet) error {
	if sts == nil {
		// skip if not created yet
		return nil
	}

	tc.Status.TiProxy.StatefulSet = &sts.Status
	upgrading, err := m.statefulSetIsUpgradingFn(m.deps.PodLister, sts, tc)
	if err != nil {
		tc.Status.TiProxy.Synced = false
		return err
	}
	if tc.Spec.TiProxy.Replicas != *sts.Spec.Replicas {
		tc.Status.TiProxy.Phase = v1alpha1.ScalePhase
	} else if upgrading {
		tc.Status.TiProxy.Phase = v1alpha1.UpgradePhase
	} else {
		tc.Status.TiProxy.Phase = v1alpha1.NormalPhase
	}

	pods := helper.GetPodOrdinals(tc.Status.TiProxy.StatefulSet.Replicas, sts)
	oldMembers := tc.Status.TiProxy.Members
	members := make(map[string]v1alpha1.TiProxyMember)
	for id := range pods {
		name := fmt.Sprintf("%s-%d", controller.TiProxyMemberName(tc.GetName()), id)
		memberStatus := v1alpha1.TiProxyMember{
			Name:               name,
			LastTransitionTime: metav1.Now(),
		}
		healthInfo, err := m.deps.ProxyControl.IsHealth(tc, id)
		if err != nil {
			klog.Infof("tiproxy[%d] is not health: %+v", id, err)
			memberStatus.Health = false
		} else {
			memberStatus.Health = true
			memberStatus.Info = healthInfo.String()
		}
		oldMemberStatus, exist := oldMembers[name]
		if exist && memberStatus.Health == oldMemberStatus.Health {
			memberStatus.LastTransitionTime = oldMemberStatus.LastTransitionTime
		}
		members[name] = memberStatus
	}

	tc.Status.TiProxy.Members = members
	tc.Status.TiProxy.Synced = true
	return nil
}

func (m *tiproxyMemberManager) syncProxyService(tc *v1alpha1.TidbCluster, peer bool) error {
	svcLabel := labelTiProxy(tc)
	newSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       tc.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: corev1.ServiceSpec{
			Selector:                 svcLabel,
			PublishNotReadyAddresses: true,
		},
	}
	if peer {
		newSvc.ObjectMeta.Name = controller.TiProxyPeerMemberName(tc.GetName())
		newSvc.ObjectMeta.Labels = svcLabel.Copy().UsedByPeer()
		newSvc.Spec.Type = corev1.ServiceTypeClusterIP
		newSvc.Spec.ClusterIP = "None"
		newSvc.Spec.Ports = append(newSvc.Spec.Ports,
			corev1.ServicePort{
				Name:       "tiproxy-api",
				Port:       3080,
				TargetPort: intstr.FromInt(int(3080)),
				Protocol:   corev1.ProtocolTCP,
			},
			corev1.ServicePort{
				Name:       "tiproxy-peer",
				Port:       3081,
				TargetPort: intstr.FromInt(int(3081)),
				Protocol:   corev1.ProtocolTCP,
			},
		)
	} else {
		newSvc.ObjectMeta.Name = controller.TiProxyMemberName(tc.GetName())
		newSvc.ObjectMeta.Labels = svcLabel.Copy().UsedByEndUser()
		newSvc.Spec.Type = corev1.ServiceTypeNodePort
		newSvc.Spec.Ports = append(newSvc.Spec.Ports,
			corev1.ServicePort{
				Name:       "tiproxy-api",
				Port:       3080,
				TargetPort: intstr.FromInt(int(3080)),
				Protocol:   corev1.ProtocolTCP,
			},
			corev1.ServicePort{
				Name:       "tiproxy-sql",
				Port:       6000,
				TargetPort: intstr.FromInt(int(6000)),
				Protocol:   corev1.ProtocolTCP,
			},
		)
	}
	if tc.Spec.PreferIPv6 {
		SetServiceWhenPreferIPv6(newSvc)
	}

	oldSvcTmp, err := m.deps.ServiceLister.Services(tc.GetNamespace()).Get(newSvc.ObjectMeta.Name)
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return m.deps.ServiceControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return fmt.Errorf("syncProxyService: failed to get svc %s for cluster %s/%s, error: %s", controller.TiProxyPeerMemberName(tc.GetName()), tc.GetNamespace(), tc.GetName(), err)
	}

	oldSvc := oldSvcTmp.DeepCopy()

	_, err = m.deps.ServiceControl.SyncComponentService(
		tc,
		newSvc,
		oldSvc,
		false)

	if err != nil {
		return err
	}

	return nil
}

// Only Use config file if cm is not nil
func (m *tiproxyMemberManager) getNewStatefulSet(tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
	var err error

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	baseTiProxySpec := tc.BaseTiProxySpec()
	stsLabels := labelTiProxy(tc)
	stsName := controller.TiProxyMemberName(tcName)
	podLabels := util.CombineStringMap(stsLabels, baseTiProxySpec.Labels())
	podAnnotations := util.CombineStringMap(baseTiProxySpec.Annotations(), controller.AnnProm(3080, "/api/metrics"))
	stsAnnotations := getStsAnnotations(tc.Annotations, label.TiProxyLabelVal)
	headlessSvcName := controller.TiProxyPeerMemberName(tcName)

	annMount, annVolume := annotationsMountVolume()
	vols := []corev1.Volume{
		{Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cm.Name,
					},
					Items: []corev1.KeyToPath{
						{Key: "config-file", Path: "proxy.toml"},
						{Key: "startup-script", Path: "start.sh"},
					},
				},
			},
		},
		annVolume,
	}
	volMounts := []corev1.VolumeMount{
		{Name: "config", ReadOnly: true, MountPath: "/etc/proxy"},
		annMount,
	}

	var (
		tlsVols      []corev1.Volume
		tlsVolMounts []corev1.VolumeMount
	)
	if tc.Spec.TiProxy.CertLayout == v1alpha1.TiProxyCertLayoutV1 {
		tlsVols, tlsVolMounts = m.buildVolumesAndVolumeMountsForTLSV1(tc)
	} else {
		tlsVols, tlsVolMounts = m.buildVolumesAndVolumeMountsForTLSLegacy(tc)
	}
	vols = append(vols, tlsVols...)
	volMounts = append(volMounts, tlsVolMounts...)

	// handle StorageVolumes and AdditionalVolumeMounts in ComponentSpec
	storageVolMounts, additionalPVCs := util.BuildStorageVolumeAndVolumeMount(tc.Spec.TiProxy.StorageVolumes, tc.Spec.TiProxy.StorageClassName, v1alpha1.TiProxyMemberType)
	volMounts = append(volMounts, storageVolMounts...)
	volMounts = append(volMounts, tc.Spec.TiProxy.AdditionalVolumeMounts...)

	envs := []corev1.EnvVar{
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: "NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name:  "HEADLESS_SERVICE_NAME",
			Value: headlessSvcName,
		},
	}

	tiproxyContainer := corev1.Container{
		Name:            v1alpha1.TiProxyMemberType.String(),
		Image:           tc.TiProxyImage(),
		ImagePullPolicy: baseTiProxySpec.ImagePullPolicy(),
		Command:         []string{"/bin/sh", "/etc/proxy/start.sh"},
		Ports: []corev1.ContainerPort{
			{
				Name:          "tiproxy",
				ContainerPort: int32(6000),
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "tiproxy-api",
				ContainerPort: int32(3080),
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "tiproxy-peer",
				ContainerPort: int32(3081),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: volMounts,
		Resources:    controller.ContainerResource(tc.Spec.TiProxy.ResourceRequirements),
		Env:          util.AppendEnv(envs, baseTiProxySpec.Env()),
		EnvFrom:      baseTiProxySpec.EnvFrom(),
	}

	podSpec := baseTiProxySpec.BuildPodSpec()

	podSpec.Containers, err = MergePatchContainers([]corev1.Container{tiproxyContainer}, baseTiProxySpec.AdditionalContainers())
	if err != nil {
		return nil, fmt.Errorf("failed to merge containers spec for TiProxy of [%s/%s], error: %v", ns, tcName, err)
	}

	podSpec.Volumes = append(vols, baseTiProxySpec.AdditionalVolumes()...)
	podSpec.ServiceAccountName = tc.Spec.TiProxy.ServiceAccount
	podSpec.InitContainers = append(podSpec.InitContainers, baseTiProxySpec.InitContainers()...)
	if podSpec.ServiceAccountName == "" {
		podSpec.ServiceAccountName = tc.Spec.ServiceAccount
	}

	updateStrategy := apps.StatefulSetUpdateStrategy{}
	if baseTiProxySpec.StatefulSetUpdateStrategy() == apps.OnDeleteStatefulSetStrategyType {
		updateStrategy.Type = apps.OnDeleteStatefulSetStrategyType
	} else {
		updateStrategy.Type = apps.RollingUpdateStatefulSetStrategyType
		updateStrategy.RollingUpdate = &apps.RollingUpdateStatefulSetStrategy{
			Partition: pointer.Int32Ptr(tc.TiProxyStsDesiredReplicas()),
		}
	}

	tiproxySts := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            stsName,
			Namespace:       ns,
			Labels:          stsLabels.Labels(),
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(tc.TiProxyStsDesiredReplicas()),
			Selector: stsLabels.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
			ServiceName:         headlessSvcName,
			PodManagementPolicy: baseTiProxySpec.PodManagementPolicy(),
			UpdateStrategy:      updateStrategy,
		},
	}
	tiproxySts.Spec.VolumeClaimTemplates = append(tiproxySts.Spec.VolumeClaimTemplates, additionalPVCs...)
	return tiproxySts, nil
}

func (m *tiproxyMemberManager) modifyConfigMapForTLSLegacy(tc *v1alpha1.TidbCluster, cfgWrapper *v1alpha1.TiProxyConfigWraper) {
	tlsCluster := tc.IsTLSClusterEnabled()
	tlsTiDB := tc.Spec.TiDB != nil && tc.Spec.TiDB.IsTLSClientEnabled()
	if tlsCluster {
		cfgWrapper.Set("security.cluster-tls.ca", path.Join(util.ClusterClientTLSPath, "ca.crt"))
		cfgWrapper.Set("security.cluster-tls.key", path.Join(util.ClusterClientTLSPath, "tls.key"))
		cfgWrapper.Set("security.cluster-tls.cert", path.Join(util.ClusterClientTLSPath, "tls.crt"))
	}
	if tlsTiDB {
		cfgWrapper.Set("security.server-tls.ca", path.Join(tiproxyServerPath, "ca.crt"))
		cfgWrapper.Set("security.server-tls.key", path.Join(tiproxyServerPath, "tls.key"))
		cfgWrapper.Set("security.server-tls.cert", path.Join(tiproxyServerPath, "tls.crt"))
		if cfgWrapper.Get("security.server-tls.skip-ca") == nil {
			cfgWrapper.Set("security.server-tls.skip-ca", true)
		}

		if tc.Spec.TiProxy.SSLEnableTiDB || !tc.SkipTLSWhenConnectTiDB() {
			if cfgWrapper.Get("security.sql-tls.skip-ca") == nil && tc.Spec.TiDB.TLSClient.SkipInternalClientCA {
				cfgWrapper.Set("security.sql-tls.skip-ca", true)
			} else {
				cfgWrapper.Set("security.sql-tls.ca", path.Join(tiproxySQLPath, "ca.crt"))
			}
			if !tc.Spec.TiDB.TLSClient.DisableClientAuthn {
				cfgWrapper.Set("security.sql-tls.key", path.Join(tiproxySQLPath, "tls.key"))
				cfgWrapper.Set("security.sql-tls.cert", path.Join(tiproxySQLPath, "tls.crt"))
			}
		}
	}
	// TODO: this should only be set on `tlsCluster`. `tlsTiDB` check is for backward compatibility.
	// and it should be removed in the future.
	if tlsCluster || tlsTiDB {
		p := tiproxyServerPath
		if !tlsTiDB {
			p = tiproxyHTTPServerPath
		}
		cfgWrapper.Set("security.server-http-tls.ca", path.Join(p, "ca.crt"))
		cfgWrapper.Set("security.server-http-tls.key", path.Join(p, "tls.key"))
		cfgWrapper.Set("security.server-http-tls.cert", path.Join(p, "tls.crt"))
		cfgWrapper.Set("security.server-http-tls.skip-ca", true)
	}
}

func (m *tiproxyMemberManager) buildVolumesAndVolumeMountsForTLSLegacy(tc *v1alpha1.TidbCluster) ([]corev1.Volume, []corev1.VolumeMount) {
	var (
		vols      []corev1.Volume
		volMounts []corev1.VolumeMount
	)

	tlsCluster := tc.IsTLSClusterEnabled()
	tlsTiDB := tc.Spec.TiDB != nil && tc.Spec.TiDB.IsTLSClientEnabled()
	if tlsCluster {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name:      util.ClusterClientVolName,
			ReadOnly:  true,
			MountPath: util.ClusterClientTLSPath,
		})

		vols = append(vols, corev1.Volume{
			Name: util.ClusterClientVolName, VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterClientTLSSecretName(tc.Name),
				},
			},
		})
	}
	if tlsTiDB {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "tidb-server-tls", ReadOnly: true, MountPath: tiproxyServerPath,
		})

		vols = append(vols, corev1.Volume{
			Name: "tidb-server-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.TiDBServerTLSSecretName(tc.Name),
				},
			},
		})

		if tc.Spec.TiProxy.SSLEnableTiDB || !tc.SkipTLSWhenConnectTiDB() {
			volMounts = append(volMounts, corev1.VolumeMount{
				Name: "tidb-client-tls", ReadOnly: true, MountPath: tiproxySQLPath,
			})
			vols = append(vols, corev1.Volume{
				Name: "tidb-client-tls", VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: util.TiDBClientTLSSecretName(tc.Name, tc.Spec.TiProxy.TLSClientSecretName),
					},
				},
			})
		}
	}
	if tlsCluster && !tlsTiDB {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "tiproxy-tls", ReadOnly: true, MountPath: tiproxyHTTPServerPath,
		})

		vols = append(vols, corev1.Volume{
			Name: "tiproxy-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterTLSSecretName(tc.Name, label.TiProxyLabelVal),
				},
			},
		})
	}
	return vols, volMounts
}

func (m *tiproxyMemberManager) buildVolumesAndVolumeMountsForTLSV1(tc *v1alpha1.TidbCluster) ([]corev1.Volume, []corev1.VolumeMount) {
	var (
		vols      []corev1.Volume
		volMounts []corev1.VolumeMount
	)

	tlsCluster := tc.IsTLSClusterEnabled()
	tlsTiDB := tc.Spec.TiDB != nil && tc.Spec.TiDB.IsTLSClientEnabled()
	if tlsCluster {
		// This cert is not directly used by tiproxy itself, but mount it is useful for debug purpose.
		// the common client mTLS cert
		volMounts = append(volMounts, corev1.VolumeMount{
			Name:      util.ClusterClientVolName,
			ReadOnly:  true,
			MountPath: util.ClusterClientTLSPath,
		})
		vols = append(vols, corev1.Volume{
			Name: util.ClusterClientVolName, VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterClientTLSSecretName(tc.Name),
				},
			},
		})

		// inter-components (cluster) mTLS cert for tiproxy
		volMounts = append(volMounts, corev1.VolumeMount{
			Name:      tiProxyClusterCertVolumeMount,
			ReadOnly:  true,
			MountPath: tiproxyClusterCertPath,
		})
		vols = append(vols, corev1.Volume{
			Name: tiProxyClusterCertVolumeMount, VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterTLSSecretName(tc.Name, label.TiProxyLabelVal),
				},
			},
		})
	}

	// use the same server-side cert at SQL port (default 4000) for tiproxy and tidb
	if tlsTiDB {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "tidb-server-tls", ReadOnly: true, MountPath: tiproxyServerPath,
		})

		vols = append(vols, corev1.Volume{
			Name: "tidb-server-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.TiDBServerTLSSecretName(tc.Name),
				},
			},
		})
	}

	return vols, volMounts
}

func (m *tiproxyMemberManager) modifyConfigMapForTLSV1(tc *v1alpha1.TidbCluster, cfgWrapper *v1alpha1.TiProxyConfigWraper) {
	tlsCluster := tc.IsTLSClusterEnabled()
	tlsTiDB := tc.Spec.TiDB != nil && tc.Spec.TiDB.IsTLSClientEnabled()
	if tlsCluster {
		cfgWrapper.Set("security.cluster-tls.ca", path.Join(tiproxyClusterCertPath, "ca.crt"))
		cfgWrapper.Set("security.cluster-tls.key", path.Join(tiproxyClusterCertPath, "tls.key"))
		cfgWrapper.Set("security.cluster-tls.cert", path.Join(tiproxyClusterCertPath, "tls.crt"))
	}

	if tlsTiDB {
		cfgWrapper.Set("security.server-tls.key", path.Join(tiproxyServerPath, "tls.key"))
		cfgWrapper.Set("security.server-tls.cert", path.Join(tiproxyServerPath, "tls.crt"))
		if cfgWrapper.Get("security.server-tls.skip-ca") == nil {
			cfgWrapper.Set("security.server-tls.skip-ca", true)
		}

		// Note: We don't present any client cert/key when tiproxy connect to tidb server. Instead, just mount the ca cert
		// of tidb server-side cert to verify tidb server-side cert.
		// Client cert/key is not required for enabling TLS between SQL client and server.
		if tc.Spec.TiProxy.SSLEnableTiDB || !tc.SkipTLSWhenConnectTiDB() {
			if cfgWrapper.Get("security.sql-tls.skip-ca") == nil && tc.Spec.TiDB.TLSClient.SkipInternalClientCA {
				cfgWrapper.Set("security.sql-tls.skip-ca", true)
			} else {
				// the ca.crt in `tiproxyServerPath` is the CA cert of tidb server-side cert. So we use
				// it to verify the tidb server-side cert presented by tidb-server when tiproxy connect to tidb-server.
				cfgWrapper.Set("security.sql-tls.ca", path.Join(tiproxyServerPath, "ca.crt"))
			}
		}
	}

	// use the same cert as cluster mTLS cert for tiproxy HTTP server to simplify the cert management.
	cfgWrapper.Set("security.server-http-tls.ca", path.Join(tiproxyClusterCertPath, "ca.crt"))
	cfgWrapper.Set("security.server-http-tls.key", path.Join(tiproxyClusterCertPath, "tls.key"))
	cfgWrapper.Set("security.server-http-tls.cert", path.Join(tiproxyClusterCertPath, "tls.crt"))
	// todo: Mount `db-cluster-client-secret` to TiproxyControl (pkg/controller/tiproxy_control.go) so that we can remove this.
	cfgWrapper.Set("security.server-http-tls.skip-ca", true)
}

func (m *tiproxyMemberManager) statefulSetIsUpgradingFn(podLister corelisters.PodLister, set *apps.StatefulSet, tc *v1alpha1.TidbCluster) (bool, error) {
	if mngerutils.StatefulSetIsUpgrading(set) {
		return true, nil
	}
	instanceName := tc.GetInstanceName()
	selector, err := label.New().Instance(instanceName).TiProxy().Selector()
	if err != nil {
		return false, err
	}
	tiproxyPods, err := podLister.Pods(tc.GetNamespace()).List(selector)
	if err != nil {
		return false, fmt.Errorf("tiproxyStatefulSetIsUpgrading: failed to list pods for cluster %s/%s, selector %s, error: %s", tc.GetNamespace(), tc.GetName(), selector, err)
	}
	for _, pod := range tiproxyPods {
		revisionHash, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return false, nil
		}
		if revisionHash != tc.Status.TiProxy.StatefulSet.UpdateRevision {
			return true, nil
		}
	}

	return false, nil
}

const tiproxySupportLabelsMinVersion = "1.1.0"

func (m *tiproxyMemberManager) setLabelsForTiProxy(tc *v1alpha1.TidbCluster) (int, error) {
	tiproxyVersion := tc.TiProxyVersion()
	isOlder, err := cmpver.Compare(tiproxyVersion, cmpver.Less, tiproxySupportLabelsMinVersion)
	// meet a custom build of tiproxy without version in tag, directly return as if it was old tiproxy that doesn't support set labels
	if err != nil {
		klog.Warningf("parse tiproxy version '%s' failed, skip setting labels for TiProxy of TiDB cluster %s/%s. err: %v", tiproxyVersion, tc.Namespace, tc.Name, err)
		return 0, nil
	}
	// meet an old version tiproxy, directly return because tiproxy doesn't have configs for labels
	if isOlder {
		return 0, nil
	}
	if m.deps.NodeLister == nil || m.deps.PodLister == nil {
		klog.Infof("Node lister or pod listener is unavailable, skip setting labels for TiProxy of TiDB cluster %s/%s. This may be caused by no relevant permissions", tc.Namespace, tc.Name)
		return 0, nil
	}

	ns := tc.GetNamespace()
	// for unit test
	setCount := 0

	pdCli := controller.GetPDClient(m.deps.PDControl, tc)
	config, err := pdCli.GetConfig()
	if err != nil {
		return setCount, err
	}

	var zoneLabel string
outer:
	for _, label := range topologyZoneLabels {
		for _, l := range config.Replication.LocationLabels {
			if l == label {
				zoneLabel = l
				break outer
			}
		}
	}

	if zoneLabel == "" {
		klog.V(4).Infof("zone labels not found in pd location-labels %v, skip set labels", config.Replication.LocationLabels)
		return 0, nil
	}

	for name, proxy := range tc.Status.TiProxy.Members {
		if !proxy.Health {
			continue
		}
		ordinal, err := parserOrdinal(name)
		if err != nil {
			return setCount, err
		}

		pod, err := m.deps.PodLister.Pods(ns).Get(name)
		if err != nil || pod == nil {
			klog.Warningf("failed to get pod %s for cluster %s/%s, error: %+v", name, ns, tc.GetName(), err)
			continue
		}
		if len(pod.Spec.NodeName) == 0 {
			klog.V(4).Infof("node name of pod %s in cluster %s/%s is empty", name, ns, tc.GetName())
			continue
		}
		labels, err := getNodeLabels(m.deps.NodeLister, pod.Spec.NodeName, config.Replication.LocationLabels)
		if err != nil || len(labels) == 0 {
			klog.Warningf("node: [%s] has no node labels %v, skipping set labels for Pod: [%s/%s]", pod.Spec.NodeName, config.Replication.LocationLabels, ns, name)
			continue
		}
		// add the special `zone` label because tiproxy depends on this label for az-aware routing.
		labels[tidbDCLabel] = labels[zoneLabel]

		if err := m.deps.ProxyControl.SetLabels(tc, ordinal, labels); err != nil {
			klog.Warningf("cluster %s/%s set server labels for pod %s failed, error: %v", ns, tc.GetName(), name, err)
			continue
		}

		setCount++
	}

	return setCount, nil
}

type FakeTiProxyMemberManager struct {
	err error
}

func NewFakeTiProxyMemberManager() *FakeTiProxyMemberManager {
	return &FakeTiProxyMemberManager{}
}

func (m *FakeTiProxyMemberManager) SetSyncError(err error) {
	m.err = err
}

func (m *FakeTiProxyMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	if m.err != nil {
		return m.err
	}
	return nil
}
