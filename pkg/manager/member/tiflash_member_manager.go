// Copyright 2020 PingCAP, Inc.
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
	"reflect"
	"regexp"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/manager/member/startscript"
	"github.com/pingcap/tidb-operator/pkg/manager/suspender"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"
	"github.com/pingcap/tidb-operator/pkg/manager/volumes"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/util"

	"github.com/pingcap/kvproto/pkg/metapb"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
)

const (
	// find a better way to manage store only managed by tiflash in Operator
	tiflashStoreLimitPattern = `%s-tiflash-\d+\.%s-tiflash-peer\.%s\.svc%s:\d+`
	tiflashCertPath          = "/var/lib/tiflash-tls"
	tiflashCertVolumeName    = "tiflash-tls"
)

// tiflashMemberManager implements manager.Manager.
type tiflashMemberManager struct {
	deps                     *controller.Dependencies
	failover                 Failover
	scaler                   Scaler
	upgrader                 Upgrader
	suspender                suspender.Suspender
	podVolumeModifier        volumes.PodVolumeModifier
	statefulSetIsUpgradingFn func(corelisters.PodLister, pdapi.PDControlInterface, *apps.StatefulSet, *v1alpha1.TidbCluster) (bool, error)
}

// NewTiFlashMemberManager returns a *tiflashMemberManager
func NewTiFlashMemberManager(deps *controller.Dependencies, tiflashFailover Failover, tiflashScaler Scaler, tiflashUpgrader Upgrader, spder suspender.Suspender, pvm volumes.PodVolumeModifier) manager.Manager {
	m := tiflashMemberManager{
		deps:              deps,
		failover:          tiflashFailover,
		scaler:            tiflashScaler,
		upgrader:          tiflashUpgrader,
		suspender:         spder,
		podVolumeModifier: pvm,
	}
	m.statefulSetIsUpgradingFn = tiflashStatefulSetIsUpgrading
	return &m
}

// Sync fulfills the manager.Manager interface
func (m *tiflashMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	if tc.Spec.TiFlash == nil {
		return nil
	}

	// skip sync if tiflash is suspended
	component := v1alpha1.TiFlashMemberType
	needSuspend, err := m.suspender.SuspendComponent(tc, component)
	if err != nil {
		return fmt.Errorf("suspend %s failed: %v", component, err)
	}
	if needSuspend {
		klog.Infof("component %s for cluster %s/%s is suspended, skip syncing", component, tc.GetNamespace(), tc.GetName())
		return nil
	}

	if err := m.syncRecoveryForTiFlash(tc); err != nil {
		klog.Info("sync recovery for TiFlash", err.Error())
		return nil
	}

	err = m.enablePlacementRules(tc)
	if err != nil {
		klog.Errorf("Enable placement rules failed, error: %v", err)
		// No need to return err here, just continue to sync tiflash
	}
	// Sync TiFlash Headless Service
	if err = m.syncHeadlessService(tc); err != nil {
		return err
	}

	return m.syncStatefulSet(tc)
}

