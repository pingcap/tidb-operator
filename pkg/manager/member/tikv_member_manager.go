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
	"reflect"
	"strings"

	"github.com/golang/glog"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/apis"
)

// tikvMemberManager implements manager.Manager.
type tikvMemberManager struct {
	setControl                   controller.StatefulSetControlInterface
	svcControl                   controller.ServiceControlInterface
	pdControl                    pdapi.PDControlInterface
	setLister                    v1.StatefulSetLister
	svcLister                    corelisters.ServiceLister
	podLister                    corelisters.PodLister
	nodeLister                   corelisters.NodeLister
	autoFailover                 bool
	tikvFailover                 Failover
	tikvScaler                   Scaler
	tikvUpgrader                 Upgrader
	tikvStatefulSetIsUpgradingFn func(corelisters.PodLister, pdapi.PDControlInterface, *apps.StatefulSet, *v1alpha1.TidbCluster) (bool, error)
}

// NewTiKVMemberManager returns a *tikvMemberManager
func NewTiKVMemberManager(pdControl pdapi.PDControlInterface,
	setControl controller.StatefulSetControlInterface,
	svcControl controller.ServiceControlInterface,
	setLister v1.StatefulSetLister,
	svcLister corelisters.ServiceLister,
	podLister corelisters.PodLister,
	nodeLister corelisters.NodeLister,
	autoFailover bool,
	tikvFailover Failover,
	tikvScaler Scaler,
	tikvUpgrader Upgrader) manager.Manager {
	kvmm := tikvMemberManager{
		pdControl:    pdControl,
		podLister:    podLister,
		nodeLister:   nodeLister,
		setControl:   setControl,
		svcControl:   svcControl,
		setLister:    setLister,
		svcLister:    svcLister,
		autoFailover: autoFailover,
		tikvFailover: tikvFailover,
		tikvScaler:   tikvScaler,
		tikvUpgrader: tikvUpgrader,
	}
	kvmm.tikvStatefulSetIsUpgradingFn = tikvStatefulSetIsUpgrading
	return &kvmm
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
func (tkmm *tikvMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if !tc.PDIsAvailable() {
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
		if err := tkmm.syncServiceForTidbCluster(tc, svc); err != nil {
			return err
		}
	}
	return tkmm.syncStatefulSetForTidbCluster(tc)
}

func (tkmm *tikvMemberManager) syncServiceForTidbCluster(tc *v1alpha1.TidbCluster, svcConfig SvcConfig) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := tkmm.getNewServiceForTidbCluster(tc, svcConfig)
	oldSvcTmp, err := tkmm.svcLister.Services(ns).Get(svcConfig.MemberName(tcName))
	if errors.IsNotFound(err) {
		err = SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return tkmm.svcControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return err
	}

	oldSvc := oldSvcTmp.DeepCopy()

	equal, err := serviceEqual(newSvc, oldSvc)
	if err != nil {
		return err
	}
	if !equal {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		// TODO add unit test
		svc.Spec.ClusterIP = oldSvc.Spec.ClusterIP
		err = SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		_, err = tkmm.svcControl.UpdateService(tc, &svc)
		return err
	}

	return nil
}

