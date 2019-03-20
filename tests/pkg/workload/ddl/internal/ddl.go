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
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/golang/glog"
	"github.com/juju/errors"
)

// The DDL test case is intended to test the correctness of DDL operations. It
// generates test cases by probability so that it should be run in background for
// enough time to see if there are any issues.
//
// The DDL test case have multiple go routines run in parallel, one for DML operations,
// other for DDL operations. The feature of each operation (for example, covering
// what kind of scenario) is determined and generated at start up time (See
// `generateDMLOps`, `generateDDLOps``), while the order of each operation is
// randomized in each round.
//
// If there are remaining DDL operations while all DML operations are performed, a
// new round of DML operations will be started (with new randomized order) and when
// all DDL operations are done, the remaining DML operations are discarded. vice
// versa.
//
// Since there are some conflicts between some DDL operations and DML operations,
// for example, inserting a row while removing a column may cause errors in
// inserting because of incorrect column numbers, some locks and some conflicting
// detections are introduced. The conflicting detection will ignore errors raised
// in such scenarios. In addition, the data in memory is stored by column instead
// of by row to minimize data conflicts in adding and removing columns.

type DDLCaseConfig struct {
	Concurrency     int         `toml:"concurrency"`
	MySQLCompatible bool        `toml:"mysql_compactible"`
	TablesToCreate  int         `toml:"tables_to_create"`
	TestTp          DDLTestType `toml:"test_type"`
}

type DDLTestType int

const (
	SerialDDLTest DDLTestType = iota
	ParallelDDLTest
)

type ExecuteDDLFunc func(*testCase, []ddlTestOpExecutor, func() error) error
type ExecuteDMLFunc func(*testCase, []dmlTestOpExecutor, func() error) error

type DDLCase struct {
	cfg   *DDLCaseConfig
	cases []*testCase
}

func (c *DDLCase) String() string {
	return "ddl"
}

// Execute executes each goroutine (i.e. `testCase`) concurrently.
func (c *DDLCase) Execute(ctx context.Context, dbss [][]*sql.DB, exeDDLFunc ExecuteDDLFunc, exeDMLFunc ExecuteDMLFunc) error {
	for _, dbs := range dbss {
		for _, db := range dbs {
			enableTiKVGC(db)
		}
	}

	glog.Infof("[%s] start to test...", c)
	defer func() {
		glog.Infof("[%s] test end...", c)
	}()
	var wg sync.WaitGroup
	for i := 0; i < c.cfg.Concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				err := c.cases[i].execute(exeDDLFunc, exeDMLFunc)
				if err != nil {
					for _, dbs := range dbss {
						for _, db := range dbs {
							disableTiKVGC(db)
						}
					}
					glog.Fatalf("[ddl] [instance %d] ERROR: %s", i, errors.ErrorStack(err))
				}
			}
		}(i)
	}
	wg.Wait()
	return nil
}

