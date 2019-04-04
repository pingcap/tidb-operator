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
package internal

import (
	"database/sql"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"golang.org/x/net/context"
)

var defaultPushMetricsInterval = 15 * time.Second
var enableTransactionTestFlag = "0"
var enableTransactionTest = false

func init() {
	if enableTransactionTestFlag == "1" {
		enableTransactionTest = true
	}
}

func getPromAddr() string {
	return "" // TODO
}

func openDB(dsn string, maxIdleConns int) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(maxIdleConns)
	glog.V(4).Info("DB opens successfully")
	return db, nil
}

func Run(ctx context.Context, dbDSN string, concurrency int, tablesToCreate int, mysqlCompatible bool, testTp DDLTestType) {
	glog.V(4).Infof("[ddl] Enable transaction test is: %v", enableTransactionTest)

	dbss := make([][]*sql.DB, 0, concurrency)
	for i := 0; i < concurrency; i++ {
		dbs := make([]*sql.DB, 0, 2)
		// Parallel send DDL request need more connection to send DDL request concurrently
		db0, err := openDB(dbDSN, 20)
		if err != nil {
			glog.Fatalf("[ddl] create db client error %v", err)
		}
		db1, err := openDB(dbDSN, 1)
		if err != nil {
			glog.Fatalf("[ddl] create db client error %v", err)
		}
		dbs = append(dbs, db0)
		dbs = append(dbs, db1)
		dbss = append(dbss, dbs)
	}

	if promAddr := getPromAddr(); len(promAddr) > 0 {
		go func() {
			for {
				err := push.FromGatherer("ddl", push.HostnameGroupingKey(), promAddr, prometheus.DefaultGatherer)
				if err != nil {
					glog.Errorf("[ddl] could not push metrics to prometheus push gateway: %v", err)
				}

				time.Sleep(defaultPushMetricsInterval)
			}
		}()
	}

	cfg := DDLCaseConfig{
		Concurrency:     concurrency,
		TablesToCreate:  tablesToCreate,
		MySQLCompatible: mysqlCompatible,
		TestTp:          testTp,
	}
	ddl := NewDDLCase(&cfg)
	exeDDLFunc := SerialExecuteOperations
	if cfg.TestTp == ParallelDDLTest {
		exeDDLFunc = ParallelExecuteOperations
	}
	execDMLFunc := SerialExecuteDML
	if enableTransactionTest {
		execDMLFunc = TransactionExecuteOperations
	}
	if err := ddl.Initialize(ctx, dbss); err != nil {
		glog.Fatalf("[ddl] initialze error %v", err)
	}
	if err := ddl.Execute(ctx, dbss, exeDDLFunc, execDMLFunc); err != nil {
		glog.Fatalf("[ddl] execute error %v", err)
	}
}

func ignore_error(err error) bool {
	if err == nil {
		return true
	}
	errStr := err.Error()
	if strings.Contains(errStr, "Information schema is changed") {
		return true
	}
	if strings.Contains(errStr, "try again later") {
		return true
	}
	return false
}
