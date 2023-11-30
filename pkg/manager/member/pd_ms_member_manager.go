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

	"github.com/Masterminds/semver"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/manager/member/startscript"
	"github.com/pingcap/tidb-operator/pkg/manager/suspender"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"
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
	deps      *controller.Dependencies
	scaler    Scaler
	upgrader  Upgrader
	suspender suspender.Suspender
	// current micro service spec
	curSpec *v1alpha1.PDMSSpec
}

// NewPDMSMemberManager returns a *pdMSMemberManager
func NewPDMSMemberManager(dependencies *controller.Dependencies, pdMSScaler Scaler, pdMSUpgrader Upgrader, spder suspender.Suspender) manager.Manager {
	return &pdMSMemberManager{
		deps:      dependencies,
		scaler:    pdMSScaler,
		upgrader:  pdMSUpgrader,
		suspender: spder,
	}
}

// Sync for all PD Micro Service components.
func (m *pdMSMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	if tc.Spec.PDMS == nil {
		return nil
	}

	// Need to start PD API
	if tc.Spec.PD == nil || tc.Spec.PD.Mode != "ms" {
		return nil
	}
	// init pdMS status
	if tc.Status.PDMS == nil {
		tc.Status.PDMS = make(map[string]*v1alpha1.PDMSStatus)
	}

	for _, comp := range tc.Spec.PDMS {
		m.curSpec = comp
		if tc.Status.PDMS[m.curSpec.Name] == nil {
			tc.Status.PDMS[m.curSpec.Name] = &v1alpha1.PDMSStatus{Name: m.curSpec.Name}
		}
		if err := m.syncSingleService(tc); err != nil {
			klog.Errorf("syncSingleService failed, error: %v, component: %s", err, m.curSpec.Name)
			return err
		}
	}

	return nil
}

// syncSingleService for single PD Micro Service components.
func (m *pdMSMemberManager) syncSingleService(tc *v1alpha1.TidbCluster) error {
	metrics.ClusterUpdateErrors.WithLabelValues(tc.GetNamespace(), tc.GetName(), m.curSpec.Name).Inc()
	// Skip sync if PD Micro Service is suspended
	componentMemberType := v1alpha1.PDMSMemberType(m.curSpec.Name)
	needSuspend, err := m.suspender.SuspendComponent(tc, componentMemberType)
	if err != nil {
		return fmt.Errorf("suspend %s failed: %v", componentMemberType, err)
	}
	if needSuspend {
		klog.Infof("component %s for cluster %s/%s is suspended, skip syncing", m.curSpec.Name, tc.GetNamespace(), tc.GetName())
		return nil
	}

	// Sync PD Micro Service
	if err := m.syncPDMSService(tc); err != nil {
		return err
	}

	// Sync PD Micro Service Headless Service
	if err := m.syncPDMSHeadlessService(tc); err != nil {
		return err
	}

	// Sync PD Micro Service StatefulSet
	return m.syncPDMSStatefulSet(tc)
}

