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
	"fmt"
	"path"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	startscriptv1 "github.com/pingcap/tidb-operator/pkg/manager/member/startscript/v1"
	"github.com/pingcap/tidb-operator/pkg/util"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	initContainerName   = "wait"
	containerName       = "mysql-client"
	passwdKey           = "password"
	passwdPath          = "/etc/tidb/password" // nolint: gosec
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
	deps *controller.Dependencies
}

// NewTiDBInitManager return tidbInitManager
func NewTiDBInitManager(deps *controller.Dependencies) InitManager {
	return &tidbInitManager{deps: deps}
}

func (m *tidbInitManager) Sync(ti *v1alpha1.TidbInitializer) error {
	ns := ti.Namespace
	tcName := ti.Spec.Clusters.Name
	tc, err := m.deps.TiDBClusterLister.TidbClusters(ns).Get(tcName)
	if err != nil {
		return fmt.Errorf("TidbInitManager.Sync: failed to get tidbcluster %s for TidbInitializer %s/%s, error: %s", tcName, ns, ti.Name, err)
	}
	if tc.Spec.TiDB == nil {
		klog.Infof("TidbInitManager.Sync: Spec.TiDB is nil in tidbcluster %s, skip syncing TidbInitializer %s/%s", tcName, ns, ti.Name)
		return nil
	}

	err = m.syncTiDBInitConfigMap(ti, tc)
	if err != nil {
		return err
	}
	err = m.syncTiDBInitJob(ti)
	if err != nil {
		return err
	}
	return m.updateStatus(ti.DeepCopy())
}

func (m *tidbInitManager) updateStatus(ti *v1alpha1.TidbInitializer) error {
	name := controller.TiDBInitializerMemberName(ti.Spec.Clusters.Name)
	ns := ti.Namespace
	job, err := m.deps.JobLister.Jobs(ns).Get(name)
	if err != nil {
		return fmt.Errorf("updateStatus: failed to get job %s for TidbInitializer %s/%s, error: %s", name, ns, ti.Name, err)
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
		_, err = m.updateInitializer(ti)
		return err
	}
	return nil
}

func (m *tidbInitManager) updateInitializer(ti *v1alpha1.TidbInitializer) (*v1alpha1.TidbInitializer, error) {
	ns := ti.GetNamespace()
	tiName := ti.GetName()

	status := ti.Status.DeepCopy()
	var update *v1alpha1.TidbInitializer

	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var updateErr error
		update, updateErr = m.deps.Clientset.PingcapV1alpha1().TidbInitializers(ns).Update(context.TODO(), ti, metav1.UpdateOptions{})
		if updateErr == nil {
			klog.Infof("TidbInitializer: [%s/%s] updated successfully", ns, tiName)
			return nil
		}
		klog.V(4).Infof("failed to update TidbInitializer: [%s/%s], error: %v", ns, tiName, updateErr)

		if updated, err := m.deps.TiDBInitializerLister.TidbInitializers(ns).Get(tiName); err == nil {
			// make a copy so we don't mutate the shared cache
			ti = updated.DeepCopy()
			ti.Status = *status
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated TidbInitializer %s/%s from lister: %v", ns, tiName, err))
		}

		return updateErr
	})
	if err != nil {
		klog.Errorf("failed to update TidbInitializer: [%s/%s], error: %v", ns, tiName, err)
	}
	return update, err
}

func (m *tidbInitManager) syncTiDBInitConfigMap(ti *v1alpha1.TidbInitializer, tc *v1alpha1.TidbCluster) error {
	name := controller.TiDBInitializerMemberName(ti.Spec.Clusters.Name)
	ns := ti.Namespace
	cm := &corev1.ConfigMap{}

	if tc == nil {
		return fmt.Errorf("tc of TidbInitializer %s/%s is nil", ns, ti.Name)
	}

	exist, err := m.deps.TypedControl.Exist(client.ObjectKey{
		Namespace: ns,
		Name:      name,
	}, cm)
	if err != nil {
		return err
	}
	if exist {
		return nil
	}

	tlsClientEnabled, skipCA := false, false
	if tc.Spec.TiDB.IsTLSClientEnabled() && !tc.SkipTLSWhenConnectTiDB() {
		tlsClientEnabled = true
		skipCA = tc.Spec.TiDB.TLSClient.SkipInternalClientCA
	}
	tidbSvcPort := tc.Spec.TiDB.GetServicePort()
	newCm, err := getTiDBInitConfigMap(ti, tlsClientEnabled, skipCA, tidbSvcPort)
	if err != nil {
		return err
	}

	err = m.deps.TypedControl.Create(ti, newCm)
	if errors.IsAlreadyExists(err) {
		klog.Infof("Configmap %s/%s already exists", newCm.Namespace, newCm.Name)
		return nil
	}
	return err
}

