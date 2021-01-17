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
	"regexp"
	"strconv"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
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

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	oldPDSetTmp, err := dependencies.StatefulSetLister.StatefulSets(ns).Get(controller.PDMemberName(tcName))
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncPDStatefulSetForTidbCluster: fail to get sts %s for cluster %s/%s, error: %s", controller.PDMemberName(tcName), ns, tcName, err)
	}
	setNotExist := errors.IsNotFound(err)

	oldPDSet := oldPDSetTmp.DeepCopy()

	if err := ComponentSyncTidbClusterStatus(context, oldPDSet); err != nil {
		klog.Errorf("failed to sync TidbCluster: [%s/%s]'s status, error: %v", ns, tcName, err)
	}

	if tc.Spec.Paused {
		klog.V(4).Infof("tidb cluster %s/%s is paused, skip syncing for pd statefulset", tc.GetNamespace(), tc.GetName())
		return nil
	}

	cm, err := ComponentSyncConfigMap(context, oldPDSet)
	if err != nil {
		return err
	}
	newPDSet, err := getNewPDSetForTidbCluster(tc, cm)
	if err != nil {
		return err
	}
	if setNotExist {
		err = SetStatefulSetLastAppliedConfigAnnotation(newPDSet)
		if err != nil {
			return err
		}
		if err := dependencies.StatefulSetControl.CreateStatefulSet(tc, newPDSet); err != nil {
			return err
		}
		tc.Status.PD.StatefulSet = &apps.StatefulSetStatus{}
		return controller.RequeueErrorf("TidbCluster: [%s/%s], waiting for PD cluster running", ns, tcName)
	}

	// Force update takes precedence over scaling because force upgrade won't take effect when cluster gets stuck at scaling
	if !tc.Status.PD.Synced && NeedForceUpgrade(tc.Annotations) {
		tc.Status.PD.Phase = v1alpha1.UpgradePhase
		setUpgradePartition(newPDSet, 0)
		errSTS := UpdateStatefulSet(dependencies.StatefulSetControl, tc, newPDSet, oldPDSet)
		return controller.RequeueErrorf("tidbcluster: [%s/%s]'s pd needs force upgrade, %v", ns, tcName, errSTS)
	}

	// Scaling takes precedence over upgrading because:
	// - if a pd fails in the upgrading, users may want to delete it or add
	//   new replicas
	// - it's ok to scale in the middle of upgrading (in statefulset controller
	//   scaling takes precedence over upgrading too)
	if err := ComponentScale(context, oldPDSet, newPDSet); err != nil {
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

	if !templateEqual(newPDSet, oldPDSet) || tc.Status.PD.Phase == v1alpha1.UpgradePhase {
		if err := ComponentUpgrade(context, oldPDSet, newPDSet); err != nil {
			return err
		}
	}

	return UpdateStatefulSet(dependencies.StatefulSetControl, tc, newPDSet, oldPDSet)
}

func ComponentSyncTidbClusterStatus(context *ComponentContext, set *apps.StatefulSet) error {
	tc := context.tc
	component := context.component

	if set == nil {
		// skip if not created yet
		return nil
	}

	switch component {
	case label.PDLabelVal:
		tc.Status.PD.StatefulSet = &set.Status

		err := syncComponentPhase(context, set)
		if err != nil {
			return err
		}
		err = syncComponentMembers(context)
		if err != nil {
			return err
		}

		err = syncComponentImage(context, set)
		if err != nil {
			return err
		}

		// k8s check
		pdStatus := tc.Status.PD.Members
		err = ComponentCollectUnjoinedMembers(context, set, pdStatus)
		if err != nil {
			return err
		}
	case label.TiKVLabelVal:
		tc.Status.TiKV.StatefulSet = &set.Status

		err := syncComponentPhase(context, set)
		if err != nil {
			return err
		}
		err = syncComponentMembers(context)
		if err != nil {
			return err
		}
		err = syncComponentImage(context, set)
		if err != nil {
			return err
		}

		tc.Status.TiKV.BootStrapped = true
	case label.TiFlashLabelVal:
		tc.Status.TiFlash.StatefulSet = &set.Status

		err := syncComponentPhase(context, set)
		if err != nil {
			return err
		}
		err = syncComponentMembers(context)
		if err != nil {
			return err
		}
	case label.TiDBLabelVal:
		tc.Status.TiDB.StatefulSet = &set.Status

		err := syncComponentPhase(context, set)
		if err != nil {
			return err
		}
		err = syncComponentMembers(context)
		if err != nil {
			return err
		}
		err = syncComponentMembers(context)
		if err != nil {
			return err
		}
	case label.TiCDCLabelVal:
		tc.Status.TiCDC.StatefulSet = &set.Status

		err := syncComponentPhase(context, set)
		if err != nil {
			return err
		}
		err = syncComponentMembers(context)
		if err != nil {
			return err
		}

	case label.PumpLabelVal:
		tc.Status.Pump.StatefulSet = &set.Status

		err := syncComponentPhase(context, set)
		if err != nil {
			return err
		}
	}

	return nil
}

