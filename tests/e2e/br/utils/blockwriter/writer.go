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

package blockwriter

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/pingcap/tidb-operator/tests/pkg/util"
)

const (
	defaultTableNum  = 16
	defaultRecordNum = 256
	defaultBatchNum  = 100
	defaultBatchSize = 1024

	createTableExpr = `
CREATE TABLE IF NOT EXISTS block_writer%d (
    id BIGINT NOT NULL AUTO_INCREMENT,
    raw_bytes BLOB NOT NULL,
    PRIMARY KEY (id)
);
`

	insertExpr = `
INSERT INTO block_writer%d (raw_bytes) VALUES %s;
`
)

type BlockWriter interface {
	Write(ctx context.Context, dsn string) error
}

type blockWriter struct {
	tableNum  int
	recordNum int
}

func NewDefault() BlockWriter {
	return New(defaultTableNum, defaultRecordNum)
}

func New(tableNum, recordNum int) BlockWriter {
	return &blockWriter{
		tableNum:  tableNum,
		recordNum: recordNum,
	}
}

func (bw *blockWriter) Write(ctx context.Context, dsn string) error {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}

	for i := 0; i < bw.tableNum; i++ {
		expr := fmt.Sprintf(createTableExpr, i)
		if _, err := db.Exec(expr); err != nil {
			return err
		}
	}

	errors := []error{}
	wg := sync.WaitGroup{}
	for i := 0; i < bw.tableNum; i++ {
		index := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := db.Conn(ctx)
			if err != nil {
				errors = append(errors, err)
				return
			}
			defer conn.Close()

			for k := 0; k < bw.recordNum; k++ {
				values := make([]string, defaultBatchNum)
				for k := 0; k < defaultBatchNum; k++ {
					blockData := util.RandString(defaultBatchSize)
					values[k] = fmt.Sprintf("('%s')", blockData)
				}
				expr := fmt.Sprintf(insertExpr, index, strings.Join(values, ","))

				if _, err := conn.ExecContext(ctx, expr); err != nil {
					errors = append(errors, err)
					break
				}
			}
		}()
	}
	wg.Wait()
	if len(errors) != 0 {
		return fmt.Errorf("write errors: %v", errors)
	}
	return nil
}
