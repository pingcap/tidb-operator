// Copyright 2023 PingCAP, Inc.
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
	"math"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"

	"github.com/dustin/go-humanize"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	pingcapv1alpha1 "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/fedvolumebackup"
)

const (
	reasonKubeConfigNotFound       = "KubeConfigNotFound"
	reasonVolumeBackupMemberFailed = "VolumeBackupMemberFailed"
)

type backupManager struct {
	deps *controller.BrFedDependencies
}

// NewBackupManager return backupManager
func NewBackupManager(deps *controller.BrFedDependencies) fedvolumebackup.BackupManager {
	return &backupManager{
		deps: deps,
	}
}

func (bm *backupManager) Sync(volumeBackup *v1alpha1.VolumeBackup) error {
	// because a finalizer is installed on the VolumeBackup on creation, when the VolumeBackup is deleted,
	// volumeBackup.DeletionTimestamp will be set, controller will be informed with an onUpdate event,
	// this is the moment that we can do clean up work.
	if volumeBackup.DeletionTimestamp != nil {
		return bm.cleanVolumeBackup(volumeBackup)
	}

	return bm.syncBackup(volumeBackup)
}

// UpdateStatus updates the status for a Backup, include condition and status info.
func (bm *backupManager) UpdateStatus(backup *v1alpha1.VolumeBackup, newStatus *v1alpha1.VolumeBackupStatus) error {
	name := backup.Name
	ns := backup.Namespace
	ctx := context.Background()
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		latestBackup, err := bm.deps.Clientset.FederationV1alpha1().VolumeBackups(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			klog.Warningf("backup %s/%s get backup error: %s", ns, name, err.Error())
			return err
		}
		if apiequality.Semantic.DeepEqual(&latestBackup.Status, newStatus) {
			return nil
		}
		latestBackup.Status = *newStatus
		_, err = bm.deps.Clientset.FederationV1alpha1().VolumeBackups(ns).Update(ctx, latestBackup, metav1.UpdateOptions{})
		return err
	})
}

func (bm *backupManager) syncBackup(volumeBackup *v1alpha1.VolumeBackup) error {
	ns := volumeBackup.GetNamespace()
	name := volumeBackup.GetName()
	klog.Infof("sync VolumeBackup %s/%s", ns, name)

	if !v1alpha1.IsVolumeBackupRunning(volumeBackup) {
		bm.setVolumeBackupRunning(&volumeBackup.Status)
	}

	ctx := context.Background()
	backupMembers, err := bm.listAllBackupMembers(ctx, volumeBackup)
	if err != nil {
		return err
	}

	if len(backupMembers) == 0 {
		return bm.initializeVolumeBackup(ctx, volumeBackup)
	}
	if err := bm.waitBackupMemberInitialized(ctx, volumeBackup, backupMembers); err != nil {
		return err
	}

	newMemberCreated, err := bm.executeVolumeBackup(ctx, volumeBackup, backupMembers)
	if err != nil {
		return err
	}
	if newMemberCreated {
		return nil
	}
	if err := bm.waitVolumeSnapshotsComplete(ctx, volumeBackup, backupMembers); err != nil {
		return err
	}

	memberUpdated, err := bm.teardownVolumeBackup(ctx, volumeBackup, backupMembers)
	if err != nil {
		return err
	}
	if memberUpdated {
		return nil
	}
	if err := bm.waitVolumeBackupComplete(ctx, volumeBackup, backupMembers); err != nil {
		return err
	}

	if err := bm.setVolumeBackupComplete(&volumeBackup.Status, backupMembers); err != nil {
		return err
	}
	return nil
}

func (bm *backupManager) cleanVolumeBackup(volumeBackup *v1alpha1.VolumeBackup) error {
	ctx := context.Background()
	backupMembers, err := bm.listAllBackupMembers(ctx, volumeBackup)
	if err != nil {
		return err
	}

	if len(backupMembers) == 0 {
		bm.setVolumeBackupCleaned(&volumeBackup.Status)
		return nil
	}

	for _, backupMember := range backupMembers {
		if backupMember.backup.DeletionTimestamp != nil {
			continue
		}
		// delete data plane backup,
		// backup member existing means it's kube client must exist, we don't need to check it
		kubeClient := bm.deps.FedClientset[backupMember.k8sClusterName]
		if err := kubeClient.PingcapV1alpha1().Backups(backupMember.backup.Namespace).
			Delete(ctx, backupMember.backup.Name, metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("delete backup member %s of cluster %s error: %s", backupMember.backup.Name, backupMember.k8sClusterName, err.Error())
		}
	}
	return nil
}

