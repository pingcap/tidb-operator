// Copyright 2019 PingCAP, Inc.
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
	"context"
	"crypto/tls"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/util/config"
	"github.com/pingcap/tidb-operator/pkg/binlog"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/manager/member/startscript"
	"github.com/pingcap/tidb-operator/pkg/manager/suspender"
	mngerutils "github.com/pingcap/tidb-operator/pkg/manager/utils"
	"github.com/pingcap/tidb-operator/pkg/manager/volumes"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/pkg/util"

	apps "k8s.io/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
)

const (
	pumpCertVolumeMount = "pump-tls"
	pumpCertPath        = "/var/lib/pump-tls"
)

type binlogClient interface {
	PumpNodeStatus(ctx context.Context) (status []*v1alpha1.PumpNodeStatus, err error)
	Close() error
}

type pumpMemberManager struct {
	deps   *controller.Dependencies
	scaler Scaler
	// only use for test
	binlogClient      binlogClient
	suspender         suspender.Suspender
	podVolumeModifier volumes.PodVolumeModifier
}

// NewPumpMemberManager returns a controller to reconcile pump clusters
func NewPumpMemberManager(deps *controller.Dependencies, scaler Scaler, spder suspender.Suspender, pvm volumes.PodVolumeModifier) manager.Manager {
	return &pumpMemberManager{
		deps:              deps,
		scaler:            scaler,
		suspender:         spder,
		podVolumeModifier: pvm,
	}
}

func (m *pumpMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	if tc.Spec.Pump == nil {
		return nil
	}

	// skip sync if pump is suspended
	component := v1alpha1.PumpMemberType
	needSuspend, err := m.suspender.SuspendComponent(tc, component)
	if err != nil {
		return fmt.Errorf("suspend %s failed: %v", component, err)
	}
	if needSuspend {
		klog.Infof("component %s for cluster %s/%s is suspended, skip syncing", component, tc.GetNamespace(), tc.GetName())
		return nil
	}

	if err := m.syncHeadlessService(tc); err != nil {
		return err
	}
	return m.syncPumpStatefulSetForTidbCluster(tc)
}

// syncPumpStatefulSetForTidbCluster sync statefulset status of pump to tidbcluster
func (m *pumpMemberManager) syncPumpStatefulSetForTidbCluster(tc *v1alpha1.TidbCluster) error {
	oldPumpSetTemp, err := m.deps.StatefulSetLister.StatefulSets(tc.Namespace).Get(controller.PumpMemberName(tc.Name))
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncPumpStatefulSetForTidbCluster: failed to get sts %s for cluster %s/%s, error: %s", controller.PumpMemberName(tc.Name), tc.GetNamespace(), tc.GetName(), err)
	}
	notFound := errors.IsNotFound(err)
	oldSet := oldPumpSetTemp.DeepCopy()

	if err := m.syncTiDBClusterStatus(tc, oldSet); err != nil {
		klog.Errorf("failed to sync TidbCluster: [%s/%s]'s status, error: %v", tc.Namespace, tc.Name, err)
		return err
	}

	if tc.Spec.Paused {
		klog.V(4).Infof("tikv cluster %s/%s is paused, skip syncing for pump statefulset", tc.GetNamespace(), tc.GetName())
		return nil
	}

	cm, err := m.syncConfigMap(tc, oldSet)
	if err != nil {
		return err
	}

	newSet, err := getNewPumpStatefulSet(tc, cm)
	if err != nil {
		return err
	}
	if notFound {
		err = mngerutils.SetStatefulSetLastAppliedConfigAnnotation(newSet)
		if err != nil {
			return err
		}
		return m.deps.StatefulSetControl.CreateStatefulSet(tc, newSet)
	}

	if err := m.scaler.Scale(tc, oldSet, newSet); err != nil {
		return err
	}

	// Wait for PD & TiKV upgrading done
	// NO check for v1alpha1.ScalePhase now, as it shouldn't block when some other components scaling to 0 and deleting Pump
	if tc.Status.TiFlash.Phase == v1alpha1.UpgradePhase ||
		tc.Status.PD.Phase == v1alpha1.UpgradePhase ||
		tc.Status.TiKV.Phase == v1alpha1.UpgradePhase {
		klog.Infof("TidbCluster: [%s/%s]'s tiflash status is %s, "+
			"pd status is %s, tikv status is %s, can not upgrade pump",
			tc.Namespace, tc.Name,
			tc.Status.TiFlash.Phase, tc.Status.PD.Phase, tc.Status.TiKV.Phase)
		return nil
	}

	return mngerutils.UpdateStatefulSetWithPrecheck(m.deps, tc, "FailedUpdatePumpSTS", newSet, oldSet)
}

