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
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"k8s.io/apimachinery/pkg/util/uuid"
)

type testCase struct {
	cfg        *DDLCaseConfig
	dbs        []*sql.DB
	caseIndex  int
	ddlOps     []ddlTestOpExecutor
	dmlOps     []dmlTestOpExecutor
	tables     map[string]*ddlTestTable
	tablesLock sync.RWMutex
	stop       int32
	lastDDLID  int
}

type ddlTestErrorConflict struct {
}

func (err ddlTestErrorConflict) Error() string {
	return "Conflict operation"
}

func (c *testCase) stopTest() {
	atomic.StoreInt32(&c.stop, 1)
}

func (c *testCase) isStop() bool {
	return atomic.LoadInt32(&c.stop) == 1
}

// pickupRandomTables picks a table randomly. The callee should ensure that
// during this function call the table list is not modified.
//
// Normally the DML op callee should acquire a lock before calling this function
// because the table list may be modified by another parallel DDL op. However
// the DDL op callee doesn't need to acquire a lock because no one will modify the
// table list in parallel ---- DDL ops are executed one by one.
func (c *testCase) pickupRandomTable() *ddlTestTable {
	tableNames := make([]string, 0)
	for name, table := range c.tables {
		if table.isDeleted() {
			continue
		}
		tableNames = append(tableNames, name)
	}
	if len(tableNames) == 0 {
		return nil
	}
	name := tableNames[rand.Intn(len(tableNames))]
	return c.tables[name]
}

func (c *testCase) isTableDeleted(table *ddlTestTable) bool {
	if _, ok := c.tables[table.name]; ok {
		return false
	}
	return true
}

type ddlTestTable struct {
	deleted      int32
	name         string
	id           string // table_id , get from admin show ddl jobs
	columns      []*ddlTestColumn
	indexes      []*ddlTestIndex
	numberOfRows int
	lock         sync.RWMutex
}

func (table *ddlTestTable) isDeleted() bool {
	return atomic.LoadInt32(&table.deleted) != 0
}

func (table *ddlTestTable) setDeleted() {
	atomic.StoreInt32(&table.deleted, 1)
}

func (table *ddlTestTable) filterColumns(predicate func(*ddlTestColumn) bool) []*ddlTestColumn {
	retColumns := make([]*ddlTestColumn, 0)
	for index, col := range table.columns {
		if predicate(col) && !col.isDeleted() {
			retColumns = append(retColumns, table.columns[index])
		}
	}
	return retColumns
}

func (table *ddlTestTable) predicateAll(col *ddlTestColumn) bool {
	return true
}

func (table *ddlTestTable) predicateNotGenerated(col *ddlTestColumn) bool {
	return col.notGenerated()
}

func (table *ddlTestTable) predicatePrimaryKey(col *ddlTestColumn) bool {
	return col.isPrimaryKey
}

func (table *ddlTestTable) predicateNonPrimaryKey(col *ddlTestColumn) bool {
	return !col.isPrimaryKey
}

func (table *ddlTestTable) predicateNonPrimaryKeyAndCanBeWhere(col *ddlTestColumn) bool {
	return !col.isPrimaryKey && col.canBeWhere()
}

func (table *ddlTestTable) predicateNonPrimaryKeyAndNotGen(col *ddlTestColumn) bool {
	return !col.isPrimaryKey && col.notGenerated()
}

// isColumnDeleted checks the col is deleted in this table
// col.isDeleted() will be true before when dropColumnJob(),
// but the col is really deleted after remote TiDB successful execute drop column ddl, and then, the col will be deleted from table.columns.
func (table *ddlTestTable) isColumnDeleted(col *ddlTestColumn) bool {
	for i := range table.columns {
		if col.name == table.columns[i].name {
			return false
		}
	}
	return true
}

