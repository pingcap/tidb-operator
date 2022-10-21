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

package restore

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup"
	"github.com/pingcap/tidb-operator/pkg/backup/constants"
	"github.com/pingcap/tidb-operator/pkg/backup/snapshotter"
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

type restoreManager struct {
	deps          *controller.Dependencies
	statusUpdater controller.RestoreConditionUpdaterInterface
}

// NewRestoreManager return restoreManager
func NewRestoreManager(deps *controller.Dependencies) backup.RestoreManager {
	return &restoreManager{
		deps:          deps,
		statusUpdater: controller.NewRealRestoreConditionUpdater(deps.Clientset, deps.RestoreLister, deps.Recorder),
	}
}

func (rm *restoreManager) Sync(restore *v1alpha1.Restore) error {
	return rm.syncRestoreJob(restore)
}

func (rm *restoreManager) UpdateCondition(restore *v1alpha1.Restore, condition *v1alpha1.RestoreCondition) error {
	return rm.statusUpdater.Update(restore, condition, nil)
}

func (rm *restoreManager) syncRestoreJob(restore *v1alpha1.Restore) error {
	ns := restore.GetNamespace()
	name := restore.GetName()

	var (
		err              error
		tc               *v1alpha1.TidbCluster
		restoreNamespace string
	)

	if restore.Spec.BR == nil {
		err = backuputil.ValidateRestore(restore, "")
	} else {
		restoreNamespace = restore.GetNamespace()
		if restore.Spec.BR.ClusterNamespace != "" {
			restoreNamespace = restore.Spec.BR.ClusterNamespace
		}

		tc, err = rm.deps.TiDBClusterLister.TidbClusters(restoreNamespace).Get(restore.Spec.BR.Cluster)
		if err != nil {
			reason := fmt.Sprintf("failed to fetch tidbcluster %s/%s", restoreNamespace, restore.Spec.BR.Cluster)
			rm.statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
				Type:    v1alpha1.RestoreRetryFailed,
				Status:  corev1.ConditionTrue,
				Reason:  reason,
				Message: err.Error(),
			}, nil)
			return err
		}

		tikvImage := tc.TiKVImage()
		err = backuputil.ValidateRestore(restore, tikvImage)
	}

	if err != nil {
		rm.statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreInvalid,
			Status:  corev1.ConditionTrue,
			Reason:  "InvalidSpec",
			Message: err.Error(),
		}, nil)

		return controller.IgnoreErrorf("invalid restore spec %s/%s", ns, name)
	}

	if restore.Spec.BR != nil && restore.Spec.Mode == v1alpha1.RestoreModeVolumeSnapshot {
		// restore based snapshot for cloud provider
		reason, err := rm.tryRestoreIfCanSnapshot(restore, tc)
		if err != nil {
			rm.statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
				Type:    v1alpha1.RestoreRetryFailed,
				Status:  corev1.ConditionTrue,
				Reason:  reason,
				Message: err.Error(),
			}, nil)
			return err
		}
		if !tc.PDAllMembersReady() {
			return controller.RequeueErrorf("restore %s/%s: waiting for all PD members are ready in tidbcluster %s/%s", ns, name, tc.Namespace, tc.Name)
		}
		if v1alpha1.IsRestoreVolumeComplete(restore) && !v1alpha1.IsRestoreDataComplete(restore) && !tc.AllTiKVsAreAvailable() {
			return controller.RequeueErrorf("restore %s/%s: waiting for all TiKVs are available in tidbcluster %s/%s", ns, name, tc.Namespace, tc.Name)
		}
	}

	restoreJobName := restore.GetRestoreJobName()
	_, err = rm.deps.JobLister.Jobs(ns).Get(restoreJobName)
	if err == nil {
		klog.Infof("restore job %s/%s has been created, skip", ns, restoreJobName)
		return nil
	} else if !errors.IsNotFound(err) {
		return fmt.Errorf("restore %s/%s get job %s failed, err: %v", ns, name, restoreJobName, err)
	}

	var (
		job    *batchv1.Job
		reason string
	)
	if restore.Spec.BR == nil {
		job, reason, err = rm.makeImportJob(restore)
		if err != nil {
			rm.statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
				Type:    v1alpha1.RestoreRetryFailed,
				Status:  corev1.ConditionTrue,
				Reason:  reason,
				Message: err.Error(),
			}, nil)
			return err
		}

		reason, err = rm.ensureRestorePVCExist(restore)
		if err != nil {
			rm.statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
				Type:    v1alpha1.RestoreRetryFailed,
				Status:  corev1.ConditionTrue,
				Reason:  reason,
				Message: err.Error(),
			}, nil)
			return err
		}
	} else {
		job, reason, err = rm.makeRestoreJob(restore)
		if err != nil {
			rm.statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
				Type:    v1alpha1.RestoreRetryFailed,
				Status:  corev1.ConditionTrue,
				Reason:  reason,
				Message: err.Error(),
			}, nil)
			return err
		}
	}

	if err := rm.deps.JobControl.CreateJob(restore, job); err != nil {
		errMsg := fmt.Errorf("create restore %s/%s job %s failed, err: %v", ns, name, restoreJobName, err)
		rm.statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreRetryFailed,
			Status:  corev1.ConditionTrue,
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
		return rm.statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
			Type:   v1alpha1.RestoreScheduled,
			Status: corev1.ConditionTrue,
		}, nil)
	}
	return nil
}

