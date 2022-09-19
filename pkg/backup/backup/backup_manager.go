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
	"strings"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	backuputil "github.com/pingcap/tidb-operator/pkg/backup/util"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
)

type backupManager struct {
	deps          *controller.Dependencies
	backupCleaner BackupCleaner
	backupTracker BackupTracker
	statusUpdater controller.BackupConditionUpdaterInterface
}

// NewBackupManager return backupManager
func NewBackupManager(deps *controller.Dependencies) backup.BackupManager {
	statusUpdater := controller.NewRealBackupConditionUpdater(deps.Clientset, deps.BackupLister, deps.Recorder)
	return &backupManager{
		deps:          deps,
		backupCleaner: NewBackupCleaner(deps, statusUpdater),
		backupTracker: NewBackupTracker(deps, statusUpdater),
		statusUpdater: statusUpdater,
	}
}

func (bm *backupManager) Sync(backup *v1alpha1.Backup) error {
	// because a finalizer is installed on the backup on creation, when backup is deleted,
	// backup.DeletionTimestamp will be set, controller will be informed with an onUpdate event,
	// this is the moment that we can do clean up work.
	if err := bm.backupCleaner.Clean(backup); err != nil {
		return err
	}

	if backup.DeletionTimestamp != nil {
		// backup is being deleted, don't do anything, return directly.
		return nil
	}

	return bm.syncBackupJob(backup)
}

func (bm *backupManager) UpdateCondition(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition) error {
	return bm.statusUpdater.Update(backup, condition, nil)
}

func (bm *backupManager) syncBackupJob(backup *v1alpha1.Backup) error {
	ns := backup.GetNamespace()
	name := backup.GetName()
	backupJobName := backup.GetBackupJobName()
	logBackupSubcommand := v1alpha1.ParseLogBackupSubcommand(backup)
	var err error

	// validate backup
	if err = bm.validateBackup(backup); err != nil {
		klog.Errorf("backup %s/%s validate error %v.", ns, name, err)
		return err
	}

	// skip backup
	skip := false
	if skip, err = bm.skipBackupSync(backup); err != nil {
		klog.Errorf("backup %s/%s skip error %v.", ns, name, err)
		return err
	}
	if skip {
		klog.Infof("backup %s/%s is already done and skip sync.", ns, name)
		return nil
	}

	// wait pre task done
	if err = bm.waitPreTaskDone(backup); err != nil {
		klog.Errorf("backup %s/%s wait pre task done error %v.", ns, name, err)
		return err
	}

	// make bakcup job
	var job *batchv1.Job
	var reason string
	var updateStatus *controller.BackupUpdateStatus
	if job, updateStatus, reason, err = bm.makeBackupJob(backup); err != nil {
		klog.Errorf("backup %s/%s create job %s failed, reason is %s, error %v.", ns, name, backupJobName, reason, err)
		return err
	}

	// create k8s job
	if err := bm.deps.JobControl.CreateJob(backup, job); err != nil {
		errMsg := fmt.Errorf("create backup %s/%s job %s failed, err: %v", ns, name, backupJobName, err)
		bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Command: logBackupSubcommand,
			Type:    v1alpha1.BackupRetryFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "CreateBackupJobFailed",
			Message: errMsg.Error(),
		}, nil)
		return errMsg
	}

	return bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Command: logBackupSubcommand,
		Type:    v1alpha1.BackupScheduled,
		Status:  corev1.ConditionTrue,
	}, updateStatus)
}

