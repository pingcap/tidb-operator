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
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

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

const (
	TiKVConfigEncryptionMethod      = "security.encryption.data-encryption-method"
	TiKVConfigEncryptionMasterKeyId = "security.encryption.master-key.key-id"
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
		err = backuputil.ValidateRestore(restore, "", false)
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
		err = backuputil.ValidateRestore(restore, tikvImage, tc.Spec.AcrossK8s)
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
		err = rm.validateRestore(restore, tc)
		if err != nil {
			rm.statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
				Type:    v1alpha1.RestoreInvalid,
				Status:  corev1.ConditionTrue,
				Reason:  "InvalidSpec",
				Message: err.Error(),
			}, nil)
			return err
		}
		reason, err := rm.volumeSnapshotRestore(restore, tc)
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

		if v1alpha1.IsRestoreVolumeComplete(restore) && !v1alpha1.IsRestoreTiKVComplete(restore) {
			if isWarmUpSync(restore) {
				if !v1alpha1.IsRestoreWarmUpStarted(restore) {
					return rm.warmUpTiKVVolumesSync(restore, tc)
				}
				if !v1alpha1.IsRestoreWarmUpComplete(restore) {
					return rm.waitWarmUpJobsFinished(restore)
				}
				if restore.Spec.WarmupStrategy == v1alpha1.RestoreWarmupStrategyCheckOnly {
					return rm.checkWALOnlyFinish(restore)
				}
			} else if isWarmUpAsync(restore) {
				if !v1alpha1.IsRestoreWarmUpStarted(restore) {
					return rm.warmUpTiKVVolumesAsync(restore, tc)
				}
			}

			if !tc.AllTiKVsAreAvailable(restore.Spec.TolerateSingleTiKVOutage) {
				return controller.RequeueErrorf("restore %s/%s: waiting for all TiKVs are available in tidbcluster %s/%s", ns, name, tc.Namespace, tc.Name)
			} else {
				return rm.statusUpdater.Update(restore, &v1alpha1.RestoreCondition{
					Type:   v1alpha1.RestoreTiKVComplete,
					Status: corev1.ConditionTrue,
				}, nil)
			}
		}

		if isWarmUpAsync(restore) && v1alpha1.IsRestoreWarmUpStarted(restore) && !v1alpha1.IsRestoreWarmUpComplete(restore) {
			if err := rm.waitWarmUpJobsFinished(restore); err != nil {
				return err
			}
		}

		if v1alpha1.IsRestoreTiKVComplete(restore) && restore.Spec.FederalVolumeRestorePhase == v1alpha1.FederalVolumeRestoreVolume {
			return nil
		}

		if restore.Spec.FederalVolumeRestorePhase == v1alpha1.FederalVolumeRestoreFinish {
			if !v1alpha1.IsRestoreComplete(restore) {
				return controller.RequeueErrorf("restore %s/%s: waiting for restore status complete in tidbcluster %s/%s", ns, name, tc.Namespace, tc.Name)
			}
			return nil
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

// read cluster meta from external storage since k8s size limitation on annotation/configMap
// after volume restore job complete, br output a meta file for controller to reconfig the tikvs
// since the meta file may big, so we use remote storage as bridge to pass it from restore manager to controller
func (rm *restoreManager) readRestoreMetaFromExternalStorage(r *v1alpha1.Restore) (*snapshotter.CloudSnapBackup, string, error) {
	// since the restore meta is small (~5M), assume 1 minutes is enough
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Minute*1))
	defer cancel()

	// read restore meta from output of BR 1st restore
	klog.Infof("read the restore meta from external storage")
	cred := backuputil.GetStorageCredential(r.Namespace, r.Spec.StorageProvider, rm.deps.SecretLister)
	externalStorage, err := backuputil.NewStorageBackend(r.Spec.StorageProvider, cred)
	if err != nil {
		return nil, "NewStorageBackendFailed", err
	}

	// if file doesn't exist, br create volume has problem
	exist, err := externalStorage.Exists(ctx, constants.ClusterRestoreMeta)
	if err != nil {
		return nil, "FileExistedInExternalStorageFailed", err
	}
	if !exist {
		return nil, "FileNotExists", fmt.Errorf("%s does not exist", constants.ClusterRestoreMeta)
	}

	restoreMeta, err := externalStorage.ReadAll(ctx, constants.ClusterRestoreMeta)
	if err != nil {
		return nil, "ReadAllOnExternalStorageFailed", err
	}

	csb := &snapshotter.CloudSnapBackup{}
	err = json.Unmarshal(restoreMeta, csb)
	if err != nil {
		return nil, "ParseCloudSnapBackupFailed", err
	}

	return csb, "", nil
}
func (rm *restoreManager) validateRestore(r *v1alpha1.Restore, tc *v1alpha1.TidbCluster) error {
	// check tiflash and tikv replicas
	err := rm.checkTiFlashAndTiKVReplicasFromBackupMeta(r, tc)
	if err != nil {
		klog.Errorf("check tikv/tiflash failure with reason %v", err)
		return err
	}

	// Check recovery mode is on for EBS br across k8s
	if r.Spec.Mode == v1alpha1.RestoreModeVolumeSnapshot && r.Spec.FederalVolumeRestorePhase != v1alpha1.FederalVolumeRestoreFinish && !tc.Spec.RecoveryMode {
		klog.Errorf("recovery mode is not set for across k8s EBS snapshot restore")
		return fmt.Errorf("recovery mode is off")
	}

	// check tikv encrypt config
	if err = rm.checkTiKVEncryption(r, tc); err != nil {
		return fmt.Errorf("TiKV encryption missmatched with backup with error %v", err)
	}
	return nil
}

