// Copyright 2026 PingCAP, Inc.
// Licensed under the Apache License, Version 2.0 (the "License").
package restore

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/testutils"
	"github.com/pingcap/tidb-operator/pkg/controller"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
)

// =============================================================================
// Helper unit tests (independent of cascade)
// =============================================================================

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

func TestCompareStorageLocation_UnsupportedType_HasClearMessage(t *testing.T) {
	g := NewGomegaWithT(t)

	// Local storage on Restore, S3 on CompactBackup → falls into default branch.
	a := v1alpha1.StorageProvider{Local: &v1alpha1.LocalStorageProvider{}}
	b := v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{Bucket: "b"}}
	err := compareStorageLocation(a, b)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("unsupported storage provider type"),
		"error must explicitly state unsupported, not the misleading 'type differs'")
}

// =============================================================================
// Cascade test fixtures
// =============================================================================

func newHandlerForTest(t *testing.T) (*replicationHandler, *testutils.Helper) {
	h := testutils.NewHelper(t)
	handler := newReplicationHandler(
		h.Deps,
		controller.NewRealRestoreConditionUpdater(h.Deps.Clientset, h.Deps.RestoreLister, h.Deps.Recorder),
		h.Deps.Recorder,
	)
	return handler, h
}

func newReplicationRestoreFixture(name, ns, cbName string, wait *metav1.Duration) *v1alpha1.Restore {
	return &v1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			// Default to "just created" so lookupCompactBackup's
			// time.Since(CreationTimestamp) > waitTimeout check doesn't fire
			// spuriously. Tests that exercise the timeout path override this
			// to backdate the timestamp.
			CreationTimestamp: metav1.NewTime(time.Now()),
		},
		Spec: v1alpha1.RestoreSpec{
			Mode: v1alpha1.RestoreModePiTR,
			BR:   &v1alpha1.BRConfig{Cluster: "tc1", ClusterNamespace: ns},
			StorageProvider: v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{Bucket: "b", Prefix: "/p", Region: "us-west-2"},
			},
			ReplicationConfig: &v1alpha1.ReplicationConfig{
				CompactBackupName: cbName,
				WaitTimeout:       wait,
			},
			ToolImage: "pingcap/br:v8.5.0",
		},
	}
}

func newReplicationRestoreInStep(name, ns, cbName, step string) *v1alpha1.Restore {
	r := newReplicationRestoreFixture(name, ns, cbName, nil)
	r.Status.ReplicationStep = step
	switch step {
	case "snapshot-restore":
		r.Status.Phase = v1alpha1.RestoreSnapshotRestore
	case "log-restore":
		r.Status.Phase = v1alpha1.RestoreLogRestore
	}
	return r
}

func newCompactBackupFixture(name, ns string, state v1alpha1.BackupConditionType) *v1alpha1.CompactBackup {
	return &v1alpha1.CompactBackup{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: v1alpha1.CompactSpec{
			BR: &v1alpha1.BRConfig{Cluster: "tc1", ClusterNamespace: ns},
			StorageProvider: v1alpha1.StorageProvider{
				S3: &v1alpha1.S3StorageProvider{Bucket: "b", Prefix: "/p", Region: "us-west-2"},
			},
		},
		Status: v1alpha1.CompactStatus{State: string(state)},
	}
}

func newReplicationJobNoCondition(restoreName, ns, step string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", restoreName, step),
			Namespace: ns,
			Labels: map[string]string{
				label.NameLabelKey:            "restore",
				label.ComponentLabelKey:       "restore",
				label.ManagedByLabelKey:       label.TiDBOperator,
				label.RestoreLabelKey:         restoreName,
				label.ReplicationStepLabelKey: step,
			},
		},
	}
}

func newReplicationJob(restoreName, ns, step string, condType batchv1.JobConditionType) *batchv1.Job {
	j := newReplicationJobNoCondition(restoreName, ns, step)
	j.Status.Conditions = []batchv1.JobCondition{
		{Type: condType, Status: corev1.ConditionTrue},
	}
	return j
}

