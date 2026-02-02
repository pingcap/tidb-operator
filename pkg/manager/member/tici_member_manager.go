// Copyright 2026 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package member

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/manager/suspender"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"
	"github.com/pingcap/tidb-operator/pkg/manager/volumes"
	"github.com/pingcap/tidb-operator/pkg/util"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
)

// ticiMemberManager implements manager.Manager.
type ticiMemberManager struct {
	deps              *controller.Dependencies
	scaler            Scaler
	suspender         suspender.Suspender
	podVolumeModifier volumes.PodVolumeModifier
}

// NewTiCIMemberManager returns a TiCI member manager.
func NewTiCIMemberManager(deps *controller.Dependencies, scaler Scaler, spder suspender.Suspender, pvm volumes.PodVolumeModifier) manager.Manager {
	return &ticiMemberManager{
		deps:              deps,
		scaler:            scaler,
		suspender:         spder,
		podVolumeModifier: pvm,
	}
}

func (m *ticiMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	if tc.Spec.TiCI == nil {
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if tc.Spec.PD != nil && !tc.PDIsAvailable() {
		return controller.RequeueErrorf("TidbCluster: [%s/%s], TiCI is waiting for PD cluster running", ns, tcName)
	}
	if tc.Spec.TiKV != nil && !tc.TiKVIsAvailable() {
		return controller.RequeueErrorf("TidbCluster: [%s/%s], TiCI is waiting for TiKV cluster running", ns, tcName)
	}
	if tc.Spec.TiDB != nil && !tc.TiDBAllMembersReady() {
		return controller.RequeueErrorf("TidbCluster: [%s/%s], TiCI is waiting for TiDB cluster running", ns, tcName)
	}

	if tc.Spec.TiCI.Meta != nil {
		needSuspend, err := m.suspender.SuspendComponent(tc, v1alpha1.TiCIMetaMemberType)
		if err != nil {
			return fmt.Errorf("suspend %s failed: %v", v1alpha1.TiCIMetaMemberType, err)
		}
		if !needSuspend {
			if err := m.syncTiCIMeta(tc); err != nil {
				return err
			}
		} else {
			klog.Infof("component %s for cluster %s/%s is suspended, skip syncing", v1alpha1.TiCIMetaMemberType, ns, tcName)
		}
	}

	if tc.Spec.TiCI.Worker != nil {
		needSuspend, err := m.suspender.SuspendComponent(tc, v1alpha1.TiCIWorkerMemberType)
		if err != nil {
			return fmt.Errorf("suspend %s failed: %v", v1alpha1.TiCIWorkerMemberType, err)
		}
		if !needSuspend {
			if err := m.syncTiCIWorker(tc); err != nil {
				return err
			}
		} else {
			klog.Infof("component %s for cluster %s/%s is suspended, skip syncing", v1alpha1.TiCIWorkerMemberType, ns, tcName)
		}
	}

	return nil
}

func (m *ticiMemberManager) syncTiCIMeta(tc *v1alpha1.TidbCluster) error {
	if err := m.syncTiCIMetaHeadlessService(tc); err != nil {
		return err
	}
	return m.syncTiCIMetaStatefulSet(tc)
}

func (m *ticiMemberManager) syncTiCIWorker(tc *v1alpha1.TidbCluster) error {
	if err := m.syncTiCIWorkerHeadlessService(tc); err != nil {
		return err
	}
	return m.syncTiCIWorkerStatefulSet(tc)
}

func (m *ticiMemberManager) syncTiCIMetaHeadlessService(tc *v1alpha1.TidbCluster) error {
	if tc.Spec.TiCI == nil || tc.Spec.TiCI.Meta == nil {
		return nil
	}

	newSvc := getNewTiCIHeadlessService(tc, v1alpha1.TiCIMetaMemberType)
	oldSvcTmp, err := m.deps.ServiceLister.Services(tc.Namespace).Get(newSvc.Name)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncTiCIMetaHeadlessService: failed to get svc %s for cluster %s/%s, error: %s", newSvc.Name, tc.Namespace, tc.Name, err)
	}
	if errors.IsNotFound(err) {
		if err := controller.SetServiceLastAppliedConfigAnnotation(newSvc); err != nil {
			return err
		}
		return m.deps.ServiceControl.CreateService(tc, newSvc)
	}
	oldSvc := oldSvcTmp.DeepCopy()
	equal, err := controller.ServiceEqual(newSvc, oldSvc)
	if err != nil {
		return err
	}
	if equal {
		return nil
	}
	svc := *oldSvc
	svc.Spec = newSvc.Spec
	if err := controller.SetServiceLastAppliedConfigAnnotation(&svc); err != nil {
		return err
	}
	_, err = m.deps.ServiceControl.UpdateService(tc, &svc)
	return err
}