func (m *pdMSMemberManager) syncPDMSService(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	if tc.Spec.Paused {
		klog.Infof("tidb cluster %s/%s is paused, skip syncing for pdMS service", ns, tcName)
		return nil
	}

	newSvc := m.getNewPDMSService(tc)
	oldSvcTmp, err := m.deps.ServiceLister.Services(ns).Get(controller.PDMSMemberName(tcName, m.curSpec.Name))
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return m.deps.ServiceControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return fmt.Errorf("syncpdMSServiceForTidbCluster: failed to get svc %s for cluster %s/%s, error: %s", controller.PDMSMemberName(tcName, m.curSpec.Name), ns, tcName, err)
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

func (m *pdMSMemberManager) syncPDMSHeadlessService(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	if tc.Spec.Paused {
		klog.Infof("tidb cluster %s/%s is paused, skip syncing for %s headless service", ns, tcName, m.curSpec.Name)
		return nil
	}

	newSvc := getNewPDMSHeadlessService(tc, m.curSpec.Name)
	oldSvcTmp, err := m.deps.ServiceLister.Services(ns).Get(controller.PDMSPeerMemberName(tcName, m.curSpec.Name))
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return m.deps.ServiceControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return fmt.Errorf("syncpdMSHeadlessService: failed to get svc %s for cluster %s/%s, error: %s", controller.PDMSPeerMemberName(tcName, m.curSpec.Name), ns, tcName, err)
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

func (m *pdMSMemberManager) syncPDMSStatefulSet(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	componentName := m.curSpec.Name
	oldPDMSSetTmp, err := m.deps.StatefulSetLister.StatefulSets(ns).Get(controller.PDMSMemberName(tcName, componentName))
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncPDMSStatefulSet: fail to get sts %s for cluster %s/%s, error: %s", controller.PDMSMemberName(tcName, componentName), ns, tcName, err)
	}

	setNotExist := errors.IsNotFound(err)
	oldPDMSSet := oldPDMSSetTmp.DeepCopy()
	if err := m.syncStatus(tc, oldPDMSSet); err != nil {
		klog.Errorf("failed to sync TidbCluster: [%s/%s]'s status for %s, error: %v", ns, tcName, componentName, err)
	}

	if tc.Spec.Paused {
		klog.Infof("tidb cluster %s/%s is paused, skip syncing for pdMS statefulset", tc.GetNamespace(), tc.GetName())
		return nil
	}

	cm, err := m.syncPDMSConfigMap(tc, oldPDMSSet)
	if err != nil {
		return err
	}

	newPDMSSet, err := m.getNewPDMSStatefulSet(tc, cm)
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
		if tc.Status.PDMS[m.curSpec.Name] == nil {
			tc.Status.PDMS[m.curSpec.Name] = &v1alpha1.PDMSStatus{Name: m.curSpec.Name}
		}
		tc.Status.PDMS[m.curSpec.Name].StatefulSet = &apps.StatefulSetStatus{}

		return controller.RequeueErrorf("TidbCluster: [%s/%s], waiting for PDMS cluster running", ns, tcName)
	}

	// Scaling takes precedence over upgrading because:
	// - if a pdMS fails in the upgrading, users may want to delete it or add
	//   new replicas
	// - it's ok to scale in the middle of upgrading (in statefulset controller
	//   scaling takes precedence over upgrading too)
	if err := m.scaler.Scale(tc, oldPDMSSet, newPDMSSet); err != nil {
		return err
	}

	if !templateEqual(newPDMSSet, oldPDMSSet) || tc.Status.PDMS[m.curSpec.Name].Phase == v1alpha1.UpgradePhase {
		if err := m.upgrader.Upgrade(tc, oldPDMSSet, newPDMSSet); err != nil {
			return err
		}
	}

	return mngerutils.UpdateStatefulSetWithPrecheck(m.deps, tc, "FailedUpdatePDMSSTS", newPDMSSet, oldPDMSSet)
}

func (m *pdMSMemberManager) syncStatus(tc *v1alpha1.TidbCluster, sts *apps.StatefulSet) error {
	if sts == nil {
		// skip if not created yet
		return nil
	}

	tc.Status.PDMS[m.curSpec.Name].Name = m.curSpec.Name
	tc.Status.PDMS[m.curSpec.Name].StatefulSet = &sts.Status
	upgrading, err := m.pdMSStatefulSetIsUpgrading(sts, tc)
	if err != nil {
		return err
	}

	// Scaling takes precedence over upgrading.
	if tc.PDMSStsDesiredReplicas(m.curSpec.Name) != *sts.Spec.Replicas {
		tc.Status.PDMS[m.curSpec.Name].Phase = v1alpha1.ScalePhase
	} else if upgrading {
		tc.Status.PDMS[m.curSpec.Name].Phase = v1alpha1.UpgradePhase
	} else {
		tc.Status.PDMS[m.curSpec.Name].Phase = v1alpha1.NormalPhase
	}

	pdClient := controller.GetPDClient(m.deps.PDControl, tc)
	// pdMS member
	members, err := pdClient.GetServiceMembers(m.curSpec.Name)
	if err != nil {
		return err
	}
	tc.Status.PDMS[m.curSpec.Name].Members = members
	tc.Status.PDMS[m.curSpec.Name].Synced = true
	return nil
}