func (m *tiflashMemberManager) syncRecoveryForTiFlash(tc *v1alpha1.TidbCluster) error {
	// Check whether the cluster is in recovery mode
	// and whether the volumes have been restored for TiKV
	if !tc.Spec.RecoveryMode {
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()
	anns := tc.GetAnnotations()
	if _, ok := anns[label.AnnTiKVVolumesReadyKey]; !ok {
		return controller.RequeueErrorf("TidbCluster: [%s/%s], TiFlash is waiting for TiKV volumes ready", ns, tcName)
	}

	if mark, _ := m.checkRecoveringMark(tc); mark {
		return controller.RequeueErrorf("TidbCluster: [%s/%s], TiFlash is waiting for recovery mode unmask", ns, tcName)
	}

	return nil
}

// check the recovering mark from pd
// volume-snapshot restore requires pd allocate id set done and then start tiflash. the purpose is to solve tiflash store id conflict with tikvs'
// pd recovering mark indicates the pd allcate id had been set properly.
func (m *tiflashMemberManager) checkRecoveringMark(tc *v1alpha1.TidbCluster) (bool, error) {
	pdCli := controller.GetPDClient(m.deps.PDControl, tc)
	mark, err := pdCli.GetRecoveringMark()
	if err != nil {
		return false, err
	}

	return mark, nil
}

func (m *tiflashMemberManager) enablePlacementRules(tc *v1alpha1.TidbCluster) error {
	pdCli := controller.GetPDClient(m.deps.PDControl, tc)
	config, err := pdCli.GetConfig()
	if err != nil {
		return err
	}
	if config.Replication.EnablePlacementRules != nil && (!*config.Replication.EnablePlacementRules) {
		klog.Infof("Cluster %s/%s enable-placement-rules is %v, set it to true", tc.Namespace, tc.Name, *config.Replication.EnablePlacementRules)
		enable := true
		rep := pdapi.PDReplicationConfig{
			EnablePlacementRules: &enable,
		}
		return pdCli.UpdateReplicationConfig(rep)
	}
	return nil
}

func (m *tiflashMemberManager) syncHeadlessService(tc *v1alpha1.TidbCluster) error {
	if tc.Spec.Paused {
		klog.V(4).Infof("tiflash cluster %s/%s is paused, skip syncing for tiflash service", tc.GetNamespace(), tc.GetName())
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := getNewHeadlessService(tc)
	oldSvcTmp, err := m.deps.ServiceLister.Services(ns).Get(controller.TiFlashPeerMemberName(tcName))
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return m.deps.ServiceControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return fmt.Errorf("syncHeadlessService: failed to get svc %s for cluster %s/%s, error: %s", controller.TiFlashPeerMemberName(tcName), ns, tcName, err)
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

func (m *tiflashMemberManager) syncStatefulSet(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	oldSetTmp, err := m.deps.StatefulSetLister.StatefulSets(ns).Get(controller.TiFlashMemberName(tcName))
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncStatefulSet: fail to get sts %s for cluster %s/%s, error: %s", controller.TiFlashMemberName(tcName), ns, tcName, err)
	}
	setNotExist := errors.IsNotFound(err)

	// if TiFlash is scale from 0 and with previous StatefulSet, we delete the previous StatefulSet first
	// to avoid some fileds (e.g storage request) reused and cause unexpected behavior (e.g scale down).
	if oldSetTmp != nil && *oldSetTmp.Spec.Replicas == 0 && oldSetTmp.Status.UpdatedReplicas == 0 && tc.Spec.TiFlash.Replicas > 0 {
		if err := m.deps.StatefulSetControl.DeleteStatefulSet(tc, oldSetTmp, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("syncStatefulSet: fail to delete sts %s for cluster %s/%s, error: %s", controller.TiFlashMemberName(tcName), ns, tcName, err)
		}
		return controller.RequeueErrorf("wait for previous sts %s for cluster %s/%s to be deleted", controller.TiFlashMemberName(tcName), ns, tcName)
	}

	oldSet := oldSetTmp.DeepCopy()

	if err := m.syncTidbClusterStatus(tc, oldSet); err != nil {
		return err
	}

	if tc.Spec.Paused {
		klog.V(4).Infof("tiflash cluster %s/%s is paused, skip syncing for tiflash statefulset", tc.GetNamespace(), tc.GetName())
		return nil
	}

	cm, err := m.syncConfigMap(tc, oldSet)
	if err != nil {
		return err
	}

	// Recover failed stores if any before generating desired statefulset
	if len(tc.Status.TiFlash.FailureStores) > 0 {
		m.failover.RemoveUndesiredFailures(tc)
	}
	if len(tc.Status.TiFlash.FailureStores) > 0 &&
		(tc.Spec.TiFlash.RecoverFailover || tc.Status.TiFlash.FailoverUID == tc.Spec.TiFlash.GetRecoverByUID()) &&
		shouldRecover(tc, label.TiFlashLabelVal, m.deps.PodLister) {
		m.failover.Recover(tc)
	}

	newSet, err := getNewStatefulSet(tc, cm)
	if err != nil {
		return err
	}
	if setNotExist {
		if !tc.PDIsAvailable() {
			klog.Infof("TidbCluster: %s/%s, waiting for PD cluster running", ns, tcName)
			return nil
		}
		err = mngerutils.SetStatefulSetLastAppliedConfigAnnotation(newSet)
		if err != nil {
			return err
		}
		err = m.deps.StatefulSetControl.CreateStatefulSet(tc, newSet)
		if err != nil {
			return err
		}
		tc.Status.TiFlash.StatefulSet = &apps.StatefulSetStatus{}
		return nil
	}

	if _, err := m.setStoreLabelsForTiFlash(tc); err != nil {
		return err
	}

	// Scaling takes precedence over upgrading because:
	// - if a tiflash fails in the upgrading, users may want to delete it or add
	//   new replicas
	// - it's ok to scale in the middle of upgrading (in statefulset controller
	//   scaling takes precedence over upgrading too)
	if err := m.scaler.Scale(tc, oldSet, newSet); err != nil {
		return err
	}

	if m.deps.CLIConfig.AutoFailover && tc.Spec.TiFlash.MaxFailoverCount != nil {
		if tc.TiFlashAllPodsStarted() && !tc.TiFlashAllStoresReady() {
			if err := m.failover.Failover(tc); err != nil {
				return err
			}
		}
	}

	if !templateEqual(newSet, oldSet) || tc.Status.TiFlash.Phase == v1alpha1.UpgradePhase {
		if err := m.upgrader.Upgrade(tc, oldSet, newSet); err != nil {
			return err
		}
	}

	return mngerutils.UpdateStatefulSetWithPrecheck(m.deps, tc, "FailedUpdateTiFlashSTS", newSet, oldSet)
}

func (m *tiflashMemberManager) syncConfigMap(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) (*corev1.ConfigMap, error) {
	newCm, err := getTiFlashConfigMap(tc)
	if err != nil {
		return nil, err
	}

	var inUseName string
	if set != nil {
		inUseName = mngerutils.FindConfigMapVolume(&set.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, controller.TiFlashMemberName(tc.Name))
		})
	} else {
		inUseName, err = mngerutils.FindConfigMapNameFromTCAnno(context.Background(), m.deps.ConfigMapLister, tc, v1alpha1.TiFlashMemberType, newCm)
		if err != nil {
			return nil, err
		}
	}

	err = mngerutils.UpdateConfigMapIfNeed(m.deps.ConfigMapLister, tc.BaseTiFlashSpec().ConfigUpdateStrategy(), inUseName, newCm)
	if err != nil {
		return nil, err
	}
	return m.deps.TypedControl.CreateOrUpdateConfigMap(tc, newCm)
}