func (m *ticiMemberManager) syncTiCIWorkerHeadlessService(tc *v1alpha1.TidbCluster) error {
	if tc.Spec.TiCI == nil || tc.Spec.TiCI.Worker == nil {
		return nil
	}

	newSvc := getNewTiCIHeadlessService(tc, v1alpha1.TiCIWorkerMemberType)
	oldSvcTmp, err := m.deps.ServiceLister.Services(tc.Namespace).Get(newSvc.Name)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncTiCIWorkerHeadlessService: failed to get svc %s for cluster %s/%s, error: %s", newSvc.Name, tc.Namespace, tc.Name, err)
	}
	if errors.IsNotFound(err) {
		if err := controller.SetServiceLastAppliedConfigAnnotation(newSvc); err != nil {
			return err
		}
		return m.deps.ServiceControl.CreateService(tc, newSvc)
	}
	oldSvc := oldSvcTmp.DeepCopy()
	equal, err := controller.ServiceEqual(newSvc, oldSvc)
	if err != nil {
		return err
	}
	if equal {
		return nil
	}
	svc := *oldSvc
	svc.Spec = newSvc.Spec
	if err := controller.SetServiceLastAppliedConfigAnnotation(&svc); err != nil {
		return err
	}
	_, err = m.deps.ServiceControl.UpdateService(tc, &svc)
	return err
}

func (m *ticiMemberManager) syncTiCIMetaStatefulSet(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	oldStsTmp, err := m.deps.StatefulSetLister.StatefulSets(ns).Get(controller.TiCIMetaMemberName(tcName))
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncTiCIMetaStatefulSet: failed to get sts %s for cluster %s/%s, error: %s", controller.TiCIMetaMemberName(tcName), ns, tcName, err)
	}
	stsNotExist := errors.IsNotFound(err)
	oldSts := oldStsTmp.DeepCopy()

	if err := m.syncTiCIMetaStatus(tc, oldSts); err != nil {
		klog.Errorf("failed to sync TidbCluster: [%s/%s]'s tici meta status, error: %v", ns, tcName, err)
	}

	if tc.Spec.Paused {
		klog.Infof("TidbCluster %s/%s is paused, skip syncing tici meta statefulset", tc.GetNamespace(), tc.GetName())
		return nil
	}

	cm, err := m.syncTiCIMetaConfigMap(tc, oldSts)
	if err != nil {
		return err
	}

	newSts, err := getNewTiCIMetaStatefulSet(tc, cm)
	if err != nil {
		return err
	}

	if stsNotExist {
		err = mngerutils.SetStatefulSetLastAppliedConfigAnnotation(newSts)
		if err != nil {
			return err
		}
		return m.deps.StatefulSetControl.CreateStatefulSet(tc, newSts)
	}

	if err := m.scaler.Scale(tc, oldSts, newSts); err != nil {
		return err
	}

	if !templateEqual(newSts, oldSts) || tc.Status.TiCIMeta.Phase == v1alpha1.UpgradePhase {
		return mngerutils.UpdateStatefulSetWithPrecheck(m.deps, tc, "FailedUpdateTiCIMetaSTS", newSts, oldSts)
	}
	return nil
}

