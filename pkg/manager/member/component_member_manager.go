// Copyright 2021 PingCAP, Inc.
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
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"k8s.io/utils/pointer"
)

// ComponentContext stores the variables from outside.
// Don't pass the internal variable with this struct.
type ComponentContext struct {
	tc           *v1alpha1.TidbCluster
	dependencies *controller.Dependencies
	component    string
}

func ComponentSyncStatefulSetForTidbCluster(context *ComponentContext) error {
	tc := context.tc
	dependencies := context.dependencies
	component := context.component

	ns := tc.GetNamespace()
	tcName := tc.GetName()
	componentMemberName := getComponentMemberName(context)

	oldSetTmp, err := dependencies.StatefulSetLister.StatefulSets(ns).Get(componentMemberName)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncPDStatefulSetForTidbCluster: fail to get sts %s for cluster %s/%s, error: %s", componentMemberName, ns, tcName, err)
	}
	setNotExist := errors.IsNotFound(err)

	oldSet := oldSetTmp.DeepCopy()

	if err := ComponentSyncTidbClusterStatus(context, oldSet); err != nil {
		klog.Errorf("failed to sync TidbCluster: [%s/%s]'s status, error: %v", ns, tcName, err)
	}

	if tc.Spec.Paused {
		klog.V(4).Infof("tidb cluster %s/%s is paused, skip syncing for %s statefulset", tc.GetNamespace(), tc.GetName(), component)
		return nil
	}

	cm, err := ComponentSyncConfigMap(context, oldSet)
	if err != nil {
		return err
	}

	if component == label.TiKVLabelVal {
		// Recover failed stores if any before generating desired statefulset
		if len(tc.Status.TiKV.FailureStores) > 0 {
			ComponentRemoveUndesiredFailures(context)
		}
		if len(tc.Status.TiKV.FailureStores) > 0 &&
			tc.Spec.TiKV.RecoverFailover &&
			ComponentShouldRecover(context) {
			ComponentRecover(context)
		}
	} else if component == label.TiFlashLabelVal {
		// Recover failed stores if any before generating desired statefulset
		if len(tc.Status.TiFlash.FailureStores) > 0 {
			ComponentRemoveUndesiredFailures(context)
		}
		if len(tc.Status.TiFlash.FailureStores) > 0 &&
			tc.Spec.TiFlash.RecoverFailover &&
			ComponentShouldRecover(context) {
			ComponentRecover(context)
		}
	}

	newSet, err := ComponentGetNewSetForTidbCluster(context, cm)
	if err != nil {
		return err
	}
	if setNotExist {
		err = SetStatefulSetLastAppliedConfigAnnotation(newSet)
		if err != nil {
			return err
		}

		if err := dependencies.StatefulSetControl.CreateStatefulSet(tc, newSet); err != nil {
			return err
		}

		if component != label.TiCDCLabelVal && component != label.PumpLabelVal {
			if err := syncNewComponentStatefulset(context); err != nil {
				return err
			}
		}

		return controller.RequeueErrorf("TidbCluster: [%s/%s], waiting for %s cluster running", ns, tcName, component)
	}

	switch component {
	case label.PDLabelVal:
		// Force update takes precedence over scaling because force upgrade won't take effect when cluster gets stuck at scaling
		if !tc.Status.PD.Synced && NeedForceUpgrade(tc.Annotations) {
			tc.Status.PD.Phase = v1alpha1.UpgradePhase
			setUpgradePartition(newSet, 0)
			errSTS := UpdateStatefulSet(dependencies.StatefulSetControl, tc, newSet, oldSet)
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s pd needs force upgrade, %v", ns, tcName, errSTS)
		}

		// Scaling takes precedence over upgrading because:
		// - if a pd fails in the upgrading, users may want to delete it or add
		//   new replicas
		// - it's ok to scale in the middle of upgrading (in statefulset controller
		//   scaling takes precedence over upgrading too)
		if err := ComponentScale(context, oldSet, newSet); err != nil {
			return err
		}

		if dependencies.CLIConfig.AutoFailover {
			if ComponentShouldRecover(context) {
				ComponentRecover(context)
			} else if tc.PDAllPodsStarted() && !tc.PDAllMembersReady() || tc.PDAutoFailovering() {
				if err := ComponentFailover(context); err != nil {
					return err
				}
			}
		}

		if !templateEqual(newSet, oldSet) || tc.Status.PD.Phase == v1alpha1.UpgradePhase {
			if err := ComponentUpgrade(context, oldSet, newSet); err != nil {
				return err
			}
		}
	case label.TiKVLabelVal:
		if _, err := setStoreLabelsForTiKV(context); err != nil {
			return err
		}

		// Scaling takes precedence over upgrading because:
		// - if a store fails in the upgrading, users may want to delete it or add
		//   new replicas
		// - it's ok to scale in the middle of upgrading (in statefulset controller
		//   scaling takes precedence over upgrading too)
		if err := ComponentScale(context, oldSet, newSet); err != nil {
			return err
		}

		// Perform failover logic if necessary. Note that this will only update
		// TidbCluster status. The actual scaling performs in next sync loop (if a
		// new replica needs to be added).
		if dependencies.CLIConfig.AutoFailover && tc.Spec.TiKV.MaxFailoverCount != nil {
			if tc.TiKVAllPodsStarted() && !tc.TiKVAllStoresReady() {
				if err := ComponentFailover(context); err != nil {
					return err
				}
			}
		}

		if !templateEqual(newSet, oldSet) || tc.Status.TiKV.Phase == v1alpha1.UpgradePhase {
			if err := ComponentUpgrade(context, oldSet, newSet); err != nil {
				return err
			}
		}
	case label.TiFlashLabelVal:
		// Scaling takes precedence over upgrading because:
		// - if a tiflash fails in the upgrading, users may want to delete it or add
		//   new replicas
		// - it's ok to scale in the middle of upgrading (in statefulset controller
		//   scaling takes precedence over upgrading too)
		if err := ComponentScale(context, oldSet, newSet); err != nil {
			return err
		}

		if dependencies.CLIConfig.AutoFailover && tc.Spec.TiFlash.MaxFailoverCount != nil {
			if tc.TiFlashAllPodsStarted() && !tc.TiFlashAllStoresReady() {
				if err := ComponentFailover(context); err != nil {
					return err
				}
			}
		}

		if !templateEqual(newSet, oldSet) {
			if err := ComponentUpgrade(context, oldSet, newSet); err != nil {
				return err
			}
		}
	case label.TiDBLabelVal:
		if dependencies.CLIConfig.AutoFailover {
			if ComponentShouldRecover(context) {
				ComponentRecover(context)
			} else if tc.TiDBAllPodsStarted() && !tc.TiDBAllMembersReady() {
				if err := ComponentFailover(context); err != nil {
					return err
				}
			}
		}
	case label.TiCDCLabelVal, label.PumpLabelVal:
		if tc.PDUpgrading() || tc.TiKVUpgrading() {
			klog.Warningf("pd or tikv is upgrading, skipping upgrade ticdc")
			return nil
		}
	}

	return UpdateStatefulSet(dependencies.StatefulSetControl, tc, newSet, oldSet)
}

func ComponentSyncTidbClusterStatus(context *ComponentContext, set *apps.StatefulSet) error {
	tc := context.tc
	component := context.component

	// skip if not created yet
	if set == nil {
		return nil
	}

	if err := syncExistedComponentStatefulset(context, set); err != nil {
		return err
	}
	if err := syncComponentPhase(context, set); err != nil {
		return err
	}
	if component != label.PumpLabelVal {
		if err := syncComponentMembers(context, set); err != nil {
			return err
		}
	}
	if component != label.TiFlashLabelVal && component != label.TiCDCLabelVal && component != label.PumpLabelVal {
		if err := syncComponentImage(context, set); err != nil {
			return err
		}
	}

	switch component {
	case label.PDLabelVal:
		// k8s check
		pdStatus := tc.Status.PD.Members
		if err := collectUnjoinedPDMembers(context, set, pdStatus); err != nil {
			return err
		}
	case label.TiKVLabelVal:
		tc.Status.TiKV.BootStrapped = true
	}
	return nil
}

func ComponentClusterVersionGreaterThanOrEqualTo4(version string) (bool, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return true, err
	}

	return v.Major() >= 4, nil
}

func collectUnjoinedPDMembers(context *ComponentContext, set *apps.StatefulSet, pdStatus map[string]v1alpha1.PDMember) error {
	tc := context.tc
	dependencies := context.dependencies

	podSelector, podSelectErr := metav1.LabelSelectorAsSelector(set.Spec.Selector)
	if podSelectErr != nil {
		return podSelectErr
	}
	pods, podErr := dependencies.PodLister.Pods(tc.Namespace).List(podSelector)
	if podErr != nil {
		return fmt.Errorf("collectUnjoinedMembers: failed to list pods for cluster %s/%s, selector %s, error %v", tc.GetNamespace(), tc.GetName(), set.Spec.Selector, podErr)
	}
	for _, pod := range pods {
		var joined = false
		for pdName := range pdStatus {
			ordinal, err := util.GetOrdinalFromPodName(pod.Name)
			if err != nil {
				return fmt.Errorf("unexpected pod name %q: %v", pod.Name, err)
			}
			if strings.EqualFold(PdName(tc.Name, ordinal, tc.Namespace, tc.Spec.ClusterDomain), pdName) {
				joined = true
				break
			}
		}
		if !joined {
			if tc.Status.PD.UnjoinedMembers == nil {
				tc.Status.PD.UnjoinedMembers = map[string]v1alpha1.UnjoinedMember{}
			}
			ordinal, err := util.GetOrdinalFromPodName(pod.Name)
			if err != nil {
				return err
			}
			pvcName := ordinalPVCName(v1alpha1.PDMemberType, controller.PDMemberName(tc.Name), ordinal)
			pvc, err := dependencies.PVCLister.PersistentVolumeClaims(tc.Namespace).Get(pvcName)
			if err != nil {
				return fmt.Errorf("collectUnjoinedMembers: failed to get pvc %s of cluster %s/%s, error %v", pvcName, tc.GetNamespace(), tc.GetName(), err)
			}
			tc.Status.PD.UnjoinedMembers[pod.Name] = v1alpha1.UnjoinedMember{
				PodName:   pod.Name,
				PVCUID:    pvc.UID,
				CreatedAt: metav1.Now(),
			}
		} else {
			if tc.Status.PD.UnjoinedMembers != nil {
				delete(tc.Status.PD.UnjoinedMembers, pod.Name)
			}
		}
	}
	return nil
}

