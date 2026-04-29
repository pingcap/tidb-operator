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

	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/util"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

// replicationHandler owns the reconcile loop for replication restore
// (Spec.Mode == pitr && Spec.ReplicationConfig != nil).
//
// Sync uses a single-step cascade: each reconcile evaluates the current state
// (Phase, ReplicationStep, condition markers, observed Job/CompactBackup) and
// performs at most ONE K8s write before returning. The next state transition
// is driven by the next reconcile (triggered by the watch event of whatever
// was just written, or by Job / CompactBackup events). This matches the
// codebase-wide "write once, return, watch-driven next round" pattern (see
// restore_manager.go:syncRestoreJob) and avoids the crash windows that come
// with multi-write-per-reconcile designs.
//
// CompactBackup observation is decoupled from snapshot-phase progression: the
// top of Sync observes CompactBackup until CompactSettled is written (any
// reason — success, partial-fail, mismatch, wait-timeout), then short-circuits.
// Snapshot phase advances regardless of cb state below; the gate at B9 joins
// them. CB issues no longer fail the Restore — they settle as marker reasons
// that BR phase-2 reads to decide compact-vs-fallback.
//
// Branches in the cascade (evaluated in order; first match wins and returns):
//
//	B1     terminal short-circuit (Phase ∈ {Complete, Failed})
//	(top)  CompactBackup observation block — gated by !hasMarker(CompactSettled).
//	       Routes the four CB outcomes (Complete / Failed / Mismatch /
//	       WaitTimeout) to a CompactSettled marker write. NotFound-but-still-
//	       waiting emits a Warning Event and falls through so snapshot can
//	       proceed in parallel.
//	B3     enter SnapshotRestore phase: write Phase=SnapshotRestore.
//	       Independent of cb — snapshot starts even with CompactSettled unset.
//	B4     bump Step to "snapshot-restore"
//	B5–B9  active while Phase==SnapshotRestore && Step=="snapshot-restore":
//	       B5 create/rebuild phase-1 Job; B6 fail on Job Failed; B7 write
//	       SnapshotRestored marker on Job Complete; B9 gate passed → write
//	       Phase=LogRestore. (B8 — in-cascade CB-terminal observation — was
//	       moved up to the top block; numbers kept stable for traceability.)
//	       The Phase guard on this block matters: after B9 writes
//	       Phase=LogRestore the next reconcile briefly sees Step still
//	       "snapshot-restore" — without the guard B5 would re-fire and
//	       try to recreate a phase-1 Job. The guard hands control to B10.
//	B10    bump Step to "log-restore"
//	B11    create phase-2 Job
//	B12    phase-2 Job Failed → failRestore("LogRestoreFailed")
//	       Otherwise: backup-manager drives Phase=Running/Complete/Failed
type replicationHandler struct {
	deps          *controller.Dependencies
	statusUpdater controller.RestoreConditionUpdaterInterface
	recorder      record.EventRecorder
}

var _ replicationHandlerInterface = (*replicationHandler)(nil)

func newReplicationHandler(
	deps *controller.Dependencies,
	statusUpdater controller.RestoreConditionUpdaterInterface,
	recorder record.EventRecorder,
) *replicationHandler {
	return &replicationHandler{deps: deps, statusUpdater: statusUpdater, recorder: recorder}
}

