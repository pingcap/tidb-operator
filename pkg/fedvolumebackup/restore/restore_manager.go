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

package restore

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

	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	pingcapv1alpha1 "github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/fedvolumebackup"
)

const (
	reasonKubeConfigNotFound                   = "KubeConfigNotFound"
	reasonVolumeRestoreMemberFailed            = "VolumeRestoreMemberFailed"
	reasonVolumeRestoreMemberInvalid           = "VolumeRestoreMemberInvalid"
	reasonVolumeRestoreMemberResolvedTsInvalid = "VolumeRestoreMemberResolvedTsInvalid"
)

type restoreManager struct {
	deps *controller.BrFedDependencies
}

// NewRestoreManager return restoreManager
func NewRestoreManager(deps *controller.BrFedDependencies) fedvolumebackup.RestoreManager {
	return &restoreManager{
		deps: deps,
	}
}

func (rm *restoreManager) Sync(volumeRestore *v1alpha1.VolumeRestore) error {
	// because a finalizer is installed on the VolumeRestore on creation, when the VolumeRestore is deleted,
	// VolumeRestore.DeletionTimestamp will be set, controller will be informed with an onUpdate event,
	// this is the moment that we can do clean up work.
	if volumeRestore.DeletionTimestamp != nil {
		return rm.cleanVolumeRestore(volumeRestore)
	}

	return rm.syncRestore(volumeRestore)
}

// UpdateStatus updates the status for a VolumeRestore, include condition and status info.
func (rm *restoreManager) UpdateStatus(volumeRestore *v1alpha1.VolumeRestore, newStatus *v1alpha1.VolumeRestoreStatus) error {
	name := volumeRestore.Name
	ns := volumeRestore.Namespace
	ctx := context.Background()
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		latestRestore, err := rm.deps.Clientset.FederationV1alpha1().VolumeRestores(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			klog.Warningf("restore %s/%s get restore error: %s", ns, name, err.Error())
			return err
		}
		if apiequality.Semantic.DeepEqual(&latestRestore.Status, newStatus) {
			return nil
		}
		latestRestore.Status = *newStatus
		_, err = rm.deps.Clientset.FederationV1alpha1().VolumeRestores(ns).Update(ctx, latestRestore, metav1.UpdateOptions{})
		return err
	})
}

func (rm *restoreManager) syncRestore(volumeRestore *v1alpha1.VolumeRestore) error {
	ns := volumeRestore.GetNamespace()
	name := volumeRestore.GetName()

	klog.Infof("sync VolumeRestore %s/%s", ns, name)

	if rm.skipVolumeRestore(volumeRestore) {
		return nil
	}
	if err := rm.checkFederalMembers(volumeRestore); err != nil {
		return err
	}

	if !v1alpha1.IsVolumeRestoreRunning(volumeRestore) {
		rm.setVolumeRestoreRunning(&volumeRestore.Status)
	}

	ctx := context.Background()
	restoreMembers, err := rm.listRestoreMembers(ctx, volumeRestore)
	if err != nil {
		return err
	}

	if err := rm.executeRestoreVolumePhase(ctx, volumeRestore, restoreMembers); err != nil {
		return err
	}
	if err := rm.waitRestoreVolumeComplete(volumeRestore, restoreMembers); err != nil {
		return err
	}

	if err := rm.executeRestoreDataPhase(ctx, volumeRestore, restoreMembers); err != nil {
		return err
	}
	if err := rm.waitRestoreDataComplete(volumeRestore, restoreMembers); err != nil {
		return err
	}

	if err := rm.executeRestoreFinishPhase(ctx, volumeRestore, restoreMembers); err != nil {
		return err
	}
	if err := rm.waitRestoreComplete(volumeRestore, restoreMembers); err != nil {
		return err
	}

	rm.setVolumeRestoreComplete(&volumeRestore.Status)
	return nil
}