// ComponentShouldRecover checks whether we should perform recovery operation.
func ComponentShouldRecover(context *ComponentContext) bool {
	tc := context.tc
	component := context.component
	dependencies := context.dependencies

	var pdMembers map[string]v1alpha1.PDMember
	var stores map[string]v1alpha1.TiKVStore
	var tidbMembers map[string]v1alpha1.TiDBMember
	var pdFailureMembers map[string]v1alpha1.PDFailureMember
	var failureStores map[string]v1alpha1.TiKVFailureStore
	var tidbFailureMembers map[string]v1alpha1.TiDBFailureMember
	var ordinals sets.Int32
	var podPrefix string

	switch component {
	case label.TiKVLabelVal:
		stores = tc.Status.TiKV.Stores
		failureStores = tc.Status.TiKV.FailureStores
		if failureStores == nil {
			return false
		}
		ordinals = tc.TiKVStsDesiredOrdinals(true)
		podPrefix = controller.TiKVMemberName(tc.Name)
	case label.TiFlashLabelVal:
		stores = tc.Status.TiFlash.Stores
		failureStores = tc.Status.TiFlash.FailureStores
		if failureStores == nil {
			return false
		}
		ordinals = tc.TiFlashStsDesiredOrdinals(true)
		podPrefix = controller.TiFlashMemberName(tc.Name)
	case label.PDLabelVal:
		pdMembers = tc.Status.PD.Members
		pdFailureMembers = tc.Status.PD.FailureMembers
		if pdFailureMembers == nil {
			return false
		}
		ordinals = tc.PDStsDesiredOrdinals(true)
		podPrefix = controller.PDMemberName(tc.Name)
	case label.TiDBLabelVal:
		tidbMembers = tc.Status.TiDB.Members
		tidbFailureMembers = tc.Status.TiDB.FailureMembers
		if tidbFailureMembers == nil {
			return false
		}
		ordinals = tc.TiDBStsDesiredOrdinals(true)
		podPrefix = controller.TiDBMemberName(tc.Name)
	default:
		klog.Warningf("Unexpected component %s for %s/%s in shouldRecover", component, tc.Namespace, tc.Name)
		return false
	}

	// If all desired replicas (excluding failover pods) of tidb cluster are
	// healthy, we can perform our failover recovery operation.
	// Note that failover pods may fail (e.g. lack of resources) and we don't care
	// about them because we're going to delete them.
	for ordinal := range ordinals {
		name := fmt.Sprintf("%s-%d", podPrefix, ordinal)
		pod, err := dependencies.PodLister.Pods(tc.Namespace).Get(name)
		if err != nil {
			klog.Errorf("pod %s/%s does not exist: %v", tc.Namespace, name, err)
			return false
		}
		if !podutil.IsPodReady(pod) {
			return false
		}
		var exist bool
		switch component {
		case label.TiKVLabelVal, label.TiFlashLabelVal:
			for _, v := range stores {
				if v.PodName == pod.Name {
					exist = true
					if v.State != v1alpha1.TiKVStateUp {
						return false
					}
				}
			}
		case label.PDLabelVal:
			for pdName, pdMember := range pdMembers {
				if strings.Split(pdName, ".")[0] == pod.Name {
					if !pdMember.Health {
						return false
					}
					exist = true
					break
				}
			}
		case label.TiDBLabelVal:
			status, ok := tidbMembers[pod.Name]
			if !ok || !status.Health {
				return false
			}
		}

		if !exist && component != label.TiDBLabelVal {
			return false
		}
	}
	return true
}

// sync service

func componentSyncGeneralServiceForTidbCluster(context *ComponentContext, isHeadless bool) error {
	tc := context.tc
	dependencies := context.dependencies
	component := context.component

	if tc.Spec.Paused {
		klog.V(4).Infof("tidb cluster %s/%s is paused, skip syncing for %s service", tc.GetNamespace(), tc.GetName(), component)
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()
	componentMemberName := getComponentMemberName(context)

	var newSvc *corev1.Service
	if isHeadless {
		newSvc = ComponentGetNewServiceForTidbCluster(context)
	} else {
		newSvc = ComponentGetNewHeadlessServiceForTidbCluster(context)
	}

	oldSvcTmp, err := dependencies.ServiceLister.Services(ns).Get(componentMemberName)
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return dependencies.ServiceControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return fmt.Errorf("syncServiceForTidbCluster: failed to get svc %s for cluster %s/%s, error: %s", componentMemberName, ns, tcName, err)
	}

	oldSvc := oldSvcTmp.DeepCopy()

	equal, err := controller.ServiceEqual(newSvc, oldSvc)
	if err != nil {
		return err
	}
	if !equal {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		err = controller.SetServiceLastAppliedConfigAnnotation(&svc)
		if err != nil {
			return err
		}
		svc.Spec.ClusterIP = oldSvc.Spec.ClusterIP
		_, err = dependencies.ServiceControl.UpdateService(tc, &svc)
		return err
	}

	return nil
}

func ComponentSyncServiceForTidbCluster(context *ComponentContext) error {
	isHeadless := false
	return componentSyncGeneralServiceForTidbCluster(context, isHeadless)
}

func ComponentSyncHeadlessServiceForTidbCluster(context *ComponentContext) error {
	isHeadless := true
	return componentSyncGeneralServiceForTidbCluster(context, isHeadless)
}

func componentGetNewGeneralServiceForTidbCluster(context *ComponentContext, isHeadless bool) *corev1.Service {
	tc := context.tc
	component := context.component

	// Common pre handling
	ns := tc.Namespace
	tcName := tc.Name
	instanceName := tc.GetInstanceName()
	var componentService *corev1.Service

	if isHeadless {
		switch component {
		case label.PDLabelVal:
			svcName := controller.PDPeerMemberName(tcName)
			pdSelector := label.New().Instance(instanceName).PD()
			pdLabels := pdSelector.Copy().UsedByPeer().Labels()
			componentService = &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:            svcName,
					Namespace:       ns,
					Labels:          pdLabels,
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
					Selector:                 pdSelector.Labels(),
					PublishNotReadyAddresses: true,
				},
			}
		case label.TiFlashLabelVal:
			svcName := controller.TiFlashPeerMemberName(tcName)
			svcLabel := label.New().Instance(instanceName).TiFlash().Labels()

			componentService = &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:            svcName,
					Namespace:       ns,
					Labels:          svcLabel,
					OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
				},
				Spec: corev1.ServiceSpec{
					ClusterIP: "None",
					Ports: []corev1.ServicePort{
						{
							Name:       "tiflash",
							Port:       3930,
							TargetPort: intstr.FromInt(int(3930)),
							Protocol:   corev1.ProtocolTCP,
						},
						{
							Name:       "proxy",
							Port:       20170,
							TargetPort: intstr.FromInt(int(20170)),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector:                 svcLabel,
					PublishNotReadyAddresses: true,
				},
			}
		}
	} else {
		switch component {
		case label.PDLabelVal:
			svcName := controller.PDMemberName(tcName)
			svcSelector := label.New().Instance(instanceName).PD()
			svcLabels := svcSelector.Copy().UsedByEndUser().Labels()
			componentService := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:            svcName,
					Namespace:       ns,
					Labels:          svcLabels,
					OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name:       "client",
							Port:       2379,
							TargetPort: intstr.FromInt(2379),
							Protocol:   corev1.ProtocolTCP,
						},
					},
					Selector: svcSelector.Labels(),
				},
			}

			componentService.Spec.Type = controller.GetServiceType(tc.Spec.Services, v1alpha1.PDMemberType.String())

			svcSpec := tc.Spec.PD.Service

			if svcSpec != nil {
				if svcSpec.Type != "" {
					componentService.Spec.Type = svcSpec.Type
				}
				componentService.ObjectMeta.Annotations = CopyAnnotations(svcSpec.Annotations)
				if svcSpec.LoadBalancerIP != nil {
					componentService.Spec.LoadBalancerIP = *svcSpec.LoadBalancerIP
				}
				if svcSpec.ClusterIP != nil {
					componentService.Spec.ClusterIP = *svcSpec.ClusterIP
				}
				if svcSpec.PortName != nil {
					componentService.Spec.Ports[0].Name = *svcSpec.PortName
				}
			}
		case label.TiKVLabelVal:
			svcConfig := SvcConfig{
				Name:       "peer",
				Port:       20160,
				Headless:   true,
				SvcLabel:   func(l label.Label) label.Label { return l.TiKV() },
				MemberName: controller.TiKVPeerMemberName,
			}

			svcName := svcConfig.MemberName(tcName)
			svcSelector := svcConfig.SvcLabel(label.New().Instance(instanceName))
			svcLabel := svcSelector.Copy()
			if svcConfig.Headless {
				svcLabel = svcLabel.UsedByPeer()
			} else {
				svcLabel = svcLabel.UsedByEndUser()
			}

			componentService := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:            svcName,
					Namespace:       ns,
					Labels:          svcLabel.Labels(),
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
					Selector:                 svcSelector.Labels(),
					PublishNotReadyAddresses: true,
				},
			}
			if svcConfig.Headless {
				componentService.Spec.ClusterIP = "None"
			} else {
				componentService.Spec.Type = controller.GetServiceType(tc.Spec.Services, v1alpha1.TiKVMemberType.String())
			}
		case label.TiDBLabelVal:
			svcSpec := tc.Spec.TiDB.Service
			if svcSpec == nil {
				return nil
			}

			tidbSelector := label.New().Instance(instanceName).TiDB()
			svcName := controller.TiDBMemberName(tcName)
			tidbLabels := tidbSelector.Copy().UsedByEndUser().Labels()
			portName := "mysql-client"
			if svcSpec.PortName != nil {
				portName = *svcSpec.PortName
			}
			ports := []corev1.ServicePort{
				{
					Name:       portName,
					Port:       4000,
					TargetPort: intstr.FromInt(4000),
					Protocol:   corev1.ProtocolTCP,
					NodePort:   svcSpec.GetMySQLNodePort(),
				},
			}
			ports = append(ports, tc.Spec.TiDB.Service.AdditionalPorts...)
			if svcSpec.ShouldExposeStatus() {
				ports = append(ports, corev1.ServicePort{
					Name:       "status",
					Port:       10080,
					TargetPort: intstr.FromInt(10080),
					Protocol:   corev1.ProtocolTCP,
					NodePort:   svcSpec.GetStatusNodePort(),
				})
			}

			componentService := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:            svcName,
					Namespace:       ns,
					Labels:          tidbLabels,
					Annotations:     CopyAnnotations(svcSpec.Annotations),
					OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
				},
				Spec: corev1.ServiceSpec{
					Type:     svcSpec.Type,
					Ports:    ports,
					Selector: tidbSelector.Labels(),
				},
			}
			if svcSpec.Type == corev1.ServiceTypeLoadBalancer {
				if svcSpec.LoadBalancerIP != nil {
					componentService.Spec.LoadBalancerIP = *svcSpec.LoadBalancerIP
				}
				if svcSpec.LoadBalancerSourceRanges != nil {
					componentService.Spec.LoadBalancerSourceRanges = svcSpec.LoadBalancerSourceRanges
				}
			}
			if svcSpec.ExternalTrafficPolicy != nil {
				componentService.Spec.ExternalTrafficPolicy = *svcSpec.ExternalTrafficPolicy
			}
			if svcSpec.ClusterIP != nil {
				componentService.Spec.ClusterIP = *svcSpec.ClusterIP
			}
		}
	}
	return componentService
}

func ComponentGetNewServiceForTidbCluster(context *ComponentContext) *corev1.Service {
	isHeadless := false
	return componentGetNewGeneralServiceForTidbCluster(context, isHeadless)
}

func ComponentGetNewHeadlessServiceForTidbCluster(context *ComponentContext) *corev1.Service {
	isHeadless := true
	return componentGetNewGeneralServiceForTidbCluster(context, isHeadless)
}