func ptrDuration(d time.Duration) *metav1.Duration { return &metav1.Duration{Duration: d} }

// seedCompactBackup creates the CompactBackup via Clientset and waits for the
// lister cache to see it. Returns once visible.
func seedCompactBackup(g *WithT, h *testutils.Helper, cb *v1alpha1.CompactBackup) {
	_, err := h.Deps.Clientset.PingcapV1alpha1().CompactBackups(cb.Namespace).Create(
		context.TODO(), cb, metav1.CreateOptions{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Eventually(func() error {
		_, e := h.Deps.CompactBackupLister.CompactBackups(cb.Namespace).Get(cb.Name)
		return e
	}, time.Second*10).Should(BeNil())
}

// seedJob creates a Job via KubeClientset and waits for the JobLister to see it.
func seedJob(g *WithT, h *testutils.Helper, job *batchv1.Job) {
	_, err := h.Deps.KubeClientset.BatchV1().Jobs(job.Namespace).Create(
		context.TODO(), job, metav1.CreateOptions{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Eventually(func() error {
		_, e := h.Deps.JobLister.Jobs(job.Namespace).Get(job.Name)
		return e
	}, time.Second*10).Should(BeNil())
}

// pushRestoreStatus pushes a Restore (with status) through Clientset.Update so
// the next Sync sees the seeded status. Used for branches that pre-condition
// on existing markers / Phase.
func pushRestoreStatus(g *WithT, h *testutils.Helper, r *v1alpha1.Restore) {
	_, err := h.Deps.Clientset.PingcapV1alpha1().Restores(r.Namespace).Update(
		context.TODO(), r, metav1.UpdateOptions{})
	g.Expect(err).NotTo(HaveOccurred())
}

func getRestore(g *WithT, h *testutils.Helper, ns, name string) *v1alpha1.Restore {
	r, err := h.Deps.Clientset.PingcapV1alpha1().Restores(ns).Get(context.TODO(), name, metav1.GetOptions{})
	g.Expect(err).NotTo(HaveOccurred())
	return r
}

func listJobs(g *WithT, h *testutils.Helper, ns string) []batchv1.Job {
	jobs, err := h.Deps.KubeClientset.BatchV1().Jobs(ns).List(context.TODO(), metav1.ListOptions{})
	g.Expect(err).NotTo(HaveOccurred())
	return jobs.Items
}

// expectEvent asserts that at least one buffered Event matches the given
// substring. Drains matching event so subsequent assertions on different
// reasons in the same test don't conflict.
func expectEvent(g *WithT, h *testutils.Helper, substr string) {
	fr, ok := h.Deps.Recorder.(*record.FakeRecorder)
	g.Expect(ok).To(BeTrue(), "Recorder must be FakeRecorder for assertions")
	g.Eventually(fr.Events, time.Second*5).Should(Receive(ContainSubstring(substr)),
		"expected Event matching %q", substr)
}

// expectNoEvent asserts no buffered Event for a brief settling window.
func expectNoEvent(g *WithT, h *testutils.Helper) {
	fr, ok := h.Deps.Recorder.(*record.FakeRecorder)
	g.Expect(ok).To(BeTrue())
	select {
	case e := <-fr.Events:
		g.Expect(e).To(BeEmpty(), "did not expect any Event, got: %s", e)
	case <-time.After(100 * time.Millisecond):
	}
}

// =============================================================================
// B1: terminal short-circuit
// =============================================================================

func TestSync_B1_Terminal_NoOp(t *testing.T) {
	g := NewGomegaWithT(t)
	h := testutils.NewHelper(t)
	defer h.Close()

	// Build a handler with no recorder to ensure B1 short-circuits before any
	// recorder dereference would happen.
	handler := &replicationHandler{deps: h.Deps}

	for _, cond := range []v1alpha1.RestoreConditionType{v1alpha1.RestoreComplete, v1alpha1.RestoreFailed} {
		r := &v1alpha1.Restore{
			Status: v1alpha1.RestoreStatus{
				Conditions: []v1alpha1.RestoreCondition{{Type: cond, Status: corev1.ConditionTrue}},
			},
		}
		g.Expect(handler.Sync(r)).NotTo(HaveOccurred(), "terminal %s must short-circuit", cond)
	}
}

// =============================================================================
// TopBlock: CompactBackup observation (replaces former B2 + B8)
//
// All four CB outcomes (Complete / Failed / Mismatch / WaitTimeout) settle
// as CompactSettled marker reasons rather than Restore-level Failed phases.
// Snapshot phase progression is independent — covered under B3.
// =============================================================================

func TestSync_TopBlock_ConsistencyMismatch_WritesCompactSettledFailure(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	cb := newCompactBackupFixture("cb1", "ns1", v1alpha1.BackupRunning)
	cb.Spec.BR.Cluster = "wrong-cluster"
	seedCompactBackup(g, h, cb)

	r := newReplicationRestoreFixture("r1", "ns1", "cb1", nil)
	h.CreateRestore(r)

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())

	updated := getRestore(g, h, "ns1", "r1")
	g.Expect(updated.Status.Phase).NotTo(Equal(v1alpha1.RestoreFailed),
		"mismatch must NOT fail the Restore — settle as marker reason instead")
	_, c := v1alpha1.GetRestoreCondition(&updated.Status, v1alpha1.RestoreCompactSettled)
	g.Expect(c).NotTo(BeNil(), "CompactSettled marker must be written")
	g.Expect(c.Status).To(Equal(corev1.ConditionTrue))
	g.Expect(c.Reason).To(Equal("CompactBackupMismatch"))
	g.Expect(updated.Status.Phase).To(BeEmpty(),
		"single-step: Phase must NOT advance in same Sync as the marker write")
	g.Expect(listJobs(g, h, "ns1")).To(BeEmpty())
}

func TestSync_TopBlock_CBTerminalAtStart_WritesCompactSettled(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	cb := newCompactBackupFixture("cb1", "ns1", v1alpha1.BackupComplete)
	seedCompactBackup(g, h, cb)

	r := newReplicationRestoreFixture("r1", "ns1", "cb1", nil)
	h.CreateRestore(r)

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())

	// Single write: CompactSettled marker; Phase still empty.
	updated := getRestore(g, h, "ns1", "r1")
	_, c := v1alpha1.GetRestoreCondition(&updated.Status, v1alpha1.RestoreCompactSettled)
	g.Expect(c).NotTo(BeNil(), "CompactSettled marker must be written")
	g.Expect(c.Status).To(Equal(corev1.ConditionTrue))
	g.Expect(c.Reason).To(Equal("AllShardsComplete"))
	g.Expect(updated.Status.Phase).To(BeEmpty(), "Phase must NOT advance in this same Sync")
	g.Expect(updated.Status.ReplicationStep).To(BeEmpty())
}

func TestSync_TopBlock_TimeoutExpired_WritesCompactSettledFailure(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreFixture("r1", "ns1", "missing-cb", ptrDuration(time.Nanosecond))
	r.CreationTimestamp = metav1.NewTime(time.Now().Add(-time.Hour))
	h.CreateRestore(r)

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())

	updated := getRestore(g, h, "ns1", "r1")
	g.Expect(updated.Status.Phase).NotTo(Equal(v1alpha1.RestoreFailed),
		"timeout must NOT fail the Restore — settle as marker reason instead")
	_, c := v1alpha1.GetRestoreCondition(&updated.Status, v1alpha1.RestoreCompactSettled)
	g.Expect(c).NotTo(BeNil())
	g.Expect(c.Status).To(Equal(corev1.ConditionTrue))
	g.Expect(c.Reason).To(Equal("CompactBackupWaitTimeout"))
}

func TestSync_TopBlock_CBBecomesTerminalDuringSnapshot_WritesCompactSettled(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	// Snapshot already in progress (Phase=SnapshotRestore + Step=snapshot-restore),
	// CB transitions to Complete after snapshot started — top block catches it.
	r := newReplicationRestoreInStep("r1", "ns1", "cb1", "snapshot-restore")
	h.CreateRestore(r)
	pushRestoreStatus(g, h, r)
	seedJob(g, h, newReplicationJobNoCondition("r1", "ns1", label.ReplicationStepSnapshotRestoreVal))
	seedCompactBackup(g, h, newCompactBackupFixture("cb1", "ns1", v1alpha1.BackupComplete))

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())

	updated := getRestore(g, h, "ns1", "r1")
	_, c := v1alpha1.GetRestoreCondition(&updated.Status, v1alpha1.RestoreCompactSettled)
	g.Expect(c).NotTo(BeNil())
	g.Expect(c.Status).To(Equal(corev1.ConditionTrue))
	g.Expect(c.Reason).To(Equal("AllShardsComplete"))
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreSnapshotRestore),
		"marker write must NOT change Phase")
}