// volume snapshot restore support
//
//	both backup and restore with the same encryption
//	backup without encryption and restore has encryption
//
// volume snapshot restore does not support
//
//	backup has encryption and restore has not
func (rm *restoreManager) checkTiKVEncryption(r *v1alpha1.Restore, tc *v1alpha1.TidbCluster) error {
	backupConfig, reason, err := rm.readTiKVConfigFromBackupMeta(r)
	if err != nil {
		klog.Errorf("read tiflash replica failure with reason %s", reason)
		return err
	}

	// nothing configured in crd during the backup
	if backupConfig == nil {
		return nil
	}

	// check if encryption is enabled in backup tikv config
	backupEncryptMethod := backupConfig.Get(TiKVConfigEncryptionMethod)
	if backupEncryptMethod == nil || backupEncryptMethod.Interface() == "plaintext" {
		return nil //encryption is disabled
	}

	// tikv backup encryption is enabled
	config := tc.Spec.TiKV.Config
	if config == nil {
		return fmt.Errorf("TiKV encryption config missmatched, backup configured TiKV encryption, however, restore tc.spec.tikv.config doesn't contains encryption, please check TiKV encryption config. e.g. download s3 backupmeta, check kubernetes.crd_tidb_cluster.spec, and then edit restore tc.")
	}

	restoreEncryptMethod := config.Get(TiKVConfigEncryptionMethod)
	if backupEncryptMethod.Interface() != restoreEncryptMethod.Interface() {
		// restore crd must contains data-encryption
		return fmt.Errorf("TiKV encryption config missmatched, backup data enabled TiKV encryption, restore crd does not enabled TiKV encryption")
	}

	// if backup tikv configured encryption, restore require tc to have the same encryption configured.
	// since master key is is unique, only check master key id is enough. e.g. https://docs.aws.amazon.com/kms/latest/cryptographic-details/basic-concepts.html
	backupMasterKey := backupConfig.Get(TiKVConfigEncryptionMasterKeyId)
	if backupMasterKey != nil {
		restoreMasterKey := config.Get(TiKVConfigEncryptionMasterKeyId)
		if restoreMasterKey == nil {
			return fmt.Errorf("TiKV encryption config missmatched, backup data has master key, restore crd have not one")
		}

		if backupMasterKey.Interface() != restoreMasterKey.Interface() {
			return fmt.Errorf("TiKV encryption config master key missmatched")
		}
	}
	return nil
}

