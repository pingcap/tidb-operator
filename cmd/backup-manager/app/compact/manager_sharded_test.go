package compact

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/compact/options"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	listers "github.com/pingcap/tidb-operator/pkg/client/listers/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

func TestBuildCompactArgsDefaultMode(t *testing.T) {
	manager := &Manager{
		options: options.CompactOpts{
			FromTS:      11,
			UntilTS:     22,
			Concurrency: 4,
		},
	}

	args := manager.buildCompactArgs("storage-base64")
	want := []string{
		"--log-level", "INFO",
		"--log-format", "json",
		"compact-log-backup",
		"--storage-base64", "storage-base64",
		"--from", "11",
		"--until", "22",
		"-N", "4",
	}

	assertStringSliceEqual(t, args, want)
}

func TestBuildCompactArgsShardedMode(t *testing.T) {
	manager := &Manager{
		options: options.CompactOpts{
			FromTS:      11,
			UntilTS:     22,
			Concurrency: 4,
			Sharded:     true,
			ShardIndex:  1,
			ShardCount:  3,
		},
	}

	args := manager.buildCompactArgs("storage-base64")
	want := []string{
		"--log-level", "INFO",
		"--log-format", "json",
		"compact-log-backup",
		"--storage-base64", "storage-base64",
		"--from", "11",
		"--until", "18446744073709551615",
		"-N", "4",
		"--shard", "1/3",
		"--minimal-compaction-size", "0",
	}

	assertStringSliceEqual(t, args, want)
}

func TestProcessCompactFailsWhenShardedRuntimeIndexIsInvalid(t *testing.T) {
	testCases := []struct {
		name     string
		envValue *string
		wantErr  string
	}{
		{
			name:    "missing env",
			wantErr: "JOB_COMPLETION_INDEX",
		},
		{
			name:     "invalid env",
			envValue: stringPtr("abc"),
			wantErr:  "failed to parse compact options",
		},
		{
			name:     "out of range env",
			envValue: stringPtr("4"),
			wantErr:  "out of range",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envValue == nil {
				_ = os.Unsetenv("JOB_COMPLETION_INDEX")
				t.Cleanup(func() {
					_ = os.Unsetenv("JOB_COMPLETION_INDEX")
				})
			} else {
				t.Setenv("JOB_COMPLETION_INDEX", *tc.envValue)
			}

			compact := newShardedCompactBackupForManagerTest()
			manager := newManagerForProcessCompactTest(t, compact)

			err := manager.ProcessCompact()
			if err == nil {
				t.Fatal("expected ProcessCompact to fail")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func assertStringSliceEqual(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("unexpected arg count: got %d want %d; got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected arg at %d: got %q want %q; got=%v", i, got[i], want[i], got)
		}
	}
}

func newManagerForProcessCompactTest(t *testing.T, compact *v1alpha1.CompactBackup) *Manager {
	t.Helper()

	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	if err := indexer.Add(compact); err != nil {
		t.Fatalf("failed to add compact backup to indexer: %v", err)
	}

	manager := NewManager(
		listers.NewCompactBackupLister(indexer),
		&fakeCompactStatusUpdater{},
		options.CompactOpts{
			Namespace:    compact.Namespace,
			ResourceName: compact.Name,
		},
	)
	if manager == nil {
		t.Fatal("expected manager to be created")
	}
	return manager
}

func newShardedCompactBackupForManagerTest() *v1alpha1.CompactBackup {
	shardCount := int32(4)
	return &v1alpha1.CompactBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "compact-sharded",
			Namespace: "default",
		},
		Spec: v1alpha1.CompactSpec{
			StartTs:     "400036290571534337",
			EndTs:       "400036290571534338",
			Concurrency: 4,
			Mode:        v1alpha1.CompactModeSharded,
			ShardCount:  &shardCount,
		},
	}
}

func stringPtr(v string) *string {
	return &v
}

type fakeCompactStatusUpdater struct{}

func (f *fakeCompactStatusUpdater) OnSchedule(_ context.Context, _ *v1alpha1.CompactBackup, _ error) error {
	return nil
}

func (f *fakeCompactStatusUpdater) OnCreateJob(_ context.Context, _ *v1alpha1.CompactBackup, _ error) error {
	return nil
}

func (f *fakeCompactStatusUpdater) OnStart(_ context.Context, _ *v1alpha1.CompactBackup) error {
	return nil
}

func (f *fakeCompactStatusUpdater) OnProgress(_ context.Context, _ *v1alpha1.CompactBackup, _ *controller.Progress, _ string) error {
	return nil
}

func (f *fakeCompactStatusUpdater) OnFinish(_ context.Context, _ *v1alpha1.CompactBackup, _ error) error {
	return nil
}

func (f *fakeCompactStatusUpdater) OnJobComplete(_ context.Context, _ *v1alpha1.CompactBackup, _, _ string) error {
	return nil
}

func (f *fakeCompactStatusUpdater) OnJobFailed(_ context.Context, _ *v1alpha1.CompactBackup, _, _, _ string) error {
	return nil
}

func (f *fakeCompactStatusUpdater) UpdateStatus(_ *v1alpha1.CompactBackup, _ v1alpha1.CompactStatus) error {
	return nil
}

func (f *fakeCompactStatusUpdater) UpdateShardIndexes(_ *v1alpha1.CompactBackup, _ batchv1.JobStatus) error {
	return nil
}
