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
	"strconv"

	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

type pdMemberManager struct {
	pdControl    pdapi.PDControlInterface
	setControl   controller.StatefulSetControlInterface
	svcControl   controller.ServiceControlInterface
	setLister    v1.StatefulSetLister
	svcLister    corelisters.ServiceLister
	podLister    corelisters.PodLister
	epsLister    corelisters.EndpointsLister
	podControl   controller.PodControlInterface
	pvcLister    corelisters.PersistentVolumeClaimLister
	pdScaler     Scaler
	pdUpgrader   Upgrader
	autoFailover bool
	pdFailover   Failover
}

// NewPDMemberManager returns a *pdMemberManager
func NewPDMemberManager(pdControl pdapi.PDControlInterface,
	setControl controller.StatefulSetControlInterface,
	svcControl controller.ServiceControlInterface,
	setLister v1.StatefulSetLister,
	svcLister corelisters.ServiceLister,
	podLister corelisters.PodLister,
	epsLister corelisters.EndpointsLister,
	podControl controller.PodControlInterface,
	pvcLister corelisters.PersistentVolumeClaimLister,
	pdScaler Scaler,
	pdUpgrader Upgrader,
	autoFailover bool,
	pdFailover Failover) manager.Manager {
	return &pdMemberManager{
		pdControl,
		setControl,
		svcControl,
		setLister,
		svcLister,
		podLister,
		epsLister,
		podControl,
		pvcLister,
		pdScaler,
		pdUpgrader,
		autoFailover,
		pdFailover}
}

func (pmm *pdMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	// Sync PD Service
	if err := pmm.syncPDServiceForTidbCluster(tc); err != nil {
		return err
	}

	// Sync PD Headless Service
	if err := pmm.syncPDHeadlessServiceForTidbCluster(tc); err != nil {
		return err
	}

	// Sync PD StatefulSet
	return pmm.syncPDStatefulSetForTidbCluster(tc)
}

func (pmm *pdMemberManager) syncPDServiceForTidbCluster(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := pmm.getNewPDServiceForTidbCluster(tc)
	oldSvcTmp, err := pmm.svcLister.Services(ns).Get(controller.PDMemberName(tcName))
	if errors.IsNotFound(err) {
		err = SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return pmm.svcControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return err
	}

	oldSvc := oldSvcTmp.DeepCopy()

	equal, err := serviceEqual(newSvc, oldSvc)
	if err != nil {
		return err
	}
	if !equal {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		// TODO add unit test
		svc.Spec.ClusterIP = oldSvc.Spec.ClusterIP
		err = SetServiceLastAppliedConfigAnnotation(&svc)
		if err != nil {
			return err
		}
		_, err = pmm.svcControl.UpdateService(tc, &svc)
		return err
	}

	return nil
}

func (pmm *pdMemberManager) syncPDHeadlessServiceForTidbCluster(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := pmm.getNewPDHeadlessServiceForTidbCluster(tc)
	oldSvc, err := pmm.svcLister.Services(ns).Get(controller.PDPeerMemberName(tcName))
	if errors.IsNotFound(err) {
		err = SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return pmm.svcControl.CreateService(tc, newSvc)
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
		err = SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		_, err = pmm.svcControl.UpdateService(tc, &svc)
		return err
	}

	return nil
}