func (rm *restoreManager) checkTiFlashAndTiKVReplicasFromBackupMeta(r *v1alpha1.Restore, tc *v1alpha1.TidbCluster) error {
	metaInfo, err := backuputil.GetVolSnapBackupMetaData(r, rm.deps.SecretLister)
	if err != nil {
		klog.Errorf("GetVolSnapBackupMetaData failed")
		return err
	}

	// Check mismatch of tiflash config
	if (metaInfo.KubernetesMeta.TiDBCluster.Spec.TiFlash == nil ||
		metaInfo.KubernetesMeta.TiDBCluster.Spec.TiFlash.Replicas == 0) &&
		tc.Spec.TiFlash != nil && tc.Spec.TiFlash.Replicas > 0 {
		klog.Errorf("tiflash is enabled in TC but disabled in backup metadata")
		return fmt.Errorf("tiflash replica mismatched")
	} else if (tc.Spec.TiFlash == nil || tc.Spec.TiFlash.Replicas == 0) &&
		metaInfo.KubernetesMeta.TiDBCluster.Spec.TiFlash != nil &&
		metaInfo.KubernetesMeta.TiDBCluster.Spec.TiFlash.Replicas > 0 {
		klog.Errorf("tiflash is disabled in TC enabled in backup metadata")
		return fmt.Errorf("tiflash replica mismatched")
	} else if tc.Spec.TiFlash != nil && metaInfo.KubernetesMeta.TiDBCluster.Spec.TiFlash != nil &&
		tc.Spec.TiFlash.Replicas != metaInfo.KubernetesMeta.TiDBCluster.Spec.TiFlash.Replicas {
		klog.Errorf("tiflash number in TC is %d but is %d in backup metadata", tc.Spec.TiFlash.Replicas, metaInfo.KubernetesMeta.TiDBCluster.Spec.TiFlash.Replicas)
		return fmt.Errorf("tiflash replica mismatched")
	}

	// TiKV node must be there
	if tc.Spec.TiKV == nil || tc.Spec.TiKV.Replicas == 0 {
		klog.Errorf("ebs snapshot restore doesn't support cluster without tikv nodes")
		return fmt.Errorf("restore to no tikv cluster")
	} else if metaInfo.KubernetesMeta.TiDBCluster.Spec.TiKV == nil ||
		metaInfo.KubernetesMeta.TiDBCluster.Spec.TiKV.Replicas == 0 {
		klog.Errorf("backup source tc has no tikv nodes")
		return fmt.Errorf("backup source tc has no tivk nodes")
	} else if tc.Spec.TiKV.Replicas != metaInfo.KubernetesMeta.TiDBCluster.Spec.TiKV.Replicas ||
		(r.Spec.TolerateSingleTiKVOutage && metaInfo.KubernetesMeta.TiDBCluster.Spec.TiKV.Replicas-1 == tc.Spec.TiKV.Replicas) {
		klog.Errorf("mismatch tikv replicas, tc has %d, while backup has %d", tc.Spec.TiKV.Replicas, metaInfo.KubernetesMeta.TiDBCluster.Spec.TiKV)
		return fmt.Errorf("tikv replica mismatch")
	}

	// Check volume number
	if len(tc.Spec.TiKV.StorageVolumes) != len(metaInfo.KubernetesMeta.TiDBCluster.Spec.TiKV.StorageVolumes) {
		klog.Errorf("additional volumes # not match. tc has %d, and backup has %d", len(tc.Spec.TiKV.StorageVolumes), len(metaInfo.KubernetesMeta.TiDBCluster.Spec.TiKV.StorageVolumes))
		return fmt.Errorf("additional volumes mismatched")
	}

	return nil
}

func (rm *restoreManager) readTiKVConfigFromBackupMeta(r *v1alpha1.Restore) (*v1alpha1.TiKVConfigWraper, string, error) {
	metaInfo, err := backuputil.GetVolSnapBackupMetaData(r, rm.deps.SecretLister)
	if err != nil {
		return nil, "GetVolSnapBackupMetaData failed", err
	}

	if metaInfo.KubernetesMeta.TiDBCluster.Spec.TiKV == nil {
		return nil, "BackupMetaDoesnotContainTiKV", fmt.Errorf("TiKV is not configure in backup")
	}

	return metaInfo.KubernetesMeta.TiDBCluster.Spec.TiKV.Config, "", nil
}

