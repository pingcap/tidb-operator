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
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	brv1alpha1 "github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	corev1alpha1 "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/client"
	"github.com/pingcap/tidb-operator/pkg/controllers/br/manager/constants"
	"github.com/pingcap/tidb-operator/pkg/controllers/br/manager/util"
	backuputil "github.com/pingcap/tidb-operator/pkg/controllers/br/manager/util"
	pdm "github.com/pingcap/tidb-operator/pkg/timanager/pd"
)

var (
	_ BackupManager = &backupManager{}
)

// BackupUpdateStatus represents the status of a backup to be updated.
// This structure should keep synced with the fields in `BackupStatus`
// except for `Phase` and `Conditions`.
type BackupUpdateStatus struct {
	// BackupPath is the location of the backup.
	BackupPath *string
	// TimeStarted is the time at which the backup was started.
	TimeStarted *metav1.Time
	// TimeCompleted is the time at which the backup was completed.
	TimeCompleted *metav1.Time
	// BackupSizeReadable is the data size of the backup.
	// the difference with BackupSize is that its format is human readable
	BackupSizeReadable *string
	// BackupSize is the data size of the backup.
	BackupSize *int64
	// CommitTs is the snapshot time point of tidb cluster.
	CommitTs *string
	// LogCheckpointTs is the ts of log backup process.
	LogCheckpointTs *string
	// LogSuccessTruncateUntil is log backup already successfully truncate until timestamp.
	LogSuccessTruncateUntil *string
	// LogTruncatingUntil is log backup truncate until timestamp which is used to mark the truncate command.
	LogTruncatingUntil *string
	// ProgressStep the step name of progress.
	ProgressStep *string
	// Progress is the step's progress value.
	Progress *int
	// ProgressUpdateTime is the progress update time.
	ProgressUpdateTime *metav1.Time

	// RetryNum is the number of retry
	RetryNum *int
	// DetectFailedAt is the time when detect failure
	DetectFailedAt *metav1.Time
	// ExpectedRetryAt is the time we calculate and expect retry after it
	ExpectedRetryAt *metav1.Time
	// RealRetryAt is the time when the retry was actually initiated
	RealRetryAt *metav1.Time
	// Reason is the reason of retry
	RetryReason *string
	// OriginalReason is the original reason of backup job or pod failed
	OriginalReason *string
}

// BackupManager implements the logic for manage backup.
type BackupManager interface {
	// Sync	implements the logic for syncing Backup.
	Sync(backup *v1alpha1.Backup) error
	// UpdateStatus updates the status for a Backup, include condition and status info.
	UpdateStatus(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, newStatus *BackupUpdateStatus) error
}

type backupManager struct {
	cli                client.Client
	backupCleaner      BackupCleaner
	backupTracker      BackupTracker
	statusUpdater      BackupConditionUpdaterInterface
	backupManagerImage string
}

// NewBackupManager return backupManager
func NewBackupManager(cli client.Client, pdcm pdm.PDClientManager, eventRecorder record.EventRecorder, backupManagerImage string) BackupManager {
	statusUpdater := NewRealBackupConditionUpdater(cli, eventRecorder)

	return &backupManager{
		cli:                cli,
		backupCleaner:      NewBackupCleaner(cli, statusUpdater, backupManagerImage),
		backupTracker:      NewBackupTracker(cli, pdcm, statusUpdater),
		statusUpdater:      statusUpdater,
		backupManagerImage: backupManagerImage,
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

// UpdateStatus updates the status for a Backup, include condition and status info.
func (bm *backupManager) UpdateStatus(backup *v1alpha1.Backup, condition *v1alpha1.BackupCondition, newStatus *BackupUpdateStatus) error {
	return bm.statusUpdater.Update(backup, condition, newStatus)
}

func (bm *backupManager) syncBackupJob(backup *v1alpha1.Backup) error {
	ns := backup.GetNamespace()
	name := backup.GetName()
	backupJobName := backup.GetBackupJobName()
	logBackupSubcommand := v1alpha1.ParseLogBackupSubcommand(backup)
	var err error

	if v1alpha1.IsBackupComplete(backup) || v1alpha1.IsBackupFailed(backup) {
		return nil
	}

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

	// make backup job
	var job *batchv1.Job
	var reason string
	var updateStatus *BackupUpdateStatus
	if job, updateStatus, reason, err = bm.makeBackupJob(backup); err != nil {
		klog.Errorf("backup %s/%s create job %s failed, reason is %s, error %v.", ns, name, backupJobName, reason, err)
		return err
	}

	// create k8s job
	klog.Infof("backup %s/%s creating job %s.", ns, name, backupJobName)
	if err := bm.cli.Create(context.TODO(), job); err != nil && !apierrors.IsAlreadyExists(err) {
		errMsg := fmt.Errorf("create backup %s/%s job %s failed, err: %w", ns, name, backupJobName, err)
		_ = bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Command: logBackupSubcommand,
			Condition: metav1.Condition{
				Type:    string(v1alpha1.BackupRetryTheFailed),
				Status:  metav1.ConditionTrue,
				Reason:  "CreateBackupJobFailed",
				Message: errMsg.Error(),
			},
		}, nil)
		return errMsg
	}

	return bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Command: logBackupSubcommand,
		Condition: metav1.Condition{
			Type:   string(v1alpha1.BackupScheduled),
			Status: metav1.ConditionTrue,
		},
	}, updateStatus)
}

