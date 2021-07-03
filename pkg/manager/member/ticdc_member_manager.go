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
	"strings"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
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
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
)

const (
	ticdcCertPath        = "/var/lib/ticdc-tls"
	ticdcSinkCertPath    = "/var/lib/sink-tls"
	ticdcCertVolumeMount = "ticdc-tls"
)

// ticdcMemberManager implements manager.Manager.
type ticdcMemberManager struct {
	deps                     *controller.Dependencies
	scaler                   Scaler
	ticdcUpgrader            Upgrader
	statefulSetIsUpgradingFn func(corelisters.PodLister, pdapi.PDControlInterface, *apps.StatefulSet, *v1alpha1.TidbCluster) (bool, error)
}

func getTiCDCConfigMap(tc *v1alpha1.TidbCluster) (*corev1.ConfigMap, error) {
	config := tc.Spec.TiCDC.Config
	if config == nil {
		return nil, nil
	}

	confText, err := config.MarshalTOML()
	if err != nil {
		return nil, err
	}

	data := map[string]string{
		"config-file": string(confText),
	}

	name := controller.TiCDCMemberName(tc.Name)
	instanceName := tc.GetInstanceName()
	cdcLabels := label.New().Instance(instanceName).TiCDC().Labels()

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       tc.Namespace,
			Labels:          cdcLabels,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Data: data,
	}

	return cm, nil

}

// NewTiCDCMemberManager returns a *ticdcMemberManager
func NewTiCDCMemberManager(deps *controller.Dependencies, scaler Scaler, ticdcUpgrader Upgrader) manager.Manager {
	m := &ticdcMemberManager{
		deps:          deps,
		scaler:        scaler,
		ticdcUpgrader: ticdcUpgrader,
	}
	m.statefulSetIsUpgradingFn = ticdcStatefulSetIsUpgrading
	return m
}

func (m *ticdcMemberManager) syncTiCDCConfigMap(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) (*corev1.ConfigMap, error) {
	if tc.Spec.TiCDC.Config == nil || tc.Spec.TiCDC.Config.OnlyOldItems() {
		return nil, nil
	}

	newCm, err := getTiCDCConfigMap(tc)
	if err != nil {
		return nil, err
	}

	var inUseName string
	if set != nil {
		inUseName = FindConfigMapVolume(&set.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, controller.TiCDCMemberName(tc.Name))
		})
	}

	klog.V(3).Info("get ticdc in use config map name: ", inUseName)

	err = updateConfigMapIfNeed(m.deps.ConfigMapLister, tc.BaseTiCDCSpec().ConfigUpdateStrategy(), inUseName, newCm)
	if err != nil {
		return nil, err
	}
	return m.deps.TypedControl.CreateOrUpdateConfigMap(tc, newCm)
}

// Sync fulfills the manager.Manager interface
func (m *ticdcMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if tc.Spec.TiCDC == nil {
		return nil
	}
	if tc.Spec.Paused {
		klog.Infof("TidbCluster %s/%s is paused, skip syncing ticdc deployment", ns, tcName)
		return nil
	}

	// Sync CDC Headless Service
	if err := m.syncCDCHeadlessService(tc); err != nil {
		return err
	}

	return m.syncStatefulSet(tc)
}

func (m *ticdcMemberManager) syncStatefulSet(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	oldStsTmp, err := m.deps.StatefulSetLister.StatefulSets(ns).Get(controller.TiCDCMemberName(tcName))
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncStatefulSet: failed to get sts %s for cluster %s/%s, error: %s", controller.TiCDCMemberName(tcName), ns, tcName, err)
	}

	stsNotExist := errors.IsNotFound(err)
	oldSts := oldStsTmp.DeepCopy()

	// failed to sync ticdc status will not affect subsequent logic, just print the errors.
	if err := m.syncTiCDCStatus(tc, oldSts); err != nil {
		klog.Errorf("failed to sync TidbCluster: [%s/%s]'s ticdc status, error: %v",
			ns, tcName, err)
	}

	cm, err := m.syncTiCDCConfigMap(tc, oldSts)
	if err != nil {
		return err
	}

	newSts, err := getNewTiCDCStatefulSet(tc, cm)
	if err != nil {
		return err
	}

	if stsNotExist {
		if !tc.PDIsAvailable() {
			klog.Infof("TidbCluster: %s/%s, waiting for PD cluster running", ns, tcName)
			return nil
		}
		err = SetStatefulSetLastAppliedConfigAnnotation(newSts)
		if err != nil {
			return err
		}
		err = m.deps.StatefulSetControl.CreateStatefulSet(tc, newSts)
		if err != nil {
			return err
		}
		return nil
	}

	// Scaling takes precedence over upgrading because:
	// - if a pod fails in the upgrading, users may want to delete it or add
	//   new replicas
	// - it's ok to scale in the middle of upgrading (in statefulset controller
	//   scaling takes precedence over upgrading too)
	if err := m.scaler.Scale(tc, oldSts, newSts); err != nil {
		return err
	}

	if !templateEqual(newSts, oldSts) || tc.Status.TiCDC.Phase == v1alpha1.UpgradePhase {
		if err := m.ticdcUpgrader.Upgrade(tc, oldSts, newSts); err != nil {
			return err
		}
	}

	return UpdateStatefulSet(m.deps.StatefulSetControl, tc, newSts, oldSts)
}

