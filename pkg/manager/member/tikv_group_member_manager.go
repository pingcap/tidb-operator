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
	"regexp"
	"strings"

	"github.com/Masterminds/semver"
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
	appslister "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (

	//find a better way to manage store only managed by tikv in Operator
	tikvGroupStoreLimitPattern = `%s-tikv-group-\d+\.%s-tikv-group-peer\.%s\.svc\:\d+`
)

var (
	tikvV401Version *semver.Version
)

func init() {
	var err error
	tikvV401Version, err = semver.NewVersion("v4.0.1")
	if err != nil {
		klog.Fatal(err)
	}
}

type tikvGroupMemberManager struct {
	genericCli        client.Client
	svcLister         corelisters.ServiceLister
	setLister         appslister.StatefulSetLister
	podLister         corelisters.PodLister
	svcControl        controller.ServiceControlInterface
	setControl        controller.StatefulSetControlInterface
	typedControl      controller.TypedControlInterface
	tikvGroupScaler   Scaler
	tikvGroupUpgrader *TikvGroupUpgrader
	pdControl         pdapi.PDControlInterface
}

func NewTiKVGroupMemberManager(
	genericCli client.Client,
	svcLister corelisters.ServiceLister,
	setLister appslister.StatefulSetLister,
	podLister corelisters.PodLister,
	svcControl controller.ServiceControlInterface,
	setControl controller.StatefulSetControlInterface,
	typedControl controller.TypedControlInterface,
	tikvGroupScaler Scaler,
	tikvUpgrader *TikvGroupUpgrader,
	pdControl pdapi.PDControlInterface) manager.TiKVGroupManager {
	return &tikvGroupMemberManager{
		genericCli:        genericCli,
		svcLister:         svcLister,
		setLister:         setLister,
		podLister:         podLister,
		svcControl:        svcControl,
		setControl:        setControl,
		typedControl:      typedControl,
		tikvGroupScaler:   tikvGroupScaler,
		tikvGroupUpgrader: tikvUpgrader,
		pdControl:         pdControl,
	}
}

func (tgm *tikvGroupMemberManager) SyncTiKVGroup(tg *v1alpha1.TiKVGroup, tc *v1alpha1.TidbCluster) error {
	err := tgm.checkWhetherRegistered(tg, tc)
	if err != nil {
		klog.Error(err)
		return err
	}
	klog.V(4).Infof("tikvgroup member [%s/%s] allowed to syncing", tg.Namespace, tg.Name)

	if err := tgm.syncServiceForTiKVGroup(tg); err != nil {
		klog.Error(err)
		return err
	}

	if err := tgm.syncStatefulSetForTiKVGroup(tg, tc); err != nil {
		klog.Error(err)
		return err
	}

	return nil
}

// TODO: add unit test
// checkWhetherRegistered will check whether the tikvgroup have already registered itself
// to the target tidbcluster. If have already, the tikvgroup will be allowed to syncing.
// If not, the tikvgroup will try to register itself to the tidbcluster and wait for the next round.
func (tgm *tikvGroupMemberManager) checkWhetherRegistered(tg *v1alpha1.TiKVGroup, tc *v1alpha1.TidbCluster) error {
	if tc.Status.TiKVGroups == nil || len(tc.Status.TiKVGroups) < 1 {
		return tgm.registerTiKVGroup(tg, tc)
	}

	for _, tikvGroup := range tc.Status.TiKVGroups {
		// found tikvgroup in the tidbcluster's status, allowed to syncing
		if tikvGroup.Reference.Name == tg.Name {
			return nil
		}
	}

	klog.Infof("start to register tikvGroup[%s/%s] to tc[%s/%s]", tg.Namespace, tg.Name, tc.Namespace, tc.Name)
	return tgm.registerTiKVGroup(tg, tc)
}

// register itself to the target tidbcluster
func (tgm *tikvGroupMemberManager) registerTiKVGroup(tg *v1alpha1.TiKVGroup, tc *v1alpha1.TidbCluster) error {
	tcName := tg.Spec.ClusterName
	tcNamespace := tg.Namespace
	// register itself to the target tidbcluster
	newGroups := append(tc.Status.TiKVGroups, v1alpha1.GroupRef{Reference: corev1.LocalObjectReference{Name: tg.Name}})
	err := controller.GuaranteedUpdate(tgm.genericCli, tc, func() error {
		tc.Status.TiKVGroups = newGroups
		return nil
	})
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("tg[%s/%s] register itself to tc[%s/%s] successfully, requeue", tg.Namespace, tg.Name, tcNamespace, tcName)
	return controller.RequeueErrorf(msg)
}