func ComponentStatefulSetIsUpgrading(set *apps.StatefulSet, context *ComponentContext) (bool, error) {
	tc := context.tc
	dependencies := context.dependencies

	if statefulSetIsUpgrading(set) {
		return true, nil
	}
	instanceName := tc.GetInstanceName()

	componentLabel := getComponentLabel(context, instanceName)
	selector, err := componentLabel.Selector()

	if err != nil {
		return false, err
	}
	componentPods, err := dependencies.PodLister.Pods(tc.GetNamespace()).List(selector)
	if err != nil {
		return false, fmt.Errorf("StatefulSetIsUpgrading: failed to list pods for cluster %s/%s, selector %s, error: %v", tc.GetNamespace(), instanceName, selector, err)
	}

	for _, pod := range componentPods {
		revisionHash, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return false, nil
		}

		componentUpdateRevision := getComponentUpdataRevision(context)

		if revisionHash != componentUpdateRevision {
			return true, nil
		}
	}
	return false, nil
}

func ComponentGetNewSetForTidbCluster(context *ComponentContext, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
	component := context.component

	switch component {
	case label.PDLabelVal:
		return componentPDGetNewSetForTiDBCluster(context, cm)
	case label.TiKVLabelVal:
		return componentTiKVGetNewSetForTiDBCluster(context, cm)
	case label.TiFlashLabelVal:
		return componentTiFlashGetNewSetForTiDBCluster(context, cm)
	case label.TiDBLabelVal:
		return componentTiDBGetNewSetForTiDBCluster(context, cm)
	case label.TiCDCLabelVal:
		return componentTiCDCGetNewSetForTiDBCluster(context, cm)
	case label.PumpLabelVal:
		return componentPumpGetNewSetForTiDBCluster(context, cm)
	}

	return &apps.StatefulSet{}, nil
}

// ComponentSyncConfigMap syncs the configmap
func ComponentSyncConfigMap(context *ComponentContext, set *apps.StatefulSet) (*corev1.ConfigMap, error) {
	tc := context.tc
	dependencies := context.dependencies
	component := context.component

	componentMemberName := getComponentMemberName(context)

	var componentConfigUpdateStrategy v1alpha1.ConfigUpdateStrategy
	// For backward compatibility, only sync tidb configmap when .config is non-nil
	switch component {
	case label.PDLabelVal:
		if tc.Spec.PD.Config == nil {
			return nil, nil
		}
		componentConfigUpdateStrategy = tc.BasePDSpec().ConfigUpdateStrategy()
	case label.TiKVLabelVal:
		if tc.Spec.TiKV.Config == nil {
			return nil, nil
		}
		componentConfigUpdateStrategy = tc.BasePDSpec().ConfigUpdateStrategy()
	case label.TiFlashLabelVal:
		if tc.Spec.TiFlash.Config == nil {
			return nil, nil
		}
		componentConfigUpdateStrategy = tc.BasePDSpec().ConfigUpdateStrategy()
	case label.TiDBLabelVal:
		if tc.Spec.TiDB.Config == nil {
			return nil, nil
		}
		componentConfigUpdateStrategy = tc.BasePDSpec().ConfigUpdateStrategy()
	case label.PumpLabelVal:
		basePumpSpec, createPump := tc.BasePumpSpec()
		if !createPump {
			return nil, nil
		}
		componentConfigUpdateStrategy = basePumpSpec.ConfigUpdateStrategy()
	}

	newCm, err := ComponentGetConfigMap(context)
	if err != nil {
		return nil, err
	}

	var inUseName string
	if set != nil {
		inUseName = FindConfigMapVolume(&set.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, componentMemberName)
		})
	}

	err = updateConfigMapIfNeed(dependencies.ConfigMapLister, componentConfigUpdateStrategy, inUseName, newCm)
	if err != nil {
		return nil, err
	}
	return dependencies.TypedControl.CreateOrUpdateConfigMap(tc, newCm)
}

func ComponentGetConfigMap(context *ComponentContext) (*corev1.ConfigMap, error) {
	tc := context.tc
	component := context.component

	var cm *corev1.ConfigMap
	switch component {
	case label.PDLabelVal:
		config := tc.Spec.PD.Config
		if config == nil {
			return nil, nil
		}

		clusterVersionGE4, err := clusterVersionGreaterThanOrEqualTo4(tc.PDVersion())
		if err != nil {
			klog.V(4).Infof("cluster version: %s is not semantic versioning compatible", tc.PDVersion())
		}

		// override CA if tls enabled
		if tc.IsTLSClusterEnabled() {
			config.Set("security.cacert-path", path.Join(pdClusterCertPath, tlsSecretRootCAKey))
			config.Set("security.cert-path", path.Join(pdClusterCertPath, corev1.TLSCertKey))
			config.Set("security.key-path", path.Join(pdClusterCertPath, corev1.TLSPrivateKeyKey))
		}
		// Versions below v4.0 do not support Dashboard
		if tc.Spec.TiDB.IsTLSClientEnabled() && !tc.SkipTLSWhenConnectTiDB() && clusterVersionGE4 {
			config.Set("dashboard.tidb-cacert-path", path.Join(tidbClientCertPath, tlsSecretRootCAKey))
			config.Set("dashboard.tidb-cert-path", path.Join(tidbClientCertPath, corev1.TLSCertKey))
			config.Set("dashboard.tidb-key-path", path.Join(tidbClientCertPath, corev1.TLSPrivateKeyKey))
		}

		if tc.Spec.PD.EnableDashboardInternalProxy != nil {
			config.Set("dashboard.internal-proxy", *tc.Spec.PD.EnableDashboardInternalProxy)
		}

		confText, err := config.MarshalTOML()
		if err != nil {
			return nil, err
		}
		startScript, err := RenderPDStartScript(&PDStartScriptModel{
			Scheme:        tc.Scheme(),
			DataDir:       filepath.Join(pdDataVolumeMountPath, tc.Spec.PD.DataSubDir),
			ClusterDomain: tc.Spec.ClusterDomain,
		})
		if err != nil {
			return nil, err
		}

		instanceName := tc.GetInstanceName()
		pdLabel := label.New().Instance(instanceName).PD().Labels()
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:            controller.PDMemberName(tc.Name),
				Namespace:       tc.Namespace,
				Labels:          pdLabel,
				OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
			},
			Data: map[string]string{
				"config-file":    string(confText),
				"startup-script": startScript,
			},
		}
	case label.TiKVLabelVal:
		config := tc.Spec.TiKV.Config
		if config == nil {
			return nil, nil
		}

		scriptModel := &TiKVStartScriptModel{
			EnableAdvertiseStatusAddr: false,
			DataDir:                   filepath.Join(tikvDataVolumeMountPath, tc.Spec.TiKV.DataSubDir),
			ClusterDomain:             tc.Spec.ClusterDomain,
		}
		if tc.Spec.EnableDynamicConfiguration != nil && *tc.Spec.EnableDynamicConfiguration {
			scriptModel.AdvertiseStatusAddr = "${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc" + controller.FormatClusterDomain(tc.Spec.ClusterDomain)
			scriptModel.EnableAdvertiseStatusAddr = true
		}

		if tc.IsHeterogeneous() {
			scriptModel.PDAddress = tc.Scheme() + "://" + controller.PDMemberName(tc.Spec.Cluster.Name) + ":2379"
		} else {
			scriptModel.PDAddress = tc.Scheme() + "://${CLUSTER_NAME}-pd:2379"
		}
		cm, err := getTikVConfigMapForTiKVSpec(tc.Spec.TiKV, tc, scriptModel)
		if err != nil {
			return nil, err
		}
		instanceName := tc.GetInstanceName()
		tikvLabel := label.New().Instance(instanceName).TiKV().Labels()
		cm.ObjectMeta = metav1.ObjectMeta{
			Name:            controller.TiKVMemberName(tc.Name),
			Namespace:       tc.Namespace,
			Labels:          tikvLabel,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		}
	case label.TiFlashLabelVal:
		config := getTiFlashConfig(tc)

		configText, err := config.Common.MarshalTOML()
		if err != nil {
			return nil, err
		}
		proxyText, err := config.Proxy.MarshalTOML()
		if err != nil {
			return nil, err
		}

		instanceName := tc.GetInstanceName()
		tiflashLabel := label.New().Instance(instanceName).TiFlash().Labels()
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:            controller.TiFlashMemberName(tc.Name),
				Namespace:       tc.Namespace,
				Labels:          tiflashLabel,
				OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
			},
			Data: map[string]string{
				"config_templ.toml": string(configText),
				"proxy_templ.toml":  string(proxyText),
			},
		}
	case label.TiDBLabelVal:
		config := tc.Spec.TiDB.Config
		if config == nil {
			return nil, nil
		}

		// override CA if tls enabled
		if tc.IsTLSClusterEnabled() {
			config.Set("security.cluster-ssl-ca", path.Join(clusterCertPath, tlsSecretRootCAKey))
			config.Set("security.cluster-ssl-cert", path.Join(clusterCertPath, corev1.TLSCertKey))
			config.Set("security.cluster-ssl-key", path.Join(clusterCertPath, corev1.TLSPrivateKeyKey))
		}
		if tc.Spec.TiDB.IsTLSClientEnabled() {
			config.Set("security.ssl-ca", path.Join(serverCertPath, tlsSecretRootCAKey))
			config.Set("security.ssl-cert", path.Join(serverCertPath, corev1.TLSCertKey))
			config.Set("security.ssl-key", path.Join(serverCertPath, corev1.TLSPrivateKeyKey))
		}
		confText, err := config.MarshalTOML()
		if err != nil {
			return nil, err
		}

		plugins := tc.Spec.TiDB.Plugins
		tidbStartScriptModel := &TidbStartScriptModel{
			EnablePlugin:    len(plugins) > 0,
			PluginDirectory: "/plugins",
			PluginList:      strings.Join(plugins, ","),
			ClusterDomain:   tc.Spec.ClusterDomain,
		}

		if tc.IsHeterogeneous() {
			tidbStartScriptModel.Path = controller.PDMemberName(tc.Spec.Cluster.Name) + ":2379"
		} else {
			tidbStartScriptModel.Path = "${CLUSTER_NAME}-pd:2379"
		}

		startScript, err := RenderTiDBStartScript(tidbStartScriptModel)
		if err != nil {
			return nil, err
		}
		data := map[string]string{
			"config-file":    string(confText),
			"startup-script": startScript,
		}
		name := controller.TiDBMemberName(tc.Name)
		instanceName := tc.GetInstanceName()
		tidbLabels := label.New().Instance(instanceName).TiDB().Labels()

		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:            name,
				Namespace:       tc.Namespace,
				Labels:          tidbLabels,
				OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
			},
			Data: data,
		}
	}

	return cm, nil
}