func (table *ddlTestTable) debugPrintToString() string {
	var buffer bytes.Buffer
	table.lock.RLock()
	buffer.WriteString(fmt.Sprintf("======== DEBUG BEGIN  ========\n"))
	buffer.WriteString(fmt.Sprintf("Dumping expected contents for table `%s`:\n", table.name))
	if table.isDeleted() {
		buffer.WriteString("[WARN] This table is marked as DELETED.\n")
	}
	buffer.WriteString("## Non-Primary Indexes: \n")
	for i, index := range table.indexes {
		buffer.WriteString(fmt.Sprintf("Index #%d: Name = `%s`, Columnns = [", i, index.name))
		for _, column := range index.columns {
			buffer.WriteString(fmt.Sprintf("`%s`, ", column.name))
		}
		buffer.WriteString("]\n")
	}
	buffer.WriteString("## Columns: \n")
	for i, column := range table.columns {
		buffer.WriteString(fmt.Sprintf("Column #%d", i))
		if column.isDeleted() {
			buffer.WriteString(" [DELETED]")
		}
		buffer.WriteString(fmt.Sprintf(": Name = `%s`, Definition = %s, isPrimaryKey = %v, used in %d indexes\n",
			column.name, column.getDefinition(), column.isPrimaryKey, column.indexReferences))
	}
	buffer.WriteString(fmt.Sprintf("## Values (number of rows = %d): \n", table.numberOfRows))
	for i := 0; i < table.numberOfRows; i++ {
		buffer.WriteString("#")
		buffer.WriteString(padRight(fmt.Sprintf("%d", i), " ", 4))
		buffer.WriteString(": ")
		for _, col := range table.columns {
			buffer.WriteString(padLeft(fmt.Sprintf("%v", col.rows[i]), " ", 11))
			buffer.WriteString(", ")
		}
		buffer.WriteString("\n")
	}
	buffer.WriteString("======== DEBUG END ========\n")
	table.lock.RUnlock()
	return buffer.String()
}

type ddlTestColumnDescriptor struct {
	column *ddlTestColumn
	value  interface{}
}

func (ddlt *ddlTestColumnDescriptor) getValueString() string {
	// make bit data visible
	if ddlt.column.k == KindBit {
		return fmt.Sprintf("b'%v'", ddlt.value)
	} else {
		return fmt.Sprintf("'%v'", ddlt.value)
	}
}

func (ddlt *ddlTestColumnDescriptor) buildConditionSQL() string {
	var sql string
	if ddlt.value == ddlTestValueNull || ddlt.value == nil {
		sql += fmt.Sprintf("`%s` IS NULL", ddlt.column.name)
	} else {
		switch ddlt.column.k {
		case KindFloat:
			sql += fmt.Sprintf("abs(`%s` - %v) < 0.0000001", ddlt.column.name, ddlt.getValueString())
		case KindDouble:
			sql += fmt.Sprintf("abs(`%s` - %v) < 0.0000000000000001", ddlt.column.name, ddlt.getValueString())
		default:
			sql += fmt.Sprintf("`%s` = %v", ddlt.column.name, ddlt.getValueString())
		}
	}
	return sql
}

type ddlTestColumn struct {
	k         int
	deleted   int32
	name      string
	fieldType string

	filedTypeM      int //such as:  VARCHAR(10) ,    filedTypeM = 10
	filedTypeD      int //such as:  DECIMAL(10,5) ,  filedTypeD = 5
	filedPrecision  int
	defaultValue    interface{}
	isPrimaryKey    bool
	rows            []interface{}
	indexReferences int

	dependenciedCols []*ddlTestColumn
	dependency       *ddlTestColumn
	mValue           map[string]interface{}
	nameOfGen        string

	setValue []string //for enum , set data type
}

func (col *ddlTestColumn) isDeleted() bool {
	return atomic.LoadInt32(&col.deleted) != 0
}

func (col *ddlTestColumn) setDeleted() {
	atomic.StoreInt32(&col.deleted, 1)
}

func (col *ddlTestColumn) setDeletedRecover() {
	atomic.StoreInt32(&col.deleted, 0)
}