// syncPDMSConfigMap syncs the configmap of PDMS
func (m *pdMSMemberManager) syncPDMSConfigMap(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) (*corev1.ConfigMap, error) {
	// For backward compatibility, only sync tidb configmap when .PDMS.config is non-nil
	if m.curSpec.Config == nil {
		return nil, nil
	}
	newCm, err := m.getPDMSConfigMap(tc)
	if err != nil {
		return nil, err
	}

	var inUseName string
	if set != nil {
		inUseName = mngerutils.FindConfigMapVolume(&set.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, controller.PDMSMemberName(tc.Name, m.curSpec.Name))
		})
	}

	err = mngerutils.UpdateConfigMapIfNeed(m.deps.ConfigMapLister, tc.BasePDMSSpec(m.curSpec).ConfigUpdateStrategy(), inUseName, newCm)
	if err != nil {
		return nil, err
	}
	return m.deps.TypedControl.CreateOrUpdateConfigMap(tc, newCm)
}

func (m *pdMSMemberManager) getNewPDMSService(tc *v1alpha1.TidbCluster) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	svcName := controller.PDMSMemberName(tcName, m.curSpec.Name)
	instanceName := tc.GetInstanceName()
	pdMSSelector := label.New().Instance(instanceName).PDMS(m.curSpec.Name)
	pdMSLabels := pdMSSelector.Copy().UsedByEndUser().Labels()

	pdMSService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          pdMSLabels,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: corev1.ServiceSpec{
			Type: controller.GetServiceType(tc.Spec.Services, m.curSpec.Name),
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
	svcSpec := m.curSpec.Service
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

func (m *pdMSMemberManager) pdMSStatefulSetIsUpgrading(set *apps.StatefulSet, tc *v1alpha1.TidbCluster) (bool, error) {
	if mngerutils.StatefulSetIsUpgrading(set) {
		return true, nil
	}
	instanceName := tc.GetInstanceName()
	selector, err := label.New().
		Instance(instanceName).
		PDMS(m.curSpec.Name).
		Selector()
	if err != nil {
		return false, err
	}
	pdMSPods, err := m.deps.PodLister.Pods(tc.GetNamespace()).List(selector)
	if err != nil {
		return false, fmt.Errorf("pdMSStatefulSetIsUpgrading: failed to list pods for cluster %s/%s, selector %s, error: %v", tc.GetNamespace(), instanceName, selector, err)
	}
	for _, pod := range pdMSPods {
		revisionHash, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return false, nil
		}
		if revisionHash != tc.Status.PDMS[m.curSpec.Name].StatefulSet.UpdateRevision {
			return true, nil
		}
	}
	return false, nil
}

