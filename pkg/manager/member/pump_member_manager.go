// Copyright 2019. PingCAP, Inc.
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
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	v1 "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

const (
	defaultPumpLogLevel = "info"
)

type pumpMemberManager struct {
	setControl   controller.StatefulSetControlInterface
	svcControl   controller.ServiceControlInterface
	typedControl controller.TypedControlInterface
	setLister    v1.StatefulSetLister
	svcLister    corelisters.ServiceLister
	cmLister     corelisters.ConfigMapLister
}

// NewPumpMemberManager returns a controller to reconcile pump clusters
func NewPumpMemberManager(
	setControl controller.StatefulSetControlInterface,
	svcControl controller.ServiceControlInterface,
	typedControl controller.TypedControlInterface,
	setLister v1.StatefulSetLister,
	svcLister corelisters.ServiceLister,
	cmLister corelisters.ConfigMapLister) manager.Manager {
	return &pumpMemberManager{
		setControl,
		svcControl,
		typedControl,
		setLister,
		svcLister,
		cmLister,
	}
}

func (pmm *pumpMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	if tc.Spec.Pump == nil {
		return nil
	}
	if err := pmm.syncHeadlessService(tc); err != nil {
		return err
	}
	return pmm.syncStatefulSet(tc)
}

// syncStatefulSet syncs the pump statefulset
// TODO: sync statefulset status of pump to tidbcluster
func (pmm *pumpMemberManager) syncStatefulSet(tc *v1alpha1.TidbCluster) error {

	oldPumpSetTemp, err := pmm.setLister.StatefulSets(tc.Namespace).Get(controller.PumpMemberName(tc.Name))
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	notFound := errors.IsNotFound(err)
	oldPumpSet := oldPumpSetTemp.DeepCopy()

	cm, err := pmm.syncConfigMap(tc, oldPumpSet)
	if err != nil {
		return err
	}

	newPumpSet, err := getNewPumpStatefulSet(tc, cm)
	if err != nil {
		return err
	}
	if notFound {
		err = SetStatefulSetLastAppliedConfigAnnotation(newPumpSet)
		if err != nil {
			return err
		}
		return pmm.setControl.CreateStatefulSet(tc, newPumpSet)
	}

	isOrphan := metav1.GetControllerOf(oldPumpSet) == nil

	if !statefulSetEqual(*newPumpSet, *oldPumpSet) || isOrphan {
		set := *oldPumpSet
		set.Spec.Template = newPumpSet.Spec.Template
		*set.Spec.Replicas = *newPumpSet.Spec.Replicas
		set.Spec.UpdateStrategy = newPumpSet.Spec.UpdateStrategy
		err := SetStatefulSetLastAppliedConfigAnnotation(&set)
		if err != nil {
			return err
		}
		if isOrphan {
			set.OwnerReferences = newPumpSet.OwnerReferences
			set.Labels = newPumpSet.Labels
		}
		_, err = pmm.setControl.UpdateStatefulSet(tc, &set)
		return err
	}
	return nil
}

func (pmm *pumpMemberManager) syncHeadlessService(tc *v1alpha1.TidbCluster) error {

	newSvc := getNewPumpHeadlessService(tc)
	oldSvc, err := pmm.svcLister.Services(newSvc.Namespace).Get(newSvc.Name)
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
	isOrphan := metav1.GetControllerOf(oldSvc) == nil

	if !equal || isOrphan {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		err = SetServiceLastAppliedConfigAnnotation(&svc)
		if err != nil {
			return err
		}
		// Adopt headless-service created by helm
		if isOrphan {
			svc.OwnerReferences = newSvc.OwnerReferences
			svc.Labels = newSvc.Labels
		}
		_, err = pmm.svcControl.UpdateService(tc, &svc)
		return err
	}
	return nil
}

