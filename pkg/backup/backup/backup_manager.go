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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	backuputil "github.com/pingcap/tidb-operator/pkg/backup/util"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/label"
	"github.com/pingcap/tidb-operator/pkg/util"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

type backupManager struct {
	deps          *controller.Dependencies
	backupCleaner BackupCleaner
	statusUpdater controller.BackupConditionUpdaterInterface
}

// NewBackupManager return backupManager
func NewBackupManager(deps *controller.Dependencies) backup.BackupManager {
	statusUpdater := controller.NewRealBackupConditionUpdater(deps.Clientset, deps.BackupLister, deps.Recorder)
	return &backupManager{
		deps:          deps,
		backupCleaner: NewBackupCleaner(deps, statusUpdater),
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
			Type:    v1alpha1.BackupInvalid,
			Status:  corev1.ConditionTrue,
			Reason:  "InvalidSpec",
			Message: err.Error(),
		}, nil)

		return controller.IgnoreErrorf("invalid backup spec %s/%s cause %s", ns, name, err.Error())
	}

	_, err = bm.deps.JobLister.Jobs(ns).Get(backupJobName)
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
			}, nil)
			return err
		}

		reason, err = bm.ensureBackupPVCExist(backup)
		if err != nil {
			bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
				Type:    v1alpha1.BackupRetryFailed,
				Status:  corev1.ConditionTrue,
				Reason:  reason,
				Message: err.Error(),
			}, nil)
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
			}, nil)
			return err
		}
	}

	if err := bm.deps.JobControl.CreateJob(backup, job); err != nil {
		errMsg := fmt.Errorf("create backup %s/%s job %s failed, err: %v", ns, name, backupJobName, err)
		bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
			Type:    v1alpha1.BackupRetryFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "CreateBackupJobFailed",
			Message: errMsg.Error(),
		}, nil)
		return errMsg
	}

	return bm.statusUpdater.Update(backup, &v1alpha1.BackupCondition{
		Type:   v1alpha1.BackupScheduled,
		Status: corev1.ConditionTrue,
	}, nil)
}

func (bm *backupManager) makeExportJob(backup *v1alpha1.Backup) (*batchv1.Job, string, error) {
	ns := backup.GetNamespace()
	name := backup.GetName()

	envVars, reason, err := backuputil.GenerateTidbPasswordEnv(ns, name, backup.Spec.From.SecretName, backup.Spec.UseKMS, bm.deps.KubeClientset)
	if err != nil {
		return nil, reason, err
	}

	storageEnv, reason, err := backuputil.GenerateStorageCertEnv(ns, backup.Spec.UseKMS, backup.Spec.StorageProvider, bm.deps.KubeClientset)
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

// makeBackupJob requires that backup.Spec.BR != nil
func (bm *backupManager) makeBackupJob(backup *v1alpha1.Backup) (*batchv1.Job, string, error) {
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
		envVars, reason, err = backuputil.GenerateTidbPasswordEnv(ns, name, backup.Spec.From.SecretName, backup.Spec.UseKMS, bm.deps.KubeClientset)
		if err != nil {
			return nil, reason, err
		}
	}

	storageEnv, reason, err := backuputil.GenerateStorageCertEnv(ns, backup.Spec.UseKMS, backup.Spec.StorageProvider, bm.deps.KubeClientset)
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
