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

package tests_test

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pingcap/tidb-operator/cmd/ebs-warmup/filereader"
	"github.com/pingcap/tidb-operator/cmd/ebs-warmup/worker/tasks"
	"github.com/stretchr/testify/require"
)

type ToBeWarmedUp struct {
	basicPath string
	files     []os.FileInfo

	allocID int

	tCtx *testing.T
}

func (t *ToBeWarmedUp) newFileName() string {
	t.allocID += 1
	return filepath.Join(t.basicPath, fmt.Sprintf("%06d.bin", t.allocID))
}

func (t *ToBeWarmedUp) createFile(size int, sync bool) {
	fName := t.newFileName()
	fd, err := os.Create(fName)
	require.NoError(t.tCtx, err)
	defer fd.Close()
	total := 0
	for total < size {
		n, err := fd.Write(make([]byte, 4096))
		require.NoError(t.tCtx, err)
		total += n
	}
	stat, err := os.Stat(fName)
	require.NoError(t.tCtx, err)
	if sync {
		require.NoError(t.tCtx, fd.Sync())
	}
	t.files = append(t.files, stat)
}

func createTestDataSet(t *testing.T) *ToBeWarmedUp {
	dir := t.TempDir()
	return &ToBeWarmedUp{
		basicPath: dir,

		tCtx: t,
	}
}

func (t *ToBeWarmedUp) saveCheckpoint(ckp int64) {
	fd, err := os.Create(t.defaultConfig().CheckpointFile)
	require.NoError(t.tCtx, err)
	_, err = fd.WriteString(fmt.Sprintf("%d", ckp))
	require.NoError(t.tCtx, err)
}

func (t *ToBeWarmedUp) defaultConfig() filereader.Config {
	config := filereader.Config{
		Files:           t.basicPath,
		Type:            "whole",
		RateLimit:       math.Inf(1),
		NWorkers:        2,
		Direct:          false,
		CheckpointEvery: 10,
		CheckpointFile:  filepath.Join(t.basicPath, "warmup-checkpoint.txt"),
	}
	return config
}

type Collector struct {
	mu      sync.Mutex
	records map[string]int
}

func NewCollector() *Collector {
	return &Collector{
		records: make(map[string]int),
	}
}

func (c *Collector) OnStep(file os.FileInfo, size int, dur time.Duration) {
	c.mu.Lock()
	c.records[file.Name()] += size
	c.mu.Unlock()
}

func (c *Collector) CheckWith(t *testing.T, fileInfos []os.FileInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, f := range fileInfos {
		require.GreaterOrEqual(t, int(f.Size()), c.records[f.Name()], "file %s size %d but should be %d", f.Name(), c.records[f.Name()], f.Size())
	}
}

func TestBasic(t *testing.T) {
	tw := createTestDataSet(t)

	for i := 0; i < 100; i++ {
		tw.createFile(4096, false)
	}
	for i := 0; i < 5; i++ {
		tw.createFile(4096*20, false)
	}

	cfg := tw.defaultConfig()
	coll := NewCollector()
	cfg.OnStep = coll.OnStep
	runner := filereader.New(cfg)
	runner.RunAndClose(context.Background())

	coll.CheckWith(t, tw.files)
}

func TestCheckpoint(t *testing.T) {
	tw := createTestDataSet(t)

	for i := 0; i < 100; i++ {
		tw.createFile(4096, false)
		if i == 42 {
			time.Sleep(100 * time.Millisecond)
		}
	}
	for i := 0; i < 5; i++ {
		tw.createFile(4096*20, false)
	}

	ckp := tw.files[42].ModTime().UnixMilli()
	tw.saveCheckpoint(ckp)

	cfg := tw.defaultConfig()
	coll := NewCollector()
	cfg.OnStep = coll.OnStep
	runner := filereader.New(cfg)
	runner.RunAndClose(context.Background())

	require.Len(t, coll.records, 43)
	coll.CheckWith(t, tw.files[:43])
	require.NoFileExists(t, cfg.CheckpointFile)
}

func TestSigAndCheckpoint(t *testing.T) {
	tw := createTestDataSet(t)

	for i := 0; i < 300; i++ {
		tw.createFile(4096+95, i < 200)
	}
	for i := 0; i < 10; i++ {
		tw.createFile(4096*20-42, false)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cfg := tw.defaultConfig()
	coll := NewCollector()
	cfg.OnStep = coll.OnStep
	cfg.OnFireRequest = func(rf *tasks.ReadFile) {
		if strings.HasSuffix(rf.File.Name(), "099.bin") {
			cancel()
		}
	}
	runner := filereader.New(cfg)
	runner.RunAndClose(ctx)
	// Some of file might not be saved.
	coll.CheckWith(t, tw.files[101:])
	require.FileExists(t, cfg.CheckpointFile)

	ctx = context.Background()
	coll2 := NewCollector()
	cfg.OnStep = coll2.OnStep
	runner2 := filereader.New(cfg)
	runner2.RunAndClose(ctx)
	coll2.CheckWith(t, tw.files[:101])
	require.NoFileExists(t, cfg.CheckpointFile)
}