func (m *ticiMemberManager) syncTiCIWorkerStatefulSet(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	oldStsTmp, err := m.deps.StatefulSetLister.StatefulSets(ns).Get(controller.TiCIWorkerMemberName(tcName))
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncTiCIWorkerStatefulSet: failed to get sts %s for cluster %s/%s, error: %s", controller.TiCIWorkerMemberName(tcName), ns, tcName, err)
	}
	stsNotExist := errors.IsNotFound(err)
	oldSts := oldStsTmp.DeepCopy()

	if err := m.syncTiCIWorkerStatus(tc, oldSts); err != nil {
		klog.Errorf("failed to sync TidbCluster: [%s/%s]'s tici worker status, error: %v", ns, tcName, err)
	}

	if tc.Spec.Paused {
		klog.Infof("TidbCluster %s/%s is paused, skip syncing tici worker statefulset", tc.GetNamespace(), tc.GetName())
		return nil
	}

	cm, err := m.syncTiCIWorkerConfigMap(tc, oldSts)
	if err != nil {
		return err
	}

	newSts, err := getNewTiCIWorkerStatefulSet(tc, cm)
	if err != nil {
		return err
	}

	if stsNotExist {
		err = mngerutils.SetStatefulSetLastAppliedConfigAnnotation(newSts)
		if err != nil {
			return err
		}
		return m.deps.StatefulSetControl.CreateStatefulSet(tc, newSts)
	}

	if err := m.scaler.Scale(tc, oldSts, newSts); err != nil {
		return err
	}

	if !templateEqual(newSts, oldSts) || tc.Status.TiCIWorker.Phase == v1alpha1.UpgradePhase {
		return mngerutils.UpdateStatefulSetWithPrecheck(m.deps, tc, "FailedUpdateTiCIWorkerSTS", newSts, oldSts)
	}
	return nil
}

func (m *ticiMemberManager) syncTiCIMetaStatus(tc *v1alpha1.TidbCluster, sts *appsv1.StatefulSet) error {
	if sts == nil {
		tc.Status.TiCIMeta.StatefulSet = &appsv1.StatefulSetStatus{}
		tc.Status.TiCIMeta.Synced = false
		return nil
	}

	tc.Status.TiCIMeta.StatefulSet = &sts.Status
	if tc.Spec.TiCI.Meta.Replicas != *sts.Spec.Replicas {
		tc.Status.TiCIMeta.Phase = v1alpha1.ScalePhase
	} else if sts.Status.CurrentRevision != sts.Status.UpdateRevision {
		tc.Status.TiCIMeta.Phase = v1alpha1.UpgradePhase
	} else {
		tc.Status.TiCIMeta.Phase = v1alpha1.NormalPhase
	}
	if tc.Spec.TiCI.Meta.Replicas == sts.Status.ReadyReplicas {
		tc.Status.TiCIMeta.Synced = true
	} else {
		tc.Status.TiCIMeta.Synced = false
	}

	return volumes.SyncVolumeStatus(m.podVolumeModifier, m.deps.PodLister, tc, v1alpha1.TiCIMetaMemberType)
}

func (m *ticiMemberManager) syncTiCIWorkerStatus(tc *v1alpha1.TidbCluster, sts *appsv1.StatefulSet) error {
	if sts == nil {
		tc.Status.TiCIWorker.StatefulSet = &appsv1.StatefulSetStatus{}
		tc.Status.TiCIWorker.Synced = false
		return nil
	}

	tc.Status.TiCIWorker.StatefulSet = &sts.Status
	if tc.Spec.TiCI.Worker.Replicas != *sts.Spec.Replicas {
		tc.Status.TiCIWorker.Phase = v1alpha1.ScalePhase
	} else if sts.Status.CurrentRevision != sts.Status.UpdateRevision {
		tc.Status.TiCIWorker.Phase = v1alpha1.UpgradePhase
	} else {
		tc.Status.TiCIWorker.Phase = v1alpha1.NormalPhase
	}
	if tc.Spec.TiCI.Worker.Replicas == sts.Status.ReadyReplicas {
		tc.Status.TiCIWorker.Synced = true
	} else {
		tc.Status.TiCIWorker.Synced = false
	}

	return volumes.SyncVolumeStatus(m.podVolumeModifier, m.deps.PodLister, tc, v1alpha1.TiCIWorkerMemberType)
}

