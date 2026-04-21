package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCompactOnFinishDoesNotFinalizeShardedState(t *testing.T) {
	testCases := []struct {
		name   string
		err    error
		verify func(t *testing.T, compact *v1alpha1.CompactBackup)
	}{
		{
			name: "success",
			verify: func(t *testing.T, compact *v1alpha1.CompactBackup) {
				if compact.Status.State != string(v1alpha1.BackupRunning) {
					t.Fatalf("expected state to remain %q, got %q", v1alpha1.BackupRunning, compact.Status.State)
				}
				if compact.Status.Message != "" {
					t.Fatalf("expected message to remain empty, got %q", compact.Status.Message)
				}
			},
		},
		{
			name: "failure",
			err:  errors.New("shard failed"),
			verify: func(t *testing.T, compact *v1alpha1.CompactBackup) {
				if compact.Status.State != string(v1alpha1.BackupRunning) {
					t.Fatalf("expected state to remain %q, got %q", v1alpha1.BackupRunning, compact.Status.State)
				}
				if compact.Status.Message != "" {
					t.Fatalf("expected message to remain empty, got %q", compact.Status.Message)
				}
				if len(compact.Status.RetryStatus) != 0 {
					t.Fatalf("expected retry status to remain empty, got %#v", compact.Status.RetryStatus)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			deps := NewFakeDependencies()
			compact := newShardedCompactBackupForStatusUpdaterTest()

			if err := deps.InformerFactory.Pingcap().V1alpha1().CompactBackups().Informer().GetIndexer().Add(compact); err != nil {
				t.Fatalf("failed to seed compact backup indexer: %v", err)
			}
			if _, err := deps.Clientset.PingcapV1alpha1().CompactBackups(compact.Namespace).Create(context.TODO(), compact, metav1.CreateOptions{}); err != nil {
				t.Fatalf("failed to seed compact backup client: %v", err)
			}

			updater := NewCompactStatusUpdater(
				deps.Recorder,
				deps.InformerFactory.Pingcap().V1alpha1().CompactBackups().Lister(),
				deps.Clientset,
			)
			if err := updater.OnFinish(context.TODO(), compact, tc.err); err != nil {
				t.Fatalf("OnFinish returned error: %v", err)
			}

			updated, err := deps.Clientset.PingcapV1alpha1().CompactBackups(compact.Namespace).Get(context.TODO(), compact.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("failed to fetch updated compact backup: %v", err)
			}
			tc.verify(t, updated)
		})
	}
}

func newShardedCompactBackupForStatusUpdaterTest() *v1alpha1.CompactBackup {
	shardCount := int32(3)
	return &v1alpha1.CompactBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "compact-sharded",
			Namespace: "default",
		},
		Spec: v1alpha1.CompactSpec{
			StartTs:     "100",
			EndTs:       "200",
			Concurrency: 4,
			Mode:        v1alpha1.CompactModeSharded,
			ShardCount:  &shardCount,
		},
		Status: v1alpha1.CompactStatus{
			State: string(v1alpha1.BackupRunning),
		},
	}
}
