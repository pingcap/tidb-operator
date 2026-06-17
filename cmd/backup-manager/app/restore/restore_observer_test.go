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
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestRestoreLogObserverUpdatesOperationAndCapturesLatestLockBlocker(t *testing.T) {
	restore := newRestoreForObserverTest()
	statusUpdater := &recordingRestoreStatusUpdater{}
	observer := newRestoreLogObserver(restore, statusUpdater)

	observer.observeLine(`{"message":"BR operation started","operation_id":"op-a","command":"restore full"}`)

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

	observer.observeLine(`{"time":"2026-06-17T10:01:00Z","remote_owner_id":"remote-a","remote_lock_type":"restore-exclusive","remote_hint":"operation_started_at=2026-06-17T09:00:00Z","path":"lock-a"}`)
	observer.observeLine(`{"time":"2026-06-17T10:02:00Z","remote_owner_id":"remote-b","remote_lock_type":"restore-exclusive","remote_hint":"operation_started_at=2026-06-17T09:01:00Z","path":"lock-b"}`)

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

func TestRestoreLogObserverClearsBlockerWhenFailureHasNoCandidate(t *testing.T) {
	restore := newRestoreForObserverTest()
	statusUpdater := &recordingRestoreStatusUpdater{}
	observer := newRestoreLogObserver(restore, statusUpdater)

	observer.updateLockBlockerAfterFailure()

	if len(statusUpdater.updates) != 1 {
		t.Fatalf("expected one status update, got %d", len(statusUpdater.updates))
	}
	clear := statusUpdater.updates[0].ClearLockBlocker
	if clear == nil || !*clear {
		t.Fatalf("expected clear lock blocker update, got %#v", statusUpdater.updates[0])
	}
}

func TestRestoreDataObservesStdoutAndStderrAndClearsBlockerOnSuccess(t *testing.T) {
	dir := t.TempDir()
	oldBRBinPath := brBinPath
	brBinPath = dir
	t.Cleanup(func() {
		brBinPath = oldBRBinPath
	})

	argvFile := filepath.Join(dir, "argv")
	script := `#!/bin/sh
printf '%s\n' "$@" > "$BR_ARGV_FILE"
printf '{"message":"BR operation started","operation_id":"op-stdout","command":"restore full"}\n'
printf 'stdout, [progress] [step="Full Restore"] [progress=42%%]\n'
printf '{"time":"2026-06-17T10:01:00Z","remote_owner_id":"remote-stdout","remote_lock_type":"restore-exclusive","path":"lock-stdout"}\n'
printf '{"message":"BR operation started","operation_id":"op-stderr","command":"restore full"}\n' >&2
printf 'stderr, [progress] [step="Full Restore"] [progress=99%%]\n' >&2
printf '{"time":"2026-06-17T10:02:00Z","remote_owner_id":"remote-stderr","remote_lock_type":"restore-exclusive","path":"lock-stderr"}\n' >&2
exit 0
`
	if err := os.WriteFile(filepath.Join(dir, "br"), []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake br: %v", err)
	}
	t.Setenv("BR_ARGV_FILE", argvFile)

	statusUpdater := &recordingRestoreStatusUpdater{}
	err := (&Options{}).restoreData(context.Background(), newRestoreForObserverTest(), statusUpdater, nil)
	if err != nil {
		t.Fatalf("expected restore to succeed, got %v", err)
	}

	args, err := os.ReadFile(argvFile)
	if err != nil {
		t.Fatalf("failed to read argv: %v", err)
	}
	argLines := strings.Split(strings.TrimSpace(string(args)), "\n")
	if len(argLines) == 0 || argLines[len(argLines)-1] != "--log-format=json" {
		t.Fatalf("expected --log-format=json to be the last br arg, got %q", argLines)
	}

	var sawStdoutOperation, sawStderrOperation, sawClearBlocker bool
	var progressValues []float64
	for _, update := range statusUpdater.updates {
		if update.BROperation != nil {
			switch update.BROperation.OperationID {
			case "op-stdout":
				sawStdoutOperation = true
			case "op-stderr":
				sawStderrOperation = true
			}
		}
		if update.Progress != nil {
			progressValues = append(progressValues, *update.Progress)
		}
		if update.ClearLockBlocker != nil && *update.ClearLockBlocker {
			sawClearBlocker = true
		}
	}
	if !sawStdoutOperation || !sawStderrOperation {
		t.Fatalf("expected operations from stdout and stderr, got %#v", statusUpdater.updates)
	}
	if len(progressValues) != 1 || progressValues[0] != 42 {
		t.Fatalf("expected only stdout progress update 42, got %v", progressValues)
	}
	if !sawClearBlocker {
		t.Fatalf("expected successful restore to clear lock blocker, got %#v", statusUpdater.updates)
	}
}

func TestRestoreDataFailureWritesLatestBlockerAndErrorMessageOnlyIncludesErrorLines(t *testing.T) {
	dir := t.TempDir()
	oldBRBinPath := brBinPath
	brBinPath = dir
	t.Cleanup(func() {
		brBinPath = oldBRBinPath
	})

	script := `#!/bin/sh
printf 'stdout ordinary log\n'
printf 'stdout [ERROR] failed stdout\n'
printf '{"time":"2026-06-17T10:02:00Z","remote_owner_id":"remote-stderr","remote_lock_type":"restore-exclusive","path":"lock-stderr"}\n' >&2
printf 'stderr ordinary log\n' >&2
printf 'stderr [ERROR] failed stderr\n' >&2
exit 1
`
	if err := os.WriteFile(filepath.Join(dir, "br"), []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake br: %v", err)
	}

	statusUpdater := &recordingRestoreStatusUpdater{}
	err := (&Options{}).restoreData(context.Background(), newRestoreForObserverTest(), statusUpdater, nil)
	if err == nil {
		t.Fatal("expected restore to fail")
	}
	msg := err.Error()
	for _, want := range []string{"stdout [ERROR] failed stdout", "stderr [ERROR] failed stderr"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected error message to contain %q, got %q", want, msg)
		}
	}
	for _, unwanted := range []string{"stdout ordinary log", "stderr ordinary log"} {
		if strings.Contains(msg, unwanted) {
			t.Fatalf("expected error message to omit %q, got %q", unwanted, msg)
		}
	}

	var blocker *v1alpha1.BRLockBlocker
	for _, update := range statusUpdater.updates {
		if update.LockBlocker != nil {
			blocker = update.LockBlocker
		}
	}
	if blocker == nil {
		t.Fatalf("expected failed restore to write latest lock blocker, got %#v", statusUpdater.updates)
	}
	if blocker.RemoteOperationID != "remote-stderr" || blocker.LockPath != "lock-stderr" {
		t.Fatalf("expected stderr blocker to be latest, got %#v", blocker)
	}
}

func newRestoreForObserverTest() *v1alpha1.Restore {
	return &v1alpha1.Restore{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Restore",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-restore",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test-restore"),
		},
		Spec: v1alpha1.RestoreSpec{
			StorageProvider: v1alpha1.StorageProvider{
				Local: &v1alpha1.LocalStorageProvider{
					VolumeMount: corev1.VolumeMount{
						MountPath: "/backup",
					},
					Prefix: "restore",
				},
			},
			BR: &v1alpha1.BRConfig{
				Cluster: "demo",
				Options: []string{
					"--log-format=text",
				},
			},
		},
	}
}

type recordingRestoreStatusUpdater struct {
	updates []*controller.RestoreUpdateStatus
}

func (r *recordingRestoreStatusUpdater) Update(_ *v1alpha1.Restore, _ *v1alpha1.RestoreCondition, newStatus *controller.RestoreUpdateStatus) error {
	r.updates = append(r.updates, newStatus)
	return nil
}