// validateBackup validates backup and returns error if backup is invalid
func (bm *backupManager) validateBackup(backup *v1alpha1.Backup) error {
	ns := backup.GetNamespace()
	name := backup.GetName()
	var err error
	logBackupSubcommand := v1alpha1.ParseLogBackupSubcommand(backup)

	if backup.Spec.Mode == v1alpha1.BackupModeLog && logBackupSubcommand == v1alpha1.LogUnknownCommand {
		err = fmt.Errorf("log backup %s/%s subcommand `%s` is not supported", ns, name, backup.Spec.LogSubcommand)
		_ = bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Command: logBackupSubcommand,
			Condition: metav1.Condition{
				Type:    string(v1alpha1.BackupRetryTheFailed),
				Status:  metav1.ConditionTrue,
				Reason:  err.Error(),
				Message: err.Error(),
			},
		}, nil)
		return err
	}

	backupNamespace := backup.GetNamespace()
	if backup.Spec.BR.ClusterNamespace != "" {
		backupNamespace = backup.Spec.BR.ClusterNamespace
	}

	cluster := &corev1alpha1.Cluster{}
	err = bm.cli.Get(context.TODO(), types.NamespacedName{Namespace: backupNamespace, Name: backup.Spec.BR.Cluster}, cluster)
	if err != nil {
		reason := fmt.Sprintf("failed to fetch tidbcluster %s/%s", backupNamespace, backup.Spec.BR.Cluster)
		_ = bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Command: logBackupSubcommand,
			Condition: metav1.Condition{
				Type:    string(v1alpha1.BackupRetryTheFailed),
				Status:  metav1.ConditionTrue,
				Reason:  reason,
				Message: err.Error(),
			},
		}, nil)
		return err
	}
	tikvGroup, err := util.FirstTikvGroup(bm.cli, backupNamespace, backup.Spec.BR.Cluster)
	if err != nil {
		return err
	}

	err = backuputil.ValidateBackup(backup, tikvGroup.Spec.Template.Spec.Version, cluster)
	if err != nil {
		_ = bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Command: logBackupSubcommand,
			Condition: metav1.Condition{
				Type:    string(v1alpha1.BackupInvalid),
				Status:  metav1.ConditionTrue,
				Reason:  "InvalidSpec",
				Message: err.Error(),
			},
		}, nil)

		// TODO(ideascf): Use IgnoreErrorf
		return fmt.Errorf("invalid backup spec %s/%s cause %s", ns, name, err.Error())
	}
	return nil
}

// skipBackupSync skip backup sync, if return true, backup can be skipped directly.
func (bm *backupManager) skipBackupSync(backup *v1alpha1.Backup) (bool, error) {
	if backup.Spec.Mode == v1alpha1.BackupModeLog {
		return bm.skipLogBackupSync(backup)
	} else {
		return bm.skipSnapshotBackupSync(backup)
	}
}

// waitPreTaskDone waits pre task done. this can make sure there is only one backup job running.
// 1. wait other command done, such as truncate/stop wait start done.
// 2. wait command's job done
func (bm *backupManager) waitPreTaskDone(backup *v1alpha1.Backup) error {
	ns := backup.GetNamespace()
	name := backup.GetName()
	backupJobName := backup.GetBackupJobName()

	// check whether backup should wait and requeue
	if shouldLogBackupCommandRequeue(backup) {
		logBackupSubcommand := v1alpha1.ParseLogBackupSubcommand(backup)
		klog.Infof("log backup %s/%s subcommand %s should wait log backup start complete, will requeue.", ns, name, logBackupSubcommand)
		// TODO(ideascf): use controller.RequeueErrorf
		return fmt.Errorf("log backup %s/%s command %s should wait log backup start complete", ns, name, logBackupSubcommand)
	}

	// log backup should wait old job done
	oldJob := &batchv1.Job{}
	err := bm.cli.Get(context.TODO(), client.ObjectKey{Namespace: ns, Name: backupJobName}, oldJob)
	if err == nil {
		return waitOldBackupJobDone(ns, name, backupJobName, bm, backup, oldJob)
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("backup %s/%s get job %s failed, err: %w", ns, name, backupJobName, err)
	}
	return nil
}

