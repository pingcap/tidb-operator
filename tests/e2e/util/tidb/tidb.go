// Copyright 2020 PingCAP, Inc.
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

package tidb

import (
	"context"
	"database/sql"
	"fmt"

	// To register MySQL driver
	_ "github.com/go-sql-driver/mysql"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/tests/e2e/util/portforward"
	"k8s.io/apimachinery/pkg/util/wait"
)

var dummyCancel = func() {}

// GetTiDBDSN returns a DSN to use
func GetTiDBDSN(fw portforward.PortForward, ns, tc, user, password, database string) (string, context.CancelFunc, error) {
	localHost, localPort, cancel, err := portforward.ForwardOnePort(fw, ns, fmt.Sprintf("svc/%s", controller.TiDBMemberName(tc)), 4000)
	if err != nil {
		return "", dummyCancel, err
	}
	return fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=utf8", user, password, localHost, localPort, database), cancel, nil
}

// TiDBIsConnectable checks whether the tidb cluster is connectable.
func TiDBIsConnectable(fw portforward.PortForward, ns, tc, user, password string) wait.ConditionFunc {
	return func() (bool, error) {
		var db *sql.DB
		dsn, cancel, err := GetTiDBDSN(fw, ns, tc, "root", password, "test")
		if err != nil {
			return false, err
		}
		defer cancel()
		if db, err = sql.Open("mysql", dsn); err != nil {
			return false, err
		}
		defer db.Close()
		if err := db.Ping(); err != nil {
			return false, err
		}
		return true, nil
	}
}

// TiDBIsInserted checks whether the tidb cluster has insert some data.
func TiDBIsInserted(fw portforward.PortForward, ns, tc, user, password, dbName, tableName string) wait.ConditionFunc {
	return func() (bool, error) {
		var db *sql.DB
		dsn, cancel, err := GetTiDBDSN(fw, ns, tc, user, password, dbName)
		if err != nil {
			return false, err
		}

		defer cancel()
		if db, err = sql.Open("mysql", dsn); err != nil {
			return false, err
		}

		defer db.Close()
		if err := db.Ping(); err != nil {
			return false, err
		}

		getCntFn := func(db *sql.DB, tableName string) (int, error) {
			var cnt int
			row := db.QueryRow(fmt.Sprintf("SELECT count(*) FROM %s", tableName))

			err := row.Scan(&cnt)
			if err != nil {
				return cnt, fmt.Errorf("failed to scan count from %s, %v", tableName, err)
			}
			return cnt, nil

		}

		cnt, err := getCntFn(db, tableName)
		if err != nil {
			return false, err
		}
		if cnt == 0 {
			return false, nil
		}

		return true, nil
	}
}
