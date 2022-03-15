// Copyright 2021 PingCAP, Inc.
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

package db

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// CreateTiFlashReplicationAndWaitToComplete create TiFlash replication and wait it to complete.
//
// TODO: support to set replica to 0.
func CreateTiFlashReplicationAndWaitToComplete(a *TiFlashAction, db, table string, replicaCount int, timeout time.Duration) error {
	err := a.CreateReplicas(db, table, replicaCount)
	if err != nil {
		return fmt.Errorf("failed to create replica: %v", err)
	}

	var lastErr error
	err = wait.PollImmediate(time.Second*5, timeout, func() (done bool, err error) {
		r, err := a.QueryReplication(db, table)
		if err != nil {
			lastErr = err
			return false, nil
		}

		if !r.IsComplete() {
			lastErr = fmt.Errorf("replication %f is not complete yet", r.Progress)
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		return lastErr
	}
	return nil
}

// Count count the number of rows in a table.
func Count(db *Database, engine string, dbName string, tableName string) (int, error) {
	var cnt int
	var queryErr error
	query := func(s *Session) {
		if engine != "" {
			if queryErr = s.SetEngine(context.Background(), engine); queryErr != nil {
				return
			}
		}
		cnt, queryErr = s.Count(context.Background(), dbName, tableName)
	}

	err := db.QueryInSession(query)
	if err != nil {
		return cnt, err
	}

	if queryErr != nil {
		return cnt, fmt.Errorf("failed to query table %s/%s: %v", dbName, tableName, queryErr)
	}

	return cnt, nil
}