func getNewHeadlessService(tc *v1alpha1.TidbCluster) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	instanceName := tc.GetInstanceName()
	svcName := controller.TiFlashPeerMemberName(tcName)
	svcLabel := label.New().Instance(instanceName).TiFlash().Labels()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          svcLabel,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:       "tiflash",
					Port:       v1alpha1.DefaultTiFlashFlashPort,
					TargetPort: intstr.FromInt(int(v1alpha1.DefaultTiFlashFlashPort)),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "proxy",
					Port:       v1alpha1.DefaultTiFlashProxyPort,
					TargetPort: intstr.FromInt(int(v1alpha1.DefaultTiFlashProxyPort)),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "metrics",
					Port:       v1alpha1.DefaultTiFlashMetricsPort,
					TargetPort: intstr.FromInt(int(v1alpha1.DefaultTiFlashMetricsPort)),
					Protocol:   corev1.ProtocolTCP,
				},

				{
					Name:       "proxy-metrics",
					Port:       v1alpha1.DefaultTiFlashProxyStatusPort,
					TargetPort: intstr.FromInt(int(v1alpha1.DefaultTiFlashProxyStatusPort)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector:                 svcLabel,
			PublishNotReadyAddresses: true,
		},
	}

	if tc.Spec.PreferIPv6 {
		SetServiceWhenPreferIPv6(svc)
	}

	return svc
}

