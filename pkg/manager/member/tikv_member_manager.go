// Copyright 2018 PingCAP, Inc.
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
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
)

const (
	// tikvDataVolumeMountPath is the mount path for tikv data volume
	tikvDataVolumeMountPath = "/var/lib/tikv"

	// tikvClusterCertPath is where the cert for inter-cluster communication stored (if any)
	tikvClusterCertPath = "/var/lib/tikv-tls"

	//find a better way to manage store only managed by tikv in Operator
	tikvStoreLimitPattern = `%s-tikv-\d+\.%s-tikv-peer\.%s\.svc%s\:\d+`
)

// tikvMemberManager implements manager.Manager.
type tikvMemberManager struct {
	deps                     *controller.Dependencies
	failover                 Failover
	scaler                   Scaler
	upgrader                 TiKVUpgrader
	statefulSetIsUpgradingFn func(corelisters.PodLister, pdapi.PDControlInterface, *apps.StatefulSet, *v1alpha1.TidbCluster) (bool, error)
}

// NewTiKVMemberManager returns a *tikvMemberManager
func NewTiKVMemberManager(deps *controller.Dependencies, failover Failover, scaler Scaler, upgrader TiKVUpgrader) manager.Manager {
	m := &tikvMemberManager{
		deps:     deps,
		failover: failover,
		scaler:   scaler,
		upgrader: upgrader,
	}
	m.statefulSetIsUpgradingFn = tikvStatefulSetIsUpgrading
	return m
}

// SvcConfig corresponds to a K8s service
type SvcConfig struct {
	Name       string
	Port       int32
	SvcLabel   func(label.Label) label.Label
	MemberName func(clusterName string) string
	Headless   bool
}

