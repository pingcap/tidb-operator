// Copyright 2019 PingCAP, Inc.
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
	"fmt"
	"os"
	"path/filepath"
)

// Valid file extensions for plugin executables.
var ExecutableFileExtensions = []string{""}

// FindInPath returns the full path of the binary by searching in the provided path
func FindInPath(binary string, paths []string) (string, error) {
	if binary == "" {
		return "", fmt.Errorf("no binary name provided")
	}

	if len(paths) == 0 {
		return "", fmt.Errorf("no paths provided")
	}

	for _, path := range paths {
		for _, fe := range ExecutableFileExtensions {
			fullpath := filepath.Join(path, binary) + fe
			if fi, err := os.Stat(fullpath); err == nil && fi.Mode().IsRegular() {
				return fullpath, nil
			}
		}
	}

	return "", fmt.Errorf("failed to find binary %s in path %s", binary, paths)
}