// Sync is the entry point called from restoreManager.syncRestoreJob when
// the restore is in replication mode. See type doc above for the cascade
// shape and per-branch semantics.
func (h *replicationHandler) Sync(restore *v1alpha1.Restore) error {
	// B1: terminal short-circuit.
	if v1alpha1.IsRestoreComplete(restore) || v1alpha1.IsRestoreFailed(restore) {
		return nil
	}

	// Top block: observe CompactBackup until CompactSettled is written, then
	// short-circuit. Snapshot phase progression below is INDEPENDENT of cb —
	// the gate at B9 is the only place where the two paths join. CB issues
	// (mismatch, wait-timeout) are NOT terminal failures of the Restore;
	// they settle as marker reasons and BR phase-2 reads the reason to
	// decide compact-vs-fallback.
	//
	// Settling outcomes (each routed to a CompactSettled marker reason):
	//   "AllShardsComplete"        — CB Complete
	//   "ShardsPartialFailed"      — CB Failed
	//   "CompactBackupMismatch"    — cross-CR consistency check fails
	//   "CompactBackupWaitTimeout" — CB never appeared within WaitTimeout
	//
	// Note: consistency check runs every Sync that reaches this block (until
	// settled). The previous one-shot guard ("Phase==''") is gone — this is
	// intentional. Under the new design we don't fail the Restore on
	// mid-flight CB mutation; we settle with the mismatch reason. So the
	// re-check is harmless and catches user error sooner.
	if !hasMarkerTrue(restore, v1alpha1.RestoreCompactSettled) {
		cb, err := h.lookupCompactBackup(restore)
		if err != nil {
			if controller.IsIgnoreError(err) {
				return h.updateRestoreMarker(restore, v1alpha1.RestoreCompactSettled,
					"CompactBackupWaitTimeout", err.Error())
			}
			return err
		}
		if cb == nil {
			// NotFound but still waiting (WaitTimeout unset / 0 / not yet
			// expired). Emit a Warning Event for visibility, then fall through
			// so snapshot can proceed in parallel. K8s recorder dedups within
			// a window so retry storms don't flood.
			h.recorder.Eventf(restore, corev1.EventTypeWarning, "CompactBackupNotFound",
				"CompactBackup %q not found in namespace %s, waiting",
				restore.Spec.ReplicationConfig.CompactBackupName, restore.Namespace)
		} else {
			if checkErr := checkCrossCRConsistency(restore, cb); checkErr != nil {
				return h.updateRestoreMarker(restore, v1alpha1.RestoreCompactSettled,
					"CompactBackupMismatch", checkErr.Error())
			}
			if compactIsTerminal(cb) {
				return h.updateRestoreMarker(restore, v1alpha1.RestoreCompactSettled,
					compactTerminalReason(cb), cb.Status.Message)
			}
		}
	}

	// B3: enter SnapshotRestore phase. Independent of cb — snapshot starts
	// even when CompactSettled is not yet written. The gate at B9 enforces
	// the join.
	if restore.Status.Phase == "" {
		var statusUpdate *controller.RestoreUpdateStatus
		if restore.Status.TimeStarted.IsZero() {
			now := metav1.Now()
			statusUpdate = &controller.RestoreUpdateStatus{TimeStarted: &now}
		}
		return h.statusUpdater.Update(
			restore,
			&v1alpha1.RestoreCondition{
				Type:   v1alpha1.RestoreSnapshotRestore,
				Status: corev1.ConditionTrue,
				Reason: "SnapshotRestoreStarted",
			},
			statusUpdate,
		)
	}

	// B4: bump ReplicationStep once Phase has settled into SnapshotRestore.
	if restore.Status.Phase == v1alpha1.RestoreSnapshotRestore &&
		restore.Status.ReplicationStep == "" {
		return h.setReplicationStep(restore, label.ReplicationStepSnapshotRestoreVal)
	}

	// B5–B9: branches active while we're still in phase-1 (Phase==SnapshotRestore
	// AND Step=="snapshot-restore"). Constraining on Phase here matters: after
	// B9 writes Phase=LogRestore the next reconcile will see Step still
	// "snapshot-restore" briefly — without the Phase guard the cascade would
	// re-enter B5 and try to recreate the phase-1 Job. The Phase guard hands
	// control to B10 instead, which bumps Step to "log-restore".
	if restore.Status.Phase == v1alpha1.RestoreSnapshotRestore &&
		restore.Status.ReplicationStep == label.ReplicationStepSnapshotRestoreVal {
		job, jobErr := h.findJobForStep(restore, label.ReplicationStepSnapshotRestoreVal)
		if jobErr != nil {
			return jobErr
		}

		// B5: create phase-1 Job. Covers both first-time creation (right after
		// B4 wrote Step) and crash recovery (Step was persisted but the previous
		// reconcile's Job creation never landed).
		if job == nil {
			created, err := h.ensureJobForStep(restore, label.ReplicationStepSnapshotRestoreVal)
			if err != nil {
				return err
			}
			if created {
				h.recorder.Event(restore, corev1.EventTypeNormal, "SnapshotRestoreStarted",
					"Created phase-1 BR Job for replication restore")
			}
			return nil
		}

		// B6: phase-1 Job Failed → terminal failure.
		if jobHasCondition(job, batchv1.JobFailed) {
			return h.failRestore(restore, "SnapshotRestoreFailed", jobFailureMessage(job))
		}

		// B7: phase-1 Job Complete + marker not yet written.
		if jobHasCondition(job, batchv1.JobComplete) &&
			!hasMarkerTrue(restore, v1alpha1.RestoreSnapshotRestored) {
			return h.updateRestoreMarker(restore, v1alpha1.RestoreSnapshotRestored,
				"JobComplete", "")
		}

		// B8 was the in-cascade CB-terminal observation; moved to the top
		// block. Slot kept empty for traceability with prior versions.

		// B9: gate passed → write Phase=LogRestore.
		if hasMarkerTrue(restore, v1alpha1.RestoreSnapshotRestored) &&
			hasMarkerTrue(restore, v1alpha1.RestoreCompactSettled) {
			return h.statusUpdater.Update(
				restore,
				&v1alpha1.RestoreCondition{
					Type:   v1alpha1.RestoreLogRestore,
					Status: corev1.ConditionTrue,
					Reason: "GatePassed",
				},
				nil,
			)
		}

		// At least one marker still missing; nothing to do this reconcile.
		return nil
	}

	// B10: bump Step to "log-restore" once Phase has settled into LogRestore.
	if restore.Status.Phase == v1alpha1.RestoreLogRestore &&
		restore.Status.ReplicationStep == label.ReplicationStepSnapshotRestoreVal {
		return h.setReplicationStep(restore, label.ReplicationStepLogRestoreVal)
	}

	// B11–B12: branches active while ReplicationStep == "log-restore".
	if restore.Status.ReplicationStep == label.ReplicationStepLogRestoreVal {
		job, jobErr := h.findJobForStep(restore, label.ReplicationStepLogRestoreVal)
		if jobErr != nil {
			return jobErr
		}

		// B11: create phase-2 Job. Covers first-time creation + crash recovery.
		if job == nil {
			created, err := h.ensureJobForStep(restore, label.ReplicationStepLogRestoreVal)
			if err != nil {
				return err
			}
			if created {
				h.recorder.Event(restore, corev1.EventTypeNormal, "LogRestoreStarted",
					"Created phase-2 BR Job for replication restore")
			}
			return nil
		}

		// B12: phase-2 Job Failed → terminal failure.
		if jobHasCondition(job, batchv1.JobFailed) {
			return h.failRestore(restore, "LogRestoreFailed", jobFailureMessage(job))
		}

		// Otherwise: backup-manager drives Phase=Running/Complete/Failed.
		return nil
	}

	// Defensive: an unknown ReplicationStep value indicates a bug elsewhere.
	klog.Warningf("unknown ReplicationStep %q for restore %s/%s",
		restore.Status.ReplicationStep, restore.Namespace, restore.Name)
	return nil
}

