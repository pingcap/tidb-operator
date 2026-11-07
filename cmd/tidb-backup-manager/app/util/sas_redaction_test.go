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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSASRedactorForArgs(t *testing.T) {
	const token = "sv=2025-01-01&sig=secret%2Bsignature%3D" //nolint:gosec // G101: synthetic SAS token used to verify redaction.
	const fullToken = token + "&sp=rl"
	args := []string{"restore", "point", "--storage=azure://logs/path?sas-token=" + url.QueryEscape(token), "--full-backup-storage", "azure://full/path?sas-token=" + url.QueryEscape(fullToken)}
	original := append([]string(nil), args...)
	redactor := SASRedactorForArgs(args)
	encoded, err := json.Marshal(token)
	require.NoError(t, err)
	for _, value := range []string{token, fullToken, url.QueryEscape(token), url.QueryEscape(fullToken), string(encoded[1 : len(encoded)-1])} {
		require.Equal(t, "before [REDACTED] after", redactor.Replace("before "+value+" after"))
	}
	require.Equal(t, original, args)
	safe := redactor.Replace(strings.Join(args, " "))
	require.Contains(t, safe, "--storage=azure://logs/path?sas-token=[REDACTED]")
	require.Contains(t, safe, "--full-backup-storage azure://full/path?sas-token=[REDACTED]")
	require.Equal(t, "diagnostic", SASRedactorForArgs([]string{"backup", "--storage=s3://bucket", "?sas-token="}).Replace("diagnostic"))
}

func TestSASRedactorForNativeBRFlags(t *testing.T) {
	const token = "sv=2025-01-01&sig=native-flag-secret" //nolint:gosec // G101: synthetic SAS token used to verify redaction.
	for _, flags := range [][]string{{"--azblob.sas-token=" + token}, {"--azblob.sas-token", token}} {
		args := append([]string{"backup", "full", "--storage=azure://data/"}, flags...)
		redactor := SASRedactorForArgs(args)
		require.NotContains(t, redactor.Replace(strings.Join(args, " ")), "native-flag-secret")
		require.Equal(t, "[ERROR] [REDACTED]", redactor.Replace("[ERROR] "+token))
	}
}