func (m *ticiMemberManager) syncTiCIMetaConfigMap(tc *v1alpha1.TidbCluster, set *appsv1.StatefulSet) (*corev1.ConfigMap, error) {
	newCm, err := getTiCIMetaConfigMap(tc)
	if err != nil {
		return nil, err
	}

	var inUseName string
	if set != nil {
		inUseName = mngerutils.FindConfigMapVolume(&set.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, controller.TiCIMetaMemberName(tc.Name))
		})
	}

	err = mngerutils.UpdateConfigMapIfNeed(m.deps.ConfigMapLister, tc.BaseTiCIMetaSpec().ConfigUpdateStrategy(), inUseName, newCm)
	if err != nil {
		return nil, err
	}
	return m.deps.TypedControl.CreateOrUpdateConfigMap(tc, newCm)
}

func (m *ticiMemberManager) syncTiCIWorkerConfigMap(tc *v1alpha1.TidbCluster, set *appsv1.StatefulSet) (*corev1.ConfigMap, error) {
	newCm, err := getTiCIWorkerConfigMap(tc)
	if err != nil {
		return nil, err
	}

	var inUseName string
	if set != nil {
		inUseName = mngerutils.FindConfigMapVolume(&set.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, controller.TiCIWorkerMemberName(tc.Name))
		})
	}

	err = mngerutils.UpdateConfigMapIfNeed(m.deps.ConfigMapLister, tc.BaseTiCIWorkerSpec().ConfigUpdateStrategy(), inUseName, newCm)
	if err != nil {
		return nil, err
	}
	return m.deps.TypedControl.CreateOrUpdateConfigMap(tc, newCm)
}

func getTiCIMetaConfigMap(tc *v1alpha1.TidbCluster) (*corev1.ConfigMap, error) {
	configText, err := buildTiCIMetaConfig(tc)
	if err != nil {
		return nil, err
	}
	name := controller.TiCIMetaMemberName(tc.Name)
	instanceName := tc.GetInstanceName()
	labels := label.New().Instance(instanceName).TiCIMeta().Labels()

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       tc.Namespace,
			Labels:          labels,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Data: map[string]string{"config-file": configText},
	}
	return cm, nil
}

func getTiCIWorkerConfigMap(tc *v1alpha1.TidbCluster) (*corev1.ConfigMap, error) {
	configText, err := buildTiCIWorkerConfig(tc)
	if err != nil {
		return nil, err
	}
	name := controller.TiCIWorkerMemberName(tc.Name)
	instanceName := tc.GetInstanceName()
	labels := label.New().Instance(instanceName).TiCIWorker().Labels()

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       tc.Namespace,
			Labels:          labels,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Data: map[string]string{"config-file": configText},
	}
	return cm, nil
}

