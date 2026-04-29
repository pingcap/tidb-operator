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
	"reflect"
	"testing"
)

// TestOptions_ReplicationPhase_DefaultsToZero documents the sentinel
// contract: a newly constructed Options has ReplicationPhase == 0,
// which is the in-band signal for "this is NOT a replication restore"
// (the cobra flag was not set). Non-zero values 1 / 2 are the only
// other legal values; range validation lives in the cmd package.
func TestOptions_ReplicationPhase_DefaultsToZero(t *testing.T) {
	opts := Options{}
	if opts.ReplicationPhase != 0 {
		t.Fatalf("expected ReplicationPhase default 0, got %d", opts.ReplicationPhase)
	}
}

// TestReplicationBRFlags verifies the replication-specific BR flags
// produced by replicationBRFlags. Phase 0 returns nil (allowing
// `append(args, replicationBRFlags(0)...)` to be a no-op for standard
// PiTR); phases 1 and 2 produce the three flags spec §6 mandates,
// with the constants for sub-prefix and concurrency wired in.
func TestReplicationBRFlags(t *testing.T) {
	cases := []struct {
		phase int
		want  []string
	}{
		{phase: 0, want: nil},
		{phase: 1, want: []string{
			"--replication-storage-phase=1",
			"--replication-status-sub-prefix=ccr",
			"--pitr-concurrency=1024",
		}},
		{phase: 2, want: []string{
			"--replication-storage-phase=2",
			"--replication-status-sub-prefix=ccr",
			"--pitr-concurrency=1024",
		}},
	}
	for _, c := range cases {
		got := replicationBRFlags(c.phase)
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("phase=%d: got %v, want %v", c.phase, got, c.want)
		}
	}
}