// setReplicationStep persists Status.ReplicationStep. Reads via Clientset
// (not lister) to avoid lister-staleness clobbering a recently-written
// status.Phase: under single-step the Phase write happened in a previous
// reconcile, but the lister cache may still lag, and Update via Clientset
// rewrites the entire object. Reading via Clientset guarantees we see the
// latest Phase before bumping ReplicationStep. Wrapped in retry.OnError so
// concurrent reconciles (e.g. Job event triggering another reconcile) don't
// lose the Step write.
func (h *replicationHandler) setReplicationStep(restore *v1alpha1.Restore, step string) error {
	ns, name := restore.Namespace, restore.Name
	return retry.OnError(retry.DefaultRetry, func(e error) bool { return e != nil }, func() error {
		latest, err := h.deps.Clientset.PingcapV1alpha1().Restores(ns).Get(
			context.TODO(), name, metav1.GetOptions{},
		)
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
	})
}

// failRestore writes a Failed condition and returns nil on success, so callers
// can do "return h.failRestore(...)". Note: local restore.Status is NOT mutated
// here — statusUpdater re-reads via the lister and writes a separate object.
// All callers are currently return sites, so this is harmless; the comment
// exists to keep it that way if new callers are added.
//
// Always emits a Warning Event with the same reason/message before attempting
// the status write — so that even if the status update itself fails, operators
// get a kubectl-describe trail of why the restore was failed. K8s recorder
// dedups identical events within a window, so retry storms don't flood.
func (h *replicationHandler) failRestore(restore *v1alpha1.Restore, reason, message string) error {
	h.recorder.Event(restore, corev1.EventTypeWarning, reason, message)
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

// ensureJobForStep creates the BR Job for the given step if missing.
// Returns created=true only when a Job was actually created in this call,
// so callers can emit Events on the real-creation path without firing on
// the lister-stale "already exists" race.
func (h *replicationHandler) ensureJobForStep(restore *v1alpha1.Restore, step string) (created bool, err error) {
	jobs, err := h.listJobsBySelector(restore.Namespace, restore.Name, step)
	if err != nil {
		return false, err
	}
	if len(jobs) > 0 {
		return false, nil
	}
	job, err := h.makeReplicationBRJob(restore, step)
	if err != nil {
		return false, err
	}
	if err := h.deps.JobControl.CreateJob(restore, job); err != nil {
		return false, err
	}
	return true, nil
}

func (h *replicationHandler) listJobsBySelector(ns, restoreName, step string) ([]*batchv1.Job, error) {
	selector := labels.SelectorFromSet(labels.Set{
		label.RestoreLabelKey:         restoreName,
		label.ReplicationStepLabelKey: step,
	})
	return h.deps.JobLister.Jobs(ns).List(selector)
}

// makeReplicationBRJob builds the phase-specific BR Job with the correct
// replication-step label and the --replicationPhase arg consumed by
// backup-manager. Phase 1 (snapshot-restore) uses --replicationPhase=1;
// phase 2 (log-restore) uses --replicationPhase=2.
//
// NOTE: the Job produced here is functionally insufficient for backup-manager
// to run BR end-to-end; see buildPiTRBaseArgs for the full rationale.
func (h *replicationHandler) makeReplicationBRJob(restore *v1alpha1.Restore, step string) (*batchv1.Job, error) {
	phaseArg := "--replicationPhase=1"
	if step == label.ReplicationStepLogRestoreVal {
		phaseArg = "--replicationPhase=2"
	}

	jobName := fmt.Sprintf("%s-%s", restore.Name, step)
	// Use the canonical restore Job label set so replication Jobs are
	// indistinguishable from standard restore Jobs to external tooling
	// (kubectl label selectors, monitoring), then add the replication-step
	// label that distinguishes phase-1 from phase-2.
	jobLabels := util.CombineStringMap(
		label.NewRestore().
			Instance(restore.GetInstanceName()).
			RestoreJob().
			Restore(restore.Name),
		restore.Labels,
	)
	jobLabels[label.ReplicationStepLabelKey] = step

	args := append(buildPiTRBaseArgs(restore), phaseArg)

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:            jobName,
			Namespace:       restore.Namespace,
			Labels:          jobLabels,
			OwnerReferences: []metav1.OwnerReference{controller.GetRestoreOwnerRef(restore)},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: jobLabels},
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

// buildPiTRBaseArgs returns a MINIMAL set of backup-manager CLI args
// common to both replication phases. This is intentionally a stub for
// PR 1: the full PiTR arg set (env vars, TLS volume mounts, password
// secrets, resource requirements) lives in restoreManager.makeRestoreJobWithMode
// and is NOT replicated here.
//
// As a result, the Job produced by makeReplicationBRJob is functionally
// insufficient for backup-manager to actually run BR end-to-end. PR 2 (or
// a follow-up after Task 9 wires the handler to the restoreManager) must
// complete the integration by either:
//
//	(a) calling rm.makeRestoreJobWithMode to build the base Job and
//	    post-processing it (add replication-step label, append
//	    --replicationPhase arg), OR
//	(b) extracting a shared helper next to makeRestoreJobWithMode that
//	    both call sites consume.
//
// The unit tests in this PR verify only the replication-specific
// additions (label + --replicationPhase arg).
func buildPiTRBaseArgs(restore *v1alpha1.Restore) []string {
	return []string{
		"restore",
		fmt.Sprintf("--namespace=%s", restore.Namespace),
		fmt.Sprintf("--restoreName=%s", restore.Name),
		fmt.Sprintf("--mode=%s", v1alpha1.RestoreModePiTR),
	}
}

// compactIsTerminal returns true if CompactBackup has reached either
// BackupComplete or BackupFailed. The Failed sub-case is intentional — per
// the spec, the gate proceeds to phase-2 even when compact failed, because
// internal BR is contracted to fall back to uncompacted log. CompactBackup
// failure is observable via the CompactSettled marker's Reason field
// ("ShardsPartialFailed") and via CompactBackup.Status.{CompletedIndexes,
// FailedIndexes} — but does NOT block the Restore from progressing.
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

// hasMarkerTrue returns true iff the named condition exists and Status is ConditionTrue.
func hasMarkerTrue(restore *v1alpha1.Restore, markerType v1alpha1.RestoreConditionType) bool {
	_, c := v1alpha1.GetRestoreCondition(&restore.Status, markerType)
	return c != nil && c.Status == corev1.ConditionTrue
}

// findJobForStep returns the BR Job for the given step, or nil if not found.
// Reuses listJobsBySelector (3-arg form from T6).
func (h *replicationHandler) findJobForStep(restore *v1alpha1.Restore, step string) (*batchv1.Job, error) {
	jobs, err := h.listJobsBySelector(restore.Namespace, restore.Name, step)
	if err != nil || len(jobs) == 0 {
		return nil, err
	}
	if len(jobs) > 1 {
		klog.Warningf("Found %d Jobs for restore=%s/%s step=%s, expected 1; using %s",
			len(jobs), restore.Namespace, restore.Name, step, jobs[0].Name)
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
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			return c.Message
		}
	}
	return "phase-1 Job failed"
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
//     emits a Warning Event and continues (snapshot proceeds in parallel)
//   - (nil, controller.IgnoreErrorf) when WaitTimeout has expired — caller
//     writes a CompactSettled marker with reason CompactBackupWaitTimeout
//     (does NOT fail the Restore; BR phase-2 falls back to uncompacted log)
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
	if cfg.WaitTimeout == nil || cfg.WaitTimeout.Duration <= 0 {
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
	// to the "unsupported type" message.
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
		return fmt.Errorf("unsupported storage provider type for replication restore (only s3/gcs/azblob supported)")
	}
}