// Sync fulfills the manager.Manager interface
func (m *tikvMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	// If tikv is not specified return
	if tc.Spec.TiKV == nil {
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if tc.Spec.PD != nil && !tc.PDIsAvailable() {
		return controller.RequeueErrorf("TidbCluster: [%s/%s], waiting for PD cluster running", ns, tcName)
	}

	svcList := []SvcConfig{
		{
			Name:       "peer",
			Port:       20160,
			Headless:   true,
			SvcLabel:   func(l label.Label) label.Label { return l.TiKV() },
			MemberName: controller.TiKVPeerMemberName,
		},
	}
	for _, svc := range svcList {
		if err := m.syncServiceForTidbCluster(tc, svc); err != nil {
			return err
		}
	}
	return m.syncStatefulSetForTidbCluster(tc)
}

func (m *tikvMemberManager) syncServiceForTidbCluster(tc *v1alpha1.TidbCluster, svcConfig SvcConfig) error {
	if tc.Spec.Paused {
		klog.V(4).Infof("tikv cluster %s/%s is paused, skip syncing for tikv service", tc.GetNamespace(), tc.GetName())
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := getNewServiceForTidbCluster(tc, svcConfig)
	oldSvcTmp, err := m.deps.ServiceLister.Services(ns).Get(svcConfig.MemberName(tcName))
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return m.deps.ServiceControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return fmt.Errorf("syncServiceForTidbCluster: failed to get svc %s for cluster %s/%s, error: %s", svcConfig.MemberName(tcName), ns, tcName, err)
	}

	oldSvc := oldSvcTmp.DeepCopy()

	equal, err := controller.ServiceEqual(newSvc, oldSvc)
	if err != nil {
		return err
	}
	if !equal {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		// TODO add unit test
		err = controller.SetServiceLastAppliedConfigAnnotation(&svc)
		if err != nil {
			return err
		}
		svc.Spec.ClusterIP = oldSvc.Spec.ClusterIP
		_, err = m.deps.ServiceControl.UpdateService(tc, &svc)
		return err
	}

	return nil
}

func (m *tikvMemberManager) syncStatefulSetForTidbCluster(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	oldSetTmp, err := m.deps.StatefulSetLister.StatefulSets(ns).Get(controller.TiKVMemberName(tcName))
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncStatefulSetForTidbCluster: failed to get sts %s for cluster %s/%s, error: %s", controller.TiKVMemberName(tcName), ns, tcName, err)
	}
	setNotExist := errors.IsNotFound(err)

	oldSet := oldSetTmp.DeepCopy()

	if err := m.syncTidbClusterStatus(tc, oldSet); err != nil {
		return err
	}

	if tc.Spec.Paused {
		klog.V(4).Infof("tikv cluster %s/%s is paused, skip syncing for tikv statefulset", tc.GetNamespace(), tc.GetName())
		return nil
	}

	cm, err := m.syncTiKVConfigMap(tc, oldSet)
	if err != nil {
		return err
	}

	// Recover failed stores if any before generating desired statefulset
	if len(tc.Status.TiKV.FailureStores) > 0 {
		m.failover.RemoveUndesiredFailures(tc)
	}
	if len(tc.Status.TiKV.FailureStores) > 0 &&
		tc.Spec.TiKV.RecoverFailover &&
		shouldRecover(tc, label.TiKVLabelVal, m.deps.PodLister) {
		m.failover.Recover(tc)
	}

	newSet, err := getNewTiKVSetForTidbCluster(tc, cm)
	if err != nil {
		return err
	}
	if setNotExist {
		err = SetStatefulSetLastAppliedConfigAnnotation(newSet)
		if err != nil {
			return err
		}
		err = m.deps.StatefulSetControl.CreateStatefulSet(tc, newSet)
		if err != nil {
			return err
		}
		tc.Status.TiKV.StatefulSet = &apps.StatefulSetStatus{}
		return nil
	}

	if _, err := m.setStoreLabelsForTiKV(tc); err != nil {
		return err
	}

	// Scaling takes precedence over upgrading because:
	// - if a store fails in the upgrading, users may want to delete it or add
	//   new replicas
	// - it's ok to scale in the middle of upgrading (in statefulset controller
	//   scaling takes precedence over upgrading too)
	if err := m.scaler.Scale(tc, oldSet, newSet); err != nil {
		return err
	}

	// Perform failover logic if necessary. Note that this will only update
	// TidbCluster status. The actual scaling performs in next sync loop (if a
	// new replica needs to be added).
	if m.deps.CLIConfig.AutoFailover && tc.Spec.TiKV.MaxFailoverCount != nil {
		if tc.TiKVAllPodsStarted() && !tc.TiKVAllStoresReady() {
			if err := m.failover.Failover(tc); err != nil {
				return err
			}
		}
	}

	if !templateEqual(newSet, oldSet) || tc.Status.TiKV.Phase == v1alpha1.UpgradePhase {
		if err := m.upgrader.Upgrade(tc, oldSet, newSet); err != nil {
			return err
		}
	}

	return UpdateStatefulSet(m.deps.StatefulSetControl, tc, newSet, oldSet)
}

func (m *tikvMemberManager) syncTiKVConfigMap(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) (*corev1.ConfigMap, error) {
	// For backward compatibility, only sync tidb configmap when .tikv.config is non-nil
	if tc.Spec.TiKV.Config == nil {
		return nil, nil
	}
	newCm, err := getTikVConfigMap(tc)
	if err != nil {
		return nil, err
	}

	var inUseName string
	if set != nil {
		inUseName = FindConfigMapVolume(&set.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, controller.TiKVMemberName(tc.Name))
		})
	}

	err = updateConfigMapIfNeed(m.deps.ConfigMapLister, tc.BaseTiKVSpec().ConfigUpdateStrategy(), inUseName, newCm)
	if err != nil {
		return nil, err
	}
	return m.deps.TypedControl.CreateOrUpdateConfigMap(tc, newCm)
}

func getNewServiceForTidbCluster(tc *v1alpha1.TidbCluster, svcConfig SvcConfig) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	instanceName := tc.GetInstanceName()
	svcName := svcConfig.MemberName(tcName)
	svcSelector := svcConfig.SvcLabel(label.New().Instance(instanceName))
	svcLabel := svcSelector.Copy()
	if svcConfig.Headless {
		svcLabel = svcLabel.UsedByPeer()
	} else {
		svcLabel = svcLabel.UsedByEndUser()
	}

	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          svcLabel.Labels(),
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       svcConfig.Name,
					Port:       svcConfig.Port,
					TargetPort: intstr.FromInt(int(svcConfig.Port)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector:                 svcSelector.Labels(),
			PublishNotReadyAddresses: true,
		},
	}
	if svcConfig.Headless {
		svc.Spec.ClusterIP = "None"
	} else {
		svc.Spec.Type = controller.GetServiceType(tc.Spec.Services, v1alpha1.TiKVMemberType.String())
	}
	return &svc
}

