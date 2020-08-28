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
	"fmt"
	"reflect"
	"regexp"
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
	v1 "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
)

const (
	//find a better way to manage store only managed by tiflash in Operator
	tiflashStoreLimitPattern = `%s-tiflash-\d+\.%s-tiflash-peer\.%s\.svc\:\d+`
	tiflashCertPath          = "/var/lib/tiflash-tls"
	tiflashCertVolumeName    = "tiflash-tls"
)

// tiflashMemberManager implements manager.Manager.
type tiflashMemberManager struct {
	setControl                      controller.StatefulSetControlInterface
	svcControl                      controller.ServiceControlInterface
	pdControl                       pdapi.PDControlInterface
	typedControl                    controller.TypedControlInterface
	setLister                       v1.StatefulSetLister
	svcLister                       corelisters.ServiceLister
	podLister                       corelisters.PodLister
	nodeLister                      corelisters.NodeLister
	autoFailover                    bool
	tiflashFailover                 Failover
	tiflashScaler                   Scaler
	tiflashUpgrader                 Upgrader
	tiflashStatefulSetIsUpgradingFn func(corelisters.PodLister, pdapi.PDControlInterface, *apps.StatefulSet, *v1alpha1.TidbCluster) (bool, error)
}

// NewTiFlashMemberManager returns a *tiflashMemberManager
func NewTiFlashMemberManager(
	pdControl pdapi.PDControlInterface,
	setControl controller.StatefulSetControlInterface,
	svcControl controller.ServiceControlInterface,
	typedControl controller.TypedControlInterface,
	setLister v1.StatefulSetLister,
	svcLister corelisters.ServiceLister,
	podLister corelisters.PodLister,
	nodeLister corelisters.NodeLister,
	autoFailover bool,
	tiflashFailover Failover,
	tiflashScaler Scaler,
	tiflashUpgrader Upgrader) manager.Manager {
	kvmm := tiflashMemberManager{
		pdControl:       pdControl,
		podLister:       podLister,
		nodeLister:      nodeLister,
		setControl:      setControl,
		svcControl:      svcControl,
		typedControl:    typedControl,
		setLister:       setLister,
		svcLister:       svcLister,
		autoFailover:    autoFailover,
		tiflashFailover: tiflashFailover,
		tiflashScaler:   tiflashScaler,
		tiflashUpgrader: tiflashUpgrader,
	}
	kvmm.tiflashStatefulSetIsUpgradingFn = tiflashStatefulSetIsUpgrading
	return &kvmm
}

