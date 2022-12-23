// Copyright 2020 PingCAP, Inc.
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
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager"
	startscriptv1 "github.com/pingcap/tidb-operator/pkg/manager/member/startscript/v1"
	"github.com/pingcap/tidb-operator/pkg/manager/suspender"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"
	"github.com/pingcap/tidb-operator/pkg/util"

	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"k8s.io/utils/pointer"
)

const (
	// dmMasterDataVolumeMountPath is the mount path for dm-master data volume
	dmMasterDataVolumeMountPath = "/var/lib/dm-master"
	// dmMasterClusterCertPath is where the cert for inter-cluster communication stored (if any)
	dmMasterClusterCertPath = "/var/lib/dm-master-tls"
	// DefaultStorageSize is the default pvc request storage size for dm
	DefaultStorageSize = "10Gi"
)

type masterMemberManager struct {
	deps      *controller.Dependencies
	scaler    Scaler
	upgrader  DMUpgrader
	failover  DMFailover
	suspender suspender.Suspender
}

// NewMasterMemberManager returns a *masterMemberManager
func NewMasterMemberManager(deps *controller.Dependencies, masterScaler Scaler, masterUpgrader DMUpgrader, masterFailover DMFailover, spder suspender.Suspender) manager.DMManager {
	return &masterMemberManager{
		deps:      deps,
		scaler:    masterScaler,
		upgrader:  masterUpgrader,
		failover:  masterFailover,
		suspender: spder,
	}
}

func (m *masterMemberManager) SyncDM(dc *v1alpha1.DMCluster) error {
	// skip sync if dm master is suspended
	component := v1alpha1.DMMasterMemberType
	needSuspend, err := m.suspender.SuspendComponent(dc, component)
	if err != nil {
		return fmt.Errorf("suspend %s failed: %v", component, err)
	}
	if needSuspend {
		klog.Infof("component %s for cluster %s/%s is suspended, skip syncing", component, dc.GetNamespace(), dc.GetName())
		return nil
	}

	// Sync dm-master Service
	if err := m.syncMasterServiceForDMCluster(dc); err != nil {
		return err
	}

	// Sync dm-master Headless Service
	if err := m.syncMasterHeadlessServiceForDMCluster(dc); err != nil {
		return err
	}

	// Sync dm-master StatefulSet
	return m.syncMasterStatefulSetForDMCluster(dc)
}

func (m *masterMemberManager) syncMasterServiceForDMCluster(dc *v1alpha1.DMCluster) error {
	if dc.Spec.Paused {
		klog.V(4).Infof("dm cluster %s/%s is paused, skip syncing for dm-master service", dc.GetNamespace(), dc.GetName())
		return nil
	}

	ns := dc.GetNamespace()
	dcName := dc.GetName()

	newSvc := m.getNewMasterServiceForDMCluster(dc)
	oldSvcTmp, err := m.deps.ServiceLister.Services(ns).Get(controller.DMMasterMemberName(dcName))
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return m.deps.ServiceControl.CreateService(dc, newSvc)
	}
	if err != nil {
		return fmt.Errorf("syncMasterServiceForDMCluster: failed to get svc %s for cluster %s/%s, error: %s", controller.DMMasterMemberName(dcName), ns, dcName, err)
	}

	oldSvc := oldSvcTmp.DeepCopy()
	util.RetainManagedFields(newSvc, oldSvc)

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
		for k, v := range newSvc.Annotations {
			svc.Annotations[k] = v
		}
		_, err = m.deps.ServiceControl.UpdateService(dc, &svc)
		return err
	}

	return nil
}

