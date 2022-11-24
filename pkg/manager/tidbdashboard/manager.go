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

package tidbdashboard

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
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
	"k8s.io/utils/pointer"
)

// Manager manages the specific kubernetes native resources for tidb dashboard.
type Manager struct {
	deps *controller.Dependencies
}

func NewManager(deps *controller.Dependencies) *Manager {
	return &Manager{
		deps: deps,
	}
}

func (m *Manager) Sync(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) error {
	var err error

	err = m.syncService(td)
	if err != nil {
		return err
	}

	err = m.syncCore(td, tc)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) syncService(td *v1alpha1.TidbDashboard) error {
	ns := td.GetNamespace()
	name := td.GetName()

	newSvc := generateTiDBDashboardNodePortService(td)
	oldSvc, err := m.deps.ServiceLister.Services(newSvc.Namespace).Get(newSvc.Name)
	svcNotFound := errors.IsNotFound(err)

	if err != nil && !svcNotFound {
		return fmt.Errorf("syncService: failed to get nodeport svc %s/%s for tidb dashboard %s/%s, error %s", newSvc.Namespace, newSvc.Name, ns, name, err)
	}

	// Create the service if not found.
	if svcNotFound {
		err := controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return m.deps.ServiceControl.CreateService(td, newSvc)
	}

	// Update the existing service.
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
		_, err = m.deps.ServiceControl.UpdateService(td, &svc)
		return err
	}

	return nil
}

func (m *Manager) syncCore(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) error {
	ns := td.GetNamespace()

	// Get the old statefulset.
	stsName := StatefulSetName(td.GetName())
	var oldSts *apps.StatefulSet
	var stsNotFound bool
	if oldStsTemp, err := m.deps.StatefulSetLister.StatefulSets(ns).Get(stsName); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get sts %s for tidb dashboard %s/%s, error: %s", stsName, ns, td.GetName(), err)
	} else {
		oldSts = oldStsTemp.DeepCopy()
		stsNotFound = errors.IsNotFound(err)
	}

	// Sync status.
	err := m.populateStatus(td, oldSts)
	if err != nil {
		klog.Errorf("failed to sync status of tidb dashboard %s/%s, error: %v", ns, td.GetName(), err)
		return err
	}

	// Generate the new statefulset.
	newSts, err := generateTiDBDashboardStatefulSet(td, tc)
	if err != nil {
		return err
	}

	// Create the new statefulset if not found.
	if stsNotFound {
		err = mngerutils.SetStatefulSetLastAppliedConfigAnnotation(newSts)
		if err != nil {
			return err
		}
		return m.deps.StatefulSetControl.CreateStatefulSet(td, newSts)
	}

	// Update the existing one.
	return mngerutils.UpdateStatefulSetWithPrecheck(m.deps, tc, "FailedUpdateNGMSTS", newSts, oldSts)
}

func (m *Manager) populateStatus(td *v1alpha1.TidbDashboard, sts *apps.StatefulSet) error {
	if sts == nil {
		return nil
	}

	td.Status.StatefulSet = &sts.Status

	upgrading, err := m.confirmStatefulSetIsUpgrading(td, sts)
	if err != nil {
		td.Status.Synced = false
		return err
	}
	if upgrading {
		td.Status.Phase = v1alpha1.UpgradePhase
	} else {
		td.Status.Phase = v1alpha1.NormalPhase
	}

	td.Status.Synced = true

	return nil
}