// Sync fulfills the manager.Manager interface
func (tfmm *tiflashMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	if tc.Spec.TiFlash == nil {
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if tc.Spec.PD != nil && !tc.PDIsAvailable() {
		return controller.RequeueErrorf("TidbCluster: [%s/%s], waiting for PD cluster running", ns, tcName)
	}

	err := tfmm.enablePlacementRules(tc)
	if err != nil {
		klog.Errorf("Enable placement rules failed, error: %v", err)
		// No need to return err here, just continue to sync tiflash
	}
	// Sync TiFlash Headless Service
	if err = tfmm.syncHeadlessService(tc); err != nil {
		return err
	}

	return tfmm.syncStatefulSet(tc)
}

func (tfmm *tiflashMemberManager) enablePlacementRules(tc *v1alpha1.TidbCluster) error {
	pdCli := controller.GetPDClient(tfmm.pdControl, tc)
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

func (tfmm *tiflashMemberManager) syncHeadlessService(tc *v1alpha1.TidbCluster) error {
	if tc.Spec.Paused {
		klog.V(4).Infof("tiflash cluster %s/%s is paused, skip syncing for tiflash service", tc.GetNamespace(), tc.GetName())
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := getNewHeadlessService(tc)
	oldSvcTmp, err := tfmm.svcLister.Services(ns).Get(controller.TiFlashPeerMemberName(tcName))
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return tfmm.svcControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return fmt.Errorf("syncHeadlessService: failed to get svc %s for cluster %s/%s, error: %s", controller.TiFlashPeerMemberName(tcName), ns, tcName, err)
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
		_, err = tfmm.svcControl.UpdateService(tc, &svc)
		return err
	}

	return nil
}

func (tfmm *tiflashMemberManager) syncStatefulSet(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	oldSetTmp, err := tfmm.setLister.StatefulSets(ns).Get(controller.TiFlashMemberName(tcName))
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncStatefulSet: fail to get sts %s for cluster %s/%s, error: %s", controller.TiFlashMemberName(tcName), ns, tcName, err)
	}
	setNotExist := errors.IsNotFound(err)

	oldSet := oldSetTmp.DeepCopy()

	if err := tfmm.syncTidbClusterStatus(tc, oldSet); err != nil {
		return err
	}

	if tc.Spec.Paused {
		klog.V(4).Infof("tiflash cluster %s/%s is paused, skip syncing for tiflash statefulset", tc.GetNamespace(), tc.GetName())
		return nil
	}

	cm, err := tfmm.syncConfigMap(tc, oldSet)
	if err != nil {
		return err
	}

	// Recover failed stores if any before generating desired statefulset
	if len(tc.Status.TiFlash.FailureStores) > 0 {
		tfmm.tiflashFailover.Recover(tc)
	}

	newSet, err := getNewStatefulSet(tc, cm)
	if err != nil {
		return err
	}
	if setNotExist {
		err = SetStatefulSetLastAppliedConfigAnnotation(newSet)
		if err != nil {
			return err
		}
		err = tfmm.setControl.CreateStatefulSet(tc, newSet)
		if err != nil {
			return err
		}
		tc.Status.TiFlash.StatefulSet = &apps.StatefulSetStatus{}
		return nil
	}

	if _, err := tfmm.setStoreLabelsForTiFlash(tc); err != nil {
		return err
	}

	if !templateEqual(newSet, oldSet) {
		if err := tfmm.tiflashUpgrader.Upgrade(tc, oldSet, newSet); err != nil {
			return err
		}
	}

	if err := tfmm.tiflashScaler.Scale(tc, oldSet, newSet); err != nil {
		return err
	}

	if tfmm.autoFailover && tc.Spec.TiFlash.MaxFailoverCount != nil {
		if tc.TiFlashAllPodsStarted() && !tc.TiFlashAllStoresReady() {
			if err := tfmm.tiflashFailover.Failover(tc); err != nil {
				return err
			}
		}
	}

	return updateStatefulSet(tfmm.setControl, tc, newSet, oldSet)
}

func (tfmm *tiflashMemberManager) syncConfigMap(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) (*corev1.ConfigMap, error) {
	newCm, err := getTiFlashConfigMap(tc)
	if err != nil {
		return nil, err
	}
	if set != nil && tc.BaseTiFlashSpec().ConfigUpdateStrategy() == v1alpha1.ConfigUpdateStrategyInPlace {
		inUseName := FindConfigMapVolume(&set.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, controller.TiFlashMemberName(tc.Name))
		})
		if inUseName != "" {
			newCm.Name = inUseName
		}
	}

	return tfmm.typedControl.CreateOrUpdateConfigMap(tc, newCm)
}

func getNewHeadlessService(tc *v1alpha1.TidbCluster) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	instanceName := tc.GetInstanceName()
	svcName := controller.TiFlashPeerMemberName(tcName)
	svcLabel := label.New().Instance(instanceName).TiFlash().Labels()

	svc := corev1.Service{
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
					Port:       3930,
					TargetPort: intstr.FromInt(int(3930)),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "proxy",
					Port:       20170,
					TargetPort: intstr.FromInt(int(20170)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector:                 svcLabel,
			PublishNotReadyAddresses: true,
		},
	}
	return &svc
}