func (m *masterMemberManager) syncMasterHeadlessServiceForDMCluster(dc *v1alpha1.DMCluster) error {
	if dc.Spec.Paused {
		klog.V(4).Infof("dm cluster %s/%s is paused, skip syncing for dm-master headless service", dc.GetNamespace(), dc.GetName())
		return nil
	}

	ns := dc.GetNamespace()
	dcName := dc.GetName()

	newSvc := getNewMasterHeadlessServiceForDMCluster(dc)
	oldSvc, err := m.deps.ServiceLister.Services(ns).Get(controller.DMMasterPeerMemberName(dcName))
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return m.deps.ServiceControl.CreateService(dc, newSvc)
	}
	if err != nil {
		return fmt.Errorf("syncMasterHeadlessServiceForDMCluster: failed to get svc %s for cluster %s/%s, error: %s", controller.DMMasterPeerMemberName(dcName), ns, dcName, err)
	}

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
		_, err = m.deps.ServiceControl.UpdateService(dc, &svc)
		return err
	}

	return nil
}

func (m *masterMemberManager) syncMasterStatefulSetForDMCluster(dc *v1alpha1.DMCluster) error {
	ns := dc.GetNamespace()
	dcName := dc.GetName()

	oldMasterSetTmp, err := m.deps.StatefulSetLister.StatefulSets(ns).Get(controller.DMMasterMemberName(dcName))
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncMasterStatefulSetForDMCluster: fail to get sts %s for cluster %s/%s, error: %s", controller.DMMasterMemberName(dcName), ns, dcName, err)
	}

	setNotExist := errors.IsNotFound(err)
	oldMasterSet := oldMasterSetTmp.DeepCopy()

	if err := m.syncDMClusterStatus(dc, oldMasterSet); err != nil {
		klog.Errorf("failed to sync DMCluster: [%s/%s]'s status, error: %v", ns, dcName, err)
	}

	if dc.Spec.Paused {
		klog.V(4).Infof("dm cluster %s/%s is paused, skip syncing for dm-master statefulset", dc.GetNamespace(), dc.GetName())
		return nil
	}

	cm, err := m.syncMasterConfigMap(dc, oldMasterSet)
	if err != nil {
		return err
	}
	newMasterSet, err := getNewMasterSetForDMCluster(dc, cm)
	if err != nil {
		return err
	}
	if setNotExist {
		err = mngerutils.SetStatefulSetLastAppliedConfigAnnotation(newMasterSet)
		if err != nil {
			return err
		}
		if err := m.deps.StatefulSetControl.CreateStatefulSet(dc, newMasterSet); err != nil {
			return err
		}
		dc.Status.Master.StatefulSet = &apps.StatefulSetStatus{}
		return controller.RequeueErrorf("DMCluster: [%s/%s], waiting for dm-master cluster running", ns, dcName)
	}

	// Force update takes precedence over scaling because force upgrade won't take effect when cluster gets stuck at scaling
	if !dc.Status.Master.Synced && NeedForceUpgrade(dc.Annotations) {
		dc.Status.Master.Phase = v1alpha1.UpgradePhase
		mngerutils.SetUpgradePartition(newMasterSet, 0)
		errSTS := mngerutils.UpdateStatefulSet(m.deps.StatefulSetControl, dc, newMasterSet, oldMasterSet)
		return controller.RequeueErrorf("dmcluster: [%s/%s]'s dm-master needs force upgrade, %v", ns, dcName, errSTS)
	}

	// Scaling takes precedence over normal upgrading because:
	// - if a dm-master fails in the upgrading, users may want to delete it or add
	//   new replicas
	// - it's ok to scale in the middle of upgrading (in statefulset controller
	//   scaling takes precedence over upgrading too)
	if err := m.scaler.Scale(dc, oldMasterSet, newMasterSet); err != nil {
		return err
	}

	// Perform failover logic if necessary. Note that this will only update
	// DMCluster status. The actual scaling performs in next sync loop (if a
	// new replica needs to be added).
	if m.deps.CLIConfig.AutoFailover {
		if m.shouldRecover(dc) {
			m.failover.Recover(dc)
		} else if dc.MasterAllPodsStarted() && !dc.MasterAllMembersReady() || dc.MasterAutoFailovering() {
			if err := m.failover.Failover(dc); err != nil {
				return err
			}
		}
	}

	if !templateEqual(newMasterSet, oldMasterSet) || dc.Status.Master.Phase == v1alpha1.UpgradePhase {
		if err := m.upgrader.Upgrade(dc, oldMasterSet, newMasterSet); err != nil {
			return err
		}
	}

	return mngerutils.UpdateStatefulSet(m.deps.StatefulSetControl, dc, newMasterSet, oldMasterSet)
}