func (rm *restoreManager) volumeSnapshotRestore(r *v1alpha1.Restore, tc *v1alpha1.TidbCluster) (string, error) {
	if v1alpha1.IsRestoreComplete(r) {
		return "", nil
	}

	ns := r.Namespace
	name := r.Name
	if r.Spec.FederalVolumeRestorePhase == v1alpha1.FederalVolumeRestoreFinish {
		klog.Infof("%s/%s restore-manager prepares to deal with the phase restore-finish", ns, name)

		if !tc.Spec.RecoveryMode {
			klog.Infof("%s/%s recovery mode of tc %s/%s is false, ignore restore-finish phase", ns, name, tc.Namespace, tc.Name)
			return "", nil
		}
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
				klog.Infof("%s/%s restore-manager restarts pod %s/%s", ns, name, pod.Namespace, pod.Name)
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
		newStatus := &controller.RestoreUpdateStatus{
			TimeCompleted: &metav1.Time{Time: time.Now()},
		}
		if err := rm.statusUpdater.Update(r, &v1alpha1.RestoreCondition{
			Type:   v1alpha1.RestoreComplete,
			Status: corev1.ConditionTrue,
		}, newStatus); err != nil {
			return "UpdateRestoreCompleteFailed", err
		}
		return "", nil
	}

	if v1alpha1.IsRestoreVolumeComplete(r) && r.Spec.FederalVolumeRestorePhase == v1alpha1.FederalVolumeRestoreVolume {
		klog.Infof("%s/%s restore-manager prepares to deal with the phase VolumeComplete", ns, name)

		// TiKV volumes are ready, we can skip prepare restore metadata.
		if _, ok := tc.Annotations[label.AnnTiKVVolumesReadyKey]; ok {
			return "", nil
		}

		if isWarmUpSync(r) {
			if v1alpha1.IsRestoreWarmUpComplete(r) {

				return rm.startTiKVIfNeeded(r, tc)
			}
			if v1alpha1.IsRestoreWarmUpStarted(r) {
				return "", nil
			}
		}

		s, reason, err := snapshotter.NewSnapshotterForRestore(r.Spec.Mode, rm.deps)
		if err != nil {
			return reason, err
		}
		// setRestoreVolumeID for all PVs, and reset PVC/PVs,
		// then commit all PVC/PVs for TiKV restore volumes
		csb, reason, err := rm.readRestoreMetaFromExternalStorage(r)
		if err != nil {
			return reason, err
		}

		if reason, err := s.PrepareRestoreMetadata(r, csb); err != nil {
			return reason, err
		}
		klog.Infof("Restore %s/%s prepare restore metadata finished", r.Namespace, r.Name)
		if !isWarmUpSync(r) {
			return rm.startTiKVIfNeeded(r, tc)
		}
	}

	return "", nil
}

