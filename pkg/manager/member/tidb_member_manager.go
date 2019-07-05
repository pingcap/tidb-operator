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
	"strconv"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap.com/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/manager"
	"github.com/pingcap/tidb-operator/pkg/util"
	apps "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/listers/apps/v1beta1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

const (
	slowQueryLogVolumeName = "slowlog"
	slowQueryLogDir        = "/var/log/tidb"
	slowQueryLogFile       = slowQueryLogDir + "/slowlog"
)

type tidbMemberManager struct {
	setControl                   controller.StatefulSetControlInterface
	svcControl                   controller.ServiceControlInterface
	tidbControl                  controller.TiDBControlInterface
	setLister                    v1beta1.StatefulSetLister
	svcLister                    corelisters.ServiceLister
	podLister                    corelisters.PodLister
	tidbUpgrader                 Upgrader
	autoFailover                 bool
	operatorImage                string
	tidbFailover                 Failover
	tidbStatefulSetIsUpgradingFn func(corelisters.PodLister, *apps.StatefulSet, *v1alpha1.TidbCluster) (bool, error)
}

// NewTiDBMemberManager returns a *tidbMemberManager
func NewTiDBMemberManager(setControl controller.StatefulSetControlInterface,
	svcControl controller.ServiceControlInterface,
	tidbControl controller.TiDBControlInterface,
	setLister v1beta1.StatefulSetLister,
	svcLister corelisters.ServiceLister,
	podLister corelisters.PodLister,
	tidbUpgrader Upgrader,
	autoFailover bool,
	operatorImage string,
	tidbFailover Failover) manager.Manager {
	return &tidbMemberManager{
		setControl:                   setControl,
		svcControl:                   svcControl,
		tidbControl:                  tidbControl,
		setLister:                    setLister,
		svcLister:                    svcLister,
		podLister:                    podLister,
		tidbUpgrader:                 tidbUpgrader,
		autoFailover:                 autoFailover,
		operatorImage:                operatorImage,
		tidbFailover:                 tidbFailover,
		tidbStatefulSetIsUpgradingFn: tidbStatefulSetIsUpgrading,
	}
}

func (tmm *tidbMemberManager) Sync(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	// Sync Tidb StatefulSet
	if err := tmm.syncTiDBStatefulSetForTidbCluster(tc); err != nil {
		return err
	}

	if !tc.TiKVIsAvailable() {
		return controller.RequeueErrorf("TidbCluster: [%s/%s], waiting for TiKV cluster running", ns, tcName)
	}

	// Sync TiDB Headless Service
	return tmm.syncTiDBHeadlessServiceForTidbCluster(tc)
}

func (tmm *tidbMemberManager) syncTiDBHeadlessServiceForTidbCluster(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newSvc := tmm.getNewTiDBHeadlessServiceForTidbCluster(tc)
	oldSvcTmp, err := tmm.svcLister.Services(ns).Get(controller.TiDBPeerMemberName(tcName))
	if errors.IsNotFound(err) {
		err = SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		return tmm.svcControl.CreateService(tc, newSvc)
	}
	if err != nil {
		return err
	}

	oldSvc := oldSvcTmp.DeepCopy()

	equal, err := serviceEqual(newSvc, oldSvc)
	if err != nil {
		return err
	}
	if !equal {
		svc := *oldSvc
		svc.Spec = newSvc.Spec
		err = SetServiceLastAppliedConfigAnnotation(newSvc)
		if err != nil {
			return err
		}
		_, err = tmm.svcControl.UpdateService(tc, &svc)
		return err
	}

	return nil
}