// validateBackup validates backup and returns error if backup is invalid
func (bm *backupManager) validateBackup(backup *v1alpha1.Backup) error {
	ns := backup.GetNamespace()
	name := backup.GetName()
	logBackupSubcommand := v1alpha1.ParseLogBackupSubcommand(backup)
	var err error
	if backup.Spec.BR == nil {
		err = backuputil.ValidateBackup(backup, "")
	} else {
		backupNamespace := backup.GetNamespace()
		if backup.Spec.BR.ClusterNamespace != "" {
			backupNamespace = backup.Spec.BR.ClusterNamespace
		}

		var tc *v1alpha1.TidbCluster
		tc, err = bm.deps.TiDBClusterLister.TidbClusters(backupNamespace).Get(backup.Spec.BR.Cluster)
		if err != nil {
			reason := fmt.Sprintf("failed to fetch tidbcluster %s/%s", backupNamespace, backup.Spec.BR.Cluster)
			bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Command: logBackupSubcommand,
				Type:    v1alpha1.BackupRetryFailed,
				Status:  corev1.ConditionTrue,
				Reason:  reason,
				Message: err.Error(),
			}, nil)
			return err
		}

		tikvImage := tc.TiKVImage()
		err = backuputil.ValidateBackup(backup, tikvImage)
	}

	if err != nil {
		bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Command: logBackupSubcommand,
			Type:    v1alpha1.BackupInvalid,
			Status:  corev1.ConditionTrue,
			Reason:  "InvalidSpec",
			Message: err.Error(),
		}, nil)

		return controller.IgnoreErrorf("invalid backup spec %s/%s cause %s", ns, name, err.Error())
	}
	return nil
}

// skipBackupSync skip backup sync, if return true, backup can be skiped directly.
func (bm *backupManager) skipBackupSync(backup *v1alpha1.Backup) (bool, error) {
	if backup.Spec.Mode == v1alpha1.BackupModeLog {
		return bm.skipLogBackupSync(backup)
	}
	return bm.skipSnapshotBackupSync(backup)
}

// waitPreTaskDone waits pre task done.
// just log backup needs to wait pre task done. Mainly including:
// 1. wait other command done, such as truncate/stop wait start done.
// 2. wait command's job done
func (bm *backupManager) waitPreTaskDone(backup *v1alpha1.Backup) error {
	if backup.Spec.Mode != v1alpha1.BackupModeLog {
		return nil
	}

	ns := backup.GetNamespace()
	name := backup.GetName()
	backupJobName := backup.GetBackupJobName()
	logBackupSubcommand := v1alpha1.ParseLogBackupSubcommand(backup)

	// check whether backup should wait and requeue
	if shouldLogBackupCommandRequeue(backup) {
		klog.Infof("log backup %s/%s subcommand %s should wait log backup start complete, will requeue.", ns, name, logBackupSubcommand)
		return controller.RequeueErrorf(fmt.Sprintf("log backup %s/%s command %s should wait log backup start complete", ns, name, logBackupSubcommand))
	}

	// log backup should wait old job done
	oldJob, err := bm.deps.JobLister.Jobs(ns).Get(backupJobName)
	if oldJob != nil {
		return waitOldLogBackupJobDone(ns, name, backupJobName, bm, backup, oldJob)
	}

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("log backup %s/%s get job %s failed, err: %v", ns, name, backupJobName, err)
	}
	return nil
}

func (bm *backupManager) makeBackupJob(backup *v1alpha1.Backup) (*batchv1.Job, *controller.BackupUpdateStatus, string, error) {
	var (
		job          *batchv1.Job
		updateStatus *controller.BackupUpdateStatus
		reason       string
		err          error
	)

	if backup.Spec.BR == nil {
		// not found backup job, so we need to create it
		job, reason, err = bm.makeExportJob(backup)
		if err != nil {
			bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Type:    v1alpha1.BackupRetryFailed,
				Status:  corev1.ConditionTrue,
				Reason:  reason,
				Message: err.Error(),
			}, nil)
			return nil, nil, "", err
		}

		reason, err = bm.ensureBackupPVCExist(backup)
		if err != nil {
			bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Type:    v1alpha1.BackupRetryFailed,
				Status:  corev1.ConditionTrue,
				Reason:  reason,
				Message: err.Error(),
			}, nil)
			return nil, nil, "", err
		}

	} else {
		logBackupSubcommand := v1alpha1.ParseLogBackupSubcommand(backup)
		// not found backup job, so we need to create it
		job, reason, err = bm.makeBRBackupJob(backup)
		if err != nil {
			bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Command: logBackupSubcommand,
				Type:    v1alpha1.BackupRetryFailed,
				Status:  corev1.ConditionTrue,
				Reason:  reason,
				Message: err.Error(),
			}, nil)
			return nil, nil, "", err
		}

		if logBackupSubcommand == v1alpha1.LogStartCommand {
			// log start need to start tracker
			bm.backupTracker.StartTrackLogBackupProgress(backup)
		} else if logBackupSubcommand == v1alpha1.LogTruncateCommand {
			updateStatus = &controller.BackupUpdateStatus{
				LogTruncatingUntil: &backup.Spec.LogTruncateUntil,
			}
		}
	}
	return job, updateStatus, reason, nil
}