func (col *ddlTestColumn) getMatchedColumnDescriptor(descriptors []*ddlTestColumnDescriptor) *ddlTestColumnDescriptor {
	for _, d := range descriptors {
		if d.column == col {
			return d
		}
	}
	return nil
}

func (col *ddlTestColumn) getDefinition() string {
	if col.isPrimaryKey {
		return col.fieldType
	}

	if col.isGenerated() {
		return fmt.Sprintf("%s AS (JSON_EXTRACT(`%s`,'$.%s'))", col.fieldType, col.dependency.name, col.nameOfGen)
	}

	if col.canHaveDefaultValue() {
		return fmt.Sprintf("%s NULL DEFAULT %v", col.fieldType, col.getDefaultValueString())
	} else {
		return fmt.Sprintf("%s NULL", col.fieldType)
	}

}

func (col *ddlTestColumn) getSelectName() string {
	if col.k == KindBit {
		return fmt.Sprintf("bin(`%s`)", col.name)
	} else {
		return fmt.Sprintf("`%s`", col.name)
	}
}

func (col *ddlTestColumn) getDefaultValueString() string {
	if col.k == KindBit {
		return fmt.Sprintf("b'%v'", col.defaultValue)
	} else {
		return fmt.Sprintf("'%v'", col.defaultValue)
	}
}

func (col *ddlTestColumn) isEqual(r int, str string) bool {
	vstr := fmt.Sprintf("%v", col.rows[r])
	return strings.Compare(vstr, str) == 0
}

func (col *ddlTestColumn) getDependenciedColsValue(genCol *ddlTestColumn) interface{} {
	if col.mValue == nil {
		return nil
	}
	v := col.mValue[genCol.nameOfGen]
	switch genCol.k {
	case KindChar, KindVarChar, KindTEXT, KindBLOB:
		v = fmt.Sprintf("\"%v\"", v)
	}
	return v
}

func getDDLTestColumn(n int) *ddlTestColumn {
	column := &ddlTestColumn{
		k:         n,
		name:      string(uuid.NewUUID()),
		fieldType: ALLFieldType[n],
		rows:      make([]interface{}, 0),
		deleted:   0,
	}
	switch n {
	case KindChar, KindVarChar, KindBLOB, KindTEXT, KindBit:
		maxLen := getMaxLenByKind(n)
		column.filedTypeM = int(rand.Intn(maxLen))
		for column.filedTypeM == 0 && column.k == KindBit {
			column.filedTypeM = int(rand.Intn(maxLen))
		}

		for column.filedTypeM < 3 && column.k != KindBit { // len('""') = 2
			column.filedTypeM = int(rand.Intn(maxLen))
		}
		column.fieldType = fmt.Sprintf("%s(%d)", ALLFieldType[n], column.filedTypeM)
	case KindTINYBLOB, KindMEDIUMBLOB, KindLONGBLOB, KindTINYTEXT, KindMEDIUMTEXT, KindLONGTEXT:
		column.filedTypeM = getMaxLenByKind(n)
	case KindDECIMAL:
		column.filedTypeM, column.filedTypeD = randMD()
		column.fieldType = fmt.Sprintf("%s(%d,%d)", ALLFieldType[n], column.filedTypeM, column.filedTypeD)
	case KindEnum, KindSet:
		maxLen := getMaxLenByKind(n)
		l := maxLen + 1
		column.setValue = make([]string, l)
		m := make(map[string]struct{})
		column.fieldType += "("
		for i := 0; i < l; i++ {
			column.setValue[i] = randEnumString(m)
			if i > 0 {
				column.fieldType += ", "
			}
			column.fieldType += fmt.Sprintf("\"%s\"", column.setValue[i])
		}
		column.fieldType += ")"
	}

	if column.canHaveDefaultValue() {
		column.defaultValue = column.randValue()
	}

	return column
}

func getRandDDLTestColumn() *ddlTestColumn {
	var n int
	for {
		n = randDataType()
		if n != KindJSON {
			break
		}
	}
	return getDDLTestColumn(n)
}

