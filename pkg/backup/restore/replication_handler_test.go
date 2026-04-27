// Copyright 2026 PingCAP, Inc.
// Licensed under the Apache License, Version 2.0 (the "License").
package restore

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/label"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/backup/testutils"
	"github.com/pingcap/tidb-operator/pkg/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

// --- helpers for syncInitial tests ---

func newHandlerForTest(t *testing.T) (*replicationHandler, *testutils.Helper) {
	h := testutils.NewHelper(t)
	handler := newReplicationHandler(
		h.Deps,
		controller.NewRealRestoreConditionUpdater(h.Deps.Clientset, h.Deps.RestoreLister, h.Deps.Recorder),
	)
	return handler, h
}

func newReplicationRestoreFixture(name, ns, cbName string, wait *metav1.Duration) *v1alpha1.Restore {
	return &v1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
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

func ptrDuration(d time.Duration) *metav1.Duration { return &metav1.Duration{Duration: d} }

// --- syncInitial tests ---

func TestSyncInitial_NotFoundCompactBackup_WithinTimeout_Requeues(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreFixture("r1", "ns1", "nonexistent-cb", ptrDuration(10*time.Minute))
	h.CreateRestore(r)

	err := handler.Sync(r)

	g.Expect(err).NotTo(HaveOccurred(), "NotFound within timeout: requeue silently")
	g.Expect(r.Status.ReplicationStep).To(Equal(""), "no Phase or Step set yet")

	// No phase-1 Job created
	jobs, _ := h.Deps.KubeClientset.BatchV1().Jobs("ns1").List(context.TODO(), metav1.ListOptions{})
	g.Expect(jobs.Items).To(BeEmpty())
}

func TestSyncInitial_NotFoundCompactBackup_TimeoutExpired_FailsRestore(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	r := newReplicationRestoreFixture("r1", "ns1", "nonexistent-cb", ptrDuration(time.Nanosecond))
	r.CreationTimestamp = metav1.NewTime(time.Now().Add(-time.Hour))
	h.CreateRestore(r)

	err := handler.Sync(r)

	g.Expect(err).NotTo(HaveOccurred()) // Failed status write returns nil
	updated, getErr := h.Deps.Clientset.PingcapV1alpha1().Restores("ns1").Get(context.TODO(), "r1", metav1.GetOptions{})
	g.Expect(getErr).NotTo(HaveOccurred())
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreFailed))
}

func TestSyncInitial_FoundAndConsistent_CreatesPhase1JobAndSetsPhase(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	cb := newCompactBackupFixture("cb1", "ns1", v1alpha1.BackupRunning)
	_, err := h.Deps.Clientset.PingcapV1alpha1().CompactBackups("ns1").Create(context.TODO(), cb, metav1.CreateOptions{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Eventually(func() error {
		_, e := h.Deps.CompactBackupLister.CompactBackups("ns1").Get("cb1")
		return e
	}, time.Second*10).Should(BeNil())

	r := newReplicationRestoreFixture("r1", "ns1", "cb1", nil)
	h.CreateRestore(r)

	err = handler.Sync(r)
	g.Expect(err).NotTo(HaveOccurred())

	updated, _ := h.Deps.Clientset.PingcapV1alpha1().Restores("ns1").Get(context.TODO(), "r1", metav1.GetOptions{})
	g.Expect(updated.Status.ReplicationStep).To(Equal("snapshot-restore"))
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreSnapshotRestore))

	// phase-1 Job created with the right label
	jobs, _ := h.Deps.KubeClientset.BatchV1().Jobs("ns1").List(context.TODO(), metav1.ListOptions{})
	g.Expect(jobs.Items).To(HaveLen(1))
	g.Expect(jobs.Items[0].Labels[label.ReplicationStepLabelKey]).
		To(Equal(label.ReplicationStepSnapshotRestoreVal))
}

func TestSyncInitial_Inconsistent_FailsRestore(t *testing.T) {
	g := NewGomegaWithT(t)
	handler, h := newHandlerForTest(t)
	defer h.Close()

	cb := newCompactBackupFixture("cb1", "ns1", v1alpha1.BackupRunning)
	cb.Spec.BR.Cluster = "wrong-cluster" // mismatch
	_, err := h.Deps.Clientset.PingcapV1alpha1().CompactBackups("ns1").Create(context.TODO(), cb, metav1.CreateOptions{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Eventually(func() error {
		_, e := h.Deps.CompactBackupLister.CompactBackups("ns1").Get("cb1")
		return e
	}, time.Second*10).Should(BeNil())

	r := newReplicationRestoreFixture("r1", "ns1", "cb1", nil)
	h.CreateRestore(r)

	_ = handler.Sync(r)

	updated, _ := h.Deps.Clientset.PingcapV1alpha1().Restores("ns1").Get(context.TODO(), "r1", metav1.GetOptions{})
	g.Expect(updated.Status.Phase).To(Equal(v1alpha1.RestoreFailed))

	jobs, _ := h.Deps.KubeClientset.BatchV1().Jobs("ns1").List(context.TODO(), metav1.ListOptions{})
	g.Expect(jobs.Items).To(BeEmpty())
}
