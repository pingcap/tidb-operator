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
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

// lintHistory compares generated snapshots without loading or running either revision.
func lintHistory(root, base string) error {
	if base == "" {
		return fmt.Errorf("--base-root is required")
	}
	source, err := os.ReadFile(filepath.Join(root, logPath))
	if err != nil {
		return err
	}
	current, err := parseLogs(source)
	if err != nil {
		return fmt.Errorf("parse current history: %w", err)
	}
	return compareBase(base, current)
}

func compareBase(base string, current []snapshot) error {
	info, err := os.Stat(base)
	if err != nil {
		return fmt.Errorf("read base repository: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("base repository is not a directory: %s", base)
	}

	source, err := os.ReadFile(filepath.Join(base, logPath))
	if os.IsNotExist(err) {
		// Releases before feature logging have no published revisions. Initial
		// adoption must still append exactly one revision (rev 0).
		return checkAppendOnly(nil, current)
	}
	if err != nil {
		return fmt.Errorf("read base generated logs: %w", err)
	}
	previous, err := parseLogs(source)
	if err != nil {
		return fmt.Errorf("parse base history: %w", err)
	}
	return checkAppendOnly(previous, current)
}

func checkAppendOnly(base, current []snapshot) error {
	if len(current) < len(base) {
		return fmt.Errorf("feature history is not append-only: revisions removed")
	}
	for rev := range base {
		if !reflect.DeepEqual(current[rev], base[rev]) {
			return fmt.Errorf("feature history is not append-only: revision %d changed; append changes at revision %d", rev, len(base))
		}
	}
	if len(current) > len(base)+1 {
		return fmt.Errorf("feature history must append only one new revision: combine all new changes at revision %d", len(base))
	}
	return nil
}

func main() {
	root := flag.String("root", ".", "repository root")
	base := flag.String("base-root", "", "path to the fetched old-version repository")
	flag.Parse()
	if err := lintHistory(*root, *base); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