// shouldRecover checks whether we should perform recovery operation.
func (m *masterMemberManager) shouldRecover(dc *v1alpha1.DMCluster) bool {
	if dc.Status.Master.FailureMembers == nil {
		return false
	}
	// If all desired replicas (excluding failover pods) of dm cluster are
	// healthy, we can perform our failover recovery operation.
	// Note that failover pods may fail (e.g. lack of resources) and we don't care
	// about them because we're going to delete them.
	for ordinal := range dc.MasterStsDesiredOrdinals(true) {
		name := fmt.Sprintf("%s-%d", controller.DMMasterMemberName(dc.GetName()), ordinal)
		pod, err := m.deps.PodLister.Pods(dc.Namespace).Get(name)
		if err != nil {
			klog.Errorf("pod %s/%s does not exist: %v", dc.Namespace, name, err)
			return false
		}
		if !podutil.IsPodReady(pod) {
			return false
		}
		status, ok := dc.Status.Master.Members[pod.Name]
		if !ok || !status.Health {
			return false
		}
	}
	return true
}

func (m *masterMemberManager) syncDMClusterStatus(dc *v1alpha1.DMCluster, set *apps.StatefulSet) error {
	if set == nil {
		// skip if not created yet
		return nil
	}

	ns := dc.GetNamespace()
	dcName := dc.GetName()

	dc.Status.Master.StatefulSet = &set.Status

	upgrading, err := m.masterStatefulSetIsUpgrading(set, dc)
	if err != nil {
		return err
	}

	// Scaling takes precedence over upgrading.
	if dc.MasterStsDesiredReplicas() != *set.Spec.Replicas {
		dc.Status.Master.Phase = v1alpha1.ScalePhase
	} else if upgrading {
		dc.Status.Master.Phase = v1alpha1.UpgradePhase
	} else {
		dc.Status.Master.Phase = v1alpha1.NormalPhase
	}

	dmClient := controller.GetMasterClient(m.deps.DMMasterControl, dc)

	mastersInfo, err := dmClient.GetMasters()
	if err != nil {
		dc.Status.Master.Synced = false
		// get endpoints info
		eps, epErr := m.deps.EndpointLister.Endpoints(ns).Get(controller.DMMasterMemberName(dcName))
		if epErr != nil {
			return fmt.Errorf("syncDMClusterStatus: failed to get endpoints %s for cluster %s/%s, err: %s, epErr %s", controller.DMMasterMemberName(dcName), ns, dcName, err, epErr)
		}
		// dm-master service has no endpoints
		if eps != nil && len(eps.Subsets) == 0 {
			return fmt.Errorf("%s, service %s/%s has no endpoints", err, ns, controller.DMMasterMemberName(dcName))
		}
		return err
	}

	leader, err := dmClient.GetLeader()
	if err != nil {
		dc.Status.Master.Synced = false
		return err
	}
	masterStatus := map[string]v1alpha1.MasterMember{}
	for _, master := range mastersInfo {
		id := master.MemberID
		var clientURL string
		if len(master.ClientURLs) > 0 {
			clientURL = master.ClientURLs[0]
		}
		name := master.Name
		if len(name) == 0 {
			klog.Warningf("dm-master member: [%s] doesn't have a name, clientUrls: [%s], dm-master Info: [%#v] in [%s/%s]",
				id, master.ClientURLs, master, ns, dcName)
			continue
		}

		status := v1alpha1.MasterMember{
			Name:      name,
			ID:        id,
			ClientURL: clientURL,
			Health:    master.Alive,
		}

		oldMasterMember, exist := dc.Status.Master.Members[name]

		status.LastTransitionTime = metav1.Now()
		if exist && status.Health == oldMasterMember.Health {
			status.LastTransitionTime = oldMasterMember.LastTransitionTime
		}

		masterStatus[name] = status
	}

	dc.Status.Master.Synced = true
	dc.Status.Master.Members = masterStatus
	dc.Status.Master.Leader = dc.Status.Master.Members[leader.Name]
	dc.Status.Master.Image = ""
	c := findContainerByName(set, "dm-master")
	if c != nil {
		dc.Status.Master.Image = c.Image
	}

	// k8s check
	err = m.collectUnjoinedMembers(dc, set, masterStatus)
	if err != nil {
		return err
	}
	return nil
}

