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
	"fmt"
	"path"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batchlisters "k8s.io/client-go/listers/batch/v1"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client/clientset/versioned"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
)

const (
	initContainerName   = "wait"
	containerName       = "mysql-client"
	passwdKey           = "password"
	passwdPath          = "/etc/tidb/password"
	sqlKey              = "init-sql"
	sqlPath             = "init.sql"
	sqlDir              = "/data"
	startScriptPath     = "start_script.py"
	initStartScriptPath = "init_start_script.sh"
	startScriptDir      = "/usr/local/bin"
	startKey            = "start-script"
	initStartKey        = "init-start-script"
)

// InitManager implements the logic for syncing TidbInitializer.
type InitManager interface {
	// Sync	implements the logic for syncing TidbInitializer.
	Sync(*v1alpha1.TidbInitializer) error
}

type tidbInitManager struct {
	jobLister  batchlisters.JobLister
	jobControl controller.JobControlInterface
	cmControl  controller.ConfigMapControlInterface
	cli        versioned.Interface
}

// NewTiDBInitManager return tidbInitManager
func NewTiDBInitManager(
	jobLister batchlisters.JobLister,
	jobControl controller.JobControlInterface,
	cmControl controller.ConfigMapControlInterface,
	cli versioned.Interface,
) InitManager {
	return &tidbInitManager{
		jobLister,
		jobControl,
		cmControl,
		cli,
	}
}

func (tm *tidbInitManager) Sync(ti *v1alpha1.TidbInitializer) error {
	if ti.DeletionTimestamp != nil {
		return nil
	}

	name := controller.TiDBInitializerMemberName(ti.Spec.Clusters.Name)
	ns := ti.Namespace
	ok, err := tm.cmControl.ConfigMapExist(ns, name)
	if err != nil {
		return err
	}
	if !ok {
		err = tm.syncTiDBInitConfigMap(ti)
		if err != nil {
			return err
		}
	}

	job, err := tm.jobLister.Jobs(ns).Get(name)
	if err == nil {
		// job has been created
		return tm.checkStatusUpdate(ti.DeepCopy(), job)
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("TiDBInitializer %s/%s get job %s failed, err: %v", ns, ti.Name, name, err)
	}

	return tm.syncTiDBInitJob(ti)
}

func (tm *tidbInitManager) checkStatusUpdate(ti *v1alpha1.TidbInitializer, job *batchv1.Job) error {
	if job == nil || ti == nil {
		return nil
	}
	if apiequality.Semantic.DeepEqual(ti.Status.JobStatus, job.Status) {
		return nil
	}
	job.Status.DeepCopyInto(&ti.Status.JobStatus)
	_, err := tm.cli.PingcapV1alpha1().TidbInitializers(ti.Namespace).Update(ti)
	return err
}

func (tm *tidbInitManager) syncTiDBInitConfigMap(ti *v1alpha1.TidbInitializer) error {
	newCm, err := getTiDBInitConfigMap(ti)
	if err != nil {
		return err
	}

	_, err = tm.cmControl.CreateConfigMap(ti, newCm)
	return err
}

func (tm *tidbInitManager) syncTiDBInitJob(ti *v1alpha1.TidbInitializer) error {
	ns := ti.GetNamespace()
	name := ti.GetName()
	tidbInitJobName := controller.TiDBInitializerMemberName(ti.Spec.Clusters.Name)

	job, err := tm.makeTiDBInitJob(ti)
	if err != nil {
		return err
	}

	if err := tm.jobControl.CreateJob(ti, job); err != nil {
		return fmt.Errorf("create job %s for TiDBInitializer %s/%s failed, err: %v", tidbInitJobName, ns, name, err)
	}

	return nil
}

