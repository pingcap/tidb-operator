// Copyright 2021 PingCAP, Inc.
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

package tidbngmonitoring

import (
	"fmt"
	"path"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/util/config"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager/member"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"
	"github.com/pingcap/tidb-operator/pkg/util"

	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
)

const (
	ngmPodDataVolumeMountDir   = "/var/lib/ng-monitoring" // the mount path for ng monitoring data volume
	ngmPodConfigVolumeMountDir = "/etc/ng-monitoring"     // the dir for ng monitoring config
	ngmPodConfigFilename       = "ng-monitoring.toml"     // the filename of config file
	ngmTCClientTLSMountDir     = "/var/lib/tc-client-tls" // the dir for tc client tls

	ngmConfigMapConfigKey = "config-file" // the key for config data in config map

	ngmServicePort = 12020
)

type ngMonitoringManager struct {
	deps *controller.Dependencies
}

func NewNGMonitorManager(deps *controller.Dependencies) *ngMonitoringManager {
	return &ngMonitoringManager{
		deps: deps,
	}
}

func (m *ngMonitoringManager) Sync(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) error {
	var err error

	err = m.syncService(tngm)
	if err != nil {
		return err
	}

	err = m.syncCore(tngm, tc)
	if err != nil {
		return err
	}

	return nil
}

func (m *ngMonitoringManager) syncService(tngm *v1alpha1.TidbNGMonitoring) error {
	ns := tngm.GetNamespace()
	name := tngm.GetName()

	if tngm.Spec.Paused {
		klog.V(4).Infof("tidb ng monitoring %s/%s is paused, skip syncing for ng monitoring headless service", ns, name)
		return nil
	}

	newSvc := GenerateNGMonitoringHeadlessService(tngm)
	oldSvc, err := m.deps.ServiceLister.Services(newSvc.Namespace).Get(newSvc.Name)
	svcNotFound := errors.IsNotFound(err)

	if err != nil && !svcNotFound {
		return fmt.Errorf("syncService: failed to get headless svc %s/%s for ng monitoring %s/%s, error %s", newSvc.Namespace, newSvc.Name, ns, name, err)
	}

	// first creation
	if svcNotFound {
		err := controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return m.deps.ServiceControl.CreateService(tngm, newSvc)
	}

	// update existing service if needed
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
		_, err = m.deps.ServiceControl.UpdateService(tngm, &svc)
		return err
	}

	return nil
}

func (m *ngMonitoringManager) syncCore(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) error {
	ns := tngm.GetNamespace()
	name := tngm.GetName()

	stsName := NGMonitoringName(name)
	oldStsTemp, err := m.deps.StatefulSetLister.StatefulSets(ns).Get(stsName)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("populateStatus: failed to get sts %s for tidb ng monitor %s/%s, error: %s", stsName, ns, name, err)
	}
	stsNotFound := errors.IsNotFound(err)
	oldSts := oldStsTemp.DeepCopy()

	// sync status
	err = m.populateStatus(tngm, oldSts)
	if err != nil {
		klog.Errorf("failed to sync ng monitoring's status of tidb ng monitring %s/%s, error: %v", ns, name, err)
		return err
	}

	if tngm.Spec.Paused {
		klog.V(4).Infof("tidb ng monitring %s/%s is paused, skip syncing for ng monitoring statefulset", ns, name)
		return nil
	}

	// sync resources

	cm, err := m.syncConfigMap(tngm, tc, oldSts)
	if err != nil {
		klog.Errorf("failed to sync ng monitoring's configmap of tidb ng monitring %s/%s, error: %v", ns, name, err)
		return err
	}

	newSts, err := GenerateNGMonitoringStatefulSet(tngm, tc, cm)
	if err != nil {
		return err
	}

	// first creation
	if stsNotFound {
		err = mngerutils.SetStatefulSetLastAppliedConfigAnnotation(newSts)
		if err != nil {
			return err
		}
		return m.deps.StatefulSetControl.CreateStatefulSet(tngm, newSts)
	}

	// update existing statefulset if needed
	return mngerutils.UpdateStatefulSetWithPrecheck(m.deps, tc, "FailedUpdateNGMSTS", newSts, oldSts)
}