func (rm *restoreManager) startTiKVIfNeeded(r *v1alpha1.Restore, tc *v1alpha1.TidbCluster) (reason string, err error) {
	if r.Spec.WarmupStrategy == v1alpha1.RestoreWarmupStrategyCheckOnly {
		return "", nil
	}
	restoreMark := fmt.Sprintf("%s/%s", r.Namespace, r.Name)
	if len(tc.GetAnnotations()) == 0 {
		tc.Annotations = make(map[string]string)
	}
	tc.Annotations[label.AnnTiKVVolumesReadyKey] = restoreMark
	if _, err := rm.deps.TiDBClusterControl.Update(tc); err != nil {
		return "AddTCAnnWaitTiKVFailed", err
	}
	klog.Infof("Restore %s/%s start TiKV", r.Namespace, r.Name)
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

	if len(restore.Spec.AdditionalVolumes) > 0 {
		volumes = append(volumes, restore.Spec.AdditionalVolumes...)
	}
	if len(restore.Spec.AdditionalVolumeMounts) > 0 {
		volumeMounts = append(volumeMounts, restore.Spec.AdditionalVolumeMounts...)
	}

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
			if restore.Spec.VolumeAZ != "" {
				args = append(args, fmt.Sprintf("--target-az=%s", restore.Spec.VolumeAZ))
			}
			if restore.Spec.WarmupStrategy == v1alpha1.RestoreWarmupStrategyFsr {
				args = append(args, "--use-fsr=true")
			}
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

		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "tidb-client-tls",
			ReadOnly:  true,
			MountPath: util.TiDBClientTLSPath,
		})
		volumes = append(volumes, corev1.Volume{
			Name: "tidb-client-tls",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: util.TiDBClientTLSSecretName(restore.Spec.BR.Cluster, restore.Spec.To.TLSClientSecretName),
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

type pvcInfo struct {
	pvc        *corev1.PersistentVolumeClaim
	volumeName string
	number     int
}

func (rm *restoreManager) warmUpTiKVVolumesSync(r *v1alpha1.Restore, tc *v1alpha1.TidbCluster) error {
	warmUpImage := r.Spec.WarmupImage
	if warmUpImage == "" {
		rm.statusUpdater.Update(r, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreRetryFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "NoWarmupImage",
			Message: "warmup image is empty",
		}, nil)
		return fmt.Errorf("warmup image is empty")
	}

	klog.Infof("Restore %s/%s start to warm up TiKV synchronously", r.Namespace, r.Name)
	ns := tc.Namespace
	sel, err := label.New().Instance(tc.Name).TiKV().Selector()
	if err != nil {
		return err
	}
	pvcs, err := rm.deps.PVCLister.PersistentVolumeClaims(ns).List(sel)
	if err != nil {
		return err
	}

	stsName := controller.TiKVMemberName(tc.Name)
	reStr := fmt.Sprintf(`^(.+)-%s-(\d+)$`, stsName)
	re := regexp.MustCompile(reStr)
	pvcInfoMap := make(map[int][]*pvcInfo, len(pvcs))
	for _, pvc := range pvcs {
		subMatches := re.FindStringSubmatch(pvc.Name)
		if len(subMatches) != 3 {
			return fmt.Errorf("pvc name %s doesn't match regex %s", pvc.Name, reStr)
		}
		volumeName := subMatches[1]
		numberStr := subMatches[2]
		number, err := strconv.Atoi(numberStr)
		if err != nil {
			return fmt.Errorf("parse index %s of pvc %s to int: %s", numberStr, pvc.Name, err.Error())
		}
		pvcInfoMap[number] = append(pvcInfoMap[number], &pvcInfo{
			pvc:        pvc,
			volumeName: volumeName,
			number:     number,
		})
		klog.Infof("Restore %s/%s warmup get pvc %s/%s", r.Namespace, r.Name, pvc.Namespace, pvc.Name)
	}

	volumesCount := len(tc.Spec.TiKV.StorageVolumes) + 1
	for number, podPVCs := range pvcInfoMap {
		if len(podPVCs) != volumesCount {
			return fmt.Errorf("expected pvc count %d, got pvc count %d, not equal", volumesCount, len(podPVCs))
		}
		warmUpJobName := fmt.Sprintf("%s-%s-%d-warm-up", r.Name, stsName, number)
		_, err := rm.deps.JobLister.Jobs(ns).Get(warmUpJobName)
		if err == nil {
			klog.Infof("Restore %s/%s warmup job %s/%s exists, pass it", r.Namespace, r.Name, ns, warmUpJobName)
			continue
		} else if !errors.IsNotFound(err) {
			return fmt.Errorf("get warm up job %s/%s error: %s", ns, warmUpJobName, err.Error())
		}

		warmUpJob, err := rm.makeSyncWarmUpJob(r, tc, podPVCs, warmUpJobName, warmUpImage)
		if err != nil {
			return err
		}
		if err = rm.deps.JobControl.CreateJob(r, warmUpJob); err != nil {
			return err
		}
		klog.Infof("Restore %s/%s creates warmup job %s/%s successfully", r.Namespace, r.Name, ns, warmUpJobName)
	}
	return rm.statusUpdater.Update(r, &v1alpha1.RestoreCondition{
		Type:   v1alpha1.RestoreWarmUpStarted,
		Status: corev1.ConditionTrue,
	}, nil)
}

func (rm *restoreManager) warmUpTiKVVolumesAsync(r *v1alpha1.Restore, tc *v1alpha1.TidbCluster) error {
	warmUpImage := r.Spec.WarmupImage
	if warmUpImage == "" {
		rm.statusUpdater.Update(r, &v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreRetryFailed,
			Status:  corev1.ConditionTrue,
			Reason:  "NoWarmupImage",
			Message: "warmup image is empty",
		}, nil)
		return fmt.Errorf("warmup image is empty")
	}

	klog.Infof("Restore %s/%s start to warm up TiKV asynchronously", r.Namespace, r.Name)
	sel, err := label.New().Instance(tc.Name).TiKV().Selector()
	if err != nil {
		return err
	}
	tikvPods, err := rm.deps.PodLister.Pods(tc.Namespace).List(sel)
	if err != nil {
		return err
	}
	if int32(len(tikvPods)) != tc.Spec.TiKV.Replicas {
		return fmt.Errorf("wait all TiKV pods started to warm up volumes")
	}
	for _, pod := range tikvPods {
		if pod.Status.Phase != corev1.PodRunning {
			return fmt.Errorf("wait TiKV pod %s/%s running", pod.Namespace, pod.Name)
		}
	}

	tikvMountPaths := []string{constants.TiKVDataVolumeMountPath}
	for _, vol := range tc.Spec.TiKV.StorageVolumes {
		tikvMountPaths = append(tikvMountPaths, vol.MountPath)
	}
	for _, pod := range tikvPods {
		ns, podName := pod.Namespace, pod.Name
		warmUpJobName := fmt.Sprintf("%s-%s-warm-up", r.Name, podName)
		_, err := rm.deps.JobLister.Jobs(ns).Get(warmUpJobName)
		if err == nil {
			klog.Infof("Restore %s/%s warmup job %s/%s of tikv pod %s/%s exists, pass it", r.Namespace, r.Name, ns, warmUpJobName, ns, podName)
			continue
		} else if !errors.IsNotFound(err) {
			return fmt.Errorf("get warm up job %s/%s of tikv pod %s/%s error: %s", ns, warmUpJobName, ns, podName, err.Error())
		}

		warmUpJob, err := rm.makeAsyncWarmUpJob(r, pod, tikvMountPaths, warmUpJobName, warmUpImage)
		if err != nil {
			return err
		}
		if err = rm.deps.JobControl.CreateJob(r, warmUpJob); err != nil {
			return err
		}
		klog.Infof("Restore %s/%s creates warmup job %s/%s for tikv pod %s/%s successfully", r.Namespace, r.Name, ns, warmUpJobName, ns, podName)
	}

	return rm.statusUpdater.Update(r, &v1alpha1.RestoreCondition{
		Type:   v1alpha1.RestoreWarmUpStarted,
		Status: corev1.ConditionTrue,
	}, nil)
}

