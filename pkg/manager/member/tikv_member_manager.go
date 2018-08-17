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
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/listers/apps/v1beta1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/apis"
)

// tikvMemberManager implements manager.Manager.
type tikvMemberManager struct {
	setControl controller.StatefulSetControlInterface
	svcControl controller.ServiceControlInterface
	pdControl  controller.PDControlInterface
	setLister  v1beta1.StatefulSetLister
	svcLister  corelisters.ServiceLister
	podLister  corelisters.PodLister
	nodeLister corelisters.NodeLister
}

// NewTiKVMemberManager returns a *tikvMemberManager
func NewTiKVMemberManager(pdControl controller.PDControlInterface,
	setControl controller.StatefulSetControlInterface,
	svcControl controller.ServiceControlInterface,
	setLister v1beta1.StatefulSetLister,
	svcLister corelisters.ServiceLister,
	podLister corelisters.PodLister,
	nodeLister corelisters.NodeLister) manager.Manager {
	kvmm := tikvMemberManager{
		pdControl:  pdControl,
		podLister:  podLister,
		nodeLister: nodeLister,
		setControl: setControl,
		svcControl: svcControl,
		setLister:  setLister,
		svcLister:  svcLister,
	}
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
	oldSvc, err := tkmm.svcLister.Services(ns).Get(svcConfig.MemberName(tcName))
	if errors.IsNotFound(err) {
		return tkmm.svcControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(oldSvc.Spec, newSvc.Spec) {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		// TODO add unit test
		svc.Spec.ClusterIP = oldSvc.Spec.ClusterIP
		return tkmm.svcControl.UpdateService(tc, &svc)
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

	oldSet, err := tkmm.setLister.StatefulSets(ns).Get(controller.TiKVMemberName(tcName))
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if errors.IsNotFound(err) {
		controller.SetLastAppliedConfigAnnotation(newSet)
		err = tkmm.setControl.CreateStatefulSet(tc, newSet)
		if err != nil {
			return err
		}
		tc.Status.TiKV.StatefulSet = &apps.StatefulSetStatus{}
		return nil
	}

	if err = tkmm.syncTidbClusterStatus(tc, oldSet); err != nil {
		return err
	}

	if err = tkmm.upgrade(tc, oldSet, newSet); err != nil {
		return err
	}

	if err = tkmm.scaleDown(tc, oldSet, newSet); err != nil {
		return err
	}

	equal, err := controller.EqualStatefulSet(*newSet, *oldSet)
	if err != nil {
		return err
	}
	if !equal {
		set := *oldSet
		set.Spec.Template = newSet.Spec.Template
		*set.Spec.Replicas = *newSet.Spec.Replicas
		controller.SetLastAppliedConfigAnnotation(&set)
		return tkmm.setControl.UpdateStatefulSet(tc, &set)
	}

	return nil
}

func (tkmm *tikvMemberManager) getNewServiceForTidbCluster(tc *v1alpha1.TidbCluster, svcConfig SvcConfig) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	svcName := svcConfig.MemberName(tcName)
	svcLabel := svcConfig.SvcLabel(label.New().Cluster(tcName)).Labels()

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
	tikvConfigMap := controller.TiKVMemberName(tcName)
	annMount, annVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		annMount,
		{Name: v1alpha1.TiKVMemberType.String(), MountPath: "/var/lib/tikv"},
		{Name: "config", ReadOnly: true, MountPath: "/etc/tikv"},
		{Name: "startup-script", ReadOnly: true, MountPath: "/usr/local/bin"},
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
	pgwVolMounts := []corev1.VolumeMount{} // pushgateway volumeMounts
	if tc.Spec.Localtime {
		tzMount, tzVolume := timezoneMountVolume()
		volMounts = append(volMounts, tzMount)
		vols = append(vols, tzVolume)
		pgwVolMounts = []corev1.VolumeMount{tzMount}
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
	capacity := controller.TiKVCapacity(tc.Spec.TiKV.Limits)
	headlessSvcName := controller.TiKVPeerMemberName(tcName)
	storageClassName := tc.Spec.TiKV.StorageClassName
	if storageClassName == "" {
		storageClassName = controller.DefaultStorageClassName
	}

	tikvset := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            setName,
			Namespace:       ns,
			Labels:          tikvLabel.Labels(),
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: func() *int32 { r := tc.Spec.TiKV.Replicas; return &r }(),
			Selector: tikvLabel.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      tikvLabel.Labels(),
					Annotations: controller.AnnProm(),
				},
				Spec: corev1.PodSpec{
					Affinity: util.AffinityForNodeSelector(
						ns,
						tc.Spec.TiKV.NodeSelectorRequired,
						tikvLabel,
						tc.Spec.TiKV.NodeSelector,
					),
					Containers: []corev1.Container{
						{
							Name:    v1alpha1.TiKVMemberType.String(),
							Image:   tc.Spec.TiKV.Image,
							Command: []string{"/bin/sh", "/usr/local/bin/tikv_start_script.sh"},
							Ports: []corev1.ContainerPort{
								{
									Name:          "server",
									ContainerPort: int32(20160),
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: volMounts,
							Resources:    util.ResourceRequirement(tc.Spec.TiKV.ContainerSpec),
							Env:          tkmm.envVars(tcName, headlessSvcName, capacity),
						},
						{
							Name:  v1alpha1.PushGatewayMemberType.String(),
							Image: controller.GetPushgatewayImage(tc),
							Ports: []corev1.ContainerPort{
								{
									Name:          "metrics",
									ContainerPort: int32(9091),
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: pgwVolMounts,
							Resources: util.ResourceRequirement(tc.Spec.TiKVPromGateway.ContainerSpec,
								controller.DefaultPushGatewayRequest()),
						},
					},
					RestartPolicy: corev1.RestartPolicyAlways,
					Volumes:       vols,
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				tkmm.volumeClaimTemplate(q, v1alpha1.TiKVMemberType.String(), &storageClassName),
			},
			ServiceName:         headlessSvcName,
			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy:      apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType},
		},
	}
	return tikvset, nil
}

func (tkmm *tikvMemberManager) envVars(tcName, headlessSvcName, capacity string) []corev1.EnvVar {
	return []corev1.EnvVar{
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
	}
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
	tcName := tc.GetName()
	return label.New().Cluster(tcName).TiKV()
}

func (tkmm *tikvMemberManager) upgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	if oldSet.Status.CurrentRevision == oldSet.Status.UpdateRevision {
		tc.Status.TiKV.Phase = v1alpha1.NormalPhase
	}

	upgrade, err := tkmm.needUpgrade(tc, newSet, oldSet)
	if err != nil {
		return err
	}
	if upgrade {
		tc.Status.TiKV.Phase = v1alpha1.UpgradePhase
	} else {
		_, podSpec, err := controller.GetLastAppliedConfig(oldSet)
		if err != nil {
			return err
		}
		newSet.Spec.Template.Spec = *podSpec
	}
	return nil
}

func (tkmm *tikvMemberManager) scaleDown(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	// can not scale tikv when it is upgrading
	if tc.Status.TiKV.Phase == v1alpha1.UpgradePhase {
		newSet.Spec.Replicas = oldSet.Spec.Replicas
		glog.Infof("the TidbCluster: [%s/%s]'s tikv is upgrading,can not scale until upgrade have completed", tc.GetNamespace(), tc.GetName())
		return nil
	}

	// we can only remove one member at a time when scale down
	if tkmm.needReduce(tc, *oldSet.Spec.Replicas) {
		// We need remove member from cluster before reducing statefulset replicas
		ordinal := *oldSet.Spec.Replicas - 1
		if err := tkmm.removeOneMember(tc, ordinal); err != nil {
			return err
		}
		newSet.Spec.Replicas = &ordinal
	}

	return nil
}

func (tkmm *tikvMemberManager) needUpgrade(tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, oldSet *apps.StatefulSet) (bool, error) {
	if tc.Status.PD.Phase == v1alpha1.UpgradePhase {
		return false, nil
	}
	same, err := controller.EqualTemplate(newSet.Spec.Template, oldSet.Spec.Template)
	if err != nil {
		return false, err
	}
	return !same, nil
}

func (tkmm *tikvMemberManager) needReduce(tc *v1alpha1.TidbCluster, oldReplicas int32) bool {
	return tc.Spec.TiKV.Replicas < oldReplicas
}

// remove tikv store from cluster, return nil only when store status becomes tombstone
func (tkmm *tikvMemberManager) removeOneMember(tc *v1alpha1.TidbCluster, ordinal int32) error {
	name := fmt.Sprintf("%s-tikv-%d", tc.Name, ordinal)
	var state string
	var id uint64
	var err error
	for _, store := range tc.Status.TiKV.Stores.CurrentStores {
		if store.PodName == name {
			state = store.State
			id, err = strconv.ParseUint(store.ID, 10, 64)
			if err != nil {
				return err
			}
			if state != util.StoreOfflineState {
				if err := tkmm.pdControl.GetPDClient(tc).DeleteStore(id); err != nil {
					return err
				}
			}
			return fmt.Errorf("TiKV %s store %d  still in cluster, state: %s", name, id, state)
		}
	}
	for _, store := range tc.Status.TiKV.Stores.TombStoneStores {
		if store.PodName == name {
			state = store.State
			id, err = strconv.ParseUint(store.ID, 10, 64)
			if err != nil {
				return err
			}
			// TODO: double check if store is really not in Up/Offline/Down state
			glog.Infof("TiKV %s store %d becomes tombstone", name, id)
			return nil
		}
	}

	// store not found in TidbCluster status,
	// this can happen when TiKV joins cluster but we haven't synced its status
	// so return error to wait another round for safety
	return fmt.Errorf("TiKV %s not found in cluster", name)
}

func (tkmm *tikvMemberManager) getStateSet(tc *v1alpha1.TidbCluster) (*apps.StatefulSet, error) {
	tcName := tc.GetName()
	setName := controller.TiKVMemberName(tcName)
	stateSet, err := (tkmm.setLister).StatefulSets(tc.Namespace).Get(setName)
	if err != nil {
		return nil, err
	}
	return stateSet, nil
}

func (tkmm *tikvMemberManager) syncTidbClusterStatus(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	tc.Status.TiKV.StatefulSet = &set.Status

	previousStores := tc.Status.TiKV.Stores
	tikvStores := v1alpha1.TiKVStores{
		CurrentStores:   map[string]v1alpha1.TiKVStore{},
		TombStoneStores: map[string]v1alpha1.TiKVStore{},
	}

	pdCli := tkmm.pdControl.GetPDClient(tc)

	// This only returns Up/Down/Offline stores
	storesInfo, err := pdCli.GetStores()
	if err != nil {
		glog.Errorf("failed to get stores from PD for TidbCluster: [%s/%s], %v", ns, tcName, err)
		return err
	}
	for _, store := range storesInfo.Stores {
		status := tkmm.getTiKVStore(store)
		if status == nil {
			continue
		}

		err = tkmm.setStoreLabelsForTiKV(pdCli, store, ns, status.PodName)
		if err != nil {
			glog.Errorf("failed to setStoreLabelsForPod: [%s/%s], %v", ns, status.PodName, err)
			return err
		}

		// avoid LastHeartbeatTime be overwrite by zero time when pd lost LastHeartbeatTime
		if status.LastHeartbeatTime.IsZero() {
			if oldStatus, ok := previousStores.CurrentStores[status.ID]; ok {
				glog.Warningf("the pod:%s's store LastHeartbeatTime is zero,so will keep in %v", status.PodName, oldStatus.LastHeartbeatTime)
				status.LastHeartbeatTime = oldStatus.LastHeartbeatTime
			}
		}

		tikvStores.CurrentStores[status.ID] = *status
	}

	//this returns all tombstone stores
	tombstoneStoresInfo, err := pdCli.GetTombStoneStores()
	if err != nil {
		glog.Errorf("failed to get tombstone stores from PD for TidbCluster: [%s/%s], %v", ns, tcName, err)
		return err
	}
	for _, store := range tombstoneStoresInfo.Stores {
		status := tkmm.getTiKVStore(store)
		if status == nil {
			continue
		}

		tikvStores.TombStoneStores[status.ID] = *status
	}

	tc.Status.TiKV.Stores = tikvStores
	return nil
}

func (tkmm *tikvMemberManager) getTiKVStore(store *controller.StoreInfo) *v1alpha1.TiKVStore {
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
		State:             store.Store.StateName,
		LastHeartbeatTime: metav1.Time{Time: store.Status.LastHeartbeatTS},
	}
}

