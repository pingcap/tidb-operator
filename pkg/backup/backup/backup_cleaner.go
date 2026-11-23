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
	"context"
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

const startLogBackupJobObserved = "StartLogBackupJobObserved"

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
	if backup.Spec.Mode == v1alpha1.BackupModeLog && !IsLogBackupStoppedForCleanup(backup) {
		return bc.StopLogBackup(backup)
	}
	return bc.CleanData(backup)
}

func (bc *backupCleaner) StopLogBackup(backup *v1alpha1.Backup) error {
	if backup.Spec.Mode != v1alpha1.BackupModeLog || IsLogBackupStoppedForCleanup(backup) {
		return nil
	}
	started := v1alpha1.IsLogBackupAlreadyStart(backup)
	if !started {
		current, needsStop, err := bc.logStartNeedsStop(backup)
		if err != nil {
			return err
		}
		backup = current
		if IsLogBackupStoppedForCleanup(backup) {
			return nil
		}
		if !needsStop {
			if isLogBackupTaskAlreadyExists(backup) {
				klog.Warningf("skip stopping log backup %s/%s because BR rejected start due to an existing log task", backup.Namespace, backup.Name)
			}
			return bc.markLogBackupStopped(backup)
		}
		started = v1alpha1.IsLogBackupAlreadyStart(backup)
	}
	if backup.Spec.BR == nil {
		return fmt.Errorf("backup %s/%s spec.BR shouldn't be nil", backup.GetNamespace(), backup.GetName())
	}

	ns := backup.GetNamespace()
	name := backup.GetName()
	stopLogJobName := backup.GetStopLogBackupJobName()
	clusterNamespace := backup.Spec.BR.ClusterNamespace
	if clusterNamespace == "" {
		clusterNamespace = ns
	}
	tc, err := bc.deps.TiDBClusterLister.TidbClusters(clusterNamespace).Get(backup.Spec.BR.Cluster)
	if !started && errors.IsNotFound(err) {
		// Confirm absence before skipping stop, since the informer can lag the API server.
		tc, err = bc.deps.Clientset.PingcapV1alpha1().TidbClusters(clusterNamespace).Get(context.TODO(), backup.Spec.BR.Cluster, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			klog.Warningf("skip stopping log backup %s/%s because TidbCluster %s/%s was not found", ns, name, clusterNamespace, backup.Spec.BR.Cluster)
			return bc.markLogBackupStopped(backup)
		}
	}
	if err != nil {
		return fmt.Errorf("failed to fetch tidbcluster %s/%s: %w", clusterNamespace, backup.Spec.BR.Cluster, err)
	}

	stopJob, err := bc.deps.JobLister.Jobs(ns).Get(stopLogJobName)
	if errors.IsNotFound(err) {
		stopJob, err = bc.deps.KubeClientset.BatchV1().Jobs(ns).Get(context.TODO(), stopLogJobName, metav1.GetOptions{})
	}
	waitForStop := func() error {
		return controller.RequeueErrorf("backup %s/%s is waiting for stop job %s", ns, name, stopLogJobName)
	}
	if err == nil {
		if stopJob.DeletionTimestamp != nil {
			return waitForStop()
		}
		for _, condition := range stopJob.Status.Conditions {
			if condition.Status != corev1.ConditionTrue {
				continue
			}
			switch condition.Type {
			case batchv1.JobFailed:
				if err := bc.deps.JobControl.DeleteJob(backup, stopJob); err != nil && !errors.IsNotFound(err) {
					return err
				}
				return waitForStop()
			case batchv1.JobComplete:
				return nil
			}
		}
		// Job status changes alone do not enqueue the Backup.
		return waitForStop()
	} else if !errors.IsNotFound(err) {
		bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupRetryTheFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "GetBackupFailed",
			Message: err.Error(),
		}, nil)
		return err
	}

	if !started {
		// Let any in-flight start Job exit before stop checks whether the BR task exists.
		finished, err := bc.ensureBackupJobFinished(backup)
		if err != nil {
			return err
		}
		if !finished {
			return controller.RequeueErrorf("backup %s/%s is waiting for log backup jobs to finish", ns, name)
		}
	}

	job, reason, err := bc.makeStopLogBackupJob(backup, tc)
	if err != nil {
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

	if err := bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Command: v1alpha1.LogStopCommand,
		Type:    v1alpha1.BackupScheduled,
		Status:  corev1.ConditionTrue,
	}, nil); err != nil {
		return err
	}
	return waitForStop()
}