func (bm *backupManager) makeExportJob(backup *v1alpha1.Backup) (*batchv1.Job, string, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()

	envVars, reason, err := backuputil.GenerateTidbPasswordEnv(ns, name, backup.Spec.From.SecretName, backup.Spec.UseKMS, bm.deps.SecretLister)
	if err != nil {
		return nil, reason, err
	}

	storageEnv, reason, err := backuputil.GenerateStorageCertEnv(ns, backup.Spec.UseKMS, backup.Spec.StorageProvider, bm.deps.SecretLister)
	if err != nil {
		return nil, reason, fmt.Errorf("backup %s/%s, %v", ns, name, err)
	}
	envVars = append(envVars, storageEnv...)

	// set env vars specified in backup.Spec.Env
	envVars = util.AppendOverwriteEnv(envVars, backup.Spec.Env)

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

	volumeMounts := []corev1.VolumeMount{}
	volumes := []corev1.Volume{}
	initContainers := []corev1.Container{}

	if backup.Spec.From.TLSClientSecretName != nil {
		args = append(args, "--client-tls=true")
		clientSecretName := *backup.Spec.From.TLSClientSecretName
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

	if backup.Spec.ToolImage != "" {
		dumplingVolumeMount := corev1.VolumeMount{
			Name:      "dumpling-bin",
			ReadOnly:  false,
			MountPath: util.DumplingBinPath,
		}
		volumeMounts = append(volumeMounts, dumplingVolumeMount)
		volumes = append(volumes, corev1.Volume{
			Name: "dumpling-bin",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
		initContainers = append(initContainers, corev1.Container{
			Name:            "dumpling",
			Image:           backup.Spec.ToolImage,
			Command:         []string{"/bin/sh", "-c"},
			Args:            []string{fmt.Sprintf("cp /dumpling %s/dumpling; echo 'dumpling copy finished'", util.DumplingBinPath)},
			ImagePullPolicy: corev1.PullIfNotPresent,
			VolumeMounts:    []corev1.VolumeMount{dumplingVolumeMount},
			Resources:       backup.Spec.ResourceRequirements,
		})
	}

	serviceAccount := constants.DefaultServiceAccountName
	if backup.Spec.ServiceAccount != "" {
		serviceAccount = backup.Spec.ServiceAccount
	}

	jobLabels := util.CombineStringMap(label.NewBackup().Instance(backup.GetInstanceName()).BackupJob().Backup(name), backup.Labels)
	podLabels := jobLabels
	jobAnnotations := backup.Annotations
	podAnnotations := backup.Annotations

	// TODO: need add ResourceRequirement for backup job
	podSpec := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      podLabels,
			Annotations: podAnnotations,
		},
		Spec: corev1.PodSpec{
			SecurityContext:    backup.Spec.PodSecurityContext,
			ServiceAccountName: serviceAccount,
			InitContainers:     initContainers,
			Containers: []corev1.Container{
				{
					Name:            label.BackupJobLabelVal,
					Image:           bm.deps.CLIConfig.TiDBBackupManagerImage,
					Args:            args,
					ImagePullPolicy: corev1.PullIfNotPresent,
					VolumeMounts: append([]corev1.VolumeMount{
						{Name: label.BackupJobLabelVal, MountPath: constants.BackupRootPath},
					}, volumeMounts...),
					Env:       util.AppendEnvIfPresent(envVars, "TZ"),
					Resources: backup.Spec.ResourceRequirements,
				},
			},
			RestartPolicy:    corev1.RestartPolicyNever,
			Tolerations:      backup.Spec.Tolerations,
			ImagePullSecrets: backup.Spec.ImagePullSecrets,
			Affinity:         backup.Spec.Affinity,
			Volumes: append([]corev1.Volume{
				{
					Name: label.BackupJobLabelVal,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: backup.GetBackupPVCName(),
						},
					},
				},
			}, volumes...),
			PriorityClassName: backup.Spec.PriorityClassName,
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        backup.GetBackupJobName(),
			Namespace:   ns,
			Labels:      jobLabels,
			Annotations: jobAnnotations,
			OwnerReferences: []metav1.OwnerReference{
				controller.GetBackupOwnerRef(backup),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: pointer.Int32Ptr(0),
			Template:     *podSpec,
		},
	}

	return job, "", nil
}