func (rm *restoreManager) tryRestoreIfCanSnapshot(r *v1alpha1.Restore, tc *v1alpha1.TidbCluster) (string, error) {
	if v1alpha1.IsRestoreComplete(r) {
		return "", nil
	}

	if v1alpha1.IsRestoreDataComplete(r) {
		klog.Infof("restore-manager prepares to deal with the phase DataComplete")

		// When restore is based on volume snapshot, we need to restart all TiKV pods
		// after restore data is complete.
		sel, err := label.New().Instance(tc.Name).TiKV().Selector()
		if err != nil {
			return "BuildTiKVSelectorFailed", err
		}
		pods, err := rm.deps.PodLister.Pods(tc.Namespace).List(sel)
		if err != nil {
			return "ListTiKVPodsFailed", err
		}
		for _, pod := range pods {
			if pod.DeletionTimestamp == nil {
				klog.Infof("restore-manager restarts pod %s/%s", pod.Namespace, pod.Name)
				if err := rm.deps.PodControl.DeletePod(tc, pod); err != nil {
					return "DeleteTiKVPodFailed", err
				}
			}
		}

		tc.Spec.RecoveryMode = false
		delete(tc.Annotations, label.AnnTiKVVolumesReadyKey)
		if _, err := rm.deps.TiDBClusterControl.Update(tc); err != nil {
			return "ClearTCRecoveryMarkFailed", err
		}

		// restore TidbCluster completed
		if err := rm.statusUpdater.Update(r, &v1alpha1.RestoreCondition{
			Type:   v1alpha1.RestoreComplete,
			Status: corev1.ConditionTrue,
		}, nil); err != nil {
			return "UpdateRestoreCompleteFailed", err
		}
		return "", nil
	}

	if v1alpha1.IsRestoreVolumeComplete(r) {
		klog.Infof("restore-manager prepares to deal with the phase VolumeComplete")

		// TiKV volumes are ready, we can skip prepare restore metadata.
		if _, ok := tc.Annotations[label.AnnTiKVVolumesReadyKey]; ok {
			return "", nil
		}

		s, reason, err := snapshotter.NewSnapshotterForRestore(r.Spec.Mode, rm.deps)
		if err != nil {
			return reason, err
		}
		// setRestoreVolumeID for all PVs, and reset PVC/PVs,
		// then commit all PVC/PVs for TiKV restore volumes
		if reason, err := s.PrepareRestoreMetadata(r); err != nil {
			return reason, err
		}

		restoreMark := fmt.Sprintf("%s/%s", r.Namespace, r.Name)
		if len(tc.GetAnnotations()) == 0 {
			tc.Annotations = make(map[string]string)
		}
		tc.Annotations[label.AnnTiKVVolumesReadyKey] = restoreMark
		if _, err := rm.deps.TiDBClusterControl.Update(tc); err != nil {
			return "AddTCAnnWaitTiKVFailed", err
		}
		return "", nil
	}

	return "", nil
}

