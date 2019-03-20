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

//import (
//	"database/sql"
//	"sync"
//)
//
//func OpenDB(dsn string, maxIdleConns int) (*sql.DB, error) { panic("unimplemented") }
//
//func Run(dbName string) {
//	dbss := make([][]*sql.DB, 0)
//}
//
//type DDLCaseConfig struct {
//	MySQLCompatible bool
//	Concurrency     int
//	TablesToCreate  int
//}
//
//type DDLCase struct {
//	cfg   *DDLCaseConfig
//	cases []*testCase
//}
//
//type ddlJobTask struct {
//	ddlID   int
//	k       DDLKind
//	tblInfo *ddlTestTable
//	sql     string
//	arg     ddlJobArg
//	err     error // remote TiDB execute error
//}
//
//type ddlTestOpExecutor struct {
//	executeFunc func(interface{}, chan *ddlJobTask) error
//	config      interface{}
//	ddlKind     DDLKind
//}
//
//type dmlTestOpExecutor struct {
//	prepareFunc func(interface{}, chan *dmlJobTask) error
//	config      interface{}
//}
//
//type testCase struct {
//	cfg        *DDLCaseConfig
//	dbs        []*sql.DB
//	caseIndex  int
//	ddlOps     []ddlTestOpExecutor
//	dmlOps     []dmlTestOpExecutor
//	tables     map[string]*ddlTestTable
//	tablesLock sync.RWMutex
//	stop       int32
//	lastDDLID  int
//}
//
//func NewDDLCase(cfg *DDLCaseConfig) *DDLCase {
//	cases := make([]*testCase, cfg.Concurrency)
//	for i := 0; i < cfg.Concurrency; i++ {
//		cases[i] = &testCase{
//			cfg:       cfg,
//			tables:    make(map[string]*ddlTestTable),
//			ddlOps:    make([]ddlTestOpExecutor, 0),
//			dmlOps:    make([]dmlTestOpExecutor, 0),
//			caseIndex: i,
//			stop:      0,
//		}
//	}
//	b := &DDLCase{
//		cfg:   cfg,
//		cases: cases,
//	}
//	return b
//}
