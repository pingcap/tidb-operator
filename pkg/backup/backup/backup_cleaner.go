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
	if backup.DeletionTimestamp == nil {
		// The backup object has not been deleted，do nothing
		return nil
	}

	klog.Infof("prepare to clean backup %s/%s", backup.GetNamespace(), backup.GetName())
	if err := bc.StopLogBackup(backup); err != nil {
		return err
	}
	return bc.CleanData(backup)
}

func (bc *backupCleaner) StopLogBackup(backup *v1alpha1.Backup) error {
	if backup.Spec.Mode != v1alpha1.BackupModeLog {
		return nil
	}
	if backup.Spec.BR == nil {
		return fmt.Errorf("backup %s/%s spec.BR shouldn't be nil", backup.GetNamespace(), backup.GetName())
	}
	if !v1alpha1.IsLogBackupAlreadyStart(backup) {
		return nil
	}
	if v1alpha1.IsLogBackupAlreadyStop(backup) {
		return nil
	}

	ns := backup.GetNamespace()
	name := backup.GetName()
	stopLogJobName := backup.GetStopLogBackupJobName()

	var err error
	_, err = bc.deps.JobLister.Jobs(ns).Get(stopLogJobName)
	if err == nil {
		// already have a clean job running，return directly
		return nil
	} else if !errors.IsNotFound(err) {
		bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupRetryTheFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetBackupFailed",
			Message: err.Error(),
		}, nil)
		return err
	}

	// make backup job
	var job *batchv1.Job
	var reason string
	if job, reason, err = bc.makeStopLogBackupJob(backup); err != nil {
		klog.Errorf("backup %s/%s create job %s failed, reason is %s, error %v.", ns, name, stopLogJobName, reason, err)
		return err
	}

	// create k8s job
	if err := bc.deps.JobControl.CreateJob(backup, job); err != nil {
		errMsg := fmt.Errorf("stop log backup %s/%s job %s failed, err: %v", ns, name, stopLogJobName, err)
		bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Command: v1alpha1.LogStopCommand,
			Type:    v1alpha1.BackupRetryTheFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "StopLogBackupJobFailed",
			Message: errMsg.Error(),
		}, nil)
		return errMsg
	}

	return bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Command: v1alpha1.LogStopCommand,
		Type:    v1alpha1.BackupScheduled,
		Status:  corev1.ConditionTrue,
	}, nil)
}