func getNewTiKVSetForTidbCluster(tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	baseTiKVSpec := tc.BaseTiKVSpec()

	tikvConfigMap := controller.MemberConfigMapName(tc, v1alpha1.TiKVMemberType)
	if cm != nil {
		tikvConfigMap = cm.Name
	}

	annoMount, annoVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		annoMount,
		{Name: v1alpha1.TiKVMemberType.String(), MountPath: tikvDataVolumeMountPath},
		{Name: "config", ReadOnly: true, MountPath: "/etc/tikv"},
		{Name: "startup-script", ReadOnly: true, MountPath: "/usr/local/bin"},
	}
	volMounts = append(volMounts, tc.Spec.TiKV.AdditionalVolumeMounts...)
	if tc.IsTLSClusterEnabled() {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "tikv-tls", ReadOnly: true, MountPath: "/var/lib/tikv-tls",
		})
		if tc.Spec.TiKV.MountClusterClientSecret != nil && *tc.Spec.TiKV.MountClusterClientSecret {
			volMounts = append(volMounts, corev1.VolumeMount{
				Name: util.ClusterClientVolName, ReadOnly: true, MountPath: util.ClusterClientTLSPath,
			})
		}
	}

	vols := []corev1.Volume{
		annoVolume,
		{Name: "config", VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: tikvConfigMap,
				},
				Items: []corev1.KeyToPath{{Key: "config-file", Path: "tikv.toml"}},
			}},
		},
		{Name: "startup-script", VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: tikvConfigMap,
				},
				Items: []corev1.KeyToPath{{Key: "startup-script", Path: "tikv_start_script.sh"}},
			}},
		},
	}
	if tc.IsTLSClusterEnabled() {
		vols = append(vols, corev1.Volume{
			Name: "tikv-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterTLSSecretName(tc.Name, label.TiKVLabelVal),
				},
			},
		})
		if tc.Spec.TiKV.MountClusterClientSecret != nil && *tc.Spec.TiKV.MountClusterClientSecret {
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
	storageVolMounts, additionalPVCs := util.BuildStorageVolumeAndVolumeMount(tc.Spec.TiKV.StorageVolumes, tc.Spec.TiKV.StorageClassName, v1alpha1.TiKVMemberType)
	volMounts = append(volMounts, storageVolMounts...)

	sysctls := "sysctl -w"
	var initContainers []corev1.Container
	if baseTiKVSpec.Annotations() != nil {
		init, ok := baseTiKVSpec.Annotations()[label.AnnSysctlInit]
		if ok && (init == label.AnnSysctlInitVal) {
			if baseTiKVSpec.PodSecurityContext() != nil && len(baseTiKVSpec.PodSecurityContext().Sysctls) > 0 {
				for _, sysctl := range baseTiKVSpec.PodSecurityContext().Sysctls {
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
					Resources: controller.ContainerResource(tc.Spec.TiKV.ResourceRequirements),
				})
			}
		}
	}
	// Init container is only used for the case where allowed-unsafe-sysctls
	// cannot be enabled for kubelet, so clean the sysctl in statefulset
	// SecurityContext if init container is enabled
	podSecurityContext := baseTiKVSpec.PodSecurityContext().DeepCopy()
	if len(initContainers) > 0 {
		podSecurityContext.Sysctls = []corev1.Sysctl{}
	}

	storageRequest, err := controller.ParseStorageRequest(tc.Spec.TiKV.Requests)
	if err != nil {
		return nil, fmt.Errorf("cannot parse storage request for tikv, tidbcluster %s/%s, error: %v", tc.Namespace, tc.Name, err)
	}

	tikvLabel := labelTiKV(tc)
	setName := controller.TiKVMemberName(tcName)
	podAnnotations := CombineAnnotations(controller.AnnProm(20180), baseTiKVSpec.Annotations())
	stsAnnotations := getStsAnnotations(tc.Annotations, label.TiKVLabelVal)
	capacity := controller.TiKVCapacity(tc.Spec.TiKV.Limits)
	headlessSvcName := controller.TiKVPeerMemberName(tcName)

	deleteSlotsNumber, err := util.GetDeleteSlotsNumber(stsAnnotations)
	if err != nil {
		return nil, fmt.Errorf("get delete slots number failed, err:%v", err)
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
			Value: tc.Spec.Timezone,
		},
	}
	tikvContainer := corev1.Container{
		Name:            v1alpha1.TiKVMemberType.String(),
		Image:           tc.TiKVImage(),
		ImagePullPolicy: baseTiKVSpec.ImagePullPolicy(),
		Command:         []string{"/bin/sh", "/usr/local/bin/tikv_start_script.sh"},
		SecurityContext: &corev1.SecurityContext{
			Privileged: tc.TiKVContainerPrivilege(),
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "server",
				ContainerPort: int32(20160),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: volMounts,
		Resources:    controller.ContainerResource(tc.Spec.TiKV.ResourceRequirements),
	}
	podSpec := baseTiKVSpec.BuildPodSpec()
	if baseTiKVSpec.HostNetwork() {
		podSpec.DNSPolicy = corev1.DNSClusterFirstWithHostNet
		env = append(env, corev1.EnvVar{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		})
	}
	tikvContainer.Env = util.AppendEnv(env, baseTiKVSpec.Env())
	podSpec.Volumes = append(vols, baseTiKVSpec.AdditionalVolumes()...)
	podSpec.SecurityContext = podSecurityContext
	podSpec.InitContainers = initContainers
	podSpec.Containers = append([]corev1.Container{tikvContainer}, baseTiKVSpec.AdditionalContainers()...)
	podSpec.ServiceAccountName = tc.Spec.TiKV.ServiceAccount
	if podSpec.ServiceAccountName == "" {
		podSpec.ServiceAccountName = tc.Spec.ServiceAccount
	}

	updateStrategy := apps.StatefulSetUpdateStrategy{}
	if baseTiKVSpec.StatefulSetUpdateStrategy() == apps.OnDeleteStatefulSetStrategyType {
		updateStrategy.Type = apps.OnDeleteStatefulSetStrategyType
	} else {
		updateStrategy.Type = apps.RollingUpdateStatefulSetStrategyType
		updateStrategy.RollingUpdate = &apps.RollingUpdateStatefulSetStrategy{
			Partition: pointer.Int32Ptr(tc.TiKVStsDesiredReplicas() + deleteSlotsNumber),
		}
	}

	tikvset := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            setName,
			Namespace:       ns,
			Labels:          tikvLabel.Labels(),
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(tc.TiKVStsDesiredReplicas()),
			Selector: tikvLabel.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      tikvLabel.Labels(),
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				util.VolumeClaimTemplate(storageRequest, v1alpha1.TiKVMemberType.String(), tc.Spec.TiKV.StorageClassName),
			},
			ServiceName:         headlessSvcName,
			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy:      updateStrategy,
		},
	}

	tikvset.Spec.VolumeClaimTemplates = append(tikvset.Spec.VolumeClaimTemplates, additionalPVCs...)
	return tikvset, nil
}