// =============================================================================
// B3: enter SnapshotRestore phase (independent of cb under the new design)
// =============================================================================

func TestSync_B3_NotFoundCompactBackup_WithinTimeout_EmitsWarningAndProceedsToSnapshot(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreFixture("r1", "ns1", "missing-cb", ptrDuration(10*time.Minute))
	h.CreateRestore(r)

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())

	// New behavior: snapshot phase advances even though CB hasn't appeared.
	// The gate at B9 will block until CompactSettled is written by a future
	// reconcile (CB shows up, or WaitTimeout expires).
	updated := getRestore(g, h, "ns1", "r1")
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreSnapshotRestore),
		"snapshot must advance even without CB present")
	_, c := v1alpha1.GetRestoreCondition(&updated.Status, v1alpha1.RestoreCompactSettled)
	g.Expect(c).To(BeNil(), "CompactSettled must NOT be written while still waiting")
	expectEvent(g, h, "CompactBackupNotFound")
}

func TestSync_B3_NotFoundCompactBackup_NoTimeoutSet_StillProceedsToSnapshot(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreFixture("r1", "ns1", "missing-cb", nil)
	h.CreateRestore(r)

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())

	updated := getRestore(g, h, "ns1", "r1")
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreSnapshotRestore))
	expectEvent(g, h, "CompactBackupNotFound")
}

