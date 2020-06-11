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
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
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
	// initContainerImage is the image for init container
	initContainerImage = "busybox:1.31.1"
)

// InitManager implements the logic for syncing TidbInitializer.
type InitManager interface {
	// Sync	implements the logic for syncing TidbInitializer.
	Sync(*v1alpha1.TidbInitializer) error
}

type tidbInitManager struct {
	jobLister    batchlisters.JobLister
	genericCli   client.Client
	tiLister     listers.TidbInitializerLister
	tcLister     listers.TidbClusterLister
	typedControl controller.TypedControlInterface
}

// NewTiDBInitManager return tidbInitManager
func NewTiDBInitManager(
	jobLister batchlisters.JobLister,
	genericCli client.Client,
	tiLister listers.TidbInitializerLister,
	tcLister listers.TidbClusterLister,
	typedControl controller.TypedControlInterface,
) InitManager {
	return &tidbInitManager{
		jobLister,
		genericCli,
		tiLister,
		tcLister,
		typedControl,
	}
}

func (tm *tidbInitManager) Sync(ti *v1alpha1.TidbInitializer) error {
	err := tm.syncTiDBInitConfigMap(ti)
	if err != nil {
		return err
	}
	err = tm.syncTiDBInitJob(ti)
	if err != nil {
		return err
	}
	return tm.updateStatus(ti.DeepCopy())
}

func (tm *tidbInitManager) updateStatus(ti *v1alpha1.TidbInitializer) error {
	name := controller.TiDBInitializerMemberName(ti.Spec.Clusters.Name)
	ns := ti.Namespace
	job, err := tm.jobLister.Jobs(ns).Get(name)
	if err != nil {
		return err
	}

	phase := v1alpha1.InitializePhaseRunning
	if len(job.Status.Conditions) > 0 {
		for _, c := range job.Status.Conditions {
			if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
				phase = v1alpha1.InitializePhaseCompleted
				break
			}
			if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
				phase = v1alpha1.InitializePhaseFailed
				break
			}
		}
	}

	var update bool
	if !apiequality.Semantic.DeepEqual(ti.Status.JobStatus, job.Status) {
		job.Status.DeepCopyInto(&ti.Status.JobStatus)
		update = true
	}
	if ti.Status.Phase != phase {
		ti.Status.Phase = phase
		update = true
	}
	if update {
		status := ti.Status.DeepCopy()
		return controller.GuaranteedUpdate(tm.genericCli, ti, func() error {
			ti.Status = *status
			return nil
		})
	}
	return nil
}

func (tm *tidbInitManager) syncTiDBInitConfigMap(ti *v1alpha1.TidbInitializer) error {
	name := controller.TiDBInitializerMemberName(ti.Spec.Clusters.Name)
	ns := ti.Namespace
	cm := &corev1.ConfigMap{}
	tcName := ti.Spec.Clusters.Name

	exist, err := tm.typedControl.Exist(client.ObjectKey{
		Namespace: ns,
		Name:      name,
	}, cm)
	if err != nil {
		return err
	}
	if exist {
		return nil
	}

	tc, err := tm.tcLister.TidbClusters(ns).Get(tcName)
	if err != nil {
		return err
	}

	newCm, err := getTiDBInitConfigMap(ti, tc.Spec.TiDB.IsTLSClientEnabled())
	if err != nil {
		return err
	}

	err = tm.typedControl.Create(ti, newCm)
	if errors.IsAlreadyExists(err) {
		klog.Infof("Configmap %s/%s already exists", newCm.Namespace, newCm.Name)
		return nil
	}
	return err
}

func (tm *tidbInitManager) syncTiDBInitJob(ti *v1alpha1.TidbInitializer) error {
	ns := ti.GetNamespace()
	name := ti.GetName()
	jobName := controller.TiDBInitializerMemberName(ti.Spec.Clusters.Name)

	job, err := tm.jobLister.Jobs(ns).Get(jobName)
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("TiDBInitializer %s/%s get job %s failed, err: %v", ns, ti.Name, name, err)
	}

	job, err = tm.makeTiDBInitJob(ti)
	if err != nil {
		return err
	}

	err = tm.typedControl.Create(ti, job)
	if errors.IsAlreadyExists(err) {
		klog.Infof("Job %s/%s already exists", job.Namespace, job.Name)
		return nil
	}
	return err
}

func (tm *tidbInitManager) makeTiDBInitJob(ti *v1alpha1.TidbInitializer) (*batchv1.Job, error) {
	jobName := controller.TiDBInitializerMemberName(ti.Spec.Clusters.Name)
	ns := ti.Namespace
	tcName := ti.Spec.Clusters.Name

	tc, err := tm.tcLister.TidbClusters(ns).Get(tcName)
	if err != nil {
		return nil, err
	}

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

	if tc.Spec.TiDB.IsTLSClientEnabled() && !tc.SkipTLSWhenConnectTiDB() {
		secretName := util.TiDBClientTLSSecretName(tcName)
		if ti.Spec.TLSClientSecretName != nil {
			secretName = *ti.Spec.TLSClientSecretName
		}

		vms = append(vms, corev1.VolumeMount{
			Name:      "tidb-client-tls",
			ReadOnly:  true,
			MountPath: util.TiDBClientTLSPath,
		})
		vs = append(vs, corev1.Volume{
			Name: "tidb-client-tls",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
				},
			},
		})
	}
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
					Image:   initContainerImage,
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
	if ti.Spec.ImagePullSecrets != nil {
		podSpec.Spec.ImagePullSecrets = ti.Spec.ImagePullSecrets
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

func getTiDBInitConfigMap(ti *v1alpha1.TidbInitializer, tlsClientEnabled bool) (*corev1.ConfigMap, error) {
	var initSQL, passwdSet bool

	permitHost := ti.GetPermitHost()

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

	initModel := &TiDBInitStartScriptModel{
		ClusterName: ti.Spec.Clusters.Name,
		PermitHost:  permitHost,
		InitSQL:     initSQL,
		PasswordSet: passwdSet,
	}
	if tlsClientEnabled {
		initModel.TLS = true
		initModel.CAPath = path.Join(util.TiDBClientTLSPath, corev1.ServiceAccountRootCAKey)
		initModel.CertPath = path.Join(util.TiDBClientTLSPath, corev1.TLSCertKey)
		initModel.KeyPath = path.Join(util.TiDBClientTLSPath, corev1.TLSPrivateKeyKey)
	}
	startScript, err := RenderTiDBInitStartScript(initModel)
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
	initLabel := label.NewInitializer().Instance(ti.Name).Initializer(ti.Name)

	objMeta := metav1.ObjectMeta{
		Name:      name,
		Namespace: ti.Namespace,
		Labels:    initLabel,
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