// transformTiKVConfigMap change the `wait-for-lock-timeout` and `wake-up-delay-duration` due to their content type.
// If either of their content is numeric, it would be rendered as numeric in toml in the tikv configmap.
// In https://github.com/tikv/tikv/pull/7197 , these 2 configurations become string type from int32 type, so we add
// this transforming steps to make tikv config compatible with both 4.0.0 version or under 4.0.0 version
func transformTiKVConfigMap(srcStr string, tc *v1alpha1.TidbCluster) string {
	config := tc.Spec.TiKV.Config
	if config == nil {
		return srcStr
	}

	if v := config.Get("pessimistic-txn.wait-for-lock-timeout"); v != nil {
		if str, err := v.AsString(); err == nil {
			_, err := strconv.ParseInt(str, 10, 64)
			if err == nil {
				waitForLockTimeOutKey := "wait-for-lock-timeout"
				old := fmt.Sprintf(`%s = "%s"`, waitForLockTimeOutKey, str)
				newString := fmt.Sprintf(`%s = %s`, waitForLockTimeOutKey, str)
				srcStr = strings.ReplaceAll(srcStr, old, newString)
			}
		} else {
			klog.Warningf("pessimistic-txn.wait-for-lock-timeout is not string type: %v", err)
		}
	}

	if v := config.Get("pessimistic-txn.wake-up-delay-duration"); v != nil {
		if str, err := v.AsString(); err == nil {
			_, err := strconv.ParseInt(str, 10, 64)
			if err == nil {
				wakeUpDelayDuration := "wake-up-delay-duration"
				old := fmt.Sprintf(`%s = "%s"`, wakeUpDelayDuration, str)
				newString := fmt.Sprintf(`%s = %s`, wakeUpDelayDuration, str)
				srcStr = strings.ReplaceAll(srcStr, old, newString)
			}
		} else {
			klog.Warningf("pessimistic-txn.wake-up-delay-duration is not string type: %v", err)
		}
	}

	return srcStr
}

