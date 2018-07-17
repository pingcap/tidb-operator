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

	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/listers/apps/v1beta1"
	corelisters "k8s.io/client-go/listers/core/v1"

	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/new-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/new-operator/pkg/util"
	"github.com/pingcap/tidb-operator/new-operator/pkg/util/label"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type pdMemberManager struct {
	setControl controller.StatefulSetControlInterface
	svcControl controller.ServiceControlInterface
	setLister  v1beta1.StatefulSetLister
	svcLister  corelisters.ServiceLister
}

// NewPDMemberManager returns a *pdMemberManager
func NewPDMemberManager(setControl controller.StatefulSetControlInterface,
	svcControl controller.ServiceControlInterface,
	setLister v1beta1.StatefulSetLister,
	svcLister corelisters.ServiceLister) MemberManager {
	return &pdMemberManager{setControl, svcControl, setLister, svcLister}
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
	oldSvc, err := pmm.svcLister.Services(ns).Get(controller.PDSvcName(tcName))
	if errors.IsNotFound(err) {
		return pmm.svcControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(oldSvc.Spec, newSvc.Spec) {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		return pmm.svcControl.UpdateService(tc, &svc)
	}

	return nil
}

func (pmm *pdMemberManager) syncPDHeadlessServiceForTidbCluster(tc *v1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := pmm.getNewPDHeadlessServiceForTidbCluster(tc)
	oldSvc, err := pmm.svcLister.Services(ns).Get(controller.PDPeerSvcName(tcName))
	if errors.IsNotFound(err) {
		return pmm.svcControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(oldSvc.Spec, newSvc.Spec) {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		return pmm.svcControl.UpdateService(tc, &svc)
	}

	return nil
}

func (pmm *pdMemberManager) syncPDStatefulSetForTidbCluster(tc *v1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newPDSet, err := pmm.getNewPDSetForTidbCluster(tc)
	if err != nil {
		return err
	}

	oldPDSet, err := pmm.setLister.StatefulSets(ns).Get(controller.PDSetName(tcName))
	if errors.IsNotFound(err) {
		err = pmm.setControl.CreateStatefulSet(tc, newPDSet)
		if err != nil {
			return err
		}
		tc.Status.PD.StatefulSet = &apps.StatefulSetStatus{}
		return nil
	}
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(oldPDSet.Spec, newPDSet.Spec) {
		set := *oldPDSet
		set.Spec = newPDSet.Spec
		err := pmm.setControl.UpdateStatefulSet(tc, &set)
		if err != nil {
			return err
		}
		tc.Status.PD.StatefulSet = &oldPDSet.Status
	}

	return nil
}

func (pmm *pdMemberManager) getNewPDServiceForTidbCluster(tc *v1.TidbCluster) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	svcName := controller.PDSvcName(tcName)
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
	svcName := controller.PDPeerSvcName(tcName)
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

func (pmm *pdMemberManager) getNewPDSetForTidbCluster(tc *v1.TidbCluster) (*apps.StatefulSet, error) {
	ns := tc.Namespace
	tcName := tc.Name

	volMounts := []corev1.VolumeMount{
		{Name: "timezone", ReadOnly: true, MountPath: "/etc/localtime"},
		{Name: "annotations", ReadOnly: true, MountPath: "/etc/podinfo"},
		{Name: "config", ReadOnly: true, MountPath: "/etc/pd"},
		{Name: "pd", MountPath: "/var/lib/pd"},
	}
	vols := []corev1.Volume{
		{Name: "timezone", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/etc/localtime"}}},
		{Name: "annotations", VolumeSource: corev1.VolumeSource{
			DownwardAPI: &corev1.DownwardAPIVolumeSource{
				Items: []corev1.DownwardAPIVolumeFile{
					{
						Path:     "annotations",
						FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations"},
					},
				},
			},
		}},
		{Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: tc.ConfigMapName()},
					Items:                []corev1.KeyToPath{{Key: "pd-config", Path: "pd.toml"}},
				},
			},
		},
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

	pdSet := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            controller.PDSetName(tcName),
			Namespace:       ns,
			Labels:          pdLabel.Labels(),
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: func() *int32 { r := tc.Spec.PD.Replicas; return &r }(),
			Selector: pdLabel.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: pdLabel.Labels(),
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
							Command: []string{"/entrypoint.sh"},
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
									Name: "MY_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "MY_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name:  "MY_SERVICE_NAME",
									Value: controller.PDSvcName(tcName),
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
						Name: "pd",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						StorageClassName: func() *string { str := tc.Spec.PD.StorageClassName; return &str }(),
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: q,
							},
						},
					},
				},
			},
			ServiceName:         controller.PDSvcName(tcName),
			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy:      apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType},
		},
	}
	return pdSet, nil
}
