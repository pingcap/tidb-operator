// Copyright 2026 PingCAP, Inc.
// Licensed under the Apache License, Version 2.0 (the "License").
package restore

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
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