func getNewTiCIMetaStatefulSet(tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap) (*appsv1.StatefulSet, error) {
	if tc.Spec.TiCI == nil || tc.Spec.TiCI.Meta == nil {
		return nil, nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()
	baseSpec := tc.BaseTiCIMetaSpec()
	spec := tc.Spec.TiCI.Meta
	stsLabels := labelTiCIMeta(tc)
	stsName := controller.TiCIMetaMemberName(tcName)
	podLabels := util.CombineStringMap(stsLabels, baseSpec.Labels())
	podAnnotations := util.CombineStringMap(baseSpec.Annotations(), controller.AnnProm(v1alpha1.DefaultTiCIMetaStatusPort, "/metrics"))
	stsAnnotations := getStsAnnotations(tc.Annotations, label.TiCIMetaLabelVal)
	headlessSvcName := controller.TiCIMetaPeerMemberName(tcName)

	volMounts := []corev1.VolumeMount{}
	vols := []corev1.Volume{}
	storageVolMounts, additionalPVCs := util.BuildStorageVolumeAndVolumeMount(spec.StorageVolumes, spec.StorageClassName, spec.VolumeAttributesClassName, v1alpha1.TiCIMetaMemberType)
	volMounts = append(volMounts, storageVolMounts...)
	volMounts = append(volMounts, spec.AdditionalVolumeMounts...)

	configMountPath := "/etc/tici"
	if cm != nil {
		volMounts = append(volMounts, corev1.VolumeMount{Name: "config", MountPath: configMountPath})
		vols = append(vols, corev1.Volume{Name: "config", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: cm.Name},
			Items:                []corev1.KeyToPath{{Key: "config-file", Path: "tici.toml"}},
		}}})
	}

	args := renderTiCIStartArgs(tc, v1alpha1.TiCIMetaMemberType, headlessSvcName)

	envs := []corev1.EnvVar{
		{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
	}

	container := corev1.Container{
		Name:            v1alpha1.TiCIMetaMemberType.String(),
		Image:           tc.TiCIMetaImage(),
		ImagePullPolicy: baseSpec.ImagePullPolicy(),
		Command:         []string{"/bin/sh", "-c"},
		Args:            []string{args},
		Ports:           []corev1.ContainerPort{{ContainerPort: v1alpha1.DefaultTiCIMetaPort}, {ContainerPort: v1alpha1.DefaultTiCIMetaStatusPort}},
		Resources:       controller.ContainerResource(spec.ResourceRequirements),
		VolumeMounts:    volMounts,
		Env:             util.AppendEnv(envs, baseSpec.Env()),
		EnvFrom:         baseSpec.EnvFrom(),
	}

	podSpec := baseSpec.BuildPodSpec()
	var err error
	podSpec.Containers, err = MergePatchContainers([]corev1.Container{container}, baseSpec.AdditionalContainers())
	if err != nil {
		return nil, fmt.Errorf("failed to merge containers spec for TiCI meta of [%s/%s], error: %v", ns, tcName, err)
	}
	podSpec.Volumes = append(vols, baseSpec.AdditionalVolumes()...)
	podSpec.ServiceAccountName = spec.ServiceAccount
	podSpec.InitContainers = append(podSpec.InitContainers, baseSpec.InitContainers()...)
	if podSpec.ServiceAccountName == "" {
		podSpec.ServiceAccountName = tc.Spec.ServiceAccount
	}

	deleteSlotsNumber, err := util.GetDeleteSlotsNumber(stsAnnotations)
	if err != nil {
		return nil, fmt.Errorf("get delete slots number of statefulset %s/%s failed, err:%v", ns, stsName, err)
	}

	updateStrategy := appsv1.StatefulSetUpdateStrategy{}
	if baseSpec.StatefulSetUpdateStrategy() == appsv1.OnDeleteStatefulSetStrategyType {
		updateStrategy.Type = appsv1.OnDeleteStatefulSetStrategyType
	} else {
		updateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
		updateStrategy.RollingUpdate = &appsv1.RollingUpdateStatefulSetStrategy{Partition: pointer.Int32Ptr(spec.Replicas + deleteSlotsNumber)}
	}

	set := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            stsName,
			Namespace:       ns,
			Labels:          stsLabels,
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:             pointer.Int32Ptr(spec.Replicas),
			ServiceName:          headlessSvcName,
			Selector:             &metav1.LabelSelector{MatchLabels: stsLabels},
			Template:             corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: podLabels, Annotations: podAnnotations}, Spec: podSpec},
			PodManagementPolicy:  baseSpec.PodManagementPolicy(),
			UpdateStrategy:       updateStrategy,
			VolumeClaimTemplates: additionalPVCs,
		},
	}

	return set, nil
}