func (tmm *tidbMemberManager) syncTiDBStatefulSetForTidbCluster(tc *v1alpha1.TidbCluster) error {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	newTiDBSet := tmm.getNewTiDBSetForTidbCluster(tc)
	oldTiDBSetTemp, err := tmm.setLister.StatefulSets(ns).Get(controller.TiDBMemberName(tcName))
	if errors.IsNotFound(err) {
		err = SetLastAppliedConfigAnnotation(newTiDBSet)
		if err != nil {
			return err
		}
		err = tmm.setControl.CreateStatefulSet(tc, newTiDBSet)
		if err != nil {
			return err
		}
		tc.Status.TiDB.StatefulSet = &apps.StatefulSetStatus{}
		return nil
	}

	if !tc.TiKVIsAvailable() {
		return controller.RequeueErrorf("TidbCluster: [%s/%s], waiting for TiKV cluster running", ns, tcName)
	}

	oldTiDBSet := oldTiDBSetTemp.DeepCopy()
	if err != nil {
		return err
	}

	if err = tmm.syncTidbClusterStatus(tc, oldTiDBSet); err != nil {
		return err
	}

	if !templateEqual(newTiDBSet.Spec.Template, oldTiDBSet.Spec.Template) || tc.Status.TiDB.Phase == v1alpha1.UpgradePhase {
		if err := tmm.tidbUpgrader.Upgrade(tc, oldTiDBSet, newTiDBSet); err != nil {
			return err
		}
	}

	if tmm.autoFailover {
		if tc.TiDBAllPodsStarted() && tc.TiDBAllMembersReady() && tc.Status.TiDB.FailureMembers != nil {
			tmm.tidbFailover.Recover(tc)
		} else if tc.TiDBAllPodsStarted() && !tc.TiDBAllMembersReady() {
			if err := tmm.tidbFailover.Failover(tc); err != nil {
				return err
			}
		}
	}

	if !statefulSetEqual(*newTiDBSet, *oldTiDBSet) {
		set := *oldTiDBSet
		set.Spec.Template = newTiDBSet.Spec.Template
		*set.Spec.Replicas = *newTiDBSet.Spec.Replicas
		set.Spec.UpdateStrategy = newTiDBSet.Spec.UpdateStrategy
		err := SetLastAppliedConfigAnnotation(&set)
		if err != nil {
			return err
		}
		_, err = tmm.setControl.UpdateStatefulSet(tc, &set)
		return err
	}

	return nil
}

func (tmm *tidbMemberManager) getNewTiDBHeadlessServiceForTidbCluster(tc *v1alpha1.TidbCluster) *corev1.Service {
	ns := tc.Namespace
	tcName := tc.Name
	instanceName := tc.GetLabels()[label.InstanceLabelKey]
	svcName := controller.TiDBPeerMemberName(tcName)
	tidbLabel := label.New().Instance(instanceName).TiDB().Labels()

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            svcName,
			Namespace:       ns,
			Labels:          tidbLabel,
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:       "status",
					Port:       10080,
					TargetPort: intstr.FromInt(10080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: tidbLabel,
		},
	}
}

func (tmm *tidbMemberManager) getNewTiDBSetForTidbCluster(tc *v1alpha1.TidbCluster) *apps.StatefulSet {
	ns := tc.GetNamespace()
	tcName := tc.GetName()
	instanceName := tc.GetLabels()[label.InstanceLabelKey]
	tidbConfigMap := controller.MemberConfigMapName(tc, v1alpha1.TiDBMemberType)

	annMount, annVolume := annotationsMountVolume()
	volMounts := []corev1.VolumeMount{
		annMount,
		{Name: "config", ReadOnly: true, MountPath: "/etc/tidb"},
		{Name: "startup-script", ReadOnly: true, MountPath: "/usr/local/bin"},
	}
	vols := []corev1.Volume{
		annVolume,
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

	var containers []corev1.Container
	if tc.Spec.TiDB.SeparateSlowLog {
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
			Image:           controller.GetSlowLogTailerImage(tc),
			ImagePullPolicy: tc.Spec.TiDB.SlowLogTailer.ImagePullPolicy,
			Resources:       util.ResourceRequirement(tc.Spec.TiDB.SlowLogTailer.ContainerSpec),
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
	if tc.Spec.TiDB.SeparateSlowLog {
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
			Value: strconv.FormatBool(tc.Spec.TiDB.BinlogEnabled),
		},
		{
			Name:  "SLOW_LOG_FILE",
			Value: slowLogFileEnvVal,
		},
	}

	containers = append(containers, corev1.Container{
		Name:            v1alpha1.TiDBMemberType.String(),
		Image:           tc.Spec.TiDB.Image,
		Command:         []string{"/bin/sh", "/usr/local/bin/tidb_start_script.sh"},
		ImagePullPolicy: tc.Spec.TiDB.ImagePullPolicy,
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
		Resources:    util.ResourceRequirement(tc.Spec.TiDB.ContainerSpec),
		Env:          envs,
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/status",
					Port: intstr.FromInt(10080),
				},
			},
			InitialDelaySeconds: int32(10),
		},
	})

	tidbLabel := label.New().Instance(instanceName).TiDB()
	podAnnotations := CombineAnnotations(controller.AnnProm(10080), tc.Spec.TiDB.Annotations)
	tidbSet := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            controller.TiDBMemberName(tcName),
			Namespace:       ns,
			Labels:          tidbLabel.Labels(),
			OwnerReferences: []metav1.OwnerReference{controller.GetOwnerRef(tc)},
		},
		Spec: apps.StatefulSetSpec{
			Replicas: func() *int32 { r := tc.TiDBRealReplicas(); return &r }(),
			Selector: tidbLabel.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      tidbLabel.Labels(),
					Annotations: podAnnotations,
				},
				Spec: corev1.PodSpec{
					SchedulerName:  tc.Spec.SchedulerName,
					Affinity:       tc.Spec.TiDB.Affinity,
					NodeSelector:   tc.Spec.TiDB.NodeSelector,
					InitContainers: []corev1.Container{WaitForPDContainer(tc.GetName(), tmm.operatorImage)},
					Containers:     containers,
					RestartPolicy:  corev1.RestartPolicyAlways,
					Tolerations:    tc.Spec.TiDB.Tolerations,
					Volumes:        vols,
				},
			},
			ServiceName:         controller.TiDBPeerMemberName(tcName),
			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy: apps.StatefulSetUpdateStrategy{Type: apps.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &apps.RollingUpdateStatefulSetStrategy{Partition: func() *int32 { r := tc.TiDBRealReplicas(); return &r }()},
			},
		},
	}
	return tidbSet
}