func (m *pdMSMemberManager) getNewPDMSStatefulSet(tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
	ns := tc.Namespace
	tcName := tc.Name
	basePDMSSpec := tc.BasePDMSSpec(m.curSpec)
	instanceName := tc.GetInstanceName()
	pdMSConfigMap := controller.MemberConfigMapName(tc, v1alpha1.PDMSMemberType(m.curSpec.Name))
	if cm != nil {
		pdMSConfigMap = cm.Name
	}

	microServicesVersion, err := PDMSSupportMicroServices(tc.PDMSVersion())
	if err != nil {
		klog.Errorf("cluster version: %s is not semantic versioning compatible", tc.PDMSVersion())
		return nil, err
	}

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
		if m.curSpec.MountClusterClientSecret != nil && *m.curSpec.MountClusterClientSecret {
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
					Items: []corev1.KeyToPath{{Key: "startup-script", Path: "pdMS_start_script.sh"}},
				},
			},
		},
	}
	if tc.IsTLSClusterEnabled() {
		vols = append(vols, corev1.Volume{
			Name: "pd-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterTLSSecretName(tc.Name, label.PDMSLabel(m.curSpec.Name)),
				},
			},
		})
		if m.curSpec.MountClusterClientSecret != nil && *m.curSpec.MountClusterClientSecret && microServicesVersion {
			vols = append(vols, corev1.Volume{
				Name: util.ClusterClientVolName, VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: util.ClusterClientTLSSecretName(tc.Name),
					},
				},
			})
		}
	}
	if tc.Spec.TiDB != nil && tc.Spec.TiDB.IsTLSClientEnabled() && !tc.SkipTLSWhenConnectTiDB() {
		vols = append(vols, corev1.Volume{
			Name: "tidb-client-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.TiDBClientTLSSecretName(tc.Name, m.curSpec.TLSClientSecretName),
				},
			},
		})
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
					Resources: controller.ContainerResource(m.curSpec.ResourceRequirements),
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

	setName := controller.PDMSMemberName(tcName, m.curSpec.Name)
	stsLabels := label.New().Instance(instanceName).PDMS(m.curSpec.Name)
	podLabels := util.CombineStringMap(stsLabels, basePDMSSpec.Labels())
	podAnnotations := util.CombineStringMap(basePDMSSpec.Annotations(), controller.AnnProm(v1alpha1.DefaultPDClientPort, "/metrics"))
	stsAnnotations := getStsAnnotations(tc.Annotations, label.PDMSLabel(m.curSpec.Name))

	deleteSlotsNumber, err := util.GetDeleteSlotsNumber(stsAnnotations)
	if err != nil {
		return nil, fmt.Errorf("get delete slots number of statefulset %s/%s failed, err:%v", ns, setName, err)
	}

	pdMSContainer := corev1.Container{
		Name:            m.curSpec.Name,
		Image:           tc.PDImage(),
		ImagePullPolicy: basePDMSSpec.ImagePullPolicy(),
		Command:         []string{"/bin/sh", "/usr/local/bin/pdMS_start_script.sh"},
		Ports: []corev1.ContainerPort{
			{
				Name:          "server",
				ContainerPort: v1alpha1.DefaultPDClientPort,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: volMounts,
		Resources:    controller.ContainerResource(m.curSpec.ResourceRequirements),
	}

	headlessSvcName := controller.PDMSPeerMemberName(tcName, m.curSpec.Name)
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
			Value: controller.PDMSPeerMemberName(tcName, m.curSpec.Name),
		},
		{
			Name:  "SERVICE_NAME",
			Value: controller.PDMSMemberName(tcName, m.curSpec.Name),
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
		return nil, fmt.Errorf("failed to merge containers spec for PDMS of [%s/%s], error: %v", tc.Namespace, tc.Name, err)
	}

	podSpec.ServiceAccountName = m.curSpec.ServiceAccount
	if podSpec.ServiceAccountName == "" {
		podSpec.ServiceAccountName = m.curSpec.ServiceAccount
	}
	podSpec.SecurityContext = podSecurityContext
	podSpec.InitContainers = append(initContainers, basePDMSSpec.InitContainers()...)

	updateStrategy := apps.StatefulSetUpdateStrategy{}
	if basePDMSSpec.StatefulSetUpdateStrategy() == apps.OnDeleteStatefulSetStrategyType {
		updateStrategy.Type = apps.OnDeleteStatefulSetStrategyType
	} else {
		updateStrategy.Type = apps.RollingUpdateStatefulSetStrategyType
		updateStrategy.RollingUpdate = &apps.RollingUpdateStatefulSetStrategy{
			Partition: pointer.Int32Ptr(tc.PDMSStsDesiredReplicas(m.curSpec.Name) + deleteSlotsNumber),
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
			Replicas: pointer.Int32Ptr(tc.PDMSStsDesiredReplicas(m.curSpec.Name)),
			Selector: stsLabels.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
			ServiceName:         controller.PDMSPeerMemberName(tcName, m.curSpec.Name),
			PodManagementPolicy: basePDMSSpec.PodManagementPolicy(),
			UpdateStrategy:      updateStrategy,
		},
	}

	return pdMSSet, nil
}

func (m *pdMSMemberManager) getPDMSConfigMap(tc *v1alpha1.TidbCluster) (*corev1.ConfigMap, error) {
	if m.curSpec.Config == nil {
		return nil, nil
	}
	config := m.curSpec.Config.DeepCopy() // use copy to not update tc spec

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

	startScript, err := startscript.RenderPDMCSStartScript(tc, m.curSpec.Name)
	if err != nil {
		return nil, err
	}

	instanceName := tc.GetInstanceName()
	pdMSLabel := label.New().Instance(instanceName).PDMS(m.curSpec.Name).Labels()
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            controller.PDMSMemberName(tc.Name, m.curSpec.Name),
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

// PDMSSupportMicroServices returns true if the given version of PDMS supports micro services.
func PDMSSupportMicroServices(version string) (bool, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return true, err
	}
	return v.Major() >= 7 && v.Minor() >= 1 && v.Patch() >= 0, nil
}

// TODO: seems not used
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
