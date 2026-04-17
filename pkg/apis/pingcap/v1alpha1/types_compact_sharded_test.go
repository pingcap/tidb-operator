// Copyright 2018 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import "testing"

func TestCompactSpecShardedFieldsDeepCopy(t *testing.T) {
	shardCount := int32(3)
	original := &CompactSpec{
		Mode:       CompactModeSharded,
		ShardCount: &shardCount,
	}

	copied := original.DeepCopy()
	if copied == nil {
		t.Fatal("expected DeepCopy to return a value")
	}
	if copied == original {
		t.Fatal("expected DeepCopy to return a distinct object")
	}
	if copied.Mode != original.Mode {
		t.Fatalf("expected Mode to be copied, got %q want %q", copied.Mode, original.Mode)
	}
	if copied.ShardCount == nil {
		t.Fatal("expected ShardCount to be copied")
	}
	if copied.ShardCount == original.ShardCount {
		t.Fatal("expected ShardCount pointer to be copied")
	}
	if *copied.ShardCount != *original.ShardCount {
		t.Fatalf("expected ShardCount to be copied, got %d want %d", *copied.ShardCount, *original.ShardCount)
	}
}

func TestCompactStatusShardedFieldsDeepCopy(t *testing.T) {
	original := &CompactStatus{
		CompletedIndexes: "1,3-5,7",
		FailedIndexes:    "2,4",
	}

	copied := original.DeepCopy()
	if copied == nil {
		t.Fatal("expected DeepCopy to return a value")
	}
	if copied == original {
		t.Fatal("expected DeepCopy to return a distinct object")
	}
	if copied.CompletedIndexes != original.CompletedIndexes {
		t.Fatalf("expected CompletedIndexes to be copied, got %q want %q", copied.CompletedIndexes, original.CompletedIndexes)
	}
	if copied.FailedIndexes != original.FailedIndexes {
		t.Fatalf("expected FailedIndexes to be copied, got %q want %q", copied.FailedIndexes, original.FailedIndexes)
	}
}
