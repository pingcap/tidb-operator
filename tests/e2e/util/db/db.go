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

	"k8s.io/kubernetes/test/e2e/framework"
)

type baseAction struct {
	db *Database
}

func (a *baseAction) driver() *sql.DB {
	return a.db.driver
}

// Database is a wrapper of sql.DB in order to provide some common fuction for test.
type Database struct {
	dsn    string
	driver *sql.DB
}

func NewDatabaseOrDie(dataSourceName string) *Database {
	db, err := NewDatabase(dataSourceName)
	framework.ExpectNoError(err, "failed to open db")
	return db
}

func NewDatabase(dataSourceName string) (*Database, error) {
	db := &Database{
		dsn: dataSourceName,
	}

	driver, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %v", err)
	}
	db.driver = driver

	// Ping to verify the data source name is valid
	err = db.driver.Ping()
	if err != nil {
		return nil, fmt.Errorf("first ping failed: %v", err)
	}

	return db, nil
}

// QueryInSession creates a new session and call the query function.
func (d *Database) QueryInSession(query func(s *Session)) error {
	s, err := d.Session()
	if err != nil {
		return fmt.Errorf("failed to open session: %v", err)
	}
	defer s.Close()

	query(s)

	return nil
}

// Session creates a new session.
func (d *Database) Session() (*Session, error) {
	conn, err := d.driver.Conn(context.TODO())
	if err != nil {
		return nil, err
	}

	return &Session{
		conn: conn,
	}, nil
}

// TiFlashAction create a TiFlashAction
func (d *Database) TiFlashAction() *TiFlashAction {
	return &TiFlashAction{
		baseAction: &baseAction{
			db: d,
		},
	}
}

func (d *Database) Ping() error {
	return d.driver.Ping()
}

func (d *Database) Close() error {
	return d.driver.Close()
}
