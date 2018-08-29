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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/util"
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
	svcLister corelisters.ServiceLister) manager.Manager {
	return &tidbMemberManager{
		setControl: setControl,
		svcControl: svcControl,
		setLister:  setLister,
		svcLister:  svcLister,
	}
}

func (tmm *tidbMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	// Sync TiDB Service
	if err := tmm.syncTiDBServiceForTidbCluster(tc); err != nil {
		return err
	}

	// Sync Tidb StatefulSet
	return tmm.syncTiDBStatefulSetForTidbCluster(tc)
}

func (tmm *tidbMemberManager) syncTiDBServiceForTidbCluster(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := tmm.getNewTiDBServiceForTidbCluster(tc)
	oldSvc, err := tmm.svcLister.Services(ns).Get(controller.TiDBMemberName(tcName))
	if errors.IsNotFound(err) {
		err = SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return tmm.svcControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return err
	}

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
		return tmm.svcControl.UpdateService(tc, &svc)
	}

	return nil
}

func (tmm *tidbMemberManager) syncTiDBStatefulSetForTidbCluster(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newTiDBSet := tmm.getNewTiDBSetForTidbCluster(tc)
	oldTiDBSet, err := tmm.setLister.StatefulSets(ns).Get(controller.TiDBMemberName(tcName))
	if errors.IsNotFound(err) {
		err = SetLastAppliedConfigAnnotation(newTiDBSet)
		if err != nil {
			return err
		}
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

	if err = tmm.syncTidbClusterStatus(tc, oldTiDBSet); err != nil {
		return err
	}

	if err = tmm.upgrade(tc, oldTiDBSet, newTiDBSet); err != nil {
		return err
	}

	if err != nil {
		return err
	}
	if !statefulSetEqual(*oldTiDBSet, *newTiDBSet) {
		set := *oldTiDBSet
		set.Spec.Template = newTiDBSet.Spec.Template
		*set.Spec.Replicas = *newTiDBSet.Spec.Replicas
		err = SetLastAppliedConfigAnnotation(&set)
		if err != nil {
			return err
		}
		return tmm.setControl.UpdateStatefulSet(tc, &set)
	}

	return nil
}

func (tmm *tidbMemberManager) getNewTiDBServiceForTidbCluster(tc *v1alpha1.TidbCluster) *corev1.Service {
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
			Type: controller.GetServiceType(tc.Spec.Services, v1alpha1.TiDBMemberType.String()),
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

func (tmm *tidbMemberManager) getNewTiDBSetForTidbCluster(tc *v1alpha1.TidbCluster) *apps.StatefulSet {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	tidbConfigMap := controller.TiDBMemberName(tcName)

	annMount, annVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		annMount,
		{Name: "config", ReadOnly: true, MountPath: "/etc/tidb"},
		{Name: "startup-script", ReadOnly: true, MountPath: "/usr/local/bin"},
	}
	vols := []corev1.Volume{
		annVolume,
		{Name: "config", VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: tidbConfigMap,
				},
				Items: []corev1.KeyToPath{{Key: "config-file", Path: "tidb.toml"}},
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

	if tc.Spec.Localtime {
		tzMount, tzVolume := timezoneMountVolume()
		volMounts = append(volMounts, tzMount)
		vols = append(vols, tzVolume)
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
					Labels:      tidbLabel.Labels(),
					Annotations: controller.AnnProm(),
				},
				Spec: corev1.PodSpec{
					Affinity: util.AffinityForNodeSelector(
						ns,
						tc.Spec.TiDB.NodeSelectorRequired,
						label.New().Cluster(tcName).TiDB(),
						tc.Spec.TiDB.NodeSelector,
					),
					Containers: []corev1.Container{
						{
							Name:    v1alpha1.TiDBMemberType.String(),
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
						},
					},
					RestartPolicy: corev1.RestartPolicyAlways,
					Volumes:       vols,
				},
			},
			ServiceName:         controller.TiDBMemberName(tcName),
			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy:      apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType},
		},
	}
	return tidbSet
}

func (tmm *tidbMemberManager) syncTidbClusterStatus(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) error {
	tc.Status.TiDB.StatefulSet = &set.Status

	if statefulSetInNormal(set) {
		tc.Status.TiDB.Phase = v1alpha1.NormalPhase
	} else {
		tc.Status.TiDB.Phase = v1alpha1.UpgradePhase
	}
	return nil
}

func (tmm *tidbMemberManager) upgrade(tc *v1alpha1.TidbCluster, oldSet *apps.StatefulSet, newSet *apps.StatefulSet) error {
	if tmm.needUpgrade(tc, newSet, oldSet) {
		tc.Status.TiDB.Phase = v1alpha1.UpgradePhase
	} else {
		_, podSpec, err := GetLastAppliedConfig(oldSet)
		if err != nil {
			return err
		}
		newSet.Spec.Template.Spec = *podSpec
	}
	return nil
}

func (tmm *tidbMemberManager) needUpgrade(tc *v1alpha1.TidbCluster, newSet *apps.StatefulSet, oldSet *apps.StatefulSet) bool {
	if tc.Status.PD.Phase == v1alpha1.UpgradePhase {
		return false
	}

	if tc.Status.TiKV.Phase == v1alpha1.UpgradePhase {
		return false
	}

	return !templateEqual(newSet.Spec.Template, oldSet.Spec.Template)
}
