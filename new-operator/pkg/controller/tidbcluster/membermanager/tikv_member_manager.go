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

package membermanager

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/glog"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/new-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/new-operator/pkg/util"
	"github.com/pingcap/tidb-operator/new-operator/pkg/util/label"
	"github.com/pingcap/tidb-operator/pkg/util/errors"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/listers/apps/v1beta1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/kubernetes/pkg/kubelet/apis"
)

// TikvMemberManager implements MemberManager.
// Having a separate type here is not necessary, but may help with clarity.
// embedding rather as opposed to just newtyping automatically extends the MemberManager interface.
type TikvMemberManager struct {
	StateSvcMemberManager
	pdControl  controller.PDControlInterface
	podLister  corelisters.PodLister
	nodeLister corelisters.NodeLister
}

var _ MemberManager = (*TikvMemberManager)(nil)

// NewTiKVMemberManager returns a *tikvMemberManager
func NewTiKVMemberManager(pdControl controller.PDControlInterface,
	setControl controller.StatefulSetControlInterface,
	svcControl controller.ServiceControlInterface,
	setLister v1beta1.StatefulSetLister,
	svcLister corelisters.ServiceLister,
	podLister corelisters.PodLister,
	nodeLister corelisters.NodeLister) *TikvMemberManager {

	kvmm := TikvMemberManager{
		pdControl:  pdControl,
		podLister:  podLister,
		nodeLister: nodeLister,
		StateSvcMemberManager: StateSvcMemberManager{
			StateSvcControlList: NewStateSvcControlList(setControl, svcControl, setLister, svcLister),
			MemberType:          v1.TiKVMemberType,
			SvcList: []SvcConfig{
				{
					Name:       "peer",
					Port:       20160,
					Headless:   true,
					SvcLabel:   func(l label.Label) label.Label { return l.TiKV() },
					MemberName: controller.TiKVPeerMemberName,
				},
			},
			GetNewSetForTidbCluster: getNewSetForTidbClusterTiKV,
		},
	}

	kvmm.StatusUpdate = func(tc *v1.TidbCluster, status *apps.StatefulSetStatus) error {
		tc.Status.TiKV.StatefulSet = status
		return kvmm.syncTidbClusterStatus(tc)
	}
	return &kvmm
}

func labelTiKV(tc *v1.TidbCluster) label.Label {
	tcName := tc.GetName()
	return label.New().Cluster(tcName).TiKV()
}

func getNewSetForTidbClusterTiKV(tc *v1.TidbCluster) (*apps.StatefulSet, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	tikvConfigMap := controller.TiKVMemberName(tcName)
	tzMount, tzVolume := timezoneMountVolume()
	annMount, annVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		tzMount,
		annMount,
		{Name: "tikv", MountPath: "/var/lib/tikv"},
		{Name: "config", ReadOnly: true, MountPath: "/etc/tikv"},
		{Name: "startup-script", ReadOnly: true, MountPath: "/usr/local/bin"},
	}
	vols := []corev1.Volume{
		tzVolume,
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

	var q resource.Quantity
	var err error

	if tc.Spec.TiKV.Requests != nil {
		size := tc.Spec.TiKV.Requests.Storage
		q, err = resource.ParseQuantity(size)
		if err != nil {
			return nil, fmt.Errorf("cant' get storage size: %s for TidbCluster: %s/%s, %v", size, ns, tcName, err)
		}
	}

	tikvLabel := labelTiKV(tc)
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
							Name:    v1.TiKVMemberType.String(),
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
							Env:          envVars(tcName, headlessSvcName, capacity),
						},
						{
							Name:  v1.PushGatewayMemberType.String(),
							Image: controller.GetPushgatewayImage(tc),
							Ports: []corev1.ContainerPort{
								{
									Name:          "metrics",
									ContainerPort: int32(9091),
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: []corev1.VolumeMount{tzMount},
							Resources: util.ResourceRequirement(tc.Spec.TiKVPromGateway.ContainerSpec,
								controller.DefaultPushGatewayRequest()),
						},
					},
					RestartPolicy: corev1.RestartPolicyAlways,
					Volumes:       vols,
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				volumeClaimTemplate(q, "tikv", &storageClassName),
			},
			ServiceName:         headlessSvcName,
			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy:      apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType},
		},
	}
	return tikvset, nil
}