func (p *pumpMemberManager) buildBinlogClient(tc *v1alpha1.TidbCluster, control pdapi.PDControlInterface) (client binlogClient, err error) {
	if p.binlogClient != nil {
		return p.binlogClient, nil
	}

	return buildBinlogClient(tc, control)
}

func buildBinlogClient(tc *v1alpha1.TidbCluster, control pdapi.PDControlInterface) (client *binlog.Client, err error) {
	var endpoints []string
	var tlsConfig *tls.Config
	if tc.Heterogeneous() && tc.WithoutLocalPD() {
		// connect to pd of other cluster and use own cert
		endpoints, tlsConfig, err = control.GetEndpoints(pdapi.Namespace(tc.Spec.Cluster.Namespace), tc.Spec.Cluster.Name, tc.IsTLSClusterEnabled(),
			pdapi.TLSCertFromTC(pdapi.Namespace(tc.Namespace), tc.Name),
			pdapi.ClusterRef(tc.Spec.Cluster.ClusterDomain),
		)
	} else {
		endpoints, tlsConfig, err = control.GetEndpoints(pdapi.Namespace(tc.Namespace), tc.Name, tc.IsTLSClusterEnabled())
	}
	if err != nil {
		return nil, err
	}

	// support x-k8s tidbcluster without local pd
	for _, pdMember := range tc.Status.PD.PeerMembers {
		endpoints = append(endpoints, pdMember.ClientURL)
	}

	client, err = binlog.NewBinlogClient(endpoints, tlsConfig, 5*time.Second)
	if err != nil {
		return nil, err
	}

	return
}

func (m *pumpMemberManager) syncTiDBClusterStatus(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) error {
	if set == nil {
		// skip if not created yet
		return nil
	}

	tc.Status.Pump.StatefulSet = &set.Status

	upgrading, err := m.pumpStatefulSetIsUpgrading(set, tc)
	if err != nil {
		return err
	}

	if upgrading {
		tc.Status.Pump.Phase = v1alpha1.UpgradePhase
	} else {
		tc.Status.Pump.Phase = v1alpha1.NormalPhase
	}

	client, err := m.buildBinlogClient(tc, m.deps.PDControl)
	if err != nil {
		return err
	}
	defer client.Close()

	status, err := client.PumpNodeStatus(context.TODO())
	if err != nil {
		return err
	}

	tc.Status.Pump.Members = status

	err = volumes.SyncVolumeStatus(m.podVolumeModifier, m.deps.PodLister, tc, v1alpha1.PumpMemberType)
	if err != nil {
		return fmt.Errorf("failed to sync volume status for pump: %v", err)
	}

	return nil
}

func (m *pumpMemberManager) syncHeadlessService(tc *v1alpha1.TidbCluster) error {
	if tc.Spec.Paused {
		klog.V(4).Infof("tikv cluster %s/%s is paused, skip syncing for pump headless service", tc.GetNamespace(), tc.GetName())
		return nil
	}

	newSvc := getNewPumpHeadlessService(tc)
	oldSvc, err := m.deps.ServiceLister.Services(newSvc.Namespace).Get(newSvc.Name)
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return m.deps.ServiceControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return fmt.Errorf("syncHeadlessService: failed to get svc %s/%s for cluster %s/%s, error %s", newSvc.Namespace, newSvc.Name, tc.GetNamespace(), tc.GetName(), err)
	}

	equal, err := controller.ServiceEqual(newSvc, oldSvc)
	if err != nil {
		return err
	}
	isOrphan := metav1.GetControllerOf(oldSvc) == nil

	if !equal || isOrphan {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		err = controller.SetServiceLastAppliedConfigAnnotation(&svc)
		if err != nil {
			return err
		}
		// Adopt headless-service created by helm
		if isOrphan {
			svc.OwnerReferences = newSvc.OwnerReferences
			svc.Labels = newSvc.Labels
		}
		_, err = m.deps.ServiceControl.UpdateService(tc, &svc)
		return err
	}
	return nil
}

