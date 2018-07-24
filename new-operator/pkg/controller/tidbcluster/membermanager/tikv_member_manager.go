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

	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/new-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/new-operator/pkg/util"
	"github.com/pingcap/tidb-operator/new-operator/pkg/util/label"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/listers/apps/v1beta1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// TikvMemberManager implements MemberManager.
// Having a separate type here is not necessary, but may help with clarity.
// embedding rather as opposed to just newtyping automatically extends the MemberManager interface.
type TikvMemberManager struct{ StateSvcMemberManager }

var _ MemberManager = (*TikvMemberManager)(nil)

// NewTiKVMemberManager returns a *tikvMemberManager
func NewTiKVMemberManager(setControl controller.StatefulSetControlInterface,
	svcControl controller.ServiceControlInterface,
	setLister v1beta1.StatefulSetLister,
	svcLister corelisters.ServiceLister) *TikvMemberManager {
	return &TikvMemberManager{StateSvcMemberManager{
		StateSvcControlList: NewStateSvcControlList(setControl, svcControl, setLister, svcLister),
		Set: SvcConfig{
			Name:       "server",
			Port:       20160,
			SvcLabel:   func(l label.Label) label.Label { return l.TiKV() },
			MemberName: controller.TiKVMemberName,
		},
		MemberType:              v1.TiKVMemberType,
		Headless:                nil,
		StatusUpdate:            statusUpdateTiKV,
		GetNewSetForTidbCluster: getNewSetForTidbClusterTiKV,
	}}
}

func statusUpdateTiKV(tc *v1.TidbCluster, status *apps.StatefulSetStatus) {
	tc.Status.TiKV.StatefulSet = status
}

func getNewSetForTidbClusterTiKV(tc *v1.TidbCluster, service SvcConfig) (*apps.StatefulSet, error) {
	ns := tc.GetNamespace()
	clusterName := tc.GetName()
	tcName := tc.Name

	cmds := []string{
		"/entrypoint.sh",
	}
	args := []string{
		fmt.Sprintf("--pd=%s-pd:2379", clusterName),
		"--addr=0.0.0.0:20160",
		fmt.Sprintf("--advertise-addr=$(MY_POD_NAME).%s-tikv-peer.%s.svc:20160", clusterName, ns),
		"--data-dir=/var/lib/tidb/tikv",
		"--capacity=" + controller.TiKVCapacity(tc.Spec.TiKV.Limits),
	}
	tzMount, tzVolume := timezoneMountVolume()
	annMount, annVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		tzMount,
		annMount,
		{Name: "tikv", MountPath: "/var/lib/tikv"},
		{Name: "config", ReadOnly: true, MountPath: "/etc/tikv"},
	}
	vols := []corev1.Volume{
		tzVolume,
		annVolume,
		{Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: controller.TiKVMemberName(tcName)},
					Items:                []corev1.KeyToPath{{Key: "config-file", Path: "tikv.toml"}},
				},
			},
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

	tikvLabel := label.New().Cluster(tcName).TiKV()
	setName := controller.TiKVMemberName(tcName)

	tikvset := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            setName,
			Namespace:       ns,
			Labels:          label.New().Cluster(clusterName).TiKV(),
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: func() *int32 { r := tc.Spec.TiKV.Replicas; return &r }(),
			Selector: tikvLabel.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Affinity: util.AffinityForNodeSelector(
						ns,
						tc.Spec.TiKV.NodeSelectorRequired,
						label.New().Cluster(clusterName).TiKV(),
						tc.Spec.TiKV.NodeSelector,
					),
					Containers: []corev1.Container{
						{
							Name:    v1.TiKVMemberType.String(),
							Image:   tc.Spec.TiKV.Image,
							Command: cmds,
							Args:    args,
							Ports: []corev1.ContainerPort{
								{
									Name:          "server",
									ContainerPort: int32(20160),
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: volMounts,
							Resources:    util.ResourceRequirement(tc.Spec.TiKV.ContainerSpec),
							Env:          envVars(setName, controller.TiKVMemberName(tcName)),
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
				volumeClaimTemplate(q, "tikv", func() *string { str := tc.Spec.TiKV.StorageClassName; return &str }()),
			},
			ServiceName:         controller.TiKVMemberName(tcName),
			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy:      apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType},
		},
	}
	return tikvset, nil
}