func getNewStatefulSet(tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	baseTiFlashSpec := tc.BaseTiFlashSpec()
	spec := tc.Spec.TiFlash

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
	annMount, annVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		annMount,
	}
	for k := range spec.StorageClaims {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: fmt.Sprintf("data%d", k), MountPath: fmt.Sprintf("/data%d", k)})
	}

	if tc.IsTLSClusterEnabled() {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: tiflashCertVolumeName, ReadOnly: true, MountPath: tiflashCertPath,
		})
	}

	vols := []corev1.Volume{
		annVolume,
		{Name: "config", VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: tiflashConfigMap,
				},
			}},
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
	initContainers = append(initContainers, corev1.Container{
		Name:  "init",
		Image: tc.HelperImage(),
		Command: []string{
			"sh",
			"-c",
			"set -ex;ordinal=`echo ${POD_NAME} | awk -F- '{print $NF}'`;sed s/POD_NUM/${ordinal}/g /etc/tiflash/config_templ.toml > /data0/config.toml;sed s/POD_NUM/${ordinal}/g /etc/tiflash/proxy_templ.toml > /data0/proxy.toml",
		},
		Env:          initEnv,
		VolumeMounts: initVolMounts,
	})

	tiflashLabel := labelTiFlash(tc)
	setName := controller.TiFlashMemberName(tcName)
	podAnnotations := CombineAnnotations(controller.AnnProm(8234), baseTiFlashSpec.Annotations())
	podAnnotations = CombineAnnotations(controller.AnnAdditionalProm("tiflash.proxy", 20292), podAnnotations)
	stsAnnotations := getStsAnnotations(tc.Annotations, label.TiFlashLabelVal)
	capacity := controller.TiKVCapacity(tc.Spec.TiFlash.Limits)
	headlessSvcName := controller.TiFlashPeerMemberName(tcName)

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
	tiflashContainer := corev1.Container{
		Name:            v1alpha1.TiFlashMemberType.String(),
		Image:           tc.TiFlashImage(),
		ImagePullPolicy: baseTiFlashSpec.ImagePullPolicy(),
		Command:         []string{"/bin/sh", "-c", "/tiflash/tiflash server --config-file /data0/config.toml"},
		SecurityContext: &corev1.SecurityContext{
			Privileged: tc.TiFlashContainerPrivilege(),
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "tiflash",
				ContainerPort: int32(3930),
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "proxy",
				ContainerPort: int32(20170),
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "tcp",
				ContainerPort: int32(9000),
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "http",
				ContainerPort: int32(8123),
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "internal",
				ContainerPort: int32(9009),
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "metrics",
				ContainerPort: int32(8234),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: volMounts,
		Resources:    controller.ContainerResource(tc.Spec.TiFlash.ResourceRequirements),
	}
	podSpec := baseTiFlashSpec.BuildPodSpec()
	if baseTiFlashSpec.HostNetwork() {
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
	tiflashContainer.Env = util.AppendEnv(env, baseTiFlashSpec.Env())
	podSpec.Volumes = append(vols, baseTiFlashSpec.AdditionalVolumes()...)
	podSpec.SecurityContext = podSecurityContext
	podSpec.InitContainers = initContainers
	podSpec.Containers = append([]corev1.Container{tiflashContainer}, buildTiFlashSidecarContainers(tc)...)
	podSpec.Containers = append(podSpec.Containers, baseTiFlashSpec.AdditionalContainers()...)
	podSpec.ServiceAccountName = tc.Spec.TiFlash.ServiceAccount

	tiflashset := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            setName,
			Namespace:       ns,
			Labels:          tiflashLabel.Labels(),
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(tc.TiFlashStsDesiredReplicas()),
			Selector: tiflashLabel.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      tiflashLabel.Labels(),
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
			VolumeClaimTemplates: pvcs,
			ServiceName:          headlessSvcName,
			PodManagementPolicy:  apps.ParallelPodManagement,
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: apps.RollingUpdateStatefulSetStrategyType,
			},
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
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("data%d", k)},
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
	config := getTiFlashConfig(tc)

	configText, err := MarshalTOML(config.CommonConfig)
	if err != nil {
		return nil, err
	}
	proxyText, err := MarshalTOML(config.ProxyConfig)
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

	if tc.BaseTiFlashSpec().ConfigUpdateStrategy() == v1alpha1.ConfigUpdateStrategyRollingUpdate {
		if err := AddConfigMapDigestSuffix(cm); err != nil {
			return nil, err
		}
	}

	return cm, nil
}

func labelTiFlash(tc *v1alpha1.TidbCluster) label.Label {
	instanceName := tc.GetInstanceName()
	return label.New().Instance(instanceName).TiFlash()
}

