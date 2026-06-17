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

package backup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
)

func TestBackupLogTruncateObserverUpdatesOperationAndCapturesLatestLockBlocker(t *testing.T) {
	backup := &v1alpha1.Backup{}
	statusUpdater := &recordingBackupStatusUpdater{}
	observer := newBackupLogTruncateObserver(backup, statusUpdater)

	observer.observeLine(`[2026/06/17 10:00:00.000 +00:00] [INFO] [context.go:122] ["BR operation started"] [operation_id=op-a] [command="log truncate"]`)

	if len(statusUpdater.updates) != 1 {
		t.Fatalf("expected one status update, got %d", len(statusUpdater.updates))
	}
	operation := statusUpdater.updates[0].BROperation
	if operation == nil {
		t.Fatal("expected BR operation status update")
	}
	if operation.OperationID != "op-a" {
		t.Fatalf("unexpected operation id: %q", operation.OperationID)
	}
	if operation.ObservedAt.IsZero() {
		t.Fatal("expected observer to fill zero ObservedAt")
	}

	observer.observeLine(`[2026/06/17 10:01:00.000 +00:00] [WARN] [stream_metas.go:1497] ["Encountered lock"] [remote_owner_id=remote-a] [remote_lock_type=log-truncate-exclusive] [remote_hint=operation_started_at=2026-06-17T09:00:00Z] [path=lock-a]`)
	observer.observeLine(`[2026/06/17 10:02:00.000 +00:00] [WARN] [stream_metas.go:1497] ["Encountered lock"] [remote_owner_id=remote-b] [remote_lock_type=log-truncate-exclusive] [remote_hint=operation_started_at=2026-06-17T09:01:00Z] [path=lock-b]`)

	blocker := observer.lockBlocker
	if blocker == nil {
		t.Fatal("expected lock blocker candidate")
	}
	if blocker.RemoteOperationID != "remote-b" {
		t.Fatalf("expected latest lock blocker to win, got %q", blocker.RemoteOperationID)
	}
	if blocker.LockPath != "lock-b" {
		t.Fatalf("unexpected lock path: %q", blocker.LockPath)
	}
}

func TestBRCommandRunWithLogCallbackKeepsLegacyStdoutOnlyCallback(t *testing.T) {
	dir := t.TempDir()
	oldBRBinPath := brBinPath
	brBinPath = dir
	t.Cleanup(func() {
		brBinPath = oldBRBinPath
	})

	script := `#!/bin/sh
printf 'stdout [ERROR] one\n'
printf 'stdout two\n'
printf 'stderr [ERROR] one\n' >&2
printf 'stderr two\n' >&2
exit 1
`
	if err := os.WriteFile(filepath.Join(dir, "br"), []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake br: %v", err)
	}

	var callbackLines []string
	err := (&Options{}).brCommandRunWithLogCallback(context.Background(), []string{"log", "start"}, func(line string) {
		callbackLines = append(callbackLines, line)
	})

	if err == nil {
		t.Fatal("expected br command failure")
	}
	for _, line := range callbackLines {
		if strings.Contains(line, "stderr") {
			t.Fatalf("expected stderr to skip legacy callback, got line %q", line)
		}
	}
	if !strings.Contains(strings.Join(callbackLines, ""), "stdout [ERROR] one\nstdout two\n") {
		t.Fatalf("expected callback to receive stdout lines, got %q", callbackLines)
	}
	msg := err.Error()
	for _, want := range []string{"stdout [ERROR] one", "stderr [ERROR] one", "stderr two"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected error message to contain %q, got %q", want, msg)
		}
	}
}

func TestBRCommandRunWithLogObserverStreamsStdoutAndStderr(t *testing.T) {
	dir := t.TempDir()
	oldBRBinPath := brBinPath
	brBinPath = dir
	t.Cleanup(func() {
		brBinPath = oldBRBinPath
	})

	script := `#!/bin/sh
printf 'stdout [ERROR] one\n'
printf '{"level":"error","message":"json stdout failed"}\n'
printf 'stdout two\n'
printf 'stderr [ERROR] one\n' >&2
printf '{"level":"fatal","message":"json stderr failed"}\n' >&2
printf 'stderr two\n' >&2
exit 1
`
	if err := os.WriteFile(filepath.Join(dir, "br"), []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake br: %v", err)
	}

	var observedLines []string
	err := (&Options{}).brCommandRunWithLogObserver(context.Background(), []string{"log", "truncate"}, func(line string) {
		observedLines = append(observedLines, line)
	})

	if err == nil {
		t.Fatal("expected br command failure")
	}
	observed := strings.Join(observedLines, "\n")
	for _, want := range []string{"stdout [ERROR] one", "json stdout failed", "stdout two", "stderr [ERROR] one", "json stderr failed", "stderr two"} {
		if !strings.Contains(observed, want) {
			t.Fatalf("expected observer to receive %q, got %q", want, observedLines)
		}
	}
	msg := err.Error()
	for _, want := range []string{"stdout [ERROR] one", "json stdout failed", "stderr [ERROR] one", "json stderr failed"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected error message to contain %q, got %q", want, msg)
		}
	}
	if strings.Contains(msg, "stdout two") || strings.Contains(msg, "stderr two") {
		t.Fatalf("expected error message to include only ERROR lines from observed streams, got %q", msg)
	}
}

func TestBRCommandRunWithLogObserverGracefullyStopsOnContextCancel(t *testing.T) {
	dir := t.TempDir()
	oldBRBinPath := brBinPath
	brBinPath = dir
	t.Cleanup(func() {
		brBinPath = oldBRBinPath
	})

	script := `#!/bin/sh
exec sleep 1
`
	if err := os.WriteFile(filepath.Join(dir, "br"), []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake br: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := (&Options{}).brCommandRunWithLogObserver(ctx, []string{"log", "truncate"}, nil)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected br command to fail after context cancellation")
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("expected context cancellation to stop br quickly, took %s", elapsed)
	}
}

type recordingBackupStatusUpdater struct {
	updates []*controller.BackupUpdateStatus
}

func (r *recordingBackupStatusUpdater) Update(_ *v1alpha1.Backup, _ *v1alpha1.BackupCondition, newStatus *controller.BackupUpdateStatus) error {
	r.updates = append(r.updates, newStatus)
	return nil
}