// makeBRBackupJob requires that backup.Spec.BR != nil
func (bm *backupManager) makeBRBackupJob(backup *v1alpha1.Backup) (*batchv1.Job, string, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()
	backupNamespace := ns
	if backup.Spec.BR.ClusterNamespace != "" {
		backupNamespace = backup.Spec.BR.ClusterNamespace
	}
	tc, err := bm.deps.TiDBClusterLister.TidbClusters(backupNamespace).Get(backup.Spec.BR.Cluster)
	if err != nil {
		return nil, fmt.Sprintf("failed to fetch tidbcluster %s/%s", backupNamespace, backup.Spec.BR.Cluster), err
	}

	var (
		envVars []corev1.EnvVar
		reason  string
	)
	if backup.Spec.From != nil {
		envVars, reason, err = backuputil.GenerateTidbPasswordEnv(ns, name, backup.Spec.From.SecretName, backup.Spec.UseKMS, bm.deps.SecretLister)
		if err != nil {
			return nil, reason, err
		}
	}

	storageEnv, reason, err := backuputil.GenerateStorageCertEnv(ns, backup.Spec.UseKMS, backup.Spec.StorageProvider, bm.deps.SecretLister)
	if err != nil {
		return nil, reason, fmt.Errorf("backup %s/%s, %v", ns, name, err)
	}

	envVars = append(envVars, storageEnv...)
	envVars = append(envVars, corev1.EnvVar{
		Name:  "BR_LOG_TO_TERM",
		Value: string(rune(1)),
	})

	// set env vars specified in backup.Spec.Env
	envVars = util.AppendOverwriteEnv(envVars, backup.Spec.Env)

	args := []string{
		"backup",
		fmt.Sprintf("--namespace=%s", ns),
		fmt.Sprintf("--backupName=%s", name),
		fmt.Sprintf("--mode=%s", backup.Spec.Mode),
	}
	tikvImage := tc.TiKVImage()
	_, tikvVersion := backuputil.ParseImage(tikvImage)
	if tikvVersion != "" {
		args = append(args, fmt.Sprintf("--tikvVersion=%s", tikvVersion))
	}

	if backup.Spec.Mode == v1alpha1.BackupModeLog {
		subcommand := v1alpha1.ParseLogBackupSubcommand(backup)
		args = append(args, fmt.Sprintf("--subcommand=%s", subcommand))
		switch subcommand {
		case v1alpha1.LogStartCommand:
			if backup.Spec.CommitTs != "" {
				args = append(args, fmt.Sprintf("--commit-ts=%s", backup.Spec.CommitTs))
			}
		case v1alpha1.LogTruncateCommand:
			if backup.Spec.LogTruncateUntil != "" {
				args = append(args, fmt.Sprintf("--truncate-until=%s", backup.Spec.LogTruncateUntil))
			}
		}
	}

	jobLabels := util.CombineStringMap(label.NewBackup().Instance(backup.GetInstanceName()).BackupJob().Backup(name), backup.Labels)
	podLabels := jobLabels
	jobAnnotations := backup.Annotations
	podAnnotations := jobAnnotations

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
					SecretName: util.ClusterClientTLSSecretName(backup.Spec.BR.Cluster),
				},
			},
		})
	}

	if backup.Spec.From != nil && tc.Spec.TiDB != nil && tc.Spec.TiDB.TLSClient != nil && tc.Spec.TiDB.TLSClient.Enabled && !tc.SkipTLSWhenConnectTiDB() {
		args = append(args, "--client-tls=true")
		if tc.Spec.TiDB.TLSClient.SkipInternalClientCA {
			args = append(args, "--skipClientCA=true")
		}

		clientSecretName := util.TiDBClientTLSSecretName(backup.Spec.BR.Cluster)
		if backup.Spec.From.TLSClientSecretName != nil {
			clientSecretName = *backup.Spec.From.TLSClientSecretName
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

	brVolumeMount := corev1.VolumeMount{
		Name:      "br-bin",
		ReadOnly:  false,
		MountPath: util.BRBinPath,
	}
	volumeMounts = append(volumeMounts, brVolumeMount)

	volumes = append(volumes, corev1.Volume{
		Name: "br-bin",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	// mount volumes if specified
	if backup.Spec.Local != nil {
		volumes = append(volumes, backup.Spec.Local.Volume)
		volumeMounts = append(volumeMounts, backup.Spec.Local.VolumeMount)
	}

	serviceAccount := constants.DefaultServiceAccountName
	if backup.Spec.ServiceAccount != "" {
		serviceAccount = backup.Spec.ServiceAccount
	}

	brImage := "pingcap/br:" + tikvVersion
	if backup.Spec.ToolImage != "" {
		toolImage := backup.Spec.ToolImage
		if !strings.ContainsRune(backup.Spec.ToolImage, ':') {
			toolImage = fmt.Sprintf("%s:%s", toolImage, tikvVersion)
		}

		brImage = toolImage
	}

	podSpec := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      podLabels,
			Annotations: podAnnotations,
		},
		Spec: corev1.PodSpec{
			SecurityContext:    backup.Spec.PodSecurityContext,
			ServiceAccountName: serviceAccount,
			InitContainers: []corev1.Container{
				{
					Name:            "br",
					Image:           brImage,
					Command:         []string{"/bin/sh", "-c"},
					Args:            []string{fmt.Sprintf("cp /br %s/br; echo 'BR copy finished'", util.BRBinPath)},
					ImagePullPolicy: corev1.PullIfNotPresent,
					VolumeMounts:    []corev1.VolumeMount{brVolumeMount},
					Resources:       backup.Spec.ResourceRequirements,
				},
			},
			Containers: []corev1.Container{
				{
					Name:            label.BackupJobLabelVal,
					Image:           bm.deps.CLIConfig.TiDBBackupManagerImage,
					Args:            args,
					ImagePullPolicy: corev1.PullIfNotPresent,
					VolumeMounts:    volumeMounts,
					Env:             util.AppendEnvIfPresent(envVars, "TZ"),
					Resources:       backup.Spec.ResourceRequirements,
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
			Name:        backup.GetBackupJobName(),
			Namespace:   ns,
			Labels:      jobLabels,
			Annotations: jobAnnotations,
			OwnerReferences: []metav1.OwnerReference{
				controller.GetBackupOwnerRef(backup),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: pointer.Int32Ptr(0),
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
	pvc, err := bm.deps.PVCLister.PersistentVolumeClaims(ns).Get(backupPVCName)

	if err == nil {
		if pvcRs := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; pvcRs.Cmp(rs) == -1 {
			return "PVCStorageSizeTooSmall", fmt.Errorf("%s/%s's backup pvc %s's storage size %s is less than expected storage size %s, please delete old pvc to continue", ns, name, pvc.GetName(), pvcRs.String(), rs.String())
		}
		return "", nil
	}

	if !errors.IsNotFound(err) {
		return "GetPVCFailed", fmt.Errorf("backup %s/%s get pvc %s failed, err: %v", ns, name, backupPVCName, err)
	}

	// not found PVC, so we need to create PVC for backup job
	pvc = &corev1.PersistentVolumeClaim{
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

	if err := bm.deps.GeneralPVCControl.CreatePVC(backup, pvc); err != nil {
		errMsg := fmt.Errorf("backup %s/%s create backup pvc %s failed, err: %v", ns, name, pvc.GetName(), err)
		return "CreatePVCFailed", errMsg
	}
	return "", nil
}

// skipSnapshotBackupSync skip snapshot backup, returns true if can be skipped.
func (bm *backupManager) skipSnapshotBackupSync(backup *v1alpha1.Backup) (bool, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()
	backupJobName := backup.GetBackupJobName()

	_, err := bm.deps.JobLister.Jobs(ns).Get(backupJobName)
	if err == nil {
		return true, nil
	}

	if !errors.IsNotFound(err) {
		return false, fmt.Errorf("backup %s/%s get job %s failed, err: %v", ns, name, backupJobName, err)
	}
	return false, nil
}

// skipLogBackupSync skip log backup, returns true if can be skipped.
func (bm *backupManager) skipLogBackupSync(backup *v1alpha1.Backup) (bool, error) {
	if backup.Spec.Mode != v1alpha1.BackupModeLog {
		return false, nil
	}
	var skip bool
	var err error
	command := v1alpha1.ParseLogBackupSubcommand(backup)
	switch command {
	case v1alpha1.LogStartCommand:
		skip = v1alpha1.IsLogBackupAlreadyStart(backup)
	case v1alpha1.LogTruncateCommand:
		if v1alpha1.IsLogBackupAlreadyTruncate(backup) {
			skip = true
			// if skip truncate, we need update truncate to be complete, and truncating util is the spec's truncate until.
			updateStatus := &controller.BackupUpdateStatus{
				TimeStarted:        &metav1.Time{Time: time.Now()},
				TimeCompleted:      &metav1.Time{Time: time.Now()},
				LogTruncatingUntil: &backup.Spec.LogTruncateUntil,
			}
			err = bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Command: v1alpha1.LogTruncateCommand,
				Type:    v1alpha1.BackupComplete,
				Status:  corev1.ConditionTrue,
			}, updateStatus)
		}
	case v1alpha1.LogStopCommand:
		skip = v1alpha1.IsLogBackupAlreadyStop(backup)
	default:
		return false, nil
	}

	if skip {
		klog.Infof("log backup %s/%s subcommand %s is already done, will skip sync.", backup.Namespace, backup.Name, command)
	}
	return skip, err
}

// shouldLogBackupCommandRequeue returns whether log backup subcommand should requeue.
// truncate and stop should wait start complete, otherwise, we should requeue this key.
func shouldLogBackupCommandRequeue(backup *v1alpha1.Backup) bool {
	if backup.Spec.Mode != v1alpha1.BackupModeLog {
		return false
	}
	command := v1alpha1.ParseLogBackupSubcommand(backup)

	if command == v1alpha1.LogTruncateCommand || command == v1alpha1.LogStopCommand {
		return backup.Status.CommitTs == ""
	}
	return false
}

// waitOldLogBackupJobDone wait old log backup job done
func waitOldLogBackupJobDone(ns, name, backupJobName string, bm *backupManager, backup *v1alpha1.Backup, oldJob *batchv1.Job) error {
	if backup.Spec.Mode != v1alpha1.BackupModeLog {
		return nil
	}

	if oldJob.DeletionTimestamp != nil {
		return controller.RequeueErrorf(fmt.Sprintf("log backup %s/%s job %s is being deleted", ns, name, backupJobName))
	}
	finished := false
	for _, c := range oldJob.Status.Conditions {
		if (c.Type == batchv1.JobComplete || c.Type == batchv1.JobFailed) && c.Status == corev1.ConditionTrue {
			finished = true
			break
		}
	}
	if finished {
		klog.Infof("log backup %s/%s job %s has complete or failed, will delete job", ns, name, backupJobName)
		if err := bm.deps.JobControl.DeleteJob(backup, oldJob); err != nil {
			return fmt.Errorf("log backup %s/%s delete job %s failed, err: %v", ns, name, backupJobName, err)
		}
	}
	// job running no need to requeue, because delete job will call update and it will requeue
	return controller.IgnoreErrorf("log backup %s/%s job %s is running, will be ignored", ns, name, backupJobName)
}

var _ backup.BackupManager = &backupManager{}

type FakeBackupManager struct {
	err error
}

func NewFakeBackupManager() *FakeBackupManager {
	return &FakeBackupManager{}
}

func (m *FakeBackupManager) SetSyncError(err error) {
	m.err = err
}

func (m *FakeBackupManager) Sync(_ *v1alpha1.Backup) error {
	return m.err
}

func (m *FakeBackupManager) UpdateCondition(_ *v1alpha1.Backup, _ *v1alpha1.BackupCondition) error {
	return nil
}

var _ backup.BackupManager = &FakeBackupManager{}
