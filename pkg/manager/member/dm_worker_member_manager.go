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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/util"

	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
)

const (
	// dmWorkerDataVolumeMountPath is the mount path for dm-worker data volume
	dmWorkerDataVolumeMountPath = "/var/lib/dm-worker"
	// dmWorkerClusterCertPath is where the cert for inter-cluster communication stored (if any)
	dmWorkerClusterCertPath = "/var/lib/dm-worker-tls"
)

type workerMemberManager struct {
	deps     *controller.Dependencies
	scaler   Scaler
	failover DMFailover
}

// NewWorkerMemberManager returns a *ticdcMemberManager
func NewWorkerMemberManager(deps *controller.Dependencies, scaler Scaler, failover DMFailover) manager.DMManager {
	return &workerMemberManager{
		deps:     deps,
		scaler:   scaler,
		failover: failover,
	}
}

func (m *workerMemberManager) SyncDM(dc *v1alpha1.DMCluster) error {
	ns := dc.GetNamespace()
	dcName := dc.GetName()

	if dc.Spec.Worker == nil {
		return nil
	}
	if dc.Spec.Paused {
		klog.Infof("DMCluster %s/%s is paused, skip syncing dm-worker deployment", ns, dcName)
		return nil
	}
	if !dc.MasterIsAvailable() {
		return controller.RequeueErrorf("DMCluster: %s/%s, waiting for dm-master cluster running", ns, dcName)
	}

	// Sync dm-worker Headless Service
	if err := m.syncWorkerHeadlessServiceForDMCluster(dc); err != nil {
		return err
	}

	// Sync dm-worker StatefulSet
	return m.syncWorkerStatefulSetForDMCluster(dc)
}

func (m *workerMemberManager) syncWorkerHeadlessServiceForDMCluster(dc *v1alpha1.DMCluster) error {
	ns := dc.GetNamespace()
	dcName := dc.GetName()

	newSvc := getNewWorkerHeadlessServiceForDMCluster(dc)
	oldSvcTmp, err := m.deps.ServiceLister.Services(ns).Get(controller.DMWorkerPeerMemberName(dcName))
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return m.deps.ServiceControl.CreateService(dc, newSvc)
	}
	if err != nil {
		return fmt.Errorf("syncWorkerHeadlessServiceForDMCluster: failed to get svc %s for cluster %s/%s, error: %s", controller.DMWorkerPeerMemberName(dcName), ns, dcName, err)
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
		_, err = m.deps.ServiceControl.UpdateService(dc, &svc)
		return err
	}

	return nil
}

func getNewWorkerHeadlessServiceForDMCluster(dc *v1alpha1.DMCluster) *corev1.Service {
	ns := dc.Namespace
	dcName := dc.Name
	instanceName := dc.GetInstanceName()
	svcName := controller.DMWorkerPeerMemberName(dcName)
	svcLabel := label.NewDM().Instance(instanceName).DMWorker().Labels()

	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          svcLabel,
			OwnerReferences: []metav1.OwnerReference{controller.GetDMOwnerRef(dc)},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:       "dm-worker",
					Port:       8262,
					TargetPort: intstr.FromInt(int(8262)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector:                 svcLabel,
			PublishNotReadyAddresses: true,
		},
	}
	return &svc
}