func getNewStatefulSet(tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	baseTiFlashSpec := tc.BaseTiFlashSpec()
	spec := tc.Spec.TiFlash
	mountCMInTiflashContainer := spec.DoesMountCMInTiflashContainer()

	tiflashConfigMap := controller.MemberConfigMapName(tc, v1alpha1.TiFlashMemberType)
	if cm != nil {
		tiflashConfigMap = cm.Name
	}

	// This should not happen as we have validaton for this field
	if len(spec.StorageClaims) < 1 {
		return nil, fmt.Errorf("storageClaims should be configured at least one item for tiflash, tidbcluster %s/%s", tc.Namespace, tc.Name)
	}
	pvcs, err := flashVolumeClaimTemplate(tc.Spec.TiFlash.StorageClaims)
	if err != nil {
		return nil, fmt.Errorf("cannot parse storage request for tiflash.StorageClaims, tidbcluster %s/%s, error: %v", tc.Namespace, tc.Name, err)
	}
	annoMount, annoVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		annoMount,
	}
	for k := range spec.StorageClaims {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: fmt.Sprintf("data%d", k), MountPath: fmt.Sprintf("/data%d", k),
		})
	}
	volMounts = append(volMounts, tc.Spec.TiFlash.AdditionalVolumeMounts...)

	if tc.IsTLSClusterEnabled() {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: tiflashCertVolumeName, ReadOnly: true, MountPath: tiflashCertPath,
		})
	}
	// with mountCMInTiflashContainer enabled, the tiflash container should directly read config from `/etc/tiflash/xxx.toml` mounted from ConfigMap
	// rather than `/data0/xxx.toml` created by initContainer.
	if mountCMInTiflashContainer {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "config", ReadOnly: true, MountPath: "/etc/tiflash",
		})
	}

	vols := []corev1.Volume{
		annoVolume,
		{
			Name: "config", VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: tiflashConfigMap,
					},
				},
			},
		},
	}

	if tc.IsTLSClusterEnabled() {
		vols = append(vols, corev1.Volume{
			Name: tiflashCertVolumeName, VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterTLSSecretName(tc.Name, label.TiFlashLabelVal),
				},
			},
		})
	}

	sysctls := "sysctl -w"
	var initContainers []corev1.Container
	if baseTiFlashSpec.Annotations() != nil {
		init, ok := baseTiFlashSpec.Annotations()[label.AnnSysctlInit]
		if ok && (init == label.AnnSysctlInitVal) {
			if baseTiFlashSpec.PodSecurityContext() != nil && len(baseTiFlashSpec.PodSecurityContext().Sysctls) > 0 {
				for _, sysctl := range baseTiFlashSpec.PodSecurityContext().Sysctls {
					sysctls = sysctls + fmt.Sprintf(" %s=%s", sysctl.Name, sysctl.Value)
				}
				privileged := true
				initContainers = append(initContainers, corev1.Container{
					Name:  "sysctl",
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
					Resources: controller.ContainerResource(tc.Spec.TiFlash.ResourceRequirements),
				})
			}
		}
	}
	// Init container is only used for the case where allowed-unsafe-sysctls
	// cannot be enabled for kubelet, so clean the sysctl in statefulset
	// SecurityContext if init container is enabled
	podSecurityContext := baseTiFlashSpec.PodSecurityContext().DeepCopy()
	if len(initContainers) > 0 {
		podSecurityContext.Sysctls = []corev1.Sysctl{}
	}

	// the work of initContainer is substituted by new start scripts which moves the configuration items depending on running env
	// to command args.
	if !mountCMInTiflashContainer {
		// Append init container for config files initialization
		initVolMounts := []corev1.VolumeMount{
			{Name: "data0", MountPath: "/data0"},
			{Name: "config", ReadOnly: true, MountPath: "/etc/tiflash"},
		}
		initEnv := []corev1.EnvVar{
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
		}

		// NOTE: the config content should respect the init script
		initScript, err := startscript.RenderTiFlashInitScript(tc)
		if err != nil {
			return nil, fmt.Errorf("render start-script for tc %s/%s failed: %v", tc.Namespace, tc.Name, err)
		}

		initializer := corev1.Container{
			Name:  "init",
			Image: tc.HelperImage(),
			Command: []string{
				"sh",
				"-c",
				initScript,
			},
			Env:          initEnv,
			VolumeMounts: initVolMounts,
		}
		if spec.Initializer != nil {
			initializer.Resources = controller.ContainerResource(spec.Initializer.ResourceRequirements)
		}
		initContainers = append(initContainers, initializer)
	}

	stsLabels := labelTiFlash(tc)
	setName := controller.TiFlashMemberName(tcName)
	podLabels := util.CombineStringMap(stsLabels, baseTiFlashSpec.Labels())
	podAnnotations := util.CombineStringMap(baseTiFlashSpec.Annotations(), controller.AnnProm(v1alpha1.DefaultTiFlashMetricsPort, "/metrics"))
	podAnnotations = util.CombineStringMap(controller.AnnAdditionalProm("tiflash.proxy", v1alpha1.DefaultTiFlashProxyStatusPort), podAnnotations)
	stsAnnotations := getStsAnnotations(tc.Annotations, label.TiFlashLabelVal)
	capacity := controller.TiKVCapacity(tc.Spec.TiFlash.Limits)
	headlessSvcName := controller.TiFlashPeerMemberName(tcName)

	deleteSlotsNumber, err := util.GetDeleteSlotsNumber(stsAnnotations)
	if err != nil {
		return nil, fmt.Errorf("get delete slots number of statefulset %s/%s failed, err:%v", ns, setName, err)
	}

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
			Name:  "CLUSTER_NAME",
			Value: tcName,
		},
		{
			Name:  "HEADLESS_SERVICE_NAME",
			Value: headlessSvcName,
		},
		{
			Name:  "CAPACITY",
			Value: capacity,
		},
		{
			Name:  "TZ",
			Value: tc.Timezone(),
		},
	}

	startScript, err := startscript.RenderTiFlashStartScript(tc)
	if err != nil {
		return nil, fmt.Errorf("render start-script for tc %s/%s failed: %v", tc.Namespace, tc.Name, err)
	}

	tiflashContainer := corev1.Container{
		Name:            v1alpha1.TiFlashMemberType.String(),
		Image:           tc.TiFlashImage(),
		ImagePullPolicy: baseTiFlashSpec.ImagePullPolicy(),
		Command:         []string{"/bin/sh", "-c", startScript},
		SecurityContext: &corev1.SecurityContext{
			Privileged: tc.TiFlashContainerPrivilege(),
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "tiflash",
				ContainerPort: v1alpha1.DefaultTiFlashFlashPort,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "proxy",
				ContainerPort: v1alpha1.DefaultTiFlashProxyPort,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "tcp",
				ContainerPort: v1alpha1.DefaultTiFlashTcpPort,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "http",
				ContainerPort: v1alpha1.DefaultTiFlashHttpPort,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "internal",
				ContainerPort: v1alpha1.DefaultTiFlashInternalPort,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "metrics",
				ContainerPort: v1alpha1.DefaultTiFlashMetricsPort,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: volMounts,
		Resources:    controller.ContainerResource(tc.Spec.TiFlash.ResourceRequirements),
	}
	podSpec := baseTiFlashSpec.BuildPodSpec()
	if baseTiFlashSpec.HostNetwork() {
		env = append(env, corev1.EnvVar{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		})
	}
	tiflashContainer.Env = util.AppendEnv(env, baseTiFlashSpec.Env())
	tiflashContainer.EnvFrom = baseTiFlashSpec.EnvFrom()
	podSpec.Volumes = append(vols, baseTiFlashSpec.AdditionalVolumes()...)
	podSpec.SecurityContext = podSecurityContext
	podSpec.InitContainers = append(initContainers, baseTiFlashSpec.InitContainers()...)
	containers, err := buildTiFlashSidecarContainers(tc)
	if err != nil {
		return nil, err
	}

	podSpec.Containers = append(podSpec.Containers, tiflashContainer)

	if tc.Spec.TiFlash.LogTailer != nil && tc.Spec.TiFlash.LogTailer.UseSidecar {
		podSpec.InitContainers = append(podSpec.InitContainers, containers...)
	} else {
		podSpec.Containers = append(podSpec.Containers, containers...)
	}

	podSpec.Containers, err = MergePatchContainers(podSpec.Containers, baseTiFlashSpec.AdditionalContainers())
	if err != nil {
		return nil, fmt.Errorf("failed to merge containers spec for TiFlash of [%s/%s], err: %v", ns, tcName, err)
	}

	podSpec.ServiceAccountName = tc.Spec.TiFlash.ServiceAccount
	if podSpec.ServiceAccountName == "" {
		podSpec.ServiceAccountName = tc.Spec.ServiceAccount
	}

	updateStrategy := apps.StatefulSetUpdateStrategy{}
	if baseTiFlashSpec.StatefulSetUpdateStrategy() == apps.OnDeleteStatefulSetStrategyType {
		updateStrategy.Type = apps.OnDeleteStatefulSetStrategyType
	} else {
		updateStrategy.Type = apps.RollingUpdateStatefulSetStrategyType
		updateStrategy.RollingUpdate = &apps.RollingUpdateStatefulSetStrategy{
			Partition: pointer.Int32Ptr(tc.TiFlashStsDesiredReplicas() + deleteSlotsNumber),
		}
	}

	tiflashset := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            setName,
			Namespace:       ns,
			Labels:          stsLabels.Labels(),
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(tc.TiFlashStsDesiredReplicas()),
			Selector: stsLabels.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
			VolumeClaimTemplates: pvcs,
			ServiceName:          headlessSvcName,
			PodManagementPolicy:  baseTiFlashSpec.PodManagementPolicy(),
			UpdateStrategy:       updateStrategy,
		},
	}
	return tiflashset, nil
}