func (tkmm *tikvMemberManager) syncStatefulSetForTidbCluster(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSet, err := tkmm.getNewSetForTidbCluster(tc)
	if err != nil {
		return err
	}

	oldSetTmp, err := tkmm.setLister.StatefulSets(ns).Get(controller.TiKVMemberName(tcName))
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if errors.IsNotFound(err) {
		err = SetLastAppliedConfigAnnotation(newSet)
		if err != nil {
			return err
		}
		err = tkmm.setControl.CreateStatefulSet(tc, newSet)
		if err != nil {
			return err
		}
		tc.Status.TiKV.StatefulSet = &apps.StatefulSetStatus{}
		return nil
	}

	oldSet := oldSetTmp.DeepCopy()

	if err := tkmm.syncTidbClusterStatus(tc, oldSet); err != nil {
		return err
	}

	if _, err := tkmm.setStoreLabelsForTiKV(tc); err != nil {
		return err
	}

	if !templateEqual(newSet.Spec.Template, oldSet.Spec.Template) || tc.Status.TiKV.Phase == v1alpha1.UpgradePhase {
		if err := tkmm.tikvUpgrader.Upgrade(tc, oldSet, newSet); err != nil {
			return err
		}
	}

	if *newSet.Spec.Replicas > *oldSet.Spec.Replicas {
		if err := tkmm.tikvScaler.ScaleOut(tc, oldSet, newSet); err != nil {
			return err
		}
	}

	if *newSet.Spec.Replicas < *oldSet.Spec.Replicas {
		if err := tkmm.tikvScaler.ScaleIn(tc, oldSet, newSet); err != nil {
			return err
		}
	}

	if tkmm.autoFailover {
		if tc.TiKVAllPodsStarted() && !tc.TiKVAllStoresReady() {
			if err := tkmm.tikvFailover.Failover(tc); err != nil {
				return err
			}
		}
	}

	if !statefulSetEqual(*newSet, *oldSet) {
		set := *oldSet
		set.Spec.Template = newSet.Spec.Template
		*set.Spec.Replicas = *newSet.Spec.Replicas
		set.Spec.UpdateStrategy = newSet.Spec.UpdateStrategy
		err := SetLastAppliedConfigAnnotation(&set)
		if err != nil {
			return err
		}
		_, err = tkmm.setControl.UpdateStatefulSet(tc, &set)
		return err
	}

	return nil
}

func (tkmm *tikvMemberManager) getNewServiceForTidbCluster(tc *v1alpha1.TidbCluster, svcConfig SvcConfig) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	instanceName := tc.GetLabels()[label.InstanceLabelKey]
	svcName := svcConfig.MemberName(tcName)
	svcLabel := svcConfig.SvcLabel(label.New().Instance(instanceName)).Labels()

	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          svcLabel,
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
			Selector: svcLabel,
		},
	}
	if svcConfig.Headless {
		svc.Spec.ClusterIP = "None"
	} else {
		svc.Spec.Type = controller.GetServiceType(tc.Spec.Services, v1alpha1.TiKVMemberType.String())
	}
	return &svc
}