func (m *workerMemberManager) syncWorkerStatefulSetForDMCluster(dc *v1alpha1.DMCluster) error {
	ns := dc.GetNamespace()
	dcName := dc.GetName()

	oldStsTmp, err := m.deps.StatefulSetLister.StatefulSets(ns).Get(controller.DMWorkerMemberName(dcName))
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncWorkerStatefulSetForDMCluster: failed to get sts %s for cluster %s/%s, error: %s", controller.DMWorkerMemberName(dcName), ns, dcName, err)
	}

	stsNotExist := errors.IsNotFound(err)
	oldSts := oldStsTmp.DeepCopy()

	// failed to sync dm-worker status will not affect subsequent logic, just print the errors.
	if err := m.syncDMClusterStatus(dc, oldSts); err != nil {
		klog.Errorf("failed to sync DMCluster: [%s/%s]'s dm-worker status, error: %v", ns, dcName, err)
	}

	if dc.Spec.Paused {
		klog.V(4).Infof("dm cluster %s/%s is paused, skip syncing for dm-worker statefulset", dc.GetNamespace(), dc.GetName())
		return nil
	}

	cm, err := m.syncWorkerConfigMap(dc, oldSts)
	if err != nil {
		return err
	}

	// Recover failed workers if any before generating desired statefulset
	if len(dc.Status.Worker.FailureMembers) > 0 {
		m.failover.RemoveUndesiredFailures(dc)
	}
	if len(dc.Status.Worker.FailureMembers) > 0 &&
		dc.Spec.Worker.RecoverFailover &&
		shouldRecoverDM(dc, label.DMWorkerLabelVal, m.deps.PodLister) {
		m.failover.Recover(dc)
	}

	newSts, err := getNewWorkerSetForDMCluster(dc, cm)
	if err != nil {
		return err
	}

	if stsNotExist {
		err = SetStatefulSetLastAppliedConfigAnnotation(newSts)
		if err != nil {
			return err
		}
		err = m.deps.StatefulSetControl.CreateStatefulSet(dc, newSts)
		if err != nil {
			return err
		}
		return nil
	}

	if err := m.scaler.Scale(dc, oldSts, newSts); err != nil {
		return err
	}

	// Perform failover logic if necessary. Note that this will only update
	// DMCluster status. The actual scaling performs in next sync loop (if a
	// new replica needs to be added).
	if m.deps.CLIConfig.AutoFailover && dc.Spec.Worker.MaxFailoverCount != nil {
		if dc.WorkerAllPodsStarted() && !dc.WorkerAllMembersReady() {
			if err := m.failover.Failover(dc); err != nil {
				return err
			}
		}
	}

	return updateStatefulSet(m.deps.StatefulSetControl, dc, newSts, oldSts)
}

func (m *workerMemberManager) syncDMClusterStatus(dc *v1alpha1.DMCluster, set *apps.StatefulSet) error {
	if set == nil {
		// skip if not created yet
		return nil
	}

	dc.Status.Worker.StatefulSet = &set.Status

	upgrading, err := m.workerStatefulSetIsUpgrading(set, dc)
	if err != nil {
		return err
	}
	if upgrading {
		dc.Status.Worker.Phase = v1alpha1.UpgradePhase
	} else if dc.WorkerStsDesiredReplicas() != *set.Spec.Replicas {
		dc.Status.Worker.Phase = v1alpha1.ScalePhase
	} else {
		dc.Status.Worker.Phase = v1alpha1.NormalPhase
	}

	dmClient := controller.GetMasterClient(m.deps.DMMasterControl, dc)

	workersInfo, err := dmClient.GetWorkers()
	if err != nil {
		dc.Status.Master.Synced = false
		return err
	}

	workerStatus := map[string]v1alpha1.WorkerMember{}
	for _, worker := range workersInfo {
		name := worker.Name
		status := v1alpha1.WorkerMember{
			Name:  name,
			Addr:  worker.Addr,
			Stage: worker.Stage,
		}

		oldWorkerMember, exist := dc.Status.Worker.Members[name]

		status.LastTransitionTime = metav1.Now()
		if exist && status.Stage == oldWorkerMember.Stage {
			status.LastTransitionTime = oldWorkerMember.LastTransitionTime
		}

		workerStatus[name] = status

		// offline the workers that already been scaled-in
		if status.Stage == "offline" {
			if !isWorkerPodDesired(dc, name) {
				err := dmClient.DeleteWorker(name)
				if err != nil {
					klog.Errorf("fail to remove worker %s, err: %s", worker.Name, err)
				}
			}
		}
	}

	dc.Status.Worker.Synced = true
	dc.Status.Worker.Members = workerStatus
	dc.Status.Worker.Image = ""
	c := filterContainer(set, "dm-worker")
	if c != nil {
		dc.Status.Worker.Image = c.Image
	}
	return nil
}

