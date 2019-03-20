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

package ddl

import (
	"context"

	"github.com/juju/errors"
	"github.com/pingcap/tidb-operator/tests/pkg/workload"
	"github.com/pingcap/tidb-operator/tests/pkg/workload/ddl/internal"
)

var _ = workload.Workload(&DDLWorkload{})

type DDLWorkload struct {
	DSN         string
	Concurrency int
	Tables      int

	ctx    context.Context
	cancel context.CancelFunc
}

func New(dsn string, concurrency int, tables int) workload.Workload {
	return &DDLWorkload{DSN: dsn, Concurrency: concurrency, Tables: tables}
}

func (w *DDLWorkload) Enter() error {
	if w.ctx != nil {
		return errors.New("already in ddl workload context")
	}
	w.ctx, w.cancel = context.WithCancel(context.Background())
	go internal.Run(w.ctx, w.DSN, w.Concurrency, w.Tables, false, internal.ParallelDDLTest)
	return nil
}

func (w *DDLWorkload) Leave() {
	w.cancel()
}
