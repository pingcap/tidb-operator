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