func (bm *backupManager) makeBackupJob(backup *v1alpha1.Backup) (*batchv1.Job, *BackupUpdateStatus, string, error) {
	var (
		job          *batchv1.Job
		updateStatus *BackupUpdateStatus
		reason       string
		err          error
	)

	{
		logBackupSubcommand := v1alpha1.ParseLogBackupSubcommand(backup)
		// not found backup job, so we need to create it
		job, reason, err = bm.makeBRBackupJob(backup)
		if err != nil {
			// don't retry on dup metadata file existing error
			if reason == "FileExistedInExternalStorage" {
				_ = bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
					Command: logBackupSubcommand,
					Condition: metav1.Condition{
						Type:    string(v1alpha1.BackupFailed),
						Status:  metav1.ConditionTrue,
						Reason:  reason,
						Message: err.Error(),
					},
				}, nil)
				// TODO(ideascf): use controller.IgnoreErrorf
				return nil, nil, "", fmt.Errorf("%s, reason is %s", err.Error(), reason)
			} else {
				_ = bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
					Command: logBackupSubcommand,
					Condition: metav1.Condition{
						Type:    string(v1alpha1.BackupRetryTheFailed),
						Status:  metav1.ConditionTrue,
						Reason:  reason,
						Message: err.Error(),
					},
				}, nil)
				return nil, nil, "", err
			}
		}

		if logBackupSubcommand == v1alpha1.LogStartCommand {
			// log start need to start tracker
			err = bm.backupTracker.StartTrackLogBackupProgress(backup)
			if err != nil {
				_ = bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
					Command: logBackupSubcommand,
					Condition: metav1.Condition{
						Type:    string(v1alpha1.BackupRetryTheFailed),
						Status:  metav1.ConditionTrue,
						Reason:  "start log backup progress tracker error",
						Message: err.Error(),
					},
				}, nil)
				return nil, nil, "", err
			}
		} else if logBackupSubcommand == v1alpha1.LogTruncateCommand {
			updateStatus = &BackupUpdateStatus{
				LogTruncatingUntil: &backup.Spec.LogTruncateUntil,
			}
		}
	}
	return job, updateStatus, reason, nil
}

// nolint: gocyclo
// makeBRBackupJob requires that backup.Spec.BR != nil
func (bm *backupManager) makeBRBackupJob(backup *v1alpha1.Backup) (*batchv1.Job, string, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()
	jobName := backup.GetBackupJobName()
	backupNamespace := ns
	if backup.Spec.BR.ClusterNamespace != "" {
		backupNamespace = backup.Spec.BR.ClusterNamespace
	}

	cluster := &corev1alpha1.Cluster{}
	err := bm.cli.Get(context.TODO(), client.ObjectKey{Namespace: backupNamespace, Name: backup.Spec.BR.Cluster}, cluster)
	if err != nil {
		return nil, fmt.Sprintf("failed to fetch tidbcluster %s/%s", backupNamespace, backup.Spec.BR.Cluster), err
	}
	tikvGroup, err := util.FirstTikvGroup(bm.cli, ns, backup.Spec.BR.Cluster)
	if err != nil {
		return nil, fmt.Sprintf("failed to get tikv group %s/%s", ns, backup.Spec.BR.Cluster), err
	}
	pdGroup, err := util.FirstPDGroup(bm.cli, ns, cluster.Name)
	if err != nil {
		return nil, fmt.Sprintf("failed to get first pd group: %v", err), err
	}

	var (
		envVars []corev1.EnvVar
		reason  string
	)

	storageEnv, reason, err := backuputil.GenerateStorageCertEnv(ns, backup.Spec.UseKMS, backup.Spec.StorageProvider, bm.cli)
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
		fmt.Sprintf("--pd-addr=%s", fmt.Sprintf("%s-pd.%s:%d", pdGroup.Spec.Cluster.Name, pdGroup.Namespace, coreutil.PDGroupClientPort(pdGroup))),
	}
	tikvVersion := tikvGroup.Spec.Template.Spec.Version
	if tikvVersion != "" {
		args = append(args, fmt.Sprintf("--tikvVersion=%s", tikvVersion))
	}

	switch backup.Spec.Mode {
	case v1alpha1.BackupModeLog:
		args = append(args, fmt.Sprintf("--mode=%s", v1alpha1.BackupModeLog))
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
	default:
		args = append(args, fmt.Sprintf("--mode=%s", v1alpha1.BackupModeSnapshot))
	}

	jobLabels := util.CombineStringMap(metav1alpha1.NewBackup().Instance(backup.GetInstanceName()).BackupJob().Backup(name), backup.Labels)
	podLabels := jobLabels
	jobAnnotations := backup.Annotations
	podAnnotations := jobAnnotations

	volumeMounts := []corev1.VolumeMount{}
	volumes := []corev1.Volume{}

	if coreutil.IsTLSClusterEnabled(cluster) {
		args = append(args, "--cluster-tls=true")
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      corev1alpha1.VolumeNameClusterClientTLS,
			ReadOnly:  true,
			MountPath: corev1alpha1.DirPathClusterClientTLS,
		})
		volumes = append(volumes, corev1.Volume{
			Name: corev1alpha1.VolumeNameClusterClientTLS,
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
					Name:            brv1alpha1.LabelValComponentBackup,
					Image:           bm.backupManagerImage,
					Command:         []string{"/backup-manager"},
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

// skipSnapshotBackupSync skip snapshot backup, returns true if can be skipped.
func (bm *backupManager) skipSnapshotBackupSync(backup *v1alpha1.Backup) (bool, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()
	backupJobName := backup.GetBackupJobName()

	job := &batchv1.Job{}
	err := bm.cli.Get(context.TODO(), client.ObjectKey{Namespace: ns, Name: backupJobName}, job)
	if err == nil {
		return true, nil
	}

	if !errors.IsNotFound(err) {
		return false, fmt.Errorf("backup %s/%s get job %s failed, err: %w", ns, name, backupJobName, err)
	}
	return false, nil
}

// skipLogBackupSync skips log backup, returns true if it can be skipped.
func (bm *backupManager) skipLogBackupSync(backup *v1alpha1.Backup) (bool, error) {
	if backup.Spec.Mode != v1alpha1.BackupModeLog {
		return false, nil
	}

	command := v1alpha1.ParseLogBackupSubcommand(backup)
	if command != v1alpha1.LogTruncateCommand && v1alpha1.IsLogSubcommandAlreadySync(backup, command) {
		return true, nil
	}

	// Handle the special case for LogTruncateCommand
	var err error
	if command == v1alpha1.LogTruncateCommand && v1alpha1.IsLogBackupAlreadyTruncate(backup) {
		// If skipping truncate, update status
		updateStatus := &BackupUpdateStatus{
			TimeStarted:        &metav1.Time{Time: time.Now()},
			TimeCompleted:      &metav1.Time{Time: time.Now()},
			LogTruncatingUntil: &backup.Spec.LogTruncateUntil,
		}
		err = bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Command: v1alpha1.LogTruncateCommand,
			Condition: metav1.Condition{
				Type:   string(v1alpha1.BackupComplete),
				Status: metav1.ConditionTrue,
			},
		}, updateStatus)
		klog.Infof("log backup %s/%s subcommand %s is already done, will skip sync.", backup.Namespace, backup.Name, command)
		return true, err
	}

	return false, nil
}