func getTikVConfigMap(tc *v1alpha1.TidbCluster) (*corev1.ConfigMap, error) {
	config := tc.Spec.TiKV.Config
	if config == nil {
		return nil, nil
	}

	scriptModel := &TiKVStartScriptModel{
		EnableAdvertiseStatusAddr: false,
		DataDir:                   filepath.Join(tikvDataVolumeMountPath, tc.Spec.TiKV.DataSubDir),
		ClusterDomain:             tc.Spec.ClusterDomain,
	}
	if tc.Spec.EnableDynamicConfiguration != nil && *tc.Spec.EnableDynamicConfiguration {
		scriptModel.AdvertiseStatusAddr = "${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc" + controller.FormatClusterDomain(tc.Spec.ClusterDomain)
		scriptModel.EnableAdvertiseStatusAddr = true
	}

	if tc.IsHeterogeneous() {
		scriptModel.PDAddress = tc.Scheme() + "://" + controller.PDMemberName(tc.Spec.Cluster.Name) + ":2379"
	} else {
		scriptModel.PDAddress = tc.Scheme() + "://${CLUSTER_NAME}-pd:2379"
	}
	cm, err := getTikVConfigMapForTiKVSpec(tc.Spec.TiKV, tc, scriptModel)
	if err != nil {
		return nil, err
	}
	instanceName := tc.GetInstanceName()
	tikvLabel := label.New().Instance(instanceName).TiKV().Labels()
	cm.ObjectMeta = metav1.ObjectMeta{
		Name:            controller.TiKVMemberName(tc.Name),
		Namespace:       tc.Namespace,
		Labels:          tikvLabel,
		OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
	}

	return cm, nil
}

func labelTiKV(tc *v1alpha1.TidbCluster) label.Label {
	instanceName := tc.GetInstanceName()
	return label.New().Instance(instanceName).TiKV()
}

