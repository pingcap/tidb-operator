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
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

// replicationHandler owns the reconcile loop for replication restore
// (Spec.Mode == pitr && Spec.ReplicationConfig != nil).
//
// Sync evaluates the current state and performs at most ONE K8s write before
// returning. The next transition is driven by the next reconcile, triggered by
// the watch event of whatever was just written, or by Job / CompactBackup
// events. This matches the codebase-wide "write once, return, watch-driven
// next round" pattern (see restore_manager.go:syncRestoreJob) and avoids the
// crash windows that come with multi-write-per-reconcile designs.
//
// ReplicationStep is the primary dispatch key. Phase and Step are written
// together in a single Update (via RestoreUpdateStatus.ReplicationStep), so
// observers never see a (Phase, Step) pair that's only half-advanced:
//
//   - "" initializes: Phase="" → enterSnapshotRestore writes both
//     Phase=SnapshotRestore and Step="snapshot-restore" atomically.
//   - "snapshot-restore" tracks phase-1 progress with markers:
//     SnapshotRestored=True and CompactSettled=True. Once both markers are
//     present, enterLogRestore writes Phase=LogRestore and Step="log-restore"
//     atomically.
//   - "log-restore" manages the phase-2 Job. In this step, backup-manager owns
//     user-visible Phase transitions such as Running / Complete / Failed.
//
// CompactBackup observation is independent of the step dispatch. Until
// CompactSettled is written, each Sync tries to settle it with one of the
// marker reasons consumed by phase-2 BR: AllShardsComplete,
// ShardsPartialFailed, CompactBackupMismatch, or CompactBackupWaitTimeout.
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
// the restore is in replication mode. It dispatches by ReplicationStep, while
// keeping CompactBackup observation as a top-level parallel path joined by the
// snapshot marker gate.
func (h *replicationHandler) Sync(restore *v1alpha1.Restore) error {
	if v1alpha1.IsRestoreComplete(restore) || v1alpha1.IsRestoreFailed(restore) {
		return nil
	}

	handled, err := h.settleCompactBackupIfReady(restore)
	if handled || err != nil {
		return err
	}

	switch restore.Status.ReplicationStep {
	case "":
		return h.syncEmptyReplicationStep(restore)
	case label.ReplicationStepSnapshotRestoreVal:
		return h.syncSnapshotReplicationStep(restore)
	case label.ReplicationStepLogRestoreVal:
		return h.syncLogRestoreStep(restore)
	default:
		return h.handleUnexpectedReplicationState(restore)
	}
}

// settleCompactBackupIfReady observes CompactBackup until CompactSettled is
// written. It returns handled=true only when it performed the single write for
// this reconcile; NotFound-but-waiting emits an Event and lets snapshot advance.
func (h *replicationHandler) settleCompactBackupIfReady(restore *v1alpha1.Restore) (bool, error) {
	if hasMarkerTrue(restore, v1alpha1.RestoreCompactSettled) {
		return false, nil
	}

	cb, err := h.lookupCompactBackup(restore)
	if err != nil {
		if controller.IsIgnoreError(err) {
			return true, h.updateRestoreMarker(restore, v1alpha1.RestoreCompactSettled,
				"CompactBackupWaitTimeout", err.Error())
		}
		return false, err
	}
	if cb == nil {
		// NotFound but still waiting (WaitTimeout unset / 0 / not yet
		// expired). Emit a Warning Event for visibility, then continue to
		// step dispatch so snapshot can proceed in parallel.
		h.recorder.Eventf(restore, corev1.EventTypeWarning, "CompactBackupNotFound",
			"CompactBackup %q not found in namespace %s, waiting",
			restore.Spec.ReplicationConfig.CompactBackupName, restore.Namespace)
		return false, nil
	}

	if checkErr := checkCrossCRConsistency(restore, cb); checkErr != nil {
		return true, h.updateRestoreMarker(restore, v1alpha1.RestoreCompactSettled,
			"CompactBackupMismatch", checkErr.Error())
	}
	if compactIsTerminal(cb) {
		return true, h.updateRestoreMarker(restore, v1alpha1.RestoreCompactSettled,
			compactTerminalReason(cb), cb.Status.Message)
	}

	return false, nil
}