func (bc *backupCleaner) logStartNeedsStop(backup *v1alpha1.Backup) (*v1alpha1.Backup, bool, error) {
	// The start Job is the first entry, even if the CR now requests a different subcommand.
	startJobName := backup.GetAllLogBackupJobName()[0]
	job, err := bc.deps.KubeClientset.BatchV1().Jobs(backup.Namespace).Get(context.TODO(), startJobName, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return nil, false, fmt.Errorf("failed to fetch start job %s/%s: %w", backup.Namespace, startJobName, err)
	}
	if err != nil || !metav1.IsControlledBy(job, backup) {
		job = nil
	}

	// The Backup status can reach the API server before the Job exits and the informer catches up.
	backup, err = bc.getLatestBackup(backup)
	if err != nil {
		return nil, false, err
	}
	if IsLogBackupStoppedForCleanup(backup) || v1alpha1.IsLogBackupAlreadyStart(backup) {
		return backup, v1alpha1.IsLogBackupAlreadyStart(backup), nil
	}

	start := backup.Status.LogSubCommandStatuses[v1alpha1.LogStartCommand]
	reachedBR := logStartReachedBR(start.Phase)
	var metadataFailed, startFailed, startJobObserved bool
	checkCondition := func(condition v1alpha1.BackupCondition) {
		reachedBR = reachedBR || logStartReachedBR(condition.Type)
		metadataFailed = metadataFailed || condition.Reason == "GetLogBackupMetadataFailed"
		startFailed = startFailed || condition.Reason == "StartLogBackupFailed"
		startJobObserved = startJobObserved || condition.Reason == startLogBackupJobObserved
	}
	for _, condition := range start.Conditions {
		checkCondition(condition)
	}
	for _, condition := range backup.Status.Conditions {
		// Start conditions written during deletion retain their command. Before
		// deletion, the top-level start failure reasons can be recorded without it.
		if condition.Command == v1alpha1.LogStartCommand ||
			(condition.Command == "" && (condition.Reason == "GetLogBackupMetadataFailed" || condition.Reason == "StartLogBackupFailed")) {
			checkCondition(condition)
		}
	}
	taskAlreadyExists := isLogBackupTaskAlreadyExists(backup)
	needsStop := !taskAlreadyExists && (metadataFailed || (!startFailed && (reachedBR || startJobObserved)))
	if job == nil {
		return backup, needsStop, nil
	}

	if !needsStop && !startFailed && !taskAlreadyExists {
		// Preserve the evidence across reconciles before cancelling the start Job.
		if err := bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Command: v1alpha1.LogStartCommand,
			Type:    v1alpha1.BackupScheduled,
			Status:  corev1.ConditionTrue,
			Reason:  startLogBackupJobObserved,
		}, nil); err != nil {
			return nil, false, err
		}
		return nil, false, controller.RequeueErrorf("backup %s/%s recorded start job %s for cleanup", backup.Namespace, backup.Name, startJobName)
	}

	if job.DeletionTimestamp == nil {
		if bc.isJobDoneOrFailed(job) {
			return backup, needsStop, nil
		}
		// Let a Job running BR report its outcome before deciding whether to stop the task.
		if !reachedBR || startFailed || metadataFailed || taskAlreadyExists {
			if err := bc.deps.JobControl.DeleteJob(backup, job); err != nil && !errors.IsNotFound(err) {
				return nil, false, err
			}
		}
	}
	return nil, false, controller.RequeueErrorf("backup %s/%s is waiting for start job %s to exit", backup.Namespace, backup.Name, startJobName)
}

func (bc *backupCleaner) getLatestBackup(backup *v1alpha1.Backup) (*v1alpha1.Backup, error) {
	current, err := bc.deps.Clientset.PingcapV1alpha1().Backups(backup.Namespace).Get(context.TODO(), backup.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to refresh backup %s/%s: %w", backup.Namespace, backup.Name, err)
	}
	if current.UID != backup.UID {
		return nil, fmt.Errorf("backup %s/%s was replaced during cleanup", backup.Namespace, backup.Name)
	}
	return current, nil
}

func isLogBackupTaskAlreadyExists(backup *v1alpha1.Backup) bool {
	if backup.Spec.Mode != v1alpha1.BackupModeLog || v1alpha1.IsLogBackupAlreadyStart(backup) {
		return false
	}
	isTaskConflict := func(condition v1alpha1.BackupCondition) bool {
		return condition.Reason == constants.LogBackupTaskAlreadyExists ||
			(condition.Reason == "StartLogBackupFailed" && strings.Contains(condition.Message, constants.BRStreamLogTaskExistsErrorCode))
	}
	for _, condition := range backup.Status.LogSubCommandStatuses[v1alpha1.LogStartCommand].Conditions {
		if isTaskConflict(condition) {
			return true
		}
	}
	for _, condition := range backup.Status.Conditions {
		if (condition.Command == "" || condition.Command == v1alpha1.LogStartCommand) && isTaskConflict(condition) {
			return true
		}
	}
	return false
}

func logStartReachedBR(phase v1alpha1.BackupConditionType) bool {
	switch phase {
	case v1alpha1.BackupPrepare, v1alpha1.BackupRunning, v1alpha1.BackupComplete:
		return true
	default:
		return false
	}
}

func (bc *backupCleaner) markLogBackupStopped(backup *v1alpha1.Backup) error {
	return bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Command: v1alpha1.LogStopCommand,
		Type:    v1alpha1.BackupComplete,
		Status:  corev1.ConditionTrue,
	}, nil)
}