func (m *ngMonitoringManager) syncConfigMap(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster, sts *apps.StatefulSet) (*corev1.ConfigMap, error) {
	spec := tngm.BaseNGMonitoringSpec()

	newCM, err := GenerateNGMonitoringConfigMap(tngm, tc)
	if err != nil {
		return nil, err
	}

	var inUseName string
	if sts != nil {
		inUseName = mngerutils.FindConfigMapVolume(&sts.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, NGMonitoringName(tngm.Name))
		})
	}

	err = mngerutils.UpdateConfigMapIfNeed(m.deps.ConfigMapLister, spec.ConfigUpdateStrategy(), inUseName, newCM)
	if err != nil {
		return nil, err
	}

	return m.deps.TypedControl.CreateOrUpdateConfigMap(tngm, newCM)
}

func (m *ngMonitoringManager) populateStatus(tngm *v1alpha1.TidbNGMonitoring, sts *apps.StatefulSet) error {
	if sts == nil {
		return nil // skip if not created yet
	}

	tngm.Status.NGMonitoring.StatefulSet = &sts.Status

	upgrading, err := m.confirmStatefulSetIsUpgrading(tngm, sts)
	if err != nil {
		tngm.Status.NGMonitoring.Synced = false
		return err
	}
	if upgrading {
		tngm.Status.NGMonitoring.Phase = v1alpha1.UpgradePhase
	} else {
		tngm.Status.NGMonitoring.Phase = v1alpha1.NormalPhase
	}

	tngm.Status.NGMonitoring.Synced = true

	return nil
}

func (m *ngMonitoringManager) confirmStatefulSetIsUpgrading(tngm *v1alpha1.TidbNGMonitoring, oldSts *apps.StatefulSet) (bool, error) {
	if mngerutils.StatefulSetIsUpgrading(oldSts) {
		return true, nil
	}

	selector, err := label.NewTiDBNGMonitoring().
		Instance(tngm.GetInstanceName()).
		NGMonitoring().
		Selector()
	if err != nil {
		return false, err
	}

	pods, err := m.deps.PodLister.Pods(tngm.GetNamespace()).List(selector)
	if err != nil {
		return false, fmt.Errorf("confirmStatefulSetIsUpgrading: failed to list pod for tidb ng monitor %s/%s, selector %s, error: %s", tngm.GetNamespace(), tngm.GetName(), selector, err)
	}

	for _, pod := range pods {
		revisionHash, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return false, nil
		}
		if revisionHash != oldSts.Status.UpdateRevision {
			return true, nil
		}
	}
	return false, nil
}

