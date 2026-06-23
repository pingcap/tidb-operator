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
	if event.Operation.OperationID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected operation id: %q", event.Operation.OperationID)
	}
	if event.Operation.Command != "log-restore" {
		t.Fatalf("unexpected command: %q", event.Operation.Command)
	}
	assertOptionalTime(t, "started at", event.Operation.StartedAt, "2026-06-17T10:00:00Z")
	assertTime(t, "observed at", event.Operation.ObservedAt.Time, "2026-06-17T10:00:00Z")
}

func TestParseLineTextOperationStarted(t *testing.T) {
	line := `[2026/06/17 10:00:00.000 +00:00] [INFO] [context.go:122] ["BR operation started"] [operation_id=11111111-1111-1111-1111-111111111111] [operation_started_at=2026-06-17T10:00:00Z] [host=br-pod] [pid=123] [command=log-restore]`

	event := ParseLine(line)
	if event.Type != EventOperationStarted {
		t.Fatalf("expected event type %q, got %q", EventOperationStarted, event.Type)
	}
	if event.Operation == nil {
		t.Fatal("expected operation")
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

func TestParseLineIgnoresOperationStartedWithoutOperationID(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{
			name: "json missing operation id",
			line: `{"level":"info","time":"2026/06/17 10:00:00.000 +00:00","message":"BR operation started","command":"log-restore"}`,
		},
		{
			name: "text missing operation id",
			line: `[2026/06/17 10:00:00.000 +00:00] [INFO] [context.go:122] ["BR operation started"] [command=log-restore]`,
		},
		{
			name: "ordinary message contains marker",
			line: `[2026/06/17 10:00:00.000 +00:00] [INFO] [context.go:122] ["ordinary log"] [detail="BR operation started"] [operation_id=op-a]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := ParseLine(tt.line)
			if event.Type != EventNone || event.Operation != nil {
				t.Fatalf("expected empty event, got %#v", event)
			}
		})
	}
}

func TestParseLineIgnoresPlainTextWithoutBRFields(t *testing.T) {
	event := ParseLine("plain text")
	if event.Type != EventNone || event.Operation != nil {
		t.Fatalf("expected empty event, got %#v", event)
	}
}

func TestParseLineIgnoresLockConflictMetadata(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{
			name: "terminal truncate lock conflict",
			line: `[2026/06/17 10:01:00.000 +00:00] [WARN] [stream_metas.go:1497] ["Failed to acquire log truncate lock"] [error=locked] [path=truncating.lock] [remote_blocker_0_owner_id=remote-op]`,
		},
		{
			name: "retry lock conflict",
			line: `[2026/06/17 10:02:00.000 +00:00] [INFO] [locking.go:570] ["Encountered lock, will retry"] [path=truncating.lock] [remote_owner_id=remote-op]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := ParseLine(tt.line)
			if event.Type != EventNone || event.Operation != nil {
				t.Fatalf("expected lock conflict metadata to stay in BR logs only, got %#v", event)
			}
		})
	}
}

func TestIsErrorLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{
			name: "legacy text error",
			line: "2026-06-17 [ERROR] failed to truncate log",
			want: true,
		},
		{
			name: "legacy text fatal",
			line: "2026-06-17 [FATAL] br crashed",
			want: true,
		},
		{
			name: "legacy text panic",
			line: "2026-06-17 [PANIC] br panicked",
			want: true,
		},
		{
			name: "json error level",
			line: `{"level":"error","message":"failed to truncate log"}`,
			want: true,
		},
		{
			name: "json fatal level uppercase",
			line: `{"level":"FATAL","message":"br crashed"}`,
			want: true,
		},
		{
			name: "json info level",
			line: `{"level":"info","message":"restore progress"}`,
			want: false,
		},
		{
			name: "non json",
			line: "plain warning",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsErrorLine(tt.line); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestParseLineIgnoresUnknownJSON(t *testing.T) {
	event := ParseLine(`{"level":"info","time":"2026/06/17 10:00:00.000 +00:00","message":"unknown"}`)
	if event.Type != EventNone || event.Operation != nil {
		t.Fatalf("expected empty event, got %#v", event)
	}
}

func TestParseLineOperationStartedMessageFallbacks(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{
			name: "msg",
			line: `{"level":"info","time":"2026/06/17 10:00:00.000 +00:00","msg":"BR operation started","operation_id":"op-a","command":"log-restore"}`,
		},
		{
			name: "Message",
			line: `{"level":"info","time":"2026/06/17 10:00:00.000 +00:00","Message":"BR operation started","operation_id":"op-a","command":"log-restore"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := ParseLine(tt.line)
			if event.Type != EventOperationStarted {
				t.Fatalf("expected event type %q, got %q", EventOperationStarted, event.Type)
			}
			if event.Operation == nil || event.Operation.OperationID != "op-a" {
				t.Fatalf("unexpected operation: %#v", event.Operation)
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