func TestSync_B3_NegativeWaitTimeout_TreatedAsInfinite(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	// Negative duration must be normalized to "wait indefinitely".
	r := newReplicationRestoreFixture("r1", "ns1", "missing-cb", ptrDuration(-time.Hour))
	r.CreationTimestamp = metav1.NewTime(time.Now().Add(-2 * time.Hour))
	h.CreateRestore(r)

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())

	updated := getRestore(g, h, "ns1", "r1")
	_, settled := v1alpha1.GetRestoreCondition(&updated.Status, v1alpha1.RestoreCompactSettled)
	g.Expect(settled).To(BeNil(),
		"negative WaitTimeout must NOT trigger immediate timeout-settle")
}

func TestSync_B3_CBFoundAndNotTerminal_WritesPhaseSnapshotRestore(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	cb := newCompactBackupFixture("cb1", "ns1", v1alpha1.BackupRunning)
	seedCompactBackup(g, h, cb)

	r := newReplicationRestoreFixture("r1", "ns1", "cb1", nil)
	h.CreateRestore(r)

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())

	updated := getRestore(g, h, "ns1", "r1")
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreSnapshotRestore),
		"Phase must advance to SnapshotRestore")
	g.Expect(updated.Status.ReplicationStep).To(BeEmpty(),
		"Step must NOT be set in same Sync (B4 owns that)")
	g.Expect(listJobs(g, h, "ns1")).To(BeEmpty(),
		"Job must NOT be created in same Sync (B5 owns that)")
	_, c := v1alpha1.GetRestoreCondition(&updated.Status, v1alpha1.RestoreSnapshotRestore)
	g.Expect(c).NotTo(BeNil())
	g.Expect(c.Reason).To(Equal("SnapshotRestoreStarted"),
		"Reason must be intent-named (no 'Phase1' / 'CreatedPhase1Job')")
	_, settled := v1alpha1.GetRestoreCondition(&updated.Status, v1alpha1.RestoreCompactSettled)
	g.Expect(settled).To(BeNil(),
		"CB Running → CompactSettled stays unset until CB terminates")
}

