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
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/golang/glog"
	"github.com/juju/errors"
)

func (c *testCase) generateDDLOps() error {
	if err := c.generateAddTable(); err != nil {
		return errors.Trace(err)
	}
	if err := c.generateDropTable(); err != nil {
		return errors.Trace(err)
	}
	if err := c.generateAddIndex(); err != nil {
		return errors.Trace(err)
	}
	if err := c.generateDropIndex(); err != nil {
		return errors.Trace(err)
	}
	if err := c.generateAddColumn(); err != nil {
		return errors.Trace(err)
	}
	if err := c.generateDropColumn(); err != nil {
		return errors.Trace(err)
	}
	return nil
}

type DDLKind = int

const (
	ddlAddTable DDLKind = iota
	ddlAddIndex
	ddlAddColumn

	ddlDropTable
	ddlDropIndex
	ddlDropColumn

	ddlKindNil
)

var mapOfDDLKind = map[string]DDLKind{
	"create table": ddlAddTable,
	"add index":    ddlAddIndex,
	"add column":   ddlAddColumn,

	"drop table":  ddlDropTable,
	"drop index":  ddlDropIndex,
	"drop column": ddlDropColumn,
}

var mapOfDDLKindToString = map[DDLKind]string{
	ddlAddTable:  "create table",
	ddlAddIndex:  "add index",
	ddlAddColumn: "add column",

	ddlDropTable:  "drop table",
	ddlDropIndex:  "drop index",
	ddlDropColumn: "drop column",
}

// mapOfDDLKindProbability use to control every kind of ddl request execute probability.
var mapOfDDLKindProbability = map[DDLKind]float64{
	ddlAddTable:  0.15,
	ddlDropTable: 0.15,

	ddlAddIndex:  0.8,
	ddlDropIndex: 0.5,

	ddlAddColumn:  0.8,
	ddlDropColumn: 0.5,
}

type ddlJob struct {
	id        int
	tableName string
	k         DDLKind
	jobState  string
	tableID   string
}

type ddlJobArg unsafe.Pointer

type ddlJobTask struct {
	ddlID   int
	k       DDLKind
	tblInfo *ddlTestTable
	sql     string
	arg     ddlJobArg
	err     error // remote TiDB execute error
}

func (c *testCase) updateTableInfo(task *ddlJobTask) error {
	switch task.k {
	case ddlAddTable:
		return c.addTableInfo(task)
	case ddlDropTable:
		return c.dropTableJob(task)
	case ddlAddIndex:
		return c.addIndexJob(task)
	case ddlDropIndex:
		return c.dropIndexJob(task)
	case ddlAddColumn:
		return c.addColumnJob(task)
	case ddlDropColumn:
		return c.dropColumnJob(task)
	}
	return fmt.Errorf("unknow ddl task , %v", *task)
}

/*
execParaDDLSQL get a batch of ddl from taskCh, and then:
1. Parallel send every kind of DDL request to TiDB
2. Wait all DDL SQLs request finish
3. Send `admin show ddl jobs` request to TiDB to confirm parallel DDL requests execute order
4. Do the same DDL change on local with the same DDL requests executed order of TiDB
5. Judge the every DDL execution result of TiDB and local. If both of local and TiDB execute result are no wrong, or both are wrong it will be ok. Otherwise, It must be something wrong.
*/
func (c *testCase) execParaDDLSQL(taskCh chan *ddlJobTask, num int) error {
	if num == 0 {
		return nil
	}
	tasks := make([]*ddlJobTask, 0, num)
	var wg sync.WaitGroup
	for i := 0; i < num; i++ {
		task := <-taskCh
		tasks = append(tasks, task)
		wg.Add(1)
		go func(task *ddlJobTask) {
			defer wg.Done()
			opStart := time.Now()
			db := c.dbs[0]
			_, err := db.Exec(task.sql)
			glog.Infof("[ddl] [instance %d] TiDB execute %s , err %v, table_id %s, elapsed time:%v", c.caseIndex, task.sql, err, task.tblInfo.id, time.Since(opStart).Seconds())
			task.err = err
		}(task)
	}
	wg.Wait()
	db := c.dbs[0]
	SortTasks, err := c.getSortTask(db, tasks)
	if err != nil {
		return err
	}
	for _, task := range SortTasks {
		err := c.updateTableInfo(task)
		glog.Infof("[ddl] [instance %d] local execute %s, err %v , table_id %s, ddlID %v", c.caseIndex, task.sql, err, task.tblInfo.id, task.ddlID)
		if err == nil && task.err != nil || err != nil && task.err == nil {
			return fmt.Errorf("Error when executing SQL: %s\n, local err: %#v, remote tidb err: %#v\n%s\n", task.sql, err, task.err, task.tblInfo.debugPrintToString())
		}
	}
	return nil
}