func (rm *restoreManager) makeImportJob(restore *v1alpha1.Restore) (*batchv1.Job, string, error) {
	ns := restore.GetNamespace()
	name := restore.GetName()

	envVars, reason, err := backuputil.GenerateTidbPasswordEnv(ns, name, restore.Spec.To.SecretName, restore.Spec.UseKMS, rm.deps.SecretLister)
	if err != nil {
		return nil, reason, err
	}

	storageEnv, reason, err := backuputil.GenerateStorageCertEnv(ns, restore.Spec.UseKMS, restore.Spec.StorageProvider, rm.deps.SecretLister)
	if err != nil {
		return nil, reason, fmt.Errorf("restore %s/%s, %v", ns, name, err)
	}

	backupPath, reason, err := backuputil.GetBackupDataPath(restore.Spec.StorageProvider)
	if err != nil {
		return nil, reason, fmt.Errorf("restore %s/%s, %v", ns, name, err)
	}

	envVars = append(envVars, storageEnv...)
	// set env vars specified in backup.Spec.Env
	envVars = util.AppendOverwriteEnv(envVars, restore.Spec.Env)

	args := []string{
		"import",
		fmt.Sprintf("--namespace=%s", ns),
		fmt.Sprintf("--restoreName=%s", name),
		fmt.Sprintf("--backupPath=%s", backupPath),
	}

	volumeMounts := []corev1.VolumeMount{}
	volumes := []corev1.Volume{}
	initContainers := []corev1.Container{}

	if restore.Spec.To.TLSClientSecretName != nil {
		args = append(args, "--client-tls=true")
		clientSecretName := *restore.Spec.To.TLSClientSecretName
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

	if restore.Spec.ToolImage != "" {
		lightningVolumeMount := corev1.VolumeMount{
			Name:      "lightning-bin",
			ReadOnly:  false,
			MountPath: util.LightningBinPath,
		}
		volumeMounts = append(volumeMounts, lightningVolumeMount)
		volumes = append(volumes, corev1.Volume{
			Name: "lightning-bin",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
		initContainers = append(initContainers, corev1.Container{
			Name:            "lightning",
			Image:           restore.Spec.ToolImage,
			Command:         []string{"/bin/sh", "-c"},
			Args:            []string{fmt.Sprintf("cp /tidb-lightning %s/tidb-lightning; echo 'tidb-lightning copy finished'", util.LightningBinPath)},
			ImagePullPolicy: corev1.PullIfNotPresent,
			VolumeMounts:    []corev1.VolumeMount{lightningVolumeMount},
			Resources:       restore.Spec.ResourceRequirements,
		})
	}

	jobLabels := util.CombineStringMap(label.NewRestore().Instance(restore.GetInstanceName()).RestoreJob().Restore(name), restore.Labels)
	podLabels := jobLabels
	jobAnnotations := restore.Annotations
	podAnnotations := jobAnnotations

	serviceAccount := constants.DefaultServiceAccountName
	if restore.Spec.ServiceAccount != "" {
		serviceAccount = restore.Spec.ServiceAccount
	}

	podSpec := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      podLabels,
			Annotations: podAnnotations,
		},
		Spec: corev1.PodSpec{
			SecurityContext:    restore.Spec.PodSecurityContext,
			ServiceAccountName: serviceAccount,
			InitContainers:     initContainers,
			Containers: []corev1.Container{
				{
					Name:            label.RestoreJobLabelVal,
					Image:           rm.deps.CLIConfig.TiDBBackupManagerImage,
					Args:            args,
					ImagePullPolicy: corev1.PullIfNotPresent,
					VolumeMounts: append([]corev1.VolumeMount{
						{Name: label.RestoreJobLabelVal, MountPath: constants.BackupRootPath},
					}, volumeMounts...),
					Env:       util.AppendEnvIfPresent(envVars, "TZ"),
					Resources: restore.Spec.ResourceRequirements,
				},
			},
			RestartPolicy:    corev1.RestartPolicyNever,
			Tolerations:      restore.Spec.Tolerations,
			ImagePullSecrets: restore.Spec.ImagePullSecrets,
			Affinity:         restore.Spec.Affinity,
			Volumes: append([]corev1.Volume{
				{
					Name: label.RestoreJobLabelVal,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: restore.GetRestorePVCName(),
						},
					},
				},
			}, volumes...),
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
				controller.GetRestoreOwnerRef(restore),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: pointer.Int32Ptr(0),
			Template:     *podSpec,
		},
	}

	return job, "", nil
}

