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
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	router := gin.New()
	router.Use(middlewares.LoggingMiddleware(), gin.Recovery()) // log with custom format

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: router,
	}
	// init the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Info("The listener closed", zap.String("addr", cfg.Addr), zap.Error(err))
		}
	}()

	// wait for interrupt signal to gracefully shutdown the server with a timeout of 5 seconds
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown", zap.Error(err))
	}
	log.Info("Server exiting")
}
