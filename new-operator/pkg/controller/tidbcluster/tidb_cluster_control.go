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

package tidbcluster

import (
	"fmt"
	"reflect"
	"time"

	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/new-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/new-operator/pkg/util"
	"github.com/pingcap/tidb-operator/new-operator/pkg/util/label"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
)

const (
	pdConnTimeout = 2 * time.Second
)

// ControlInterface implements the control logic for updating TidbClusters and their children StatefulSets.
// It is implemented as an interface to allow for extensions that provide different semantics.
// Currently, there is only one implementation.
type ControlInterface interface {
	// UpdateTidbCluster implements the control logic for StatefulSet creation, update, and deletion
	UpdateTidbCluster(*v1.TidbCluster, []*apps.StatefulSet) error
}

// NewDefaultTidbClusterControl returns a new instance of the default implementation TidbClusterControlInterface that
// implements the documented semantics for TidbClusters.
func NewDefaultTidbClusterControl(
	setControl controller.StatefulSetControlInterface,
	statusUpdater StatusUpdaterInterface,
	recorder record.EventRecorder) ControlInterface {
	return &defaultTidbClusterControl{
		setControl,
		statusUpdater,
		recorder,
	}
}

type defaultTidbClusterControl struct {
	setControl    controller.StatefulSetControlInterface
	statusUpdater StatusUpdaterInterface
	recorder      record.EventRecorder
}

// UpdateStatefulSet executes the core logic loop for a tidbcluster.
func (tcc *defaultTidbClusterControl) UpdateTidbCluster(tc *v1.TidbCluster, sets []*apps.StatefulSet) error {
	// PDControl
	// pdStatus, err := tcc.pdControl.GetPDStatus(tc)

	// perform the main update function and get the status
	status, err := tcc.updateTidbCluster(tc, sets /*, pdStatus */)
	if err != nil {
		return err
	}

	// update the tidbCluster's status
	return tcc.updateTidbClusterStatus(tc, status)
}

func (tcc *defaultTidbClusterControl) updateTidbCluster(tc *v1.TidbCluster,
	sets []*apps.StatefulSet /*, pdStatus */) (*v1.ClusterStatus, error) {
	// PD StatefulSet
	// TiKV StatefulSet
	// TiDB StatefulSet
	// Services
	// ConfigMaps
	// Secrets
	// PrivilegedTiDB
	// Monitor
	// E2E
	// TiKV file log
	// decrease a middle pod
	// update the middle tidb pod

	status := &tc.Status
	newPDSet, err := tcc.getNewPDSetForTidbCluster(tc)
	if err != nil {
		return status, err
	}
	oldPDSet, err := getStatefulSetFrom(sets, v1.PDContainerType)
	if err != nil {
		err = tcc.setControl.CreateStatefulSet(tc, newPDSet)
		if err != nil {
			return status, err
		}
		status.PDStatus.StatefulSet.Name = newPDSet.Name
	} else {
		if !reflect.DeepEqual(oldPDSet.Spec, newPDSet.Spec) {
			set := *oldPDSet
			set.Spec = newPDSet.Spec
			err = tcc.setControl.UpdateStatefulSet(tc, &set)
			if err != nil {
				return status, err
			}
			status.PDStatus.StatefulSet.Name = set.Name
		}
	}

	return status, nil
}

func (tcc *defaultTidbClusterControl) updateTidbClusterStatus(tc *v1.TidbCluster, status *v1.ClusterStatus) error {
	tc = tc.DeepCopy()
	status = status.DeepCopy()
	return tcc.statusUpdater.UpdateTidbClusterStatus(tc, status)
}

func getStatefulSetFrom(sets []*apps.StatefulSet, typ v1.ContainerType) (*apps.StatefulSet, error) {
	for _, set := range sets {
		if label.Label(set.Labels)[label.AppLabelKey] == typ.String() {
			return set, nil
		}
	}

	return nil, fmt.Errorf("can't find a %s StatefulSet", typ.String())
}

func (tcc *defaultTidbClusterControl) getNewPDSetForTidbCluster(tc *v1.TidbCluster) (*apps.StatefulSet, error) {
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
			OwnerReferences: []metav1.OwnerReference{getOwnerRef(tc)},
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
									Value: pdSvcName(clusterName),
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
			ServiceName:         pdSvcName(clusterName),
			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy:      apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType},
		},
	}
	return pdSet, nil
}