func (ssmm *TikvMemberManager) getStateSet(tc *v1.TidbCluster) (*apps.StatefulSet, error) {
	tcName := tc.GetName()
	setName := controller.TiKVMemberName(tcName)
	stateSet, err := (ssmm.setLister).StatefulSets(tc.Namespace).Get(setName)
	if err != nil {
		return nil, err
	}
	return stateSet, nil
}

func (ssmm *TikvMemberManager) syncTidbClusterStatus(tc *v1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	previousStores := tc.Status.TiKV.Stores
	tikvStores := v1.TiKVStores{
		CurrentStores:   map[string]v1.TiKVStore{},
		TombStoneStores: map[string]v1.TiKVStore{},
	}

	pdCli := ssmm.pdControl.GetPDClient(tc)

	// This only returns Up/Down/Offline stores
	storesInfo, err := pdCli.GetStores()
	if err != nil {
		glog.Errorf("failed to get stores from PD for TidbCluster: [%s/%s], %v", ns, tcName, err)
		return err
	}
	for _, store := range storesInfo.Stores {
		status := ssmm.getTiKVStore(store)
		if status == nil {
			continue
		}

		err = ssmm.setStoreLabelsForTiKV(pdCli, store, ns, status.PodName)
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
		status := ssmm.getTiKVStore(store)
		if status == nil {
			continue
		}

		tikvStores.TombStoneStores[status.ID] = *status
	}

	tc.Status.TiKV.Stores = tikvStores
	return nil
}

func (ssmm *TikvMemberManager) getTiKVStore(store *controller.StoreInfo) *v1.TiKVStore {
	if store.Store == nil || store.Status == nil {
		return nil
	}
	storeID := fmt.Sprintf("%d", store.Store.GetId())
	ip := strings.Split(store.Store.GetAddress(), ":")[0]
	podName := strings.Split(ip, ".")[0]

	return &v1.TiKVStore{
		ID:                storeID,
		PodName:           podName,
		IP:                ip,
		State:             store.Store.StateName,
		LastHeartbeatTime: metav1.Time{Time: store.Status.LastHeartbeatTS},
	}
}

func (ssmm *TikvMemberManager) setStoreLabelsForTiKV(pdClient controller.PDClient, store *controller.StoreInfo, ns, podName string) error {
	pod, err := ssmm.podLister.Pods(ns).Get(podName)
	if err != nil {
		return err
	}

	nodeName := pod.Spec.NodeName
	ls, err := ssmm.getNodeLabels(nodeName)
	if err != nil {
		glog.Errorf("Node: [%s] has no node labels, skipping set store labels for Pod: [%s/%s]", nodeName, ns, podName)
		return err
	}

	glog.V(2).Infof("Pod: [%s/%s] is on node: [%s]. Node: [%s]'s labels: %v", ns, podName, nodeName, nodeName, ls)
	if !storeLabelsEqualNodeLabels(store.Store.Labels, ls) {
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

func (ssmm *TikvMemberManager) getNodeLabels(nodeName string) (map[string]string, error) {
	node, err := ssmm.nodeLister.Get(nodeName)
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
	return nil, errors.Errorf("labels not found")
}

// storeLabelsEqualNodeLabels compares store labels with node labels
// for historic reasons, PD stores TiKV labels as []*StoreLabel which is a key-value pair slice
func storeLabelsEqualNodeLabels(storeLabels []*metapb.StoreLabel, nodeLabels map[string]string) bool {
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
