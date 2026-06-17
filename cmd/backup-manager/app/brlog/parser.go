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
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EventType string

const (
	EventNone             EventType = ""
	EventOperationStarted EventType = "operation-started"
	EventLockConflict     EventType = "lock-conflict"
)

const (
	messageOperationStarted = "BR operation started"

	fieldMessage       = "message"
	fieldMsg           = "msg"
	fieldMessageCompat = "Message"
	fieldTime          = "time"
	fieldLevel         = "level"

	fieldOperationID        = "operation_id"
	fieldOperationStartedAt = "operation_started_at"
	fieldCommand            = "command"

	fieldPath = "path"

	fieldRemoteBlocker0Path     = "remote_blocker_0_path"
	fieldRemoteBlocker0OwnerID  = "remote_blocker_0_owner_id"
	fieldRemoteBlocker0LockType = "remote_blocker_0_lock_type"
	fieldRemoteBlocker0Hint     = "remote_blocker_0_hint"

	fieldRemoteOwnerID  = "remote_owner_id"
	fieldRemoteLockType = "remote_lock_type"
	fieldRemoteHint     = "remote_hint"
)

var bracketFieldPattern = regexp.MustCompile(`\[([A-Za-z0-9_]+)=((?:"(?:\\.|[^"\\])*")|[^\]]*)\]`)

// Event is one parsed BR structured-log event.
type Event struct {
	Type        EventType
	Operation   *v1alpha1.BROperation
	LockBlocker *v1alpha1.BRLockBlocker
}

// ParseLine extracts a BR operation or external-storage-lock event from one BR log line.
func ParseLine(line string) Event {
	fields := parseFields(line)
	if len(fields) == 0 {
		return Event{}
	}

	if logMessage(fields) == messageOperationStarted {
		return Event{
			Type: EventOperationStarted,
			Operation: &v1alpha1.BROperation{
				OperationID: fields[fieldOperationID],
				StartedAt:   parseOptionalMetav1Time(fields[fieldOperationStartedAt]),
				Command:     fields[fieldCommand],
				ObservedAt:  parseMetav1Time(fields[fieldTime]),
			},
		}
	}

	if remoteOperationID := fields[fieldRemoteBlocker0OwnerID]; remoteOperationID != "" {
		hint := fields[fieldRemoteBlocker0Hint]
		return Event{
			Type: EventLockConflict,
			LockBlocker: &v1alpha1.BRLockBlocker{
				LockPath:          firstNonEmpty(fields[fieldRemoteBlocker0Path], fields[fieldPath]),
				RemoteOperationID: remoteOperationID,
				RemoteStartedAt:   parseOperationStartedAtFromHint(hint),
				ResourceType:      fields[fieldRemoteBlocker0LockType],
				ObservedAt:        parseMetav1Time(fields[fieldTime]),
			},
		}
	}

	if remoteOperationID := fields[fieldRemoteOwnerID]; remoteOperationID != "" {
		hint := fields[fieldRemoteHint]
		return Event{
			Type: EventLockConflict,
			LockBlocker: &v1alpha1.BRLockBlocker{
				LockPath:          fields[fieldPath],
				RemoteOperationID: remoteOperationID,
				RemoteStartedAt:   parseOperationStartedAtFromHint(hint),
				ResourceType:      fields[fieldRemoteLockType],
				ObservedAt:        parseMetav1Time(fields[fieldTime]),
			},
		}
	}

	return Event{}
}

// IsErrorLine reports whether a BR log line should be included in command error output.
func IsErrorLine(line string) bool {
	if strings.Contains(line, "[ERROR]") {
		return true
	}

	fields := parseJSONFields(line)
	if len(fields) == 0 {
		return false
	}

	switch strings.ToLower(fields[fieldLevel]) {
	case "error", "fatal":
		return true
	default:
		return false
	}
}

func parseFields(line string) map[string]string {
	if fields := parseJSONFields(line); len(fields) > 0 {
		return fields
	}
	return parseTextFields(line)
}

func parseJSONFields(line string) map[string]string {
	rawFields := map[string]interface{}{}
	if err := json.Unmarshal([]byte(line), &rawFields); err != nil {
		return nil
	}

	fields := make(map[string]string, len(rawFields))
	for key, value := range rawFields {
		if str, ok := value.(string); ok {
			fields[key] = str
		}
	}
	return fields
}

func parseTextFields(line string) map[string]string {
	fields := map[string]string{}
	for _, match := range bracketFieldPattern.FindAllStringSubmatch(line, -1) {
		fields[match[1]] = unquoteLogValue(strings.TrimSpace(match[2]))
	}

	if strings.Contains(line, `"`+messageOperationStarted+`"`) || strings.Contains(line, messageOperationStarted) {
		fields[fieldMessage] = messageOperationStarted
	}

	for _, token := range bracketTokens(line) {
		if parsed := parseMetav1Time(token); !parsed.IsZero() {
			fields[fieldTime] = token
			break
		}
	}

	return fields
}

func bracketTokens(line string) []string {
	var tokens []string
	for {
		start := strings.IndexByte(line, '[')
		if start == -1 {
			return tokens
		}
		line = line[start+1:]
		end := strings.IndexByte(line, ']')
		if end == -1 {
			return tokens
		}
		tokens = append(tokens, line[:end])
		line = line[end+1:]
	}
}

func unquoteLogValue(value string) string {
	if unquoted, err := strconv.Unquote(value); err == nil {
		return unquoted
	}
	return value
}

func logMessage(fields map[string]string) string {
	if message := fields[fieldMessage]; message != "" {
		return message
	}
	if message := fields[fieldMsg]; message != "" {
		return message
	}
	return fields[fieldMessageCompat]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func parseOptionalMetav1Time(value string) *metav1.Time {
	parsed := parseMetav1Time(value)
	if parsed.IsZero() {
		return nil
	}
	return &parsed
}

func parseMetav1Time(value string) metav1.Time {
	if value == "" {
		return metav1.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, "2006/01/02 15:04:05.000 -07:00"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return metav1.NewTime(parsed)
		}
	}
	return metav1.Time{}
}

func parseOperationStartedAtFromHint(hint string) *metav1.Time {
	for _, token := range strings.Fields(hint) {
		key, value, ok := strings.Cut(token, "=")
		if !ok || key != fieldOperationStartedAt {
			continue
		}
		return parseOptionalMetav1Time(value)
	}
	return nil
}
