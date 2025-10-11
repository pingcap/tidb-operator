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
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	action   string
	host     string
	port     string
	user     string
	password string

	// Flags for workload action
	durationMinutes     int
	maxConnections      int
	sleepInterval       int
	longTxnSleepSeconds int
	maxLifeTimeSeconds  int
	tiflashReplicas     int
	maxExecutionTime    int

	// Flags for import action
	batchSize        int
	totalRows        int
	importTable      string
	splitRegionCount int

	// Flags for TLS support
	enableTLS          bool
	tlsCertFile        string
	tlsKeyFile         string
	tlsCAFile          string
	tlsMountPath       string
	tlsFromEnv         bool
	insecureSkipVerify bool

	// Flags for PD region API access
	pdEndpointsStr string
)

const (
	defaultReadHeaderTimeout = 3 * time.Second

	defaultDurationMinutes     = 10
	defaultMaxConnetions       = 30
	defaultSleepInterval       = 100
	defaultLongTxnSleepSeconds = 10
	defaultMaxLifeTimeSeconds  = 60
	defaultMaxExecutionTime    = 2000

	defaultBatchSize = 1000
	defaultTotalRows = 500000
)

func main() {
	parseFlag()

	ctx := signals.SetupSignalHandler()

	ctx, cancel := context.WithCancel(ctx)

	http.HandleFunc("/cancel", CancelHandler(cancel))

	server := &http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: defaultReadHeaderTimeout,
	}
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			fmt.Println("stop server: ", err)
		}
	}()

	run(ctx)

	fmt.Println("workload is done")
}

func run(ctx context.Context) {
	switch action {
	case "pd-region":
		if err := PDRegionAccess(ctx); err != nil {
			panic(err)
		}
	default:
		// enable "cleartext client side plugin" for `tidb_auth_token`.
		// ref: https://github.com/go-sql-driver/mysql?tab=readme-ov-file#allowcleartextpasswords
		params := []string{
			"charset=utf8mb4",
			"allowCleartextPasswords=true",
			"timeout=5s",
		}

		// Setup TLS if enabled
		if enableTLS {
			tlsConfigName, err := setupTLSConfig()
			if err != nil {
				panic(fmt.Errorf("failed to setup TLS config: %w", err))
			}
			params = append(params, fmt.Sprintf("tls=%s", tlsConfigName))
		}

		db, err := sql.Open(
			"mysql",
			fmt.Sprintf("%s:%s@(%s:%s)/test?%s",
				user,
				password,
				host, port,
				strings.Join(params, "&"),
			),
		)
		if err != nil {
			panic(err)
		}
		defer func() {
			if err := db.Close(); err != nil {
				panic(err)
			}
		}()

		switch action {
		case "ping":
			if err := Ping(ctx, db); err != nil {
				panic(err)
			}
		case "workload":
			if err := Workload(ctx, db); err != nil {
				panic(err)
			}
		case "import":
			importCfg := ImportDataConfig{
				DB:               db,
				BatchSize:        batchSize,
				TotalRows:        totalRows,
				TableName:        importTable,
				SplitRegionCount: splitRegionCount,
				TiFlashReplicas:  tiflashReplicas,
			}
			if err := ImportData(ctx, importCfg); err != nil {
				panic(err)
			}
		default:
			panic("unknown action: " + action)
		}
	}
}