func (pmm *pdMemberManager) syncPDStatefulSetForTidbCluster(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newPDSet, err := pmm.getNewPDSetForTidbCluster(tc)
	if err != nil {
		return err
	}

	oldPDSetTmp, err := pmm.setLister.StatefulSets(ns).Get(controller.PDMemberName(tcName))
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if errors.IsNotFound(err) {
		err = SetLastAppliedConfigAnnotation(newPDSet)
		if err != nil {
			return err
		}
		if err := pmm.setControl.CreateStatefulSet(tc, newPDSet); err != nil {
			return err
		}
		tc.Status.PD.StatefulSet = &apps.StatefulSetStatus{}
		return controller.RequeueErrorf("TidbCluster: [%s/%s], waiting for PD cluster running", ns, tcName)
	}

	oldPDSet := oldPDSetTmp.DeepCopy()

	if err := pmm.syncTidbClusterStatus(tc, oldPDSet); err != nil {
		glog.Errorf("failed to sync TidbCluster: [%s/%s]'s status, error: %v", ns, tcName, err)
	}

	if !tc.Status.PD.Synced {
		force := needForceUpgrade(tc)
		if force {
			tc.Status.PD.Phase = v1alpha1.UpgradePhase
			setUpgradePartition(newPDSet, 0)
			errSTS := pmm.updateStatefulSet(tc, newPDSet, oldPDSet)
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s pd needs force upgrade, %v", ns, tcName, errSTS)
		}
	}

	if !templateEqual(newPDSet.Spec.Template, oldPDSet.Spec.Template) || tc.Status.PD.Phase == v1alpha1.UpgradePhase {
		if err := pmm.pdUpgrader.Upgrade(tc, oldPDSet, newPDSet); err != nil {
			return err
		}
	}

	if *newPDSet.Spec.Replicas > *oldPDSet.Spec.Replicas {
		if err := pmm.pdScaler.ScaleOut(tc, oldPDSet, newPDSet); err != nil {
			return err
		}
	}
	if *newPDSet.Spec.Replicas < *oldPDSet.Spec.Replicas {
		if err := pmm.pdScaler.ScaleIn(tc, oldPDSet, newPDSet); err != nil {
			return err
		}
	}

	if pmm.autoFailover {
		if tc.PDAllPodsStarted() && tc.PDAllMembersReady() && tc.Status.PD.FailureMembers != nil {
			pmm.pdFailover.Recover(tc)
		} else if tc.PDAllPodsStarted() && !tc.PDAllMembersReady() || tc.PDAutoFailovering() {
			if err := pmm.pdFailover.Failover(tc); err != nil {
				return err
			}
		}
	}

	return pmm.updateStatefulSet(tc, newPDSet, oldPDSet)
}
func (pmm *pdMemberManager) updateStatefulSet(tc *v1alpha1.TidbCluster, newPDSet, oldPDSet *apps.StatefulSet) error {
	if !statefulSetEqual(*newPDSet, *oldPDSet) {
		set := *oldPDSet
		set.Spec.Template = newPDSet.Spec.Template
		*set.Spec.Replicas = *newPDSet.Spec.Replicas
		set.Spec.UpdateStrategy = newPDSet.Spec.UpdateStrategy
		err := SetLastAppliedConfigAnnotation(&set)
		if err != nil {
			return err
		}
		_, err = pmm.setControl.UpdateStatefulSet(tc, &set)
		return err
	}

	return nil
}

