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
	"bytes"
	"fmt"
	"html/template"
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

	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
)

const tiproxyVolumeMountPath = "/var/lib/tiproxy"
const tiproxyCertVolume = "tiproxy-tls"
const tiproxyCertVolumeMountPath = "/var/lib/tiproxy-tls"

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
	PDAddr := fmt.Sprintf("%s:2379", controller.PDMemberName(tc.Name))
	// TODO: support it
	if tc.AcrossK8s() {
		return nil, fmt.Errorf("across k8s is not supported for")
	}
	if tc.Heterogeneous() && tc.WithoutLocalPD() {
		PDAddr = fmt.Sprintf("%s:2379", controller.PDMemberName(tc.Spec.Cluster.Name)) // use pd of reference cluster
	}

	state := map[string]interface{}{
		"WorkDir": filepath.Join(tiproxyVolumeMountPath, "work"),
		"PDAddr":  PDAddr,
	}
	if tc.IsTLSClusterEnabled() {
		state["PeerTLS"] = tiproxyCertVolumeMountPath
		state["ClusterClientTLS"] = util.ClusterClientTLSPath
	}
	if tc.Spec.TiDB != nil && tc.Spec.TiDB.IsTLSClientEnabled() && !tc.SkipTLSWhenConnectTiDB() {
		state["SQLClientTLS"] = tidbClientCertPath
	}

	confBuf := &bytes.Buffer{}
	if err := template.Must(template.New("config").Parse(`
workdir: {{ .WorkDir }}
proxy:
  addr: "0.0.0.0:6000"
  tcp-keep-alive: true
  max-connections: 1000
  require-backend-tls: false
  pd-addrs: {{ .PDAddr }}
  # proxy-protocol: "v2"
metrics:
api:
  addr: "0.0.0.0:3080"
  enable-basic-auth: false
  user: ""
  password: ""
log:
  level: "info"
  encoder: "tidb"
  log-file:
    filename: ""
    max-size: 300
    max-days: 1
    max-backups: 1
security:
  rsa-key-size: 4096
  sql-tls:
    {{if .SQLClientTLS}}
    ca: {{ .SQLClientTLS }}/ca.crt
    cert: {{ .SQLClientTLS }}/tls.crt
    key: {{ .SQLClientTLS }}/tls.key
    {{else}}
    skip-ca: true
    {{end}}
  cluster-tls:
    {{if .ClusterClientTLS}}
    ca: {{ .ClusterClientTLS }}/ca.crt
    cert: {{ .ClusterClientTLS }}/tls.crt
    key: {{ .ClusterClientTLS }}/tls.key
    {{end}}
  server-tls:
    {{if .ServerTLS}}
    ca: {{ .ServerTLS }}/ca.crt
    cert: {{ .ServerTLS }}/tls.crt
    key: {{ .ServerTLS }}/tls.key
    {{end}}
  peer-tls:
    {{if .PeerTLS}}
    ca: {{ .PeerTLS }}/ca.crt
    cert: {{ .PeerTLS }}/tls.crt
    key: {{ .PeerTLS }}/tls.key
    {{end}}
`)).Execute(confBuf, state); err != nil {
		return nil, err
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
			"config-file":    confBuf.String(),
			"startup-script": startScript,
		},
	}

	var inUseName string
	if set != nil {
		inUseName = mngerutils.FindConfigMapVolume(&set.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, controller.TiProxyMemberName(tc.Name))
		})
	}

	klog.Info("get tiproxy in use config map name: ", inUseName)

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

	replicas := 0
	sameConfigProxy := true
	pods := helper.GetPodOrdinals(tc.Status.TiProxy.StatefulSet.Replicas, sts)
	for id := range pods {
		podHealthy, err := m.deps.ProxyControl.IsPodHealthy(tc, int32(id))
		if err != nil {
			return err
		}
		if podHealthy {
			replicas++
		}
	}
	if replicas > 0 {
		cfg, err := m.deps.ProxyControl.GetConfigProxy(tc, pods.UnsortedList()[0])
		if err != nil {
			return err
		}
		if tc.Spec.TiProxy.Proxy != nil && (tc.Spec.TiProxy.Proxy.MaxConnections != cfg.MaxConnections || tc.Spec.TiProxy.Proxy.TCPKeepAlive != cfg.TCPKeepAlive) {
			sameConfigProxy = false
			tc.Status.TiProxy.Proxy = *cfg
			if err := m.deps.ProxyControl.SetConfigProxy(tc, pods.UnsortedList()[0], tc.Spec.TiProxy.Proxy); err != nil {
				return err
			}
		}
	}

	tc.Status.TiProxy.Synced = replicas == int(tc.TiProxyDeployDesiredReplicas()) && sameConfigProxy
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

	equal, err := controller.ServiceEqual(newSvc, oldSvc)
	if err != nil {
		return err
	}
	if !equal {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		err = controller.SetServiceLastAppliedConfigAnnotation(&svc)
		if err != nil {
			return err
		}
		_, err = m.deps.ServiceControl.UpdateService(tc, &svc)
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
	podAnnotations := util.CombineStringMap(controller.AnnProm(3080, "/api/metrics"), baseTiProxySpec.Annotations())
	stsAnnotations := getStsAnnotations(tc.Annotations, label.TiProxyLabelVal)
	headlessSvcName := controller.TiProxyPeerMemberName(tcName)

	dataVolumeName := string(v1alpha1.GetStorageVolumeName("", v1alpha1.TiProxyMemberType))
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
		{
			Name:      dataVolumeName,
			MountPath: tiproxyVolumeMountPath,
		},
		annMount,
	}

	if tc.IsTLSClusterEnabled() {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name:      tiproxyCertVolume,
			ReadOnly:  true,
			MountPath: tiproxyCertVolumeMountPath,
		}, corev1.VolumeMount{
			Name:      util.ClusterClientVolName,
			ReadOnly:  true,
			MountPath: util.ClusterClientTLSPath,
		})

		vols = append(vols, corev1.Volume{
			Name: tiproxyCertVolume, VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterTLSSecretName(tc.Name, label.TiProxyLabelVal),
				},
			},
		}, corev1.Volume{
			Name: util.ClusterClientVolName, VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterClientTLSSecretName(tc.Name),
				},
			},
		})
	}
	if tc.Spec.TiDB != nil && tc.Spec.TiDB.IsTLSClientEnabled() && !tc.SkipTLSWhenConnectTiDB() {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "tidb-client-tls", ReadOnly: true, MountPath: tidbClientCertPath,
		})

		clientSecretName := util.TiDBClientTLSSecretName(tc.Name)
		if tc.Spec.TiProxy.TLSClientSecretName != nil {
			clientSecretName = *tc.Spec.TiProxy.TLSClientSecretName
		}
		vols = append(vols, corev1.Volume{
			Name: "tidb-client-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: clientSecretName,
				},
			},
		})
	}

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

	storageRequest, err := controller.ParseStorageRequest(tc.Spec.TiProxy.Requests)
	if err != nil {
		return nil, fmt.Errorf("cannot parse storage request for TiProxy, tidbcluster %s/%s, error: %v", tc.Namespace, tc.Name, err)
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
			Partition: pointer.Int32Ptr(tc.TiProxyDeployDesiredReplicas()),
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
			Replicas: pointer.Int32Ptr(tc.TiProxyDeployDesiredReplicas()),
			Selector: stsLabels.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				util.VolumeClaimTemplate(storageRequest, dataVolumeName, tc.Spec.TiProxy.StorageClassName),
			},
			ServiceName:         headlessSvcName,
			PodManagementPolicy: baseTiProxySpec.PodManagementPolicy(),
			UpdateStrategy:      updateStrategy,
		},
	}
	tiproxySts.Spec.VolumeClaimTemplates = append(tiproxySts.Spec.VolumeClaimTemplates, additionalPVCs...)
	return tiproxySts, nil
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