// =============================================================================
// B4: bump ReplicationStep
// =============================================================================

func TestSync_B4_PhaseSnapshotRestoreEmptyStep_BumpsStep(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreFixture("r1", "ns1", "cb1", nil)
	r.Status.Phase = v1alpha1.RestoreSnapshotRestore
	h.CreateRestore(r)
	pushRestoreStatus(g, h, r)
	// No need to seed CompactBackup; B4 doesn't depend on it.

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())

	updated := getRestore(g, h, "ns1", "r1")
	g.Expect(updated.Status.ReplicationStep).To(Equal("snapshot-restore"))
	g.Expect(listJobs(g, h, "ns1")).To(BeEmpty(),
		"Job creation belongs to B5, not B4")
}

// =============================================================================
// B5: create phase-1 Job
// =============================================================================

func TestSync_B5_NoPhase1Job_CreatesJobAndEmitsNormalEvent(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreInStep("r1", "ns1", "cb1", "snapshot-restore")
	h.CreateRestore(r)

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())

	jobs := listJobs(g, h, "ns1")
	g.Expect(jobs).To(HaveLen(1))
	g.Expect(jobs[0].Labels[label.ReplicationStepLabelKey]).
		To(Equal(label.ReplicationStepSnapshotRestoreVal))
	expectEvent(g, h, "SnapshotRestoreStarted")
}

func TestSync_B5_JobCreationIdempotent_DoubleSyncCreatesOnlyOne(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreInStep("r1", "ns1", "cb1", "snapshot-restore")
	h.CreateRestore(r)

	// First Sync: B5 creates Job.
	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())
	g.Eventually(func() int {
		jobs, _ := h.Deps.JobLister.Jobs("ns1").List(labels.Everything())
		return len(jobs)
	}, time.Second*5).Should(Equal(1))

	// Second Sync: B5 must not duplicate the Job.
	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())
	g.Expect(listJobs(g, h, "ns1")).To(HaveLen(1),
		"second Sync must not create a duplicate Job")
}

// Crash-recovery: ReplicationStep="snapshot-restore" was persisted but the
// previous Sync's ensureJobForStep didn't land. B5 must rebuild.
func TestSync_B5_CrashRecovery_StepWrittenButJobMissing_Rebuilds(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	// Simulate post-crash state: Step set, no Job exists.
	r := newReplicationRestoreInStep("r1", "ns1", "cb1", "snapshot-restore")
	h.CreateRestore(r)
	pushRestoreStatus(g, h, r)

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())
	g.Expect(listJobs(g, h, "ns1")).To(HaveLen(1),
		"crash-recovery: missing phase-1 Job must be rebuilt by B5")
}

// =============================================================================
// B6: phase-1 Job Failed → terminal failure with Reason + Event
// =============================================================================

func TestSync_B6_Phase1JobFailed_FailsWithReasonAndEvent(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreInStep("r1", "ns1", "cb1", "snapshot-restore")
	h.CreateRestore(r)
	seedJob(g, h, newReplicationJob("r1", "ns1", label.ReplicationStepSnapshotRestoreVal, batchv1.JobFailed))

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())

	updated := getRestore(g, h, "ns1", "r1")
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreFailed))
	_, c := v1alpha1.GetRestoreCondition(&updated.Status, v1alpha1.RestoreFailed)
	g.Expect(c).NotTo(BeNil())
	g.Expect(c.Reason).To(Equal("SnapshotRestoreFailed"))
	expectEvent(g, h, "SnapshotRestoreFailed")
}

// =============================================================================
// B7: write SnapshotRestored marker (Phase preserved)
// =============================================================================

