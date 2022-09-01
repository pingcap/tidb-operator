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

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	backuputil "github.com/pingcap/tidb-operator/pkg/backup/util"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
)

var _ BackupCleaner = &backupCleaner{}

// BackupCleaner implements the logic for cleaning backup
type BackupCleaner interface {
	Clean(backup *v1alpha1.Backup) error
}

type backupCleaner struct {
	deps          *controller.Dependencies
	statusUpdater controller.BackupConditionUpdaterInterface
}

// NewBackupCleaner returns a BackupCleaner
func NewBackupCleaner(deps *controller.Dependencies, statusUpdater controller.BackupConditionUpdaterInterface) BackupCleaner {
	return &backupCleaner{
		deps:          deps,
		statusUpdater: statusUpdater,
	}
}

func (bc *backupCleaner) Clean(backup *v1alpha1.Backup) error {
	if backup.DeletionTimestamp == nil || !v1alpha1.IsCleanCandidate(backup) || v1alpha1.NeedNotClean(backup) {
		// The backup object has not been deleted or we need to retain backup data，do nothing
		return nil
	}
	ns := backup.GetNamespace()
	name := backup.GetName()

	klog.Infof("start to ensure that backup %s/%s jobs have finished", ns, name)

	finished, err := bc.ensureBackupJobFinished(backup)
	if err != nil {
		return fmt.Errorf("ensure %s/%s jobs finished failed: %s", ns, name, err)
	}
	if !finished {
		klog.Infof("wait for backup %s/%s jobs to finish", ns, name)
		return nil
	}

	klog.Infof("start to clean backup %s/%s", ns, name)

	cleanJobName := backup.GetCleanJobName()
	_, err = bc.deps.JobLister.Jobs(ns).Get(cleanJobName)
	if err == nil {
		// already have a clean job running，return directly
		return nil
	} else if !errors.IsNotFound(err) {
		bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupRetryFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetBackupFailed",
			Message: err.Error(),
		}, nil)
		return err
	}

	// no found the clean job, we start to create the clean job.
	if backup.Status.BackupPath == "" {
		// the backup path is empty, so there is no need to clean up backup data
		return bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:   v1alpha1.BackupClean,
			Status: corev1.ConditionTrue,
		}, nil)
	}

	// not found clean job, create it
	job, reason, err := bc.makeCleanJob(backup)
	if err != nil {
		bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupRetryFailed,
			Status:  corev1.ConditionTrue,
			Reason:  reason,
			Message: err.Error(),
		}, nil)
		return err
	}

	if err := bc.deps.JobControl.CreateJob(backup, job); err != nil {
		errMsg := fmt.Errorf("create backup %s/%s job %s failed, err: %v", ns, name, cleanJobName, err)
		bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupRetryFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "CreateCleanJobFailed",
			Message: errMsg.Error(),
		}, nil)
		return errMsg
	}

	return bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupClean,
		Status: corev1.ConditionFalse,
	}, nil)
}

func (bc *backupCleaner) makeCleanJob(backup *v1alpha1.Backup) (*batchv1.Job, string, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()

	envVars, reason, err := backuputil.GenerateStorageCertEnv(ns, backup.Spec.UseKMS, backup.Spec.StorageProvider, bc.deps.SecretLister)
	if err != nil {
		return nil, reason, err
	}

	// set env vars specified in backup.Spec.Env
	envVars = util.AppendOverwriteEnv(envVars, backup.Spec.Env)

	args := []string{
		"clean",
		fmt.Sprintf("--namespace=%s", ns),
		fmt.Sprintf("--backupName=%s", name),
	}

	var volumes []corev1.Volume
	var volumeMounts []corev1.VolumeMount

	// mount volumes if specified
	if backup.Spec.Local != nil {
		klog.Info("mounting local volumes in Backup.Spec")
		localVolume := backup.Spec.Local.Volume
		localVolumeMount := backup.Spec.Local.VolumeMount
		volumes = append(volumes, localVolume)
		volumeMounts = append(volumeMounts, localVolumeMount)
	}

	serviceAccount := constants.DefaultServiceAccountName
	if backup.Spec.ServiceAccount != "" {
		serviceAccount = backup.Spec.ServiceAccount
	}

	backupLabel := label.NewBackup().Instance(backup.GetInstanceName()).CleanJob().Backup(name)
	jobLabels := util.CombineStringMap(backupLabel, backup.Labels)
	podLabels := jobLabels

	podSpec := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      podLabels,
			Annotations: backup.Annotations,
		},
		Spec: corev1.PodSpec{
			SecurityContext:    backup.Spec.PodSecurityContext,
			ServiceAccountName: serviceAccount,
			Containers: []corev1.Container{
				{
					Name:            label.BackupJobLabelVal,
					Image:           bc.deps.CLIConfig.TiDBBackupManagerImage,
					Args:            args,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Env:             util.AppendEnvIfPresent(envVars, "TZ"),
					Resources:       backup.Spec.ResourceRequirements,
					VolumeMounts:    volumeMounts,
				},
			},
			RestartPolicy:     corev1.RestartPolicyNever,
			Tolerations:       backup.Spec.Tolerations,
			ImagePullSecrets:  backup.Spec.ImagePullSecrets,
			Affinity:          backup.Spec.Affinity,
			Volumes:           volumes,
			PriorityClassName: backup.Spec.PriorityClassName,
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        backup.GetCleanJobName(),
			Namespace:   ns,
			Labels:      jobLabels,
			Annotations: backup.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				controller.GetBackupOwnerRef(backup),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: pointer.Int32Ptr(constants.DefaultBackoffLimit),
			Template:     *podSpec,
		},
	}

	return job, "", nil
}

// ensureBackupJobDeleted ensure that backup Job have finished, it will delete the job if it is running
func (bc *backupCleaner) ensureBackupJobFinished(backup *v1alpha1.Backup) (bool, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()

	// log backup may have multiple jobs
	backupJobNames := make([]string, 0)
	if backup.Spec.Mode == v1alpha1.BackupModeLog {
		backupJobNames = append(backupJobNames, backup.GetAllLogBackupJobName()...)
	} else {
		backupJobNames = append(backupJobNames, backup.GetBackupJobName())
	}

	isAllFinished := true
	errs := make([]error, 0)
	for _, backupJobName := range backupJobNames {
		backupJob, err := bc.deps.JobLister.Jobs(ns).Get(backupJobName)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			errs = append(errs, err)
			continue
		}
		// job is being deleted
		if backupJob.DeletionTimestamp != nil {
			klog.Infof("backup %s/%s job %s is being deleted, cleaner will wait", ns, name, backupJobName)
			isAllFinished = false
			continue
		}

		finished := false
		for _, c := range backupJob.Status.Conditions {
			if (c.Type == batchv1.JobComplete || c.Type == batchv1.JobFailed) && c.Status == corev1.ConditionTrue {
				finished = true
				break
			}
		}
		if finished {
			continue
		}

		isAllFinished = false
		klog.Infof("backup %s/%s job %s is running, cleaner will delete it and wait it done", ns, name, backupJobName)
		// delete job if job is running
		err = bc.deps.JobControl.DeleteJob(backup, backupJob)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return false, errorutils.NewAggregate(errs)
	}
	// no job or all job finished
	return isAllFinished, nil
}