func (tgm *tikvGroupMemberManager) syncServiceForTiKVGroup(tg *v1alpha1.TiKVGroup) error {
	//TODO: support Pause

	ns := tg.Namespace
	svcName := controller.TiKVGroupPeerMemberName(tg.Name)

	newSvc := newServiceForTiKVGroup(tg, svcName)
	oldSvcTmp, err := tgm.svcLister.Services(ns).Get(svcName)
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return tgm.svcControl.CreateService(tg, newSvc)
	}
	if err != nil {
		return fmt.Errorf("syncServiceForTiKVGroup: failed to get svc %s for tikvgroup %s/%s, error: %s", svcName, ns, tg.GetName(), err)
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
		svc.Spec.ClusterIP = oldSvc.Spec.ClusterIP
		_, err = tgm.svcControl.UpdateService(tg, &svc)
		return err
	}

	return nil
}

func (tgm *tikvGroupMemberManager) syncStatefulSetForTiKVGroup(tg *v1alpha1.TiKVGroup, tc *v1alpha1.TidbCluster) error {
	ns := tg.GetNamespace()
	tcName := tg.GetName()

	oldSetTmp, err := tgm.setLister.StatefulSets(ns).Get(controller.TiKVGroupMemberName(tcName))
	if err != nil && !errors.IsNotFound(err) {
		err = fmt.Errorf("syncStatefulSetForTiKVGroup: failed to get sts %s for tikvgroup %s/%s, error: %s", controller.TiKVGroupMemberName(tcName), ns, tcName, err)
		klog.Error(err)
		return err
	}
	setNotExist := errors.IsNotFound(err)
	oldSet := oldSetTmp.DeepCopy()

	if err := tgm.syncTiKVGroupStatus(tg, tc, oldSet); err != nil {
		klog.Error(err)
		return err
	}

	//TODO: support pause

	cm, err := tgm.syncTiKVConfigMap(tg, tc, oldSet)
	if err != nil {
		klog.Error(err)
		return err
	}

	newSet, err := getNewTiKVSetForTiKVGroup(tg, tc, cm)
	if err != nil {
		klog.Error(err)
		return err
	}
	if setNotExist {
		err = SetStatefulSetLastAppliedConfigAnnotation(newSet)
		if err != nil {
			klog.Error(err)
			return err
		}
		err = tgm.setControl.CreateStatefulSet(tg, newSet)
		if err != nil {
			klog.Error(err)
			return err
		}
		tg.Status.StatefulSet = &apps.StatefulSetStatus{}
		return nil
	}

	// TODO: Failure Recover

	// TODO: sync store Labels

	// Scale TiKVGroup
	if err := tgm.tikvGroupScaler.Scale(tg, oldSet, newSet); err != nil {
		return err
	}

	// TODO: TiKVGroup Auto Failover

	// Upgrade TiKVGroup
	if !templateEqual(newSet, oldSet) || tg.Status.Phase == v1alpha1.UpgradePhase {
		if err := tgm.tikvGroupUpgrader.Upgrade(tg, tc, oldSet, newSet); err != nil {
			return err
		}
	}

	return updateStatefulSet(tgm.setControl, tg, newSet, oldSet)
}

func (tgm *tikvGroupMemberManager) syncTiKVConfigMap(tg *v1alpha1.TiKVGroup, tc *v1alpha1.TidbCluster, set *apps.StatefulSet) (*corev1.ConfigMap, error) {
	// If config is undefined for TiKVGroup, we will directly set it as empty
	if tg.Spec.Config == nil {
		tg.Spec.Config = map[string]interface{}{}
	}

	newCm, err := getTikVConfigMapForTiKVGroup(tg, tc)
	if err != nil {
		return nil, err
	}
	if set != nil && tc.BaseTiKVSpec().ConfigUpdateStrategy() == v1alpha1.ConfigUpdateStrategyInPlace {
		inUseName := FindConfigMapVolume(&set.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, controller.TiKVGroupMemberName(tg.Name))
		})
		if inUseName != "" {
			newCm.Name = inUseName
		}
	}
	return tgm.typedControl.CreateOrUpdateConfigMap(tg, newCm)
}