func TestSync_B7_Phase1JobComplete_WritesSnapshotRestoredMarker(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreInStep("r1", "ns1", "cb1", "snapshot-restore")
	h.CreateRestore(r)
	seedJob(g, h, newReplicationJob("r1", "ns1", label.ReplicationStepSnapshotRestoreVal, batchv1.JobComplete))

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())

	updated := getRestore(g, h, "ns1", "r1")
	_, c := v1alpha1.GetRestoreCondition(&updated.Status, v1alpha1.RestoreSnapshotRestored)
	g.Expect(c).NotTo(BeNil(), "SnapshotRestored marker must be written")
	g.Expect(c.Status).To(Equal(corev1.ConditionTrue))
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreSnapshotRestore),
		"marker write must NOT change Phase")
}

// =============================================================================
// Gate (B9) — partial gate variants + transition
// =============================================================================

// Partial gate: only SnapshotRestored True. Must wait — no Phase change.
func TestSync_OnlySnapshotRestoredTrue_GateWaits(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreInStep("r1", "ns1", "cb1", "snapshot-restore")
	appendRestoreMarker(r, v1alpha1.RestoreSnapshotRestored, "JobComplete", "")
	// CompactSettled NOT set
	h.CreateRestore(r)
	pushRestoreStatus(g, h, r)
	seedJob(g, h, newReplicationJob("r1", "ns1", label.ReplicationStepSnapshotRestoreVal, batchv1.JobComplete))
	seedCompactBackup(g, h, newCompactBackupFixture("cb1", "ns1", v1alpha1.BackupRunning))

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())

	updated := getRestore(g, h, "ns1", "r1")
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreSnapshotRestore),
		"Phase must NOT advance with only SnapshotRestored True")
	g.Expect(listJobs(g, h, "ns1")).To(HaveLen(1), "phase-2 Job must not be created")
}

// Mirror gate: only CompactSettled True. Same wait expectation.
func TestSync_OnlyCompactSettledTrue_GateWaits(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreInStep("r1", "ns1", "cb1", "snapshot-restore")
	appendRestoreMarker(r, v1alpha1.RestoreCompactSettled, "AllShardsComplete", "")
	// SnapshotRestored NOT set
	h.CreateRestore(r)
	pushRestoreStatus(g, h, r)
	// phase-1 Job exists but not Complete yet
	seedJob(g, h, newReplicationJobNoCondition("r1", "ns1", label.ReplicationStepSnapshotRestoreVal))
	seedCompactBackup(g, h, newCompactBackupFixture("cb1", "ns1", v1alpha1.BackupComplete))

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())

	updated := getRestore(g, h, "ns1", "r1")
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreSnapshotRestore),
		"Phase must NOT advance with only CompactSettled True")
	g.Expect(listJobs(g, h, "ns1")).To(HaveLen(1), "phase-2 Job must not be created")
}

func TestSync_B9_BothMarkersTrue_TransitionsPhaseToLogRestore(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreInStep("r1", "ns1", "cb1", "snapshot-restore")
	appendRestoreMarker(r, v1alpha1.RestoreSnapshotRestored, "JobComplete", "")
	appendRestoreMarker(r, v1alpha1.RestoreCompactSettled, "AllShardsComplete", "")
	h.CreateRestore(r)
	pushRestoreStatus(g, h, r)
	seedJob(g, h, newReplicationJob("r1", "ns1", label.ReplicationStepSnapshotRestoreVal, batchv1.JobComplete))
	seedCompactBackup(g, h, newCompactBackupFixture("cb1", "ns1", v1alpha1.BackupComplete))

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())

	updated := getRestore(g, h, "ns1", "r1")
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreLogRestore),
		"both markers True: Phase must transition to LogRestore")
	g.Expect(updated.Status.ReplicationStep).To(Equal("snapshot-restore"),
		"Step bump belongs to B10, not B9")
	g.Expect(listJobs(g, h, "ns1")).To(HaveLen(1),
		"phase-2 Job creation belongs to B11, not B9")

	_, c := v1alpha1.GetRestoreCondition(&updated.Status, v1alpha1.RestoreLogRestore)
	g.Expect(c).NotTo(BeNil())
	g.Expect(c.Reason).To(Equal("GatePassed"))
}