func (tkmm *tikvMemberManager) getNewSetForTidbCluster(tc *v1alpha1.TidbCluster) (*apps.StatefulSet, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	tikvConfigMap := controller.MemberConfigMapName(tc, v1alpha1.TiKVMemberType)
	annMount, annVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		annMount,
		{Name: v1alpha1.TiKVMemberType.String(), MountPath: "/var/lib/tikv"},
		{Name: "config", ReadOnly: true, MountPath: "/etc/tikv"},
		{Name: "startup-script", ReadOnly: true, MountPath: "/usr/local/bin"},
	}
	if tc.Spec.EnableTLSCluster {
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
	if tc.Spec.EnableTLSCluster {
		vols = append(vols, corev1.Volume{
			Name: "tikv-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: controller.TiKVMemberName(tcName),
				},
			},
		})
	}

	var q resource.Quantity
	var err error

	if tc.Spec.TiKV.Requests != nil {
		size := tc.Spec.TiKV.Requests.Storage
		q, err = resource.ParseQuantity(size)
		if err != nil {
			return nil, fmt.Errorf("cant' get storage size: %s for TidbCluster: %s/%s, %v", size, ns, tcName, err)
		}
	}

	tikvLabel := tkmm.labelTiKV(tc)
	setName := controller.TiKVMemberName(tcName)
	podAnnotations := CombineAnnotations(controller.AnnProm(20180), tc.Spec.TiKV.Annotations)
	capacity := controller.TiKVCapacity(tc.Spec.TiKV.Limits)
	headlessSvcName := controller.TiKVPeerMemberName(tcName)
	storageClassName := tc.Spec.TiKV.StorageClassName
	if storageClassName == "" {
		storageClassName = controller.DefaultStorageClassName
	}

	dnsPolicy := corev1.DNSClusterFirst // same as k8s defaults
	if tc.Spec.TiKV.HostNetwork {
		dnsPolicy = corev1.DNSClusterFirstWithHostNet
	}

	tikvset := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            setName,
			Namespace:       ns,
			Labels:          tikvLabel.Labels(),
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: controller.Int32Ptr(tc.TiKVRealReplicas()),
			Selector: tikvLabel.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      tikvLabel.Labels(),
					Annotations: podAnnotations,
				},
				Spec: corev1.PodSpec{
					SchedulerName: tc.Spec.SchedulerName,
					Affinity:      tc.Spec.TiKV.Affinity,
					NodeSelector:  tc.Spec.TiKV.NodeSelector,
					HostNetwork:   tc.Spec.TiKV.HostNetwork,
					DNSPolicy:     dnsPolicy,
					Containers: []corev1.Container{
						{
							Name:            v1alpha1.TiKVMemberType.String(),
							Image:           tc.Spec.TiKV.Image,
							Command:         []string{"/bin/sh", "/usr/local/bin/tikv_start_script.sh"},
							ImagePullPolicy: tc.Spec.TiKV.ImagePullPolicy,
							SecurityContext: &corev1.SecurityContext{
								Privileged: &tc.Spec.TiKV.Privileged,
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "server",
									ContainerPort: int32(20160),
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: volMounts,
							Resources:    util.ResourceRequirement(tc.Spec.TiKV.ContainerSpec),
							Env: []corev1.EnvVar{
								{
									Name: "NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
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
							},
						},
					},
					RestartPolicy:     corev1.RestartPolicyAlways,
					Tolerations:       tc.Spec.TiKV.Tolerations,
					Volumes:           vols,
					SecurityContext:   tc.Spec.TiKV.PodSecurityContext,
					PriorityClassName: tc.Spec.TiKV.PriorityClassName,
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				tkmm.volumeClaimTemplate(q, v1alpha1.TiKVMemberType.String(), &storageClassName),
			},
			ServiceName:         headlessSvcName,
			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: apps.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &apps.RollingUpdateStatefulSetStrategy{
					Partition: controller.Int32Ptr(tc.TiKVRealReplicas()),
				},
			},
		},
	}
	return tikvset, nil
}

func (tkmm *tikvMemberManager) volumeClaimTemplate(q resource.Quantity, metaName string, storageClassName *string) corev1.PersistentVolumeClaim {
	return corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: metaName},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			StorageClassName: storageClassName,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: q,
				},
			},
		},
	}
}

func (tkmm *tikvMemberManager) labelTiKV(tc *v1alpha1.TidbCluster) label.Label {
	instanceName := tc.GetLabels()[label.InstanceLabelKey]
	return label.New().Instance(instanceName).TiKV()
}

