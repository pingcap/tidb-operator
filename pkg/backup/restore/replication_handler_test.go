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
