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

package cmd

import (
	"testing"

	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/restore"
)

// TestNewRestoreCommand_BindsReplicationPhase verifies that the
// --replicationPhase cobra flag is registered on the restore command
// and parses an integer value into the bound Options field.
func TestNewRestoreCommand_BindsReplicationPhase(t *testing.T) {
	cmd := NewRestoreCommand()
	if err := cmd.Flags().Set("replicationPhase", "2"); err != nil {
		t.Fatalf("setting --replicationPhase=2: %v", err)
	}
	got, err := cmd.Flags().GetInt("replicationPhase")
	if err != nil {
		t.Fatalf("GetInt(replicationPhase): %v", err)
	}
	if got != 2 {
		t.Fatalf("want 2, got %d", got)
	}
}

// TestValidateReplicationPhase exercises the range check on the CLI
// flag value. 0 / 1 / 2 are accepted (0 is the "flag absent" sentinel,
// 1 / 2 are the two legal phases). Anything else is rejected.
func TestValidateReplicationPhase(t *testing.T) {
	cases := []struct {
		in   int
		want bool // true = expect error
	}{
		{-1, true},
		{0, false},
		{1, false},
		{2, false},
		{3, true},
		{99, true},
	}
	for _, c := range cases {
		err := validateReplicationPhase(c.in)
		if (err != nil) != c.want {
			t.Fatalf("phase=%d: expected error=%v, got %v", c.in, c.want, err)
		}
	}
}

// TestSelectRestoreStatusUpdater_Phase1ReturnsNoOp verifies that phase 1
// swaps the real updater for the no-op wrapper (Option B activation):
// the returned interface must NOT be the sentinel "real" we passed in,
// and calling Update on it must be a no-op (return nil with nil args).
func TestSelectRestoreStatusUpdater_Phase1ReturnsNoOp(t *testing.T) {
	got := selectRestoreStatusUpdater(restore.Options{ReplicationPhase: 1}, nil)
	if got == nil {
		t.Fatal("phase=1: expected non-nil no-op wrapper, got nil")
	}
	if err := got.Update(nil, nil, nil); err != nil {
		t.Fatalf("phase=1: no-op wrapper.Update should return nil, got %v", err)
	}
}

// TestSelectRestoreStatusUpdater_NonPhase1ReturnsReal verifies that
// phase 0 (no replication) and phase 2 (log restore) both pass the
// real updater through unchanged. We use nil as the sentinel "real"
// to make pointer-equality assertion trivial.
func TestSelectRestoreStatusUpdater_NonPhase1ReturnsReal(t *testing.T) {
	for _, phase := range []int{0, 2} {
		got := selectRestoreStatusUpdater(restore.Options{ReplicationPhase: phase}, nil)
		if got != nil {
			t.Fatalf("phase=%d: expected to pass through nil real updater, got non-nil %T", phase, got)
		}
	}
}
