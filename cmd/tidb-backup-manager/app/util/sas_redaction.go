// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"encoding/json"
	"net/url"
	"sort"
	"strings"
)

// SASRedactorForArgs redacts Azure SAS credentials in diagnostics while leaving
// the arguments passed to BR unchanged. Both storage and PiTR full-backup URLs
// may carry credentials.
func SASRedactorForArgs(args []string) *strings.Replacer {
	var tokens []string
	for i, arg := range args {
		if value, ok := strings.CutPrefix(arg, "--azblob.sas-token="); ok {
			tokens = append(tokens, value)
		} else if arg == "--azblob.sas-token" && i+1 < len(args) {
			tokens = append(tokens, args[i+1])
		}
		_, query, ok := strings.Cut(arg, "?")
		if !ok {
			continue
		}
		values, _ := url.ParseQuery(query)
		tokens = append(tokens, values["sas-token"]...)
	}
	return newSASRedactor(tokens...)
}

// newSASRedactor removes known SAS credentials and their URL/JSON encodings
// from command diagnostics.
func newSASRedactor(credentials ...string) *strings.Replacer {
	secrets := make(map[string]struct{})
	for _, token := range credentials {
		if token == "" {
			continue
		}
		secrets[token] = struct{}{}
		secrets[url.QueryEscape(token)] = struct{}{}
		encoded, _ := json.Marshal(token)
		secrets[string(encoded[1:len(encoded)-1])] = struct{}{}
	}

	tokens := make([]string, 0, len(secrets))
	for token := range secrets {
		tokens = append(tokens, token)
	}
	// Match complete credentials before any shorter credential they contain.
	sort.Slice(tokens, func(i, j int) bool { return len(tokens[i]) > len(tokens[j]) })
	pairs := make([]string, 0, 2*len(tokens))
	for _, token := range tokens {
		pairs = append(pairs, token, "[REDACTED]")
	}
	return strings.NewReplacer(pairs...)
}