func (m *pumpMemberManager) syncConfigMap(tc *v1alpha1.TidbCluster, set *appsv1.StatefulSet) (*corev1.ConfigMap, error) {
	basePumpSpec := tc.BasePumpSpec()

	newCm, err := getNewPumpConfigMap(tc)
	if err != nil {
		return nil, err
	}

	var inUseName string
	if set != nil {
		inUseName = mngerutils.FindConfigMapVolume(&set.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, controller.PumpMemberName(tc.Name))
		})
	}

	err = mngerutils.UpdateConfigMapIfNeed(m.deps.ConfigMapLister, basePumpSpec.ConfigUpdateStrategy(), inUseName, newCm)
	if err != nil {
		return nil, err
	}
	return m.deps.TypedControl.CreateOrUpdateConfigMap(tc, newCm)
}

func getNewPumpHeadlessService(tc *v1alpha1.TidbCluster) *corev1.Service {
	if tc.Spec.Pump == nil {
		return nil
	}

	objMeta, pumpLabel := getPumpMeta(tc, controller.PumpPeerMemberName)

	return &corev1.Service{
		ObjectMeta: objMeta,
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:       "pump",
					Port:       8250,
					TargetPort: intstr.FromInt(8250),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector:                 pumpLabel,
			PublishNotReadyAddresses: true,
		},
	}
}

// getNewPumpConfigMap returns a configMap for pump
func getNewPumpConfigMap(tc *v1alpha1.TidbCluster) (*corev1.ConfigMap, error) {
	spec := tc.Spec.Pump
	objMeta, _ := getPumpMeta(tc, controller.PumpMemberName)

	var cfg *config.GenericConfig
	if spec.Config != nil {
		cfg = spec.Config.DeepCopy()
	}

	if tc.IsTLSClusterEnabled() {
		if cfg == nil {
			cfg = config.New(map[string]interface{}{})
		}

		cfg.Set("security.ssl-ca", path.Join(pumpCertPath, corev1.ServiceAccountRootCAKey))
		cfg.Set("security.ssl-cert", path.Join(pumpCertPath, corev1.TLSCertKey))
		cfg.Set("security.ssl-key", path.Join(pumpCertPath, corev1.TLSPrivateKeyKey))
	}

	confText, err := cfg.MarshalTOML()
	if err != nil {
		return nil, err
	}

	name := controller.PumpMemberName(tc.Name)
	confTextStr := string(confText)

	data := map[string]string{
		"pump-config": confTextStr,
	}

	objMeta.Name = name

	return &corev1.ConfigMap{
		ObjectMeta: objMeta,
		Data:       data,
	}, nil
}