func parseFlag() {
	flag.StringVar(&action, "action", "ping", "ping, workload, import, pd-region")
	flag.StringVar(&host, "host", "", "host")
	flag.StringVar(&port, "port", "4000", "port")
	flag.StringVar(&user, "user", "root", "db user")
	flag.StringVar(&password, "password", "", "db password")

	flag.IntVar(&durationMinutes, "duration", defaultDurationMinutes, "duration in minutes")
	flag.IntVar(&maxConnections, "max-connections", defaultMaxConnetions, "max connections")
	flag.IntVar(&sleepInterval, "sleep-interval", defaultSleepInterval, "sleep interval in milliseconds")
	flag.IntVar(&longTxnSleepSeconds, "long-txn-sleep", defaultLongTxnSleepSeconds, "how many seconds to sleep to simulate a long transaction")
	flag.IntVar(&maxLifeTimeSeconds, "max-lifetime", defaultMaxLifeTimeSeconds, "max lifetime in seconds")
	flag.IntVar(&tiflashReplicas, "tiflash-replicas", 0, "replicas of tiflash")
	flag.IntVar(&maxExecutionTime, "max-execution-time", defaultMaxExecutionTime, "max_execution_time of tidb")

	// Flags for import action
	flag.IntVar(&batchSize, "batch-size", defaultBatchSize, "batch size for import action")
	flag.IntVar(&totalRows, "total-rows", defaultTotalRows, "total rows to import for import action")
	flag.StringVar(&importTable, "import-table", "e2e_test", "table name for import action")
	flag.IntVar(&splitRegionCount, "split-region-count", 0, "number of regions to split for import action")

	// Flags for TLS support
	flag.BoolVar(&enableTLS, "enable-tls", false, "enable TLS connection")
	flag.StringVar(&tlsCertFile, "tls-cert", "", "path to TLS certificate file")
	flag.StringVar(&tlsKeyFile, "tls-key", "", "path to TLS private key file")
	flag.StringVar(&tlsCAFile, "tls-ca", "", "path to TLS CA certificate file")
	flag.StringVar(&tlsMountPath, "tls-mount-path", "",
		"path to mounted TLS certificates directory (for Kubernetes secrets)")
	flag.BoolVar(&tlsFromEnv, "tls-from-env", false, "load TLS certificates from environment variables")
	flag.BoolVar(&insecureSkipVerify, "tls-insecure-skip-verify", false, "skip TLS certificate verification")

	// Flags for PD region API access
	flag.StringVar(&pdEndpointsStr, "pd-endpoints", "", "comma-separated PD endpoints for pd-region action")

	flag.Parse()
}

// setupTLSConfig configures TLS for the MySQL connection
func setupTLSConfig() (string, error) {
	var tlsConfig *tls.Config
	var err error

	// Priority: mount path > environment variables > individual files
	switch {
	case tlsMountPath != "":
		fmt.Printf("Loading TLS config from mount path: %s\n", tlsMountPath)
		tlsConfig, err = TLSConfigFromMount(tlsMountPath, insecureSkipVerify)
		if err != nil {
			return "", fmt.Errorf("failed to load TLS config from mount path: %w", err)
		}
	case tlsFromEnv:
		fmt.Println("Loading TLS config from environment variables")
		tlsConfig, err = TLSConfigFromEnv(insecureSkipVerify)
		if err != nil {
			return "", fmt.Errorf("failed to load TLS config from environment: %w", err)
		}
	default:
		// Fallback to individual file paths
		fmt.Println("Loading TLS config from individual file paths")
		tlsConfig = &tls.Config{
			InsecureSkipVerify: insecureSkipVerify, //nolint:gosec // user controllable via flag
		}

		// Load CA certificate if provided
		if tlsCAFile != "" {
			caCert, err := os.ReadFile(tlsCAFile)
			if err != nil {
				return "", fmt.Errorf("failed to read CA file %s: %w", tlsCAFile, err)
			}

			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return "", fmt.Errorf("failed to append CA certs from %s", tlsCAFile)
			}
			tlsConfig.RootCAs = caCertPool
		}

		// Load client certificate and key if provided
		if tlsCertFile != "" && tlsKeyFile != "" {
			cert, err := tls.LoadX509KeyPair(tlsCertFile, tlsKeyFile)
			if err != nil {
				return "", fmt.Errorf("failed to load client certificate from %s and %s: %w", tlsCertFile, tlsKeyFile, err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
	}

	// Register TLS config with MySQL driver
	tlsConfigName := "tidb-testing-workload"
	if err := mysql.RegisterTLSConfig(tlsConfigName, tlsConfig); err != nil {
		return "", fmt.Errorf("failed to register TLS config: %w", err)
	}

	fmt.Printf("TLS config registered successfully with name: %s\n", tlsConfigName)
	return tlsConfigName, nil
}

func CancelHandler(cancel context.CancelFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		cancel()
		fmt.Println("gracefully stopping workload")
		w.WriteHeader(http.StatusOK)
	}
}
