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

package fedvolumebackup

import (
	"context"
	"fmt"
	"math"
	"time"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/pingcap/tidb-operator/pkg/apis/federation/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/client/federation/clientset/versioned"
	informers "github.com/pingcap/tidb-operator/pkg/client/federation/informers/externalversions/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/fedvolumebackup"
	"github.com/pingcap/tidb-operator/pkg/metrics"
	"github.com/pingcap/tidb-operator/pkg/third_party/k8s"
)

// ControlInterface implements the control logic for updating VolumeBackup
// It is implemented as an interface to allow for extensions that provide different semantics.
// Currently, there is only one implementation.
type ControlInterface interface {
	// UpdateBackup implements the control logic for VolumeBackup's creation, deletion
	UpdateBackup(volumeBackup *v1alpha1.VolumeBackup) error
	// UpdateStatus updates the status for a VolumeBackup, include condition and status info
	// NOTE(federation): add a separate struct for newStatus as non-federation backup control did if needed
	UpdateStatus(volumeBackup *v1alpha1.VolumeBackup, newStatus *v1alpha1.VolumeBackupStatus) error
}

// NewDefaultVolumeBackupControl returns a new instance of the default VolumeBackup ControlInterface implementation.
func NewDefaultVolumeBackupControl(
	cli versioned.Interface,
	backupManager fedvolumebackup.BackupManager) ControlInterface {
	return &defaultBackupControl{
		cli,
		backupManager,
	}
}

type defaultBackupControl struct {
	cli           versioned.Interface
	backupManager fedvolumebackup.BackupManager
}

// UpdateBackup executes the core logic loop for a VolumeBackup.
func (c *defaultBackupControl) UpdateBackup(volumeBackup *v1alpha1.VolumeBackup) error {
	volumeBackup.SetGroupVersionKind(controller.FedVolumeBackupControllerKind)
	if err := c.addProtectionFinalizer(volumeBackup); err != nil {
		return err
	}

	if err := c.removeProtectionFinalizer(volumeBackup); err != nil {
		return err
	}

	return c.updateBackup(volumeBackup)
}

// UpdateStatus updates the status for a VolumeBackup, include condition and status info
func (c *defaultBackupControl) UpdateStatus(volumeBackup *v1alpha1.VolumeBackup, newStatus *v1alpha1.VolumeBackupStatus) error {
	return c.backupManager.UpdateStatus(volumeBackup, newStatus)
}

func (c *defaultBackupControl) updateBackup(volumeBackup *v1alpha1.VolumeBackup) error {
	oldBackup := volumeBackup.DeepCopy()
	ns := volumeBackup.GetNamespace()
	name := volumeBackup.GetName()

	err := c.backupManager.Sync(volumeBackup)
	if err != nil {
		klog.Warningf("VolumeBackup %s/%s sync error: %s", ns, name, err.Error())
	}

	c.initBackupStatusMetrics(volumeBackup)
	if !apiequality.Semantic.DeepEqual(oldBackup.Status, volumeBackup.Status) {
		klog.Infof("VolumeBackup %s/%s update status from %s to %s",
			ns, name, oldBackup.Status.Phase, volumeBackup.Status.Phase)
		if sErr := c.backupManager.UpdateStatus(volumeBackup, &volumeBackup.Status); sErr != nil {
			klog.Warningf("VolumeBackup %s/%s update status error: %s", ns, name, sErr.Error())
			if err == nil {
				err = sErr
			}
		} else {
			c.updateMetrics(oldBackup, volumeBackup)
		}
	}

	return err
}

func (c *defaultBackupControl) updateMetrics(oldBackup, newBackup *v1alpha1.VolumeBackup) {
	if !v1alpha1.IsVolumeBackupComplete(oldBackup) && v1alpha1.IsVolumeBackupComplete(newBackup) {
		c.updateVolumeBackupMetrics(newBackup)
	} else if !v1alpha1.IsVolumeBackupFailed(oldBackup) && v1alpha1.IsVolumeBackupFailed(newBackup) {
		c.updateVolumeBackupMetrics(newBackup)
	} else if !v1alpha1.IsVolumeBackupCleaned(oldBackup) && v1alpha1.IsVolumeBackupCleaned(newBackup) {
		c.updateVolumeBackupCleanupMetrics(newBackup)
	} else if !v1alpha1.IsVolumeBackupCleanFailed(oldBackup) && v1alpha1.IsVolumeBackupCleanFailed(newBackup) {
		c.updateVolumeBackupCleanupMetrics(newBackup)
	}
}

func (c *defaultBackupControl) updateVolumeBackupMetrics(volumeBackup *v1alpha1.VolumeBackup) {
	ns := volumeBackup.Namespace
	tcName := volumeBackup.GetCombinedTCName()
	status := string(volumeBackup.Status.Phase)
	metrics.FedVolumeBackupStatusCounterVec.WithLabelValues(ns, tcName, status).Inc()
	metrics.FedVolumeBackupTotalTimeCounterVec.WithLabelValues(ns, tcName).
		Add(volumeBackup.Status.TimeCompleted.Sub(volumeBackup.Status.TimeStarted.Time).Seconds())
	if volumeBackup.Status.BackupSize > 0 {
		metrics.FedVolumeBackupTotalSizeCounterVec.WithLabelValues(ns, tcName).Add(
			float64(volumeBackup.Status.BackupSize) / math.Pow(1024, 3))
	}
}