// execSerialDDLSQL gets a job from taskCh, and then executes the job.
func (c *testCase) execSerialDDLSQL(taskCh chan *ddlJobTask) error {
	if len(taskCh) < 1 {
		return nil
	}
	task := <-taskCh
	db := c.dbs[0]
	opStart := time.Now()
	_, err := db.Exec(task.sql)
	glog.Infof("[ddl] [instance %d] %s, elapsed time:%v", c.caseIndex, task.sql, time.Since(opStart).Seconds())
	if err != nil {
		return fmt.Errorf("Error when executing SQL: %s\n remote tidb Err: %#v\n%s\n", task.sql, err, task.tblInfo.debugPrintToString())
	}
	err = c.updateTableInfo(task)
	if err != nil {
		return fmt.Errorf("Error when executing SQL: %s\n local Err: %#v\n%s\n", task.sql, err, task.tblInfo.debugPrintToString())
	}
	return nil
}

func (c *testCase) generateAddTable() error {
	c.ddlOps = append(c.ddlOps, ddlTestOpExecutor{c.prepareAddTable, nil, ddlAddTable})
	return nil
}

func (c *testCase) prepareAddTable(cfg interface{}, taskCh chan *ddlJobTask) error {
	columnCount := rand.Intn(c.cfg.TablesToCreate) + 2
	tableColumns := make([]*ddlTestColumn, 0, columnCount)
	for i := 0; i < columnCount; i++ {
		columns := getRandDDLTestColumns()
		tableColumns = append(tableColumns, columns...)
	}

	// Generate primary key with [0, 3) size
	primaryKeyFields := rand.Intn(3)
	primaryKeys := make([]int, 0)
	if primaryKeyFields > 0 {
		// Random elections column as primary key, but also check the column whether can be primary key.
		perm := rand.Perm(len(tableColumns))[0:primaryKeyFields]
		for _, columnIndex := range perm {
			if tableColumns[columnIndex].canBePrimary() {
				tableColumns[columnIndex].isPrimaryKey = true
				primaryKeys = append(primaryKeys, columnIndex)
			}
		}
		primaryKeyFields = len(primaryKeys)
	}

	tableInfo := ddlTestTable{
		name:         string(uuid.NewUUID()),
		columns:      tableColumns,
		indexes:      make([]*ddlTestIndex, 0),
		numberOfRows: 0,
		deleted:      0,
	}

	sql := fmt.Sprintf("CREATE TABLE `%s` (", tableInfo.name)
	for i := 0; i < len(tableInfo.columns); i++ {
		if i > 0 {
			sql += ", "
		}
		sql += fmt.Sprintf("`%s` %s", tableColumns[i].name, tableColumns[i].getDefinition())
	}
	if primaryKeyFields > 0 {
		sql += ", PRIMARY KEY ("
		for i, columnIndex := range primaryKeys {
			if i > 0 {
				sql += ", "
			}
			sql += fmt.Sprintf("`%s`", tableColumns[columnIndex].name)
		}
		sql += ")"
	}
	sql += ")"

	task := &ddlJobTask{
		k:       ddlAddTable,
		sql:     sql,
		tblInfo: &tableInfo,
	}
	taskCh <- task
	return nil
}

func (c *testCase) addTableInfo(task *ddlJobTask) error {
	c.tablesLock.Lock()
	defer c.tablesLock.Unlock()
	c.tables[task.tblInfo.name] = task.tblInfo

	return nil
}

func (c *testCase) generateDropTable() error {
	c.ddlOps = append(c.ddlOps, ddlTestOpExecutor{c.prepareDropTable, nil, ddlDropTable})
	return nil
}

