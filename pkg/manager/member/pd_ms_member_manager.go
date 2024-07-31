// Copyright 2023 PingCAP, Inc.
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
	"fmt"
	"path"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/manager/member/startscript"
	"github.com/pingcap/tidb-operator/pkg/manager/suspender"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"
	"github.com/pingcap/tidb-operator/pkg/manager/volumes"
	"github.com/pingcap/tidb-operator/pkg/metrics"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
)

type pdMSMemberManager struct {
	deps              *controller.Dependencies
	scaler            Scaler
	upgrader          Upgrader
	suspender         suspender.Suspender
	podVolumeModifier volumes.PodVolumeModifier
}

// NewPDMSMemberManager returns a *pdMSMemberManager
func NewPDMSMemberManager(dependencies *controller.Dependencies, pdMSScaler Scaler, pdMSUpgrader Upgrader, spder suspender.Suspender, pvm volumes.PodVolumeModifier) manager.Manager {
	return &pdMSMemberManager{
		deps:              dependencies,
		scaler:            pdMSScaler,
		upgrader:          pdMSUpgrader,
		suspender:         spder,
		podVolumeModifier: pvm,
	}
}

// Sync for all PD Micro Service components.
func (m *pdMSMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	// Need to start PD API
	if tc.Spec.PDMS != nil && tc.Spec.PD == nil {
		klog.Infof("PD Micro Service is enabled, but PD is not enabled, skip syncing PD Micro Service")
		return nil
	}
	// remove all micro service components if PDMS is not enabled
	// PDMS need to be enabled when PD.Mode is ms && PDMS is not nil
	if tc.Spec.PDMS == nil || (tc.Spec.PD != nil && tc.Spec.PD.Mode != "ms") {
		for _, comp := range tc.Status.PDMS {
			ns := tc.GetNamespace()
			tcName := tc.GetName()
			curService := comp.Name

			oldPDMSSetTmp, err := m.deps.StatefulSetLister.StatefulSets(ns).Get(controller.PDMSMemberName(tcName, curService))
			if err != nil {
				if errors.IsNotFound(err) {
					continue
				}
				return fmt.Errorf("syncPDMSStatefulSet: fail to get sts %s PDMS component %s for cluster [%s/%s], error: %s",
					controller.PDMSMemberName(tcName, curService), curService, ns, tcName, err)
			}

			oldPDMSSet := oldPDMSSetTmp.DeepCopy()
			newPDMSSet := oldPDMSSetTmp.DeepCopy()
			if oldPDMSSet.Status.Replicas == 0 {
				continue
			}
			tc.Status.PDMS[curService].Synced = true
			*newPDMSSet.Spec.Replicas = 0
			if err := m.scaler.Scale(tc, oldPDMSSet, newPDMSSet); err != nil {
				return err
			}
			mngerutils.UpdateStatefulSetWithPrecheck(m.deps, tc, "FailedUpdatePDMSSTS", newPDMSSet, oldPDMSSet)
		}
		return nil
	}

	// init PD Micro Service status
	if tc.Status.PDMS == nil {
		tc.Status.PDMS = make(map[string]*v1alpha1.PDMSStatus)
	}

	for _, comp := range tc.Spec.PDMS {
		curSpec := comp
		if tc.Status.PDMS[curSpec.Name] == nil {
			tc.Status.PDMS[curSpec.Name] = &v1alpha1.PDMSStatus{Name: curSpec.Name}
		}
		if err := m.syncSingleService(tc, curSpec); err != nil {
			metrics.ClusterUpdateErrors.WithLabelValues(tc.GetNamespace(), tc.GetName(), curSpec.Name).Inc()
			klog.Errorf("syncSingleService failed, error: %v, component: %s for cluster %s/%s", err, curSpec.Name, tc.GetNamespace(), tc.GetName())
			return err
		}
	}

	return nil
}

