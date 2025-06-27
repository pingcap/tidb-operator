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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
)

// TLSConfigFromMount loads TLS configuration from mounted certificates
// This is useful when running in Kubernetes environment where certificates
// are mounted as volumes from secrets
func TLSConfigFromMount(mountPath string, insecureSkipVerify bool) (*tls.Config, error) {
	if mountPath == "" {
		return nil, fmt.Errorf("mount path cannot be empty")
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecureSkipVerify, //nolint:gosec // user controllable
	}

	// Check for CA certificate
	caPath := filepath.Join(mountPath, "ca.crt")
	if fileExists(caPath) {
		caCert, err := os.ReadFile(caPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file %s: %w", caPath, err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to append CA certs from %s", caPath)
		}
		tlsConfig.RootCAs = caCertPool
	}

	// Check for client certificate and key
	certPath := filepath.Join(mountPath, "tls.crt")
	keyPath := filepath.Join(mountPath, "tls.key")

	if fileExists(certPath) && fileExists(keyPath) {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate from %s and %s: %w", certPath, keyPath, err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// TLSConfigFromEnv loads TLS configuration from environment variables
// This allows passing certificate content directly via environment variables
func TLSConfigFromEnv(insecureSkipVerify bool) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecureSkipVerify, //nolint:gosec // user controllable
	}

	// Load CA certificate from environment
	if caCert := os.Getenv("TLS_CA_CERT"); caCert != "" {
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM([]byte(caCert)) {
			return nil, fmt.Errorf("failed to append CA certs from environment variable TLS_CA_CERT")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// Load client certificate and key from environment
	clientCert := os.Getenv("TLS_CLIENT_CERT")
	clientKey := os.Getenv("TLS_CLIENT_KEY")

	if clientCert != "" && clientKey != "" {
		cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate from environment variables: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
