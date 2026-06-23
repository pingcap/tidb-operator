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
)

// Event is one parsed BR structured-log event.
type Event struct {
	Type      EventType
	Operation *v1alpha1.BROperation
}

// ParseLine extracts a BR operation or external-storage-lock event from one BR log line.
func ParseLine(line string) Event {
	fields := parseFields(line)
	if len(fields) == 0 {
		return Event{}
	}

	if operation, ok := parseOperationStarted(fields); ok {
		return Event{
			Type:      EventOperationStarted,
			Operation: operation,
		}
	}

	return Event{}
}

// IsErrorLine reports whether a BR log line should be included in command error output.
func IsErrorLine(line string) bool {
	if isTextErrorLine(line) {
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

func parseOperationStarted(fields map[string]string) (*v1alpha1.BROperation, bool) {
	operationID := fields[fieldOperationID]
	if logMessage(fields) != messageOperationStarted || operationID == "" {
		return nil, false
	}

	return &v1alpha1.BROperation{
		OperationID: operationID,
		StartedAt:   parseOptionalMetav1Time(fields[fieldOperationStartedAt]),
		Command:     fields[fieldCommand],
		ObservedAt:  parseMetav1Time(fields[fieldTime]),
	}, true
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

	for _, token := range bracketTokens(line) {
		if key, value, ok := strings.Cut(token, "="); ok && isFieldKey(key) {
			fields[key] = unquoteLogValue(strings.TrimSpace(value))
			continue
		}
		if message := unquoteLogValue(strings.TrimSpace(token)); message == messageOperationStarted {
			fields[fieldMessage] = message
			continue
		}
		if parsed := parseMetav1Time(token); !parsed.IsZero() {
			fields[fieldTime] = token
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
		end := closingBracketIndex(line)
		if end == -1 {
			return tokens
		}
		tokens = append(tokens, line[:end])
		line = line[end+1:]
	}
}

func closingBracketIndex(value string) int {
	quoted := false
	escaped := false
	for i, r := range value {
		if escaped {
			escaped = false
			continue
		}
		switch r {
		case '\\':
			escaped = quoted
		case '"':
			quoted = !quoted
		case ']':
			if !quoted {
				return i
			}
		}
	}
	return -1
}

func isFieldKey(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func isTextErrorLine(line string) bool {
	for _, token := range bracketTokens(line) {
		switch strings.ToUpper(strings.TrimSpace(token)) {
		case "ERROR", "FATAL", "PANIC":
			return true
		}
	}
	return false
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