func getRandDDLTestColumnForJson() *ddlTestColumn {
	var n int
	for {
		n = randDataType()
		if n != KindJSON && n != KindBit && n != KindSet && n != KindEnum {
			break
		}
	}
	return getDDLTestColumn(n)
}

func getRandDDLTestColumns() []*ddlTestColumn {
	n := randDataType()
	cols := make([]*ddlTestColumn, 0)

	if n == KindJSON {
		cols = getRandJsonCol()
	} else {
		column := getDDLTestColumn(n)
		cols = append(cols, column)
	}
	return cols
}

const JsonFieldNum = 5

func getRandJsonCol() []*ddlTestColumn {
	fieldNum := rand.Intn(JsonFieldNum) + 1

	cols := make([]*ddlTestColumn, 0, fieldNum+1)

	column := &ddlTestColumn{
		k:         KindJSON,
		name:      string(uuid.NewUUID()),
		fieldType: ALLFieldType[KindJSON],
		rows:      make([]interface{}, 0),
		deleted:   0,

		dependenciedCols: make([]*ddlTestColumn, 0, fieldNum),
	}

	m := make(map[string]interface{}, 0)
	for i := 0; i < fieldNum; i++ {
		col := getRandDDLTestColumnForJson()
		col.nameOfGen = randFieldName(m)
		m[col.nameOfGen] = col.randValue()
		col.dependency = column

		column.dependenciedCols = append(column.dependenciedCols, col)
		cols = append(cols, col)
	}
	column.mValue = m
	cols = append(cols, column)
	return cols
}

func (col *ddlTestColumn) isGenerated() bool {
	return col.dependency != nil
}

func (col *ddlTestColumn) notGenerated() bool {
	return col.dependency == nil
}

func (col *ddlTestColumn) hasGenerateCol() bool {
	return len(col.dependenciedCols) > 0
}

// randValue return a rand value of the column
func (col *ddlTestColumn) randValue() interface{} {
	switch col.k {
	case KindTINYINT:
		return rand.Int31n(1<<8) - 1<<7
	case KindSMALLINT:
		return rand.Int31n(1<<16) - 1<<15
	case KindMEDIUMINT:
		return rand.Int31n(1<<24) - 1<<23
	case KindInt32:
		return rand.Int63n(1<<32) - 1<<31
	case KindBigInt:
		if rand.Intn(2) == 1 {
			return rand.Int63()
		}
		return -1 - rand.Int63()
	case KindBit:
		if col.filedTypeM >= 64 {
			return fmt.Sprintf("%b", rand.Uint64())
		} else {
			m := col.filedTypeM
			if col.filedTypeM > 7 { // it is a bug
				m = m - 1
			}
			n := (int64)((1 << (uint)(m)) - 1)
			return fmt.Sprintf("%b", rand.Int63n(n))
		}
	case KindFloat:
		return rand.Float32() + 1
	case KindDouble:
		return rand.Float64() + 1
	case KindDECIMAL:
		return randDecimal(col.filedTypeM, col.filedTypeD)
	case KindChar, KindVarChar, KindBLOB, KindTINYBLOB, KindMEDIUMBLOB, KindLONGBLOB, KindTEXT, KindTINYTEXT, KindMEDIUMTEXT, KindLONGTEXT:
		if col.filedTypeM == 0 {
			return ""
		} else {
			if col.isGenerated() {
				if col.filedTypeM <= 2 {
					return ""
				}
				return randSeq(rand.Intn(col.filedTypeM - 2))
			}
			return randSeq(rand.Intn(col.filedTypeM))
		}
	case KindBool:
		return rand.Intn(2)
	case KindDATE:
		randTime := time.Unix(MinDATETIME.Unix()+rand.Int63n(GapDATETIMEUnix), 0)
		return randTime.Format(TimeFormatForDATE)
	case KindTIME:
		randTime := time.Unix(MinTIMESTAMP.Unix()+rand.Int63n(GapTIMESTAMPUnix), 0)
		return randTime.Format(TimeFormatForTIME)
	case KindDATETIME:
		randTime := randTime(MinDATETIME, GapDATETIMEUnix)
		return randTime.Format(TimeFormat)
	case KindTIMESTAMP:
		randTime := randTime(MinTIMESTAMP, GapTIMESTAMPUnix)
		return randTime.Format(TimeFormat)
	case KindYEAR:
		return rand.Intn(254) + 1901 //1901 ~ 2155
	case KindJSON:
		return col.randJsonValue()
	case KindEnum:
		i := rand.Intn(len(col.setValue))
		return col.setValue[i]
	case KindSet:
		var l int
		for l == 0 {
			l = rand.Intn(len(col.setValue))
		}
		idxs := make([]int, l)
		m := make(map[int]struct{})
		for i := 0; i < l; i++ {
			idx := rand.Intn(len(col.setValue))
			_, ok := m[idx]
			for ok {
				idx = rand.Intn(len(col.setValue))
				_, ok = m[idx]
			}
			m[idx] = struct{}{}
			idxs[i] = idx
		}
		sort.Ints(idxs)
		s := ""
		for i := range idxs {
			if i > 0 {
				s += ","
			}
			s += col.setValue[idxs[i]]
		}
		return s
	default:
		return nil
	}
}

