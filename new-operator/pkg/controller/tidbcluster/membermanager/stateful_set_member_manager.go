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

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/pingcap/tidb-operator/new-operator/pkg/apis/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/new-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/new-operator/pkg/util/label"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/listers/apps/v1beta1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

// StateSvcControlList are the parameters passed by NewController
type StateSvcControlList struct {
	setControl controller.StatefulSetControlInterface
	svcControl controller.ServiceControlInterface
	setLister  v1beta1.StatefulSetLister
	svcLister  corelisters.ServiceLister
}

// SvcConfig corresponds to a K8s service
type SvcConfig struct {
	Name       string
	Port       int32
	SvcLabel   func(label.Label) label.Label
	MemberName func(clusterName string) string
	Headless   bool
}

// StateSvcMemberManager implements MemberManager
// It accepts StateSvcControlList fron NewController
// The rest is configuration that differs for each stateful service
type StateSvcMemberManager struct {
	StateSvcControlList
	SvcList                 []SvcConfig
	MemberType              v1.MemberType
	StatusUpdate            func(*v1.TidbCluster, *apps.StatefulSetStatus)
	GetNewSetForTidbCluster func(*v1.TidbCluster) (*apps.StatefulSet, error)
}

var _ MemberManager = (*StateSvcMemberManager)(nil)

// NewStateSvcControlList returns a StateSvcControlList.
func NewStateSvcControlList(setControl controller.StatefulSetControlInterface,
	svcControl controller.ServiceControlInterface,
	setLister v1beta1.StatefulSetLister,
	svcLister corelisters.ServiceLister) StateSvcControlList {
	return StateSvcControlList{setControl, svcControl, setLister, svcLister}
}

// Sync fulfills the MemberManager interface
func (ssmm *StateSvcMemberManager) Sync(tc *v1.TidbCluster) error {
	for _, svc := range ssmm.SvcList {
		if err := ssmm.syncServiceForTidbCluster(tc, svc); err != nil {
			return err
		}
	}
	if err := ssmm.syncStatefulSetForTidbCluster(tc); err != nil {
		return err
	}

	return nil
}

func (ssmm *StateSvcMemberManager) syncServiceForTidbCluster(tc *v1.TidbCluster, svcConfig SvcConfig) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := ssmm.getNewServiceForTidbCluster(tc, svcConfig)
	oldSvc, err := ssmm.svcLister.Services(ns).Get(svcConfig.MemberName(tcName))
	if errors.IsNotFound(err) {
		return ssmm.svcControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(oldSvc.Spec, newSvc.Spec) {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		return ssmm.svcControl.UpdateService(tc, &svc)
	}

	return nil
}

func (ssmm *StateSvcMemberManager) syncStatefulSetForTidbCluster(tc *v1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSet, err := ssmm.GetNewSetForTidbCluster(tc)
	if err != nil {
		return err
	}

	oldSet, err := ssmm.setLister.StatefulSets(ns).Get(controller.TiKVMemberName(tcName))
	if errors.IsNotFound(err) {
		err = ssmm.setControl.CreateStatefulSet(tc, newSet)
		if err != nil {
			return err
		}
		ssmm.StatusUpdate(tc, &apps.StatefulSetStatus{})
		return nil
	}
	if err != nil {
		return err
	}

	ssmm.StatusUpdate(tc, &oldSet.Status)

	if !reflect.DeepEqual(oldSet.Spec, newSet.Spec) {
		set := *oldSet
		set.Spec = newSet.Spec
		err := ssmm.setControl.UpdateStatefulSet(tc, &set)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ssmm *StateSvcMemberManager) getNewServiceForTidbCluster(tc *v1.TidbCluster, svcConfig SvcConfig) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	svcName := svcConfig.MemberName(tcName)
	svcLabel := svcConfig.SvcLabel(label.New().Cluster(tcName)).Labels()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          svcLabel,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       svcConfig.Name,
					Port:       svcConfig.Port,
					TargetPort: intstr.FromInt(int(svcConfig.Port)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: svcLabel,
		},
	}
	if svcConfig.Headless {
		svc.Spec.ClusterIP = "None"
	} else {
		svc.Spec.Type = controller.GetServiceType(tc.Spec.Services, ssmm.MemberType.String())
	}
	return svc
}

func envVars(tcName, headlessSvcName, capacity string) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: "NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name:  "CLUSTER_NAME",
			Value: tcName,
		},
		{
			Name:  "HEADLESS_SERVICE_NAME",
			Value: headlessSvcName,
		},
		{
			Name:  "CAPACITY",
			Value: capacity,
		},
	}
}

func timezoneMountVolume() (corev1.VolumeMount, corev1.Volume) {
	return corev1.VolumeMount{Name: "timezone", MountPath: "/etc/localtime"},
		corev1.Volume{
			Name:         "timezone",
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/etc/localtime"}},
		}
}

func annotationsMountVolume() (corev1.VolumeMount, corev1.Volume) {
	m := corev1.VolumeMount{Name: "annotations", ReadOnly: true, MountPath: "/etc/podinfo"}
	v := corev1.Volume{
		Name: "annotations",
		VolumeSource: corev1.VolumeSource{
			DownwardAPI: &corev1.DownwardAPIVolumeSource{
				Items: []corev1.DownwardAPIVolumeFile{
					{
						Path:     "annotations",
						FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations"},
					},
				},
			},
		},
	}
	return m, v
}

func volumeClaimTemplate(q resource.Quantity, metaName string, storageClassName *string) corev1.PersistentVolumeClaim {
	return corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: metaName},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			StorageClassName: storageClassName,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: q,
				},
			},
		},
	}
}