func (m *workerMemberManager) workerStatefulSetIsUpgrading(set *apps.StatefulSet, dc *v1alpha1.DMCluster) (bool, error) {
	if statefulSetIsUpgrading(set) {
		return true, nil
	}
	instanceName := dc.GetInstanceName()
	selector, err := label.NewDM().
		Instance(instanceName).
		DMWorker().
		Selector()
	if err != nil {
		return false, err
	}
	workerPods, err := m.deps.PodLister.Pods(dc.GetNamespace()).List(selector)
	if err != nil {
		return false, fmt.Errorf("workerStatefulSetIsUpgrading: failed to list pods for cluster %s/%s, selector %s, error: %v", dc.GetNamespace(), instanceName, selector, err)
	}
	for _, pod := range workerPods {
		revisionHash, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return false, nil
		}
		if revisionHash != dc.Status.Worker.StatefulSet.UpdateRevision {
			return true, nil
		}
	}
	return false, nil
}

// syncWorkerConfigMap syncs the configmap of dm-worker
func (m *workerMemberManager) syncWorkerConfigMap(dc *v1alpha1.DMCluster, set *apps.StatefulSet) (*corev1.ConfigMap, error) {
	if dc.Spec.Worker.Config == nil {
		return nil, nil
	}
	newCm, err := getWorkerConfigMap(dc)
	if err != nil {
		return nil, err
	}
	return m.deps.TypedControl.CreateOrUpdateConfigMap(dc, newCm)
}

func getNewWorkerSetForDMCluster(dc *v1alpha1.DMCluster, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
	ns := dc.Namespace
	dcName := dc.Name
	baseWorkerSpec := dc.BaseWorkerSpec()
	instanceName := dc.GetInstanceName()
	workerConfigMap := cm.Name

	annMount, annVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		annMount,
		{Name: "config", ReadOnly: true, MountPath: "/etc/dm-worker"},
		{Name: "startup-script", ReadOnly: true, MountPath: "/usr/local/bin"},
		{Name: v1alpha1.DMWorkerMemberType.String(), MountPath: dmWorkerDataVolumeMountPath},
	}
	if dc.IsTLSClusterEnabled() {
		volMounts = append(volMounts, corev1.VolumeMount{
			Name: "dm-worker-tls", ReadOnly: true, MountPath: "/var/lib/dm-worker-tls",
		})
	}

	vols := []corev1.Volume{
		annVolume,
		{Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: workerConfigMap,
					},
					Items: []corev1.KeyToPath{{Key: "config-file", Path: "dm-worker.toml"}},
				},
			},
		},
		{Name: "startup-script",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: workerConfigMap,
					},
					Items: []corev1.KeyToPath{{Key: "startup-script", Path: "dm_worker_start_script.sh"}},
				},
			},
		},
	}
	if dc.IsTLSClusterEnabled() {
		vols = append(vols, corev1.Volume{
			Name: "dm-worker-tls", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterTLSSecretName(dc.Name, label.DMWorkerLabelVal),
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
	if dc.Spec.Worker.StorageSize != "" {
		storageSize = dc.Spec.Worker.StorageSize
	}
	rs, err := resource.ParseQuantity(storageSize)
	if err != nil {
		return nil, fmt.Errorf("cannot parse storage request for dm-worker, dmcluster %s/%s, error: %v", dc.Namespace, dc.Name, err)
	}
	storageRequest := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceStorage: rs,
		},
	}

	workerLabel := label.NewDM().Instance(instanceName).DMWorker()
	setName := controller.DMWorkerMemberName(dcName)
	podAnnotations := CombineAnnotations(controller.AnnProm(8262), baseWorkerSpec.Annotations())
	stsAnnotations := getStsAnnotations(dc.Annotations, label.DMWorkerLabelVal)

	workerContainer := corev1.Container{
		Name:            v1alpha1.DMWorkerMemberType.String(),
		Image:           dc.WorkerImage(),
		ImagePullPolicy: baseWorkerSpec.ImagePullPolicy(),
		Command:         []string{"/bin/sh", "/usr/local/bin/dm_worker_start_script.sh"},
		Ports: []corev1.ContainerPort{
			{
				Name:          "client",
				ContainerPort: int32(8262),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: volMounts,
		Resources:    controller.ContainerResource(dc.Spec.Worker.ResourceRequirements),
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
			Name:  "CLUSTER_NAME",
			Value: dcName,
		},
		{
			Name:  "HEADLESS_SERVICE_NAME",
			Value: controller.DMWorkerPeerMemberName(dcName),
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

	podSpec := baseWorkerSpec.BuildPodSpec()
	if baseWorkerSpec.HostNetwork() {
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
	workerContainer.Env = util.AppendEnv(env, baseWorkerSpec.Env())
	podSpec.Volumes = append(vols, baseWorkerSpec.AdditionalVolumes()...)
	podSpec.Containers = append([]corev1.Container{workerContainer}, baseWorkerSpec.AdditionalContainers()...)

	workerSet := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            setName,
			Namespace:       ns,
			Labels:          workerLabel.Labels(),
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetDMOwnerRef(dc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(dc.WorkerStsDesiredReplicas()),
			Selector: workerLabel.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      workerLabel.Labels(),
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: v1alpha1.DMWorkerMemberType.String(),
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						StorageClassName: dc.Spec.Worker.StorageClassName,
						Resources:        storageRequest,
					},
				},
			},
			ServiceName:         controller.DMWorkerPeerMemberName(dcName),
			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: apps.RollingUpdateStatefulSetStrategyType,
			},
		},
	}

	return workerSet, nil
}