func syncComponentPhase(context *ComponentContext, set *apps.StatefulSet) error {
	tc := context.tc
	component := context.component

	var phase v1alpha1.MemberPhase
	var desiredReplicas int32

	switch component {
	case label.PDLabelVal:
		desiredReplicas = tc.PDStsDesiredReplicas()
	case label.TiKVLabelVal:
		desiredReplicas = tc.TiKVStsDesiredReplicas()
	case label.TiFlashLabelVal:
		desiredReplicas = tc.TiFlashStsDesiredReplicas()
	case label.TiDBLabelVal:
		desiredReplicas = tc.TiDBStsDesiredReplicas()
	}

	upgrading, err := ComponentStatefulSetIsUpgrading(set, context)
	if err != nil {
		return err
	}

	// Scaling takes precedence over upgrading.
	switch component {
	case label.PDLabelVal:
		if desiredReplicas != *set.Spec.Replicas {
			phase = v1alpha1.ScalePhase
		} else if upgrading {
			phase = v1alpha1.UpgradePhase
		} else {
			phase = v1alpha1.NormalPhase
		}
		tc.Status.PD.Phase = phase
	case label.TiFlashLabelVal:
		if desiredReplicas != *set.Spec.Replicas {
			phase = v1alpha1.ScalePhase
		} else if upgrading {
			phase = v1alpha1.UpgradePhase
		} else {
			phase = v1alpha1.NormalPhase
		}
		tc.Status.TiFlash.Phase = phase
	case label.TiKVLabelVal:
		if desiredReplicas != *set.Spec.Replicas {
			phase = v1alpha1.ScalePhase
		} else if upgrading && tc.Status.PD.Phase != v1alpha1.UpgradePhase {
			phase = v1alpha1.UpgradePhase
		} else {
			phase = v1alpha1.NormalPhase
		}
		tc.Status.TiKV.Phase = phase
	case label.TiDBLabelVal:
		if desiredReplicas != *set.Spec.Replicas {
			phase = v1alpha1.ScalePhase
		} else if upgrading && tc.Status.TiKV.Phase != v1alpha1.UpgradePhase &&
			tc.Status.PD.Phase != v1alpha1.UpgradePhase && tc.Status.Pump.Phase != v1alpha1.UpgradePhase {
			phase = v1alpha1.UpgradePhase
		} else {
			phase = v1alpha1.NormalPhase
		}
		tc.Status.TiDB.Phase = phase
	case label.TiCDCLabelVal:
		if upgrading {
			phase = v1alpha1.UpgradePhase
		} else {
			phase = v1alpha1.NormalPhase
		}
		tc.Status.TiCDC.Phase = phase
	case label.PumpLabelVal:
		if upgrading {
			phase = v1alpha1.UpgradePhase
		} else {
			phase = v1alpha1.NormalPhase
		}
		tc.Status.Pump.Phase = phase
	}

	return nil
}