// syncMasterConfigMap syncs the configmap of dm-master
func (m *masterMemberManager) syncMasterConfigMap(dc *v1alpha1.DMCluster, set *apps.StatefulSet) (*corev1.ConfigMap, error) {
	newCm, err := getMasterConfigMap(dc)
	if err != nil {
		return nil, err
	}

	var inUseName string
	if set != nil {
		inUseName = mngerutils.FindConfigMapVolume(&set.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, controller.DMMasterMemberName(dc.Name))
		})
	}

	err = mngerutils.UpdateConfigMapIfNeed(m.deps.ConfigMapLister, dc.BaseMasterSpec().ConfigUpdateStrategy(), inUseName, newCm)
	if err != nil {
		return nil, err
	}
	return m.deps.TypedControl.CreateOrUpdateConfigMap(dc, newCm)
}

func (m *masterMemberManager) getNewMasterServiceForDMCluster(dc *v1alpha1.DMCluster) *corev1.Service {
	ns := dc.Namespace
	dcName := dc.Name
	svcName := controller.DMMasterMemberName(dcName)
	instanceName := dc.GetInstanceName()
	masterSelector := label.NewDM().Instance(instanceName).DMMaster()
	masterLabels := masterSelector.Copy().UsedByEndUser().Labels()

	ports := []corev1.ServicePort{
		{
			Name:       "dm-master",
			Port:       8261,
			TargetPort: intstr.FromInt(8261),
			Protocol:   corev1.ProtocolTCP,
		},
	}
	masterSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          masterLabels,
			OwnerReferences: []metav1.OwnerReference{controller.GetDMOwnerRef(dc)},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Ports:    ports,
			Selector: masterSelector.Labels(),
		},
	}

	// override fields with user-defined ServiceSpec
	svcSpec := dc.Spec.Master.Service
	if svcSpec != nil {
		if svcSpec.Type != "" {
			masterSvc.Spec.Type = svcSpec.Type
		}
		masterSvc.ObjectMeta.Annotations = util.CopyStringMap(svcSpec.Annotations)
		masterSvc.ObjectMeta.Labels = util.CombineStringMap(masterSvc.ObjectMeta.Labels, svcSpec.Labels)
		masterSvc.Spec.Ports[0].NodePort = svcSpec.GetMasterNodePort()
		if svcSpec.Type == corev1.ServiceTypeLoadBalancer {
			if svcSpec.LoadBalancerIP != nil {
				masterSvc.Spec.LoadBalancerIP = *svcSpec.LoadBalancerIP
			}
			if svcSpec.LoadBalancerSourceRanges != nil {
				masterSvc.Spec.LoadBalancerSourceRanges = svcSpec.LoadBalancerSourceRanges
			}
		}
		if svcSpec.ExternalTrafficPolicy != nil {
			masterSvc.Spec.ExternalTrafficPolicy = *svcSpec.ExternalTrafficPolicy
		}
		if svcSpec.ClusterIP != nil {
			masterSvc.Spec.ClusterIP = *svcSpec.ClusterIP
		}
		if svcSpec.PortName != nil {
			masterSvc.Spec.Ports[0].Name = *svcSpec.PortName
		}
	}
	return masterSvc
}

func getNewMasterHeadlessServiceForDMCluster(dc *v1alpha1.DMCluster) *corev1.Service {
	ns := dc.Namespace
	tcName := dc.Name
	svcName := controller.DMMasterPeerMemberName(tcName)
	instanceName := dc.GetInstanceName()
	masterSelector := label.NewDM().Instance(instanceName).DMMaster()
	masterLabels := masterSelector.Copy().UsedByPeer().Labels()

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          masterLabels,
			OwnerReferences: []metav1.OwnerReference{controller.GetDMOwnerRef(dc)},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:       "dm-master-peer",
					Port:       8291,
					TargetPort: intstr.FromInt(8291),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector:                 masterSelector.Labels(),
			PublishNotReadyAddresses: true,
		},
	}
}