func getWorkerConfigMap(dc *v1alpha1.DMCluster) (*corev1.ConfigMap, error) {
	// For backward compatibility, only sync dm configmap when .worker.config is non-nil
	config := dc.Spec.Worker.Config
	if config == nil {
		return nil, nil
	}

	// override CA if tls enabled
	if dc.IsTLSClusterEnabled() {
		config.SSLCA = pointer.StringPtr(path.Join(dmWorkerClusterCertPath, tlsSecretRootCAKey))
		config.SSLCert = pointer.StringPtr(path.Join(dmWorkerClusterCertPath, corev1.TLSCertKey))
		config.SSLKey = pointer.StringPtr(path.Join(dmWorkerClusterCertPath, corev1.TLSPrivateKeyKey))
	}

	confText, err := MarshalTOML(config)
	if err != nil {
		return nil, err
	}
	startScript, err := RenderDMWorkerStartScript(&DMWorkerStartScriptModel{
		DataDir:       filepath.Join(dmWorkerDataVolumeMountPath, dc.Spec.Worker.DataSubDir),
		MasterAddress: controller.DMMasterMemberName(dc.Name) + ":8261",
	})
	if err != nil {
		return nil, err
	}

	instanceName := dc.GetInstanceName()
	workerLabel := label.NewDM().Instance(instanceName).DMWorker().Labels()
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            controller.DMWorkerMemberName(dc.Name),
			Namespace:       dc.Namespace,
			Labels:          workerLabel,
			OwnerReferences: []metav1.OwnerReference{controller.GetDMOwnerRef(dc)},
		},
		Data: map[string]string{
			"config-file":    string(confText),
			"startup-script": startScript,
		},
	}

	if err := AddConfigMapDigestSuffix(cm); err != nil {
		return nil, err
	}
	return cm, nil
}

func isWorkerPodDesired(dc *v1alpha1.DMCluster, podName string) bool {
	ordinals := dc.WorkerStsDesiredOrdinals(false)
	ordinal, err := util.GetOrdinalFromPodName(podName)
	if err != nil {
		klog.Errorf("unexpected pod name %q: %v", podName, err)
		return false
	}
	return ordinals.Has(ordinal)
}
