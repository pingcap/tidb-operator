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

package backup

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	backuputil "github.com/pingcap/tidb-operator/pkg/backup/util"
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

type backupManager struct {
	backupCleaner BackupCleaner
	statusUpdater controller.BackupConditionUpdaterInterface
	secretLister  corelisters.SecretLister
	jobLister     batchlisters.JobLister
	jobControl    controller.JobControlInterface
	pvcLister     corelisters.PersistentVolumeClaimLister
	pvcControl    controller.GeneralPVCControlInterface
}

// NewBackupManager return backupManager
func NewBackupManager(
	backupCleaner BackupCleaner,
	statusUpdater controller.BackupConditionUpdaterInterface,
	secretLister corelisters.SecretLister,
	jobLister batchlisters.JobLister,
	jobControl controller.JobControlInterface,
	pvcLister corelisters.PersistentVolumeClaimLister,
	pvcControl controller.GeneralPVCControlInterface,
) backup.BackupManager {
	return &backupManager{
		backupCleaner,
		statusUpdater,
		secretLister,
		jobLister,
		jobControl,
		pvcLister,
		pvcControl,
	}
}

func (bm *backupManager) Sync(backup *v1alpha1.Backup) error {
	if err := bm.backupCleaner.Clean(backup); err != nil {
		return err
	}

	if backup.DeletionTimestamp != nil {
		// backup is being deleted, don't do anything, return directly.
		return nil
	}

	return bm.syncBackupJob(backup)
}

func (bm *backupManager) syncBackupJob(backup *v1alpha1.Backup) error {
	ns := backup.GetNamespace()
	name := backup.GetName()
	backupJobName := backup.GetBackupJobName()

	_, err := bm.jobLister.Jobs(ns).Get(backupJobName)
	if err == nil {
		// already have a backup job running，return directly
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("backup %s/%s get job %s failed, err: %v", ns, name, backupJobName, err)
	}

	// not found backup job, so we need to create it
	job, reason, err := bm.makeBackupJob(backup)
	if err != nil {
		bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  reason,
			Message: err.Error(),
		})
		return err
	}

	reason, err = bm.ensureBackupPVCExist(backup)
	if err != nil {
		bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  reason,
			Message: err.Error(),
		})
		return err
	}

	if err := bm.jobControl.CreateJob(backup, job); err != nil {
		errMsg := fmt.Errorf("create backup %s/%s job %s failed, err: %v", ns, name, backupJobName, err)
		bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "CreateBackupJobFailed",
			Message: errMsg.Error(),
		})
		return errMsg
	}

	return bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupScheduled,
		Status: corev1.ConditionTrue,
	})
}

func (bm *backupManager) makeBackupJob(backup *v1alpha1.Backup) (*batchv1.Job, string, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()

	user, password, reason, err := backuputil.GetTidbUserAndPassword(ns, name, backup.Spec.TidbSecretName, bm.secretLister)
	if err != nil {
		return nil, reason, err
	}

	storageEnv, reason, err := backuputil.GenerateStorageCertEnv(backup, bm.secretLister)
	if err != nil {
		return nil, reason, err
	}

	// TODO: make pvc request storage size configurable
	reason, err = bm.ensureBackupPVCExist(backup)
	if err != nil {
		return nil, reason, err
	}

	args := []string{
		"backup",
		fmt.Sprintf("--namespace=%s", ns),
		fmt.Sprintf("--tidbcluster=%s", backup.Spec.Cluster),
		fmt.Sprintf("--backupName=%s", name),
		fmt.Sprintf("--tidbservice=%s", controller.TiDBMemberName(backup.Spec.Cluster)),
		fmt.Sprintf("--storageType=%s", backup.Spec.StorageType),
		fmt.Sprintf("--password=%s", password),
		fmt.Sprintf("--user=%s", user),
	}

	backupLabel := label.NewBackup().Instance(backup.Spec.Cluster).BackupJob().Backup(name)

	// TODO: need add ResourceRequirement for backup job
	podSpec := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: backupLabel.Labels(),
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: constants.DefaultServiceAccountName,
			Containers: []corev1.Container{
				{
					Name:            label.BackupJobLabelVal,
					Image:           controller.TidbBackupManagerImage,
					Args:            args,
					ImagePullPolicy: corev1.PullAlways,
					VolumeMounts: []corev1.VolumeMount{
						{Name: label.BackupJobLabelVal, MountPath: constants.BackupRootPath},
					},
					Env: storageEnv,
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Volumes: []corev1.Volume{
				{
					Name: label.BackupJobLabelVal,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: backup.GetBackupPVCName(),
						},
					},
				},
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backup.GetBackupJobName(),
			Namespace: ns,
			Labels:    backupLabel,
			OwnerReferences: []metav1.OwnerReference{
				controller.GetBackupOwnerRef(backup),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: controller.Int32Ptr(0),
			Template:     *podSpec,
		},
	}

	return job, "", nil
}

func (bm *backupManager) ensureBackupPVCExist(backup *v1alpha1.Backup) (string, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()

	storageSize := constants.DefaultStorageSize
	if backup.Spec.StorageSize != "" {
		storageSize = backup.Spec.StorageSize
	}
	rs, err := resource.ParseQuantity(storageSize)
	if err != nil {
		errMsg := fmt.Errorf("backup %s/%s parse storage size %s failed, err: %v", ns, name, constants.DefaultStorageSize, err)
		return "ParseStorageSizeFailed", errMsg
	}
	backupPVCName := backup.GetBackupPVCName()
	_, err = bm.pvcLister.PersistentVolumeClaims(ns).Get(backupPVCName)

	if err == nil {
		return "", nil
	}

	if !errors.IsNotFound(err) {
		return "GetPVCFailed", fmt.Errorf("backup %s/%s get pvc %s failed, err: %v", ns, name, backupPVCName, err)
	}

	// not found PVC, so we need to create PVC for backup job
	storageClassName := controller.DefaultBackupStorageClassName
	if backup.Spec.StorageClassName != "" {
		storageClassName = backup.Spec.StorageClassName
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupPVCName,
			Namespace: ns,
			Labels:    label.NewBackup().Instance(backup.Spec.Cluster),
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

	if err := bm.pvcControl.CreatePVC(backup, pvc); err != nil {
		errMsg := fmt.Errorf("backup %s/%s create backup pvc %s failed, err: %v", ns, name, pvc.GetName(), err)
		return "CreatePVCFailed", errMsg
	}
	return "", nil
}

var _ backup.BackupManager = &backupManager{}

type FakeBackupManager struct {
	err error
}

func NewFakeBackupManager() *FakeBackupManager {
	return &FakeBackupManager{}
}

func (fbm *FakeBackupManager) SetSyncError(err error) {
	fbm.err = err
}

func (fbm *FakeBackupManager) Sync(_ *v1alpha1.Backup) error {
	return fbm.err
}

var _ backup.BackupManager = &FakeBackupManager{}
