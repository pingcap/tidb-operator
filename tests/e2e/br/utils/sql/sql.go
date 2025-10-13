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
// See the License for the specific language governing permissions and
// limitations under the License.

package sql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/pdapi"
	"github.com/pingcap/tidb-operator/tests/e2e/br/utils/portforward"
	"github.com/pingcap/tidb-operator/tests/e2e/br/utils/types"
	"github.com/pingcap/tidb-operator/tests/e2e/br/utils/verify"
	"github.com/pingcap/tidb-operator/tests/third_party/k8s/log"
	_ "github.com/go-sql-driver/mysql"
)

// ExecuteLogBackupCommandViaSQL executes log backup commands via TiDB SQL (RECOMMENDED APPROACH)
// This method bypasses the Operator and directly modifies kernel state, creating real inconsistency
func ExecuteLogBackupCommandViaSQL(fw portforward.PortForwarder, ns, clusterName, command, taskName string) error {
	ctx := context.Background()
	
	// Reuse existing E2E port forwarding mechanism
	// E2E tests ensure TiDB.Replicas = 1, so we connect to the single TiDB instance
	tidbHost, err := portforward.ForwardOnePort(ctx, fw, ns, 
		GetTiDBServiceResourceName(clusterName), int(v1alpha1.DefaultTiDBServerPort))
	if err != nil {
		return fmt.Errorf("failed to port forward to TiDB: %v", err)
	}
	
	// Reuse existing E2E DSN generation logic
	dsn := GetDefaultDSN(tidbHost, "")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to TiDB: %v", err)
	}
	defer db.Close()
	
	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping TiDB: %v", err)
	}
	
	var sqlCmd string
	switch command {
	case "pause":
		sqlCmd = fmt.Sprintf("PAUSE LOG BACKUP TASK '%s';", taskName)
	case "resume":
		sqlCmd = fmt.Sprintf("RESUME LOG BACKUP TASK '%s';", taskName)
	case "stop":
		sqlCmd = fmt.Sprintf("STOP LOG BACKUP TASK '%s';", taskName)
	default:
		return fmt.Errorf("unsupported command: %s", command)
	}
	
	log.Logf("Executing manual SQL command: %s", sqlCmd)
	_, err = db.Exec(sqlCmd)
	if err != nil {
		return fmt.Errorf("SQL execution failed: %v", err)
	}
	
	log.Logf("Manual log backup command executed successfully via SQL: %s", sqlCmd)
	return nil
}

// GetTiDBServiceResourceName reuses existing E2E helper function
func GetTiDBServiceResourceName(tcName string) string {
	return "svc/" + tcName + "-tidb"
}

// GetDefaultDSN reuses existing E2E helper function  
func GetDefaultDSN(host, dbName string) string {
	user := "root"
	password := ""
	dsn := fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8", user, password, host, dbName)
	return dsn
}

// WaitForKernelStateChange polls etcd until kernel state matches expected state
func WaitForKernelStateChange(etcdClient pdapi.PDEtcdClient, backupName string, expectedState types.LogBackupKernelState, timeout time.Duration) error {
	return verify.WaitForKernelStateChange(etcdClient, backupName, expectedState, timeout)
}