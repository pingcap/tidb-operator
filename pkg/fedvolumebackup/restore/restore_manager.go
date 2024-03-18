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
	"time"

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

	if err := rm.syncRestore(volumeRestore); err != nil {
		brFailedErr, ok := err.(*fedvolumebackup.BRDataPlaneFailedError)
		if !ok {
			return err
		}

		klog.Errorf("VolumeRestore %s/%s restore member failed, set status failed. err: %s", volumeRestore.Name, volumeRestore.Namespace, err.Error())
		rm.setVolumeRestoreFailed(&volumeRestore.Status, brFailedErr.Reason, brFailedErr.Message)
	}
	return nil
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

	if !v1alpha1.IsVolumeRestoreRunning(volumeRestore) {
		klog.Infof("VolumeRestore %s/%s set status running", ns, name)
		rm.setVolumeRestoreRunning(&volumeRestore.Status)
	}

	ctx := context.Background()
	restoreMembers, err := rm.listRestoreMembers(ctx, volumeRestore)
	if err != nil {
		return err
	}
	rm.updateVolumeRestoreMembersToStatus(&volumeRestore.Status, restoreMembers)

	memberCreated, err := rm.executeRestoreVolumePhase(ctx, volumeRestore, restoreMembers)
	if err != nil {
		return err
	}
	// restore members in data plane are created just now, wait for next loop.
	if memberCreated {
		return nil
	}

	if err := rm.waitRestoreVolumeComplete(volumeRestore, restoreMembers); err != nil {
		return err
	}

	rm.syncWarmUpStatus(volumeRestore, restoreMembers)

	if err := rm.waitRestoreTiKVComplete(volumeRestore, restoreMembers); err != nil {
		return err
	}

	// When playing check wal, we should stop here.
	if volumeRestore.Spec.Template.WarmupStrategy == pingcapv1alpha1.RestoreWarmupStrategyCheckOnly {
		klog.Infof("VolumeRestore %s/%s check WAL complete", ns, name)
		v1alpha1.FinishVolumeRestoreStep(&volumeRestore.Status, v1alpha1.VolumeRestoreStepRestartTiKV)
		return nil
	}

	memberUpdated, err := rm.executeRestoreDataPhase(ctx, volumeRestore, restoreMembers)
	if err != nil {
		return err
	}
	// just execute restore data phase, wait for next loop.
	if memberUpdated {
		return nil
	}
	if err := rm.waitRestoreDataComplete(volumeRestore, restoreMembers); err != nil {
		return err
	}

	memberUpdated, err = rm.executeRestoreFinishPhase(ctx, volumeRestore, restoreMembers)
	if err != nil {
		return err
	}
	// just execute restore finish phase, wait for next loop.
	if memberUpdated {
		return nil
	}
	if err := rm.waitRestoreComplete(volumeRestore, restoreMembers); err != nil {
		return err
	}
	v1alpha1.FinishVolumeRestoreStep(&volumeRestore.Status, v1alpha1.VolumeRestoreStepRestartTiKV)

	if isWarmUpAsync(volumeRestore) && !v1alpha1.IsVolumeRestoreWarmUpComplete(volumeRestore) {
		klog.Infof("VolumeRestore %s/%s data planes all complete, but warmup doesn't complete, wait warmup complete", ns, name)
		return nil
	}

	klog.Infof("VolumeRestore %s/%s restore complete", ns, name)
	rm.setVolumeRestoreComplete(&volumeRestore.Status)
	return nil
}