func (tkmm *tikvMemberManager) syncTidbClusterStatus(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) error {
	tc.Status.TiKV.StatefulSet = &set.Status
	upgrading, err := tkmm.tikvStatefulSetIsUpgradingFn(tkmm.podLister, tkmm.pdControl, set, tc)
	if err != nil {
		return err
	}
	if upgrading && tc.Status.PD.Phase != v1alpha1.UpgradePhase {
		tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
	} else {
		tc.Status.TiKV.Phase = v1alpha1.NormalPhase
	}

	previousStores := tc.Status.TiKV.Stores
	stores := map[string]v1alpha1.TiKVStore{}
	tombstoneStores := map[string]v1alpha1.TiKVStore{}

	pdCli := controller.GetPDClient(tkmm.pdControl, tc)
	// This only returns Up/Down/Offline stores
	storesInfo, err := pdCli.GetStores()
	if err != nil {
		tc.Status.TiKV.Synced = false
		return err
	}

	for _, store := range storesInfo.Stores {
		status := tkmm.getTiKVStore(store)
		if status == nil {
			continue
		}
		// avoid LastHeartbeatTime be overwrite by zero time when pd lost LastHeartbeatTime
		if status.LastHeartbeatTime.IsZero() {
			if oldStatus, ok := previousStores[status.ID]; ok {
				glog.V(4).Infof("the pod:%s's store LastHeartbeatTime is zero,so will keep in %v", status.PodName, oldStatus.LastHeartbeatTime)
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
		status := tkmm.getTiKVStore(store)
		if status == nil {
			continue
		}
		tombstoneStores[status.ID] = *status
	}

	tc.Status.TiKV.Synced = true
	tc.Status.TiKV.Stores = stores
	tc.Status.TiKV.TombstoneStores = tombstoneStores
	return nil
}

func (tkmm *tikvMemberManager) getTiKVStore(store *pdapi.StoreInfo) *v1alpha1.TiKVStore {
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

func (tkmm *tikvMemberManager) setStoreLabelsForTiKV(tc *v1alpha1.TidbCluster) (int, error) {
	ns := tc.GetNamespace()
	// for unit test
	setCount := 0

	pdCli := controller.GetPDClient(tkmm.pdControl, tc)
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

	for _, store := range storesInfo.Stores {
		status := tkmm.getTiKVStore(store)
		if status == nil {
			continue
		}
		podName := status.PodName

		pod, err := tkmm.podLister.Pods(ns).Get(podName)
		if err != nil {
			return setCount, err
		}

		nodeName := pod.Spec.NodeName
		ls, err := tkmm.getNodeLabels(nodeName, locationLabels)
		if err != nil || len(ls) == 0 {
			glog.Warningf("node: [%s] has no node labels, skipping set store labels for Pod: [%s/%s]", nodeName, ns, podName)
			continue
		}

		if !tkmm.storeLabelsEqualNodeLabels(store.Store.Labels, ls) {
			set, err := pdCli.SetStoreLabels(store.Store.Id, ls)
			if err != nil {
				glog.Warningf("failed to set pod: [%s/%s]'s store labels: %v", ns, podName, ls)
				continue
			}
			if set {
				setCount++
				glog.Infof("pod: [%s/%s] set labels: %v successfully", ns, podName, ls)
			}
		}
	}

	return setCount, nil
}

func (tkmm *tikvMemberManager) getNodeLabels(nodeName string, storeLabels []string) (map[string]string, error) {
	node, err := tkmm.nodeLister.Get(nodeName)
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
			if host, found := ls[apis.LabelHostname]; found {
				labels[storeLabel] = host
			}
		}

	}
	return labels, nil
}

// storeLabelsEqualNodeLabels compares store labels with node labels
// for historic reasons, PD stores TiKV labels as []*StoreLabel which is a key-value pair slice
func (tkmm *tikvMemberManager) storeLabelsEqualNodeLabels(storeLabels []*metapb.StoreLabel, nodeLabels map[string]string) bool {
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
	instanceName := tc.GetLabels()[label.InstanceLabelKey]
	selector, err := label.New().Instance(instanceName).TiKV().Selector()
	if err != nil {
		return false, err
	}
	tikvPods, err := podLister.Pods(tc.GetNamespace()).List(selector)
	if err != nil {
		return false, err
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

func (ftmm *FakeTiKVMemberManager) SetSyncError(err error) {
	ftmm.err = err
}

func (ftmm *FakeTiKVMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	if ftmm.err != nil {
		return ftmm.err
	}
	if len(tc.Status.TiKV.Stores) != 0 {
		// simulate status update
		tc.Status.ClusterID = string(uuid.NewUUID())
	}
	return nil
}