func generateWarmUpArgs(strategy v1alpha1.RestoreWarmupStrategy, mountPoints []corev1.VolumeMount) ([]string, error) {
	res := make([]string, 0, len(mountPoints))
	for _, p := range mountPoints {
		switch strategy {
		case v1alpha1.RestoreWarmupStrategyFio:
			res = append(res, "--block", p.MountPath)
		case v1alpha1.RestoreWarmupStrategyHybrid:
			if p.MountPath == constants.TiKVDataVolumeMountPath {
				res = append(res, "--fs", constants.TiKVDataVolumeMountPath)
			} else {
				res = append(res, "--block", p.MountPath)
			}
		case v1alpha1.RestoreWarmupStrategyFsr, v1alpha1.RestoreWarmupStrategyCheckOnly:
			if p.MountPath == constants.TiKVDataVolumeMountPath {
				// data volume has been warmed up by enabling FSR or can be skipped
				continue
			} else {
				res = append(res, "--block", p.MountPath)
			}
		default:
			return nil, fmt.Errorf("unknown warmup strategy %q", strategy)
		}
	}
	return res, nil
}

func (rm *restoreManager) makeSyncWarmUpJob(r *v1alpha1.Restore, tc *v1alpha1.TidbCluster, pvcs []*pvcInfo, warmUpJobName, warmUpImage string) (*batchv1.Job, error) {
	podVolumes := make([]corev1.Volume, 0, len(pvcs))
	podVolumeMounts := make([]corev1.VolumeMount, 0, len(pvcs))
	for _, pvc := range pvcs {
		podVolumes = append(podVolumes, corev1.Volume{
			Name: pvc.volumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.pvc.Name,
				},
			},
		})

		mountPath := fmt.Sprintf("/var/lib/%s", pvc.volumeName)
		podVolumeMounts = append(podVolumeMounts, corev1.VolumeMount{
			Name:      pvc.volumeName,
			MountPath: mountPath,
		})
	}

	// all the warmup pods will attach the additional volumes
	// if the volume source can't be attached to multi pods, the warmup pods may be stuck.
	if len(r.Spec.AdditionalVolumes) > 0 {
		podVolumes = append(podVolumes, r.Spec.AdditionalVolumes...)
	}
	if len(r.Spec.AdditionalVolumeMounts) > 0 {
		podVolumeMounts = append(podVolumeMounts, r.Spec.AdditionalVolumeMounts...)
	}

	nodeSelector := make(map[string]string, len(tc.Spec.TiKV.NodeSelector))
	for k, v := range tc.Spec.NodeSelector {
		nodeSelector[k] = v
	}
	for k, v := range tc.Spec.TiKV.NodeSelector {
		nodeSelector[k] = v
	}

	tolerations := make([]corev1.Toleration, 0, len(tc.Spec.TiKV.Tolerations))
	for _, toleration := range tc.Spec.TiKV.Tolerations {
		tolerations = append(tolerations, *toleration.DeepCopy())
	}
	if len(tolerations) == 0 {
		// if the tolerations of tikv is empty, use the tolerations of tidb cluster
		for _, toleration := range tc.Spec.Tolerations {
			tolerations = append(tolerations, *toleration.DeepCopy())
		}
	}

	args, err := generateWarmUpArgs(r.Spec.WarmupStrategy, podVolumeMounts)
	if err != nil {
		return nil, err
	}
	resourceRequirements := getWarmUpResourceRequirements(tc)

	podAnnotations := r.Annotations
	podLabels := r.Labels

	warmUpPod := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: podAnnotations,
			Labels:      podLabels,
		},
		Spec: corev1.PodSpec{
			Volumes:       podVolumes,
			RestartPolicy: corev1.RestartPolicyNever,
			Affinity:      tc.Spec.TiKV.Affinity.DeepCopy(),
			NodeSelector:  nodeSelector,
			Tolerations:   tolerations,
			Containers: []corev1.Container{
				{
					Name:            "warm-up",
					Image:           warmUpImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/warmup_steps"},
					Args:            args,
					Resources:       *resourceRequirements,
					VolumeMounts:    podVolumeMounts,
					SecurityContext: &corev1.SecurityContext{
						Privileged: pointer.BoolPtr(true),
					},
				},
			},
		},
	}

	jobLabel := label.NewRestore().RestoreWarmUpJob().Restore(r.Name)
	warmUpJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      warmUpJobName,
			Namespace: tc.Namespace,
			Labels:    jobLabel,
			OwnerReferences: []metav1.OwnerReference{
				controller.GetRestoreOwnerRef(r),
			},
		},
		Spec: batchv1.JobSpec{
			Template:     *warmUpPod,
			BackoffLimit: pointer.Int32Ptr(1),
		},
	}
	return warmUpJob, nil
}

