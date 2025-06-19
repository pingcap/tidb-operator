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

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileExists(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	// File should not exist initially
	if fileExists(tmpFile) {
		t.Errorf("fileExists returned true for non-existent file")
	}

	// Create the file
	if err := os.WriteFile(tmpFile, []byte("test"), 0o600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// File should exist now
	if !fileExists(tmpFile) {
		t.Errorf("fileExists returned false for existing file")
	}
}

func TestTLSConfigFromMount(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with empty mount path
	_, err := TLSConfigFromMount("", false)
	if err == nil {
		t.Errorf("Expected error for empty mount path")
	}

	// Test with non-existent directory
	_, err = TLSConfigFromMount("/non/existent/path", false)
	if err != nil {
		// This is expected behavior - we should handle missing files gracefully
		t.Logf("Expected error for non-existent path: %v", err)
	}

	// Test with empty directory
	config, err := TLSConfigFromMount(tmpDir, false)
	if err != nil {
		t.Errorf("Unexpected error for empty directory: %v", err)
	}
	if config == nil {
		t.Errorf("Expected non-nil config")
	}
}

//nolint:errcheck,staticcheck,gosec
func TestTLSConfigFromEnv(t *testing.T) {
	// Save original environment
	originalCA := os.Getenv("TLS_CA_CERT")
	originalCert := os.Getenv("TLS_CLIENT_CERT")
	originalKey := os.Getenv("TLS_CLIENT_KEY")

	// Clean up environment
	defer func() {
		if originalCA != "" {
			os.Setenv("TLS_CA_CERT", originalCA)
		} else {
			os.Unsetenv("TLS_CA_CERT")
		}
		if originalCert != "" {
			os.Setenv("TLS_CLIENT_CERT", originalCert)
		} else {
			os.Unsetenv("TLS_CLIENT_CERT")
		}
		if originalKey != "" {
			os.Setenv("TLS_CLIENT_KEY", originalKey)
		} else {
			os.Unsetenv("TLS_CLIENT_KEY")
		}
	}()

	// Clear environment
	os.Unsetenv("TLS_CA_CERT")
	os.Unsetenv("TLS_CLIENT_CERT")
	os.Unsetenv("TLS_CLIENT_KEY")

	// Test with no environment variables
	config, err := TLSConfigFromEnv(false)
	if err != nil {
		t.Errorf("Unexpected error with no env vars: %v", err)
	}
	if config == nil {
		t.Errorf("Expected non-nil config")
	}
	if config.InsecureSkipVerify {
		t.Errorf("Expected InsecureSkipVerify to be false")
	}

	// Test with InsecureSkipVerify = true
	config, err = TLSConfigFromEnv(true)
	if err != nil {
		t.Errorf("Unexpected error with InsecureSkipVerify=true: %v", err)
	}
	if !config.InsecureSkipVerify {
		t.Errorf("Expected InsecureSkipVerify to be true")
	}
}