// =============================================================================
// Parallel-progression invariant: snapshot is independent of CB presence
// =============================================================================

// Multi-Sync trace: with no CB, snapshot still drives all the way through to
// phase-1 Job creation. Phase / Step / Job advances in successive reconciles
// without ever waiting for CompactSettled.
func TestSync_NoCBYet_SnapshotPhaseDrivesIndependently(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	// CB never created; WaitTimeout long enough not to expire during the test.
	r := newReplicationRestoreFixture("r1", "ns1", "missing-cb", ptrDuration(10*time.Minute))
	h.CreateRestore(r)

	// Sync 1: top block emits Warning, no settle yet → B3 writes Phase=SnapshotRestore.
	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())
	upd := getRestore(g, h, "ns1", "r1")
	g.Expect(upd.Status.Phase).To(Equal(v1alpha1.RestoreSnapshotRestore))
	g.Expect(upd.Status.ReplicationStep).To(BeEmpty())

	// Sync 2: B4 bumps Step.
	g.Expect(handler.Sync(upd)).NotTo(HaveOccurred())
	upd = getRestore(g, h, "ns1", "r1")
	g.Expect(upd.Status.ReplicationStep).To(Equal("snapshot-restore"))
	g.Expect(listJobs(g, h, "ns1")).To(BeEmpty(), "Job creation belongs to B5, not B4")

	// Sync 3: B5 creates phase-1 Job — even though CompactSettled is still unset.
	g.Expect(handler.Sync(upd)).NotTo(HaveOccurred())
	g.Eventually(func() int {
		jobs, _ := h.Deps.JobLister.Jobs("ns1").List(labels.Everything())
		return len(jobs)
	}, time.Second*5).Should(Equal(1), "phase-1 Job created without CB")

	upd = getRestore(g, h, "ns1", "r1")
	_, settled := v1alpha1.GetRestoreCondition(&upd.Status, v1alpha1.RestoreCompactSettled)
	g.Expect(settled).To(BeNil(),
		"CompactSettled stays unset; gate at B9 will block until either CB resolves or WaitTimeout expires")
}

// CompactSettled with a non-success Reason (e.g. CompactBackupWaitTimeout from
// the top block) must NOT block the gate at B9. BR phase-2 reads the reason
// and decides fallback; the controller's responsibility ends at "marker is
// True", regardless of why.
func TestSync_CompactSettledFailure_DoesNotBlockGate(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreInStep("r1", "ns1", "cb1", "snapshot-restore")
	// Pre-set both markers as if previous reconciles wrote them — but
	// CompactSettled carries a failure reason from a CB issue, not success.
	appendRestoreMarker(r, v1alpha1.RestoreSnapshotRestored, "JobComplete", "")
	appendRestoreMarker(r, v1alpha1.RestoreCompactSettled, "CompactBackupWaitTimeout",
		"CompactBackup never appeared")
	h.CreateRestore(r)
	pushRestoreStatus(g, h, r)
	seedJob(g, h, newReplicationJob("r1", "ns1", label.ReplicationStepSnapshotRestoreVal, batchv1.JobComplete))
	// No CB seeded — irrelevant since CompactSettled is already True.

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())

	updated := getRestore(g, h, "ns1", "r1")
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreLogRestore),
		"gate must pass on CompactSettled=True regardless of reason")
	_, c := v1alpha1.GetRestoreCondition(&updated.Status, v1alpha1.RestoreLogRestore)
	g.Expect(c).NotTo(BeNil())
	g.Expect(c.Reason).To(Equal("GatePassed"))
}

// =============================================================================
// B10 / B11: Step bump + phase-2 Job creation
// =============================================================================

