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

	_, err := rm.jobLister.Jobs(ns).Get(restoreJobName)
	if err == nil {
		// already have a backup job running，return directly
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("restore %s/%s get job %s failed, err: %v", ns, name, restoreJobName, err)
	}

	// not found restore job, need to create it
	backup, reason, err := rm.getBackupFromRestore(restore)
	if err != nil {
		rm.statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  reason,
			Message: err.Error(),
		})
		return err
	}

	job, reason, err := rm.makeRestoreJob(restore, backup)
	if err != nil {
		rm.statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  reason,
			Message: err.Error(),
		})
		return err
	}

	reason, err = rm.ensureRestorePVCExist(restore)
	if err != nil {
		rm.statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  reason,
			Message: err.Error(),
		})
		return err
	}

	if err := rm.jobControl.CreateJob(restore, job); err != nil {
		errMsg := fmt.Errorf("create restore %s/%s job %s failed, err: %v", ns, name, restoreJobName, err)
		rm.statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
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

func (rm *restoreManager) getBackupFromRestore(restore *v1alpha1.Restore) (*v1alpha1.Backup, string, error) {
	backupNs := restore.Spec.BackupNamespace
	ns := restore.GetNamespace()
	name := restore.GetName()

	backup, err := rm.backupLister.Backups(backupNs).Get(restore.Spec.Backup)
	if err != nil {
		errMsg := fmt.Errorf("restore %s/%s get backup %s/%s failed, err: %v", ns, name, backupNs, restore.Spec.Backup, err)
		return nil, "BackupNotFound", errMsg
	}
	if backup.Status.BackupPath == "" {
		errMsg := fmt.Errorf("restore %s/%s backup %s/%s backupPath is empty", ns, name, backupNs, restore.Spec.Backup)
		return nil, "BackupPathIsEmpty", errMsg
	}

	return backup, "", nil
}

func (rm *restoreManager) makeRestoreJob(restore *v1alpha1.Restore, backup *v1alpha1.Backup) (*batchv1.Job, string, error) {
	ns := restore.GetNamespace()
	name := restore.GetName()

	user, password, reason, err := backuputil.GetTidbUserAndPassword(ns, name, restore.Spec.TidbSecretName, rm.secretLister)
	if err != nil {
		return nil, reason, err
	}

	storageEnv, reason, err := backuputil.GenerateStorageCertEnv(backup, rm.secretLister)
	if err != nil {
		return nil, reason, err
	}

	args := []string{
		"restore",
		fmt.Sprintf("--namespace=%s", ns),
		fmt.Sprintf("--restoreName=%s", name),
		fmt.Sprintf("--tidbcluster=%s", restore.Spec.Cluster),
		fmt.Sprintf("--backupPath=%s", backup.Status.BackupPath),
		fmt.Sprintf("--backupName=%s", backup.GetName()),
		fmt.Sprintf("--tidbservice=%s", controller.TiDBMemberName(restore.Spec.Cluster)),
		fmt.Sprintf("--password=%s", password),
		fmt.Sprintf("--user=%s", user),
	}

	restoreLabel := label.NewBackup().Instance(restore.Spec.Cluster).RestoreJob().Restore(name)

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
					Env: storageEnv,
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
				Labels:    label.NewRestore().Instance(restore.Spec.Cluster),
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
