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
// PiTR); phases 1 and 2 produce the flags spec §6 mandates.
func TestReplicationBRFlags(t *testing.T) {
	cases := []struct {
		name                    string
		phase                   int
		retainLatestMVCCVersion bool
		want                    []string
	}{
		{name: "standard restore", phase: 0, retainLatestMVCCVersion: true, want: nil},
		{name: "phase 1 retains latest mvcc", phase: 1, retainLatestMVCCVersion: true, want: []string{
			"--restore-phase=1",
			"--pitr-concurrency=1024",
			"--metadata-download-batch-size=512",
			"--retain-latest-mvcc-version",
		}},
		{name: "phase 2 retains latest mvcc after all compact shards complete", phase: 2, retainLatestMVCCVersion: true, want: []string{
			"--restore-phase=2",
			"--pitr-concurrency=1024",
			"--metadata-download-batch-size=512",
			"--retain-latest-mvcc-version",
		}},
		{name: "phase 2 omits retain latest mvcc when compact has failed shards", phase: 2, retainLatestMVCCVersion: false, want: []string{
			"--restore-phase=2",
			"--pitr-concurrency=1024",
			"--metadata-download-batch-size=512",
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := replicationBRFlags(c.phase, c.retainLatestMVCCVersion)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("phase=%d: got %v, want %v", c.phase, got, c.want)
			}
		})
	}
}
