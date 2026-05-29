// Copyright 2020 PingCAP, Inc.
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

package options

import (
	"os"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
)

func TestParseCompactOptionsShardedFields(t *testing.T) {
	shardCount := int32(4)
	t.Setenv("JOB_COMPLETION_INDEX", "2")
	opts := &CompactOpts{}

	compact := &v1alpha1.CompactBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "compact-sharded",
		},
		Spec: v1alpha1.CompactSpec{
			StartTs:                   "400036290571534337",
			EndTs:                     "400036290571534338",
			Concurrency:               8,
			Mode:                      v1alpha1.CompactModeSharded,
			ShardCount:                &shardCount,
			PhysicalFileCacheCapacity: "200G",
			Name:                      "compact-task-a",
		},
	}

	if err := ParseCompactOptions(compact, opts); err != nil {
		t.Fatalf("ParseCompactOptions returned error: %v", err)
	}

	if !opts.Sharded {
		t.Fatal("expected sharded mode to be parsed")
	}
	if opts.ShardCount != int(shardCount) {
		t.Fatalf("expected shard count %d, got %d", shardCount, opts.ShardCount)
	}
	if opts.ShardIndex != 2 {
		t.Fatalf("expected shard index 2, got %d", opts.ShardIndex)
	}
	if opts.PhysicalFileCacheCapacity != "200G" {
		t.Fatalf("expected physical file cache capacity %q, got %q", "200G", opts.PhysicalFileCacheCapacity)
	}
	if opts.Name != "compact-task-a" {
		t.Fatalf("expected name %q, got %q", "compact-task-a", opts.Name)
	}
}

func TestParseCompactOptionsShardedFieldsRequireRuntimeIndex(t *testing.T) {
	shardCount := int32(4)
	opts := &CompactOpts{}

	oldValue, hadValue := os.LookupEnv("JOB_COMPLETION_INDEX")
	if hadValue {
		t.Cleanup(func() {
			_ = os.Setenv("JOB_COMPLETION_INDEX", oldValue)
		})
	} else {
		t.Cleanup(func() {
			_ = os.Unsetenv("JOB_COMPLETION_INDEX")
		})
	}
	_ = os.Unsetenv("JOB_COMPLETION_INDEX")

	compact := &v1alpha1.CompactBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "compact-sharded-missing-index",
		},
		Spec: v1alpha1.CompactSpec{
			StartTs:                   "400036290571534337",
			EndTs:                     "400036290571534338",
			Concurrency:               8,
			Mode:                      v1alpha1.CompactModeSharded,
			ShardCount:                &shardCount,
			PhysicalFileCacheCapacity: "200G",
		},
	}

	err := ParseCompactOptions(compact, opts)
	if err == nil {
		t.Fatal("expected ParseCompactOptions to fail without JOB_COMPLETION_INDEX")
	}
	if !strings.Contains(err.Error(), "JOB_COMPLETION_INDEX") {
		t.Fatalf("expected JOB_COMPLETION_INDEX error, got %v", err)
	}
}

func TestParseCompactOptionsShardedFieldsRejectInvalidRuntimeIndex(t *testing.T) {
	testCases := []struct {
		name     string
		envValue string
		wantErr  string
	}{
		{
			name:     "non numeric",
			envValue: "abc",
			wantErr:  "failed to parse JOB_COMPLETION_INDEX",
		},
		{
			name:     "out of range",
			envValue: "4",
			wantErr:  "out of range",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			shardCount := int32(4)
			t.Setenv("JOB_COMPLETION_INDEX", tc.envValue)
			opts := &CompactOpts{}

			compact := &v1alpha1.CompactBackup{
				ObjectMeta: metav1.ObjectMeta{
					Name: "compact-sharded-invalid-index",
				},
				Spec: v1alpha1.CompactSpec{
					StartTs:                   "400036290571534337",
					EndTs:                     "400036290571534338",
					Concurrency:               8,
					Mode:                      v1alpha1.CompactModeSharded,
					ShardCount:                &shardCount,
					PhysicalFileCacheCapacity: "200G",
				},
			}

			err := ParseCompactOptions(compact, opts)
			if err == nil {
				t.Fatal("expected ParseCompactOptions to fail with invalid JOB_COMPLETION_INDEX")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestParseCompactOptionsDefaultModeClearsShardedFields(t *testing.T) {
	opts := &CompactOpts{
		ShardIndex: 1,
		ShardCount: 3,
		Sharded:    true,
	}

	compact := &v1alpha1.CompactBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "compact-default",
		},
		Spec: v1alpha1.CompactSpec{
			StartTs:     "400036290571534337",
			EndTs:       "400036290571534338",
			Concurrency: 4,
		},
	}

	if err := ParseCompactOptions(compact, opts); err != nil {
		t.Fatalf("ParseCompactOptions returned error: %v", err)
	}

	if opts.Sharded {
		t.Fatal("expected default mode to remain non-sharded")
	}
	if opts.ShardCount != 0 {
		t.Fatalf("expected shard count to be cleared, got %d", opts.ShardCount)
	}
	if opts.ShardIndex != 0 {
		t.Fatalf("expected shard index to be cleared, got %d", opts.ShardIndex)
	}
	if opts.Name != "" {
		t.Fatalf("expected metadata name not to be used as compact --name, got %q", opts.Name)
	}
}

func TestParseCompactOptionsShardedRequiresPhysicalFileCacheCapacity(t *testing.T) {
	shardCount := int32(4)
	t.Setenv("JOB_COMPLETION_INDEX", "2")
	opts := &CompactOpts{}

	compact := &v1alpha1.CompactBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "compact-sharded",
		},
		Spec: v1alpha1.CompactSpec{
			StartTs:     "400036290571534337",
			EndTs:       "400036290571534338",
			Concurrency: 8,
			Mode:        v1alpha1.CompactModeSharded,
			ShardCount:  &shardCount,
		},
	}

	err := ParseCompactOptions(compact, opts)
	if err == nil {
		t.Fatal("expected ParseCompactOptions to fail without physicalFileCacheCapacity")
	}
	if !strings.Contains(err.Error(), "physicalFileCacheCapacity") {
		t.Fatalf("expected physicalFileCacheCapacity validation error, got %v", err)
	}
}

