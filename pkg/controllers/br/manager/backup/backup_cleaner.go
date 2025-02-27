// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package backup

import (
	"context"
	"fmt"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	rtClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	brv1alpha1 "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	corev1alpha1 "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/controllers/br/manager/constants"
	"github.com/pingcap/tidb-operator/pkg/controllers/br/manager/util"
	backuputil "github.com/pingcap/tidb-operator/pkg/controllers/br/manager/util"
)

var _ BackupCleaner = &backupCleaner{}

// BackupCleaner implements the logic for cleaning backup
type BackupCleaner interface {
	Clean(backup *v1alpha1.Backup) error
}

type backupCleaner struct {
	cli                client.Client
	statusUpdater      BackupConditionUpdaterInterface
	backupManagerImage string
}

// NewBackupCleaner returns a BackupCleaner
func NewBackupCleaner(cli client.Client, statusUpdater BackupConditionUpdaterInterface, backupManagerImage string) BackupCleaner {
	return &backupCleaner{
		cli:                cli,
		statusUpdater:      statusUpdater,
		backupManagerImage: backupManagerImage,
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
		return bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Command: v1alpha1.LogStopCommand,
			Condition: metav1.Condition{
				Type:    string(v1alpha1.BackupComplete),
				Status:  metav1.ConditionTrue,
				Reason:  "LogBackupNotStarted",
				Message: "Log backup has not started, so there is no need to stop it",
			},
		}, nil)
	}
	if v1alpha1.IsLogBackupAlreadyStop(backup) {
		return nil
	}

	ns := backup.GetNamespace()
	name := backup.GetName()
	stopLogJobName := backup.GetStopLogBackupJobName()

	var err error
	job := &batchv1.Job{}
	err = bc.cli.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: stopLogJobName}, job)
	if err == nil {
		// already have a clean job running，return directly
		return nil
	} else if !errors.IsNotFound(err) {
		_ = bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Condition: metav1.Condition{
				Type:    string(v1alpha1.BackupRetryTheFailed),
				Status:  metav1.ConditionTrue,
				Reason:  "GetBackupFailed",
				Message: err.Error(),
			},
		}, nil)
		return err
	}

	// make backup job
	var reason string
	if job, reason, err = bc.makeStopLogBackupJob(backup); err != nil {
		klog.Errorf("backup %s/%s create job %s failed, reason is %s, error %v.", ns, name, stopLogJobName, reason, err)
		return err
	}

	// create k8s job
	if err := bc.cli.Create(context.Background(), job); err != nil {
		errMsg := fmt.Errorf("stop log backup %s/%s job %s failed, err: %w", ns, name, stopLogJobName, err)
		_ = bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Command: v1alpha1.LogStopCommand,
			Condition: metav1.Condition{
				Type:    string(v1alpha1.BackupRetryTheFailed),
				Status:  metav1.ConditionTrue,
				Reason:  "StopLogBackupJobFailed",
				Message: fmt.Sprintf("failed to create stop log backup job, err: %s", err),
			},
		}, nil)
		return errMsg
	}

	return bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Command: v1alpha1.LogStopCommand,
		Condition: metav1.Condition{
			Type:    string(v1alpha1.BackupScheduled),
			Status:  metav1.ConditionTrue,
			Reason:  "StopLogBackupJobCreated",
			Message: fmt.Sprintf("Stop log backup job %s created", stopLogJobName),
		},
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
		return fmt.Errorf("ensure %s/%s jobs finished failed: %w", ns, name, err)
	}
	if !finished {
		klog.Infof("wait for backup %s/%s jobs to finish", ns, name)
		return nil
	}

	klog.Infof("start to clean backup %s/%s", ns, name)

	cleanJobName := backup.GetCleanJobName()
	cleanJob := &batchv1.Job{}
	err = bc.cli.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: cleanJobName}, cleanJob)
	if err == nil {
		// already have a clean job running，return directly
		return nil
	} else if !errors.IsNotFound(err) {
		_ = bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Condition: metav1.Condition{
				Type:    string(v1alpha1.BackupRetryTheFailed),
				Status:  metav1.ConditionTrue,
				Reason:  "GetBackupFailed",
				Message: err.Error(),
			},
		}, nil)
		return err
	}

	// no found the clean job, we start to create the clean job.
	if backup.Status.BackupPath == "" {
		// the backup path is empty, so there is no need to clean up backup data
		return bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Condition: metav1.Condition{
				Type:    string(v1alpha1.BackupClean),
				Status:  metav1.ConditionTrue,
				Reason:  "DataCleaned",
				Message: "The backup path is empty, so there is no need to clean up backup data",
			},
		}, nil)
	}

	// not found clean job, create it
	job, reason, err := bc.makeCleanJob(backup)
	if err != nil {
		_ = bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Condition: metav1.Condition{
				Type:    string(v1alpha1.BackupRetryTheFailed),
				Status:  metav1.ConditionTrue,
				Reason:  reason,
				Message: err.Error(),
			},
		}, nil)
		return err
	}

	if err := bc.cli.Create(context.TODO(), job); err != nil {
		errMsg := fmt.Errorf("create backup %s/%s job %s failed, err: %w", ns, name, cleanJobName, err)
		_ = bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Condition: metav1.Condition{
				Type:    string(v1alpha1.BackupRetryTheFailed),
				Status:  metav1.ConditionTrue,
				Reason:  "CreateCleanJobFailed",
				Message: errMsg.Error(),
			},
		}, nil)
		return errMsg
	}

	return bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Condition: metav1.Condition{
			Type:    string(v1alpha1.BackupClean),
			Status:  metav1.ConditionFalse,
			Reason:  "CleanJobCreated",
			Message: fmt.Sprintf("Clean job %s created", cleanJobName),
		},
	}, nil)
}