func (rm *restoreManager) makeRestoreJob(restore *v1alpha1.Restore) (*batchv1.Job, string, error) {
	ns := restore.GetNamespace()
	name := restore.GetName()
	restoreNamespace := ns
	if restore.Spec.BR.ClusterNamespace != "" {
		restoreNamespace = restore.Spec.BR.ClusterNamespace
	}
	tc, err := rm.deps.TiDBClusterLister.TidbClusters(restoreNamespace).Get(restore.Spec.BR.Cluster)
	if err != nil {
		return nil, fmt.Sprintf("failed to fetch tidbcluster %s/%s", restoreNamespace, restore.Spec.BR.Cluster), err
	}

	var (
		envVars []corev1.EnvVar
		reason  string
	)
	if restore.Spec.To != nil {
		envVars, reason, err = backuputil.GenerateTidbPasswordEnv(ns, name, restore.Spec.To.SecretName, restore.Spec.UseKMS, rm.deps.SecretLister)
		if err != nil {
			return nil, reason, err
		}
	}

	storageEnv, reason, err := backuputil.GenerateStorageCertEnv(ns, restore.Spec.UseKMS, restore.Spec.StorageProvider, rm.deps.SecretLister)
	if err != nil {
		return nil, reason, fmt.Errorf("restore %s/%s, %v", ns, name, err)
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
	}
	tikvImage := tc.TiKVImage()
	_, tikvVersion := backuputil.ParseImage(tikvImage)
	if tikvVersion != "" {
		args = append(args, fmt.Sprintf("--tikvVersion=%s", tikvVersion))
	}

	switch restore.Spec.Mode {
	case v1alpha1.RestoreModePiTR:
		args = append(args, fmt.Sprintf("--mode=%s", v1alpha1.RestoreModePiTR))
		args = append(args, fmt.Sprintf("--pitrRestoredTs=%s", restore.Spec.PitrRestoredTs))
	case v1alpha1.RestoreModeVolumeSnapshot:
		args = append(args, fmt.Sprintf("--mode=%s", v1alpha1.RestoreModeVolumeSnapshot))
		if !v1alpha1.IsRestoreVolumeComplete(restore) {
			args = append(args, "--prepare")
		}
	default:
		args = append(args, fmt.Sprintf("--mode=%s", v1alpha1.RestoreModeSnapshot))
	}

	jobLabels := util.CombineStringMap(label.NewRestore().Instance(restore.GetInstanceName()).RestoreJob().Restore(name), restore.Labels)
	podLabels := jobLabels
	jobAnnotations := restore.Annotations
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
					SecretName: util.ClusterClientTLSSecretName(restore.Spec.BR.Cluster),
				},
			},
		})
	}

	if restore.Spec.To != nil && tc.Spec.TiDB != nil && tc.Spec.TiDB.TLSClient != nil && tc.Spec.TiDB.TLSClient.Enabled && !tc.SkipTLSWhenConnectTiDB() {
		args = append(args, "--client-tls=true")
		if tc.Spec.TiDB.TLSClient.SkipInternalClientCA {
			args = append(args, "--skipClientCA=true")
		}

		clientSecretName := util.TiDBClientTLSSecretName(restore.Spec.BR.Cluster)
		if restore.Spec.To.TLSClientSecretName != nil {
			clientSecretName = *restore.Spec.To.TLSClientSecretName
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
					Args:            []string{fmt.Sprintf("cp /br %s/br; echo 'BR copy finished'", util.BRBinPath)},
					ImagePullPolicy: corev1.PullIfNotPresent,
					VolumeMounts:    []corev1.VolumeMount{brVolumeMount},
					Resources:       restore.Spec.ResourceRequirements,
				},
			},
			Containers: []corev1.Container{
				{
					Name:            label.RestoreJobLabelVal,
					Image:           rm.deps.CLIConfig.TiDBBackupManagerImage,
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
				controller.GetRestoreOwnerRef(restore),
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: pointer.Int32Ptr(0),
			Template:     *podSpec,
		},
	}

	return job, "", nil
}