func (m *tikvMemberManager) syncTidbClusterStatus(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) error {
	if set == nil {
		// skip if not created yet
		return nil
	}
	tc.Status.TiKV.StatefulSet = &set.Status
	upgrading, err := m.statefulSetIsUpgradingFn(m.deps.PodLister, m.deps.PDControl, set, tc)
	if err != nil {
		return err
	}
	// Scaling takes precedence over upgrading.
	if tc.TiKVStsDesiredReplicas() != *set.Spec.Replicas {
		tc.Status.TiKV.Phase = v1alpha1.ScalePhase
	} else if upgrading && tc.Status.PD.Phase != v1alpha1.UpgradePhase {
		tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
	} else {
		tc.Status.TiKV.Phase = v1alpha1.NormalPhase
	}

	previousStores := tc.Status.TiKV.Stores
	previousPeerStores := tc.Status.TiKV.PeerStores
	stores := map[string]v1alpha1.TiKVStore{}
	peerStores := map[string]v1alpha1.TiKVStore{}
	tombstoneStores := map[string]v1alpha1.TiKVStore{}

	pdCli := controller.GetPDClient(m.deps.PDControl, tc)
	// This only returns Up/Down/Offline stores
	storesInfo, err := pdCli.GetStores()
	if err != nil {
		if pdapi.IsTiKVNotBootstrappedError(err) {
			klog.Infof("TiKV of Cluster %s/%s not bootstrapped yet", tc.Namespace, tc.Name)
			tc.Status.TiKV.Synced = true
			tc.Status.TiKV.BootStrapped = false
			return nil
		}
		tc.Status.TiKV.Synced = false
		return err
	}

	pattern, err := regexp.Compile(fmt.Sprintf(tikvStoreLimitPattern, tc.Name, tc.Name, tc.Namespace, controller.FormatClusterDomainForRegex(tc.Spec.ClusterDomain)))
	if err != nil {
		return err
	}
	for _, store := range storesInfo.Stores {
		status := getTiKVStore(store)
		if status == nil {
			continue
		}

		oldStore, exist := previousStores[status.ID]
		if !exist {
			oldStore, exist = previousPeerStores[status.ID]
		}

		// avoid LastHeartbeatTime be overwrite by zero time when pd lost LastHeartbeatTime
		if status.LastHeartbeatTime.IsZero() && exist {
			klog.V(4).Infof("the pod:%s's store LastHeartbeatTime is zero,so will keep in %v", status.PodName, oldStore.LastHeartbeatTime)
			status.LastHeartbeatTime = oldStore.LastHeartbeatTime
		}

		status.LastTransitionTime = metav1.Now()
		if exist && status.State == oldStore.State {
			status.LastTransitionTime = oldStore.LastTransitionTime
		}

		// In theory, the external tikv can join the cluster, and the operator would only manage the internal tikv.
		// So we check the store owner to make sure it.
		if store.Store != nil {
			if pattern.Match([]byte(store.Store.Address)) {
				stores[status.ID] = *status
			} else if util.MatchLabelFromStoreLabels(store.Store.Labels, label.TiKVLabelVal) {
				peerStores[status.ID] = *status
			}
		}
	}

	//this returns all tombstone stores
	tombstoneStoresInfo, err := pdCli.GetTombStoneStores()
	if err != nil {
		tc.Status.TiKV.Synced = false
		return err
	}
	for _, store := range tombstoneStoresInfo.Stores {
		if store.Store != nil && !pattern.Match([]byte(store.Store.Address)) {
			continue
		}
		status := getTiKVStore(store)
		if status == nil {
			continue
		}
		tombstoneStores[status.ID] = *status
	}

	tc.Status.TiKV.Synced = true
	tc.Status.TiKV.Stores = stores
	tc.Status.TiKV.PeerStores = peerStores
	tc.Status.TiKV.TombstoneStores = tombstoneStores
	tc.Status.TiKV.BootStrapped = true
	tc.Status.TiKV.Image = ""
	c := filterContainer(set, "tikv")
	if c != nil {
		tc.Status.TiKV.Image = c.Image
	}
	return nil
}

func getTiKVStore(store *pdapi.StoreInfo) *v1alpha1.TiKVStore {
	if store.Store == nil || store.Status == nil {
		return nil
	}
	storeID := fmt.Sprintf("%d", store.Store.GetId())
	ip := strings.Split(store.Store.GetAddress(), ":")[0]
	podName := strings.Split(ip, ".")[0]

	return &v1alpha1.TiKVStore{
		ID:                storeID,
		PodName:           podName,
		IP:                ip,
		LeaderCount:       int32(store.Status.LeaderCount),
		State:             store.Store.StateName,
		LastHeartbeatTime: metav1.Time{Time: store.Status.LastHeartbeatTS},
	}
}

