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

// Event is one parsed BR structured-log event.
type Event struct {
	Type        EventType
	Operation   *v1alpha1.BROperation
	LockBlocker *v1alpha1.BRLockBlocker
}

// ParseLine extracts a BR operation or external-storage-lock event from one JSON log line.
func ParseLine(line string) Event {
	fields := map[string]interface{}{}
	if err := json.Unmarshal([]byte(line), &fields); err != nil {
		return Event{}
	}

	if logMessage(fields) == messageOperationStarted {
		return Event{
			Type: EventOperationStarted,
			Operation: &v1alpha1.BROperation{
				OperationID: getString(fields, fieldOperationID),
				StartedAt:   parseOptionalMetav1Time(getString(fields, fieldOperationStartedAt)),
				Command:     getString(fields, fieldCommand),
				ObservedAt:  parseMetav1Time(getString(fields, fieldTime)),
			},
		}
	}

	if remoteOperationID := getString(fields, fieldRemoteBlocker0OwnerID); remoteOperationID != "" {
		hint := getString(fields, fieldRemoteBlocker0Hint)
		return Event{
			Type: EventLockConflict,
			LockBlocker: &v1alpha1.BRLockBlocker{
				LockPath:          firstNonEmpty(getString(fields, fieldRemoteBlocker0Path), getString(fields, fieldPath)),
				RemoteOperationID: remoteOperationID,
				RemoteStartedAt:   parseOperationStartedAtFromHint(hint),
				ResourceType:      getString(fields, fieldRemoteBlocker0LockType),
				ObservedAt:        parseMetav1Time(getString(fields, fieldTime)),
			},
		}
	}

	if remoteOperationID := getString(fields, fieldRemoteOwnerID); remoteOperationID != "" {
		hint := getString(fields, fieldRemoteHint)
		return Event{
			Type: EventLockConflict,
			LockBlocker: &v1alpha1.BRLockBlocker{
				LockPath:          getString(fields, fieldPath),
				RemoteOperationID: remoteOperationID,
				RemoteStartedAt:   parseOperationStartedAtFromHint(hint),
				ResourceType:      getString(fields, fieldRemoteLockType),
				ObservedAt:        parseMetav1Time(getString(fields, fieldTime)),
			},
		}
	}

	return Event{}
}

func logMessage(fields map[string]interface{}) string {
	if message := getString(fields, fieldMessage); message != "" {
		return message
	}
	if message := getString(fields, fieldMsg); message != "" {
		return message
	}
	return getString(fields, fieldMessageCompat)
}

func getString(fields map[string]interface{}, key string) string {
	value, ok := fields[key]
	if !ok {
		return ""
	}
	str, ok := value.(string)
	if !ok {
		return ""
	}
	return str
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
