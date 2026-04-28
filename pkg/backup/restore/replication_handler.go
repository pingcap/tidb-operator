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
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
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

var _ replicationHandlerInterface = (*replicationHandler)(nil)

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

func (h *replicationHandler) syncInitial(restore *v1alpha1.Restore) error {
	cb, err := h.lookupCompactBackup(restore)
	if err != nil {
		if controller.IsIgnoreError(err) {
			return h.failRestore(restore, "CompactBackupWaitTimeout", err.Error())
		}
		return err
	}
	if cb != nil {
		if checkErr := checkCrossCRConsistency(restore, cb); checkErr != nil {
			return h.failRestore(restore, "CompactBackupMismatch", checkErr.Error())
		}
		if compactIsTerminal(cb) {
			reason := compactTerminalReason(cb)
			if markerErr := h.updateRestoreMarker(restore, v1alpha1.RestoreCompactSettled, reason, cb.Status.Message); markerErr != nil {
				return markerErr
			}
		}
	}

	var statusUpdate *controller.RestoreUpdateStatus
	if restore.Status.TimeStarted.IsZero() {
		now := metav1.Now()
		statusUpdate = &controller.RestoreUpdateStatus{TimeStarted: &now}
	}
	if err := h.statusUpdater.Update(
		restore,
		&v1alpha1.RestoreCondition{
			Type:   v1alpha1.RestoreSnapshotRestore,
			Status: corev1.ConditionTrue,
			Reason: "CreatedPhase1Job",
		},
		statusUpdate,
	); err != nil {
		return err
	}
	if err := h.setReplicationStep(restore, "snapshot-restore"); err != nil {
		return err
	}

	return h.ensureJobForStep(restore, label.ReplicationStepSnapshotRestoreVal)
}

// setReplicationStep persists Status.ReplicationStep. Reads via Clientset
// (not lister) because statusUpdater.Update wrote synchronously via Clientset
// just before this call — the lister cache may not have caught up yet, and
// reading stale state would clobber the just-written Phase. Coupled to
// realRestoreConditionUpdater's synchronous Clientset write semantics:
// switching statusUpdater to a deferred / batched / cache-only writer would
// silently break this assumption. Wrapped in retry.OnError so a self-race
// (e.g. concurrent reconcile from a Job event) doesn't lose the Step write.
func (h *replicationHandler) setReplicationStep(restore *v1alpha1.Restore, step string) error {
	ns, name := restore.Namespace, restore.Name
	return retry.OnError(retry.DefaultRetry, func(e error) bool { return e != nil }, func() error {
		// Read via Clientset (NOT lister) to see the just-written Phase from
		// statusUpdater.Update's synchronous API call. Coupled to
		// realRestoreConditionUpdater's Clientset write semantics — must
		// remain a Clientset read; switching to the lister would race.
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

func (h *replicationHandler) ensureJobForStep(restore *v1alpha1.Restore, step string) error {
	jobs, err := h.listJobsBySelector(restore.Namespace, restore.Name, step)
	if err != nil {
		return err
	}
	if len(jobs) > 0 {
		return nil
	}
	job, err := h.makeReplicationBRJob(restore, step)
	if err != nil {
		return err
	}
	return h.deps.JobControl.CreateJob(restore, job)
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

func (h *replicationHandler) syncSnapshotRestore(restore *v1alpha1.Restore) error {
	// 1. Observe phase-1 Job; update SnapshotRestored marker or fail Restore.
	phase1Job, err := h.findJobForStep(restore, label.ReplicationStepSnapshotRestoreVal)
	if err != nil {
		return err
	}
	if phase1Job != nil {
		if jobHasCondition(phase1Job, batchv1.JobFailed) {
			return h.failRestore(restore, "SnapshotRestoreFailed", jobFailureMessage(phase1Job))
		}
		if jobHasCondition(phase1Job, batchv1.JobComplete) {
			if !hasMarkerTrue(restore, v1alpha1.RestoreSnapshotRestored) {
				if err := h.updateRestoreMarker(restore, v1alpha1.RestoreSnapshotRestored,
					"JobComplete", ""); err != nil {
					return err
				}
			}
		}
	}

	// 2. Observe CompactBackup; write CompactSettled if terminal. Note: cross-CR
	// consistency was validated once at syncInitial — we do not re-validate here
	// per spec ("consistency check runs only once"). If a user mutates the
	// referenced CompactBackup spec between syncInitial and terminal state,
	// that change is silently accepted. CompactBackup spec is generally
	// immutable post-creation in practice; if this becomes a real concern,
	// add a re-check before the marker write below.
	cb, err := h.lookupCompactBackup(restore)
	if err != nil {
		if controller.IsIgnoreError(err) {
			return h.failRestore(restore, "CompactBackupWaitTimeout", err.Error())
		}
		return err
	}
	if cb != nil && compactIsTerminal(cb) {
		if !hasMarkerTrue(restore, v1alpha1.RestoreCompactSettled) {
			if err := h.updateRestoreMarker(restore, v1alpha1.RestoreCompactSettled,
				compactTerminalReason(cb), cb.Status.Message); err != nil {
				return err
			}
		}
	}

	// 3. Gate: re-read latest from Clientset (markers may have just been written),
	//    check both, and transition if both True.
	//    Must use Clientset, not lister: marker writes above went through
	//    updateRestoreMarker → Clientset; lister cache may not have caught up.
	latest, err := h.deps.Clientset.PingcapV1alpha1().Restores(restore.Namespace).Get(
		context.TODO(), restore.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if !hasMarkerTrue(latest, v1alpha1.RestoreSnapshotRestored) ||
		!hasMarkerTrue(latest, v1alpha1.RestoreCompactSettled) {
		return nil // keep waiting
	}

	// Transition: Phase=LogRestore, Step=log-restore, create phase-2 Job.
	// Phase=LogRestore is written via statusUpdater (which reads from the lister
	// inside retry.OnError). The lister cache may be stale immediately after our
	// just-written markers — Update will see a Conflict on the first attempt and
	// the retry loop in realRestoreConditionUpdater absorbs it. Self-healing; do
	// not convert to a non-retrying call.
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

func (h *replicationHandler) syncLogRestore(restore *v1alpha1.Restore) error {
	job, err := h.findJobForStep(restore, label.ReplicationStepLogRestoreVal)
	if err != nil {
		return err
	}
	if job == nil {
		// Re-entry edge case: the transition in syncSnapshotRestore successfully
		// wrote Phase=LogRestore and Step=log-restore but ensureJobForStep failed
		// before the Job was persisted (e.g. controller restart between the two
		// writes). On the next reconcile we land here with no Job — re-create it
		// so backup-manager can proceed rather than leaving the restore permanently
		// stuck in log-restore step with no running workload.
		return h.ensureJobForStep(restore, label.ReplicationStepLogRestoreVal)
	}
	if jobHasCondition(job, batchv1.JobFailed) && !v1alpha1.IsRestoreFailed(restore) {
		return h.failRestore(restore, "LogRestoreFailed", jobFailureMessage(job))
	}
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