func (rm *restoreManager) listRestoreMembers(ctx context.Context, volumeRestore *v1alpha1.VolumeRestore) ([]*volumeRestoreMember, error) {
	restoreMembers := make([]*volumeRestoreMember, 0, len(volumeRestore.Spec.Clusters))
	for _, memberCluster := range volumeRestore.Spec.Clusters {
		k8sClusterName := memberCluster.K8sClusterName
		kubeClient := rm.deps.FedClientset[k8sClusterName]
		restoreName := rm.generateRestoreMemberName(volumeRestore.Name, k8sClusterName)
		restoreMember, err := kubeClient.PingcapV1alpha1().Restores(memberCluster.TCNamespace).Get(ctx, restoreName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return nil, fmt.Errorf("get restore %s from cluster %s error: %s", restoreName, k8sClusterName, err.Error())
		}

		restoreMembers = append(restoreMembers, &volumeRestoreMember{
			restore:        restoreMember,
			k8sClusterName: k8sClusterName,
		})
	}
	return restoreMembers, nil
}

func (rm *restoreManager) executeRestoreVolumePhase(ctx context.Context, volumeRestore *v1alpha1.VolumeRestore, restoreMembers []*volumeRestoreMember) error {
	restoreMemberMap := make(map[string]*volumeRestoreMember, len(restoreMembers))
	for _, restoreMember := range restoreMembers {
		restoreMemberMap[restoreMember.restore.Name] = restoreMember
	}

	for i := range volumeRestore.Spec.Clusters {
		memberCluster := volumeRestore.Spec.Clusters[i]
		k8sClusterName := memberCluster.K8sClusterName
		restoreName := rm.generateRestoreMemberName(volumeRestore.Name, k8sClusterName)
		if _, ok := restoreMemberMap[restoreName]; ok {
			continue
		}

		kubeClient := rm.deps.FedClientset[k8sClusterName]
		restoreMember := rm.buildRestoreMember(volumeRestore.Name, &memberCluster, &volumeRestore.Spec.Template)
		if _, err := kubeClient.PingcapV1alpha1().Restores(memberCluster.TCNamespace).Create(ctx, restoreMember, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("create restore member %s to cluster %s error: %s", restoreMember.Name, k8sClusterName, err.Error())
		}
	}
	return nil
}

func (rm *restoreManager) waitRestoreVolumeComplete(volumeRestore *v1alpha1.VolumeRestore, restoreMembers []*volumeRestoreMember) error {
	for _, restoreMember := range restoreMembers {
		restoreMemberName := restoreMember.restore.Name
		k8sClusterName := restoreMember.k8sClusterName
		if pingcapv1alpha1.IsRestoreInvalid(restoreMember.restore) {
			errMsg := fmt.Sprintf("restore member %s of cluster %s is invalid", restoreMemberName, k8sClusterName)
			rm.setVolumeRestoreFailed(&volumeRestore.Status, reasonVolumeRestoreMemberInvalid, errMsg)
			return controller.IgnoreErrorf(errMsg)
		}
		if pingcapv1alpha1.IsRestoreFailed(restoreMember.restore) {
			errMsg := fmt.Sprintf("restore member %s of cluster %s is failed", restoreMemberName, k8sClusterName)
			rm.setVolumeRestoreFailed(&volumeRestore.Status, reasonVolumeRestoreMemberFailed, errMsg)
			return controller.IgnoreErrorf(errMsg)
		}

		if !pingcapv1alpha1.IsRestoreTiKVComplete(restoreMember.restore) {
			return controller.IgnoreErrorf("restore member %s of cluster %s is not tikv complete", restoreMemberName, k8sClusterName)
		}
	}
	return nil
}

