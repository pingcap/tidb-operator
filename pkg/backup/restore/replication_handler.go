// Copyright 2026 PingCAP, Inc.
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
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/retry"
)

// replicationHandler owns the reconcile loop for replication restore
// (Spec.Mode == pitr && Spec.ReplicationConfig != nil). Dispatch keys on
// Status.ReplicationStep so that re-entry after controller restart is
// deterministic regardless of transient Phase writes by backup-manager in
// phase-2.
//
// State flow:
//   ""               → create phase-1 Job, set Phase=SnapshotRestore, Step="snapshot-restore"
//   "snapshot-restore" → observe phase-1 Job + CompactBackup,
//                        write markers, gate to LogRestore when both True
//   "log-restore"    → observe phase-2 Job; backup-manager drives Phase
//                      to Running / Complete / Failed via the real updater
type replicationHandler struct {
	deps          *controller.Dependencies
	statusUpdater controller.RestoreConditionUpdaterInterface
}

func newReplicationHandler(
	deps *controller.Dependencies,
	statusUpdater controller.RestoreConditionUpdaterInterface,
) *replicationHandler {
	return &replicationHandler{deps: deps, statusUpdater: statusUpdater}
}

// Sync is the entry point called from restoreManager.syncRestoreJob when
// the restore is in replication mode.
func (h *replicationHandler) Sync(restore *v1alpha1.Restore) error {
	if v1alpha1.IsRestoreComplete(restore) || v1alpha1.IsRestoreFailed(restore) {
		return nil
	}
	switch restore.Status.ReplicationStep {
	case "":
		return h.syncInitial(restore)
	case "snapshot-restore":
		return h.syncSnapshotRestore(restore)
	case "log-restore":
		return h.syncLogRestore(restore)
	default:
		// Defensive: unknown step is a bug in a past reconcile. Do not
		// create work or write status from here; surface via log and let
		// next reconcile re-evaluate.
		return nil
	}
}

func (h *replicationHandler) syncInitial(_ *v1alpha1.Restore) error {
	// Implemented in Task 6.
	return nil
}

func (h *replicationHandler) syncSnapshotRestore(_ *v1alpha1.Restore) error {
	// Implemented in Task 7.
	return nil
}

func (h *replicationHandler) syncLogRestore(_ *v1alpha1.Restore) error {
	// Implemented in Task 8.
	return nil
}

// appendRestoreMarker sets or replaces a condition marker in restore.Status.Conditions
// WITHOUT touching restore.Status.Phase. Use for SnapshotRestored and CompactSettled.
//
// This is why we do NOT go through v1alpha1.UpdateRestoreCondition: that helper
// unconditionally assigns status.Phase = condition.Type, which would overwrite
// the controller-set SnapshotRestore / LogRestore phase.
func appendRestoreMarker(
	restore *v1alpha1.Restore,
	markerType v1alpha1.RestoreConditionType,
	reason, message string,
) {
	now := metav1.Now()
	newCond := v1alpha1.RestoreCondition{
		Type:               markerType,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	}
	for i, c := range restore.Status.Conditions {
		if c.Type == markerType {
			// Replace; preserve original LastTransitionTime if Status unchanged
			if c.Status == corev1.ConditionTrue {
				newCond.LastTransitionTime = c.LastTransitionTime
			}
			restore.Status.Conditions[i] = newCond
			return
		}
	}
	restore.Status.Conditions = append(restore.Status.Conditions, newCond)
}

// updateRestoreMarker persists an appendRestoreMarker change via the lister+client.
// Retries on conflict using retry.OnError, matching realRestoreConditionUpdater pattern.
func (h *replicationHandler) updateRestoreMarker(
	restore *v1alpha1.Restore,
	markerType v1alpha1.RestoreConditionType,
	reason, message string,
) error {
	ns, name := restore.Namespace, restore.Name
	err := retry.OnError(retry.DefaultRetry, func(e error) bool { return e != nil }, func() error {
		latest, getErr := h.deps.RestoreLister.Restores(ns).Get(name)
		if getErr != nil {
			utilruntime.HandleError(fmt.Errorf("error getting updated restore %s/%s from lister: %v", ns, name, getErr))
			return getErr
		}
		updated := latest.DeepCopy()
		appendRestoreMarker(updated, markerType, reason, message)
		_, updateErr := h.deps.Clientset.PingcapV1alpha1().Restores(ns).Update(
			context.TODO(), updated, metav1.UpdateOptions{},
		)
		return updateErr
	})
	return err
}