func (m *ticdcMemberManager) syncTiCDCStatus(tc *v1alpha1.TidbCluster, sts *apps.StatefulSet) error {
	if sts == nil {
		// skip if not created yet
		return nil
	}

	ns := tc.GetNamespace()
	tcName := tc.GetName()

	tc.Status.TiCDC.StatefulSet = &sts.Status
	upgrading, err := m.statefulSetIsUpgradingFn(m.deps.PodLister, m.deps.PDControl, sts, tc)
	if err != nil {
		return err
	}
	if upgrading {
		tc.Status.TiCDC.Phase = v1alpha1.UpgradePhase
	} else {
		tc.Status.TiCDC.Phase = v1alpha1.NormalPhase
	}

	ticdcCaptures := map[string]v1alpha1.TiCDCCapture{}
	for id := range helper.GetPodOrdinals(tc.Status.TiCDC.StatefulSet.Replicas, sts) {
		podName := fmt.Sprintf("%s-%d", controller.TiCDCMemberName(tc.GetName()), id)
		capture, err := m.deps.CDCControl.GetStatus(tc, int32(id))
		if err != nil {
			klog.Warningf("Failed to get status for Pod %s of [%s/%s], error: %v", podName, ns, tcName, err)
		} else {
			ticdcCaptures[podName] = v1alpha1.TiCDCCapture{
				PodName: podName,
				ID:      capture.ID,
				Version: capture.Version,
				IsOwner: capture.IsOwner,
			}
		}
	}
	if len(ticdcCaptures) == int(tc.TiCDCDeployDesiredReplicas()) {
		tc.Status.TiCDC.Synced = true
	}
	tc.Status.TiCDC.Captures = ticdcCaptures

	return nil
}

func (m *ticdcMemberManager) syncCDCHeadlessService(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := getNewCDCHeadlessService(tc)
	oldSvcTmp, err := m.deps.ServiceLister.Services(ns).Get(controller.TiCDCPeerMemberName(tcName))
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return m.deps.ServiceControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return fmt.Errorf("syncCDCHeadlessService: failed to get svc %s for cluster %s/%s, error: %s", controller.TiCDCPeerMemberName(tcName), ns, tcName, err)
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
		_, err = m.deps.ServiceControl.UpdateService(tc, &svc)
		return err
	}

	return nil
}

func getNewCDCHeadlessService(tc *v1alpha1.TidbCluster) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	instanceName := tc.GetInstanceName()
	svcName := controller.TiCDCPeerMemberName(tcName)
	svcLabel := label.New().Instance(instanceName).TiCDC().Labels()

	svc := corev1.Service{
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
					Name:       "ticdc",
					Port:       8301,
					TargetPort: intstr.FromInt(int(8301)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector:                 svcLabel,
			PublishNotReadyAddresses: true,
		},
	}
	return &svc
}

