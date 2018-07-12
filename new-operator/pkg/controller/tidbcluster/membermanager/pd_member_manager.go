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
	"reflect"

	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/listers/apps/v1beta1"

	"fmt"

	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/new-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/new-operator/pkg/util"
	"github.com/pingcap/tidb-operator/new-operator/pkg/util/label"
	"k8s.io/apimachinery/pkg/api/errors"
)

type pdMemberManager struct {
	setControl controller.StatefulSetControlInterface
	setLister  v1beta1.StatefulSetLister
}

// NewPDMemberManager returns a *pdMemberManager
func NewPDMemberManager(setControl controller.StatefulSetControlInterface, setLister v1beta1.StatefulSetLister) MemberManager {
	return &pdMemberManager{setControl, setLister}
}

func (pmm *pdMemberManager) Sync(tc *v1.TidbCluster) error {
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
		tc.Status.PDStatus.StatefulSet.Name = newPDSet.Name
		return nil
	}
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(oldPDSet.Spec, newPDSet.Spec) {
		set := *oldPDSet
		set.Spec = newPDSet.Spec
		err = pmm.setControl.UpdateStatefulSet(tc, &set)
		if err != nil {
			return err
		}
		tc.Status.PDStatus.StatefulSet.Name = set.Name
	}

	return nil
}

func (pmm *pdMemberManager) getNewPDSetForTidbCluster(tc *v1.TidbCluster) (*apps.StatefulSet, error) {
	ns := tc.Namespace
	clusterName := tc.Name
	volMounts := []corev1.VolumeMount{
		{Name: v1.TiDBVolumeName, MountPath: "/var/lib/tidb"},
		{Name: "timezone", ReadOnly: true, MountPath: "/etc/localtime"},
		{Name: "labels", ReadOnly: true, MountPath: "/etc/k8s-pod-info"},
		{Name: "config", ReadOnly: true, MountPath: "/etc/tidb"},
	}
	vols := []corev1.Volume{
		{Name: v1.TiDBVolumeName, VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "PVC-NAME"}}},
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
					LocalObjectReference: corev1.LocalObjectReference{Name: tc.GetConfigMapName()},
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
			return nil, fmt.Errorf("cant' get storage size: %s for TidbCluster: %s/%s, %v", size, ns, clusterName, err)
		}
	}

	replicas := tc.Spec.PD.Size
	storageClassName := "tidb-operator-volume-provioner"
	pdSet := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("%s-pd", clusterName),
			Namespace:       ns,
			Labels:          label.New().Cluster(clusterName).PD().Labels(),
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: &replicas,
			Selector: label.New().Cluster(clusterName).PD().LabelSelector(),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Affinity: util.AffinityForNodeSelector(
						ns,
						tc.Spec.PD.NodeSelectorRequired,
						label.New().Cluster(clusterName).PD(),
						tc.Spec.PD.NodeSelector,
					),
					Containers: []corev1.Container{
						{
							Name:    v1.PDContainerType.String(),
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
									Value: controller.PDSvcName(clusterName),
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
						StorageClassName: &storageClassName,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: q,
							},
						},
					},
				},
			},
			ServiceName:         controller.PDSvcName(clusterName),
			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy:      apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType},
		},
	}
	return pdSet, nil
}
