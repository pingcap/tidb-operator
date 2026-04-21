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
			StartTs:     "400036290571534337",
			EndTs:       "400036290571534338",
			Concurrency: 8,
			Mode:        v1alpha1.CompactModeSharded,
			ShardCount:  &shardCount,
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
			StartTs:     "400036290571534337",
			EndTs:       "400036290571534338",
			Concurrency: 8,
			Mode:        v1alpha1.CompactModeSharded,
			ShardCount:  &shardCount,
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
					StartTs:     "400036290571534337",
					EndTs:       "400036290571534338",
					Concurrency: 8,
					Mode:        v1alpha1.CompactModeSharded,
					ShardCount:  &shardCount,
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
}

func TestCompactOptsVerifyRejectsInvalidShardedFields(t *testing.T) {
	testCases := []struct {
		name    string
		opts    CompactOpts
		wantErr string
	}{
		{
			name: "non-positive shard count",
			opts: CompactOpts{
				FromTS:      1,
				UntilTS:     2,
				Concurrency: 1,
				Sharded:     true,
				ShardIndex:  0,
				ShardCount:  0,
			},
			wantErr: "shard-count",
		},
		{
			name: "negative shard index",
			opts: CompactOpts{
				FromTS:      1,
				UntilTS:     2,
				Concurrency: 1,
				Sharded:     true,
				ShardIndex:  -1,
				ShardCount:  3,
			},
			wantErr: "shard-index",
		},
		{
			name: "out of range shard index",
			opts: CompactOpts{
				FromTS:      1,
				UntilTS:     2,
				Concurrency: 1,
				Sharded:     true,
				ShardIndex:  3,
				ShardCount:  3,
			},
			wantErr: "shard-index",
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