func (tfmm *tiflashMemberManager) syncTidbClusterStatus(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) error {
	if set == nil {
		// skip if not created yet
		return nil
	}
	tc.Status.TiFlash.StatefulSet = &set.Status
	upgrading, err := tfmm.tiflashStatefulSetIsUpgradingFn(tfmm.podLister, tfmm.pdControl, set, tc)
	if err != nil {
		return err
	}
	if upgrading {
		tc.Status.TiFlash.Phase = v1alpha1.UpgradePhase
	} else {
		tc.Status.TiFlash.Phase = v1alpha1.NormalPhase
	}

	previousStores := tc.Status.TiFlash.Stores
	stores := map[string]v1alpha1.TiKVStore{}
	tombstoneStores := map[string]v1alpha1.TiKVStore{}

	pdCli := controller.GetPDClient(tfmm.pdControl, tc)
	// This only returns Up/Down/Offline stores
	storesInfo, err := pdCli.GetStores()
	if err != nil {
		tc.Status.TiFlash.Synced = false
		return err
	}

	pattern, err := regexp.Compile(fmt.Sprintf(tiflashStoreLimitPattern, tc.Name, tc.Name, tc.Namespace))
	if err != nil {
		return err
	}
	for _, store := range storesInfo.Stores {
		// In theory, the external tiflash can join the cluster, and the operator would only manage the internal tiflash.
		// So we check the store owner to make sure it.
		if store.Store != nil && !pattern.Match([]byte(store.Store.Address)) {
			continue
		}
		status := tfmm.getTiFlashStore(store)
		if status == nil {
			continue
		}
		// avoid LastHeartbeatTime be overwrite by zero time when pd lost LastHeartbeatTime
		if status.LastHeartbeatTime.IsZero() {
			if oldStatus, ok := previousStores[status.ID]; ok {
				klog.V(4).Infof("the pod:%s's store LastHeartbeatTime is zero,so will keep in %v", status.PodName, oldStatus.LastHeartbeatTime)
				status.LastHeartbeatTime = oldStatus.LastHeartbeatTime
			}
		}

		oldStore, exist := previousStores[status.ID]

		status.LastTransitionTime = metav1.Now()
		if exist && status.State == oldStore.State {
			status.LastTransitionTime = oldStore.LastTransitionTime
		}

		stores[status.ID] = *status
	}

	//this returns all tombstone stores
	tombstoneStoresInfo, err := pdCli.GetTombStoneStores()
	if err != nil {
		tc.Status.TiFlash.Synced = false
		return err
	}
	for _, store := range tombstoneStoresInfo.Stores {
		if store.Store != nil && !pattern.Match([]byte(store.Store.Address)) {
			continue
		}
		status := tfmm.getTiFlashStore(store)
		if status == nil {
			continue
		}
		tombstoneStores[status.ID] = *status
	}

	tc.Status.TiFlash.Synced = true
	tc.Status.TiFlash.Stores = stores
	tc.Status.TiFlash.TombstoneStores = tombstoneStores
	tc.Status.TiFlash.Image = ""
	c := filterContainer(set, "tiflash")
	if c != nil {
		tc.Status.TiFlash.Image = c.Image
	}
	return nil
}

func (tfmm *tiflashMemberManager) getTiFlashStore(store *pdapi.StoreInfo) *v1alpha1.TiKVStore {
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

func (tfmm *tiflashMemberManager) setStoreLabelsForTiFlash(tc *v1alpha1.TidbCluster) (int, error) {
	ns := tc.GetNamespace()
	// for unit test
	setCount := 0

	pdCli := controller.GetPDClient(tfmm.pdControl, tc)
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

	pattern, err := regexp.Compile(fmt.Sprintf(tiflashStoreLimitPattern, tc.Name, tc.Name, tc.Namespace))
	if err != nil {
		return -1, err
	}
	for _, store := range storesInfo.Stores {
		// In theory, the external tiflash can join the cluster, and the operator would only manage the internal tiflash.
		// So we check the store owner to make sure it.
		if store.Store != nil && !pattern.Match([]byte(store.Store.Address)) {
			continue
		}
		status := tfmm.getTiFlashStore(store)
		if status == nil {
			continue
		}
		podName := status.PodName

		pod, err := tfmm.podLister.Pods(ns).Get(podName)
		if err != nil {
			return setCount, fmt.Errorf("setStoreLabelsForTiFlash: failed to get pods %s for store %s, error: %v", podName, status.ID, err)
		}

		nodeName := pod.Spec.NodeName
		ls, err := tfmm.getNodeLabels(nodeName, locationLabels)
		if err != nil || len(ls) == 0 {
			klog.Warningf("node: [%s] has no node labels, skipping set store labels for Pod: [%s/%s]", nodeName, ns, podName)
			continue
		}

		if !tfmm.storeLabelsEqualNodeLabels(store.Store.Labels, ls) {
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

func (tfmm *tiflashMemberManager) getNodeLabels(nodeName string, storeLabels []string) (map[string]string, error) {
	node, err := tfmm.nodeLister.Get(nodeName)
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
// for historic reasons, PD stores TiFlash labels as []*StoreLabel which is a key-value pair slice
func (tfmm *tiflashMemberManager) storeLabelsEqualNodeLabels(storeLabels []*metapb.StoreLabel, nodeLabels map[string]string) bool {
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
	if statefulSetIsUpgrading(set) {
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

func (ftmm *FakeTiFlashMemberManager) SetSyncError(err error) {
	ftmm.err = err
}

func (ftmm *FakeTiFlashMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	if ftmm.err != nil {
		return ftmm.err
	}
	if len(tc.Status.TiFlash.Stores) != 0 {
		// simulate status update
		tc.Status.ClusterID = string(uuid.NewUUID())
	}
	return nil
}