func (m *masterMemberManager) masterStatefulSetIsUpgrading(set *apps.StatefulSet, dc *v1alpha1.DMCluster) (bool, error) {
	if mngerutils.StatefulSetIsUpgrading(set) {
		return true, nil
	}
	instanceName := dc.GetInstanceName()
	selector, err := label.NewDM().
		Instance(instanceName).
		DMMaster().
		Selector()
	if err != nil {
		return false, err
	}
	masterPods, err := m.deps.PodLister.Pods(dc.GetNamespace()).List(selector)
	if err != nil {
		return false, fmt.Errorf("masterStatefulSetIsUpgrading: failed to list pods for cluster %s/%s, selector %s, error: %v", dc.GetNamespace(), instanceName, selector, err)
	}
	for _, pod := range masterPods {
		revisionHash, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return false, nil
		}
		if revisionHash != dc.Status.Master.StatefulSet.UpdateRevision {
			return true, nil
		}
	}
	return false, nil
}

func getDMMasterFailureReplicas(dc *v1alpha1.DMCluster) int {
	failureReplicas := 0
	for _, failureMember := range dc.Status.Master.FailureMembers {
		if failureMember.MemberDeleted {
			failureReplicas++
		}
	}
	return failureReplicas
}

func getNewMasterSetForDMCluster(dc *v1alpha1.DMCluster, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
	ns := dc.Namespace
	dcName := dc.Name
	baseMasterSpec := dc.BaseMasterSpec()
	instanceName := dc.GetInstanceName()
	if cm == nil {
		return nil, fmt.Errorf("config map for dm-master is not found, dmcluster %s/%s", dc.Namespace, dc.Name)
	}
	masterConfigMap := cm.Name

	annoMount, annoVolume := annotationsMountVolume()
	dataVolumeName := string(v1alpha1.GetStorageVolumeName("", v1alpha1.DMMasterMemberType))
	volMounts := []corev1.VolumeMount{
		annoMount,
		{Name: "config", ReadOnly: true, MountPath: "/etc/dm-master"},
		{Name: "startup-script", ReadOnly: true, MountPath: "/usr/local/bin"},
		{Name: dataVolumeName, MountPath: dmMasterDataVolumeMountPath},
	}
	volMounts = append(volMounts, dc.Spec.Master.AdditionalVolumeMounts...)

	if dc.IsTLSClusterEnabled() {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "dm-master-tls", ReadOnly: true, MountPath: "/var/lib/dm-master-tls",
		})
	}

	vols := []corev1.Volume{
		annoVolume,
		{Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: masterConfigMap,
					},
					Items: []corev1.KeyToPath{{Key: "config-file", Path: "dm-master.toml"}},
				},
			},
		},
		{Name: "startup-script",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: masterConfigMap,
					},
					Items: []corev1.KeyToPath{{Key: "startup-script", Path: "dm_master_start_script.sh"}},
				},
			},
		},
	}
	if dc.IsTLSClusterEnabled() {
		vols = append(vols, corev1.Volume{
			Name: "dm-master-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterTLSSecretName(dc.Name, label.DMMasterLabelVal),
				},
			},
		})
	}

	for _, tlsClientSecretName := range dc.Spec.TLSClientSecretNames {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: tlsClientSecretName, ReadOnly: true, MountPath: fmt.Sprintf("/var/lib/source-tls/%s", tlsClientSecretName),
		})
		vols = append(vols, corev1.Volume{
			Name: tlsClientSecretName, VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: tlsClientSecretName,
				},
			},
		})
	}

	storageSize := DefaultStorageSize
	if dc.Spec.Master.StorageSize != "" {
		storageSize = dc.Spec.Master.StorageSize
	}
	rs, err := resource.ParseQuantity(storageSize)
	if err != nil {
		return nil, fmt.Errorf("cannot parse storage request for dm-master, dmcluster %s/%s, error: %v", dc.Namespace, dc.Name, err)
	}
	storageRequest := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceStorage: rs,
		},
	}

	setName := controller.DMMasterMemberName(dcName)
	stsLabels := label.NewDM().Instance(instanceName).DMMaster()
	podLabels := util.CombineStringMap(stsLabels, baseMasterSpec.Labels())
	stsAnnotations := getStsAnnotations(dc.Annotations, label.DMMasterLabelVal)
	podAnnotations := util.CombineStringMap(controller.AnnProm(8261, "/metrics"), baseMasterSpec.Annotations())
	failureReplicas := getDMMasterFailureReplicas(dc)

	deleteSlotsNumber, err := util.GetDeleteSlotsNumber(stsAnnotations)
	if err != nil {
		return nil, fmt.Errorf("get delete slots number of statefulset %s/%s failed, err:%v", ns, setName, err)
	}

	masterContainer := corev1.Container{
		Name:            v1alpha1.DMMasterMemberType.String(),
		Image:           dc.MasterImage(),
		ImagePullPolicy: baseMasterSpec.ImagePullPolicy(),
		Command:         []string{"/bin/sh", "/usr/local/bin/dm_master_start_script.sh"},
		Ports: []corev1.ContainerPort{
			{
				Name:          "peer",
				ContainerPort: int32(8291),
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "client",
				ContainerPort: int32(8261),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: volMounts,
		Resources:    controller.ContainerResource(dc.Spec.Master.ResourceRequirements),
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
			Value: controller.DMMasterPeerMemberName(dcName),
		},
		{
			Name:  "SERVICE_NAME",
			Value: controller.DMMasterMemberName(dcName),
		},
		{
			Name:  "SET_NAME",
			Value: setName,
		},
		{
			Name:  "TZ",
			Value: dc.Timezone(),
		},
	}

	podSpec := baseMasterSpec.BuildPodSpec()
	if baseMasterSpec.HostNetwork() {
		env = append(env, corev1.EnvVar{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		})
	}
	masterContainer.Env = util.AppendEnv(env, baseMasterSpec.Env())
	masterContainer.EnvFrom = baseMasterSpec.EnvFrom()
	podSpec.Volumes = append(vols, baseMasterSpec.AdditionalVolumes()...)

	podSpec.Containers, err = MergePatchContainers([]corev1.Container{masterContainer}, baseMasterSpec.AdditionalContainers())
	if err != nil {
		return nil, fmt.Errorf("failed to merge containers spec for DC [%s/%s], error: %v", dc.Namespace, dc.Name, err)
	}

	var initContainers []corev1.Container // no default initContainers now
	podSpec.InitContainers = append(initContainers, baseMasterSpec.InitContainers()...)

	updateStrategy := apps.StatefulSetUpdateStrategy{}
	if baseMasterSpec.StatefulSetUpdateStrategy() == apps.OnDeleteStatefulSetStrategyType {
		updateStrategy.Type = apps.OnDeleteStatefulSetStrategyType
	} else {
		updateStrategy.Type = apps.RollingUpdateStatefulSetStrategyType
		updateStrategy.RollingUpdate = &apps.RollingUpdateStatefulSetStrategy{
			Partition: pointer.Int32Ptr(dc.Spec.Master.Replicas + int32(failureReplicas) + deleteSlotsNumber),
		}
	}

	masterSet := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            setName,
			Namespace:       ns,
			Labels:          stsLabels,
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetDMOwnerRef(dc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(dc.Spec.Master.Replicas + int32(failureReplicas)),
			Selector: stsLabels.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: dataVolumeName,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						StorageClassName: dc.Spec.Master.StorageClassName,
						Resources:        storageRequest,
					},
				},
			},
			ServiceName:         controller.DMMasterPeerMemberName(dcName),
			PodManagementPolicy: baseMasterSpec.PodManagementPolicy(),
			UpdateStrategy:      updateStrategy,
		},
	}

	return masterSet, nil
}