func TestCompactOptsVerifyRejectsInvalidPhysicalFileCacheCapacity(t *testing.T) {
	testCases := []struct {
		name     string
		capacity string
		wantErr  string
	}{
		{
			name:     "invalid quantity",
			capacity: "150GB",
			wantErr:  "invalid physicalFileCacheCapacity",
		},
		{
			name:     "zero quantity",
			capacity: "0",
			wantErr:  "must be greater than 0",
		},
		{
			name:     "negative quantity",
			capacity: "-1G",
			wantErr:  "must be greater than 0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := CompactOpts{
				FromTS:                    1,
				UntilTS:                   2,
				Concurrency:               1,
				PhysicalFileCacheCapacity: tc.capacity,
				Sharded:                   true,
				ShardIndex:                0,
				ShardCount:                3,
			}

			err := opts.Verify()
			if err == nil {
				t.Fatal("expected Verify to fail")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestCompactOptsVerifyRejectsInvalidKubernetesShardIndex(t *testing.T) {
	testCases := []struct {
		name    string
		opts    CompactOpts
		wantErr string
	}{
		{
			name: "non-positive shard count",
			opts: CompactOpts{
				FromTS:                    1,
				UntilTS:                   2,
				Concurrency:               1,
				PhysicalFileCacheCapacity: "200G",
				Sharded:                   true,
				ShardIndex:                0,
				ShardCount:                0,
			},
			wantErr: "shard-count",
		},
		{
			name: "negative shard index",
			opts: CompactOpts{
				FromTS:                    1,
				UntilTS:                   2,
				Concurrency:               1,
				PhysicalFileCacheCapacity: "200G",
				Sharded:                   true,
				ShardIndex:                -1,
				ShardCount:                3,
			},
			wantErr: "kubernetes shard-index",
		},
		{
			name: "out of range shard index",
			opts: CompactOpts{
				FromTS:                    1,
				UntilTS:                   2,
				Concurrency:               1,
				PhysicalFileCacheCapacity: "200G",
				Sharded:                   true,
				ShardIndex:                3,
				ShardCount:                3,
			},
			wantErr: "kubernetes shard-index",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.opts.Verify()
			if err == nil {
				t.Fatal("expected Verify to fail")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestCompactOptsVerifyAllowsUnsetUntilTSOnlyWhenSharded(t *testing.T) {
	sharded := CompactOpts{
		FromTS:                    1,
		UntilTS:                   0,
		Concurrency:               1,
		PhysicalFileCacheCapacity: "200G",
		Sharded:                   true,
		ShardIndex:                0,
		ShardCount:                3,
	}
	if err := sharded.Verify(); err != nil {
		t.Fatalf("expected sharded unset UntilTS to be accepted, got %v", err)
	}

	nonSharded := CompactOpts{
		FromTS:      1,
		UntilTS:     0,
		Concurrency: 1,
		Sharded:     false,
	}
	err := nonSharded.Verify()
	if err == nil {
		t.Fatal("expected non-sharded unset UntilTS to be rejected")
	}
	if !strings.Contains(err.Error(), "until-ts must be set") {
		t.Fatalf("expected until-ts validation error, got %v", err)
	}
}