func (bm *backupManager) setVolumeBackupRunning(volumeBackupStatus *v1alpha1.VolumeBackupStatus) {
	volumeBackupStatus.TimeStarted = metav1.Now()
	v1alpha1.UpdateVolumeBackupCondition(volumeBackupStatus, &v1alpha1.VolumeBackupCondition{
		Type:   v1alpha1.VolumeBackupRunning,
		Status: corev1.ConditionTrue,
	})
}

func (bm *backupManager) listAllBackupMembers(ctx context.Context, volumeBackup *v1alpha1.VolumeBackup) ([]*volumeBackupMember, error) {
	backupMembers := make([]*volumeBackupMember, 0, len(volumeBackup.Spec.Clusters))
	for _, memberCluster := range volumeBackup.Spec.Clusters {
		k8sClusterName := memberCluster.K8sClusterName
		kubeClient, ok := bm.deps.FedClientset[memberCluster.K8sClusterName]
		if !ok {
			errMsg := fmt.Sprintf("not find kube client of cluster %s", memberCluster.K8sClusterName)
			if volumeBackup.DeletionTimestamp == nil {
				bm.setVolumeBackupFailed(&volumeBackup.Status, nil, reasonKubeConfigNotFound, errMsg)
			} else {
				bm.setVolumeBackupCleanFailed(&volumeBackup.Status, reasonKubeConfigNotFound, errMsg)
			}
			return nil, errors.New(errMsg)
		}
		backupMemberName := bm.generateBackupMemberName(volumeBackup.Name, memberCluster.K8sClusterName)
		backupMember, err := kubeClient.PingcapV1alpha1().Backups(memberCluster.TCNamespace).Get(ctx, backupMemberName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return nil, fmt.Errorf("get backup %s from cluster %s error: %s", backupMemberName, k8sClusterName, err.Error())
		}
		backupMembers = append(backupMembers, &volumeBackupMember{
			backup:         backupMember,
			k8sClusterName: k8sClusterName,
		})
	}
	return backupMembers, nil
}

func (bm *backupManager) initializeVolumeBackup(ctx context.Context, volumeBackup *v1alpha1.VolumeBackup) error {
	initializeMember := volumeBackup.Spec.Clusters[0]
	kubeClient, ok := bm.deps.FedClientset[initializeMember.K8sClusterName]
	if !ok {
		errMsg := fmt.Sprintf("not find kube client of cluster %s", initializeMember.K8sClusterName)
		bm.setVolumeBackupFailed(&volumeBackup.Status, nil, reasonKubeConfigNotFound, errMsg)
		return errors.New(errMsg)
	}

	backupMember := bm.buildBackupMember(volumeBackup.Name, &initializeMember, &volumeBackup.Spec.Template, true)
	backupMember, err := kubeClient.PingcapV1alpha1().Backups(backupMember.Namespace).Create(ctx, backupMember, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create initialize backup member %s to cluster %s error: %s", backupMember.Name, initializeMember.K8sClusterName, err.Error())
	}
	return nil
}

func (bm *backupManager) waitBackupMemberInitialized(ctx context.Context, volumeBackup *v1alpha1.VolumeBackup, backupMembers []*volumeBackupMember) error {
	for _, backupMember := range backupMembers {
		if pingcapv1alpha1.IsVolumeBackupInitialized(backupMember.backup) {
			return nil
		}
		if pingcapv1alpha1.IsVolumeBackupInitializeFailed(backupMember.backup) {
			errMsg := fmt.Sprintf("backup member %s of cluster %s initialize failed", backupMember.backup.Name, backupMember.k8sClusterName)
			bm.setVolumeBackupFailed(&volumeBackup.Status, backupMembers, reasonVolumeBackupMemberFailed, errMsg)
			return errors.New(errMsg)
		}
	}
	return errors.New("not backup member initialized, waiting")
}

