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

import "testing"

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
