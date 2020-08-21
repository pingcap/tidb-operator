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

package util

import (
	"database/sql"
	"fmt"
	"strings"

	"k8s.io/klog"
)

// OpenDB opens db
func OpenDB(dsn string, maxIdleConns int) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(maxIdleConns)
	klog.V(4).Info("DB opens successfully")
	return db, nil
}

// Show master commit ts of TiDB
func ShowMasterCommitTS(dsn string) (int64, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	rows, err := db.Query("SHOW MASTER STATUS")
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return 0, err
	}
	idx := -1
	vals := make([]interface{}, len(cols))
	for i := range cols {
		if strings.ToLower(cols[i]) == "position" {
			vals[i] = new(int64)
			idx = i
		} else {
			vals[i] = new(sql.RawBytes)
		}
	}
	if idx < 0 {
		return 0, fmt.Errorf("Error show master commit ts of %s, cannot find 'Position' column", dsn)
	}
	if !rows.Next() {
		return 0, fmt.Errorf("Error show master commit ts of %s, empty result set", dsn)
	}
	if err = rows.Scan(vals...); err != nil {
		return 0, err
	}
	return *vals[idx].(*int64), nil
}