func syncComponentMembers(context *ComponentContext, set *apps.StatefulSet) error {
	tc := context.tc
	component := context.component
	dependencies := context.dependencies

	ns := tc.GetNamespace()
	tcName := tc.Name
	switch component {
	case label.PDLabelVal:
		pdClient := controller.GetPDClient(dependencies.PDControl, tc)

		healthInfo, err := pdClient.GetHealth()
		if err != nil {
			tc.Status.PD.Synced = false
			// get endpoints info
			eps, epErr := dependencies.EndpointLister.Endpoints(ns).Get(controller.PDMemberName(tcName))
			if epErr != nil {
				return fmt.Errorf("syncTidbClusterStatus: failed to get endpoints %s for cluster %s/%s, err: %s, epErr %s", controller.PDMemberName(tcName), ns, tcName, err, epErr)
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

		pattern, err := regexp.Compile(fmt.Sprintf(pdMemberLimitPattern, tc.Name, tc.Name, tc.Namespace, controller.FormatClusterDomainForRegex(tc.Spec.ClusterDomain)))
		if err != nil {
			return err
		}
		pdStatus := map[string]v1alpha1.PDMember{}
		peerPDStatus := map[string]v1alpha1.PDMember{}
		for _, memberHealth := range healthInfo.Healths {
			id := memberHealth.MemberID
			memberID := fmt.Sprintf("%d", id)
			var clientURL string
			if len(memberHealth.ClientUrls) > 0 {
				clientURL = memberHealth.ClientUrls[0]
			}
			name := memberHealth.Name
			if len(name) == 0 {
				klog.Warningf("PD member: [%d] doesn't have a name, and can't get it from clientUrls: [%s], memberHealth Info: [%v] in [%s/%s]",
					id, memberHealth.ClientUrls, memberHealth, tc.GetNamespace(), tc.GetName())
				continue
			}

			status := v1alpha1.PDMember{
				Name:      name,
				ID:        memberID,
				ClientURL: clientURL,
				Health:    memberHealth.Health,
			}
			status.LastTransitionTime = metav1.Now()

			if pattern.Match([]byte(clientURL)) {
				oldPDMember, exist := tc.Status.PD.Members[name]
				if exist && status.Health == oldPDMember.Health {
					status.LastTransitionTime = oldPDMember.LastTransitionTime
				}
				pdStatus[name] = status
			} else {
				oldPDMember, exist := tc.Status.PD.PeerMembers[name]
				if exist && status.Health == oldPDMember.Health {
					status.LastTransitionTime = oldPDMember.LastTransitionTime
				}
				peerPDStatus[name] = status
			}

			if name == leader.GetName() {
				tc.Status.PD.Leader = status
			}
		}
		tc.Status.PD.Synced = true
		tc.Status.PD.Members = pdStatus
		tc.Status.PD.PeerMembers = peerPDStatus
	case label.TiKVLabelVal:
		previousStores := tc.Status.TiKV.Stores
		previousPeerStores := tc.Status.TiKV.PeerStores
		stores := map[string]v1alpha1.TiKVStore{}
		peerStores := map[string]v1alpha1.TiKVStore{}
		tombstoneStores := map[string]v1alpha1.TiKVStore{}

		pdClient := controller.GetPDClient(dependencies.PDControl, tc)
		// This only returns Up/Down/Offline stores
		storesInfo, err := pdClient.GetStores()
		if err != nil {
			if pdapi.IsTiKVNotBootstrappedError(err) {
				klog.Infof("TiKV of Cluster %s/%s not bootstrapped yet", tc.Namespace, tc.Name)
				tc.Status.TiKV.Synced = true
				tc.Status.TiKV.BootStrapped = false
				return nil
			}
			tc.Status.TiKV.Synced = false
			return err
		}

		pattern, err := regexp.Compile(fmt.Sprintf(tikvStoreLimitPattern, tc.Name, tc.Name, tc.Namespace, controller.FormatClusterDomainForRegex(tc.Spec.ClusterDomain)))
		if err != nil {
			return err
		}
		for _, store := range storesInfo.Stores {
			status := getTiKVStore(store)
			if status == nil {
				continue
			}

			oldStore, exist := previousStores[status.ID]
			if !exist {
				oldStore, exist = previousPeerStores[status.ID]
			}

			// avoid LastHeartbeatTime be overwrite by zero time when pd lost LastHeartbeatTime
			if status.LastHeartbeatTime.IsZero() && exist {
				klog.V(4).Infof("the pod:%s's store LastHeartbeatTime is zero,so will keep in %v", status.PodName, oldStore.LastHeartbeatTime)
				status.LastHeartbeatTime = oldStore.LastHeartbeatTime
			}

			status.LastTransitionTime = metav1.Now()
			if exist && status.State == oldStore.State {
				status.LastTransitionTime = oldStore.LastTransitionTime
			}

			// In theory, the external tikv can join the cluster, and the operator would only manage the internal tikv.
			// So we check the store owner to make sure it.
			if store.Store != nil {
				if pattern.Match([]byte(store.Store.Address)) {
					stores[status.ID] = *status
				} else if util.MatchLabelFromStoreLabels(store.Store.Labels, label.TiKVLabelVal) {
					peerStores[status.ID] = *status
				}
			}
		}

		//this returns all tombstone stores
		tombstoneStoresInfo, err := pdClient.GetTombStoneStores()
		if err != nil {
			tc.Status.TiKV.Synced = false
			return err
		}
		for _, store := range tombstoneStoresInfo.Stores {
			if store.Store != nil && !pattern.Match([]byte(store.Store.Address)) {
				continue
			}
			status := getTiKVStore(store)
			if status == nil {
				continue
			}
			tombstoneStores[status.ID] = *status
		}

		tc.Status.TiKV.Synced = true
		tc.Status.TiKV.Stores = stores
		tc.Status.TiKV.PeerStores = peerStores
		tc.Status.TiKV.TombstoneStores = tombstoneStores
	case label.TiFlashLabelVal:
		previousStores := tc.Status.TiFlash.Stores
		previousPeerStores := tc.Status.TiFlash.PeerStores
		stores := map[string]v1alpha1.TiKVStore{}
		peerStores := map[string]v1alpha1.TiKVStore{}
		tombstoneStores := map[string]v1alpha1.TiKVStore{}

		pdClient := controller.GetPDClient(dependencies.PDControl, tc)
		// This only returns Up/Down/Offline stores
		storesInfo, err := pdClient.GetStores()
		if err != nil {
			tc.Status.TiFlash.Synced = false
			return err
		}

		pattern, err := regexp.Compile(fmt.Sprintf(tiflashStoreLimitPattern, tc.Name, tc.Name, tc.Namespace, controller.FormatClusterDomainForRegex(tc.Spec.ClusterDomain)))
		if err != nil {
			return err
		}
		for _, store := range storesInfo.Stores {
			status := getTiFlashStore(store)
			if status == nil {
				continue
			}

			oldStore, exist := previousStores[status.ID]
			if !exist {
				oldStore, exist = previousPeerStores[status.ID]
			}

			// avoid LastHeartbeatTime be overwrite by zero time when pd lost LastHeartbeatTime
			if status.LastHeartbeatTime.IsZero() && exist {
				klog.V(4).Infof("the pod:%s's store LastHeartbeatTime is zero,so will keep in %v", status.PodName, oldStore.LastHeartbeatTime)
				status.LastHeartbeatTime = oldStore.LastHeartbeatTime
			}

			status.LastTransitionTime = metav1.Now()
			if exist && status.State == oldStore.State {
				status.LastTransitionTime = oldStore.LastTransitionTime
			}

			if store.Store != nil {
				if pattern.Match([]byte(store.Store.Address)) {
					stores[status.ID] = *status
				} else if util.MatchLabelFromStoreLabels(store.Store.Labels, label.TiFlashLabelVal) {
					peerStores[status.ID] = *status
				}
			}
		}

		//this returns all tombstone stores
		tombstoneStoresInfo, err := pdClient.GetTombStoneStores()
		if err != nil {
			tc.Status.TiFlash.Synced = false
			return err
		}
		for _, store := range tombstoneStoresInfo.Stores {
			if store.Store != nil && !pattern.Match([]byte(store.Store.Address)) {
				continue
			}
			status := getTiFlashStore(store)
			if status == nil {
				continue
			}
			tombstoneStores[status.ID] = *status
		}

		tc.Status.TiFlash.Synced = true
		tc.Status.TiFlash.Stores = stores
		tc.Status.TiFlash.PeerStores = peerStores
		tc.Status.TiFlash.TombstoneStores = tombstoneStores
	case label.TiDBLabelVal:
		tidbStatus := map[string]v1alpha1.TiDBMember{}
		for id := range helper.GetPodOrdinals(tc.Status.TiDB.StatefulSet.Replicas, set) {
			name := fmt.Sprintf("%s-%d", controller.TiDBMemberName(tc.GetName()), id)
			health, err := dependencies.TiDBControl.GetHealth(tc, int32(id))
			if err != nil {
				return err
			}
			newTidbMember := v1alpha1.TiDBMember{
				Name:   name,
				Health: health,
			}
			oldTidbMember, exist := tc.Status.TiDB.Members[name]

			newTidbMember.LastTransitionTime = metav1.Now()
			if exist {
				newTidbMember.NodeName = oldTidbMember.NodeName
				if oldTidbMember.Health == newTidbMember.Health {
					newTidbMember.LastTransitionTime = oldTidbMember.LastTransitionTime
				}
			}
			pod, err := dependencies.PodLister.Pods(tc.GetNamespace()).Get(name)
			if err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("syncTidbClusterStatus: failed to get pods %s for cluster %s/%s, error: %s", name, tc.GetNamespace(), tc.GetName(), err)
			}
			if pod != nil && pod.Spec.NodeName != "" {
				// Update assiged node if pod exists and is scheduled
				newTidbMember.NodeName = pod.Spec.NodeName
			}
			tidbStatus[name] = newTidbMember
		}
		tc.Status.TiDB.Members = tidbStatus
	case label.TiCDCLabelVal:
		ticdcCaptures := map[string]v1alpha1.TiCDCCapture{}
		for id := range helper.GetPodOrdinals(tc.Status.TiCDC.StatefulSet.Replicas, set) {
			podName := fmt.Sprintf("%s-%d", controller.TiCDCMemberName(tc.GetName()), id)
			capture, err := dependencies.CDCControl.GetStatus(tc, int32(id))
			if err != nil {
				return err
			}
			ticdcCaptures[podName] = v1alpha1.TiCDCCapture{
				PodName: podName,
				ID:      capture.ID,
			}
		}
		tc.Status.TiCDC.Synced = true
		tc.Status.TiCDC.Captures = ticdcCaptures
	}
	return nil
}

func getTiFlashStore(store *pdapi.StoreInfo) *v1alpha1.TiKVStore {
	if store.Store == nil || store.Status == nil {
		return nil
	}
	storeID := fmt.Sprintf("%d", store.Store.GetId())
	ip := strings.Split(store.Store.GetAddress(), ":")[0]
	podName := strings.Split(ip, ".")[0]

	return &v1alpha1.TiKVStore{
		ID:                storeID,
		PodName:           podName,
		IP:                ip,
		LeaderCount:       int32(store.Status.LeaderCount),
		State:             store.Store.StateName,
		LastHeartbeatTime: metav1.Time{Time: store.Status.LastHeartbeatTS},
	}
}

func setStoreLabelsForTiKV(context *ComponentContext) (int, error) {
	tc := context.tc
	dependencies := context.dependencies

	ns := tc.GetNamespace()
	// for unit test
	setCount := 0

	if !tc.TiKVBootStrapped() {
		klog.Infof("TiKV of Cluster %s/%s is not bootstrapped yet, no need to set store labels", tc.Namespace, tc.Name)
		return setCount, nil
	}

	pdCli := controller.GetPDClient(dependencies.PDControl, tc)
	storesInfo, err := pdCli.GetStores()
	if err != nil {
		return setCount, err
	}

	config, err := pdCli.GetConfig()
	if err != nil {
		return setCount, err
	}

	locationLabels := []string(config.Replication.LocationLabels)
	if locationLabels == nil {
		return setCount, nil
	}

	pattern, err := regexp.Compile(fmt.Sprintf(tikvStoreLimitPattern, tc.Name, tc.Name, tc.Namespace, controller.FormatClusterDomainForRegex(tc.Spec.ClusterDomain)))
	if err != nil {
		return -1, err
	}
	for _, store := range storesInfo.Stores {
		// In theory, the external tikv can join the cluster, and the operator would only manage the internal tikv.
		// So we check the store owner to make sure it.
		if store.Store != nil && !pattern.Match([]byte(store.Store.Address)) {
			continue
		}
		status := getTiKVStore(store)
		if status == nil {
			continue
		}
		podName := status.PodName

		pod, err := dependencies.PodLister.Pods(ns).Get(podName)
		if err != nil {
			return setCount, fmt.Errorf("setStoreLabelsForTiKV: failed to get pods %s for cluster %s/%s, error: %s", podName, ns, tc.GetName(), err)
		}

		nodeName := pod.Spec.NodeName
		ls, err := getNodeLabels(context, nodeName, locationLabels)
		if err != nil || len(ls) == 0 {
			klog.Warningf("node: [%s] has no node labels, skipping set store labels for Pod: [%s/%s]", nodeName, ns, podName)
			continue
		}

		if !storeLabelsEqualNodeLabels(store.Store.Labels, ls) {
			set, err := pdCli.SetStoreLabels(store.Store.Id, ls)
			if err != nil {
				msg := fmt.Sprintf("failed to set labels %v for store (id: %d, pod: %s/%s): %v ",
					ls, store.Store.Id, ns, podName, err)
				dependencies.Recorder.Event(tc, corev1.EventTypeWarning, FailedSetStoreLabels, msg)
				continue
			}
			if set {
				setCount++
				klog.Infof("pod: [%s/%s] set labels: %v successfully", ns, podName, ls)
			}
		}
	}

	return setCount, nil
}

func getNodeLabels(context *ComponentContext, nodeName string, storeLabels []string) (map[string]string, error) {
	dependencies := context.dependencies

	node, err := dependencies.NodeLister.Get(nodeName)
	if err != nil {
		return nil, err
	}
	labels := map[string]string{}
	ls := node.GetLabels()
	for _, storeLabel := range storeLabels {
		if value, found := ls[storeLabel]; found {
			labels[storeLabel] = value
			continue
		}

		// TODO after pd supports storeLabel containing slash character, these codes should be deleted
		if storeLabel == "host" {
			if host, found := ls[corev1.LabelHostname]; found {
				labels[storeLabel] = host
			}
		}

	}
	return labels, nil
}

func storeLabelsEqualNodeLabels(storeLabels []*metapb.StoreLabel, nodeLabels map[string]string) bool {
	ls := map[string]string{}
	for _, label := range storeLabels {
		key := label.GetKey()
		if _, ok := nodeLabels[key]; ok {
			val := label.GetValue()
			ls[key] = val
		}
	}
	return reflect.DeepEqual(ls, nodeLabels)
}

func componentPDGetNewSetForTiDBCluster(context *ComponentContext, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
	tc := context.tc
	ns := tc.Namespace
	tcName := tc.Name
	basePDSpec := tc.BasePDSpec()
	instanceName := tc.GetInstanceName()
	pdConfigMap := controller.MemberConfigMapName(tc, v1alpha1.PDMemberType)
	if cm != nil {
		pdConfigMap = cm.Name
	}

	clusterVersionGE4, err := clusterVersionGreaterThanOrEqualTo4(tc.PDVersion())
	if err != nil {
		klog.V(4).Infof("cluster version: %s is not semantic versioning compatible", tc.PDVersion())
	}

	annMount, annVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		annMount,
		{Name: "config", ReadOnly: true, MountPath: "/etc/pd"},
		{Name: "startup-script", ReadOnly: true, MountPath: "/usr/local/bin"},
		{Name: v1alpha1.PDMemberType.String(), MountPath: pdDataVolumeMountPath},
	}
	if tc.IsTLSClusterEnabled() {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "pd-tls", ReadOnly: true, MountPath: "/var/lib/pd-tls",
		})
		if tc.Spec.PD.MountClusterClientSecret != nil && *tc.Spec.PD.MountClusterClientSecret {
			volMounts = append(volMounts, corev1.VolumeMount{
				Name: util.ClusterClientVolName, ReadOnly: true, MountPath: util.ClusterClientTLSPath,
			})
		}
	}
	if tc.Spec.TiDB.IsTLSClientEnabled() && !tc.SkipTLSWhenConnectTiDB() && clusterVersionGE4 {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "tidb-client-tls", ReadOnly: true, MountPath: tidbClientCertPath,
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
	if tc.IsTLSClusterEnabled() {
		vols = append(vols, corev1.Volume{
			Name: "pd-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterTLSSecretName(tc.Name, label.PDLabelVal),
				},
			},
		})
		if tc.Spec.PD.MountClusterClientSecret != nil && *tc.Spec.PD.MountClusterClientSecret {
			vols = append(vols, corev1.Volume{
				Name: util.ClusterClientVolName, VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: util.ClusterClientTLSSecretName(tc.Name),
					},
				},
			})
		}
	}
	if tc.Spec.TiDB.IsTLSClientEnabled() && !tc.SkipTLSWhenConnectTiDB() && clusterVersionGE4 {
		clientSecretName := util.TiDBClientTLSSecretName(tc.Name)
		if tc.Spec.PD.TLSClientSecretName != nil {
			clientSecretName = *tc.Spec.PD.TLSClientSecretName
		}
		vols = append(vols, corev1.Volume{
			Name: "tidb-client-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: clientSecretName,
				},
			},
		})
	}
	// handle StorageVolumes and AdditionalVolumeMounts in ComponentSpec
	storageVolMounts, additionalPVCs := util.BuildStorageVolumeAndVolumeMount(tc.Spec.PD.StorageVolumes, tc.Spec.PD.StorageClassName, v1alpha1.PDMemberType)
	volMounts = append(volMounts, storageVolMounts...)
	volMounts = append(volMounts, tc.Spec.PD.AdditionalVolumeMounts...)

	sysctls := "sysctl -w"
	var initContainers []corev1.Container
	if basePDSpec.Annotations() != nil {
		init, ok := basePDSpec.Annotations()[label.AnnSysctlInit]
		if ok && (init == label.AnnSysctlInitVal) {
			if basePDSpec.PodSecurityContext() != nil && len(basePDSpec.PodSecurityContext().Sysctls) > 0 {
				for _, sysctl := range basePDSpec.PodSecurityContext().Sysctls {
					sysctls = sysctls + fmt.Sprintf(" %s=%s", sysctl.Name, sysctl.Value)
				}
				privileged := true
				initContainers = append(initContainers, corev1.Container{
					Name:  "init",
					Image: tc.HelperImage(),
					Command: []string{
						"sh",
						"-c",
						sysctls,
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: &privileged,
					},
					// Init container resourceRequirements should be equal to app container.
					// Scheduling is done based on effective requests/limits,
					// which means init containers can reserve resources for
					// initialization that are not used during the life of the Pod.
					// ref:https://kubernetes.io/docs/concepts/workloads/pods/init-containers/#resources
					Resources: controller.ContainerResource(tc.Spec.PD.ResourceRequirements),
				})
			}
		}
	}
	// Init container is only used for the case where allowed-unsafe-sysctls
	// cannot be enabled for kubelet, so clean the sysctl in statefulset
	// SecurityContext if init container is enabled
	podSecurityContext := basePDSpec.PodSecurityContext().DeepCopy()
	if len(initContainers) > 0 {
		podSecurityContext.Sysctls = []corev1.Sysctl{}
	}

	storageRequest, err := controller.ParseStorageRequest(tc.Spec.PD.Requests)
	if err != nil {
		return nil, fmt.Errorf("cannot parse storage request for PD, tidbcluster %s/%s, error: %v", tc.Namespace, tc.Name, err)
	}

	pdLabel := label.New().Instance(instanceName).PD()
	setName := controller.PDMemberName(tcName)
	podAnnotations := CombineAnnotations(controller.AnnProm(2379), basePDSpec.Annotations())
	stsAnnotations := getStsAnnotations(tc.Annotations, label.PDLabelVal)

	pdContainer := corev1.Container{
		Name:            v1alpha1.PDMemberType.String(),
		Image:           tc.PDImage(),
		ImagePullPolicy: basePDSpec.ImagePullPolicy(),
		Command:         []string{"/bin/sh", "/usr/local/bin/pd_start_script.sh"},
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
		Resources:    controller.ContainerResource(tc.Spec.PD.ResourceRequirements),
	}
	env := []corev1.EnvVar{
		{
			Name: "NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
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
	}

	podSpec := basePDSpec.BuildPodSpec()
	if basePDSpec.HostNetwork() {
		podSpec.DNSPolicy = corev1.DNSClusterFirstWithHostNet
		env = append(env, corev1.EnvVar{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		})
	}
	pdContainer.Env = util.AppendEnv(env, basePDSpec.Env())
	podSpec.Volumes = append(vols, basePDSpec.AdditionalVolumes()...)
	podSpec.Containers = append([]corev1.Container{pdContainer}, basePDSpec.AdditionalContainers()...)
	podSpec.ServiceAccountName = tc.Spec.PD.ServiceAccount
	if podSpec.ServiceAccountName == "" {
		podSpec.ServiceAccountName = tc.Spec.ServiceAccount
	}
	podSpec.SecurityContext = podSecurityContext
	podSpec.InitContainers = initContainers

	updateStrategy := apps.StatefulSetUpdateStrategy{}
	if basePDSpec.StatefulSetUpdateStrategy() == apps.OnDeleteStatefulSetStrategyType {
		updateStrategy.Type = apps.OnDeleteStatefulSetStrategyType
	} else {
		updateStrategy.Type = apps.RollingUpdateStatefulSetStrategyType
		updateStrategy.RollingUpdate = &apps.RollingUpdateStatefulSetStrategy{
			Partition: pointer.Int32Ptr(tc.PDStsDesiredReplicas()),
		}
	}

	pdSet := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            setName,
			Namespace:       ns,
			Labels:          pdLabel.Labels(),
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(tc.PDStsDesiredReplicas()),
			Selector: pdLabel.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      pdLabel.Labels(),
					Annotations: podAnnotations,
				},
				Spec: podSpec,
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
						StorageClassName: tc.Spec.PD.StorageClassName,
						Resources:        storageRequest,
					},
				},
			},
			ServiceName:         controller.PDPeerMemberName(tcName),
			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy:      updateStrategy,
		},
	}

	pdSet.Spec.VolumeClaimTemplates = append(pdSet.Spec.VolumeClaimTemplates, additionalPVCs...)
	return pdSet, nil
}

