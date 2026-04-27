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
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
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