func getMasterConfigMap(dc *v1alpha1.DMCluster) (*corev1.ConfigMap, error) {
	config := v1alpha1.NewMasterConfig()
	if dc.Spec.Master.Config != nil {
		config = dc.Spec.Master.Config.DeepCopy()
	}

	// override CA if tls enabled
	if dc.IsTLSClusterEnabled() {
		config.Set("ssl-ca", path.Join(dmMasterClusterCertPath, tlsSecretRootCAKey))
		config.Set("ssl-cert", path.Join(dmMasterClusterCertPath, corev1.TLSCertKey))
		config.Set("ssl-key", path.Join(dmMasterClusterCertPath, corev1.TLSPrivateKeyKey))
	}

	confText, err := config.MarshalTOML()
	if err != nil {
		return nil, err
	}

	startScript, err := startscriptv1.RenderDMMasterStartScript(&startscriptv1.DMMasterStartScriptModel{
		Scheme:  dc.Scheme(),
		DataDir: filepath.Join(dmMasterDataVolumeMountPath, dc.Spec.Master.DataSubDir),
	})
	if err != nil {
		return nil, err
	}

	instanceName := dc.GetInstanceName()
	masterLabel := label.NewDM().Instance(instanceName).DMMaster().Labels()
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            controller.DMMasterMemberName(dc.Name),
			Namespace:       dc.Namespace,
			Labels:          masterLabel,
			OwnerReferences: []metav1.OwnerReference{controller.GetDMOwnerRef(dc)},
		},
		Data: map[string]string{
			"config-file":    string(confText),
			"startup-script": startScript,
		},
	}
	return cm, nil
}