func (c *testCase) prepareDropTable(cfg interface{}, taskCh chan *ddlJobTask) error {
	c.tablesLock.Lock()
	defer c.tablesLock.Unlock()
	tableToDrop := c.pickupRandomTable()
	if len(c.tables) <= 1 || tableToDrop == nil {
		return nil
	}
	tableToDrop.setDeleted()
	sql := fmt.Sprintf("DROP TABLE `%s`", tableToDrop.name)

	task := &ddlJobTask{
		k:       ddlDropTable,
		sql:     sql,
		tblInfo: tableToDrop,
	}
	taskCh <- task
	return nil
}

func (c *testCase) dropTableJob(task *ddlJobTask) error {
	c.tablesLock.Lock()
	defer c.tablesLock.Unlock()
	if c.isTableDeleted(task.tblInfo) {
		return fmt.Errorf("table %s is not exists", task.tblInfo.name)
	}
	delete(c.tables, task.tblInfo.name)
	return nil
}

type ddlTestIndexStrategy = int

const (
	ddlTestIndexStrategyBegin ddlTestIndexStrategy = iota
	ddlTestIndexStrategySingleColumnAtBeginning
	ddlTestIndexStrategySingleColumnAtEnd
	ddlTestIndexStrategySingleColumnRandom
	ddlTestIndexStrategyMultipleColumnRandom
	ddlTestIndexStrategyEnd
)

type ddlTestAddIndexConfig struct {
	strategy ddlTestIndexStrategy
}

type ddlIndexJobArg struct {
	index *ddlTestIndex
}

func (c *testCase) generateAddIndex() error {
	c.ddlOps = append(c.ddlOps, ddlTestOpExecutor{c.prepareAddIndex, nil, ddlAddIndex})
	return nil
}

func (c *testCase) prepareAddIndex(_ interface{}, taskCh chan *ddlJobTask) error {
	table := c.pickupRandomTable()
	if table == nil {
		return nil
	}
	strategy := rand.Intn(ddlTestIndexStrategyMultipleColumnRandom) + ddlTestIndexStrategySingleColumnAtBeginning
	// build index definition
	index := ddlTestIndex{
		name:      string(uuid.NewUUID()),
		signature: "",
		columns:   make([]*ddlTestColumn, 0),
	}

	switch strategy {
	case ddlTestIndexStrategySingleColumnAtBeginning:
		if !table.columns[0].canBeIndex() {
			return nil
		}
		index.columns = append(index.columns, table.columns[0])
	case ddlTestIndexStrategySingleColumnAtEnd:
		if !table.columns[len(table.columns)-1].canBeIndex() {
			return nil
		}
		index.columns = append(index.columns, table.columns[len(table.columns)-1])
	case ddlTestIndexStrategySingleColumnRandom:
		col := table.columns[rand.Intn(len(table.columns))]
		if !col.canBeIndex() {
			return nil
		}
		index.columns = append(index.columns, col)
	case ddlTestIndexStrategyMultipleColumnRandom:
		numberOfColumns := rand.Intn(len(table.columns)) + 1
		// Multiple columns of one index should no more than 16.
		if numberOfColumns > 10 {
			numberOfColumns = 10
		}
		perm := rand.Perm(len(table.columns))[:numberOfColumns]
		for _, idx := range perm {
			if table.columns[idx].canBeIndex() {
				index.columns = append(index.columns, table.columns[idx])
			}
		}
	}

	if len(index.columns) == 0 {
		return nil
	}

	signature := ""
	for _, col := range index.columns {
		signature += col.name + ","
	}
	index.signature = signature

	// check whether index duplicates
	for _, idx := range table.indexes {
		if idx.signature == index.signature {
			return nil
		}
	}

	// build SQL
	sql := fmt.Sprintf("ALTER TABLE `%s` ADD INDEX `%s` (", table.name, index.name)
	for i, column := range index.columns {
		if i > 0 {
			sql += ", "
		}
		sql += fmt.Sprintf("`%s`", column.name)
	}
	sql += ")"

	arg := &ddlIndexJobArg{index: &index}
	task := &ddlJobTask{
		k:       ddlAddIndex,
		sql:     sql,
		tblInfo: table,
		arg:     ddlJobArg(arg),
	}
	taskCh <- task
	return nil
}