// syncSingleService for single PD Micro Service components.
func (m *pdMSMemberManager) syncSingleService(tc *v1alpha1.TidbCluster, curSpec *v1alpha1.PDMSSpec) error {
	curService := curSpec.Name
	// Skip sync if PD Micro Service is suspended
	componentMemberType := v1alpha1.PDMSMemberType(curService)
	needSuspend, err := m.suspender.SuspendComponent(tc, componentMemberType)
	if err != nil {
		return fmt.Errorf("PDMS component %s for cluster [%s/%s] suspend failed: %v", curService, tc.GetNamespace(), tc.GetName(), err)
	}
	if needSuspend {
		klog.Infof("PDMS component %s for cluster [%s/%s] is suspended, skip syncing", curService, tc.GetNamespace(), tc.GetName())
		return nil
	}

	// Sync PD Micro Service
	if err := m.syncPDMSService(tc, curSpec); err != nil {
		return err
	}

	// Sync PD Micro Service Headless Service
	if err := m.syncPDMSHeadlessService(tc, curSpec); err != nil {
		return err
	}

	// Sync PD Micro Service StatefulSet
	return m.syncPDMSStatefulSet(tc, curSpec)
}

func (m *pdMSMemberManager) syncPDMSService(tc *v1alpha1.TidbCluster, curSpec *v1alpha1.PDMSSpec) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	curService := curSpec.Name
	if tc.Spec.Paused {
		klog.Infof("tidb cluster %s/%s is paused, skip syncing for pdMS component %s", ns, tcName, curService)
		return nil
	}

	newSvc := m.getNewPDMSService(tc, curSpec)
	oldSvcTmp, err := m.deps.ServiceLister.Services(ns).Get(controller.PDMSMemberName(tcName, curService))
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return m.deps.ServiceControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return fmt.Errorf("syncpdMSServiceForTidbCluster: failed to get svc %s PDMS component %s for cluster [%s/%s], error: %s",
			controller.PDMSMemberName(tcName, curService), curService, ns, tcName, err)
	}

	oldSvc := oldSvcTmp.DeepCopy()

	_, err = m.deps.ServiceControl.SyncComponentService(
		tc,
		newSvc,
		oldSvc,
		true)

	if err != nil {
		return err
	}

	return nil
}

func (m *pdMSMemberManager) syncPDMSHeadlessService(tc *v1alpha1.TidbCluster, curSpec *v1alpha1.PDMSSpec) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	curService := curSpec.Name
	if tc.Spec.Paused {
		klog.Infof("tidb cluster %s/%s is paused, skip syncing for %s headless service", ns, tcName, curService)
		return nil
	}

	newSvc := getNewPDMSHeadlessService(tc, curService)
	oldSvcTmp, err := m.deps.ServiceLister.Services(ns).Get(controller.PDMSPeerMemberName(tcName, curService))
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return m.deps.ServiceControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return fmt.Errorf("syncpdMSHeadlessService: failed to get svc %s PDMS component %s for cluster [%s/%s], error: %s",
			controller.PDMSPeerMemberName(tcName, curService), curService, ns, tcName, err)
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

func getNewPDMSHeadlessService(tc *v1alpha1.TidbCluster, componentName string) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	svcName := controller.PDMSPeerMemberName(tcName, componentName)
	instanceName := tc.GetInstanceName()
	pdMSSelector := label.New().Instance(instanceName).PDMS(componentName)
	pdMSLabels := pdMSSelector.Copy().UsedByPeer().Labels()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          pdMSLabels,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:       fmt.Sprintf("tcp-peer-%d", v1alpha1.DefaultPDPeerPort),
					Port:       v1alpha1.DefaultPDPeerPort,
					TargetPort: intstr.FromInt(int(v1alpha1.DefaultPDPeerPort)),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       fmt.Sprintf("tcp-peer-%d", v1alpha1.DefaultPDClientPort),
					Port:       v1alpha1.DefaultPDClientPort,
					TargetPort: intstr.FromInt(int(v1alpha1.DefaultPDClientPort)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector:                 pdMSSelector.Labels(),
			PublishNotReadyAddresses: true,
		},
	}

	if tc.Spec.PreferIPv6 {
		SetServiceWhenPreferIPv6(svc)
	}

	return svc
}

