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

package restore

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	backuputil "github.com/pingcap/tidb-operator/pkg/backup/util"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	v1alpha1listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	batchlisters "k8s.io/client-go/listers/batch/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/utils/pointer"
)

type restoreManager struct {
	backupLister  listers.BackupLister
	statusUpdater controller.RestoreConditionUpdaterInterface
	kubeCli       kubernetes.Interface
	jobLister     batchlisters.JobLister
	jobControl    controller.JobControlInterface
	pvcLister     corelisters.PersistentVolumeClaimLister
	tcLister      v1alpha1listers.TidbClusterLister
	pvcControl    controller.GeneralPVCControlInterface
}

// NewRestoreManager return restoreManager
func NewRestoreManager(
	backupLister listers.BackupLister,
	statusUpdater controller.RestoreConditionUpdaterInterface,
	kubeCli kubernetes.Interface,
	jobLister batchlisters.JobLister,
	jobControl controller.JobControlInterface,
	pvcLister corelisters.PersistentVolumeClaimLister,
	tcLister v1alpha1listers.TidbClusterLister,
	pvcControl controller.GeneralPVCControlInterface,
) backup.RestoreManager {
	return &restoreManager{
		backupLister,
		statusUpdater,
		kubeCli,
		jobLister,
		jobControl,
		pvcLister,
		tcLister,
		pvcControl,
	}
}

func (rm *restoreManager) Sync(restore *v1alpha1.Restore) error {
	return rm.syncRestoreJob(restore)
}

func (rm *restoreManager) syncRestoreJob(restore *v1alpha1.Restore) error {
	ns := restore.GetNamespace()
	name := restore.GetName()
	restoreJobName := restore.GetRestoreJobName()

	err := backuputil.ValidateRestore(restore)
	if err != nil {
		rm.statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreInvalid,
			Status:  corev1.ConditionTrue,
			Reason:  "InvalidSpec",
			Message: err.Error(),
		})

		return controller.IgnoreErrorf("invalid restore spec %s/%s", ns, name)
	}

	_, err = rm.jobLister.Jobs(ns).Get(restoreJobName)
	if err == nil {
		// already have a backup job running，return directly
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("restore %s/%s get job %s failed, err: %v", ns, name, restoreJobName, err)
	}

	var (
		job    *batchv1.Job
		reason string
	)
	if restore.Spec.BR == nil {
		job, reason, err = rm.makeImportJob(restore)
		if err != nil {
			rm.statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
				Type:    v1alpha1.RestoreRetryFailed,
				Status:  corev1.ConditionTrue,
				Reason:  reason,
				Message: err.Error(),
			})
			return err
		}

		reason, err = rm.ensureRestorePVCExist(restore)
		if err != nil {
			rm.statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
				Type:    v1alpha1.RestoreRetryFailed,
				Status:  corev1.ConditionTrue,
				Reason:  reason,
				Message: err.Error(),
			})
			return err
		}
	} else {
		job, reason, err = rm.makeRestoreJob(restore)
		if err != nil {
			rm.statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
				Type:    v1alpha1.RestoreRetryFailed,
				Status:  corev1.ConditionTrue,
				Reason:  reason,
				Message: err.Error(),
			})
			return err
		}
	}

	if err := rm.jobControl.CreateJob(restore, job); err != nil {
		errMsg := fmt.Errorf("create restore %s/%s job %s failed, err: %v", ns, name, restoreJobName, err)
		rm.statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreRetryFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "CreateRestoreJobFailed",
			Message: errMsg.Error(),
		})
		return errMsg
	}

	return rm.statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
		Type:   v1alpha1.RestoreScheduled,
		Status: corev1.ConditionTrue,
	})
}