func (bm *backupManager) executeVolumeBackup(ctx context.Context, volumeBackup *v1alpha1.VolumeBackup, backupMembers []*volumeBackupMember) (newMemberCreated bool, err error) {
	backupMemberMap := make(map[string]*volumeBackupMember, len(backupMembers))
	for _, backupMember := range backupMembers {
		if backupMember.backup.Spec.FederalVolumeBackupPhase == pingcapv1alpha1.FederalVolumeBackupInitialize {
			backupMember.backup.Spec.FederalVolumeBackupPhase = pingcapv1alpha1.FederalVolumeBackupExecute
			kubeClient, ok := bm.deps.FedClientset[backupMember.k8sClusterName]
			if !ok {
				errMsg := fmt.Sprintf("not find kube client of cluster %s", backupMember.k8sClusterName)
				bm.setVolumeBackupFailed(&volumeBackup.Status, backupMembers, reasonKubeConfigNotFound, errMsg)
				return false, errors.New(errMsg)
			}
			if updatedBackup, err := kubeClient.PingcapV1alpha1().Backups(backupMember.backup.Namespace).Update(ctx, backupMember.backup, metav1.UpdateOptions{}); err != nil {
				return false, fmt.Errorf("update backup member %s of cluster %s to execute phase error: %s", backupMember.backup.Name, backupMember.k8sClusterName, err.Error())
			} else {
				backupMember.backup = updatedBackup
			}
		}
		backupMemberMap[backupMember.backup.Name] = backupMember
	}

	for _, memberCluster := range volumeBackup.Spec.Clusters {
		backupMemberName := bm.generateBackupMemberName(volumeBackup.Name, memberCluster.K8sClusterName)
		if _, ok := backupMemberMap[backupMemberName]; ok {
			continue
		}

		kubeClient, ok := bm.deps.FedClientset[memberCluster.K8sClusterName]
		if !ok {
			errMsg := fmt.Sprintf("not find kube client of cluster %s", memberCluster.K8sClusterName)
			bm.setVolumeBackupFailed(&volumeBackup.Status, backupMembers, reasonKubeConfigNotFound, errMsg)
			return false, errors.New(errMsg)
		}
		backupMember := bm.buildBackupMember(volumeBackup.Name, &memberCluster, &volumeBackup.Spec.Template, false)
		if _, err := kubeClient.PingcapV1alpha1().Backups(memberCluster.TCNamespace).Create(ctx, backupMember, metav1.CreateOptions{}); err != nil {
			return false, fmt.Errorf("create backup member %s to cluster %s error: %s", backupMember.Name, memberCluster.K8sClusterName, err.Error())
		}
		newMemberCreated = true
	}
	return
}

func (bm *backupManager) waitVolumeSnapshotsComplete(ctx context.Context, volumeBackup *v1alpha1.VolumeBackup, backupMembers []*volumeBackupMember) error {
	for _, backupMember := range backupMembers {
		if pingcapv1alpha1.IsVolumeBackupInitializeFailed(backupMember.backup) || pingcapv1alpha1.IsVolumeBackupFailed(backupMember.backup) {
			errMsg := fmt.Sprintf("backup member %s of cluster %s failed", backupMember.backup.Name, backupMember.k8sClusterName)
			bm.setVolumeBackupFailed(&volumeBackup.Status, backupMembers, reasonVolumeBackupMemberFailed, errMsg)
			return errors.New(errMsg)
		}
		if !pingcapv1alpha1.IsVolumeBackupComplete(backupMember.backup) {
			return fmt.Errorf("backup member %s of cluster %s is not volume snapshots complete", backupMember.backup.Name, backupMember.k8sClusterName)
		}
	}
	return nil
}

func (bm *backupManager) teardownVolumeBackup(ctx context.Context, volumeBackup *v1alpha1.VolumeBackup, backupMembers []*volumeBackupMember) (memberUpdated bool, err error) {
	for _, backupMember := range backupMembers {
		if backupMember.backup.Spec.FederalVolumeBackupPhase == pingcapv1alpha1.FederalVolumeBackupTeardown {
			continue
		}

		backupMember.backup.Spec.FederalVolumeBackupPhase = pingcapv1alpha1.FederalVolumeBackupTeardown
		kubeClient, ok := bm.deps.FedClientset[backupMember.k8sClusterName]
		if !ok {
			errMsg := fmt.Sprintf("not find kube client of cluster %s", backupMember.k8sClusterName)
			bm.setVolumeBackupFailed(&volumeBackup.Status, backupMembers, reasonKubeConfigNotFound, errMsg)
			return false, errors.New(errMsg)
		}
		if _, err := kubeClient.PingcapV1alpha1().Backups(backupMember.backup.Namespace).Update(ctx, backupMember.backup, metav1.UpdateOptions{}); err != nil {
			return false, fmt.Errorf("update backup member %s of cluster %s to teardown phase error: %s", backupMember.backup.Name, backupMember.k8sClusterName, err.Error())
		}
		memberUpdated = true
	}
	return
}

