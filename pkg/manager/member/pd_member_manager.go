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
	"github.com/golang/glog"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/listers/apps/v1beta1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

type pdMemberManager struct {
	pdControl      controller.PDControlInterface
	setControl     controller.StatefulSetControlInterface
	svcControl     controller.ServiceControlInterface
	setLister      v1beta1.StatefulSetLister
	svcLister      corelisters.ServiceLister
	podLister      corelisters.PodLister
	podControl     controller.PodControlInterface
	pvcLister      corelisters.PersistentVolumeClaimLister
	pdScaler       Scaler
	pdUpgrader     Upgrader
	autoFailover   bool
	pdFailover     Failover
	pdSetCreator   PDSetCreator
	pdStatusSyncer PDStatusSyncer
}

// NewPDMemberManager returns a *pdMemberManager
func NewPDMemberManager(pdControl controller.PDControlInterface,
	setControl controller.StatefulSetControlInterface,
	svcControl controller.ServiceControlInterface,
	setLister v1beta1.StatefulSetLister,
	svcLister corelisters.ServiceLister,
	podLister corelisters.PodLister,
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
		podControl,
		pvcLister,
		pdScaler,
		pdUpgrader,
		autoFailover,
		pdFailover,
		&pdSetCreator{setLister, pvcLister, setControl},
		&pdStatusSyncer{setLister, pdControl},
	}
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
	oldSvc, err := pmm.svcLister.Services(ns).Get(controller.PDMemberName(tcName, tc.Spec.PD.Name))
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
		// TODO add unit test
		svc.Spec.ClusterIP = oldSvc.Spec.ClusterIP
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

	if err := pmm.pdSetCreator.CreatePDStatefulSet(tc); err != nil {
		return err
	}
	if err := pmm.pdSetCreator.CreateExtraPDStatefulSets(tc); err != nil {
		return err
	}
	if err := pmm.pdStatusSyncer.SyncStatefulSetStatus(tc); err != nil {
		return err
	}
	if err := pmm.pdStatusSyncer.SyncExtraStatefulSetsStatus(tc); err != nil {
		return err
	}
	if err := pmm.pdStatusSyncer.SyncPDMembers(tc); err != nil {
		return err
	}

	var err error
	var newPDSet *apps.StatefulSet
	var oldPDSet *apps.StatefulSet
	if newPDSet, err = generateNewPDSetFrom(tc, tc.Spec.PD); err != nil {
		return err
	}
	if oldPDSet, err = pmm.setLister.StatefulSets(ns).Get(controller.PDMemberName(tcName, tc.Spec.PD.Name)); err != nil && !errors.IsNotFound(err) {
		return err
	}

	if err := pmm.syncTidbClusterStatus(tc, oldPDSet); err != nil {
		glog.Errorf("failed to sync TidbCluster: [%s/%s]'s status, error: %v", ns, tcName, err)
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

	// TODO FIXME equal is false every time
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

// TODO move this function to PDUpgrader
func (pmm *pdMemberManager) syncTidbClusterStatus(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) error {
	upgrading, err := pmm.pdStatefulSetIsUpgrading(set, tc)
	if err != nil {
		return err
	}
	if upgrading {
		tc.Status.PD.Phase = v1alpha1.UpgradePhase
	} else {
		tc.Status.PD.Phase = v1alpha1.NormalPhase
	}

	return nil
}

func (pmm *pdMemberManager) getNewPDServiceForTidbCluster(tc *v1alpha1.TidbCluster) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	svcName := controller.PDMemberName(tcName, tc.Spec.PD.Name)
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

type FakePDMemberManager struct {
	err error
}

func NewFakePDMemberManager() *FakePDMemberManager {
	return &FakePDMemberManager{}
}

func (fpmm *FakePDMemberManager) SetSyncError(err error) {
	fpmm.err = err
}

func (fpmm *FakePDMemberManager) Sync(_ *v1alpha1.TidbCluster) error {
	if fpmm.err != nil {
		return fpmm.err
	}
	return nil
}