// Only Use config file if cm is not nil
func getNewTiCDCStatefulSet(tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap) (*apps.StatefulSet, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	baseTiCDCSpec := tc.BaseTiCDCSpec()
	stsLabels := labelTiCDC(tc)
	stsName := controller.TiCDCMemberName(tcName)
	podLabels := util.CombineStringMap(stsLabels, baseTiCDCSpec.Labels())
	podAnnotations := util.CombineStringMap(controller.AnnProm(8301), baseTiCDCSpec.Annotations())
	stsAnnotations := getStsAnnotations(tc.Annotations, label.TiCDCLabelVal)
	headlessSvcName := controller.TiCDCPeerMemberName(tcName)

	cmdArgs := []string{"/cdc server", "--addr=0.0.0.0:8301", fmt.Sprintf("--advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc%s:8301", controller.FormatClusterDomain(tc.Spec.ClusterDomain))}
	cmdArgs = append(cmdArgs, fmt.Sprintf("--gc-ttl=%d", tc.TiCDCGCTTL()))
	cmdArgs = append(cmdArgs, fmt.Sprintf("--log-file=%s", tc.TiCDCLogFile()))
	cmdArgs = append(cmdArgs, fmt.Sprintf("--log-level=%s", tc.TiCDCLogLevel()))

	var (
		volMounts []corev1.VolumeMount
		vols      []corev1.Volume
	)

	if tc.IsTLSClusterEnabled() {
		cmdArgs = append(cmdArgs, fmt.Sprintf("--ca=%s", path.Join(ticdcCertPath, corev1.ServiceAccountRootCAKey)))
		cmdArgs = append(cmdArgs, fmt.Sprintf("--cert=%s", path.Join(ticdcCertPath, corev1.TLSCertKey)))
		cmdArgs = append(cmdArgs, fmt.Sprintf("--key=%s", path.Join(ticdcCertPath, corev1.TLSPrivateKeyKey)))
		if tc.Spec.ClusterDomain == "" {
			cmdArgs = append(cmdArgs, fmt.Sprintf("--pd=https://%s-pd:2379", tcName))
		} else {
			cmdArgs = append(cmdArgs, "--pd=${result}")
		}

		volMounts = append(volMounts, corev1.VolumeMount{
			Name:      ticdcCertVolumeMount,
			ReadOnly:  true,
			MountPath: ticdcCertPath,
		}, corev1.VolumeMount{
			Name:      util.ClusterClientVolName,
			ReadOnly:  true,
			MountPath: util.ClusterClientTLSPath,
		})

		vols = append(vols, corev1.Volume{
			Name: ticdcCertVolumeMount, VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterTLSSecretName(tc.Name, label.TiCDCLabelVal),
				},
			},
		}, corev1.Volume{
			Name: util.ClusterClientVolName, VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterClientTLSSecretName(tc.Name),
				},
			},
		})
	} else {
		if tc.Spec.ClusterDomain == "" {
			cmdArgs = append(cmdArgs, fmt.Sprintf("--pd=http://%s-pd:2379", tcName))
		} else {
			cmdArgs = append(cmdArgs, "--pd=${result}")
		}
	}

	if cm != nil {
		cmdArgs = append(cmdArgs, fmt.Sprintf("--config=%s", "/etc/ticdc/ticdc.toml"))
	}

	// handle StorageVolumes and AdditionalVolumeMounts in ComponentSpec
	storageVolMounts, additionalPVCs := util.BuildStorageVolumeAndVolumeMount(tc.Spec.TiCDC.StorageVolumes, tc.Spec.TiCDC.StorageClassName, v1alpha1.TiCDCMemberType)
	volMounts = append(volMounts, storageVolMounts...)
	volMounts = append(volMounts, tc.Spec.TiCDC.AdditionalVolumeMounts...)

	var script string

	if tc.Spec.ClusterDomain != "" {
		var pdAddr string
		if tc.IsTLSClusterEnabled() {
			pdAddr = fmt.Sprintf("https://%s-pd:2379", tcName)
		} else {
			pdAddr = fmt.Sprintf("http://%s-pd:2379", tcName)
		}
		formatClusterDomain := controller.FormatClusterDomain(tc.Spec.ClusterDomain)

		str := `set -uo pipefail
pd_url="%s"
encoded_domain_url=$(echo $pd_url | base64 | tr "\n" " " | sed "s/ //g")
discovery_url="%s-discovery.${NAMESPACE}.svc%s:10261"
until result=$(wget -qO- -T 3 http://${discovery_url}/verify/${encoded_domain_url} 2>/dev/null); do
echo "waiting for the verification of PD endpoints ..."
sleep 2
done
`

		script += fmt.Sprintf(str, pdAddr, tc.GetName(), formatClusterDomain)
		script += "\n" + strings.Join(append([]string{"exec"}, cmdArgs...), " ")
	} else {
		script = strings.Join(cmdArgs, " ")
	}

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
		Command:         []string{"/bin/sh", "-c", script},
		Ports: []corev1.ContainerPort{
			{
				Name:          "ticdc",
				ContainerPort: int32(8301),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: volMounts,
		Resources:    controller.ContainerResource(tc.Spec.TiCDC.ResourceRequirements),
		Env:          util.AppendEnv(envs, baseTiCDCSpec.Env()),
	}
	if cm != nil {
		ticdcContainer.VolumeMounts = append(ticdcContainer.VolumeMounts, corev1.VolumeMount{
			Name: "config", ReadOnly: true, MountPath: "/etc/ticdc",
		})
	}

	for _, tlsClientSecretName := range tc.Spec.TiCDC.TLSClientSecretNames {
		ticdcContainer.VolumeMounts = append(ticdcContainer.VolumeMounts, corev1.VolumeMount{
			Name: tlsClientSecretName, ReadOnly: true, MountPath: fmt.Sprintf("%s/%s", ticdcSinkCertPath, tlsClientSecretName),
		})
	}

	podSpec := baseTiCDCSpec.BuildPodSpec()
	podSpec.Containers = []corev1.Container{ticdcContainer}
	podSpec.Volumes = append(vols, baseTiCDCSpec.AdditionalVolumes()...)
	podSpec.ServiceAccountName = tc.Spec.TiCDC.ServiceAccount
	podSpec.InitContainers = append(podSpec.InitContainers, baseTiCDCSpec.InitContainers()...)
	if podSpec.ServiceAccountName == "" {
		podSpec.ServiceAccountName = tc.Spec.ServiceAccount
	}

	for _, tlsClientSecretName := range tc.Spec.TiCDC.TLSClientSecretNames {
		podSpec.Volumes = append(podSpec.Volumes, corev1.Volume{
			Name: tlsClientSecretName, VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: tlsClientSecretName,
				},
			},
		})
	}

	if cm != nil {
		podSpec.Volumes = append(podSpec.Volumes, corev1.Volume{
			Name: "config", VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cm.Name,
					},
					Items: []corev1.KeyToPath{{Key: "config-file", Path: "ticdc.toml"}},
				}},
		})
	}

	updateStrategy := apps.StatefulSetUpdateStrategy{}
	if baseTiCDCSpec.StatefulSetUpdateStrategy() == apps.OnDeleteStatefulSetStrategyType {
		updateStrategy.Type = apps.OnDeleteStatefulSetStrategyType
	} else {
		updateStrategy.Type = apps.RollingUpdateStatefulSetStrategyType
		updateStrategy.RollingUpdate = &apps.RollingUpdateStatefulSetStrategy{
			Partition: pointer.Int32Ptr(tc.TiCDCDeployDesiredReplicas()),
		}
	}

	ticdcSts := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            stsName,
			Namespace:       ns,
			Labels:          stsLabels.Labels(),
			Annotations:     stsAnnotations,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(tc.TiCDCDeployDesiredReplicas()),
			Selector: stsLabels.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: podAnnotations,
				},
				Spec: podSpec,
			},
			ServiceName:         headlessSvcName,
			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy:      updateStrategy,
		},
	}
	ticdcSts.Spec.VolumeClaimTemplates = append(ticdcSts.Spec.VolumeClaimTemplates, additionalPVCs...)
	return ticdcSts, nil
}