func (bm *backupManager) waitVolumeBackupComplete(ctx context.Context, volumeBackup *v1alpha1.VolumeBackup, backupMembers []*volumeBackupMember) error {
	for _, backupMember := range backupMembers {
		if pingcapv1alpha1.IsVolumeBackupInitializeFailed(backupMember.backup) || pingcapv1alpha1.IsBackupFailed(backupMember.backup) {
			errMsg := fmt.Sprintf("backup member %s of cluster %s failed", backupMember.backup.Name, backupMember.k8sClusterName)
			bm.setVolumeBackupFailed(&volumeBackup.Status, backupMembers, reasonVolumeBackupMemberFailed, errMsg)
			return errors.New(errMsg)
		}
		if !pingcapv1alpha1.IsBackupComplete(backupMember.backup) {
			return fmt.Errorf("backup member %s of cluster %s is not complete", backupMember.backup.Name, backupMember.k8sClusterName)
		}
	}
	return nil
}

func (bm *backupManager) setVolumeBackupComplete(volumeBackupStatus *v1alpha1.VolumeBackupStatus, backupMembers []*volumeBackupMember) error {
	volumeBackupStatus.TimeCompleted = metav1.Now()
	volumeBackupStatus.TimeTaken = volumeBackupStatus.TimeCompleted.Sub(volumeBackupStatus.TimeStarted.Time).String()
	bm.setVolumeBackupSize(volumeBackupStatus, backupMembers)
	if err := bm.setVolumeBackupCommitTs(volumeBackupStatus, backupMembers); err != nil {
		return err
	}
	v1alpha1.UpdateVolumeBackupCondition(volumeBackupStatus, &v1alpha1.VolumeBackupCondition{
		Type:   v1alpha1.VolumeBackupComplete,
		Status: corev1.ConditionTrue,
	})
	return nil
}

func (bm *backupManager) setVolumeBackupFailed(volumeBackupStatus *v1alpha1.VolumeBackupStatus, backupMembers []*volumeBackupMember, reason, message string) {
	volumeBackupStatus.TimeCompleted = metav1.Now()
	volumeBackupStatus.TimeTaken = volumeBackupStatus.TimeCompleted.Sub(volumeBackupStatus.TimeStarted.Time).String()
	bm.setVolumeBackupSize(volumeBackupStatus, backupMembers)
	v1alpha1.UpdateVolumeBackupCondition(volumeBackupStatus, &v1alpha1.VolumeBackupCondition{
		Type:    v1alpha1.VolumeBackupFailed,
		Status:  corev1.ConditionTrue,
		Reason:  reason,
		Message: message,
	})
}

func (bm *backupManager) setVolumeBackupCleanFailed(volumeBackupStatus *v1alpha1.VolumeBackupStatus, reason, message string) {
	v1alpha1.UpdateVolumeBackupCondition(volumeBackupStatus, &v1alpha1.VolumeBackupCondition{
		Type:    v1alpha1.VolumeBackupCleanFailed,
		Status:  corev1.ConditionTrue,
		Reason:  reason,
		Message: message,
	})
}

func (bm *backupManager) setVolumeBackupCleaned(volumeBackupStatus *v1alpha1.VolumeBackupStatus) {
	v1alpha1.UpdateVolumeBackupCondition(volumeBackupStatus, &v1alpha1.VolumeBackupCondition{
		Type:   v1alpha1.VolumeBackupCleaned,
		Status: corev1.ConditionTrue,
	})
}

