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

package filereader

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"time"

	"github.com/docker/go-units"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/cmd/ebs-warmup/worker"
	"github.com/pingcap/tidb-operator/cmd/ebs-warmup/worker/tasks"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
	"k8s.io/klog/v2"
)

const (
	channelBufSize      = 128
	minimalSegmentSize  = 64 * 1024
	defaultSegmentCount = 16
)

type sendFileHook func(tasks.ReadFile) (brk bool)

type StatedFile struct {
	Info os.FileInfo
	Path string
}

func StatFilesByGlob(glob string) ([]StatedFile, error) {
	klog.V(1).InfoS("Entering dir.", "recPath", glob)
	files, err := filepath.Glob(glob)
	if err != nil {
		return nil, errors.Annotatef(err, "failed to glob files with glob %s", glob)
	}
	stats := make([]StatedFile, 0, len(files))
	klog.V(3).InfoS("Working over files.", "files", files)
	for _, file := range files {
		s, err := os.Stat(file)
		if err != nil {
			return nil, errors.Annotatef(err, "failed to stat file %s", file)
		}
		if s.IsDir() {
			recPath := filepath.Join(file, "*")
			klog.V(2).InfoS("Recursively entering dir.", "recPath", recPath)
			recContent, err := StatFilesByGlob(recPath)
			if err != nil {
				return nil, errors.Annotatef(err, "failed to stat files in dir %s (globing %s)", file, recPath)
			}
			stats = append(stats, recContent...)
		} else {
			stats = append(stats, StatedFile{Info: s, Path: file})
		}
	}
	return stats, nil
}

func WarmUpFooters(glob string, sendToWorker sendFileHook) error {
	files, err := StatFilesByGlob(glob)
	if err != nil {
		return errors.Annotatef(err, "failed to stat files with glob %s", glob)
	}
	for _, file := range files {
		shouldBrk := sendToWorker(tasks.ReadFile{
			Type:     tasks.ReadLastNBytes(16 * 1024),
			File:     file.Info,
			FilePath: file.Path,
		})
		if shouldBrk {
			return nil
		}
	}
	return nil
}

func WarmUpWholeFile(glob string, sendToWorker sendFileHook) error {
	return warmUpWholeFileBy(glob, func(sf StatedFile) bool {
		return sendFileWithSegmenting(sf, defaultSegmentCount, sendToWorker)
	})
}

func warmUpWholeFileBy(glob string, onFile func(StatedFile) (brk bool)) error {
	files, err := StatFilesByGlob(glob)
	if err != nil {
		return errors.Annotatef(err, "failed to stat files with glob %s", glob)
	}
	sort.Slice(files, func(i, j int) bool {
		// Desc order of modify time.
		return files[i].Info.ModTime().After(files[j].Info.ModTime())
	})
	for _, file := range files {
		if onFile(file) {
			return context.Canceled
		}
	}
	return nil
}

func sendFileWithSegmenting(file StatedFile, partitions int, sendToWorker sendFileHook) (brk bool) {
	partitionSize := file.Info.Size() / int64(partitions)
	if partitionSize < minimalSegmentSize {
		partitionSize = minimalSegmentSize
	}
	offset := int64(0)
	for offset <= file.Info.Size() {
		length := partitionSize
		if offset+partitionSize > file.Info.Size() {
			length = file.Info.Size() - offset
		}
		shouldBrk := sendToWorker(
			tasks.ReadFile{
				Type: tasks.ReadOffsetAndLength{
					Offset: offset,
					Length: length,
				},
				File:     file.Info,
				FilePath: file.Path,
			},
		)
		if shouldBrk {
			return true
		}
		offset += partitionSize
	}
	return false
}

type WorkersOpt struct {
	ObserveTotalSize *uint64
	RateLimitInMiB   float64
	OnStep           worker.OnStepHook
}