func randTime(minTime time.Time, gap int64) time.Time {
	// https://github.com/chronotope/chrono-tz/issues/23
	// see all invalid time: https://timezonedb.com/time-zones/Asia/Shanghai
	var randTime time.Time
	for {
		randTime = time.Unix(minTime.Unix()+rand.Int63n(gap), 0).In(local)
		if notAmbiguousTime(randTime) {
			break
		}
	}
	return randTime
}

func (col *ddlTestColumn) randJsonValue() string {
	for _, dCol := range col.dependenciedCols {
		col.mValue[dCol.nameOfGen] = dCol.randValue()
	}
	jsonRow, _ := json.Marshal(col.mValue)
	return string(jsonRow)
}

func notAmbiguousTime(t time.Time) bool {
	ok := true
	for _, amt := range ambiguousTimeSlice {
		if t.Unix() >= amt.start && t.Unix() <= amt.end {
			ok = false
			break
		}
	}
	return ok
}

// randValueUnique use for primary key column to get unique value
func (col *ddlTestColumn) randValueUnique(PreValue []interface{}) (interface{}, bool) {
	// retry times
	for i := 0; i < 10; i++ {
		v := col.randValue()
		flag := true
		for _, pv := range PreValue {
			if v == pv {
				flag = false
				break
			}
		}
		if flag {
			return v, true
		}
	}
	return nil, false
}

func (col *ddlTestColumn) canBePrimary() bool {
	return col.canBeIndex() && col.notGenerated()
}

func (col *ddlTestColumn) canBeIndex() bool {
	switch col.k {
	case KindChar, KindVarChar:
		if col.filedTypeM == 0 {
			return false
		} else {
			return true
		}
	case KindBLOB, KindTINYBLOB, KindMEDIUMBLOB, KindLONGBLOB, KindTEXT, KindTINYTEXT, KindMEDIUMTEXT, KindLONGTEXT, KindJSON:
		return false
	default:
		return true
	}
}

func (col *ddlTestColumn) canBeSet() bool {
	return col.notGenerated()
}

func (col *ddlTestColumn) canBeWhere() bool {
	switch col.k {
	case KindJSON:
		return false
	default:
		return true
	}
}

//BLOB, TEXT, GEOMETRY or JSON column 'b' can't have a default value")
func (col *ddlTestColumn) canHaveDefaultValue() bool {
	switch col.k {
	case KindBLOB, KindTINYBLOB, KindMEDIUMBLOB, KindLONGBLOB, KindTEXT, KindTINYTEXT, KindMEDIUMTEXT, KindLONGTEXT, KindJSON:
		return false
	default:
		return true
	}
}

type ddlTestIndex struct {
	name      string
	signature string
	columns   []*ddlTestColumn
}