func (rm *restoreManager) makeAsyncWarmUpJob(r *v1alpha1.Restore, tikvPod *corev1.Pod, warmUpPaths []string, warmUpJobName, warmUpImage string) (*batchv1.Job, error) {
	ns, podName := tikvPod.Namespace, tikvPod.Name
	var tikvContainer *corev1.Container
	for _, container := range tikvPod.Spec.Containers {
		if container.Name == v1alpha1.TiKVMemberType.String() {
			tikvContainer = container.DeepCopy()
			break
		}
	}
	if tikvContainer == nil {
		return nil, fmt.Errorf("not found TiKV container in pod %s/%s", ns, podName)
	}

	warmUpVolumeMounts := make([]corev1.VolumeMount, 0, len(warmUpPaths))
	for _, mountPath := range warmUpPaths {
		for _, volumeMount := range tikvContainer.VolumeMounts {
			if strings.TrimRight(volumeMount.MountPath, string(filepath.Separator)) == strings.TrimRight(mountPath, string(filepath.Separator)) {
				warmUpVolumeMounts = append(warmUpVolumeMounts, *volumeMount.DeepCopy())
				break
			}
		}
	}

	warmUpVolumes := make([]corev1.Volume, 0, len(warmUpVolumeMounts))
	for _, volumeMount := range warmUpVolumeMounts {
		for _, volume := range tikvPod.Spec.Volumes {
			if volumeMount.Name == volume.Name {
				warmUpVolumes = append(warmUpVolumes, *volume.DeepCopy())
				break
			}
		}
	}

	// all the warmup pods will attach the additional volumes
	// if the volume source can't be attached to multi pods, the warmup pods may be stuck.
	if len(r.Spec.AdditionalVolumes) > 0 {
		warmUpVolumes = append(warmUpVolumes, r.Spec.AdditionalVolumes...)
	}
	if len(r.Spec.AdditionalVolumeMounts) > 0 {
		warmUpVolumeMounts = append(warmUpVolumeMounts, r.Spec.AdditionalVolumeMounts...)
	}

	args, err := generateWarmUpArgs(r.Spec.WarmupStrategy, warmUpVolumeMounts)
	if err != nil {
		return nil, err
	}

	podAnnotations := r.Annotations
	podLabels := r.Labels

	warmUpPod := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: podAnnotations,
			Labels:      podLabels,
		},
		Spec: corev1.PodSpec{
			Volumes:       warmUpVolumes,
			RestartPolicy: corev1.RestartPolicyNever,
			NodeName:      tikvPod.Spec.NodeName,
			Containers: []corev1.Container{
				{
					Name:            "warm-up",
					Image:           warmUpImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/warmup_steps"},
					Args:            args,
					VolumeMounts:    warmUpVolumeMounts,
					SecurityContext: &corev1.SecurityContext{
						Privileged: pointer.BoolPtr(true),
					},
				},
			},
		},
	}

	jobLabel := label.NewRestore().RestoreWarmUpJob().Restore(r.Name)
	jobLabel[label.RestoreWarmUpLabelKey] = podName
	warmUpJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      warmUpJobName,
			Namespace: ns,
			Labels:    jobLabel,
			OwnerReferences: []metav1.OwnerReference{
				controller.GetRestoreOwnerRef(r),
			},
		},
		Spec: batchv1.JobSpec{
			Template:     *warmUpPod,
			BackoffLimit: pointer.Int32Ptr(1),
		},
	}
	return warmUpJob, nil
}