func (m *pdMSMemberManager) syncPDMSStatefulSet(tc *v1alpha1.TidbCluster, curSpec *v1alpha1.PDMSSpec) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	curService := curSpec.Name

	oldPDMSSetTmp, err := m.deps.StatefulSetLister.StatefulSets(ns).Get(controller.PDMSMemberName(tcName, curService))
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncPDMSStatefulSet: fail to get sts %s PDMS component %s for cluster [%s/%s], error: %s",
			controller.PDMSMemberName(tcName, curService), curService, ns, tcName, err)
	}

	setNotExist := errors.IsNotFound(err)
	oldPDMSSet := oldPDMSSetTmp.DeepCopy()

	if err := m.syncStatus(tc, oldPDMSSet, curSpec); err != nil {
		klog.Errorf("failed to sync PDMS component %s for cluster [%s/%s]'s status, error: %v", curService, ns, tcName, err)
	}

	if tc.Spec.Paused {
		klog.Infof("tidb cluster %s/%s is paused, skip syncing for PDMS component %s statefulset", tc.GetNamespace(), tc.GetName(), curService)
		return nil
	}

	cm, err := m.syncPDMSConfigMap(tc, oldPDMSSet, curSpec)
	if err != nil {
		return err
	}

	newPDMSSet, err := m.getNewPDMSStatefulSet(tc, cm, curSpec)
	if err != nil {
		return err
	}
	if setNotExist {
		err = mngerutils.SetStatefulSetLastAppliedConfigAnnotation(newPDMSSet)
		if err != nil {
			return err
		}
		if err := m.deps.StatefulSetControl.CreateStatefulSet(tc, newPDMSSet); err != nil {
			return err
		}
		if tc.Status.PDMS[curService] == nil {
			tc.Status.PDMS[curService] = &v1alpha1.PDMSStatus{Name: curService}
		}
		tc.Status.PDMS[curService].StatefulSet = &apps.StatefulSetStatus{}

		return controller.RequeueErrorf("waiting for PDMS component %s for cluster [%s/%s] running", curService, ns, tcName)
	}

	// Scaling takes precedence over upgrading because:
	// - if a pdMS fails in the upgrading, users may want to delete it or add
	//   new replicas
	// - it's ok to scale in the middle of upgrading (in statefulset controller
	//   scaling takes precedence over upgrading too)
	if err := m.scaler.Scale(tc, oldPDMSSet, newPDMSSet); err != nil {
		return err
	}

	//TODO: Add FailOver logic

	if !templateEqual(newPDMSSet, oldPDMSSet) || tc.Status.PDMS[curService].Phase == v1alpha1.UpgradePhase {
		if err := m.upgrader.Upgrade(tc, oldPDMSSet, newPDMSSet); err != nil {
			return err
		}
	}

	return mngerutils.UpdateStatefulSetWithPrecheck(m.deps, tc, "FailedUpdatePDMSSTS", newPDMSSet, oldPDMSSet)
}

