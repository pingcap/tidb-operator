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
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Exercise the shared shell library against a local Git remote: no network or
// old-version operator code is needed for this integration test.
func TestFetchOldVersion(t *testing.T) {
	repo := t.TempDir()
	checkout := filepath.Join(t.TempDir(), "checkout")
	git := func(dir string, args ...string) string {
		t.Helper()
		out, err := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git: %v: %s", err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git(repo, "init", "--quiet")
	git(repo, "checkout", "-b", "release/test")
	path := filepath.Join(repo, logPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	commit := func(source string) string {
		t.Helper()
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
		git(repo, "add", ".")
		git(repo, "-c", "user.name=Feature Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "snapshot")
		return git(repo, "rev-parse", "HEAD")
	}
	first := commit(generated(alpha))
	git(repo, "tag", "v-test")
	script, err := filepath.Abs("../../hack/lib/repo.sh")
	if err != nil {
		t.Fatal(err)
	}
	fetch := func(ref string) (string, error) {
		cmd := exec.CommandContext(context.Background(), "bash", "-c", `source "$1"; repo::fetch`, "repo-test", script)
		// Explicit values override any caller's e2e environment.
		cmd.Env = append(os.Environ(), "V_REPO_REF="+ref, "V_REPO_URL="+repo, "V_REPO_DIR="+checkout)
		output, err := cmd.Output()
		return strings.TrimSpace(string(output)), err
	}
	assertFetch := func(ref, want string) {
		t.Helper()
		actual, err := fetch(ref)
		if err != nil {
			t.Fatal(err)
		}
		if actual != checkout {
			t.Fatalf("stdout must contain only checkout path, got %q", actual)
		}
		if actual := git(checkout, "rev-parse", "HEAD"); actual != want {
			t.Fatalf("want %s, got %s", want, actual)
		}
	}
	assertFetch("release/test", first)
	if err := os.WriteFile(filepath.Join(checkout, logPath), []byte("dirty"), 0o600); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(checkout, "stale-generated.go")
	if err := os.WriteFile(stale, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	second := commit(generated(alpha, beta))
	assertFetch("release/test", second)
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale source remains: %v", err)
	}
	if err := lintHistory(repo, checkout); err != nil {
		t.Fatal(err)
	}
	assertFetch(first, first)
	assertFetch("v-test", first)
	for _, ref := range []string{"missing-ref", "--help"} {
		if output, err := fetch(ref); err == nil || output != "" {
			t.Fatalf("failed fetch returned usable checkout: %q, %v", output, err)
		}
	}
}