func componentTiKVGetNewSetForTiDBCluster(context *ComponentContext, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
	tc := context.tc

	ns := tc.GetNamespace()
	tcName := tc.GetName()
	baseTiKVSpec := tc.BaseTiKVSpec()

	tikvConfigMap := controller.MemberConfigMapName(tc, v1alpha1.TiKVMemberType)
	if cm != nil {
		tikvConfigMap = cm.Name
	}

	annoMount, annoVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		annoMount,
		{Name: v1alpha1.TiKVMemberType.String(), MountPath: tikvDataVolumeMountPath},
		{Name: "config", ReadOnly: true, MountPath: "/etc/tikv"},
		{Name: "startup-script", ReadOnly: true, MountPath: "/usr/local/bin"},
	}
	volMounts = append(volMounts, tc.Spec.TiKV.AdditionalVolumeMounts...)
	if tc.IsTLSClusterEnabled() {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "tikv-tls", ReadOnly: true, MountPath: "/var/lib/tikv-tls",
		})
		if tc.Spec.TiKV.MountClusterClientSecret != nil && *tc.Spec.TiKV.MountClusterClientSecret {
			volMounts = append(volMounts, corev1.VolumeMount{
				Name: util.ClusterClientVolName, ReadOnly: true, MountPath: util.ClusterClientTLSPath,
			})
		}
	}

	vols := []corev1.Volume{
		annoVolume,
		{Name: "config", VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: tikvConfigMap,
				},
				Items: []corev1.KeyToPath{{Key: "config-file", Path: "tikv.toml"}},
			}},
		},
		{Name: "startup-script", VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: tikvConfigMap,
				},
				Items: []corev1.KeyToPath{{Key: "startup-script", Path: "tikv_start_script.sh"}},
			}},
		},
	}
	if tc.IsTLSClusterEnabled() {
		vols = append(vols, corev1.Volume{
			Name: "tikv-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterTLSSecretName(tc.Name, label.TiKVLabelVal),
				},
			},
		})
		if tc.Spec.TiKV.MountClusterClientSecret != nil && *tc.Spec.TiKV.MountClusterClientSecret {
			vols = append(vols, corev1.Volume{
				Name: util.ClusterClientVolName, VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: util.ClusterClientTLSSecretName(tc.Name),
					},
				},
			})
		}
	}
	// handle StorageVolumes and AdditionalVolumeMounts in ComponentSpec
	storageVolMounts, additionalPVCs := util.BuildStorageVolumeAndVolumeMount(tc.Spec.TiKV.StorageVolumes, tc.Spec.TiKV.StorageClassName, v1alpha1.TiKVMemberType)
	volMounts = append(volMounts, storageVolMounts...)

	sysctls := "sysctl -w"
	var initContainers []corev1.Container
	if baseTiKVSpec.Annotations() != nil {
		init, ok := baseTiKVSpec.Annotations()[label.AnnSysctlInit]
		if ok && (init == label.AnnSysctlInitVal) {
			if baseTiKVSpec.PodSecurityContext() != nil && len(baseTiKVSpec.PodSecurityContext().Sysctls) > 0 {
				for _, sysctl := range baseTiKVSpec.PodSecurityContext().Sysctls {
					sysctls = sysctls + fmt.Sprintf(" %s=%s", sysctl.Name, sysctl.Value)
				}
				privileged := true
				initContainers = append(initContainers, corev1.Container{
					Name:  "init",
					Image: tc.HelperImage(),
					Command: []string{
						"sh",
						"-c",
						sysctls,
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: &privileged,
					},
					// Init container resourceRequirements should be equal to app container.
					// Scheduling is done based on effective requests/limits,
					// which means init containers can reserve resources for
					// initialization that are not used during the life of the Pod.
					// ref:https://kubernetes.io/docs/concepts/workloads/pods/init-containers/#resources
					Resources: controller.ContainerResource(tc.Spec.TiKV.ResourceRequirements),
				})
			}
		}
	}
	// Init container is only used for the case where allowed-unsafe-sysctls
	// cannot be enabled for kubelet, so clean the sysctl in statefulset
	// SecurityContext if init container is enabled
	podSecurityContext := baseTiKVSpec.PodSecurityContext().DeepCopy()
	if len(initContainers) > 0 {
		podSecurityContext.Sysctls = []corev1.Sysctl{}
	}

	storageRequest, err := controller.ParseStorageRequest(tc.Spec.TiKV.Requests)
	if err != nil {
		return nil, fmt.Errorf("cannot parse storage request for tikv, tidbcluster %s/%s, error: %v", tc.Namespace, tc.Name, err)
	}

	tikvLabel := labelTiKV(tc)
	setName := controller.TiKVMemberName(tcName)
	podAnnotations := CombineAnnotations(controller.AnnProm(20180), baseTiKVSpec.Annotations())
	stsAnnotations := getStsAnnotations(tc.Annotations, label.TiKVLabelVal)
	capacity := controller.TiKVCapacity(tc.Spec.TiKV.Limits)
	headlessSvcName := controller.TiKVPeerMemberName(tcName)

	env := []corev1.EnvVar{
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
		{
			Name:  "TZ",
			Value: tc.Spec.Timezone,
		},
	}
	tikvContainer := corev1.Container{
		Name:            v1alpha1.TiKVMemberType.String(),
		Image:           tc.TiKVImage(),
		ImagePullPolicy: baseTiKVSpec.ImagePullPolicy(),
		Command:         []string{"/bin/sh", "/usr/local/bin/tikv_start_script.sh"},
		SecurityContext: &corev1.SecurityContext{
			Privileged: tc.TiKVContainerPrivilege(),
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "server",
				ContainerPort: int32(20160),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: volMounts,
		Resources:    controller.ContainerResource(tc.Spec.TiKV.ResourceRequirements),
	}
	podSpec := baseTiKVSpec.BuildPodSpec()
	if baseTiKVSpec.HostNetwork() {
		podSpec.DNSPolicy = corev1.DNSClusterFirstWithHostNet
		env = append(env, corev1.EnvVar{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		})
	}
	tikvContainer.Env = util.AppendEnv(env, baseTiKVSpec.Env())
	podSpec.Volumes = append(vols, baseTiKVSpec.AdditionalVolumes()...)
	podSpec.SecurityContext = podSecurityContext
	podSpec.InitContainers = initContainers
	podSpec.Containers = append([]corev1.Container{tikvContainer}, baseTiKVSpec.AdditionalContainers()...)
	podSpec.ServiceAccountName = tc.Spec.TiKV.ServiceAccount
	if podSpec.ServiceAccountName == "" {
		podSpec.ServiceAccountName = tc.Spec.ServiceAccount
	}

	updateStrategy := apps.StatefulSetUpdateStrategy{}
	if baseTiKVSpec.StatefulSetUpdateStrategy() == apps.OnDeleteStatefulSetStrategyType {
		updateStrategy.Type = apps.OnDeleteStatefulSetStrategyType
	} else {
		updateStrategy.Type = apps.RollingUpdateStatefulSetStrategyType
		updateStrategy.RollingUpdate = &apps.RollingUpdateStatefulSetStrategy{
			Partition: pointer.Int32Ptr(tc.TiKVStsDesiredReplicas()),
		}
	}

	tikvset := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            setName,
			Namespace:       ns,
			Labels:          tikvLabel.Labels(),
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(tc.TiKVStsDesiredReplicas()),
			Selector: tikvLabel.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      tikvLabel.Labels(),
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				util.VolumeClaimTemplate(storageRequest, v1alpha1.TiKVMemberType.String(), tc.Spec.TiKV.StorageClassName),
			},
			ServiceName:         headlessSvcName,
			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy:      updateStrategy,
		},
	}

	tikvset.Spec.VolumeClaimTemplates = append(tikvset.Spec.VolumeClaimTemplates, additionalPVCs...)
	return tikvset, nil
}