func GenerateNGMonitoringStatefulSet(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
	ns := tngm.GetNamespace()
	name := tngm.GetName()

	if cm == nil {
		return nil, fmt.Errorf("tidb ng monitoring [%s/%s] config map is nil", ns, name)
	}

	spec := tngm.BaseNGMonitoringSpec()
	meta, stsLabels := GenerateNGMonitoringMeta(tngm, NGMonitoringName(tngm.Name))
	headlessServiceName := NGMonitoringHeadlessServiceName(name)
	replicas := int32(1) // only support one replica now

	dataVolumeName := v1alpha1.NGMonitoringMemberType.String()
	configVolumeName := "config"

	baseContainers := []corev1.Container{}
	nmContainerName := v1alpha1.NGMonitoringMemberType.String()
	startScript, err := GenerateNGMonitoringStartScript(tngm, tc)
	if err != nil {
		return nil, fmt.Errorf("tidb ng monitoring [%s/%s] cannot render start-script, error: %v", ns, name, err)
	}
	nmVolumeMounts := []corev1.VolumeMount{
		{Name: configVolumeName, ReadOnly: true, MountPath: ngmPodConfigVolumeMountDir}, // config
		{Name: dataVolumeName, MountPath: ngmPodDataVolumeMountDir},                     // data
	}
	baseContainers = append(baseContainers, corev1.Container{
		Name:            nmContainerName,
		Image:           tngm.NGMonitoringImage(),
		ImagePullPolicy: spec.ImagePullPolicy(),
		Command:         []string{"/bin/sh", "-c", startScript},
		Ports: []corev1.ContainerPort{
			{
				Name:          v1alpha1.NGMonitoringMemberType.String(),
				ContainerPort: ngmServicePort,
			},
		},
		VolumeMounts: nmVolumeMounts,
		Resources:    controller.ContainerResource(tngm.Spec.NGMonitoring.ResourceRequirements),
		Env: []corev1.EnvVar{
			{
				Name:  "HEADLESS_SERVICE_NAME",
				Value: headlessServiceName,
			},
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
		},
	})
	baseVolumes := []corev1.Volume{
		{
			Name: configVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cm.Name,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  ngmConfigMapConfigKey,
							Path: ngmPodConfigFilename,
						},
					},
				},
			},
		},
	}
	// base pod spec
	// TODO: place it in builder in order to reuse
	podSpec := spec.BuildPodSpec()
	podSpec.InitContainers = spec.InitContainers()
	podSpec.DNSPolicy = spec.DnsPolicy()
	podSpec.SecurityContext = spec.PodSecurityContext()
	podSpec.Containers = baseContainers
	podSpec.Volumes = baseVolumes
	basePodTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: stsLabels,
		},
		Spec: podSpec,
	}
	// base statefulset
	storageRequest, err := controller.ParseStorageRequest(tngm.Spec.NGMonitoring.Requests)
	if err != nil {
		return nil, fmt.Errorf("tidb ng monitoring [%s/%s] cannot parse storage request, error: %v", ns, name, err)
	}
	baseSts := &apps.StatefulSet{
		ObjectMeta: meta,
		Spec: apps.StatefulSetSpec{
			Selector:    stsLabels.LabelSelector(),
			ServiceName: headlessServiceName,
			Replicas:    &replicas,

			Template: basePodTemplate,
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: dataVolumeName,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						StorageClassName: tngm.Spec.NGMonitoring.StorageClassName,
						Resources:        storageRequest,
					},
				},
			},
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: spec.StatefulSetUpdateStrategy(),
			},
			PodManagementPolicy: spec.PodManagementPolicy(),
		},
	}

	builder := mngerutils.NewStatefulSetBuilder(baseSts)

	builder.PodTemplateSpecBuilder().ContainerBuilder(nmContainerName).AddEnvs(spec.Env()...)
	builder.PodTemplateSpecBuilder().ContainerBuilder(nmContainerName).AddEnvFroms(spec.EnvFrom()...)
	builder.PodTemplateSpecBuilder().AddLabels(spec.Labels())
	builder.PodTemplateSpecBuilder().AddAnnotations(spec.Annotations())
	// additional storage volumes
	storageVolMounts, additionalPVCs := util.BuildStorageVolumeAndVolumeMount(tngm.Spec.NGMonitoring.StorageVolumes, tngm.Spec.NGMonitoring.StorageClassName, v1alpha1.NGMonitoringMemberType)
	builder.PodTemplateSpecBuilder().ContainerBuilder(nmContainerName).AddVolumeMounts(storageVolMounts...)
	builder.AddVolumeClaims(additionalPVCs...)
	// additional volumes and mounts
	builder.PodTemplateSpecBuilder().ContainerBuilder(nmContainerName).AddVolumeMounts(spec.AdditionalVolumeMounts()...)
	builder.PodTemplateSpecBuilder().AddVolumes(spec.AdditionalVolumes()...)
	// additional containers
	builder.PodTemplateSpecBuilder().Get().Spec.Containers, err = member.MergePatchContainers(builder.PodTemplateSpecBuilder().Get().Spec.Containers, spec.AdditionalContainers())
	if err != nil {
		return nil, fmt.Errorf("failed to merge containers spec for TNGM of [%s/%s], error: %v", tngm.Namespace, tngm.Name, err)
	}

	// tc enable tls
	if tc.IsTLSClusterEnabled() {
		builder.PodTemplateSpecBuilder().ContainerBuilder(nmContainerName).AddVolumeMounts(corev1.VolumeMount{
			Name: "tc-client-tls", ReadOnly: true, MountPath: ngmTCClientTLSMountDir,
		})
		builder.PodTemplateSpecBuilder().AddVolumes(corev1.Volume{
			Name: "tc-client-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: TCClientTLSSecretName(name),
				},
			},
		})
	}

	return builder.Get(), nil
}