func (pmm *pdMemberManager) syncTidbClusterStatus(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	tc.Status.PD.StatefulSet = &set.Status

	upgrading, err := pmm.pdStatefulSetIsUpgrading(set, tc)
	if err != nil {
		return err
	}
	if upgrading {
		tc.Status.PD.Phase = v1alpha1.UpgradePhase
	} else {
		tc.Status.PD.Phase = v1alpha1.NormalPhase
	}

	pdClient := controller.GetPDClient(pmm.pdControl, tc)

	healthInfo, err := pdClient.GetHealth()
	if err != nil {
		tc.Status.PD.Synced = false
		// get endpoints info
		eps, epErr := pmm.epsLister.Endpoints(ns).Get(controller.PDMemberName(tcName))
		if epErr != nil {
			return fmt.Errorf("%s, %s", err, epErr)
		}
		// pd service has no endpoints
		if eps != nil && len(eps.Subsets) == 0 {
			return fmt.Errorf("%s, service %s/%s has no endpoints", err, ns, controller.PDMemberName(tcName))
		}
		return err
	}

	cluster, err := pdClient.GetCluster()
	if err != nil {
		tc.Status.PD.Synced = false
		return err
	}
	tc.Status.ClusterID = strconv.FormatUint(cluster.Id, 10)
	leader, err := pdClient.GetPDLeader()
	if err != nil {
		tc.Status.PD.Synced = false
		return err
	}
	pdStatus := map[string]v1alpha1.PDMember{}
	for _, memberHealth := range healthInfo.Healths {
		id := memberHealth.MemberID
		memberID := fmt.Sprintf("%d", id)
		var clientURL string
		if len(memberHealth.ClientUrls) > 0 {
			clientURL = memberHealth.ClientUrls[0]
		}
		name := memberHealth.Name
		if len(name) == 0 {
			glog.Warningf("PD member: [%d] doesn't have a name, and can't get it from clientUrls: [%s], memberHealth Info: [%v] in [%s/%s]",
				id, memberHealth.ClientUrls, memberHealth, ns, tcName)
			continue
		}

		status := v1alpha1.PDMember{
			Name:      name,
			ID:        memberID,
			ClientURL: clientURL,
			Health:    memberHealth.Health,
		}

		oldPDMember, exist := tc.Status.PD.Members[name]

		status.LastTransitionTime = metav1.Now()
		if exist && status.Health == oldPDMember.Health {
			status.LastTransitionTime = oldPDMember.LastTransitionTime
		}

		pdStatus[name] = status
	}

	tc.Status.PD.Synced = true
	tc.Status.PD.Members = pdStatus
	tc.Status.PD.Leader = tc.Status.PD.Members[leader.GetName()]

	return nil
}

func (pmm *pdMemberManager) getNewPDServiceForTidbCluster(tc *v1alpha1.TidbCluster) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	svcName := controller.PDMemberName(tcName)
	instanceName := tc.GetLabels()[label.InstanceLabelKey]
	pdLabel := label.New().Instance(instanceName).PD().Labels()

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          pdLabel,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: corev1.ServiceSpec{
			Type: controller.GetServiceType(tc.Spec.Services, v1alpha1.PDMemberType.String()),
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

func (pmm *pdMemberManager) getNewPDHeadlessServiceForTidbCluster(tc *v1alpha1.TidbCluster) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	svcName := controller.PDPeerMemberName(tcName)
	instanceName := tc.GetLabels()[label.InstanceLabelKey]
	pdLabel := label.New().Instance(instanceName).PD().Labels()

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

func (pmm *pdMemberManager) pdStatefulSetIsUpgrading(set *apps.StatefulSet, tc *v1alpha1.TidbCluster) (bool, error) {
	if statefulSetIsUpgrading(set) {
		return true, nil
	}
	selector, err := label.New().
		Instance(tc.GetLabels()[label.InstanceLabelKey]).
		PD().
		Selector()
	if err != nil {
		return false, err
	}
	pdPods, err := pmm.podLister.Pods(tc.GetNamespace()).List(selector)
	if err != nil {
		return false, err
	}
	for _, pod := range pdPods {
		revisionHash, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return false, nil
		}
		if revisionHash != tc.Status.PD.StatefulSet.UpdateRevision {
			return true, nil
		}
	}
	return false, nil
}