func CreateWorkers(ctx context.Context, n int, opt WorkersOpt) ([]chan<- tasks.ReadFile, *errgroup.Group) {
	result := make([]chan<- tasks.ReadFile, 0, n)
	eg, ectx := errgroup.WithContext(ctx)

	loopCounter := uint64(0)
	lastTotalSize := uint64(0)
	lastTime := time.Now()
	var limiter *rate.Limiter
	if !math.IsInf(opt.RateLimitInMiB, 0) && !math.Signbit(opt.RateLimitInMiB) {
		limiter = rate.NewLimiter(rate.Limit(opt.RateLimitInMiB*units.MiB), 8*units.MiB)
	}
	for i := 0; i < n; i++ {
		ch := make(chan tasks.ReadFile, channelBufSize)
		i := i
		eg.Go(func() error {
			wr := worker.New(ch)
			wr.RateLimiter = limiter
			wr.OnStep = func(file os.FileInfo, readBytes int, take time.Duration) {
				if opt.OnStep != nil {
					opt.OnStep(file, readBytes, take)
				}
				if opt.ObserveTotalSize != nil {
					new := atomic.AddUint64(opt.ObserveTotalSize, uint64(readBytes))
					if atomic.AddUint64(&loopCounter, 1)%2048 == 0 {
						now := time.Now()
						diff := new - atomic.LoadUint64(&lastTotalSize)
						atomic.StoreUint64(&lastTotalSize, new)
						rate := units.HumanSizeWithPrecision(float64(diff)/now.Sub(lastTime).Seconds(), 4)
						klog.InfoS("Printing rate info.", "rate/s", rate)
						lastTime = time.Now()
					}
				}
				klog.V(2).InfoS(
					"Read bytes from file.", "file", file.Name(),
					"size", file.Size(),
					"read", readBytes,
					"take", take,
					"worker", i,
				)
			}
			err := wr.MainLoop(ectx)
			klog.InfoS("Background worker exits.", "id", i, "err", err)
			return err
		})
		result = append(result, ch)
	}
	return result, eg
}

func RoundRobin[T any](ts []T) func() T {
	n := len(ts)
	choose := func() T {
		n++
		if n >= len(ts) {
			n = 0
		}
		return ts[n]
	}
	return choose
}