// GenerateNGMonitoringConfigMap generate ConfigMap from tidb ng monitoring
func GenerateNGMonitoringConfigMap(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) (*corev1.ConfigMap, error) {
	tcName := tc.Name
	tcNS := tc.Namespace

	var ngmConfig *config.GenericConfig
	if tngm.Spec.NGMonitoring.Config != nil {
		ngmConfig = tngm.Spec.NGMonitoring.Config.DeepCopy()
	}

	if tc.IsTLSClusterEnabled() {
		if ngmConfig == nil {
			ngmConfig = config.New(map[string]interface{}{})
		}

		ngmConfig.Set("security.ca-path", path.Join(ngmTCClientTLSMountDir, assetKey(tcName, tcNS, corev1.ServiceAccountRootCAKey)))
		ngmConfig.Set("security.cert-path", path.Join(ngmTCClientTLSMountDir, assetKey(tcName, tcNS, corev1.TLSCertKey)))
		ngmConfig.Set("security.key-path", path.Join(ngmTCClientTLSMountDir, assetKey(tcName, tcNS, corev1.TLSPrivateKeyKey)))
	}

	confText, err := ngmConfig.MarshalTOML()
	if err != nil {
		return nil, err
	}

	meta, _ := GenerateNGMonitoringMeta(tngm, NGMonitoringConfigMapName(tngm.Name))
	data := map[string]string{
		ngmConfigMapConfigKey: string(confText),
	}
	return &corev1.ConfigMap{
		ObjectMeta: meta,
		Data:       data,
	}, nil
}

// GenerateNGMonitoringMeta build ObjectMeta and Label for ng monitoring
func GenerateNGMonitoringMeta(tngm *v1alpha1.TidbNGMonitoring, name string) (metav1.ObjectMeta, label.Label) {
	instanceName := tngm.GetInstanceName()
	label := label.NewTiDBNGMonitoring().Instance(instanceName).NGMonitoring()

	objMeta := metav1.ObjectMeta{
		Name:            name,
		Namespace:       tngm.GetNamespace(),
		Labels:          label,
		OwnerReferences: []metav1.OwnerReference{controller.GetTiDBNGMonitoringOwnerRef(tngm)},
	}
	return objMeta, label
}

// GenerateNGMonitoringHeadlessService build headless service for ng monitoring
func GenerateNGMonitoringHeadlessService(tngm *v1alpha1.TidbNGMonitoring) *corev1.Service {
	meta, labels := GenerateNGMonitoringMeta(tngm, NGMonitoringHeadlessServiceName(tngm.Name))

	return &corev1.Service{
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:       "ng-monitoring",
					Port:       ngmServicePort,
					TargetPort: intstr.FromInt(ngmServicePort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector:                 labels,
			PublishNotReadyAddresses: true,
		},
	}
}

func GenerateNGMonitoringStartScript(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) (string, error) {
	model := &NGMonitoringStartScriptModel{
		TCName:            tc.Name,
		TCNamespace:       tc.Namespace,
		TCClusterDomain:   tc.Spec.ClusterDomain,
		TNGMName:          tngm.Name,
		TNGMNamespace:     tngm.Namespace,
		TNGMClusterDomain: tngm.Spec.ClusterDomain,
	}

	script, err := model.RenderStartScript()
	if err != nil {
		return "", err
	}

	return script, nil
}

type FakeNGMonitoringManager struct {
	sync func(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) error
}

func NewFakeNGMonitoringManager() *FakeNGMonitoringManager {
	return &FakeNGMonitoringManager{}
}

func (m *FakeNGMonitoringManager) MockSync(sync func(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) error) {
	m.sync = sync
}

func (m *FakeNGMonitoringManager) Sync(tngm *v1alpha1.TidbNGMonitoring, tc *v1alpha1.TidbCluster) error {
	if m.sync == nil {
		return nil
	}
	return m.sync(tngm, tc)
}