func (pmm *pdMemberManager) getNewPDSetForTidbCluster(tc *v1alpha1.TidbCluster) (*apps.StatefulSet, error) {
	ns := tc.Namespace
	tcName := tc.Name
	instanceName := tc.GetLabels()[label.InstanceLabelKey]
	pdConfigMap := controller.MemberConfigMapName(tc, v1alpha1.PDMemberType)

	annMount, annVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		annMount,
		{Name: "config", ReadOnly: true, MountPath: "/etc/pd"},
		{Name: "startup-script", ReadOnly: true, MountPath: "/usr/local/bin"},
		{Name: v1alpha1.PDMemberType.String(), MountPath: "/var/lib/pd"},
	}
	if tc.Spec.EnableTLSCluster {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "pd-tls", ReadOnly: true, MountPath: "/var/lib/pd-tls",
		})
	}

	vols := []corev1.Volume{
		annVolume,
		{Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: pdConfigMap,
					},
					Items: []corev1.KeyToPath{{Key: "config-file", Path: "pd.toml"}},
				},
			},
		},
		{Name: "startup-script",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: pdConfigMap,
					},
					Items: []corev1.KeyToPath{{Key: "startup-script", Path: "pd_start_script.sh"}},
				},
			},
		},
	}
	if tc.Spec.EnableTLSCluster {
		vols = append(vols, corev1.Volume{
			Name: "pd-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: controller.PDMemberName(tcName),
				},
			},
		})
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
	pdLabel := label.New().Instance(instanceName).PD()
	setName := controller.PDMemberName(tcName)
	podAnnotations := CombineAnnotations(controller.AnnProm(2379), tc.Spec.PD.Annotations)
	storageClassName := tc.Spec.PD.StorageClassName
	if storageClassName == "" {
		storageClassName = controller.DefaultStorageClassName
	}
	failureReplicas := 0
	for _, failureMember := range tc.Status.PD.FailureMembers {
		if failureMember.MemberDeleted {
			failureReplicas++
		}
	}

	dnsPolicy := corev1.DNSClusterFirst // same as k8s defaults
	if tc.Spec.PD.HostNetwork {
		dnsPolicy = corev1.DNSClusterFirstWithHostNet
	}

	pdSet := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            setName,
			Namespace:       ns,
			Labels:          pdLabel.Labels(),
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: controller.Int32Ptr(tc.Spec.PD.Replicas + int32(failureReplicas)),
			Selector: pdLabel.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      pdLabel.Labels(),
					Annotations: podAnnotations,
				},
				Spec: corev1.PodSpec{
					SchedulerName: tc.Spec.SchedulerName,
					Affinity:      tc.Spec.PD.Affinity,
					NodeSelector:  tc.Spec.PD.NodeSelector,
					HostNetwork:   tc.Spec.PD.HostNetwork,
					DNSPolicy:     dnsPolicy,
					Containers: []corev1.Container{
						{
							Name:            v1alpha1.PDMemberType.String(),
							Image:           tc.Spec.PD.Image,
							Command:         []string{"/bin/sh", "/usr/local/bin/pd_start_script.sh"},
							ImagePullPolicy: tc.Spec.PD.ImagePullPolicy,
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
							Resources:    util.ResourceRequirement(tc.Spec.PD.ContainerSpec),
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
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name:  "PEER_SERVICE_NAME",
									Value: controller.PDPeerMemberName(tcName),
								},
								{
									Name:  "SERVICE_NAME",
									Value: controller.PDMemberName(tcName),
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
					RestartPolicy:     corev1.RestartPolicyAlways,
					Tolerations:       tc.Spec.PD.Tolerations,
					Volumes:           vols,
					SecurityContext:   tc.Spec.PD.PodSecurityContext,
					PriorityClassName: tc.Spec.PD.PriorityClassName,
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
					Partition: controller.Int32Ptr(tc.Spec.PD.Replicas + int32(failureReplicas)),
				}},
		},
	}

	return pdSet, nil
}

type FakePDMemberManager struct {
	err error
}

func NewFakePDMemberManager() *FakePDMemberManager {
	return &FakePDMemberManager{}
}

func (fpmm *FakePDMemberManager) SetSyncError(err error) {
	fpmm.err = err
}

func (fpmm *FakePDMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	if fpmm.err != nil {
		return fpmm.err
	}
	if len(tc.Status.PD.Members) != 0 {
		// simulate status update
		tc.Status.ClusterID = string(uuid.NewUUID())
	}
	return nil
}
