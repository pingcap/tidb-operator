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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/listers/apps/v1beta1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// PDSetCreator implements the logic for creating the main pd statefulset and extra pd statefulsets
type PDSetCreator interface {
	CreatePDStatefulSet(*v1alpha1.TidbCluster) error
	CreateExtraPDStatefulSets(*v1alpha1.TidbCluster) error
}

type pdSetCreator struct {
	setLister  v1beta1.StatefulSetLister
	pvcLister  corelisters.PersistentVolumeClaimLister
	setControl controller.StatefulSetControlInterface
}

func (psc *pdSetCreator) CreatePDStatefulSet(tc *v1alpha1.TidbCluster) error {
	return psc.createStatefulSet(tc, tc.Spec.PD)
}

func (psc *pdSetCreator) CreateExtraPDStatefulSets(tc *v1alpha1.TidbCluster) error {
	for _, pdSpec := range tc.Spec.PDs {
		if err := psc.createStatefulSet(tc, pdSpec); err != nil {
			return err
		}
	}

	return nil
}

func (psc *pdSetCreator) createStatefulSet(tc *v1alpha1.TidbCluster, pdSpec v1alpha1.PDSpec) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	setName := controller.PDMemberName(tcName, pdSpec.Name)

	var err error
	var newPDSet *apps.StatefulSet
	if newPDSet, err = generateNewPDSetFrom(tc, pdSpec); err != nil {
		return err
	}
	if _, err = psc.setLister.StatefulSets(ns).Get(setName); err != nil && !errors.IsNotFound(err) {
		return err
	} else if err == nil {
		return nil
	}

	if err := SetLastAppliedConfigAnnotation(newPDSet); err != nil {
		return err
	}
	return psc.setControl.CreateStatefulSet(tc, newPDSet)
}

func generateNewPDSetFrom(tc *v1alpha1.TidbCluster, pdSpec v1alpha1.PDSpec) (*apps.StatefulSet, error) {
	ns := tc.Namespace
	tcName := tc.Name
	instanceName := tc.GetLabels()[label.InstanceLabelKey]
	pdConfigMapName := controller.PDMemberName(tcName, tc.Spec.PD.Name)

	annMount, annVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		annMount,
		{Name: "config", ReadOnly: true, MountPath: "/etc/pd"},
		{Name: "startup-script", ReadOnly: true, MountPath: "/usr/local/bin"},
		{Name: v1alpha1.PDMemberType.String(), MountPath: "/var/lib/pd"},
	}
	vols := []corev1.Volume{
		annVolume,
		{Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: pdConfigMapName,
					},
					Items: []corev1.KeyToPath{{Key: "config-file", Path: "pd.toml"}},
				},
			},
		},
		{Name: "startup-script",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: pdConfigMapName,
					},
					Items: []corev1.KeyToPath{{Key: "startup-script", Path: "pd_start_script.sh"}},
				},
			},
		},
	}

	var q resource.Quantity
	var err error
	if pdSpec.Requests != nil {
		size := pdSpec.Requests.Storage
		q, err = resource.ParseQuantity(size)
		if err != nil {
			return nil, fmt.Errorf("cant' get storage size: %s for TidbCluster: %s/%s, %v", size, ns, tcName, err)
		}
	}
	pdLabel := label.New().Instance(instanceName).PD()
	setName := controller.PDMemberName(tcName, pdSpec.Name)
	storageClassName := pdSpec.StorageClassName
	if storageClassName == "" {
		storageClassName = controller.DefaultStorageClassName
	}
	failureReplicas := 0
	for _, failureMember := range tc.Status.PD.FailureMembers {
		if failureMember.MemberDeleted {
			failureReplicas++
		}
	}

	pdSet := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            setName,
			Namespace:       ns,
			Labels:          pdLabel.Labels(),
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: func() *int32 { r := pdSpec.Replicas + int32(failureReplicas); return &r }(),
			Selector: pdLabel.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      pdLabel.Labels(),
					Annotations: controller.AnnProm(2379),
				},
				Spec: corev1.PodSpec{
					SchedulerName: tc.Spec.SchedulerName,
					Affinity: util.AffinityForNodeSelector(
						ns,
						pdSpec.NodeSelectorRequired,
						label.New().Instance(instanceName).PD(),
						pdSpec.NodeSelector,
					),
					Containers: []corev1.Container{
						{
							Name:            v1alpha1.PDMemberType.String(),
							Image:           pdSpec.Image,
							Command:         []string{"/bin/sh", "/usr/local/bin/pd_start_script.sh"},
							ImagePullPolicy: pdSpec.ImagePullPolicy,
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
							},
							VolumeMounts: volMounts,
							Resources:    util.ResourceRequirement(pdSpec.ContainerSpec),
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
									Name:  "SERVICE_NAME",
									Value: controller.PDMemberName(tcName, pdSpec.Name),
								},
								{
									Name:  "SET_NAME",
									Value: setName,
								},
								{
									Name:  "TZ",
									Value: tc.Spec.Timezone,
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyAlways,
					Tolerations:   pdSpec.Tolerations,
					Volumes:       vols,
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: v1alpha1.PDMemberType.String(),
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
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: apps.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &apps.RollingUpdateStatefulSetStrategy{
					Partition: func() *int32 { r := pdSpec.Replicas + int32(failureReplicas); return &r }(),
				}},
		},
	}

	return pdSet, nil
}