func flashVolumeClaimTemplate(storageClaims []v1alpha1.StorageClaim) ([]corev1.PersistentVolumeClaim, error) {
	var pvcs []corev1.PersistentVolumeClaim
	for k := range storageClaims {
		storageRequest, err := controller.ParseStorageRequest(storageClaims[k].Resources.Requests)
		if err != nil {
			return nil, err
		}
		pvcs = append(pvcs, corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: string(v1alpha1.GetStorageVolumeNameForTiFlash(k))},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				StorageClassName: storageClaims[k].StorageClassName,
				Resources:        storageRequest,
			},
		})
	}
	return pvcs, nil
}

func getTiFlashConfigMap(tc *v1alpha1.TidbCluster) (*corev1.ConfigMap, error) {
	config := GetTiFlashConfig(tc)

	configText, err := config.Common.MarshalTOML()
	if err != nil {
		return nil, err
	}
	proxyText, err := config.Proxy.MarshalTOML()
	if err != nil {
		return nil, err
	}

	instanceName := tc.GetInstanceName()
	tiflashLabel := label.New().Instance(instanceName).TiFlash().Labels()
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            controller.TiFlashMemberName(tc.Name),
			Namespace:       tc.Namespace,
			Labels:          tiflashLabel,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Data: map[string]string{
			"config_templ.toml": string(configText),
			"proxy_templ.toml":  string(proxyText),
		},
	}

	return cm, nil
}