func (rm *restoreManager) makeImportJob(restore *v1alpha1.Restore) (*batchv1.Job, string, error) {
	ns := restore.GetNamespace()
	name := restore.GetName()

	envVars, reason, err := backuputil.GenerateTidbPasswordEnv(ns, name, restore.Spec.To.SecretName, restore.Spec.UseKMS, rm.kubeCli)
	if err != nil {
		return nil, reason, err
	}

	storageEnv, reason, err := backuputil.GenerateStorageCertEnv(ns, restore.Spec.UseKMS, restore.Spec.StorageProvider, rm.kubeCli)
	if err != nil {
		return nil, reason, fmt.Errorf("restore %s/%s, %v", ns, name, err)
	}

	backupPath, reason, err := backuputil.GetBackupDataPath(restore.Spec.StorageProvider)
	if err != nil {
		return nil, reason, fmt.Errorf("restore %s/%s, %v", ns, name, err)
	}

	envVars = append(envVars, storageEnv...)
	args := []string{
		"import",
		fmt.Sprintf("--namespace=%s", ns),
		fmt.Sprintf("--restoreName=%s", name),
		fmt.Sprintf("--backupPath=%s", backupPath),
	}

	volumeMounts := []corev1.VolumeMount{}
	volumes := []corev1.Volume{}
	if restore.Spec.To.TLSClientSecretName != nil {
		args = append(args, "--client-tls=true")
		clientSecretName := *restore.Spec.To.TLSClientSecretName
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "tidb-client-tls",
			ReadOnly:  true,
			MountPath: util.TiDBClientTLSPath,
		})
		volumes = append(volumes, corev1.Volume{
			Name: "tidb-client-tls",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: clientSecretName,
				},
			},
		})
	}

	restoreLabel := label.NewBackup().Instance(restore.GetInstanceName()).RestoreJob().Restore(name)
	serviceAccount := constants.DefaultServiceAccountName
	if restore.Spec.ServiceAccount != "" {
		serviceAccount = restore.Spec.ServiceAccount
	}

	// TODO: need add ResourceRequirement for restore job
	podSpec := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      restoreLabel.Labels(),
			Annotations: restore.Annotations,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: serviceAccount,
			Containers: []corev1.Container{
				{
					Name:            label.RestoreJobLabelVal,
					Image:           controller.TidbBackupManagerImage,
					Args:            args,
					ImagePullPolicy: corev1.PullIfNotPresent,
					VolumeMounts: append([]corev1.VolumeMount{
						{Name: label.RestoreJobLabelVal, MountPath: constants.BackupRootPath},
					}, volumeMounts...),
					Env:       util.AppendEnvIfPresent(envVars, "TZ"),
					Resources: restore.Spec.ResourceRequirements,
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Affinity:      restore.Spec.Affinity,
			Tolerations:   restore.Spec.Tolerations,
			Volumes: append([]corev1.Volume{
				{
					Name: label.RestoreJobLabelVal,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: restore.GetRestorePVCName(),
						},
					},
				},
			}, volumes...),
		},
	}

	if restore.Spec.ImagePullSecrets != nil {
		podSpec.Spec.ImagePullSecrets = restore.Spec.ImagePullSecrets
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      restore.GetRestoreJobName(),
			Namespace: ns,
			Labels:    restoreLabel,
			OwnerReferences: []metav1.OwnerReference{
				controller.GetRestoreOwnerRef(restore),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: pointer.Int32Ptr(0),
			Template:     *podSpec,
		},
	}
	return job, "", nil
}