func (rm *restoreManager) executeRestoreDataPhase(ctx context.Context, volumeRestore *v1alpha1.VolumeRestore, restoreMembers []*volumeRestoreMember) error {
	minResolvedTs := int64(math.MaxInt64)
	var minResolvedTsMember *volumeRestoreMember
	for _, restoreMember := range restoreMembers {
		// already in 2nd or 3rd phase
		if restoreMember.restore.Spec.FederalVolumeRestorePhase == pingcapv1alpha1.FederalVolumeRestoreData ||
			restoreMember.restore.Spec.FederalVolumeRestorePhase == pingcapv1alpha1.FederalVolumeRestoreFinish {
			return nil
		}

		restoreMemberName := restoreMember.restore.Name
		k8sClusterName := restoreMember.k8sClusterName
		commitTsStr := restoreMember.restore.Status.CommitTs
		if commitTsStr == "" {
			errMsg := fmt.Sprintf("commit ts in restore member %s of cluster %s is empty", restoreMemberName, k8sClusterName)
			rm.setVolumeRestoreFailed(&volumeRestore.Status, reasonVolumeRestoreMemberResolvedTsInvalid, errMsg)
			return controller.IgnoreErrorf(errMsg)
		}
		resolvedTs, err := strconv.ParseInt(commitTsStr, 10, 64)
		if err != nil {
			errMsg := fmt.Sprintf("parse resolved ts %s to int64 in restore member %s of cluster %s error: %s", commitTsStr, restoreMemberName, k8sClusterName, err.Error())
			rm.setVolumeRestoreFailed(&volumeRestore.Status, reasonVolumeRestoreMemberResolvedTsInvalid, errMsg)
			return controller.IgnoreErrorf(errMsg)
		}

		if minResolvedTs > resolvedTs {
			minResolvedTs = resolvedTs
			minResolvedTsMember = restoreMember
		}
	}

	restoreCR := minResolvedTsMember.restore.DeepCopy()
	restoreCR.Spec.FederalVolumeRestorePhase = pingcapv1alpha1.FederalVolumeRestoreData
	kubeClient := rm.deps.FedClientset[minResolvedTsMember.k8sClusterName]
	if _, err := kubeClient.PingcapV1alpha1().Restores(restoreCR.Namespace).Update(ctx, restoreCR, metav1.UpdateOptions{}); err != nil {
		return controller.RequeueErrorf("update FederalVolumeRestorePhase to restore-data in restore member %s of cluster %s error: %s", minResolvedTsMember.restore.Name, minResolvedTsMember.k8sClusterName, err.Error())
	}
	return nil
}

func (rm *restoreManager) waitRestoreDataComplete(volumeRestore *v1alpha1.VolumeRestore, restoreMembers []*volumeRestoreMember) error {
	for _, restoreMember := range restoreMembers {
		restoreMemberName := restoreMember.restore.Name
		k8sClusterName := restoreMember.k8sClusterName
		if pingcapv1alpha1.IsRestoreDataComplete(restoreMember.restore) {
			return nil
		}

		if pingcapv1alpha1.IsRestoreFailed(restoreMember.restore) {
			errMsg := fmt.Sprintf("restore member %s of cluster %s is failed", restoreMemberName, k8sClusterName)
			rm.setVolumeRestoreFailed(&volumeRestore.Status, reasonVolumeRestoreMemberFailed, errMsg)
			return controller.IgnoreErrorf(errMsg)
		}
	}

	return controller.IgnoreErrorf("restore member is not restore data complete, waiting")
}

func (rm *restoreManager) executeRestoreFinishPhase(ctx context.Context, volumeRestore *v1alpha1.VolumeRestore, restoreMembers []*volumeRestoreMember) error {
	for _, restoreMember := range restoreMembers {
		if restoreMember.restore.Spec.FederalVolumeRestorePhase == pingcapv1alpha1.FederalVolumeRestoreFinish {
			continue
		}

		restoreMemberName := restoreMember.restore.Name
		k8sClusterName := restoreMember.k8sClusterName
		restoreCR := restoreMember.restore.DeepCopy()
		restoreCR.Spec.FederalVolumeRestorePhase = pingcapv1alpha1.FederalVolumeRestoreFinish
		kubeClient := rm.deps.FedClientset[k8sClusterName]
		if _, err := kubeClient.PingcapV1alpha1().Restores(restoreCR.Namespace).Update(ctx, restoreCR, metav1.UpdateOptions{}); err != nil {
			return controller.RequeueErrorf("update FederalVolumeRestorePhase to restore-finish in restore member %s of cluster %s error: %s", restoreMemberName, k8sClusterName, err.Error())
		}
	}
	return nil
}