func (m *masterMemberManager) collectUnjoinedMembers(dc *v1alpha1.DMCluster, set *apps.StatefulSet, masterStatus map[string]v1alpha1.MasterMember) error {
	ns := dc.GetNamespace()
	podSelector, podSelectErr := metav1.LabelSelectorAsSelector(set.Spec.Selector)
	if podSelectErr != nil {
		return podSelectErr
	}
	pods, podErr := m.deps.PodLister.Pods(dc.Namespace).List(podSelector)
	if podErr != nil {
		return fmt.Errorf("collectUnjoinedMembers: failed to list pods for cluster %s/%s, selector %s, error %v", dc.GetNamespace(), dc.GetName(), set.Spec.Selector, podErr)
	}
	for _, pod := range pods {
		var joined = false
		for podName := range masterStatus {
			if strings.EqualFold(pod.Name, podName) {
				joined = true
				break
			}
		}
		if !joined {
			if dc.Status.Master.UnjoinedMembers == nil {
				dc.Status.Master.UnjoinedMembers = map[string]v1alpha1.UnjoinedMember{}
			}
			pvcs, err := util.ResolvePVCFromPod(pod, m.deps.PVCLister)
			if err != nil {
				return fmt.Errorf("collectUnjoinedMembers: failed to get pvcs for pod %s/%s, error: %s", ns, pod.Name, err)
			}
			pvcUIDSet := make(map[types.UID]v1alpha1.EmptyStruct)
			for _, pvc := range pvcs {
				pvcUIDSet[pvc.UID] = v1alpha1.EmptyStruct{}
			}
			dc.Status.Master.UnjoinedMembers[pod.Name] = v1alpha1.UnjoinedMember{
				PodName:   pod.Name,
				PVCUIDSet: pvcUIDSet,
				CreatedAt: metav1.Now(),
			}
		} else {
			if dc.Status.Master.UnjoinedMembers != nil {
				delete(dc.Status.Master.UnjoinedMembers, pod.Name)
			}
		}
	}
	return nil
}

// TODO: seems not used
type FakeMasterMemberManager struct {
	err error
}

func NewFakeMasterMemberManager() *FakeMasterMemberManager {
	return &FakeMasterMemberManager{}
}

func (m *FakeMasterMemberManager) SetSyncError(err error) {
	m.err = err
}

func (m *FakeMasterMemberManager) SyncDM(dc *v1alpha1.DMCluster) error {
	if m.err != nil {
		return m.err
	}
	return nil
}