func (bm *backupManager) setVolumeBackupSize(volumeBackupStatus *v1alpha1.VolumeBackupStatus, backupMembers []*volumeBackupMember) {
	var totalBackupSize int64
	for _, backupMember := range backupMembers {
		totalBackupSize += backupMember.backup.Status.BackupSize
	}
	backupSizeReadable := humanize.Bytes(uint64(totalBackupSize))
	volumeBackupStatus.BackupSize = totalBackupSize
	volumeBackupStatus.BackupSizeReadable = backupSizeReadable
}

func (bm *backupManager) setVolumeBackupCommitTs(volumeBackupStatus *v1alpha1.VolumeBackupStatus, backupMembers []*volumeBackupMember) error {
	minCommitTs := int64(math.MaxInt64)
	for _, backupMember := range backupMembers {
		commitTs, err := strconv.ParseInt(backupMember.backup.Status.CommitTs, 10, 64)
		if err != nil {
			return fmt.Errorf("parse commit ts %s of backup member %s error: %s", backupMember.backup.Status.CommitTs, backupMember.backup.Name, err.Error())
		}
		if commitTs < minCommitTs {
			minCommitTs = commitTs
		}
	}
	volumeBackupStatus.CommitTs = strconv.FormatInt(minCommitTs, 10)
	return nil
}

func (bm *backupManager) updateVolumeBackupMembersToStatus(volumeBackup *v1alpha1.VolumeBackup, backupMembers []*volumeBackupMember) {
	for _, backupMember := range backupMembers {
		v1alpha1.UpdateVolumeBackupMemberStatus(&volumeBackup.Status, backupMember.k8sClusterName, backupMember.backup)
	}
}

func (bm *backupManager) buildBackupMember(volumeBackupName string, clusterMember *v1alpha1.VolumeBackupMemberCluster, backupTemplate *v1alpha1.VolumeBackupMemberSpec, initialize bool) *pingcapv1alpha1.Backup {
	backupMember := &pingcapv1alpha1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bm.generateBackupMemberName(volumeBackupName, clusterMember.K8sClusterName),
			Namespace: clusterMember.TCNamespace,
		},
		Spec: pingcapv1alpha1.BackupSpec{
			Mode:                     pingcapv1alpha1.BackupModeVolumeSnapshot,
			FederalVolumeBackupPhase: pingcapv1alpha1.FederalVolumeBackupExecute,
			ResourceRequirements:     backupTemplate.ResourceRequirements,
			Env:                      backupTemplate.Env,
			BR:                       backupTemplate.BR.ToBRMemberConfig(clusterMember.TCName, clusterMember.TCNamespace),
			StorageProvider:          backupTemplate.StorageProvider,
			Tolerations:              backupTemplate.Tolerations,
			ToolImage:                backupTemplate.ToolImage,
			ImagePullSecrets:         backupTemplate.ImagePullSecrets,
			ServiceAccount:           backupTemplate.ServiceAccount,
			CleanPolicy:              backupTemplate.CleanPolicy,
			PriorityClassName:        backupTemplate.PriorityClassName,
		},
	}
	backupMember.Spec.S3.Prefix = fmt.Sprintf("%s-%s", backupMember.Spec.S3.Prefix, clusterMember.K8sClusterName)
	if initialize {
		backupMember.Spec.FederalVolumeBackupPhase = pingcapv1alpha1.FederalVolumeBackupInitialize
	}
	return backupMember
}

func (bm *backupManager) generateBackupMemberName(volumeBackupName, k8sClusterName string) string {
	return fmt.Sprintf("fed-%s-%s", volumeBackupName, k8sClusterName)
}

type volumeBackupMember struct {
	backup         *pingcapv1alpha1.Backup
	k8sClusterName string
}

var _ fedvolumebackup.BackupManager = &backupManager{}

type FakeBackupManager struct {
	err error
}

func NewFakeBackupManager() *FakeBackupManager {
	return &FakeBackupManager{}
}

func (m *FakeBackupManager) SetSyncError(err error) {
	m.err = err
}

func (m *FakeBackupManager) Sync(_ *v1alpha1.VolumeBackup) error {
	return m.err
}

// UpdateStatus updates the status for a Backup, include condition and status info.
func (m *FakeBackupManager) UpdateStatus(_ *v1alpha1.VolumeBackup, newStatus *v1alpha1.VolumeBackupStatus) error {
	return nil
}

var _ fedvolumebackup.BackupManager = &FakeBackupManager{}
