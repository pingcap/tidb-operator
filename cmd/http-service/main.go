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
	"flag"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/pingcap/log"
	"go.uber.org/zap"

	"github.com/pingcap/tidb-operator/http-service/middlewares"
	"github.com/pingcap/tidb-operator/http-service/version"
)

const (
	defaultLogMaxDays = 7
	defaultLogMaxSize = 512 // MB
)

func main() {
	// init config
	cfg := NewConfig()
	err := cfg.Parse(os.Args[1:])
	switch err {
	case nil:
	case flag.ErrHelp:
		os.Exit(0)
	default:
		os.Exit(2)
	}

	// init logger
	logger, props, err := log.InitLogger(&log.Config{
		Level: cfg.LogLevel,
		File: log.FileLogConfig{
			Filename: cfg.LogFile,
			MaxSize:  defaultLogMaxSize,
			MaxDays:  defaultLogMaxDays,
		}})
	if err != nil {
		panic(err)
	}
	log.ReplaceGlobals(logger, props)
	log.Info("Starting http-service", zap.String("version", version.GetRawInfo()))

	r := gin.New()
	r.Use(middlewares.LoggingMiddleware(), gin.Recovery()) // log with custom format

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.Run(cfg.Addr)
}