func (m *pdMSMemberManager) syncStatus(tc *v1alpha1.TidbCluster, sts *apps.StatefulSet, curSpec *v1alpha1.PDMSSpec) error {
	if sts == nil {
		// skip if not created yet
		return nil
	}

	curService := curSpec.Name
	tc.Status.PDMS[curService].Name = curService
	tc.Status.PDMS[curService].StatefulSet = &sts.Status
	upgrading, err := m.pdMSStatefulSetIsUpgrading(sts, tc, curSpec)
	if err != nil {
		return err
	}

	// Scaling takes precedence over upgrading.
	if tc.PDMSStsDesiredReplicas(curService) != *sts.Spec.Replicas {
		tc.Status.PDMS[curService].Phase = v1alpha1.ScalePhase
	} else if upgrading {
		tc.Status.PDMS[curService].Phase = v1alpha1.UpgradePhase
	} else {
		tc.Status.PDMS[curService].Phase = v1alpha1.NormalPhase
	}

	pdClient := controller.GetPDClient(m.deps.PDControl, tc)
	// pdMS member
	members, err := pdClient.GetMSMembers(curService)
	if err != nil {
		return err
	}
	tc.Status.PDMS[curService].Members = members
	tc.Status.PDMS[curService].Synced = true

	// sync volumes
	err = volumes.SyncVolumeStatus(m.podVolumeModifier, m.deps.PodLister, tc, v1alpha1.PDMSMemberType(curService))
	if err != nil {
		return fmt.Errorf("failed to sync volume status for pd: %v", err)
	}

	return nil
}

// syncPDMSConfigMap syncs the configmap of PDMS
func (m *pdMSMemberManager) syncPDMSConfigMap(tc *v1alpha1.TidbCluster, set *apps.StatefulSet, curSpec *v1alpha1.PDMSSpec) (*corev1.ConfigMap, error) {
	newCm, err := m.getPDMSConfigMap(tc, curSpec)
	if err != nil {
		return nil, err
	}

	var inUseName string
	if set != nil {
		inUseName = mngerutils.FindConfigMapVolume(&set.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, controller.PDMSMemberName(tc.Name, curSpec.Name))
		})
	}

	err = mngerutils.UpdateConfigMapIfNeed(m.deps.ConfigMapLister, tc.BasePDMSSpec(curSpec).ConfigUpdateStrategy(), inUseName, newCm)
	if err != nil {
		return nil, err
	}
	return m.deps.TypedControl.CreateOrUpdateConfigMap(tc, newCm)
}

func (m *pdMSMemberManager) getNewPDMSService(tc *v1alpha1.TidbCluster, curSpec *v1alpha1.PDMSSpec) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	curService := curSpec.Name
	svcName := controller.PDMSMemberName(tcName, curService)
	instanceName := tc.GetInstanceName()
	pdMSSelector := label.New().Instance(instanceName).PDMS(curService)
	pdMSLabels := pdMSSelector.Copy().UsedByEndUser().Labels()

	pdMSService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          pdMSLabels,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: corev1.ServiceSpec{
			Type: controller.GetServiceType(tc.Spec.Services, curService),
			Ports: []corev1.ServicePort{
				{
					Name:       "client",
					Port:       v1alpha1.DefaultPDClientPort,
					TargetPort: intstr.FromInt(int(v1alpha1.DefaultPDClientPort)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: pdMSSelector.Labels(),
		},
	}

	// override fields with user-defined ServiceSpec
	svcSpec := curSpec.Service
	if svcSpec != nil {
		if svcSpec.Type != "" {
			pdMSService.Spec.Type = svcSpec.Type
		}
		pdMSService.ObjectMeta.Annotations = util.CopyStringMap(svcSpec.Annotations)
		pdMSService.ObjectMeta.Labels = util.CombineStringMap(pdMSService.ObjectMeta.Labels, svcSpec.Labels)
		if svcSpec.LoadBalancerIP != nil {
			pdMSService.Spec.LoadBalancerIP = *svcSpec.LoadBalancerIP
		}
		if svcSpec.ClusterIP != nil {
			pdMSService.Spec.ClusterIP = *svcSpec.ClusterIP
		}
		if svcSpec.PortName != nil {
			pdMSService.Spec.Ports[0].Name = *svcSpec.PortName
		}
	}

	if tc.Spec.PreferIPv6 {
		SetServiceWhenPreferIPv6(pdMSService)
	}

	return pdMSService
}

func (m *pdMSMemberManager) pdMSStatefulSetIsUpgrading(set *apps.StatefulSet, tc *v1alpha1.TidbCluster, curSpec *v1alpha1.PDMSSpec) (bool, error) {
	if mngerutils.StatefulSetIsUpgrading(set) {
		return true, nil
	}
	instanceName := tc.GetInstanceName()
	curService := curSpec.Name
	selector, err := label.New().
		Instance(instanceName).
		PDMS(curService).
		Selector()
	if err != nil {
		return false, err
	}
	pdMSPods, err := m.deps.PodLister.Pods(tc.GetNamespace()).List(selector)
	if err != nil {
		return false, fmt.Errorf("pdMSStatefulSetIsUpgrading: failed to list pods for PDMS component %s for cluster [%s/%s] selector %s, error: %v",
			curService, tc.GetNamespace(), instanceName, selector, err)
	}
	for _, pod := range pdMSPods {
		revisionHash, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return false, nil
		}
		if revisionHash != tc.Status.PDMS[curService].StatefulSet.UpdateRevision {
			return true, nil
		}
	}
	return false, nil
}

