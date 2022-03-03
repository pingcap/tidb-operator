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
	"fmt"
	"strings"
)

type TiFlashReplication struct {
	ReplicaCount int
	Available    bool
	Progress     float32
}

func (r TiFlashReplication) IsComplete() bool {
	return r.Progress == 1.0
}

func (r TiFlashReplication) IsAvailable() bool {
	return r.Available
}

type TiFlashAction struct {
	*baseAction
}

// CreateReplicas create TiFLash replicas for a table
func (a *TiFlashAction) CreateReplicas(db, table string, count int) error {
	sql := fmt.Sprintf("ALTER TABLE %s.%s SET TIFLASH REPLICA %d;", db, table, count)
	_, err := a.driver().Exec(sql)
	return err
}

// QueryReplication query replication progress
func (a *TiFlashAction) QueryReplication(db, table string) (*TiFlashReplication, error) {
	columns := []string{"REPLICA_COUNT", "AVAILABLE", "PROGRESS"}
	sql := fmt.Sprintf("SELECT %s FROM information_schema.tiflash_replica WHERE TABLE_SCHEMA='%s' and TABLE_NAME='%s';",
		strings.Join(columns, ","), db, table)

	row := a.driver().QueryRow(sql)
	if row.Err() != nil {
		return nil, row.Err()
	}

	var (
		replicaCount int
		available    bool
		progress     float32
	)

	err := row.Scan(&replicaCount, &available, &progress)
	if err != nil {
		return nil, err
	}

	return &TiFlashReplication{
		ReplicaCount: replicaCount,
		Available:    available,
		Progress:     progress,
	}, nil
}