// Initialize initializes each concurrent goroutine (i.e. `testCase`).
func (c *DDLCase) Initialize(ctx context.Context, dbss [][]*sql.DB) error {
	for i := 0; i < c.cfg.Concurrency; i++ {
		err := c.cases[i].initialize(dbss[i])
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// NewDDLCase returns a DDLCase, which contains specified `testCase`s.
func NewDDLCase(cfg *DDLCaseConfig) *DDLCase {
	cases := make([]*testCase, cfg.Concurrency)
	for i := 0; i < cfg.Concurrency; i++ {
		cases[i] = &testCase{
			cfg:       cfg,
			tables:    make(map[string]*ddlTestTable),
			ddlOps:    make([]ddlTestOpExecutor, 0),
			dmlOps:    make([]dmlTestOpExecutor, 0),
			caseIndex: i,
			stop:      0,
		}
	}
	b := &DDLCase{
		cfg:   cfg,
		cases: cases,
	}
	return b
}

const (
	ddlTestValueNull    string = "NULL"
	ddlTestValueInvalid int32  = -99
)

type ddlTestOpExecutor struct {
	executeFunc func(interface{}, chan *ddlJobTask) error
	config      interface{}
	ddlKind     DDLKind
}

type dmlTestOpExecutor struct {
	prepareFunc func(interface{}, chan *dmlJobTask) error
	config      interface{}
}

type DMLKind int

const (
	dmlInsert DMLKind = iota
	dmlUpdate
	dmlDelete
)

type dmlJobArg unsafe.Pointer

type dmlJobTask struct {
	k            DMLKind
	tblInfo      *ddlTestTable
	sql          string
	assigns      []*ddlTestColumnDescriptor
	whereColumns []*ddlTestColumnDescriptor
	err          error
}

// initialize generates possible DDL and DML operations for one `testCase`.
// Different `testCase`s will be run in parallel according to the concurrent configuration.
func (c *testCase) initialize(dbs []*sql.DB) error {
	var err error
	c.dbs = dbs
	if err = c.generateDDLOps(); err != nil {
		return errors.Trace(err)
	}
	if err = c.generateDMLOps(); err != nil {
		return errors.Trace(err)
	}
	// Create 2 table before executes DDL & DML
	taskCh := make(chan *ddlJobTask, 2)
	c.prepareAddTable(nil, taskCh)
	c.prepareAddTable(nil, taskCh)
	if c.cfg.TestTp == SerialDDLTest {
		err = c.execSerialDDLSQL(taskCh)
		if err != nil {
			return errors.Trace(err)
		}
		err = c.execSerialDDLSQL(taskCh)
	} else {
		err = c.execParaDDLSQL(taskCh, len(taskCh))
	}
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

/*
ParallelExecuteOperations executes process:
1. Generate many kind of DDL SQLs
2. Parallel send every kind of DDL request to TiDB
3. Wait all DDL SQLs request finish
4. Send `admin show ddl jobs` request to TiDB to confirm parallel DDL requests execute order
5. Do the same DDL change on local with the same DDL requests executed order of TiDB
6. Judge the every DDL execution result of TiDB and local. If both of local and TiDB execute result are no wrong, or both are wrong it will be ok. Otherwise, It must be something wrong.
*/
func ParallelExecuteOperations(c *testCase, ops []ddlTestOpExecutor, postOp func() error) error {
	perm := rand.Perm(len(ops))
	taskCh := make(chan *ddlJobTask, len(ops))
	for _, idx := range perm {
		if c.isStop() {
			return nil
		}
		op := ops[idx]
		if rand.Float64() > mapOfDDLKindProbability[op.ddlKind] {
			continue
		}
		op.executeFunc(op.config, taskCh)
	}
	err := c.execParaDDLSQL(taskCh, len(taskCh))
	if err != nil {
		return errors.Trace(err)
	}
	close(taskCh)
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	return nil
}

func SerialExecuteOperations(c *testCase, ops []ddlTestOpExecutor, postOp func() error) error {
	perm := rand.Perm(len(ops))
	taskCh := make(chan *ddlJobTask, 1)
	for _, idx := range perm {
		if c.isStop() {
			return nil
		}
		op := ops[idx]
		if rand.Float64() > mapOfDDLKindProbability[op.ddlKind] {
			continue
		}
		op.executeFunc(op.config, taskCh)
		err := c.execSerialDDLSQL(taskCh)
		if err != nil {
			return errors.Trace(err)
		}
	}
	close(taskCh)
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	return nil
}

func TransactionExecuteOperations(c *testCase, ops []dmlTestOpExecutor, postOp func() error) error {
	transactionOpsLen := rand.Intn(len(ops))
	if transactionOpsLen < 1 {
		transactionOpsLen = 1
	}
	taskCh := make(chan *dmlJobTask, len(ops))
	opNum := 0
	perm := rand.Perm(len(ops))
	for i, idx := range perm {
		if c.isStop() {
			return nil
		}
		op := ops[idx]
		err := op.prepareFunc(op.config, taskCh)
		if err != nil {
			if err.Error() != "Conflict operation" {
				return errors.Trace(err)
			}
			continue
		}
		opNum++
		if opNum >= transactionOpsLen {
			err = c.execDMLInTransactionSQL(taskCh)
			if err != nil {
				return errors.Trace(err)
			}
			transactionOpsLen = rand.Intn(len(ops))
			if transactionOpsLen < 1 {
				transactionOpsLen = 1
			}
			if transactionOpsLen > (len(ops) - i) {
				transactionOpsLen = len(ops) - i
			}
			opNum = 0
			if postOp != nil {
				err = postOp()
				if err != nil {
					return errors.Trace(err)
				}
			}
			time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
		}
	}
	return nil
}

func SerialExecuteDML(c *testCase, ops []dmlTestOpExecutor, postOp func() error) error {
	perm := rand.Perm(len(ops))
	taskCh := make(chan *dmlJobTask, 1)
	for _, idx := range perm {
		if c.isStop() {
			return nil
		}
		op := ops[idx]
		err := op.prepareFunc(op.config, taskCh)
		if err != nil {
			if err.Error() != "Conflict operation" {
				return errors.Trace(err)
			}
			continue
		}
		err = c.execSerialDMLSQL(taskCh)
		if err != nil {
			return errors.Trace(err)
		}
	}
	close(taskCh)
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	return nil
}

// execute iterates over two list of operations concurrently, one is
// ddl operations, one is dml operations.
// When one list completes, it starts over from the beginning again.
// When both of them ONCE complete, it exits.
func (c *testCase) execute(executeDDL ExecuteDDLFunc, exeDMLFunc ExecuteDMLFunc) error {
	var (
		ddlAllComplete int32 = 0
		dmlAllComplete int32 = 0
	)

	err := parallel(func() error {
		var err error
		for {
			err = executeDDL(c, c.ddlOps, nil)
			atomic.StoreInt32(&ddlAllComplete, 1)
			if atomic.LoadInt32(&ddlAllComplete) != 0 && atomic.LoadInt32(&dmlAllComplete) != 0 || err != nil {
				break
			}
		}
		return errors.Trace(err)
	}, func() error {
		var err error
		for {
			err = exeDMLFunc(c, c.dmlOps, func() error {
				return c.executeVerifyIntegrity()
			})
			atomic.StoreInt32(&dmlAllComplete, 1)
			if atomic.LoadInt32(&ddlAllComplete) != 0 && atomic.LoadInt32(&dmlAllComplete) != 0 || err != nil {
				break
			}
		}
		return errors.Trace(err)
	})

	if err != nil {
		ddlFailedCounter.Inc()
		return errors.Trace(err)
	}

	glog.Infof("[ddl] [instance %d] Round completed", c.caseIndex)
	glog.Infof("[ddl] [instance %d] Executing post round operations...", c.caseIndex)

	if !c.cfg.MySQLCompatible {
		err := c.executeAdminCheck()
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

var selectID int32

// executeVerifyIntegrity verifies the integrity of the data in the database
// by comparing the data in memory (that we expected) with the data in the database.
func (c *testCase) executeVerifyIntegrity() error {
	c.tablesLock.RLock()
	tablesSnapshot := make([]*ddlTestTable, 0)
	for _, table := range c.tables {
		tablesSnapshot = append(tablesSnapshot, table)
	}
	gotTableTime := time.Now()
	c.tablesLock.RUnlock()

	uniqID := atomic.AddInt32(&selectID, 1)

	for _, table := range tablesSnapshot {
		table.lock.RLock()
		columnsSnapshot := table.filterColumns(table.predicateAll)
		table.lock.RUnlock()

		// build SQL
		sql := "SELECT "
		for i, column := range columnsSnapshot {
			if i > 0 {
				sql += ", "
			}
			sql += fmt.Sprintf("%s", column.getSelectName())
		}
		sql += fmt.Sprintf(" FROM `%s`", table.name)

		dbIdx := rand.Intn(len(c.dbs))
		db := c.dbs[dbIdx]

		// execute
		opStart := time.Now()
		rows, err := db.Query(sql)
		glog.Infof("[ddl] [instance %d] %s, elapsed time:%v, got table time:%v, selectID:%v", c.caseIndex, sql, time.Since(opStart).Seconds(), gotTableTime, uniqID)
		if err == nil {
			defer rows.Close()
		}
		// When column is removed, SELECT statement may return error so that we ignore them here.
		if table.isDeleted() {
			return nil
		}
		for _, column := range columnsSnapshot {
			if column.isDeleted() {
				return nil
			}
		}
		if err != nil {
			return errors.Annotatef(err, "Error when executing SQL: %s\n%s", sql, table.debugPrintToString())
		}

		// Read all rows.
		var actualRows [][]interface{}
		for rows.Next() {
			cols, err1 := rows.Columns()
			if err1 != nil {
				return errors.Trace(err)
			}

			glog.Infof("[ddl] [instance %d] rows.Columns():%v, len(cols):%v, selectID:%v", c.caseIndex, cols, len(cols), uniqID)

			// See https://stackoverflow.com/questions/14477941/read-select-columns-into-string-in-go
			rawResult := make([][]byte, len(cols))
			result := make([]interface{}, len(cols))
			dest := make([]interface{}, len(cols))
			for i := range rawResult {
				dest[i] = &rawResult[i]
			}

			err1 = rows.Scan(dest...)
			if err1 != nil {
				return errors.Trace(err)
			}

			for i, raw := range rawResult {
				if raw == nil {
					result[i] = ddlTestValueNull
				} else {
					result[i] = trimValue(columnsSnapshot[i].k, raw)
				}
			}

			actualRows = append(actualRows, result)
		}
		if rows.Err() != nil {
			return errors.Trace(rows.Err())
		}

		// Even if SQL executes successfully, column deletion will cause different data as well
		if table.isDeleted() {
			return nil
		}
		for _, column := range columnsSnapshot {
			if column.isDeleted() {
				return nil
			}
		}

		// Make signatures for actual rows.
		actualRowsMap := make(map[string]int)
		for _, row := range actualRows {
			rowString := ""
			for _, col := range row {
				rowString += fmt.Sprintf("%v,", col)
			}
			_, ok := actualRowsMap[rowString]
			if !ok {
				actualRowsMap[rowString] = 0
			}
			actualRowsMap[rowString]++
		}

		// Compare with expecting rows.
		checkTime := time.Now()
		for i := 0; i < table.numberOfRows; i++ {
			rowString := ""
			for _, column := range columnsSnapshot {
				if column.rows[i] == nil {
					rowString += fmt.Sprintf("NULL,")
				} else {
					rowString += fmt.Sprintf("%v,", column.rows[i])
				}
			}
			_, ok := actualRowsMap[rowString]
			if !ok {
				c.stopTest()
				err = fmt.Errorf("Expecting row %s in table `%s` but not found, sql: %s, selectID:%v, checkTime:%v, rowErr:%v, actualRowsMap:%#v\n%s", rowString, table.name, sql, uniqID, checkTime, rows.Err(), actualRowsMap, table.debugPrintToString())
				glog.Infof("err: %v", err)
				return errors.Trace(err)
			}
			actualRowsMap[rowString]--
			if actualRowsMap[rowString] < 0 {
				c.stopTest()
				err = fmt.Errorf("Expecting row %s in table `%s` but not found, sql: %s, selectID:%v, checkTime:%v, rowErr:%v, actualRowsMap:%#v\n%s", rowString, table.name, sql, uniqID, checkTime, rows.Err(), actualRowsMap, table.debugPrintToString())
				glog.Infof("err: %v", err)
				return errors.Trace(err)
			}
		}
		for rowString, occurs := range actualRowsMap {
			if occurs > 0 {
				c.stopTest()
				err = fmt.Errorf("Unexpected row %s in table `%s`, sql: %s, selectID:%v, checkTime:%v, rowErr:%v, actualRowsMap:%#v\n%s", rowString, table.name, sql, uniqID, checkTime, rows.Err(), actualRowsMap, table.debugPrintToString())
				glog.Infof("err: %v", err)
				return errors.Trace(err)
			}
		}
	}
	return nil
}

func trimValue(tp int, val []byte) string {
	// a='{"DnOJQOlx":52,"ZmvzPtdm":82}'
	// eg: set a={"a":"b","b":"c"}
	//     get a={"a": "b", "b": "c"} , so have to remove the space
	if tp == KindJSON {
		for i := 1; i < len(val)-2; i++ {
			if val[i-1] == '"' && val[i] == ':' && val[i+1] == ' ' {
				val = append(val[:i+1], val[i+2:]...)
			}
			if val[i-1] == ',' && val[i] == ' ' && val[i+1] == '"' {
				val = append(val[:i], val[i+1:]...)
			}
		}
	}
	return string(val)
}

func (c *testCase) executeAdminCheck() error {
	if len(c.tables) == 0 {
		return nil
	}

	// build SQL
	sql := "ADMIN CHECK TABLE "
	i := 0
	for _, table := range c.tables {
		if i > 0 {
			sql += ", "
		}
		sql += fmt.Sprintf("`%s`", table.name)
		i++
	}
	dbIdx := rand.Intn(len(c.dbs))
	db := c.dbs[dbIdx]
	// execute
	glog.Infof("[ddl] [instance %d] %s", c.caseIndex, sql)
	_, err := db.Exec(sql)
	if err != nil {
		if ignore_error(err) {
			return nil
		}
		return errors.Annotatef(err, "Error when executing SQL: %s", sql)
	}
	return nil
}