func (m *pdMSMemberManager) getNewPDMSStatefulSet(tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap, curSpec *v1alpha1.PDMSSpec) (*apps.StatefulSet, error) {
	ns := tc.Namespace
	tcName := tc.Name
	curService := curSpec.Name
	basePDMSSpec := tc.BasePDMSSpec(curSpec)
	instanceName := tc.GetInstanceName()
	pdMSConfigMap := controller.MemberConfigMapName(tc, v1alpha1.PDMSMemberType(curService))
	if cm != nil {
		pdMSConfigMap = cm.Name
	}

	// TODO: Add version check

	annMount, annVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		annMount,
		{Name: "config", ReadOnly: true, MountPath: "/etc/pd"},
		{Name: "startup-script", ReadOnly: true, MountPath: "/usr/local/bin"},
	}
	if tc.IsTLSClusterEnabled() {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "pd-tls", ReadOnly: true, MountPath: "/var/lib/pd-tls",
		})
		if curSpec.MountClusterClientSecret != nil && *curSpec.MountClusterClientSecret {
			volMounts = append(volMounts, corev1.VolumeMount{
				Name: util.ClusterClientVolName, ReadOnly: true, MountPath: util.ClusterClientTLSPath,
			})
		}
	}

	vols := []corev1.Volume{
		annVolume,
		{Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: pdMSConfigMap,
					},
					Items: []corev1.KeyToPath{{Key: "config-file", Path: "pd.toml"}},
				},
			},
		},
		{Name: "startup-script",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: pdMSConfigMap,
					},
					Items: []corev1.KeyToPath{{Key: "startup-script", Path: "pdms_start_script.sh"}},
				},
			},
		},
	}
	if tc.IsTLSClusterEnabled() {
		// reuse the secret created for pd
		vols = append(vols, corev1.Volume{
			Name: "pd-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterTLSSecretName(tc.Name, label.PDLabelVal),
				},
			},
		})
		if curSpec.MountClusterClientSecret != nil && *curSpec.MountClusterClientSecret {
			vols = append(vols, corev1.Volume{
				Name: util.ClusterClientVolName, VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: util.ClusterClientTLSSecretName(tc.Name),
					},
				},
			})
		}
	}

	// handle StorageVolumes and AdditionalVolumeMounts in ComponentSpec
	var additionalPVCs []corev1.PersistentVolumeClaim
	// default in nil
	if curSpec.StorageVolumes != nil {
		storageVolMounts, addPVCs := util.BuildStorageVolumeAndVolumeMount(curSpec.StorageVolumes, curSpec.StorageClassName, v1alpha1.PDMSMemberType(curService))
		volMounts = append(volMounts, storageVolMounts...)
		volMounts = append(volMounts, curSpec.AdditionalVolumeMounts...)
		additionalPVCs = addPVCs
	}

	sysctls := "sysctl -w"
	var initContainers []corev1.Container
	if basePDMSSpec.Annotations() != nil {
		init, ok := basePDMSSpec.Annotations()[label.AnnSysctlInit]
		if ok && (init == label.AnnSysctlInitVal) {
			if basePDMSSpec.PodSecurityContext() != nil && len(basePDMSSpec.PodSecurityContext().Sysctls) > 0 {
				for _, sysctl := range basePDMSSpec.PodSecurityContext().Sysctls {
					sysctls = sysctls + fmt.Sprintf(" %s=%s", sysctl.Name, sysctl.Value)
				}
				privileged := true
				initContainers = append(initContainers, corev1.Container{
					Name:  "init",
					Image: tc.HelperImage(),
					Command: []string{
						"sh",
						"-c",
						sysctls,
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: &privileged,
					},
					// Init container resourceRequirements should be equal to app container.
					// Scheduling is done based on effective requests/limits,
					// which means init containers can reserve resources for
					// initialization that are not used during the life of the Pod.
					// ref:https://kubernetes.io/docs/concepts/workloads/pods/init-containers/#resources
					Resources: controller.ContainerResource(curSpec.ResourceRequirements),
				})
			}
		}
	}
	// Init container is only used for the case where allowed-unsafe-sysctls
	// cannot be enabled for kubelet, so clean the sysctl in statefulset
	// SecurityContext if init container is enabled
	podSecurityContext := basePDMSSpec.PodSecurityContext().DeepCopy()
	if len(initContainers) > 0 {
		podSecurityContext.Sysctls = []corev1.Sysctl{}
	}

	setName := controller.PDMSMemberName(tcName, curService)
	stsLabels := label.New().Instance(instanceName).PDMS(curService)
	podLabels := util.CombineStringMap(stsLabels, basePDMSSpec.Labels())
	podAnnotations := util.CombineStringMap(basePDMSSpec.Annotations(), controller.AnnProm(v1alpha1.DefaultPDClientPort, "/metrics"))
	stsAnnotations := getStsAnnotations(tc.Annotations, label.PDMSLabel(curService))

	deleteSlotsNumber, err := util.GetDeleteSlotsNumber(stsAnnotations)
	if err != nil {
		return nil, fmt.Errorf("get delete slots number of statefulset PDMS component %s for cluster [%s/%s] %s failed, err:%v",
			curService, ns, setName, curService, err)
	}

	pdMSContainer := corev1.Container{
		Name:            v1alpha1.PDMSMemberType(curService).String(),
		Image:           tc.PDMSImage(curSpec),
		ImagePullPolicy: basePDMSSpec.ImagePullPolicy(),
		Command:         []string{"/bin/sh", "/usr/local/bin/pdms_start_script.sh"},
		Ports: []corev1.ContainerPort{
			{
				Name:          "server",
				ContainerPort: v1alpha1.DefaultPDClientPort,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: volMounts,
		Resources:    controller.ContainerResource(curSpec.ResourceRequirements),
	}

	headlessSvcName := controller.PDMSPeerMemberName(tcName, curService)
	env := []corev1.EnvVar{
		{
			Name: "NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name:  "PEER_SERVICE_NAME",
			Value: controller.PDMSPeerMemberName(tcName, curService),
		},
		{
			Name:  "SERVICE_NAME",
			Value: controller.PDMSMemberName(tcName, curService),
		},
		{
			Name:  "SET_NAME",
			Value: setName,
		},
		{
			Name:  "TZ",
			Value: tc.Spec.Timezone,
		},
		{
			Name:  "CLUSTER_NAME",
			Value: tc.Name,
		},
		{
			Name:  "HEADLESS_SERVICE_NAME",
			Value: headlessSvcName,
		},
	}

	podSpec := basePDMSSpec.BuildPodSpec()
	if podSpec.HostNetwork {
		env = append(env, corev1.EnvVar{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		})
	}
	pdMSContainer.Env = util.AppendEnv(env, basePDMSSpec.Env())
	pdMSContainer.EnvFrom = basePDMSSpec.EnvFrom()
	podSpec.Volumes = append(vols, basePDMSSpec.AdditionalVolumes()...)
	podSpec.Containers, err = MergePatchContainers([]corev1.Container{pdMSContainer}, basePDMSSpec.AdditionalContainers())
	if err != nil {
		return nil, fmt.Errorf("failed to merge containers spec for PDMS %s of [%s/%s], error: %v", curService, tc.Namespace, tc.Name, err)
	}

	podSpec.ServiceAccountName = curSpec.ServiceAccount
	if podSpec.ServiceAccountName == "" {
		podSpec.ServiceAccountName = curSpec.ServiceAccount
	}
	podSpec.SecurityContext = podSecurityContext
	podSpec.InitContainers = append(initContainers, basePDMSSpec.InitContainers()...)

	updateStrategy := apps.StatefulSetUpdateStrategy{}
	if basePDMSSpec.StatefulSetUpdateStrategy() == apps.OnDeleteStatefulSetStrategyType {
		updateStrategy.Type = apps.OnDeleteStatefulSetStrategyType
	} else {
		updateStrategy.Type = apps.RollingUpdateStatefulSetStrategyType
		updateStrategy.RollingUpdate = &apps.RollingUpdateStatefulSetStrategy{
			Partition: pointer.Int32Ptr(tc.PDMSStsDesiredReplicas(curService) + deleteSlotsNumber),
		}
	}

	pdMSSet := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            setName,
			Namespace:       ns,
			Labels:          stsLabels.Labels(),
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(tc.PDMSStsDesiredReplicas(curService)),
			Selector: stsLabels.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
			ServiceName:         controller.PDMSPeerMemberName(tcName, curService),
			PodManagementPolicy: basePDMSSpec.PodManagementPolicy(),
			UpdateStrategy:      updateStrategy,
		},
	}
	// default in nil
	if curSpec.StorageVolumes != nil {
		pdMSSet.Spec.VolumeClaimTemplates = append(pdMSSet.Spec.VolumeClaimTemplates, additionalPVCs...)
	}

	return pdMSSet, nil
}