func getNewTiCIWorkerStatefulSet(tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap) (*appsv1.StatefulSet, error) {
	if tc.Spec.TiCI == nil || tc.Spec.TiCI.Worker == nil {
		return nil, nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()
	baseSpec := tc.BaseTiCIWorkerSpec()
	spec := tc.Spec.TiCI.Worker
	stsLabels := labelTiCIWorker(tc)
	stsName := controller.TiCIWorkerMemberName(tcName)
	podLabels := util.CombineStringMap(stsLabels, baseSpec.Labels())
	podAnnotations := util.CombineStringMap(baseSpec.Annotations(), controller.AnnProm(v1alpha1.DefaultTiCIWorkerStatusPort, "/metrics"))
	stsAnnotations := getStsAnnotations(tc.Annotations, label.TiCIWorkerLabelVal)
	headlessSvcName := controller.TiCIWorkerPeerMemberName(tcName)

	volMounts := []corev1.VolumeMount{}
	vols := []corev1.Volume{}
	storageVolMounts, additionalPVCs := util.BuildStorageVolumeAndVolumeMount(spec.StorageVolumes, spec.StorageClassName, spec.VolumeAttributesClassName, v1alpha1.TiCIWorkerMemberType)
	volMounts = append(volMounts, storageVolMounts...)
	volMounts = append(volMounts, spec.AdditionalVolumeMounts...)

	configMountPath := "/etc/tici"
	if cm != nil {
		volMounts = append(volMounts, corev1.VolumeMount{Name: "config", MountPath: configMountPath})
		vols = append(vols, corev1.Volume{Name: "config", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: cm.Name},
			Items:                []corev1.KeyToPath{{Key: "config-file", Path: "tici.toml"}},
		}}})
	}

	args := renderTiCIStartArgs(tc, v1alpha1.TiCIWorkerMemberType, headlessSvcName)

	envs := []corev1.EnvVar{
		{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
	}

	container := corev1.Container{
		Name:            v1alpha1.TiCIWorkerMemberType.String(),
		Image:           tc.TiCIWorkerImage(),
		ImagePullPolicy: baseSpec.ImagePullPolicy(),
		Command:         []string{"/bin/sh", "-c"},
		Args:            []string{args},
		Ports:           []corev1.ContainerPort{{ContainerPort: v1alpha1.DefaultTiCIWorkerPort}, {ContainerPort: v1alpha1.DefaultTiCIWorkerStatusPort}},
		Resources:       controller.ContainerResource(spec.ResourceRequirements),
		VolumeMounts:    volMounts,
		Env:             util.AppendEnv(envs, baseSpec.Env()),
		EnvFrom:         baseSpec.EnvFrom(),
	}

	podSpec := baseSpec.BuildPodSpec()
	var err error
	podSpec.Containers, err = MergePatchContainers([]corev1.Container{container}, baseSpec.AdditionalContainers())
	if err != nil {
		return nil, fmt.Errorf("failed to merge containers spec for TiCI worker of [%s/%s], error: %v", ns, tcName, err)
	}
	podSpec.Volumes = append(vols, baseSpec.AdditionalVolumes()...)
	podSpec.ServiceAccountName = spec.ServiceAccount
	podSpec.InitContainers = append(podSpec.InitContainers, baseSpec.InitContainers()...)
	if podSpec.ServiceAccountName == "" {
		podSpec.ServiceAccountName = tc.Spec.ServiceAccount
	}

	deleteSlotsNumber, err := util.GetDeleteSlotsNumber(stsAnnotations)
	if err != nil {
		return nil, fmt.Errorf("get delete slots number of statefulset %s/%s failed, err:%v", ns, stsName, err)
	}

	updateStrategy := appsv1.StatefulSetUpdateStrategy{}
	if baseSpec.StatefulSetUpdateStrategy() == appsv1.OnDeleteStatefulSetStrategyType {
		updateStrategy.Type = appsv1.OnDeleteStatefulSetStrategyType
	} else {
		updateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
		updateStrategy.RollingUpdate = &appsv1.RollingUpdateStatefulSetStrategy{Partition: pointer.Int32Ptr(spec.Replicas + deleteSlotsNumber)}
	}

	set := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            stsName,
			Namespace:       ns,
			Labels:          stsLabels,
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:             pointer.Int32Ptr(spec.Replicas),
			ServiceName:          headlessSvcName,
			Selector:             &metav1.LabelSelector{MatchLabels: stsLabels},
			Template:             corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: podLabels, Annotations: podAnnotations}, Spec: podSpec},
			PodManagementPolicy:  baseSpec.PodManagementPolicy(),
			UpdateStrategy:       updateStrategy,
			VolumeClaimTemplates: additionalPVCs,
		},
	}

	return set, nil
}

