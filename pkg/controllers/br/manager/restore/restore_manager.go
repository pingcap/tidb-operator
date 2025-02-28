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

package restore

import (
	"context"
	"fmt"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/pingcap/tidb-operator/api/v2/br/v1alpha1"
	corev1alpha1 "github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
	metav1alpha1 "github.com/pingcap/tidb-operator/api/v2/meta/v1alpha1"
	coreutil "github.com/pingcap/tidb-operator/pkg/apiutil/core/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controllers/br/manager/constants"
	"github.com/pingcap/tidb-operator/pkg/controllers/br/manager/util"
	backuputil "github.com/pingcap/tidb-operator/pkg/controllers/br/manager/util"
	"github.com/pingcap/tidb-operator/pkg/controllers/common"
)

const (
	TiKVConfigEncryptionMethod      = "security.encryption.data-encryption-method"
	TiKVConfigEncryptionMasterKeyId = "security.encryption.master-key.key-id"
)

// RestoreManager implements the logic for manage restore.
type RestoreManager interface {
	// Sync	implements the logic for syncing Restore.
	Sync(ctx context.Context, restore *v1alpha1.Restore) error
	// UpdateCondition updates the condition for a Restore.
	UpdateCondition(ctx context.Context, restore *v1alpha1.Restore, condition *metav1.Condition) error
}

type restoreManager struct {
	cli                client.Client
	statusUpdater      RestoreConditionUpdaterInterface
	backupManagerImage string
}

// NewRestoreManager return restoreManager
func NewRestoreManager(cli client.Client, eventRecorder record.EventRecorder, backupManagerImage string) RestoreManager {
	return &restoreManager{
		cli:                cli,
		statusUpdater:      NewRealRestoreConditionUpdater(cli, eventRecorder),
		backupManagerImage: backupManagerImage,
	}
}

func (rm *restoreManager) Sync(ctx context.Context, restore *v1alpha1.Restore) error {
	return rm.syncRestoreJob(ctx, restore)
}

func (rm *restoreManager) UpdateCondition(ctx context.Context, restore *v1alpha1.Restore, condition *metav1.Condition) error {
	return rm.statusUpdater.Update(ctx, restore, condition, nil)
}

func (rm *restoreManager) syncRestoreJob(ctx context.Context, restore *v1alpha1.Restore) error {
	logger := log.FromContext(ctx)
	ns := restore.GetNamespace()
	name := restore.GetName()

	var (
		err              error
		cluster          *corev1alpha1.Cluster
		restoreNamespace string
	)

	restoreNamespace = restore.GetNamespace()
	if restore.Spec.BR.ClusterNamespace != "" {
		restoreNamespace = restore.Spec.BR.ClusterNamespace
	}

	cluster = &corev1alpha1.Cluster{}
	err = rm.cli.Get(ctx, client.ObjectKey{Namespace: restoreNamespace, Name: restore.Spec.BR.Cluster}, cluster)
	if err != nil {
		reason := fmt.Sprintf("failed to fetch tidbcluster %s/%s", restoreNamespace, restore.Spec.BR.Cluster)
		_ = rm.statusUpdater.Update(ctx, restore, &metav1.Condition{
			Type:    string(v1alpha1.RestoreRetryFailed),
			Status:  metav1.ConditionTrue,
			Reason:  reason,
			Message: err.Error(),
		}, nil)
		return err
	}
	tikvGroup, err := util.FirstTikvGroup(ctx, rm.cli, cluster.Namespace, cluster.Name)
	if err != nil {
		return fmt.Errorf("failed to get first tikv group: %w", err)
	}

	err = backuputil.ValidateRestore(restore, tikvGroup.Spec.Template.Spec.Version)
	if err != nil {
		_ = rm.statusUpdater.Update(ctx, restore, &metav1.Condition{
			Type:    string(v1alpha1.RestoreInvalid),
			Status:  metav1.ConditionTrue,
			Reason:  "InvalidSpec",
			Message: err.Error(),
		}, nil)

		return common.IgnoreErrorf("invalid restore spec %s/%s", ns, name)
	}

	if v1alpha1.IsRestoreFailed(restore) {
		return nil
	}

	restoreJobName := restore.GetRestoreJobName()
	restoreJob := &batchv1.Job{}
	err = rm.cli.Get(ctx, client.ObjectKey{Namespace: restore.Namespace, Name: restoreJobName}, restoreJob)
	if err == nil {
		logger.Info("restore job has been created, sync job status", "namespace", ns, "restoreJobName", restoreJobName)
		return rm.syncK8sJobStatus(ctx, restore, restoreJob)
	} else if !errors.IsNotFound(err) {
		return fmt.Errorf("restore %s/%s get job %s failed, err: %w", ns, name, restoreJobName, err)
	}

	var (
		job    *batchv1.Job
		reason string
	)
	job, reason, err = rm.makeRestoreJob(ctx, restore)
	if err != nil {
		_ = rm.statusUpdater.Update(ctx, restore, &metav1.Condition{
			Type:    string(v1alpha1.RestoreRetryFailed),
			Status:  metav1.ConditionTrue,
			Reason:  reason,
			Message: err.Error(),
		}, nil)
		return err
	}

	logger.Info("restore creating job", "namespace", ns, "restoreJobName", restoreJobName)
	if err := rm.cli.Create(ctx, job); err != nil && !apierrors.IsAlreadyExists(err) {
		errMsg := fmt.Errorf("create restore %s/%s job %s failed, err: %w", ns, name, restoreJobName, err)
		_ = rm.statusUpdater.Update(ctx, restore, &metav1.Condition{
			Type:    string(v1alpha1.RestoreRetryFailed),
			Status:  metav1.ConditionTrue,
			Reason:  "CreateRestoreJobFailed",
			Message: errMsg.Error(),
		}, nil)
		return errMsg
	}

	// Currently, the restore phase reuses the condition type and is updated when the condition is changed.
	// However, conditions are only used to describe the detailed status of the restore job. It is not suitable
	// for describing a state machine.
	//
	// Some restore such as volume-snapshot will create multiple jobs, and the phase will be changed to
	// running when the first job is running. To avoid the phase going back from running to scheduled, we
	// don't update the condition when the scheduled condition has already been set to true.
	if !v1alpha1.IsRestoreScheduled(restore) {
		return rm.statusUpdater.Update(ctx, restore, &metav1.Condition{
			Type:   string(v1alpha1.RestoreScheduled),
			Status: metav1.ConditionTrue,
		}, nil)
	}
	return nil
}

