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
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batchlisters "k8s.io/client-go/listers/batch/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

type restoreManager struct {
	backupLister  listers.BackupLister
	statusUpdater controller.RestoreConditionUpdaterInterface
	secretLister  corelisters.SecretLister
	jobLister     batchlisters.JobLister
	jobControl    controller.JobControlInterface
	pvcLister     corelisters.PersistentVolumeClaimLister
	pvcControl    controller.GeneralPVCControlInterface
}

// NewRestoreManager return restoreManager
func NewRestoreManager(
	backupLister listers.BackupLister,
	statusUpdater controller.RestoreConditionUpdaterInterface,
	secretLister corelisters.SecretLister,
	jobLister batchlisters.JobLister,
	jobControl controller.JobControlInterface,
	pvcLister corelisters.PersistentVolumeClaimLister,
	pvcControl controller.GeneralPVCControlInterface,
) backup.RestoreManager {
	return &restoreManager{
		backupLister,
		statusUpdater,
		secretLister,
		jobLister,
		jobControl,
		pvcLister,
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

	envVars, reason, err := backuputil.GenerateTidbPasswordEnv(ns, name, restore.Spec.To.SecretName, rm.secretLister)
	if err != nil {
		return nil, reason, err
	}

	storageEnv, reason, err := backuputil.GenerateStorageCertEnv(ns, restore.Spec.StorageProvider, rm.secretLister)
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
		fmt.Sprintf("--host=%s", restore.Spec.To.Host),
		fmt.Sprintf("--port=%d", restore.Spec.To.Port),
		fmt.Sprintf("--user=%s", restore.Spec.To.User),
		fmt.Sprintf("--backupPath=%s", backupPath),
	}

	restoreLabel := label.NewBackup().Instance(restore.GetInstanceName()).RestoreJob().Restore(name)

	// TODO: need add ResourceRequirement for restore job
	podSpec := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: restoreLabel.Labels(),
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: constants.DefaultServiceAccountName,
			Containers: []corev1.Container{
				{
					Name:            label.RestoreJobLabelVal,
					Image:           controller.TidbBackupManagerImage,
					Args:            args,
					ImagePullPolicy: corev1.PullAlways,
					VolumeMounts: []corev1.VolumeMount{
						{Name: label.RestoreJobLabelVal, MountPath: constants.BackupRootPath},
					},
					Env: envVars,
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Volumes: []corev1.Volume{
				{
					Name: label.RestoreJobLabelVal,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: restore.GetRestorePVCName(),
						},
					},
				},
			},
		},
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
			BackoffLimit: controller.Int32Ptr(0),
			Template:     *podSpec,
		},
	}
	return job, "", nil
}

func (rm *restoreManager) makeRestoreJob(restore *v1alpha1.Restore) (*batchv1.Job, string, error) {
	ns := restore.GetNamespace()
	name := restore.GetName()

	envVars, reason, err := backuputil.GenerateStorageCertEnv(ns, restore.Spec.StorageProvider, rm.secretLister)
	if err != nil {
		return nil, reason, fmt.Errorf("restore %s/%s, %v", ns, name, err)
	}

	args := []string{
		"restore",
		fmt.Sprintf("--namespace=%s", ns),
		fmt.Sprintf("--restoreName=%s", name),
	}

	restoreLabel := label.NewBackup().Instance(restore.GetInstanceName()).RestoreJob().Restore(name)

	// TODO: need add ResourceRequirement for restore job
	podSpec := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: restoreLabel.Labels(),
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: constants.DefaultServiceAccountName,
			Containers: []corev1.Container{
				{
					Name:            label.RestoreJobLabelVal,
					Image:           controller.TidbBackupManagerImage,
					Args:            args,
					ImagePullPolicy: corev1.PullAlways,
					Env:             envVars,
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
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
			BackoffLimit: controller.Int32Ptr(0),
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
	_, err = rm.pvcLister.PersistentVolumeClaims(ns).Get(restorePVCName)
	if err != nil {
		// get the object from the local cache, the error can only be IsNotFound,
		// so we need to create PVC for restore job
		storageClassName := controller.DefaultBackupStorageClassName
		if restore.Spec.StorageClassName != "" {
			storageClassName = restore.Spec.StorageClassName
		}
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      restorePVCName,
				Namespace: ns,
				Labels:    label.NewRestore().Instance(restore.GetInstanceName()),
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				StorageClassName: &storageClassName,
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: rs,
					},
				},
			},
		}
		if err := rm.pvcControl.CreatePVC(restore, pvc); err != nil {
			errMsg := fmt.Errorf(" %s/%s create restore pvc %s failed, err: %v", ns, name, pvc.GetName(), err)
			return "CreatePVCFailed", errMsg
		}
	}
	return "", nil
}

var _ backup.RestoreManager = &restoreManager{}
