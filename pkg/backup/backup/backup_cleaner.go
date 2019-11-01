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
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	backuputil "github.com/pingcap/tidb-operator/pkg/backup/util"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batchlisters "k8s.io/client-go/listers/batch/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	glog "k8s.io/klog"
)

// BackupCleaner implements the logic for cleaning backup
type BackupCleaner interface {
	Clean(backup *v1alpha1.Backup) error
}

type backupCleaner struct {
	statusUpdater controller.BackupConditionUpdaterInterface
	secretLister  corelisters.SecretLister
	jobLister     batchlisters.JobLister
	jobControl    controller.JobControlInterface
}

// NewBackupCleaner returns a BackupCleaner
func NewBackupCleaner(
	statusUpdater controller.BackupConditionUpdaterInterface,
	secretLister corelisters.SecretLister,
	jobLister batchlisters.JobLister,
	jobControl controller.JobControlInterface) BackupCleaner {
	return &backupCleaner{
		statusUpdater,
		secretLister,
		jobLister,
		jobControl,
	}
}

func (bc *backupCleaner) Clean(backup *v1alpha1.Backup) error {
	if backup.DeletionTimestamp == nil {
		// The backup object has not been deleted，do nothing
		return nil
	}
	ns := backup.GetNamespace()
	name := backup.GetName()

	glog.Infof("start to clean backup %s/%s", ns, name)

	cleanJobName := backup.GetCleanJobName()
	_, err := bc.jobLister.Jobs(ns).Get(cleanJobName)
	if err == nil {
		// already have a clean job running，return directly
		return nil
	}

	if backup.Status.BackupPath == "" {
		// the backup path is empty, so there is no need to clean up backup data
		return bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:   v1alpha1.BackupClean,
			Status: corev1.ConditionTrue,
		})
	}
	// not found clean job, create it
	job, reason, err := bc.makeCleanJob(backup)
	if err != nil {
		bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupRetryFailed,
			Status:  corev1.ConditionTrue,
			Reason:  reason,
			Message: err.Error(),
		})
		return err
	}

	if err := bc.jobControl.CreateJob(backup, job); err != nil {
		errMsg := fmt.Errorf("create backup %s/%s job %s failed, err: %v", ns, name, cleanJobName, err)
		bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupRetryFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "CreateCleanJobFailed",
			Message: errMsg.Error(),
		})
		return errMsg
	}

	return bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupClean,
		Status: corev1.ConditionFalse,
	})
}

func (bc *backupCleaner) makeCleanJob(backup *v1alpha1.Backup) (*batchv1.Job, string, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()

	storageEnv, reason, err := backuputil.GenerateStorageCertEnv(backup, bc.secretLister)
	if err != nil {
		return nil, reason, err
	}

	args := []string{
		"clean",
		fmt.Sprintf("--namespace=%s", ns),
		fmt.Sprintf("--tidbcluster=%s", backup.Spec.Cluster),
		fmt.Sprintf("--backupName=%s", name),
	}

	backupLabel := label.NewBackup().Instance(backup.Spec.Cluster).CleanJob().Backup(name)

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
					Env:             storageEnv,
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backup.GetCleanJobName(),
			Namespace: ns,
			Labels:    backupLabel,
			OwnerReferences: []metav1.OwnerReference{
				controller.GetBackupOwnerRef(backup),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: controller.Int32Ptr(constants.DefaultBackoffLimit),
			Template:     *podSpec,
		},
	}
	return job, "", nil
}

var _ BackupCleaner = &backupCleaner{}