func labelTiFlash(tc *v1alpha1.TidbCluster) label.Label {
	instanceName := tc.GetInstanceName()
	return label.New().Instance(instanceName).TiFlash()
}

func (m *tiflashMemberManager) syncTidbClusterStatus(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) error {
	if set == nil {
		// skip if not created yet
		return nil
	}
	tc.Status.TiFlash.StatefulSet = &set.Status
	upgrading, err := m.statefulSetIsUpgradingFn(m.deps.PodLister, m.deps.PDControl, set, tc)
	if err != nil {
		return err
	}
	if tc.TiFlashStsDesiredReplicas() != *set.Spec.Replicas {
		tc.Status.TiFlash.Phase = v1alpha1.ScalePhase
	} else if upgrading {
		tc.Status.TiFlash.Phase = v1alpha1.UpgradePhase
	} else {
		tc.Status.TiFlash.Phase = v1alpha1.NormalPhase
	}

	previousStores := tc.Status.TiFlash.Stores
	previousPeerStores := tc.Status.TiFlash.PeerStores
	previousTombstoneStores := tc.Status.TiFlash.TombstoneStores
	stores := map[string]v1alpha1.TiKVStore{}
	peerStores := map[string]v1alpha1.TiKVStore{}
	tombstoneStores := map[string]v1alpha1.TiKVStore{}

	pdCli := controller.GetPDClient(m.deps.PDControl, tc)
	// This only returns Up/Down/Offline stores
	storesInfo, err := pdCli.GetStores()
	if err != nil {
		tc.Status.TiFlash.Synced = false
		klog.Warningf("Fail to GetStores for TidbCluster %s/%s: %s", tc.Namespace, tc.Name, err)
		return err
	}

	pattern, err := regexp.Compile(fmt.Sprintf(tiflashStoreLimitPattern, tc.Name, tc.Name, tc.Namespace, controller.FormatClusterDomainForRegex(tc.Spec.ClusterDomain)))
	if err != nil {
		return err
	}
	for _, store := range storesInfo.Stores {
		status := m.getTiFlashStore(store)
		if status == nil {
			continue
		}

		oldStore, exist := previousStores[status.ID]
		if !exist {
			oldStore, exist = previousPeerStores[status.ID]
		}

		status.LastTransitionTime = metav1.Now()
		if exist && status.State == oldStore.State {
			status.LastTransitionTime = oldStore.LastTransitionTime
		}

		if store.Store != nil {
			if pattern.Match([]byte(store.Store.Address)) {
				stores[status.ID] = *status
			} else if util.MatchLabelFromStoreLabels(store.Store.Labels, label.TiFlashLabelVal) {
				peerStores[status.ID] = *status
			}
		}
	}

	// this returns all tombstone stores
	tombstoneStoresInfo, err := pdCli.GetTombStoneStores()
	if err != nil {
		tc.Status.TiFlash.Synced = false
		klog.Warningf("Fail to GetTombStoneStores for TidbCluster %s/%s", tc.Namespace, tc.Name)
		return err
	}
	for _, store := range tombstoneStoresInfo.Stores {
		if store.Store != nil && !pattern.Match([]byte(store.Store.Address)) {
			continue
		}
		status := m.getTiFlashStore(store)
		if status == nil {
			continue
		}

		oldStore, exist := previousTombstoneStores[status.ID]
		status.LastTransitionTime = metav1.Now()
		if exist && status.State == oldStore.State {
			status.LastTransitionTime = oldStore.LastTransitionTime
		}
		tombstoneStores[status.ID] = *status
	}

	tc.Status.TiFlash.Synced = true
	tc.Status.TiFlash.Stores = stores
	tc.Status.TiFlash.PeerStores = peerStores
	tc.Status.TiFlash.TombstoneStores = tombstoneStores
	tc.Status.TiFlash.Image = ""
	c := findContainerByName(set, "tiflash")
	if c != nil {
		tc.Status.TiFlash.Image = c.Image
	}

	err = volumes.SyncVolumeStatus(m.podVolumeModifier, m.deps.PodLister, tc, v1alpha1.TiFlashMemberType)
	if err != nil {
		return fmt.Errorf("failed to sync volume status for tiflash: %v", err)
	}

	return nil
}