func getNewPumpStatefulSet(tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap) (*appsv1.StatefulSet, error) {
	spec := tc.BasePumpSpec()
	objMeta, stsLabels := getPumpMeta(tc, controller.PumpMemberName)
	replicas := tc.Spec.Pump.Replicas
	storageClass := tc.Spec.Pump.StorageClassName
	podLabels := util.CombineStringMap(stsLabels.Labels(), spec.Labels())
	podAnnos := util.CombineStringMap(controller.AnnProm(8250), spec.Annotations())
	storageRequest, err := controller.ParseStorageRequest(tc.Spec.Pump.Requests)
	if err != nil {
		return nil, fmt.Errorf("cannot parse storage request for pump, tidbcluster %s/%s, error: %v", tc.Namespace, tc.Name, err)
	}
	startScript, err := startscript.RenderPumpStartScript(tc)
	if err != nil {
		return nil, fmt.Errorf("render start-script for tc %s/%s failed: %v", tc.Namespace, tc.Name, err)
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

	dataVolumeName := string(v1alpha1.GetStorageVolumeName("", v1alpha1.PumpMemberType))
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      dataVolumeName,
			MountPath: "/data",
		},
		{
			Name:      "config",
			MountPath: "/etc/pump",
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
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name: pumpCertVolumeMount, ReadOnly: true, MountPath: pumpCertPath,
		})
		volumes = append(volumes, corev1.Volume{
			Name: pumpCertVolumeMount, VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterTLSSecretName(tc.Name, label.PumpLabelVal),
				},
			},
		})
	}
	// For compatibility, add the volume mount for anno when startscript is not v1
	if tc.StartScriptVersion() != v1alpha1.StartScriptV1 {
		annMount, annVolume := annotationsMountVolume()
		volumeMounts = append(volumeMounts, annMount)
		volumes = append(volumes, annVolume)
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
			EnvFrom:      spec.EnvFrom(),
			VolumeMounts: volumeMounts,
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.FromInt(8250),
					},
				},
			},
		},
	}

	volumeClaims := []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: dataVolumeName,
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

	// TODO: set serviceAccountName in BuildPodSpec
	serviceAccountName := tc.Spec.Pump.ServiceAccount
	if serviceAccountName == "" {
		serviceAccountName = tc.Spec.ServiceAccount
	}

	podSpec := spec.BuildPodSpec()
	podSpec.Containers, err = MergePatchContainers(containers, tc.Spec.Pump.AdditionalContainers)
	if err != nil {
		return nil, fmt.Errorf("failed to merge containers spec for Pump of [%s/%s], error: %v", objMeta.Namespace, objMeta.Name, err)
	}

	podSpec.Volumes = volumes
	podSpec.ServiceAccountName = serviceAccountName
	// TODO: change to set field in BuildPodSpec
	podSpec.InitContainers = spec.InitContainers()
	// TODO: change to set field in BuildPodSpec
	podSpec.DNSPolicy = spec.DnsPolicy()

	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: podAnnos,
			Labels:      podLabels,
		},
		Spec: podSpec,
	}

	// To compatible with default podManagementPolicy of pump is "OrderedReady"
	podManagementPolicy := apps.OrderedReadyPodManagement
	if len(tc.Spec.Pump.PodManagementPolicy) != 0 || len(tc.Spec.PodManagementPolicy) != 0 {
		podManagementPolicy = spec.PodManagementPolicy()
	}

	return &appsv1.StatefulSet{
		ObjectMeta: objMeta,
		Spec: appsv1.StatefulSetSpec{
			Selector:    stsLabels.LabelSelector(),
			ServiceName: controller.PumpMemberName(tc.Name),
			Replicas:    &replicas,

			Template:             podTemplate,
			VolumeClaimTemplates: volumeClaims,
			PodManagementPolicy:  podManagementPolicy,
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: spec.StatefulSetUpdateStrategy(),
			},
		},
	}, nil
}

func getPumpMeta(tc *v1alpha1.TidbCluster, nameFunc func(string) string) (metav1.ObjectMeta, label.Label) {
	instanceName := tc.GetInstanceName()
	pumpLabel := label.New().Instance(instanceName).Pump()

	objMeta := metav1.ObjectMeta{
		Name:            nameFunc(tc.Name),
		Namespace:       tc.Namespace,
		Labels:          pumpLabel,
		OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
	}
	return objMeta, pumpLabel
}

func (m *pumpMemberManager) pumpStatefulSetIsUpgrading(set *apps.StatefulSet, tc *v1alpha1.TidbCluster) (bool, error) {
	if mngerutils.StatefulSetIsUpgrading(set) {
		return true, nil
	}
	selector, err := label.New().
		Instance(tc.GetInstanceName()).
		Pump().
		Selector()
	if err != nil {
		return false, err
	}
	pumpPods, err := m.deps.PodLister.Pods(tc.GetNamespace()).List(selector)
	if err != nil {
		return false, fmt.Errorf("pumpStatefulSetIsUpgrading: failed to list pods for cluster %s/%s, selector %s, error: %s", tc.GetNamespace(), tc.GetName(), selector, err)
	}
	for _, pod := range pumpPods {
		revisionHash, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return false, nil
		}
		if revisionHash != tc.Status.Pump.StatefulSet.UpdateRevision {
			return true, nil
		}
	}
	return false, nil
}

type FakePumpMemberManager struct {
	err error
}

func NewFakePumpMemberManager() *FakePumpMemberManager {
	return &FakePumpMemberManager{}
}

func (m *FakePumpMemberManager) SetSyncError(err error) {
	m.err = err
}

func (m *FakePumpMemberManager) Sync(*v1alpha1.TidbCluster) error {
	if m.err != nil {
		return m.err
	}
	return nil
}