func (bc *backupCleaner) CleanData(backup *v1alpha1.Backup) error {
	if backup.DeletionTimestamp == nil || !v1alpha1.IsCleanCandidate(backup) || v1alpha1.NeedRetainData(backup) {
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
			Type:    v1alpha1.BackupRetryTheFailed,
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
			Type:    v1alpha1.BackupRetryTheFailed,
			Status:  corev1.ConditionTrue,
			Reason:  reason,
			Message: err.Error(),
		}, nil)
		return err
	}

	if err := bc.deps.JobControl.CreateJob(backup, job); err != nil {
		errMsg := fmt.Errorf("create backup %s/%s job %s failed, err: %v", ns, name, cleanJobName, err)
		bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupRetryTheFailed,
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

	if len(backup.Spec.AdditionalVolumes) > 0 {
		volumes = append(volumes, backup.Spec.AdditionalVolumes...)
	}
	if len(backup.Spec.AdditionalVolumeMounts) > 0 {
		volumeMounts = append(volumeMounts, backup.Spec.AdditionalVolumeMounts...)
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

func (bc *backupCleaner) makeStopLogBackupJob(backup *v1alpha1.Backup) (*batchv1.Job, string, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()
	jobName := backup.GetStopLogBackupJobName()
	backupNamespace := ns
	if backup.Spec.BR.ClusterNamespace != "" {
		backupNamespace = backup.Spec.BR.ClusterNamespace
	}

	tc, err := bc.deps.TiDBClusterLister.TidbClusters(backupNamespace).Get(backup.Spec.BR.Cluster)
	if err != nil {
		return nil, fmt.Sprintf("failed to fetch tidbcluster %s/%s", backupNamespace, backup.Spec.BR.Cluster), err
	}

	var (
		envVars []corev1.EnvVar
		reason  string
	)
	if backup.Spec.From != nil {
		envVars, reason, err = backuputil.GenerateTidbPasswordEnv(ns, name, backup.Spec.From.SecretName, backup.Spec.UseKMS, bc.deps.SecretLister)
		if err != nil {
			return nil, reason, err
		}
	}

	storageEnv, reason, err := backuputil.GenerateStorageCertEnv(ns, backup.Spec.UseKMS, backup.Spec.StorageProvider, bc.deps.SecretLister)
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
	}
	tikvImage := tc.TiKVImage()
	_, tikvVersion := backuputil.ParseImage(tikvImage)
	if tikvVersion != "" {
		args = append(args, fmt.Sprintf("--tikvVersion=%s", tikvVersion))
	}

	args = append(args, fmt.Sprintf("--mode=%s", v1alpha1.BackupModeLog))
	args = append(args, fmt.Sprintf("--subcommand=%s", v1alpha1.LogStopCommand))

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

		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "tidb-client-tls",
			ReadOnly:  true,
			MountPath: util.TiDBClientTLSPath,
		})
		volumes = append(volumes, corev1.Volume{
			Name: "tidb-client-tls",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.TiDBClientTLSSecretName(backup.Spec.BR.Cluster, backup.Spec.From.TLSClientSecretName),
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

	if len(backup.Spec.AdditionalVolumes) > 0 {
		volumes = append(volumes, backup.Spec.AdditionalVolumes...)
	}
	if len(backup.Spec.AdditionalVolumeMounts) > 0 {
		volumeMounts = append(volumeMounts, backup.Spec.AdditionalVolumeMounts...)
	}

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
					Image:           bc.deps.CLIConfig.TiDBBackupManagerImage,
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
			Name:        jobName,
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

// ensureBackupJobFinished ensures that all backup jobs have finished, deleting any running jobs.
func (bc *backupCleaner) ensureBackupJobFinished(backup *v1alpha1.Backup) (bool, error) {

	ns := backup.GetNamespace()
	name := backup.GetName()
	backupJobNames := bc.getBackupJobNames(backup)

	isAllFinished := true
	var errs []error

	for _, jobName := range backupJobNames {
		job, err := bc.deps.JobLister.Jobs(ns).Get(jobName)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			errs = append(errs, err)
			continue
		}

		if job.DeletionTimestamp != nil {
			klog.Infof("backup %s/%s job %s is being deleted, cleaner will wait", ns, name, jobName)
			isAllFinished = false
			continue
		}

		if bc.isJobDoneOrFailed(job) {
			continue
		}

		klog.Infof("backup %s/%s job %s is running, cleaner will delete it and wait it done", ns, name, jobName)
		if err := bc.deps.JobControl.DeleteJob(backup, job); err != nil {
			errs = append(errs, err)
		}

		isAllFinished = false
	}

	if len(errs) > 0 {
		return false, errorutils.NewAggregate(errs)
	}

	if backup.Spec.Mode == v1alpha1.BackupModeLog {
		isLogStopJobFinished, err := bc.isLogStopJobFinished(backup)
		if err != nil {
			return false, err
		}
		if !isLogStopJobFinished {
			return false, nil
		}
	}
	return isAllFinished, nil
}

func (bc *backupCleaner) isLogStopJobFinished(backup *v1alpha1.Backup) (bool, error) {
	if backup.Spec.Mode != v1alpha1.BackupModeLog {
		return true, nil
	}
	if v1alpha1.IsLogBackupAlreadyStop(backup) {
		return true, nil
	}

	ns := backup.GetNamespace()
	name := backup.GetName()
	stopLogJob := backup.GetStopLogBackupJobName()
	job, err := bc.deps.JobLister.Jobs(ns).Get(stopLogJob)
	if err == nil {
		if !bc.isJobDoneOrFailed(job) {
			klog.Infof("log backup %s/%s is running stop task, cleaner will wait until done", ns, name)
			return false, nil
		} else if bc.isJobFailed(job) {
			klog.Errorf("log backup %s/%s stopping task %s failed", ns, name, stopLogJob)
		}
		return true, nil
	} else if errors.IsNotFound(err) {
		klog.Warningf("log backup %s/%s stopping-task %s not found, log backup may has failed before cleaning", ns, name, stopLogJob)
		return true, nil
	}
	return false, err
}

func (bc *backupCleaner) getBackupJobNames(backup *v1alpha1.Backup) []string {
	// log backup may have multiple jobs
	backupJobNames := make([]string, 0)
	if backup.Spec.Mode == v1alpha1.BackupModeLog {
		backupJobNames = append(backupJobNames, backup.GetAllLogBackupJobName()...)
	} else {
		backupJobNames = append(backupJobNames, backup.GetBackupJobName())
	}
	return backupJobNames
}

func (bc *backupCleaner) isJobDoneOrFailed(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if (condition.Type == batchv1.JobComplete || condition.Type == batchv1.JobFailed) && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (bc *backupCleaner) isJobFailed(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
