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

package controller

import (
	"context"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
)

type spyCompactStatusUpdater struct {
	onStartCalls    int
	onProgressCalls int
	onFinishCalls   int
}

func (s *spyCompactStatusUpdater) OnSchedule(_ context.Context, _ *v1alpha1.CompactBackup, _ error) error {
	return nil
}
func (s *spyCompactStatusUpdater) OnCreateJob(_ context.Context, _ *v1alpha1.CompactBackup, _ error) error {
	return nil
}
func (s *spyCompactStatusUpdater) OnStart(_ context.Context, _ *v1alpha1.CompactBackup) error {
	s.onStartCalls++
	return nil
}
func (s *spyCompactStatusUpdater) OnProgress(_ context.Context, _ *v1alpha1.CompactBackup, _ *Progress, _ string) error {
	s.onProgressCalls++
	return nil
}
func (s *spyCompactStatusUpdater) OnFinish(_ context.Context, _ *v1alpha1.CompactBackup, _ error) error {
	s.onFinishCalls++
	return nil
}
func (s *spyCompactStatusUpdater) OnJobComplete(_ context.Context, _ *v1alpha1.CompactBackup, _, _ string) error {
	return nil
}
func (s *spyCompactStatusUpdater) OnJobFailed(_ context.Context, _ *v1alpha1.CompactBackup, _, _, _ string) error {
	return nil
}
func (s *spyCompactStatusUpdater) UpdateShardIndexes(_ *v1alpha1.CompactBackup, _ batchv1.JobStatus) error {
	return nil
}

func TestShardedCompactStatusUpdaterForwardsOnStart(t *testing.T) {
	spy := &spyCompactStatusUpdater{}
	sharded := NewShardedCompactStatusUpdater(spy)

	if err := sharded.OnStart(context.TODO(), &v1alpha1.CompactBackup{}); err != nil {
		t.Fatalf("OnStart returned error: %v", err)
	}

	if spy.onStartCalls != 1 {
		t.Fatalf("expected OnStart to be forwarded once, got %d", spy.onStartCalls)
	}
}

func TestShardedCompactStatusUpdaterSuppressesOnProgressAndOnFinish(t *testing.T) {
	spy := &spyCompactStatusUpdater{}
	sharded := NewShardedCompactStatusUpdater(spy)

	if err := sharded.OnProgress(context.TODO(), &v1alpha1.CompactBackup{}, nil, ""); err != nil {
		t.Fatalf("OnProgress returned error: %v", err)
	}
	if err := sharded.OnFinish(context.TODO(), &v1alpha1.CompactBackup{}, nil); err != nil {
		t.Fatalf("OnFinish returned error: %v", err)
	}

	if spy.onProgressCalls != 0 {
		t.Fatalf("expected OnProgress to be suppressed, got %d forwarded calls", spy.onProgressCalls)
	}
	if spy.onFinishCalls != 0 {
		t.Fatalf("expected OnFinish to be suppressed, got %d forwarded calls", spy.onFinishCalls)
	}
}
