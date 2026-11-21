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

package util

import (
	"net/url"
	"strings"
)

// RedactBRArgs returns a copy for command logs and errors. Never pass the result
// to BR: the original arguments must reach the process unchanged.
func RedactBRArgs(args []string) []string {
	redacted := append([]string(nil), args...)
	for i := 0; i < len(redacted); i++ {
		name, value, hasValue := strings.Cut(redacted[i], "=")
		switch name {
		case "--s3.sse-kms-context", "--storage", "--full-backup-storage", "--checkpoint-storage",
			"--pitr-add-index-sql-storage", "--upstream-storage", "--downstream-storage", "-s":
			if !hasValue {
				if i+1 == len(redacted) || strings.HasPrefix(redacted[i+1], "--") {
					continue
				}
				i++
				value = redacted[i]
			}
			if name == "--s3.sse-kms-context" {
				value = "<redacted>"
			} else {
				value = redactStorageURL(value)
			}
			if hasValue {
				redacted[i] = name + "=" + value
			} else {
				redacted[i] = value
			}
		default:
			// pflag also accepts an attached value for the -s shorthand.
			if value, ok := strings.CutPrefix(redacted[i], "-s"); ok {
				redacted[i] = "-s" + redactStorageURL(value)
			}
		}
	}
	return redacted
}

func redactStorageURL(value string) string {
	u, err := url.Parse(value)
	if err != nil {
		return "<redacted>"
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		// Do not expose malformed input while reporting a BR startup failure.
		return "<redacted>"
	}
	changed := u.User != nil
	u.User = nil
	if u.Scheme != "" && u.Fragment != "" {
		// Unescaped '#' in JSON can put part of the context in the fragment.
		u.Fragment, u.RawFragment = "", ""
		changed = true
	}
	for key := range query {
		switch strings.ReplaceAll(strings.ToLower(key), "_", "-") {
		case "sse-kms-context", "access-key", "secret-access-key", "session-token":
			query.Set(key, "<redacted>")
			changed = true
		}
	}
	if !changed {
		return value
	}
	u.RawQuery = query.Encode()
	return u.String()
}