func (tkmm *tikvMemberManager) setStoreLabelsForTiKV(pdClient controller.PDClient, store *controller.StoreInfo, ns, podName string) error {
	pod, err := tkmm.podLister.Pods(ns).Get(podName)
	if err != nil {
		return err
	}

	nodeName := pod.Spec.NodeName
	ls, err := tkmm.getNodeLabels(nodeName)
	if err != nil {
		glog.Errorf("Node: [%s] has no node labels, skipping set store labels for Pod: [%s/%s]", nodeName, ns, podName)
		return err
	}

	glog.V(2).Infof("Pod: [%s/%s] is on node: [%s]. Node: [%s]'s labels: %v", ns, podName, nodeName, nodeName, ls)
	if !tkmm.storeLabelsEqualNodeLabels(store.Store.Labels, ls) {
		updated, err := pdClient.SetStoreLabels(store.Store.Id, ls)
		if err != nil {
			return err
		}
		if updated {
			glog.Infof("Pod: [%s/%s] set labels successfully,labels: %v ", ns, podName, nodeName, ls)
		}
	}
	return nil
}

func (tkmm *tikvMemberManager) getNodeLabels(nodeName string) (map[string]string, error) {
	node, err := tkmm.nodeLister.Get(nodeName)
	if err != nil {
		return nil, err
	}
	if ls := node.GetLabels(); ls != nil {
		labels := map[string]string{}
		if region, found := ls["region"]; found {
			labels["region"] = region
		}
		if zone, found := ls["zone"]; found {
			labels["zone"] = zone
		}
		if rack, found := ls["rack"]; found {
			labels["rack"] = rack
		}
		if host, found := ls[apis.LabelHostname]; found {
			labels["host"] = host
		}
		return labels, nil
	}
	return nil, fmt.Errorf("labels not found")
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
