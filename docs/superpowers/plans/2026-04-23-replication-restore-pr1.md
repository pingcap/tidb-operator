# Replication Restore — Operator Side (PR 1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement operator-side orchestration for replication restore (Restore CR with `mode=pitr` + `replicationConfig` set). Controller owns all `status.phase` writes, creates two BR Jobs (phase-1 snapshot restore, phase-2 log restore) gated on CompactBackup terminal state. backup-manager CLI changes are in PR 2.

**Architecture:** Handler pattern — in `restore_manager.syncRestoreJob`, intercept `mode=pitr && replicationConfig != nil` and dispatch to `replicationHandler.Sync`. Handler state machine keys on `status.ReplicationStep` (`""` / `"snapshot-restore"` / `"log-restore"`). Two BR Jobs identified by labels. CompactBackup observed via shared informer (lister already in `Dependencies`). Phase writes go through `UpdateRestoreCondition` (which sets `status.Phase`). Marker writes (`SnapshotRestored`, `CompactSettled`) go through handler-private `appendRestoreMarker` which only mutates `status.Conditions[]`, preserving Phase.

**Tech Stack:** Go, tidb-operator controller framework, `client-go` informers/listers, `k8s.io/api/batch/v1` Jobs.

**Spec reference:** `docs/superpowers/specs/2026-04-23-replication-restore-design.md`

---

## File-level scope

| File | Create / Modify |
|------|-----------------|
| `pkg/apis/pingcap/v1alpha1/types.go` | Modify — add `ReplicationConfig` struct + 2 fields |
| `pkg/apis/pingcap/v1alpha1/restore.go` | Modify — add 4 `RestoreConditionType` constants |
| `pkg/apis/pingcap/v1alpha1/zz_generated.deepcopy.go` | Regenerate |
| `pkg/apis/pingcap/v1alpha1/openapi_generated.go` | Regenerate |
| `manifests/crd/v1/pingcap.com_restores.yaml`, `manifests/crd.yaml` | Regenerate |
| `pkg/apis/label/label.go` | Modify — add 3 constants |
| `pkg/controller/restore_status_updater.go` | Modify — add `replicationRestoreStatusUpdater` no-op |
| `pkg/controller/restore_status_updater_test.go` | Modify — unit test for wrapper |
| `pkg/backup/restore/replication_handler.go` | **Create** — handler + helpers |
| `pkg/backup/restore/replication_handler_test.go` | **Create** — handler unit tests |
| `pkg/backup/restore/restore_manager.go` | Modify — inject handler, add interception |
| `pkg/controller/restore/restore_controller.go` | Modify — add CompactBackup EventHandler |

---

## Task 1: CRD types and condition constants

**Files:**
- Modify: `pkg/apis/pingcap/v1alpha1/types.go`
- Modify: `pkg/apis/pingcap/v1alpha1/restore.go`
- Regenerate: `pkg/apis/pingcap/v1alpha1/zz_generated.deepcopy.go`, `pkg/apis/pingcap/v1alpha1/openapi_generated.go`, `manifests/crd/v1/pingcap.com_restores.yaml`, `manifests/crd.yaml`

- [ ] **Step 1: Add `ReplicationConfig` struct and Restore spec/status fields**

In `pkg/apis/pingcap/v1alpha1/types.go`, find `RestoreSpec struct` and append inside the struct block:

```go
	// ReplicationConfig is the optional configuration for replication restore.
	// When Mode == pitr and this field is non-nil, the controller runs a
	// two-phase replication restore (snapshot restore + log restore gated on
	// CompactBackup terminal state). When nil, the controller runs a standard
	// PiTR restore (existing behavior unchanged).
	// +optional
	ReplicationConfig *ReplicationConfig `json:"replicationConfig,omitempty"`
```

Add the struct type near `RestoreSpec` (same file, after it):

```go
// ReplicationConfig holds the replication-specific configuration for PiTR restore.
type ReplicationConfig struct {
	// CompactBackupName references a CompactBackup CR in the same namespace.
	// The referenced CR must reach a terminal state (Complete or Failed)
	// before the controller proceeds from phase-1 (snapshot restore) to
	// phase-2 (log restore).
	CompactBackupName string `json:"compactBackupName"`

	// WaitTimeout bounds how long the controller waits for a missing
	// CompactBackup CR to appear. 0 means wait indefinitely.
	// This timeout does NOT apply when the CompactBackup exists but is
	// still running; compaction duration is business-dependent.
	// +optional
	WaitTimeout *metav1.Duration `json:"waitTimeout,omitempty"`
}
```

In the same file, find `RestoreStatus struct` and append inside the struct block:

```go
	// ReplicationStep identifies the current phase of replication restore.
	// Values: "" (not replication), "snapshot-restore", "log-restore".
	// Set by the controller when creating each phase's Job.
	// +optional
	ReplicationStep string `json:"replicationStep,omitempty"`
```

- [ ] **Step 2: Add condition type constants**

In `pkg/apis/pingcap/v1alpha1/restore.go`, find the `const (` block that defines `RestoreScheduled`, `RestoreRunning`, etc., and append:

```go
	// RestoreSnapshotRestore means the replication restore is in phase-1:
	// BR snapshot restore running in parallel with CompactBackup shards.
	// This is a phase value (drives status.phase).
	RestoreSnapshotRestore RestoreConditionType = "SnapshotRestore"

	// RestoreLogRestore means the replication restore is in phase-2:
	// BR log restore after the phase-1 gate has passed.
	// This is a phase value (drives status.phase).
	RestoreLogRestore RestoreConditionType = "LogRestore"

	// RestoreSnapshotRestored indicates the phase-1 BR snapshot restore Job
	// has completed successfully. This is a condition marker — it is
	// appended to status.conditions but does NOT drive status.phase.
	// Written by handler.appendRestoreMarker, not by UpdateRestoreCondition.
	RestoreSnapshotRestored RestoreConditionType = "SnapshotRestored"

	// RestoreCompactSettled indicates the referenced CompactBackup has
	// reached a terminal state (Complete or Failed). This is a condition
	// marker — it is appended to status.conditions but does NOT drive
	// status.phase. Written by handler.appendRestoreMarker.
	RestoreCompactSettled RestoreConditionType = "CompactSettled"
```

- [ ] **Step 3: Regenerate deepcopy, openapi, and CRD manifests**

Run:

```bash
make generate
make manifests
```

Expected: changes appear in `zz_generated.deepcopy.go` (DeepCopy/DeepCopyInto methods for `ReplicationConfig` and new fields in `RestoreSpec` / `RestoreStatus`), `openapi_generated.go`, and the CRD YAML files show the new fields with correct schema.

- [ ] **Step 4: Verify types compile and round-trip**

```bash
go build ./pkg/apis/...
go test -run TestRestoreDeepCopy ./pkg/apis/pingcap/v1alpha1/... 2>&1 | tail -5
```

Expected: build succeeds; any existing deepcopy tests still pass (new fields have generated deepcopy methods).

- [ ] **Step 5: Commit**

```bash
git add pkg/apis/pingcap/v1alpha1/types.go \
        pkg/apis/pingcap/v1alpha1/restore.go \
        pkg/apis/pingcap/v1alpha1/zz_generated.deepcopy.go \
        pkg/apis/pingcap/v1alpha1/openapi_generated.go \
        manifests/crd/v1/pingcap.com_restores.yaml \
        manifests/crd.yaml
git commit -m "feat(replication-restore): add CRD types and condition constants

Adds ReplicationConfig struct (CompactBackupName, WaitTimeout) under
RestoreSpec.ReplicationConfig, RestoreStatus.ReplicationStep string,
and four new RestoreConditionType constants: SnapshotRestore and
LogRestore (phase values) plus SnapshotRestored and CompactSettled
(markers). No behavior change in this commit; consumed by subsequent
handler and wrapper commits."
```

