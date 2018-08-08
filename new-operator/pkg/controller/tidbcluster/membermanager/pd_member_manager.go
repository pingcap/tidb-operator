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
	"strconv"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/new-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/new-operator/pkg/util"
	"github.com/pingcap/tidb-operator/new-operator/pkg/util/label"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/listers/apps/v1beta1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

const defaultReplicas = 3

type pdMemberManager struct {
	pdControl  controller.PDControlInterface
	setControl controller.StatefulSetControlInterface
	svcControl controller.ServiceControlInterface
	setLister  v1beta1.StatefulSetLister
	svcLister  corelisters.ServiceLister
}

// NewPDMemberManager returns a *pdMemberManager
func NewPDMemberManager(pdControl controller.PDControlInterface, setControl controller.StatefulSetControlInterface,
	svcControl controller.ServiceControlInterface,
	setLister v1beta1.StatefulSetLister,
	svcLister corelisters.ServiceLister) MemberManager {
	return &pdMemberManager{pdControl, setControl, svcControl, setLister, svcLister}
}

func (pmm *pdMemberManager) Sync(tc *v1.TidbCluster) error {
	// Sync PD Service
	if err := pmm.syncPDServiceForTidbCluster(tc); err != nil {
		return err
	}

	// Sync PD Headless Service
	if err := pmm.syncPDHeadlessServiceForTidbCluster(tc); err != nil {
		return err
	}

	// Sync PD StatefulSet
	if err := pmm.syncPDStatefulSetForTidbCluster(tc); err != nil {
		return err
	}

	return nil
}

func (pmm *pdMemberManager) syncPDServiceForTidbCluster(tc *v1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := pmm.getNewPDServiceForTidbCluster(tc)
	oldSvc, err := pmm.svcLister.Services(ns).Get(controller.PDMemberName(tcName))
	if errors.IsNotFound(err) {
		return pmm.svcControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(oldSvc.Spec, newSvc.Spec) {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		// TODO add unit test
		svc.Spec.ClusterIP = oldSvc.Spec.ClusterIP
		return pmm.svcControl.UpdateService(tc, &svc)
	}

	return nil
}

func (pmm *pdMemberManager) syncPDHeadlessServiceForTidbCluster(tc *v1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := pmm.getNewPDHeadlessServiceForTidbCluster(tc)
	oldSvc, err := pmm.svcLister.Services(ns).Get(controller.PDPeerMemberName(tcName))
	if errors.IsNotFound(err) {
		return pmm.svcControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(oldSvc.Spec, newSvc.Spec) {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		// TODO add unit test
		svc.Spec.ClusterIP = oldSvc.Spec.ClusterIP
		return pmm.svcControl.UpdateService(tc, &svc)
	}

	return nil
}

func (pmm *pdMemberManager) syncPDStatefulSetForTidbCluster(tc *v1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	oldPDSet, err := pmm.setLister.StatefulSets(ns).Get(controller.PDMemberName(tcName))
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if errors.IsNotFound(err) {
		newPDSet, err := pmm.getNewPDSetForTidbCluster(tc, nil)
		if err != nil {
			return err
		}
		if err := pmm.setControl.CreateStatefulSet(tc, newPDSet); err != nil {
			return err
		}
		tc.Status.PD.StatefulSet = &apps.StatefulSetStatus{}
		return nil
	}
	newPDSet, err := pmm.getNewPDSetForTidbCluster(tc, oldPDSet.GetAnnotations())
	if err != nil {
		return err
	}

	if err := pmm.syncTidbClusterStatus(tc, oldPDSet); err != nil {
		return err
	}

	if pmm.needReduce(tc, *oldPDSet.Spec.Replicas) {
		// We need remove member from cluster before reducing statefulset replicas
		ordinal := *oldPDSet.Spec.Replicas - 1
		if err := pmm.removeOneMember(tc, ordinal); err != nil {
			return err
		}
		set := *oldPDSet
		replicas := *oldPDSet.Spec.Replicas - 1 // we can only remove one member at a time
		set.Spec.Replicas = &replicas
		return pmm.setControl.UpdateStatefulSet(tc, &set)
	}

	if !reflect.DeepEqual(oldPDSet.Spec, newPDSet.Spec) {
		set := *oldPDSet
		set.Spec = newPDSet.Spec
		return pmm.setControl.UpdateStatefulSet(tc, &set)
	}

	return nil
}

func (pmm *pdMemberManager) syncTidbClusterStatus(tc *v1.TidbCluster, set *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	tc.Status.PD.StatefulSet = &set.Status

	pdClient := pmm.pdControl.GetPDClient(tc)
	healthInfo, err := pdClient.GetHealth()
	if err != nil {
		return err
	}
	pdStatus := map[string]v1.PDMember{}
	for _, memberHealth := range healthInfo.Healths {
		id := memberHealth.MemberID
		memberID := fmt.Sprintf("%d", id)
		var clientURL string
		if len(memberHealth.ClientUrls) > 0 {
			clientURL = memberHealth.ClientUrls[0]
		}
		name := memberHealth.Name
		if len(name) == 0 {
			glog.Warningf("PD member: [%d] don't have a name, and can't get it from clientUrls: [%s], memberHealth Info: [%v] in [%s/%s]",
				id, memberHealth.ClientUrls, memberHealth, ns, tcName)
			continue
		}

		status := v1.PDMember{
			Name:      name,
			ID:        memberID,
			ClientURL: clientURL,
			Health:    memberHealth.Health,
		}

		pdStatus[name] = status
	}

	tc.Status.PD.Members = pdStatus

	return nil
}

func (pmm *pdMemberManager) getNewPDServiceForTidbCluster(tc *v1.TidbCluster) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	svcName := controller.PDMemberName(tcName)
	pdLabel := label.New().Cluster(tcName).PD().Labels()

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          pdLabel,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: corev1.ServiceSpec{
			Type: controller.GetServiceType(tc.Spec.Services, v1.PDMemberType.String()),
			Ports: []corev1.ServicePort{
				{
					Name:       "client",
					Port:       2379,
					TargetPort: intstr.FromInt(2379),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: pdLabel,
		},
	}
}

func (pmm *pdMemberManager) getNewPDHeadlessServiceForTidbCluster(tc *v1.TidbCluster) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	svcName := controller.PDPeerMemberName(tcName)
	pdLabel := label.New().Cluster(tcName).PD().Labels()

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          pdLabel,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:       "peer",
					Port:       2380,
					TargetPort: intstr.FromInt(2380),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: pdLabel,
		},
	}
}

