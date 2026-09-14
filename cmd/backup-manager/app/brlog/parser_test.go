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

func TestParseOperationStartedLineIgnoresJSONOperationStarted(t *testing.T) {
	line := `{"level":"info","time":"2026/06/17 10:00:00.000 +00:00","caller":"operation/context.go:122","message":"BR operation started","operation_id":"11111111-1111-1111-1111-111111111111","operation_started_at":"2026-06-17T10:00:00Z","host":"br-pod","pid":123,"command":"log-restore"}`

	operation, ok := ParseOperationStartedLine(line)
	if ok || operation != nil {
		t.Fatalf("expected JSON operation log to be ignored, got %#v", operation)
	}
}

func TestParseOperationStartedLineTextOperationStarted(t *testing.T) {
	line := `[2026/06/17 10:00:00.000 +00:00] [INFO] [context.go:122] ["BR operation started"] [operation_id=11111111-1111-1111-1111-111111111111] [operation_started_at=2026-06-17T10:00:00Z] [host=br-pod] [pid=123] [command=log-restore]`

	operation, ok := ParseOperationStartedLine(line)
	if !ok {
		t.Fatal("expected operation")
	}
	if operation == nil {
		t.Fatal("expected operation")
	}
	if operation.OperationID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected operation id: %q", operation.OperationID)
	}
	if operation.Command != "log-restore" {
		t.Fatalf("unexpected command: %q", operation.Command)
	}
	assertOptionalTime(t, "started at", operation.StartedAt, "2026-06-17T10:00:00Z")
	assertTime(t, "observed at", operation.ObservedAt.Time, "2026-06-17T10:00:00Z")
}

func TestParseOperationStartedLineTextOperationStartedWithQuotedBracketValue(t *testing.T) {
	line := `[2026/06/17 10:00:00.000 +00:00] [INFO] [context.go:122] ["BR operation started"] [operation_id=op-a] [command="log truncate [manual]"]`

	operation, ok := ParseOperationStartedLine(line)
	if !ok {
		t.Fatal("expected operation")
	}
	if operation == nil {
		t.Fatal("expected operation")
	}
	if operation.OperationID != "op-a" {
		t.Fatalf("unexpected operation id: %q", operation.OperationID)
	}
	if operation.Command != "log truncate [manual]" {
		t.Fatalf("unexpected command: %q", operation.Command)
	}
}

func TestParseOperationStartedLineIgnoresOperationStartedWithoutOperationID(t *testing.T) {
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
			operation, ok := ParseOperationStartedLine(tt.line)
			if ok || operation != nil {
				t.Fatalf("expected no operation, got %#v", operation)
			}
		})
	}
}

func TestParseOperationStartedLineIgnoresPlainTextWithoutBRFields(t *testing.T) {
	operation, ok := ParseOperationStartedLine("plain text")
	if ok || operation != nil {
		t.Fatalf("expected no operation, got %#v", operation)
	}
}

func TestParseOperationStartedLineIgnoresLockConflictMetadata(t *testing.T) {
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
			operation, ok := ParseOperationStartedLine(tt.line)
			if ok || operation != nil {
				t.Fatalf("expected lock conflict metadata to stay in BR logs only, got %#v", operation)
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
			name: "json error level stays ordinary stdout text",
			line: `{"level":"error","message":"failed to truncate log"}`,
			want: false,
		},
		{
			name: "json fatal level uppercase stays ordinary stdout text",
			line: `{"level":"FATAL","message":"br crashed"}`,
			want: false,
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

func TestParseOperationStartedLineIgnoresUnknownJSON(t *testing.T) {
	operation, ok := ParseOperationStartedLine(`{"level":"info","time":"2026/06/17 10:00:00.000 +00:00","message":"unknown"}`)
	if ok || operation != nil {
		t.Fatalf("expected no operation, got %#v", operation)
	}
}

func TestParseOperationStartedLineIgnoresJSONMessageFallbacks(t *testing.T) {
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
			operation, ok := ParseOperationStartedLine(tt.line)
			if ok || operation != nil {
				t.Fatalf("expected JSON operation log to be ignored, got %#v", operation)
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
