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
	"path"
	"strconv"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
	v1 "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"k8s.io/utils/pointer"
)

const (
	// pdClusterCertPath is where the cert for inter-cluster communication stored (if any)
	pdClusterCertPath  = "/var/lib/pd-tls"
	tidbClientCertPath = "/var/lib/tidb-client-tls"
)

type pdMemberManager struct {
	pdControl    pdapi.PDControlInterface
	setControl   controller.StatefulSetControlInterface
	svcControl   controller.ServiceControlInterface
	podControl   controller.PodControlInterface
	typedControl controller.TypedControlInterface
	setLister    v1.StatefulSetLister
	svcLister    corelisters.ServiceLister
	podLister    corelisters.PodLister
	epsLister    corelisters.EndpointsLister
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
	podControl controller.PodControlInterface,
	typedControl controller.TypedControlInterface,
	setLister v1.StatefulSetLister,
	svcLister corelisters.ServiceLister,
	podLister corelisters.PodLister,
	epsLister corelisters.EndpointsLister,
	pvcLister corelisters.PersistentVolumeClaimLister,
	pdScaler Scaler,
	pdUpgrader Upgrader,
	autoFailover bool,
	pdFailover Failover) manager.Manager {
	return &pdMemberManager{
		pdControl,
		setControl,
		svcControl,
		podControl,
		typedControl,
		setLister,
		svcLister,
		podLister,
		epsLister,
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
	if tc.Spec.Paused {
		klog.V(4).Infof("tidb cluster %s/%s is paused, skip syncing for pd service", tc.GetNamespace(), tc.GetName())
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := pmm.getNewPDServiceForTidbCluster(tc)
	oldSvcTmp, err := pmm.svcLister.Services(ns).Get(controller.PDMemberName(tcName))
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return pmm.svcControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return err
	}

	oldSvc := oldSvcTmp.DeepCopy()

	equal, err := controller.ServiceEqual(newSvc, oldSvc)
	if err != nil {
		return err
	}
	if !equal {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		// TODO add unit test
		err = controller.SetServiceLastAppliedConfigAnnotation(&svc)
		if err != nil {
			return err
		}
		svc.Spec.ClusterIP = oldSvc.Spec.ClusterIP
		_, err = pmm.svcControl.UpdateService(tc, &svc)
		return err
	}

	return nil
}

func (pmm *pdMemberManager) syncPDHeadlessServiceForTidbCluster(tc *v1alpha1.TidbCluster) error {
	if tc.Spec.Paused {
		klog.V(4).Infof("tidb cluster %s/%s is paused, skip syncing for pd headless service", tc.GetNamespace(), tc.GetName())
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := getNewPDHeadlessServiceForTidbCluster(tc)
	oldSvc, err := pmm.svcLister.Services(ns).Get(controller.PDPeerMemberName(tcName))
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return pmm.svcControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return err
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
		_, err = pmm.svcControl.UpdateService(tc, &svc)
		return err
	}

	return nil
}

func (pmm *pdMemberManager) syncPDStatefulSetForTidbCluster(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	oldPDSetTmp, err := pmm.setLister.StatefulSets(ns).Get(controller.PDMemberName(tcName))
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	setNotExist := errors.IsNotFound(err)

	oldPDSet := oldPDSetTmp.DeepCopy()

	if err := pmm.syncTidbClusterStatus(tc, oldPDSet); err != nil {
		klog.Errorf("failed to sync TidbCluster: [%s/%s]'s status, error: %v", ns, tcName, err)
	}

	if tc.Spec.Paused {
		klog.V(4).Infof("tidb cluster %s/%s is paused, skip syncing for pd statefulset", tc.GetNamespace(), tc.GetName())
		return nil
	}

	cm, err := pmm.syncPDConfigMap(tc, oldPDSet)
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
		if err := pmm.setControl.CreateStatefulSet(tc, newPDSet); err != nil {
			return err
		}
		tc.Status.PD.StatefulSet = &apps.StatefulSetStatus{}
		return controller.RequeueErrorf("TidbCluster: [%s/%s], waiting for PD cluster running", ns, tcName)
	}

	if !tc.Status.PD.Synced {
		force := NeedForceUpgrade(tc)
		if force {
			tc.Status.PD.Phase = v1alpha1.UpgradePhase
			setUpgradePartition(newPDSet, 0)
			errSTS := updateStatefulSet(pmm.setControl, tc, newPDSet, oldPDSet)
			return controller.RequeueErrorf("tidbcluster: [%s/%s]'s pd needs force upgrade, %v", ns, tcName, errSTS)
		}
	}

	if !templateEqual(newPDSet, oldPDSet) || tc.Status.PD.Phase == v1alpha1.UpgradePhase {
		if err := pmm.pdUpgrader.Upgrade(tc, oldPDSet, newPDSet); err != nil {
			return err
		}
	}

	if err := pmm.pdScaler.Scale(tc, oldPDSet, newPDSet); err != nil {
		return err
	}

	if pmm.autoFailover {
		if pmm.shouldRecover(tc) {
			pmm.pdFailover.Recover(tc)
		} else if tc.PDAllPodsStarted() && !tc.PDAllMembersReady() || tc.PDAutoFailovering() {
			if err := pmm.pdFailover.Failover(tc); err != nil {
				return err
			}
		}
	}

	return updateStatefulSet(pmm.setControl, tc, newPDSet, oldPDSet)
}