// lookupCompactBackup finds the referenced CompactBackup. Returns:
//   - (cb, nil) when found
//   - (nil, nil) when not found and WaitTimeout has NOT expired — caller
//     should requeue and continue reconciling
//   - (nil, controller.IgnoreErrorf) when WaitTimeout has expired
//     (caller writes Phase=Failed with reason CompactBackupWaitTimeout)
//   - (nil, err) on other errors
func (h *replicationHandler) lookupCompactBackup(restore *v1alpha1.Restore) (*v1alpha1.CompactBackup, error) {
	cfg := restore.Spec.ReplicationConfig
	cb, err := h.deps.CompactBackupLister.CompactBackups(restore.Namespace).Get(cfg.CompactBackupName)
	if err == nil {
		return cb, nil
	}
	if !errors.IsNotFound(err) {
		return nil, fmt.Errorf("lookup CompactBackup %s/%s: %w",
			restore.Namespace, cfg.CompactBackupName, err)
	}

	// NotFound: decide late-binding wait vs timeout failure.
	if cfg.WaitTimeout == nil || cfg.WaitTimeout.Duration == 0 {
		return nil, nil // wait indefinitely; requeue
	}
	if time.Since(restore.CreationTimestamp.Time) > cfg.WaitTimeout.Duration {
		return nil, controller.IgnoreErrorf(
			"CompactBackup %s/%s not found after waitTimeout %v",
			restore.Namespace, cfg.CompactBackupName, cfg.WaitTimeout.Duration,
		)
	}
	return nil, nil // waiting
}

// checkCrossCRConsistency validates that the Restore and CompactBackup target
// the same cluster and storage location. Returns nil on consistency; an error
// with the mismatching field(s) otherwise. secretName differences are ignored
// because the two CRs may reference different Secrets that point at the same
// storage.
func checkCrossCRConsistency(restore *v1alpha1.Restore, cb *v1alpha1.CompactBackup) error {
	if restore.Spec.BR == nil || cb.Spec.BR == nil {
		return fmt.Errorf("missing BR spec on one of the CRs")
	}
	if restore.Spec.BR.Cluster != cb.Spec.BR.Cluster {
		return fmt.Errorf("br.cluster mismatch: restore=%q compact=%q",
			restore.Spec.BR.Cluster, cb.Spec.BR.Cluster)
	}
	if restore.Spec.BR.ClusterNamespace != cb.Spec.BR.ClusterNamespace {
		return fmt.Errorf("br.clusterNamespace mismatch: restore=%q compact=%q",
			restore.Spec.BR.ClusterNamespace, cb.Spec.BR.ClusterNamespace)
	}
	return compareStorageLocation(restore.Spec.StorageProvider, cb.Spec.StorageProvider)
}

// compareStorageLocation compares only location-defining fields (bucket/prefix/
// region for S3; bucket/prefix for GCS; container/prefix for AzBlob). Credentials
// (SecretName, etc.) are intentionally ignored.
func compareStorageLocation(a, b v1alpha1.StorageProvider) error {
	// Both sides empty: pragmatically a no-op (validating webhook should
	// reject this upstream), but be explicit rather than falling through
	// to the "type differs" message.
	if a.S3 == nil && a.Gcs == nil && a.Azblob == nil &&
		b.S3 == nil && b.Gcs == nil && b.Azblob == nil {
		return nil
	}
	switch {
	case a.S3 != nil && b.S3 != nil:
		if a.S3.Bucket != b.S3.Bucket {
			return fmt.Errorf("s3.bucket mismatch: %q vs %q", a.S3.Bucket, b.S3.Bucket)
		}
		if a.S3.Prefix != b.S3.Prefix {
			return fmt.Errorf("s3.prefix mismatch: %q vs %q", a.S3.Prefix, b.S3.Prefix)
		}
		if a.S3.Region != b.S3.Region {
			return fmt.Errorf("s3.region mismatch: %q vs %q", a.S3.Region, b.S3.Region)
		}
		return nil
	case a.Gcs != nil && b.Gcs != nil:
		if a.Gcs.Bucket != b.Gcs.Bucket {
			return fmt.Errorf("gcs.bucket mismatch: %q vs %q", a.Gcs.Bucket, b.Gcs.Bucket)
		}
		if a.Gcs.Prefix != b.Gcs.Prefix {
			return fmt.Errorf("gcs.prefix mismatch: %q vs %q", a.Gcs.Prefix, b.Gcs.Prefix)
		}
		return nil
	case a.Azblob != nil && b.Azblob != nil:
		if a.Azblob.Container != b.Azblob.Container {
			return fmt.Errorf("azblob.container mismatch: %q vs %q", a.Azblob.Container, b.Azblob.Container)
		}
		if a.Azblob.Prefix != b.Azblob.Prefix {
			return fmt.Errorf("azblob.prefix mismatch: %q vs %q", a.Azblob.Prefix, b.Azblob.Prefix)
		}
		return nil
	default:
		return fmt.Errorf("storage provider type differs between Restore and CompactBackup")
	}
}
