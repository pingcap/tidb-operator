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
	"database/sql"
	"fmt"
	"strings"
)

type Session struct {
	conn *sql.Conn
}

// SetEngine set the storage engine of the database.
func (s *Session) SetEngine(ctx context.Context, engines ...string) error {
	sql := fmt.Sprintf(`set SESSION tidb_isolation_read_engines = "%s";`, strings.Join(engines, ","))
	_, err := s.conn.ExecContext(ctx, sql)
	return err
}

// Close the connection.
func (s *Session) Close() error {
	return s.conn.Close()
}

// Count the number of rows in the table.
func (s *Session) Count(ctx context.Context, dbName string, table string) (int, error) {
	sql := fmt.Sprintf("SELECT count(*) FROM %s.%s", dbName, table)

	row := s.conn.QueryRowContext(ctx, sql)

	var cnt int
	err := row.Scan(&cnt)
	return cnt, err
}

func (s *Session) Query(ctx context.Context, sql string) (*sql.Rows, error) {
	return s.conn.QueryContext(ctx, sql)
}