// IsLogBackupStoppedForCleanup checks stop completion before cleaning or finalizing a log backup.
func IsLogBackupStoppedForCleanup(backup *v1alpha1.Backup) bool {
	if v1alpha1.IsLogBackupAlreadyStop(backup) {
		return true
	}
	if _, stopped := v1alpha1.GetBackupCondition(&backup.Status, v1alpha1.BackupStopped); stopped != nil && stopped.Status == corev1.ConditionTrue {
		return true
	}
	if backup.Status.LogSubCommandStatuses[v1alpha1.LogStopCommand].Phase == v1alpha1.BackupComplete {
		return true
	}
	// Older backups can record stop completion only in the top-level conditions.
	_, completed := v1alpha1.GetBackupCondition(&backup.Status, v1alpha1.BackupComplete)
	return completed != nil && completed.Command == v1alpha1.LogStopCommand && completed.Status == corev1.ConditionTrue
}

func (bc *backupCleaner) CleanData(backup *v1alpha1.Backup) error {
	if backup.DeletionTimestamp == nil || !v1alpha1.IsCleanCandidate(backup) {
		return nil
	}
	logBackup := backup.Spec.Mode == v1alpha1.BackupModeLog
	if logBackup && !IsLogBackupStoppedForCleanup(backup) {
		return nil
	}
	checkLogStart := logBackup && !v1alpha1.IsLogBackupAlreadyStart(backup)
	if !checkLogStart && v1alpha1.NeedRetainData(backup) {
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

	if checkLogStart {
		backup, err = bc.getLatestBackup(backup)
		if err != nil {
			return err
		}
		if !IsLogBackupStoppedForCleanup(backup) {
			return nil
		}
		if isLogBackupTaskAlreadyExists(backup) {
			klog.Warningf("skip cleaning log backup %s/%s because BR rejected start due to an existing log task", ns, name)
			return bc.markBackupClean(backup)
		}
		if v1alpha1.NeedRetainData(backup) {
			return nil
		}
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
		return bc.markBackupClean(backup)
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

func (bc *backupCleaner) markBackupClean(backup *v1alpha1.Backup) error {
	return bc.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupClean,
		Status: corev1.ConditionTrue,
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
			SecurityContext:              backup.Spec.PodSecurityContext,
			ServiceAccountName:           serviceAccount,
			AutomountServiceAccountToken: backup.Spec.AutomountServiceAccountToken,
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

	if backup.Spec.AutomountServiceAccountToken != nil && !*backup.Spec.AutomountServiceAccountToken {
		podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, util.SATokenProjectionVolume())
		podSpec.Spec.Containers[0].VolumeMounts = append(podSpec.Spec.Containers[0].VolumeMounts, util.SATokenProjectionVolumeMount())
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

func (bc *backupCleaner) makeStopLogBackupJob(backup *v1alpha1.Backup, tc *v1alpha1.TidbCluster) (*batchv1.Job, string, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()
	jobName := backup.GetStopLogBackupJobName()

	var (
		envVars []corev1.EnvVar
		reason  string
		err     error
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
			SecurityContext:              backup.Spec.PodSecurityContext,
			ServiceAccountName:           serviceAccount,
			AutomountServiceAccountToken: backup.Spec.AutomountServiceAccountToken,
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

	if backup.Spec.AutomountServiceAccountToken != nil && !*backup.Spec.AutomountServiceAccountToken {
		podSpec.Spec.Volumes = append(podSpec.Spec.Volumes, util.SATokenProjectionVolume())
		podSpec.Spec.Containers[0].VolumeMounts = append(podSpec.Spec.Containers[0].VolumeMounts, util.SATokenProjectionVolumeMount())
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
			BackoffLimit: pointer.Int32Ptr(constants.DefaultBackoffLimit),
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
	getJob := bc.deps.JobLister.Jobs(ns).Get
	logBackup := backup.Spec.Mode == v1alpha1.BackupModeLog
	if logBackup {
		// Use the live Job status when checking whether a log backup can stop or clean.
		jobs := bc.deps.KubeClientset.BatchV1().Jobs(ns)
		getJob = func(jobName string) (*batchv1.Job, error) {
			return jobs.Get(context.TODO(), jobName, metav1.GetOptions{})
		}
	}

	for _, jobName := range backupJobNames {
		job, err := getJob(jobName)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			errs = append(errs, err)
			continue
		}

		if logBackup && !metav1.IsControlledBy(job, backup) {
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

	return isAllFinished && len(errs) == 0, errorutils.NewAggregate(errs)
}

func (bc *backupCleaner) getBackupJobNames(backup *v1alpha1.Backup) []string {
	if backup.Spec.Mode == v1alpha1.BackupModeLog {
		return backup.GetAllLogBackupJobName()
	}
	return []string{backup.GetBackupJobName()}
}

func (bc *backupCleaner) isJobDoneOrFailed(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if (condition.Type == batchv1.JobComplete || condition.Type == batchv1.JobFailed) && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
