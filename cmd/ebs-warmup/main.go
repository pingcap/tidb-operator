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
	"math"

	"github.com/pingcap/tidb-operator/cmd/ebs-warmup/filereader"
	"github.com/spf13/pflag"
)

var (
	files               = pflag.String("files", "*", "What files should be warmed up? This can be a bash glob.")
	ty                  = pflag.String("type", "footer", "Where to warm up? `footer` or `whole`.")
	rateLimit           = pflag.Float64P("ratelimit", "r", math.Inf(1), "What is the max speed of reading? (in MiB/s)")
	nWorkers            = pflag.IntP("workers", "P", 32, "How many workers should we start?")
	direct              = pflag.Bool("direct", false, "Should we use direct I/O?")
	checkpointFileCount = pflag.Uint64("checkpoint.every", 100, "After processing how many files, should we save the checkpoint?")
	checkpointFile      = pflag.String("checkpoint.at", "warmup-checkpoint.txt", "Where should we save & read the checkpoint?")
)

func main() {
	pflag.Parse()

	config := filereader.Config{
		Files:           *files,
		Type:            *ty,
		RateLimit:       *rateLimit,
		NWorkers:        *nWorkers,
		Direct:          *direct,
		CheckpointEvery: *checkpointFileCount,
		CheckpointFile:  *checkpointFile,
	}

	rd := filereader.New(context.Background(), config)
	rd.Run()
}
