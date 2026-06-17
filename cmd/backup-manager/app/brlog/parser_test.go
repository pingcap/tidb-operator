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

package brlog

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseLineOperationStarted(t *testing.T) {
	line := `{"level":"info","time":"2026/06/17 10:00:00.000 +00:00","caller":"operation/context.go:122","message":"BR operation started","operation_id":"11111111-1111-1111-1111-111111111111","operation_started_at":"2026-06-17T10:00:00Z","host":"br-pod","pid":123,"command":"log-restore"}`

	event := ParseLine(line)
	if event.Type != EventOperationStarted {
		t.Fatalf("expected event type %q, got %q", EventOperationStarted, event.Type)
	}
	if event.Operation == nil {
		t.Fatal("expected operation")
	}
	if event.LockBlocker != nil {
		t.Fatalf("expected no lock blocker, got %#v", event.LockBlocker)
	}
	if event.Operation.OperationID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected operation id: %q", event.Operation.OperationID)
	}
	if event.Operation.Command != "log-restore" {
		t.Fatalf("unexpected command: %q", event.Operation.Command)
	}
	assertOptionalTime(t, "started at", event.Operation.StartedAt, "2026-06-17T10:00:00Z")
	assertTime(t, "observed at", event.Operation.ObservedAt.Time, "2026-06-17T10:00:00Z")
}

func TestParseLineLockConflict(t *testing.T) {
	line := `{"level":"warn","time":"2026/06/17 10:01:00.000 +00:00","caller":"stream/stream_metas.go:1497","message":"Failed to acquire log truncate lock","error":"locked","path":"truncating.lock","local_owner_id":"local-op","local_lock_type":"log-truncate-exclusive","local_hint":"operation_started_at=2026-06-17T10:00:00Z","remote_blocker_count":1,"remote_blocker_0_path":"truncating.lock","remote_blocker_0_owner_id":"remote-op","remote_blocker_0_lock_type":"log-truncate-exclusive","remote_blocker_0_hint":"operation_started_at=2026-06-17T09:00:00Z restore_id=123"}`

	event := ParseLine(line)
	if event.Type != EventLockConflict {
		t.Fatalf("expected event type %q, got %q", EventLockConflict, event.Type)
	}
	if event.LockBlocker == nil {
		t.Fatal("expected lock blocker")
	}
	if event.Operation != nil {
		t.Fatalf("expected no operation, got %#v", event.Operation)
	}
	if event.LockBlocker.LockPath != "truncating.lock" {
		t.Fatalf("unexpected lock path: %q", event.LockBlocker.LockPath)
	}
	if event.LockBlocker.RemoteOperationID != "remote-op" {
		t.Fatalf("unexpected remote operation id: %q", event.LockBlocker.RemoteOperationID)
	}
	if event.LockBlocker.ResourceType != "log-truncate-exclusive" {
		t.Fatalf("unexpected resource type: %q", event.LockBlocker.ResourceType)
	}
	assertOptionalTime(t, "remote started at", event.LockBlocker.RemoteStartedAt, "2026-06-17T09:00:00Z")
	assertTime(t, "observed at", event.LockBlocker.ObservedAt.Time, "2026-06-17T10:01:00Z")
}

func TestParseLineIgnoresNonJSON(t *testing.T) {
	event := ParseLine("plain text")
	if event.Type != EventNone || event.Operation != nil || event.LockBlocker != nil {
		t.Fatalf("expected empty event, got %#v", event)
	}
}

func TestParseLineIgnoresUnknownJSON(t *testing.T) {
	event := ParseLine(`{"level":"info","time":"2026/06/17 10:00:00.000 +00:00","message":"unknown"}`)
	if event.Type != EventNone || event.Operation != nil || event.LockBlocker != nil {
		t.Fatalf("expected empty event, got %#v", event)
	}
}

func TestParseLineLockConflictFallbackRemoteFields(t *testing.T) {
	line := `{"level":"warn","time":"2026-06-17T10:02:00Z","message":"Encountered lock, will retry","path":"truncating.lock","remote_owner_id":"remote-op","remote_lock_type":"log-truncate-exclusive","remote_hint":"operation_started_at=2026-06-17T09:30:00Z restore_id=456"}`

	event := ParseLine(line)
	if event.Type != EventLockConflict {
		t.Fatalf("expected event type %q, got %q", EventLockConflict, event.Type)
	}
	if event.LockBlocker == nil {
		t.Fatal("expected lock blocker")
	}
	if event.LockBlocker.LockPath != "truncating.lock" {
		t.Fatalf("unexpected lock path: %q", event.LockBlocker.LockPath)
	}
	if event.LockBlocker.RemoteOperationID != "remote-op" {
		t.Fatalf("unexpected remote operation id: %q", event.LockBlocker.RemoteOperationID)
	}
	if event.LockBlocker.ResourceType != "log-truncate-exclusive" {
		t.Fatalf("unexpected resource type: %q", event.LockBlocker.ResourceType)
	}
	assertOptionalTime(t, "remote started at", event.LockBlocker.RemoteStartedAt, "2026-06-17T09:30:00Z")
	assertTime(t, "observed at", event.LockBlocker.ObservedAt.Time, "2026-06-17T10:02:00Z")
}

func TestParseLineIgnoresLockConflictWithoutRemoteOperationID(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{
			name: "sampled remote owner empty",
			line: `{"level":"warn","time":"2026-06-17T10:02:00Z","message":"Failed to acquire log truncate lock","path":"truncating.lock","remote_blocker_0_owner_id":"","remote_blocker_0_lock_type":"log-truncate-exclusive"}`,
		},
		{
			name: "sampled remote owner non-string",
			line: `{"level":"warn","time":"2026-06-17T10:02:00Z","message":"Failed to acquire log truncate lock","path":"truncating.lock","remote_blocker_0_owner_id":123,"remote_blocker_0_lock_type":"log-truncate-exclusive"}`,
		},
		{
			name: "fallback remote owner empty",
			line: `{"level":"warn","time":"2026-06-17T10:02:00Z","message":"Encountered lock, will retry","path":"truncating.lock","remote_owner_id":"","remote_lock_type":"log-truncate-exclusive"}`,
		},
		{
			name: "fallback remote owner non-string",
			line: `{"level":"warn","time":"2026-06-17T10:02:00Z","message":"Encountered lock, will retry","path":"truncating.lock","remote_owner_id":123,"remote_lock_type":"log-truncate-exclusive"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := ParseLine(tt.line)
			if event.Type != EventNone || event.Operation != nil || event.LockBlocker != nil {
				t.Fatalf("expected empty event, got %#v", event)
			}
		})
	}
}

func assertOptionalTime(t *testing.T, name string, got *metav1.Time, want string) {
	t.Helper()

	if got == nil {
		t.Fatalf("expected %s to be set", name)
	}
	assertTime(t, name, got.Time, want)
}

func assertTime(t *testing.T, name string, got time.Time, want string) {
	t.Helper()

	wantTime, err := time.Parse(time.RFC3339, want)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(wantTime) {
		t.Fatalf("unexpected %s: got %s, want %s", name, got.Format(time.RFC3339), wantTime.Format(time.RFC3339))
	}
}
