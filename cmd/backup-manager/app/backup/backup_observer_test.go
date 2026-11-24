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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStartLogBackupFailureReason(t *testing.T) {
	for _, tc := range []struct {
		name, output, redirect, wantReason string
		info                               string
	}{
		{
			name:       "Cobra stderr",
			output:     "Error: failed to start the log backup, allow only one running task: [BR:Stream:ErrStreamLogTaskExist]stream task already exists",
			redirect:   ">&2",
			wantReason: "LogBackupTaskAlreadyExists",
		},
		{
			name:       "BR stdout log",
			output:     `[2026/11/23 10:00:00.000 +00:00] [ERROR] [main.go:43] ["br failed"] [error="failed to start the log backup, allow only one running task: [BR:Stream:ErrStreamLogTaskExist]stream task already exists"]`,
			wantReason: "LogBackupTaskAlreadyExists",
		},
		{
			name:       "ordinary error",
			output:     "Error: failed to connect to PD: context deadline exceeded",
			redirect:   ">&2",
			wantReason: "StartLogBackupFailed",
		},
		{
			name:       "different error code with the same prefix",
			output:     "Error: [BR:Stream:ErrStreamLogTaskExists]stream task already exists",
			redirect:   ">&2",
			wantReason: "StartLogBackupFailed",
		},
		{
			name:       "error code echoed in a BR info log",
			info:       `[2026/11/23 10:00:00.000 +00:00] [INFO] [context.go:122] [storage="s3://bucket/[BR:Stream:ErrStreamLogTaskExist]"]`,
			output:     "Error: failed to connect to PD: context deadline exceeded",
			redirect:   ">&2",
			wantReason: "StartLogBackupFailed",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			previous := brBinPath
			brBinPath = dir
			t.Cleanup(func() { brBinPath = previous })
			script := fmt.Sprintf(`#!/bin/sh
[ "$1 $2 $3" = 'log start --task-name=log-backup' ] || exit 99
printf '%%s\n' '%s'
printf '%%s\n' '%s' %s
exit 1
`, tc.info, tc.output, tc.redirect)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "br"), []byte(script), 0755))
			backup := &v1alpha1.Backup{
				ObjectMeta: metav1.ObjectMeta{Name: "log-backup", Namespace: "ns"},
				Spec: v1alpha1.BackupSpec{
					Mode:            v1alpha1.BackupModeLog,
					BR:              &v1alpha1.BRConfig{Cluster: "target"},
					StorageProvider: v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{Bucket: "bucket", Prefix: "logs"}},
				},
			}
			updater := &recordingBackupStatusUpdater{}
			bm := NewManager(nil, updater, Options{})
			bm.SubCommand = string(v1alpha1.LogStartCommand)
			err := bm.performLogBackup(context.Background(), backup)
			require.ErrorContains(t, err, "exit status 1")
			require.ErrorContains(t, err, tc.output)
			if tc.wantReason == "LogBackupTaskAlreadyExists" {
				require.ErrorIs(t, err, errLogBackupTaskAlreadyExists)
			} else {
				require.NotErrorIs(t, err, errLogBackupTaskAlreadyExists)
			}
			failure := updater.conditions[len(updater.conditions)-1]
			require.Equal(t, v1alpha1.LogStartCommand, failure.Command)
			require.Equal(t, v1alpha1.BackupFailed, failure.Type)
			require.Equal(t, corev1.ConditionTrue, failure.Status)
			require.Equal(t, tc.wantReason, failure.Reason)
			require.Contains(t, failure.Message, tc.output)
		})
	}
}