// add zero to the backup status counters; this sets the backup counter metric to zero when the federated manager restarts.
func (c *defaultBackupControl) initBackupStatusMetrics(volumeBackup *v1alpha1.VolumeBackup) {
	ns := volumeBackup.Namespace
	tcName := volumeBackup.GetCombinedTCName()
	metrics.FedVolumeBackupStatusCounterVec.GetMetricWithLabelValues(ns, tcName, string(v1alpha1.VolumeBackupRunning))
	metrics.FedVolumeBackupStatusCounterVec.GetMetricWithLabelValues(ns, tcName, string(v1alpha1.VolumeBackupSnapshotsCreated))
	metrics.FedVolumeBackupStatusCounterVec.GetMetricWithLabelValues(ns, tcName, string(v1alpha1.VolumeBackupComplete))
	metrics.FedVolumeBackupStatusCounterVec.GetMetricWithLabelValues(ns, tcName, string(v1alpha1.VolumeBackupFailed))
	metrics.FedVolumeBackupStatusCounterVec.GetMetricWithLabelValues(ns, tcName, string(v1alpha1.VolumeBackupCleaned))
}

func (c *defaultBackupControl) updateVolumeBackupCleanupMetrics(volumeBackup *v1alpha1.VolumeBackup) {
	ns := volumeBackup.Namespace
	tcName := volumeBackup.GetCombinedTCName()
	status := string(volumeBackup.Status.Phase)
	metrics.FedVolumeBackupCleanupStatusCounterVec.WithLabelValues(ns, tcName, status).Inc()
	metrics.FedVolumeBackupCleanupTotalTimeCounterVec.WithLabelValues(ns, tcName).
		Add(time.Since(volumeBackup.DeletionTimestamp.Time).Seconds())
}

// addProtectionFinalizer will be called when the VolumeBackup CR is created
func (c *defaultBackupControl) addProtectionFinalizer(volumeBackup *v1alpha1.VolumeBackup) error {
	ns := volumeBackup.GetNamespace()
	name := volumeBackup.GetName()

	if needToAddFinalizer(volumeBackup) {
		volumeBackup.Finalizers = append(volumeBackup.Finalizers, label.BackupProtectionFinalizer)
		_, err := c.cli.FederationV1alpha1().VolumeBackups(ns).Update(context.TODO(), volumeBackup, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("add VolumeBackup %s/%s protection finalizers failed, err: %v", ns, name, err)
		}
	}
	return nil
}

func (c *defaultBackupControl) removeProtectionFinalizer(volumeBackup *v1alpha1.VolumeBackup) error {
	ns := volumeBackup.GetNamespace()
	name := volumeBackup.GetName()

	if needToRemoveFinalizer(volumeBackup) {
		volumeBackup.Finalizers = k8s.RemoveString(volumeBackup.Finalizers, label.BackupProtectionFinalizer, nil)
		_, err := c.cli.FederationV1alpha1().VolumeBackups(ns).Update(context.TODO(), volumeBackup, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("remove VolumeBackup %s/%s protection finalizers failed, err: %v", ns, name, err)
		}
		klog.Infof("remove VolumeBackup %s/%s protection finalizers success", ns, name)
		return controller.RequeueErrorf(fmt.Sprintf("VolumeBackup %s/%s has been cleaned up", ns, name))
	}
	return nil
}

func needToAddFinalizer(volumeBackup *v1alpha1.VolumeBackup) bool {
	return volumeBackup.DeletionTimestamp == nil &&
		!k8s.ContainsString(volumeBackup.Finalizers, label.BackupProtectionFinalizer, nil)
}

func needToRemoveFinalizer(volumeBackup *v1alpha1.VolumeBackup) bool {
	return isDeletionCandidate(volumeBackup) && v1alpha1.IsVolumeBackupCleaned(volumeBackup)
}

func isDeletionCandidate(volumeBackup *v1alpha1.VolumeBackup) bool {
	return volumeBackup.DeletionTimestamp != nil && k8s.ContainsString(volumeBackup.Finalizers, label.BackupProtectionFinalizer, nil)
}

var _ ControlInterface = &defaultBackupControl{}

// FakeBackupControl is a fake BackupControlInterface
type FakeBackupControl struct {
	backupIndexer       cache.Indexer
	updateBackupTracker controller.RequestTracker
	condition           *v1alpha1.VolumeBackupCondition
}

// NewFakeBackupControl returns a FakeBackupControl
func NewFakeBackupControl(backupInformer informers.VolumeBackupInformer) *FakeBackupControl {
	return &FakeBackupControl{
		backupInformer.Informer().GetIndexer(),
		controller.RequestTracker{},
		nil,
	}
}

// SetUpdateBackupError sets the error attributes of updateBackupTracker
func (c *FakeBackupControl) SetUpdateBackupError(err error, after int) {
	c.updateBackupTracker.SetError(err).SetAfter(after)
}

// UpdateBackup adds the VolumeBackup to BackupIndexer
func (c *FakeBackupControl) UpdateBackup(volumeBackup *v1alpha1.VolumeBackup) error {
	defer c.updateBackupTracker.Inc()
	if c.updateBackupTracker.ErrorReady() {
		defer c.updateBackupTracker.Reset()
		return c.updateBackupTracker.GetError()
	}

	return c.backupIndexer.Add(volumeBackup)
}

// UpdateStatus updates the status for a VolumeBackup, include condition and status info
func (c *FakeBackupControl) UpdateStatus(_ *v1alpha1.VolumeBackup, newStatus *v1alpha1.VolumeBackupStatus) error {
	if len(newStatus.Conditions) > 0 {
		c.condition = &newStatus.Conditions[0]
	}
	return nil
}

var _ ControlInterface = &FakeBackupControl{}