func (pmm *pdMemberManager) getNewPDSetForTidbCluster(tc *v1.TidbCluster, annotations map[string]string) (*apps.StatefulSet, error) {
	ns := tc.Namespace
	tcName := tc.Name
	pdConfigMap := controller.PDMemberName(tcName)

	annMount, annVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		annMount,
		{Name: "config", ReadOnly: true, MountPath: "/etc/pd"},
		{Name: "startup-script", ReadOnly: true, MountPath: "/usr/local/bin"},
		{Name: v1.PDMemberType.String(), MountPath: "/var/lib/pd"},
	}
	vols := []corev1.Volume{
		annVolume,
		{Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: pdConfigMap,
					},
					Items: []corev1.KeyToPath{{Key: "config-file", Path: "pd.toml"}},
				},
			},
		},
		{Name: "startup-script",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: pdConfigMap,
					},
					Items: []corev1.KeyToPath{{Key: "startup-script", Path: "pd_start_script.sh"}},
				},
			},
		},
	}

	if tc.Spec.Localtime {
		tzMount, tzVolume := timezoneMountVolume()
		volMounts = append(volMounts, tzMount)
		vols = append(vols, tzVolume)
	}

	var q resource.Quantity
	var err error
	if tc.Spec.PD.Requests != nil {
		size := tc.Spec.PD.Requests.Storage
		q, err = resource.ParseQuantity(size)
		if err != nil {
			return nil, fmt.Errorf("cant' get storage size: %s for TidbCluster: %s/%s, %v", size, ns, tcName, err)
		}
	}
	pdLabel := label.New().Cluster(tcName).PD()
	setName := controller.PDMemberName(tcName)
	storageClassName := tc.Spec.PD.StorageClassName
	if storageClassName == "" {
		storageClassName = controller.DefaultStorageClassName
	}

	var initialReplicas int
	if annotations == nil {
		initialReplicas = int(tc.Spec.PD.Replicas)
		annotations = map[string]string{
			label.AnnInitialPDReplicas: fmt.Sprintf("%d", initialReplicas),
		}
	} else {
		initialReplicas, err = strconv.Atoi(annotations[label.AnnInitialPDReplicas])
		if err != nil {
			// This should not happen, initial-replicas must be consistent in the cluster lifecycle
			initialReplicas = defaultReplicas
			glog.Warningf("can not find initial-pd-replicas from previous statefulset, use default %d", defaultReplicas)
		}
	}

	pdSet := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            setName,
			Namespace:       ns,
			Labels:          pdLabel.Labels(),
			Annotations:     annotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: func() *int32 { r := tc.Spec.PD.Replicas; return &r }(),
			Selector: pdLabel.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      pdLabel.Labels(),
					Annotations: controller.AnnProm(),
				},
				Spec: corev1.PodSpec{
					Affinity: util.AffinityForNodeSelector(
						ns,
						tc.Spec.PD.NodeSelectorRequired,
						label.New().Cluster(tcName).PD(),
						tc.Spec.PD.NodeSelector,
					),
					Containers: []corev1.Container{
						{
							Name:    v1.PDMemberType.String(),
							Image:   tc.Spec.PD.Image,
							Command: []string{"/bin/sh", "/usr/local/bin/pd_start_script.sh"},
							Ports: []corev1.ContainerPort{
								{
									Name:          "server",
									ContainerPort: int32(2380),
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "client",
									ContainerPort: int32(2379),
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "metrics", // This is for metrics pull
									ContainerPort: int32(2379),
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: volMounts,
							Resources:    util.ResourceRequirement(tc.Spec.PD.ContainerSpec),
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
									Name:  "PEER_SERVICE_NAME",
									Value: controller.PDPeerMemberName(tcName),
								},
								{
									Name:  "SET_NAME",
									Value: setName,
								},
								{
									Name:  "INITIAL_REPLICAS",
									Value: fmt.Sprintf("%d", initialReplicas),
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyAlways,
					Volumes:       vols,
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: v1.PDMemberType.String(),
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						StorageClassName: &storageClassName,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: q,
							},
						},
					},
				},
			},
			ServiceName:         controller.PDPeerMemberName(tcName),
			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy:      apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType},
		},
	}
	return pdSet, nil
}

func (pmm *pdMemberManager) needReduce(tc *v1.TidbCluster, oldReplicas int32) bool {
	return tc.Spec.PD.Replicas < oldReplicas
}

func (pmm *pdMemberManager) removeOneMember(tc *v1.TidbCluster, ordinal int32) error {
	memberName := fmt.Sprintf("%s-pd-%d", tc.Name, ordinal)
	return pmm.pdControl.GetPDClient(tc).DeleteMember(memberName)
}