func TestDoStopLogBackup(t *testing.T) {
	const missing = "Error: no such task log-backup: [BR:PiTR:ErrTaskNotFound]task not found"
	const overrideMissing = "Error: no such task override: [BR:PiTR:ErrTaskNotFound]task not found"
	for _, tc := range []struct {
		name, stderr string
		options      []string
		exitCode     int
		wantError    bool
	}{
		{name: "stopped"},
		{name: "task absent", stderr: missing, exitCode: 1},
		{name: "task name with equals", options: []string{"--task-name=override"}, stderr: overrideMissing, exitCode: 1},
		{name: "last task name with separate value", options: []string{"--task-name=first", "--log-format", "text", "--task-name", "override"}, stderr: overrideMissing, exitCode: 1},
		{name: "empty task name override returns error", options: []string{"--task-name=first", "--task-name="}, stderr: "Error: Miss parameters task-name: [BR:Common:ErrInvalidArgument]invalid argument", exitCode: 1, wantError: true},
		{name: "empty task name does not match backup name", options: []string{"--task-name="}, stderr: missing, exitCode: 1, wantError: true},
		{name: "empty task name is invalid even with matching stderr", options: []string{"--task-name="}, stderr: "Error: no such task : [BR:PiTR:ErrTaskNotFound]task not found", exitCode: 1, wantError: true},
		{name: "default task name does not match override", options: []string{"--task-name=override"}, stderr: missing, exitCode: 1, wantError: true},
		{name: "PD connection failed", stderr: "Error: failed to connect to PD: context deadline exceeded", exitCode: 1, wantError: true},
		{name: "different task absent", stderr: "Error: no such task other: [BR:PiTR:ErrTaskNotFound]task not found", exitCode: 1, wantError: true},
		{name: "different exit code", stderr: missing, exitCode: 2, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			previous := brBinPath
			brBinPath = dir
			t.Cleanup(func() { brBinPath = previous })
			script := fmt.Sprintf(`#!/bin/sh
[ "$1 $2 $3" = 'log stop --task-name=log-backup' ] || exit 99
printf '%%s\n' '%s' >&2
exit %d
`, tc.stderr, tc.exitCode)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "br"), []byte(script), 0755))
			backup := &v1alpha1.Backup{
				ObjectMeta: metav1.ObjectMeta{Name: "log-backup", Namespace: "ns"},
				Spec: v1alpha1.BackupSpec{
					Mode:            v1alpha1.BackupModeLog,
					BR:              &v1alpha1.BRConfig{Cluster: "target", Options: tc.options},
					StorageProvider: v1alpha1.StorageProvider{S3: &v1alpha1.S3StorageProvider{Bucket: "bucket", Prefix: "logs"}},
				},
			}
			err := (&Options{}).doStopLogBackup(context.Background(), backup)
			if tc.wantError {
				require.ErrorContains(t, err, tc.stderr)
				require.ErrorContains(t, err, fmt.Sprintf("exit status %d", tc.exitCode))
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestBackupLogTruncateObserverUpdatesOperationOnly(t *testing.T) {
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
	if len(statusUpdater.updates) != 1 {
		t.Fatalf("expected lock conflict logs to stay in BR logs only, got updates %#v", statusUpdater.updates)
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
printf 'stdout two\n'
printf 'stderr [ERROR] one\n' >&2
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
	for _, want := range []string{"stdout [ERROR] one", "stdout two", "stderr [ERROR] one", "stderr two"} {
		if !strings.Contains(observed, want) {
			t.Fatalf("expected observer to receive %q, got %q", want, observedLines)
		}
	}
	msg := err.Error()
	for _, want := range []string{"stdout [ERROR] one", "stderr [ERROR] one", "stderr two"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected error message to contain %q, got %q", want, msg)
		}
	}
	if strings.Contains(msg, "stdout two") {
		t.Fatalf("expected error message to omit ordinary stdout, got %q", msg)
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
trap 'touch "$0.term"; kill "$child" 2>/dev/null; exit 143' TERM
sleep 1 &
child=$!
wait "$child"
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
	if _, statErr := os.Stat(filepath.Join(dir, "br.term")); statErr != nil {
		t.Fatalf("expected graceful shutdown to send SIGTERM, got %v", statErr)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("expected context cancellation to stop br quickly, took %s", elapsed)
	}
}

func TestBRCommandRunWithLogCallbackLeavesCommandRunningWithoutVolumeSnapshotInitialize(t *testing.T) {
	dir := t.TempDir()
	oldBRBinPath := brBinPath
	brBinPath = dir
	t.Cleanup(func() {
		brBinPath = oldBRBinPath
	})

	script := `#!/bin/sh
trap 'touch "$0.term"; kill "$child" 2>/dev/null; exit 143' TERM
sleep 0.2 &
child=$!
wait "$child"
`
	if err := os.WriteFile(filepath.Join(dir, "br"), []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake br: %v", err)
	}

	tests := []struct {
		name       string
		mode       string
		initialize bool
	}{
		{
			name: "ordinary backup",
		},
		{
			name: "volume snapshot without initialize",
			mode: string(v1alpha1.BackupModeVolumeSnapshot),
		},
		{
			name:       "initialize without volume snapshot",
			initialize: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			termMarker := filepath.Join(dir, "br.term")
			if err := os.Remove(termMarker); err != nil && !os.IsNotExist(err) {
				t.Fatalf("failed to remove term marker: %v", err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				time.Sleep(20 * time.Millisecond)
				cancel()
			}()

			bo := &Options{}
			bo.Mode = tt.mode
			bo.Initialize = tt.initialize

			start := time.Now()
			err := bo.brCommandRunWithLogCallback(ctx, []string{"backup", "full"}, nil)
			elapsed := time.Since(start)
			if err != nil {
				t.Fatalf("expected br command to finish normally, got %v", err)
			}
			if _, statErr := os.Stat(termMarker); !os.IsNotExist(statErr) {
				t.Fatalf("expected command to keep running without SIGTERM, statErr: %v", statErr)
			}
			if elapsed < 150*time.Millisecond {
				t.Fatalf("expected command to keep running without graceful shutdown, took %s", elapsed)
			}
		})
	}
}

func TestBRCommandRunWithLogCallbackGracefullyStopsVolumeSnapshotInitialize(t *testing.T) {
	dir := t.TempDir()
	oldBRBinPath := brBinPath
	brBinPath = dir
	t.Cleanup(func() {
		brBinPath = oldBRBinPath
	})

	script := `#!/bin/sh
trap 'touch "$0.term"; kill "$child" 2>/dev/null; exit 143' TERM
sleep 1 &
child=$!
wait "$child"
`
	if err := os.WriteFile(filepath.Join(dir, "br"), []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake br: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	bo := &Options{}
	bo.Mode = string(v1alpha1.BackupModeVolumeSnapshot)
	bo.Initialize = true

	start := time.Now()
	err := bo.brCommandRunWithLogCallback(ctx, []string{"backup", "full"}, nil)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected br command to fail after context cancellation")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "br.term")); statErr != nil {
		t.Fatalf("expected graceful shutdown to send SIGTERM, got %v", statErr)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("expected context cancellation to stop volume snapshot initialize quickly, took %s", elapsed)
	}
}

type recordingBackupStatusUpdater struct {
	updates    []*controller.BackupUpdateStatus
	conditions []*v1alpha1.BackupCondition
}

func (r *recordingBackupStatusUpdater) Update(_ *v1alpha1.Backup, condition *v1alpha1.BackupCondition, newStatus *controller.BackupUpdateStatus) error {
	r.updates = append(r.updates, newStatus)
	r.conditions = append(r.conditions, condition)
	return nil
}