func (rm *restoreManager) listRestoreMembers(ctx context.Context, volumeRestore *v1alpha1.VolumeRestore) ([]*volumeRestoreMember, error) {
	existedMembers := make(map[string]*v1alpha1.VolumeRestoreMemberStatus, len(volumeRestore.Status.Restores))
	for i := range volumeRestore.Status.Restores {
		existedMembers[volumeRestore.Status.Restores[i].K8sClusterName] = &volumeRestore.Status.Restores[i]
	}

	restoreMembers := make([]*volumeRestoreMember, 0, len(volumeRestore.Spec.Clusters))
	for _, memberCluster := range volumeRestore.Spec.Clusters {
		k8sClusterName := memberCluster.K8sClusterName
		kubeClient, ok := rm.deps.FedClientset[k8sClusterName]
		if !ok {
			return nil, controller.RequeueErrorf("not find kube client of cluster %s", k8sClusterName)
		}
		var restoreName string
		if existedRestoreMember, ok := existedMembers[k8sClusterName]; ok {
			restoreName = existedRestoreMember.RestoreName
		} else {
			restoreName = rm.generateRestoreMemberName(volumeRestore.Name)
		}
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

func (rm *restoreManager) updateVolumeRestoreMembersToStatus(volumeRestoreStatus *v1alpha1.VolumeRestoreStatus, restoreMembers []*volumeRestoreMember) {
	for _, restoreMember := range restoreMembers {
		v1alpha1.UpdateVolumeRestoreMemberStatus(volumeRestoreStatus, restoreMember.k8sClusterName, restoreMember.restore)
	}
}

func (rm *restoreManager) executeRestoreVolumePhase(ctx context.Context, volumeRestore *v1alpha1.VolumeRestore, restoreMembers []*volumeRestoreMember) (memberCreated bool, err error) {
	v1alpha1.StartVolumeRestoreStep(&volumeRestore.Status, v1alpha1.VolumeRestoreStepRestoreVolume)
	restoreMemberMap := make(map[string]*volumeRestoreMember, len(restoreMembers))
	for _, restoreMember := range restoreMembers {
		restoreMemberMap[restoreMember.k8sClusterName] = restoreMember
	}

	for i := range volumeRestore.Spec.Clusters {
		memberCluster := volumeRestore.Spec.Clusters[i]
		k8sClusterName := memberCluster.K8sClusterName
		if _, ok := restoreMemberMap[k8sClusterName]; ok {
			continue
		}

		kubeClient := rm.deps.FedClientset[k8sClusterName]
		restoreMember := rm.buildRestoreMember(volumeRestore.Name, &memberCluster, &volumeRestore.Spec.Template, volumeRestore.Annotations, volumeRestore.Labels)
		if _, err := kubeClient.PingcapV1alpha1().Restores(memberCluster.TCNamespace).Create(ctx, restoreMember, metav1.CreateOptions{}); err != nil {
			return false, fmt.Errorf("create restore member %s to cluster %s error: %s", restoreMember.Name, k8sClusterName, err.Error())
		}
		memberCreated = true
		klog.Infof("VolumeRestore %s/%s create restore member %s to cluster %s successfully", volumeRestore.Namespace, volumeRestore.Name, restoreMember.Name, k8sClusterName)
		v1alpha1.UpdateVolumeRestoreMemberStatus(&volumeRestore.Status, k8sClusterName, restoreMember)
	}
	return
}

func (rm *restoreManager) waitRestoreVolumeComplete(volumeRestore *v1alpha1.VolumeRestore, restoreMembers []*volumeRestoreMember) error {
	// check if restore members failed in data plane
	for _, restoreMember := range restoreMembers {
		restoreMemberName := restoreMember.restore.Name
		k8sClusterName := restoreMember.k8sClusterName
		if pingcapv1alpha1.IsRestoreInvalid(restoreMember.restore) {
			errMsg := fmt.Sprintf("restore member %s of cluster %s is invalid", restoreMemberName, k8sClusterName)
			return &fedvolumebackup.BRDataPlaneFailedError{
				Reason:  reasonVolumeRestoreMemberInvalid,
				Message: errMsg,
			}
		}
		if pingcapv1alpha1.IsRestoreFailed(restoreMember.restore) {
			errMsg := fmt.Sprintf("restore member %s of cluster %s is failed", restoreMemberName, k8sClusterName)
			return &fedvolumebackup.BRDataPlaneFailedError{
				Reason:  reasonVolumeRestoreMemberFailed,
				Message: errMsg,
			}
		}
	}

	for _, restoreMember := range restoreMembers {
		restoreMemberName := restoreMember.restore.Name
		k8sClusterName := restoreMember.k8sClusterName
		if !pingcapv1alpha1.IsRestoreVolumeComplete(restoreMember.restore) {
			return controller.IgnoreErrorf("restore member %s of cluster %s is not volume complete", restoreMemberName, k8sClusterName)
		}
	}
	// restore volume complete
	if !v1alpha1.IsVolumeRestoreVolumeComplete(volumeRestore) {
		rm.setVolumeRestoreVolumeComplete(volumeRestore)
	}

	return nil
}

func (rm *restoreManager) waitRestoreTiKVComplete(volumeRestore *v1alpha1.VolumeRestore, restoreMembers []*volumeRestoreMember) error {
	// check if restore members failed in data plane
	for _, restoreMember := range restoreMembers {
		restoreMemberName := restoreMember.restore.Name
		k8sClusterName := restoreMember.k8sClusterName
		if pingcapv1alpha1.IsRestoreInvalid(restoreMember.restore) {
			errMsg := fmt.Sprintf("restore member %s of cluster %s is invalid", restoreMemberName, k8sClusterName)
			return &fedvolumebackup.BRDataPlaneFailedError{
				Reason:  reasonVolumeRestoreMemberInvalid,
				Message: errMsg,
			}
		}
		if pingcapv1alpha1.IsRestoreFailed(restoreMember.restore) {
			errMsg := fmt.Sprintf("restore member %s of cluster %s is failed", restoreMemberName, k8sClusterName)
			return &fedvolumebackup.BRDataPlaneFailedError{
				Reason:  reasonVolumeRestoreMemberFailed,
				Message: errMsg,
			}
		}
	}

	walCheckCompleted := 0
	for _, restoreMember := range restoreMembers {
		restoreMemberName := restoreMember.restore.Name
		k8sClusterName := restoreMember.k8sClusterName
		// Here, it is possible restore already complete when warmup strategy is "check-wal-only".
		if restoreMember.restore.Spec.WarmupStrategy == pingcapv1alpha1.RestoreWarmupStrategyCheckOnly &&
			pingcapv1alpha1.IsRestoreComplete(restoreMember.restore) {
			walCheckCompleted += 1
			continue
		}
		if !pingcapv1alpha1.IsRestoreTiKVComplete(restoreMember.restore) {
			return controller.IgnoreErrorf("restore member %s of cluster %s is not tikv complete", restoreMemberName, k8sClusterName)
		}
	}

	klog.InfoS("all warm up tasks are completed.", "restore", volumeRestore.Name, "wal-check-only-completed", walCheckCompleted, "total", len(restoreMembers))
	if walCheckCompleted == len(restoreMembers) {
		// check wal complete, should we give it a special status?
		rm.setVolumeRestoreComplete(&volumeRestore.Status)
	} else if !v1alpha1.IsVolumeRestoreTiKVComplete(volumeRestore) {
		// restore tikv complete
		rm.setVolumeRestoreTiKVComplete(&volumeRestore.Status)
	}

	return nil
}

func (rm *restoreManager) syncWarmUpStatus(volumeRestore *v1alpha1.VolumeRestore, restoreMembers []*volumeRestoreMember) {
	for _, member := range restoreMembers {
		if !pingcapv1alpha1.IsRestoreWarmUpStarted(member.restore) {
			return
		}
	}
	if !v1alpha1.IsVolumeRestoreWarmUpStarted(volumeRestore) {
		rm.setVolumeRestoreWarmUpStarted(&volumeRestore.Status)
		return
	}

	for _, member := range restoreMembers {
		if !pingcapv1alpha1.IsRestoreWarmUpComplete(member.restore) {
			return
		}
	}
	if !v1alpha1.IsVolumeRestoreWarmUpComplete(volumeRestore) {
		rm.setVolumeRestoreWarmUpComplete(volumeRestore)
		return
	}
}

func (rm *restoreManager) executeRestoreDataPhase(ctx context.Context, volumeRestore *v1alpha1.VolumeRestore, restoreMembers []*volumeRestoreMember) (memberUpdated bool, err error) {
	if len(restoreMembers) != len(volumeRestore.Spec.Clusters) {
		return false, controller.RequeueErrorf("expect %d restore members but get %d when restore data", len(volumeRestore.Spec.Clusters), len(restoreMembers))
	}

	v1alpha1.StartVolumeRestoreStep(&volumeRestore.Status, v1alpha1.VolumeRestoreStepRestoreData)
	minResolvedTs := int64(math.MaxInt64)
	var minResolvedTsMember *volumeRestoreMember
	for _, restoreMember := range restoreMembers {
		// already in 2nd or 3rd phase
		if restoreMember.restore.Spec.FederalVolumeRestorePhase == pingcapv1alpha1.FederalVolumeRestoreData ||
			restoreMember.restore.Spec.FederalVolumeRestorePhase == pingcapv1alpha1.FederalVolumeRestoreFinish {
			return false, nil
		}

		restoreMemberName := restoreMember.restore.Name
		k8sClusterName := restoreMember.k8sClusterName
		commitTsStr := restoreMember.restore.Status.CommitTs
		if commitTsStr == "" {
			errMsg := fmt.Sprintf("commit ts in restore member %s of cluster %s is empty", restoreMemberName, k8sClusterName)
			return false, &fedvolumebackup.BRDataPlaneFailedError{
				Reason:  reasonVolumeRestoreMemberResolvedTsInvalid,
				Message: errMsg,
			}
		}
		resolvedTs, err := strconv.ParseInt(commitTsStr, 10, 64)
		if err != nil {
			errMsg := fmt.Sprintf("parse resolved ts %s to int64 in restore member %s of cluster %s error: %s", commitTsStr, restoreMemberName, k8sClusterName, err.Error())
			return false, &fedvolumebackup.BRDataPlaneFailedError{
				Reason:  reasonVolumeRestoreMemberResolvedTsInvalid,
				Message: errMsg,
			}
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
		return false, controller.RequeueErrorf("update FederalVolumeRestorePhase to restore-data in restore member %s of cluster %s error: %s", minResolvedTsMember.restore.Name, minResolvedTsMember.k8sClusterName, err.Error())
	}
	memberUpdated = true
	klog.Infof("VolumeRestore %s/%s update restore member %s to restore data", volumeRestore.Namespace, volumeRestore.Name, restoreCR.Name)
	return
}

func (rm *restoreManager) waitRestoreDataComplete(volumeRestore *v1alpha1.VolumeRestore, restoreMembers []*volumeRestoreMember) error {
	for _, restoreMember := range restoreMembers {
		restoreMemberName := restoreMember.restore.Name
		k8sClusterName := restoreMember.k8sClusterName
		if pingcapv1alpha1.IsRestoreDataComplete(restoreMember.restore) {
			if !v1alpha1.IsVolumeRestoreDataComplete(volumeRestore) {
				rm.setVolumeRestoreDataComplete(&volumeRestore.Status, restoreMember.restore.Status.CommitTs)
			}
			return nil
		}

		if pingcapv1alpha1.IsRestoreFailed(restoreMember.restore) {
			errMsg := fmt.Sprintf("restore member %s of cluster %s is failed", restoreMemberName, k8sClusterName)
			return &fedvolumebackup.BRDataPlaneFailedError{
				Reason:  reasonVolumeRestoreMemberFailed,
				Message: errMsg,
			}
		}
	}

	return controller.IgnoreErrorf("restore member is not restore data complete, waiting")
}

func (rm *restoreManager) executeRestoreFinishPhase(ctx context.Context, volumeRestore *v1alpha1.VolumeRestore, restoreMembers []*volumeRestoreMember) (memberUpdated bool, err error) {
	v1alpha1.StartVolumeRestoreStep(&volumeRestore.Status, v1alpha1.VolumeRestoreStepRestartTiKV)
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
			return false, controller.RequeueErrorf("update FederalVolumeRestorePhase to restore-finish in restore member %s of cluster %s error: %s", restoreMemberName, k8sClusterName, err.Error())
		}
		memberUpdated = true
		klog.Infof("VolumeRestore %s/%s update restore member %s of cluster %s to restore finish", volumeRestore.Namespace, volumeRestore.Name, restoreCR.Name, k8sClusterName)
	}
	return
}

func (rm *restoreManager) waitRestoreComplete(volumeRestore *v1alpha1.VolumeRestore, restoreMembers []*volumeRestoreMember) error {
	for _, restoreMember := range restoreMembers {
		restoreMemberName := restoreMember.restore.Name
		k8sClusterName := restoreMember.k8sClusterName
		if pingcapv1alpha1.IsRestoreFailed(restoreMember.restore) {
			errMsg := fmt.Sprintf("restore member %s of cluster %s is failed", restoreMemberName, k8sClusterName)
			return &fedvolumebackup.BRDataPlaneFailedError{
				Reason:  reasonVolumeRestoreMemberFailed,
				Message: errMsg,
			}
		}
		if !pingcapv1alpha1.IsRestoreComplete(restoreMember.restore) {
			return controller.IgnoreErrorf("restore member %s of cluster %s is not complete, waiting", restoreMemberName, k8sClusterName)
		}
	}
	return nil
}

func (rm *restoreManager) cleanVolumeRestore(volumeRestore *v1alpha1.VolumeRestore) error {
	ctx := context.Background()
	restoreMembers, err := rm.listRestoreMembers(ctx, volumeRestore)
	if err != nil {
		return err
	}

	for _, restoreMember := range restoreMembers {
		k8sClusterName := restoreMember.k8sClusterName
		restoreMemberName := restoreMember.restore.Name
		restoreMemberNamespace := restoreMember.restore.Namespace
		kubeClient := rm.deps.FedClientset[k8sClusterName]
		err := kubeClient.PingcapV1alpha1().Restores(restoreMemberNamespace).Delete(ctx, restoreMemberName, metav1.DeleteOptions{})
		if err != nil {
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

func (rm *restoreManager) setVolumeRestoreVolumeComplete(volumeRestore *v1alpha1.VolumeRestore) {
	volumeRestoreStatus := &volumeRestore.Status
	v1alpha1.UpdateVolumeRestoreCondition(volumeRestoreStatus, &v1alpha1.VolumeRestoreCondition{
		Type:   v1alpha1.VolumeRestoreVolumeComplete,
		Status: corev1.ConditionTrue,
	})
	v1alpha1.FinishVolumeRestoreStep(volumeRestoreStatus, v1alpha1.VolumeRestoreStepRestoreVolume)

	if isWarmUpSync(volumeRestore) {
		v1alpha1.StartVolumeRestoreStep(volumeRestoreStatus, v1alpha1.VolumeRestoreStepWarmUp)
	} else if isWarmUpAsync(volumeRestore) {
		v1alpha1.StartVolumeRestoreStep(volumeRestoreStatus, v1alpha1.VolumeRestoreStepStartTiKV)
		v1alpha1.StartVolumeRestoreStep(volumeRestoreStatus, v1alpha1.VolumeRestoreStepWarmUp)
	} else {
		v1alpha1.StartVolumeRestoreStep(volumeRestoreStatus, v1alpha1.VolumeRestoreStepStartTiKV)
	}
}

func (rm *restoreManager) setVolumeRestoreWarmUpStarted(volumeRestoreStatus *v1alpha1.VolumeRestoreStatus) {
	v1alpha1.UpdateVolumeRestoreCondition(volumeRestoreStatus, &v1alpha1.VolumeRestoreCondition{
		Type:   v1alpha1.VolumeRestoreWarmUpStarted,
		Status: corev1.ConditionTrue,
	})
}

func (rm *restoreManager) setVolumeRestoreWarmUpComplete(volumeRestore *v1alpha1.VolumeRestore) {
	volumeRestoreStatus := &volumeRestore.Status
	v1alpha1.UpdateVolumeRestoreCondition(volumeRestoreStatus, &v1alpha1.VolumeRestoreCondition{
		Type:   v1alpha1.VolumeRestoreWarmUpComplete,
		Status: corev1.ConditionTrue,
	})
	v1alpha1.FinishVolumeRestoreStep(volumeRestoreStatus, v1alpha1.VolumeRestoreStepWarmUp)

	if isWarmUpSync(volumeRestore) {
		v1alpha1.StartVolumeRestoreStep(volumeRestoreStatus, v1alpha1.VolumeRestoreStepStartTiKV)
	}
}

func (rm *restoreManager) setVolumeRestoreTiKVComplete(volumeRestoreStatus *v1alpha1.VolumeRestoreStatus) {
	v1alpha1.UpdateVolumeRestoreCondition(volumeRestoreStatus, &v1alpha1.VolumeRestoreCondition{
		Type:   v1alpha1.VolumeRestoreTiKVComplete,
		Status: corev1.ConditionTrue,
	})
	v1alpha1.FinishVolumeRestoreStep(volumeRestoreStatus, v1alpha1.VolumeRestoreStepStartTiKV)
}

func (rm *restoreManager) setVolumeRestoreDataComplete(volumeRestoreStatus *v1alpha1.VolumeRestoreStatus, commitTs string) {
	volumeRestoreStatus.CommitTs = commitTs
	v1alpha1.UpdateVolumeRestoreCondition(volumeRestoreStatus, &v1alpha1.VolumeRestoreCondition{
		Type:   v1alpha1.VolumeRestoreDataComplete,
		Status: corev1.ConditionTrue,
	})
	v1alpha1.FinishVolumeRestoreStep(volumeRestoreStatus, v1alpha1.VolumeRestoreStepRestoreData)
}

func (rm *restoreManager) setVolumeRestoreComplete(volumeRestoreStatus *v1alpha1.VolumeRestoreStatus) {
	volumeRestoreStatus.TimeCompleted = metav1.Now()
	volumeRestoreStatus.TimeTaken = volumeRestoreStatus.TimeCompleted.Sub(volumeRestoreStatus.TimeStarted.Time).Round(time.Second).String()
	v1alpha1.UpdateVolumeRestoreCondition(volumeRestoreStatus, &v1alpha1.VolumeRestoreCondition{
		Type:   v1alpha1.VolumeRestoreComplete,
		Status: corev1.ConditionTrue,
	})
}

func (rm *restoreManager) setVolumeRestoreFailed(volumeRestoreStatus *v1alpha1.VolumeRestoreStatus, reason, message string) {
	volumeRestoreStatus.TimeCompleted = metav1.Now()
	volumeRestoreStatus.TimeTaken = volumeRestoreStatus.TimeCompleted.Sub(volumeRestoreStatus.TimeStarted.Time).Round(time.Second).String()
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

func (rm *restoreManager) skipVolumeRestore(volumeRestore *v1alpha1.VolumeRestore) bool {
	return v1alpha1.IsVolumeRestoreComplete(volumeRestore) || v1alpha1.IsVolumeRestoreFailed(volumeRestore)
}

func (rm *restoreManager) buildRestoreMember(volumeRestoreName string, memberCluster *v1alpha1.VolumeRestoreMemberCluster, template *v1alpha1.VolumeRestoreMemberSpec, annotations map[string]string, labels map[string]string) *pingcapv1alpha1.Restore {
	restoreMember := &pingcapv1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:        rm.generateRestoreMemberName(volumeRestoreName),
			Namespace:   memberCluster.TCNamespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: pingcapv1alpha1.RestoreSpec{
			Mode:                      pingcapv1alpha1.RestoreModeVolumeSnapshot,
			Type:                      pingcapv1alpha1.BackupTypeFull,
			FederalVolumeRestorePhase: pingcapv1alpha1.FederalVolumeRestoreVolume,
			StorageProvider:           *memberCluster.Backup.StorageProvider.DeepCopy(),
			ResourceRequirements:      *template.ResourceRequirements.DeepCopy(),
			Env:                       template.Env,
			VolumeAZ:                  memberCluster.AZName,
			BR:                        template.BR.ToBRMemberConfig(memberCluster.TCName, memberCluster.TCNamespace),
			Tolerations:               template.Tolerations,
			ToolImage:                 template.ToolImage,
			ImagePullSecrets:          template.ImagePullSecrets,
			ServiceAccount:            template.ServiceAccount,
			PriorityClassName:         template.PriorityClassName,
			Warmup:                    template.Warmup,
			WarmupImage:               template.WarmupImage,
			WarmupStrategy:            template.WarmupStrategy,
			AdditionalVolumes:         template.AdditionalVolumes,
			AdditionalVolumeMounts:    template.AdditionalVolumeMounts,
		},
	}
	return restoreMember
}

func (rm *restoreManager) generateRestoreMemberName(volumeRestoreName string) string {
	return fmt.Sprintf("fed-%s", volumeRestoreName)
}

func isWarmUpSync(volumeRestore *v1alpha1.VolumeRestore) bool {
	return volumeRestore.Spec.Template.Warmup == pingcapv1alpha1.RestoreWarmupModeSync
}

func isWarmUpAsync(volumeRestore *v1alpha1.VolumeRestore) bool {
	return volumeRestore.Spec.Template.Warmup == pingcapv1alpha1.RestoreWarmupModeASync
}

type volumeRestoreMember struct {
	restore        *pingcapv1alpha1.Restore
	k8sClusterName string
}

var _ fedvolumebackup.RestoreManager = &restoreManager{}

type FakeRestoreManager struct {
	err           error
	updateStatus  bool
	statusUpdated bool
}

func NewFakeRestoreManager() *FakeRestoreManager {
	return &FakeRestoreManager{}
}

func (m *FakeRestoreManager) SetSyncError(err error) {
	m.err = err
}

func (m *FakeRestoreManager) SetUpdateStatus() {
	m.updateStatus = true
}

func (m *FakeRestoreManager) Sync(volumeRestore *v1alpha1.VolumeRestore) error {
	if m.updateStatus {
		v1alpha1.UpdateVolumeRestoreCondition(&volumeRestore.Status, &v1alpha1.VolumeRestoreCondition{
			Type:   v1alpha1.VolumeRestoreComplete,
			Status: corev1.ConditionTrue,
		})
	}
	return m.err
}

// UpdateStatus updates the status for a VolumeRestore, include condition and status info.
func (m *FakeRestoreManager) UpdateStatus(volumeRestore *v1alpha1.VolumeRestore, newStatus *v1alpha1.VolumeRestoreStatus) error {
	m.statusUpdated = true
	return nil
}

func (m *FakeRestoreManager) IsStatusUpdated() bool {
	return m.statusUpdated
}

var _ fedvolumebackup.RestoreManager = &FakeRestoreManager{}