func (tgm *tikvGroupMemberManager) syncTiKVGroupStatus(tg *v1alpha1.TiKVGroup, tc *v1alpha1.TidbCluster, set *apps.StatefulSet) error {
	if set == nil {
		// skip if not created yet
		return nil
	}
	tg.Status.StatefulSet = &set.Status
	upgrading, err := tgm.tikvGroupStatefulSetIsUpgrading(tg, set)
	if err != nil {
		tg.Status.Synced = false
		klog.Error(err)
		return err
	}
	if upgrading {
		tg.Status.Phase = v1alpha1.UpgradePhase
	} else if tg.TiKVStsDesiredReplicas() != *set.Spec.Replicas {
		tg.Status.Phase = v1alpha1.ScalePhase
	} else {
		tg.Status.Phase = v1alpha1.NormalPhase
	}

	previousStores := tg.Status.Stores
	stores := map[string]v1alpha1.TiKVStore{}
	tombstoneStores := map[string]v1alpha1.TiKVStore{}

	pdCli := controller.GetPDClient(tgm.pdControl, tc)
	// This only returns Up/Down/Offline stores
	storesInfo, err := pdCli.GetStores()
	if err != nil {
		tg.Status.Synced = false
		klog.Error(err)
		return err
	}

	pattern, err := regexp.Compile(fmt.Sprintf(tikvGroupStoreLimitPattern, tg.Name, tg.Name, tg.Namespace))
	if err != nil {
		tg.Status.Synced = false
		klog.Error(err)
		return err
	}

	for _, store := range storesInfo.Stores {
		// In theory, the external tikv can join the cluster, and the operator would only manage the internal tikv.
		// So we check the store owner to make sure it.
		if store.Store != nil && !pattern.Match([]byte(store.Store.Address)) {
			klog.V(4).Infof("discard store address %s", store.Store.Address)
			continue
		}
		status := getTiKVStore(store)
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

	tg.Status.Synced = true
	tg.Status.Stores = stores
	tg.Status.TombstoneStores = tombstoneStores
	tg.Status.Image = ""
	c := filterContainer(set, "tikv")
	if c != nil {
		tc.Status.TiKV.Image = c.Image
	}
	return nil
}

func (tgm *tikvGroupMemberManager) tikvGroupStatefulSetIsUpgrading(tg *v1alpha1.TiKVGroup, set *apps.StatefulSet) (bool, error) {
	if statefulSetIsUpgrading(set) {
		return true, nil
	}
	selector, err := label.NewGroup().Instance(tg.Name).TiKV().Selector()
	if err != nil {
		return false, err
	}
	tikvPods, err := tgm.podLister.Pods(set.Namespace).List(selector)
	if err != nil {
		return false, fmt.Errorf("tikvGroupStatefulSetIsUpgrading: failed to list pods for tikvgroup %s/%s, selector %s, error: %s", tg.GetNamespace(), tg.GetName(), selector, err)
	}
	for _, pod := range tikvPods {
		revisionHash, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return false, nil
		}
		if revisionHash != tg.Status.StatefulSet.UpdateRevision {
			return true, nil
		}
	}

	return false, nil
}

// TODO: add unit test
func newServiceForTiKVGroup(tg *v1alpha1.TiKVGroup, svcName string) *corev1.Service {
	ns := tg.Namespace
	tgName := tg.Name
	tgLabel := label.NewGroup().TiKV().Instance(tgName)
	svcLabel := label.NewGroup().TiKV().Instance(tgName).UsedByPeer()
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: ns,
			Labels:    svcLabel.Labels(),
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "peer",
					Port:       20160,
					TargetPort: intstr.FromInt(20160),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "status",
					Port:       20180,
					TargetPort: intstr.FromInt(20180),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector:                 tgLabel.Labels(),
			PublishNotReadyAddresses: true,
			ClusterIP:                "None",
		},
	}
	return svc
}