func (tmm *tidbMemberManager) syncTidbClusterStatus(tc *v1alpha1.TidbCluster, set *apps.StatefulSet) error {
	tc.Status.TiDB.StatefulSet = &set.Status

	upgrading, err := tmm.tidbStatefulSetIsUpgradingFn(tmm.podLister, set, tc)
	if err != nil {
		return err
	}
	if upgrading && tc.Status.TiKV.Phase != v1alpha1.UpgradePhase && tc.Status.PD.Phase != v1alpha1.UpgradePhase {
		tc.Status.TiDB.Phase = v1alpha1.UpgradePhase
	} else {
		tc.Status.TiDB.Phase = v1alpha1.NormalPhase
	}

	tidbStatus := map[string]v1alpha1.TiDBMember{}
	tidbHealth := tmm.tidbControl.GetHealth(tc)
	for name, health := range tidbHealth {
		newTidbMember := v1alpha1.TiDBMember{
			Name:   name,
			Health: health,
		}
		oldTidbMember, exist := tc.Status.TiDB.Members[name]
		if exist {
			newTidbMember.LastTransitionTime = oldTidbMember.LastTransitionTime
			newTidbMember.NodeName = oldTidbMember.NodeName
		}
		if !exist || oldTidbMember.Health != newTidbMember.Health {
			newTidbMember.LastTransitionTime = metav1.Now()
		}
		pod, err := tmm.podLister.Pods(tc.GetNamespace()).Get(name)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		if pod != nil && pod.Spec.NodeName != "" {
			// Update assiged node if pod exists and is scheduled
			newTidbMember.NodeName = pod.Spec.NodeName
		}
		tidbStatus[name] = newTidbMember
	}
	tc.Status.TiDB.Members = tidbStatus

	return nil
}

func tidbStatefulSetIsUpgrading(podLister corelisters.PodLister, set *apps.StatefulSet, tc *v1alpha1.TidbCluster) (bool, error) {
	if statefulSetIsUpgrading(set) {
		return true, nil
	}
	selector, err := label.New().
		Instance(tc.GetLabels()[label.InstanceLabelKey]).
		TiDB().
		Selector()
	if err != nil {
		return false, err
	}
	tidbPods, err := podLister.Pods(tc.GetNamespace()).List(selector)
	if err != nil {
		return false, err
	}
	for _, pod := range tidbPods {
		revisionHash, exist := pod.Labels[apps.ControllerRevisionHashLabelKey]
		if !exist {
			return false, nil
		}
		if revisionHash != tc.Status.TiDB.StatefulSet.UpdateRevision {
			return true, nil
		}
	}
	return false, nil
}

type FakeTiDBMemberManager struct {
	err error
}

func NewFakeTiDBMemberManager() *FakeTiDBMemberManager {
	return &FakeTiDBMemberManager{}
}

func (ftmm *FakeTiDBMemberManager) SetSyncError(err error) {
	ftmm.err = err
}

func (ftmm *FakeTiDBMemberManager) Sync(_ *v1alpha1.TidbCluster) error {
	if ftmm.err != nil {
		return ftmm.err
	}
	return nil
}