// shouldRecover checks whether we should perform recovery operation.
func (pmm *pdMemberManager) shouldRecover(tc *v1alpha1.TidbCluster) bool {
	if tc.Status.PD.FailureMembers == nil {
		return false
	}
	// If all desired replicas (excluding failover pods) of tidb cluster are
	// healthy, we can perform our failover recovery operation.
	// Note that failover pods may fail (e.g. lack of resources) and we don't care
	// about them because we're going to delete them.
	for ordinal := range tc.PDStsDesiredOrdinals(true) {
		name := fmt.Sprintf("%s-%d", controller.PDMemberName(tc.GetName()), ordinal)
		pod, err := pmm.podLister.Pods(tc.Namespace).Get(name)
		if err != nil {
			klog.Errorf("pod %s/%s does not exist: %v", tc.Namespace, name, err)
			return false
		}
		if !podutil.IsPodReady(pod) {
			return false
		}
		status, ok := tc.Status.PD.Members[pod.Name]
		if !ok || !status.Health {
			return false
		}
	}
	return true
}

func (pmm *pdMemberManager) syncTidbClusterStatus(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) error {
	if set == nil {
		// skip if not created yet
		return nil
	}

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
			klog.Warningf("PD member: [%d] doesn't have a name, and can't get it from clientUrls: [%s], memberHealth Info: [%v] in [%s/%s]",
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
	tc.Status.PD.Image = ""
	c := filterContainer(set, "pd")
	if c != nil {
		tc.Status.PD.Image = c.Image
	}

	// k8s check
	err = pmm.collectUnjoinedMembers(tc, set, pdStatus)
	if err != nil {
		return err
	}
	return nil
}

// syncPDConfigMap syncs the configmap of PD
func (pmm *pdMemberManager) syncPDConfigMap(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) (*corev1.ConfigMap, error) {

	// For backward compatibility, only sync tidb configmap when .pd.config is non-nil
	if tc.Spec.PD.Config == nil {
		return nil, nil
	}
	newCm, err := getPDConfigMap(tc)
	if err != nil {
		return nil, err
	}
	if set != nil && tc.BasePDSpec().ConfigUpdateStrategy() == v1alpha1.ConfigUpdateStrategyInPlace {
		inUseName := FindConfigMapVolume(&set.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, controller.PDMemberName(tc.Name))
		})
		if inUseName != "" {
			newCm.Name = inUseName
		}
	}

	return pmm.typedControl.CreateOrUpdateConfigMap(tc, newCm)
}

func (pmm *pdMemberManager) getNewPDServiceForTidbCluster(tc *v1alpha1.TidbCluster) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	svcName := controller.PDMemberName(tcName)
	instanceName := tc.GetInstanceName()
	pdSelector := label.New().Instance(instanceName).PD()
	pdLabels := pdSelector.Copy().UsedByEndUser().Labels()

	pdService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          pdLabels,
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
			Selector: pdSelector.Labels(),
		},
	}
	// if set pd service type ,overwrite global variable services
	svcSpec := tc.Spec.PD.Service
	if svcSpec != nil {
		if svcSpec.Type != "" {
			pdService.Spec.Type = svcSpec.Type
		}
		pdService.ObjectMeta.Annotations = copyAnnotations(svcSpec.Annotations)
		if svcSpec.LoadBalancerIP != nil {
			pdService.Spec.LoadBalancerIP = *svcSpec.LoadBalancerIP
		}
		if svcSpec.ClusterIP != nil {
			pdService.Spec.ClusterIP = *svcSpec.ClusterIP
		}
		if svcSpec.PortName != nil {
			pdService.Spec.Ports[0].Name = *svcSpec.PortName
		}
	}
	return pdService
}

func getNewPDHeadlessServiceForTidbCluster(tc *v1alpha1.TidbCluster) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	svcName := controller.PDPeerMemberName(tcName)
	instanceName := tc.GetInstanceName()
	pdSelector := label.New().Instance(instanceName).PD()
	pdLabels := pdSelector.Copy().UsedByPeer().Labels()

	return &corev1.Service{
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
}