func (c *testCase) addIndexJob(task *ddlJobTask) error {
	jobArg := (*ddlIndexJobArg)(task.arg)
	tblInfo := task.tblInfo

	if c.isTableDeleted(tblInfo) {
		return fmt.Errorf("table %s is not exists", tblInfo.name)
	}

	for _, column := range jobArg.index.columns {
		if tblInfo.isColumnDeleted(column) {
			return fmt.Errorf("local Execute add index %s on column %s error , column is deleted", jobArg.index.name, column.name)
		}
	}
	tblInfo.indexes = append(tblInfo.indexes, jobArg.index)
	for _, column := range jobArg.index.columns {
		column.indexReferences++
	}
	return nil
}

func (c *testCase) generateDropIndex() error {
	c.ddlOps = append(c.ddlOps, ddlTestOpExecutor{c.prepareDropIndex, nil, ddlDropIndex})
	return nil
}

func (c *testCase) prepareDropIndex(_ interface{}, taskCh chan *ddlJobTask) error {
	table := c.pickupRandomTable()
	if table == nil {
		return nil
	}
	if len(table.indexes) == 0 {
		return nil
	}
	indexToDropIndex := rand.Intn(len(table.indexes))
	indexToDrop := table.indexes[indexToDropIndex]
	sql := fmt.Sprintf("ALTER TABLE `%s` DROP INDEX `%s`", table.name, indexToDrop.name)

	arg := &ddlIndexJobArg{index: indexToDrop}
	task := &ddlJobTask{
		k:       ddlDropIndex,
		sql:     sql,
		tblInfo: table,
		arg:     ddlJobArg(arg),
	}
	taskCh <- task
	return nil
}

func (c *testCase) dropIndexJob(task *ddlJobTask) error {
	jobArg := (*ddlIndexJobArg)(task.arg)
	tblInfo := task.tblInfo

	if c.isTableDeleted(tblInfo) {
		return fmt.Errorf("table %s is not exists", tblInfo.name)
	}

	iOfDropIndex := -1
	for i := range tblInfo.indexes {
		if jobArg.index.name == tblInfo.indexes[i].name {
			iOfDropIndex = i
			break
		}
	}
	if iOfDropIndex == -1 {
		return fmt.Errorf("table %s , index %s is not exists", tblInfo.name, jobArg.index.name)
	}

	for _, column := range jobArg.index.columns {
		column.indexReferences--
		if column.indexReferences < 0 {
			return fmt.Errorf("drop index, index.column %s Unexpected index reference", column.name)
		}
	}
	tblInfo.indexes = append(tblInfo.indexes[:iOfDropIndex], tblInfo.indexes[iOfDropIndex+1:]...)
	return nil
}

type ddlTestAddDropColumnStrategy = int

const (
	ddlTestAddDropColumnStrategyBegin ddlTestAddDropColumnStrategy = iota
	ddlTestAddDropColumnStrategyAtBeginning
	ddlTestAddDropColumnStrategyAtEnd
	ddlTestAddDropColumnStrategyAtRandom
	ddlTestAddDropColumnStrategyEnd
)

type ddlTestAddDropColumnConfig struct {
	strategy ddlTestAddDropColumnStrategy
}

type ddlColumnJobArg struct {
	column            *ddlTestColumn
	strategy          ddlTestAddDropColumnStrategy
	insertAfterColumn *ddlTestColumn
}

func (c *testCase) generateAddColumn() error {
	c.ddlOps = append(c.ddlOps, ddlTestOpExecutor{c.prepareAddColumn, nil, ddlAddColumn})
	return nil
}

func (c *testCase) prepareAddColumn(_ interface{}, taskCh chan *ddlJobTask) error {
	table := c.pickupRandomTable()
	if table == nil {
		return nil
	}
	strategy := rand.Intn(ddlTestAddDropColumnStrategyAtRandom) + ddlTestAddDropColumnStrategyAtBeginning
	newColumn := getRandDDLTestColumn()
	insertAfterPosition := -1
	// build SQL
	sql := fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s", table.name, newColumn.name, newColumn.getDefinition())
	switch strategy {
	case ddlTestAddDropColumnStrategyAtBeginning:
		sql += " FIRST"
	case ddlTestAddDropColumnStrategyAtEnd:
		// do nothing
	case ddlTestAddDropColumnStrategyAtRandom:
		insertAfterPosition = rand.Intn(len(table.columns))
		sql += fmt.Sprintf(" AFTER `%s`", table.columns[insertAfterPosition].name)
	}

	arg := &ddlColumnJobArg{
		column:   newColumn,
		strategy: strategy,
	}
	if insertAfterPosition != -1 {
		arg.insertAfterColumn = table.columns[insertAfterPosition]
	}
	task := &ddlJobTask{
		k:       ddlAddColumn,
		sql:     sql,
		tblInfo: table,
		arg:     ddlJobArg(arg),
	}
	taskCh <- task
	return nil
}

