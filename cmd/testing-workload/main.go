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
	"flag"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var (
	host              string
	durationInMinutes int
	maxConnections    int
	sleepIntervalSec  int
	longTxnSleepSec   int
)

//nolint:mnd // default values
func main() {
	flag.StringVar(&host, "host", "", "host")
	flag.IntVar(&durationInMinutes, "duration", 10, "duration in minutes")
	flag.IntVar(&maxConnections, "max-connections", 30, "max connections")
	flag.IntVar(&sleepIntervalSec, "sleep-interval", 1, "sleep interval in seconds")
	flag.IntVar(&longTxnSleepSec, "long-txn-sleep", 10, "how many seconds to sleep to simulate a long transaction")
	flag.Parse()

	db, err := sql.Open("mysql", fmt.Sprintf("root:@(%s:4000)/test?charset=utf8mb4", host))
	if err != nil {
		panic(err)
	}
	if err = db.Ping(); err != nil {
		panic(err)
	}
	defer db.Close()
	db.SetConnMaxLifetime(time.Minute)
	db.SetMaxIdleConns(maxConnections)
	db.SetMaxOpenConns(maxConnections)

	table := "test.e2e_test"
	str := fmt.Sprintf("create table if not exists %s(id int primary key auto_increment, v int);", table)
	_, err = db.Exec(str)
	if err != nil {
		panic(err)
	}

	var totalCount, failCount atomic.Uint64
	var wg sync.WaitGroup
	clientCtx, cancel := context.WithTimeout(context.Background(), time.Duration(durationInMinutes)*time.Minute)
	defer cancel()

	for i := 1; i <= maxConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-clientCtx.Done():
					return
				default:
					err := executeSimpleTransaction(db, id, table)
					totalCount.Add(1)
					if err != nil {
						fmt.Printf("[%d-%s] failed to execute simple transaction(long: %v): %v\n", id, time.Now().String(), id%3 == 0, err)
						failCount.Add(1)
					}
					time.Sleep(time.Duration(sleepIntervalSec) * time.Second)
				}
			}
		}(i)
	}
	wg.Wait()
	fmt.Printf("total count: %d, fail count: %d\n", totalCount.Load(), failCount.Load())
	if failCount.Load() > 0 {
		panic("there are failed transactions")
	}
}

// ExecuteSimpleTransaction performs a transaction to insert or update the given id in the specified table.
func executeSimpleTransaction(db *sql.DB, id int, table string) error {
	tx, err := db.Begin()
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
	if _, err = tx.Exec(str, id, id); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to exec statement: %w", err)
	}

	// Simulate a different operation by updating the value
	if _, err = tx.Exec(fmt.Sprintf("update %s set v = ? where id = ?;", table), id*2, id); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to exec update statement: %w", err)
	}

	// Simulate a long transaction by sleeping for 10 seconds
	if id%3 == 0 {
		time.Sleep(time.Duration(longTxnSleepSec) * time.Second)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return fmt.Errorf("failed to commit txn: %w", err)
	}
	return nil
}
