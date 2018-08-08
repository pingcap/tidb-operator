// Copyright 2017 PingCAP, Inc.
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
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pingcap/pd/pkg/faketikv"
	"github.com/pingcap/pd/pkg/faketikv/simutil"
	"github.com/pingcap/pd/pkg/logutil"
	"github.com/pingcap/pd/server"
	"github.com/pingcap/pd/server/api"
	"github.com/pingcap/pd/server/schedule"
	log "github.com/sirupsen/logrus"
	// Register schedulers.
	_ "github.com/pingcap/pd/server/schedulers"
	// Register namespace classifiers.
	_ "github.com/pingcap/pd/table"
)

var (
	confName       = flag.String("conf", "", "config name")
	serverLogLevel = flag.String("serverLog", "fatal", "pd server log level.")
	simLogLevel    = flag.String("simLog", "fatal", "simulator log level.")
)

func main() {
	flag.Parse()

	simutil.InitLogger(*simLogLevel)
	start := time.Now()

	schedule.Simulating = true
	_, local, clean := NewSingleServer()
	err := local.Run(context.Background())
	if err != nil {
		simutil.Logger.Fatal("run server error:", err)
	}
	driver := faketikv.NewDriver(local.GetAddr(), *confName)
	err = driver.Prepare()
	if err != nil {
		simutil.Logger.Fatal("simulator prepare error:", err)
	}
	tick := time.NewTicker(100 * time.Millisecond)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	simResult := "FAIL"

EXIT:
	for {
		select {
		case <-tick.C:
			driver.Tick()
			if driver.Check() {
				simResult = "OK"
				break EXIT
			}
		case <-sc:
			break EXIT
		}
	}

	driver.Stop()
	clean()

	fmt.Printf("%s [%s] total iteration: %d, time cost: %v\n", simResult, *confName, driver.TickCount(), time.Since(start))

	if simResult != "OK" {
		os.Exit(1)
	}
}

// NewSingleServer creates a pd server for simulator.
func NewSingleServer() (*server.Config, *server.Server, server.CleanupFunc) {
	cfg := server.NewTestSingleConfig()
	cfg.Log.Level = *serverLogLevel
	err := logutil.InitLogger(&cfg.Log)
	if err != nil {
		log.Fatalf("initialize logger error: %s\n", err)
	}

	s, err := server.CreateServer(cfg, api.NewHandler)
	if err != nil {
		panic("create server failed")
	}

	cleanup := func() {
		s.Close()
		cleanServer(cfg)
	}
	return cfg, s, cleanup
}

func cleanServer(cfg *server.Config) {
	// Clean data directory
	os.RemoveAll(cfg.DataDir)
}
