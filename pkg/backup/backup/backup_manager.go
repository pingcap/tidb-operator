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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	batchlisters "k8s.io/client-go/listers/batch/v1"
	corelisters "k8s.io/client-go/listers/core/v1"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	backuputil "github.com/pingcap/tidb-operator/pkg/backup/util"
	v1alpha1listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
)

type backupManager struct {
	backupCleaner BackupCleaner
	statusUpdater controller.BackupConditionUpdaterInterface
	kubeCli       kubernetes.Interface
	jobLister     batchlisters.JobLister
	jobControl    controller.JobControlInterface
	pvcLister     corelisters.PersistentVolumeClaimLister
	tcLister      v1alpha1listers.TidbClusterLister
	pvcControl    controller.GeneralPVCControlInterface
}

// NewBackupManager return backupManager
func NewBackupManager(
	backupCleaner BackupCleaner,
	statusUpdater controller.BackupConditionUpdaterInterface,
	kubeCli kubernetes.Interface,
	jobLister batchlisters.JobLister,
	jobControl controller.JobControlInterface,
	pvcLister corelisters.PersistentVolumeClaimLister,
	tcLister v1alpha1listers.TidbClusterLister,
	pvcControl controller.GeneralPVCControlInterface,
) backup.BackupManager {
	return &backupManager{
		backupCleaner,
		statusUpdater,
		kubeCli,
		jobLister,
		jobControl,
		pvcLister,
		tcLister,
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

	err := backuputil.ValidateBackup(backup)
	if err != nil {
		bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupInvalid,
			Status:  corev1.ConditionTrue,
			Reason:  "InvalidSpec",
			Message: err.Error(),
		})

		return controller.IgnoreErrorf("invalid backup spec %s/%s", ns, name)
	}

	_, err = bm.jobLister.Jobs(ns).Get(backupJobName)
	if err == nil {
		// already have a backup job running，return directly
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("backup %s/%s get job %s failed, err: %v", ns, name, backupJobName, err)
	}

	var job *batchv1.Job
	var reason string
	if backup.Spec.BR == nil {
		// not found backup job, so we need to create it
		job, reason, err = bm.makeExportJob(backup)
		if err != nil {
			bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Type:    v1alpha1.BackupRetryFailed,
				Status:  corev1.ConditionTrue,
				Reason:  reason,
				Message: err.Error(),
			})
			return err
		}

		reason, err = bm.ensureBackupPVCExist(backup)
		if err != nil {
			bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Type:    v1alpha1.BackupRetryFailed,
				Status:  corev1.ConditionTrue,
				Reason:  reason,
				Message: err.Error(),
			})
			return err
		}

	} else {
		// not found backup job, so we need to create it
		job, reason, err = bm.makeBackupJob(backup)
		if err != nil {
			bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Type:    v1alpha1.BackupRetryFailed,
				Status:  corev1.ConditionTrue,
				Reason:  reason,
				Message: err.Error(),
			})
			return err
		}
	}

	if err := bm.jobControl.CreateJob(backup, job); err != nil {
		errMsg := fmt.Errorf("create backup %s/%s job %s failed, err: %v", ns, name, backupJobName, err)
		bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupRetryFailed,
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

func (bm *backupManager) makeExportJob(backup *v1alpha1.Backup) (*batchv1.Job, string, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()

	envVars, reason, err := backuputil.GenerateTidbPasswordEnv(ns, name, backup.Spec.From.SecretName, backup.Spec.UseKMS, bm.kubeCli)
	if err != nil {
		return nil, reason, err
	}

	storageEnv, reason, err := backuputil.GenerateStorageCertEnv(ns, backup.Spec.UseKMS, backup.Spec.StorageProvider, bm.kubeCli)
	if err != nil {
		return nil, reason, fmt.Errorf("backup %s/%s, %v", ns, name, err)
	}
	envVars = append(envVars, storageEnv...)
	// TODO: make pvc request storage size configurable
	reason, err = bm.ensureBackupPVCExist(backup)
	if err != nil {
		return nil, reason, err
	}

	bucketName, reason, err := backuputil.GetBackupBucketName(backup)
	if err != nil {
		return nil, reason, err
	}

	args := []string{
		"export",
		fmt.Sprintf("--namespace=%s", ns),
		fmt.Sprintf("--backupName=%s", name),
		fmt.Sprintf("--bucket=%s", bucketName),
		fmt.Sprintf("--storageType=%s", backuputil.GetStorageType(backup.Spec.StorageProvider)),
	}
	if backup.Spec.S3 != nil && backup.Spec.S3.Prefix != "" {
		args = append(args,
			fmt.Sprintf("--prefix=%s", backup.Spec.S3.Prefix),
		)
	}

	serviceAccount := constants.DefaultServiceAccountName
	if backup.Spec.ServiceAccount != "" {
		serviceAccount = backup.Spec.ServiceAccount
	}
	backupLabel := label.NewBackup().Instance(backup.GetInstanceName()).BackupJob().Backup(name)
	// TODO: need add ResourceRequirement for backup job
	podSpec := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      backupLabel.Labels(),
			Annotations: backup.Annotations,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: serviceAccount,
			Containers: []corev1.Container{
				{
					Name:            label.BackupJobLabelVal,
					Image:           controller.TidbBackupManagerImage,
					Args:            args,
					ImagePullPolicy: corev1.PullIfNotPresent,
					VolumeMounts: []corev1.VolumeMount{
						{Name: label.BackupJobLabelVal, MountPath: constants.BackupRootPath},
					},
					Env: envVars,
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Affinity:      backup.Spec.Affinity,
			Tolerations:   backup.Spec.Tolerations,
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

func (bm *backupManager) makeBackupJob(backup *v1alpha1.Backup) (*batchv1.Job, string, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()
	backupNamespace := ns
	if backup.Spec.BR.ClusterNamespace != "" {
		backupNamespace = backup.Spec.BR.ClusterNamespace
	}
	tc, err := bm.tcLister.TidbClusters(backupNamespace).Get(backup.Spec.BR.Cluster)
	if err != nil {
		return nil, fmt.Sprintf("failed to fetch tidbcluster %s/%s", backupNamespace, backup.Spec.BR.Cluster), err
	}

	envVars, reason, err := backuputil.GenerateTidbPasswordEnv(ns, name, backup.Spec.From.SecretName, backup.Spec.UseKMS, bm.kubeCli)
	if err != nil {
		return nil, reason, err
	}

	storageEnv, reason, err := backuputil.GenerateStorageCertEnv(ns, backup.Spec.UseKMS, backup.Spec.StorageProvider, bm.kubeCli)
	if err != nil {
		return nil, reason, fmt.Errorf("backup %s/%s, %v", ns, name, err)
	}

	envVars = append(envVars, storageEnv...)
	envVars = append(envVars, corev1.EnvVar{
		Name:  "BR_LOG_TO_TERM",
		Value: string(1),
	})

	args := []string{
		"backup",
		fmt.Sprintf("--namespace=%s", ns),
		fmt.Sprintf("--backupName=%s", name),
	}
	tikvImage := tc.TiKVImage()
	_, tikvVersion := backuputil.ParseImage(tikvImage)
	if tikvVersion != "" {
		args = append(args, fmt.Sprintf("--tikvVersion=%s", tikvVersion))
	}

	backupLabel := label.NewBackup().Instance(backup.GetInstanceName()).BackupJob().Backup(name)
	volumeMounts := []corev1.VolumeMount{}
	volumes := []corev1.Volume{}
	if tc.Spec.TLSCluster != nil && tc.Spec.TLSCluster.Enabled {
		args = append(args, "--cluster-tls=true")
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "cluster-client-tls",
			ReadOnly:  true,
			MountPath: util.ClusterClientTLSPath,
		})
		volumes = append(volumes, corev1.Volume{
			Name: "cluster-client-tls",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.ClusterClientTLSSecretName(backup.Spec.BR.Cluster),
				},
			},
		})
	}
	if tc.Spec.TiDB.TLSClient != nil && tc.Spec.TiDB.TLSClient.Enabled && !tc.SkipTLSWhenConnectTiDB() {
		args = append(args, "--client-tls=true")
		clientSecretName := util.TiDBClientTLSSecretName(backup.Spec.BR.Cluster)
		if backup.Spec.From.TLSClient != nil && backup.Spec.From.TLSClient.TLSSecret != "" {
			clientSecretName = backup.Spec.From.TLSClient.TLSSecret
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
	if backup.Spec.ServiceAccount != "" {
		serviceAccount = backup.Spec.ServiceAccount
	}
	podSpec := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      backupLabel.Labels(),
			Annotations: backup.Annotations,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: serviceAccount,
			Containers: []corev1.Container{
				{
					Name:            label.BackupJobLabelVal,
					Image:           controller.TidbBackupManagerImage,
					Args:            args,
					ImagePullPolicy: corev1.PullIfNotPresent,
					VolumeMounts:    volumeMounts,
					Env:             envVars,
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Affinity:      backup.Spec.Affinity,
			Tolerations:   backup.Spec.Tolerations,
			Volumes:       volumes,
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
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupPVCName,
			Namespace: ns,
			Labels:    label.NewBackup().Instance(backup.GetInstanceName()),
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
			StorageClassName: backup.Spec.StorageClassName,
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