func componentTiFlashGetNewSetForTiDBCluster(context *ComponentContext, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
	tc := context.tc

	ns := tc.GetNamespace()
	tcName := tc.GetName()
	baseTiFlashSpec := tc.BaseTiFlashSpec()
	spec := tc.Spec.TiFlash

	tiflashConfigMap := controller.MemberConfigMapName(tc, v1alpha1.TiFlashMemberType)
	if cm != nil {
		tiflashConfigMap = cm.Name
	}

	// This should not happen as we have validaton for this field
	if len(spec.StorageClaims) < 1 {
		return nil, fmt.Errorf("storageClaims should be configured at least one item for tiflash, tidbcluster %s/%s", tc.Namespace, tc.Name)
	}
	pvcs, err := flashVolumeClaimTemplate(tc.Spec.TiFlash.StorageClaims)
	if err != nil {
		return nil, fmt.Errorf("cannot parse storage request for tiflash.StorageClaims, tidbcluster %s/%s, error: %v", tc.Namespace, tc.Name, err)
	}
	annoMount, annoVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		annoMount,
	}
	for k := range spec.StorageClaims {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: fmt.Sprintf("data%d", k), MountPath: fmt.Sprintf("/data%d", k)})
	}
	volMounts = append(volMounts, tc.Spec.TiFlash.AdditionalVolumeMounts...)

	if tc.IsTLSClusterEnabled() {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: tiflashCertVolumeName, ReadOnly: true, MountPath: tiflashCertPath,
		})
	}

	vols := []corev1.Volume{
		annoVolume,
		{Name: "config", VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: tiflashConfigMap,
				},
			}},
		},
	}

	if tc.IsTLSClusterEnabled() {
		vols = append(vols, corev1.Volume{
			Name: tiflashCertVolumeName, VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterTLSSecretName(tc.Name, label.TiFlashLabelVal),
				},
			},
		})
	}

	sysctls := "sysctl -w"
	var initContainers []corev1.Container
	if baseTiFlashSpec.Annotations() != nil {
		init, ok := baseTiFlashSpec.Annotations()[label.AnnSysctlInit]
		if ok && (init == label.AnnSysctlInitVal) {
			if baseTiFlashSpec.PodSecurityContext() != nil && len(baseTiFlashSpec.PodSecurityContext().Sysctls) > 0 {
				for _, sysctl := range baseTiFlashSpec.PodSecurityContext().Sysctls {
					sysctls = sysctls + fmt.Sprintf(" %s=%s", sysctl.Name, sysctl.Value)
				}
				privileged := true
				initContainers = append(initContainers, corev1.Container{
					Name:  "init",
					Image: tc.HelperImage(),
					Command: []string{
						"sh",
						"-c",
						sysctls,
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: &privileged,
					},
					// Init container resourceRequirements should be equal to app container.
					// Scheduling is done based on effective requests/limits,
					// which means init containers can reserve resources for
					// initialization that are not used during the life of the Pod.
					// ref:https://kubernetes.io/docs/concepts/workloads/pods/init-containers/#resources
					Resources: controller.ContainerResource(tc.Spec.TiFlash.ResourceRequirements),
				})
			}
		}
	}
	// Init container is only used for the case where allowed-unsafe-sysctls
	// cannot be enabled for kubelet, so clean the sysctl in statefulset
	// SecurityContext if init container is enabled
	podSecurityContext := baseTiFlashSpec.PodSecurityContext().DeepCopy()
	if len(initContainers) > 0 {
		podSecurityContext.Sysctls = []corev1.Sysctl{}
	}

	// Append init container for config files initialization
	initVolMounts := []corev1.VolumeMount{
		{Name: "data0", MountPath: "/data0"},
		{Name: "config", ReadOnly: true, MountPath: "/etc/tiflash"},
	}
	initEnv := []corev1.EnvVar{
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
	}
	initContainers = append(initContainers, corev1.Container{
		Name:  "init",
		Image: tc.HelperImage(),
		Command: []string{
			"sh",
			"-c",
			"set -ex;ordinal=`echo ${POD_NAME} | awk -F- '{print $NF}'`;sed s/POD_NUM/${ordinal}/g /etc/tiflash/config_templ.toml > /data0/config.toml;sed s/POD_NUM/${ordinal}/g /etc/tiflash/proxy_templ.toml > /data0/proxy.toml",
		},
		Env:          initEnv,
		VolumeMounts: initVolMounts,
	})

	tiflashLabel := labelTiFlash(tc)
	setName := controller.TiFlashMemberName(tcName)
	podAnnotations := CombineAnnotations(controller.AnnProm(8234), baseTiFlashSpec.Annotations())
	podAnnotations = CombineAnnotations(controller.AnnAdditionalProm("tiflash.proxy", 20292), podAnnotations)
	stsAnnotations := getStsAnnotations(tc.Annotations, label.TiFlashLabelVal)
	capacity := controller.TiKVCapacity(tc.Spec.TiFlash.Limits)
	headlessSvcName := controller.TiFlashPeerMemberName(tcName)

	env := []corev1.EnvVar{
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
		{
			Name:  "TZ",
			Value: tc.Timezone(),
		},
	}
	tiflashContainer := corev1.Container{
		Name:            v1alpha1.TiFlashMemberType.String(),
		Image:           tc.TiFlashImage(),
		ImagePullPolicy: baseTiFlashSpec.ImagePullPolicy(),
		Command:         []string{"/bin/sh", "-c", "/tiflash/tiflash server --config-file /data0/config.toml"},
		SecurityContext: &corev1.SecurityContext{
			Privileged: tc.TiFlashContainerPrivilege(),
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "tiflash",
				ContainerPort: int32(3930),
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "proxy",
				ContainerPort: int32(20170),
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "tcp",
				ContainerPort: int32(9000),
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "http",
				ContainerPort: int32(8123),
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "internal",
				ContainerPort: int32(9009),
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "metrics",
				ContainerPort: int32(8234),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: volMounts,
		Resources:    controller.ContainerResource(tc.Spec.TiFlash.ResourceRequirements),
	}
	podSpec := baseTiFlashSpec.BuildPodSpec()
	if baseTiFlashSpec.HostNetwork() {
		podSpec.DNSPolicy = corev1.DNSClusterFirstWithHostNet
		env = append(env, corev1.EnvVar{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		})
	}
	tiflashContainer.Env = util.AppendEnv(env, baseTiFlashSpec.Env())
	podSpec.Volumes = append(vols, baseTiFlashSpec.AdditionalVolumes()...)
	podSpec.SecurityContext = podSecurityContext
	podSpec.InitContainers = initContainers
	containers, err := buildTiFlashSidecarContainers(tc)
	if err != nil {
		return nil, err
	}
	podSpec.Containers = append([]corev1.Container{tiflashContainer}, containers...)
	podSpec.Containers = append(podSpec.Containers, baseTiFlashSpec.AdditionalContainers()...)
	podSpec.ServiceAccountName = tc.Spec.TiFlash.ServiceAccount
	if podSpec.ServiceAccountName == "" {
		podSpec.ServiceAccountName = tc.Spec.ServiceAccount
	}

	tiflashset := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            setName,
			Namespace:       ns,
			Labels:          tiflashLabel.Labels(),
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(tc.TiFlashStsDesiredReplicas()),
			Selector: tiflashLabel.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      tiflashLabel.Labels(),
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
			VolumeClaimTemplates: pvcs,
			ServiceName:          headlessSvcName,
			PodManagementPolicy:  apps.ParallelPodManagement,
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: baseTiFlashSpec.StatefulSetUpdateStrategy(),
			},
		},
	}
	return tiflashset, nil
}

func componentTiDBGetNewSetForTiDBCluster(context *ComponentContext, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
	tc := context.tc
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	headlessSvcName := controller.TiDBPeerMemberName(tcName)
	baseTiDBSpec := tc.BaseTiDBSpec()
	instanceName := tc.GetInstanceName()
	tidbConfigMap := controller.MemberConfigMapName(tc, v1alpha1.TiDBMemberType)
	if cm != nil {
		tidbConfigMap = cm.Name
	}

	annoMount, annoVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		annoMount,
		{Name: "config", ReadOnly: true, MountPath: "/etc/tidb"},
		{Name: "startup-script", ReadOnly: true, MountPath: "/usr/local/bin"},
	}
	if tc.IsTLSClusterEnabled() {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "tidb-tls", ReadOnly: true, MountPath: clusterCertPath,
		})
	}
	if tc.Spec.TiDB.IsTLSClientEnabled() {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "tidb-server-tls", ReadOnly: true, MountPath: serverCertPath,
		})
	}

	vols := []corev1.Volume{
		annoVolume,
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
	if tc.IsTLSClusterEnabled() {
		vols = append(vols, corev1.Volume{
			Name: "tidb-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterTLSSecretName(tcName, label.TiDBLabelVal),
				},
			},
		})
	}
	if tc.Spec.TiDB.IsTLSClientEnabled() {
		secretName := tlsClientSecretName(tc)
		vols = append(vols, corev1.Volume{
			Name: "tidb-server-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
				},
			},
		})
	}

	sysctls := "sysctl -w"
	var initContainers []corev1.Container
	if baseTiDBSpec.Annotations() != nil {
		init, ok := baseTiDBSpec.Annotations()[label.AnnSysctlInit]
		if ok && (init == label.AnnSysctlInitVal) {
			if baseTiDBSpec.PodSecurityContext() != nil && len(baseTiDBSpec.PodSecurityContext().Sysctls) > 0 {
				for _, sysctl := range baseTiDBSpec.PodSecurityContext().Sysctls {
					sysctls = sysctls + fmt.Sprintf(" %s=%s", sysctl.Name, sysctl.Value)
				}
				privileged := true
				initContainers = append(initContainers, corev1.Container{
					Name:  "init",
					Image: tc.HelperImage(),
					Command: []string{
						"sh",
						"-c",
						sysctls,
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: &privileged,
					},
					// Init container resourceRequirements should be equal to app container.
					// Scheduling is done based on effective requests/limits,
					// which means init containers can reserve resources for
					// initialization that are not used during the life of the Pod.
					// ref:https://kubernetes.io/docs/concepts/workloads/pods/init-containers/#resources
					Resources: controller.ContainerResource(tc.Spec.TiDB.ResourceRequirements),
				})
			}
		}
	}
	// Init container is only used for the case where allowed-unsafe-sysctls
	// cannot be enabled for kubelet, so clean the sysctl in statefulset
	// SecurityContext if init container is enabled
	podSecurityContext := baseTiDBSpec.PodSecurityContext().DeepCopy()
	if len(initContainers) > 0 {
		podSecurityContext.Sysctls = []corev1.Sysctl{}
	}

	var containers []corev1.Container
	if tc.Spec.TiDB.ShouldSeparateSlowLog() {
		// mount a shared volume and tail the slow log to STDOUT using a sidecar.
		vols = append(vols, corev1.Volume{
			Name: slowQueryLogVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
		volMounts = append(volMounts, corev1.VolumeMount{Name: slowQueryLogVolumeName, MountPath: slowQueryLogDir})
		containers = append(containers, corev1.Container{
			Name:            v1alpha1.SlowLogTailerMemberType.String(),
			Image:           tc.HelperImage(),
			ImagePullPolicy: tc.HelperImagePullPolicy(),
			Resources:       controller.ContainerResource(tc.Spec.TiDB.GetSlowLogTailerSpec().ResourceRequirements),
			VolumeMounts: []corev1.VolumeMount{
				{Name: slowQueryLogVolumeName, MountPath: slowQueryLogDir},
			},
			Command: []string{
				"sh",
				"-c",
				fmt.Sprintf("touch %s; tail -n0 -F %s;", slowQueryLogFile, slowQueryLogFile),
			},
		})
	}

	slowLogFileEnvVal := ""
	if tc.Spec.TiDB.ShouldSeparateSlowLog() {
		slowLogFileEnvVal = slowQueryLogFile
	}
	envs := []corev1.EnvVar{
		{
			Name:  "CLUSTER_NAME",
			Value: tc.GetName(),
		},
		{
			Name:  "TZ",
			Value: tc.Spec.Timezone,
		},
		{
			Name:  "BINLOG_ENABLED",
			Value: strconv.FormatBool(tc.IsTiDBBinlogEnabled()),
		},
		{
			Name:  "SLOW_LOG_FILE",
			Value: slowLogFileEnvVal,
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
			Name: "NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name:  "HEADLESS_SERVICE_NAME",
			Value: headlessSvcName,
		},
	}

	// handle StorageVolumes and AdditionalVolumeMounts in ComponentSpec
	storageVolMounts, additionalPVCs := util.BuildStorageVolumeAndVolumeMount(tc.Spec.TiDB.StorageVolumes, tc.Spec.TiDB.StorageClassName, v1alpha1.TiDBMemberType)
	volMounts = append(volMounts, storageVolMounts...)
	volMounts = append(volMounts, tc.Spec.TiDB.AdditionalVolumeMounts...)

	c := corev1.Container{
		Name:            v1alpha1.TiDBMemberType.String(),
		Image:           tc.TiDBImage(),
		Command:         []string{"/bin/sh", "/usr/local/bin/tidb_start_script.sh"},
		ImagePullPolicy: baseTiDBSpec.ImagePullPolicy(),
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
		},
		VolumeMounts: volMounts,
		Resources:    controller.ContainerResource(tc.Spec.TiDB.ResourceRequirements),
		Env:          util.AppendEnv(envs, baseTiDBSpec.Env()),
		ReadinessProbe: &corev1.Probe{
			Handler:             buildTiDBReadinessProbHandler(tc),
			InitialDelaySeconds: int32(10),
		},
	}
	if tc.Spec.TiDB.Lifecycle != nil {
		c.Lifecycle = tc.Spec.TiDB.Lifecycle
	}

	containers = append(containers, c)

	podSpec := baseTiDBSpec.BuildPodSpec()
	podSpec.Containers = append(containers, baseTiDBSpec.AdditionalContainers()...)
	podSpec.Volumes = append(vols, baseTiDBSpec.AdditionalVolumes()...)
	podSpec.SecurityContext = podSecurityContext
	podSpec.InitContainers = initContainers
	podSpec.ServiceAccountName = tc.Spec.TiDB.ServiceAccount
	if podSpec.ServiceAccountName == "" {
		podSpec.ServiceAccountName = tc.Spec.ServiceAccount
	}

	if baseTiDBSpec.HostNetwork() {
		podSpec.DNSPolicy = corev1.DNSClusterFirstWithHostNet
	}

	tidbLabel := label.New().Instance(instanceName).TiDB()
	podAnnotations := CombineAnnotations(controller.AnnProm(10080), baseTiDBSpec.Annotations())
	stsAnnotations := getStsAnnotations(tc.Annotations, label.TiDBLabelVal)

	updateStrategy := apps.StatefulSetUpdateStrategy{}
	if baseTiDBSpec.StatefulSetUpdateStrategy() == apps.OnDeleteStatefulSetStrategyType {
		updateStrategy.Type = apps.OnDeleteStatefulSetStrategyType
	} else {
		updateStrategy.Type = apps.RollingUpdateStatefulSetStrategyType
		updateStrategy.RollingUpdate = &apps.RollingUpdateStatefulSetStrategy{
			Partition: pointer.Int32Ptr(tc.TiDBStsDesiredReplicas()),
		}
	}

	tidbSet := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            controller.TiDBMemberName(tcName),
			Namespace:       ns,
			Labels:          tidbLabel.Labels(),
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(tc.TiDBStsDesiredReplicas()),
			Selector: tidbLabel.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      tidbLabel.Labels(),
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
			ServiceName:         controller.TiDBPeerMemberName(tcName),
			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy:      updateStrategy,
		},
	}

	tidbSet.Spec.VolumeClaimTemplates = append(tidbSet.Spec.VolumeClaimTemplates, additionalPVCs...)
	return tidbSet, nil
}