func TrySync(ctx context.Context, workers []chan<- tasks.ReadFile) error {
	chs := make([]chan struct{}, 0)
	for _, w := range workers {
		ch := make(chan struct{})
		w <- tasks.ReadFile{Type: tasks.Sync{C: ch}}
		chs = append(chs, ch)
	}
	for _, ch := range chs {
		select {
		case <-ch:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

type ExecContext struct {
	config Config

	wkrs     []chan<- tasks.ReadFile
	eg       *errgroup.Group
	cnt      uint64
	total    uint64
	chooser  func() chan<- tasks.ReadFile
	lastSent uint64
	start    time.Time

	cancel context.CancelFunc
}

func (execCtx *ExecContext) perhapsCheckpoint() (uint64, error) {
	file, err := os.ReadFile(execCtx.config.CheckpointFile)
	if err != nil {
		return 0, errors.Annotatef(err, "failed to open checkpoint file %s", execCtx.config.CheckpointFile)
	}
	var cnt uint64
	_, err = fmt.Sscanf(string(file), "%d", &cnt)
	if err != nil {
		return 0, errors.Annotatef(err, "failed to parse checkpoint file %s", execCtx.config.CheckpointFile)
	}
	return cnt, nil
}

func (execCtx *ExecContext) checkpoint() uint64 {
	ckp, err := execCtx.perhapsCheckpoint()
	if err != nil {
		klog.InfoS("Failed to read checkpoint file. Will use time.Now() as checkpoint.", "err", err)
		// add a 1s offset for newly created files. (usually in test)
		return uint64(time.Now().UnixMilli() + 1000)
	}
	return ckp
}

func (execCtx *ExecContext) saveCheckpoint(ckp uint64) error {
	return os.WriteFile(execCtx.config.CheckpointFile, []byte(fmt.Sprintf("%d", ckp)), 0o644)
}

func New(config Config) *ExecContext {
	execCtx := &ExecContext{
		config: config,
	}
	wCtx, cancel := context.WithCancel(context.Background())
	execCtx.wkrs, execCtx.eg = CreateWorkers(wCtx, execCtx.config.NWorkers, WorkersOpt{
		ObserveTotalSize: &execCtx.total,
		RateLimitInMiB:   execCtx.config.RateLimit,
		OnStep:           config.OnStep,
	})
	execCtx.start = time.Now()
	execCtx.chooser = RoundRobin(execCtx.wkrs)
	execCtx.cnt = uint64(0)
	execCtx.lastSent = execCtx.checkpoint()
	execCtx.cancel = cancel
	return execCtx
}

func (execCtx *ExecContext) RunAndClose(ctx context.Context) error {
	defer execCtx.cancel()

	klog.InfoS("Using checkpoint.", "checkpoint", execCtx.lastSent, "time", time.UnixMilli(int64(execCtx.lastSent)).String())
	var err error
	switch execCtx.config.Type {
	case "footer":
		err = WarmUpFooters(execCtx.config.Files, func(rf tasks.ReadFile) bool {
			return execCtx.sendToWorker(ctx, rf)
		})
	case "whole":
		err = WarmUpWholeFile(execCtx.config.Files, func(rf tasks.ReadFile) bool {
			return execCtx.sendToWorker(ctx, rf)
		})
	}
	if err != nil {
		klog.ErrorS(err, "Failed to initialize the task.")
		return err
	}

	for _, wkr := range execCtx.wkrs {
		close(wkr)
	}
	if err := execCtx.eg.Wait(); err != nil {
		return err
	}

	take := time.Since(execCtx.start)
	total := atomic.LoadUint64(&execCtx.total)
	rate := float64(total) / take.Seconds()
	klog.InfoS("Done.", "take", take, "total", total, "rate", fmt.Sprintf("%s/s", units.HumanSize(rate)))
	err = os.Remove(execCtx.config.CheckpointFile)
	klog.InfoS("Try remove the checkpoint file.", "err", err, "file", execCtx.config.CheckpointFile)
	return nil
}

func (execCtx *ExecContext) checkpointTick(ctx context.Context) {
	execCtx.cnt += 1
	if execCtx.cnt%execCtx.config.CheckpointEvery == 0 {
		ckp := execCtx.lastSent
		now := time.Now()
		if err := TrySync(ctx, execCtx.wkrs); err != nil {
			klog.ErrorS(err, "skip saving checkpoint: the worker is closing", "err", err)
			return
		}
		err := execCtx.saveCheckpoint(ckp)
		if err != nil {
			klog.ErrorS(err, "Failed to save checkpoint.", "checkpoint", ckp, "take", time.Since(now))
		}
	}
}

func (execCtx *ExecContext) sendToWorker(ctx context.Context, rf tasks.ReadFile) bool {
	createTs := rf.File.ModTime().UnixMilli()
	if execCtx.config.OnFireRequest != nil {
		execCtx.config.OnFireRequest(&rf)
	}
	if ctx.Err() != nil {
		klog.InfoS("early exit due to context canceled.", "err", ctx.Err())
		return true
	}
	if createTs > int64(execCtx.lastSent) {
		return false
	}
	execCtx.checkpointTick(ctx)
	rf.Direct = execCtx.config.Direct
	execCtx.chooser() <- rf
	if execCtx.lastSent < uint64(createTs) {
		klog.Warningln("unordered files: checkpoint is unavailable.", "checkpoint=", execCtx.checkpoint(),
			"createTs=", uint64(createTs), "lastSent=", execCtx.lastSent)
	}
	execCtx.lastSent = uint64(createTs)
	return false
}