// shouldLogBackupCommandRequeue returns whether log backup subcommand should requeue.
// truncate and stop should wait start complete, otherwise, we should requeue this key.
func shouldLogBackupCommandRequeue(backup *v1alpha1.Backup) bool {
	if backup.Spec.Mode != v1alpha1.BackupModeLog {
		return false
	}
	command := v1alpha1.ParseLogBackupSubcommand(backup)

	if command == v1alpha1.LogTruncateCommand || command == v1alpha1.LogStopCommand || command == v1alpha1.LogPauseCommand {
		return backup.Status.CommitTs == ""
	}
	return false
}

// waitOldBackupJobDone wait old backup job done
func waitOldBackupJobDone(ns, name, backupJobName string, bm *backupManager, backup *v1alpha1.Backup, oldJob *batchv1.Job) error {
	if oldJob.DeletionTimestamp != nil {
		// TODO(ideascf): use controller.RequeueErrorf
		return fmt.Errorf("backup %s/%s job %s is being deleted", ns, name, backupJobName)
	}
	finished := false
	for _, c := range oldJob.Status.Conditions {
		if (c.Type == batchv1.JobComplete || c.Type == batchv1.JobFailed) && c.Status == corev1.ConditionTrue {
			finished = true
			break
		}
	}
	if finished {
		klog.Infof("backup %s/%s job %s has complete or failed, will delete job", ns, name, backupJobName)
		if err := bm.cli.Delete(context.TODO(), oldJob); err != nil {
			return fmt.Errorf("backup %s/%s delete job %s failed, err: %w", ns, name, backupJobName, err)
		}
	}

	// job running no need to requeue, because delete job will call update and it will requeue
	// TODO(ideascf): use controller.IgnoreErrorf
	return fmt.Errorf("backup %s/%s job %s is running, will be ignored", ns, name, backupJobName)
}

var _ BackupManager = &backupManager{}

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

// UpdateStatus updates the status for a Backup, include condition and status info.
func (m *FakeBackupManager) UpdateStatus(_ *v1alpha1.Backup, _ *v1alpha1.BackupCondition, newStatus *BackupUpdateStatus) error {
	return nil
}

var _ BackupManager = &FakeBackupManager{}