func (rm *restoreManager) makeRestoreJob(restore *v1alpha1.Restore) (*batchv1.Job, string, error) {
	ns := restore.GetNamespace()
	name := restore.GetName()
	restoreNamespace := ns
	if restore.Spec.BR.ClusterNamespace != "" {
		restoreNamespace = restore.Spec.BR.ClusterNamespace
	}
	tc, err := rm.tcLister.TidbClusters(restoreNamespace).Get(restore.Spec.BR.Cluster)
	if err != nil {
		return nil, fmt.Sprintf("failed to fetch tidbcluster %s/%s", restoreNamespace, restore.Spec.BR.Cluster), err
	}

	envVars, reason, err := backuputil.GenerateTidbPasswordEnv(ns, name, restore.Spec.To.SecretName, restore.Spec.UseKMS, rm.kubeCli)
	if err != nil {
		return nil, reason, err
	}

	storageEnv, reason, err := backuputil.GenerateStorageCertEnv(ns, restore.Spec.UseKMS, restore.Spec.StorageProvider, rm.kubeCli)
	if err != nil {
		return nil, reason, fmt.Errorf("restore %s/%s, %v", ns, name, err)
	}

	envVars = append(envVars, storageEnv...)
	envVars = append(envVars, corev1.EnvVar{
		Name:  "BR_LOG_TO_TERM",
		Value: string(1),
	})
	args := []string{
		"restore",
		fmt.Sprintf("--namespace=%s", ns),
		fmt.Sprintf("--restoreName=%s", name),
	}
	tikvImage := tc.TiKVImage()
	_, tikvVersion := backuputil.ParseImage(tikvImage)
	if tikvVersion != "" {
		args = append(args, fmt.Sprintf("--tikvVersion=%s", tikvVersion))
	}

	restoreLabel := label.NewBackup().Instance(restore.GetInstanceName()).RestoreJob().Restore(name)
	volumeMounts := []corev1.VolumeMount{}
	volumes := []corev1.Volume{}
	if tc.IsTLSClusterEnabled() {
		args = append(args, "--cluster-tls=true")
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      util.ClusterClientVolName,
			ReadOnly:  true,
			MountPath: util.ClusterClientTLSPath,
		})
		volumes = append(volumes, corev1.Volume{
			Name: util.ClusterClientVolName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterClientTLSSecretName(restore.Spec.BR.Cluster),
				},
			},
		})
	}

	if tc.Spec.TiDB.TLSClient != nil && tc.Spec.TiDB.TLSClient.Enabled && !tc.SkipTLSWhenConnectTiDB() {
		args = append(args, "--client-tls=true")
		clientSecretName := util.TiDBClientTLSSecretName(restore.Spec.BR.Cluster)
		if restore.Spec.To.TLSClientSecretName != nil {
			clientSecretName = *restore.Spec.To.TLSClientSecretName
		}
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "tidb-client-tls",
			ReadOnly:  true,
			MountPath: util.TiDBClientTLSPath,
		})
		volumes = append(volumes, corev1.Volume{
			Name: "tidb-client-tls",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: clientSecretName,
				},
			},
		})
	}

	serviceAccount := constants.DefaultServiceAccountName
	if restore.Spec.ServiceAccount != "" {
		serviceAccount = restore.Spec.ServiceAccount
	}
	// TODO: need add ResourceRequirement for restore job
	podSpec := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      restoreLabel.Labels(),
			Annotations: restore.Annotations,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: serviceAccount,
			Containers: []corev1.Container{
				{
					Name:            label.RestoreJobLabelVal,
					Image:           controller.TidbBackupManagerImage,
					Args:            args,
					ImagePullPolicy: corev1.PullIfNotPresent,
					VolumeMounts:    volumeMounts,
					Env:             util.AppendEnvIfPresent(envVars, "TZ"),
					Resources:       restore.Spec.ResourceRequirements,
				},
			},
			Volumes:       volumes,
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	if restore.Spec.ImagePullSecrets != nil {
		podSpec.Spec.ImagePullSecrets = restore.Spec.ImagePullSecrets
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      restore.GetRestoreJobName(),
			Namespace: ns,
			Labels:    restoreLabel,
			OwnerReferences: []metav1.OwnerReference{
				controller.GetRestoreOwnerRef(restore),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: pointer.Int32Ptr(0),
			Template:     *podSpec,
		},
	}
	return job, "", nil
}

func (rm *restoreManager) ensureRestorePVCExist(restore *v1alpha1.Restore) (string, error) {
	ns := restore.GetNamespace()
	name := restore.GetName()

	storageSize := constants.DefaultStorageSize
	if restore.Spec.StorageSize != "" {
		storageSize = restore.Spec.StorageSize
	}
	rs, err := resource.ParseQuantity(storageSize)
	if err != nil {
		errMsg := fmt.Errorf("backup %s/%s parse storage size %s failed, err: %v", ns, name, constants.DefaultStorageSize, err)
		return "ParseStorageSizeFailed", errMsg
	}

	restorePVCName := restore.GetRestorePVCName()
	pvc, err := rm.pvcLister.PersistentVolumeClaims(ns).Get(restorePVCName)
	if err != nil {
		// get the object from the local cache, the error can only be IsNotFound,
		// so we need to create PVC for restore job
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      restorePVCName,
				Namespace: ns,
				Labels:    label.NewRestore().Instance(restore.GetInstanceName()),
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: rs,
					},
				},
				StorageClassName: restore.Spec.StorageClassName,
			},
		}
		if err := rm.pvcControl.CreatePVC(restore, pvc); err != nil {
			errMsg := fmt.Errorf(" %s/%s create restore pvc %s failed, err: %v", ns, name, pvc.GetName(), err)
			return "CreatePVCFailed", errMsg
		}
	} else if pvcRs := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; pvcRs.Cmp(rs) == -1 {
		return "PVCStorageSizeTooSmall", fmt.Errorf("%s/%s's restore pvc %s's storage size %s is less than expected storage size %s, please delete old pvc to continue", ns, name, pvc.GetName(), pvcRs.String(), rs.String())
	}
	return "", nil
}

var _ backup.RestoreManager = &restoreManager{}

type FakeRestoreManager struct {
	err error
}

func NewFakeRestoreManager() *FakeRestoreManager {
	return &FakeRestoreManager{}
}

func (frm *FakeRestoreManager) SetSyncError(err error) {
	frm.err = err
}

func (frm *FakeRestoreManager) Sync(_ *v1alpha1.Restore) error {
	return frm.err
}

var _ backup.RestoreManager = &FakeRestoreManager{}