func getNewTiCIHeadlessService(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType) *corev1.Service {
	name := ""
	labels := map[string]string{}
	port := int32(0)
	statusPort := int32(0)
	instanceName := tc.GetInstanceName()
	if memberType == v1alpha1.TiCIMetaMemberType {
		name = controller.TiCIMetaPeerMemberName(tc.Name)
		labels = label.New().Instance(instanceName).TiCIMeta().Labels()
		port = v1alpha1.DefaultTiCIMetaPort
		statusPort = v1alpha1.DefaultTiCIMetaStatusPort
	} else {
		name = controller.TiCIWorkerPeerMemberName(tc.Name)
		labels = label.New().Instance(instanceName).TiCIWorker().Labels()
		port = v1alpha1.DefaultTiCIWorkerPort
		statusPort = v1alpha1.DefaultTiCIWorkerStatusPort
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       tc.Namespace,
			Labels:          labels,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP:                "None",
			PublishNotReadyAddresses: true,
			Ports: []corev1.ServicePort{
				{Name: "peer", Port: port, TargetPort: intstr.FromInt(int(port))},
				{Name: "status", Port: statusPort, TargetPort: intstr.FromInt(int(statusPort))},
			},
			Selector: labels,
		},
	}
}

func labelTiCIMeta(tc *v1alpha1.TidbCluster) label.Label {
	instanceName := tc.GetInstanceName()
	return label.New().Instance(instanceName).TiCIMeta()
}

func labelTiCIWorker(tc *v1alpha1.TidbCluster) label.Label {
	instanceName := tc.GetInstanceName()
	return label.New().Instance(instanceName).TiCIWorker()
}

func renderTiCIStartArgs(tc *v1alpha1.TidbCluster, memberType v1alpha1.MemberType, headlessSvcName string) string {
	ns := tc.GetNamespace()
	pdAddr := fmt.Sprintf("%s:%d", controller.PDMemberName(tc.Name), v1alpha1.DefaultPDClientPort)
	advertiseHost := fmt.Sprintf("${POD_NAME}.%s.%s.svc%s", headlessSvcName, ns, controller.FormatClusterDomain(tc.Spec.ClusterDomain))

	args := []string{"exec ${TICI_BIN:-/tici-server}"}
	if memberType == v1alpha1.TiCIMetaMemberType {
		args = append(args, "meta")
		args = append(args, fmt.Sprintf("--config=%s", "/etc/tici/tici.toml"))
		args = append(args, "--host=0.0.0.0")
		args = append(args, fmt.Sprintf("--port=%d", v1alpha1.DefaultTiCIMetaPort))
		args = append(args, fmt.Sprintf("--status-port=%d", v1alpha1.DefaultTiCIMetaStatusPort))
		args = append(args, fmt.Sprintf("--advertise-host=%s", advertiseHost))
		args = append(args, fmt.Sprintf("--pd-addr=%s", pdAddr))
	} else {
		args = append(args, "worker")
		args = append(args, fmt.Sprintf("--config=%s", "/etc/tici/tici.toml"))
		args = append(args, "--host=0.0.0.0")
		args = append(args, fmt.Sprintf("--port=%d", v1alpha1.DefaultTiCIWorkerPort))
		args = append(args, fmt.Sprintf("--status-port=%d", v1alpha1.DefaultTiCIWorkerStatusPort))
		args = append(args, fmt.Sprintf("--advertise-host=%s", advertiseHost))
		args = append(args, fmt.Sprintf("--pd-addr=%s", pdAddr))
	}

	return strings.Join(args, " ")
}