func (c *testCase) addColumnJob(task *ddlJobTask) error {
	jobArg := (*ddlColumnJobArg)(task.arg)
	table := task.tblInfo
	table.lock.Lock()
	defer table.lock.Unlock()

	if c.isTableDeleted(table) {
		return fmt.Errorf("table %s is not exists", table.name)
	}
	newColumn := jobArg.column
	strategy := jobArg.strategy

	newColumn.rows = make([]interface{}, table.numberOfRows)
	for i := 0; i < table.numberOfRows; i++ {
		newColumn.rows[i] = newColumn.defaultValue
	}

	switch strategy {
	case ddlTestAddDropColumnStrategyAtBeginning:
		table.columns = append([]*ddlTestColumn{newColumn}, table.columns...)
	case ddlTestAddDropColumnStrategyAtEnd:
		table.columns = append(table.columns, newColumn)
	case ddlTestAddDropColumnStrategyAtRandom:
		insertAfterPosition := -1
		for i := range table.columns {
			if jobArg.insertAfterColumn.name == table.columns[i].name {
				insertAfterPosition = i
				break
			}
		}
		if insertAfterPosition == -1 {
			return fmt.Errorf("table %s ,insert column %s after column, column %s is not exists ", table.name, newColumn.name, jobArg.insertAfterColumn.name)
		}
		table.columns = append(table.columns[:insertAfterPosition+1], append([]*ddlTestColumn{newColumn}, table.columns[insertAfterPosition+1:]...)...)
	}
	return nil
}

func (c *testCase) generateDropColumn() error {
	c.ddlOps = append(c.ddlOps, ddlTestOpExecutor{c.prepareDropColumn, nil, ddlDropColumn})
	return nil
}

