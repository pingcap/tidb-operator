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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util/label"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
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
	pdctl                   controller.PDControlInterface
	StatusUpdate            func(*v1.TidbCluster, *apps.StatefulSetStatus) error
	GetNewSetForTidbCluster func(*v1.TidbCluster) (*apps.StatefulSet, error)
	needReduce              func(*v1.TidbCluster, int32) bool
	removeOneMember         func(controller.PDControlInterface, *v1.TidbCluster, int32) error
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
		// TODO add unit test
		svc.Spec.ClusterIP = oldSvc.Spec.ClusterIP
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
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if errors.IsNotFound(err) {
		err = ssmm.setControl.CreateStatefulSet(tc, newSet)
		if err != nil {
			return err
		}
		return ssmm.StatusUpdate(tc, &apps.StatefulSetStatus{})
	}

	if err = ssmm.StatusUpdate(tc, &oldSet.Status); err != nil {
		return err
	}

	if ssmm.needReduce(tc, *oldSet.Spec.Replicas) {
		ordinal := *oldSet.Spec.Replicas - 1
		if err := ssmm.removeOneMember(ssmm.pdctl, tc, ordinal); err != nil {
			return err
		}
		set := *oldSet
		replicas := *oldSet.Spec.Replicas - 1 // we can only remove one member at a time
		set.Spec.Replicas = &replicas
		return ssmm.setControl.UpdateStatefulSet(tc, &set)
	}

	if !reflect.DeepEqual(oldSet.Spec, newSet.Spec) {
		set := *oldSet
		set.Spec = newSet.Spec
		return ssmm.setControl.UpdateStatefulSet(tc, &set)
	}

	return nil
}

func (ssmm *StateSvcMemberManager) getNewServiceForTidbCluster(tc *v1.TidbCluster, svcConfig SvcConfig) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	svcName := svcConfig.MemberName(tcName)
	svcLabel := svcConfig.SvcLabel(label.New().Cluster(tcName)).Labels()

	svc := corev1.Service{
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
	return &svc
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
	return corev1.VolumeMount{Name: "timezone", MountPath: "/etc/localtime", ReadOnly: true},
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
