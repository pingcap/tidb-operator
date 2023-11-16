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

package main

import (
	"context"
	"flag"
	"math"
	"os"
	"os/signal"
	"time"

	"github.com/pingcap/tidb-operator/cmd/ebs-warmup/filereader"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
)

var (
	files               = pflag.String("files", "*", "What files should be warmed up? This can be a bash glob.")
	ty                  = pflag.String("type", "footer", "Where to warm up? `footer`, `whole` or `hybrid`.")
	rateLimit           = pflag.Float64P("ratelimit", "r", math.Inf(1), "What is the max speed of reading? (in MiB/s)")
	nWorkers            = pflag.IntP("workers", "P", 32, "How many workers should we start?")
	direct              = pflag.Bool("direct", false, "Should we use direct I/O?")
	checkpointFileCount = pflag.Uint64("checkpoint.every", 100, "After processing how many files, should we save the checkpoint?")
	checkpointFile      = pflag.String("checkpoint.at", "warmup-checkpoint.txt", "Where should we save & read the checkpoint?")
	warmupFrom          = pflag.String("hybrid.full-from-recent", time.Now().Add(-24*time.Hour).Format(time.RFC3339),
		"Will only warm up full files from this timestamp(RFC3339 format). Other files will only be warmed up with footer.")
)

// /warmup --type=hibrid --hybrid.full-from-recent 1970-01-01T00:00:01Z -v=3 --files=/etc -P256 --direct

func main() {
	fs := flag.NewFlagSet("klog", flag.ContinueOnError)
	klog.InitFlags(fs)
	pflag.CommandLine.AddGoFlagSet(fs)
	pflag.Parse()

	warmupFromTs, err := time.Parse(time.RFC3339, *warmupFrom)
	if err != nil {
		klog.ErrorS(err, "Failed to parse the time from `hybrid.full-from-recent`.")
		os.Exit(1)
	}
	config := filereader.Config{
		Files:           *files,
		Type:            *ty,
		RateLimit:       *rateLimit,
		NWorkers:        *nWorkers,
		Direct:          *direct,
		CheckpointEvery: *checkpointFileCount,
		CheckpointFile:  *checkpointFile,
		WarmupAfter:     warmupFromTs,
	}

	rd := filereader.New(config)
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		klog.Warning("Received interrupt, stopping...")
		signal.Stop(ch)
		cancel()
	}()
	if err := rd.RunAndClose(ctx); err != nil {
		klog.ErrorS(err, "Run failed.")
		os.Exit(1)
	}
}
