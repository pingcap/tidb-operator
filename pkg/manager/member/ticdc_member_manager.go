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
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
)

const (
	ticdcCertPath        = "/var/lib/ticdc-tls"
	ticdcCertVolumeMount = "ticdc-tls"
)

// ticdcMemberManager implements manager.Manager.
type ticdcMemberManager struct {
	pdControl                     pdapi.PDControlInterface
	cdcControl                    controller.TiCDCControlInterface
	typedControl                  controller.TypedControlInterface
	stsLister                     appslisters.StatefulSetLister
	svcLister                     corelisters.ServiceLister
	podLister                     corelisters.PodLister
	cmLister                      corelisters.ConfigMapLister
	svcControl                    controller.ServiceControlInterface
	stsControl                    controller.StatefulSetControlInterface
	ticdcStatefulSetIsUpgradingFn func(corelisters.PodLister, pdapi.PDControlInterface, *apps.StatefulSet, *v1alpha1.TidbCluster) (bool, error)
}

// NewTiCDCMemberManager returns a *ticdcMemberManager
func NewTiCDCMemberManager(
	pdControl pdapi.PDControlInterface,
	cdcControl controller.TiCDCControlInterface,
	typedControl controller.TypedControlInterface,
	stsLister appslisters.StatefulSetLister,
	svcLister corelisters.ServiceLister,
	podLister corelisters.PodLister,
	cmLister corelisters.ConfigMapLister,
	svcControl controller.ServiceControlInterface,
	stsControl controller.StatefulSetControlInterface) manager.Manager {
	tcmm := &ticdcMemberManager{
		pdControl:    pdControl,
		cdcControl:   cdcControl,
		typedControl: typedControl,
		stsLister:    stsLister,
		svcLister:    svcLister,
		podLister:    podLister,
		cmLister:     cmLister,
		svcControl:   svcControl,
		stsControl:   stsControl,
	}
	tcmm.ticdcStatefulSetIsUpgradingFn = ticdcStatefulSetIsUpgrading
	return tcmm
}

// Sync fulfills the manager.Manager interface
func (tcmm *ticdcMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	if tc.Spec.TiCDC == nil {
		return nil
	}
	if tc.Spec.Paused {
		klog.Infof("TidbCluster %s/%s is paused, skip syncing ticdc deployment", ns, tcName)
		return nil
	}
	if !tc.PDIsAvailable() {
		return controller.RequeueErrorf("TidbCluster: %s/%s, waiting for PD cluster running", ns, tcName)
	}

	// Sync CDC Headless Service
	if err := tcmm.syncCDCHeadlessService(tc); err != nil {
		return err
	}

	return tcmm.syncStatefulSet(tc)
}

func (tcmm *ticdcMemberManager) syncStatefulSet(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	oldStsTmp, err := tcmm.stsLister.StatefulSets(ns).Get(controller.TiCDCMemberName(tcName))
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("syncStatefulSet: failed to get sts %s for cluster %s/%s, error: %s", controller.TiCDCMemberName(tcName), ns, tcName, err)
	}

	stsNotExist := errors.IsNotFound(err)
	oldSts := oldStsTmp.DeepCopy()

	// failed to sync ticdc status will not affect subsequent logic, just print the errors.
	if err := tcmm.syncTiCDCStatus(tc, oldSts); err != nil {
		klog.Errorf("failed to sync TidbCluster: [%s/%s]'s ticdc status, error: %v",
			ns, tcName, err)
	}

	newSts, err := getNewTiCDCStatefulSet(tc)
	if err != nil {
		return err
	}

	if stsNotExist {
		err = SetStatefulSetLastAppliedConfigAnnotation(newSts)
		if err != nil {
			return err
		}
		err = tcmm.stsControl.CreateStatefulSet(tc, newSts)
		if err != nil {
			return err
		}
		return nil
	}

	if tc.PDUpgrading() || tc.TiKVUpgrading() {
		klog.Warningf("pd or tikv is upgrading, skipping upgrade ticdc")
		return nil
	}

	return updateStatefulSet(tcmm.stsControl, tc, newSts, oldSts)
}

func (tcmm *ticdcMemberManager) syncTiCDCStatus(tc *v1alpha1.TidbCluster, sts *apps.StatefulSet) error {
	if sts == nil {
		// skip if not created yet
		return nil
	}

	tc.Status.TiCDC.StatefulSet = &sts.Status
	upgrading, err := tcmm.ticdcStatefulSetIsUpgradingFn(tcmm.podLister, tcmm.pdControl, sts, tc)
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
		capture, err := tcmm.cdcControl.GetStatus(tc, int32(id))
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

	return nil
}

func (tcmm *ticdcMemberManager) syncCDCHeadlessService(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := getNewCDCHeadlessService(tc)
	oldSvcTmp, err := tcmm.svcLister.Services(ns).Get(controller.TiCDCPeerMemberName(tcName))
	if errors.IsNotFound(err) {
		err = controller.SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return tcmm.svcControl.CreateService(tc, newSvc)
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
		_, err = tcmm.svcControl.UpdateService(tc, &svc)
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

func getNewTiCDCStatefulSet(tc *v1alpha1.TidbCluster) (*apps.StatefulSet, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	baseTiCDCSpec := tc.BaseTiCDCSpec()
	ticdcLabel := labelTiCDC(tc)
	stsName := controller.TiCDCMemberName(tcName)
	podAnnotations := CombineAnnotations(controller.AnnProm(8301), baseTiCDCSpec.Annotations())
	stsAnnotations := getStsAnnotations(tc.Annotations, label.TiCDCLabelVal)
	headlessSvcName := controller.TiCDCPeerMemberName(tcName)

	cmdArgs := []string{"/cdc server", "--addr=0.0.0.0:8301", "--advertise-addr=${POD_NAME}.${HEADLESS_SERVICE_NAME}.${NAMESPACE}.svc:8301"}
	// Better do not specified the value if user not specified explicitly.
	// Also no config file for cdc, so other item will be just ignored.
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
				Type: apps.RollingUpdateStatefulSetStrategyType,
			},
		},
	}
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

func (ftmm *FakeTiCDCMemberManager) SetSyncError(err error) {
	ftmm.err = err
}

func (ftmm *FakeTiCDCMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	if ftmm.err != nil {
		return ftmm.err
	}
	return nil
}