func (rm *restoreManager) waitRestoreComplete(volumeRestore *v1alpha1.VolumeRestore, restoreMembers []*volumeRestoreMember) error {
	for _, restoreMember := range restoreMembers {
		restoreMemberName := restoreMember.restore.Name
		k8sClusterName := restoreMember.k8sClusterName
		if pingcapv1alpha1.IsRestoreFailed(restoreMember.restore) {
			errMsg := fmt.Sprintf("restore member %s of cluster %s is failed", restoreMemberName, k8sClusterName)
			rm.setVolumeRestoreFailed(&volumeRestore.Status, reasonVolumeRestoreMemberFailed, errMsg)
			return controller.IgnoreErrorf(errMsg)
		}
		if !pingcapv1alpha1.IsRestoreComplete(restoreMember.restore) {
			return controller.IgnoreErrorf("restore member %s of cluster %s is not complete, waiting", restoreMemberName, k8sClusterName)
		}
	}
	return nil
}

func (rm *restoreManager) cleanVolumeRestore(volumeRestore *v1alpha1.VolumeRestore) error {
	name := volumeRestore.Name
	ns := volumeRestore.Namespace
	ctx := context.Background()
	for _, memberCluster := range volumeRestore.Spec.Clusters {
		k8sClusterName := memberCluster.K8sClusterName
		kubeClient, ok := rm.deps.FedClientset[k8sClusterName]
		if !ok {
			klog.Errorf("not find kube client of cluster %s when clean volume restore %s/%s", k8sClusterName, ns, name)
			continue
		}

		restoreMemberName := rm.generateRestoreMemberName(name, k8sClusterName)
		err := kubeClient.PingcapV1alpha1().Restores(memberCluster.TCNamespace).Delete(ctx, restoreMemberName, metav1.DeleteOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return controller.RequeueErrorf("delete restore member %s of cluster %s error: %s", restoreMemberName, k8sClusterName, err.Error())
		}
	}

	rm.setVolumeRestoreCleaned(&volumeRestore.Status)
	return nil
}

func (rm *restoreManager) setVolumeRestoreRunning(volumeRestoreStatus *v1alpha1.VolumeRestoreStatus) {
	volumeRestoreStatus.TimeStarted = metav1.Now()
	v1alpha1.UpdateVolumeRestoreCondition(volumeRestoreStatus, &v1alpha1.VolumeRestoreCondition{
		Type:   v1alpha1.VolumeRestoreRunning,
		Status: corev1.ConditionTrue,
	})
}

func (rm *restoreManager) setVolumeRestoreComplete(volumeRestoreStatus *v1alpha1.VolumeRestoreStatus) {
	volumeRestoreStatus.TimeCompleted = metav1.Now()
	volumeRestoreStatus.TimeTaken = volumeRestoreStatus.TimeCompleted.Sub(volumeRestoreStatus.TimeStarted.Time).String()
	v1alpha1.UpdateVolumeRestoreCondition(volumeRestoreStatus, &v1alpha1.VolumeRestoreCondition{
		Type:   v1alpha1.VolumeRestoreComplete,
		Status: corev1.ConditionTrue,
	})
}

func (rm *restoreManager) setVolumeRestoreFailed(volumeRestoreStatus *v1alpha1.VolumeRestoreStatus, reason, message string) {
	volumeRestoreStatus.TimeCompleted = metav1.Now()
	volumeRestoreStatus.TimeTaken = volumeRestoreStatus.TimeCompleted.Sub(volumeRestoreStatus.TimeStarted.Time).String()
	v1alpha1.UpdateVolumeRestoreCondition(volumeRestoreStatus, &v1alpha1.VolumeRestoreCondition{
		Type:    v1alpha1.VolumeRestoreFailed,
		Status:  corev1.ConditionTrue,
		Reason:  reason,
		Message: message,
	})
}