func TestSync_B10_PhaseLogRestoreStepStillSnapshotRestore_BumpsStep(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreInStep("r1", "ns1", "cb1", "snapshot-restore")
	r.Status.Phase = v1alpha1.RestoreLogRestore
	h.CreateRestore(r)
	pushRestoreStatus(g, h, r)

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())

	updated := getRestore(g, h, "ns1", "r1")
	g.Expect(updated.Status.ReplicationStep).To(Equal("log-restore"))
	g.Expect(listJobs(g, h, "ns1")).To(BeEmpty(), "B11 owns phase-2 Job creation, not B10")
}

func TestSync_B11_NoPhase2Job_CreatesJobAndEmitsNormalEvent(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreInStep("r1", "ns1", "cb1", "log-restore")
	h.CreateRestore(r)

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())

	jobs := listJobs(g, h, "ns1")
	g.Expect(jobs).To(HaveLen(1))
	g.Expect(jobs[0].Labels[label.ReplicationStepLabelKey]).
		To(Equal(label.ReplicationStepLogRestoreVal))
	expectEvent(g, h, "LogRestoreStarted")
}

// =============================================================================
// B12: phase-2 Job Failed + default no-op
// =============================================================================

func TestSync_B12_Phase2JobFailed_FailsWithReasonAndEvent(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreInStep("r1", "ns1", "cb1", "log-restore")
	h.CreateRestore(r)
	seedJob(g, h, newReplicationJob("r1", "ns1", label.ReplicationStepLogRestoreVal, batchv1.JobFailed))

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())

	updated := getRestore(g, h, "ns1", "r1")
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreFailed))
	_, c := v1alpha1.GetRestoreCondition(&updated.Status, v1alpha1.RestoreFailed)
	g.Expect(c).NotTo(BeNil())
	g.Expect(c.Reason).To(Equal("LogRestoreFailed"))
	expectEvent(g, h, "LogRestoreFailed")
}

func TestSync_B12_Phase2JobRunning_NoOp(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreInStep("r1", "ns1", "cb1", "log-restore")
	h.CreateRestore(r)
	// Job present but no terminal condition — backup-manager owns Phase writes.
	seedJob(g, h, newReplicationJobNoCondition("r1", "ns1", label.ReplicationStepLogRestoreVal))

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred())

	updated := getRestore(g, h, "ns1", "r1")
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreLogRestore))
}

// =============================================================================
// Defensive: unknown ReplicationStep
// =============================================================================

func TestSync_UnknownReplicationStep_ReturnsNilSilently(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreFixture("r1", "ns1", "cb1", nil)
	r.Status.Phase = v1alpha1.RestoreSnapshotRestore // any non-empty Phase
	r.Status.ReplicationStep = "some-future-step"    // unknown value
	h.CreateRestore(r)

	g.Expect(handler.Sync(r)).NotTo(HaveOccurred(),
		"unknown step must not error; warning is logged via klog")
	g.Expect(listJobs(g, h, "ns1")).To(BeEmpty())
}

// =============================================================================
// makeReplicationBRJob — label + arg shape
// =============================================================================

func TestMakeReplicationBRJob_SnapshotRestore_HasCorrectLabelsAndArgs(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreFixture("r1", "ns1", "cb1", nil)
	job, err := handler.makeReplicationBRJob(r, label.ReplicationStepSnapshotRestoreVal)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(job.Name).To(Equal("r1-snapshot-restore"))
	g.Expect(job.Labels[label.ReplicationStepLabelKey]).
		To(Equal(label.ReplicationStepSnapshotRestoreVal))
	g.Expect(job.Spec.Template.Spec.Containers[0].Args).
		To(ContainElement("--replicationPhase=1"))
}

func TestMakeReplicationBRJob_LogRestore_HasCorrectPhaseArg(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreFixture("r1", "ns1", "cb1", nil)
	job, err := handler.makeReplicationBRJob(r, label.ReplicationStepLogRestoreVal)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(job.Name).To(Equal("r1-log-restore"))
	g.Expect(job.Labels[label.ReplicationStepLabelKey]).
		To(Equal(label.ReplicationStepLogRestoreVal))
	g.Expect(job.Spec.Template.Spec.Containers[0].Args).
		To(ContainElement("--replicationPhase=2"))
}