// extras

func ComponentClusterVersionGreaterThanOrEqualTo4(version string) (bool, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return true, err
	}

	return v.Major() >= 4, nil
}

func ComponentCollectUnjoinedMembers(context *ComponentContext, set *apps.StatefulSet, pdStatus map[string]v1alpha1.PDMember) error {
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
	selector, err := label.New().
		Instance(instanceName).
		PD().
		Selector()
	if err != nil {
		return false, err
	}
	pdPods, err := dependencies.PodLister.Pods(tc.GetNamespace()).List(selector)
	if err != nil {
		return false, fmt.Errorf("StatefulSetIsUpgrading: failed to list pods for cluster %s/%s, selector %s, error: %v", tc.GetNamespace(), instanceName, selector, err)
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

func ComponentGetNewSetForTidbCluster(context *ComponentContext, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
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
	case label.TiDBLabelVal:
		desiredReplicas = tc.TiDBStsDesiredReplicas()
	}

	upgrading, err := ComponentStatefulSetIsUpgrading(set, context)
	if err != nil {
		return err
	}

	switch component {
	case label.PDLabelVal, label.TiFlashLabelVal:
		// Scaling takes precedence over upgrading.
		if desiredReplicas != *set.Spec.Replicas {
			phase = v1alpha1.ScalePhase
		} else if upgrading {
			phase = v1alpha1.UpgradePhase
		} else {
			phase = v1alpha1.NormalPhase
		}
		if component == label.PDLabelVal {
			tc.Status.PD.Phase = phase
		} else if component == label.TiFlashLabelVal {
			tc.Status.TiFlash.Phase = phase
		}
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
	case label.TiCDCLabelVal, label.PumpLabelVal:
		if upgrading {
			phase = v1alpha1.UpgradePhase
		} else {
			phase = v1alpha1.NormalPhase
		}
		if component == label.TiCDCLabelVal {
			tc.Status.TiCDC.Phase = phase
		} else if component == label.PumpLabelVal {
			tc.Status.Pump.Phase = phase
		}
	}

	return nil
}

func syncComponentMembers(context *ComponentContext) error {
	tc := context.tc
	component := context.component
	dependencies := context.dependencies

	switch component {
	case label.PDLabelVal:
		ns := tc.GetNamespace()
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
			status := m.getTiFlashStore(store)
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
			status := m.getTiFlashStore(store)
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

func getComponentMemberName(context *ComponentContext) string {
	tc := context.tc
	component := context.component

	tcName := tc.GetName()
	var componentMemberName string
	switch component {
	case label.PDLabelVal:
		componentMemberName = controller.PDMemberName(tcName)
	case label.TiKVLabelVal:
		componentMemberName = controller.TiKVMemberName(tcName)
	case label.TiFlashLabelVal:
		componentMemberName = controller.TiFlashMemberName(tcName)
	case label.TiDBLabelVal:
		componentMemberName = controller.TiDBMemberName(tcName)
	case label.TiCDCLabelVal:
		componentMemberName = controller.TiCDCMemberName(tcName)
	case label.PumpLabelVal:
		componentMemberName = controller.PumpMemberName(tcName)
	}
	return componentMemberName
}

func syncComponentImage(context *ComponentContext, set *apps.StatefulSet) error {
	tc := context.tc
	component := context.component

	switch component {
	case label.PDLabelVal:
		tc.Status.PD.Image = ""
		c := filterContainer(set, "pd")
		if c != nil {
			tc.Status.PD.Image = c.Image
		}
	case label.TiKVLabelVal:
		tc.Status.TiKV.Image = ""
		c := filterContainer(set, "tikv")
		if c != nil {
			tc.Status.TiKV.Image = c.Image
		}
	case label.TiFlashLabelVal:
		tc.Status.TiFlash.Image = ""
		c := filterContainer(set, "tiflash")
		if c != nil {
			tc.Status.TiFlash.Image = c.Image
		}
	case label.TiDBLabelVal:
		tc.Status.TiDB.Image = ""
		c := filterContainer(set, "tidb")
		if c != nil {
			tc.Status.TiDB.Image = c.Image
		}
	}
	return nil
}