// TODO: add unit test
func getTikVConfigMapForTiKVGroup(tg *v1alpha1.TiKVGroup, tc *v1alpha1.TidbCluster) (*corev1.ConfigMap, error) {
	if tg.Spec.Config == nil {
		tg.Spec.Config = map[string]interface{}{}
	}
	version, err := semver.NewVersion(tc.TiKVVersion())
	if err != nil {
		return nil, fmt.Errorf("tikvgroup[%s/%s] failed to parse version,err:%v", tg.Namespace, tg.Name, err)
	}
	enableAdvertiseStatusAddr := true
	// For the version less than v4.0.1, don't use advertise-status-addr start arg
	if version.LessThan(tikvV401Version) {
		enableAdvertiseStatusAddr = false
	}
	scriptModel := &TiKVStartScriptModel{
		PDAddress:                 tc.Scheme() + "://${CLUSTER_NAME}-pd:2379",
		EnableAdvertiseStatusAddr: enableAdvertiseStatusAddr,
		DataDir:                   filepath.Join(tikvDataVolumeMountPath, tg.Spec.DataSubDir),
		AdvertiseStatusAddr:       "${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc",
	}
	cm, err := getTikVConfigMapForTiKVSpec(&tg.Spec.TiKVSpec, tc, scriptModel)
	if err != nil {
		return nil, err
	}
	instanceName := tg.Name
	tikvLabel := label.NewGroup().Instance(instanceName).TiKV().Labels()
	cm.ObjectMeta = metav1.ObjectMeta{
		Name:            controller.TiKVGroupMemberName(tg.Name),
		Namespace:       tg.Namespace,
		Labels:          tikvLabel,
		OwnerReferences: []metav1.OwnerReference{controller.GetTiKVGroupOwnerRef(tg)},
	}
	if tg.BaseTiKVSpec(tc).ConfigUpdateStrategy() == v1alpha1.ConfigUpdateStrategyRollingUpdate {
		if err := AddConfigMapDigestSuffix(cm); err != nil {
			return nil, err
		}
	}
	return cm, nil
}

// TODO: add unit test
func getNewTiKVSetForTiKVGroup(tg *v1alpha1.TiKVGroup, tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	tgName := tg.Name
	baseTiKVSpec := tg.BaseTiKVSpec(tc)
	tikvConfigMap := cm.Name

	annMount, annVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		annMount,
		{Name: v1alpha1.TiKVMemberType.String(), MountPath: tikvDataVolumeMountPath},
		{Name: "config", ReadOnly: true, MountPath: "/etc/tikv"},
		{Name: "startup-script", ReadOnly: true, MountPath: "/usr/local/bin"},
	}
	if tc.IsTLSClusterEnabled() {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "tikv-tls", ReadOnly: true, MountPath: "/var/lib/tikv-tls",
		})
	}
	vols := []corev1.Volume{
		annVolume,
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
	}
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

	storageRequest, err := controller.ParseStorageRequest(tg.Spec.Requests)
	if err != nil {
		return nil, fmt.Errorf("cannot parse storage request for tikv, tikvgroup %s/%s, error: %v", tg.Namespace, tg.Name, err)
	}

	tikvLabel := label.NewGroup().TiKV().Instance(tgName)
	setName := controller.TiKVGroupMemberName(tgName)
	podAnnotations := CombineAnnotations(controller.AnnProm(20180), baseTiKVSpec.Annotations())
	// TODO: support asts
	stsAnnotations := map[string]string{}
	capacity := controller.TiKVCapacity(tg.Spec.Limits)
	headlessSvcName := controller.TiKVGroupPeerMemberName(tgName)
	image := tc.TiKVImage()
	if len(tg.Spec.Image) > 0 {
		image = tg.Spec.Image
	}
	if tg.Spec.Version != nil && len(tg.Spec.BaseImage) > 0 {
		image = fmt.Sprintf("%s:%s", tg.Spec.BaseImage, *tg.Spec.Version)
	}
	privileged := pointer.BoolPtr(false)
	if tg.Spec.Privileged != nil {
		privileged = tg.Spec.Privileged
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
		Image:           image,
		ImagePullPolicy: baseTiKVSpec.ImagePullPolicy(),
		Command:         []string{"/bin/sh", "/usr/local/bin/tikv_start_script.sh"},
		SecurityContext: &corev1.SecurityContext{
			Privileged: privileged,
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "server",
				ContainerPort: int32(20160),
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "status",
				ContainerPort: int32(20180),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: volMounts,
		Resources:    controller.ContainerResource(tc.Spec.TiKV.ResourceRequirements),
	}
	podSpec := baseTiKVSpec.BuildPodSpec()
	// TODO: make tidb-scheduler support TiKVGroup
	podSpec.SchedulerName = "default-scheduler"
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
	podSpec.ServiceAccountName = tg.Spec.ServiceAccount

	tikvset := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            setName,
			Namespace:       ns,
			Labels:          tikvLabel.Labels(),
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetTiKVGroupOwnerRef(tg)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(tg.TiKVStsDesiredReplicas()),
			Selector: tikvLabel.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      tikvLabel.Labels(),
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				volumeClaimTemplate(storageRequest, v1alpha1.TiKVMemberType.String(), tg.Spec.StorageClassName),
			},
			ServiceName:         headlessSvcName,
			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: apps.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &apps.RollingUpdateStatefulSetStrategy{
					Partition: pointer.Int32Ptr(tg.TiKVStsDesiredReplicas()),
				},
			},
		},
	}
	return tikvset, nil
}
