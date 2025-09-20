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
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

func Workload(ctx context.Context, db *sql.DB) error {
	if err := Ping(ctx, db); err != nil {
		return err
	}

	// keep it to pass e2e
	// TODO(liubo02): it's not reasonable for graceful shutdown test
	db.SetConnMaxLifetime(time.Duration(maxLifeTimeSeconds) * time.Second)

	db.SetMaxIdleConns(maxConnections)
	db.SetMaxOpenConns(maxConnections)
	// Set these variable to avoid too long retry time in testing.
	// Downtime may be short but default timeout is too long.
	fmt.Println("try to set max_execution_time to 2000ms")
	if _, err := db.Exec("set global max_execution_time = 2000;"); err != nil {
		return fmt.Errorf("set max_execute_time failed: %w", err)
	}
	fmt.Println("try to set tidb_backoff_weight to 1")
	if _, err := db.Exec("set global tidb_backoff_weight = 1;"); err != nil {
		return fmt.Errorf("set max_execute_time failed: %w", err)
	}

	table := "test.e2e_test"
	createTableSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id INT PRIMARY KEY AUTO_INCREMENT, v VARCHAR(1000))", table)
	if _, err := db.ExecContext(ctx, createTableSQL); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	var totalCount, failCount atomic.Uint64
	var wg sync.WaitGroup

	ctx, cancel := context.WithTimeout(ctx, time.Duration(durationMinutes)*time.Minute)
	defer cancel()
	for i := 1; i <= maxConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for index := 0; ; index++ {
				select {
				case <-ctx.Done():
					return
				default:
					err := executeSimpleTransaction(ctx, db, id, table, index)
					totalCount.Add(1)
					if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
						fmt.Printf("[%d-%s] failed to execute simple transaction(long: %v): %v\n",
							id, time.Now().String(), id%3 == 0, err,
						)
						failCount.Add(1)
					}
					time.Sleep(time.Duration(sleepInterval) * time.Millisecond)
				}
			}
		}(i)
	}
	wg.Wait()
	fmt.Printf("total count: %d, fail count: %d\n", totalCount.Load(), failCount.Load())
	if failCount.Load() > 0 {
		return fmt.Errorf("there are failed transactions")
	}

	return nil
}

// ExecuteSimpleTransaction performs a transaction to insert or update the given id in the specified table.
func executeSimpleTransaction(ctx context.Context, db *sql.DB, id int, table string, index int) error {
	if tiflashReplicas != 0 {
		if _, err := db.ExecContext(ctx, "set session tidb_enforce_mpp = 1;"); err != nil {
			return fmt.Errorf("set session tidb tidb_enforce_mpp failed: %w", err)
		}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin txn: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
		}
	}()

	// Prepare SQL statement to replace or insert a record
	//nolint:gosec // only for testing
	str := fmt.Sprintf("replace into %s(id, v) values(?, ?);", table)
	if _, err = tx.ExecContext(ctx, str, id, strconv.Itoa(index)); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to exec statement: %w", err)
	}

	rows, err := tx.QueryContext(ctx, fmt.Sprintf("select count(*) from %s;", table))
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to query: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("failed to close query result: %w", err)
	}

	// Simulate a different operation by updating the value
	if _, err = tx.ExecContext(ctx, fmt.Sprintf("update %s set v = ? where id = ?;", table), strconv.Itoa(index*2), id); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to exec update statement: %w", err)
	}

	// Simulate a long transaction by sleeping for 10 seconds
	if id%3 == 0 {
		time.Sleep(time.Duration(longTxnSleepSeconds) * time.Second)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return fmt.Errorf("failed to commit txn: %w", err)
	}
	return nil
}