---

## Task 2: Label constants for Job identification

**Files:**
- Modify: `pkg/apis/label/label.go`

- [ ] **Step 1: Add replication-step label constants**

In `pkg/apis/label/label.go`, find the const block with `RestoreLabelKey` (~line 63) and append:

```go
	// ReplicationStepLabelKey distinguishes the two BR Jobs of a replication
	// restore. Used by Restore controller to locate Jobs by label selector
	// rather than name pattern.
	ReplicationStepLabelKey string = "tidb.pingcap.com/replication-step"

	// ReplicationStepSnapshotRestoreVal is the label value for the phase-1 BR
	// Job (snapshot restore + checkpoint set).
	ReplicationStepSnapshotRestoreVal string = "snapshot-restore"

	// ReplicationStepLogRestoreVal is the label value for the phase-2 BR
	// Job (log restore).
	ReplicationStepLogRestoreVal string = "log-restore"
```

- [ ] **Step 2: Verify compiles**

```bash
go build ./pkg/apis/label/...
```

Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add pkg/apis/label/label.go
git commit -m "feat(replication-restore): add ReplicationStepLabelKey constants

Labels are used to identify phase-1 vs phase-2 BR Jobs by selector
rather than name pattern, so renaming the Job doesn't break the
handler's idempotence checks."
```

---

## Task 3: No-op wrapper for backup-manager phase-1

**Files:**
- Modify: `pkg/controller/restore_status_updater.go`
- Modify: `pkg/controller/restore_status_updater_test.go`

- [ ] **Step 1: Write failing test**

In `pkg/controller/restore_status_updater_test.go`, append:

```go
func TestReplicationRestoreStatusUpdater_Update_IsNoOp(t *testing.T) {
	g := NewGomegaWithT(t)

	u := NewReplicationRestoreStatusUpdater()

	err := u.Update(
		&v1alpha1.Restore{},
		&v1alpha1.RestoreCondition{Type: v1alpha1.RestoreRunning, Status: corev1.ConditionTrue},
		&RestoreUpdateStatus{},
	)
	g.Expect(err).NotTo(HaveOccurred())

	// Nil args must also be safe.
	err = u.Update(nil, nil, nil)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestReplicationRestoreStatusUpdater_ImplementsInterface(t *testing.T) {
	g := NewGomegaWithT(t)
	var _ RestoreConditionUpdaterInterface = NewReplicationRestoreStatusUpdater()
	g.Expect(true).To(BeTrue()) // compile-time check
}
```

Imports needed in that file (add if missing): `corev1 "k8s.io/api/core/v1"`, `"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"`, `. "github.com/onsi/gomega"`.

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestReplicationRestoreStatusUpdater -v ./pkg/controller/...
```

Expected: FAIL with `NewReplicationRestoreStatusUpdater` undefined.

- [ ] **Step 3: Implement the wrapper**

In `pkg/controller/restore_status_updater.go`, append at end of file:

```go
// replicationRestoreStatusUpdater is a no-op RestoreConditionUpdaterInterface
// used by backup-manager in replication restore phase-1. All writes are
// suppressed so that the controller remains the sole writer of status.Phase
// and condition markers during phase-1. Phase-2 uses NewRealRestoreConditionUpdater
// directly (no wrap) because from backup-manager's perspective phase-2 is
// a standard PiTR restore.
//
// The activation point is cmd/backup-manager/app/cmd/restore.go (landed in PR 2).
type replicationRestoreStatusUpdater struct{}

// NewReplicationRestoreStatusUpdater returns a no-op updater. See type doc
// above for the rationale (Option B in the spec: controller owns all Phase
// writes during phase-1; progress/timing writes from backup-manager are
// dropped because the CompactBackup sharded status already carries the
// user-visible progress, and TimeStarted is written by the controller when
// it creates the phase-1 Job).
func NewReplicationRestoreStatusUpdater() RestoreConditionUpdaterInterface {
	return &replicationRestoreStatusUpdater{}
}

func (r *replicationRestoreStatusUpdater) Update(
	_ *v1alpha1.Restore,
	_ *v1alpha1.RestoreCondition,
	_ *RestoreUpdateStatus,
) error {
	return nil
}

var _ RestoreConditionUpdaterInterface = &replicationRestoreStatusUpdater{}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run TestReplicationRestoreStatusUpdater -v ./pkg/controller/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/controller/restore_status_updater.go \
        pkg/controller/restore_status_updater_test.go
git commit -m "feat(replication-restore): add no-op status updater wrapper

In replication restore phase-1, backup-manager must not write
status.phase or conditions — the controller is the sole writer.
This wrapper drops all Update calls. Activated from the
backup-manager CLI in PR 2 via --replicationPhase=1; phase-2 uses
the real updater, matching the wrap-pattern shipped for CompactBackup
sharded mode."
```

---

## Task 4: Replication handler skeleton and dispatcher

**Files:**
- Create: `pkg/backup/restore/replication_handler.go`
- Create: `pkg/backup/restore/replication_handler_test.go`

- [ ] **Step 1: Write failing test for dispatch table**

Create `pkg/backup/restore/replication_handler_test.go` with:

```go
// Copyright 2026 PingCAP, Inc.
// Licensed under the Apache License, Version 2.0 (the "License").
package restore

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

func TestReplicationHandler_Dispatch_TerminalIsNoOp(t *testing.T) {
	g := NewGomegaWithT(t)
	h := &replicationHandler{}

	terminal := &v1alpha1.Restore{
		Status: v1alpha1.RestoreStatus{
			Conditions: []v1alpha1.RestoreCondition{
				{Type: v1alpha1.RestoreComplete, Status: "True"},
			},
		},
	}
	err := h.Sync(terminal)
	g.Expect(err).NotTo(HaveOccurred())

	failed := &v1alpha1.Restore{
		Status: v1alpha1.RestoreStatus{
			Conditions: []v1alpha1.RestoreCondition{
				{Type: v1alpha1.RestoreFailed, Status: "True"},
			},
		},
	}
	err = h.Sync(failed)
	g.Expect(err).NotTo(HaveOccurred())
}
```

- [ ] **Step 2: Run to verify failing**

```bash
go test -run TestReplicationHandler_Dispatch -v ./pkg/backup/restore/...
```

Expected: FAIL with `replicationHandler` undefined.

- [ ] **Step 3: Create skeleton**

Create `pkg/backup/restore/replication_handler.go`:

```go
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
```

- [ ] **Step 4: Run to verify passing**

```bash
go test -run TestReplicationHandler_Dispatch -v ./pkg/backup/restore/...
```

Expected: PASS (terminal restores return nil without touching anything).

- [ ] **Step 5: Commit**

```bash
git add pkg/backup/restore/replication_handler.go \
        pkg/backup/restore/replication_handler_test.go
git commit -m "feat(replication-restore): add handler skeleton with dispatch

Skeleton dispatches by Status.ReplicationStep (empty / snapshot-restore /
log-restore). Terminal restores are no-ops. Per-step logic arrives in
subsequent commits."
```

---

## Task 5: Handler helpers (appendRestoreMarker, CompactBackup lookup, consistency check)

**Files:**
- Modify: `pkg/backup/restore/replication_handler.go`
- Modify: `pkg/backup/restore/replication_handler_test.go`

- [ ] **Step 1: Write failing tests**

Append to `replication_handler_test.go`:

```go
func TestAppendRestoreMarker_PreservesPhase(t *testing.T) {
	g := NewGomegaWithT(t)

	r := &v1alpha1.Restore{
		Status: v1alpha1.RestoreStatus{
			Phase: v1alpha1.RestoreSnapshotRestore,
		},
	}
	appendRestoreMarker(r, v1alpha1.RestoreSnapshotRestored, "JobComplete", "phase-1 Job complete")

	g.Expect(r.Status.Phase).To(Equal(v1alpha1.RestoreSnapshotRestore), "Phase must not be touched")
	g.Expect(r.Status.Conditions).To(HaveLen(1))
	g.Expect(r.Status.Conditions[0].Type).To(Equal(v1alpha1.RestoreSnapshotRestored))
	g.Expect(r.Status.Conditions[0].Reason).To(Equal("JobComplete"))
}

func TestAppendRestoreMarker_Idempotent(t *testing.T) {
	g := NewGomegaWithT(t)

	r := &v1alpha1.Restore{
		Status: v1alpha1.RestoreStatus{
			Phase: v1alpha1.RestoreSnapshotRestore,
		},
	}
	appendRestoreMarker(r, v1alpha1.RestoreCompactSettled, "AllShardsComplete", "")
	appendRestoreMarker(r, v1alpha1.RestoreCompactSettled, "AllShardsComplete", "")

	g.Expect(r.Status.Conditions).To(HaveLen(1), "duplicate same-type marker must replace, not append")
}

func TestConsistencyCheck_StorageBucketMismatch(t *testing.T) {
	g := NewGomegaWithT(t)

	r := &v1alpha1.Restore{
		Spec: v1alpha1.RestoreSpec{
			BR: &v1alpha1.BRConfig{Cluster: "c1", ClusterNamespace: "ns"},
			StorageProvider: v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{Bucket: "a", Prefix: "/p", Region: "us-west-2"},
			},
		},
	}
	cb := &v1alpha1.CompactBackup{
		Spec: v1alpha1.CompactSpec{
			BR: &v1alpha1.BRConfig{Cluster: "c1", ClusterNamespace: "ns"},
			StorageProvider: v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{Bucket: "b", Prefix: "/p", Region: "us-west-2"},
			},
		},
	}
	err := checkCrossCRConsistency(r, cb)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("bucket"))
}

func TestConsistencyCheck_OK(t *testing.T) {
	g := NewGomegaWithT(t)

	spec := v1alpha1.StorageProvider{
		S3: &v1alpha1.S3StorageProvider{Bucket: "a", Prefix: "/p", Region: "us-west-2", SecretName: "different-secret"},
	}
	r := &v1alpha1.Restore{
		Spec: v1alpha1.RestoreSpec{
			BR:              &v1alpha1.BRConfig{Cluster: "c1", ClusterNamespace: "ns"},
			StorageProvider: spec,
		},
	}
	cb := &v1alpha1.CompactBackup{
		Spec: v1alpha1.CompactSpec{
			BR: &v1alpha1.BRConfig{Cluster: "c1", ClusterNamespace: "ns"},
			StorageProvider: v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{Bucket: "a", Prefix: "/p", Region: "us-west-2", SecretName: "original-secret"},
			},
		},
	}
	err := checkCrossCRConsistency(r, cb)
	g.Expect(err).NotTo(HaveOccurred(), "SecretName difference must be ignored")
}
```

- [ ] **Step 2: Run tests — expect failures**

```bash
go test -run 'TestAppendRestoreMarker|TestConsistencyCheck' -v ./pkg/backup/restore/...
```

Expected: FAIL — `appendRestoreMarker` and `checkCrossCRConsistency` undefined.

- [ ] **Step 3: Implement helpers**

Append to `pkg/backup/restore/replication_handler.go`:

```go
import (
	"context"
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
)

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
// Retries on conflict using the standard retry.OnError pattern used elsewhere in
// pkg/controller.
func (h *replicationHandler) updateRestoreMarker(
	restore *v1alpha1.Restore,
	markerType v1alpha1.RestoreConditionType,
	reason, message string,
) error {
	ns, name := restore.Namespace, restore.Name
	latest, err := h.deps.RestoreLister.Restores(ns).Get(name)
	if err != nil {
		return fmt.Errorf("get restore %s/%s from lister: %w", ns, name, err)
	}
	updated := latest.DeepCopy()
	appendRestoreMarker(updated, markerType, reason, message)
	_, err = h.deps.Clientset.PingcapV1alpha1().Restores(ns).Update(
		context.TODO(), updated, metav1.UpdateOptions{},
	)
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
```

- [ ] **Step 4: Run tests — expect pass**

```bash
go test -run 'TestAppendRestoreMarker|TestConsistencyCheck' -v ./pkg/backup/restore/...
```

Expected: PASS on all 4 tests.

- [ ] **Step 5: Commit**

```bash
git add pkg/backup/restore/replication_handler.go \
        pkg/backup/restore/replication_handler_test.go
git commit -m "feat(replication-restore): add handler helpers

appendRestoreMarker writes markers to Conditions without touching Phase
(preserves SnapshotRestore / LogRestore phase values during marker
observation). lookupCompactBackup encapsulates late-binding with
WaitTimeout. checkCrossCRConsistency validates br.cluster/namespace and
storage location, ignoring secretName differences."
```

---

## Task 6: Handler initial case (`""` → phase-1 Job creation)

> **Plan deviation (superseded by commit `850a5be40`):** The `failRestore("CompactBackupMismatch", ...)` and "compact failure ⇒ Restore Failed" paths described below were replaced during implementation by writing CompactBackup outcomes to the `CompactSettled` condition marker. Per the design spec, internal BR phase-2 falls back to uncompacted log on compact failure, so the Restore is no longer terminated. References to `failRestore("CompactBackup*", ...)` in this and the next task are kept for historical context only.
>
> **Job-name + UID labelling fix-up:** Job lookup uses Get-by-name and verifies the new `label.RestoreUIDLabelKey` written by `makeReplicationBRJob`, so a Job leftover from a deleted same-name predecessor is rejected explicitly rather than mistaken for ours. The `listJobsBySelector` helper described in this task was removed.

**Files:**
- Modify: `pkg/backup/restore/replication_handler.go`
- Modify: `pkg/backup/restore/replication_handler_test.go`

- [ ] **Step 1: Write failing tests**

Append to `replication_handler_test.go`:

```go
func TestSyncInitial_NotFoundCompactBackup_WithinTimeout_Requeues(t *testing.T) {
	g := NewGomegaWithT(t)
	h, _ := newTestReplicationHandler(t)

	r := newReplicationRestoreFixture("r1", "ns", "nonexistent-cb", ptrDuration(10*time.Minute))
	err := h.Sync(r)

	// NotFound within timeout: return nil (requeue from the Restore controller's
	// own resync pacing + CompactBackup informer).
	g.Expect(err).NotTo(HaveOccurred())
	// Must NOT have created any Job or set Phase.
	g.Expect(r.Status.ReplicationStep).To(Equal(""))
}

func TestSyncInitial_NotFoundCompactBackup_TimeoutExpired_FailsRestore(t *testing.T) {
	g := NewGomegaWithT(t)
	h, fixtures := newTestReplicationHandler(t)

	r := newReplicationRestoreFixture("r1", "ns", "nonexistent-cb", ptrDuration(time.Nanosecond))
	r.CreationTimestamp = metav1.NewTime(time.Now().Add(-time.Hour))
	fixtures.restoreClient.Create(context.TODO(), r, metav1.CreateOptions{})

	err := h.Sync(r)

	// IgnoreError wraps a Failed status write.
	g.Expect(err).NotTo(HaveOccurred()) // handler returns nil after writing Failed
	updated, _ := fixtures.restoreClient.Get(context.TODO(), "r1", metav1.GetOptions{})
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreFailed))
	// ... (reason: "CompactBackupWaitTimeout")
}

func TestSyncInitial_FoundAndConsistent_CreatesPhase1JobAndSetsPhase(t *testing.T) {
	g := NewGomegaWithT(t)
	h, fixtures := newTestReplicationHandler(t)

	cb := newCompactBackupFixture("cb1", "ns", v1alpha1.BackupRunning)
	fixtures.compactIndexer.Add(cb)

	r := newReplicationRestoreFixture("r1", "ns", "cb1", nil)
	err := h.Sync(r)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(r.Status.ReplicationStep).To(Equal("snapshot-restore"))
	g.Expect(r.Status.Phase).To(Equal(v1alpha1.RestoreSnapshotRestore))
	g.Expect(fixtures.createdJobs).To(HaveLen(1))
	g.Expect(fixtures.createdJobs[0].Labels[label.ReplicationStepLabelKey]).
		To(Equal(label.ReplicationStepSnapshotRestoreVal))
}

func TestSyncInitial_Inconsistent_FailsRestore(t *testing.T) {
	g := NewGomegaWithT(t)
	h, fixtures := newTestReplicationHandler(t)

	cb := newCompactBackupFixture("cb1", "ns", v1alpha1.BackupRunning)
	cb.Spec.BR.Cluster = "wrong-cluster" // mismatch
	fixtures.compactIndexer.Add(cb)

	r := newReplicationRestoreFixture("r1", "ns", "cb1", nil)
	_ = h.Sync(r)

	updated, _ := fixtures.restoreClient.Get(context.TODO(), "r1", metav1.GetOptions{})
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreFailed))
	g.Expect(len(fixtures.createdJobs)).To(Equal(0))
}
```

Add the fixture helpers at the top of the test file (below imports):

```go
type handlerFixtures struct {
	restoreClient   pingcapclient.RestoreInterface
	compactIndexer  cache.Indexer
	jobClient       batchv1client.JobInterface
	createdJobs     []*batchv1.Job
}

// newTestReplicationHandler returns a handler wired with fake clients and
// per-test fixtures. Implementation should use fake.NewSimpleClientset and
// cache.NewIndexer; lister adapters wrap the indexer. createdJobs captures
// Jobs from a reactor on jobClient.
func newTestReplicationHandler(t *testing.T) (*replicationHandler, *handlerFixtures) {
	// TODO on first use: wire up fake Dependencies.
	// Keep pattern consistent with existing pkg/backup/restore/restore_manager_test.go helpers.
	t.Helper()
	// ...
	return nil, nil
}

func newReplicationRestoreFixture(name, ns, cbName string, wait *metav1.Duration) *v1alpha1.Restore { /* ... */ }
func newCompactBackupFixture(name, ns string, state v1alpha1.BackupConditionType) *v1alpha1.CompactBackup { /* ... */ }
func ptrDuration(d time.Duration) *metav1.Duration { return &metav1.Duration{Duration: d} }
```

> **Note:** `newTestReplicationHandler` and the fixture builders are test infrastructure — implement them by copying the pattern from `pkg/backup/restore/restore_manager_test.go` (look for how `controller.Dependencies` is faked there) rather than inventing a new harness.

- [ ] **Step 2: Run tests — expect failures**

```bash
go test -run TestSyncInitial -v ./pkg/backup/restore/...
```

Expected: FAIL — tests depend on unimplemented `syncInitial`.

- [ ] **Step 3: Implement `syncInitial`**

Replace the stub `syncInitial` with:

```go
import (
	batchv1 "k8s.io/api/batch/v1"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
)

func (h *replicationHandler) syncInitial(restore *v1alpha1.Restore) error {
	cb, err := h.lookupCompactBackup(restore)
	if err != nil {
		if controller.IsIgnoreError(err) {
			return h.failRestore(restore, "CompactBackupWaitTimeout", err.Error())
		}
		return err
	}
	if cb != nil {
		// Consistency check before we commit to the phase-1 Job.
		if checkErr := checkCrossCRConsistency(restore, cb); checkErr != nil {
			return h.failRestore(restore, "CompactBackupMismatch", checkErr.Error())
		}
		// If CompactBackup is already terminal, write CompactSettled eagerly so
		// the snapshot-restore case can gate immediately when phase-1 Job
		// completes.
		if compactIsTerminal(cb) {
			reason := compactTerminalReason(cb)
			if markerErr := h.updateRestoreMarker(restore, v1alpha1.RestoreCompactSettled, reason, cb.Status.Message); markerErr != nil {
				return markerErr
			}
		}
	}

	// Enter phase-1: advance Phase and ReplicationStep, then create the Job.
	now := metav1.Now()
	if err := h.statusUpdater.Update(
		restore,
		&v1alpha1.RestoreCondition{
			Type:   v1alpha1.RestoreSnapshotRestore,
			Status: corev1.ConditionTrue,
			Reason: "CreatedPhase1Job",
		},
		&controller.RestoreUpdateStatus{TimeStarted: &now},
	); err != nil {
		return err
	}
	if err := h.setReplicationStep(restore, "snapshot-restore"); err != nil {
		return err
	}

	return h.ensureJobForStep(restore, label.ReplicationStepSnapshotRestoreVal)
}

// setReplicationStep is a tiny status sub-resource write — it uses client.Update
// because RestoreUpdateStatus (used by statusUpdater) does not carry ReplicationStep.
// ReplicationStep is controller-owned; backup-manager never writes it.
func (h *replicationHandler) setReplicationStep(restore *v1alpha1.Restore, step string) error {
	ns, name := restore.Namespace, restore.Name
	latest, err := h.deps.RestoreLister.Restores(ns).Get(name)
	if err != nil {
		return err
	}
	if latest.Status.ReplicationStep == step {
		return nil
	}
	updated := latest.DeepCopy()
	updated.Status.ReplicationStep = step
	_, err = h.deps.Clientset.PingcapV1alpha1().Restores(ns).Update(
		context.TODO(), updated, metav1.UpdateOptions{},
	)
	return err
}

// failRestore writes Phase=Failed with the given reason/message and returns nil.
// Callers use this for terminal orchestration failures (waitTimeout, mismatch,
// phase-1 Job Failed) — the condition is meaningful, not an "error".
func (h *replicationHandler) failRestore(restore *v1alpha1.Restore, reason, message string) error {
	return h.statusUpdater.Update(
		restore,
		&v1alpha1.RestoreCondition{
			Type:    v1alpha1.RestoreFailed,
			Status:  corev1.ConditionTrue,
			Reason:  reason,
			Message: message,
		},
		nil,
	)
}

// ensureJobForStep creates the BR Job for a given phase if not already present.
// Uses label selector (NOT name pattern) for the idempotence check.
func (h *replicationHandler) ensureJobForStep(restore *v1alpha1.Restore, step string) error {
	// Build selector: tidb.pingcap.com/restore=<name>,tidb.pingcap.com/replication-step=<step>
	sel := fmt.Sprintf("%s=%s,%s=%s",
		label.RestoreLabelKey, restore.Name,
		label.ReplicationStepLabelKey, step,
	)
	jobs, err := h.listJobsBySelector(restore.Namespace, sel)
	if err != nil {
		return err
	}
	if len(jobs) > 0 {
		return nil // already created on a prior reconcile
	}
	job, err := h.makeReplicationBRJob(restore, step)
	if err != nil {
		return err
	}
	return h.deps.JobControl.CreateJob(restore, job)
}

func (h *replicationHandler) listJobsBySelector(ns, selector string) ([]*batchv1.Job, error) {
	// Implement using h.deps.JobLister.Jobs(ns).List(labels.Parse(selector))
	// (Standard pattern in pkg/controller; see compactbackup controller.)
	return nil, nil // placeholder — replaced by concrete impl below
}

// makeReplicationBRJob is the phase-dependent Job builder.
// Stub: filled out in Task 8 alongside the spec's BR-args section. For the
// initial Task 6 pass we only need the Job skeleton with the correct labels
// so that the ensureJobForStep idempotence check works.
func (h *replicationHandler) makeReplicationBRJob(restore *v1alpha1.Restore, step string) (*batchv1.Job, error) {
	// Implementation in Task 8.
	return nil, fmt.Errorf("makeReplicationBRJob not yet implemented for step %q", step)
}

// compactIsTerminal returns true if CompactBackup has reached Complete or Failed.
func compactIsTerminal(cb *v1alpha1.CompactBackup) bool {
	switch v1alpha1.BackupConditionType(cb.Status.State) {
	case v1alpha1.BackupComplete, v1alpha1.BackupFailed:
		return true
	}
	return false
}

func compactTerminalReason(cb *v1alpha1.CompactBackup) string {
	if cb.Status.State == string(v1alpha1.BackupComplete) {
		return "AllShardsComplete"
	}
	return "ShardsPartialFailed"
}
```

> **Note:** `makeReplicationBRJob` is stubbed; `syncInitial`'s test expects a Job to be created, so implement a minimal version here that sets labels and an empty Pod template. The full BR-args assembly is Task 8 below.

Replace the stub with a minimal labels-only skeleton:

```go
func (h *replicationHandler) makeReplicationBRJob(restore *v1alpha1.Restore, step string) (*batchv1.Job, error) {
	jobName := fmt.Sprintf("%s-%s", restore.Name, step)
	labels := map[string]string{
		label.NameLabelKey:              "restore",
		label.ComponentLabelKey:         "restore",
		label.ManagedByLabelKey:         label.TiDBOperator,
		label.RestoreLabelKey:           restore.Name,
		label.ReplicationStepLabelKey:   step,
	}
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:            jobName,
			Namespace:       restore.Namespace,
			Labels:          labels,
			OwnerReferences: []metav1.OwnerReference{controller.GetRestoreOwnerRef(restore)},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					// phase-specific args filled in Task 8
					Containers:    []corev1.Container{{Name: "br", Image: restore.Spec.ToolImage}},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}, nil
}
```

- [ ] **Step 4: Run tests — expect pass**

```bash
go test -run TestSyncInitial -v ./pkg/backup/restore/...
```

Expected: PASS on all 4 sync-initial tests.

- [ ] **Step 5: Commit**

```bash
git add pkg/backup/restore/replication_handler.go \
        pkg/backup/restore/replication_handler_test.go
git commit -m "feat(replication-restore): implement handler initial case

On empty ReplicationStep, the handler validates cross-CR consistency,
writes CompactSettled if CompactBackup is already terminal, advances
Phase to SnapshotRestore, sets ReplicationStep=snapshot-restore, and
creates the phase-1 BR Job with replication-step label. Late-binding
and waitTimeout are handled via lookupCompactBackup."
```

---

## Task 7: Handler `snapshot-restore` case (observation + gate)

**Files:**
- Modify: `pkg/backup/restore/replication_handler.go`
- Modify: `pkg/backup/restore/replication_handler_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestSyncSnapshotRestore_Phase1JobComplete_WritesMarker(t *testing.T) {
	g := NewGomegaWithT(t)
	h, fixtures := newTestReplicationHandler(t)
	r := newReplicationRestoreInStep("r1", "ns", "cb1", "snapshot-restore")
	job := newReplicationJob("r1", "ns", label.ReplicationStepSnapshotRestoreVal, batchv1.JobComplete)
	fixtures.jobIndexer.Add(job)

	_ = h.Sync(r)

	updated, _ := fixtures.restoreClient.Get(context.TODO(), "r1", metav1.GetOptions{})
	_, c := v1alpha1.GetRestoreCondition(&updated.Status, v1alpha1.RestoreSnapshotRestored)
	g.Expect(c).NotTo(BeNil())
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreSnapshotRestore),
		"marker write must not change Phase")
}

func TestSyncSnapshotRestore_Phase1JobFailed_FailsRestore(t *testing.T) {
	g := NewGomegaWithT(t)
	h, fixtures := newTestReplicationHandler(t)
	r := newReplicationRestoreInStep("r1", "ns", "cb1", "snapshot-restore")
	job := newReplicationJob("r1", "ns", label.ReplicationStepSnapshotRestoreVal, batchv1.JobFailed)
	fixtures.jobIndexer.Add(job)

	_ = h.Sync(r)

	updated, _ := fixtures.restoreClient.Get(context.TODO(), "r1", metav1.GetOptions{})
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreFailed))
}

func TestSyncSnapshotRestore_BothMarkersTrue_TransitionsToLogRestore(t *testing.T) {
	g := NewGomegaWithT(t)
	h, fixtures := newTestReplicationHandler(t)

	r := newReplicationRestoreInStep("r1", "ns", "cb1", "snapshot-restore")
	appendRestoreMarker(r, v1alpha1.RestoreSnapshotRestored, "JobComplete", "")
	appendRestoreMarker(r, v1alpha1.RestoreCompactSettled, "AllShardsComplete", "")
	fixtures.restoreIndexer.Add(r)

	_ = h.Sync(r)

	updated, _ := fixtures.restoreClient.Get(context.TODO(), "r1", metav1.GetOptions{})
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreLogRestore))
	g.Expect(updated.Status.ReplicationStep).To(Equal("log-restore"))
	g.Expect(fixtures.createdJobs).To(HaveLen(1))
	g.Expect(fixtures.createdJobs[0].Labels[label.ReplicationStepLabelKey]).
		To(Equal(label.ReplicationStepLogRestoreVal))
}

func TestSyncSnapshotRestore_OnlyOneMarkerTrue_Waits(t *testing.T) {
	g := NewGomegaWithT(t)
	h, fixtures := newTestReplicationHandler(t)

	r := newReplicationRestoreInStep("r1", "ns", "cb1", "snapshot-restore")
	appendRestoreMarker(r, v1alpha1.RestoreSnapshotRestored, "JobComplete", "")
	// CompactSettled NOT set

	_ = h.Sync(r)

	g.Expect(fixtures.createdJobs).To(BeEmpty(), "phase-2 Job must not be created yet")
}
```

- [ ] **Step 2: Run — expect fail**

```bash
go test -run TestSyncSnapshotRestore -v ./pkg/backup/restore/...
```

- [ ] **Step 3: Implement `syncSnapshotRestore`**

```go
func (h *replicationHandler) syncSnapshotRestore(restore *v1alpha1.Restore) error {
	// 1. Observe phase-1 Job; update SnapshotRestored marker or fail the Restore.
	phase1Job, err := h.findJobForStep(restore, label.ReplicationStepSnapshotRestoreVal)
	if err != nil {
		return err
	}
	if phase1Job != nil {
		if jobHasCondition(phase1Job, batchv1.JobFailed) {
			return h.failRestore(restore, "SnapshotRestoreFailed",
				jobFailureMessage(phase1Job))
		}
		if jobHasCondition(phase1Job, batchv1.JobComplete) {
			if _, existing := v1alpha1.GetRestoreCondition(&restore.Status, v1alpha1.RestoreSnapshotRestored); existing == nil {
				if err := h.updateRestoreMarker(restore, v1alpha1.RestoreSnapshotRestored,
					"JobComplete", ""); err != nil {
					return err
				}
			}
		}
	}

	// 2. Observe CompactBackup; write CompactSettled if terminal.
	cb, err := h.lookupCompactBackup(restore)
	if err != nil {
		if controller.IsIgnoreError(err) {
			return h.failRestore(restore, "CompactBackupWaitTimeout", err.Error())
		}
		return err
	}
	if cb != nil && compactIsTerminal(cb) {
		if _, existing := v1alpha1.GetRestoreCondition(&restore.Status, v1alpha1.RestoreCompactSettled); existing == nil {
			if err := h.updateRestoreMarker(restore, v1alpha1.RestoreCompactSettled,
				compactTerminalReason(cb), cb.Status.Message); err != nil {
				return err
			}
		}
	}

	// 3. Gate: both markers True → transition to log-restore.
	latest, err := h.deps.RestoreLister.Restores(restore.Namespace).Get(restore.Name)
	if err != nil {
		return err
	}
	if !hasMarkerTrue(latest, v1alpha1.RestoreSnapshotRestored) ||
		!hasMarkerTrue(latest, v1alpha1.RestoreCompactSettled) {
		return nil // keep waiting
	}

	// Transition: Phase=LogRestore, Step=log-restore, create phase-2 Job.
	if err := h.statusUpdater.Update(
		latest,
		&v1alpha1.RestoreCondition{
			Type:   v1alpha1.RestoreLogRestore,
			Status: corev1.ConditionTrue,
			Reason: "GatePassed",
		},
		nil,
	); err != nil {
		return err
	}
	if err := h.setReplicationStep(latest, "log-restore"); err != nil {
		return err
	}
	return h.ensureJobForStep(latest, label.ReplicationStepLogRestoreVal)
}

func hasMarkerTrue(restore *v1alpha1.Restore, markerType v1alpha1.RestoreConditionType) bool {
	_, c := v1alpha1.GetRestoreCondition(&restore.Status, markerType)
	return c != nil && c.Status == corev1.ConditionTrue
}

func (h *replicationHandler) findJobForStep(restore *v1alpha1.Restore, step string) (*batchv1.Job, error) {
	sel := fmt.Sprintf("%s=%s,%s=%s",
		label.RestoreLabelKey, restore.Name,
		label.ReplicationStepLabelKey, step,
	)
	jobs, err := h.listJobsBySelector(restore.Namespace, sel)
	if err != nil || len(jobs) == 0 {
		return nil, err
	}
	return jobs[0], nil
}

func jobHasCondition(job *batchv1.Job, cond batchv1.JobConditionType) bool {
	for _, c := range job.Status.Conditions {
		if c.Type == cond && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func jobFailureMessage(job *batchv1.Job) string {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobFailed {
			return c.Message
		}
	}
	return "phase-1 Job failed"
}
```

- [ ] **Step 4: Run — expect pass**

```bash
go test -run TestSyncSnapshotRestore -v ./pkg/backup/restore/...
```

Expected: PASS on all 4 tests.

- [ ] **Step 5: Commit**

```bash
git add pkg/backup/restore/replication_handler.go \
        pkg/backup/restore/replication_handler_test.go
git commit -m "feat(replication-restore): implement snapshot-restore case

Handler observes phase-1 Job (writes SnapshotRestored marker or fails
the Restore on JobFailed) and CompactBackup (writes CompactSettled on
terminal state). When both markers are True, transitions Phase to
LogRestore, sets ReplicationStep=log-restore, and creates the phase-2
BR Job."
```

---

## Task 8: Handler `log-restore` case + phase-specific Job args

**Files:**
- Modify: `pkg/backup/restore/replication_handler.go`
- Modify: `pkg/backup/restore/replication_handler_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestSyncLogRestore_Phase2JobFailed_FailsRestore(t *testing.T) {
	g := NewGomegaWithT(t)
	h, fixtures := newTestReplicationHandler(t)
	r := newReplicationRestoreInStep("r1", "ns", "cb1", "log-restore")
	r.Status.Phase = v1alpha1.RestoreLogRestore
	job := newReplicationJob("r1", "ns", label.ReplicationStepLogRestoreVal, batchv1.JobFailed)
	fixtures.jobIndexer.Add(job)

	_ = h.Sync(r)

	updated, _ := fixtures.restoreClient.Get(context.TODO(), "r1", metav1.GetOptions{})
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreFailed))
}

func TestSyncLogRestore_Phase2JobRunning_IsNoOp(t *testing.T) {
	g := NewGomegaWithT(t)
	h, fixtures := newTestReplicationHandler(t)
	r := newReplicationRestoreInStep("r1", "ns", "cb1", "log-restore")
	// Job present with no terminal condition → backup-manager owns Phase writes.
	job := newReplicationJobNoCondition("r1", "ns", label.ReplicationStepLogRestoreVal)
	fixtures.jobIndexer.Add(job)

	err := h.Sync(r)

	g.Expect(err).NotTo(HaveOccurred())
	// No handler-written status change; backup-manager will write Running.
}

func TestMakeReplicationBRJob_SnapshotRestore_HasCorrectLabels(t *testing.T) {
	g := NewGomegaWithT(t)
	h, _ := newTestReplicationHandler(t)
	r := newReplicationRestoreFixture("r1", "ns", "cb1", nil)

	job, err := h.makeReplicationBRJob(r, label.ReplicationStepSnapshotRestoreVal)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(job.Name).To(Equal("r1-snapshot-restore"))
	g.Expect(job.Labels[label.ReplicationStepLabelKey]).
		To(Equal(label.ReplicationStepSnapshotRestoreVal))
	// Pod args must contain --replicationPhase=1 (consumed by PR 2).
	g.Expect(job.Spec.Template.Spec.Containers[0].Args).
		To(ContainElement("--replicationPhase=1"))
}
```

- [ ] **Step 2: Run — expect fail**

```bash
go test -run 'TestSyncLogRestore|TestMakeReplicationBRJob' -v ./pkg/backup/restore/...
```

- [ ] **Step 3: Implement `syncLogRestore` and finalize `makeReplicationBRJob`**

Replace `syncLogRestore`:

```go
func (h *replicationHandler) syncLogRestore(restore *v1alpha1.Restore) error {
	// During log-restore, backup-manager drives status.phase through
	// Running / Complete / Failed via the real updater (it is NOT wrapped in
	// phase-2). The handler only guards against Job failures that manage to
	// terminate before backup-manager has written a terminal condition —
	// treat that as our fallback failure path.
	job, err := h.findJobForStep(restore, label.ReplicationStepLogRestoreVal)
	if err != nil {
		return err
	}
	if job != nil && jobHasCondition(job, batchv1.JobFailed) &&
		!v1alpha1.IsRestoreFailed(restore) {
		return h.failRestore(restore, "LogRestoreFailed", jobFailureMessage(job))
	}
	return nil
}
```

Finalize `makeReplicationBRJob` to include `--replicationPhase` arg. Build on the Task-6 skeleton by adding the args array derived from the existing `restore_manager.makeRestoreJobWithMode` (follow the same pattern for BR base args). The replication-specific addition is a single arg:

```go
func (h *replicationHandler) makeReplicationBRJob(restore *v1alpha1.Restore, step string) (*batchv1.Job, error) {
	// Reuse the existing BR Job builder for PiTR mode to get all common args
	// (cluster, namespace, pitrFullBackupStorageProvider, pitrRestoredTs,
	// S3 creds, etc.), then append the replication-phase arg.
	//
	// For the initial implementation, mirror the skeleton from Task 6 but
	// derive base args. Keep this function focused on the replication-specific
	// additions; common args should come from a shared helper.
	phaseArg := "--replicationPhase=1"
	if step == label.ReplicationStepLogRestoreVal {
		phaseArg = "--replicationPhase=2"
	}

	jobName := fmt.Sprintf("%s-%s", restore.Name, step)
	labels := map[string]string{
		label.NameLabelKey:            "restore",
		label.ComponentLabelKey:       "restore",
		label.ManagedByLabelKey:       label.TiDBOperator,
		label.RestoreLabelKey:         restore.Name,
		label.ReplicationStepLabelKey: step,
	}

	// baseArgs mirrors restore_manager.makeRestoreJobWithMode for pitr mode.
	// Extract into a shared helper if it proves duplicative; for this first
	// implementation inline the minimal set consumed by backup-manager.
	baseArgs := buildPiTRBaseArgs(restore)
	args := append(baseArgs, phaseArg)

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:            jobName,
			Namespace:       restore.Namespace,
			Labels:          labels,
			OwnerReferences: []metav1.OwnerReference{controller.GetRestoreOwnerRef(restore)},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					ServiceAccountName: restore.Spec.ServiceAccount,
					Containers: []corev1.Container{{
						Name:  "br",
						Image: restore.Spec.ToolImage,
						Args:  args,
					}},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}, nil
}

// buildPiTRBaseArgs returns the backup-manager CLI args common to both
// replication phases (mirroring the standard PiTR args assembly in
// restore_manager.go). The caller appends --replicationPhase on top.
func buildPiTRBaseArgs(restore *v1alpha1.Restore) []string {
	// Implementer: copy the relevant arg assembly from
	// pkg/backup/restore/restore_manager.go makeRestoreJobWithMode for
	// mode=pitr. At minimum: --namespace, --restoreName, --tikvVersion,
	// --mode=pitr. Keep in sync with existing PiTR arg set to avoid drift.
	return []string{
		"restore",
		fmt.Sprintf("--namespace=%s", restore.Namespace),
		fmt.Sprintf("--restoreName=%s", restore.Name),
		"--mode=pitr",
	}
}
```

> **Note:** `buildPiTRBaseArgs` is intentionally minimal in the plan. Reviewers/implementers should inspect `makeRestoreJobWithMode` in `pkg/backup/restore/restore_manager.go` and extract the PiTR arg set there rather than re-deriving it. If the arg set is large, consider making `buildPiTRBaseArgs` live next to `makeRestoreJobWithMode` so both call sites evolve together.

- [ ] **Step 4: Run — expect pass**

```bash
go test -run 'TestSyncLogRestore|TestMakeReplicationBRJob' -v ./pkg/backup/restore/...
```

- [ ] **Step 5: Commit**

```bash
git add pkg/backup/restore/replication_handler.go \
        pkg/backup/restore/replication_handler_test.go
git commit -m "feat(replication-restore): implement log-restore case and Job builder

In log-restore, backup-manager drives status.phase via the real
updater (no wrap in phase-2). Handler only guards against Job-level
failures that terminate before backup-manager writes a terminal
condition. makeReplicationBRJob builds phase-specific BR Jobs with
the correct labels and passes --replicationPhase={1,2} to
backup-manager (consumed in PR 2)."
```

---

## Task 9: Intercept in `restoreManager.syncRestoreJob`

**Files:**
- Modify: `pkg/backup/restore/restore_manager.go`
- Modify: `pkg/backup/restore/restore_manager_test.go`

- [ ] **Step 1: Write failing test for routing**

Append to `restore_manager_test.go`:

```go
func TestRestoreManager_RoutesReplicationToHandler(t *testing.T) {
	g := NewGomegaWithT(t)
	deps, statusUpdater := newFakeDeps(t) // existing helper
	rm := &restoreManager{
		deps:               deps,
		statusUpdater:      statusUpdater,
		replicationHandler: &fakeReplicationHandler{},
	}

	r := newReplicationRestoreFixture("r1", "ns", "cb1", nil)
	// Not failed / not complete / not prune — routes into syncRestoreJob.
	err := rm.Sync(r)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(rm.replicationHandler.(*fakeReplicationHandler).syncCalls).To(Equal(1))
}

func TestRestoreManager_StandardPiTRDoesNotRouteToHandler(t *testing.T) {
	g := NewGomegaWithT(t)
	deps, statusUpdater := newFakeDeps(t)
	rm := &restoreManager{
		deps:               deps,
		statusUpdater:      statusUpdater,
		replicationHandler: &fakeReplicationHandler{},
	}

	r := newReplicationRestoreFixture("r1", "ns", "cb1", nil)
	r.Spec.ReplicationConfig = nil // standard PiTR

	_ = rm.Sync(r)

	g.Expect(rm.replicationHandler.(*fakeReplicationHandler).syncCalls).To(Equal(0))
}

type fakeReplicationHandler struct{ syncCalls int }
func (f *fakeReplicationHandler) Sync(_ *v1alpha1.Restore) error {
	f.syncCalls++
	return nil
}
```

- [ ] **Step 2: Run — expect fail**

```bash
go test -run TestRestoreManager_Routes -v ./pkg/backup/restore/...
```

- [ ] **Step 3: Add handler field, constructor wire, and interception**

Modify `pkg/backup/restore/restore_manager.go`:

First, add an interface for testability above the struct:

```go
// replicationHandlerInterface is the minimal surface restoreManager calls on
// the replication handler. Allows injecting a fake in tests.
type replicationHandlerInterface interface {
	Sync(restore *v1alpha1.Restore) error
}
```

Change the struct:

```go
type restoreManager struct {
	deps               *controller.Dependencies
	statusUpdater      controller.RestoreConditionUpdaterInterface
	replicationHandler replicationHandlerInterface
}
```

Change the constructor:

```go
func NewRestoreManager(deps *controller.Dependencies) backup.RestoreManager {
	statusUpdater := controller.NewRealRestoreConditionUpdater(deps.Clientset, deps.RestoreLister, deps.Recorder)
	return &restoreManager{
		deps:               deps,
		statusUpdater:      statusUpdater,
		replicationHandler: newReplicationHandler(deps, statusUpdater),
	}
}
```

Add the interception inside `syncRestoreJob`. Locate the existing check `if v1alpha1.IsRestoreFailed(restore) { return nil }` (~line 283). Insert immediately after it:

```go
	if v1alpha1.IsRestoreFailed(restore) {
		return nil
	}

	// Replication restore: delegate to dedicated handler that owns the
	// two-Job state machine and coordinates with CompactBackup. All logic
	// below this line assumes a single-Job Restore (GetRestoreJobName etc.)
	// which does not apply to replication restore.
	if restore.Spec.Mode == v1alpha1.RestoreModePiTR && restore.Spec.ReplicationConfig != nil {
		return rm.replicationHandler.Sync(restore)
	}
```

- [ ] **Step 4: Run — expect pass**

```bash
go test -run 'TestRestoreManager_Routes|TestRestoreManager_StandardPiTR' -v ./pkg/backup/restore/...
go test ./pkg/backup/restore/... 2>&1 | tail -5
```

Expected: PASS on new tests; no regression in existing restore_manager tests.

- [ ] **Step 5: Commit**

```bash
git add pkg/backup/restore/restore_manager.go \
        pkg/backup/restore/restore_manager_test.go
git commit -m "feat(replication-restore): intercept replication mode in restore manager

When Spec.Mode==pitr and Spec.ReplicationConfig!=nil, syncRestoreJob
delegates to replicationHandler.Sync. All legacy single-Job machinery
below the interception point is skipped; standard PiTR, snapshot, and
volume-snapshot paths are unchanged."
```

---

## Task 10: Controller CompactBackup informer wire-up

**Files:**
- Modify: `pkg/controller/restore/restore_controller.go`
- Modify: `pkg/controller/restore/restore_controller_test.go`

- [ ] **Step 1: Write failing test for enqueue**

```go
func TestController_CompactBackupUpdate_EnqueuesReferencingRestore(t *testing.T) {
	g := NewGomegaWithT(t)
	c, fakes := newFakeRestoreController(t) // existing test helper or extend

	restore := newReplicationRestoreFixture("r1", "ns", "cb1", nil)
	fakes.restoreIndexer.Add(restore)
	unrelated := newReplicationRestoreFixture("r2", "ns", "cb-other", nil)
	fakes.restoreIndexer.Add(unrelated)

	cb := newCompactBackupFixture("cb1", "ns", v1alpha1.BackupComplete)
	c.enqueueRestoresReferencing(cb)

	g.Expect(c.queue.Len()).To(Equal(1))
	key, _ := c.queue.Get()
	g.Expect(key).To(Equal("ns/r1"))
}
```

- [ ] **Step 2: Run — expect fail**

```bash
go test -run TestController_CompactBackupUpdate -v ./pkg/controller/restore/...
```

- [ ] **Step 3: Implement informer wire + enqueue**

In `pkg/controller/restore/restore_controller.go`, modify `NewController` to also register a CompactBackup event handler:

```go
func NewController(deps *controller.Dependencies) *Controller {
	c := &Controller{
		deps:    deps,
		control: NewDefaultRestoreControl(restore.NewRestoreManager(deps)),
		queue: workqueue.NewNamedRateLimitingQueue(
			controller.NewControllerRateLimiter(1*time.Second, 100*time.Second),
			"restore",
		),
	}

	restoreInformer := deps.InformerFactory.Pingcap().V1alpha1().Restores()
	restoreInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.updateRestore,
		UpdateFunc: func(old, cur interface{}) {
			c.updateRestore(cur)
		},
		DeleteFunc: c.enqueueRestore,
	})

	// Watch CompactBackup changes so that CompactBackup reaching terminal
	// state (or being newly created for a late-binding Restore) wakes the
	// right Restore reconciler.
	compactInformer := deps.InformerFactory.Pingcap().V1alpha1().CompactBackups()
	compactInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) { c.enqueueRestoresReferencing(obj) },
		UpdateFunc: func(_, cur interface{}) { c.enqueueRestoresReferencing(cur) },
		// Delete handled naturally: handler's next reconcile sees NotFound and
		// re-enters late-binding logic.
	})

	return c
}

// enqueueRestoresReferencing finds all Restores in the same namespace whose
// Spec.ReplicationConfig.CompactBackupName matches the given CompactBackup, and
// enqueues them. O(N) over in-namespace Restores; acceptable for expected
// fleet sizes (< ~100 Restores per namespace).
func (c *Controller) enqueueRestoresReferencing(obj interface{}) {
	cb, ok := obj.(*v1alpha1.CompactBackup)
	if !ok {
		return
	}
	restores, err := c.deps.RestoreLister.Restores(cb.Namespace).List(labels.Everything())
	if err != nil {
		utilruntime.HandleError(fmt.Errorf(
			"list restores in %s: %w", cb.Namespace, err))
		return
	}
	for _, r := range restores {
		if r.Spec.ReplicationConfig != nil &&
			r.Spec.ReplicationConfig.CompactBackupName == cb.Name {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(r)
			if err != nil {
				utilruntime.HandleError(err)
				continue
			}
			c.queue.Add(key)
		}
	}
}
```

Add imports: `"k8s.io/apimachinery/pkg/labels"`, `"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"`.

- [ ] **Step 4: Run — expect pass**

```bash
go test -run TestController_CompactBackupUpdate -v ./pkg/controller/restore/...
go test ./pkg/controller/restore/... 2>&1 | tail -5
```

Expected: PASS on new test; no regression.

- [ ] **Step 5: Verify whole tree builds and passes**

```bash
go build ./...
go test ./pkg/backup/restore/... ./pkg/controller/restore/... ./pkg/controller/... 2>&1 | tail -10
```

Expected: build succeeds; all relevant package tests pass.

- [ ] **Step 6: Commit**

```bash
git add pkg/controller/restore/restore_controller.go \
        pkg/controller/restore/restore_controller_test.go
git commit -m "feat(replication-restore): wake Restore on CompactBackup changes

Registers a CompactBackup EventHandler on the Restore controller that
enqueues Restores whose Spec.ReplicationConfig.CompactBackupName
matches the changed CompactBackup. Closes the observation loop so that
a late-binding Restore unblocks as soon as its referenced CompactBackup
is created, and so that a Restore in SnapshotRestore gates promptly
when CompactBackup reaches terminal state."
```

---

## Final verification

- [ ] **Full build + tests**

```bash
go build ./...
go vet ./...
go test ./pkg/... 2>&1 | tail -20
```

Expected: all packages build and pass.

- [ ] **Scan for placeholder code**

```bash
grep -nE "TODO|FIXME|XXX" \
    pkg/backup/restore/replication_handler.go \
    pkg/controller/restore_status_updater.go \
    pkg/controller/restore/restore_controller.go
```

Expected: only the intentional TODO references to PR 2 (if any). Anything else must be replaced with real code before PR opens.

- [ ] **PR body draft**

Prepare a PR description referencing:
- Spec: `docs/superpowers/specs/2026-04-23-replication-restore-design.md`
- Note that PR 2 (backup-manager CLI + BR flags) will follow and is gated on internal BR `--replication-storage-phase` readiness
- Behavior guarantee: existing PiTR / snapshot / volume-snapshot users unaffected (ReplicationConfig==nil falls through to original code path)

- [ ] **Push branch, open PR against master**

```bash
git push -u origin <feature-branch>
# Open PR via gh or GitHub UI; base = master
```

---

## Self-Review Checklist (run before handoff)

1. **Spec coverage**
   - Section 1 (CR shape): Task 1 ✓
   - Section 2 (state machine): Task 1 (constants) + Task 6/7/8 (transitions) ✓
   - Section 3 (write ownership Option B): Task 3 (wrapper) + Task 5 (appendRestoreMarker helper) ✓
   - Section 4.1 (interception point): Task 9 ✓
   - Section 4.2 (state machine): Task 6/7/8 ✓
   - Section 4.3 (label-based Job lookup): Task 2 + Task 6 (ensureJobForStep/findJobForStep) ✓
   - Section 4.4 (CompactBackup informer): Task 10 ✓
   - Section 5 (cross-CR binding): Task 5 (consistency + lookup) ✓
   - Section 6 (BR CLI): **out of scope** (PR 2)
   - Section 7 (module layout): all tasks map to stated files ✓
   - Section 8 (backward compat): Task 9 interception condition enforces this ✓
   - Section 10 (test plan): Tasks 3/5/6/7/8/9/10 cover the operator-side scenarios; integration tests requiring real BR are deferred to PR 2 (noted in spec) ✓

2. **Placeholder scan**
   - `buildPiTRBaseArgs` in Task 8 is a deliberate stub; the implementer is expected to extract the real arg set from `makeRestoreJobWithMode`. Flagged inline.
   - `newTestReplicationHandler` fixture helpers in Task 6 are stubbed; implementer instructed to copy from existing `restore_manager_test.go` patterns. Flagged inline.

3. **Type consistency**
   - `RestoreSnapshotRestore` / `RestoreLogRestore` / `RestoreSnapshotRestored` / `RestoreCompactSettled` used consistently across tasks.
   - `ReplicationStepLabelKey` / `ReplicationStepSnapshotRestoreVal` / `ReplicationStepLogRestoreVal` used consistently.
   - `replicationHandler` struct + `replicationHandlerInterface` — the interface is introduced in Task 9 (for testability), ok to do so after concrete type exists.