func (m *tikvMemberManager) setStoreLabelsForTiKV(tc *v1alpha1.TidbCluster) (int, error) {
	ns := tc.GetNamespace()
	// for unit test
	setCount := 0

	if !tc.TiKVBootStrapped() {
		klog.Infof("TiKV of Cluster %s/%s is not bootstrapped yet, no need to set store labels", tc.Namespace, tc.Name)
		return setCount, nil
	}

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

	pattern, err := regexp.Compile(fmt.Sprintf(tikvStoreLimitPattern, tc.Name, tc.Name, tc.Namespace, controller.FormatClusterDomainForRegex(tc.Spec.ClusterDomain)))
	if err != nil {
		return -1, err
	}
	for _, store := range storesInfo.Stores {
		// In theory, the external tikv can join the cluster, and the operator would only manage the internal tikv.
		// So we check the store owner to make sure it.
		if store.Store != nil && !pattern.Match([]byte(store.Store.Address)) {
			continue
		}
		status := getTiKVStore(store)
		if status == nil {
			continue
		}
		podName := status.PodName

		pod, err := m.deps.PodLister.Pods(ns).Get(podName)
		if err != nil {
			return setCount, fmt.Errorf("setStoreLabelsForTiKV: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tc.GetName(), err)
		}

		nodeName := pod.Spec.NodeName
		ls, err := m.getNodeLabels(nodeName, locationLabels)
		if err != nil || len(ls) == 0 {
			klog.Warningf("node: [%s] has no node labels, skipping set store labels for Pod: [%s/%s]", nodeName, ns, podName)
			continue
		}

		if !m.storeLabelsEqualNodeLabels(store.Store.Labels, ls) {
			set, err := pdCli.SetStoreLabels(store.Store.Id, ls)
			if err != nil {
				msg := fmt.Sprintf("failed to set labels %v for store (id: %d, pod: %s/%s): %v ",
					ls, store.Store.Id, ns, podName, err)
				m.deps.Recorder.Event(tc, corev1.EventTypeWarning, FailedSetStoreLabels, msg)
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

func (m *tikvMemberManager) getNodeLabels(nodeName string, storeLabels []string) (map[string]string, error) {
	node, err := m.deps.NodeLister.Get(nodeName)
	if err != nil {
		return nil, err
	}
	labels := map[string]string{}
	ls := node.GetLabels()
	for _, storeLabel := range storeLabels {
		if value, found := ls[storeLabel]; found {
			labels[storeLabel] = value
			continue
		}

		// TODO after pd supports storeLabel containing slash character, these codes should be deleted
		if storeLabel == "host" {
			if host, found := ls[corev1.LabelHostname]; found {
				labels[storeLabel] = host
			}
		}

	}
	return labels, nil
}

// storeLabelsEqualNodeLabels compares store labels with node labels
// for historic reasons, PD stores TiKV labels as []*StoreLabel which is a key-value pair slice
func (m *tikvMemberManager) storeLabelsEqualNodeLabels(storeLabels []*metapb.StoreLabel, nodeLabels map[string]string) bool {
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

func tikvStatefulSetIsUpgrading(podLister corelisters.PodLister, pdControl pdapi.PDControlInterface, set *apps.StatefulSet, tc *v1alpha1.TidbCluster) (bool, error) {
	if statefulSetIsUpgrading(set) {
		return true, nil
	}
	instanceName := tc.GetInstanceName()
	selector, err := label.New().Instance(instanceName).TiKV().Selector()
	if err != nil {
		return false, err
	}
	tikvPods, err := podLister.Pods(tc.GetNamespace()).List(selector)
	if err != nil {
		return false, fmt.Errorf("tikvStatefulSetIsUpgrading: failed to get pods for cluster %s/%s, selector %s, error: %s", tc.GetNamespace(), instanceName, selector, err)
	}
	for _, pod := range tikvPods {
		revisionHash, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return false, nil
		}
		if revisionHash != tc.Status.TiKV.StatefulSet.UpdateRevision {
			return true, nil
		}
	}

	return false, nil
}

type FakeTiKVMemberManager struct {
	err error
}

func NewFakeTiKVMemberManager() *FakeTiKVMemberManager {
	return &FakeTiKVMemberManager{}
}

func (m *FakeTiKVMemberManager) SetSyncError(err error) {
	m.err = err
}

func (m *FakeTiKVMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	if m.err != nil {
		return m.err
	}
	if len(tc.Status.TiKV.Stores) != 0 {
		// simulate status update
		tc.Status.ClusterID = string(uuid.NewUUID())
	}
	return nil
}