func (m *Manager) confirmStatefulSetIsUpgrading(td *v1alpha1.TidbDashboard, oldSts *apps.StatefulSet) (bool, error) {
	if mngerutils.StatefulSetIsUpgrading(oldSts) {
		return true, nil
	}

	selector, err := label.NewTiDBDashboard().
		Instance(td.Name).
		TiDBDashboard().
		Selector()
	if err != nil {
		return false, err
	}

	pods, err := m.deps.PodLister.Pods(td.GetNamespace()).List(selector)
	if err != nil {
		return false, fmt.Errorf("confirmStatefulSetIsUpgrading: failed to list pod for tidb dashboard %s/%s, selector %s, error: %s", td.GetNamespace(), td.GetName(), selector, err)
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

func generateTiDBDashboardStatefulSet(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) (*apps.StatefulSet, error) {
	memberName := v1alpha1.TiDBDashboardMemberType.String()
	clusterTLSEnabled := tc.IsTLSClusterEnabled()
	mysqlTLSEnabled := tc.Spec.TiDB != nil && tc.Spec.TiDB.IsTLSClientEnabled() && !tc.SkipTLSWhenConnectTiDB()

	startArgs := dashboardStartArgs(port, tc.Spec.Version, clusterTLSEnabled, mysqlTLSEnabled, tc)
	spec := td.BaseTidbDashboardSpec()
	meta, stsLabels := generateTiDBDashboardMeta(td, StatefulSetName(td.Name))

	volumeMounts := []corev1.VolumeMount{{Name: dataPVCVolumeName, MountPath: dataPVCMountPath}}
	if clusterTLSEnabled {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: clusterTLSVolumeName, MountPath: clusterTLSMountPath, ReadOnly: true})
	}
	if mysqlTLSEnabled {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: mysqlTLSVolumeName, MountPath: mysqlTLSMountPath, ReadOnly: true})
	}

	storageRequest, err := controller.ParseStorageRequest(td.Spec.Requests)
	if err != nil {
		return nil, fmt.Errorf("tidb dashboard [%s/%s] cannot parse storage request, error: %v", td.GetNamespace(), td.GetName(), err)
	}

	var baseContainers []corev1.Container
	baseContainers = append(baseContainers, corev1.Container{
		Name:            memberName,
		Image:           ensureImage(td),
		ImagePullPolicy: spec.ImagePullPolicy(),
		Args:            startArgs,
		Ports: []corev1.ContainerPort{
			{
				Name:          memberName,
				ContainerPort: port,
			},
		},
		VolumeMounts: volumeMounts,
		Resources:    controller.ContainerResource(td.Spec.ResourceRequirements),
	})

	var baseVolumes []corev1.Volume
	if clusterTLSEnabled {
		baseVolumes = append(baseVolumes, corev1.Volume{
			Name: clusterTLSVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: TCClusterClientTLSSecretName(td.Name),
				},
			},
		})
	}
	if mysqlTLSEnabled {
		baseVolumes = append(baseVolumes, corev1.Volume{
			Name: mysqlTLSVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: TCMySQLClientTLSSecretName(td.Name),
				},
			},
		})
	}

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

	baseSts := &apps.StatefulSet{
		ObjectMeta: meta,
		Spec: apps.StatefulSetSpec{
			Selector:    stsLabels.LabelSelector(),
			ServiceName: NodePortServiceName(td.GetName()),
			// Default to 1 replica.
			Replicas: pointer.Int32Ptr(1),

			Template: basePodTemplate,
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: dataPVCVolumeName,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						StorageClassName: td.Spec.StorageClassName,
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
	builder.PodTemplateSpecBuilder().ContainerBuilder(memberName).AddEnvs(spec.Env()...)
	builder.PodTemplateSpecBuilder().ContainerBuilder(memberName).AddEnvFroms(spec.EnvFrom()...)
	builder.PodTemplateSpecBuilder().AddLabels(spec.Labels())
	builder.PodTemplateSpecBuilder().AddAnnotations(spec.Annotations())
	// Additional storage volume claims.
	storageVolMounts, additionalPVCs := util.BuildStorageVolumeAndVolumeMount(td.Spec.StorageVolumes, td.Spec.StorageClassName, v1alpha1.TiDBDashboardMemberType)
	builder.PodTemplateSpecBuilder().ContainerBuilder(memberName).AddVolumeMounts(storageVolMounts...)
	builder.AddVolumeClaims(additionalPVCs...)
	// Additional volumes and mounts.
	builder.PodTemplateSpecBuilder().ContainerBuilder(memberName).AddVolumeMounts(spec.AdditionalVolumeMounts()...)
	builder.PodTemplateSpecBuilder().AddVolumes(spec.AdditionalVolumes()...)
	// Additional containers.
	builder.PodTemplateSpecBuilder().Get().Spec.Containers, err = member.MergePatchContainers(builder.PodTemplateSpecBuilder().Get().Spec.Containers, spec.AdditionalContainers())
	if err != nil {
		return nil, fmt.Errorf("failed to merge containers spec for tidb dashboard of [%s/%s], error: %v", td.Namespace, td.Name, err)
	}

	return builder.Get(), nil
}

func ensureImage(td *v1alpha1.TidbDashboard) string {
	image := td.Spec.Image
	baseImage := td.Spec.BaseImage
	// BaseImage takes higher priority.
	if baseImage != "" {
		version := td.Spec.Version
		if version == nil || *version == "" {
			image = baseImage
		} else if l := strings.LastIndex(baseImage, ":"); l >= 0 {
			// Version is specified and base image has tag suffix, override the tag.
			image = baseImage[:l+1] + *version
		} else {
			// Version is specified and base image does not have tag suffix, append the version.
			image = baseImage + ":" + *version
		}
	}
	return image
}

func generateTiDBDashboardMeta(td *v1alpha1.TidbDashboard, name string) (metav1.ObjectMeta, label.Label) {
	ls := label.NewTiDBDashboard().Instance(td.Name).TiDBDashboard()

	objMeta := metav1.ObjectMeta{
		Name:            name,
		Namespace:       td.GetNamespace(),
		Labels:          ls,
		OwnerReferences: []metav1.OwnerReference{controller.GetTiDBDashboardOwnerRef(td)},
	}
	return objMeta, ls
}

func generateTiDBDashboardNodePortService(td *v1alpha1.TidbDashboard) *corev1.Service {
	meta, labels := generateTiDBDashboardMeta(td, NodePortServiceName(td.Name))

	return &corev1.Service{
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{
				{
					Name:       "tidb-dashboard",
					Port:       port,
					TargetPort: intstr.FromInt(port),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector:                 labels,
			PublishNotReadyAddresses: true,
		},
	}
}

func dashboardStartArgs(port int, featureVersion string, clusterTLSEnable, mysqlTLSEnable bool, tc *v1alpha1.TidbCluster) []string {
	base := []string{
		"-h=0.0.0.0",
		fmt.Sprintf("-p=%d", port),
		fmt.Sprintf("--data-dir=%s", dataPVCMountPath),
		fmt.Sprintf("--temp-dir=%s", dataPVCMountPath),
		"--path-prefix=\"\"",
		fmt.Sprintf("--feature-version=%s", featureVersion),
	}

	// WARNING(@sabaping): the data key of the secret object must be "ca.crt", "tls.crt" and "tls.key" separately.
	// Highlight this in the documentation!
	if clusterTLSEnable {
		base = append(base, []string{
			"--pd=https://" + fmt.Sprintf("%s.%s:2379", controller.PDMemberName(tc.Name), tc.Namespace),
			fmt.Sprintf("--cluster-ca=%s/ca.crt", clusterTLSMountPath),
			fmt.Sprintf("--cluster-cert=%s/tls.crt", clusterTLSMountPath),
			fmt.Sprintf("--cluster-key=%s/tls.key", clusterTLSMountPath),
		}...)
	} else {
		base = append(base, "--pd=http://"+fmt.Sprintf("%s.%s:2379", controller.PDMemberName(tc.Name), tc.Namespace))
	}

	if mysqlTLSEnable {
		base = append(base, []string{
			fmt.Sprintf("--mysql-ca=%s/ca.crt", mysqlTLSMountPath),
			fmt.Sprintf("--mysql-cert=%s/tls.crt", mysqlTLSMountPath),
			fmt.Sprintf("--mysql-key=%s/tls.key", mysqlTLSMountPath),
		}...)
	}

	return base
}

type FakeManager struct {
	sync func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) error
}

func NewFakeManager() *FakeManager {
	return &FakeManager{}
}

func (m *FakeManager) MockSync(sync func(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) error) {
	m.sync = sync
}

func (m *FakeManager) Sync(td *v1alpha1.TidbDashboard, tc *v1alpha1.TidbCluster) error {
	if m.sync == nil {
		return nil
	}
	return m.sync(td, tc)
}