func (c *testCase) prepareDropColumn(_ interface{}, taskCh chan *ddlJobTask) error {
	table := c.pickupRandomTable()
	if table == nil {
		return nil
	}

	columnsSnapshot := table.filterColumns(table.predicateAll)
	if len(columnsSnapshot) <= 1 {
		return nil
	}

	strategy := rand.Intn(ddlTestAddDropColumnStrategyAtRandom) + ddlTestAddDropColumnStrategyAtBeginning
	columnToDropIndex := -1
	switch strategy {
	case ddlTestAddDropColumnStrategyAtBeginning:
		columnToDropIndex = 0
	case ddlTestAddDropColumnStrategyAtEnd:
		columnToDropIndex = len(table.columns) - 1
	case ddlTestAddDropColumnStrategyAtRandom:
		columnToDropIndex = rand.Intn(len(table.columns))
	}

	columnToDrop := table.columns[columnToDropIndex]

	// primary key columns cannot be dropped
	if columnToDrop.isPrimaryKey {
		return nil
	}

	// column cannot be dropped if the column has generated column dependency
	if columnToDrop.hasGenerateCol() {
		return nil
	}

	// we does not support dropping a column with index
	if columnToDrop.indexReferences > 0 {
		return nil
	}
	columnToDrop.setDeleted()
	sql := fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`", table.name, columnToDrop.name)

	arg := &ddlColumnJobArg{
		column:            columnToDrop,
		strategy:          strategy,
		insertAfterColumn: nil,
	}
	task := &ddlJobTask{
		k:       ddlDropColumn,
		sql:     sql,
		tblInfo: table,
		arg:     ddlJobArg(arg),
	}
	taskCh <- task
	return nil
}

func (c *testCase) dropColumnJob(task *ddlJobTask) error {
	jobArg := (*ddlColumnJobArg)(task.arg)
	table := task.tblInfo
	table.lock.Lock()
	defer table.lock.Unlock()
	if c.isTableDeleted(table) {
		return fmt.Errorf("table %s is not exists", table.name)
	}
	columnToDrop := jobArg.column
	if columnToDrop.indexReferences > 0 {
		columnToDrop.setDeletedRecover()
		return fmt.Errorf("local Execute drop column %s on table %s error , column has index reference", jobArg.column.name, table.name)
	}
	dropColumnPosition := -1
	for i := range table.columns {
		if columnToDrop.name == table.columns[i].name {
			dropColumnPosition = i
			break
		}
	}
	if dropColumnPosition == -1 {
		return fmt.Errorf("table %s ,drop column , column %s is not exists ", table.name, columnToDrop.name)
	}
	// update table definitions
	table.columns = append(table.columns[:dropColumnPosition], table.columns[dropColumnPosition+1:]...)
	// if the drop column is a generated column , we should update the dependency column
	if columnToDrop.isGenerated() {
		col := columnToDrop.dependency
		i := 0
		for i = range col.dependenciedCols {
			if col.dependenciedCols[i].name == columnToDrop.name {
				break
			}
		}
		col.dependenciedCols = append(col.dependenciedCols[:i], col.dependenciedCols[i+1:]...)
	}
	return nil
}

// getHistoryDDLJobs send "admin show ddl jobs" to TiDB to get ddl jobs execute order.
// Use TABLE_NAME or TABLE_ID, and JOB_TYPE to confirm which ddl job is the DDL request we send to TiDB.
// We cannot send the same DDL type to same table more than once in a batch of parallel DDL request. The reason is below:
// For example, execute SQL1: "ALTER TABLE t1 DROP COLUMN c1" , SQL2:"ALTER TABLE t1 DROP COLUMN c2", and the "admin show ddl jobs" result is:
// +--------+---------+------------+--------------+--------------+-----------+----------+-----------+-----------------------------------+--------+
// | JOB_ID | DB_NAME | TABLE_NAME | JOB_TYPE     | SCHEMA_STATE | SCHEMA_ID | TABLE_ID | ROW_COUNT | START_TIME                        | STATE  |
// +--------+---------+------------+--------------+--------------+-----------+----------+-----------+-----------------------------------+--------+
// | 47     | test    | t1         | drop column  | none         | 1         | 44       | 0         | 2018-07-13 13:13:55.57 +0800 CST  | synced |
// | 46     | test    | t1         | drop column  | none         | 1         | 44       | 0         | 2018-07-13 13:13:52.523 +0800 CST | synced |
// +--------+---------+------------+--------------+--------------+-----------+----------+-----------+-----------------------------------+--------+
// We cannot confirm which DDL execute first.
func (c *testCase) getHistoryDDLJobs(db *sql.DB, tasks []*ddlJobTask) ([]*ddlJob, error) {
	// build SQL
	sql := "admin show ddl jobs"
	// execute
	opStart := time.Now()
	rows, err := db.Query(sql)
	glog.Infof("%s, elapsed time:%v", sql, time.Since(opStart).Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]*ddlJob, 0, len(tasks))
	// Read all rows.
	var actualRows [][]string
	for rows.Next() {
		cols, err1 := rows.Columns()
		if err1 != nil {
			return nil, err1
		}

		rawResult := make([][]byte, len(cols))
		result := make([]string, len(cols))
		dest := make([]interface{}, len(cols))
		for i := range rawResult {
			dest[i] = &rawResult[i]
		}

		err1 = rows.Scan(dest...)
		if err1 != nil {
			return nil, err1
		}

		for i, raw := range rawResult {
			if raw == nil {
				result[i] = "NULL"
			} else {
				val := string(raw)
				result[i] = val
			}
		}
		actualRows = append(actualRows, result)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	/*********************************
	  +--------+---------+--------------------------------------+--------------+--------------+-----------+----------+-----------+-----------------------------------+-----------+
	  | JOB_ID | DB_NAME | TABLE_NAME                           | JOB_TYPE     | SCHEMA_STATE | SCHEMA_ID | TABLE_ID | ROW_COUNT | START_TIME                        | STATE     |
	  +--------+---------+--------------------------------------+--------------+--------------+-----------+----------+-----------+-----------------------------------+-----------+
	  | 49519  | test    |                                      | add column   | none         | 49481     | 49511    | 0         | 2018-07-09 21:29:02.249 +0800 CST | cancelled |
	  | 49518  | test    |                                      | drop table   | none         | 49481     | 49511    | 0         | 2018-07-09 21:29:01.999 +0800 CST | synced    |
	  | 49517  | test    | ea5be232-50ce-43b1-8d40-33de2ae08bca | create table | public       | 49481     | 49515    | 0         | 2018-07-09 21:29:01.999 +0800 CST | synced    |
	  +--------+---------+--------------------------------------+--------------+--------------+-----------+----------+-----------+-----------------------------------+-----------+
	  *********************************/
	for _, row := range actualRows {
		if len(row) < 9 {
			return nil, fmt.Errorf("%s return error, no enough column return , return row: %s", sql, row)
		}
		id, err := strconv.Atoi(row[0])
		if err != nil {
			return nil, err
		}
		if id <= c.lastDDLID {
			continue
		}
		k, ok := mapOfDDLKind[row[3]]
		if !ok {
			continue
		}
		job := ddlJob{
			id:        id,
			tableName: row[2],
			k:         k,
			tableID:   row[6], // table id
			jobState:  row[9],
		}
		jobs = append(jobs, &job)
	}
	return jobs, nil
}

// getSortTask return the tasks sort by ddl JOB_ID
func (c *testCase) getSortTask(db *sql.DB, tasks []*ddlJobTask) ([]*ddlJobTask, error) {
	jobs, err := c.getHistoryDDLJobs(db, tasks)
	if err != nil {
		return nil, err
	}
	sortTasks := make([]*ddlJobTask, 0, len(tasks))
	for _, job := range jobs {
		for _, task := range tasks {
			if task.k == ddlAddTable && job.k == ddlAddTable && task.tblInfo.name == job.tableName {
				task.ddlID = job.id
				task.tblInfo.id = job.tableID
				sortTasks = append(sortTasks, task)
				break
			}
			if task.k != ddlAddTable && job.k == task.k && task.tblInfo.id == job.tableID {
				task.ddlID = job.id
				sortTasks = append(sortTasks, task)
				break
			}
		}
		if len(sortTasks) == len(tasks) {
			break
		}
	}

	if len(sortTasks) != len(tasks) {
		str := "admin show ddl jobs len != len(tasks)\n"
		str += "admin get job\n"
		str += fmt.Sprintf("%v\t%v\t%v\t%v\t%v\n", "Job_ID", "TABLE_NAME", "JOB_TYPE", "TABLE_ID", "JOB_STATE")
		for _, job := range jobs {
			str += fmt.Sprintf("%v\t%v\t%v\t%v\t%v\n", job.id, job.tableName, mapOfDDLKindToString[job.k], job.tableID, job.jobState)
		}
		str += "ddl tasks\n"
		str += fmt.Sprintf("%v\t%v\t%v\t%v\n", "Job_ID", "TABLE_NAME", "JOB_TYPE", "TABLE_ID")
		for _, task := range tasks {
			str += fmt.Sprintf("%v\t%v\t%v\t%v\n", task.ddlID, task.tblInfo.name, mapOfDDLKindToString[task.k], task.tblInfo.id)
		}

		str += "ddl sort tasks\n"
		str += fmt.Sprintf("%v\t%v\t%v\t%v\n", "Job_ID", "TABLE_NAME", "JOB_TYPE", "TABLE_ID")
		for _, task := range sortTasks {
			str += fmt.Sprintf("%v\t%v\t%v\t%v\n", task.ddlID, task.tblInfo.name, mapOfDDLKindToString[task.k], task.tblInfo.id)
		}
		return nil, fmt.Errorf(str)
	}

	sort.Sort(ddlJobTasks(sortTasks))
	if len(sortTasks) > 0 {
		c.lastDDLID = sortTasks[len(sortTasks)-1].ddlID
	}
	return sortTasks, nil
}

type ddlJobTasks []*ddlJobTask

func (tasks ddlJobTasks) Swap(i, j int) {
	tasks[i], tasks[j] = tasks[j], tasks[i]
}

func (tasks ddlJobTasks) Len() int {
	return len(tasks)
}

func (tasks ddlJobTasks) Less(i, j int) bool {
	return tasks[i].ddlID < tasks[j].ddlID
}