func (m *tiflashMemberManager) getTiFlashStore(store *pdapi.StoreInfo) *v1alpha1.TiKVStore {
	if store.Store == nil || store.Status == nil {
		return nil
	}
	storeID := fmt.Sprintf("%d", store.Store.GetId())
	ip := strings.Split(store.Store.GetAddress(), ":")[0]
	podName := strings.Split(ip, ".")[0]

	return &v1alpha1.TiKVStore{
		ID:          storeID,
		PodName:     podName,
		IP:          ip,
		LeaderCount: int32(store.Status.LeaderCount),
		State:       store.Store.StateName,
	}
}

func (m *tiflashMemberManager) setStoreLabelsForTiFlash(tc *v1alpha1.TidbCluster) (int, error) {
	if m.deps.NodeLister == nil {
		klog.V(4).Infof("Node lister is unavailable, skip setting store labels for TiFlash of TiDB cluster %s/%s. This may be caused by no relevant permissions", tc.Namespace, tc.Name)
		return 0, nil
	}

	ns := tc.GetNamespace()
	// for unit test
	setCount := 0

	pdCli := controller.GetPDClient(m.deps.PDControl, tc)
	storesInfo, err := pdCli.GetStores()
	if err != nil {
		return setCount, err
	}

	config, err := pdCli.GetConfig()
	if err != nil {
		return setCount, err
	}

	locationLabels := []string(config.Replication.LocationLabels)
	if locationLabels == nil {
		return setCount, nil
	}

	pattern, err := regexp.Compile(fmt.Sprintf(tiflashStoreLimitPattern, tc.Name, tc.Name, tc.Namespace, controller.FormatClusterDomainForRegex(tc.Spec.ClusterDomain)))
	if err != nil {
		return -1, err
	}
	for _, store := range storesInfo.Stores {
		// In theory, the external tiflash can join the cluster, and the operator would only manage the internal tiflash.
		// So we check the store owner to make sure it.
		if store.Store != nil && !pattern.Match([]byte(store.Store.Address)) {
			continue
		}
		status := m.getTiFlashStore(store)
		if status == nil {
			continue
		}
		podName := status.PodName

		pod, err := m.deps.PodLister.Pods(ns).Get(podName)
		if err != nil {
			return setCount, fmt.Errorf("setStoreLabelsForTiFlash: failed to get pods %s for store %s, error: %v", podName, status.ID, err)
		}

		nodeName := pod.Spec.NodeName
		ls, err := getNodeLabels(m.deps.NodeLister, nodeName, locationLabels)
		if err != nil || len(ls) == 0 {
			klog.Warningf("node: [%s] has no node labels %v, skipping set store labels for Pod: [%s/%s]", nodeName, locationLabels, ns, podName)
			continue
		}

		if !m.storeLabelsEqualNodeLabels(store.Store.Labels, ls) {
			set, err := pdCli.SetStoreLabels(store.Store.Id, ls)
			if err != nil {
				klog.Warningf("failed to set pod: [%s/%s]'s store labels: %v", ns, podName, ls)
				continue
			}
			if set {
				setCount++
				klog.Infof("pod: [%s/%s] set labels: %v successfully", ns, podName, ls)
			}
		}
	}

	return setCount, nil
}