func componentPumpGetNewSetForTiDBCluster(context *ComponentContext, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
	tc := context.tc

	spec, ok := tc.BasePumpSpec()
	if !ok {
		return nil, nil
	}
	objMeta, pumpLabel := getPumpMeta(tc, controller.PumpMemberName)
	replicas := tc.Spec.Pump.Replicas
	storageClass := tc.Spec.Pump.StorageClassName
	podAnnos := CombineAnnotations(controller.AnnProm(8250), spec.Annotations())
	storageRequest, err := controller.ParseStorageRequest(tc.Spec.Pump.Requests)
	if err != nil {
		return nil, fmt.Errorf("cannot parse storage request for pump, tidbcluster %s/%s, error: %v", tc.Namespace, tc.Name, err)
	}
	startScript, err := getPumpStartScript(tc)
	if err != nil {
		return nil, fmt.Errorf("cannot render start-script for pump, tidbcluster %s/%s, error: %v", tc.Namespace, tc.Name, err)
	}

	var envs []corev1.EnvVar
	if tc.Spec.Pump.SetTimeZone != nil && *tc.Spec.Pump.SetTimeZone {
		envs = append(envs, corev1.EnvVar{
			Name:  "TZ",
			Value: tc.Spec.Timezone,
		})
	}
	if spec.HostNetwork() {
		// For backward compatibility, set HOSTNAME to POD_NAME in hostNetwork mode
		envs = append(envs, corev1.EnvVar{
			Name: "HOSTNAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		})
	}
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "data",
			MountPath: "/data",
		},
		{
			Name:      "config",
			MountPath: "/etc/pump",
		},
	}
	if tc.IsTLSClusterEnabled() {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name: pumpCertVolumeMount, ReadOnly: true, MountPath: pumpCertPath,
		})
	}
	containers := []corev1.Container{
		{
			Name:            "pump",
			Image:           *tc.PumpImage(),
			ImagePullPolicy: spec.ImagePullPolicy(),
			Command: []string{
				"/bin/sh",
				"-c",
				startScript,
			},
			Ports: []corev1.ContainerPort{{
				Name:          "pump",
				ContainerPort: 8250,
			}},
			Resources:    controller.ContainerResource(tc.Spec.Pump.ResourceRequirements),
			Env:          util.AppendEnv(envs, spec.Env()),
			VolumeMounts: volumeMounts,
		},
	}

	// Keep backward compatibility for pump created by helm
	volumes := []corev1.Volume{
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cm.Name,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "pump-config",
							Path: "pump.toml",
						},
					},
				},
			},
		},
	}

	if tc.IsTLSClusterEnabled() {
		volumes = append(volumes, corev1.Volume{
			Name: pumpCertVolumeMount, VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterTLSSecretName(tc.Name, label.PumpLabelVal),
				},
			},
		})
	}

	volumeClaims := []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "data",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				StorageClassName: storageClass,
				Resources:        storageRequest,
			},
		},
	}

	serviceAccountName := tc.Spec.Pump.ServiceAccount
	if serviceAccountName == "" {
		serviceAccountName = tc.Spec.ServiceAccount
	}

	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: podAnnos,
			Labels:      pumpLabel,
		},
		Spec: corev1.PodSpec{
			Containers:         containers,
			ServiceAccountName: serviceAccountName,
			Volumes:            volumes,

			Affinity:         spec.Affinity(),
			Tolerations:      spec.Tolerations(),
			NodeSelector:     spec.NodeSelector(),
			SchedulerName:    spec.SchedulerName(),
			SecurityContext:  spec.PodSecurityContext(),
			HostNetwork:      spec.HostNetwork(),
			DNSPolicy:        spec.DnsPolicy(),
			ImagePullSecrets: spec.ImagePullSecrets(),
		},
	}

	return &appsv1.StatefulSet{
		ObjectMeta: objMeta,
		Spec: appsv1.StatefulSetSpec{
			Selector:    pumpLabel.LabelSelector(),
			ServiceName: controller.PumpMemberName(tc.Name),
			Replicas:    &replicas,

			Template:             podTemplate,
			VolumeClaimTemplates: volumeClaims,
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: spec.StatefulSetUpdateStrategy(),
			},
		},
	}, nil
}

func componentTiCDCGetNewSetForTiDBCluster(context *ComponentContext, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
	tc := context.tc
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	baseTiCDCSpec := tc.BaseTiCDCSpec()
	ticdcLabel := labelTiCDC(tc)
	stsName := controller.TiCDCMemberName(tcName)
	podAnnotations := CombineAnnotations(controller.AnnProm(8301), baseTiCDCSpec.Annotations())
	stsAnnotations := getStsAnnotations(tc.Annotations, label.TiCDCLabelVal)
	headlessSvcName := controller.TiCDCPeerMemberName(tcName)

	cmdArgs := []string{"/cdc server", "--addr=0.0.0.0:8301", fmt.Sprintf("--advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc%s:8301", controller.FormatClusterDomain(tc.Spec.ClusterDomain))}
	cmdArgs = append(cmdArgs, fmt.Sprintf("--gc-ttl=%d", tc.TiCDCGCTTL()))
	cmdArgs = append(cmdArgs, fmt.Sprintf("--log-file=%s", tc.TiCDCLogFile()))
	cmdArgs = append(cmdArgs, fmt.Sprintf("--log-level=%s", tc.TiCDCLogLevel()))

	if tc.IsTLSClusterEnabled() {
		cmdArgs = append(cmdArgs, fmt.Sprintf("--ca=%s", path.Join(ticdcCertPath, corev1.ServiceAccountRootCAKey)))
		cmdArgs = append(cmdArgs, fmt.Sprintf("--cert=%s", path.Join(ticdcCertPath, corev1.TLSCertKey)))
		cmdArgs = append(cmdArgs, fmt.Sprintf("--key=%s", path.Join(ticdcCertPath, corev1.TLSPrivateKeyKey)))
		cmdArgs = append(cmdArgs, fmt.Sprintf("--pd=https://%s-pd:2379", tcName))
	} else {
		cmdArgs = append(cmdArgs, fmt.Sprintf("--pd=http://%s-pd:2379", tcName))
	}

	cmd := strings.Join(cmdArgs, " ")

	envs := []corev1.EnvVar{
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: "NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name:  "HEADLESS_SERVICE_NAME",
			Value: headlessSvcName,
		},
		{
			Name:  "TZ",
			Value: tc.TiCDCTimezone(),
		},
	}

	ticdcContainer := corev1.Container{
		Name:            v1alpha1.TiCDCMemberType.String(),
		Image:           tc.TiCDCImage(),
		ImagePullPolicy: baseTiCDCSpec.ImagePullPolicy(),
		Command:         []string{"/bin/sh", "-c", cmd},
		Ports: []corev1.ContainerPort{
			{
				Name:          "ticdc",
				ContainerPort: int32(8301),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Resources: controller.ContainerResource(tc.Spec.TiCDC.ResourceRequirements),
		Env:       util.AppendEnv(envs, baseTiCDCSpec.Env()),
	}

	if tc.IsTLSClusterEnabled() {
		ticdcContainer.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      ticdcCertVolumeMount,
				ReadOnly:  true,
				MountPath: ticdcCertPath,
			},
			{
				Name:      util.ClusterClientVolName,
				ReadOnly:  true,
				MountPath: util.ClusterClientTLSPath,
			},
		}
	}

	podSpec := baseTiCDCSpec.BuildPodSpec()
	podSpec.Containers = []corev1.Container{ticdcContainer}
	podSpec.ServiceAccountName = tc.Spec.TiCDC.ServiceAccount
	if podSpec.ServiceAccountName == "" {
		podSpec.ServiceAccountName = tc.Spec.ServiceAccount
	}

	if tc.IsTLSClusterEnabled() {
		podSpec.Volumes = []corev1.Volume{
			{
				Name: ticdcCertVolumeMount, VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: util.ClusterTLSSecretName(tc.Name, label.TiCDCLabelVal),
					},
				},
			},
			{
				Name: util.ClusterClientVolName, VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: util.ClusterClientTLSSecretName(tc.Name),
					},
				},
			},
		}
	}

	ticdcSts := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            stsName,
			Namespace:       ns,
			Labels:          ticdcLabel.Labels(),
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(tc.TiCDCDeployDesiredReplicas()),
			Selector: ticdcLabel.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      ticdcLabel.Labels(),
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
			ServiceName:         headlessSvcName,
			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: baseTiCDCSpec.StatefulSetUpdateStrategy(),
			},
		},
	}
	return ticdcSts, nil
}
