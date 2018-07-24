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

	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/new-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/new-operator/pkg/util"
	"github.com/pingcap/tidb-operator/new-operator/pkg/util/label"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/listers/apps/v1beta1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

type tidbMemberManager struct {
	setControl controller.StatefulSetControlInterface
	svcControl controller.ServiceControlInterface
	setLister  v1beta1.StatefulSetLister
	svcLister  corelisters.ServiceLister
}

// NewTiDBMemberManager returns a *tidbMemberManager
func NewTiDBMemberManager(setControl controller.StatefulSetControlInterface,
	svcControl controller.ServiceControlInterface,
	setLister v1beta1.StatefulSetLister,
	svcLister corelisters.ServiceLister) MemberManager {
	return &tidbMemberManager{
		setControl: setControl,
		svcControl: svcControl,
		setLister:  setLister,
		svcLister:  svcLister,
	}
}

func (tmm *tidbMemberManager) Sync(tc *v1.TidbCluster) error {
	// Sync TiDB Service
	if err := tmm.syncTiDBServiceForTidbCluster(tc); err != nil {
		return err
	}

	// Sync Tidb StatefulSet
	return tmm.syncTiDBStatefulSetForTidbCluster(tc)
}

func (tmm *tidbMemberManager) syncTiDBServiceForTidbCluster(tc *v1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := getNewTiDBServiceForTidbCluster(tc)
	oldSvc, err := tmm.svcLister.Services(ns).Get(controller.TiDBMemberName(tcName))
	if errors.IsNotFound(err) {
		return tmm.svcControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(oldSvc.Spec, newSvc.Spec) {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		return tmm.svcControl.UpdateService(tc, &svc)
	}

	return nil
}

func (tmm *tidbMemberManager) syncTiDBStatefulSetForTidbCluster(tc *v1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newTiDBSet := getNewTiDBSetForTidbCluster(tc)

	oldTiDBSet, err := tmm.setLister.StatefulSets(ns).Get(controller.TiDBMemberName(tcName))
	if errors.IsNotFound(err) {
		err = tmm.setControl.CreateStatefulSet(tc, newTiDBSet)
		if err != nil {
			return err
		}
		tc.Status.TiDB.StatefulSet = &apps.StatefulSetStatus{}
		return nil
	}
	if err != nil {
		return err
	}

	tc.Status.TiDB.StatefulSet = &oldTiDBSet.Status

	if !reflect.DeepEqual(oldTiDBSet.Spec, newTiDBSet.Spec) {
		set := *oldTiDBSet
		set.Spec = newTiDBSet.Spec
		return tmm.setControl.UpdateStatefulSet(tc, &set)
	}

	return nil
}

func getNewTiDBServiceForTidbCluster(tc *v1.TidbCluster) *corev1.Service {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	svcName := controller.TiDBMemberName(tcName)
	tidbLabel := label.New().Cluster(tcName).TiDB().Labels()

	tidbSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          tidbLabel,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: corev1.ServiceSpec{
			Type: controller.GetServiceType(tc.Spec.Services, v1.TiDBMemberType.String()),
			Ports: []corev1.ServicePort{
				{
					Name:       "mysql-client",
					Port:       4000,
					TargetPort: intstr.FromInt(4000),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "status",
					Port:       10080,
					TargetPort: intstr.FromInt(10080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: tidbLabel,
		},
	}
	if tidbSvc.Spec.Type == corev1.ServiceTypeNodePort || tidbSvc.Spec.Type == corev1.ServiceTypeLoadBalancer {
		tidbSvc.Spec.ExternalTrafficPolicy = corev1.ServiceExternalTrafficPolicyType(controller.ExternalTrafficPolicy)
	}
	return tidbSvc
}

func getNewTiDBSetForTidbCluster(tc *v1.TidbCluster) *apps.StatefulSet {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	containers := []corev1.Container{
		getTiDBContainer(tc),
	}
	tidbLabel := label.New().Cluster(tcName).TiDB()

	tidbSet := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            controller.TiDBMemberName(tcName),
			Namespace:       ns,
			Labels:          tidbLabel.Labels(),
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: func() *int32 { r := tc.Spec.TiDB.Replicas; return &r }(),
			Selector: tidbLabel.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: tidbLabel.Labels(),
				},
				Spec: corev1.PodSpec{
					Affinity: util.AffinityForNodeSelector(
						ns,
						tc.Spec.TiDB.NodeSelectorRequired,
						label.New().Cluster(tcName).TiDB(),
						tc.Spec.TiDB.NodeSelector,
					),
					Containers:    containers,
					RestartPolicy: corev1.RestartPolicyAlways,
					Volumes:       getTiDBVolumes(tcName),
				},
			},
			ServiceName:         controller.TiDBMemberName(tcName),
			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy:      apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType},
		},
	}
	return tidbSet
}

func getTiDBVolumes(tcName string) []corev1.Volume {
	tidbConfigMap := controller.TiDBMemberName(tcName)
	return []corev1.Volume{
		{Name: "timezone", VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/etc/localtime",
			}},
		},
		{Name: "config", VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: tidbConfigMap,
				},
				Items: []corev1.KeyToPath{{Key: "config-file", Path: "tidb.toml"}},
			}},
		},
		{Name: "annotations", VolumeSource: corev1.VolumeSource{
			DownwardAPI: &corev1.DownwardAPIVolumeSource{
				Items: []corev1.DownwardAPIVolumeFile{
					{
						Path:     "labels",
						FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations"},
					},
				},
			}},
		},
		{Name: "startup-script", VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: tidbConfigMap,
				},
				Items: []corev1.KeyToPath{{Key: "startup-script", Path: "tidb_start_script.sh"}},
			}},
		},
	}
}

func getTiDBContainer(tc *v1.TidbCluster) corev1.Container {
	volMounts := []corev1.VolumeMount{
		{Name: "timezone", ReadOnly: true, MountPath: "/etc/localtime"},
		{Name: "config", ReadOnly: true, MountPath: "/etc/tidb"},
		{Name: "annotations", ReadOnly: true, MountPath: "/etc/podinfo"},
		{Name: "startup-script", ReadOnly: true, MountPath: "/usr/local/bin"},
	}
	return corev1.Container{
		Name:    v1.TiDBMemberType.String(),
		Image:   tc.Spec.TiDB.Image,
		Command: []string{"/bin/sh", "/usr/local/bin/tidb_start_script.sh"},
		Ports: []corev1.ContainerPort{
			{
				Name:          "server",
				ContainerPort: int32(4000),
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "status", // pprof, status, metrics
				ContainerPort: int32(10080),
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "metrics", // This is used for metrics pull
				ContainerPort: int32(10080),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: volMounts,
		Resources:    util.ResourceRequirement(tc.Spec.TiDB.ContainerSpec),
		Env: []corev1.EnvVar{
			{
				Name:  "CLUSTER_NAME",
				Value: tc.GetName(),
			},
		},
	}
}