func labelTiCDC(tc *v1alpha1.TidbCluster) label.Label {
	instanceName := tc.GetInstanceName()
	return label.New().Instance(instanceName).TiCDC()
}

func ticdcStatefulSetIsUpgrading(podLister corelisters.PodLister, pdControl pdapi.PDControlInterface, set *apps.StatefulSet, tc *v1alpha1.TidbCluster) (bool, error) {
	if statefulSetIsUpgrading(set) {
		return true, nil
	}
	instanceName := tc.GetInstanceName()
	selector, err := label.New().Instance(instanceName).TiCDC().Selector()
	if err != nil {
		return false, err
	}
	ticdcPods, err := podLister.Pods(tc.GetNamespace()).List(selector)
	if err != nil {
		return false, fmt.Errorf("ticdcStatefulSetIsUpgrading: failed to list pods for cluster %s/%s, selector %s, error: %s", tc.GetNamespace(), tc.GetName(), selector, err)
	}
	for _, pod := range ticdcPods {
		revisionHash, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return false, nil
		}
		if revisionHash != tc.Status.TiCDC.StatefulSet.UpdateRevision {
			return true, nil
		}
	}

	return false, nil
}

type FakeTiCDCMemberManager struct {
	err error
}

func NewFakeTiCDCMemberManager() *FakeTiCDCMemberManager {
	return &FakeTiCDCMemberManager{}
}

func (m *FakeTiCDCMemberManager) SetSyncError(err error) {
	m.err = err
}

func (m *FakeTiCDCMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	if m.err != nil {
		return m.err
	}
	return nil
}