func buildTiCIMetaConfig(tc *v1alpha1.TidbCluster) (string, error) {
	s3, err := buildTiCIS3Config(tc)
	if err != nil {
		return "", err
	}
	if tc.Spec.TiDB == nil {
		return "", fmt.Errorf("TiCI requires TiDB to be enabled")
	}
	tidbHost := controller.TiDBMemberName(tc.Name)
	tidbPort := tc.Spec.TiDB.GetServicePort()
	pdAddr := fmt.Sprintf("%s:%d", controller.PDMemberName(tc.Name), v1alpha1.DefaultPDClientPort)

	return fmt.Sprintf(`[tidb-server]
dsns = ["mysql://root@%s:%d"]

[server]
pd-addr = "%s"

[s3]
endpoint = "%s"
region = "%s"
access-key = "%s"
secret-key = "%s"
use-path-style = %t
bucket = "%s"

[shard]
max-size = "128MB"
`, tidbHost, tidbPort, pdAddr, s3.Endpoint, s3.Region, s3.AccessKey, s3.SecretKey, s3.UsePathStyle, s3.Bucket), nil
}

func buildTiCIWorkerConfig(tc *v1alpha1.TidbCluster) (string, error) {
	s3, err := buildTiCIS3Config(tc)
	if err != nil {
		return "", err
	}
	pdAddr := fmt.Sprintf("%s:%d", controller.PDMemberName(tc.Name), v1alpha1.DefaultPDClientPort)

	return fmt.Sprintf(`[server]
pd-addr = "%s"

[s3]
endpoint = "%s"
region = "%s"
access-key = "%s"
secret-key = "%s"
use-path-style = %t
bucket = "%s"
`, pdAddr, s3.Endpoint, s3.Region, s3.AccessKey, s3.SecretKey, s3.UsePathStyle, s3.Bucket), nil
}

type tiCIS3Config struct {
	Endpoint     string
	Region       string
	AccessKey    string
	SecretKey    string
	Bucket       string
	Prefix       string
	UsePathStyle bool
}

func buildTiCIS3Config(tc *v1alpha1.TidbCluster) (*tiCIS3Config, error) {
	if tc.Spec.TiCI == nil || tc.Spec.TiCI.S3 == nil {
		return nil, fmt.Errorf("TiCI S3 config is required")
	}
	s3 := tc.Spec.TiCI.S3
	if s3.Endpoint == "" || s3.Bucket == "" {
		return nil, fmt.Errorf("TiCI S3 endpoint and bucket are required")
	}
	usePathStyle := true
	if s3.UsePathStyle != nil {
		usePathStyle = *s3.UsePathStyle
	}
	region := s3.Region
	if region == "" {
		region = "us-east-1"
	}
	prefix := s3.Prefix
	if prefix == "" {
		prefix = "tici_default_prefix"
	}
	return &tiCIS3Config{
		Endpoint:     s3.Endpoint,
		Region:       region,
		AccessKey:    s3.AccessKey,
		SecretKey:    s3.SecretKey,
		Bucket:       s3.Bucket,
		Prefix:       prefix,
		UsePathStyle: usePathStyle,
	}, nil
}

// FakeTiCIMemberManager is a fake implementation of Manager.
type FakeTiCIMemberManager struct {
	err error
}

// NewFakeTiCIMemberManager returns a fake TiCI member manager.
func NewFakeTiCIMemberManager() *FakeTiCIMemberManager {
	return &FakeTiCIMemberManager{}
}

func (m *FakeTiCIMemberManager) SetSyncError(err error) {
	m.err = err
}

func (m *FakeTiCIMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	return m.err
}

// Ensure fake manager implements Manager.
var _ manager.Manager = &FakeTiCIMemberManager{}