func (bc *backupCleaner) makeCleanJob(backup *v1alpha1.Backup) (*batchv1.Job, string, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()

	envVars, reason, err := backuputil.GenerateStorageCertEnv(ns, backup.Spec.UseKMS, backup.Spec.StorageProvider, bc.cli)
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

	backupLabel := metav1alpha1.NewBackup().Instance(backup.GetInstanceName()).CleanJob().Backup(name)
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
					Name:            metav1alpha1.BackupJobLabelVal,
					Image:           bc.backupManagerImage,
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
				brv1alpha1.GetBackupOwnerRef(backup),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: ptr.To(int32(constants.DefaultBackoffLimit)),
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

	cluster := &corev1alpha1.Cluster{}
	err := bc.cli.Get(context.TODO(), types.NamespacedName{Namespace: backupNamespace, Name: backup.Spec.BR.Cluster}, cluster)
	if err != nil {
		return nil, fmt.Sprintf("failed to fetch tidbcluster %s/%s", backupNamespace, backup.Spec.BR.Cluster), err
	}
	tikvGroup, err := util.FirstTikvGroup(bc.cli, ns, cluster.Name)
	if err != nil {
		return nil, fmt.Sprintf("failed to get first tikv group: %v", err), err
	}
	pdGroup, err := util.FirstPDGroup(bc.cli, ns, cluster.Name)
	if err != nil {
		return nil, fmt.Sprintf("failed to get first pd group: %v", err), err
	}

	var (
		envVars []corev1.EnvVar
		reason  string
	)
	storageEnv, reason, err := backuputil.GenerateStorageCertEnv(ns, backup.Spec.UseKMS, backup.Spec.StorageProvider, bc.cli)
	if err != nil {
		return nil, reason, fmt.Errorf("backup %s/%s, %w", ns, name, err)
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
		fmt.Sprintf("--pd-addr=%s", fmt.Sprintf("%s-pd.%s:%d", pdGroup.Name, pdGroup.Namespace, coreutil.PDGroupClientPort(pdGroup))),
	}
	tikvVersion := tikvGroup.Spec.Template.Spec.Version
	if tikvVersion != "" {
		args = append(args, fmt.Sprintf("--tikvVersion=%s", tikvVersion))
	}

	args = append(args, fmt.Sprintf("--mode=%s", v1alpha1.BackupModeLog))
	args = append(args, fmt.Sprintf("--subcommand=%s", v1alpha1.LogStopCommand))

	jobLabels := util.CombineStringMap(metav1alpha1.NewBackup().Instance(backup.GetInstanceName()).BackupJob().Backup(name), backup.Labels)
	podLabels := jobLabels
	jobAnnotations := backup.Annotations
	podAnnotations := jobAnnotations

	volumeMounts := []corev1.VolumeMount{}
	volumes := []corev1.Volume{}

	if coreutil.IsTLSClusterEnabled(cluster) {
		args = append(args, "--cluster-tls=true")
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      corev1alpha1.VolumeNameClusterTLS,
			ReadOnly:  true,
			MountPath: corev1alpha1.DirPathClusterTLSTiKV,
		})
		volumes = append(volumes, corev1.Volume{
			Name: corev1alpha1.VolumeNameClusterTLS,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: coreutil.TLSClusterClientSecretName(backup.Spec.BR.Cluster),
				},
			},
		})
	}

	brVolumeMount := corev1.VolumeMount{
		Name:      "br-bin",
		ReadOnly:  false,
		MountPath: v1alpha1.DirPathBRBin,
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
					Args:            []string{fmt.Sprintf("cp /br %s/br; echo 'BR copy finished'", v1alpha1.DirPathBRBin)},
					ImagePullPolicy: corev1.PullIfNotPresent,
					VolumeMounts:    []corev1.VolumeMount{brVolumeMount},
					Resources:       backup.Spec.ResourceRequirements,
				},
			},
			Containers: []corev1.Container{
				{
					Name:            metav1alpha1.BackupJobLabelVal,
					Image:           bc.backupManagerImage,
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
				brv1alpha1.GetBackupOwnerRef(backup),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: ptr.To(int32(0)),
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
		job := &batchv1.Job{}
		err := bc.cli.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: jobName}, job)
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
		if err := bc.cli.Delete(context.TODO(), job, rtClient.PropagationPolicy(metav1.DeletePropagationForeground)); client.IgnoreNotFound(err) != nil {
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
	job := &batchv1.Job{}
	err := bc.cli.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: stopLogJob}, job)
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