func (rm *restoreManager) makeRestoreJob(ctx context.Context, restore *v1alpha1.Restore) (*batchv1.Job, string, error) {
	ns := restore.GetNamespace()
	name := restore.GetName()
	restoreNamespace := ns
	if restore.Spec.BR.ClusterNamespace != "" {
		restoreNamespace = restore.Spec.BR.ClusterNamespace
	}
	cluster := &corev1alpha1.Cluster{}
	err := rm.cli.Get(ctx, client.ObjectKey{Namespace: restoreNamespace, Name: restore.Spec.BR.Cluster}, cluster)
	if err != nil {
		return nil, fmt.Sprintf("failed to fetch tidbcluster %s/%s", restoreNamespace, restore.Spec.BR.Cluster), err
	}
	tikvGroup, err := util.FirstTikvGroup(ctx, rm.cli, ns, cluster.Name)
	if err != nil {
		return nil, fmt.Sprintf("failed to get first tikv group: %v", err), err
	}
	pdGroup, err := util.FirstPDGroup(ctx, rm.cli, ns, cluster.Name)
	if err != nil {
		return nil, fmt.Sprintf("failed to get first pd group: %v", err), err
	}

	var (
		envVars []corev1.EnvVar
		reason  string
	)

	storageEnv, reason, err := backuputil.GenerateStorageCertEnv(ctx, ns, restore.Spec.UseKMS, restore.Spec.StorageProvider, rm.cli)
	if err != nil {
		return nil, reason, fmt.Errorf("restore %s/%s, %w", ns, name, err)
	}

	envVars = append(envVars, storageEnv...)
	envVars = append(envVars, corev1.EnvVar{
		Name:  "BR_LOG_TO_TERM",
		Value: string(rune(1)),
	})
	// set env vars specified in backup.Spec.Env
	envVars = util.AppendOverwriteEnv(envVars, restore.Spec.Env)

	args := []string{
		"restore",
		fmt.Sprintf("--namespace=%s", ns),
		fmt.Sprintf("--restoreName=%s", name),
		fmt.Sprintf("--pd-addr=%s", fmt.Sprintf("%s-pd.%s:%d", pdGroup.Name, pdGroup.Namespace, coreutil.PDGroupClientPort(pdGroup))),
	}
	tikvVersion := tikvGroup.Spec.Template.Spec.Version
	if tikvVersion != "" {
		args = append(args, fmt.Sprintf("--tikvVersion=%s", tikvVersion))
	}

	switch restore.Spec.Mode {
	case v1alpha1.RestoreModePiTR:
		args = append(args, fmt.Sprintf("--mode=%s", v1alpha1.RestoreModePiTR))
		args = append(args, fmt.Sprintf("--pitrRestoredTs=%s", restore.Spec.PitrRestoredTs))
	default:
		args = append(args, fmt.Sprintf("--mode=%s", v1alpha1.RestoreModeSnapshot))
	}

	jobLabels := util.CombineStringMap(metav1alpha1.NewRestore().Instance(restore.GetInstanceName()).RestoreJob().Restore(name), restore.Labels)
	podLabels := jobLabels
	jobAnnotations := restore.Annotations
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
					SecretName: coreutil.TLSClusterClientSecretName(restore.Spec.BR.Cluster),
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

	if len(restore.Spec.AdditionalVolumes) > 0 {
		volumes = append(volumes, restore.Spec.AdditionalVolumes...)
	}
	if len(restore.Spec.AdditionalVolumeMounts) > 0 {
		volumeMounts = append(volumeMounts, restore.Spec.AdditionalVolumeMounts...)
	}

	// mount volumes if specified
	if restore.Spec.Local != nil {
		volumes = append(volumes, restore.Spec.Local.Volume)
		volumeMounts = append(volumeMounts, restore.Spec.Local.VolumeMount)
	}

	serviceAccount := constants.DefaultServiceAccountName
	if restore.Spec.ServiceAccount != "" {
		serviceAccount = restore.Spec.ServiceAccount
	}

	brImage := "pingcap/br:" + tikvVersion
	if restore.Spec.ToolImage != "" {
		toolImage := restore.Spec.ToolImage
		if !strings.ContainsRune(toolImage, ':') {
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
			SecurityContext:    restore.Spec.PodSecurityContext,
			ServiceAccountName: serviceAccount,
			InitContainers: []corev1.Container{
				{
					Name:            "br",
					Image:           brImage,
					Command:         []string{"/bin/sh", "-c"},
					Args:            []string{fmt.Sprintf("cp /br %s/br; echo 'BR copy finished'", v1alpha1.DirPathBRBin)},
					ImagePullPolicy: corev1.PullIfNotPresent,
					VolumeMounts:    []corev1.VolumeMount{brVolumeMount},
					Resources:       restore.Spec.ResourceRequirements,
				},
			},
			Containers: []corev1.Container{
				{
					Name:            metav1alpha1.RestoreJobLabelVal,
					Image:           rm.backupManagerImage,
					Command:         []string{"/backup-manager"},
					Args:            args,
					ImagePullPolicy: corev1.PullIfNotPresent,
					VolumeMounts:    volumeMounts,
					Env:             util.AppendEnvIfPresent(envVars, "TZ"),
					Resources:       restore.Spec.ResourceRequirements,
				},
			},
			RestartPolicy:     corev1.RestartPolicyNever,
			Tolerations:       restore.Spec.Tolerations,
			ImagePullSecrets:  restore.Spec.ImagePullSecrets,
			Affinity:          restore.Spec.Affinity,
			Volumes:           volumes,
			PriorityClassName: restore.Spec.PriorityClassName,
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        restore.GetRestoreJobName(),
			Namespace:   ns,
			Labels:      jobLabels,
			Annotations: jobAnnotations,
			OwnerReferences: []metav1.OwnerReference{
				v1alpha1.GetRestoreOwnerRef(restore),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: ptr.To(int32(0)),
			Template:     *podSpec,
		},
	}

	return job, "", nil
}

func (rm *restoreManager) syncK8sJobStatus(ctx context.Context, restore *v1alpha1.Restore, job *batchv1.Job) error {
	// Get job condition by type
	getJobCondition := func(job *batchv1.Job, condType batchv1.JobConditionType) *batchv1.JobCondition {
		for i := range job.Status.Conditions {
			if job.Status.Conditions[i].Type == condType {
				return &job.Status.Conditions[i]
			}
		}
		return nil
	}

	// Check for job completion
	cond := getJobCondition(job, batchv1.JobComplete)
	// set only when the backup-manager job doesn't set the RestoreComplete condition
	if cond != nil && cond.Status == corev1.ConditionTrue && !v1alpha1.IsRestoreComplete(restore) {
		return rm.statusUpdater.Update(ctx, restore, &metav1.Condition{
			Type:    string(v1alpha1.RestoreComplete),
			Status:  metav1.ConditionTrue,
			Reason:  "RestoreComplete",
			Message: "Restore job completed",
		}, nil)
	}

	// Check for job failure
	cond = getJobCondition(job, batchv1.JobFailed)
	// set only when the backup-manager job doesn't set the RestoreFailed condition
	if cond != nil && cond.Status == corev1.ConditionTrue && !v1alpha1.IsRestoreFailed(restore) {
		return rm.statusUpdater.Update(ctx, restore, &metav1.Condition{
			Type:    string(v1alpha1.RestoreFailed),
			Status:  metav1.ConditionTrue,
			Reason:  "RestoreFailed",
			Message: cond.Message,
		}, nil)
	}

	return nil
}

var _ RestoreManager = &restoreManager{}

type FakeRestoreManager struct {
	err error
}

func NewFakeRestoreManager() *FakeRestoreManager {
	return &FakeRestoreManager{}
}

func (frm *FakeRestoreManager) SetSyncError(err error) {
	frm.err = err
}

func (frm *FakeRestoreManager) Sync(_ context.Context, _ *v1alpha1.Restore) error {
	return frm.err
}

func (frm *FakeRestoreManager) UpdateCondition(_ context.Context, _ *v1alpha1.Restore, _ *metav1.Condition) error {
	return nil
}

var _ RestoreManager = &FakeRestoreManager{}