func (m *tidbInitManager) syncTiDBInitJob(ti *v1alpha1.TidbInitializer) error {
	ns := ti.GetNamespace()
	name := ti.GetName()
	jobName := controller.TiDBInitializerMemberName(ti.Spec.Clusters.Name)

	_, err := m.deps.JobLister.Jobs(ns).Get(jobName)
	if err == nil {
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("TiDBInitializer %s/%s get job %s failed, err: %v", ns, ti.Name, name, err)
	}

	job, err := m.makeTiDBInitJob(ti)
	if err != nil {
		return err
	}

	err = m.deps.TypedControl.Create(ti, job)
	if errors.IsAlreadyExists(err) {
		klog.Infof("Job %s/%s already exists", job.Namespace, job.Name)
		return nil
	}
	return err
}

func (m *tidbInitManager) makeTiDBInitJob(ti *v1alpha1.TidbInitializer) (*batchv1.Job, error) {
	jobName := controller.TiDBInitializerMemberName(ti.Spec.Clusters.Name)
	ns := ti.Namespace
	tcName := ti.Spec.Clusters.Name

	tc, err := m.deps.TiDBClusterLister.TidbClusters(ns).Get(tcName)
	if err != nil {
		return nil, fmt.Errorf("makeTiDBInitJob: failed to get tidbcluster %s for TidbInitializer %s/%s, error: %s", tcName, ns, ti.Name, err)
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
			Labels:      util.CombineStringMap(initLabel, ti.ObjectMeta.Labels),
			Annotations: util.CopyStringMap(ti.ObjectMeta.Annotations),
		},
		Spec: corev1.PodSpec{
			ImagePullSecrets: ti.Spec.ImagePullSecrets,
			SecurityContext:  ti.Spec.PodSecurityContext,
			InitContainers: []corev1.Container{
				{
					Name:    initContainerName,
					Image:   ti.Spec.Image,
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
		podSpec.Spec.InitContainers[0].ImagePullPolicy = *ti.Spec.ImagePullPolicy
	}
	if ti.Spec.Resources != nil {
		podSpec.Spec.Containers[0].Resources = *ti.Spec.Resources
		podSpec.Spec.InitContainers[0].Resources = *ti.Spec.Resources
	}

	job := &batchv1.Job{
		ObjectMeta: meta,
		Spec: batchv1.JobSpec{
			BackoffLimit: pointer.Int32Ptr(0),
			Template:     *podSpec,
		},
	}

	return job, nil
}

func getTiDBInitConfigMap(ti *v1alpha1.TidbInitializer, tlsClientEnabled bool, skipCA bool, tidbSvcPort int32) (*corev1.ConfigMap, error) {
	var initSQL, passwdSet bool

	permitHost := ti.GetPermitHost()

	if ti.Spec.InitSql != nil || ti.Spec.InitSqlConfigMap != nil {
		initSQL = true
	}
	if ti.Spec.PasswordSecret != nil {
		passwdSet = true
	}

	initStartScript, err := startscriptv1.RenderTiDBInitInitStartScript(&startscriptv1.TiDBInitInitStartScriptModel{
		ClusterName:     ti.Spec.Clusters.Name,
		TiDBServicePort: tidbSvcPort,
	})
	if err != nil {
		return nil, err
	}

	initModel := &startscriptv1.TiDBInitStartScriptModel{
		ClusterName:     ti.Spec.Clusters.Name,
		PermitHost:      permitHost,
		InitSQL:         initSQL,
		PasswordSet:     passwdSet,
		TiDBServicePort: tidbSvcPort,
	}
	if tlsClientEnabled {
		initModel.TLS = true
		initModel.SkipCA = skipCA
		initModel.CAPath = path.Join(util.TiDBClientTLSPath, corev1.ServiceAccountRootCAKey)
		initModel.CertPath = path.Join(util.TiDBClientTLSPath, corev1.TLSCertKey)
		initModel.KeyPath = path.Join(util.TiDBClientTLSPath, corev1.TLSPrivateKeyKey)
	}
	startScript, err := startscriptv1.RenderTiDBInitStartScript(initModel)
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
		Name:        name,
		Namespace:   ti.Namespace,
		Labels:      util.CombineStringMap(initLabel, ti.ObjectMeta.Labels),
		Annotations: util.CopyStringMap(ti.ObjectMeta.Annotations),
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
