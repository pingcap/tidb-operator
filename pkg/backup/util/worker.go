// Copyright 2023 PingCAP, Inc.
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
	"golang.org/x/sync/errgroup"
	"k8s.io/klog/v2"
)

// WorkerPool contains a pool of workers.
type WorkerPool struct {
	limit   uint
	workers chan *Worker
	name    string
}

// Worker identified by ID.
type Worker struct {
	ID uint64
}

type taskFunc func()

// NewWorkerPool returns a WorkPool.
func NewWorkerPool(limit uint, name string) *WorkerPool {
	workers := make(chan *Worker, limit)
	for i := uint(0); i < limit; i++ {
		workers <- &Worker{ID: uint64(i + 1)}
	}
	return &WorkerPool{
		limit:   limit,
		workers: workers,
		name:    name,
	}
}

// IdleCount counts how many idle workers in the pool.
func (pool *WorkerPool) IdleCount() int {
	return len(pool.workers)
}

// Limit is the limit of the pool
func (pool *WorkerPool) Limit() int {
	return int(pool.limit)
}

// Apply executes a task.
func (pool *WorkerPool) Apply(fn taskFunc) {
	worker := pool.ApplyWorker()
	go func() {
		defer pool.RecycleWorker(worker)
		fn()
	}()
}

// ApplyOnErrorGroup executes a task in an errorgroup.
func (pool *WorkerPool) ApplyOnErrorGroup(eg *errgroup.Group, fn func() error) {
	worker := pool.ApplyWorker()
	eg.Go(func() error {
		defer pool.RecycleWorker(worker)
		return fn()
	})
}

// ApplyWorker apply a worker.
func (pool *WorkerPool) ApplyWorker() *Worker {
	var worker *Worker
	select {
	case worker = <-pool.workers:
	default:
		klog.Infof("wait for workers, pool name %s", pool.name)
		worker = <-pool.workers
	}
	return worker
}

// RecycleWorker recycle a worker.
func (pool *WorkerPool) RecycleWorker(worker *Worker) {
	if worker == nil {
		panic("invalid restore worker")
	}
	pool.workers <- worker
}