func (pmm *pumpMemberManager) syncConfigMap(tc *v1alpha1.TidbCluster, set *appsv1.StatefulSet) (*corev1.ConfigMap, error) {

	newCm, err := getNewPumpConfigMap(tc)
	if err != nil {
		return nil, err
	}
	// In-place update should pick the name of currently in-use configmap if exists to avoid rolling-update if:
	//   - user switch strategy from RollingUpdate to In-place
	//   - the statefulset and configmap is created by other clients (e.g. helm)
	if set != nil && tc.Spec.Pump.ConfigUpdateStrategy == v1alpha1.ConfigUpdateStrategyInPlace {
		inUseName := FindConfigMapVolume(&set.Spec.Template.Spec, func(name string) bool {
			return strings.HasPrefix(name, controller.PumpMemberName(tc.Name))
		})
		// find an in-use configmap, will update it in-place
		if inUseName != "" {
			newCm.Name = inUseName
		}
	}

	return pmm.typedControl.CreateOrUpdateConfigMap(tc, newCm)
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

	confText, err := MarshalTOML(spec.Config)
	if err != nil {
		return nil, err
	}

	name := controller.PumpMemberName(tc.Name)
	data := map[string]string{
		"pump-config": string(confText),
	}
	if spec.ConfigUpdateStrategy == v1alpha1.ConfigUpdateStrategyRollingUpdate {
		sum, err := Sha256Sum(data)
		if err != nil {
			return nil, err
		}
		suffix := fmt.Sprintf("%x", sum)[0:7]
		name = fmt.Sprintf("%s-%s", name, suffix)
	}
	objMeta.Name = name

	return &corev1.ConfigMap{
		ObjectMeta: objMeta,
		Data:       data,
	}, nil
}

func getNewPumpStatefulSet(tc *v1alpha1.TidbCluster, cm *corev1.ConfigMap) (*appsv1.StatefulSet, error) {
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
	containers := []corev1.Container{
		{
			Name:            "pump",
			Image:           spec.Image(),
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
			Resources: util.ResourceRequirement(tc.Spec.Pump.Resources),
			Env:       envs,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "data",
					MountPath: "/data",
				},
				{
					Name:      "config",
					MountPath: "/etc/pump",
				},
			},
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

	volumeClaims := []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "data",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				StorageClassName: &storageClass,
				Resources:        *storageRequest,
			},
		},
	}

	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: podAnnos,
			Labels:      pumpLabel,
		},
		Spec: corev1.PodSpec{
			Containers: containers,
			Volumes:    volumes,

			Affinity:        spec.Affinity(),
			Tolerations:     spec.Tolerations(),
			NodeSelector:    spec.NodeSelector(),
			SchedulerName:   spec.SchedulerName(),
			SecurityContext: spec.PodSecurityContext(),
			HostNetwork:     spec.HostNetwork(),
			DNSPolicy:       spec.DnsPolicy(),
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
		},
	}, nil
}

func getPumpMeta(tc *v1alpha1.TidbCluster, nameFunc func(string) string) (metav1.ObjectMeta, label.Label) {
	instanceName := tc.GetLabels()[label.InstanceLabelKey]
	pumpLabel := label.New().Instance(instanceName).Pump()

	objMeta := metav1.ObjectMeta{
		Name:            nameFunc(tc.Name),
		Namespace:       tc.Namespace,
		Labels:          pumpLabel,
		OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
	}
	return objMeta, pumpLabel
}

func getPumpStartScript(tc *v1alpha1.TidbCluster) (string, error) {
	// Keep the logic same as helm chart, but pump has not supported tls yet (no cert mounted)
	// TODO: support tls
	scheme := "http"
	if tc.Spec.EnableTLSCluster {
		scheme = "https"
	}
	return RenderPumpStartScript(&PumpStartScriptModel{
		Scheme:      scheme,
		ClusterName: tc.Name,
		LogLevel:    getPumpLogLevel(tc),
	})
}

func getPumpLogLevel(tc *v1alpha1.TidbCluster) string {

	config := tc.Spec.Pump.Config
	if config == nil {
		return defaultPumpLogLevel
	}

	raw, ok := config["log-level"]
	if !ok {
		return defaultPumpLogLevel
	}

	logLevel, ok := raw.(string)
	if !ok {
		return defaultPumpLogLevel
	}

	return logLevel
}

type FakePumpMemberManager struct {
	err error
}

func NewFakePumpMemberManager() *FakePumpMemberManager {
	return &FakePumpMemberManager{}
}

func (ftmm *FakePumpMemberManager) SetSyncError(err error) {
	ftmm.err = err
}

func (ftmm *FakePumpMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	if ftmm.err != nil {
		return ftmm.err
	}
	return nil
}