func (m *pdMSMemberManager) getPDMSConfigMap(tc *v1alpha1.TidbCluster, curSpec *v1alpha1.PDMSSpec) (*corev1.ConfigMap, error) {
	var config *v1alpha1.PDConfigWraper
	if curSpec.Config != nil {
		config = curSpec.Config.DeepCopy() // use copy to not update tc spec
	} else {
		config = v1alpha1.NewPDConfig()
	}

	curService := curSpec.Name
	// override CA if tls enabled
	if tc.IsTLSClusterEnabled() {
		config.Set("security.cacert-path", path.Join(pdClusterCertPath, tlsSecretRootCAKey))
		config.Set("security.cert-path", path.Join(pdClusterCertPath, corev1.TLSCertKey))
		config.Set("security.key-path", path.Join(pdClusterCertPath, corev1.TLSPrivateKeyKey))
	}

	confText, err := config.MarshalTOML()
	if err != nil {
		return nil, err
	}

	startScript, err := startscript.RenderPDMSStartScript(tc, curService)
	if err != nil {
		return nil, err
	}

	instanceName := tc.GetInstanceName()
	pdMSLabel := label.New().Instance(instanceName).PDMS(curService).Labels()
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            controller.PDMSMemberName(tc.Name, curService),
			Namespace:       tc.Namespace,
			Labels:          pdMSLabel,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Data: map[string]string{
			"config-file":    string(confText),
			"startup-script": startScript,
		},
	}
	return cm, nil
}

type FakePDMSMemberManager struct {
	err error
}

func NewFakePDMSMemberManager() *FakePDMSMemberManager {
	return &FakePDMSMemberManager{}
}

func (m *FakePDMSMemberManager) SetSyncError(err error) {
	m.err = err
}

func (m *FakePDMSMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	if m.err != nil {
		return m.err
	}
	return nil
}