func (tm *tidbInitManager) makeTiDBInitJob(ti *v1alpha1.TidbInitializer) (*batchv1.Job, error) {
	jobName := controller.TiDBInitializerMemberName(ti.Spec.Clusters.Name)

	var envs []corev1.EnvVar
	if ti.Spec.Timezone != "" {
		envs = append(envs, corev1.EnvVar{
			Name:  "TZ",
			Value: ti.Spec.Timezone,
		})
	}

	initcmds := []string{
		"sh",
		"/usr/local/bin/init_start_script.sh",
	}
	cmds := []string{
		"python",
		"/usr/local/bin/start_script.py",
	}

	var vms []corev1.VolumeMount
	var vs []corev1.Volume
	vms = append(vms, corev1.VolumeMount{
		Name:      startKey,
		ReadOnly:  true,
		MountPath: path.Join(startScriptDir, startScriptPath),
		SubPath:   startScriptPath,
	})
	vs = append(vs, corev1.Volume{
		Name: startKey,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: jobName,
				},
				Items: []corev1.KeyToPath{{Key: startKey, Path: startScriptPath}},
			},
		},
	})
	vs = append(vs, corev1.Volume{
		Name: initStartKey,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: jobName,
				},
				Items: []corev1.KeyToPath{{Key: initStartKey, Path: initStartScriptPath}},
			},
		},
	})
	if ti.Spec.PasswordSecret != nil {
		vms = append(vms, corev1.VolumeMount{
			Name: passwdKey, ReadOnly: true, MountPath: passwdPath,
		})
		vs = append(vs, corev1.Volume{
			Name: passwdKey,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: *ti.Spec.PasswordSecret,
				},
			},
		})
	}
	if ti.Spec.InitSqlConfigMap != nil {
		vms = append(vms, corev1.VolumeMount{
			Name: sqlKey, ReadOnly: true, MountPath: sqlDir,
		})
		vs = append(vs, corev1.Volume{
			Name: sqlKey,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: *ti.Spec.InitSqlConfigMap,
					},
					Items: []corev1.KeyToPath{{Key: sqlKey, Path: sqlPath}},
				},
			},
		})
	} else if ti.Spec.InitSql != nil {
		vms = append(vms, corev1.VolumeMount{
			Name: sqlKey, ReadOnly: true, MountPath: sqlDir,
		})
		vs = append(vs, corev1.Volume{
			Name: sqlKey,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: jobName,
					},
					Items: []corev1.KeyToPath{{Key: sqlKey, Path: sqlPath}},
				},
			},
		})
	}

	meta, initLabel := getInitMeta(ti)

	podSpec := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: initLabel.Labels(),
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{
					Name:    initContainerName,
					Image:   v1alpha1.DefaultHelperImage,
					Command: initcmds,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      initStartKey,
							ReadOnly:  true,
							MountPath: path.Join(startScriptDir, initStartScriptPath),
							SubPath:   initStartScriptPath,
						},
					},
					Env: envs,
				},
			},
			Containers: []corev1.Container{
				{
					Name:         containerName,
					Image:        ti.Spec.Image,
					Command:      cmds,
					VolumeMounts: vms,
					Env:          envs,
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Volumes:       vs,
		},
	}

	if ti.Spec.ImagePullPolicy != nil {
		podSpec.Spec.Containers[0].ImagePullPolicy = *ti.Spec.ImagePullPolicy
	}
	if ti.Spec.Resources != nil {
		podSpec.Spec.Containers[0].Resources = *ti.Spec.Resources
	}

	job := &batchv1.Job{
		ObjectMeta: meta,
		Spec: batchv1.JobSpec{
			BackoffLimit: controller.Int32Ptr(0),
			Template:     *podSpec,
		},
	}

	return job, nil
}

func getTiDBInitConfigMap(ti *v1alpha1.TidbInitializer) (*corev1.ConfigMap, error) {
	var permitHost string
	var initSQL, passwdSet bool

	if ti.Spec.PermitHost == nil || *ti.Spec.PermitHost == "" {
		permitHost = "%"
	} else {
		permitHost = *ti.Spec.PermitHost
	}
	if ti.Spec.InitSql != nil || ti.Spec.InitSqlConfigMap != nil {
		initSQL = true
	}
	if ti.Spec.PasswordSecret != nil {
		passwdSet = true
	}

	initStartScript, err := RenderTiDBInitInitStartScript(&TiDBInitInitStartScriptModel{
		ClusterName: ti.Spec.Clusters.Name,
	})
	if err != nil {
		return nil, err
	}

	startScript, err := RenderTiDBInitStartScript(&TiDBInitStartScriptModel{
		ClusterName: ti.Spec.Clusters.Name,
		PermitHost:  permitHost,
		InitSQL:     initSQL,
		PasswordSet: passwdSet,
	})
	if err != nil {
		return nil, err
	}

	data := map[string]string{
		initStartKey: initStartScript,
		startKey:     startScript,
	}
	if ti.Spec.InitSql != nil {
		data[sqlKey] = *ti.Spec.InitSql
	}

	meta, _ := getInitMeta(ti)

	cm := &corev1.ConfigMap{
		ObjectMeta: meta,
		Data:       data,
	}
	return cm, nil
}

func getInitMeta(ti *v1alpha1.TidbInitializer) (metav1.ObjectMeta, label.Label) {
	name := controller.TiDBInitializerMemberName(ti.Spec.Clusters.Name)
	initLabel := label.NewInitializer().Instance(ti.Name).InitJob().Init(ti.Name)

	objMeta := metav1.ObjectMeta{
		Name:            name,
		Namespace:       ti.Namespace,
		Labels:          initLabel,
		OwnerReferences: []metav1.OwnerReference{controller.GetInitOwnerRef(ti)},
	}
	return objMeta, initLabel
}

var _ InitManager = &tidbInitManager{}

// FakeTiDBInitManager FakeTiDBInitManager
type FakeTiDBInitManager struct {
	err error
}

// NewFakeTiDBInitManager NewFakeTiDBInitManager
func NewFakeTiDBInitManager() *FakeTiDBInitManager {
	return &FakeTiDBInitManager{}
}

// SetSyncError SetSyncError
func (ftm *FakeTiDBInitManager) SetSyncError(err error) {
	ftm.err = err
}

// Sync Sync
func (ftm *FakeTiDBInitManager) Sync(_ *v1alpha1.TidbInitializer) error {
	return ftm.err
}

var _ InitManager = &FakeTiDBInitManager{}