func (rm *restoreManager) checkWALOnlyFinish(r *v1alpha1.Restore) error {
	newStatus := &controller.RestoreUpdateStatus{
		TimeCompleted: &metav1.Time{Time: time.Now()},
	}
	if err := rm.statusUpdater.Update(r, &v1alpha1.RestoreCondition{
		Type:   v1alpha1.RestoreComplete,
		Status: corev1.ConditionTrue,
	}, newStatus); err != nil {
		return fmt.Errorf("UpdateRestoreCompleteFailed for %s", err.Error())
	}
	return nil
}

func (rm *restoreManager) waitWarmUpJobsFinished(r *v1alpha1.Restore) error {
	if r.Spec.Warmup == "" {
		return nil
	}

	sel, err := label.NewRestore().RestoreWarmUpJob().Restore(r.Name).Selector()
	if err != nil {
		return err
	}

	jobs, err := rm.deps.JobLister.Jobs(r.Namespace).List(sel)
	if err != nil {
		return err
	}
	if len(jobs) == 0 {
		return fmt.Errorf("waiting for warmup job being created")
	}
	for _, job := range jobs {
		finished := false
		for _, condition := range job.Status.Conditions {
			if condition.Type == batchv1.JobFailed {
				err := fmt.Errorf("warmup job %s/%s failed", job.Namespace, job.Name)
				rm.statusUpdater.Update(r, &v1alpha1.RestoreCondition{
					Type:    v1alpha1.RestoreFailed,
					Status:  corev1.ConditionTrue,
					Reason:  condition.Reason,
					Message: err.Error(),
				}, nil)
				return err
			}
			if condition.Type == batchv1.JobComplete {
				finished = true
				continue
			}
		}
		if !finished {
			if isWarmUpAsync(r) {
				return nil
			}
			return fmt.Errorf("waiting for warmup job %s/%s to finish", job.Namespace, job.Name)
		}
	}

	return rm.statusUpdater.Update(r, &v1alpha1.RestoreCondition{
		Type:   v1alpha1.RestoreWarmUpComplete,
		Status: corev1.ConditionTrue,
	}, nil)
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

func getWarmUpResourceRequirements(tc *v1alpha1.TidbCluster) *corev1.ResourceRequirements {
	tikvResourceRequirements := tc.Spec.TiKV.ResourceRequirements.DeepCopy()
	warmUpResourceRequirements := &corev1.ResourceRequirements{
		Requests: make(corev1.ResourceList, 4),
		Limits:   make(corev1.ResourceList, 4),
	}
	if quantity, ok := tikvResourceRequirements.Limits[corev1.ResourceCPU]; ok {
		warmUpResourceRequirements.Limits[corev1.ResourceCPU] = quantity
	}
	if quantity, ok := tikvResourceRequirements.Limits[corev1.ResourceMemory]; ok {
		warmUpResourceRequirements.Limits[corev1.ResourceMemory] = quantity
	}
	if quantity, ok := tikvResourceRequirements.Requests[corev1.ResourceCPU]; ok {
		warmUpResourceRequirements.Requests[corev1.ResourceCPU] = quantity
	}
	if quantity, ok := tikvResourceRequirements.Requests[corev1.ResourceMemory]; ok {
		warmUpResourceRequirements.Requests[corev1.ResourceMemory] = quantity
	}
	return warmUpResourceRequirements
}

func isWarmUpSync(r *v1alpha1.Restore) bool {
	return r.Spec.Warmup == v1alpha1.RestoreWarmupModeSync
}

func isWarmUpAsync(r *v1alpha1.Restore) bool {
	return r.Spec.Warmup == v1alpha1.RestoreWarmupModeASync
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
