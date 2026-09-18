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
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const alpha = `{Name: meta.Example, Stage: meta.FeatureStageAlpha, Default: false}`
const beta = `{Name: meta.Example, Stage: meta.FeatureStageBeta, Default: true}`
const stable = `{Name: meta.Example, Stage: meta.FeatureStageStable, Default: true}`
const other = `{Name: meta.Other, Stage: meta.FeatureStageAlpha, Default: false}`

func generated(states ...string) string {
	return "package features\nvar logs = [][]FeatureLog{\n{" + strings.Join(states, "},\n{") + "},\n}\n"
}

func TestHistoryLint(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, logPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	write := func(source string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	baseRoot := t.TempDir()
	basePath := filepath.Join(baseRoot, logPath)
	if err := os.MkdirAll(filepath.Dir(basePath), 0o700); err != nil {
		t.Fatal(err)
	}
	fixture := generated(alpha, beta)
	if err := os.WriteFile(basePath, []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, source, want string }{
		{"unchanged", fixture, ""},
		{"one revision", generated(alpha, beta, stable), ""},
		{"batch", generated(alpha, beta, stable+", "+other), ""},
		{"new feature", generated(alpha, beta, beta+", "+other), ""},
		{"ignore other declarations", fixture + "var unreloadable = 1\nconst CurrentFeatureGateDefinitionHash = \"unrelated\"\n", ""},
		{"format and field order", strings.ReplaceAll(fixture, alpha, `{Default: false, /* comment */ Stage: meta.FeatureStageAlpha, Name: meta.Example}`), ""},
		{"two revisions", generated(alpha, beta, stable, stable+", "+other), "only one new revision"},
		{"modified stage", generated(alpha, stable), "revision 1 changed"},
		{"modified default", strings.Replace(fixture, "Default: false", "Default: true", 1), "revision 0 changed"},
		{"renamed feature", strings.Replace(fixture, "meta.Example", "meta.Renamed", 1), "revision 0 changed"},
		{"removed revision", generated(alpha), "revisions removed"},
		{"insert old revision", generated(alpha+", "+other, beta), "revision 0 changed"},
		{"modify old and append", generated(alpha, stable, stable+", "+other), "revision 1 changed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			write(tc.source)
			err := lintHistory(root, baseRoot)
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
			actual, err := os.ReadFile(path)
			if err != nil || string(actual) != tc.source {
				t.Fatal("lint changed generated logs")
			}
		})
	}
	checkLintFailures(t, root, baseRoot, path, write)
}

func checkLintFailures(t *testing.T, root, baseRoot, path string, write func(string)) {
	t.Helper()
	fixture := generated(alpha, beta)

	write(fixture)
	for _, ref := range []string{"", filepath.Join(t.TempDir(), "missing")} {
		if err := lintHistory(root, ref); err == nil {
			t.Fatalf("accepted invalid base %q", ref)
		}
	}
	// No API source or other generated files exist: lint needs only the log file.
	files, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatal("lint wrote files")
	}
	for _, source := range []string{"package features\n", "invalid Go"} {
		if err := os.WriteFile(filepath.Join(baseRoot, logPath), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := lintHistory(root, baseRoot); err == nil || !strings.Contains(err.Error(), "parse base history") {
			t.Fatalf("invalid baseline accepted: %v", err)
		}
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := lintHistory(root, baseRoot); err == nil {
		t.Fatal("missing current logs accepted")
	}
	if err := os.Remove(filepath.Join(baseRoot, logPath)); err != nil {
		t.Fatal(err)
	}
	write(fixture)
	if err := lintHistory(root, baseRoot); err == nil || !strings.Contains(err.Error(), "only one new revision") {
		t.Fatalf("multiple initial revisions accepted: %v", err)
	}
}

func TestParseLogs(t *testing.T) {
	for _, source := range []string{
		"package features", "package features; var logs = [][]FeatureLog{}",
		"package features; var logs = other", "package features; var logs = []FeatureLog{}",
		generated(alpha) + "var logs = [][]FeatureLog{}", generated(alpha + ", " + alpha),
		generated(strings.Replace(alpha, "Default: false", "Default: value", 1)),
		generated(strings.Replace(alpha, "Default: false", "Default: false, Default: true", 1)),
		generated(strings.Replace(alpha, "Default: false", "Unknown: false", 1)),
		generated(strings.Replace(alpha, "meta.Example", `"Example"`, 1)),
		generated(""),
	} {
		if _, err := parseLogs([]byte(source)); err == nil {
			t.Fatalf("invalid logs accepted: %s", source)
		}
	}
	base, err := parseLogs([]byte(generated(alpha + ", " + other)))
	if err != nil {
		t.Fatal(err)
	}
	reordered, err := parseLogs([]byte(generated(other + ", " + alpha)))
	if err != nil {
		t.Fatal(err)
	}
	if err := checkAppendOnly(base, reordered); err != nil {
		t.Fatalf("entry reordering changed history: %v", err)
	}
}

func TestBaseWithoutFeatureLogs(t *testing.T) {
	root := t.TempDir()
	base := t.TempDir()
	path := filepath.Join(root, logPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(generated(alpha)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := lintHistory(root, base); err != nil {
		t.Fatalf("initial revision rejected: %v", err)
	}
	// A typo in the checkout path must not be treated as an old release.
	if err := lintHistory(root, filepath.Join(base, "missing")); err == nil {
		t.Fatal("missing repository accepted")
	}
	if err := lintHistory(root, path); err == nil {
		t.Fatal("file accepted as repository")
	}
	// Read failures other than absence still fail.
	baseLog := filepath.Join(base, logPath)
	if err := os.MkdirAll(baseLog, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := lintHistory(root, base); err == nil || !strings.Contains(err.Error(), "read base generated logs") {
		t.Fatalf("unreadable log accepted: %v", err)
	}
}