func (rm *restoreManager) ensureRestorePVCExist(restore *v1alpha1.Restore) (string, error) {
	ns := restore.GetNamespace()
	name := restore.GetName()

	storageSize := constants.DefaultStorageSize
	if restore.Spec.StorageSize != "" {
		storageSize = restore.Spec.StorageSize
	}
	rs, err := resource.ParseQuantity(storageSize)
	if err != nil {
		errMsg := fmt.Errorf("backup %s/%s parse storage size %s failed, err: %v", ns, name, constants.DefaultStorageSize, err)
		return "ParseStorageSizeFailed", errMsg
	}

	restorePVCName := restore.GetRestorePVCName()
	pvc, err := rm.deps.PVCLister.PersistentVolumeClaims(ns).Get(restorePVCName)
	if err != nil {
		// get the object from the local cache, the error can only be IsNotFound,
		// so we need to create PVC for restore job
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      restorePVCName,
				Namespace: ns,
				Labels:    label.NewRestore().Instance(restore.GetInstanceName()),
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
				StorageClassName: restore.Spec.StorageClassName,
			},
		}
		if err := rm.deps.GeneralPVCControl.CreatePVC(restore, pvc); err != nil {
			errMsg := fmt.Errorf(" %s/%s create restore pvc %s failed, err: %v", ns, name, pvc.GetName(), err)
			return "CreatePVCFailed", errMsg
		}
	} else if pvcRs := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; pvcRs.Cmp(rs) == -1 {
		return "PVCStorageSizeTooSmall", fmt.Errorf("%s/%s's restore pvc %s's storage size %s is less than expected storage size %s, please delete old pvc to continue", ns, name, pvc.GetName(), pvcRs.String(), rs.String())
	}
	return "", nil
}

var _ backup.RestoreManager = &restoreManager{}

type FakeRestoreManager struct {
	err error
}

func NewFakeRestoreManager() *FakeRestoreManager {
	return &FakeRestoreManager{}
}

func (frm *FakeRestoreManager) SetSyncError(err error) {
	frm.err = err
}

func (frm *FakeRestoreManager) Sync(_ *v1alpha1.Restore) error {
	return frm.err
}

func (frm *FakeRestoreManager) UpdateCondition(_ *v1alpha1.Restore, _ *v1alpha1.RestoreCondition) error {
	return nil
}

var _ backup.RestoreManager = &FakeRestoreManager{}