func (rm *restoreManager) setVolumeRestoreCleaned(volumeRestoreStatus *v1alpha1.VolumeRestoreStatus) {
	v1alpha1.UpdateVolumeRestoreCondition(volumeRestoreStatus, &v1alpha1.VolumeRestoreCondition{
		Type:   v1alpha1.VolumeRestoreCleaned,
		Status: corev1.ConditionTrue,
	})
}

func (rm *restoreManager) checkFederalMembers(volumeRestore *v1alpha1.VolumeRestore) error {
	for _, memberCluster := range volumeRestore.Spec.Clusters {
		if _, ok := rm.deps.FedClientset[memberCluster.K8sClusterName]; !ok {
			errMsg := fmt.Sprintf("not find kube client of cluster %s", memberCluster.K8sClusterName)
			rm.setVolumeRestoreFailed(&volumeRestore.Status, reasonKubeConfigNotFound, errMsg)
			return controller.IgnoreErrorf(errMsg)
		}
	}
	return nil
}

func (rm *restoreManager) skipVolumeRestore(volumeRestore *v1alpha1.VolumeRestore) bool {
	return v1alpha1.IsVolumeRestoreComplete(volumeRestore) || v1alpha1.IsVolumeRestoreFailed(volumeRestore)
}

func (rm *restoreManager) buildRestoreMember(volumeRestoreName string, memberCluster *v1alpha1.VolumeRestoreMemberCluster, template *v1alpha1.VolumeRestoreMemberSpec) *pingcapv1alpha1.Restore {
	restoreMember := &pingcapv1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rm.generateRestoreMemberName(volumeRestoreName, memberCluster.K8sClusterName),
			Namespace: memberCluster.TCNamespace,
		},
		Spec: pingcapv1alpha1.RestoreSpec{
			Mode:                      pingcapv1alpha1.RestoreModeVolumeSnapshot,
			Type:                      pingcapv1alpha1.BackupTypeFull,
			FederalVolumeRestorePhase: pingcapv1alpha1.FederalVolumeRestoreVolume,
			StorageProvider:           memberCluster.Backup.StorageProvider,
			ResourceRequirements:      template.ResourceRequirements,
			Env:                       template.Env,
			VolumeAZ:                  memberCluster.AZName,
			BR:                        template.BR.ToBRMemberConfig(memberCluster.TCName, memberCluster.TCNamespace),
			Tolerations:               template.Tolerations,
			ToolImage:                 template.ToolImage,
			ImagePullSecrets:          template.ImagePullSecrets,
			ServiceAccount:            template.ServiceAccount,
			PriorityClassName:         template.PriorityClassName,
		},
	}
	return restoreMember
}

func (rm *restoreManager) generateRestoreMemberName(volumeRestoreName, k8sClusterName string) string {
	return fmt.Sprintf("fed-%s-%s", volumeRestoreName, k8sClusterName)
}

type volumeRestoreMember struct {
	restore        *pingcapv1alpha1.Restore
	k8sClusterName string
}

var _ fedvolumebackup.RestoreManager = &restoreManager{}

type FakeRestoreManager struct {
	err error
}

func NewFakeRestoreManager() *FakeRestoreManager {
	return &FakeRestoreManager{}
}

func (m *FakeRestoreManager) SetSyncError(err error) {
	m.err = err
}

func (m *FakeRestoreManager) Sync(_ *v1alpha1.VolumeRestore) error {
	return m.err
}

// UpdateStatus updates the status for a VolumeRestore, include condition and status info.
func (m *FakeRestoreManager) UpdateStatus(volumeRestore *v1alpha1.VolumeRestore, newStatus *v1alpha1.VolumeRestoreStatus) error {
	return nil
}

var _ fedvolumebackup.RestoreManager = &FakeRestoreManager{}