func (pmm *pdMemberManager) pdStatefulSetIsUpgrading(set *apps.StatefulSet, tc *v1alpha1.TidbCluster) (bool, error) {
	if statefulSetIsUpgrading(set) {
		return true, nil
	}
	selector, err := label.New().
		Instance(tc.GetInstanceName()).
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

func getFailureReplicas(tc *v1alpha1.TidbCluster) int {
	failureReplicas := 0
	for _, failureMember := range tc.Status.PD.FailureMembers {
		if failureMember.MemberDeleted {
			failureReplicas++
		}
	}
	return failureReplicas
}

func getNewPDSetForTidbCluster(tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
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
		{Name: v1alpha1.PDMemberType.String(), MountPath: "/var/lib/pd"},
	}
	if tc.IsTLSClusterEnabled() {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "pd-tls", ReadOnly: true, MountPath: "/var/lib/pd-tls",
		})
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

	storageRequest, err := controller.ParseStorageRequest(tc.Spec.PD.Requests)
	if err != nil {
		return nil, fmt.Errorf("cannot parse storage request for PD, tidbcluster %s/%s, error: %v", tc.Namespace, tc.Name, err)
	}

	pdLabel := label.New().Instance(instanceName).PD()
	setName := controller.PDMemberName(tcName)
	podAnnotations := CombineAnnotations(controller.AnnProm(2379), basePDSpec.Annotations())
	stsAnnotations := getStsAnnotations(tc, label.PDLabelVal)
	failureReplicas := getFailureReplicas(tc)

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
	podSpec.Volumes = vols
	podSpec.Containers = []corev1.Container{pdContainer}

	pdSet := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            setName,
			Namespace:       ns,
			Labels:          pdLabel.Labels(),
			Annotations:     stsAnnotations,
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
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: apps.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &apps.RollingUpdateStatefulSetStrategy{
					Partition: controller.Int32Ptr(tc.Spec.PD.Replicas + int32(failureReplicas)),
				}},
		},
	}

	return pdSet, nil
}

func getPDConfigMap(tc *v1alpha1.TidbCluster) (*corev1.ConfigMap, error) {

	// For backward compatibility, only sync tidb configmap when .tidb.config is non-nil
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
		if config.Security == nil {
			config.Security = &v1alpha1.PDSecurityConfig{}
		}
		config.Security.CAPath = pointer.StringPtr(path.Join(pdClusterCertPath, tlsSecretRootCAKey))
		config.Security.CertPath = pointer.StringPtr(path.Join(pdClusterCertPath, corev1.TLSCertKey))
		config.Security.KeyPath = pointer.StringPtr(path.Join(pdClusterCertPath, corev1.TLSPrivateKeyKey))
	}
	// Versions below v4.0 do not support Dashboard
	if tc.Spec.TiDB.IsTLSClientEnabled() && !tc.SkipTLSWhenConnectTiDB() && clusterVersionGE4 {
		if config.Dashboard == nil {
			config.Dashboard = &v1alpha1.DashboardConfig{}
		}
		config.Dashboard.TiDBCAPath = pointer.StringPtr(path.Join(tidbClientCertPath, tlsSecretRootCAKey))
		config.Dashboard.TiDBCertPath = pointer.StringPtr(path.Join(tidbClientCertPath, corev1.TLSCertKey))
		config.Dashboard.TiDBKeyPath = pointer.StringPtr(path.Join(tidbClientCertPath, corev1.TLSPrivateKeyKey))
	}

	confText, err := MarshalTOML(config)
	if err != nil {
		return nil, err
	}
	startScript, err := RenderPDStartScript(&PDStartScriptModel{Scheme: tc.Scheme()})
	if err != nil {
		return nil, err
	}

	instanceName := tc.GetInstanceName()
	pdLabel := label.New().Instance(instanceName).PD().Labels()
	cm := &corev1.ConfigMap{
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
	if tc.BasePDSpec().ConfigUpdateStrategy() == v1alpha1.ConfigUpdateStrategyRollingUpdate {
		if err := AddConfigMapDigestSuffix(cm); err != nil {
			return nil, err
		}
	}
	return cm, nil
}

func clusterVersionGreaterThanOrEqualTo4(version string) (bool, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return true, err
	}

	return v.Major() >= 4, nil
}

func (pmm *pdMemberManager) collectUnjoinedMembers(tc *v1alpha1.TidbCluster, set *apps.StatefulSet, pdStatus map[string]v1alpha1.PDMember) error {
	podSelector, podSelectErr := metav1.LabelSelectorAsSelector(set.Spec.Selector)
	if podSelectErr != nil {
		return podSelectErr
	}
	pods, podErr := pmm.podLister.Pods(tc.Namespace).List(podSelector)
	if podErr != nil {
		return podErr
	}
	for _, pod := range pods {
		var joined = false
		for podName := range pdStatus {
			if strings.EqualFold(pod.Name, podName) {
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
			pvc, err := pmm.pvcLister.PersistentVolumeClaims(tc.Namespace).Get(pvcName)
			if err != nil {
				return err
			}
			tc.Status.PD.UnjoinedMembers[pod.Name] = v1alpha1.UnjoinedMember{
				PodName:   pod.Name,
				PVCUID:    pvc.UID,
				CreatedAt: metav1.Now(),
			}
		} else {
			if tc.Status.PD.UnjoinedMembers != nil {
				if _, ok := tc.Status.PD.UnjoinedMembers[pod.Name]; ok {
					delete(tc.Status.PD.UnjoinedMembers, pod.Name)
				}

			}
		}
	}
	return nil
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