// storeLabelsEqualNodeLabels compares store labels with node labels
// for historic reasons, PD stores TiFlash labels as []*StoreLabel which is a key-value pair slice
func (m *tiflashMemberManager) storeLabelsEqualNodeLabels(storeLabels []*metapb.StoreLabel, nodeLabels map[string]string) bool {
	ls := map[string]string{}
	for _, label := range storeLabels {
		key := label.GetKey()
		if _, ok := nodeLabels[key]; ok {
			val := label.GetValue()
			ls[key] = val
		}
	}
	return reflect.DeepEqual(ls, nodeLabels)
}

func tiflashStatefulSetIsUpgrading(podLister corelisters.PodLister, pdControl pdapi.PDControlInterface, set *apps.StatefulSet, tc *v1alpha1.TidbCluster) (bool, error) {
	if mngerutils.StatefulSetIsUpgrading(set) {
		return true, nil
	}
	instanceName := tc.GetInstanceName()
	selector, err := label.New().Instance(instanceName).TiFlash().Selector()
	if err != nil {
		return false, err
	}
	tiflashPods, err := podLister.Pods(tc.GetNamespace()).List(selector)
	if err != nil {
		return false, fmt.Errorf("tiflashStatefulSetIsUpgrading: failed to list pods for cluster %s/%s, selector %s, error: %v", tc.GetNamespace(), instanceName, selector, err)
	}
	for _, pod := range tiflashPods {
		revisionHash, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return false, nil
		}
		if revisionHash != tc.Status.TiFlash.StatefulSet.UpdateRevision {
			return true, nil
		}
	}

	return false, nil
}

type FakeTiFlashMemberManager struct {
	err error
}

func NewFakeTiFlashMemberManager() *FakeTiFlashMemberManager {
	return &FakeTiFlashMemberManager{}
}

func (m *FakeTiFlashMemberManager) SetSyncError(err error) {
	m.err = err
}

func (m *FakeTiFlashMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	if m.err != nil {
		return m.err
	}
	if len(tc.Status.TiFlash.Stores) != 0 {
		// simulate status update
		tc.Status.ClusterID = string(uuid.NewUUID())
	}
	return nil
}