func (h *replicationHandler) enterSnapshotRestore(restore *v1alpha1.Restore) error {
	step := label.ReplicationStepSnapshotRestoreVal
	statusUpdate := &controller.RestoreUpdateStatus{ReplicationStep: &step}
	if restore.Status.TimeStarted.IsZero() {
		now := metav1.Now()
		statusUpdate.TimeStarted = &now
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

func (h *replicationHandler) syncEmptyReplicationStep(restore *v1alpha1.Restore) error {
	if restore.Status.Phase == "" {
		return h.enterSnapshotRestore(restore)
	}
	return h.handleUnexpectedReplicationState(restore)
}

func (h *replicationHandler) syncSnapshotReplicationStep(restore *v1alpha1.Restore) error {
	if restore.Status.Phase == v1alpha1.RestoreSnapshotRestore {
		return h.syncSnapshotJobAndGate(restore)
	}
	return h.handleUnexpectedReplicationState(restore)
}

func (h *replicationHandler) syncSnapshotJobAndGate(restore *v1alpha1.Restore) error {
	job, jobErr := h.findJobForStep(restore, label.ReplicationStepSnapshotRestoreVal)
	if jobErr != nil {
		return jobErr
	}

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

	if jobHasCondition(job, batchv1.JobFailed) {
		return h.failRestore(restore, "SnapshotRestoreFailed", jobFailureMessage(job))
	}

	if jobHasCondition(job, batchv1.JobComplete) &&
		!hasMarkerTrue(restore, v1alpha1.RestoreSnapshotRestored) {
		return h.updateRestoreMarker(restore, v1alpha1.RestoreSnapshotRestored,
			"JobComplete", "")
	}

	if hasMarkerTrue(restore, v1alpha1.RestoreSnapshotRestored) &&
		hasMarkerTrue(restore, v1alpha1.RestoreCompactSettled) {
		return h.enterLogRestore(restore)
	}

	return nil
}

func (h *replicationHandler) enterLogRestore(restore *v1alpha1.Restore) error {
	step := label.ReplicationStepLogRestoreVal
	return h.statusUpdater.Update(
		restore,
		&v1alpha1.RestoreCondition{
			Type:   v1alpha1.RestoreLogRestore,
			Status: corev1.ConditionTrue,
			Reason: "GatePassed",
		},
		&controller.RestoreUpdateStatus{ReplicationStep: &step},
	)
}

func (h *replicationHandler) syncLogRestoreStep(restore *v1alpha1.Restore) error {
	job, jobErr := h.findJobForStep(restore, label.ReplicationStepLogRestoreVal)
	if jobErr != nil {
		return jobErr
	}

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

	if jobHasCondition(job, batchv1.JobFailed) {
		return h.failRestore(restore, "LogRestoreFailed", jobFailureMessage(job))
	}

	return nil
}

func (h *replicationHandler) handleUnexpectedReplicationState(restore *v1alpha1.Restore) error {
	// Defensive: an unexpected Phase / ReplicationStep pairing indicates a bug
	// elsewhere. Do not create work or write status from here.
	klog.Warningf("unexpected replication restore state phase=%q step=%q for restore %s/%s",
		restore.Status.Phase, restore.Status.ReplicationStep, restore.Namespace, restore.Name)
	return nil
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
// the cache-stale "already exists" race.
//
// Lookup is Get-by-name against the lister, then RestoreUIDLabelKey is
// compared so a Job left behind by a deleted same-name predecessor (still
// being garbage-collected) is rejected with an explicit error rather than
// being treated as ours.
func (h *replicationHandler) ensureJobForStep(restore *v1alpha1.Restore, step string) (created bool, err error) {
	jobName := fmt.Sprintf("%s-%s", restore.Name, step)

	existing, getErr := h.deps.JobLister.Jobs(restore.Namespace).Get(jobName)
	if getErr != nil && !errors.IsNotFound(getErr) {
		return false, getErr
	}
	if existing != nil {
		if existing.Labels[label.RestoreUIDLabelKey] != string(restore.UID) {
			return false, fmt.Errorf(
				"job %s/%s belongs to a prior Restore (uid=%q, current=%q); awaiting GC",
				restore.Namespace, jobName,
				existing.Labels[label.RestoreUIDLabelKey], restore.UID,
			)
		}
		return false, nil
	}

	job, err := h.makeReplicationBRJob(restore, step)
	if err != nil {
		return false, err
	}
	// Residual race: lister said NotFound but a parallel Sync's Create has
	// already landed in etcd. Surface as a normal error — the next Sync's
	// Get-first path will identify the Job as ours via the UID label.
	if err := h.deps.JobControl.CreateJob(restore, job); err != nil {
		return false, err
	}
	return true, nil
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
	jobLabels[label.RestoreUIDLabelKey] = string(restore.UID)

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
// Lookup is Get-by-name; a Job whose RestoreUIDLabelKey doesn't match the
// current Restore's UID is treated as not found, so the handler never reads
// the status of a stale Job left behind by a deleted same-name predecessor.
func (h *replicationHandler) findJobForStep(restore *v1alpha1.Restore, step string) (*batchv1.Job, error) {
	jobName := fmt.Sprintf("%s-%s", restore.Name, step)
	job, err := h.deps.JobLister.Jobs(restore.Namespace).Get(jobName)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if job.Labels[label.RestoreUIDLabelKey] != string(restore.UID) {
		return nil, nil
	}
	return job, nil
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
