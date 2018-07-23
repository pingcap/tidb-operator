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

type priTidbMemberManager struct {
	deployControl controller.DeploymentControlInterface
	svcControl    controller.ServiceControlInterface
	deployLister  v1beta1.DeploymentLister
	svcLister     corelisters.ServiceLister
}

// NewPriTiDBMemberManager returns a *priTidbMemberManager
func NewPriTiDBMemberManager(deployControl controller.DeploymentControlInterface,
	svcControl controller.ServiceControlInterface,
	deployLister v1beta1.DeploymentLister,
	svcLister corelisters.ServiceLister) MemberManager {
	return &priTidbMemberManager{
		deployControl: deployControl,
		svcControl:    svcControl,
		deployLister:  deployLister,
		svcLister:     svcLister,
	}
}

func (ptmm *priTidbMemberManager) Sync(tc *v1.TidbCluster) error {
	// Sync Privileged TiDB Service
	if err := ptmm.syncPrivilegedTiDBService(tc); err != nil {
		return err
	}

	// Sync Privileged TiDB Deployment
	return ptmm.syncPrivilegedTiDBDeployment(tc)
}

func (ptmm *priTidbMemberManager) syncPrivilegedTiDBService(tc *v1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := getNewPriTiDBServiceForTidbCluster(tc)
	oldSvc, err := ptmm.svcLister.Services(ns).Get(controller.PriTiDBMemberName(tcName))
	if errors.IsNotFound(err) {
		return ptmm.svcControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(oldSvc.Spec, newSvc.Spec) {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		return ptmm.svcControl.UpdateService(tc, &svc)
	}

	return nil
}

func (ptmm *priTidbMemberManager) syncPrivilegedTiDBDeployment(tc *v1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newDeploy := getNewPriTiDBDeploymentForTidbCluster(tc)
	oldDeploy, err := ptmm.deployLister.Deployments(ns).Get(controller.PriTiDBMemberName(tcName))
	if errors.IsNotFound(err) {
		err = ptmm.deployControl.CreateDeployment(tc, newDeploy)
		if err != nil {
			return err
		}
		tc.Status.PrivilegedTiDB.Deployment = &apps.DeploymentStatus{}
		return nil
	}
	if err != nil {
		return err
	}

	tc.Status.PrivilegedTiDB.Deployment = &oldDeploy.Status

	if !reflect.DeepEqual(oldDeploy.Spec, newDeploy.Spec) {
		deploy := *oldDeploy
		deploy.Spec = newDeploy.Spec
		return ptmm.deployControl.UpdateDeployment(tc, &deploy)
	}

	return nil
}

func getNewPriTiDBServiceForTidbCluster(tc *v1.TidbCluster) *corev1.Service {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	svcName := controller.PriTiDBMemberName(tcName)
	priTidbLabel := label.New().Cluster(tcName).App(v1.PriTiDBMemberType.String()).Labels()

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          priTidbLabel,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: corev1.ServiceSpec{
			Type: controller.GetServiceType(tc.Spec.Services, v1.PriTiDBMemberType.String()),
			Ports: []corev1.ServicePort{
				{
					Name:       "mysql-client",
					Port:       4000,
					TargetPort: intstr.FromInt(4000),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: priTidbLabel,
		},
	}
}

func getNewPriTiDBDeploymentForTidbCluster(tc *v1.TidbCluster) *apps.Deployment {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	deployName := controller.PriTiDBMemberName(tcName)
	priTidbLabel := label.New().Cluster(tcName).App(v1.PriTiDBMemberType.String())

	vols := getPriTiDBVolumes(tc.Spec.ConfigMap)
	containers := []corev1.Container{
		getPriTiDBContainer(tc),
	}
	return &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            deployName,
			Namespace:       ns,
			Labels:          priTidbLabel.Labels(),
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.DeploymentSpec{
			Replicas: &tc.Spec.PrivilegedTiDB.Replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   deployName,
					Labels: priTidbLabel.Labels(),
				},
				Spec: corev1.PodSpec{
					Affinity: util.AffinityForNodeSelector(
						ns,
						tc.Spec.PrivilegedTiDB.NodeSelectorRequired,
						priTidbLabel,
						tc.Spec.PrivilegedTiDB.NodeSelector,
					),
					RestartPolicy: corev1.RestartPolicyAlways,
					Containers:    containers,
					Volumes:       vols,
				},
			},
		},
	}
}

func getPriTiDBVolumes(cmName string) []corev1.Volume {
	items := []corev1.KeyToPath{{Key: "privileged-tidb-config", Path: "tidb.toml"}}
	return []corev1.Volume{
		{Name: "timezone", VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/etc/localtime",
			}},
		},
		{Name: "config", VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: cmName,
				},
				Items: items,
			}},
		},
	}
}

func getPriTiDBContainer(tc *v1.TidbCluster) corev1.Container {
	volMounts := []corev1.VolumeMount{
		{Name: "timezone", ReadOnly: true, MountPath: "/etc/localtime"},
		{Name: "config", ReadOnly: true, MountPath: "/etc/tidb"},
	}
	return corev1.Container{
		Name:    v1.PriTiDBMemberType.String(),
		Image:   tc.Spec.PrivilegedTiDB.Image,
		Command: []string{"/entrypoint.sh"},
		Ports: []corev1.ContainerPort{
			{
				Name:          "mysql-client",
				ContainerPort: int32(4000),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: volMounts,
		Resources:    util.ResourceRequirement(tc.Spec.PrivilegedTiDB.ContainerSpec),
	}
}
